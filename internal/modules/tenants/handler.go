package tenants

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	apierrors "rms-be/internal/api/errors"
	"rms-be/internal/api/pagination"
	"rms-be/internal/api/response"
	"rms-be/internal/api/validator"
	"rms-be/internal/middleware"
)

// Handler exposes HTTP handlers for tenants.
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

func parseIncludeSummary(c *gin.Context) bool {
	v := strings.ToLower(strings.TrimSpace(c.Query("include_summary")))
	switch v {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

// List godoc
//
//	@Summary		List tenants
//	@Description	**Nested route:** `/api/v1/organizations/{id}/tenants`. **Admin:** any org `id`. **Landlord, manager, staff:** own organization only. **Pagination:** `page`, `page_size` (max 100). **Search (AND):** optional `name`, `email`, `phone` query params — each matches case-insensitively (`ILIKE`) against name fields or email/phone respectively. Optional `status` filter (`active`, `inactive`, `archived`). Set `include_summary=true` to add `active_lease_count` and `current_unit` (from the latest active lease) per tenant.
//	@Tags			tenants
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id			path	string	true	"Organization ID (Gin param name `id`)"	Format(uuid)
//	@Param			page		query	int		false	"Page (1-based)"			minimum(1)		default(1)		example(1)
//	@Param			page_size	query	int		false	"Page size"				maximum(100)	default(20)	example(20)
//	@Param			name		query	string	false	"Search by first/last/full name (partial)"	example(Jordan)
//	@Param			email		query	string	false	"Search by email (partial)"					example(lee@)
//	@Param			phone		query	string	false	"Search by phone (partial)"				example(555)
//	@Param			status		query	string	false	"Filter by tenant status"					example(active)
//	@Param			include_summary	query	string	false	"true/1/yes to include lease summary fields"	example(true)
//	@Success		200	{object}	response.Envelope[TenantListResponse]	"Example item: full_name, email_opt_in, active_lease_count when include_summary=true"
//	@Failure		400	{object}	response.ErrorBody
//	@Failure		401	{object}	response.ErrorBody
//	@Failure		403	{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/tenants [get]
func (h *Handler) List(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var q ListTenantsQuery
	if err := validator.BindQuery(c, &q); err != nil {
		response.Error(c, err)
		return
	}
	p := pagination.FromQuery(c)
	out, err := h.svc.List(c.Request.Context(), actor, c.Param("id"), q.Name, q.Email, q.Phone, q.Status, parseIncludeSummary(c), p.Page, p.Size, p.Offset())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// Create godoc
//
//	@Summary		Create tenant
//	@Description	**Admin / landlord / manager** only. Validates `email` format when provided; `phone` max length 32. If `full_name` is omitted it is derived from first + last name. Default `status` is `active`.
//	@Tags			tenants
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path	string			true	"Organization ID"	Format(uuid)
//	@Param			body	body	CreateTenantRequest	true	"Example: {\"first_name\":\"Jordan\",\"last_name\":\"Lee\",\"email\":\"jordan@example.com\",\"phone\":\"+1-555-0100\",\"email_opt_in\":true,\"locale\":\"en-US\",\"timezone\":\"America/Chicago\"}"
//	@Success		201	{object}	response.Envelope[TenantResponse]
//	@Failure		400	{object}	response.ErrorBody
//	@Failure		401	{object}	response.ErrorBody
//	@Failure		403	{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/tenants [post]
func (h *Handler) Create(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var req CreateTenantRequest
	if err := validator.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.Create(c.Request.Context(), actor, c.Param("id"), &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, out)
}

// Get godoc
//
//	@Summary		Get tenant
//	@Description	Returns a tenant scoped to the organization. Optional `include_summary=true` adds `active_lease_count` and `current_unit` from active leases.
//	@Tags			tenants
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id			path	string	true	"Organization ID"	Format(uuid)
//	@Param			tenantId	path	string	true	"Tenant ID"		Format(uuid)
//	@Param			include_summary	query	string	false	"true/1/yes for lease summary"	example(true)
//	@Success		200	{object}	response.Envelope[TenantResponse]
//	@Failure		401	{object}	response.ErrorBody
//	@Failure		403	{object}	response.ErrorBody
//	@Failure		404	{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/tenants/{tenantId} [get]
func (h *Handler) Get(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.Get(c.Request.Context(), actor, c.Param("id"), c.Param("tenantId"), parseIncludeSummary(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// Patch godoc
//
//	@Summary		Update tenant
//	@Description	**Preferred lifecycle:** set `status` to `inactive` or `archived` instead of deleting when the tenant has lease history. Partial updates supported. Recomputes `full_name` when `first_name` or `last_name` changes unless `full_name` is sent explicitly.
//	@Tags			tenants
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id			path	string			true	"Organization ID"	Format(uuid)
//	@Param			tenantId	path	string			true	"Tenant ID"		Format(uuid)
//	@Param			body	body	PatchTenantRequest	true	"Example: {\"status\":\"inactive\"} or {\"email\":\"new@example.com\",\"phone\":\"+1-555-0199\"}"
//	@Success		200	{object}	response.Envelope[TenantResponse]
//	@Failure		400	{object}	response.ErrorBody
//	@Failure		401	{object}	response.ErrorBody
//	@Failure		403	{object}	response.ErrorBody
//	@Failure		404	{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/tenants/{tenantId} [patch]
func (h *Handler) Patch(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var req PatchTenantRequest
	if err := validator.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.Patch(c.Request.Context(), actor, c.Param("id"), c.Param("tenantId"), &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// Delete godoc
//
//	@Summary		Delete tenant
//	@Description	**Soft-delete** only when the tenant has **no** lease rows (any status). If leases exist, returns **409** `tenant_has_leases` — keep the record and **PATCH** `status` to `inactive` or `archived` instead. **Admin / landlord / manager** only.
//	@Tags			tenants
//	@Security		BearerAuth
//	@Param			id			path	string	true	"Organization ID"	Format(uuid)
//	@Param			tenantId	path	string	true	"Tenant ID"		Format(uuid)
//	@Success		204	"No Content"
//	@Failure		401	{object}	response.ErrorBody
//	@Failure		403	{object}	response.ErrorBody
//	@Failure		404	{object}	response.ErrorBody
//	@Failure		409	{object}	response.ErrorBody	"tenant_has_leases"
//	@Router			/api/v1/organizations/{id}/tenants/{tenantId} [delete]
func (h *Handler) Delete(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), actor, c.Param("id"), c.Param("tenantId")); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
