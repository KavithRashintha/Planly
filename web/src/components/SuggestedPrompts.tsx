import React from 'react';
import { Sparkles, CalendarClock, Target, Compass } from 'lucide-react';
import { Persona } from '../types';

interface SuggestedPromptsProps {
  persona?: Persona;
  onSelectPrompt: (prompt: string) => void;
}

export const SuggestedPrompts: React.FC<SuggestedPromptsProps> = ({
  persona = 'employee',
  onSelectPrompt,
}) => {
  const getPrompts = () => {
    switch (persona) {
      case 'student':
        return [
          {
            icon: Target,
            title: 'Plan Exam Revision',
            prompt: 'Help me plan a structured revision schedule for my upcoming exam subjects.',
          },
          {
            icon: CalendarClock,
            title: 'Reschedule Overdue Work',
            prompt: 'Check all my overdue homework assignments and propose new due dates.',
          },
          {
            icon: Compass,
            title: 'Analyze Today’s Focus',
            prompt: 'What tasks should I focus on today to stay on top of my study schedule?',
          },
        ];
      case 'undergraduate':
        return [
          {
            icon: Target,
            title: 'Coursework Breakdown',
            prompt: 'Break down my software engineering group project into deliverables and tasks.',
          },
          {
            icon: CalendarClock,
            title: 'Balance Lecture & Study',
            prompt: 'Schedule deep work blocks this week around my lecture timetable and deadlines.',
          },
          {
            icon: Compass,
            title: 'Workload Health Check',
            prompt: 'Analyze my workload for the next 7 days and flag days where I am over capacity.',
          },
        ];
      default: // employee
        return [
          {
            icon: Target,
            title: 'Plan Sprint Deliverables',
            prompt: 'Help me break down our upcoming product release goals into actionable tasks.',
          },
          {
            icon: CalendarClock,
            title: 'Reschedule Overdue Tasks',
            prompt: 'Scan all overdue tasks and propose a realistic reschedule respecting my daily capacity.',
          },
          {
            icon: Compass,
            title: 'Daily Priority Briefing',
            prompt: 'Give me a quick priority assessment of my tasks for today and suggest deep work slots.',
          },
        ];
    }
  };

  const prompts = getPrompts();

  return (
    <div className="w-full max-w-2xl mx-auto my-6 space-y-3">
      <div className="flex items-center gap-2 text-xs font-medium text-[#737373]">
        <Sparkles className="w-3.5 h-3.5 text-[#2383e2]" />
        <span>Suggested for {persona} space</span>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-2.5">
        {prompts.map((item, idx) => {
          const Icon = item.icon;
          return (
            <button
              key={idx}
              onClick={() => onSelectPrompt(item.prompt)}
              className="p-3 text-left rounded-lg bg-[#202020] hover:bg-[#262626] border border-[#2b2b2b] hover:border-[#383838] transition-all group cursor-pointer shadow-xs"
            >
              <div className="flex items-center gap-2 mb-1.5">
                <Icon className="w-3.5 h-3.5 text-[#888888] group-hover:text-[#2383e2] transition-colors" />
                <span className="text-xs font-semibold text-[#e6e6e6] truncate">
                  {item.title}
                </span>
              </div>
              <p className="text-[11px] text-[#888888] group-hover:text-[#a0a0a0] line-clamp-2 leading-relaxed">
                {item.prompt}
              </p>
            </button>
          );
        })}
      </div>
    </div>
  );
};
