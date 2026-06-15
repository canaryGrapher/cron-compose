// Package auth owns user identity: password hashing, signed session cookies, the
// /auth/* REST handlers, and RBAC middleware.
package auth

import "golang.org/x/crypto/bcrypt"

// Hash returns a bcrypt hash for the password.
func Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Verify reports whether password matches the stored hash.
func Verify(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
