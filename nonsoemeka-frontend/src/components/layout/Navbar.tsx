import React from 'react';
import { NavLink, useNavigate } from 'react-router-dom';
import { useAuth } from '../../lib/auth';
import { useTheme } from '../../lib/theme';
import {
  ShoppingBag,
  Package,
  AlertTriangle,
  BarChart3,
  Users,
  Settings,
  LogOut,
  Pill,
  ShieldAlert,
  Sun,
  Moon,
} from 'lucide-react';

export const Navbar: React.FC = () => {
  const { user, logout, hasRole } = useAuth();
  const { isDark, toggleTheme } = useTheme();
  const navigate = useNavigate();

  const handleLogout = async () => {
    await logout();
    navigate('/login');
  };

  return (
    <header className="sticky top-0 z-40 bg-slate-900/90 dark:bg-slate-900/90 light:bg-white/90 backdrop-blur-md border-b border-slate-800 dark:border-slate-800 light:border-slate-200 shadow-sm transition-colors duration-300">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex items-center justify-between h-16">
          {/* Logo & Brand */}
          <div className="flex items-center gap-3">
            <div className="p-2 bg-gradient-to-tr from-emerald-600 to-teal-500 rounded-xl shadow-lg shadow-emerald-900/40">
              <Pill className="w-6 h-6 text-white" />
            </div>
            <div>
              <span className="text-lg font-bold bg-gradient-to-r from-emerald-500 via-teal-500 to-emerald-600 dark:from-slate-100 dark:via-teal-200 dark:to-emerald-400 bg-clip-text text-transparent">
                Nonsoemeka POS
              </span>
              <span className="hidden sm:inline-block ml-2 px-2 py-0.5 text-[10px] font-semibold tracking-wide uppercase bg-emerald-500/10 text-emerald-500 dark:text-emerald-400 border border-emerald-500/20 rounded-full">
                FEFO Enabled
              </span>
            </div>
          </div>

          {/* Navigation Links */}
          <nav className="hidden md:flex items-center space-x-1">
            {/* Staff / POS link */}
            <NavLink
              to="/pos"
              className={({ isActive }) =>
                `flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                  isActive
                    ? 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30'
                    : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800'
                }`
              }
            >
              <ShoppingBag className="w-4 h-4" />
              <span>POS Terminal</span>
            </NavLink>

            {/* Admin Links */}
            {hasRole('ADMIN') && (
              <>
                <NavLink
                  to="/admin/inventory"
                  className={({ isActive }) =>
                    `flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                      isActive
                        ? 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30'
                        : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800'
                    }`
                  }
                >
                  <Package className="w-4 h-4" />
                  <span>Inventory</span>
                </NavLink>

                <NavLink
                  to="/admin/expiry"
                  className={({ isActive }) =>
                    `flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                      isActive
                        ? 'bg-amber-500/15 text-amber-600 dark:text-amber-400 border border-amber-500/30'
                        : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800'
                    }`
                  }
                >
                  <AlertTriangle className="w-4 h-4" />
                  <span>Expiry Alerts</span>
                </NavLink>

                <NavLink
                  to="/admin/financials"
                  className={({ isActive }) =>
                    `flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                      isActive
                        ? 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30'
                        : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800'
                    }`
                  }
                >
                  <BarChart3 className="w-4 h-4" />
                  <span>Financials</span>
                </NavLink>

                <NavLink
                  to="/admin/staff"
                  className={({ isActive }) =>
                    `flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                      isActive
                        ? 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30'
                        : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800'
                    }`
                  }
                >
                  <Users className="w-4 h-4" />
                  <span>Staff & Audit</span>
                </NavLink>

                <NavLink
                  to="/admin/settings"
                  className={({ isActive }) =>
                    `flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                      isActive
                        ? 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border border-emerald-500/30'
                        : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800'
                    }`
                  }
                >
                  <Settings className="w-4 h-4" />
                  <span>Settings</span>
                </NavLink>
              </>
            )}
          </nav>

          {/* Theme Toggle, User Profile & Logout */}
          <div className="flex items-center gap-3">
            {/* Theme Toggle Button */}
            <button
              onClick={toggleTheme}
              className="p-2 rounded-xl border border-slate-700 dark:border-slate-700 light:border-slate-200 bg-slate-800/80 dark:bg-slate-800/80 light:bg-slate-100 text-amber-400 dark:text-amber-400 light:text-indigo-600 hover:scale-105 transition-all"
              title={`Switch to ${isDark ? 'Light' : 'Dark'} Mode`}
            >
              {isDark ? <Sun className="w-5 h-5 text-amber-400" /> : <Moon className="w-5 h-5 text-indigo-600" />}
            </button>

            {user && (
              <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-slate-800/80 dark:bg-slate-800/80 light:bg-slate-100 border border-slate-700 dark:border-slate-700 light:border-slate-200">
                <div className="w-7 h-7 rounded-full bg-slate-700 dark:bg-slate-700 light:bg-slate-200 flex items-center justify-center text-xs font-bold text-emerald-500 dark:text-emerald-400">
                  {user.username.charAt(0).toUpperCase()}
                </div>
                <div className="text-xs text-left hidden sm:block">
                  <div className="font-semibold text-slate-200 dark:text-slate-200 light:text-slate-800">{user.username}</div>
                  <div className="text-[10px] text-slate-400 dark:text-slate-400 light:text-slate-500 uppercase font-medium flex items-center gap-1">
                    {user.role === 'ADMIN' ? (
                      <ShieldAlert className="w-3 h-3 text-amber-400" />
                    ) : null}
                    {user.role}
                  </div>
                </div>
              </div>
            )}

            <button
              onClick={handleLogout}
              className="p-2 rounded-xl text-slate-400 hover:text-rose-500 hover:bg-rose-500/10 transition-colors"
              title="Logout"
            >
              <LogOut className="w-5 h-5" />
            </button>
          </div>
        </div>
      </div>
    </header>
  );
};
