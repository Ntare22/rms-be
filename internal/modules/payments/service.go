package payments

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	apierrors "rms-be/internal/api/errors"
	"rms-be/internal/integrations/pesapal"
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
	repo                  *Repository
	gateway               pesapal.Client
	appBaseURL            string
	defaultNotificationID string
	ipnNotificationType   string
}

func NewService(repo *Repository, gateway pesapal.Client, appBaseURL, defaultNotificationID, ipnNotificationType string) *Service {
	t := strings.ToUpper(strings.TrimSpace(ipnNotificationType))
	if t == "" {
		t = "GET"
	}
	return &Service{
		repo:                  repo,
		gateway:               gateway,
		appBaseURL:            strings.TrimSpace(appBaseURL),
		defaultNotificationID: strings.TrimSpace(defaultNotificationID),
		ipnNotificationType:   t,
	}
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
		Provider:       "pesapal",
		ProviderStatus: "pending",
	}
	if strings.TrimSpace(actor.UserID) != "" {
		p.CreatedBy = &actor.UserID
		p.UpdatedBy = &actor.UserID
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	nextAction := "gateway_pending"
	if s.gateway != nil {
		res, err := s.gateway.SubmitOrder(ctx, pesapal.SubmitOrderRequest{
			ID:             p.ID,
			Currency:       p.Currency,
			Amount:         p.AmountMinor,
			Description:    "Lease payment",
			CallbackURL:    strings.TrimSuffix(s.appBaseURL, "/") + "/payments/callback",
			NotificationID: s.defaultNotificationID,
		})
		if err == nil && res != nil {
			if strings.TrimSpace(res.MerchantRef) != "" {
				p.ExternalRef = strings.TrimSpace(res.MerchantRef)
			}
			if strings.TrimSpace(res.OrderTrackingID) != "" {
				p.OrderTrackingID = strings.TrimSpace(res.OrderTrackingID)
			}
			p.ProviderStatus = "submitted"
			_ = s.repo.UpdateGatewayState(ctx, organizationID, p.ID, p.Provider, p.ExternalRef, p.OrderTrackingID, p.ProviderStatus)
			if strings.TrimSpace(res.RedirectURL) != "" {
				nextAction = strings.TrimSpace(res.RedirectURL)
			}
		}
	}
	return &PaymentResponse{
		ID:          p.ID,
		LeaseID:     p.LeaseID,
		AmountMinor: p.AmountMinor,
		Currency:    p.Currency,
		Method:      string(p.Method),
		Status:      string(p.Status),
		ExternalRef: p.ExternalRef,
		NextAction:  nextAction,
		CreatedAt:   p.CreatedAt,
	}, nil
}

