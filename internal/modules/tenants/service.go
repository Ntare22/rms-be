package tenants

import (
	"context"
	stderrors "errors"
	"strings"

	"gorm.io/gorm"

	apierrors "rms-be/internal/api/errors"
	"rms-be/internal/middleware"
	"rms-be/internal/modules/users"
)

// Actor is the authenticated subject for authorization.
type Actor struct {
	UserID         string
	OrganizationID string
	Role           string
}

// Service contains tenant business logic.
type Service struct {
	repo       *Repository
	onboarding tenantOnboarding
}

type tenantOnboarding interface {
	ProvisionTenantInvite(ctx context.Context, organizationID, tenantID, email, firstName, lastName string) (*users.User, error)
}

// NewService constructs a Service.
func NewService(repo *Repository, onboarding tenantOnboarding) *Service {
	return &Service{repo: repo, onboarding: onboarding}
}

// List returns paginated tenants with optional search filters.
func (s *Service) List(ctx context.Context, actor Actor, organizationID, nameQ, emailQ, phoneQ, statusFilter string, includeSummary bool, page, pageSize, offset int) (*TenantListResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return nil, apierrors.ErrValidation
	}
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canReadTenants(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	var st *TenantStatus
	if t := strings.TrimSpace(statusFilter); t != "" {
		v := parseTenantStatus(t)
		if v == "" {
			return nil, apierrors.ErrValidation
		}
		st = &v
	}
	rows, total, err := s.repo.List(ctx, organizationID, nameQ, emailQ, phoneQ, st, offset, pageSize)
	if isManagerRole(actor.Role) {
		ids, err := s.repo.ListManagerBuildingIDs(ctx, organizationID, actor.UserID)
		if err != nil {
			return nil, apierrors.Wrap(err, apierrors.ErrInternal)
		}
		rows, total, err = s.repo.ListByBuildings(ctx, organizationID, ids, nameQ, emailQ, phoneQ, st, offset, pageSize)
		if err != nil {
			return nil, apierrors.Wrap(err, apierrors.ErrInternal)
		}
	}
	if isTenantRole(actor.Role) {
		tenantID, err := s.repo.TenantIDByUser(ctx, organizationID, actor.UserID)
		if err != nil {
			return nil, apierrors.Wrap(err, apierrors.ErrInternal)
		}
		if strings.TrimSpace(tenantID) == "" {
			return &TenantListResponse{Items: []TenantResponse{}, Page: page, PageSize: pageSize, Total: 0}, nil
		}
		t, err := s.repo.GetByIDInOrg(ctx, organizationID, tenantID)
		if err != nil {
			return nil, apierrors.Wrap(err, apierrors.ErrInternal)
		}
		var counts map[string]int
		var units map[string]CurrentUnitSummary
		if includeSummary {
			counts, units, err = s.repo.ActiveLeaseStats(ctx, organizationID, []string{tenantID})
			if err != nil {
				return nil, apierrors.Wrap(err, apierrors.ErrInternal)
			}
		}
		r := toResponse(t, includeSummary, counts, units)
		return &TenantListResponse{Items: []TenantResponse{r}, Page: page, PageSize: pageSize, Total: 1}, nil
	}
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	var counts map[string]int
	var units map[string]CurrentUnitSummary
	if includeSummary && len(rows) > 0 {
		ids := make([]string, 0, len(rows))
		for i := range rows {
			ids = append(ids, rows[i].ID)
		}
		counts, units, err = s.repo.ActiveLeaseStats(ctx, organizationID, ids)
		if err != nil {
			return nil, apierrors.Wrap(err, apierrors.ErrInternal)
		}
	}
	out := make([]TenantResponse, 0, len(rows))
	for i := range rows {
		out = append(out, toResponse(&rows[i], includeSummary, counts, units))
	}
	return &TenantListResponse{Items: out, Page: page, PageSize: pageSize, Total: total}, nil
}

