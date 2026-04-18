package auth

import (
	"github.com/gin-gonic/gin"

	apierrors "rms-be/internal/api/errors"
	"rms-be/internal/api/response"
	"rms-be/internal/api/validator"
	"rms-be/internal/middleware"
)

// Handler exposes HTTP handlers for authentication.
type Handler struct {
	svc *Service
}

// NewHandler constructs a Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Register godoc
//	@Summary		Register organization and first user
//	@Description	Creates an organization and bootstrap user (intended for controlled use in production).
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		RegisterRequest	true	"Registration payload"
//	@Success		201		{object}	response.Envelope[AuthSessionResponse]
//	@Failure		400		{object}	response.ErrorBody
//	@Failure		409		{object}	response.ErrorBody
//	@Router			/api/v1/auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := validator.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.Register(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, out)
}

// Login godoc
//	@Summary		Login
//	@Description	Authenticates with email and password within an organization (by id or slug).
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		LoginRequest	true	"Login payload"
//	@Success		200		{object}	response.Envelope[AuthSessionResponse]	"Example: tokens plus user summary"
//	@Failure		401		{object}	response.ErrorBody
//	@Failure		403		{object}	response.ErrorBody
//	@Router			/api/v1/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := validator.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.Login(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// Refresh godoc
//	@Summary		Refresh tokens
//	@Description	Exchanges a refresh token for a new access and refresh token pair (rotation).
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		RefreshRequest	true	"Refresh token"
//	@Success		200		{object}	response.Envelope[RefreshResponse]
//	@Failure		401		{object}	response.ErrorBody
//	@Failure		403		{object}	response.ErrorBody
//	@Router			/api/v1/auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := validator.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.Refresh(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// Me godoc
//	@Summary		Current user
//	@Description	Returns the authenticated user profile and claim context.
//	@Tags			auth
//	@Security		BearerAuth
//	@Produce		json
//	@Success		200	{object}	response.Envelope[MeResponse]
//	@Failure		401	{object}	response.ErrorBody
//	@Failure		403	{object}	response.ErrorBody
//	@Failure		404	{object}	response.ErrorBody
//	@Router			/api/v1/auth/me [get]
func (h *Handler) Me(c *gin.Context) {
	cl, ok := middleware.ClaimsFromContext(c)
	if !ok || cl == nil {
		response.Error(c, apierrors.ErrUnauthorized)
		return
	}
	out, err := h.svc.Me(c.Request.Context(), cl.UserID, cl.OrganizationID, cl.Role)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}
