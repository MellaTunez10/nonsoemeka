package validation

import (
	"fmt"

	"github.com/shopspring/decimal"
	"net/mail"
	"regexp"
	"strings"
	"time"
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationErrors []FieldError

func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("validation errors: ")
	for i, err := range ve {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(fmt.Sprintf("%s: %s", err.Field, err.Message))
	}
	return sb.String()
}

type Validator struct {
	errors ValidationErrors
}

func New() *Validator {
	return &Validator{errors: make(ValidationErrors, 0)}
}

func (v *Validator) AddError(field, message string) {
	v.errors = append(v.errors, FieldError{Field: field, Message: message})
}

func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

func (v *Validator) Errors() ValidationErrors {
	if len(v.errors) == 0 {
		return nil
	}
	return v.errors
}

func (v *Validator) NotEmpty(field, value string) {
	if strings.TrimSpace(value) == "" {
		v.AddError(field, "must not be empty")
	}
}

func (v *Validator) ValidEmail(field, value string) {
	v.NotEmpty(field, value)
	if value != "" {
		_, err := mail.ParseAddress(value)
		if err != nil {
			v.AddError(field, "must be a valid email address")
		}
	}
}

var skuRegex = regexp.MustCompile(`^[A-Za-z0-9\-_]+$`)

func (v *Validator) ValidSKU(field, value string) {
	v.NotEmpty(field, value)
	if value != "" && !skuRegex.MatchString(value) {
		v.AddError(field, "must contain only alphanumeric characters, dashes, or underscores")
	}
}

func (v *Validator) PositiveInt(field string, value int) {
	if value <= 0 {
		v.AddError(field, "must be greater than zero")
	}
}

func (v *Validator) NonNegativeInt(field string, value int) {
	if value < 0 {
		v.AddError(field, "must not be negative")
	}
}

func (v *Validator) NonZeroInt(field string, value int) {
	if value == 0 {
		v.AddError(field, "must not be zero")
	}
}

func (v *Validator) PositiveDecimal(field string, value decimal.Decimal) {
	if value.LessThanOrEqual(decimal.Zero) {
		v.AddError(field, "must be greater than zero")
	}
}

func (v *Validator) NonNegativeDecimal(field string, value decimal.Decimal) {
	if value.LessThan(decimal.Zero) {
		v.AddError(field, "must not be negative")
	}
}

func (v *Validator) ParseDecimal(field, raw string) (decimal.Decimal, bool) {
	if strings.TrimSpace(raw) == "" {
		v.AddError(field, "must not be empty")
		return decimal.Zero, false
	}
	d, err := decimal.NewFromString(raw)
	if err != nil {
		v.AddError(field, "must be a valid decimal number")
		return decimal.Zero, false
	}
	return d, true
}

func (v *Validator) ParseOptionalDecimal(field, raw string) (*decimal.Decimal, bool) {
	if strings.TrimSpace(raw) == "" {
		return nil, true
	}
	d, err := decimal.NewFromString(raw)
	if err != nil {
		v.AddError(field, "must be a valid decimal number")
		return nil, false
	}
	return &d, true
}

func (v *Validator) ParseDate(field, raw string) (time.Time, bool) {
	if strings.TrimSpace(raw) == "" {
		v.AddError(field, "must not be empty")
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		v.AddError(field, "must be a valid date in YYYY-MM-DD format")
		return time.Time{}, false
	}
	return t, true
}

func (v *Validator) NotPastDate(field string, t time.Time) {
	today := time.Now().Truncate(24 * time.Hour)
	if t.Before(today) {
		v.AddError(field, "must not be in the past")
	}
}

func (v *Validator) IntBetween(field string, value, min, max int) {
	if value < min || value > max {
		v.AddError(field, fmt.Sprintf("must be between %d and %d", min, max))
	}
}

func (v *Validator) ValidRole(field, value string) {
	if value != "ADMIN" && value != "STAFF" {
		v.AddError(field, "role must be either ADMIN or STAFF")
	}
}
