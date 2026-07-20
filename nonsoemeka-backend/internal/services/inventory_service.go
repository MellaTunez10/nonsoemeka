package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"nonsoemeka-backend/internal/apperrors"
	"nonsoemeka-backend/internal/audit"
	"nonsoemeka-backend/internal/dto"
	"nonsoemeka-backend/internal/models"
	"nonsoemeka-backend/internal/repository"
)

type InventoryService interface {
	CreateProduct(ctx context.Context, actorID uuid.UUID, req dto.CreateProductRequest) (dto.ProductResponse, error)
	ListProducts(ctx context.Context, search string, activeOnly bool, page, pageSize int) (dto.PaginatedResponse[dto.ProductResponse], error)
	GetProductByID(ctx context.Context, id uuid.UUID) (dto.ProductResponse, error)

	RegisterBatch(ctx context.Context, actorID uuid.UUID, req dto.RegisterBatchRequest) (dto.BatchResponse, error)
	ListBatches(ctx context.Context, productID *uuid.UUID, page, pageSize int) (dto.PaginatedResponse[dto.BatchResponse], error)
	AdjustStock(ctx context.Context, actorID uuid.UUID, batchID uuid.UUID, req dto.AdjustStockRequest) (dto.BatchResponse, error)
	WriteOffStock(ctx context.Context, actorID uuid.UUID, batchID uuid.UUID, req dto.WriteOffStockRequest) (dto.BatchResponse, error)
	ListExpiringBatches(ctx context.Context, page, pageSize int) (dto.PaginatedResponse[dto.BatchResponse], error)

	ListMovements(ctx context.Context, batchID *uuid.UUID, movementType *string, page, pageSize int) (dto.PaginatedResponse[dto.InventoryMovementResponse], error)
}

type inventoryService struct {
	pool         *pgxpool.Pool
	productRepo  repository.ProductRepository
	batchRepo    repository.BatchRepository
	movementRepo repository.InventoryMovementRepository
	settingsRepo repository.SettingsRepository
	auditRepo    repository.AuditRepository
}

func NewInventoryService(
	pool *pgxpool.Pool,
	productRepo repository.ProductRepository,
	batchRepo repository.BatchRepository,
	movementRepo repository.InventoryMovementRepository,
	settingsRepo repository.SettingsRepository,
	auditRepo repository.AuditRepository,
) InventoryService {
	return &inventoryService{
		pool:         pool,
		productRepo:  productRepo,
		batchRepo:    batchRepo,
		movementRepo: movementRepo,
		settingsRepo: settingsRepo,
		auditRepo:    auditRepo,
	}
}

func (s *inventoryService) CreateProduct(ctx context.Context, actorID uuid.UUID, req dto.CreateProductRequest) (dto.ProductResponse, error) {
	p := models.Product{
		Name:        req.Name,
		SKU:         req.SKU,
		Description: req.Description,
		IsActive:    true,
	}

	created, err := s.productRepo.Create(ctx, s.pool, p)
	if err != nil {
		return dto.ProductResponse{}, err
	}

	return dto.ProductResponse{
		ID:          created.ID,
		Name:        created.Name,
		SKU:         created.SKU,
		Description: created.Description,
		IsActive:    created.IsActive,
		CreatedAt:   created.CreatedAt,
		UpdatedAt:   created.UpdatedAt,
	}, nil
}

