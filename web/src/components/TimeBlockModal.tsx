import React, { useState, useEffect } from 'react';
import { X, AlertCircle, Clock, CheckSquare } from 'lucide-react';
import { api, ApiError } from '../api/client';
import { TimeBlock, Task } from '../types';

interface TimeBlockModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
  initialStart?: string;
  initialEnd?: string;
  timeBlock?: TimeBlock | null;
}

export const TimeBlockModal: React.FC<TimeBlockModalProps> = ({
  isOpen,
  onClose,
  onSuccess,
  initialStart,
  initialEnd,
  timeBlock,
}) => {
  const [title, setTitle] = useState('');
  const [taskId, setTaskId] = useState('');
  const [startsAt, setStartsAt] = useState('');
  const [endsAt, setEndsAt] = useState('');
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(false);
  const [conflictError, setConflictError] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      setConflictError(null);
      api.get<Task[]>('/tasks?status=todo&limit=50')
        .then((res) => setTasks(res.data || []))
        .catch(() => {});

      if (timeBlock) {
        setTitle(timeBlock.title);
        setTaskId(timeBlock.task_id || '');
        setStartsAt(timeBlock.starts_at.substring(0, 16));
        setEndsAt(timeBlock.ends_at.substring(0, 16));
      } else {
        setTitle('');
        setTaskId('');
        setStartsAt(initialStart ? initialStart.substring(0, 16) : '');
        setEndsAt(initialEnd ? initialEnd.substring(0, 16) : '');
      }
    }
  }, [isOpen, timeBlock, initialStart, initialEnd]);

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim()) {
      setConflictError('Title is required');
      return;
    }
    if (!startsAt || !endsAt) {
      setConflictError('Start and end time are required');
      return;
    }
    if (new Date(endsAt) <= new Date(startsAt)) {
      setConflictError('End time must be after start time');
      return;
    }

    setLoading(true);
    setConflictError(null);

    const payload = {
      title: title.trim(),
      task_id: taskId ? taskId : null,
      starts_at: new Date(startsAt).toISOString(),
      ends_at: new Date(endsAt).toISOString(),
    };

    try {
      if (timeBlock) {
        await api.put(`/time-blocks/${timeBlock.id}`, payload);
      } else {
        await api.post('/time-blocks', payload);
      }
      onSuccess();
    } catch (err: unknown) {
      if (err instanceof ApiError && err.status === 409) {
        setConflictError('Conflict: This time slot overlaps with an existing block.');
      } else {
        const msg = err instanceof Error ? err.message : 'Failed to schedule time block';
        setConflictError(msg);
      }
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!timeBlock) return;
    if (!window.confirm('Delete this scheduled block?')) return;
    setLoading(true);
    try {
      await api.delete(`/time-blocks/${timeBlock.id}`);
      onSuccess();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Failed to delete block';
      setConflictError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/65 backdrop-blur-[2px] animate-in fade-in duration-100 font-sans">
      <div className="bg-[#202020] border border-[#333333] rounded-lg w-full max-w-sm shadow-2xl overflow-hidden flex flex-col text-xs">
        <div className="px-5 py-3 border-b border-[#2b2b2b] flex items-center justify-between">
          <span className="font-semibold text-[#ebebeb] text-sm">
            {timeBlock ? 'Edit Block' : 'Schedule Focus Block'}
          </span>
          <button
            onClick={onClose}
            className="text-[#888888] hover:text-[#ebebeb] p-1 rounded hover:bg-[#2b2b2b] transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-5 space-y-3.5">
          {conflictError && (
            <div className="flex items-center gap-2 p-2.5 bg-[#3d1f22] border border-[#592b30] rounded text-[#e57373]">
              <AlertCircle className="w-3.5 h-3.5 shrink-0" />
              <span>{conflictError}</span>
            </div>
          )}

          <div>
            <label className="block text-[#888888] mb-1">
              Title
            </label>
            <input
              type="text"
              required
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="e.g. Deep Study, Reading"
              className="w-full bg-[#1b1b1b] border border-[#2e2e2e] focus:border-[#2383e2] rounded px-3 py-1.5 text-[#ebebeb] focus:outline-none"
            />
          </div>

          <div>
            <label className="block text-[#888888] mb-1">
              Link to Task
            </label>
            <select
              value={taskId}
              onChange={(e) => {
                setTaskId(e.target.value);
                const selected = tasks.find((t) => t.id === e.target.value);
                if (selected && !title) {
                  setTitle(selected.title);
                }
              }}
              className="w-full bg-[#1b1b1b] border border-[#2e2e2e] rounded px-2.5 py-1.5 text-[#ebebeb] focus:outline-none"
            >
              <option value="">None (General)</option>
              {tasks.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.title}
                </option>
              ))}
            </select>
          </div>

          <div className="grid grid-cols-2 gap-2.5">
            <div>
              <label className="block text-[#888888] mb-1">
                Start
              </label>
              <input
                type="datetime-local"
                required
                value={startsAt}
                onChange={(e) => setStartsAt(e.target.value)}
                className="w-full bg-[#1b1b1b] border border-[#2e2e2e] rounded px-2 py-1 text-[#ebebeb] focus:outline-none [color-scheme:dark]"
              />
            </div>

            <div>
              <label className="block text-[#888888] mb-1">
                End
              </label>
              <input
                type="datetime-local"
                required
                value={endsAt}
                onChange={(e) => setEndsAt(e.target.value)}
                className="w-full bg-[#1b1b1b] border border-[#2e2e2e] rounded px-2 py-1 text-[#ebebeb] focus:outline-none [color-scheme:dark]"
              />
            </div>
          </div>

          <div className="pt-3 border-t border-[#2b2b2b] flex items-center justify-between">
            {timeBlock ? (
              <button
                type="button"
                onClick={handleDelete}
                disabled={loading}
                className="text-[#e57373] hover:underline transition-colors"
              >
                Delete
              </button>
            ) : <span />}

            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={onClose}
                className="px-3 py-1.5 text-[#888888] hover:text-[#ebebeb] rounded hover:bg-[#2b2b2b] transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={loading}
                className="px-3.5 py-1.5 bg-[#2383e2] hover:bg-[#1d72c5] text-white font-medium rounded transition-colors disabled:opacity-50 cursor-pointer"
              >
                {loading ? 'Saving...' : timeBlock ? 'Update' : 'Schedule'}
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  );
};
