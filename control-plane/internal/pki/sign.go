package pki

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"time"
)

// SignAgentCSR validates the PEM-encoded CSR and signs it as a client cert. The cert
// CommonName is the server_id, which is how we identify the agent in AgentStream.
// Returns the PEM cert and its SHA-256 fingerprint (hex) for storage on the server row.
func (b *Bundle) SignAgentCSR(csrPEM []byte, serverID string) (certPEM []byte, fingerprint string, err error) {
	block, _ := pem.Decode(csrPEM)
	if block == nil {
		return nil, "", errors.New("csr: pem decode failed")
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return nil, "", err
	}
	if err := csr.CheckSignature(); err != nil {
		return nil, "", err
	}

	tmpl := &x509.Certificate{
		SerialNumber: mustSerial(),
		Subject:      csr.Subject,
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	// Force the CN to the server_id so authentication is unambiguous.
	tmpl.Subject.CommonName = serverID

	der, err := x509.CreateCertificate(rand.Reader, tmpl, b.CAX509, csr.PublicKey, b.CAKey)
	if err != nil {
		return nil, "", err
	}
	out := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	sum := sha256.Sum256(der)
	return out, hex.EncodeToString(sum[:]), nil
}

// FingerprintDER returns the SHA-256 fingerprint (hex) of a DER-encoded cert. Used by
// AgentStream auth on the incoming peer cert.
func FingerprintDER(der []byte) string {
	sum := sha256.Sum256(der)
	return hex.EncodeToString(sum[:])
}
