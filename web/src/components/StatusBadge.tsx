import React from 'react';
import { TaskStatus } from '../types';

export const StatusBadge: React.FC<{ status: TaskStatus }> = ({ status }) => {
  const configs: Record<TaskStatus, { label: string; bg: string; text: string }> = {
    todo: { label: 'To Do', bg: 'bg-[#2b2b2b]', text: 'text-[#9b9b9b]' },
    in_progress: { label: 'In Progress', bg: 'bg-[#1b2b3a]', text: 'text-[#70a5e8]' },
    done: { label: 'Done', bg: 'bg-[#1e3224]', text: 'text-[#62b584]' },
    cancelled: { label: 'Cancelled', bg: 'bg-[#332a28]', text: 'text-[#a88277]' },
  };

  const config = configs[status] || configs.todo;

  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-[11px] font-medium leading-tight ${config.bg} ${config.text}`}>
      {config.label}
    </span>
  );
};
