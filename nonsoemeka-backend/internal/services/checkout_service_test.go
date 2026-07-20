package services_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"nonsoemeka-backend/internal/models"
)

func TestFEFO_SortAndDeduct(t *testing.T) {
	pID := uuid.New()
	b1 := models.Batch{
		ID:                uuid.New(),
		ProductID:         pID,
		BatchNumber:       "EARLY-EXPIRY",
		QuantityRemaining: 50,
		ExpiryDate:        time.Now().AddDate(0, 1, 0),
		SellingPrice:      decimal.NewFromFloat(100.00),
	}
	b2 := models.Batch{
		ID:                uuid.New(),
		ProductID:         pID,
		BatchNumber:       "LATER-EXPIRY",
		QuantityRemaining: 100,
		ExpiryDate:        time.Now().AddDate(0, 6, 0),
		SellingPrice:      decimal.NewFromFloat(100.00),
	}

	batches := []models.Batch{b1, b2}

	requestedQty := 70
	needed := requestedQty
	totalSale := decimal.Zero

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

		itemTotal := b.SellingPrice.Mul(decimal.NewFromInt(int64(deduct)))
		totalSale = totalSale.Add(itemTotal)

		if needed == 0 {
			break
		}
	}

	assert.Equal(t, 0, needed)
	assert.Equal(t, 0, batches[0].QuantityRemaining)
	assert.Equal(t, 80, batches[1].QuantityRemaining)
	assert.Equal(t, "7000.00", totalSale.StringFixed(2))
}
