package auth

import (
	"fmt"
	"strings"

	"rms-be/internal/api/security"
)

// IssueTokenPair creates access (with org + role claims) and refresh tokens for a user.
func IssueTokenPair(tokens security.TokenIssuer, userID, organizationID, role string) (TokenPair, error) {
	if userID == "" || organizationID == "" || role == "" {
		return TokenPair{}, fmt.Errorf("missing identity fields for token issue")
	}
	role = strings.ToLower(strings.TrimSpace(role))
	access, accessExp, err := tokens.IssueAccessToken(userID, map[string]any{
		"org_id": organizationID,
		"role":   role,
	})
	if err != nil {
		return TokenPair{}, err
	}
	refresh, refreshExp, err := tokens.IssueRefreshToken(userID)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{
		AccessToken:      access,
		RefreshToken:     refresh,
		AccessExpiresAt:  accessExp,
		RefreshExpiresAt: refreshExp,
	}, nil
}
