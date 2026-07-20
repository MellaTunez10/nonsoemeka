package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"nonsoemeka-backend/internal/apperrors"
	"nonsoemeka-backend/internal/dto"
	"nonsoemeka-backend/internal/models"
	"nonsoemeka-backend/internal/repository"
)

type CheckoutService interface {
	ProcessCheckout(ctx context.Context, staffID uuid.UUID, req dto.CheckoutRequest) (dto.ReceiptResponse, error)
}

type checkoutService struct {
	pool            *pgxpool.Pool
	saleRepo        repository.SaleRepository
	batchRepo       repository.BatchRepository
	productRepo     repository.ProductRepository
	movementRepo    repository.InventoryMovementRepository
	settingsRepo    repository.SettingsRepository
	userRepo        repository.UserRepository
}

func NewCheckoutService(
	pool *pgxpool.Pool,
	saleRepo repository.SaleRepository,
	batchRepo repository.BatchRepository,
	productRepo repository.ProductRepository,
	movementRepo repository.InventoryMovementRepository,
	settingsRepo repository.SettingsRepository,
	userRepo repository.UserRepository,
) CheckoutService {
	return &checkoutService{
		pool:         pool,
		saleRepo:     saleRepo,
		batchRepo:    batchRepo,
		productRepo:  productRepo,
		movementRepo: movementRepo,
		settingsRepo: settingsRepo,
		userRepo:     userRepo,
	}
}

