package notify

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/croncompose/croncompose/control-plane/internal/ids"
)

// ErrNotFound is returned when a target id doesn't exist.
var ErrNotFound = errors.New("target not found")

// Store wraps the notification_targets table.
type Store struct{ pool *pgxpool.Pool }

// NewStore wires a Store.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// List returns every target.
func (s *Store) List(ctx context.Context) ([]Target, error) {
	rows, err := s.pool.Query(ctx, `
		select id, name, kind, url, enabled, created_at
		from notification_targets order by created_at desc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Target{}
	for rows.Next() {
		var t Target
		if err := rows.Scan(&t.ID, &t.Name, &t.Kind, &t.URL, &t.Enabled, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// EnabledList returns only the enabled targets; this is what the fire path uses.
func (s *Store) EnabledList(ctx context.Context) ([]Target, error) {
	rows, err := s.pool.Query(ctx, `
		select id, name, kind, url, enabled, created_at
		from notification_targets where enabled = true
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Target{}
	for rows.Next() {
		var t Target
		if err := rows.Scan(&t.ID, &t.Name, &t.Kind, &t.URL, &t.Enabled, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Insert persists a target with default kind=webhook, enabled=true.
func (s *Store) Insert(ctx context.Context, in CreateInput) (Target, error) {
	id := ids.New()
	_, err := s.pool.Exec(ctx, `
		insert into notification_targets (id, name, kind, url, enabled)
		values ($1, $2, 'webhook', $3, true)
	`, id, in.Name, in.URL)
	if err != nil {
		return Target{}, err
	}
	var t Target
	err = s.pool.QueryRow(ctx, `
		select id, name, kind, url, enabled, created_at from notification_targets where id = $1
	`, id).Scan(&t.ID, &t.Name, &t.Kind, &t.URL, &t.Enabled, &t.CreatedAt)
	return t, err
}

// Delete drops a target by id.
func (s *Store) Delete(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `delete from notification_targets where id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
