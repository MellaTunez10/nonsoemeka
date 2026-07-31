package sync

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"nonsoemeka-backend/internal/apperrors"
	"nonsoemeka-backend/internal/models"
)

type SyncHandler struct {
	pool         *pgxpool.Pool
	service      SyncService
	trackingRepo SyncTrackingRepository
}

func NewSyncHandler(pool *pgxpool.Pool, service SyncService, trackingRepo SyncTrackingRepository) *SyncHandler {
	return &SyncHandler{
		pool:         pool,
		service:      service,
		trackingRepo: trackingRepo,
	}
}

func (h *SyncHandler) HandlePush(w http.ResponseWriter, r *http.Request) {
	var payload PushPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	resp, err := h.service.ProcessPush(r.Context(), payload)
	if err != nil {
		status, code := apperrors.ToHTTPStatus(err)
		writeError(w, status, code, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *SyncHandler) HandlePullUsers(w http.ResponseWriter, r *http.Request) {
	since := r.URL.Query().Get("since")

	resp, err := h.service.GetUsersForPull(r.Context(), since)

	if err != nil {
		status, code := apperrors.ToHTTPStatus(err)
		writeError(w, status, code, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *SyncHandler) HandleRegisterSeedAdmin(w http.ResponseWriter, r *http.Request) {
	var payload SeedAdminPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	err := h.service.RegisterSeedAdmin(r.Context(), payload)
	if err != nil {
		status, code := apperrors.ToHTTPStatus(err)
		writeError(w, status, code, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *SyncHandler) HandleListFailedSyncItems(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}

	limit := 10
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	offset := (page - 1) * limit

	movements, total, err := h.trackingRepo.ListFailedMovements(r.Context(), h.pool, offset, limit)
	if err != nil {
		status, code := apperrors.ToHTTPStatus(err)
		writeError(w, status, code, err.Error())
		return
	}

	if movements == nil {
		movements = []models.InventoryMovement{} // Avoid null in JSON response
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"failed_items": movements,
		"total":        total,
		"page":         page,
		"limit":        limit,
	})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apperrors.ErrorResponse{
		Error: apperrors.ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}
