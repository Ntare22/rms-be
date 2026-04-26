package leases

import (
	"context"
	stderrors "errors"
	"strings"
	"time"

	"gorm.io/gorm"

	apierrors "rms-be/internal/api/errors"
	"rms-be/internal/middleware"
	"rms-be/internal/modules/users"
)

// Actor is the authenticated subject.
type Actor struct {
	UserID         string
	OrganizationID string
	Role           string
}

// Service contains lease business logic.
type Service struct {
	repo *Repository
}

// NewService constructs a Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// List returns paginated leases.
func (s *Service) List(ctx context.Context, actor Actor, organizationID string, f ListFilters, page, pageSize, offset int) (*LeaseListResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return nil, apierrors.ErrValidation
	}
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canReadLeases(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	if isManagerRole(actor.Role) {
		ids, err := s.repo.ListManagerBuildingIDs(ctx, organizationID, actor.UserID)
		if err != nil {
			return nil, apierrors.Wrap(err, apierrors.ErrInternal)
		}
		if len(ids) == 0 {
			return &LeaseListResponse{Items: []LeaseResponse{}, Page: page, PageSize: pageSize, Total: 0}, nil
		}
		if strings.TrimSpace(f.BuildingID) != "" {
			match := false
			for _, id := range ids {
				if strings.EqualFold(strings.TrimSpace(id), strings.TrimSpace(f.BuildingID)) {
					match = true
					break
				}
			}
			if !match {
				return &LeaseListResponse{Items: []LeaseResponse{}, Page: page, PageSize: pageSize, Total: 0}, nil
			}
		}
		f.AllowedBuildingIDs = ids
	}
	if isTenantRole(actor.Role) {
		tid, err := s.repo.TenantIDByUser(ctx, organizationID, actor.UserID)
		if err != nil {
			return nil, apierrors.Wrap(err, apierrors.ErrInternal)
		}
		if strings.TrimSpace(tid) == "" {
			return &LeaseListResponse{Items: []LeaseResponse{}, Page: page, PageSize: pageSize, Total: 0}, nil
		}
		f.TenantID = tid
	}
	rows, total, err := s.repo.List(ctx, organizationID, f, offset, pageSize)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	out := make([]LeaseResponse, 0, len(rows))
	for i := range rows {
		out = append(out, toResponse(&rows[i]))
	}
	return &LeaseListResponse{Items: out, Page: page, PageSize: pageSize, Total: total}, nil
}

