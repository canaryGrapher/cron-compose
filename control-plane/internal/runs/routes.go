package runs

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/croncompose/croncompose/control-plane/internal/agentgw"
)

// Register attaches /jobs/:id/runs, /runs/:id, /runs/:id/logs, /runs/:id/logs/stream.
func Register(r fiber.Router, log *slog.Logger, pool *pgxpool.Pool, broker *agentgw.LogBroker) {
	h := &handler{log: log, store: NewStore(pool), broker: broker}
	r.Get("/jobs/:id/runs", h.listByJob)
	r.Get("/runs/:id", h.get)
	r.Get("/runs/:id/logs", h.logs)
	r.Get("/runs/:id/logs/stream", h.stream)
}
