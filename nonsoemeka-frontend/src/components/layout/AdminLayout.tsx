import React from 'react';
import { Outlet, useLocation, Navigate } from 'react-router-dom';
import { useAuth } from '../../lib/auth';
import { Navbar } from './Navbar';
import { AdminSidebar } from './AdminSidebar';

export const AdminLayout: React.FC = () => {
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

  if (!hasRole('ADMIN')) {
    return <Navigate to="/pos" replace />;
  }

  return (
    <div className="min-h-screen bg-slate-100 dark:bg-slate-950 text-slate-900 dark:text-slate-100 flex flex-col font-sans transition-colors duration-300">
      <Navbar variant="admin" />
      <div className="flex flex-1">
        <AdminSidebar />
        <main className="flex-1 min-w-0">
          <Outlet />
        </main>
      </div>
    </div>
  );
};
