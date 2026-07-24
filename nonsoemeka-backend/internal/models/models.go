package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type UserRole string

const (
	RoleAdmin UserRole = "ADMIN"
	RoleStaff UserRole = "STAFF"
)

type MovementType string

const (
	MovementReceived        MovementType = "RECEIVED"
	MovementDispensed       MovementType = "DISPENSED"
	MovementAdjustment      MovementType = "ADJUSTMENT"
	MovementExpiredWriteOff MovementType = "EXPIRED_WRITE_OFF"
)

type User struct {
	ID                  uuid.UUID  `json:"id"`
	Username            string     `json:"username"`
	Email               string     `json:"email"`
	PasswordHash        string     `json:"-"`
	Role                UserRole   `json:"role"`
	IsActive            bool       `json:"is_active"`
	FailedLoginAttempts int        `json:"failed_login_attempts"`
	LockedUntil         *time.Time `json:"locked_until,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type RefreshToken struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type Product struct {
	ID            uuid.UUID        `json:"id"`
	Name          string           `json:"name"`
	SKU           string           `json:"sku"`
	Description   *string          `json:"description,omitempty"`
	IsActive      bool             `json:"is_active"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	TotalQuantity int              `json:"total_quantity,omitempty"`
	SellingPrice  *decimal.Decimal `json:"selling_price,omitempty"`
}

type Batch struct {
	ID                uuid.UUID       `json:"id"`
	ProductID         uuid.UUID       `json:"product_id"`
	ProductName       string          `json:"product_name,omitempty"`
	ProductSKU        string          `json:"product_sku,omitempty"`
	BatchNumber       string          `json:"batch_number"`
	QuantityReceived  int             `json:"quantity_received"`
	QuantityRemaining int             `json:"quantity_remaining"`
	ExpiryDate        time.Time       `json:"expiry_date"`
	CostPrice         decimal.Decimal `json:"cost_price"`
	MarkupPercentage  decimal.Decimal `json:"markup_percentage"`
	SellingPrice      decimal.Decimal `json:"selling_price"`
	ReceivedAt        time.Time       `json:"received_at"`
}

type InventoryMovement struct {
	ID            uuid.UUID    `json:"id"`
	BatchID       uuid.UUID    `json:"batch_id"`
	BatchNumber   string       `json:"batch_number,omitempty"`
	ProductID     uuid.UUID    `json:"product_id,omitempty"`
	ProductName   string       `json:"product_name,omitempty"`
	MovementType  MovementType `json:"movement_type"`
	QuantityDelta int          `json:"quantity_delta"`
	ReferenceID   *uuid.UUID   `json:"reference_id,omitempty"`
	Reason        *string      `json:"reason,omitempty"`
	CreatedBy     uuid.UUID    `json:"created_by"`
	CreatedByName string       `json:"created_by_name,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
}

type Sale struct {
	ID             uuid.UUID       `json:"id"`
	StaffID        uuid.UUID       `json:"staff_id"`
	StaffName      string          `json:"staff_name,omitempty"`
	TotalAmount    decimal.Decimal `json:"total_amount"`
	IdempotencyKey string          `json:"idempotency_key"`
	CreatedAt      time.Time       `json:"created_at"`
	Items          []SaleItem      `json:"items,omitempty"`
}

type SaleItem struct {
	ID          uuid.UUID       `json:"id"`
	SaleID      uuid.UUID       `json:"sale_id"`
	ProductID   uuid.UUID       `json:"product_id"`
	ProductName string          `json:"product_name,omitempty"`
	BatchID     uuid.UUID       `json:"batch_id"`
	BatchNumber string          `json:"batch_number,omitempty"`
	Quantity    int             `json:"quantity"`
	UnitPrice   decimal.Decimal `json:"unit_price"`
	TotalPrice  decimal.Decimal `json:"total_price,omitempty"`
}

type Setting struct {
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	UpdatedBy *uuid.UUID      `json:"updated_by,omitempty"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type AuditLog struct {
	ID          uuid.UUID       `json:"id"`
	ActorID     uuid.UUID       `json:"actor_id"`
	ActorName   string          `json:"actor_name,omitempty"`
	Action      string          `json:"action"`
	TargetTable string          `json:"target_table"`
	TargetID    *uuid.UUID      `json:"target_id,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}
