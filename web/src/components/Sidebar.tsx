import React from 'react';
import { NavLink, useNavigate } from 'react-router-dom';
import { 
  Home, 
  CheckSquare, 
  Folder, 
  Calendar as CalendarIcon, 
  Inbox, 
  Settings, 
  LogOut,
  ChevronDown
} from 'lucide-react';
import { useAuth } from '../context/AuthContext';

interface SidebarProps {
  unreadCount?: number;
}

export const Sidebar: React.FC<SidebarProps> = ({ unreadCount = 0 }) => {
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const navItems = [
    { to: '/dashboard', label: 'Dashboard', icon: Home },
    { to: '/tasks', label: 'Tasks', icon: CheckSquare },
    { to: '/projects', label: 'Projects', icon: Folder },
    { to: '/calendar', label: 'Calendar', icon: CalendarIcon },
    { to: '/notifications', label: 'Inbox', icon: Inbox, badge: unreadCount },
    { to: '/settings', label: 'Settings', icon: Settings },
  ];

  return (
    <aside className="w-60 bg-[#202020] border-r border-[#2b2b2b] flex flex-col h-screen select-none shrink-0 text-[#9b9b9b]">
      {/* Notion Workspace Header */}
      <div className="p-3 border-b border-[#2b2b2b]/60">
        <div className="flex items-center justify-between px-2 py-1.5 rounded-md hover:bg-[#2c2c2c] transition-colors cursor-pointer group">
          <div className="flex items-center gap-2.5 min-w-0">
            <div className="w-5 h-5 rounded bg-[#2e2e2e] border border-[#3d3d3d] flex items-center justify-center text-xs font-bold text-[#e6e6e6]">
              P
            </div>
            <div className="min-w-0">
              <p className="text-xs font-semibold text-[#e6e6e6] truncate leading-none">
                Planly Workspace
              </p>
              <p className="text-[10px] text-[#737373] truncate mt-0.5 capitalize">
                {user?.persona || 'personal'} space
              </p>
            </div>
          </div>
          <ChevronDown className="w-3.5 h-3.5 text-[#666666] group-hover:text-[#999999]" />
        </div>
      </div>

      {/* Main Nav Items */}
      <nav className="flex-1 py-3 px-2 space-y-0.5 overflow-y-auto">
        <div className="px-2.5 py-1 text-[10px] font-semibold tracking-wider text-[#666666] uppercase">
          Workspace
        </div>

        {navItems.map((item) => {
          const Icon = item.icon;
          return (
            <NavLink
              key={item.to}
              to={item.to}
              className={({ isActive }) =>
                `flex items-center justify-between px-2.5 py-1.5 rounded-md text-xs font-medium transition-colors ${
                  isActive
                    ? 'bg-[#2b2b2b] text-[#ebebeb] font-semibold'
                    : 'text-[#9b9b9b] hover:text-[#ebebeb] hover:bg-[#272727]'
                }`
              }
            >
              <div className="flex items-center gap-2.5">
                <Icon className="w-4 h-4 shrink-0 text-[#888888]" />
                <span>{item.label}</span>
              </div>
              {item.badge && item.badge > 0 ? (
                <span className="px-1.5 py-0.2 rounded text-[10px] font-medium bg-[#333333] text-[#cccccc]">
                  {item.badge > 99 ? '99+' : item.badge}
                </span>
              ) : null}
            </NavLink>
          );
        })}
      </nav>

      {/* User Row Footer */}
      <div className="p-2 border-t border-[#2b2b2b]/60">
        <div className="flex items-center justify-between px-2 py-1.5 rounded-md hover:bg-[#272727] transition-colors">
          <div className="flex items-center gap-2.5 min-w-0">
            <div className="w-5 h-5 rounded-full bg-[#333333] flex items-center justify-center font-bold text-[#e6e6e6] text-[10px]">
              {user?.full_name ? user.full_name.charAt(0).toUpperCase() : 'U'}
            </div>
            <p className="text-xs font-medium text-[#cccccc] truncate">
              {user?.full_name || 'User'}
            </p>
          </div>
          <button
            onClick={handleLogout}
            title="Log out"
            className="text-[#666666] hover:text-[#e57373] p-1 rounded hover:bg-[#333333] transition-colors"
          >
            <LogOut className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>
    </aside>
  );
};
