package apperrors

import (
	"errors"
	"net/http"
)

var (
	ErrInsufficientStock  = errors.New("insufficient stock")
	ErrBatchExpired       = errors.New("batch expired")
	ErrDuplicateSKU       = errors.New("duplicate sku")
	ErrDuplicateBatch     = errors.New("duplicate batch")
	ErrDuplicateIdempotencyKey = errors.New("duplicate idempotency key")
	ErrUserLocked         = errors.New("user locked")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrProductInactive    = errors.New("product inactive")
	ErrNotFound           = errors.New("resource not found")
	ErrBadRequest         = errors.New("bad request")
)

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

func ToHTTPStatus(err error) (int, string) {
	if err == nil {
		return http.StatusOK, "OK"
	}

	switch {
	case errors.Is(err, ErrInsufficientStock):
		return http.StatusConflict, "INSUFFICIENT_STOCK"
	case errors.Is(err, ErrBatchExpired):
		return http.StatusConflict, "BATCH_EXPIRED"
	case errors.Is(err, ErrDuplicateSKU):
		return http.StatusConflict, "DUPLICATE_SKU"
	case errors.Is(err, ErrDuplicateBatch):
		return http.StatusConflict, "DUPLICATE_BATCH"
	case errors.Is(err, ErrUserLocked):
		return http.StatusForbidden, "USER_LOCKED"
	case errors.Is(err, ErrInvalidCredentials):
		return http.StatusUnauthorized, "INVALID_CREDENTIALS"
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized, "UNAUTHORIZED"
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden, "FORBIDDEN"
	case errors.Is(err, ErrProductInactive):
		return http.StatusUnprocessableEntity, "PRODUCT_INACTIVE"
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound, "NOT_FOUND"
	case errors.Is(err, ErrBadRequest):
		return http.StatusBadRequest, "BAD_REQUEST"
	default:
		return http.StatusInternalServerError, "INTERNAL_ERROR"
	}
}
