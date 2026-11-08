package util

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestHashPasswordIsDeterministic(t *testing.T) {
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
			got1, err := HashPassword(tt.password)
			if err != nil {
				t.Fatalf("HashPassword(%q) returned error: %v", tt.password, err)
			}
			got2, err := HashPassword(tt.password)
			if err != nil {
				t.Fatalf("HashPassword(%q) second call returned error: %v", tt.password, err)
			}
			if got1 != got2 {
				t.Errorf("HashPassword not deterministic: %q != %q", got1, got2)
			}
		})
	}
}

func TestHashPasswordMatchesSHA256(t *testing.T) {
	password := "correct horse battery staple"
	want := sha256.Sum256([]byte(password))

	got, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if got != hex.EncodeToString(want[:]) {
		t.Errorf("HashPassword = %q, want hex(SHA-256) %q", got, hex.EncodeToString(want[:]))
	}
}

func TestCheckPasswordHash(t *testing.T) {
	correctHash, err := HashPassword("s3cret")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckPasswordHash(tt.password, tt.hash); got != tt.want {
				t.Errorf("CheckPasswordHash(%q, %q) = %v, want %v", tt.password, tt.hash, got, tt.want)
			}
		})
	}
}
