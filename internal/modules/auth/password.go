package auth

import "rms-be/internal/api/security"

// PasswordHasher is an alias to the shared security contract.
type PasswordHasher = security.PasswordHasher

// DefaultPasswordHasher returns the production bcrypt implementation from internal/api/security.
func DefaultPasswordHasher() PasswordHasher {
	return security.BcryptHasher{}
}
