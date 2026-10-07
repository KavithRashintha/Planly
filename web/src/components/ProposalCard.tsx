import React, { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { 
  CheckCircle2, 
  XCircle, 
  Clock, 
  AlertCircle, 
  Check, 
  X, 
  Calendar, 
  Plus, 
  Edit3, 
  Trash2,
  CalendarClock
} from 'lucide-react';
import { Proposal, ProposalAction } from '../types';
import { agentApi } from '../api/agent';

interface ProposalCardProps {
  proposal: Proposal;
  onDecided?: (updated: Proposal) => void;
}

export const ProposalCard: React.FC<ProposalCardProps> = ({ proposal, onDecided }) => {
  const queryClient = useQueryClient();
  const [currentProposal, setCurrentProposal] = useState<Proposal>(proposal);
  const [isApproving, setIsApproving] = useState(false);
  const [isRejecting, setIsRejecting] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const handleApprove = async () => {
    setIsApproving(true);
    setErrorMsg(null);
    try {
      const updated = await agentApi.approveProposal(currentProposal.id);
      setCurrentProposal(updated);
      onDecided?.(updated);
      
      // Invalidate relevant caches so dashboard, tasks, calendar update immediately
      queryClient.invalidateQueries({ queryKey: ['tasks'] });
      queryClient.invalidateQueries({ queryKey: ['dashboard'] });
      queryClient.invalidateQueries({ queryKey: ['workload'] });
      queryClient.invalidateQueries({ queryKey: ['time-blocks'] });
      queryClient.invalidateQueries({ queryKey: ['proposals'] });
      
      window.dispatchEvent(new CustomEvent('task:created'));
    } catch (err: any) {
      setErrorMsg(err.message || 'Failed to approve proposal');
    } finally {
      setIsApproving(false);
    }
  };

  const handleReject = async () => {
    setIsRejecting(true);
    setErrorMsg(null);
    try {
      const updated = await agentApi.rejectProposal(currentProposal.id);
      setCurrentProposal(updated);
      onDecided?.(updated);
      queryClient.invalidateQueries({ queryKey: ['proposals'] });
    } catch (err: any) {
      setErrorMsg(err.message || 'Failed to reject proposal');
    } finally {
      setIsRejecting(false);
    }
  };

  const getStatusBadge = () => {
    switch (currentProposal.status) {
      case 'approved':
        return (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium bg-[#14291e] text-[#4ade80] border border-[#1f422f]">
            <CheckCircle2 className="w-3 h-3" /> Approved
          </span>
        );
      case 'rejected':
        return (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium bg-[#291414] text-[#f87171] border border-[#421f1f]">
            <XCircle className="w-3 h-3" /> Rejected
          </span>
        );
      case 'partial':
        return (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium bg-[#132438] text-[#60a5fa] border border-[#1d395a]">
            <AlertCircle className="w-3 h-3" /> Partially Applied
          </span>
        );
      case 'failed':
        return (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium bg-[#291414] text-[#f87171] border border-[#421f1f]">
            <XCircle className="w-3 h-3" /> Execution Failed
          </span>
        );
      default:
        return (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium bg-[#2d2411] text-[#fbbf24] border border-[#4d3c19]">
            <Clock className="w-3 h-3" /> Awaiting Approval
          </span>
        );
    }
  };

  const renderActionItem = (act: ProposalAction, idx: number) => {
    let icon = <Plus className="w-3.5 h-3.5 text-[#2383e2]" />;
    let title = 'Action';
    let details: React.ReactNode = null;

    if (act.tool === 'create_task') {
      icon = <Plus className="w-3.5 h-3.5 text-[#2383e2]" />;
      title = `Create Task: ${act.input?.title || 'Untitled'}`;
      details = (
        <div className="flex flex-wrap items-center gap-2 mt-1 text-[11px] text-[#888888]">
          {act.input?.priority && (
            <span className="px-1.5 py-0.5 rounded bg-[#2b2b2b] text-[#cccccc]">
              Priority {act.input.priority}
            </span>
          )}
          {act.input?.due_at && (
            <span className="inline-flex items-center gap-1 text-[#aaaaaa]">
              <Calendar className="w-3 h-3" />
              {new Date(act.input.due_at).toLocaleDateString(undefined, {
                month: 'short',
                day: 'numeric',
                hour: '2-digit',
                minute: '2-digit',
              })}
            </span>
          )}
          {act.input?.estimate_minutes && (
            <span className="text-[#888888]">~{act.input.estimate_minutes}m</span>
          )}
        </div>
      );
    } else if (act.tool === 'update_task') {
      icon = <Edit3 className="w-3.5 h-3.5 text-[#fbbf24]" />;
      title = `Update Task: ${act.input?.title || act.input?.task_id || 'Task'}`;
      details = (
        <div className="text-[11px] text-[#888888] mt-0.5">
          {act.input?.due_at && (
            <span>New due date: {new Date(act.input.due_at).toLocaleDateString()}</span>
          )}
          {act.input?.status && <span className="ml-2">Status: {act.input.status}</span>}
        </div>
      );
    } else if (act.tool === 'delete_task') {
      icon = <Trash2 className="w-3.5 h-3.5 text-[#f87171]" />;
      title = `Delete Task: ${act.input?.task_id || ''}`;
    } else if (act.tool === 'create_time_block') {
      icon = <CalendarClock className="w-3.5 h-3.5 text-[#a78bfa]" />;
      title = `Schedule Block: ${act.input?.title || 'Block'}`;
      details = (
        <div className="text-[11px] text-[#888888] mt-0.5">
          {act.input?.starts_at && act.input?.ends_at && (
            <span>
              {new Date(act.input.starts_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })} -{' '}
              {new Date(act.input.ends_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
            </span>
          )}
        </div>
      );
    } else if (act.tool === 'reschedule_tasks') {
      icon = <CalendarClock className="w-3.5 h-3.5 text-[#38bdf8]" />;
      const count = act.input?.reschedules?.length || 0;
      title = `Reschedule ${count} task(s)`;
      details = (
        <div className="space-y-1 mt-1 text-[11px] text-[#888888]">
          {act.input?.reschedules?.map((r: any, rIdx: number) => (
            <div key={rIdx} className="flex items-center justify-between">
              <span className="truncate max-w-[180px] text-[#cccccc]">{r.task_id}</span>
              <span className="text-[#a0a0a0]">
                → {new Date(r.new_due_at).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}
              </span>
            </div>
          ))}
        </div>
      );
    }

    return (
      <div 
        key={act.id || idx}
        className="p-2.5 rounded-md bg-[#232323] border border-[#2e2e2e] text-xs transition-colors"
      >
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-2 min-w-0">
            <span className="shrink-0">{icon}</span>
            <span className="font-medium text-[#e6e6e6] truncate">{title}</span>
          </div>
          {act.status && act.status !== 'pending' && (
            <span className={`text-[10px] font-semibold uppercase px-1.5 py-0.5 rounded ${
              act.status === 'applied' ? 'bg-[#14291e] text-[#4ade80]' :
              act.status === 'failed' ? 'bg-[#291414] text-[#f87171]' :
              'bg-[#222222] text-[#888888]'
            }`}>
              {act.status}
            </span>
          )}
        </div>
        {details}
        {act.error && (
          <p className="mt-1 text-[11px] text-[#f87171]">{act.error}</p>
        )}
      </div>
    );
  };

  return (
    <div className="mt-3 p-3.5 rounded-lg bg-[#1c1c1c] border border-[#2e2e2e] shadow-sm max-w-xl">
      <div className="flex items-center justify-between gap-2 mb-2.5">
        <div className="flex items-center gap-2">
          <span className="text-xs font-semibold text-[#e6e6e6]">
            Plan Proposal
          </span>
          {getStatusBadge()}
        </div>
        <span className="text-[10px] text-[#666666]">
          {new Date(currentProposal.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
        </span>
      </div>

      <p className="text-xs text-[#a0a0a0] mb-3 leading-relaxed">
        {currentProposal.summary}
      </p>

      {/* Staged Actions List */}
      <div className="space-y-1.5 mb-3">
        {currentProposal.actions.map((act, idx) => renderActionItem(act, idx))}
      </div>

      {errorMsg && (
        <div className="mb-2.5 p-2 rounded bg-[#2e1717] border border-[#4a2222] text-[11px] text-[#f87171]">
          {errorMsg}
        </div>
      )}

      {/* Decision Buttons for pending proposals */}
      {currentProposal.status === 'pending' && (
        <div className="flex items-center gap-2 pt-2 border-t border-[#2a2a2a]">
          <button
            onClick={handleApprove}
            disabled={isApproving || isRejecting}
            className="flex-1 inline-flex items-center justify-center gap-1.5 py-1.5 px-3 rounded-md text-xs font-medium bg-[#2383e2] hover:bg-[#1d6fc2] disabled:opacity-50 text-white transition-colors cursor-pointer"
          >
            <Check className="w-3.5 h-3.5" />
            {isApproving ? 'Applying Changes...' : 'Approve & Apply'}
          </button>
          <button
            onClick={handleReject}
            disabled={isApproving || isRejecting}
            className="inline-flex items-center justify-center gap-1.5 py-1.5 px-3 rounded-md text-xs font-medium bg-[#262626] hover:bg-[#303030] hover:text-[#e57373] disabled:opacity-50 text-[#a0a0a0] border border-[#333333] transition-colors cursor-pointer"
          >
            <X className="w-3.5 h-3.5" />
            {isRejecting ? 'Rejecting...' : 'Reject'}
          </button>
        </div>
      )}
    </div>
  );
};
