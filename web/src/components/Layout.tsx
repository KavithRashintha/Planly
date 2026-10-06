import React, { useState, useEffect, useCallback } from 'react';
import { Outlet } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { Navbar } from './Navbar';
import { api } from '../api/client';
import { TaskModal } from './TaskModal';
import { ErrorBoundary } from './ErrorBoundary';

export const Layout: React.FC = () => {
  const [unreadCount, setUnreadCount] = useState(0);
  const [isTaskModalOpen, setIsTaskModalOpen] = useState(false);

  const fetchUnreadCount = useCallback(async () => {
    try {
      const res = await api.get<{ count: number }>('/notifications/unread-count');
      setUnreadCount(res.data?.count || 0);
    } catch {
      // Ignore polling errors
    }
  }, []);

  useEffect(() => {
    fetchUnreadCount();
    // 30-second polling as specified in PLAN.md
    const interval = setInterval(fetchUnreadCount, 30000);
    return () => clearInterval(interval);
  }, [fetchUnreadCount]);

  return (
    <div className="flex h-screen bg-[#191919] text-[#e6e6e6] overflow-hidden font-sans">
      <Sidebar unreadCount={unreadCount} />
      
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        <Navbar 
          unreadCount={unreadCount} 
          onRefreshNotifications={fetchUnreadCount}
          onNewTask={() => setIsTaskModalOpen(true)}
        />
        
        <main className="flex-1 overflow-y-auto p-6 md:p-8 relative">
          <ErrorBoundary>
            <Outlet />
          </ErrorBoundary>
        </main>
      </div>

      <TaskModal 
        isOpen={isTaskModalOpen} 
        onClose={() => setIsTaskModalOpen(false)} 
        onSuccess={() => {
          setIsTaskModalOpen(false);
          window.dispatchEvent(new CustomEvent('task:created'));
        }}
      />
    </div>
  );
};
