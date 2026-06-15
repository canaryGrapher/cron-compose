package auth

import (
	"testing"
	"time"
)

func TestSignAndParseRoundTrip(t *testing.T) {
	secret := []byte("this-is-a-long-enough-secret")
	in := Session{UserID: "user-abc", ExpiresAt: time.Now().Add(time.Hour).Truncate(time.Second)}

	val := SignSession(secret, in)
	out, err := ParseSession(secret, val)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if out.UserID != in.UserID {
		t.Errorf("user_id: got %q, want %q", out.UserID, in.UserID)
	}
	if !out.ExpiresAt.Equal(in.ExpiresAt) {
		t.Errorf("expires_at: got %v, want %v", out.ExpiresAt, in.ExpiresAt)
	}
}

func TestParseRejectsWrongSecret(t *testing.T) {
	val := SignSession([]byte("secret-a-which-is-long-enough"),
		Session{UserID: "u1", ExpiresAt: time.Now().Add(time.Hour)})
	if _, err := ParseSession([]byte("secret-b-which-is-long-enough"), val); err == nil {
		t.Fatal("expected mismatched-secret error")
	}
}

func TestParseRejectsExpired(t *testing.T) {
	secret := []byte("this-is-a-long-enough-secret")
	val := SignSession(secret, Session{UserID: "u1", ExpiresAt: time.Now().Add(-time.Hour)})
	if _, err := ParseSession(secret, val); err == nil {
		t.Fatal("expected expired error")
	}
}

func TestParseRejectsTampered(t *testing.T) {
	secret := []byte("this-is-a-long-enough-secret")
	val := SignSession(secret, Session{UserID: "u1", ExpiresAt: time.Now().Add(time.Hour)})
	tampered := val[:len(val)-2] + "AA"
	if _, err := ParseSession(secret, tampered); err == nil {
		t.Fatal("expected tampered error")
	}
}
