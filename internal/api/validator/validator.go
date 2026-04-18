package validator

import (
	stderrors "errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	apierrors "rms-be/internal/api/errors"
)

var validate = validator.New()

// Struct validates a struct using go-playground/validator field tags.
func Struct(v any) error {
	if err := validate.Struct(v); err != nil {
		return apierrors.Wrap(err, apierrors.ErrValidation)
	}
	return nil
}

// BindJSON decodes JSON then runs validator.Struct on the destination.
func BindJSON(c *gin.Context, dst any) error {
	if err := c.ShouldBindJSON(dst); err != nil {
		return apierrors.Wrap(err, apierrors.ErrValidation)
	}
	if err := Struct(dst); err != nil {
		return err
	}
	return nil
}

// BindQuery binds query parameters then validates struct tags.
func BindQuery(c *gin.Context, dst any) error {
	if err := c.ShouldBindQuery(dst); err != nil {
		return apierrors.Wrap(err, apierrors.ErrValidation)
	}
	if err := Struct(dst); err != nil {
		return err
	}
	return nil
}

// FirstMessage returns a short validation message if err is validator.ValidationErrors.
func FirstMessage(err error) string {
	var verrs validator.ValidationErrors
	if !stderrors.As(err, &verrs) || len(verrs) == 0 {
		return strings.TrimSpace(err.Error())
	}
	return verrs[0].Error()
}
