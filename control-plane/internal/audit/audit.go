// Package audit owns the audit_log table: who did what, on which target, with what
// metadata, when.
package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Entry is the read shape returned by the REST API.
type Entry struct {
	ID          int64           `json:"id"`
	ActorUserID *string         `json:"actor_user_id,omitempty"`
	Action      string          `json:"action"`
	TargetType  *string         `json:"target_type,omitempty"`
	TargetID    *string         `json:"target_id,omitempty"`
	Metadata    json.RawMessage `json:"metadata"`
	Ts          time.Time       `json:"ts"`
}

// Writer is the tiny interface mutating handlers use; they don't need to know about
// the table or the SQL.
type Writer interface {
	Write(ctx context.Context, actorUserID, action, targetType, targetID string, metadata map[string]any)
}

// PoolWriter is the default Writer backed by Postgres. Errors are logged but never
// surface to callers, so a failed audit write never breaks the user-facing action.
type PoolWriter struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

// NewWriter wires a PoolWriter.
func NewWriter(pool *pgxpool.Pool, log *slog.Logger) *PoolWriter {
	return &PoolWriter{pool: pool, log: log}
}

// Write inserts one audit row.
func (w *PoolWriter) Write(ctx context.Context, actorUserID, action, targetType, targetID string, metadata map[string]any) {
	md := []byte("{}")
	if metadata != nil {
		if b, err := json.Marshal(metadata); err == nil {
			md = b
		}
	}
	_, err := w.pool.Exec(ctx, `
		insert into audit_log (actor_user_id, action, target_type, target_id, metadata)
		values (nullif($1,''), $2, nullif($3,''), nullif($4,''), $5)
	`, actorUserID, action, targetType, targetID, md)
	if err != nil {
		w.log.Warn("audit write failed", "action", action, "err", err)
	}
}
