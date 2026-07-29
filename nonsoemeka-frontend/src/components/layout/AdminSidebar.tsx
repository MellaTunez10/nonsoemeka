import React from 'react';
import { NavLink } from 'react-router-dom';
import {
  ShoppingBag,
  Package,
  AlertTriangle,
  BarChart3,
  Users,
  Settings,
} from 'lucide-react';

const sidebarLinks = [
  { to: '/pos', label: 'POS Terminal', icon: ShoppingBag, color: 'emerald' },
  { to: '/admin/inventory', label: 'Inventory', icon: Package, color: 'emerald' },
  { to: '/admin/expiry', label: 'Expiry Alerts', icon: AlertTriangle, color: 'amber' },
  { to: '/admin/financials', label: 'Financials', icon: BarChart3, color: 'emerald' },
  { to: '/admin/staff', label: 'Staff & Audit', icon: Users, color: 'emerald' },
  { to: '/admin/settings', label: 'Settings', icon: Settings, color: 'emerald' },
];

const activeStyles: Record<string, string> = {
  emerald: 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border-r-[3px] border-r-emerald-500',
  amber: 'bg-amber-500/15 text-amber-600 dark:text-amber-400 border-r-[3px] border-r-amber-500',
};

const iconColors: Record<string, string> = {
  emerald: 'text-emerald-500',
  amber: 'text-amber-500',
};

export const AdminSidebar: React.FC = () => {
  return (
    <aside className="hidden lg:flex flex-col w-60 shrink-0 bg-white dark:bg-slate-900 border-r border-slate-200 dark:border-slate-800 h-[calc(100vh-4rem)] sticky top-16 overflow-y-auto transition-colors duration-300">
      {/* Section Label */}
      <div className="px-5 pt-6 pb-2">
        <span className="text-[11px] font-semibold uppercase tracking-widest text-slate-400 dark:text-slate-500">
          Navigation
        </span>
      </div>

      {/* Nav Links */}
      <nav className="flex-1 flex flex-col gap-0.5 px-3 pb-6">
        {sidebarLinks.map(({ to, label, icon: Icon, color }) => (
          <NavLink
            key={to}
            to={to}
            end={to === '/pos'}
            className={({ isActive }) =>
              `group flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all duration-200 ${
                isActive
                  ? activeStyles[color]
                  : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800/60'
              }`
            }
          >
            {({ isActive }) => (
              <>
                <Icon
                  className={`w-[18px] h-[18px] shrink-0 transition-colors ${
                    isActive ? '' : iconColors[color] + ' opacity-70 group-hover:opacity-100'
                  }`}
                />
                <span>{label}</span>
              </>
            )}
          </NavLink>
        ))}
      </nav>
    </aside>
  );
};
