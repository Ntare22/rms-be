package units

import (
	"context"
	stderrors "errors"
	"strings"

	"gorm.io/gorm"

	apierrors "rms-be/internal/api/errors"
	"rms-be/internal/middleware"
	"rms-be/internal/modules/leases"
	"rms-be/internal/modules/users"
)

// Actor is the authenticated subject for authorization.
type Actor struct {
	UserID         string
	OrganizationID string
	Role           string
}

// Service contains unit business logic.
type Service struct {
	repo *Repository
}

// NewService constructs a Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// List returns paginated units for a building.
func (s *Service) List(ctx context.Context, actor Actor, organizationID, buildingID string, statusFilter string, includeOccupancy bool, page, pageSize, offset int) (*UnitListResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	buildingID = strings.TrimSpace(buildingID)
	if organizationID == "" || buildingID == "" {
		return nil, apierrors.ErrValidation
	}
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canReadUnits(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	if err := s.ensureBuilding(ctx, organizationID, buildingID); err != nil {
		return nil, err
	}
	if err := s.ensureManagerBuildingAccess(ctx, actor, organizationID, buildingID); err != nil {
		return nil, err
	}
	var st *UnitStatus
	if t := strings.TrimSpace(statusFilter); t != "" {
		v := parseUnitStatus(t)
		if v == "" {
			return nil, apierrors.ErrValidation
		}
		st = &v
	}
	rows, total, err := s.repo.ListByBuilding(ctx, organizationID, buildingID, st, offset, pageSize)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	var leaseByUnit map[string]leases.Lease
	if includeOccupancy && len(rows) > 0 {
		ids := make([]string, 0, len(rows))
		for i := range rows {
			ids = append(ids, rows[i].ID)
		}
		leaseByUnit, err = s.repo.ActiveLeasesByUnitIDs(ctx, organizationID, ids)
		if err != nil {
			return nil, apierrors.Wrap(err, apierrors.ErrInternal)
		}
	}
	out := make([]UnitResponse, 0, len(rows))
	for i := range rows {
		var occ *OccupancySummary
		if includeOccupancy {
			if l, ok := leaseByUnit[rows[i].ID]; ok {
				lid := l.ID
				tid := l.TenantID
				occ = &OccupancySummary{HasActiveLease: true, ActiveLeaseID: &lid, TenantID: &tid}
			} else {
				occ = &OccupancySummary{HasActiveLease: false}
			}
		}
		out = append(out, toResponse(&rows[i], occ))
	}
	return &UnitListResponse{Items: out, Page: page, PageSize: pageSize, Total: total}, nil
}

// Create adds a unit to a building.
func (s *Service) Create(ctx context.Context, actor Actor, organizationID, buildingID string, req *CreateUnitRequest) (*UnitResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	buildingID = strings.TrimSpace(buildingID)
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canMutateUnits(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	if err := s.ensureBuilding(ctx, organizationID, buildingID); err != nil {
		return nil, err
	}
	if err := s.ensureManagerBuildingAccess(ctx, actor, organizationID, buildingID); err != nil {
		return nil, err
	}
	st := parseUnitStatus(req.Status)
	if st == "" {
		st = UnitStatusVacant
	}
	cur := strings.ToUpper(strings.TrimSpace(req.Currency))
	if cur == "" {
		cur = "USD"
	}
	uid := strings.TrimSpace(actor.UserID)
	u := &Unit{
		OrganizationID:         organizationID,
		BuildingID:             buildingID,
		UnitLabel:              strings.TrimSpace(req.UnitLabel),
		Bedrooms:               req.Bedrooms,
		DefaultRentAmountMinor: req.DefaultRentAmountMinor,
		Currency:               cur,
		Status:                 st,
	}
	if uid != "" {
		u.CreatedBy = &uid
		u.UpdatedBy = &uid
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrConflict)
	}
	r := toResponse(u, nil)
	return &r, nil
}

// Get returns one unit.
func (s *Service) Get(ctx context.Context, actor Actor, organizationID, buildingID, unitID string, includeOccupancy bool) (*UnitResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	buildingID = strings.TrimSpace(buildingID)
	unitID = strings.TrimSpace(unitID)
	if organizationID == "" || buildingID == "" || unitID == "" {
		return nil, apierrors.ErrValidation
	}
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canReadUnits(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	if err := s.ensureBuilding(ctx, organizationID, buildingID); err != nil {
		return nil, err
	}
	if err := s.ensureManagerBuildingAccess(ctx, actor, organizationID, buildingID); err != nil {
		return nil, err
	}
	u, err := s.repo.GetByIDInBuilding(ctx, organizationID, buildingID, unitID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrNotFound
		}
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	var occ *OccupancySummary
	if includeOccupancy {
		m, err := s.repo.ActiveLeasesByUnitIDs(ctx, organizationID, []string{unitID})
		if err != nil {
			return nil, apierrors.Wrap(err, apierrors.ErrInternal)
		}
		if l, ok := m[unitID]; ok {
			lid := l.ID
			tid := l.TenantID
			occ = &OccupancySummary{HasActiveLease: true, ActiveLeaseID: &lid, TenantID: &tid}
		} else {
			occ = &OccupancySummary{HasActiveLease: false}
		}
	}
	r := toResponse(u, occ)
	return &r, nil
}

// Patch updates a unit.
func (s *Service) Patch(ctx context.Context, actor Actor, organizationID, buildingID, unitID string, req *PatchUnitRequest) (*UnitResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	buildingID = strings.TrimSpace(buildingID)
	unitID = strings.TrimSpace(unitID)
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canMutateUnits(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	if err := s.ensureBuilding(ctx, organizationID, buildingID); err != nil {
		return nil, err
	}
	if err := s.ensureManagerBuildingAccess(ctx, actor, organizationID, buildingID); err != nil {
		return nil, err
	}
	u, err := s.repo.GetByIDInBuilding(ctx, organizationID, buildingID, unitID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrNotFound
		}
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if req.Status != nil {
		t := strings.TrimSpace(*req.Status)
		if t != "" && parseUnitStatus(t) == "" {
			return nil, apierrors.ErrValidation
		}
	}
	updates := patchUpdates(req)
	if v, ok := updates["unit_label"].(string); ok && strings.TrimSpace(v) == "" {
		return nil, apierrors.ErrValidation
	}
	if len(updates) == 0 {
		r := toResponse(u, nil)
		return &r, nil
	}
	uid := strings.TrimSpace(actor.UserID)
	if uid != "" {
		updates["updated_by"] = uid
	}
	if err := s.repo.Update(ctx, organizationID, buildingID, unitID, updates); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	refreshed, err := s.repo.GetByIDInBuilding(ctx, organizationID, buildingID, unitID)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	r := toResponse(refreshed, nil)
	return &r, nil
}

// Delete soft-deletes a unit when there is no active lease.
func (s *Service) Delete(ctx context.Context, actor Actor, organizationID, buildingID, unitID string) error {
	organizationID = strings.TrimSpace(organizationID)
	buildingID = strings.TrimSpace(buildingID)
	unitID = strings.TrimSpace(unitID)
	if err := assertOrgScope(actor, organizationID); err != nil {
		return err
	}
	if !canMutateUnits(actor.Role) {
		return apierrors.ErrForbidden
	}
	if err := s.ensureBuilding(ctx, organizationID, buildingID); err != nil {
		return err
	}
	if err := s.ensureManagerBuildingAccess(ctx, actor, organizationID, buildingID); err != nil {
		return err
	}
	if _, err := s.repo.GetByIDInBuilding(ctx, organizationID, buildingID, unitID); err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return apierrors.ErrNotFound
		}
		return apierrors.Wrap(err, apierrors.ErrInternal)
	}
	n, err := s.repo.CountActiveLeasesForUnit(ctx, organizationID, unitID)
	if err != nil {
		return apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if n > 0 {
		return apierrors.ErrUnitHasActiveLease
	}
	if err := s.repo.SoftDelete(ctx, organizationID, buildingID, unitID); err != nil {
		return apierrors.Wrap(err, apierrors.ErrInternal)
	}
	return nil
}

func (s *Service) ensureBuilding(ctx context.Context, organizationID, buildingID string) error {
	ok, err := s.repo.BuildingExistsInOrg(ctx, organizationID, buildingID)
	if err != nil {
		return apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if !ok {
		return apierrors.ErrNotFound
	}
	return nil
}

func patchUpdates(req *PatchUnitRequest) map[string]any {
	out := make(map[string]any)
	if req.UnitLabel != nil {
		out["unit_label"] = strings.TrimSpace(*req.UnitLabel)
	}
	if req.Bedrooms != nil {
		out["bedrooms"] = req.Bedrooms
	}
	if req.DefaultRentAmountMinor != nil {
		out["default_rent_amount_minor"] = req.DefaultRentAmountMinor
	}
	if req.Currency != nil {
		out["currency"] = strings.ToUpper(strings.TrimSpace(*req.Currency))
	}
	if req.Status != nil {
		if st := parseUnitStatus(strings.TrimSpace(*req.Status)); st != "" {
			out["status"] = string(st)
		}
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

func canReadUnits(role string) bool {
	r := strings.TrimSpace(strings.ToLower(role))
	switch r {
	case middleware.RoleAdmin, middleware.RoleLandlord, middleware.RoleManager, string(users.UserRoleStaff):
		return true
	default:
		return false
	}
}

func canMutateUnits(role string) bool {
	r := strings.TrimSpace(strings.ToLower(role))
	switch r {
	case middleware.RoleAdmin, middleware.RoleLandlord, middleware.RoleManager:
		return true
	default:
		return false
	}
}

func parseUnitStatus(s string) UnitStatus {
	st := UnitStatus(strings.TrimSpace(strings.ToLower(s)))
	switch st {
	case UnitStatusVacant, UnitStatusOccupied, UnitStatusOffline, UnitStatusMaintenance:
		return st
	default:
		return ""
	}
}

func (s *Service) ensureManagerBuildingAccess(ctx context.Context, actor Actor, organizationID, buildingID string) error {
	if !strings.EqualFold(strings.TrimSpace(actor.Role), middleware.RoleManager) {
		return nil
	}
	ids, err := s.repo.ListManagerBuildingIDs(ctx, organizationID, actor.UserID)
	if err != nil {
		return apierrors.Wrap(err, apierrors.ErrInternal)
	}
	for _, id := range ids {
		if strings.EqualFold(strings.TrimSpace(id), strings.TrimSpace(buildingID)) {
			return nil
		}
	}
	return apierrors.ErrForbidden
}

func toResponse(u *Unit, occ *OccupancySummary) UnitResponse {
	return UnitResponse{
		ID:                     u.ID,
		OrganizationID:         u.OrganizationID,
		BuildingID:             u.BuildingID,
		UnitLabel:              u.UnitLabel,
		Bedrooms:               u.Bedrooms,
		DefaultRentAmountMinor: u.DefaultRentAmountMinor,
		Currency:               u.Currency,
		Status:                 string(u.Status),
		Occupancy:              occ,
		CreatedAt:              u.CreatedAt,
		UpdatedAt:              u.UpdatedAt,
		CreatedBy:              u.CreatedBy,
		UpdatedBy:              u.UpdatedBy,
	}
}
