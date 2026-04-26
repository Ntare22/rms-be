package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	stderrors "errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"gorm.io/gorm"

	apierrors "rms-be/internal/api/errors"
	"rms-be/internal/api/security"
	"rms-be/internal/modules/notifications"
	"rms-be/internal/modules/organizations"
	"rms-be/internal/modules/users"
)

// Service contains authentication business logic.
type Service struct {
	repo          *Repository
	pass          security.PasswordHasher
	tokens        security.TokenIssuer
	mailer        notifications.EmailSender
	passwordSetup PasswordSetupConfig
}

type PasswordSetupConfig struct {
	BaseURL          string
	FromName         string
	From             string
	TTL              time.Duration
	InviteTemplateID int64
}

// NewService constructs a Service.
func NewService(repo *Repository, pass security.PasswordHasher, tokens security.TokenIssuer, mailer notifications.EmailSender, setup PasswordSetupConfig) *Service {
	if mailer == nil {
		mailer = notifications.NoopEmailSender{}
	}
	if setup.TTL <= 0 {
		setup.TTL = 24 * time.Hour
	}
	return &Service{repo: repo, pass: pass, tokens: tokens, mailer: mailer, passwordSetup: setup}
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

// StartPasswordSetup creates and emails a one-time password setup token for an existing org user.
func (s *Service) StartPasswordSetup(ctx context.Context, req *PasswordSetupRequest) error {
	orgID := strings.TrimSpace(req.OrganizationID)
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if orgID == "" || email == "" {
		return apierrors.ErrValidation
	}
	u, err := s.repo.GetUserByEmailInOrg(ctx, orgID, email)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if !isLoginRole(u.Role) {
		return nil
	}
	setupURL, err := s.createPasswordSetupTokenAndURL(ctx, u)
	if err != nil {
		return err
	}
	return s.sendPasswordSetupInvite(ctx, u, setupURL)
}

// ConfirmPasswordSetup consumes a one-time token and sets a fresh password.
func (s *Service) ConfirmPasswordSetup(ctx context.Context, req *PasswordSetupConfirmRequest) error {
	hash := hashToken(req.Token)
	rec, err := s.repo.GetPasswordSetupTokenByHash(ctx, hash)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return apierrors.ErrUnauthorized
		}
		return apierrors.Wrap(err, apierrors.ErrInternal)
	}
	now := time.Now().UTC()
	if rec.ExpiresAt.Before(now) {
		return apierrors.ErrUnauthorized
	}
	hashPwd, err := s.pass.Hash(req.Password)
	if err != nil {
		return apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if err := s.repo.UpdateUser(ctx, rec.OrganizationID, rec.UserID, map[string]any{
		"password_hash": hashPwd,
		"status":        users.UserStatusActive,
		"updated_at":    now,
	}); err != nil {
		return apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if err := s.repo.MarkPasswordSetupTokenUsed(ctx, rec.ID, now); err != nil {
		return apierrors.Wrap(err, apierrors.ErrInternal)
	}
	return nil
}

// ProvisionTenantInvite creates (or refreshes) a tenant login and sends invite email.
func (s *Service) ProvisionTenantInvite(ctx context.Context, organizationID, tenantID, email, firstName, lastName string) (*users.User, error) {
	orgID := strings.TrimSpace(organizationID)
	addr := strings.TrimSpace(strings.ToLower(email))
	if orgID == "" || strings.TrimSpace(tenantID) == "" || addr == "" {
		return nil, apierrors.ErrValidation
	}
	u, err := s.repo.GetUserByEmailInOrg(ctx, orgID, addr)
	if err != nil && !stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if err != nil && stderrors.Is(err, gorm.ErrRecordNotFound) {
		placeholder, errGen := randomToken(24)
		if errGen != nil {
			return nil, apierrors.Wrap(errGen, apierrors.ErrInternal)
		}
		hash, errHash := s.pass.Hash(placeholder)
		if errHash != nil {
			return nil, apierrors.Wrap(errHash, apierrors.ErrInternal)
		}
		u = &users.User{
			OrganizationID: orgID,
			Email:          addr,
			PasswordHash:   hash,
			Role:           users.UserRoleTenant,
			Status:         users.UserStatusInvited,
			FirstName:      strings.TrimSpace(firstName),
			LastName:       strings.TrimSpace(lastName),
			TenantID:       ptrString(strings.TrimSpace(tenantID)),
		}
		if errCreate := s.repo.CreateUser(ctx, u); errCreate != nil {
			return nil, apierrors.Wrap(errCreate, apierrors.ErrConflict)
		}
	}
	if u.Role != users.UserRoleTenant {
		return nil, apierrors.ErrConflict
	}
	if err := s.repo.UpdateUser(ctx, orgID, u.ID, map[string]any{
		"tenant_id": tenantID,
		"status":    users.UserStatusInvited,
	}); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	u.TenantID = ptrString(tenantID)
	u.Status = users.UserStatusInvited
	setupURL, err := s.createPasswordSetupTokenAndURL(ctx, u)
	if err != nil {
		return nil, err
	}
	if err := s.sendPasswordSetupInvite(ctx, u, setupURL); err != nil {
		return nil, err
	}
	return u, nil
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
	case users.UserRoleAdmin, users.UserRoleLandlord, users.UserRoleManager, users.UserRolePropertyManager, users.UserRoleAccountant, users.UserRoleStaff, users.UserRoleTenant:
		return true
	default:
		return false
	}
}