// Create adds a tenant.
func (s *Service) Create(ctx context.Context, actor Actor, organizationID string, req *CreateTenantRequest) (*TenantResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canMutateTenants(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	if isManagerRole(actor.Role) || isTenantRole(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	st := parseTenantStatus(req.Status)
	if st == "" {
		st = TenantStatusActive
	}
	fn := strings.TrimSpace(req.FirstName)
	ln := strings.TrimSpace(req.LastName)
	ff := computeFullName(fn, ln, req.FullName)
	uid := strings.TrimSpace(actor.UserID)
	t := &Tenant{
		OrganizationID: organizationID,
		FirstName:      fn,
		LastName:       ln,
		FullName:       ff,
		Email:          strings.TrimSpace(strings.ToLower(req.Email)),
		Phone:          strings.TrimSpace(req.Phone),
		EmailOptIn:     req.EmailOptIn,
		SmsOptIn:       req.SmsOptIn,
		SmsVerified:    req.SmsVerified,
		Locale:         strings.TrimSpace(req.Locale),
		Timezone:       strings.TrimSpace(req.Timezone),
		BillingDueDay:  req.BillingDueDay,
		BillingChannel: strings.TrimSpace(strings.ToLower(req.BillingChannel)),
		Status:         st,
	}
	if uid != "" {
		t.CreatedBy = &uid
		t.UpdatedBy = &uid
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if s.onboarding != nil && strings.TrimSpace(t.Email) != "" {
		u, err := s.onboarding.ProvisionTenantInvite(ctx, organizationID, t.ID, t.Email, t.FirstName, t.LastName)
		if err != nil {
			return nil, err
		}
		if u != nil && strings.TrimSpace(u.ID) != "" {
			_ = s.repo.Update(ctx, organizationID, t.ID, map[string]any{"user_id": u.ID})
			t.UserID = &u.ID
		}
	}
	r := toResponse(t, false, nil, nil)
	return &r, nil
}

// Get returns one tenant.
func (s *Service) Get(ctx context.Context, actor Actor, organizationID, tenantID string, includeSummary bool) (*TenantResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	tenantID = strings.TrimSpace(tenantID)
	if organizationID == "" || tenantID == "" {
		return nil, apierrors.ErrValidation
	}
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canReadTenants(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	t, err := s.repo.GetByIDInOrg(ctx, organizationID, tenantID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrNotFound
		}
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if isManagerRole(actor.Role) {
		ids, err := s.repo.ListManagerBuildingIDs(ctx, organizationID, actor.UserID)
		if err != nil {
			return nil, apierrors.Wrap(err, apierrors.ErrInternal)
		}
		t, err = s.repo.GetByIDInOrgAndBuildings(ctx, organizationID, tenantID, ids)
		if err != nil {
			if stderrors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apierrors.ErrForbidden
			}
			return nil, apierrors.Wrap(err, apierrors.ErrInternal)
		}
	}
	if isTenantRole(actor.Role) {
		selfID, err := s.repo.TenantIDByUser(ctx, organizationID, actor.UserID)
		if err != nil {
			return nil, apierrors.Wrap(err, apierrors.ErrInternal)
		}
		if !strings.EqualFold(strings.TrimSpace(selfID), tenantID) {
			return nil, apierrors.ErrForbidden
		}
	}
	var counts map[string]int
	var units map[string]CurrentUnitSummary
	if includeSummary {
		counts, units, err = s.repo.ActiveLeaseStats(ctx, organizationID, []string{tenantID})
		if err != nil {
			return nil, apierrors.Wrap(err, apierrors.ErrInternal)
		}
	}
	r := toResponse(t, includeSummary, counts, units)
	return &r, nil
}

// Patch updates a tenant.
func (s *Service) Patch(ctx context.Context, actor Actor, organizationID, tenantID string, req *PatchTenantRequest) (*TenantResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	tenantID = strings.TrimSpace(tenantID)
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canMutateTenants(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	if isManagerRole(actor.Role) || isTenantRole(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	t, err := s.repo.GetByIDInOrg(ctx, organizationID, tenantID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrNotFound
		}
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if req.Status != nil {
		if parseTenantStatus(strings.TrimSpace(*req.Status)) == "" && strings.TrimSpace(*req.Status) != "" {
			return nil, apierrors.ErrValidation
		}
	}
	if req.FullName != nil && strings.TrimSpace(*req.FullName) == "" {
		return nil, apierrors.ErrValidation
	}
	updates := patchUpdates(t, req)
	if v, ok := updates["first_name"].(string); ok && strings.TrimSpace(v) == "" {
		return nil, apierrors.ErrValidation
	}
	if len(updates) == 0 {
		r := toResponse(t, false, nil, nil)
		return &r, nil
	}
	uid := strings.TrimSpace(actor.UserID)
	if uid != "" {
		updates["updated_by"] = uid
	}
	if err := s.repo.Update(ctx, organizationID, tenantID, updates); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	refreshed, err := s.repo.GetByIDInOrg(ctx, organizationID, tenantID)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	r := toResponse(refreshed, false, nil, nil)
	return &r, nil
}

// Delete soft-deletes a tenant only when they have no leases (any status). Otherwise returns ErrTenantHasLeases — use PATCH `status` to deactivate.
func (s *Service) Delete(ctx context.Context, actor Actor, organizationID, tenantID string) error {
	organizationID = strings.TrimSpace(organizationID)
	tenantID = strings.TrimSpace(tenantID)
	if err := assertOrgScope(actor, organizationID); err != nil {
		return err
	}
	if !canMutateTenants(actor.Role) {
		return apierrors.ErrForbidden
	}
	if isManagerRole(actor.Role) || isTenantRole(actor.Role) {
		return apierrors.ErrForbidden
	}
	if _, err := s.repo.GetByIDInOrg(ctx, organizationID, tenantID); err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return apierrors.ErrNotFound
		}
		return apierrors.Wrap(err, apierrors.ErrInternal)
	}
	n, err := s.repo.CountLeasesForTenant(ctx, organizationID, tenantID)
	if err != nil {
		return apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if n > 0 {
		return apierrors.ErrTenantHasLeases
	}
	if err := s.repo.SoftDelete(ctx, organizationID, tenantID); err != nil {
		return apierrors.Wrap(err, apierrors.ErrInternal)
	}
	return nil
}

func patchUpdates(cur *Tenant, req *PatchTenantRequest) map[string]any {
	out := make(map[string]any)
	fn := cur.FirstName
	if req.FirstName != nil {
		fn = strings.TrimSpace(*req.FirstName)
		out["first_name"] = fn
	}
	ln := cur.LastName
	if req.LastName != nil {
		ln = strings.TrimSpace(*req.LastName)
		out["last_name"] = ln
	}
	if req.Email != nil {
		out["email"] = strings.TrimSpace(strings.ToLower(*req.Email))
	}
	if req.Phone != nil {
		out["phone"] = strings.TrimSpace(*req.Phone)
	}
	if req.EmailOptIn != nil {
		out["email_opt_in"] = *req.EmailOptIn
	}
	if req.SmsOptIn != nil {
		out["sms_opt_in"] = *req.SmsOptIn
	}
	if req.SmsVerified != nil {
		out["sms_verified"] = *req.SmsVerified
	}
	if req.Locale != nil {
		out["locale"] = strings.TrimSpace(*req.Locale)
	}
	if req.Timezone != nil {
		out["timezone"] = strings.TrimSpace(*req.Timezone)
	}
	if req.BillingDueDay != nil {
		out["billing_due_day"] = req.BillingDueDay
	}
	if req.BillingChannel != nil {
		out["billing_channel"] = strings.TrimSpace(strings.ToLower(*req.BillingChannel))
	}
	if req.Status != nil {
		if st := parseTenantStatus(strings.TrimSpace(*req.Status)); st != "" {
			out["status"] = string(st)
		}
	}
	if req.FullName != nil {
		out["full_name"] = strings.TrimSpace(*req.FullName)
	} else if req.FirstName != nil || req.LastName != nil {
		out["full_name"] = computeFullName(fn, ln, "")
	}
	return out
}

func computeFullName(first, last, explicitFull string) string {
	if strings.TrimSpace(explicitFull) != "" {
		return strings.TrimSpace(explicitFull)
	}
	return strings.TrimSpace(strings.TrimSpace(first) + " " + strings.TrimSpace(last))
}

func assertOrgScope(actor Actor, organizationID string) error {
	if strings.EqualFold(strings.TrimSpace(actor.Role), middleware.RoleAdmin) {
		return nil
	}
	if !strings.EqualFold(strings.TrimSpace(actor.OrganizationID), organizationID) {
		return apierrors.ErrForbidden
	}
	return nil
}

func canReadTenants(role string) bool {
	r := strings.TrimSpace(strings.ToLower(role))
	switch r {
	case middleware.RoleAdmin, middleware.RoleLandlord, middleware.RoleManager, string(users.UserRoleStaff), string(users.UserRoleTenant):
		return true
	default:
		return false
	}
}

func canMutateTenants(role string) bool {
	r := strings.TrimSpace(strings.ToLower(role))
	switch r {
	case middleware.RoleAdmin, middleware.RoleLandlord:
		return true
	default:
		return false
	}
}

func isManagerRole(role string) bool {
	return strings.EqualFold(strings.TrimSpace(role), middleware.RoleManager)
}

func isTenantRole(role string) bool {
	return strings.EqualFold(strings.TrimSpace(role), string(users.UserRoleTenant))
}

func parseTenantStatus(s string) TenantStatus {
	st := TenantStatus(strings.TrimSpace(strings.ToLower(s)))
	switch st {
	case TenantStatusActive, TenantStatusInactive, TenantStatusArchived:
		return st
	default:
		return ""
	}
}

func toResponse(t *Tenant, includeSummary bool, counts map[string]int, units map[string]CurrentUnitSummary) TenantResponse {
	r := TenantResponse{
		ID:             t.ID,
		OrganizationID: t.OrganizationID,
		FirstName:      t.FirstName,
		LastName:       t.LastName,
		FullName:       t.FullName,
		Email:          t.Email,
		Phone:          t.Phone,
		UserID:         t.UserID,
		EmailOptIn:     t.EmailOptIn,
		SmsOptIn:       t.SmsOptIn,
		SmsVerified:    t.SmsVerified,
		Locale:         t.Locale,
		Timezone:       t.Timezone,
		BillingDueDay:  t.BillingDueDay,
		BillingChannel: t.BillingChannel,
		Status:         string(t.Status),
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
		CreatedBy:      t.CreatedBy,
		UpdatedBy:      t.UpdatedBy,
	}
	if !includeSummary || counts == nil {
		return r
	}
	c := 0
	if v, ok := counts[t.ID]; ok {
		c = v
	}
	r.ActiveLeaseCount = &c
	if units != nil {
		if u, ok := units[t.ID]; ok {
			uu := u
			r.CurrentUnit = &uu
		}
	}
	return r
}
