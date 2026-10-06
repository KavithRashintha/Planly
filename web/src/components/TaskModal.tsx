import React, { useState, useEffect } from 'react';
import { X, Calendar, Flag, Folder, Clock, AlertCircle } from 'lucide-react';
import { api } from '../api/client';
import { Task, Project } from '../types';

interface TaskModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
  task?: Task | null;
}

export const TaskModal: React.FC<TaskModalProps> = ({
  isOpen,
  onClose,
  onSuccess,
  task,
}) => {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [projectId, setProjectId] = useState<string>('');
  const [priority, setPriority] = useState<number>(2);
  const [status, setStatus] = useState<string>('todo');
  const [dueDate, setDueDate] = useState<string>('');
  const [estimatedMinutes, setEstimatedMinutes] = useState<number | ''>('');
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      setError(null);
      api.get<Project[]>('/projects')
        .then((res) => setProjects(res.data || []))
        .catch(() => {});

      if (task) {
        setTitle(task.title);
        setDescription(task.description || '');
        setProjectId(task.project_id || '');
        setPriority(task.priority);
        setStatus(task.status);
        setDueDate(task.due_at ? task.due_at.substring(0, 16) : task.due_date ? task.due_date.substring(0, 16) : '');
        setEstimatedMinutes(task.estimate_minutes ?? task.estimated_minutes ?? '');
      } else {
        setTitle('');
        setDescription('');
        setProjectId('');
        setPriority(2);
        setStatus('todo');
        setDueDate('');
        setEstimatedMinutes('');
      }
    }
  }, [isOpen, task]);

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim()) {
      setError('Title is required');
      return;
    }

    setLoading(true);
    setError(null);

    const dueIso = dueDate ? new Date(dueDate).toISOString() : null;
    const estVal = estimatedMinutes === '' ? null : Number(estimatedMinutes);

    const payload = {
      title: title.trim(),
      description: description.trim(),
      project_id: projectId ? projectId : null,
      priority,
      status,
      due_at: dueIso,
      estimate_minutes: estVal,
      tag_ids: [],
    };

    try {
      if (task) {
        await api.put(`/tasks/${task.id}`, payload);
      } else {
        await api.post('/tasks', payload);
      }
      onSuccess();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Failed to save task';
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/65 backdrop-blur-[2px] animate-in fade-in duration-100 font-sans">
      <div className="bg-[#202020] border border-[#333333] rounded-lg w-full max-w-lg shadow-2xl overflow-hidden flex flex-col max-h-[85vh] text-xs">
        {/* Header */}
        <div className="px-5 py-3 border-b border-[#2b2b2b] flex items-center justify-between">
          <span className="font-semibold text-[#ebebeb] text-sm">
            {task ? 'Edit Task' : 'New Task'}
          </span>
          <button
            onClick={onClose}
            className="text-[#888888] hover:text-[#ebebeb] p-1 rounded hover:bg-[#2b2b2b] transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Body */}
        <form onSubmit={handleSubmit} className="p-5 space-y-4 overflow-y-auto flex-1">
          {error && (
            <div className="flex items-center gap-2 p-2.5 bg-[#3d1f22] border border-[#592b30] rounded text-[#e57373]">
              <AlertCircle className="w-4 h-4 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          <div>
            <input
              type="text"
              required
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Task title..."
              className="w-full bg-transparent border-b border-[#333333] focus:border-[#2383e2] pb-2 text-base font-semibold text-[#ebebeb] placeholder-[#555555] focus:outline-none transition-colors"
            />
          </div>

          <div>
            <textarea
              rows={2}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Add description or notes..."
              className="w-full bg-transparent hover:bg-[#242424] focus:bg-[#242424] border border-transparent focus:border-[#383838] rounded p-2 text-xs text-[#cccccc] placeholder-[#555555] focus:outline-none resize-none transition-colors"
            />
          </div>

          {/* Properties Grid */}
          <div className="space-y-2 pt-2 border-t border-[#2b2b2b]">
            <div className="flex items-center gap-3">
              <span className="w-24 text-[#888888] flex items-center gap-1.5 shrink-0">
                <Folder className="w-3.5 h-3.5 text-[#666666]" />
                Project
              </span>
              <select
                value={projectId}
                onChange={(e) => setProjectId(e.target.value)}
                className="flex-1 bg-[#1b1b1b] border border-[#2e2e2e] rounded px-2.5 py-1 text-[#ebebeb] focus:outline-none"
              >
                <option value="">No Project</option>
                {projects.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name}
                  </option>
                ))}
              </select>
            </div>

            <div className="flex items-center gap-3">
              <span className="w-24 text-[#888888] flex items-center gap-1.5 shrink-0">
                <Flag className="w-3.5 h-3.5 text-[#666666]" />
                Priority
              </span>
              <select
                value={priority}
                onChange={(e) => setPriority(Number(e.target.value))}
                className="flex-1 bg-[#1b1b1b] border border-[#2e2e2e] rounded px-2.5 py-1 text-[#ebebeb] focus:outline-none"
              >
                <option value={1}>Low</option>
                <option value={2}>Medium</option>
                <option value={3}>High</option>
                <option value={4}>Urgent</option>
              </select>
            </div>

            <div className="flex items-center gap-3">
              <span className="w-24 text-[#888888] flex items-center gap-1.5 shrink-0">
                <Calendar className="w-3.5 h-3.5 text-[#666666]" />
                Due Date
              </span>
              <input
                type="datetime-local"
                value={dueDate}
                onChange={(e) => setDueDate(e.target.value)}
                className="flex-1 bg-[#1b1b1b] border border-[#2e2e2e] rounded px-2.5 py-1 text-[#ebebeb] focus:outline-none [color-scheme:dark]"
              />
            </div>

            <div className="flex items-center gap-3">
              <span className="w-24 text-[#888888] flex items-center gap-1.5 shrink-0">
                <Clock className="w-3.5 h-3.5 text-[#666666]" />
                Est. Min
              </span>
              <input
                type="number"
                min="0"
                step="5"
                placeholder="45"
                value={estimatedMinutes}
                onChange={(e) => setEstimatedMinutes(e.target.value ? Number(e.target.value) : '')}
                className="flex-1 bg-[#1b1b1b] border border-[#2e2e2e] rounded px-2.5 py-1 text-[#ebebeb] focus:outline-none"
              />
            </div>

            {task && (
              <div className="flex items-center gap-3">
                <span className="w-24 text-[#888888] flex items-center gap-1.5 shrink-0">
                  Status
                </span>
                <select
                  value={status}
                  onChange={(e) => setStatus(e.target.value)}
                  className="flex-1 bg-[#1b1b1b] border border-[#2e2e2e] rounded px-2.5 py-1 text-[#ebebeb] focus:outline-none capitalize"
                >
                  <option value="todo">To Do</option>
                  <option value="in_progress">In Progress</option>
                  <option value="done">Completed</option>
                  <option value="cancelled">Cancelled</option>
                </select>
              </div>
            )}
          </div>

          {/* Footer Actions */}
          <div className="pt-3 border-t border-[#2b2b2b] flex items-center justify-end gap-2">
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
              {loading ? 'Saving...' : task ? 'Update' : 'Create'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
