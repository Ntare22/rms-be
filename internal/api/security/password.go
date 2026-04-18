package security

import (
	"errors"
	"fmt"
)

// ErrPasswordNotImplemented indicates the password helper is not wired yet.
var ErrPasswordNotImplemented = errors.New("password helper not implemented")

// PasswordHasher is the contract for password hashing (bcrypt/argon2, etc.).
type PasswordHasher interface {
	Hash(plain string) (string, error)
	Compare(hash, plain string) error
}

// PlaceholderPasswordHasher is a bootstrap stub; replace with a real hasher in production.
type PlaceholderPasswordHasher struct{}

// Hash always returns ErrPasswordNotImplemented.
func (PlaceholderPasswordHasher) Hash(plain string) (string, error) {
	_ = plain
	return "", fmt.Errorf("%w", ErrPasswordNotImplemented)
}

// Compare always returns ErrPasswordNotImplemented.
func (PlaceholderPasswordHasher) Compare(hash, plain string) error {
	_, _ = hash, plain
	return fmt.Errorf("%w", ErrPasswordNotImplemented)
}
