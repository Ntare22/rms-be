package security

import (
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// BcryptHasher implements PasswordHasher using bcrypt.
type BcryptHasher struct{}

// Hash returns a bcrypt hash of the plaintext password.
func (BcryptHasher) Hash(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Compare checks bcrypt hash against plaintext.
func (BcryptHasher) Compare(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
