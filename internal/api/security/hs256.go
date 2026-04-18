package security

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"rms-be/internal/config"
)

// HS256TokenIssuer issues and validates HS256 access and refresh tokens.
type HS256TokenIssuer struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
	issuer        string
}

// NewHS256TokenIssuer builds an issuer from application config.
func NewHS256TokenIssuer(cfg *config.Config) *HS256TokenIssuer {
	return &HS256TokenIssuer{
		accessSecret:  []byte(cfg.JWTSecret),
		refreshSecret: []byte(cfg.JWTRefreshSecret),
		accessTTL:     cfg.JWTAccessTTL,
		refreshTTL:    cfg.JWTRefreshTTL,
		issuer:        "rms-be",
	}
}

type accessClaims struct {
	OrgID string `json:"org_id"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

// IssueAccessToken signs an access JWT. claims map may include org_id and role (strings).
func (s *HS256TokenIssuer) IssueAccessToken(subject string, claims map[string]any) (string, time.Time, error) {
	if subject == "" {
		return "", time.Time{}, fmt.Errorf("subject required")
	}
	orgID, _ := claims["org_id"].(string)
	role, _ := claims["role"].(string)
	now := time.Now().UTC()
	exp := now.Add(s.accessTTL)
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims{
		OrgID: orgID,
		Role:  role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   subject,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	})
	signed, err := tok.SignedString(s.accessSecret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, exp, nil
}

// IssueRefreshToken signs a refresh JWT (minimal claims).
func (s *HS256TokenIssuer) IssueRefreshToken(subject string) (string, time.Time, error) {
	if subject == "" {
		return "", time.Time{}, fmt.Errorf("subject required")
	}
	now := time.Now().UTC()
	exp := now.Add(s.refreshTTL)
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    s.issuer,
		Subject:   subject,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(exp),
	})
	signed, err := tok.SignedString(s.refreshSecret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, exp, nil
}

// ParseAccessToken validates an access token and returns the subject plus flattened claims.
func (s *HS256TokenIssuer) ParseAccessToken(tokenString string) (string, map[string]any, error) {
	tok, err := jwt.ParseWithClaims(tokenString, &accessClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.accessSecret, nil
	})
	if err != nil {
		return "", nil, fmt.Errorf("invalid access token: %w", err)
	}
	if !tok.Valid {
		return "", nil, fmt.Errorf("invalid access token")
	}
	ac, ok := tok.Claims.(*accessClaims)
	if !ok {
		return "", nil, fmt.Errorf("invalid access token claims")
	}
	sub := ac.Subject
	if sub == "" {
		return "", nil, fmt.Errorf("missing subject")
	}
	out := map[string]any{
		"org_id": ac.OrgID,
		"role":   ac.Role,
	}
	return sub, out, nil
}

// ParseRefreshToken validates a refresh JWT and returns the subject (user id).
func (s *HS256TokenIssuer) ParseRefreshToken(tokenString string) (string, error) {
	tok, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.refreshSecret, nil
	})
	if err != nil {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}
	if !tok.Valid {
		return "", fmt.Errorf("invalid refresh token")
	}
	rc, ok := tok.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return "", fmt.Errorf("invalid refresh token claims")
	}
	if rc.Subject == "" {
		return "", fmt.Errorf("missing subject")
	}
	return rc.Subject, nil
}