func (s *Service) Summary(ctx context.Context, actor Actor, organizationID string, months int) (*PaymentSummaryResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canReadPayments(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	rows, err := s.repo.MonthlySummary(ctx, organizationID, months)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	byMonth := make(map[string]*PaymentSummaryPoint)
	for i := range rows {
		r := rows[i]
		p, ok := byMonth[r.Month]
		if !ok {
			p = &PaymentSummaryPoint{Month: r.Month}
			byMonth[r.Month] = p
		}
		p.CollectedMinor += r.CollectedMinor
		p.PendingMinor += r.PendingMinor
		p.OutstandingDueMinor += r.OutstandingDueMinor
	}
	out := make([]PaymentSummaryPoint, 0, len(byMonth))
	for _, v := range byMonth {
		out = append(out, *v)
	}
	return &PaymentSummaryResponse{Items: out}, nil
}

func (s *Service) MethodSplit(ctx context.Context, actor Actor, organizationID string) (*PaymentMethodSplitResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canReadPayments(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	rows, err := s.repo.MethodSplit(ctx, organizationID)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	items := make([]PaymentMethodSplitItem, 0, len(rows))
	for i := range rows {
		items = append(items, PaymentMethodSplitItem{
			Method:         rows[i].Method,
			Count:          rows[i].Count,
			CollectedMinor: rows[i].CollectedMinor,
		})
	}
	return &PaymentMethodSplitResponse{Items: items}, nil
}

func (s *Service) ReminderCandidates(ctx context.Context, actor Actor, organizationID string) (*ReminderCandidatesResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canReadPayments(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	rows, err := s.repo.ReminderCandidates(ctx, organizationID)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	out := make([]ReminderCandidate, 0, len(rows))
	for i := range rows {
		out = append(out, ReminderCandidate(rows[i]))
	}
	return &ReminderCandidatesResponse{Items: out}, nil
}

func (s *Service) ReminderHistory(ctx context.Context, actor Actor, organizationID string, limit int) (*ReminderHistoryResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canReadPayments(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	rows, err := s.repo.ReminderHistory(ctx, organizationID, limit)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	out := make([]ReminderHistoryItem, 0, len(rows))
	for i := range rows {
		out = append(out, ReminderHistoryItem{
			ID:       rows[i].ID,
			ChargeID: rows[i].ChargeID,
			LeaseID:  rows[i].LeaseID,
			TenantID: rows[i].TenantID,
			Level:    string(rows[i].Level),
			Channel:  rows[i].Channel,
			Status:   rows[i].Status,
			Note:     rows[i].Note,
			SentAt:   rows[i].SentAt,
		})
	}
	return &ReminderHistoryResponse{Items: out}, nil
}

func (s *Service) SendReminders(ctx context.Context, actor Actor, organizationID string, req *SendRemindersRequest) (*SendRemindersResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canMutatePayments(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	byID := make(map[string]ReminderCandidate)
	candidates, err := s.ReminderCandidates(ctx, actor, organizationID)
	if err != nil {
		return nil, err
	}
	for i := range candidates.Items {
		byID[strings.TrimSpace(candidates.Items[i].ChargeID)] = candidates.Items[i]
	}
	level := strings.TrimSpace(req.Level)
	note := strings.TrimSpace(req.Note)
	sent := 0
	skipped := 0
	for _, id := range req.ChargeIDs {
		c, ok := byID[strings.TrimSpace(id)]
		if !ok {
			skipped++
			continue
		}
		lvl := level
		if lvl == "" {
			lvl = c.SuggestedLevel
		}
		channel := strings.TrimSpace(req.Channel)
		if channel == "" {
			channel = c.Channel
		}
		row := &PaymentReminder{
			OrganizationID: organizationID,
			ChargeID:       c.ChargeID,
			LeaseID:        c.LeaseID,
			TenantID:       c.TenantID,
			Level:          ReminderLevel(lvl),
			Channel:        channel,
			Status:         "sent",
			Note:           note,
			SentAt:         time.Now().UTC(),
		}
		if strings.TrimSpace(actor.UserID) != "" {
			row.CreatedBy = &actor.UserID
		}
		if err := s.repo.CreateReminder(ctx, row); err != nil {
			return nil, apierrors.Wrap(fmt.Errorf("create reminder for charge %s: %w", id, err), apierrors.ErrInternal)
		}
		sent++
	}
	return &SendRemindersResponse{
		Sent:      sent,
		Skipped:   skipped,
		Requested: len(req.ChargeIDs),
	}, nil
}

func (s *Service) RegisterPesapalIPN(ctx context.Context, actor Actor, organizationID string, req *RegisterPesapalIPNRequest) (*PesapalIPNResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canMutatePayments(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	if s.gateway == nil {
		return nil, apierrors.ErrNotImplemented
	}
	url := strings.TrimSpace(req.URL)
	if url == "" {
		url = strings.TrimSuffix(s.appBaseURL, "/") + "/api/v1/payments/ipn/pesapal"
	}
	method := strings.ToUpper(strings.TrimSpace(req.IPNNotificationType))
	if method == "" {
		method = s.ipnNotificationType
	}
	out, err := s.gateway.RegisterIPN(ctx, url, method)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if strings.TrimSpace(out.IPNID) != "" {
		s.defaultNotificationID = strings.TrimSpace(out.IPNID)
	}
	return &PesapalIPNResponse{
		URL:                            out.URL,
		IPNID:                          out.IPNID,
		CreatedDate:                    out.CreatedDate,
		IPNNotificationTypeDescription: out.IPNNotificationTypeDescription,
		IPNStatusDescription:           out.IPNStatusDescription,
		Status:                         out.Status,
	}, nil
}

func (s *Service) ListPesapalIPN(ctx context.Context, actor Actor, organizationID string) (*PesapalIPNListResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canReadPayments(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	if s.gateway == nil {
		return nil, apierrors.ErrNotImplemented
	}
	rows, err := s.gateway.GetIPNList(ctx)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	items := make([]PesapalIPNResponse, 0, len(rows))
	for i := range rows {
		items = append(items, PesapalIPNResponse{
			URL:         rows[i].URL,
			IPNID:       rows[i].IPNID,
			CreatedDate: rows[i].CreatedDate,
			Status:      rows[i].Status,
		})
	}
	return &PesapalIPNListResponse{Items: items}, nil
}

func (s *Service) HandlePesapalIPN(ctx context.Context, req *PesapalIPNCallbackRequest) (*PesapalIPNCallbackResponse, error) {
	if req == nil {
		return &PesapalIPNCallbackResponse{Status: "accepted"}, nil
	}
	p, err := s.repo.FindByMerchantRefOrTrackingID(ctx, req.OrderMerchantReference, req.OrderTrackingID)
	if err != nil {
		return &PesapalIPNCallbackResponse{Status: "accepted"}, nil
	}
	rawBytes, _ := json.Marshal(req)
	callbackRaw := string(rawBytes)
	providerStatus := strings.TrimSpace(req.OrderNotificationType)
	next := mapProviderNotificationToPaymentStatus(providerStatus, p.Status)
	if err := s.repo.UpdateFromIPN(ctx, p.ID, next, providerStatus, callbackRaw); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	return &PesapalIPNCallbackResponse{Status: "accepted"}, nil
}

func (s *Service) GetTransactionStatus(ctx context.Context, actor Actor, organizationID, orderTrackingID string) (*PesapalTransactionStatusResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canReadPayments(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	if s.gateway == nil {
		return nil, apierrors.ErrNotImplemented
	}
	out, err := s.gateway.GetTransactionStatus(ctx, orderTrackingID)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if p, err := s.repo.FindByMerchantRefOrTrackingID(ctx, out.MerchantReference, strings.TrimSpace(orderTrackingID)); err == nil && p != nil {
		next := mapProviderStatusCodeToPaymentStatus(out.StatusCode, out.PaymentStatusDescription, p.Status)
		raw, _ := json.Marshal(out)
		_ = s.repo.UpdateFromIPN(ctx, p.ID, next, strings.TrimSpace(out.PaymentStatusDescription), string(raw))
		_ = s.repo.UpdateGatewayState(ctx, p.OrganizationID, p.ID, "pesapal", out.MerchantReference, strings.TrimSpace(orderTrackingID), strings.TrimSpace(out.PaymentStatusDescription))
	}
	return &PesapalTransactionStatusResponse{
		OrderTrackingID:          strings.TrimSpace(orderTrackingID),
		PaymentMethod:            out.PaymentMethod,
		Amount:                   out.Amount,
		CreatedDate:              out.CreatedDate,
		ConfirmationCode:         out.ConfirmationCode,
		PaymentStatusDescription: out.PaymentStatusDescription,
		Description:              out.Description,
		PaymentAccount:           out.PaymentAccount,
		CallbackURL:              out.CallbackURL,
		StatusCode:               out.StatusCode,
		MerchantReference:        out.MerchantReference,
		PaymentStatusCode:        out.PaymentStatusCode,
		Currency:                 out.Currency,
		Status:                   out.Status,
		Message:                  out.Message,
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
	case middleware.RoleAdmin, middleware.RoleLandlord, middleware.RoleManager, middleware.RolePropertyManager, middleware.RoleAccountant, string(users.UserRoleTenant):
		return true
	default:
		return false
	}
}

func canReadPayments(role string) bool {
	r := strings.TrimSpace(strings.ToLower(role))
	switch r {
	case middleware.RoleAdmin, middleware.RoleLandlord, middleware.RoleManager, middleware.RolePropertyManager, middleware.RoleAccountant, string(users.UserRoleTenant):
		return true
	default:
		return false
	}
}

func canMutatePayments(role string) bool {
	r := strings.TrimSpace(strings.ToLower(role))
	switch r {
	case middleware.RoleAdmin, middleware.RoleLandlord, middleware.RoleManager, middleware.RolePropertyManager, middleware.RoleAccountant:
		return true
	default:
		return false
	}
}

func mapProviderNotificationToPaymentStatus(notification string, current PaymentStatus) PaymentStatus {
	v := strings.ToLower(strings.TrimSpace(notification))
	switch v {
	case "completed", "paid", "success", "ipn_paid":
		return PaymentStatusCompleted
	case "failed", "rejected", "cancelled", "void":
		return PaymentStatusFailed
	default:
		if current != "" {
			return current
		}
		return PaymentStatusPending
	}
}

func mapProviderStatusCodeToPaymentStatus(statusCode int, description string, current PaymentStatus) PaymentStatus {
	switch statusCode {
	case 1:
		return PaymentStatusCompleted
	case 2, 3:
		return PaymentStatusFailed
	case 0:
		return PaymentStatusPending
	default:
		return mapProviderNotificationToPaymentStatus(description, current)
	}
}
