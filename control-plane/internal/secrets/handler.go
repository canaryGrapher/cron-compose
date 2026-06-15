package secrets

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/croncompose/croncompose/control-plane/internal/audit"
	"github.com/croncompose/croncompose/control-plane/internal/auth"
	"github.com/croncompose/croncompose/control-plane/internal/cryptobox"
)

type handler struct {
	log   *slog.Logger
	store *Store
	audit audit.Writer
}

// Register attaches /secrets routes. Admin or higher required.
func Register(r fiber.Router, log *slog.Logger, pool *pgxpool.Pool, box *cryptobox.Box, writer audit.Writer) {
	h := &handler{log: log, store: NewStore(pool, box), audit: writer}
	admin := auth.RequireRole("admin")
	r.Get("/secrets", admin, h.list)
	r.Post("/secrets", admin, h.create)
	r.Delete("/secrets/:id", admin, h.delete)
}

func (h *handler) list(c fiber.Ctx) error {
	items, err := h.store.List(c.Context())
	if err != nil {
		return jsonError(c, fiber.StatusInternalServerError, "list_failed", err)
	}
	return c.JSON(fiber.Map{"items": items})
}

func (h *handler) create(c fiber.Ctx) error {
	var in CreateInput
	if err := c.Bind().Body(&in); err != nil {
		return jsonError(c, fiber.StatusBadRequest, "bad_request", err)
	}
	if in.Name == "" || in.Value == "" {
		return jsonError(c, fiber.StatusBadRequest, "missing_fields",
			errors.New("name and value are required"))
	}
	s, err := h.store.Insert(c.Context(), in)
	if err != nil {
		return jsonError(c, fiber.StatusInternalServerError, "insert_failed", err)
	}
	h.audit.Write(c.Context(), auth.CurrentUserID(c), "secret.create", "secret", s.ID, map[string]any{"name": s.Name, "scope": s.Scope})
	return c.Status(fiber.StatusCreated).JSON(s)
}

func (h *handler) delete(c fiber.Ctx) error {
	id := c.Params("id")
	if err := h.store.Delete(c.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			return jsonError(c, fiber.StatusNotFound, "not_found", err)
		}
		return jsonError(c, fiber.StatusInternalServerError, "delete_failed", err)
	}
	h.audit.Write(c.Context(), auth.CurrentUserID(c), "secret.delete", "secret", id, nil)
	return c.SendStatus(fiber.StatusNoContent)
}

func jsonError(c fiber.Ctx, status int, code string, err error) error {
	return c.Status(status).JSON(fiber.Map{
		"error": fiber.Map{"code": code, "message": err.Error()},
	})
}
