package servers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/croncompose/croncompose/control-plane/internal/ids"
)

// EnrollmentStore handles enrollment_tokens: one-time secrets used by an agent
// during its initial Enroll RPC, stored only as a hash.
type EnrollmentStore struct {
	pool *pgxpool.Pool
}

func NewEnrollmentStore(pool *pgxpool.Pool) *EnrollmentStore {
	return &EnrollmentStore{pool: pool}
}

// Issue creates a fresh enrollment token bound to a server. The plaintext token is
// returned and shown to the operator once; only its hash is persisted.
func (s *EnrollmentStore) Issue(ctx context.Context, serverID string, ttl time.Duration) (token string, expiresAt time.Time, err error) {
	plaintext, err := randomToken(32)
	if err != nil {
		return "", time.Time{}, err
	}
	hash := hashToken(plaintext)
	expiresAt = time.Now().Add(ttl).UTC()
	_, err = s.pool.Exec(ctx, `
		insert into enrollment_tokens (id, token_hash, server_id, expires_at, created_at)
		values ($1, $2, $3, $4, now())
	`, ids.New(), hash, serverID, expiresAt)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("insert enrollment token: %w", err)
	}
	return plaintext, expiresAt, nil
}

// hashToken is exported for use by the agent gateway when validating Enroll RPCs.
func hashToken(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
