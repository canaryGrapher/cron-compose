package audit

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store is the read side of audit_log.
type Store struct{ pool *pgxpool.Pool }

// NewStore wires a Store.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// List returns recent audit entries, newest first. Filters are AND-combined.
func (s *Store) List(ctx context.Context, actor, action string, limit int) ([]Entry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := `
		select id, actor_user_id, action, target_type, target_id, metadata, ts
		from audit_log
		where ($1 = '' or actor_user_id = $1)
		  and ($2 = '' or action = $2)
		order by ts desc
		limit $3
	`
	rows, err := s.pool.Query(ctx, q, actor, action, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Entry{}
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.ActorUserID, &e.Action, &e.TargetType, &e.TargetID, &e.Metadata, &e.Ts); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
