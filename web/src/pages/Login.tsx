import React, { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { AlertCircle } from 'lucide-react';

export const Login: React.FC = () => {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      await login(email.trim(), password);
      navigate('/dashboard');
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Invalid credentials';
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-[#191919] flex items-center justify-center p-4 font-sans select-none text-xs">
      <div className="w-full max-w-sm space-y-6">
        {/* Brand */}
        <div className="text-center space-y-2">
          <div className="w-10 h-10 rounded-lg bg-[#252525] border border-[#333333] flex items-center justify-center text-lg font-bold text-[#ebebeb] mx-auto shadow-sm">
            P
          </div>
          <h1 className="text-xl font-bold tracking-tight text-[#ebebeb]">
            Log in to Planly
          </h1>
          <p className="text-xs text-[#888888]">
            Minimalist work and schedule planner
          </p>
        </div>

        {/* Card */}
        <div className="notion-card p-6 shadow-xl">
          <form onSubmit={handleSubmit} className="space-y-4">
            {error && (
              <div className="flex items-center gap-2 p-2.5 bg-[#3d1f22] border border-[#592b30] rounded text-[#e57373]">
                <AlertCircle className="w-3.5 h-3.5 shrink-0" />
                <span>{error}</span>
              </div>
            )}

            <div>
              <label className="block text-[#888888] mb-1">
                Email
              </label>
              <input
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="name@example.com"
                className="w-full bg-[#1b1b1b] border border-[#2e2e2e] focus:border-[#2383e2] rounded px-3 py-1.5 text-[#ebebeb] placeholder-[#555555] focus:outline-none transition-colors"
              />
            </div>

            <div>
              <label className="block text-[#888888] mb-1">
                Password
              </label>
              <input
                type="password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                className="w-full bg-[#1b1b1b] border border-[#2e2e2e] focus:border-[#2383e2] rounded px-3 py-1.5 text-[#ebebeb] placeholder-[#555555] focus:outline-none transition-colors"
              />
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full bg-[#2383e2] hover:bg-[#1d72c5] text-white font-medium py-2 rounded text-xs transition-colors disabled:opacity-50 cursor-pointer shadow-sm"
            >
              {loading ? 'Continuing...' : 'Continue'}
            </button>
          </form>

          <div className="mt-5 pt-4 border-t border-[#2b2b2b] text-center">
            <p className="text-xs text-[#888888]">
              No account?{' '}
              <Link to="/register" className="text-[#2383e2] hover:underline font-medium">
                Sign up
              </Link>
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};
