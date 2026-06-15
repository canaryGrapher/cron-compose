package auth

import (
	"errors"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
)

type handler struct {
	log    *slog.Logger
	store  *Store
	secret []byte
	ttl    time.Duration
}

// Register attaches /auth/login, /auth/logout, /me.
func Register(r fiber.Router, log *slog.Logger, store *Store, secret []byte) {
	h := &handler{log: log, store: store, secret: secret, ttl: 7 * 24 * time.Hour}
	r.Post("/auth/login", h.login)
	r.Post("/auth/logout", h.logout)
	r.Get("/me", h.me)
}

type loginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *handler) login(c fiber.Ctx) error {
	var in loginInput
	if err := c.Bind().Body(&in); err != nil {
		return badRequest(c, "bad_request", err)
	}
	if in.Email == "" || in.Password == "" {
		return badRequest(c, "missing_fields", errors.New("email and password are required"))
	}
	u, hash, err := h.store.GetByEmailWithHash(c.Context(), in.Email)
	if err != nil || hash == "" || !Verify(hash, in.Password) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": fiber.Map{"code": "invalid_credentials", "message": "wrong email or password"},
		})
	}
	exp := time.Now().Add(h.ttl)
	value := SignSession(h.secret, Session{UserID: u.ID, ExpiresAt: exp})
	c.Cookie(&fiber.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		Expires:  exp,
		HTTPOnly: true,
		Secure:   false, // dev only; set true behind TLS
		SameSite: "Lax",
	})
	return c.JSON(u)
}

func (h *handler) logout(c fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HTTPOnly: true,
	})
	return c.SendStatus(fiber.StatusNoContent)
}

// me is mounted under the authenticated group so a 200 here doubles as a quick session
// check from the UI.
func (h *handler) me(c fiber.Ctx) error {
	u, err := h.store.GetByID(c.Context(), CurrentUserID(c))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": fiber.Map{"code": "unauthenticated", "message": err.Error()},
		})
	}
	return c.JSON(u)
}

func badRequest(c fiber.Ctx, code string, err error) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": fiber.Map{"code": code, "message": err.Error()},
	})
}
