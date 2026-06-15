package jobs

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/croncompose/croncompose/control-plane/internal/agentgw"
	"github.com/croncompose/croncompose/control-plane/internal/audit"
	"github.com/croncompose/croncompose/control-plane/internal/auth"
)

// Register attaches the jobs routes with role gating.
//
// viewer:   list, get
// operator: create, patch, delete, enable, disable, run-now
func Register(r fiber.Router, log *slog.Logger, pool *pgxpool.Pool, gw *agentgw.Gateway, writer audit.Writer) {
	h := &handler{
		log:     log,
		store:   NewStore(pool),
		gateway: gw,
		audit:   writer,
	}
	r.Get("/jobs", h.list)
	r.Get("/jobs/:id", h.get)

	op := auth.RequireRole("operator")
	r.Post("/jobs", op, h.create)
	r.Patch("/jobs/:id", op, h.patch)
	r.Delete("/jobs/:id", op, h.delete)
	r.Post("/jobs/:id/enable", op, h.enable)
	r.Post("/jobs/:id/disable", op, h.disable)
	r.Post("/jobs/:id/run", op, h.runNow)
}
