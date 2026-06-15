package runs

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a run id doesn't exist.
var ErrNotFound = errors.New("run not found")

// Store is the data-access layer for runs and run_logs.
type Store struct{ pool *pgxpool.Pool }

// NewStore wires a Store to a pgx pool.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func scanRun(row pgx.Row) (Run, error) {
	var r Run
	err := row.Scan(
		&r.ID, &r.JobID, &r.JobVersionID, &r.ServerID, &r.Trigger, &r.Status,
		&r.ScheduledFor, &r.StartedAt, &r.FinishedAt, &r.ExitCode, &r.DurationMs,
		&r.Error, &r.CreatedAt,
	)
	return r, err
}

const selectCols = `
  id, job_id, job_version_id, server_id, trigger, status,
  scheduled_for, started_at, finished_at, exit_code, duration_ms,
  coalesce(error,''), created_at
`

// ListByJob returns runs for a job, newest first.
func (s *Store) ListByJob(ctx context.Context, jobID string, limit int) ([]Run, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		select `+selectCols+` from runs where job_id = $1 order by created_at desc limit $2
	`, jobID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Run{}
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Get returns one run or ErrNotFound.
func (s *Store) Get(ctx context.Context, id string) (Run, error) {
	r, err := scanRun(s.pool.QueryRow(ctx, `select `+selectCols+` from runs where id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Run{}, ErrNotFound
	}
	return r, err
}

// Logs returns captured output for a run, ordered by stream + seq.
func (s *Store) Logs(ctx context.Context, runID string) ([]LogLine, error) {
	rows, err := s.pool.Query(ctx, `
		select stream, seq, chunk, ts from run_logs
		where run_id = $1 order by stream, seq
	`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LogLine{}
	for rows.Next() {
		var l LogLine
		if err := rows.Scan(&l.Stream, &l.Seq, &l.Chunk, &l.Ts); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
