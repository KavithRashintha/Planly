import React, { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { AlertCircle } from 'lucide-react';
import { Persona } from '../types';

export const Register: React.FC = () => {
  const { register } = useAuth();
  const navigate = useNavigate();

  const detectedTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC';

  const [fullName, setFullName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [persona, setPersona] = useState<Persona>('undergraduate');
  const [timezone, setTimezone] = useState(detectedTimezone);
  const [workStart, setWorkStart] = useState('09:00');
  const [workEnd, setWorkEnd] = useState('17:00');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      await register({
        full_name: fullName.trim(),
        email: email.trim(),
        password,
        persona,
        timezone,
        work_start: workStart,
        work_end: workEnd,
      });
      navigate('/dashboard');
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Registration failed';
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-[#191919] flex items-center justify-center p-4 font-sans select-none text-xs py-10">
      <div className="w-full max-w-sm space-y-6">
        <div className="text-center space-y-2">
          <div className="w-10 h-10 rounded-lg bg-[#252525] border border-[#333333] flex items-center justify-center text-lg font-bold text-[#ebebeb] mx-auto shadow-sm">
            P
          </div>
          <h1 className="text-xl font-bold tracking-tight text-[#ebebeb]">
            Create Planly Account
          </h1>
          <p className="text-xs text-[#888888]">
            Set up your minimal workspace
          </p>
        </div>

        <div className="notion-card p-6 shadow-xl">
          <form onSubmit={handleSubmit} className="space-y-3.5">
            {error && (
              <div className="flex items-center gap-2 p-2.5 bg-[#3d1f22] border border-[#592b30] rounded text-[#e57373]">
                <AlertCircle className="w-3.5 h-3.5 shrink-0" />
                <span>{error}</span>
              </div>
            )}

            <div>
              <label className="block text-[#888888] mb-1">
                Full Name
              </label>
              <input
                type="text"
                required
                value={fullName}
                onChange={(e) => setFullName(e.target.value)}
                placeholder="Jane Doe"
                className="w-full bg-[#1b1b1b] border border-[#2e2e2e] focus:border-[#2383e2] rounded px-3 py-1.5 text-[#ebebeb] placeholder-[#555555] focus:outline-none transition-colors"
              />
            </div>

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

            <div>
              <label className="block text-[#888888] mb-1">
                Role / Persona
              </label>
              <div className="grid grid-cols-3 gap-1.5">
                {(['student', 'undergraduate', 'employee'] as Persona[]).map((p) => (
                  <button
                    type="button"
                    key={p}
                    onClick={() => setPersona(p)}
                    className={`py-1.5 px-2 rounded border text-center transition-colors capitalize ${
                      persona === p
                        ? 'bg-[#2b2b2b] border-[#444444] text-[#ebebeb] font-semibold'
                        : 'bg-[#1b1b1b] border-[#2e2e2e] text-[#888888] hover:bg-[#222222]'
                    }`}
                  >
                    {p}
                  </button>
                ))}
              </div>
            </div>

            <div className="grid grid-cols-2 gap-2.5">
              <div>
                <label className="block text-[#888888] mb-1">
                  Timezone
                </label>
                <input
                  type="text"
                  value={timezone}
                  onChange={(e) => setTimezone(e.target.value)}
                  className="w-full bg-[#1b1b1b] border border-[#2e2e2e] focus:border-[#2383e2] rounded px-2 py-1 text-[#ebebeb] focus:outline-none"
                />
              </div>

              <div>
                <label className="block text-[#888888] mb-1">
                  Work Hours
                </label>
                <div className="flex items-center gap-1">
                  <input
                    type="time"
                    value={workStart}
                    onChange={(e) => setWorkStart(e.target.value)}
                    className="w-1/2 bg-[#1b1b1b] border border-[#2e2e2e] rounded px-1.5 py-1 text-[#ebebeb] [color-scheme:dark]"
                  />
                  <span className="text-[#666666]">-</span>
                  <input
                    type="time"
                    value={workEnd}
                    onChange={(e) => setWorkEnd(e.target.value)}
                    className="w-1/2 bg-[#1b1b1b] border border-[#2e2e2e] rounded px-1.5 py-1 text-[#ebebeb] [color-scheme:dark]"
                  />
                </div>
              </div>
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full bg-[#2383e2] hover:bg-[#1d72c5] text-white font-medium py-2 rounded text-xs transition-colors disabled:opacity-50 cursor-pointer shadow-sm mt-1"
            >
              {loading ? 'Creating account...' : 'Create Account'}
            </button>
          </form>

          <div className="mt-5 pt-4 border-t border-[#2b2b2b] text-center">
            <p className="text-xs text-[#888888]">
              Already have an account?{' '}
              <Link to="/login" className="text-[#2383e2] hover:underline font-medium">
                Log in
              </Link>
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};
