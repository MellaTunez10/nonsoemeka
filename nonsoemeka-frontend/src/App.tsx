import React from 'react';
import { BrowserRouter, Routes, Route, Navigate, useLocation, Outlet } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AuthProvider, useAuth } from './lib/auth';
import { ThemeProvider } from './lib/theme';
import { Navbar } from './components/layout/Navbar';
import { LoginPage } from './pages/LoginPage';
import { StaffPOSPage } from './pages/StaffPOSPage';
import { AdminInventoryPage } from './pages/AdminInventoryPage';
import { AdminExpiryPage } from './pages/AdminExpiryPage';
import { AdminFinancialsPage } from './pages/AdminFinancialsPage';
import { AdminStaffPage } from './pages/AdminStaffPage';
import { AdminSettingsPage } from './pages/AdminSettingsPage';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

const ProtectedLayout: React.FC<{ requiredRole?: 'ADMIN' | 'STAFF' }> = ({ requiredRole }) => {
  const { user, isLoading, hasRole } = useAuth();
  const location = useLocation();

  if (isLoading) {
    return (
      <div className="min-h-screen bg-slate-100 dark:bg-slate-950 flex items-center justify-center text-slate-600 dark:text-slate-400">
        Loading session...
      </div>
    );
  }

  if (!user) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  if (requiredRole && !hasRole(requiredRole)) {
    return <Navigate to="/pos" replace />;
  }

  return (
    <div className="min-h-screen bg-slate-100 dark:bg-slate-950 text-slate-900 dark:text-slate-100 flex flex-col font-sans transition-colors duration-300">
      <Navbar />
      <main className="flex-1">
        <Outlet />
      </main>
    </div>
  );
};

export const App: React.FC = () => {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <BrowserRouter>
          <AuthProvider>
            <Routes>
              <Route path="/login" element={<LoginPage />} />

              {/* Staff / General Protected Routes */}
              <Route element={<ProtectedLayout />}>
                <Route path="/pos" element={<StaffPOSPage />} />
              </Route>

              {/* Admin Only Protected Routes */}
              <Route element={<ProtectedLayout requiredRole="ADMIN" />}>
                <Route path="/admin/inventory" element={<AdminInventoryPage />} />
                <Route path="/admin/expiry" element={<AdminExpiryPage />} />
                <Route path="/admin/financials" element={<AdminFinancialsPage />} />
                <Route path="/admin/staff" element={<AdminStaffPage />} />
                <Route path="/admin/settings" element={<AdminSettingsPage />} />
              </Route>

              <Route path="*" element={<Navigate to="/pos" replace />} />
            </Routes>
          </AuthProvider>
        </BrowserRouter>
      </ThemeProvider>
    </QueryClientProvider>
  );
};

export default App;
