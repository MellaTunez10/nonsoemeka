package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"nonsoemeka-backend/internal/apperrors"
	"nonsoemeka-backend/internal/dto"
	"nonsoemeka-backend/internal/middleware"
	"nonsoemeka-backend/internal/services"
	"nonsoemeka-backend/internal/validation"
)

type CheckoutHandler struct {
	checkoutService services.CheckoutService
}

func NewCheckoutHandler(checkoutService services.CheckoutService) *CheckoutHandler {
	return &CheckoutHandler{
		checkoutService: checkoutService,
	}
}

func (h *CheckoutHandler) ProcessCheckout(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.writeError(w, r, apperrors.ErrUnauthorized)
		return
	}

	var req dto.CheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, r, apperrors.ErrBadRequest)
		return
	}

	v := validation.New()
	v.NotEmpty("idempotency_key", req.IdempotencyKey)
	if len(req.Items) == 0 {
		v.AddError("items", "must contain at least one line item")
	} else {
		for idx, item := range req.Items {
			if item.ProductID.String() == "" {
				v.AddError("items", "product_id is required for line item")
			}
			if item.Quantity <= 0 {
				v.AddError("items", "quantity must be greater than zero for line item")
			}
			_ = idx
		}
	}

	if v.HasErrors() {
		h.writeValidationErrors(w, r, v.Errors())
		return
	}

	receipt, err := h.checkoutService.ProcessCheckout(r.Context(), claims.UserID, req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusOK, receipt)
}

func (h *CheckoutHandler) writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *CheckoutHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	status, code := apperrors.ToHTTPStatus(err)
	reqID := middleware.GetRequestID(r.Context())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	msg := err.Error()
	if status == http.StatusInternalServerError {
		slog.Error("internal error in handler", "error", err, "request_id", reqID)
		msg = "An unexpected error occurred"
	}

	_ = json.NewEncoder(w).Encode(apperrors.ErrorResponse{
		Error: apperrors.ErrorDetail{
			Code:      code,
			Message:   msg,
			RequestID: reqID,
		},
	})
}

func (h *CheckoutHandler) writeValidationErrors(w http.ResponseWriter, r *http.Request, ve validation.ValidationErrors) {
	reqID := middleware.GetRequestID(r.Context())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(apperrors.ErrorResponse{
		Error: apperrors.ErrorDetail{
			Code:      "VALIDATION_ERROR",
			Message:   ve.Error(),
			RequestID: reqID,
		},
	})
}
