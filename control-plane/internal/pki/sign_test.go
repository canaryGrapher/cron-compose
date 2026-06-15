package pki

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"testing"
)

// TestSignAgentCSR roundtrips: bootstrap a CA, generate a CSR, sign it, parse the
// result, and check the signature against the CA.
func TestSignAgentCSR(t *testing.T) {
	dir := t.TempDir()
	bundle, err := LoadOrCreate(dir, []string{"localhost"})
	if err != nil {
		t.Fatalf("LoadOrCreate: %v", err)
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader,
		&x509.CertificateRequest{Subject: pkix.Name{CommonName: "agent-host"}},
		priv)
	if err != nil {
		t.Fatalf("create csr: %v", err)
	}
	csrPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})

	certPEM, fp, err := bundle.SignAgentCSR(csrPEM, "server-123")
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if len(fp) != 64 {
		t.Errorf("fingerprint length: got %d, want 64 (hex sha256)", len(fp))
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatal("decode cert pem failed")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse cert: %v", err)
	}
	if cert.Subject.CommonName != "server-123" {
		t.Errorf("CN: got %q, want %q (handler must override)", cert.Subject.CommonName, "server-123")
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(bundle.CACertPEM)
	if _, err := cert.Verify(x509.VerifyOptions{Roots: pool, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}); err != nil {
		t.Fatalf("verify against CA: %v", err)
	}
	if !cert.PublicKey.(ed25519.PublicKey).Equal(pub) {
		t.Error("public key in cert does not match CSR")
	}
}
