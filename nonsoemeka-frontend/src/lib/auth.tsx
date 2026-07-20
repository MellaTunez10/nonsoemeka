import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { UserProfile, UserRole } from '../types';
import { apiClient, setAccessToken as setGlobalAccessToken, configureAuthCallbacks } from './api-client';

interface AuthContextType {
  user: UserProfile | null;
  accessToken: string | null;
  isLoading: boolean;
  login: (username: string, password: string) => Promise<UserProfile>;
  logout: () => Promise<void>;
  hasRole: (role: UserRole) => boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<UserProfile | null>(null);
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(true);

  const updateAccessToken = useCallback((token: string | null) => {
    setAccessToken(token);
    setGlobalAccessToken(token);
  }, []);

  const logout = useCallback(async () => {
    try {
      await apiClient('/api/v1/auth/logout', { method: 'POST' });
    } catch {
      // Ignore network errors on logout
    } finally {
      setUser(null);
      updateAccessToken(null);
    }
  }, [updateAccessToken]);

  useEffect(() => {
    configureAuthCallbacks(
      () => {
        setUser(null);
        updateAccessToken(null);
      },
      (newToken: string) => {
        updateAccessToken(newToken);
      }
    );
  }, [updateAccessToken]);

  // Attempt initial session restore via refresh token cookie
  useEffect(() => {
    const initAuth = async () => {
      try {
        const res = await apiClient<{ access_token: string }>('/api/v1/auth/refresh', { method: 'POST' });
        if (res.access_token) {
          updateAccessToken(res.access_token);
          // Decode basic claims or payload
          const payload = parseJwtPayload(res.access_token);
          if (payload) {
            setUser({
              id: payload.user_id,
              username: payload.username,
              email: payload.email || `${payload.username}@pharmacy.com`,
              role: payload.role as UserRole,
            });
          }
        }
      } catch {
        // Refresh failed -> guest state
        updateAccessToken(null);
        setUser(null);
      } finally {
        setIsLoading(false);
      }
    };

    initAuth();
  }, [updateAccessToken]);

  const login = async (username: string, password: string): Promise<UserProfile> => {
    const data = await apiClient<{ access_token: string; user: UserProfile }>('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    });

    updateAccessToken(data.access_token);
    setUser(data.user);
    return data.user;
  };

  const hasRole = (role: UserRole): boolean => {
    if (!user) return false;
    if (user.role === 'ADMIN') return true; // ADMIN can access all routes
    return user.role === role;
  };

  return (
    <AuthContext.Provider value={{ user, accessToken, isLoading, login, logout, hasRole }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

function parseJwtPayload(token: string) {
  try {
    const base64Url = token.split('.')[1];
    const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
    const jsonPayload = decodeURIComponent(
      atob(base64)
        .split('')
        .map((c) => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
        .join('')
    );
    return JSON.parse(jsonPayload);
  } catch {
    return null;
  }
}
