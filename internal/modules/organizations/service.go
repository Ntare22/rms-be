package organizations

import (
	"context"
	stderrors "errors"
	"strings"

	"gorm.io/gorm"

	apierrors "rms-be/internal/api/errors"
	"rms-be/internal/middleware"
)

// Actor is the authenticated caller for authorization decisions.
type Actor struct {
	UserID         string
	OrganizationID string
	Role           string
}

// Service contains organization business logic.
type Service struct {
	repo *Repository
}

// NewService constructs a Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create registers a new organization (admin only). Sets audit CreatedBy.
func (s *Service) Create(ctx context.Context, actor Actor, req *CreateOrganizationRequest) (*OrganizationResponse, error) {
	if !isAdmin(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	slug := normalizeSlug(req.Slug)
	if slug == "" {
		return nil, apierrors.ErrValidation
	}
	if _, err := s.repo.GetBySlugCI(ctx, slug); err == nil {
		return nil, apierrors.ErrConflict
	} else if !stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}

	uid := strings.TrimSpace(actor.UserID)
	org := &Organization{
		Name:                strings.TrimSpace(req.Name),
		Slug:                slug,
		LegalName:           strings.TrimSpace(req.LegalName),
		BillingEmail:        strings.TrimSpace(strings.ToLower(req.BillingEmail)),
		Phone:               strings.TrimSpace(req.Phone),
		Website:             strings.TrimSpace(req.Website),
		AddressLine1:        strings.TrimSpace(req.AddressLine1),
		AddressLine2:        strings.TrimSpace(req.AddressLine2),
		City:                strings.TrimSpace(req.City),
		Region:              strings.TrimSpace(req.Region),
		PostalCode:          strings.TrimSpace(req.PostalCode),
		Country:             strings.ToUpper(strings.TrimSpace(req.Country)),
		DefaultCurrency:     strings.ToUpper(strings.TrimSpace(req.DefaultCurrency)),
		Timezone:            strings.TrimSpace(req.Timezone),
		TaxID:               strings.TrimSpace(req.TaxID),
		CompanyRegistration: strings.TrimSpace(req.CompanyRegistration),
	}
	if uid != "" {
		org.CreatedBy = &uid
		org.UpdatedBy = &uid
	}

	if err := s.repo.Create(ctx, org); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrConflict)
	}
	return toResponse(org), nil
}

// Get returns one organization if the actor may read it.
func (s *Service) Get(ctx context.Context, actor Actor, id string) (*OrganizationResponse, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, apierrors.ErrValidation
	}
	if err := s.assertSameOrganization(actor, id); err != nil {
		return nil, err
	}
	o, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrNotFound
		}
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	return toResponse(o), nil
}

// Patch updates profile fields. Slug cannot change. Admin may update any org; landlord and manager only their own. Staff cannot PATCH.
func (s *Service) Patch(ctx context.Context, actor Actor, id string, req *PatchOrganizationRequest) (*OrganizationResponse, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, apierrors.ErrValidation
	}
	if err := s.assertSameOrganization(actor, id); err != nil {
		return nil, err
	}
	if !canPatchOrganization(actor.Role) {
		return nil, apierrors.ErrForbidden
	}

	o, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrNotFound
		}
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}

	updates := patchUpdates(req)
	if v, ok := updates["name"].(string); ok && strings.TrimSpace(v) == "" {
		return nil, apierrors.ErrValidation
	}
	if len(updates) == 0 {
		return toResponse(o), nil
	}
	uid := strings.TrimSpace(actor.UserID)
	if uid != "" {
		updates["updated_by"] = uid
	}
	if err := s.repo.Update(ctx, id, updates); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	refreshed, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	return toResponse(refreshed), nil
}

func patchUpdates(req *PatchOrganizationRequest) map[string]any {
	out := make(map[string]any)
	if req.Name != nil {
		out["name"] = strings.TrimSpace(*req.Name)
	}
	if req.LegalName != nil {
		out["legal_name"] = strings.TrimSpace(*req.LegalName)
	}
	if req.BillingEmail != nil {
		out["billing_email"] = strings.TrimSpace(strings.ToLower(*req.BillingEmail))
	}
	if req.Phone != nil {
		out["phone"] = strings.TrimSpace(*req.Phone)
	}
	if req.Website != nil {
		out["website"] = strings.TrimSpace(*req.Website)
	}
	if req.AddressLine1 != nil {
		out["address_line1"] = strings.TrimSpace(*req.AddressLine1)
	}
	if req.AddressLine2 != nil {
		out["address_line2"] = strings.TrimSpace(*req.AddressLine2)
	}
	if req.City != nil {
		out["city"] = strings.TrimSpace(*req.City)
	}
	if req.Region != nil {
		out["region"] = strings.TrimSpace(*req.Region)
	}
	if req.PostalCode != nil {
		out["postal_code"] = strings.TrimSpace(*req.PostalCode)
	}
	if req.Country != nil {
		out["country"] = strings.ToUpper(strings.TrimSpace(*req.Country))
	}
	if req.DefaultCurrency != nil {
		out["default_currency"] = strings.ToUpper(strings.TrimSpace(*req.DefaultCurrency))
	}
	if req.Timezone != nil {
		out["timezone"] = strings.TrimSpace(*req.Timezone)
	}
	if req.TaxID != nil {
		out["tax_id"] = strings.TrimSpace(*req.TaxID)
	}
	if req.CompanyRegistration != nil {
		out["company_registration"] = strings.TrimSpace(*req.CompanyRegistration)
	}
	return out
}

func (s *Service) assertSameOrganization(actor Actor, organizationID string) error {
	if isAdmin(actor.Role) {
		return nil
	}
	if !strings.EqualFold(strings.TrimSpace(actor.OrganizationID), strings.TrimSpace(organizationID)) {
		return apierrors.ErrForbidden
	}
	return nil
}

func isAdmin(role string) bool {
	return strings.EqualFold(strings.TrimSpace(role), middleware.RoleAdmin)
}

func canPatchOrganization(role string) bool {
	r := strings.TrimSpace(strings.ToLower(role))
	switch r {
	case middleware.RoleAdmin, middleware.RoleLandlord, middleware.RoleManager:
		return true
	default:
		return false
	}
}

func normalizeSlug(slug string) string {
	return strings.ToLower(strings.TrimSpace(slug))
}

func toResponse(o *Organization) *OrganizationResponse {
	return &OrganizationResponse{
		ID:                  o.ID,
		Name:                o.Name,
		Slug:                o.Slug,
		LegalName:           o.LegalName,
		BillingEmail:        o.BillingEmail,
		Phone:               o.Phone,
		Website:             o.Website,
		AddressLine1:        o.AddressLine1,
		AddressLine2:        o.AddressLine2,
		City:                o.City,
		Region:              o.Region,
		PostalCode:          o.PostalCode,
		Country:             o.Country,
		DefaultCurrency:     o.DefaultCurrency,
		Timezone:            o.Timezone,
		TaxID:               o.TaxID,
		CompanyRegistration: o.CompanyRegistration,
		CreatedAt:           o.CreatedAt,
		UpdatedAt:           o.UpdatedAt,
		CreatedBy:           o.CreatedBy,
		UpdatedBy:           o.UpdatedBy,
	}
}
