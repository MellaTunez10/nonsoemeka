export type UserRole = 'ADMIN' | 'STAFF';

export interface UserProfile {
  id: string;
  username: string;
  email: string;
  role: UserRole;
}

export interface LoginResponse {
  access_token: string;
  user: UserProfile;
}

export interface Product {
  id: string;
  name: string;
  sku: string;
  description?: string;
  is_active: boolean;
  total_quantity?: number;
  selling_price?: string; // Decimal string
  created_at: string;
  updated_at: string;
}

export interface CreateProductRequest {
  name: string;
  sku: string;
  description?: string;
}

export interface Batch {
  id: string;
  product_id: string;
  product_name?: string;
  product_sku?: string;
  batch_number: string;
  quantity_received: number;
  quantity_remaining: number;
  expiry_date: string; // YYYY-MM-DD
  cost_price: string; // Decimal string
  markup_percentage: string; // Decimal string
  selling_price: string; // Decimal string
  received_at: string;
}

export interface RegisterBatchRequest {
  product_id: string;
  batch_number: string;
  quantity_received: number;
  expiry_date: string;
  cost_price: string;
  markup_percentage?: string;
}

export interface AdjustStockRequest {
  quantity_delta: number;
  reason: string;
}

export interface WriteOffStockRequest {
  reason: string;
}

export interface InventoryMovement {
  id: string;
  batch_id: string;
  batch_number: string;
  product_id: string;
  product_name: string;
  movement_type: 'RECEIVED' | 'DISPENSED' | 'ADJUSTMENT' | 'EXPIRED_WRITE_OFF';
  quantity_delta: number;
  reference_id?: string;
  reason?: string;
  created_by: string;
  created_by_name: string;
  created_at: string;
}

export interface CheckoutLineItem {
  product_id: string;
  quantity: number;
}

export interface CheckoutRequest {
  idempotency_key: string;
  items: CheckoutLineItem[];
}

export interface ReceiptItem {
  product_id: string;
  product_name: string;
  batch_id: string;
  batch_number: string;
  quantity: number;
  unit_price: string;
  total_price: string;
}

export interface Receipt {
  id: string;
  idempotency_key: string;
  pharmacy_name: string;
  footer_text: string;
  staff_id: string;
  staff_name: string;
  total_amount: string;
  issued_at: string;
  items: ReceiptItem[];
}

export interface FinancialSummary {
  total_revenue: string;
  total_cost: string;
  total_gross_profit: string;
  profit_margin_percentage: string;
  total_sales_count: number;
  total_items_sold: number;
}

export interface SalesTrendItem {
  date: string;
  total_amount: string;
  sales_count: number;
}

export interface TopProductItem {
  product_id: string;
  product_name: string;
  sku: string;
  total_quantity: number;
  total_revenue: string;
}

export interface Settings {
  default_markup_percentage: string;
  expiry_alert_days: number;
  low_stock_threshold: number;
  pharmacy_name: string;
  receipt_footer: string;
}

export interface UpdateSettingsRequest {
  default_markup_percentage?: string;
  expiry_alert_days?: number;
  low_stock_threshold?: number;
  pharmacy_name?: string;
  receipt_footer?: string;
}

export interface Staff {
  id: string;
  username: string;
  email: string;
  role: UserRole;
  is_active: boolean;
  failed_login_attempts: number;
  locked_until?: string | null;
  created_at: string;
  updated_at: string;
}

export interface CreateStaffRequest {
  username: string;
  email: string;
  password: string;
  role: UserRole;
}

export interface UpdateStaffRequest {
  is_active?: boolean;
  password?: string;
  clear_lockout?: boolean;
}

export interface AuditLog {
  id: string;
  actor_id: string;
  actor_name: string;
  action: string;
  target_table: string;
  target_id?: string;
  metadata?: Record<string, unknown>;
  created_at: string;
}

export interface PaginationMeta {
  page: number;
  page_size: number;
  total_items: number;
  total_pages: number;
}

export interface PaginatedResponse<T> {
  data: T[];
  pagination: PaginationMeta;
}

export interface ApiErrorResponse {
  error: {
    code: string;
    message: string;
    request_id?: string;
  };
}
