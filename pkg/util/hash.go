package util

import "golang.org/x/crypto/bcrypt"

// BcryptCost is the work factor for password hashing. The default (10) takes
// roughly 50-100ms on typical hardware; raise it deliberately, never inline.
const BcryptCost = bcrypt.DefaultCost

// HashPassword returns a bcrypt hash with a random per-password salt.
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CheckPasswordHash reports whether the password matches the bcrypt hash.
func CheckPasswordHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