func (s *Service) createPasswordSetupTokenAndURL(ctx context.Context, u *users.User) (string, error) {
	raw, err := randomToken(32)
	if err != nil {
		return "", apierrors.Wrap(err, apierrors.ErrInternal)
	}
	now := time.Now().UTC()
	rec := &PasswordSetupToken{
		OrganizationID: u.OrganizationID,
		UserID:         u.ID,
		TokenHash:      hashToken(raw),
		ExpiresAt:      now.Add(s.passwordSetup.TTL),
	}
	if err := s.repo.CreatePasswordSetupToken(ctx, rec); err != nil {
		return "", apierrors.Wrap(err, apierrors.ErrInternal)
	}
	base := strings.TrimSuffix(strings.TrimSpace(s.passwordSetup.BaseURL), "/")
	if base == "" {
		base = "http://localhost:5173"
	}
	return fmt.Sprintf("%s/set-password?token=%s", base, url.QueryEscape(raw)), nil
}

func (s *Service) sendPasswordSetupInvite(ctx context.Context, u *users.User, setupURL string) error {
	from := strings.TrimSpace(s.passwordSetup.From)
	if from == "" {
		return apierrors.ErrNotImplemented
	}
	name := strings.TrimSpace(strings.TrimSpace(u.FirstName) + " " + strings.TrimSpace(u.LastName))
	vars := map[string]any{
		"first_name": nameOrFallback(u.FirstName, "there"),
		"setup_url":  setupURL,
	}
	msg := notifications.EmailMessage{
		FromName:         s.passwordSetup.FromName,
		FromEmail:        from,
		ToName:           strings.TrimSpace(name),
		ToEmail:          strings.TrimSpace(u.Email),
		Subject:          "Set your RMS password",
		HTMLBody:         "<p>Hello {{var:first_name}},</p><p>Set your password to access your tenant portal:</p><p><a href=\"{{var:setup_url}}\">Set password</a></p>",
		TextBody:         "Hello {{var:first_name}},\n\nSet your password to access your tenant portal:\n{{var:setup_url}}",
		TemplateID:       s.passwordSetup.InviteTemplateID,
		TemplateLanguage: true,
		Variables:        vars,
	}
	if err := s.mailer.Send(ctx, msg); err != nil {
		return apierrors.Wrap(err, apierrors.ErrInternal)
	}
	return nil
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
}

func ptrString(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	s := strings.TrimSpace(v)
	return &s
}

func nameOrFallback(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
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
