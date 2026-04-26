package leases

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	apierrors "rms-be/internal/api/errors"
	"rms-be/internal/api/pagination"
	"rms-be/internal/api/response"
	"rms-be/internal/api/validator"
	"rms-be/internal/middleware"
)

// Handler exposes HTTP handlers for leases.
type Handler struct {
	svc *Service
}

// NewHandler constructs a Handler.
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

func listFiltersFromQuery(c *gin.Context) (ListFilters, error) {
	var q ListLeasesQuery
	if err := validator.BindQuery(c, &q); err != nil {
		return ListFilters{}, err
	}
	var f ListFilters
	if t := strings.TrimSpace(q.Status); t != "" {
		st := parseLeaseStatus(t)
		if st == "" {
			return ListFilters{}, apierrors.ErrValidation
		}
		f.Status = &st
	}
	f.BuildingID = q.BuildingID
	f.UnitID = q.UnitID
	f.TenantID = q.TenantID
	if raw := strings.TrimSpace(q.ActiveOn); raw != "" {
		day, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return ListFilters{}, apierrors.ErrValidation
		}
		f.ActiveOn = &day
	}
	return f, nil
}

// List godoc
//
//	@Summary		List leases
//	@Description	Organization-scoped list with **pagination** (`page`, `page_size`, max 100). **Filters:** `status`, `building_id` (via unit), `unit_id`, `tenant_id`, `active_on` (date `YYYY-MM-DD` — leases **active** for any time on that UTC calendar day). **Overlap rule:** creating/updating an **active + primary** lease is rejected if its `[start_date,end_date)` range intersects another active primary lease on the same unit (mirrors DB EXCLUDE constraint).
//	@Tags			leases
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id			path	string	true	"Organization ID"	Format(uuid)
//	@Param			page		query	int		false	"Page (1-based)"	minimum(1)	default(1)	example(1)
//	@Param			page_size	query	int		false	"Page size"		maximum(100)	default(20)	example(20)
//	@Param			status		query	string	false	"draft | active | ended | cancelled"	example(active)
//	@Param			building_id	query	string	false	"Filter by building (units in this building)"	Format(uuid)
//	@Param			unit_id		query	string	false	"Filter by unit"	Format(uuid)
//	@Param			tenant_id	query	string	false	"Filter by tenant"	Format(uuid)
//	@Param			active_on	query	string	false	"Date YYYY-MM-DD (UTC day) when lease must be active"	example(2026-04-18)
//	@Success		200	{object}	response.Envelope[LeaseListResponse]
//	@Failure		400	{object}	response.ErrorBody
//	@Failure		401	{object}	response.ErrorBody
//	@Failure		403	{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/leases [get]
func (h *Handler) List(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	f, err := listFiltersFromQuery(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	p := pagination.FromQuery(c)
	out, err := h.svc.List(c.Request.Context(), actor, c.Param("id"), f, p.Page, p.Size, p.Offset())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// Create godoc
//
//	@Summary		Create lease
//	@Description	**Draft** is allowed. **Active + primary** leases cannot overlap in time on the same unit (enforced here and by SQL EXCLUDE). Tenant and unit must belong to the organization. When `status` is **active**, the unit becomes **occupied** if at least one active lease exists. **Audit:** `lease.created` is recorded.
//	@Tags			leases
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path	string			true	"Organization ID"	Format(uuid)
//	@Param			body	body	CreateLeaseRequest	true	"Example: {\"tenant_id\":\"...\",\"unit_id\":\"...\",\"start_date\":\"2026-04-01T00:00:00Z\",\"monthly_rent_amount_minor\":150000,\"currency\":\"USD\",\"status\":\"draft\",\"is_primary\":true}"
//	@Success		201	{object}	response.Envelope[LeaseResponse]
//	@Failure		400	{object}	response.ErrorBody
//	@Failure		401	{object}	response.ErrorBody
//	@Failure		403	{object}	response.ErrorBody
//	@Failure		404	{object}	response.ErrorBody
//	@Failure		409	{object}	response.ErrorBody	"lease_overlap"
//	@Router			/api/v1/organizations/{id}/leases [post]
func (h *Handler) Create(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var req CreateLeaseRequest
	if err := validator.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.Create(c.Request.Context(), actor, c.Param("id"), &req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, out)
}

// Get godoc
//
//	@Summary		Get lease
//	@Tags			leases
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id		path	string	true	"Organization ID"	Format(uuid)
//	@Param			leaseId	path	string	true	"Lease ID"	Format(uuid)
//	@Success		200	{object}	response.Envelope[LeaseResponse]
//	@Failure		401	{object}	response.ErrorBody
//	@Failure		403	{object}	response.ErrorBody
//	@Failure		404	{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/leases/{leaseId} [get]
func (h *Handler) Get(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.Get(c.Request.Context(), actor, c.Param("id"), c.Param("leaseId"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// Patch godoc
//
//	@Summary		Update lease
//	@Description	State transitions (e.g. **draft → active**) run overlap checks for **active + primary** leases on the same unit. Changing dates on an active primary lease also re-validates overlap. Unit occupancy is recalculated after save. **Audit:** `lease.updated`.
//	@Tags			leases
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path	string			true	"Organization ID"	Format(uuid)
//	@Param			leaseId	path	string			true	"Lease ID"	Format(uuid)
//	@Param			body	body	PatchLeaseRequest	true	"Example: {\"status\":\"active\"} or {\"end_date\":\"2027-03-31T23:59:59Z\"}"
//	@Success		200	{object}	response.Envelope[LeaseResponse]
//	@Failure		400	{object}	response.ErrorBody
//	@Failure		401	{object}	response.ErrorBody
//	@Failure		403	{object}	response.ErrorBody
//	@Failure		404	{object}	response.ErrorBody
//	@Failure		409	{object}	response.ErrorBody	"lease_overlap"
//	@Router			/api/v1/organizations/{id}/leases/{leaseId} [patch]
func (h *Handler) Patch(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var req PatchLeaseRequest
	if err := validator.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.Patch(c.Request.Context(), actor, c.Param("id"), c.Param("leaseId"), &req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// End godoc
//
//	@Summary		End lease
//	@Description	Sets `status` to **ended** and `end_date` (request body or server time). Recalculates unit occupancy: when **no** active leases remain on the unit, status becomes **vacant**. Prefer this over deleting tenants with history. **Audit:** `lease.ended`.
//	@Tags			leases
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path	string			true	"Organization ID"	Format(uuid)
//	@Param			leaseId	path	string			true	"Lease ID"	Format(uuid)
//	@Param			body	body	EndLeaseRequest		false	"Optional end_date (RFC3339); defaults to now"
//	@Success		200	{object}	response.Envelope[LeaseResponse]
//	@Failure		400	{object}	response.ErrorBody
//	@Failure		401	{object}	response.ErrorBody
//	@Failure		403	{object}	response.ErrorBody
//	@Failure		404	{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/leases/{leaseId}/end [post]
func (h *Handler) End(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var req EndLeaseRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, apierrors.Wrap(err, apierrors.ErrValidation))
			return
		}
	}
	if err := validator.Struct(&req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.End(c.Request.Context(), actor, c.Param("id"), c.Param("leaseId"), &req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// Approve godoc
//
//	@Summary		Approve lease
//	@Description	Tenant approves a pending lease. Lease transitions to active and unit occupancy is recalculated.
//	@Tags			leases
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path	string					true	"Organization ID"	Format(uuid)
//	@Param			leaseId	path	string					true	"Lease ID"		Format(uuid)
//	@Param			body	body	LeaseDecisionRequest	false	"Optional decision note"
//	@Success		200		{object}	response.Envelope[LeaseResponse]
//	@Failure		400		{object}	response.ErrorBody
//	@Failure		401		{object}	response.ErrorBody
//	@Failure		403		{object}	response.ErrorBody
//	@Failure		404		{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/leases/{leaseId}/approve [post]
func (h *Handler) Approve(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var req LeaseDecisionRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, apierrors.Wrap(err, apierrors.ErrValidation))
			return
		}
	}
	if err := validator.Struct(&req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.Approve(c.Request.Context(), actor, c.Param("id"), c.Param("leaseId"), &req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// Reject godoc
//
//	@Summary		Reject lease
//	@Description	Tenant rejects a pending lease. Lease transitions to rejected and occupancy is recalculated.
//	@Tags			leases
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path	string					true	"Organization ID"	Format(uuid)
//	@Param			leaseId	path	string					true	"Lease ID"		Format(uuid)
//	@Param			body	body	LeaseDecisionRequest	false	"Optional decision note"
//	@Success		200		{object}	response.Envelope[LeaseResponse]
//	@Failure		400		{object}	response.ErrorBody
//	@Failure		401		{object}	response.ErrorBody
//	@Failure		403		{object}	response.ErrorBody
//	@Failure		404		{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/leases/{leaseId}/reject [post]
func (h *Handler) Reject(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var req LeaseDecisionRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, apierrors.Wrap(err, apierrors.ErrValidation))
			return
		}
	}
	if err := validator.Struct(&req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.Reject(c.Request.Context(), actor, c.Param("id"), c.Param("leaseId"), &req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// History godoc
//
//	@Summary		Lease history
//	@Tags			leases
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id		path	string	true	"Organization ID"	Format(uuid)
//	@Param			leaseId	path	string	true	"Lease ID"	Format(uuid)
//	@Success		200		{object}	response.Envelope[LeaseHistoryResponse]
//	@Failure		401		{object}	response.ErrorBody
//	@Failure		403		{object}	response.ErrorBody
//	@Failure		404		{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/leases/{leaseId}/history [get]
func (h *Handler) History(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.History(c.Request.Context(), actor, c.Param("id"), c.Param("leaseId"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// CreateRenewalOffer godoc
//
//	@Summary		Create lease renewal offer
//	@Tags			leases
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path	string						true	"Organization ID"	Format(uuid)
//	@Param			leaseId	path	string						true	"Lease ID"	Format(uuid)
//	@Param			body	body	LeaseRenewalCreateRequest	true	"Renewal offer payload"
//	@Success		201		{object}	response.Envelope[LeaseRenewalOfferResponse]
//	@Failure		400		{object}	response.ErrorBody
//	@Failure		401		{object}	response.ErrorBody
//	@Failure		403		{object}	response.ErrorBody
//	@Failure		404		{object}	response.ErrorBody
//	@Failure		500		{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/leases/{leaseId}/renewals [post]
func (h *Handler) CreateRenewalOffer(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var req LeaseRenewalCreateRequest
	if err := validator.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.CreateRenewalOffer(c.Request.Context(), actor, c.Param("id"), c.Param("leaseId"), &req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, out)
}

// AcceptRenewalOffer godoc
//
//	@Summary		Accept renewal offer
//	@Tags			leases
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path	string							true	"Organization ID"	Format(uuid)
//	@Param			leaseId	path	string							true	"Lease ID"	Format(uuid)
//	@Param			offerId	path	string							true	"Offer ID"	Format(uuid)
//	@Param			body	body	LeaseRenewalDecisionRequest	false	"Optional decision note"
//	@Success		200		{object}	response.Envelope[LeaseRenewalOfferResponse]
//	@Failure		400		{object}	response.ErrorBody
//	@Failure		401		{object}	response.ErrorBody
//	@Failure		403		{object}	response.ErrorBody
//	@Failure		404		{object}	response.ErrorBody
//	@Failure		500		{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/leases/{leaseId}/renewals/{offerId}/accept [post]
func (h *Handler) AcceptRenewalOffer(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var req LeaseRenewalDecisionRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, apierrors.Wrap(err, apierrors.ErrValidation))
			return
		}
	}
	if err := validator.Struct(&req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.AcceptRenewalOffer(c.Request.Context(), actor, c.Param("id"), c.Param("leaseId"), c.Param("offerId"), &req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// RejectRenewalOffer godoc
//
//	@Summary		Reject renewal offer
//	@Tags			leases
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path	string							true	"Organization ID"	Format(uuid)
//	@Param			leaseId	path	string							true	"Lease ID"	Format(uuid)
//	@Param			offerId	path	string							true	"Offer ID"	Format(uuid)
//	@Param			body	body	LeaseRenewalDecisionRequest	false	"Optional decision note"
//	@Success		200		{object}	response.Envelope[LeaseRenewalOfferResponse]
//	@Failure		400		{object}	response.ErrorBody
//	@Failure		401		{object}	response.ErrorBody
//	@Failure		403		{object}	response.ErrorBody
//	@Failure		404		{object}	response.ErrorBody
//	@Failure		500		{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/leases/{leaseId}/renewals/{offerId}/reject [post]
func (h *Handler) RejectRenewalOffer(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var req LeaseRenewalDecisionRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, apierrors.Wrap(err, apierrors.ErrValidation))
			return
		}
	}
	if err := validator.Struct(&req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.RejectRenewalOffer(c.Request.Context(), actor, c.Param("id"), c.Param("leaseId"), c.Param("offerId"), &req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// Closeout godoc
//
//	@Summary		Lease closeout
//	@Tags			leases
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path	string				true	"Organization ID"	Format(uuid)
//	@Param			leaseId	path	string				true	"Lease ID"	Format(uuid)
//	@Param			body	body	LeaseCloseoutRequest	true	"Closeout payload"
//	@Success		201		{object}	response.Envelope[LeaseCloseoutResponse]
//	@Failure		400		{object}	response.ErrorBody
//	@Failure		401		{object}	response.ErrorBody
//	@Failure		403		{object}	response.ErrorBody
//	@Failure		404		{object}	response.ErrorBody
//	@Failure		500		{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/leases/{leaseId}/closeout [post]
func (h *Handler) Closeout(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var req LeaseCloseoutRequest
	if err := validator.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.Closeout(c.Request.Context(), actor, c.Param("id"), c.Param("leaseId"), &req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, out)
}

// TenantStatement godoc
//
//	@Summary		Tenant statement
//	@Tags			leases
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id		path	string	true	"Organization ID"	Format(uuid)
//	@Param			tenantId	path	string	true	"Tenant ID"	Format(uuid)
//	@Success		200		{object}	response.Envelope[TenantStatementResponse]
//	@Failure		401		{object}	response.ErrorBody
//	@Failure		403		{object}	response.ErrorBody
//	@Failure		500		{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/leases/tenants/{tenantId}/statement [get]
func (h *Handler) TenantStatement(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.TenantStatement(c.Request.Context(), actor, c.Param("id"), c.Param("tenantId"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// TenantStatementExport godoc
//
//	@Summary		Export tenant statement CSV
//	@Tags			leases
//	@Security		BearerAuth
//	@Produce		text/csv
//	@Param			id		path	string	true	"Organization ID"	Format(uuid)
//	@Param			tenantId	path	string	true	"Tenant ID"	Format(uuid)
//	@Success		200		{string}	string	"CSV file"
//	@Failure		401		{object}	response.ErrorBody
//	@Failure		403		{object}	response.ErrorBody
//	@Failure		500		{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/leases/tenants/{tenantId}/statement/export [get]
func (h *Handler) TenantStatementExport(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.TenantStatement(c.Request.Context(), actor, c.Param("id"), c.Param("tenantId"))
	if err != nil {
		response.Error(c, err)
		return
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"kind", "reference_id", "amount_minor", "currency", "occurred_at", "description"})
	for i := range out.Items {
		_ = w.Write([]string{
			out.Items[i].Kind,
			out.Items[i].ReferenceID,
			fmt.Sprintf("%d", out.Items[i].AmountMinor),
			out.Items[i].Currency,
			out.Items[i].OccurredAt.UTC().Format(time.RFC3339),
			out.Items[i].Description,
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		response.Error(c, apierrors.Wrap(err, apierrors.ErrInternal))
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=tenant_statement.csv")
	c.String(http.StatusOK, buf.String())
}
