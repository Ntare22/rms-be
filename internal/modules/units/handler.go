package units

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

// Handler exposes HTTP handlers for units.
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

func parseIncludeOccupancy(c *gin.Context) bool {
	v := strings.ToLower(strings.TrimSpace(c.Query("include_occupancy")))
	switch v {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

// List godoc
//
//	@Summary		List units in a building
//	@Description	**Nested route:** `organizations` → `buildings` → `units`. **Admin:** any organization `id`. **Landlord, manager, staff:** only resources in their token organization. Paginated (`page`, `page_size`, max 100). Optional `status` filter (`vacant`, `occupied`, `offline`, `maintenance`). Set `include_occupancy=true` to add a derived `occupancy` object per unit (active lease summary when present).
//	@Tags			units
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id			path	string	true	"Organization ID (same wildcard as other org routes)"	Format(uuid)
//	@Param			buildingId	path	string	true	"Building ID (must belong to organization)"	Format(uuid)
//	@Param			page		query	int		false	"Page (1-based)"			minimum(1)		default(1)		example(1)
//	@Param			page_size	query	int		false	"Page size"				maximum(100)	default(20)	example(20)
//	@Param			status		query	string	false	"Filter by unit status"	example(vacant)
//	@Param			include_occupancy	query	string	false	"Set to true/1/yes to include occupancy summary per unit"	example(true)
//	@Success		200	{object}	response.Envelope[UnitListResponse]	"Example items include unit_label, currency, default_rent_amount_minor; occupancy when requested"
//	@Failure		400	{object}	response.ErrorBody
//	@Failure		401	{object}	response.ErrorBody
//	@Failure		403	{object}	response.ErrorBody
//	@Failure		404	{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/buildings/{buildingId}/units [get]
func (h *Handler) List(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var q ListUnitsQuery
	if err := validator.BindQuery(c, &q); err != nil {
		response.Error(c, err)
		return
	}
	p := pagination.FromQuery(c)
	out, err := h.svc.List(c.Request.Context(), actor, c.Param("id"), c.Param("buildingId"), q.Status, parseIncludeOccupancy(c), p.Page, p.Size, p.Offset())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// Create godoc
//
//	@Summary		Create unit
//	@Description	**Admin / landlord / manager:** create a unit under the given building. **Staff** cannot create. `unit_label` must be unique per building. Defaults: `currency` USD, `status` vacant.
//	@Tags			units
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id			path	string			true	"Organization ID"	Format(uuid)
//	@Param			buildingId	path	string			true	"Building ID"		Format(uuid)
//	@Param			body	body	CreateUnitRequest	true	"Example: {\"unit_label\":\"12B\",\"bedrooms\":2,\"default_rent_amount_minor\":175000,\"currency\":\"USD\",\"status\":\"vacant\"}"
//	@Success		201	{object}	response.Envelope[UnitResponse]
//	@Failure		400	{object}	response.ErrorBody
//	@Failure		401	{object}	response.ErrorBody
//	@Failure		403	{object}	response.ErrorBody
//	@Failure		404	{object}	response.ErrorBody
//	@Failure		409	{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/buildings/{buildingId}/units [post]
func (h *Handler) Create(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var req CreateUnitRequest
	if err := validator.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.Create(c.Request.Context(), actor, c.Param("id"), c.Param("buildingId"), &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, out)
}

// Get godoc
//
//	@Summary		Get unit
//	@Description	Returns a single unit scoped to organization and building. Optional `include_occupancy=true` adds lease-derived occupancy.
//	@Tags			units
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id			path	string	true	"Organization ID"	Format(uuid)
//	@Param			buildingId	path	string	true	"Building ID"		Format(uuid)
//	@Param			unitId		path	string	true	"Unit ID"			Format(uuid)
//	@Param			include_occupancy	query	string	false	"true/1/yes for occupancy block"	example(true)
//	@Success		200	{object}	response.Envelope[UnitResponse]	"Example: unit with currency USD and optional occupancy.has_active_lease"
//	@Failure		401	{object}	response.ErrorBody
//	@Failure		403	{object}	response.ErrorBody
//	@Failure		404	{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/buildings/{buildingId}/units/{unitId} [get]
func (h *Handler) Get(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.Get(c.Request.Context(), actor, c.Param("id"), c.Param("buildingId"), c.Param("unitId"), parseIncludeOccupancy(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// Patch godoc
//
//	@Summary		Update unit
//	@Description	**Admin / landlord / manager** may update. **Staff** may not. Example: `{\"status\":\"maintenance\",\"default_rent_amount_minor\":190000}`.
//	@Tags			units
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id			path	string			true	"Organization ID"	Format(uuid)
//	@Param			buildingId	path	string			true	"Building ID"		Format(uuid)
//	@Param			unitId		path	string			true	"Unit ID"			Format(uuid)
//	@Param			body	body	PatchUnitRequest	true	"Partial update"
//	@Success		200	{object}	response.Envelope[UnitResponse]
//	@Failure		400	{object}	response.ErrorBody
//	@Failure		401	{object}	response.ErrorBody
//	@Failure		403	{object}	response.ErrorBody
//	@Failure		404	{object}	response.ErrorBody
//	@Failure		409	{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/buildings/{buildingId}/units/{unitId} [patch]
func (h *Handler) Patch(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var req PatchUnitRequest
	if err := validator.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.Patch(c.Request.Context(), actor, c.Param("id"), c.Param("buildingId"), c.Param("unitId"), &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// Delete godoc
//
//	@Summary		Delete unit
//	@Description	Soft-deletes the unit. **409** if a lease with status `active` exists for this unit (no force-delete). **Admin / landlord / manager** only.
//	@Tags			units
//	@Security		BearerAuth
//	@Param			id			path	string	true	"Organization ID"	Format(uuid)
//	@Param			buildingId	path	string	true	"Building ID"		Format(uuid)
//	@Param			unitId		path	string	true	"Unit ID"			Format(uuid)
//	@Success		204	"No Content"
//	@Failure		401	{object}	response.ErrorBody
//	@Failure		403	{object}	response.ErrorBody
//	@Failure		404	{object}	response.ErrorBody
//	@Failure		409	{object}	response.ErrorBody	"unit_has_active_lease"
//	@Router			/api/v1/organizations/{id}/buildings/{buildingId}/units/{unitId} [delete]
func (h *Handler) Delete(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), actor, c.Param("id"), c.Param("buildingId"), c.Param("unitId")); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
