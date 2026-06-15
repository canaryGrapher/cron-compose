package cryptobox

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"testing"
)

func newBox(t *testing.T) *Box {
	t.Helper()
	k := make([]byte, 32)
	_, _ = rand.Read(k)
	b, err := New(hex.EncodeToString(k))
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	return b
}

func TestSealOpenRoundTrip(t *testing.T) {
	b := newBox(t)
	for _, s := range []string{"", "hello", "a longer secret with bytes!"} {
		blob, err := b.Seal([]byte(s))
		if err != nil {
			t.Fatalf("seal: %v", err)
		}
		got, err := b.Open(blob)
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		if !bytes.Equal(got, []byte(s)) {
			t.Errorf("roundtrip: got %q, want %q", got, s)
		}
	}
}

func TestOpenRejectsTampered(t *testing.T) {
	b := newBox(t)
	blob, _ := b.Seal([]byte("payload"))
	blob[len(blob)-1] ^= 0xFF
	if _, err := b.Open(blob); err == nil {
		t.Fatal("expected GCM auth failure on tampered blob")
	}
}

func TestRejectsBadKey(t *testing.T) {
	if _, err := New(""); err == nil {
		t.Error("empty key should fail")
	}
	if _, err := New("zzzz"); err == nil {
		t.Error("non-hex should fail")
	}
	if _, err := New(hex.EncodeToString(make([]byte, 16))); err == nil {
		t.Error("16-byte key should fail (we require 32)")
	}
}
