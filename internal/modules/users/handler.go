package users

import (
	"github.com/gin-gonic/gin"

	apierrors "rms-be/internal/api/errors"
	"rms-be/internal/api/pagination"
	"rms-be/internal/api/response"
	"rms-be/internal/api/validator"
	"rms-be/internal/middleware"
)

// Handler exposes HTTP handlers for organization users.
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
//	@Summary		List organization users
//	@Description	**Admin:** any organization `id`. **Landlord / manager:** only users in their own organization (`id` must match the token). **Staff** is not allowed. Paginated with `page` and `page_size` (max 100).
//	@Tags			users
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path	string	true	"Organization ID"	Format(uuid)	example(550e8400-e29b-41d4-a716-446655440000)
//	@Param			page		query		int		false	"Page number (1-based)"	minimum(1)	default(1)	example(1)
//	@Param			page_size	query		int		false	"Page size"					maximum(100)	default(20)	example(20)
//	@Success		200	{object}	response.Envelope[UserListResponse]
//	@Failure		400	{object}	response.ErrorBody
//	@Failure		401	{object}	response.ErrorBody
//	@Failure		403	{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/users [get]
func (h *Handler) List(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	p := pagination.FromQuery(c)
	out, err := h.svc.List(c.Request.Context(), actor, c.Param("id"), p.Page, p.Size, p.Offset())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// Create godoc
//
//	@Summary		Create organization user
//	@Description	**Admin:** may create users in any organization with any role. **Landlord:** may create `manager` or `staff` in their org. **Manager:** may create `staff` only. Password is required unless `status` is `invited` (server generates a temporary password). Email must be unique per organization.
//	@Tags			users
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id	path	string	true	"Organization ID"	Format(uuid)
//	@Param			body	body		CreateUserRequest	true	"Create payload"
//	@Success		201		{object}	response.Envelope[UserResponse]
//	@Failure		400		{object}	response.ErrorBody
//	@Failure		401		{object}	response.ErrorBody
//	@Failure		403		{object}	response.ErrorBody
//	@Failure		409		{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/users [post]
func (h *Handler) Create(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var req CreateUserRequest
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

// Patch godoc
//
//	@Summary		Update organization user
//	@Description	**Admin:** may update any user in any org. **Landlord:** may update `manager` and `staff` users in their org (not other landlords or admins). **Manager:** may update `staff` only. Optional password is re-hashed when provided.
//	@Tags			users
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path	string	true	"Organization ID"	Format(uuid)
//	@Param			userId	path	string	true	"User ID"			Format(uuid)
//	@Param			body	body		PatchUserRequest		true	"Example: {\"status\":\"disabled\"} or {\"role\":\"manager\",\"phone\":\"+1-555-0100\"}"
//	@Success		200		{object}	response.Envelope[UserResponse]
//	@Failure		400		{object}	response.ErrorBody
//	@Failure		401		{object}	response.ErrorBody
//	@Failure		403		{object}	response.ErrorBody
//	@Failure		404		{object}	response.ErrorBody
//	@Failure		409		{object}	response.ErrorBody
//	@Router			/api/v1/organizations/{id}/users/{userId} [patch]
func (h *Handler) Patch(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	var req PatchUserRequest
	if err := validator.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.Patch(c.Request.Context(), actor, c.Param("id"), c.Param("userId"), &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}
