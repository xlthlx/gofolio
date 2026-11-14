package util

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPasswordProducesValidBcrypt(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{name: "simple password", password: "hunter2"},
		{name: "empty password", password: ""},
		{name: "unicode password", password: "pässwörd🔐"},
		{name: "long password", password: "a very long password with spaces and punctuation!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			if err != nil {
				t.Fatalf("HashPassword(%q) returned error: %v", tt.password, err)
			}
			if !strings.HasPrefix(hash, "$2") {
				t.Errorf("hash = %q, want a bcrypt-formatted string ($2...)", hash)
			}
			if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(tt.password)); err != nil {
				t.Errorf("hash does not verify against its own password: %v", err)
			}
		})
	}
}

func TestHashPasswordIsSalted(t *testing.T) {
	// Two hashes of the same password must differ (random salt) yet both verify.
	h1, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("first HashPassword: %v", err)
	}
	h2, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("second HashPassword: %v", err)
	}
	if h1 == h2 {
		t.Error("two hashes of the same password are identical; salt is missing")
	}
	if !CheckPasswordHash("same-password", h1) || !CheckPasswordHash("same-password", h2) {
		t.Error("at least one of the two distinct hashes fails to verify")
	}
}

func TestCheckPasswordHash(t *testing.T) {
	correctHash, err := HashPassword("s3cret")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
	}{
		{name: "correct password", password: "s3cret", hash: correctHash, want: true},
		{name: "wrong password", password: "wrong", hash: correctHash, want: false},
		{name: "case differs", password: "S3CRET", hash: correctHash, want: false},
		{name: "empty password vs hash", password: "", hash: correctHash, want: false},
		{name: "malformed hash", password: "s3cret", hash: "not-a-hash", want: false},
		{name: "legacy sha256 hex hash", password: "password", hash: "ef92b778bafe771e89245b89ecbc08a44a4e166c06659911881f383d4473e94f", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckPasswordHash(tt.password, tt.hash); got != tt.want {
				t.Errorf("CheckPasswordHash(%q, %q) = %v, want %v", tt.password, tt.hash, got, tt.want)
			}
		})
	}
}

func TestBcryptCostIsSane(t *testing.T) {
	// Guard against someone lowering the cost to make tests faster.
	if BcryptCost < 10 {
		t.Errorf("BcryptCost = %d, want >= 10", BcryptCost)
	}
}
