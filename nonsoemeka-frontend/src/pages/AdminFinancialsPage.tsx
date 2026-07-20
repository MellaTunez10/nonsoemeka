import React from 'react';
import { useFinancialSummary, useSalesTrends, useTopProducts } from '../hooks/usePharmacy';
import { useTheme } from '../lib/theme';
import { formatMoney } from '../lib/money';
import {
  DollarSign,
  TrendingUp,
  PieChart as PieIcon,
  ShoppingBag,
  Award,
} from 'lucide-react';
import {
  ResponsiveContainer,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  BarChart,
  Bar,
  CartesianGrid,
} from 'recharts';

export const AdminFinancialsPage: React.FC = () => {
  const { data: fin } = useFinancialSummary();
  const { data: trends } = useSalesTrends();
  const { data: topProds } = useTopProducts(5);
  const { isDark } = useTheme();

  const formattedTrendData = (trends?.data || []).map((t) => ({
    date: t.date,
    revenue: parseFloat(t.total_amount || '0'),
    sales: t.sales_count,
  }));

  const formattedTopData = (topProds?.data || []).map((p) => ({
    name: p.product_name,
    quantity: p.total_quantity,
    revenue: parseFloat(p.total_revenue || '0'),
  }));

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-8">
      <div>
        <h1 className="text-2xl font-bold dark:text-slate-100 light:text-slate-900 flex items-center gap-2">
          <TrendingUp className="w-7 h-7 text-emerald-500" />
          Financial Analytics & Reporting
        </h1>
        <p className="text-sm dark:text-slate-400 light:text-slate-600">Revenue, cost breakdown, profit margins, and top selling products</p>
      </div>

      {/* Metric Cards Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
        <div className="dark:bg-slate-900/90 light:bg-white border dark:border-slate-800 light:border-slate-200 rounded-3xl p-5 shadow-md">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold dark:text-slate-400 light:text-slate-500 uppercase tracking-wider">Total Revenue</span>
            <div className="p-2 bg-emerald-500/10 rounded-xl text-emerald-500 border border-emerald-500/20">
              <DollarSign className="w-5 h-5" />
            </div>
          </div>
          <div className="text-2xl font-bold dark:text-slate-100 light:text-slate-900 mt-3">{formatMoney(fin?.total_revenue || '0')}</div>
          <div className="text-xs dark:text-slate-500 light:text-slate-500 mt-1">{fin?.total_sales_count || 0} completed transactions</div>
        </div>

        <div className="dark:bg-slate-900/90 light:bg-white border dark:border-slate-800 light:border-slate-200 rounded-3xl p-5 shadow-md">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold dark:text-slate-400 light:text-slate-500 uppercase tracking-wider">Inventory Cost</span>
            <div className="p-2 bg-rose-500/10 rounded-xl text-rose-500 border border-rose-500/20">
              <ShoppingBag className="w-5 h-5" />
            </div>
          </div>
          <div className="text-2xl font-bold dark:text-slate-100 light:text-slate-900 mt-3">{formatMoney(fin?.total_cost || '0')}</div>
          <div className="text-xs dark:text-slate-500 light:text-slate-500 mt-1">{fin?.total_items_sold || 0} units dispensed</div>
        </div>

        <div className="dark:bg-slate-900/90 light:bg-white border dark:border-slate-800 light:border-slate-200 rounded-3xl p-5 shadow-md">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold dark:text-slate-400 light:text-slate-500 uppercase tracking-wider">Gross Profit</span>
            <div className="p-2 bg-teal-500/10 rounded-xl text-teal-500 border border-teal-500/20">
              <TrendingUp className="w-5 h-5" />
            </div>
          </div>
          <div className="text-2xl font-bold text-emerald-500 mt-3">
            {formatMoney(fin?.total_gross_profit || '0')}
          </div>
          <div className="text-xs dark:text-slate-500 light:text-slate-500 mt-1">Calculated via FEFO cost margins</div>
        </div>

        <div className="dark:bg-slate-900/90 light:bg-white border dark:border-slate-800 light:border-slate-200 rounded-3xl p-5 shadow-md">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold dark:text-slate-400 light:text-slate-500 uppercase tracking-wider">Profit Margin</span>
            <div className="p-2 bg-amber-500/10 rounded-xl text-amber-500 border border-amber-500/20">
              <PieIcon className="w-5 h-5" />
            </div>
          </div>
          <div className="text-2xl font-bold text-amber-500 mt-3">
            {fin?.profit_margin_percentage || '0.00'}%
          </div>
          <div className="text-xs dark:text-slate-500 light:text-slate-500 mt-1">Weighted average percentage</div>
        </div>
      </div>

      {/* Visual Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Sales Trend Chart */}
        <div className="lg:col-span-7 dark:bg-slate-900/80 light:bg-white border dark:border-slate-800 light:border-slate-200 rounded-3xl p-6 shadow-md">
          <h3 className="text-base font-bold dark:text-slate-100 light:text-slate-900 mb-4 flex items-center gap-2">
            <TrendingUp className="w-5 h-5 text-emerald-500" />
            Revenue & Sales Trend Over Time
          </h3>
          <div className="h-72 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={formattedTrendData}>
                <defs>
                  <linearGradient id="colorRev" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#10b981" stopOpacity={0.4} />
                    <stop offset="95%" stopColor="#10b981" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke={isDark ? '#334155' : '#e2e8f0'} opacity={0.5} />
                <XAxis dataKey="date" stroke={isDark ? '#94a3b8' : '#64748b'} fontSize={12} />
                <YAxis stroke={isDark ? '#94a3b8' : '#64748b'} fontSize={12} />
                <Tooltip
                  contentStyle={{
                    backgroundColor: isDark ? '#0f172a' : '#ffffff',
                    borderColor: isDark ? '#334155' : '#cbd5e1',
                    color: isDark ? '#f8fafc' : '#0f172a',
                    borderRadius: '12px',
                  }}
                />
                <Area type="monotone" dataKey="revenue" stroke="#10b981" fillOpacity={1} fill="url(#colorRev)" />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Top Selling Products */}
        <div className="lg:col-span-5 dark:bg-slate-900/80 light:bg-white border dark:border-slate-800 light:border-slate-200 rounded-3xl p-6 shadow-md">
          <h3 className="text-base font-bold dark:text-slate-100 light:text-slate-900 mb-4 flex items-center gap-2">
            <Award className="w-5 h-5 text-amber-500" />
            Top 5 Dispensed Products
          </h3>
          <div className="h-72 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={formattedTopData} layout="vertical">
                <CartesianGrid strokeDasharray="3 3" stroke={isDark ? '#334155' : '#e2e8f0'} opacity={0.5} />
                <XAxis type="number" stroke={isDark ? '#94a3b8' : '#64748b'} fontSize={12} />
                <YAxis dataKey="name" type="category" stroke={isDark ? '#94a3b8' : '#64748b'} fontSize={10} width={90} />
                <Tooltip
                  contentStyle={{
                    backgroundColor: isDark ? '#0f172a' : '#ffffff',
                    borderColor: isDark ? '#334155' : '#cbd5e1',
                    color: isDark ? '#f8fafc' : '#0f172a',
                    borderRadius: '12px',
                  }}
                />
                <Bar dataKey="quantity" fill="#14b8a6" radius={[0, 8, 8, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>
      </div>
    </div>
  );
};
