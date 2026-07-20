import React from 'react';
import { useExpiringBatches, useSettings } from '../hooks/usePharmacy';
import { formatMoney } from '../lib/money';
import { AlertTriangle, Clock, Calendar, ShieldAlert } from 'lucide-react';

export const AdminExpiryPage: React.FC = () => {
  const { data: expiryData, isLoading } = useExpiringBatches();
  const { data: settings } = useSettings();

  const alertDays = settings?.expiry_alert_days || 90;

  const getExpiryCategory = (expiryDateStr: string) => {
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    const expDate = new Date(expiryDateStr);
    expDate.setHours(0, 0, 0, 0);

    const diffTime = expDate.getTime() - today.getTime();
    const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));

    if (diffDays <= 0) {
      return {
        label: 'EXPIRED',
        days: diffDays,
        bg: 'bg-rose-500/20 text-rose-300 border-rose-500/30',
        badgeBg: 'bg-rose-600 text-white',
      };
    } else if (diffDays <= alertDays) {
      return {
        label: `CRITICAL (< ${alertDays}d)`,
        days: diffDays,
        bg: 'bg-amber-500/20 text-amber-300 border-amber-500/30',
        badgeBg: 'bg-amber-600 text-white',
      };
    } else {
      return {
        label: `WARNING (< ${alertDays * 2}d)`,
        days: diffDays,
        bg: 'bg-yellow-500/10 text-yellow-300 border-yellow-500/20',
        badgeBg: 'bg-yellow-600 text-slate-950',
      };
    }
  };

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-100 flex items-center gap-2">
            <AlertTriangle className="w-7 h-7 text-amber-400" />
            Pharmaceutical Expiry Portal
          </h1>
          <p className="text-sm text-slate-400">
            Batches flagged by FEFO thresholds (Alert threshold configured at {alertDays} days)
          </p>
        </div>
      </div>

      <div className="bg-slate-900/80 border border-slate-800 rounded-3xl overflow-hidden shadow-2xl">
        {isLoading ? (
          <div className="p-12 text-center text-slate-400">Loading expiry data...</div>
        ) : expiryData?.data.length === 0 ? (
          <div className="p-12 text-center text-slate-500 space-y-2">
            <ShieldAlert className="w-12 h-12 mx-auto text-emerald-400 opacity-60" />
            <p className="text-base font-semibold text-slate-300">No Expiring Batches Detected</p>
            <p className="text-xs text-slate-500">All inventory batches are within safe shelf-life parameters.</p>
          </div>
        ) : (
          <table className="w-full text-left text-sm text-slate-300">
            <thead className="bg-slate-950 text-slate-400 uppercase text-xs">
              <tr>
                <th className="py-3.5 px-4">Alert Level</th>
                <th className="py-3.5 px-4">Batch #</th>
                <th className="py-3.5 px-4">Product Name</th>
                <th className="py-3.5 px-4">Qty Remaining</th>
                <th className="py-3.5 px-4">Expiry Date</th>
                <th className="py-3.5 px-4">Days Left</th>
                <th className="py-3.5 px-4 text-right">Selling Price</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60">
              {expiryData?.data.map((batch) => {
                const cat = getExpiryCategory(batch.expiry_date);
                return (
                  <tr key={batch.id} className={`hover:bg-slate-800/40 ${cat.bg}`}>
                    <td className="py-3.5 px-4">
                      <span className={`px-2.5 py-1 rounded-md text-[10px] font-bold tracking-wider ${cat.badgeBg}`}>
                        {cat.label}
                      </span>
                    </td>
                    <td className="py-3.5 px-4 font-mono font-semibold text-slate-100">{batch.batch_number}</td>
                    <td className="py-3.5 px-4 text-slate-200 font-medium">{batch.product_name || 'N/A'}</td>
                    <td className="py-3.5 px-4 font-bold text-slate-100">{batch.quantity_remaining} units</td>
                    <td className="py-3.5 px-4 font-mono text-xs flex items-center gap-1 text-slate-300">
                      <Calendar className="w-3.5 h-3.5 text-slate-400" />
                      {batch.expiry_date}
                    </td>
                    <td className="py-3.5 px-4 font-semibold">
                      <span className="flex items-center gap-1">
                        <Clock className="w-3.5 h-3.5" />
                        {cat.days <= 0 ? 'Expired' : `${cat.days} days`}
                      </span>
                    </td>
                    <td className="py-3.5 px-4 text-right font-bold text-slate-100">
                      {formatMoney(batch.selling_price)}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
};
