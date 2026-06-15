// Command migrate applies the SQL files in migrations/ to a Postgres database, in
// lexical order, recording each applied file in a schema_migrations table so re-runs
// are safe. It exists so the installer can set up the schema on any OS without
// requiring the psql client.
//
// Usage:
//
//	migrate                       # uses $DATABASE_URL and ./migrations
//	migrate -dir ../migrations    # explicit migrations directory
//	migrate -db postgres://...     # explicit connection string (overrides env)
//
// It reuses the control-plane module's pgx dependency, so no extra packages are
// pulled in.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "migrate: "+err.Error())
		os.Exit(1)
	}
}

func run() error {
	dir := flag.String("dir", "migrations", "directory containing *.sql migration files")
	dbURL := flag.String("db", os.Getenv("DATABASE_URL"), "Postgres connection string (defaults to $DATABASE_URL)")
	flag.Parse()

	if *dbURL == "" {
		return fmt.Errorf("no database URL: set $DATABASE_URL or pass -db")
	}

	files, err := sqlFiles(*dir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no .sql files found in %s", *dir)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, err := connect(ctx, *dbURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := ensureTable(ctx, pool); err != nil {
		return err
	}
	applied, err := appliedVersions(ctx, pool)
	if err != nil {
		return err
	}

	pending := 0
	for _, f := range files {
		version := filepath.Base(f)
		if applied[version] {
			fmt.Printf("  skip   %s (already applied)\n", version)
			continue
		}
		sql, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read %s: %w", version, err)
		}
		// Each migration file carries its own begin/commit. With no query args pgx
		// uses the simple protocol, which runs all statements in the file.
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("apply %s: %w", version, err)
		}
		if _, err := pool.Exec(ctx,
			"insert into schema_migrations (version, applied_at) values ($1, now())", version); err != nil {
			return fmt.Errorf("record %s: %w", version, err)
		}
		fmt.Printf("  apply  %s\n", version)
		pending++
	}

	if pending == 0 {
		fmt.Println("database is up to date")
	} else {
		fmt.Printf("applied %d migration(s)\n", pending)
	}
	return nil
}

// connect dials the database, retrying briefly so the tool tolerates being run the
// instant a freshly started Postgres is accepting connections.
func connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	var lastErr error
	for attempt := 1; attempt <= 15; attempt++ {
		pool, err := pgxpool.New(ctx, url)
		if err == nil {
			pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			err = pool.Ping(pingCtx)
			cancel()
			if err == nil {
				return pool, nil
			}
			pool.Close()
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return nil, fmt.Errorf("connect: %w", lastErr)
}

func ensureTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `create table if not exists schema_migrations (
		version    text primary key,
		applied_at timestamptz not null default now()
	)`)
	return err
}

func appliedVersions(ctx context.Context, pool *pgxpool.Pool) (map[string]bool, error) {
	rows, err := pool.Query(ctx, "select version from schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = true
	}
	return out, rows.Err()
}

func sqlFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".sql" {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)
	return files, nil
}
