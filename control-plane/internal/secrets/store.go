package secrets

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/croncompose/croncompose/control-plane/internal/cryptobox"
	"github.com/croncompose/croncompose/control-plane/internal/ids"
)

// ErrNotFound is returned when a secret id doesn't exist.
var ErrNotFound = errors.New("secret not found")

// Store wraps the secrets table.
type Store struct {
	pool *pgxpool.Pool
	box  *cryptobox.Box
}

// NewStore wires a Store.
func NewStore(pool *pgxpool.Pool, box *cryptobox.Box) *Store {
	return &Store{pool: pool, box: box}
}

// List returns name + scope metadata, never values.
func (s *Store) List(ctx context.Context) ([]Secret, error) {
	rows, err := s.pool.Query(ctx, `
		select id, scope, coalesce(scope_id,''), name, created_at
		from secrets order by created_at desc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Secret{}
	for rows.Next() {
		var s Secret
		if err := rows.Scan(&s.ID, &s.Scope, &s.ScopeID, &s.Name, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Insert encrypts the value and stores it.
func (s *Store) Insert(ctx context.Context, in CreateInput) (Secret, error) {
	scope := in.Scope
	if scope == "" {
		scope = "global"
	}
	blob, err := s.box.Seal([]byte(in.Value))
	if err != nil {
		return Secret{}, err
	}
	id := ids.New()
	var scopeID any = nil
	if in.ScopeID != "" {
		scopeID = in.ScopeID
	}
	_, err = s.pool.Exec(ctx, `
		insert into secrets (id, scope, scope_id, name, value_enc)
		values ($1, $2, $3, $4, $5)
	`, id, scope, scopeID, in.Name, blob)
	if err != nil {
		return Secret{}, err
	}
	var out Secret
	err = s.pool.QueryRow(ctx, `
		select id, scope, coalesce(scope_id,''), name, created_at from secrets where id = $1
	`, id).Scan(&out.ID, &out.Scope, &out.ScopeID, &out.Name, &out.CreatedAt)
	return out, err
}

// Delete removes a secret by id.
func (s *Store) Delete(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `delete from secrets where id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ResolveForJob looks up the named secrets, decrypts them, and returns name->value.
// Resolution order: a job-scoped secret with the same name beats a server-scoped one
// beats a global one. Missing names are silently skipped (logged by the caller).
func (s *Store) ResolveForJob(ctx context.Context, jobID, serverID string, names []string) (map[string]string, error) {
	if len(names) == 0 {
		return map[string]string{}, nil
	}
	rows, err := s.pool.Query(ctx, `
		select name, scope, value_enc
		from secrets
		where name = any($1)
		  and (
		    scope = 'global'
		    or (scope = 'server' and scope_id = $2)
		    or (scope = 'job' and scope_id = $3)
		  )
	`, names, serverID, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type entry struct {
		scope string
		value []byte
	}
	picked := map[string]entry{}
	for rows.Next() {
		var name, scope string
		var blob []byte
		if err := rows.Scan(&name, &scope, &blob); err != nil {
			return nil, err
		}
		if cur, ok := picked[name]; ok && scopeRank(cur.scope) >= scopeRank(scope) {
			continue
		}
		picked[name] = entry{scope: scope, value: blob}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make(map[string]string, len(picked))
	for name, e := range picked {
		plain, err := s.box.Open(e.value)
		if err != nil {
			return nil, err
		}
		out[name] = string(plain)
	}
	return out, nil
}

func scopeRank(s string) int {
	switch s {
	case "job":
		return 3
	case "server":
		return 2
	case "global":
		return 1
	}
	return 0
}

// scanRow is the column list ResolveForJob filters by pgx.Row (kept for symmetry).
var _ = pgx.ErrNoRows
