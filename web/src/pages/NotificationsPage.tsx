import React, { useState, useEffect, useCallback } from 'react';
import { Check, Clock, Sparkles, Info, Loader2, Inbox } from 'lucide-react';
import { api } from '../api/client';
import { Notification } from '../types';
import { format } from 'date-fns';

export const NotificationsPage: React.FC = () => {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [filter, setFilter] = useState<'all' | 'unread'>('all');
  const [loading, setLoading] = useState(true);

  const fetchNotifications = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.get<Notification[]>('/notifications?limit=50');
      setNotifications(Array.isArray(res.data) ? res.data : []);
    } catch (err) {
      console.error('Failed to load notifications:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchNotifications();
  }, [fetchNotifications]);

  const markAsRead = async (id: string) => {
    try {
      await api.post(`/notifications/${id}/read`);
      setNotifications((prev) =>
        prev.map((n) => (n.id === id ? { ...n, read: true } : n))
      );
    } catch (err) {
      console.error('Failed to mark read:', err);
    }
  };

  const markAllAsRead = async () => {
    const unread = notifications.filter((n) => !n.read);
    try {
      await Promise.all(unread.map((n) => api.post(`/notifications/${n.id}/read`)));
      setNotifications((prev) => prev.map((n) => ({ ...n, read: true })));
    } catch (err) {
      console.error('Failed to mark all as read:', err);
    }
  };

  const filtered = notifications.filter((n) => (filter === 'unread' ? !n.read : true));

  return (
    <div className="max-w-4xl mx-auto space-y-6 pb-16 font-sans">
      <div className="space-y-1">
        <div className="text-3xl select-none">📥</div>
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold tracking-tight text-[#ebebeb]">
            Inbox
          </h1>

          <div className="flex items-center gap-2">
            <div className="flex rounded bg-[#202020] p-0.5 border border-[#2e2e2e]">
              <button
                onClick={() => setFilter('all')}
                className={`px-2.5 py-0.5 rounded text-xs font-medium transition-colors ${
                  filter === 'all' ? 'bg-[#2e2e2e] text-[#ebebeb]' : 'text-[#888888] hover:text-[#ebebeb]'
                }`}
              >
                All
              </button>
              <button
                onClick={() => setFilter('unread')}
                className={`px-2.5 py-0.5 rounded text-xs font-medium transition-colors ${
                  filter === 'unread' ? 'bg-[#2e2e2e] text-[#ebebeb]' : 'text-[#888888] hover:text-[#ebebeb]'
                }`}
              >
                Unread
              </button>
            </div>

            <button
              onClick={markAllAsRead}
              className="text-xs text-[#888888] hover:text-[#ebebeb] px-2.5 py-1 rounded hover:bg-[#252525] transition-colors"
            >
              Mark all read
            </button>
          </div>
        </div>
      </div>

      <div className="notion-card overflow-hidden">
        {loading ? (
          <div className="py-20 flex items-center justify-center">
            <Loader2 className="w-5 h-5 text-[#888888] animate-spin" />
          </div>
        ) : filtered.length === 0 ? (
          <div className="py-16 text-center">
            <Inbox className="w-8 h-8 text-[#555555] mx-auto mb-2" />
            <p className="text-xs font-medium text-[#888888]">Inbox is clear</p>
          </div>
        ) : (
          <div className="divide-y divide-[#2a2a2a]/60">
            {filtered.map((item) => (
              <div
                key={item.id}
                className={`p-3.5 flex items-start justify-between gap-3 hover:bg-[#242424] transition-colors ${
                  item.read ? 'opacity-50' : 'bg-[#202020]'
                }`}
              >
                <div className="flex items-start gap-3 min-w-0">
                  <Clock className="w-4 h-4 text-[#888888] mt-0.5 shrink-0" />
                  <div>
                    <div className="flex items-center gap-2">
                      <h4 className="text-xs font-semibold text-[#ebebeb]">{item.title}</h4>
                      <span className="text-[10px] px-1.5 py-0.2 rounded bg-[#2b2b2b] text-[#888888] border border-[#333333] capitalize">
                        {item.kind}
                      </span>
                    </div>
                    <p className="text-xs text-[#999999] mt-0.5">{item.body}</p>
                    <span className="text-[10px] text-[#666666] mt-1 block font-mono">
                      {format(new Date(item.created_at), 'PPP p')}
                    </span>
                  </div>
                </div>

                {!item.read && (
                  <button
                    onClick={() => markAsRead(item.id)}
                    className="text-xs text-[#2383e2] hover:underline font-medium shrink-0 flex items-center gap-1"
                  >
                    <Check className="w-3 h-3" />
                    <span>Done</span>
                  </button>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};
