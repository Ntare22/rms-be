package organizations

import (
	"github.com/gin-gonic/gin"

	apierrors "rms-be/internal/api/errors"
	"rms-be/internal/api/response"
	"rms-be/internal/api/validator"
	"rms-be/internal/middleware"
)

// Handler exposes HTTP handlers for organizations.
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

// Create godoc
//
//	@Summary		Create organization
//	@Description	**Admin only.** Creates a new landlord / property-management company (tenant). Sets `created_by` / `updated_by` from the access token subject.
//	@Tags			organizations
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		CreateOrganizationRequest	true	"Organization payload"
//	@Success		201		{object}	response.Envelope[OrganizationResponse]
//	@Failure		400		{object}	response.ErrorBody
//	@Failure		401		{object}	response.ErrorBody
//	@Failure		403		{object}	response.ErrorBody	"Non-admin callers are rejected"
//	@Failure		409		{object}	response.ErrorBody
//	@Router			/api/v1/organizations [post]
func (h *Handler) Create(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var req CreateOrganizationRequest
	if err := validator.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.Create(c.Request.Context(), actor, &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, out)
}

// Get godoc
//
//	@Summary		Get organization
//	@Description	**Admin:** any organization id. **Landlord, manager, staff:** only their own `organization_id` from the token; cross-tenant access returns 403.
//	@Tags			organizations
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path		string	true	"Organization ID"	Format(uuid)	example(550e8400-e29b-41d4-a716-446655440000)
//	@Success		200	{object}	response.Envelope[OrganizationResponse]
//	@Failure		400	{object}	response.ErrorBody
//	@Failure		401	{object}	response.ErrorBody
//	@Failure		403	{object}	response.ErrorBody
//	@Failure		404	{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.Get(c.Request.Context(), actor, c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// Patch godoc
//
//	@Summary		Update organization profile
//	@Description	**Admin:** may update any organization. **Landlord and manager:** may update only their own organization. **Staff** cannot use this endpoint. Slug is immutable. Sets `updated_by` from the token subject.
//	@Tags			organizations
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Organization ID"	Format(uuid)
//	@Param			body	body		PatchOrganizationRequest	true	"Fields to update (omit to leave unchanged)"
//	@Success		200		{object}	response.Envelope[OrganizationResponse]
//	@Failure		400		{object}	response.ErrorBody
//	@Failure		401		{object}	response.ErrorBody
//	@Failure		403		{object}	response.ErrorBody
//	@Failure		404		{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id} [patch]
func (h *Handler) Patch(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var req PatchOrganizationRequest
	if err := validator.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.Patch(c.Request.Context(), actor, c.Param("id"), &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}
