import React, { useState, useEffect, useCallback } from 'react';
import { 
  Plus, 
  Search, 
  Filter, 
  Calendar, 
  Check, 
  ChevronDown, 
  ChevronRight, 
  Trash2, 
  Edit2, 
  Folder, 
  Loader2,
  CheckSquare
} from 'lucide-react';
import { api } from '../api/client';
import { Task, Project, Subtask, TaskStatus } from '../types';
import { PriorityBadge } from '../components/PriorityBadge';
import { StatusBadge } from '../components/StatusBadge';
import { TaskModal } from '../components/TaskModal';
import { format } from 'date-fns';

export const Tasks: React.FC = () => {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [loading, setLoading] = useState(true);

  // Filters & Pagination
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [priorityFilter, setPriorityFilter] = useState<string>('all');
  const [projectFilter, setProjectFilter] = useState<string>('all');
  const [page, setPage] = useState(1);
  const limit = 20;

  // Modals & Expansion
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingTask, setEditingTask] = useState<Task | null>(null);
  const [expandedTaskId, setExpandedTaskId] = useState<string | null>(null);
  const [newSubtaskTitle, setNewSubtaskTitle] = useState('');

  const fetchTasks = useCallback(async () => {
    setLoading(true);
    try {
      const offset = (page - 1) * limit;
      let url = `/tasks?limit=${limit}&offset=${offset}`;
      if (statusFilter !== 'all') url += `&status=${statusFilter}`;
      if (priorityFilter !== 'all') url += `&priority=${priorityFilter}`;
      if (projectFilter !== 'all') url += `&project_id=${projectFilter}`;

      const [tasksRes, projectsRes] = await Promise.all([
        api.get<Task[]>(url),
        api.get<Project[]>('/projects').catch(() => ({ data: [] })),
      ]);

      const tasksList = Array.isArray(tasksRes.data) ? tasksRes.data : [];
      setTasks(tasksList);
      setTotalCount(tasksRes.totalCount ?? tasksList.length);
      setProjects(Array.isArray(projectsRes.data) ? projectsRes.data : []);
    } catch (err) {
      console.error('Failed to load tasks:', err);
    } finally {
      setLoading(false);
    }
  }, [page, statusFilter, priorityFilter, projectFilter]);

  useEffect(() => {
    fetchTasks();
  }, [fetchTasks]);

  const toggleStatus = async (task: Task, e: React.MouseEvent) => {
    e.stopPropagation();
    const nextStatus: TaskStatus = task.status === 'done' ? 'todo' : 'done';
    setTasks((prev) =>
      prev.map((t) => (t.id === task.id ? { ...t, status: nextStatus } : t))
    );
    try {
      await api.put(`/tasks/${task.id}`, { status: nextStatus });
    } catch {
      fetchTasks();
    }
  };

  const deleteTask = async (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    if (!window.confirm('Delete this task?')) return;
    try {
      await api.delete(`/tasks/${id}`);
      fetchTasks();
    } catch (err) {
      console.error('Failed to delete task:', err);
    }
  };

  const addSubtask = async (taskId: string, e: React.FormEvent) => {
    e.preventDefault();
    if (!newSubtaskTitle.trim()) return;

    try {
      const res = await api.post<Subtask>(`/tasks/${taskId}/subtasks`, {
        title: newSubtaskTitle.trim(),
      });
      setNewSubtaskTitle('');
      setTasks((prev) =>
        prev.map((t) =>
          t.id === taskId
            ? { ...t, subtasks: [...(t.subtasks || []), res.data] }
            : t
        )
      );
    } catch (err) {
      console.error('Failed to add subtask:', err);
    }
  };

  const toggleSubtask = async (taskId: string, subtask: Subtask) => {
    const nextDone = !subtask.done;
    setTasks((prev) =>
      prev.map((t) =>
        t.id === taskId
          ? {
              ...t,
              subtasks: (t.subtasks || []).map((s) =>
                s.id === subtask.id ? { ...s, done: nextDone } : s
              ),
            }
          : t
      )
    );
    try {
      await api.put(`/subtasks/${subtask.id}`, { done: nextDone });
    } catch {
      fetchTasks();
    }
  };

  const deleteSubtask = async (taskId: string, subtaskId: string) => {
    try {
      await api.delete(`/subtasks/${subtaskId}`);
      setTasks((prev) =>
        prev.map((t) =>
          t.id === taskId
            ? {
                ...t,
                subtasks: (t.subtasks || []).filter((s) => s.id !== subtaskId),
              }
            : t
        )
      );
    } catch (err) {
      console.error('Failed to delete subtask:', err);
    }
  };

  const filteredTasks = (Array.isArray(tasks) ? tasks : []).filter((t) => {
    if (!search.trim()) return true;
    const q = search.toLowerCase();
    return (
      t.title.toLowerCase().includes(q) ||
      (t.description && t.description.toLowerCase().includes(q))
    );
  });

  const totalPages = Math.ceil(totalCount / limit) || 1;

  return (
    <div className="max-w-6xl mx-auto space-y-6 pb-16 font-sans">
      {/* Header */}
      <div className="space-y-1">
        <div className="text-3xl select-none">✅</div>
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold tracking-tight text-[#ebebeb]">
            Tasks
          </h1>
          <button
            onClick={() => {
              setEditingTask(null);
              setIsModalOpen(true);
            }}
            className="flex items-center gap-1.5 bg-[#2383e2] hover:bg-[#1d72c5] text-white text-xs font-medium px-3 py-1.5 rounded-md transition-colors cursor-pointer"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>New Task</span>
          </button>
        </div>
      </div>

      {/* Notion Database Filter Bar */}
      <div className="flex flex-wrap items-center justify-between gap-3 pb-2 border-b border-[#2b2b2b]">
        <div className="flex items-center gap-2 flex-1 min-w-[200px]">
          <div className="relative w-full max-w-xs">
            <Search className="w-3.5 h-3.5 text-[#666666] absolute left-2.5 top-2.5" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search tasks..."
              className="w-full bg-transparent hover:bg-[#202020] focus:bg-[#202020] border border-transparent hover:border-[#333333] focus:border-[#333333] rounded px-2 pl-8 py-1 text-xs text-[#e6e6e6] placeholder-[#666666] focus:outline-none transition-colors"
            />
          </div>
        </div>

        <div className="flex items-center gap-2">
          <select
            value={statusFilter}
            onChange={(e) => {
              setStatusFilter(e.target.value);
              setPage(1);
            }}
            className="bg-[#202020] border border-[#2e2e2e] rounded px-2 py-1 text-[11px] text-[#999999] hover:text-[#ebebeb] focus:outline-none"
          >
            <option value="all">Status: All</option>
            <option value="todo">To Do</option>
            <option value="in_progress">In Progress</option>
            <option value="done">Done</option>
            <option value="cancelled">Cancelled</option>
          </select>

          <select
            value={priorityFilter}
            onChange={(e) => {
              setPriorityFilter(e.target.value);
              setPage(1);
            }}
            className="bg-[#202020] border border-[#2e2e2e] rounded px-2 py-1 text-[11px] text-[#999999] hover:text-[#ebebeb] focus:outline-none"
          >
            <option value="all">Priority: All</option>
            <option value="1">Low</option>
            <option value="2">Medium</option>
            <option value="3">High</option>
            <option value="4">Urgent</option>
          </select>

          <select
            value={projectFilter}
            onChange={(e) => {
              setProjectFilter(e.target.value);
              setPage(1);
            }}
            className="bg-[#202020] border border-[#2e2e2e] rounded px-2 py-1 text-[11px] text-[#999999] hover:text-[#ebebeb] focus:outline-none"
          >
            <option value="all">Project: All</option>
            {projects.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Notion List View Table */}
      <div className="notion-card overflow-hidden">
        {loading ? (
          <div className="py-20 flex items-center justify-center">
            <Loader2 className="w-5 h-5 text-[#888888] animate-spin" />
          </div>
        ) : filteredTasks.length === 0 ? (
          <div className="py-16 text-center">
            <CheckSquare className="w-8 h-8 text-[#555555] mx-auto mb-2" />
            <p className="text-xs font-medium text-[#888888]">No tasks found</p>
          </div>
        ) : (
          <div className="divide-y divide-[#2a2a2a]/60 text-xs">
            {filteredTasks.map((t) => {
              const isExpanded = expandedTaskId === t.id;
              const subtasks = t.subtasks || [];
              const completedSubtasks = subtasks.filter((s) => s.done).length;

              return (
                <div key={t.id} className="group hover:bg-[#242424] transition-colors">
                  <div className="p-3 flex items-center justify-between gap-3">
                    {/* Checkbox and title */}
                    <div className="flex items-center gap-2.5 min-w-0 flex-1">
                      <button
                        onClick={(e) => toggleStatus(t, e)}
                        className="w-4 h-4 rounded border border-[#444444] hover:border-[#666666] flex items-center justify-center transition-colors shrink-0"
                      >
                        {t.status === 'done' && <Check className="w-3 h-3 text-[#5cb87a]" />}
                      </button>

                      <div className="min-w-0 flex items-center gap-2">
                        <span
                          className={`font-medium truncate ${
                            t.status === 'done' ? 'line-through text-[#666666]' : 'text-[#ebebeb]'
                          }`}
                        >
                          {t.title}
                        </span>

                        {t.project && (
                          <span className="text-[10px] px-1.5 py-0.2 rounded bg-[#2b2b2b] text-[#999999] border border-[#383838] shrink-0">
                            {t.project.name}
                          </span>
                        )}
                      </div>
                    </div>

                    {/* Properties & Actions */}
                    <div className="flex items-center gap-2 shrink-0">
                      <PriorityBadge priority={t.priority} />
                      <StatusBadge status={t.status} />

                      {(t.due_at || t.due_date) && (
                        <span className="text-[11px] text-[#888888] font-mono flex items-center gap-1">
                          <Calendar className="w-3 h-3 text-[#666666]" />
                          {format(new Date((t.due_at || t.due_date)!), 'MMM d')}
                        </span>
                      )}

                      {/* Subtask count */}
                      <button
                        onClick={() => setExpandedTaskId(isExpanded ? null : t.id)}
                        className="text-[11px] text-[#777777] hover:text-[#cccccc] px-1.5 py-0.5 rounded hover:bg-[#2b2b2b] flex items-center gap-1 transition-colors"
                      >
                        <span>{completedSubtasks}/{subtasks.length}</span>
                        {isExpanded ? (
                          <ChevronDown className="w-3 h-3" />
                        ) : (
                          <ChevronRight className="w-3 h-3" />
                        )}
                      </button>

                      {/* Edit & Delete hover controls */}
                      <div className="opacity-0 group-hover:opacity-100 flex items-center gap-1 transition-opacity">
                        <button
                          onClick={() => {
                            setEditingTask(t);
                            setIsModalOpen(true);
                          }}
                          className="p-1 text-[#888888] hover:text-[#ebebeb] rounded hover:bg-[#333333]"
                          title="Edit"
                        >
                          <Edit2 className="w-3 h-3" />
                        </button>
                        <button
                          onClick={(e) => deleteTask(t.id, e)}
                          className="p-1 text-[#888888] hover:text-[#e57373] rounded hover:bg-[#333333]"
                          title="Delete"
                        >
                          <Trash2 className="w-3 h-3" />
                        </button>
                      </div>
                    </div>
                  </div>

                  {/* Notion Toggle List Subtasks */}
                  {isExpanded && (
                    <div className="bg-[#1c1c1c] px-9 py-2.5 border-t border-[#282828] space-y-2">
                      <div className="space-y-1">
                        {subtasks.length === 0 ? (
                          <p className="text-[11px] text-[#666666]">No subtasks yet.</p>
                        ) : (
                          subtasks.map((sub) => (
                            <div
                              key={sub.id}
                              className="flex items-center justify-between text-xs py-0.5 group/sub"
                            >
                              <div className="flex items-center gap-2">
                                <button
                                  onClick={() => toggleSubtask(t.id, sub)}
                                  className="w-3.5 h-3.5 rounded border border-[#444444] hover:border-[#666666] flex items-center justify-center shrink-0"
                                >
                                  {sub.done && <Check className="w-2.5 h-2.5 text-[#5cb87a]" />}
                                </button>
                                <span
                                  className={`${
                                    sub.done ? 'line-through text-[#666666]' : 'text-[#cccccc]'
                                  }`}
                                >
                                  {sub.title}
                                </span>
                              </div>
                              <button
                                onClick={() => deleteSubtask(t.id, sub.id)}
                                className="opacity-0 group-hover/sub:opacity-100 text-[#666666] hover:text-[#e57373] transition-opacity p-0.5"
                              >
                                <Trash2 className="w-2.5 h-2.5" />
                              </button>
                            </div>
                          ))
                        )}
                      </div>

                      {/* Add subtask */}
                      <form onSubmit={(e) => addSubtask(t.id, e)} className="flex items-center gap-1.5 pt-1">
                        <input
                          type="text"
                          value={newSubtaskTitle}
                          onChange={(e) => setNewSubtaskTitle(e.target.value)}
                          placeholder="New subtask..."
                          className="bg-transparent hover:bg-[#252525] focus:bg-[#252525] border border-transparent focus:border-[#383838] rounded px-2 py-1 text-xs text-[#ebebeb] placeholder-[#666666] focus:outline-none flex-1"
                        />
                        <button
                          type="submit"
                          className="px-2 py-1 bg-[#2b2b2b] hover:bg-[#383838] text-[#cccccc] rounded text-[11px] font-medium"
                        >
                          Add
                        </button>
                      </form>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="px-4 py-2.5 border-t border-[#2b2b2b] flex items-center justify-between text-xs text-[#888888]">
            <span>
              {totalCount} total tasks
            </span>
            <div className="flex items-center gap-1">
              <button
                disabled={page <= 1}
                onClick={() => setPage((p) => p - 1)}
                className="px-2 py-1 rounded bg-[#272727] text-[#cccccc] disabled:opacity-40"
              >
                Previous
              </button>
              <span className="px-2 text-[#ebebeb]">
                {page} / {totalPages}
              </span>
              <button
                disabled={page >= totalPages}
                onClick={() => setPage((p) => p + 1)}
                className="px-2 py-1 rounded bg-[#272727] text-[#cccccc] disabled:opacity-40"
              >
                Next
              </button>
            </div>
          </div>
        )}
      </div>

      <TaskModal
        isOpen={isModalOpen}
        task={editingTask}
        onClose={() => {
          setIsModalOpen(false);
          setEditingTask(null);
        }}
        onSuccess={() => {
          setIsModalOpen(false);
          setEditingTask(null);
          fetchTasks();
        }}
      />
    </div>
  );
};
