package buildings

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apierrors "rms-be/internal/api/errors"
	"rms-be/internal/api/pagination"
	"rms-be/internal/api/response"
	"rms-be/internal/api/validator"
	"rms-be/internal/middleware"
)

// Handler exposes HTTP handlers for buildings.
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

// List godoc
//
//	@Summary		List buildings
//	@Description	**Admin:** any organization `id`. **Landlord, manager, staff:** only buildings in their own organization. Paginated via `page` and `page_size` (max 100). Optional `status` filter: `active`, `inactive`, or `archived`.
//	@Tags			buildings
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id			path	string	true	"Organization ID"	Format(uuid)
//	@Param			page		query	int		false	"Page (1-based)"			minimum(1)		default(1)		example(1)
//	@Param			page_size	query	int		false	"Page size"				maximum(100)	default(20)	example(20)
//	@Param			status		query	string	false	"Filter: active, inactive, or archived"	example(active)
//	@Success		200	{object}	response.Envelope[BuildingListResponse]
//	@Failure		400	{object}	response.ErrorBody
//	@Failure		401	{object}	response.ErrorBody
//	@Failure		403	{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/buildings [get]
func (h *Handler) List(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var q ListBuildingsQuery
	if err := validator.BindQuery(c, &q); err != nil {
		response.Error(c, err)
		return
	}
	p := pagination.FromQuery(c)
	out, err := h.svc.List(c.Request.Context(), actor, c.Param("id"), q.Status, p.Page, p.Size, p.Offset())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// Create godoc
//
//	@Summary		Create building
//	@Description	**Admin:** create in any organization. **Landlord / manager:** create only in their own organization. **Staff** cannot create. Sets audit `created_by` / `updated_by` from the token subject.
//	@Tags			buildings
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path	string					true	"Organization ID"	Format(uuid)
//	@Param			body	body	CreateBuildingRequest	true	"Example: {\"name\":\"Riverside Tower\",\"country\":\"US\",\"timezone\":\"America/Chicago\",\"status\":\"active\"}"
//	@Success		201	{object}	response.Envelope[BuildingResponse]
//	@Failure		400	{object}	response.ErrorBody
//	@Failure		401	{object}	response.ErrorBody
//	@Failure		403	{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/buildings [post]
func (h *Handler) Create(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var req CreateBuildingRequest
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
//	@Summary		Get building
//	@Description	**Admin:** any org. **Landlord, manager, staff:** building must belong to their organization (enforced by path `id` vs token). Returns 404 if the building is not in the organization.
//	@Tags			buildings
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id			path	string	true	"Organization ID"	Format(uuid)
//	@Param			buildingId	path	string	true	"Building ID"		Format(uuid)
//	@Success		200	{object}	response.Envelope[BuildingResponse]
//	@Failure		401	{object}	response.ErrorBody
//	@Failure		403	{object}	response.ErrorBody
//	@Failure		404	{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/buildings/{buildingId} [get]
func (h *Handler) Get(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.Get(c.Request.Context(), actor, c.Param("id"), c.Param("buildingId"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// Patch godoc
//
//	@Summary		Update building
//	@Description	**Admin:** update any org’s building. **Landlord / manager:** only buildings in their org. **Staff** cannot update. Example body: `{\"status\":\"inactive\",\"address_line_1\":\"200 Oak Ave\"}`.
//	@Tags			buildings
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id			path	string					true	"Organization ID"	Format(uuid)
//	@Param			buildingId	path	string					true	"Building ID"		Format(uuid)
//	@Param			body		body	PatchBuildingRequest	true	"Fields to update"
//	@Success		200	{object}	response.Envelope[BuildingResponse]
//	@Failure		400	{object}	response.ErrorBody
//	@Failure		401	{object}	response.ErrorBody
//	@Failure		403	{object}	response.ErrorBody
//	@Failure		404	{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/buildings/{buildingId} [patch]
func (h *Handler) Patch(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var req PatchBuildingRequest
	if err := validator.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.Patch(c.Request.Context(), actor, c.Param("id"), c.Param("buildingId"), &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// Delete godoc
//
//	@Summary		Delete building
//	@Description	**Admin:** delete in any org. **Landlord / manager:** only in their org. **Staff** cannot delete. Soft-deletes the building. **Conflict** if any units still reference the building (no force-delete flag).
//	@Tags			buildings
//	@Security		BearerAuth
//	@Param			id			path	string	true	"Organization ID"	Format(uuid)
//	@Param			buildingId	path	string	true	"Building ID"		Format(uuid)
//	@Success		204	"No Content"
//	@Failure		401	{object}	response.ErrorBody
//	@Failure		403	{object}	response.ErrorBody
//	@Failure		404	{object}	response.ErrorBody
//	@Failure		409	{object}	response.ErrorBody	"building_has_units"
//	@Router			/api/v1/organizations/{id}/buildings/{buildingId} [delete]
func (h *Handler) Delete(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), actor, c.Param("id"), c.Param("buildingId")); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