func (s *inventoryService) ListProducts(ctx context.Context, search string, activeOnly bool, page, pageSize int) (dto.PaginatedResponse[dto.ProductResponse], error) {
	products, total, err := s.productRepo.List(ctx, s.pool, search, activeOnly, page, pageSize)
	if err != nil {
		return dto.PaginatedResponse[dto.ProductResponse]{}, err
	}

	resList := make([]dto.ProductResponse, 0, len(products))
	for _, p := range products {
		var sellingPriceStr *string
		if p.SellingPrice != nil {
			str := p.SellingPrice.StringFixed(2)
			sellingPriceStr = &str
		}

		resList = append(resList, dto.ProductResponse{
			ID:            p.ID,
			Name:          p.Name,
			SKU:           p.SKU,
			Description:   p.Description,
			IsActive:      p.IsActive,
			TotalQuantity: p.TotalQuantity,
			SellingPrice:  sellingPriceStr,
			CreatedAt:     p.CreatedAt,
			UpdatedAt:     p.UpdatedAt,
		})
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	return dto.PaginatedResponse[dto.ProductResponse]{
		Data: resList,
		Pagination: dto.PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			TotalItems: total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *inventoryService) GetProductByID(ctx context.Context, id uuid.UUID) (dto.ProductResponse, error) {
	p, err := s.productRepo.FindByID(ctx, s.pool, id)
	if err != nil {
		return dto.ProductResponse{}, err
	}

	var sellingPriceStr *string
	if p.SellingPrice != nil {
		str := p.SellingPrice.StringFixed(2)
		sellingPriceStr = &str
	}

	return dto.ProductResponse{
		ID:            p.ID,
		Name:          p.Name,
		SKU:           p.SKU,
		Description:   p.Description,
		IsActive:      p.IsActive,
		TotalQuantity: p.TotalQuantity,
		SellingPrice:  sellingPriceStr,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}, nil
}

func (s *inventoryService) RegisterBatch(ctx context.Context, actorID uuid.UUID, req dto.RegisterBatchRequest) (dto.BatchResponse, error) {
	// Verify product exists
	product, err := s.productRepo.FindByID(ctx, s.pool, req.ProductID)
	if err != nil {
		return dto.BatchResponse{}, fmt.Errorf("invalid product: %w", err)
	}

	costPrice, err := decimal.NewFromString(req.CostPrice)
	if err != nil || costPrice.LessThan(decimal.Zero) {
		return dto.BatchResponse{}, fmt.Errorf("invalid cost_price: %w", apperrors.ErrBadRequest)
	}

	var markup decimal.Decimal
	if req.MarkupPercentage != nil && *req.MarkupPercentage != "" {
		m, err := decimal.NewFromString(*req.MarkupPercentage)
		if err != nil || m.LessThan(decimal.Zero) {
			return dto.BatchResponse{}, fmt.Errorf("invalid markup_percentage: %w", apperrors.ErrBadRequest)
		}
		markup = m
	} else {
		// Resolve default markup from settings
		defaultSetting, err := s.settingsRepo.Get(ctx, s.pool, "default_markup_percentage")
		if err == nil {
			var markupStr string
			if json.Unmarshal(defaultSetting.Value, &markupStr) == nil {
				if m, err := decimal.NewFromString(markupStr); err == nil {
					markup = m
				}
			}
		}
		if markup.IsZero() {
			markup = decimal.NewFromFloat(25.0) // default 25% if setting missing
		}
	}

	expiryDate, err := time.Parse("2006-01-02", req.ExpiryDate)
	if err != nil {
		return dto.BatchResponse{}, fmt.Errorf("invalid expiry_date format: %w", apperrors.ErrBadRequest)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dto.BatchResponse{}, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	b := models.Batch{
		ProductID:         req.ProductID,
		BatchNumber:       req.BatchNumber,
		QuantityReceived:  req.QuantityReceived,
		QuantityRemaining: req.QuantityReceived,
		ExpiryDate:        expiryDate,
		CostPrice:         costPrice,
		MarkupPercentage:  markup,
	}

	createdBatch, err := s.batchRepo.Create(ctx, tx, b)
	if err != nil {
		return dto.BatchResponse{}, err
	}

	// Record movement RECEIVED
	movement := models.InventoryMovement{
		BatchID:       createdBatch.ID,
		MovementType:  models.MovementReceived,
		QuantityDelta: req.QuantityReceived,
		CreatedBy:     actorID,
	}
	if err := s.movementRepo.Create(ctx, tx, movement); err != nil {
		return dto.BatchResponse{}, fmt.Errorf("failed to record inventory movement: %w", err)
	}

	// Record audit log
	if err := audit.LogAction(ctx, s.auditRepo, tx, actorID, "BATCH_CREATED", "batches", &createdBatch.ID, map[string]interface{}{
		"product_id":        product.ID,
		"batch_number":      createdBatch.BatchNumber,
		"quantity_received": createdBatch.QuantityReceived,
		"selling_price":     createdBatch.SellingPrice.StringFixed(2),
	}); err != nil {
		return dto.BatchResponse{}, fmt.Errorf("failed to write audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return dto.BatchResponse{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return dto.BatchResponse{
		ID:                createdBatch.ID,
		ProductID:         createdBatch.ProductID,
		ProductName:       product.Name,
		ProductSKU:        product.SKU,
		BatchNumber:       createdBatch.BatchNumber,
		QuantityReceived:  createdBatch.QuantityReceived,
		QuantityRemaining: createdBatch.QuantityRemaining,
		ExpiryDate:        createdBatch.ExpiryDate.Format("2006-01-02"),
		CostPrice:         createdBatch.CostPrice.StringFixed(2),
		MarkupPercentage:  createdBatch.MarkupPercentage.StringFixed(2),
		SellingPrice:      createdBatch.SellingPrice.StringFixed(2),
		ReceivedAt:        createdBatch.ReceivedAt,
	}, nil
}

func (s *inventoryService) ListBatches(ctx context.Context, productID *uuid.UUID, page, pageSize int) (dto.PaginatedResponse[dto.BatchResponse], error) {
	batches, total, err := s.batchRepo.List(ctx, s.pool, productID, page, pageSize)
	if err != nil {
		return dto.PaginatedResponse[dto.BatchResponse]{}, err
	}

	resList := make([]dto.BatchResponse, 0, len(batches))
	for _, b := range batches {
		resList = append(resList, dto.BatchResponse{
			ID:                b.ID,
			ProductID:         b.ProductID,
			ProductName:       b.ProductName,
			ProductSKU:        b.ProductSKU,
			BatchNumber:       b.BatchNumber,
			QuantityReceived:  b.QuantityReceived,
			QuantityRemaining: b.QuantityRemaining,
			ExpiryDate:        b.ExpiryDate.Format("2006-01-02"),
			CostPrice:         b.CostPrice.StringFixed(2),
			MarkupPercentage:  b.MarkupPercentage.StringFixed(2),
			SellingPrice:      b.SellingPrice.StringFixed(2),
			ReceivedAt:        b.ReceivedAt,
		})
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	return dto.PaginatedResponse[dto.BatchResponse]{
		Data: resList,
		Pagination: dto.PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			TotalItems: total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *inventoryService) AdjustStock(ctx context.Context, actorID uuid.UUID, batchID uuid.UUID, req dto.AdjustStockRequest) (dto.BatchResponse, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dto.BatchResponse{}, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	batch, err := s.batchRepo.FindByID(ctx, tx, batchID)
	if err != nil {
		return dto.BatchResponse{}, err
	}

	newQty := batch.QuantityRemaining + req.QuantityDelta
	if newQty < 0 || newQty > batch.QuantityReceived {
		return dto.BatchResponse{}, fmt.Errorf("invalid quantity adjustment resulting in %d: %w", newQty, apperrors.ErrBadRequest)
	}

	oldQty := batch.QuantityRemaining
	batch.QuantityRemaining = newQty

	if err := s.batchRepo.Update(ctx, tx, batch); err != nil {
		return dto.BatchResponse{}, err
	}

	reasonStr := req.Reason
	movement := models.InventoryMovement{
		BatchID:       batch.ID,
		MovementType:  models.MovementAdjustment,
		QuantityDelta: req.QuantityDelta,
		Reason:        &reasonStr,
		CreatedBy:     actorID,
	}
	if err := s.movementRepo.Create(ctx, tx, movement); err != nil {
		return dto.BatchResponse{}, err
	}

	if err := audit.LogAction(ctx, s.auditRepo, tx, actorID, "STOCK_ADJUSTED", "batches", &batch.ID, map[string]interface{}{
		"old_quantity":   oldQty,
		"new_quantity":   newQty,
		"quantity_delta": req.QuantityDelta,
		"reason":         req.Reason,
	}); err != nil {
		return dto.BatchResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return dto.BatchResponse{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return dto.BatchResponse{
		ID:                batch.ID,
		ProductID:         batch.ProductID,
		ProductName:       batch.ProductName,
		ProductSKU:        batch.ProductSKU,
		BatchNumber:       batch.BatchNumber,
		QuantityReceived:  batch.QuantityReceived,
		QuantityRemaining: batch.QuantityRemaining,
		ExpiryDate:        batch.ExpiryDate.Format("2006-01-02"),
		CostPrice:         batch.CostPrice.StringFixed(2),
		MarkupPercentage:  batch.MarkupPercentage.StringFixed(2),
		SellingPrice:      batch.SellingPrice.StringFixed(2),
		ReceivedAt:        batch.ReceivedAt,
	}, nil
}

func (s *inventoryService) WriteOffStock(ctx context.Context, actorID uuid.UUID, batchID uuid.UUID, req dto.WriteOffStockRequest) (dto.BatchResponse, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dto.BatchResponse{}, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	batch, err := s.batchRepo.FindByID(ctx, tx, batchID)
	if err != nil {
		return dto.BatchResponse{}, err
	}

	if batch.QuantityRemaining <= 0 {
		return dto.BatchResponse{}, fmt.Errorf("batch has no remaining stock to write off: %w", apperrors.ErrBadRequest)
	}

	oldQty := batch.QuantityRemaining
	delta := -oldQty
	batch.QuantityRemaining = 0

	if err := s.batchRepo.Update(ctx, tx, batch); err != nil {
		return dto.BatchResponse{}, err
	}

	reasonStr := req.Reason
	movement := models.InventoryMovement{
		BatchID:       batch.ID,
		MovementType:  models.MovementExpiredWriteOff,
		QuantityDelta: delta,
		Reason:        &reasonStr,
		CreatedBy:     actorID,
	}
	if err := s.movementRepo.Create(ctx, tx, movement); err != nil {
		return dto.BatchResponse{}, err
	}

	if err := audit.LogAction(ctx, s.auditRepo, tx, actorID, "STOCK_WRITTEN_OFF", "batches", &batch.ID, map[string]interface{}{
		"old_quantity": oldQty,
		"new_quantity": 0,
		"reason":       req.Reason,
	}); err != nil {
		return dto.BatchResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return dto.BatchResponse{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return dto.BatchResponse{
		ID:                batch.ID,
		ProductID:         batch.ProductID,
		ProductName:       batch.ProductName,
		ProductSKU:        batch.ProductSKU,
		BatchNumber:       batch.BatchNumber,
		QuantityReceived:  batch.QuantityReceived,
		QuantityRemaining: batch.QuantityRemaining,
		ExpiryDate:        batch.ExpiryDate.Format("2006-01-02"),
		CostPrice:         batch.CostPrice.StringFixed(2),
		MarkupPercentage:  batch.MarkupPercentage.StringFixed(2),
		SellingPrice:      batch.SellingPrice.StringFixed(2),
		ReceivedAt:        batch.ReceivedAt,
	}, nil
}

func (s *inventoryService) ListExpiringBatches(ctx context.Context, page, pageSize int) (dto.PaginatedResponse[dto.BatchResponse], error) {
	thresholdDays := 30
	if setting, err := s.settingsRepo.Get(ctx, s.pool, "expiry_alert_days"); err == nil {
		var days int
		if json.Unmarshal(setting.Value, &days) == nil && days > 0 {
			thresholdDays = days
		}
	}

	batches, total, err := s.batchRepo.ListExpiring(ctx, s.pool, thresholdDays, page, pageSize)
	if err != nil {
		return dto.PaginatedResponse[dto.BatchResponse]{}, err
	}

	resList := make([]dto.BatchResponse, 0, len(batches))
	for _, b := range batches {
		resList = append(resList, dto.BatchResponse{
			ID:                b.ID,
			ProductID:         b.ProductID,
			ProductName:       b.ProductName,
			ProductSKU:        b.ProductSKU,
			BatchNumber:       b.BatchNumber,
			QuantityReceived:  b.QuantityReceived,
			QuantityRemaining: b.QuantityRemaining,
			ExpiryDate:        b.ExpiryDate.Format("2006-01-02"),
			CostPrice:         b.CostPrice.StringFixed(2),
			MarkupPercentage:  b.MarkupPercentage.StringFixed(2),
			SellingPrice:      b.SellingPrice.StringFixed(2),
			ReceivedAt:        b.ReceivedAt,
		})
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	return dto.PaginatedResponse[dto.BatchResponse]{
		Data: resList,
		Pagination: dto.PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			TotalItems: total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *inventoryService) ListMovements(ctx context.Context, batchID *uuid.UUID, movementType *string, page, pageSize int) (dto.PaginatedResponse[dto.InventoryMovementResponse], error) {
	var movements []models.InventoryMovement
	var total int
	var err error

	if batchID != nil {
		movements, total, err = s.movementRepo.ListByBatch(ctx, s.pool, *batchID, page, pageSize)
	} else {
		movements, total, err = s.movementRepo.List(ctx, s.pool, movementType, page, pageSize)
	}

	if err != nil {
		return dto.PaginatedResponse[dto.InventoryMovementResponse]{}, err
	}

	resList := make([]dto.InventoryMovementResponse, 0, len(movements))
	for _, m := range movements {
		resList = append(resList, dto.InventoryMovementResponse{
			ID:            m.ID,
			BatchID:       m.BatchID,
			BatchNumber:   m.BatchNumber,
			ProductID:     m.ProductID,
			ProductName:   m.ProductName,
			MovementType:  string(m.MovementType),
			QuantityDelta: m.QuantityDelta,
			ReferenceID:   m.ReferenceID,
			Reason:        m.Reason,
			CreatedBy:     m.CreatedBy,
			CreatedByName: m.CreatedByName,
			CreatedAt:     m.CreatedAt,
		})
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	return dto.PaginatedResponse[dto.InventoryMovementResponse]{
		Data: resList,
		Pagination: dto.PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			TotalItems: total,
			TotalPages: totalPages,
		},
	}, nil
}
