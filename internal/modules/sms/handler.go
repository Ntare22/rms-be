package sms

import (
	"github.com/gin-gonic/gin"

	"rms-be/internal/api/response"
	"rms-be/internal/api/validator"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Send godoc
//
//	@Summary		Send single SMS
//	@Description	Sends a single SMS message through EGO SMS.
//	@Tags			sms
//	@Accept			json
//	@Produce		json
//	@Param			body	body	SendSMSRequest	true	"Single SMS payload"
//	@Success		200		{object}	response.Envelope[SendSMSResponse]
//	@Failure		400		{object}	response.ErrorBody
//	@Failure		429		{object}	response.ErrorBody
//	@Failure		502		{object}	response.ErrorBody
//	@Failure		504		{object}	response.ErrorBody
//	@Router			/api/v1/sms/send [post]
func (h *Handler) Send(c *gin.Context) {
	var req SendSMSRequest
	if err := validator.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.Send(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// BulkSend godoc
//
//	@Summary		Bulk send SMS
//	@Description	Sends the same message to multiple recipients.
//	@Tags			sms
//	@Accept			json
//	@Produce		json
//	@Param			body	body	BulkSendSMSRequest	true	"Bulk SMS payload"
//	@Success		200		{object}	response.Envelope[BulkSendSMSResponse]
//	@Failure		400		{object}	response.ErrorBody
//	@Failure		429		{object}	response.ErrorBody
//	@Failure		502		{object}	response.ErrorBody
//	@Failure		504		{object}	response.ErrorBody
//	@Router			/api/v1/sms/bulk-send [post]
func (h *Handler) BulkSend(c *gin.Context) {
	var req BulkSendSMSRequest
	if err := validator.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.BulkSend(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// SendTemplate godoc
//
//	@Summary		Send template SMS
//	@Description	Renders a message template and sends SMS.
//	@Tags			sms
//	@Accept			json
//	@Produce		json
//	@Param			body	body	SendTemplateSMSRequest	true	"Template SMS payload"
//	@Success		200		{object}	response.Envelope[SendSMSResponse]
//	@Failure		400		{object}	response.ErrorBody
//	@Failure		429		{object}	response.ErrorBody
//	@Failure		502		{object}	response.ErrorBody
//	@Failure		504		{object}	response.ErrorBody
//	@Router			/api/v1/sms/send-template [post]
func (h *Handler) SendTemplate(c *gin.Context) {
	var req SendTemplateSMSRequest
	if err := validator.BindJSON(c, &req); err != nil {
		response.Error(c, err)
		return
	}
	out, err := h.svc.SendTemplate(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// Health godoc
//
//	@Summary		SMS provider health
//	@Description	Checks EGO SMS provider connectivity and credentials.
//	@Tags			sms
//	@Produce		json
//	@Success		200	{object}	response.Envelope[SMSHealthResponse]
//	@Failure		502	{object}	response.ErrorBody
//	@Failure		504	{object}	response.ErrorBody
//	@Router			/api/v1/sms/health [get]
func (h *Handler) Health(c *gin.Context) {
	out, err := h.svc.Health(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}
