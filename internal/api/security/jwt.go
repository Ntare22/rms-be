package security

import (
	"errors"
	"fmt"
	"time"
)

// ErrJWTNotImplemented indicates JWT helpers are not wired yet.
var ErrJWTNotImplemented = errors.New("jwt helper not implemented")

// TokenIssuer is the contract for issuing and parsing JWTs.
type TokenIssuer interface {
	IssueAccessToken(subject string, claims map[string]any) (token string, expiresAt time.Time, err error)
	IssueRefreshToken(subject string) (token string, expiresAt time.Time, err error)
	ParseAccessToken(token string) (subject string, claims map[string]any, err error)
	ParseRefreshToken(token string) (subject string, err error)
}

// PlaceholderTokenIssuer is a bootstrap stub; replace with HS256/RS256 implementation.
type PlaceholderTokenIssuer struct{}

// IssueAccessToken is not implemented.
func (PlaceholderTokenIssuer) IssueAccessToken(subject string, claims map[string]any) (string, time.Time, error) {
	_, _ = subject, claims
	return "", time.Time{}, fmt.Errorf("%w", ErrJWTNotImplemented)
}

// IssueRefreshToken is not implemented.
func (PlaceholderTokenIssuer) IssueRefreshToken(subject string) (string, time.Time, error) {
	_ = subject
	return "", time.Time{}, fmt.Errorf("%w", ErrJWTNotImplemented)
}

// ParseAccessToken is not implemented.
func (PlaceholderTokenIssuer) ParseAccessToken(token string) (string, map[string]any, error) {
	_ = token
	return "", nil, fmt.Errorf("%w", ErrJWTNotImplemented)
}

// ParseRefreshToken is not implemented.
func (PlaceholderTokenIssuer) ParseRefreshToken(token string) (string, error) {
	_ = token
	return "", fmt.Errorf("%w", ErrJWTNotImplemented)
}
