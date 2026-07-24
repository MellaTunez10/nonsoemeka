import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../lib/api-client';
import {
  Product,
  Batch,
  CheckoutRequest,
  Receipt,
  CreateProductRequest,
  RegisterBatchRequest,
  AdjustStockRequest,
  WriteOffStockRequest,
  FinancialSummary,
  SalesTrendItem,
  TopProductItem,
  Settings,
  UpdateSettingsRequest,
  Staff,
  CreateStaffRequest,
  UpdateStaffRequest,
  AuditLog,
  PaginatedResponse,
} from '../types';

export function useProducts(page = 1, search = '') {
  return useQuery({
    queryKey: ['products', page, search],
    queryFn: async () => {
      const url = `/api/v1/products?page=${page}&page_size=20&search=${encodeURIComponent(search)}&active_only=true`;
      return apiClient<PaginatedResponse<Product>>(url);
    },
    staleTime: 5000, // 5s stale time for POS
    refetchOnWindowFocus: true,
  });
}

export function useAdminProducts(page = 1, search = '') {
  return useQuery({
    queryKey: ['admin-products', page, search],
    queryFn: async () => {
      const url = `/api/v1/admin/inventory/products?page=${page}&page_size=20&search=${encodeURIComponent(search)}&active_only=true`;
      return apiClient<PaginatedResponse<Product>>(url);
    },
  });
}

export function useBatches(page = 1, search = '') {
  return useQuery({
    queryKey: ['batches', page, search],
    queryFn: async () => {
      const url = `/api/v1/admin/inventory/batches?page=${page}&page_size=20&search=${encodeURIComponent(search)}`;
      return apiClient<PaginatedResponse<Batch>>(url);
    },
  });
}

export function useExpiringBatches() {
  return useQuery({
    queryKey: ['expiring-batches'],
    queryFn: async () => {
      return apiClient<PaginatedResponse<Batch>>('/api/v1/admin/inventory/expiry');
    },
  });
}

export function useCheckout() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (req: CheckoutRequest) => {
      return apiClient<Receipt>('/api/v1/checkout', {
        method: 'POST',
        body: JSON.stringify(req),
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['products'] });
      queryClient.invalidateQueries({ queryKey: ['batches'] });
    },
  });
}

export function useCreateProduct() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (req: CreateProductRequest) => {
      return apiClient<Product>('/api/v1/admin/inventory/products', {
        method: 'POST',
        body: JSON.stringify(req),
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['products'] });
      queryClient.invalidateQueries({ queryKey: ['admin-products'] });
    },
  });
}

export function useDeleteProduct() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      return apiClient(`/api/v1/admin/inventory/products/${id}`, {
        method: 'DELETE',
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-products'] });
    },
  });
}

export function useRegisterBatch() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (req: RegisterBatchRequest) => {
      return apiClient<Batch>('/api/v1/admin/inventory/batches', {
        method: 'POST',
        body: JSON.stringify(req),
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['batches'] });
      queryClient.invalidateQueries({ queryKey: ['products'] });
      queryClient.invalidateQueries({ queryKey: ['expiring-batches'] });
    },
  });
}

export function useAdjustStock() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, req }: { id: string; req: AdjustStockRequest }) => {
      return apiClient<{ message: string }>(`/api/v1/admin/inventory/batches/${id}/adjust`, {
        method: 'POST',
        body: JSON.stringify(req),
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['batches'] });
      queryClient.invalidateQueries({ queryKey: ['products'] });
    },
  });
}

export function useWriteOffStock() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, req }: { id: string; req: WriteOffStockRequest }) => {
      return apiClient<{ message: string }>(`/api/v1/admin/inventory/batches/${id}/write-off`, {
        method: 'POST',
        body: JSON.stringify(req),
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['batches'] });
      queryClient.invalidateQueries({ queryKey: ['products'] });
      queryClient.invalidateQueries({ queryKey: ['expiring-batches'] });
    },
  });
}

export function useFinancialSummary() {
  return useQuery({
    queryKey: ['financial-summary'],
    queryFn: async () => {
      return apiClient<FinancialSummary>('/api/v1/admin/financials/summary');
    },
  });
}

export function useSalesTrends(startDate?: string, endDate?: string) {
  return useQuery({
    queryKey: ['sales-trends', startDate, endDate],
    queryFn: async () => {
      let url = '/api/v1/admin/reports/sales-trends';
      const params = new URLSearchParams();
      if (startDate) params.set('start_date', startDate);
      if (endDate) params.set('end_date', endDate);
      if (params.toString()) url += `?${params.toString()}`;
      return apiClient<{ data: SalesTrendItem[] }>(url);
    },
  });
}

export function useTopProducts(limit = 5) {
  return useQuery({
    queryKey: ['top-products', limit],
    queryFn: async () => {
      return apiClient<{ data: TopProductItem[] }>(`/api/v1/admin/reports/top-products?limit=${limit}`);
    },
  });
}

export function useStaffList() {
  return useQuery({
    queryKey: ['staff-list'],
    queryFn: async () => {
      return apiClient<{ data: Staff[] }>('/api/v1/admin/staff');
    },
  });
}

export function useCreateStaff() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (req: CreateStaffRequest) => {
      return apiClient<Staff>('/api/v1/admin/staff', {
        method: 'POST',
        body: JSON.stringify(req),
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['staff-list'] });
    },
  });
}

export function useUpdateStaff() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, req }: { id: string; req: UpdateStaffRequest }) => {
      return apiClient<Staff>(`/api/v1/admin/staff/${id}`, {
        method: 'PUT',
        body: JSON.stringify(req),
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['staff-list'] });
    },
  });
}

export function useDeleteStaff() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      return apiClient<{ message: string }>(`/api/v1/admin/staff/${id}`, {
        method: 'DELETE',
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['staff-list'] });
    },
  });
}

export function useAuditLogs(page = 1) {
  return useQuery({
    queryKey: ['audit-logs', page],
    queryFn: async () => {
      return apiClient<PaginatedResponse<AuditLog>>(`/api/v1/admin/audit-logs?page=${page}&page_size=20`);
    },
  });
}

export function useSettings() {
  return useQuery({
    queryKey: ['settings'],
    queryFn: async () => {
      return apiClient<Settings>('/api/v1/admin/settings');
    },
  });
}

export function useUpdateSettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (req: UpdateSettingsRequest) => {
      return apiClient<Settings>('/api/v1/admin/settings', {
        method: 'PUT',
        body: JSON.stringify(req),
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['settings'] });
    },
  });
}
