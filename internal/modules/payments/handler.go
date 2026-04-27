package payments

import (
	"strconv"

	"github.com/gin-gonic/gin"

	apierrors "rms-be/internal/api/errors"
	"rms-be/internal/api/response"
	"rms-be/internal/api/validator"
	"rms-be/internal/middleware"
)

// Handler exposes payment HTTP endpoints.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func actorFromContext(c *gin.Context) (Actor, error) {
	cl, ok := middleware.ClaimsFromContext(c)
	if !ok || cl == nil {
		return Actor{}, apierrors.ErrUnauthorized
	}
	return Actor{
		UserID:         cl.UserID,
		OrganizationID: cl.OrganizationID,
		Role:           cl.Role,
	}, nil
}

// Initiate godoc
//
//	@Summary		Initiate payment
//	@Description	Creates a pending payment placeholder for tenant or manager initiated payment flow.
//	@Tags			payments
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path	string					true	"Organization ID"	Format(uuid)
//	@Param			body	body	InitiatePaymentRequest	true	"Payment initiation payload"
//	@Success		201		{object}	response.Envelope[PaymentResponse]
//	@Failure		400		{object}	response.ErrorBody
//	@Failure		401		{object}	response.ErrorBody
//	@Failure		403		{object}	response.ErrorBody
//	@Failure		404		{object}	response.ErrorBody
//	@Failure		500		{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/payments/initiate [post]
func (h *Handler) Initiate(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var req InitiatePaymentRequest
	if err := validator.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.Initiate(c.Request.Context(), actor, c.Param("id"), &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, out)
}

// RecordManual godoc
//
//	@Summary		Record manual payment
//	@Description	Records an offline/manual payment for a lease (e.g., late cash/bank payment).
//	@Tags			payments
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path	string						true	"Organization ID"	Format(uuid)
//	@Param			body	body	RecordManualPaymentRequest	true	"Manual payment payload"
//	@Success		201		{object}	response.Envelope[PaymentResponse]
//	@Failure		400		{object}	response.ErrorBody
//	@Failure		401		{object}	response.ErrorBody
//	@Failure		403		{object}	response.ErrorBody
//	@Failure		404		{object}	response.ErrorBody
//	@Failure		500		{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/payments/manual [post]
func (h *Handler) RecordManual(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var req RecordManualPaymentRequest
	if err := validator.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.RecordManual(c.Request.Context(), actor, c.Param("id"), &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, out)
}

// Summary godoc
//
//	@Summary		Payments summary
//	@Description	Returns monthly payments and outstanding aggregates.
//	@Tags			payments
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id		path	string	true	"Organization ID"	Format(uuid)
//	@Param			months	query	int		false	"Number of months (default 6)"
//	@Success		200		{object}	response.Envelope[PaymentSummaryResponse]
//	@Failure		401		{object}	response.ErrorBody
//	@Failure		403		{object}	response.ErrorBody
//	@Failure		500		{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/payments/summary [get]
func (h *Handler) Summary(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	months := 6
	if raw := c.Query("months"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			months = v
		}
	}
	out, err := h.svc.Summary(c.Request.Context(), actor, c.Param("id"), months)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// MethodSplit godoc
