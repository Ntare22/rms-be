package response

import (
	stderrors "errors"
	"net/http"

	"github.com/gin-gonic/gin"

	apierrors "rms-be/internal/api/errors"
)

// Envelope is the standard JSON envelope for successful responses.
type Envelope[T any] struct {
	Data T `json:"data"`
}

// ErrorBody is the standard JSON shape for errors.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail describes an error response.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Kind    string `json:"kind,omitempty"`
}

// OK writes a 200 response with data.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope[any]{Data: data})
}

// Created writes a 201 response with data.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Envelope[any]{Data: data})
}

// JSON writes an arbitrary status with a JSON body.
func JSON(c *gin.Context, status int, body any) {
	c.JSON(status, body)
}

// Error writes an error response from AppError or falls back to 500.
func Error(c *gin.Context, err error) {
	var ae *apierrors.AppError
	if err != nil && stderrors.As(err, &ae) {
		c.JSON(ae.Status, ErrorBody{Error: ErrorDetail{
			Code: ae.Code, Message: ae.Message, Kind: string(ae.Kind),
		}})
		return
	}
	c.JSON(http.StatusInternalServerError, ErrorBody{Error: ErrorDetail{
		Code: apierrors.ErrInternal.Code, Message: apierrors.ErrInternal.Message, Kind: string(apierrors.ErrInternal.Kind),
	}})
}
