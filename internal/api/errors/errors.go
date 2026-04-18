package errors

import (
	stderrors "errors"
	"fmt"
	"net/http"
)

// Kind categorizes errors for logging and metrics.
type Kind string

const (
	KindValidation    Kind = "validation"
	KindUnauthorized  Kind = "unauthorized"
	KindForbidden     Kind = "forbidden"
	KindNotFound      Kind = "not_found"
	KindConflict      Kind = "conflict"
	KindInternal      Kind = "internal"
	KindRateLimit     Kind = "rate_limit"
	KindNotImplemented Kind = "not_implemented"
)

// AppError is a typed application error with an HTTP status and machine code.
type AppError struct {
	Kind    Kind   `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
	Cause   error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Cause }

// New constructs an AppError.
func New(status int, kind Kind, code, message string) *AppError {
	return &AppError{Status: status, Kind: kind, Code: code, Message: message}
}

// Wrap preserves AppError if err already is one; otherwise wraps with fallback.
func Wrap(err error, fallback *AppError) error {
	if err == nil {
		return nil
	}
	var ae *AppError
	if stderrors.As(err, &ae) {
		return err
	}
	out := *fallback
	out.Cause = err
	return &out
}

// Sentinel errors for common cases.
var (
	ErrUnauthorized = New(http.StatusUnauthorized, KindUnauthorized, "unauthorized", "unauthorized")
	ErrForbidden    = New(http.StatusForbidden, KindForbidden, "forbidden", "forbidden")
	ErrNotFound     = New(http.StatusNotFound, KindNotFound, "not_found", "resource not found")
	ErrConflict     = New(http.StatusConflict, KindConflict, "conflict", "resource conflict")
	ErrValidation   = New(http.StatusBadRequest, KindValidation, "validation_error", "validation failed")
	ErrInternal        = New(http.StatusInternalServerError, KindInternal, "internal_error", "internal server error")
	ErrRateLimited     = New(http.StatusTooManyRequests, KindRateLimit, "rate_limited", "too many requests")
	ErrNotImplemented  = New(http.StatusNotImplemented, KindNotImplemented, "not_implemented", "not implemented")
	ErrInvalidCredentials = New(http.StatusUnauthorized, KindUnauthorized, "invalid_credentials", "invalid email or password")
	// ErrBuildingHasUnits is returned when a building cannot be removed because units still reference it.
	ErrBuildingHasUnits = New(http.StatusConflict, KindConflict, "building_has_units", "cannot delete building while units exist")
	// ErrUnitHasActiveLease is returned when a unit cannot be removed while an active lease exists.
	ErrUnitHasActiveLease = New(http.StatusConflict, KindConflict, "unit_has_active_lease", "cannot delete unit while an active lease exists")
	// ErrTenantHasLeases is returned when a tenant cannot be removed because leases still reference them; use PATCH to deactivate instead.
	ErrTenantHasLeases = New(http.StatusConflict, KindConflict, "tenant_has_leases", "cannot delete tenant while leases exist; deactivate with PATCH status instead")
	// ErrLeaseOverlap is returned when an active primary lease would overlap another on the same unit (service + DB EXCLUDE).
	ErrLeaseOverlap = New(http.StatusConflict, KindConflict, "lease_overlap", "lease date range overlaps another active primary lease on this unit")
)
