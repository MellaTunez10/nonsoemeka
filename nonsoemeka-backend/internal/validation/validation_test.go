package validation_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"nonsoemeka-backend/internal/validation"
)

func TestValidation_EdgeCases(t *testing.T) {
	v := validation.New()

	v.NotEmpty("username", "")
	v.ValidEmail("email", "invalid-email")
	v.ValidSKU("sku", "SKU 123@#$")
	v.PositiveInt("quantity", -5)

	dec, ok := v.ParseDecimal("price", "invalid")
	assert.False(t, ok)
	assert.True(t, dec.IsZero())

	pastDate := time.Now().AddDate(0, 0, -5)
	v.NotPastDate("expiry_date", pastDate)

	v.ValidRole("role", "SUPERADMIN")

	assert.True(t, v.HasErrors())
	errs := v.Errors()
	assert.GreaterOrEqual(t, len(errs), 6)
	assert.Contains(t, errs.Error(), "validation errors")
}

func TestValidation_Success(t *testing.T) {
	v := validation.New()

	v.NotEmpty("username", "john_doe")
	v.ValidEmail("email", "john@example.com")
	v.ValidSKU("sku", "SKU-12345")
	v.PositiveInt("quantity", 10)

	dec, ok := v.ParseDecimal("price", "299.99")
	assert.True(t, ok)
	assert.Equal(t, "299.99", dec.StringFixed(2))

	futureDate := time.Now().AddDate(0, 1, 0)
	v.NotPastDate("expiry_date", futureDate)
	v.ValidRole("role", "ADMIN")

	assert.False(t, v.HasErrors())
}
