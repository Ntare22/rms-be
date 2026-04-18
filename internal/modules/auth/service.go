package auth

import (
	"context"
	stderrors "errors"
	"strings"

	"gorm.io/gorm"

	apierrors "rms-be/internal/api/errors"
	"rms-be/internal/api/security"
	"rms-be/internal/modules/organizations"
	"rms-be/internal/modules/users"
)

// Service contains authentication business logic.
type Service struct {
	repo   *Repository
	pass   security.PasswordHasher
	tokens security.TokenIssuer
}

// NewService constructs a Service.
func NewService(repo *Repository, pass security.PasswordHasher, tokens security.TokenIssuer) *Service {
	return &Service{repo: repo, pass: pass, tokens: tokens}
}

// Register creates an organization and first user, then returns a session with tokens.
func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*AuthSessionResponse, error) {
	slug := normalizeSlug(req.OrganizationSlug)
	if slug == "" {
		return nil, apierrors.ErrValidation
	}
	if _, err := s.repo.GetOrganizationBySlug(ctx, slug); err == nil {
		return nil, apierrors.ErrConflict
	} else if !stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}

	role := users.UserRole(strings.TrimSpace(strings.ToLower(req.Role)))
	if role == "" {
		role = users.UserRoleAdmin
	}
	if !isBootstrapRole(role) {
		return nil, apierrors.ErrValidation
	}

	hash, err := s.pass.Hash(req.Password)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}

	org := &organizations.Organization{
		Name: strings.TrimSpace(req.OrganizationName),
		Slug: slug,
	}
	u := &users.User{
		Email:        strings.TrimSpace(strings.ToLower(req.Email)),
		PasswordHash: hash,
		Role:         role,
		Status:       users.UserStatusActive,
		FirstName:    strings.TrimSpace(req.FirstName),
		LastName:     strings.TrimSpace(req.LastName),
	}

	if err := s.repo.CreateOrganizationAndUser(ctx, org, u); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrConflict)
	}

	return s.buildSession(u)
}

// Login authenticates a user within an organization identified by id or slug.
func (s *Service) Login(ctx context.Context, req *LoginRequest) (*AuthSessionResponse, error) {
	org, err := s.resolveOrganization(ctx, req.OrganizationID, req.OrganizationSlug)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrInvalidCredentials
		}
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}

	u, err := s.repo.GetUserByEmailAndOrganization(ctx, req.Email, org.ID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrInvalidCredentials
		}
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}

	if err := s.pass.Compare(u.PasswordHash, req.Password); err != nil {
		return nil, apierrors.ErrInvalidCredentials
	}
	if u.Status != users.UserStatusActive {
		return nil, apierrors.ErrInvalidCredentials
	}

	if !isLoginRole(u.Role) {
		return nil, apierrors.ErrForbidden
	}

	return s.buildSession(u)
}

// Refresh rotates the refresh token and issues a new access token.
func (s *Service) Refresh(ctx context.Context, req *RefreshRequest) (*RefreshResponse, error) {
	userID, err := s.tokens.ParseRefreshToken(strings.TrimSpace(req.RefreshToken))
	if err != nil {
		return nil, apierrors.ErrUnauthorized
	}
	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrUnauthorized
		}
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}

	if !isLoginRole(u.Role) {
		return nil, apierrors.ErrForbidden
	}
	if u.Status != users.UserStatusActive {
		return nil, apierrors.ErrUnauthorized
	}

	pair, err := IssueTokenPair(s.tokens, u.ID, u.OrganizationID, string(u.Role))
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	return &RefreshResponse{
		AccessToken:      pair.AccessToken,
		RefreshToken:     pair.RefreshToken,
		AccessExpiresAt:  pair.AccessExpiresAt,
		RefreshExpiresAt: pair.RefreshExpiresAt,
	}, nil
}

// Me loads the current user profile for the authenticated subject.
func (s *Service) Me(ctx context.Context, userID, claimsOrgID, claimsRole string) (*MeResponse, error) {
	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrNotFound
		}
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if !strings.EqualFold(u.OrganizationID, claimsOrgID) {
		return nil, apierrors.ErrForbidden
	}
	if u.Status != users.UserStatusActive {
		return nil, apierrors.ErrForbidden
	}
	return &MeResponse{
		User:                 toUserSummary(u),
		ClaimsOrganizationID: claimsOrgID,
		ClaimsRole:           strings.ToLower(claimsRole),
	}, nil
}

func (s *Service) buildSession(u *users.User) (*AuthSessionResponse, error) {
	pair, err := IssueTokenPair(s.tokens, u.ID, u.OrganizationID, string(u.Role))
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	return &AuthSessionResponse{
		User:             toUserSummary(u),
		AccessToken:      pair.AccessToken,
		RefreshToken:     pair.RefreshToken,
		AccessExpiresAt:  pair.AccessExpiresAt,
		RefreshExpiresAt: pair.RefreshExpiresAt,
	}, nil
}

func (s *Service) resolveOrganization(ctx context.Context, id, slug string) (*organizations.Organization, error) {
	id = strings.TrimSpace(id)
	slug = strings.TrimSpace(slug)
	if id != "" {
		return s.repo.GetOrganizationByID(ctx, id)
	}
	if slug != "" {
		return s.repo.GetOrganizationBySlug(ctx, slug)
	}
	return nil, gorm.ErrRecordNotFound
}

func normalizeSlug(slug string) string {
	return strings.ToLower(strings.TrimSpace(slug))
}

func isBootstrapRole(r users.UserRole) bool {
	switch r {
	case users.UserRoleAdmin, users.UserRoleLandlord, users.UserRoleManager:
		return true
	default:
		return false
	}
}

func isLoginRole(r users.UserRole) bool {
	switch r {
	case users.UserRoleAdmin, users.UserRoleLandlord, users.UserRoleManager, users.UserRoleStaff:
		return true
	default:
		return false
	}
}

func toUserSummary(u *users.User) UserSummary {
	return UserSummary{
		ID:             u.ID,
		OrganizationID: u.OrganizationID,
		Email:          u.Email,
		FirstName:      u.FirstName,
		LastName:       u.LastName,
		Role:           string(u.Role),
	}
}