func (s *checkoutService) ProcessCheckout(ctx context.Context, staffID uuid.UUID, req dto.CheckoutRequest) (dto.ReceiptResponse, error) {
	// 1. Idempotency Check
	existingSale, err := s.saleRepo.FindByIdempotencyKey(ctx, s.pool, req.IdempotencyKey)
	if err != nil {
		return dto.ReceiptResponse{}, err
	}
	if existingSale != nil {
		return s.formatReceipt(ctx, *existingSale)
	}

	staffUser, err := s.userRepo.FindByID(ctx, s.pool, staffID)
	if err != nil {
		return dto.ReceiptResponse{}, fmt.Errorf("staff user not found: %w", err)
	}

	// 2. Aggregate quantities by product ID to avoid duplicate product locking in same request
	type aggregatedItem struct {
		productID uuid.UUID
		quantity  int
	}
	itemMap := make(map[uuid.UUID]int)
	for _, item := range req.Items {
		itemMap[item.ProductID] += item.Quantity
	}

	var aggregated []aggregatedItem
	for pID, qty := range itemMap {
		aggregated = append(aggregated, aggregatedItem{productID: pID, quantity: qty})
	}

	// Sort aggregated items by product ID to prevent deadlocks across concurrent checkouts
	sort.Slice(aggregated, func(i, j int) bool {
		return aggregated[i].productID.String() < aggregated[j].productID.String()
	})

	// 3. Begin Transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return dto.ReceiptResponse{}, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "SET LOCAL statement_timeout = '5s'"); err != nil {
		return dto.ReceiptResponse{}, fmt.Errorf("failed to set statement timeout: %w", err)
	}

	var saleItemsToCreate []models.SaleItem
	var movementsToCreate []models.InventoryMovement
	var batchesToUpdate []models.Batch

	totalSaleAmount := decimal.Zero
	saleID := uuid.New()

	for _, aggItem := range aggregated {
		product, err := s.productRepo.FindByID(ctx, tx, aggItem.productID)
		if err != nil {
			return dto.ReceiptResponse{}, fmt.Errorf("product %s not found: %w", aggItem.productID, err)
		}
		if !product.IsActive {
			return dto.ReceiptResponse{}, fmt.Errorf("product %s is inactive: %w", product.Name, apperrors.ErrProductInactive)
		}

		// Lock batches in FEFO order (expiry_date ASC)
		batches, err := s.batchRepo.LockAvailableBatches(ctx, tx, aggItem.productID)
		if err != nil {
			return dto.ReceiptResponse{}, fmt.Errorf("failed to lock batches for product %s: %w", product.Name, err)
		}

		totalAvailable := 0
		for _, b := range batches {
			totalAvailable += b.QuantityRemaining
		}

		if totalAvailable < aggItem.quantity {
			return dto.ReceiptResponse{}, fmt.Errorf("insufficient stock for product %s (requested: %d, available: %d): %w",
				product.Name, aggItem.quantity, totalAvailable, apperrors.ErrInsufficientStock)
		}

		needed := aggItem.quantity
		for i := range batches {
			b := &batches[i]
			if b.QuantityRemaining <= 0 {
				continue
			}

			deduct := b.QuantityRemaining
			if deduct > needed {
				deduct = needed
			}

			b.QuantityRemaining -= deduct
			needed -= deduct

			batchesToUpdate = append(batchesToUpdate, *b)

			itemTotal := b.SellingPrice.Mul(decimal.NewFromInt(int64(deduct)))
			totalSaleAmount = totalSaleAmount.Add(itemTotal)

			saleItem := models.SaleItem{
				SaleID:      saleID,
				ProductID:   product.ID,
				ProductName: product.Name,
				BatchID:     b.ID,
				BatchNumber: b.BatchNumber,
				Quantity:    deduct,
				UnitPrice:   b.SellingPrice,
			}
			saleItemsToCreate = append(saleItemsToCreate, saleItem)

			movement := models.InventoryMovement{
				BatchID:       b.ID,
				MovementType:  models.MovementDispensed,
				QuantityDelta: -deduct,
				ReferenceID:   &saleID,
				CreatedBy:     staffID,
			}
			movementsToCreate = append(movementsToCreate, movement)

			if needed == 0 {
				break
			}
		}

		if needed > 0 {
			return dto.ReceiptResponse{}, fmt.Errorf("insufficient stock for product %s: %w", product.Name, apperrors.ErrInsufficientStock)
		}
	}

	// Apply batch updates
	for _, b := range batchesToUpdate {
		if err := s.batchRepo.Update(ctx, tx, b); err != nil {
			return dto.ReceiptResponse{}, fmt.Errorf("failed to update batch %s: %w", b.ID, err)
		}
	}

	// Create Sale record
	sale := models.Sale{
		ID:             saleID,
		StaffID:        staffID,
		StaffName:      staffUser.Username,
		TotalAmount:    totalSaleAmount,
		IdempotencyKey: req.IdempotencyKey,
		CreatedAt:      time.Now(),
		Items:          saleItemsToCreate,
	}

	createdSale, err := s.saleRepo.Create(ctx, tx, sale, saleItemsToCreate)
	if err != nil {
		return dto.ReceiptResponse{}, fmt.Errorf("failed to create sale: %w", err)
	}

	// Record movements
	for _, m := range movementsToCreate {
		m.ReferenceID = &createdSale.ID
		if err := s.movementRepo.Create(ctx, tx, m); err != nil {
			return dto.ReceiptResponse{}, fmt.Errorf("failed to record movement: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return dto.ReceiptResponse{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	createdSale.StaffName = staffUser.Username
	return s.formatReceipt(ctx, createdSale)
}

func (s *checkoutService) formatReceipt(ctx context.Context, sale models.Sale) (dto.ReceiptResponse, error) {
	pharmacyName := "Nonsoemeka Pharmacy"
	footerText := "Thank you for trusting Nonsoemeka Pharmacy! Wish you good health."

	if nameSetting, err := s.settingsRepo.Get(ctx, s.pool, "pharmacy_name"); err == nil {
		var nameStr string
		if json.Unmarshal(nameSetting.Value, &nameStr) == nil && nameStr != "" {
			pharmacyName = nameStr
		}
	}

	if footerSetting, err := s.settingsRepo.Get(ctx, s.pool, "receipt_footer"); err == nil {
		var footerStr string
		if json.Unmarshal(footerSetting.Value, &footerStr) == nil && footerStr != "" {
			footerText = footerStr
		}
	}

	itemResponses := make([]dto.ReceiptItemResponse, 0, len(sale.Items))
	for _, item := range sale.Items {
		itemResponses = append(itemResponses, dto.ReceiptItemResponse{
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			BatchID:     item.BatchID,
			BatchNumber: item.BatchNumber,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice.StringFixed(2),
			TotalPrice:  item.UnitPrice.Mul(decimal.NewFromInt(int64(item.Quantity))).StringFixed(2),
		})
	}

	return dto.ReceiptResponse{
		ID:             sale.ID,
		IdempotencyKey: sale.IdempotencyKey,
		PharmacyName:   pharmacyName,
		FooterText:     footerText,
		StaffID:        sale.StaffID,
		StaffName:      sale.StaffName,
		TotalAmount:    sale.TotalAmount.StringFixed(2),
		IssuedAt:       sale.CreatedAt,
		Items:          itemResponses,
	}, nil
}
