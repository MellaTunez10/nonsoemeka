package sync

import (
	"time"

	"encoding/json"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"nonsoemeka-backend/internal/models"
)

// PushPayload is the data structure sent from LOCAL to CLOUD on each sync push.
type PushPayload struct {
	Products   []SyncProduct       `json:"products,omitempty"`
	Batches    []SyncBatch         `json:"batches,omitempty"`
	Settings   []models.Setting    `json:"settings,omitempty"`
	Sales      []SalePushItem      `json:"sales,omitempty"`
	Movements  []SyncMovement      `json:"movements,omitempty"`
	AuditLogs  []SyncAuditLog      `json:"audit_logs,omitempty"`
	UserStates []UserSecurityState `json:"user_states,omitempty"`
}

// SyncProduct is a Product serialized for the sync payload (includes all core fields).
type SyncProduct struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	SKU         string    `json:"sku"`
	Description *string   `json:"description,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SyncBatch is a Batch serialized for the sync payload.
type SyncBatch struct {
	ID                uuid.UUID       `json:"id"`
	ProductID         uuid.UUID       `json:"product_id"`
	BatchNumber       string          `json:"batch_number"`
	QuantityReceived  int             `json:"quantity_received"`
	QuantityRemaining int             `json:"quantity_remaining"`
	ExpiryDate        time.Time       `json:"expiry_date"`
	CostPrice         decimal.Decimal `json:"cost_price"`
	MarkupPercentage  decimal.Decimal `json:"markup_percentage"`
	ReceivedAt        time.Time       `json:"received_at"`
}

// SyncMovement is an InventoryMovement serialized for the sync payload.
type SyncMovement struct {
	ID            uuid.UUID           `json:"id"`
	BatchID       uuid.UUID           `json:"batch_id"`
	MovementType  models.MovementType `json:"movement_type"`
	QuantityDelta int                 `json:"quantity_delta"`
	ReferenceID   *uuid.UUID          `json:"reference_id,omitempty"`
	Reason        *string             `json:"reason,omitempty"`
	CreatedBy     uuid.UUID           `json:"created_by"`
	CreatedAt     time.Time           `json:"created_at"`
}

// SyncAuditLog is an AuditLog serialized for the sync payload.
type SyncAuditLog struct {
	ID          uuid.UUID       `json:"id"`
	ActorID     uuid.UUID       `json:"actor_id"`
	Action      string          `json:"action"`
	TargetTable string          `json:"target_table"`
	TargetID    *uuid.UUID      `json:"target_id,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

// SalePushItem bundles a sale with its items for atomic push.
type SalePushItem struct {
	Sale  SyncSale       `json:"sale"`
	Items []SyncSaleItem `json:"items"`
}

// SyncSale is a Sale serialized for the sync payload.
type SyncSale struct {
	ID             uuid.UUID       `json:"id"`
	StaffID        uuid.UUID       `json:"staff_id"`
	TotalAmount    decimal.Decimal `json:"total_amount"`
	IdempotencyKey string          `json:"idempotency_key"`
	CreatedAt      time.Time       `json:"created_at"`
}

// SyncSaleItem is a SaleItem serialized for the sync payload.
type SyncSaleItem struct {
	ID        uuid.UUID       `json:"id"`
	SaleID    uuid.UUID       `json:"sale_id"`
	ProductID uuid.UUID       `json:"product_id"`
	BatchID   uuid.UUID       `json:"batch_id"`
	Quantity  int             `json:"quantity"`
	UnitPrice decimal.Decimal `json:"unit_price"`
}

// UserSecurityState represents a Local-originated security state change for bidirectional sync.
type UserSecurityState struct {
	ID          uuid.UUID  `json:"id"`
	IsActive    bool       `json:"is_active"`
	LockedUntil *time.Time `json:"locked_until"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// PushResponse is the Cloud's response to a push, indicating which records were processed or failed.
type PushResponse struct {
	ProcessedIDs   map[string][]uuid.UUID `json:"processed_ids"`
	ProcessedKeys  []string               `json:"processed_keys,omitempty"`
	FailedIDs      map[string][]uuid.UUID `json:"failed_ids,omitempty"`
	FailureReasons map[uuid.UUID]string   `json:"failure_reasons,omitempty"`
}

// PullUsersResponse is the Cloud's response to a pull-users request.
type PullUsersResponse struct {
	Users      []SyncUser `json:"users"`
	NextCursor string     `json:"next_cursor"` // Cloud's own server timestamp, not Local's clock
}

// SyncUser is a User serialized for the sync pull response.
type SyncUser struct {
	ID                  uuid.UUID       `json:"id"`
	Username            string          `json:"username"`
	Email               string          `json:"email"`
	PasswordHash        string          `json:"password_hash"`
	Role                models.UserRole `json:"role"`
	IsActive            bool            `json:"is_active"`
	FailedLoginAttempts int             `json:"failed_login_attempts"`
	LockedUntil         *time.Time      `json:"locked_until"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

// SeedAdminPayload is the one-time payload for registering the Local seed admin on Cloud.
type SeedAdminPayload struct {
	ID           uuid.UUID       `json:"id"`
	Username     string          `json:"username"`
	Email        string          `json:"email"`
	PasswordHash string          `json:"password_hash"`
	Role         models.UserRole `json:"role"`
}
