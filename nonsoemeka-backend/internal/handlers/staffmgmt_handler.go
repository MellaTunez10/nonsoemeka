package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"nonsoemeka-backend/internal/apperrors"
	"nonsoemeka-backend/internal/dto"
	"nonsoemeka-backend/internal/middleware"
	"nonsoemeka-backend/internal/services"
	"nonsoemeka-backend/internal/validation"
)

type StaffManagementHandler struct {
	staffService services.StaffManagementService
}

func NewStaffManagementHandler(staffService services.StaffManagementService) *StaffManagementHandler {
	return &StaffManagementHandler{
		staffService: staffService,
	}
}

func (h *StaffManagementHandler) CreateStaff(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.writeError(w, r, apperrors.ErrUnauthorized)
		return
	}

	var req dto.CreateStaffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, r, apperrors.ErrBadRequest)
		return
	}

	v := validation.New()
	v.NotEmpty("username", req.Username)
	v.ValidEmail("email", req.Email)
	v.NotEmpty("password", req.Password)
	v.ValidRole("role", req.Role)

	if v.HasErrors() {
		h.writeValidationErrors(w, r, v.Errors())
		return
	}

	res, err := h.staffService.CreateStaff(r.Context(), claims.UserID, req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, res)
}

func (h *StaffManagementHandler) ListStaff(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := parsePagination(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	res, err := h.staffService.ListStaff(r.Context(), page, pageSize)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusOK, res)
}

func (h *StaffManagementHandler) UpdateStaff(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.writeError(w, r, apperrors.ErrUnauthorized)
		return
	}

	staffIDStr := chi.URLParam(r, "id")
	staffID, err := uuid.Parse(staffIDStr)
	if err != nil {
		h.writeError(w, r, apperrors.ErrBadRequest)
		return
	}

	var req dto.UpdateStaffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, r, apperrors.ErrBadRequest)
		return
	}

	res, err := h.staffService.UpdateStaff(r.Context(), claims.UserID, staffID, req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusOK, res)
}

func (h *StaffManagementHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := parsePagination(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	var actorID *uuid.UUID
	if aIDStr := r.URL.Query().Get("actor_id"); aIDStr != "" {
		if aID, err := uuid.Parse(aIDStr); err == nil {
			actorID = &aID
		}
	}

	var action *string
	if act := r.URL.Query().Get("action"); act != "" {
		action = &act
	}

	var targetTable *string
	if tt := r.URL.Query().Get("target_table"); tt != "" {
		targetTable = &tt
	}

	var startDate *string
	if sd := r.URL.Query().Get("start_date"); sd != "" {
		startDate = &sd
	}

	var endDate *string
	if ed := r.URL.Query().Get("end_date"); ed != "" {
		endDate = &ed
	}

	res, err := h.staffService.ListAuditLogs(r.Context(), actorID, action, targetTable, startDate, endDate, page, pageSize)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusOK, res)
}

func (h *StaffManagementHandler) writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *StaffManagementHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
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

func (h *StaffManagementHandler) writeValidationErrors(w http.ResponseWriter, r *http.Request, ve validation.ValidationErrors) {
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
