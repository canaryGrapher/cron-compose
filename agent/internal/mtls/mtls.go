// Package mtls owns the agent's TLS material: ed25519 keypair, CSR generation, and
// loading the signed cert + CA back into a tls.Config for the gRPC dialer.
//
// Files in DATA_DIR/tls/:
//
//	agent.key   ed25519 private key (PKCS#8 PEM)
//	agent.crt   signed client cert from the control plane (PEM)
//	ca.crt      control-plane root CA the agent should pin (PEM)
package mtls

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// CSRResult is what the enroll subcommand uses after generating a fresh keypair.
type CSRResult struct {
	CSRPEM []byte
	keyDER []byte // PKCS#8, persisted only after the cert is saved
}

// GenerateCSR makes a fresh ed25519 keypair and returns a PEM-encoded CSR for it.
// The private key is held in memory until SaveBundle stores it alongside the cert.
func GenerateCSR(commonName string) (*CSRResult, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	tmpl := &x509.CertificateRequest{
		Subject: pkix.Name{CommonName: commonName},
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, tmpl, priv)
	if err != nil {
		return nil, err
	}
	csrPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})
	keyDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, err
	}
	_ = pub
	return &CSRResult{CSRPEM: csrPEM, keyDER: keyDER}, nil
}

// SaveBundle writes key, signed cert, and CA into dir/tls.
func (c *CSRResult) SaveBundle(dir string, certPEM, caPEM []byte) error {
	tlsDir := filepath.Join(dir, "tls")
	if err := os.MkdirAll(tlsDir, 0o700); err != nil {
		return err
	}
	if err := writePEM(filepath.Join(tlsDir, "agent.key"), "PRIVATE KEY", c.keyDER, 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(tlsDir, "agent.crt"), certPEM, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(tlsDir, "ca.crt"), caPEM, 0o644); err != nil {
		return err
	}
	return nil
}

// LoadConfig reads the bundle from dir/tls and returns a tls.Config ready for the
// gRPC dialer.
func LoadConfig(dir, serverName string) (*tls.Config, error) {
	tlsDir := filepath.Join(dir, "tls")
	cert, err := tls.LoadX509KeyPair(filepath.Join(tlsDir, "agent.crt"), filepath.Join(tlsDir, "agent.key"))
	if err != nil {
		return nil, fmt.Errorf("load agent keypair: %w", err)
	}
	caPEM, err := os.ReadFile(filepath.Join(tlsDir, "ca.crt"))
	if err != nil {
		return nil, fmt.Errorf("read ca.crt: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, errors.New("ca.crt: append failed")
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
		ServerName:   serverName,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

func writePEM(path, blockType string, der []byte, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: blockType, Bytes: der})
}