//
//	@Summary		Payments method split
//	@Description	Returns collected totals grouped by payment method.
//	@Tags			payments
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path	string	true	"Organization ID"	Format(uuid)
//	@Success		200	{object}	response.Envelope[PaymentMethodSplitResponse]
//	@Failure		401	{object}	response.ErrorBody
//	@Failure		403	{object}	response.ErrorBody
//	@Failure		500	{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/payments/method-split [get]
func (h *Handler) MethodSplit(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.MethodSplit(c.Request.Context(), actor, c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// ReminderCandidates godoc
//
//	@Summary		Reminder candidates
//	@Description	Lists upcoming/overdue unpaid charge candidates for reminder workflows.
//	@Tags			payments
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path	string	true	"Organization ID"	Format(uuid)
//	@Success		200	{object}	response.Envelope[ReminderCandidatesResponse]
//	@Failure		401	{object}	response.ErrorBody
//	@Failure		403	{object}	response.ErrorBody
//	@Failure		500	{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/payments/reminders/candidates [get]
func (h *Handler) ReminderCandidates(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.ReminderCandidates(c.Request.Context(), actor, c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// ReminderHistory godoc
//
//	@Summary		Reminder history
//	@Description	Returns sent reminder history.
//	@Tags			payments
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id		path	string	true	"Organization ID"	Format(uuid)
//	@Param			limit	query	int		false	"Max rows (default 100)"
//	@Success		200		{object}	response.Envelope[ReminderHistoryResponse]
//	@Failure		401		{object}	response.ErrorBody
//	@Failure		403		{object}	response.ErrorBody
//	@Failure		500		{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/payments/reminders/history [get]
func (h *Handler) ReminderHistory(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	limit := 100
	if raw := c.Query("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			limit = v
		}
	}
	out, err := h.svc.ReminderHistory(c.Request.Context(), actor, c.Param("id"), limit)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// SendReminders godoc
//
//	@Summary		Send reminders
//	@Description	Bulk creates reminder send records for selected charge IDs.
//	@Tags			payments
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path	string				true	"Organization ID"	Format(uuid)
//	@Param			body	body	SendRemindersRequest	true	"Bulk reminder payload"
//	@Success		200		{object}	response.Envelope[SendRemindersResponse]
//	@Failure		400		{object}	response.ErrorBody
//	@Failure		401		{object}	response.ErrorBody
//	@Failure		403		{object}	response.ErrorBody
//	@Failure		500		{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/payments/reminders/send [post]
func (h *Handler) SendReminders(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var req SendRemindersRequest
	if err := validator.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.SendReminders(c.Request.Context(), actor, c.Param("id"), &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// RegisterPesapalIPN godoc
//
//	@Summary		Register Pesapal IPN
//	@Description	Registers merchant IPN URL in Pesapal and returns the generated IPN ID.
//	@Tags			payments
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			body	body	RegisterPesapalIPNRequest	false	"Optional URL/type override"
//	@Success		200		{object}	response.Envelope[PesapalIPNResponse]
//	@Failure		400		{object}	response.ErrorBody
//	@Failure		401		{object}	response.ErrorBody
//	@Failure		403		{object}	response.ErrorBody
//	@Failure		500		{object}	response.ErrorBody
//	@Router			/api/v1/payments/pesapal/ipn/register [post]
func (h *Handler) RegisterPesapalIPN(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var req RegisterPesapalIPNRequest
	if c.Request.ContentLength > 0 {
		if err := validator.BindJSON(c, &req); err != nil {
			response.Error(c, err)
			return
		}
	}
	out, err := h.svc.RegisterPesapalIPN(c.Request.Context(), actor, &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// ListPesapalIPN godoc
//
//	@Summary		List Pesapal IPNs
//	@Description	Lists registered IPN URLs for the current merchant account.
//	@Tags			payments
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{object}	response.Envelope[PesapalIPNListResponse]
//	@Failure		401	{object}	response.ErrorBody
//	@Failure		403	{object}	response.ErrorBody
//	@Failure		500	{object}	response.ErrorBody
//	@Router			/api/v1/payments/pesapal/ipn/list [get]
func (h *Handler) ListPesapalIPN(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.ListPesapalIPN(c.Request.Context(), actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// PesapalIPNCallback godoc
//
//	@Summary		Pesapal IPN callback
//	@Description	Public endpoint for Pesapal status change notifications.
//	@Tags			payments
//	@Accept			json
//	@Produce		json
//	@Param			OrderNotificationType	query	string	false	"Notification type"
//	@Param			OrderTrackingId			query	string	false	"Provider tracking ID"
//	@Param			OrderMerchantReference	query	string	false	"Merchant reference"
//	@Success		200						{object}	response.Envelope[PesapalIPNCallbackResponse]
//	@Router			/api/v1/payments/ipn/pesapal [get]
//	@Router			/api/v1/payments/ipn/pesapal [post]
func (h *Handler) PesapalIPNCallback(c *gin.Context) {
	var req PesapalIPNCallbackRequest
	if c.Request.Method == "POST" && c.Request.ContentLength > 0 {
		if err := c.ShouldBind(&req); err != nil {
			// Ack anyway to avoid repeated provider retries on parse mismatch.
			response.OK(c, PesapalIPNCallbackResponse{Status: "accepted"})
			return
		}
	} else {
		req = PesapalIPNCallbackRequest{
			OrderNotificationType:  c.Query("OrderNotificationType"),
			OrderTrackingID:        c.Query("OrderTrackingId"),
			OrderMerchantReference: c.Query("OrderMerchantReference"),
		}
	}
	out, err := h.svc.HandlePesapalIPN(c.Request.Context(), &req)
	if err != nil {
		// Return accepted to avoid provider retry storms; error is internal-only.
		response.OK(c, PesapalIPNCallbackResponse{Status: "accepted"})
		return
	}
	response.OK(c, out)
}

// GetTransactionStatus godoc
//
//	@Summary		Check Pesapal transaction status
//	@Description	Queries Pesapal GetTransactionStatus by order tracking ID and syncs local payment status.
//	@Tags			payments
//	@Security		BearerAuth
//	@Produce		json
//	@Param			order_tracking_id	query	string	true	"Pesapal order tracking ID"
//	@Success		200					{object}	response.Envelope[PesapalTransactionStatusResponse]
//	@Failure		400					{object}	response.ErrorBody
//	@Failure		401					{object}	response.ErrorBody
//	@Failure		403					{object}	response.ErrorBody
//	@Failure		500					{object}	response.ErrorBody
//	@Router			/api/v1/payments/transaction-status [get]
func (h *Handler) GetTransactionStatus(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var q TransactionStatusQuery
	if err := validator.BindQuery(c, &q); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.GetTransactionStatus(c.Request.Context(), actor, q.OrderTrackingID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// GeneratePesapalToken godoc
//
//	@Summary		Generate Pesapal token (test)
//	@Description	Generates a fresh Pesapal bearer token for diagnostics/testing.
//	@Tags			payments
//	@Produce		json
//	@Success		200	{object}	response.Envelope[PesapalTokenResponse]
//	@Failure		500	{object}	response.ErrorBody
//	@Router			/api/v1/payments/pesapal/token [post]
func (h *Handler) GeneratePesapalToken(c *gin.Context) {
	out, err := h.svc.GeneratePesapalToken(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}
