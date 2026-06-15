package auth

import "testing"

func TestPasswordHashVerify(t *testing.T) {
	h, err := Hash("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !Verify(h, "correct-horse-battery-staple") {
		t.Error("Verify should accept the correct password")
	}
	if Verify(h, "wrong") {
		t.Error("Verify should reject the wrong password")
	}
	if Verify("not-a-hash", "anything") {
		t.Error("Verify should reject a non-hash")
	}
}
