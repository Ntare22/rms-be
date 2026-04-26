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
