package handlers

import (
	"encoding/json"
	"net/http"

	"nonsoemeka-backend/internal/apperrors"
	"nonsoemeka-backend/internal/middleware"
	"nonsoemeka-backend/internal/services"
)

type ReportsHandler struct {
	reportsService services.ReportsService
}

func NewReportsHandler(reportsService services.ReportsService) *ReportsHandler {
	return &ReportsHandler{
		reportsService: reportsService,
	}
}

func (h *ReportsHandler) GetSalesTrends(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := parsePagination(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	res, err := h.reportsService.GetSalesTrends(r.Context(), startDate, endDate, page, pageSize)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusOK, res)
}

func (h *ReportsHandler) GetTopProducts(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := parsePagination(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	res, err := h.reportsService.GetTopProducts(r.Context(), startDate, endDate, page, pageSize)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusOK, res)
}

func (h *ReportsHandler) writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *ReportsHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
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
