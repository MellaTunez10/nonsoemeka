package handlers

import (
	"encoding/json"
	"net/http"

	"nonsoemeka-backend/internal/apperrors"
	"nonsoemeka-backend/internal/dto"
	"nonsoemeka-backend/internal/middleware"
	"nonsoemeka-backend/internal/services"
	"nonsoemeka-backend/internal/validation"
)

type SettingsHandler struct {
	settingsService services.SettingsService
}

func NewSettingsHandler(settingsService services.SettingsService) *SettingsHandler {
	return &SettingsHandler{
		settingsService: settingsService,
	}
}

func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	res, err := h.settingsService.GetSettings(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusOK, res)
}

func (h *SettingsHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.writeError(w, r, apperrors.ErrUnauthorized)
		return
	}

	var req dto.UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, r, apperrors.ErrBadRequest)
		return
	}

	v := validation.New()
	if req.DefaultMarkupPercentage != nil {
		mDec, okM := v.ParseDecimal("default_markup_percentage", *req.DefaultMarkupPercentage)
		if okM {
			v.NonNegativeDecimal("default_markup_percentage", mDec)
		}
	}
	if req.ExpiryAlertDays != nil {
		v.PositiveInt("expiry_alert_days", *req.ExpiryAlertDays)
	}
	if req.LowStockThreshold != nil {
		v.PositiveInt("low_stock_threshold", *req.LowStockThreshold)
	}
	if req.PharmacyName != nil {
		v.NotEmpty("pharmacy_name", *req.PharmacyName)
	}

	if v.HasErrors() {
		h.writeValidationErrors(w, r, v.Errors())
		return
	}

	res, err := h.settingsService.UpdateSettings(r.Context(), claims.UserID, req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusOK, res)
}

func (h *SettingsHandler) writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *SettingsHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	status, code := apperrors.ToHTTPStatus(err)
	reqID := middleware.GetRequestID(r.Context())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	msg := err.Error()
	if status == http.StatusInternalServerError {
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

func (h *SettingsHandler) writeValidationErrors(w http.ResponseWriter, r *http.Request, ve validation.ValidationErrors) {
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
