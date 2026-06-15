package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"
)

// Session is the decoded form of the auth cookie.
type Session struct {
	UserID    string
	ExpiresAt time.Time
}

// SignSession returns a signed cookie value for the session.
//
// Format: base64url(user_id|expires_unix) "." base64url(hmac_sha256(payload, secret))
func SignSession(secret []byte, s Session) string {
	payload := s.UserID + "|" + strconv.FormatInt(s.ExpiresAt.Unix(), 10)
	mac := hmacBase64(secret, []byte(payload))
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + mac
}

// ParseSession verifies and decodes a cookie value.
func ParseSession(secret []byte, value string) (Session, error) {
	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return Session{}, errors.New("session: bad format")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Session{}, errors.New("session: bad payload")
	}
	want := hmacBase64(secret, payload)
	if !hmac.Equal([]byte(parts[1]), []byte(want)) {
		return Session{}, errors.New("session: bad signature")
	}
	pieces := strings.SplitN(string(payload), "|", 2)
	if len(pieces) != 2 {
		return Session{}, errors.New("session: bad payload format")
	}
	exp, err := strconv.ParseInt(pieces[1], 10, 64)
	if err != nil {
		return Session{}, errors.New("session: bad expiry")
	}
	if time.Now().Unix() > exp {
		return Session{}, errors.New("session: expired")
	}
	return Session{UserID: pieces[0], ExpiresAt: time.Unix(exp, 0)}, nil
}

func hmacBase64(secret, payload []byte) string {
	m := hmac.New(sha256.New, secret)
	m.Write(payload)
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}
