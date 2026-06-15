// Package cryptobox is the control plane's symmetric-encryption primitive for
// secret values. AES-256-GCM with a single master key sourced from env. Production
// deployments swap this for envelope encryption against a KMS; the Box interface
// stays the same.
package cryptobox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
)

// Box encrypts and decrypts blobs with a fixed master key.
type Box struct {
	aead cipher.AEAD
}

// New parses a hex-encoded 32-byte master key and returns a Box.
func New(masterKeyHex string) (*Box, error) {
	if masterKeyHex == "" {
		return nil, errors.New("cryptobox: SECRETS_MASTER_KEY is required")
	}
	raw, err := hex.DecodeString(masterKeyHex)
	if err != nil {
		return nil, fmt.Errorf("cryptobox: master key not hex: %w", err)
	}
	if len(raw) != 32 {
		return nil, fmt.Errorf("cryptobox: master key must be 32 bytes (got %d)", len(raw))
	}
	block, err := aes.NewCipher(raw)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Box{aead: aead}, nil
}

// Seal returns nonce || ciphertext || tag.
func (b *Box) Seal(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	ct := b.aead.Seal(nil, nonce, plaintext, nil)
	return append(nonce, ct...), nil
}

// Open reverses Seal.
func (b *Box) Open(blob []byte) ([]byte, error) {
	ns := b.aead.NonceSize()
	if len(blob) < ns {
		return nil, errors.New("cryptobox: blob too short")
	}
	nonce, ct := blob[:ns], blob[ns:]
	return b.aead.Open(nil, nonce, ct, nil)
}
