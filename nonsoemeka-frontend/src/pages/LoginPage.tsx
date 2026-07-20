import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../lib/auth';
import { useTheme } from '../lib/theme';
import { Pill, Lock, User, AlertCircle, ArrowRight, Sun, Moon } from 'lucide-react';

export const LoginPage: React.FC = () => {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const { isDark, toggleTheme } = useTheme();
  const { login } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!username || !password) {
      setError('Please fill in both fields.');
      return;
    }

    setError(null);
    setIsSubmitting(true);

    try {
      const user = await login(username, password);
      if (user.role === 'ADMIN') {
        navigate('/admin/inventory');
      } else {
        navigate('/pos');
      }
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError('Login failed. Please check credentials.');
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div
      className={`min-h-screen flex items-center justify-center p-4 relative overflow-hidden transition-colors duration-300 ${
        isDark
          ? 'bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950 text-slate-100'
          : 'bg-gradient-to-br from-slate-100 via-emerald-50/40 to-teal-50 text-slate-800'
      }`}
    >
      {/* Decorative ambient background glows */}
      <div
        className={`absolute top-1/4 left-1/4 w-96 h-96 rounded-full blur-3xl pointer-events-none transition-opacity duration-300 ${
          isDark ? 'bg-emerald-500/10' : 'bg-emerald-400/20'
        }`}
      />
      <div
        className={`absolute bottom-1/4 right-1/4 w-96 h-96 rounded-full blur-3xl pointer-events-none transition-opacity duration-300 ${
          isDark ? 'bg-teal-500/10' : 'bg-teal-400/20'
        }`}
      />

      {/* Theme Toggle Button (Top Right) */}
      <button
        onClick={toggleTheme}
        className={`absolute top-6 right-6 p-3 rounded-2xl border transition-all shadow-md flex items-center gap-2 text-sm font-medium z-20 ${
          isDark
            ? 'bg-slate-900/90 border-slate-700 text-amber-400 hover:bg-slate-800 hover:border-amber-400/50'
            : 'bg-white/90 border-slate-200 text-slate-700 hover:bg-slate-50 hover:text-emerald-600'
        }`}
        title={`Switch to ${isDark ? 'Light' : 'Dark'} Mode`}
      >
        {isDark ? (
          <>
            <Sun className="w-5 h-5 text-amber-400" />
            <span className="text-slate-300">Light Mode</span>
          </>
        ) : (
          <>
            <Moon className="w-5 h-5 text-indigo-600" />
            <span className="text-slate-700">Dark Mode</span>
          </>
        )}
      </button>

      {/* Main Glass Card */}
      <div
        className={`w-full max-w-md rounded-3xl p-8 relative z-10 shadow-2xl transition-all duration-300 border ${
          isDark
            ? 'glass-card border-slate-800'
            : 'bg-white/80 backdrop-blur-xl border-slate-200/80 shadow-slate-300/50'
        }`}
      >
        {/* Header */}
        <div className="text-center mb-8">
          <div className="inline-flex p-3 bg-gradient-to-tr from-emerald-600 to-teal-500 rounded-2xl shadow-xl shadow-emerald-900/40 mb-4 glow-emerald">
            <Pill className="w-8 h-8 text-white" />
          </div>
          <h1
            className={`text-2xl font-bold tracking-tight ${
              isDark ? 'text-slate-100' : 'text-slate-900'
            }`}
          >
            Nonsoemeka Pharmacy
          </h1>
          <p className={`text-sm mt-1 ${isDark ? 'text-slate-400' : 'text-slate-500'}`}>
            Point of Sale & Inventory Management
          </p>
        </div>

        {/* Error Alert */}
        {error && (
          <div className="mb-6 p-4 rounded-2xl bg-rose-500/10 border border-rose-500/30 text-rose-300 text-sm flex items-center gap-3 animate-in fade-in duration-200">
            <AlertCircle className="w-5 h-5 text-rose-400 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {/* Login Form */}
        <form onSubmit={handleSubmit} className="space-y-5">
          <div>
            <label
              className={`block text-xs font-semibold uppercase tracking-wider mb-2 ${
                isDark ? 'text-slate-300' : 'text-slate-700'
              }`}
            >
              Username
            </label>
            <div className="relative">
              <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-slate-400">
                <User className="w-5 h-5" />
              </div>
              <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="Enter your username"
                className={`w-full pl-11 pr-4 py-3 rounded-xl border transition-all text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500 ${
                  isDark
                    ? 'bg-slate-900/90 border-slate-700/80 text-slate-100 placeholder-slate-500 focus:border-emerald-500'
                    : 'bg-slate-50 border-slate-300 text-slate-900 placeholder-slate-400 focus:border-emerald-500'
                }`}
                autoComplete="username"
                required
              />
            </div>
          </div>

          <div>
            <label
              className={`block text-xs font-semibold uppercase tracking-wider mb-2 ${
                isDark ? 'text-slate-300' : 'text-slate-700'
              }`}
            >
              Password
            </label>
            <div className="relative">
              <div className="absolute inset-y-0 left-0 pl-3.5 flex items-center pointer-events-none text-slate-400">
                <Lock className="w-5 h-5" />
              </div>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                className={`w-full pl-11 pr-4 py-3 rounded-xl border transition-all text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500 ${
                  isDark
                    ? 'bg-slate-900/90 border-slate-700/80 text-slate-100 placeholder-slate-500 focus:border-emerald-500'
                    : 'bg-slate-50 border-slate-300 text-slate-900 placeholder-slate-400 focus:border-emerald-500'
                }`}
                autoComplete="current-password"
                required
              />
            </div>
          </div>

          <button
            type="submit"
            disabled={isSubmitting}
            className="w-full py-3.5 px-4 bg-gradient-to-r from-emerald-600 to-teal-500 hover:from-emerald-500 hover:to-teal-400 text-white font-semibold rounded-xl shadow-lg shadow-emerald-950/30 flex items-center justify-center gap-2 transition-all group disabled:opacity-50"
          >
            <span>{isSubmitting ? 'Authenticating...' : 'Sign In to POS'}</span>
            <ArrowRight className="w-4 h-4 group-hover:translate-x-1 transition-transform" />
          </button>
        </form>

        <div
          className={`mt-8 text-center text-xs border-t pt-4 ${
            isDark ? 'text-slate-500 border-slate-800/80' : 'text-slate-500 border-slate-200'
          }`}
        >
          Default Admin:{' '}
          <code className={isDark ? 'text-slate-400' : 'text-slate-700 font-semibold'}>
            admin / AdminPass123!
          </code>{' '}
          | Staff:{' '}
          <code className={isDark ? 'text-slate-400' : 'text-slate-700 font-semibold'}>
            staff / StaffPass123!
          </code>
        </div>
      </div>
    </div>
  );
};
