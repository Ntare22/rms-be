package buildings

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

// Service contains building business logic.
type Service struct {
	repo *Repository
}

// NewService constructs a Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// List returns paginated buildings for an organization (always org-scoped in repository).
func (s *Service) List(ctx context.Context, actor Actor, organizationID string, statusFilter string, page, pageSize, offset int) (*BuildingListResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return nil, apierrors.ErrValidation
	}
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canReadBuildings(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	var st *BuildingStatus
	if t := strings.TrimSpace(statusFilter); t != "" {
		v := parseBuildingStatus(t)
		if v == "" {
			return nil, apierrors.ErrValidation
		}
		st = &v
	}
	rows, total, err := s.repo.ListByOrganization(ctx, organizationID, st, offset, pageSize)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	out := make([]BuildingResponse, 0, len(rows))
	for i := range rows {
		out = append(out, toResponse(&rows[i]))
	}
	return &BuildingListResponse{
		Items:    out,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}, nil
}

// Create adds a building in an organization.
func (s *Service) Create(ctx context.Context, actor Actor, organizationID string, req *CreateBuildingRequest) (*BuildingResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canMutateBuildings(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	st := parseBuildingStatus(req.Status)
	if st == "" {
		st = BuildingStatusActive
	}
	uid := strings.TrimSpace(actor.UserID)
	b := &Building{
		OrganizationID: organizationID,
		Name:           strings.TrimSpace(req.Name),
		Status:         st,
		AddressLine1:   strings.TrimSpace(req.AddressLine1),
		AddressLine2:   strings.TrimSpace(req.AddressLine2),
		City:           strings.TrimSpace(req.City),
		State:          strings.TrimSpace(req.State),
		PostalCode:     strings.TrimSpace(req.PostalCode),
		Country:        strings.ToUpper(strings.TrimSpace(req.Country)),
		Timezone:       strings.TrimSpace(req.Timezone),
	}
	if b.Country == "" {
		b.Country = "US"
	}
	if uid != "" {
		b.CreatedBy = &uid
		b.UpdatedBy = &uid
	}
	if err := s.repo.Create(ctx, b); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	r := toResponse(b)
	return &r, nil
}

// Get returns one building if it belongs to the organization and the actor may read it.
func (s *Service) Get(ctx context.Context, actor Actor, organizationID, buildingID string) (*BuildingResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	buildingID = strings.TrimSpace(buildingID)
	if organizationID == "" || buildingID == "" {
		return nil, apierrors.ErrValidation
	}
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canReadBuildings(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	b, err := s.repo.GetByIDInOrg(ctx, organizationID, buildingID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrNotFound
		}
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	r := toResponse(b)
	return &r, nil
}

// Patch updates a building in an organization.
func (s *Service) Patch(ctx context.Context, actor Actor, organizationID, buildingID string, req *PatchBuildingRequest) (*BuildingResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	buildingID = strings.TrimSpace(buildingID)
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canMutateBuildings(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	b, err := s.repo.GetByIDInOrg(ctx, organizationID, buildingID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrNotFound
		}
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if req.Status != nil {
		t := strings.TrimSpace(*req.Status)
		if t != "" && parseBuildingStatus(t) == "" {
			return nil, apierrors.ErrValidation
		}
	}
	updates := patchUpdates(req)
	if v, ok := updates["name"].(string); ok && strings.TrimSpace(v) == "" {
		return nil, apierrors.ErrValidation
	}
	if len(updates) == 0 {
		r := toResponse(b)
		return &r, nil
	}
	uid := strings.TrimSpace(actor.UserID)
	if uid != "" {
		updates["updated_by"] = uid
	}
	if err := s.repo.Update(ctx, organizationID, buildingID, updates); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	refreshed, err := s.repo.GetByIDInOrg(ctx, organizationID, buildingID)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	r := toResponse(refreshed)
	return &r, nil
}

// Delete soft-deletes a building when it has no units.
func (s *Service) Delete(ctx context.Context, actor Actor, organizationID, buildingID string) error {
	organizationID = strings.TrimSpace(organizationID)
	buildingID = strings.TrimSpace(buildingID)
	if err := assertOrgScope(actor, organizationID); err != nil {
		return err
	}
	if !canMutateBuildings(actor.Role) {
		return apierrors.ErrForbidden
	}
	if _, err := s.repo.GetByIDInOrg(ctx, organizationID, buildingID); err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return apierrors.ErrNotFound
		}
		return apierrors.Wrap(err, apierrors.ErrInternal)
	}
	n, err := s.repo.CountUnitsForBuilding(ctx, organizationID, buildingID)
	if err != nil {
		return apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if n > 0 {
		return apierrors.ErrBuildingHasUnits
	}
	if err := s.repo.SoftDelete(ctx, organizationID, buildingID); err != nil {
		return apierrors.Wrap(err, apierrors.ErrInternal)
	}
	return nil
}

func patchUpdates(req *PatchBuildingRequest) map[string]any {
	out := make(map[string]any)
	if req.Name != nil {
		out["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Status != nil {
		if st := parseBuildingStatus(strings.TrimSpace(*req.Status)); st != "" {
			out["status"] = string(st)
		}
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
	if req.State != nil {
		out["state"] = strings.TrimSpace(*req.State)
	}
	if req.PostalCode != nil {
		out["postal_code"] = strings.TrimSpace(*req.PostalCode)
	}
	if req.Country != nil {
		out["country"] = strings.ToUpper(strings.TrimSpace(*req.Country))
	}
	if req.Timezone != nil {
		out["timezone"] = strings.TrimSpace(*req.Timezone)
	}
	return out
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

func canReadBuildings(role string) bool {
	r := strings.TrimSpace(strings.ToLower(role))
	switch r {
	case middleware.RoleAdmin, middleware.RoleLandlord, middleware.RoleManager, string(users.UserRoleStaff):
		return true
	default:
		return false
	}
}

func canMutateBuildings(role string) bool {
	r := strings.TrimSpace(strings.ToLower(role))
	switch r {
	case middleware.RoleAdmin, middleware.RoleLandlord, middleware.RoleManager:
		return true
	default:
		return false
	}
}

func parseBuildingStatus(s string) BuildingStatus {
	st := BuildingStatus(strings.TrimSpace(strings.ToLower(s)))
	switch st {
	case BuildingStatusActive, BuildingStatusInactive, BuildingStatusArchived:
		return st
	default:
		return ""
	}
}

func toResponse(b *Building) BuildingResponse {
	return BuildingResponse{
		ID:             b.ID,
		OrganizationID: b.OrganizationID,
		Name:           b.Name,
		Status:         string(b.Status),
		AddressLine1:   b.AddressLine1,
		AddressLine2:   b.AddressLine2,
		City:           b.City,
		State:          b.State,
		PostalCode:     b.PostalCode,
		Country:        b.Country,
		Timezone:       b.Timezone,
		CreatedAt:      b.CreatedAt,
		UpdatedAt:      b.UpdatedAt,
		CreatedBy:      b.CreatedBy,
		UpdatedBy:      b.UpdatedBy,
	}
}
