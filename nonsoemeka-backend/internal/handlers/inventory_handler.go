package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"nonsoemeka-backend/internal/apperrors"
	"nonsoemeka-backend/internal/dto"
	"nonsoemeka-backend/internal/middleware"
	"nonsoemeka-backend/internal/services"
	"nonsoemeka-backend/internal/validation"
)

type InventoryHandler struct {
	inventoryService services.InventoryService
}

func NewInventoryHandler(inventoryService services.InventoryService) *InventoryHandler {
	return &InventoryHandler{
		inventoryService: inventoryService,
	}
}

func (h *InventoryHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.writeError(w, r, apperrors.ErrUnauthorized)
		return
	}

	var req dto.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, r, apperrors.ErrBadRequest)
		return
	}

	v := validation.New()
	v.NotEmpty("name", req.Name)
	v.ValidSKU("sku", req.SKU)
	if v.HasErrors() {
		h.writeValidationErrors(w, r, v.Errors())
		return
	}

	res, err := h.inventoryService.CreateProduct(r.Context(), claims.UserID, req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, res)
}

func (h *InventoryHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := parsePagination(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	search := r.URL.Query().Get("search")
	activeOnly := r.URL.Query().Get("active_only") == "true"

	res, err := h.inventoryService.ListProducts(r.Context(), search, activeOnly, page, pageSize)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusOK, res)
}

func (h *InventoryHandler) RegisterBatch(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.writeError(w, r, apperrors.ErrUnauthorized)
		return
	}

	var req dto.RegisterBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, r, apperrors.ErrBadRequest)
		return
	}

	v := validation.New()
	v.NotEmpty("product_id", req.ProductID.String())
	v.NotEmpty("batch_number", req.BatchNumber)
	v.PositiveInt("quantity_received", req.QuantityReceived)
	v.NotEmpty("expiry_date", req.ExpiryDate)
	t, okDate := v.ParseDate("expiry_date", req.ExpiryDate)
	if okDate {
		v.NotPastDate("expiry_date", t)
	}
	costDec, okCost := v.ParseDecimal("cost_price", req.CostPrice)
	if okCost {
		v.NonNegativeDecimal("cost_price", costDec)
	}
	if req.MarkupPercentage != nil && *req.MarkupPercentage != "" {
		mDec, okM := v.ParseDecimal("markup_percentage", *req.MarkupPercentage)
		if okM {
			v.NonNegativeDecimal("markup_percentage", mDec)
		}
	}

	if v.HasErrors() {
		h.writeValidationErrors(w, r, v.Errors())
		return
	}

	res, err := h.inventoryService.RegisterBatch(r.Context(), claims.UserID, req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, res)
}

func (h *InventoryHandler) ListBatches(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := parsePagination(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	var productID *uuid.UUID
	if pIDStr := r.URL.Query().Get("product_id"); pIDStr != "" {
		if pID, err := uuid.Parse(pIDStr); err == nil {
			productID = &pID
		}
	}

	res, err := h.inventoryService.ListBatches(r.Context(), productID, page, pageSize)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusOK, res)
}

func (h *InventoryHandler) AdjustStock(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.writeError(w, r, apperrors.ErrUnauthorized)
		return
	}

	batchIDStr := chi.URLParam(r, "id")
	batchID, err := uuid.Parse(batchIDStr)
	if err != nil {
		h.writeError(w, r, apperrors.ErrBadRequest)
		return
	}

	var req dto.AdjustStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, r, apperrors.ErrBadRequest)
		return
	}

	v := validation.New()
	v.NonZeroInt("quantity_delta", req.QuantityDelta)
	v.NotEmpty("reason", req.Reason)
	if v.HasErrors() {
		h.writeValidationErrors(w, r, v.Errors())
		return
	}

	res, err := h.inventoryService.AdjustStock(r.Context(), claims.UserID, batchID, req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusOK, res)
}

func (h *InventoryHandler) WriteOffStock(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.writeError(w, r, apperrors.ErrUnauthorized)
		return
	}

	batchIDStr := chi.URLParam(r, "id")
	batchID, err := uuid.Parse(batchIDStr)
	if err != nil {
		h.writeError(w, r, apperrors.ErrBadRequest)
		return
	}

	var req dto.WriteOffStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, r, apperrors.ErrBadRequest)
		return
	}

	v := validation.New()
	v.NotEmpty("reason", req.Reason)
	if v.HasErrors() {
		h.writeValidationErrors(w, r, v.Errors())
		return
	}

	res, err := h.inventoryService.WriteOffStock(r.Context(), claims.UserID, batchID, req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusOK, res)
}

func (h *InventoryHandler) ListExpiringBatches(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := parsePagination(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	res, err := h.inventoryService.ListExpiringBatches(r.Context(), page, pageSize)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusOK, res)
}

func (h *InventoryHandler) ListMovements(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := parsePagination(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	var batchID *uuid.UUID
	if bIDStr := r.URL.Query().Get("batch_id"); bIDStr != "" {
		if bID, err := uuid.Parse(bIDStr); err == nil {
			batchID = &bID
		}
	}

	var movementType *string
	if mType := r.URL.Query().Get("movement_type"); mType != "" {
		movementType = &mType
	}

	res, err := h.inventoryService.ListMovements(r.Context(), batchID, movementType, page, pageSize)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, http.StatusOK, res)
}

func parsePagination(r *http.Request) (int, int, error) {
	page := 1
	pageSize := 20

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		p, err := strconv.Atoi(pageStr)
		if err != nil || p < 1 {
			return 0, 0, apperrors.ErrBadRequest
		}
		page = p
	}

	if sizeStr := r.URL.Query().Get("page_size"); sizeStr != "" {
		s, err := strconv.Atoi(sizeStr)
		if err != nil || s < 1 {
			return 0, 0, apperrors.ErrBadRequest
		}
		if s > 100 {
			// Section 10: page_size > 100 returns 400 Bad Request
			return 0, 0, apperrors.ErrBadRequest
		}
		pageSize = s
	}

	return page, pageSize, nil
}

func (h *InventoryHandler) writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *InventoryHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
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

func (h *InventoryHandler) writeValidationErrors(w http.ResponseWriter, r *http.Request, ve validation.ValidationErrors) {
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
