import React, { useState, useEffect } from 'react';
import { Bell, Plus, Check, Clock } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { api } from '../api/client';
import { Notification } from '../types';
import { useNavigate, useLocation } from 'react-router-dom';

interface NavbarProps {
  onNewTask?: () => void;
  unreadCount: number;
  onRefreshNotifications: () => void;
}

export const Navbar: React.FC<NavbarProps> = ({ onNewTask, unreadCount, onRefreshNotifications }) => {
  const { user } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [showNotifications, setShowNotifications] = useState(false);
  const [recentNotifications, setRecentNotifications] = useState<Notification[]>([]);
  const [loadingNotifs, setLoadingNotifs] = useState(false);

  // Derive page name for breadcrumb
  const pathName = location.pathname.replace('/', '') || 'dashboard';
  const pageTitle = pathName.charAt(0).toUpperCase() + pathName.slice(1);

  useEffect(() => {
    if (showNotifications) {
      setLoadingNotifs(true);
      api.get<Notification[]>('/notifications?limit=5')
        .then((res) => setRecentNotifications(res.data || []))
        .catch(() => {})
        .finally(() => setLoadingNotifs(false));
    }
  }, [showNotifications]);

  const markAsRead = async (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      await api.post(`/notifications/${id}/read`);
      setRecentNotifications((prev) =>
        prev.map((n) => (n.id === id ? { ...n, read: true } : n))
      );
      onRefreshNotifications();
    } catch {}
  };

  return (
    <header className="h-11 border-b border-[#2b2b2b] bg-[#191919] px-6 flex items-center justify-between sticky top-0 z-30 select-none">
      {/* Notion Breadcrumbs */}
      <div className="flex items-center gap-2 text-xs">
        <span className="text-[#666666]">Workspace</span>
        <span className="text-[#444444]">/</span>
        <span className="text-[#e6e6e6] font-medium">{pageTitle}</span>

        {user?.persona && (
          <span className="ml-2 text-[10px] font-medium px-1.5 py-0.5 rounded bg-[#272727] text-[#999999] border border-[#333333] capitalize">
            {user.persona}
          </span>
        )}
      </div>

      {/* Right: Quick Action & Notification Icon */}
      <div className="flex items-center gap-2">
        {onNewTask && (
          <button
            onClick={onNewTask}
            className="flex items-center gap-1.5 bg-[#2383e2] hover:bg-[#1d72c5] text-white text-xs font-medium px-2.5 py-1 rounded-md transition-colors cursor-pointer"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>New</span>
          </button>
        )}

        {/* Notifications Popover */}
        <div className="relative">
          <button
            onClick={() => setShowNotifications(!showNotifications)}
            className="p-1.5 text-[#888888] hover:text-[#e6e6e6] hover:bg-[#272727] rounded-md relative transition-colors cursor-pointer"
            title="Notifications"
          >
            <Bell className="w-4 h-4" />
            {unreadCount > 0 && (
              <span className="absolute top-1 right-1 w-1.5 h-1.5 bg-[#e57373] rounded-full" />
            )}
          </button>

          {showNotifications && (
            <div className="absolute right-0 mt-1.5 w-76 bg-[#252525] border border-[#333333] rounded-lg shadow-xl p-3 z-50 animate-in fade-in zoom-in-95 duration-100 text-xs">
              <div className="flex items-center justify-between pb-2 border-b border-[#333333]">
                <span className="font-semibold text-[#e6e6e6]">Notifications</span>
                <button
                  onClick={() => {
                    setShowNotifications(false);
                    navigate('/notifications');
                  }}
                  className="text-[#2383e2] hover:underline font-medium text-[11px]"
                >
                  View all
                </button>
              </div>

              <div className="divide-y divide-[#333333]/50 max-h-64 overflow-y-auto mt-1">
                {loadingNotifs ? (
                  <p className="text-center py-4 text-[#888888]">Loading...</p>
                ) : recentNotifications.length === 0 ? (
                  <p className="text-center py-4 text-[#666666]">No notifications</p>
                ) : (
                  recentNotifications.map((notif) => (
                    <div
                      key={notif.id}
                      className={`py-2 px-1.5 rounded flex items-start gap-2 transition-colors ${
                        notif.read ? 'opacity-50' : 'bg-[#2b2b2b]/50'
                      }`}
                    >
                      <Clock className="w-3.5 h-3.5 text-[#888888] mt-0.5 shrink-0" />
                      <div className="flex-1 min-w-0">
                        <p className="font-medium text-[#e6e6e6] truncate">{notif.title}</p>
                        <p className="text-[#888888] text-[11px] line-clamp-1">{notif.body}</p>
                      </div>
                      {!notif.read && (
                        <button
                          onClick={(e) => markAsRead(notif.id, e)}
                          title="Mark as read"
                          className="text-[#666666] hover:text-[#5cb87a] p-0.5"
                        >
                          <Check className="w-3 h-3" />
                        </button>
                      )}
                    </div>
                  ))
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </header>
  );
};
