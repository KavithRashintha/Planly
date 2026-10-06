import React from 'react';

interface PriorityBadgeProps {
  priority: number;
}

export const PriorityBadge: React.FC<PriorityBadgeProps> = ({ priority }) => {
  const configs: Record<number, { label: string; bg: string; text: string }> = {
    1: { label: 'Low', bg: 'bg-[#2b2b2b]', text: 'text-[#9b9b9b]' },
    2: { label: 'Medium', bg: 'bg-[#1b2b3a]', text: 'text-[#70a5e8]' },
    3: { label: 'High', bg: 'bg-[#3b2b1d]', text: 'text-[#e59b5f]' },
    4: { label: 'Urgent', bg: 'bg-[#3d1f22]', text: 'text-[#e57373]' },
  };

  const config = configs[priority] || configs[2];

  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-[11px] font-medium leading-tight ${config.bg} ${config.text}`}>
      {config.label}
    </span>
  );
};
