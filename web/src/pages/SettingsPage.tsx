import React, { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import { Persona } from '../types';
import { Check, AlertCircle } from 'lucide-react';

export const SettingsPage: React.FC = () => {
  const { user, updateProfile } = useAuth();

  const [fullName, setFullName] = useState('');
  const [persona, setPersona] = useState<Persona>('employee');
  const [timezone, setTimezone] = useState('UTC');
  const [workStart, setWorkStart] = useState('09:00');
  const [workEnd, setWorkEnd] = useState('17:00');

  const [loading, setLoading] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (user) {
      setFullName(user.full_name || '');
      setPersona(user.persona || 'employee');
      setTimezone(user.timezone || 'UTC');
      setWorkStart(user.work_start ? user.work_start.substring(0, 5) : '09:00');
      setWorkEnd(user.work_end ? user.work_end.substring(0, 5) : '17:00');
    }
  }, [user]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setSuccess(false);
    setError(null);

    try {
      await updateProfile({
        full_name: fullName.trim(),
        persona,
        timezone: timezone.trim(),
        work_start: workStart,
        work_end: workEnd,
      });
      setSuccess(true);
      setTimeout(() => setSuccess(false), 3000);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Failed to update profile';
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-2xl mx-auto space-y-6 pb-16 font-sans">
      <div className="space-y-1">
        <div className="text-3xl select-none">⚙️</div>
        <h1 className="text-2xl font-bold tracking-tight text-[#ebebeb]">
          Settings
        </h1>
        <p className="text-xs text-[#888888]">
          Preferences, persona mode, and workspace schedule
        </p>
      </div>

      <div className="notion-card p-6">
        <form onSubmit={handleSubmit} className="space-y-5 text-xs">
          {success && (
            <div className="flex items-center gap-2 p-2.5 bg-[#1e3224] border border-[#2b4c34] rounded text-[#62b584]">
              <Check className="w-3.5 h-3.5 shrink-0" />
              <span>Settings saved.</span>
            </div>
          )}

          {error && (
            <div className="flex items-center gap-2 p-2.5 bg-[#3d1f22] border border-[#592b30] rounded text-[#e57373]">
              <AlertCircle className="w-3.5 h-3.5 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          <div>
            <label className="block font-medium text-[#888888] mb-1">
              Email
            </label>
            <input
              type="text"
              disabled
              value={user?.email || ''}
              className="w-full bg-[#1b1b1b] border border-[#2e2e2e] rounded px-3 py-1.5 text-[#777777] cursor-not-allowed"
            />
          </div>

          <div>
            <label className="block font-medium text-[#888888] mb-1">
              Full Name
            </label>
            <input
              type="text"
              required
              value={fullName}
              onChange={(e) => setFullName(e.target.value)}
              className="w-full bg-[#1b1b1b] focus:bg-[#202020] border border-[#2e2e2e] focus:border-[#3b82f6] rounded px-3 py-1.5 text-[#ebebeb] focus:outline-none"
            />
          </div>

          <div>
            <label className="block font-medium text-[#888888] mb-1.5">
              Persona Mode
            </label>
            <div className="grid grid-cols-3 gap-2">
              {(['student', 'undergraduate', 'employee'] as Persona[]).map((p) => (
                <button
                  type="button"
                  key={p}
                  onClick={() => setPersona(p)}
                  className={`py-2 px-3 rounded border text-center transition-colors ${
                    persona === p
                      ? 'bg-[#2b2b2b] border-[#444444] text-[#ebebeb] font-semibold'
                      : 'bg-[#1b1b1b] border-[#2e2e2e] text-[#888888] hover:bg-[#222222]'
                  }`}
                >
                  <p className="capitalize">{p}</p>
                </button>
              ))}
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block font-medium text-[#888888] mb-1">
                Timezone
              </label>
              <input
                type="text"
                required
                value={timezone}
                onChange={(e) => setTimezone(e.target.value)}
                className="w-full bg-[#1b1b1b] focus:bg-[#202020] border border-[#2e2e2e] focus:border-[#3b82f6] rounded px-3 py-1.5 text-[#ebebeb] focus:outline-none"
              />
            </div>

            <div>
              <label className="block font-medium text-[#888888] mb-1">
                Work Hours
              </label>
              <div className="flex items-center gap-1.5">
                <input
                  type="time"
                  required
                  value={workStart}
                  onChange={(e) => setWorkStart(e.target.value)}
                  className="w-1/2 bg-[#1b1b1b] border border-[#2e2e2e] rounded px-2 py-1.5 text-[#ebebeb] [color-scheme:dark]"
                />
                <span className="text-[#666666]">-</span>
                <input
                  type="time"
                  required
                  value={workEnd}
                  onChange={(e) => setWorkEnd(e.target.value)}
                  className="w-1/2 bg-[#1b1b1b] border border-[#2e2e2e] rounded px-2 py-1.5 text-[#ebebeb] [color-scheme:dark]"
                />
              </div>
            </div>
          </div>

          <div className="pt-3 border-t border-[#2e2e2e] flex justify-end">
            <button
              type="submit"
              disabled={loading}
              className="bg-[#2383e2] hover:bg-[#1d72c5] text-white font-medium px-4 py-1.5 rounded text-xs transition-colors cursor-pointer disabled:opacity-50"
            >
              {loading ? 'Saving...' : 'Save'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
