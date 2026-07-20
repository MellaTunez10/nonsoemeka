package dto

import (
	"time"

	"github.com/google/uuid"
)

type PaginationMeta struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}

type PaginatedResponse[T any] struct {
	Data       []T            `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserProfileResponse struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
}

type LoginResponse struct {
	AccessToken string              `json:"access_token"`
	User        UserProfileResponse `json:"user"`
}

type CreateProductRequest struct {
	Name        string  `json:"name"`
	SKU         string  `json:"sku"`
	Description *string `json:"description,omitempty"`
}

type ProductResponse struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	SKU           string    `json:"sku"`
	Description   *string   `json:"description,omitempty"`
	IsActive      bool      `json:"is_active"`
	TotalQuantity int       `json:"total_quantity"`
	SellingPrice  *string   `json:"selling_price,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type RegisterBatchRequest struct {
	ProductID        uuid.UUID `json:"product_id"`
	BatchNumber      string    `json:"batch_number"`
	QuantityReceived int       `json:"quantity_received"`
	ExpiryDate       string    `json:"expiry_date"` // YYYY-MM-DD
	CostPrice        string    `json:"cost_price"`
	MarkupPercentage *string   `json:"markup_percentage,omitempty"`
}

type BatchResponse struct {
	ID                uuid.UUID `json:"id"`
	ProductID         uuid.UUID `json:"product_id"`
	ProductName       string    `json:"product_name,omitempty"`
	ProductSKU        string    `json:"product_sku,omitempty"`
	BatchNumber       string    `json:"batch_number"`
	QuantityReceived  int       `json:"quantity_received"`
	QuantityRemaining int       `json:"quantity_remaining"`
	ExpiryDate        string    `json:"expiry_date"`
	CostPrice         string    `json:"cost_price"`
	MarkupPercentage  string    `json:"markup_percentage"`
	SellingPrice      string    `json:"selling_price"`
	ReceivedAt        time.Time `json:"received_at"`
}

type AdjustStockRequest struct {
	QuantityDelta int    `json:"quantity_delta"`
	Reason        string `json:"reason"`
}

type WriteOffStockRequest struct {
	Reason string `json:"reason"`
}

type CheckoutItemRequest struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
}

type CheckoutRequest struct {
	IdempotencyKey string                `json:"idempotency_key"`
	Items          []CheckoutItemRequest `json:"items"`
}

type ReceiptItemResponse struct {
	ProductID   uuid.UUID `json:"product_id"`
	ProductName string    `json:"product_name"`
	BatchID     uuid.UUID `json:"batch_id"`
	BatchNumber string    `json:"batch_number"`
	Quantity    int       `json:"quantity"`
	UnitPrice   string    `json:"unit_price"`
	TotalPrice  string    `json:"total_price"`
}

type ReceiptResponse struct {
	ID             uuid.UUID             `json:"id"`
	IdempotencyKey string                `json:"idempotency_key"`
	PharmacyName   string                `json:"pharmacy_name"`
	FooterText     string                `json:"footer_text"`
	StaffID        uuid.UUID             `json:"staff_id"`
	StaffName      string                `json:"staff_name"`
	TotalAmount    string                `json:"total_amount"`
	IssuedAt       time.Time             `json:"issued_at"`
	Items          []ReceiptItemResponse `json:"items"`
}

type CreateStaffRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type UpdateStaffRequest struct {
	IsActive      *bool   `json:"is_active,omitempty"`
	Password      *string `json:"password,omitempty"`
	ClearLockout  bool    `json:"clear_lockout,omitempty"`
}

type StaffResponse struct {
	ID                  uuid.UUID  `json:"id"`
	Username            string     `json:"username"`
	Email               string     `json:"email"`
	Role                string     `json:"role"`
	IsActive            bool       `json:"is_active"`
	FailedLoginAttempts int        `json:"failed_login_attempts"`
	LockedUntil         *time.Time `json:"locked_until,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type SettingsResponse struct {
	DefaultMarkupPercentage string `json:"default_markup_percentage"`
	ExpiryAlertDays         int    `json:"expiry_alert_days"`
	LowStockThreshold       int    `json:"low_stock_threshold"`
	PharmacyName            string `json:"pharmacy_name"`
	ReceiptFooter           string `json:"receipt_footer"`
}

type UpdateSettingsRequest struct {
	DefaultMarkupPercentage *string `json:"default_markup_percentage,omitempty"`
	ExpiryAlertDays         *int    `json:"expiry_alert_days,omitempty"`
	LowStockThreshold       *int    `json:"low_stock_threshold,omitempty"`
	PharmacyName            *string `json:"pharmacy_name,omitempty"`
	ReceiptFooter           *string `json:"receipt_footer,omitempty"`
}

type FinancialSummaryResponse struct {
	TotalRevenue    string `json:"total_revenue"`
	TotalCost       string `json:"total_cost"`
	TotalGrossProfit string `json:"total_gross_profit"`
	ProfitMarginPct string `json:"profit_margin_percentage"`
	TotalSalesCount int    `json:"total_sales_count"`
	TotalItemsSold  int    `json:"total_items_sold"`
}

type SalesTrendItem struct {
	Date        string `json:"date"`
	TotalAmount string `json:"total_amount"`
	SalesCount  int    `json:"sales_count"`
}

type TopProductItem struct {
	ProductID    uuid.UUID `json:"product_id"`
	ProductName  string    `json:"product_name"`
	SKU          string    `json:"sku"`
	TotalQuantity int      `json:"total_quantity_sold"`
	TotalRevenue string    `json:"total_revenue"`
}

type InventoryMovementResponse struct {
	ID            uuid.UUID `json:"id"`
	BatchID       uuid.UUID `json:"batch_id"`
	BatchNumber   string    `json:"batch_number,omitempty"`
	ProductID     uuid.UUID `json:"product_id,omitempty"`
	ProductName   string    `json:"product_name,omitempty"`
	MovementType  string    `json:"movement_type"`
	QuantityDelta int       `json:"quantity_delta"`
	ReferenceID   *uuid.UUID `json:"reference_id,omitempty"`
	Reason        *string   `json:"reason,omitempty"`
	CreatedBy     uuid.UUID `json:"created_by"`
	CreatedByName string    `json:"created_by_name,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type AuditLogResponse struct {
	ID          uuid.UUID   `json:"id"`
	ActorID     uuid.UUID   `json:"actor_id"`
	ActorName   string      `json:"actor_name,omitempty"`
	Action      string      `json:"action"`
	TargetTable string      `json:"target_table"`
	TargetID    *uuid.UUID  `json:"target_id,omitempty"`
	Metadata    interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
}
