package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/croncompose/croncompose/control-plane/internal/ids"
)

// User mirrors the users table (no password hash exposed).
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// ErrNotFound is returned when a user lookup misses.
var ErrNotFound = errors.New("user not found")

// Store is the data-access layer for users.
type Store struct{ pool *pgxpool.Pool }

// NewStore wires a Store to a pgx pool.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// GetByID returns a user or ErrNotFound.
func (s *Store) GetByID(ctx context.Context, id string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		select id, email, name, role, created_at from users where id = $1
	`, id).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

// GetByEmailWithHash returns the user plus their password hash for login.
func (s *Store) GetByEmailWithHash(ctx context.Context, email string) (User, string, error) {
	var u User
	var hash string
	err := s.pool.QueryRow(ctx, `
		select id, email, name, role, created_at, coalesce(password_hash,'')
		from users where email = $1
	`, email).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.CreatedAt, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, "", ErrNotFound
	}
	return u, hash, err
}

// Upsert creates or updates a user (used by the seed admin flow).
func (s *Store) Upsert(ctx context.Context, email, name, role, passwordHash string) (User, error) {
	id := ids.New()
	_, err := s.pool.Exec(ctx, `
		insert into users (id, email, name, role, password_hash, created_at)
		values ($1, $2, $3, $4, $5, now())
		on conflict (email) do update set
		  name = excluded.name,
		  role = excluded.role,
		  password_hash = excluded.password_hash
	`, id, email, name, role, passwordHash)
	if err != nil {
		return User{}, err
	}
	u, _, err := s.GetByEmailWithHash(ctx, email)
	return u, err
}
