import { z } from 'zod';

export const loginSchema = z.object({
  username: z.string().min(1, 'Username is required'),
  password: z.string().min(1, 'Password is required'),
});

export const createProductSchema = z.object({
  name: z.string().min(1, 'Product name is required'),
  sku: z.string().min(3, 'SKU must be at least 3 characters').regex(/^[A-Za-z0-9-_]+$/, 'SKU can only contain alphanumeric characters, hyphens, and underscores'),
  description: z.string().optional(),
});

export const registerBatchSchema = z.object({
  product_id: z.string().uuid('Please select a valid product'),
  batch_number: z.string().min(1, 'Batch number is required'),
  quantity_received: z.number().int().positive('Quantity received must be greater than zero'),
  expiry_date: z.string().refine((val) => {
    const d = new Date(val);
    const now = new Date();
    now.setHours(0, 0, 0, 0);
    return !isNaN(d.getTime()) && d >= now;
  }, 'Expiry date must be in the future'),
  cost_price: z.string().refine((val) => !isNaN(parseFloat(val)) && parseFloat(val) >= 0, 'Cost price must be a non-negative decimal'),
  markup_percentage: z.string().optional().refine((val) => !val || (!isNaN(parseFloat(val)) && parseFloat(val) >= 0), 'Markup percentage must be non-negative'),
});

export const adjustStockSchema = z.object({
  quantity_delta: z.number().int().refine((val) => val !== 0, 'Quantity delta cannot be zero'),
  reason: z.string().min(3, 'Reason is required (min 3 characters)'),
});

export const writeOffStockSchema = z.object({
  reason: z.string().min(3, 'Reason for write-off is required (min 3 characters)'),
});

export const createStaffSchema = z.object({
  username: z.string().min(3, 'Username must be at least 3 characters'),
  email: z.string().email('Invalid email address'),
  password: z.string().min(8, 'Password must be at least 8 characters'),
  role: z.enum(['ADMIN', 'STAFF']),
});

export const updateStaffSchema = z.object({
  is_active: z.boolean().optional(),
  password: z.string().optional().refine((val) => !val || val.length >= 8, 'Password must be at least 8 characters if provided'),
  clear_lockout: z.boolean().optional(),
});

export const updateSettingsSchema = z.object({
  default_markup_percentage: z.string().optional().refine((val) => !val || (!isNaN(parseFloat(val)) && parseFloat(val) >= 0), 'Markup must be non-negative'),
  expiry_alert_days: z.number().int().positive().optional(),
  low_stock_threshold: z.number().int().positive().optional(),
  pharmacy_name: z.string().min(1).optional(),
  receipt_footer: z.string().min(1).optional(),
});