// Create creates a lease.
func (s *Service) Create(ctx context.Context, actor Actor, organizationID string, req *CreateLeaseRequest, ip, userAgent string) (*LeaseResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canMutateLeases(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	if err := ValidateLeaseDateOrder(req.StartDate, req.EndDate); err != nil {
		return nil, err
	}
	okT, err := s.repo.TenantBelongsToOrg(ctx, organizationID, req.TenantID)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if !okT {
		return nil, apierrors.ErrNotFound
	}
	okU, err := s.repo.UnitBelongsToOrg(ctx, organizationID, req.UnitID)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if !okU {
		return nil, apierrors.ErrNotFound
	}
	if isManagerRole(actor.Role) {
		ok, err := s.managerCanAccessUnit(ctx, organizationID, actor.UserID, req.UnitID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, apierrors.ErrForbidden
		}
	}
	st := parseLeaseStatus(req.Status)
	if st == "" {
		st = LeaseStatusPendingApproval
	}
	isPrimary := true
	if req.IsPrimary != nil {
		isPrimary = *req.IsPrimary
	}
	cur := strings.ToUpper(strings.TrimSpace(req.Currency))
	if cur == "" {
		cur = "USD"
	}
	if err := s.assertNoPrimaryActiveOverlap(ctx, organizationID, req.UnitID, "", req.StartDate, req.EndDate, st, isPrimary); err != nil {
		return nil, err
	}
	uid := strings.TrimSpace(actor.UserID)
	l := &Lease{
		OrganizationID:             organizationID,
		UnitID:                     strings.TrimSpace(req.UnitID),
		TenantID:                   strings.TrimSpace(req.TenantID),
		StartDate:                  req.StartDate,
		EndDate:                    req.EndDate,
		MonthlyRentAmountMinor:     req.MonthlyRentAmountMinor,
		DepositAmountMinor:         req.DepositAmountMinor,
		Currency:                   cur,
		BillingDueDay:              req.BillingDueDay,
		BillingAmountOverrideMinor: req.BillingAmountOverrideMinor,
		BillingReminderChannel:     strings.TrimSpace(strings.ToLower(req.BillingReminderChannel)),
		Status:                     st,
		IsPrimary:                  isPrimary,
	}
	if uid != "" {
		l.CreatedBy = &uid
		l.UpdatedBy = &uid
	}
	if err := s.repo.Create(ctx, l); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if err := s.syncUnitOccupancy(ctx, organizationID, l.UnitID); err != nil {
		return nil, err
	}
	_ = s.repo.WriteAuditLog(ctx, organizationID, ptrOrNil(uid), "lease.created", l.ID, map[string]any{
		"unit_id": l.UnitID, "tenant_id": l.TenantID, "status": string(l.Status), "is_primary": l.IsPrimary,
	}, ip, userAgent)
	r := toResponse(l)
	return &r, nil
}

// Get returns one lease.
func (s *Service) Get(ctx context.Context, actor Actor, organizationID, leaseID string) (*LeaseResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	leaseID = strings.TrimSpace(leaseID)
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canReadLeases(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	l, err := s.repo.GetByIDInOrg(ctx, organizationID, leaseID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrNotFound
		}
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if isManagerRole(actor.Role) {
		ok, err := s.managerCanAccessUnit(ctx, organizationID, actor.UserID, l.UnitID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, apierrors.ErrForbidden
		}
	}
	if isTenantRole(actor.Role) {
		tid, err := s.repo.TenantIDByUser(ctx, organizationID, actor.UserID)
		if err != nil {
			return nil, apierrors.Wrap(err, apierrors.ErrInternal)
		}
		if !strings.EqualFold(strings.TrimSpace(tid), strings.TrimSpace(l.TenantID)) {
			return nil, apierrors.ErrForbidden
		}
	}
	r := toResponse(l)
	return &r, nil
}

// Patch updates a lease.
func (s *Service) Patch(ctx context.Context, actor Actor, organizationID, leaseID string, req *PatchLeaseRequest, ip, userAgent string) (*LeaseResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	leaseID = strings.TrimSpace(leaseID)
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canMutateLeases(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	cur, err := s.repo.GetByIDInOrg(ctx, organizationID, leaseID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrNotFound
		}
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if isManagerRole(actor.Role) {
		ok, err := s.managerCanAccessUnit(ctx, organizationID, actor.UserID, cur.UnitID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, apierrors.ErrForbidden
		}
	}
	if req.Status != nil {
		t := strings.TrimSpace(*req.Status)
		if t == "" || parseLeaseStatus(t) == "" {
			return nil, apierrors.ErrValidation
		}
	}
	oldUnit := cur.UnitID
	next := mergeLeasePatch(cur, req)
	if isManagerRole(actor.Role) {
		ok, err := s.managerCanAccessUnit(ctx, organizationID, actor.UserID, next.UnitID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, apierrors.ErrForbidden
		}
	}
	if err := ValidateLeaseDateOrder(next.StartDate, next.EndDate); err != nil {
		return nil, err
	}
	if err := s.assertNoPrimaryActiveOverlap(ctx, organizationID, next.UnitID, leaseID, next.StartDate, next.EndDate, next.Status, next.IsPrimary); err != nil {
		return nil, err
	}
	updates := leaseToUpdates(req, strings.TrimSpace(actor.UserID))
	if len(updates) == 0 {
		r := toResponse(cur)
		return &r, nil
	}
	if err := s.repo.Update(ctx, organizationID, leaseID, updates); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	refreshed, err := s.repo.GetByIDInOrg(ctx, organizationID, leaseID)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if err := s.syncUnitOccupancy(ctx, organizationID, oldUnit); err != nil {
		return nil, err
	}
	if refreshed.UnitID != oldUnit {
		if err := s.syncUnitOccupancy(ctx, organizationID, refreshed.UnitID); err != nil {
			return nil, err
		}
	}
	uid := strings.TrimSpace(actor.UserID)
	_ = s.repo.WriteAuditLog(ctx, organizationID, ptrOrNil(uid), "lease.updated", leaseID, map[string]any{
		"fields": keysOf(updates),
	}, ip, userAgent)
	r := toResponse(refreshed)
	return &r, nil
}

// End terminates a lease (status ended, end date set).
func (s *Service) End(ctx context.Context, actor Actor, organizationID, leaseID string, req *EndLeaseRequest, ip, userAgent string) (*LeaseResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	leaseID = strings.TrimSpace(leaseID)
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canMutateLeases(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	l, err := s.repo.GetByIDInOrg(ctx, organizationID, leaseID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrNotFound
		}
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if isManagerRole(actor.Role) {
		ok, err := s.managerCanAccessUnit(ctx, organizationID, actor.UserID, l.UnitID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, apierrors.ErrForbidden
		}
	}
	endAt := time.Now().UTC()
	if req.EndDate != nil {
		endAt = req.EndDate.UTC()
	}
	if !endAt.After(l.StartDate) {
		return nil, apierrors.ErrValidation
	}
	uid := strings.TrimSpace(actor.UserID)
	updates := map[string]any{
		"status":   LeaseStatusEnded,
		"end_date": endAt,
	}
	if uid != "" {
		updates["updated_by"] = uid
	}
	if err := s.repo.Update(ctx, organizationID, leaseID, updates); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	refreshed, err := s.repo.GetByIDInOrg(ctx, organizationID, leaseID)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if err := s.syncUnitOccupancy(ctx, organizationID, refreshed.UnitID); err != nil {
		return nil, err
	}
	_ = s.repo.WriteAuditLog(ctx, organizationID, ptrOrNil(uid), "lease.ended", leaseID, map[string]any{
		"end_date": endAt.Format(time.RFC3339), "note": strings.TrimSpace(req.Note),
	}, ip, userAgent)
	r := toResponse(refreshed)
	return &r, nil
}

// Approve allows a tenant to approve their lease, activating it.
func (s *Service) Approve(ctx context.Context, actor Actor, organizationID, leaseID string, req *LeaseDecisionRequest, ip, userAgent string) (*LeaseResponse, error) {
	if !isTenantRole(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	l, err := s.repo.GetByIDInOrg(ctx, strings.TrimSpace(organizationID), strings.TrimSpace(leaseID))
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrNotFound
		}
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	tid, err := s.repo.TenantIDByUser(ctx, organizationID, actor.UserID)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if !strings.EqualFold(strings.TrimSpace(tid), strings.TrimSpace(l.TenantID)) {
		return nil, apierrors.ErrForbidden
	}
	updates := map[string]any{"status": LeaseStatusActive}
	if err := s.repo.Update(ctx, organizationID, leaseID, updates); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	refreshed, err := s.repo.GetByIDInOrg(ctx, organizationID, leaseID)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if err := s.syncUnitOccupancy(ctx, organizationID, refreshed.UnitID); err != nil {
		return nil, err
	}
	_ = s.repo.WriteAuditLog(ctx, organizationID, ptrOrNil(strings.TrimSpace(actor.UserID)), "lease.approved", leaseID, map[string]any{
		"note": strings.TrimSpace(req.Note),
	}, ip, userAgent)
	r := toResponse(refreshed)
	return &r, nil
}

// Reject allows a tenant to reject their lease.
func (s *Service) Reject(ctx context.Context, actor Actor, organizationID, leaseID string, req *LeaseDecisionRequest, ip, userAgent string) (*LeaseResponse, error) {
	if !isTenantRole(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	l, err := s.repo.GetByIDInOrg(ctx, strings.TrimSpace(organizationID), strings.TrimSpace(leaseID))
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrNotFound
		}
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	tid, err := s.repo.TenantIDByUser(ctx, organizationID, actor.UserID)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if !strings.EqualFold(strings.TrimSpace(tid), strings.TrimSpace(l.TenantID)) {
		return nil, apierrors.ErrForbidden
	}
	updates := map[string]any{"status": LeaseStatusRejected}
	if err := s.repo.Update(ctx, organizationID, leaseID, updates); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	refreshed, err := s.repo.GetByIDInOrg(ctx, organizationID, leaseID)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if err := s.syncUnitOccupancy(ctx, organizationID, refreshed.UnitID); err != nil {
		return nil, err
	}
	_ = s.repo.WriteAuditLog(ctx, organizationID, ptrOrNil(strings.TrimSpace(actor.UserID)), "lease.rejected", leaseID, map[string]any{
		"note": strings.TrimSpace(req.Note),
	}, ip, userAgent)
	r := toResponse(refreshed)
	return &r, nil
}

func (s *Service) History(ctx context.Context, actor Actor, organizationID, leaseID string) (*LeaseHistoryResponse, error) {
	if err := assertOrgScope(actor, strings.TrimSpace(organizationID)); err != nil {
		return nil, err
	}
	if !canReadLeases(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	cur, err := s.repo.GetByIDInOrg(ctx, organizationID, leaseID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrNotFound
		}
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if isManagerRole(actor.Role) {
		ok, err := s.managerCanAccessUnit(ctx, organizationID, actor.UserID, cur.UnitID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, apierrors.ErrForbidden
		}
	}
	if isTenantRole(actor.Role) {
		tid, err := s.repo.TenantIDByUser(ctx, organizationID, actor.UserID)
		if err != nil {
			return nil, apierrors.Wrap(err, apierrors.ErrInternal)
		}
		if !strings.EqualFold(strings.TrimSpace(tid), strings.TrimSpace(cur.TenantID)) {
			return nil, apierrors.ErrForbidden
		}
	}
	rows, err := s.repo.ListHistoryForTenantAndUnit(ctx, organizationID, cur.TenantID, cur.UnitID)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	out := make([]LeaseResponse, 0, len(rows))
	for i := range rows {
		r := toResponse(&rows[i])
		out = append(out, r)
	}
	return &LeaseHistoryResponse{Items: out}, nil
}

func (s *Service) CreateRenewalOffer(ctx context.Context, actor Actor, organizationID, leaseID string, req *LeaseRenewalCreateRequest, ip, userAgent string) (*LeaseRenewalOfferResponse, error) {
	if err := assertOrgScope(actor, strings.TrimSpace(organizationID)); err != nil {
		return nil, err
	}
	if !canMutateLeases(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	l, err := s.repo.GetByIDInOrg(ctx, organizationID, leaseID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrNotFound
		}
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if isManagerRole(actor.Role) {
		ok, err := s.managerCanAccessUnit(ctx, organizationID, actor.UserID, l.UnitID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, apierrors.ErrForbidden
		}
	}
	cur := strings.ToUpper(strings.TrimSpace(req.Currency))
	if cur == "" {
		cur = l.Currency
	}
	row := &LeaseRenewalOffer{
		OrganizationID:         strings.TrimSpace(organizationID),
		LeaseID:                strings.TrimSpace(leaseID),
		StartDate:              req.StartDate,
		EndDate:                req.EndDate,
		MonthlyRentAmountMinor: req.MonthlyRentAmountMinor,
		Currency:               cur,
		Status:                 "offered",
	}
	if strings.TrimSpace(actor.UserID) != "" {
		row.CreatedBy = &actor.UserID
		row.UpdatedBy = &actor.UserID
	}
	if err := s.repo.CreateRenewalOffer(ctx, row); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	_ = s.repo.WriteAuditLog(ctx, organizationID, ptrOrNil(strings.TrimSpace(actor.UserID)), "lease.renewal_offer_created", leaseID, map[string]any{
		"offer_id": row.ID,
	}, ip, userAgent)
	resp := toRenewalOfferResponse(row)
	return &resp, nil
}

func (s *Service) AcceptRenewalOffer(ctx context.Context, actor Actor, organizationID, leaseID, offerID string, req *LeaseRenewalDecisionRequest, ip, userAgent string) (*LeaseRenewalOfferResponse, error) {
	if err := assertOrgScope(actor, strings.TrimSpace(organizationID)); err != nil {
		return nil, err
	}
	if !isTenantRole(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	l, err := s.repo.GetByIDInOrg(ctx, organizationID, leaseID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrNotFound
		}
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	tid, err := s.repo.TenantIDByUser(ctx, organizationID, actor.UserID)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if !strings.EqualFold(strings.TrimSpace(tid), strings.TrimSpace(l.TenantID)) {
		return nil, apierrors.ErrForbidden
	}
	offer, err := s.repo.GetRenewalOffer(ctx, organizationID, leaseID, offerID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrNotFound
		}
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if strings.TrimSpace(offer.Status) != "offered" {
		return nil, apierrors.ErrValidation
	}
	now := time.Now().UTC()
	if err := s.repo.UpdateRenewalOffer(ctx, organizationID, leaseID, offerID, map[string]any{
		"status":        "accepted",
		"decision_note": strings.TrimSpace(req.Note),
		"decided_at":    now,
	}); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	updates := map[string]any{"status": LeaseStatusEnded, "updated_by": actor.UserID}
	if offer.StartDate.After(l.StartDate) {
		endDate := offer.StartDate.Add(-time.Second)
		updates["end_date"] = endDate
	}
	_ = s.repo.Update(ctx, organizationID, leaseID, updates)
	newLease := &Lease{
		OrganizationID:         strings.TrimSpace(organizationID),
		UnitID:                 l.UnitID,
		TenantID:               l.TenantID,
		StartDate:              offer.StartDate,
		EndDate:                offer.EndDate,
		MonthlyRentAmountMinor: offer.MonthlyRentAmountMinor,
		Currency:               offer.Currency,
		Status:                 LeaseStatusActive,
		IsPrimary:              true,
		CreatedBy:              ptrOrNil(strings.TrimSpace(actor.UserID)),
		UpdatedBy:              ptrOrNil(strings.TrimSpace(actor.UserID)),
	}
	if err := s.repo.Create(ctx, newLease); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if err := s.syncUnitOccupancy(ctx, organizationID, l.UnitID); err != nil {
		return nil, err
	}
	_ = s.repo.WriteAuditLog(ctx, organizationID, ptrOrNil(strings.TrimSpace(actor.UserID)), "lease.renewal_offer_accepted", leaseID, map[string]any{
		"offer_id":     offerID,
		"new_lease_id": newLease.ID,
	}, ip, userAgent)
	refreshed, _ := s.repo.GetRenewalOffer(ctx, organizationID, leaseID, offerID)
	if refreshed == nil {
		refreshed = offer
		refreshed.Status = "accepted"
		refreshed.DecidedAt = &now
		refreshed.DecisionNote = strings.TrimSpace(req.Note)
	}
	resp := toRenewalOfferResponse(refreshed)
	return &resp, nil
}

func (s *Service) RejectRenewalOffer(ctx context.Context, actor Actor, organizationID, leaseID, offerID string, req *LeaseRenewalDecisionRequest, ip, userAgent string) (*LeaseRenewalOfferResponse, error) {
	if err := assertOrgScope(actor, strings.TrimSpace(organizationID)); err != nil {
		return nil, err
	}
	if !isTenantRole(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	l, err := s.repo.GetByIDInOrg(ctx, organizationID, leaseID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrNotFound
		}
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	tid, err := s.repo.TenantIDByUser(ctx, organizationID, actor.UserID)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if !strings.EqualFold(strings.TrimSpace(tid), strings.TrimSpace(l.TenantID)) {
		return nil, apierrors.ErrForbidden
	}
	offer, err := s.repo.GetRenewalOffer(ctx, organizationID, leaseID, offerID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrNotFound
		}
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	now := time.Now().UTC()
	if err := s.repo.UpdateRenewalOffer(ctx, organizationID, leaseID, offerID, map[string]any{
		"status":        "rejected",
		"decision_note": strings.TrimSpace(req.Note),
		"decided_at":    now,
	}); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	_ = s.repo.WriteAuditLog(ctx, organizationID, ptrOrNil(strings.TrimSpace(actor.UserID)), "lease.renewal_offer_rejected", leaseID, map[string]any{
		"offer_id": offerID,
	}, ip, userAgent)
	refreshed, _ := s.repo.GetRenewalOffer(ctx, organizationID, leaseID, offerID)
	if refreshed == nil {
		refreshed = offer
		refreshed.Status = "rejected"
		refreshed.DecidedAt = &now
		refreshed.DecisionNote = strings.TrimSpace(req.Note)
	}
	resp := toRenewalOfferResponse(refreshed)
	return &resp, nil
}

func (s *Service) Closeout(ctx context.Context, actor Actor, organizationID, leaseID string, req *LeaseCloseoutRequest, ip, userAgent string) (*LeaseCloseoutResponse, error) {
	if err := assertOrgScope(actor, strings.TrimSpace(organizationID)); err != nil {
		return nil, err
	}
	if !canMutateLeases(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	l, err := s.repo.GetByIDInOrg(ctx, organizationID, leaseID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrNotFound
		}
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if isManagerRole(actor.Role) {
		ok, err := s.managerCanAccessUnit(ctx, organizationID, actor.UserID, l.UnitID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, apierrors.ErrForbidden
		}
	}
	cur := strings.ToUpper(strings.TrimSpace(req.Currency))
	if cur == "" {
		cur = l.Currency
	}
	row := &LeaseCloseout{
		OrganizationID:       strings.TrimSpace(organizationID),
		LeaseID:              strings.TrimSpace(leaseID),
		MoveOutDate:          req.MoveOutDate,
		FinalSettlementMinor: req.FinalSettlementMinor,
		Currency:             cur,
		Notes:                strings.TrimSpace(req.Notes),
		CreatedBy:            ptrOrNil(strings.TrimSpace(actor.UserID)),
		UpdatedBy:            ptrOrNil(strings.TrimSpace(actor.UserID)),
	}
	if err := s.repo.CreateCloseout(ctx, row); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if err := s.repo.Update(ctx, organizationID, leaseID, map[string]any{
		"status":     LeaseStatusEnded,
		"end_date":   req.MoveOutDate,
		"updated_by": actor.UserID,
	}); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if err := s.syncUnitOccupancy(ctx, organizationID, l.UnitID); err != nil {
		return nil, err
	}
	_ = s.repo.WriteAuditLog(ctx, organizationID, ptrOrNil(strings.TrimSpace(actor.UserID)), "lease.closeout_created", leaseID, map[string]any{
		"closeout_id": row.ID,
	}, ip, userAgent)
	resp := LeaseCloseoutResponse{
		ID:                   row.ID,
		LeaseID:              row.LeaseID,
		MoveOutDate:          row.MoveOutDate,
		FinalSettlementMinor: row.FinalSettlementMinor,
		Currency:             row.Currency,
		Notes:                row.Notes,
		CreatedAt:            row.CreatedAt,
	}
	return &resp, nil
}

func (s *Service) TenantStatement(ctx context.Context, actor Actor, organizationID, tenantID string) (*TenantStatementResponse, error) {
	if err := assertOrgScope(actor, strings.TrimSpace(organizationID)); err != nil {
		return nil, err
	}
	if !canReadLeases(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	if isTenantRole(actor.Role) {
		myTenantID, err := s.repo.TenantIDByUser(ctx, organizationID, actor.UserID)
		if err != nil {
			return nil, apierrors.Wrap(err, apierrors.ErrInternal)
		}
		if !strings.EqualFold(strings.TrimSpace(myTenantID), strings.TrimSpace(tenantID)) {
			return nil, apierrors.ErrForbidden
		}
	}
	rows, err := s.repo.StatementRows(ctx, organizationID, tenantID)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	resp := &TenantStatementResponse{
		TenantID: strings.TrimSpace(tenantID),
		Items:    rows,
	}
	for i := range rows {
		if rows[i].Kind == "charge" {
			resp.TotalChargesMinor += rows[i].AmountMinor
		}
		if rows[i].Kind == "payment" {
			resp.TotalPaymentsMinor += rows[i].AmountMinor
		}
	}
	resp.OutstandingMinor = resp.TotalChargesMinor - resp.TotalPaymentsMinor
	return resp, nil
}

func (s *Service) assertNoPrimaryActiveOverlap(ctx context.Context, organizationID, unitID, excludeLeaseID string, start time.Time, end *time.Time, status LeaseStatus, isPrimary bool) error {
	if status != LeaseStatusActive || !isPrimary {
		return nil
	}
	others, err := s.repo.ActivePrimaryLeasesForUnit(ctx, organizationID, unitID, excludeLeaseID)
	if err != nil {
		return apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if OverlapsActivePrimaryLease(start, end, others) {
		return apierrors.ErrLeaseOverlap
	}
	return nil
}

func (s *Service) syncUnitOccupancy(ctx context.Context, organizationID, unitID string) error {
	n, err := s.repo.CountActiveLeasesOnUnit(ctx, organizationID, unitID)
	if err != nil {
		return apierrors.Wrap(err, apierrors.ErrInternal)
	}
	st := unitStatusVacant
	if n > 0 {
		st = unitStatusOccupied
	}
	if err := s.repo.SetUnitStatus(ctx, organizationID, unitID, st); err != nil {
		return apierrors.Wrap(err, apierrors.ErrInternal)
	}
	return nil
}

func mergeLeasePatch(cur *Lease, req *PatchLeaseRequest) Lease {
	out := *cur
	if req.StartDate != nil {
		out.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		out.EndDate = req.EndDate
	}
	if req.MonthlyRentAmountMinor != nil {
		out.MonthlyRentAmountMinor = *req.MonthlyRentAmountMinor
	}
	if req.DepositAmountMinor != nil {
		out.DepositAmountMinor = req.DepositAmountMinor
	}
	if req.Currency != nil {
		out.Currency = strings.ToUpper(strings.TrimSpace(*req.Currency))
	}
	if req.Status != nil {
		if st := parseLeaseStatus(strings.TrimSpace(*req.Status)); st != "" {
			out.Status = st
		}
	}
	if req.BillingDueDay != nil {
		out.BillingDueDay = req.BillingDueDay
	}
	if req.BillingAmountOverrideMinor != nil {
		out.BillingAmountOverrideMinor = req.BillingAmountOverrideMinor
	}
	if req.BillingReminderChannel != nil {
		out.BillingReminderChannel = strings.TrimSpace(strings.ToLower(*req.BillingReminderChannel))
	}
	if req.IsPrimary != nil {
		out.IsPrimary = *req.IsPrimary
	}
	return out
}

func leaseToUpdates(req *PatchLeaseRequest, actorUID string) map[string]any {
	out := make(map[string]any)
	if req.StartDate != nil {
		out["start_date"] = *req.StartDate
	}
	if req.EndDate != nil {
		out["end_date"] = req.EndDate
	}
	if req.MonthlyRentAmountMinor != nil {
		out["monthly_rent_amount_minor"] = *req.MonthlyRentAmountMinor
	}
	if req.DepositAmountMinor != nil {
		out["deposit_amount_minor"] = req.DepositAmountMinor
	}
	if req.Currency != nil {
		out["currency"] = strings.ToUpper(strings.TrimSpace(*req.Currency))
	}
	if req.Status != nil {
		if st := parseLeaseStatus(strings.TrimSpace(*req.Status)); st != "" {
			out["status"] = string(st)
		}
	}
	if req.BillingDueDay != nil {
		out["billing_due_day"] = req.BillingDueDay
	}
	if req.BillingAmountOverrideMinor != nil {
		out["billing_amount_override_minor"] = req.BillingAmountOverrideMinor
	}
	if req.BillingReminderChannel != nil {
		out["billing_reminder_channel"] = strings.TrimSpace(strings.ToLower(*req.BillingReminderChannel))
	}
	if req.IsPrimary != nil {
		out["is_primary"] = *req.IsPrimary
	}
	if actorUID != "" {
		out["updated_by"] = actorUID
	}
	return out
}

func keysOf(m map[string]any) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

func ptrOrNil(uid string) *string {
	if uid == "" {
		return nil
	}
	return &uid
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

func canReadLeases(role string) bool {
	r := strings.TrimSpace(strings.ToLower(role))
	switch r {
	case middleware.RoleAdmin, middleware.RoleLandlord, middleware.RoleManager, middleware.RolePropertyManager, middleware.RoleAccountant, string(users.UserRoleStaff), string(users.UserRoleTenant):
		return true
	default:
		return false
	}
}

func canMutateLeases(role string) bool {
	r := strings.TrimSpace(strings.ToLower(role))
	switch r {
	case middleware.RoleAdmin, middleware.RoleLandlord, middleware.RoleManager, middleware.RolePropertyManager:
		return true
	default:
		return false
	}
}

func parseLeaseStatus(s string) LeaseStatus {
	st := LeaseStatus(strings.TrimSpace(strings.ToLower(s)))
	switch st {
	case LeaseStatusDraft, LeaseStatusPendingApproval, LeaseStatusActive, LeaseStatusRejected, LeaseStatusEnded, LeaseStatusCancelled:
		return st
	default:
		return ""
	}
}

func toResponse(l *Lease) LeaseResponse {
	return LeaseResponse{
		ID:                         l.ID,
		OrganizationID:             l.OrganizationID,
		TenantID:                   l.TenantID,
		UnitID:                     l.UnitID,
		StartDate:                  l.StartDate,
		EndDate:                    l.EndDate,
		MonthlyRentAmountMinor:     l.MonthlyRentAmountMinor,
		DepositAmountMinor:         l.DepositAmountMinor,
		Currency:                   l.Currency,
		BillingDueDay:              l.BillingDueDay,
		BillingAmountOverrideMinor: l.BillingAmountOverrideMinor,
		BillingReminderChannel:     l.BillingReminderChannel,
		Status:                     string(l.Status),
		IsPrimary:                  l.IsPrimary,
		CreatedAt:                  l.CreatedAt,
		UpdatedAt:                  l.UpdatedAt,
		CreatedBy:                  l.CreatedBy,
		UpdatedBy:                  l.UpdatedBy,
	}
}

func toRenewalOfferResponse(r *LeaseRenewalOffer) LeaseRenewalOfferResponse {
	return LeaseRenewalOfferResponse{
		ID:                     r.ID,
		LeaseID:                r.LeaseID,
		StartDate:              r.StartDate,
		EndDate:                r.EndDate,
		MonthlyRentAmountMinor: r.MonthlyRentAmountMinor,
		Currency:               r.Currency,
		Status:                 r.Status,
		DecisionNote:           r.DecisionNote,
		DecidedAt:              r.DecidedAt,
		CreatedAt:              r.CreatedAt,
	}
}

func isManagerRole(role string) bool {
	r := strings.TrimSpace(role)
	return strings.EqualFold(r, middleware.RoleManager) || strings.EqualFold(r, middleware.RolePropertyManager)
}

func isTenantRole(role string) bool {
	return strings.EqualFold(strings.TrimSpace(role), string(users.UserRoleTenant))
}

func (s *Service) managerCanAccessUnit(ctx context.Context, organizationID, userID, unitID string) (bool, error) {
	ids, err := s.repo.ListManagerBuildingIDs(ctx, organizationID, userID)
	if err != nil {
		return false, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	ok, err := s.repo.UnitBelongsToBuildings(ctx, organizationID, unitID, ids)
	if err != nil {
		return false, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	return ok, nil
}
