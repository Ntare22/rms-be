package payments

import (
	"context"
	"strings"
	"time"

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

// Service handles payment initiation.
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Initiate creates a pending placeholder payment record.
func (s *Service) Initiate(ctx context.Context, actor Actor, organizationID string, req *InitiatePaymentRequest) (*PaymentResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canInitiatePayments(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	lease, err := s.repo.GetLeaseScope(ctx, organizationID, req.LeaseID)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrNotFound)
	}
	if strings.EqualFold(strings.TrimSpace(actor.Role), string(users.UserRoleTenant)) {
		tid, err := s.repo.TenantIDByUser(ctx, organizationID, actor.UserID)
		if err != nil {
			return nil, apierrors.Wrap(err, apierrors.ErrInternal)
		}
		if !strings.EqualFold(strings.TrimSpace(tid), strings.TrimSpace(lease.TenantID)) {
			return nil, apierrors.ErrForbidden
		}
	}
	if strings.EqualFold(strings.TrimSpace(actor.Role), middleware.RoleManager) {
		ids, err := s.repo.ListManagerBuildingIDs(ctx, organizationID, actor.UserID)
		if err != nil {
			return nil, apierrors.Wrap(err, apierrors.ErrInternal)
		}
		ok, err := s.repo.UnitInBuildings(ctx, organizationID, lease.UnitID, ids)
		if err != nil {
			return nil, apierrors.Wrap(err, apierrors.ErrInternal)
		}
		if !ok {
			return nil, apierrors.ErrForbidden
		}
	}
	method := PaymentMethod(strings.TrimSpace(strings.ToLower(req.Method)))
	if method == "" {
		method = PaymentMethodOther
	}
	cur := strings.ToUpper(strings.TrimSpace(req.Currency))
	if cur == "" {
		cur = "USD"
	}
	p := &Payment{
		OrganizationID: organizationID,
		LeaseID:        lease.ID,
		AmountMinor:    req.AmountMinor,
		Currency:       cur,
		Method:         method,
		Status:         PaymentStatusPending,
		ReceivedAt:     time.Now().UTC(),
		ExternalRef:    "placeholder_" + time.Now().UTC().Format("20060102150405"),
	}
	if strings.TrimSpace(actor.UserID) != "" {
		p.CreatedBy = &actor.UserID
		p.UpdatedBy = &actor.UserID
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	return &PaymentResponse{
		ID:          p.ID,
		LeaseID:     p.LeaseID,
		AmountMinor: p.AmountMinor,
		Currency:    p.Currency,
		Method:      string(p.Method),
		Status:      string(p.Status),
		ExternalRef: p.ExternalRef,
		NextAction:  "gateway_pending",
		CreatedAt:   p.CreatedAt,
	}, nil
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

func canInitiatePayments(role string) bool {
	r := strings.TrimSpace(strings.ToLower(role))
	switch r {
	case middleware.RoleAdmin, middleware.RoleLandlord, middleware.RoleManager, string(users.UserRoleTenant):
		return true
	default:
		return false
	}
}
