import React, { useState, useEffect } from 'react';
import { useSettings, useUpdateSettings } from '../hooks/usePharmacy';
import { Settings as SettingsIcon, Save, CheckCircle2 } from 'lucide-react';

export const AdminSettingsPage: React.FC = () => {
  const { data: settingsData, isLoading } = useSettings();
  const updateSettings = useUpdateSettings();

  const [form, setForm] = useState({
    default_markup_percentage: '25.00',
    expiry_alert_days: 90,
    low_stock_threshold: 10,
    pharmacy_name: 'Nonsoemeka Pharmacy',
    receipt_footer: 'Thank you for choosing Nonsoemeka Pharmacy!',
  });

  const [savedSuccess, setSavedSuccess] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  useEffect(() => {
    if (settingsData) {
      setForm({
        default_markup_percentage: settingsData.default_markup_percentage || '25.00',
        expiry_alert_days: settingsData.expiry_alert_days || 90,
        low_stock_threshold: settingsData.low_stock_threshold || 10,
        pharmacy_name: settingsData.pharmacy_name || 'Nonsoemeka Pharmacy',
        receipt_footer: settingsData.receipt_footer || 'Thank you for choosing Nonsoemeka Pharmacy!',
      });
    }
  }, [settingsData]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSavedSuccess(false);
    setErrorMsg(null);

    try {
      await updateSettings.mutateAsync(form);
      setSavedSuccess(true);
      setTimeout(() => setSavedSuccess(false), 3000);
    } catch (err: unknown) {
      if (err instanceof Error) setErrorMsg(err.message);
    }
  };

  return (
    <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-100 flex items-center gap-2">
          <SettingsIcon className="w-7 h-7 text-emerald-400" />
          System Settings & Configuration
        </h1>
        <p className="text-sm text-slate-400">Configure global POS behavior, expiry thresholds, and receipt layout</p>
      </div>

      {savedSuccess && (
        <div className="p-4 rounded-2xl bg-emerald-500/10 border border-emerald-500/30 text-emerald-300 text-sm flex items-center gap-2">
          <CheckCircle2 className="w-5 h-5 text-emerald-400" />
          <span>System settings updated successfully and logged to audit trail.</span>
        </div>
      )}

      {errorMsg && (
        <div className="p-4 rounded-2xl bg-rose-500/10 border border-rose-500/30 text-rose-300 text-sm">
          {errorMsg}
        </div>
      )}

      <div className="bg-slate-900/90 border border-slate-800 rounded-3xl p-6 shadow-2xl">
        {isLoading ? (
          <div className="p-8 text-center text-slate-400">Loading configuration settings...</div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-6">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <label className="block text-xs font-semibold text-slate-300 uppercase mb-2">
                  Pharmacy Business Name
                </label>
                <input
                  type="text"
                  required
                  value={form.pharmacy_name}
                  onChange={(e) => setForm({ ...form, pharmacy_name: e.target.value })}
                  className="w-full p-3 bg-slate-950 border border-slate-800 rounded-xl text-slate-100 text-sm"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-300 uppercase mb-2">
                  Default Batch Markup (%)
                </label>
                <input
                  type="text"
                  required
                  value={form.default_markup_percentage}
                  onChange={(e) => setForm({ ...form, default_markup_percentage: e.target.value })}
                  className="w-full p-3 bg-slate-950 border border-slate-800 rounded-xl text-slate-100 text-sm"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-300 uppercase mb-2">
                  Expiry Alert Threshold (Days)
                </label>
                <input
                  type="number"
                  required
                  min={1}
                  value={form.expiry_alert_days}
                  onChange={(e) => setForm({ ...form, expiry_alert_days: parseInt(e.target.value) })}
                  className="w-full p-3 bg-slate-950 border border-slate-800 rounded-xl text-slate-100 text-sm"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-300 uppercase mb-2">
                  Low Stock Threshold (Units)
                </label>
                <input
                  type="number"
                  required
                  min={1}
                  value={form.low_stock_threshold}
                  onChange={(e) => setForm({ ...form, low_stock_threshold: parseInt(e.target.value) })}
                  className="w-full p-3 bg-slate-950 border border-slate-800 rounded-xl text-slate-100 text-sm"
                />
              </div>
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-300 uppercase mb-2">
                Receipt Footer Text
              </label>
              <textarea
                required
                rows={3}
                value={form.receipt_footer}
                onChange={(e) => setForm({ ...form, receipt_footer: e.target.value })}
                className="w-full p-3 bg-slate-950 border border-slate-800 rounded-xl text-slate-100 text-sm"
              />
            </div>

            <button
              type="submit"
              disabled={updateSettings.isPending}
              className="py-3 px-6 bg-gradient-to-r from-emerald-600 to-teal-500 hover:from-emerald-500 hover:to-teal-400 text-white font-bold rounded-xl shadow-lg flex items-center justify-center gap-2 transition-all glow-emerald disabled:opacity-50"
            >
              <Save className="w-4 h-4" />
              <span>{updateSettings.isPending ? 'Saving Settings...' : 'Save Configuration'}</span>
            </button>
          </form>
        )}
      </div>
    </div>
  );
};
