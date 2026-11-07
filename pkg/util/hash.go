package util

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashPassword returns a hex-encoded SHA-256 hash.
// NOTE: simple hash for demo - use bcrypt or argon2 in production.
func HashPassword(password string) (string, error) {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:]), nil
}

func CheckPasswordHash(password, hash string) bool {
	h, err := HashPassword(password)
	if err != nil {
		return false
	}
	return h == hash
}
