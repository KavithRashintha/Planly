import React, { useState } from 'react';
import { X, Target, Calendar, Sparkles } from 'lucide-react';
import { ChatResponse } from '../types';
import { agentApi } from '../api/agent';

interface PlanGoalModalProps {
  isOpen: boolean;
  conversationId?: string;
  onClose: () => void;
  onSuccess: (res: ChatResponse) => void;
}

export const PlanGoalModal: React.FC<PlanGoalModalProps> = ({
  isOpen,
  conversationId,
  onClose,
  onSuccess,
}) => {
  const [goal, setGoal] = useState('');
  const [deadline, setDeadline] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!goal.trim()) {
      setError('Please provide a goal description');
      return;
    }
    if (!deadline) {
      setError('Please select a target deadline');
      return;
    }

    setIsSubmitting(true);
    setError(null);

    try {
      // Format deadline to ISO 8601
      const deadlineIso = new Date(deadline).toISOString();
      const res = await agentApi.planGoal(goal.trim(), deadlineIso, conversationId);
      onSuccess(res);
      onClose();
      setGoal('');
      setDeadline('');
    } catch (err: any) {
      setError(err.message || 'Failed to generate goal plan');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs">
      <div 
        className="w-full max-w-md bg-[#202020] border border-[#2e2e2e] rounded-xl shadow-2xl flex flex-col overflow-hidden text-[#e6e6e6]"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between px-5 py-4 border-b border-[#2b2b2b]">
          <div className="flex items-center gap-2">
            <Target className="w-4 h-4 text-[#2383e2]" />
            <h3 className="text-sm font-semibold text-[#f0f0f0]">Plan a Goal with AI</h3>
          </div>
          <button 
            onClick={onClose}
            className="text-[#888888] hover:text-[#e6e6e6] p-1 rounded hover:bg-[#2b2b2b] transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-5 space-y-4">
          <p className="text-xs text-[#9b9b9b] leading-relaxed">
            Tell the AI planner what you want to achieve. It will break it down into actionable tasks, estimate durations, and propose a schedule respecting your capacity.
          </p>

          {error && (
            <div className="p-2.5 rounded bg-[#2b1717] border border-[#4a2222] text-xs text-[#f87171]">
              {error}
            </div>
          )}

          <div>
            <label className="block text-xs font-medium text-[#cccccc] mb-1.5">
              Goal Description
            </label>
            <textarea
              value={goal}
              onChange={(e) => setGoal(e.target.value)}
              placeholder="e.g. Complete Machine Learning Coursework 2 and submit report"
              rows={3}
              required
              className="w-full px-3 py-2 text-xs rounded-md bg-[#181818] border border-[#2e2e2e] text-[#e6e6e6] placeholder-[#555555] focus:outline-none focus:border-[#2383e2] transition-colors resize-none"
            />
          </div>

          <div>
            <label className="block text-xs font-medium text-[#cccccc] mb-1.5">
              Target Deadline
            </label>
            <div className="relative">
              <input
                type="datetime-local"
                value={deadline}
                onChange={(e) => setDeadline(e.target.value)}
                required
                className="w-full px-3 py-2 text-xs rounded-md bg-[#181818] border border-[#2e2e2e] text-[#e6e6e6] focus:outline-none focus:border-[#2383e2] transition-colors"
              />
            </div>
          </div>

          <div className="pt-2 flex items-center justify-end gap-2 border-t border-[#2a2a2a]">
            <button
              type="button"
              onClick={onClose}
              disabled={isSubmitting}
              className="px-3 py-1.5 rounded-md text-xs font-medium text-[#a0a0a0] hover:text-[#e6e6e6] hover:bg-[#2b2b2b] transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting}
              className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-md text-xs font-medium bg-[#2383e2] hover:bg-[#1d6fc2] disabled:opacity-50 text-white transition-colors cursor-pointer shadow-sm"
            >
              <Sparkles className="w-3.5 h-3.5" />
              {isSubmitting ? 'Breaking down goal...' : 'Generate Plan'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
