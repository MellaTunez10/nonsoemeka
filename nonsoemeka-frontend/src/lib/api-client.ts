import { ApiErrorResponse } from '../types';

// Supports same-origin nginx deployments (default) and cross-domain deployments
// (e.g. Vercel frontend + fly.io backend) via the VITE_API_BASE_URL build-time variable.
// Leave unset (or empty) when using the nginx reverse-proxy at the same origin.
const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? '';

let inMemoryAccessToken: string | null = null;
let onLogoutCallback: (() => void) | null = null;
let onTokenRefreshedCallback: ((newToken: string) => void) | null = null;

export function setAccessToken(token: string | null) {
  inMemoryAccessToken = token;
}

export function getAccessToken(): string | null {
  return inMemoryAccessToken;
}

export function configureAuthCallbacks(
  onLogout: () => void,
  onTokenRefreshed: (newToken: string) => void
) {
  onLogoutCallback = onLogout;
  onTokenRefreshedCallback = onTokenRefreshed;
}

let isRefreshing = false;
let failedQueue: Array<{
  resolve: (token: string) => void;
  reject: (err: unknown) => void;
}> = [];

const processQueue = (error: unknown, token: string | null = null) => {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error);
    } else if (token) {
      prom.resolve(token);
    }
  });
  failedQueue = [];
};

export async function apiClient<T>(
  endpoint: string,
  options: RequestInit = {},
  isRetry = false
): Promise<T> {
  const headers = new Headers(options.headers || {});
  headers.set('Content-Type', 'application/json');

  if (inMemoryAccessToken) {
    headers.set('Authorization', `Bearer ${inMemoryAccessToken}`);
  }

  const fetchOptions: RequestInit = {
    ...options,
    headers,
    credentials: 'include', // Always send httpOnly refresh cookie
  };

  try {
    const response = await fetch(`${API_BASE_URL}${endpoint}`, fetchOptions);

    if (response.status === 401 && !endpoint.includes('/auth/login') && !endpoint.includes('/auth/refresh')) {
      if (isRetry) {
        // Second 401 -> force logout
        if (onLogoutCallback) onLogoutCallback();
        throw new Error('Session expired. Please log in again.');
      }

      if (isRefreshing) {
        return new Promise<string>((resolve, reject) => {
          failedQueue.push({ resolve, reject });
        }).then((newToken) => {
          headers.set('Authorization', `Bearer ${newToken}`);
          return apiClient<T>(endpoint, { ...options, headers }, true);
        });
      }

      isRefreshing = true;

      try {
        const refreshRes = await fetch(`${API_BASE_URL}/api/v1/auth/refresh`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          credentials: 'include',
        });

        if (!refreshRes.ok) {
          throw new Error('Refresh failed');
        }

        const data = await refreshRes.json();
        const newAccessToken = data.access_token;

        setAccessToken(newAccessToken);
        if (onTokenRefreshedCallback) onTokenRefreshedCallback(newAccessToken);

        processQueue(null, newAccessToken);
        isRefreshing = false;

        headers.set('Authorization', `Bearer ${newAccessToken}`);
        return apiClient<T>(endpoint, { ...options, headers }, true);
      } catch (refreshErr) {
        processQueue(refreshErr, null);
        isRefreshing = false;
        if (onLogoutCallback) onLogoutCallback();
        throw new Error('Session expired');
      }
    }

    if (!response.ok) {
      let errorMsg = `HTTP Error ${response.status}`;
      try {
        const errJson: ApiErrorResponse = await response.json();
        if (errJson?.error?.message) {
          errorMsg = errJson.error.message;
        }
      } catch {
        // fallback to generic message
      }
      throw new Error(errorMsg);
    }

    if (response.status === 204) {
      return {} as T;
    }

    return await response.json();
  } catch (error) {
    throw error;
  }
}
