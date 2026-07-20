package handlers

import (
	"encoding/json"
	"net/http"

	"nonsoemeka-backend/internal/apperrors"
	"nonsoemeka-backend/internal/middleware"
	"nonsoemeka-backend/internal/services"
)

type FinancialsHandler struct {
	financialsService services.FinancialsService
}

func NewFinancialsHandler(financialsService services.FinancialsService) *FinancialsHandler {
	return &FinancialsHandler{
		financialsService: financialsService,
	}
}

func (h *FinancialsHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	res, err := h.financialsService.GetSummary(r.Context(), startDate, endDate)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusOK, res)
}

func (h *FinancialsHandler) writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *FinancialsHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
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
