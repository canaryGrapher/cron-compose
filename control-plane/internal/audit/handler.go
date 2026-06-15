package audit

import (
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/croncompose/croncompose/control-plane/internal/auth"
)

type handler struct {
	log   *slog.Logger
	store *Store
}

// Register attaches GET /audit (admin only).
func Register(r fiber.Router, log *slog.Logger, pool *pgxpool.Pool) {
	h := &handler{log: log, store: NewStore(pool)}
	r.Get("/audit", auth.RequireRole("admin"), h.list)
}

func (h *handler) list(c fiber.Ctx) error {
	actor := c.Query("actor", "")
	action := c.Query("action", "")
	limit, _ := strconv.Atoi(c.Query("limit", "100"))
	items, err := h.store.List(c.Context(), actor, action, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fiber.Map{"code": "list_failed", "message": err.Error()},
		})
	}
	return c.JSON(fiber.Map{"items": items})
}
