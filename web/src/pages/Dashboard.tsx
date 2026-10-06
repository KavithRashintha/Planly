import React, { useState, useEffect, useCallback } from 'react';
import { 
  Check, 
  Calendar, 
  Clock, 
  Plus, 
  Loader2,
  ListTodo,
  TrendingUp,
  AlertCircle
} from 'lucide-react';
import { api } from '../api/client';
import { Task, WorkloadDay, WorkloadResponse, TimeBlock } from '../types';
import { PriorityBadge } from '../components/PriorityBadge';
import { StatusBadge } from '../components/StatusBadge';
import { TaskModal } from '../components/TaskModal';
import { ResponsiveContainer, BarChart, Bar, XAxis, YAxis, Tooltip, CartesianGrid } from 'recharts';
import { format, addDays } from 'date-fns';

export const Dashboard: React.FC = () => {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [workload, setWorkload] = useState<WorkloadDay[]>([]);
  const [todayBlocks, setTodayBlocks] = useState<TimeBlock[]>([]);
  const [loading, setLoading] = useState(true);
  const [isTaskModalOpen, setIsTaskModalOpen] = useState(false);
  const [selectedTask, setSelectedTask] = useState<Task | null>(null);

  const fetchDashboardData = useCallback(async () => {
    try {
      setLoading(true);
      const todayStr = format(new Date(), 'yyyy-MM-dd');
      const nextWeekStr = format(addDays(new Date(), 7), 'yyyy-MM-dd');

      const [tasksRes, workloadRes, blocksRes] = await Promise.all([
        api.get<Task[]>('/tasks?limit=50').catch(() => ({ data: [] })),
        api.get<WorkloadResponse | WorkloadDay[]>(`/workload?from=${todayStr}&to=${nextWeekStr}`).catch(() => ({ data: [] })),
        api.get<TimeBlock[]>(`/time-blocks?from=${todayStr}T00:00:00Z&to=${todayStr}T23:59:59Z`).catch(() => ({ data: [] })),
      ]);

      const tasksList = Array.isArray(tasksRes.data) ? tasksRes.data : [];
      setTasks(tasksList);

      let daysList: WorkloadDay[] = [];
      if (Array.isArray(workloadRes.data)) {
        daysList = workloadRes.data;
      } else if (workloadRes.data && typeof workloadRes.data === 'object' && 'days' in workloadRes.data && Array.isArray((workloadRes.data as WorkloadResponse).days)) {
        daysList = (workloadRes.data as WorkloadResponse).days;
      }
      setWorkload(daysList);

      const blocksList = Array.isArray(blocksRes.data) ? blocksRes.data : [];
      setTodayBlocks(blocksList);
    } catch (err) {
      console.error('Error fetching dashboard:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchDashboardData();
    const handleCreated = () => fetchDashboardData();
    window.addEventListener('task:created', handleCreated);
    return () => window.removeEventListener('task:created', handleCreated);
  }, [fetchDashboardData]);

  // Derived metrics
  const now = new Date();
  const todayDateStr = format(now, 'yyyy-MM-dd');

  const overdueTasks = tasks.filter((t) => {
    const due = t.due_at || t.due_date;
    if (t.status === 'done' || t.status === 'cancelled' || !due) return false;
    return new Date(due) < now;
  });

  const todayTasks = tasks.filter((t) => {
    const due = t.due_at || t.due_date;
    if (t.status === 'done' || !due) return false;
    return format(new Date(due), 'yyyy-MM-dd') === todayDateStr;
  });

  const completedCount = tasks.filter((t) => t.status === 'done').length;

  const toggleTaskStatus = async (task: Task, e: React.MouseEvent) => {
    e.stopPropagation();
    const nextStatus = task.status === 'done' ? 'todo' : 'done';
    setTasks((prev) =>
      prev.map((t) => (t.id === task.id ? { ...t, status: nextStatus } : t))
    );
    try {
      await api.put(`/tasks/${task.id}`, { status: nextStatus });
      fetchDashboardData();
    } catch {
      fetchDashboardData();
    }
  };

  if (loading) {
    return (
      <div className="h-64 flex items-center justify-center">
        <Loader2 className="w-5 h-5 text-[#888888] animate-spin" />
      </div>
    );
  }

  // Chart data
  const chartData = (Array.isArray(workload) ? workload : []).map((w) => ({
    name: format(new Date(w.date), 'EEE'),
    hours: Math.round((((w.planned_minutes ?? w.total_minutes) || 0) / 60) * 10) / 10,
    tasks: w.task_count || 0,
  }));

  return (
    <div className="max-w-5xl mx-auto space-y-8 pb-16 font-sans">
      {/* Notion Page Title & Icon */}
      <div className="space-y-2">
        <div className="text-3xl select-none">📋</div>
        <h1 className="text-2xl font-bold tracking-tight text-[#ebebeb]">
          Workspace Overview
        </h1>
        <p className="text-xs text-[#8c8c8c]">
          Planly daily schedule, capacity workload, and priority tasks
        </p>
      </div>

      {/* Notion Callout Metric Widgets */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <div className="notion-card p-3.5 flex flex-col justify-between">
          <span className="text-[11px] font-medium text-[#888888] uppercase tracking-wider">Total Tasks</span>
          <div className="mt-2 flex items-baseline gap-1.5">
            <span className="text-2xl font-bold text-[#ebebeb]">{tasks.length}</span>
            <span className="text-[11px] text-[#666666]">items</span>
          </div>
        </div>

        <div className="notion-card p-3.5 flex flex-col justify-between">
          <span className="text-[11px] font-medium text-[#70a5e8] uppercase tracking-wider">Due Today</span>
          <div className="mt-2 flex items-baseline gap-1.5">
            <span className="text-2xl font-bold text-[#70a5e8]">{todayTasks.length}</span>
            <span className="text-[11px] text-[#666666]">scheduled</span>
          </div>
        </div>

        <div className="notion-card p-3.5 flex flex-col justify-between">
          <span className="text-[11px] font-medium text-[#e57373] uppercase tracking-wider">Overdue</span>
          <div className="mt-2 flex items-baseline gap-1.5">
            <span className="text-2xl font-bold text-[#e57373]">{overdueTasks.length}</span>
            <span className="text-[11px] text-[#666666]">attention</span>
          </div>
        </div>

        <div className="notion-card p-3.5 flex flex-col justify-between">
          <span className="text-[11px] font-medium text-[#62b584] uppercase tracking-wider">Completed</span>
          <div className="mt-2 flex items-baseline gap-1.5">
            <span className="text-2xl font-bold text-[#62b584]">{completedCount}</span>
            <span className="text-[11px] text-[#666666]">done</span>
          </div>
        </div>
      </div>

      {/* Main Split: Workload & Today's Schedule */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* Workload Bar Chart */}
        <div className="md:col-span-2 notion-card p-4 flex flex-col">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-xs font-semibold text-[#ebebeb]">Upcoming 7-Day Capacity</h2>
              <p className="text-[11px] text-[#888888]">Focus hours scheduled</p>
            </div>
            <span className="text-[10px] font-medium text-[#888888] bg-[#272727] px-2 py-0.5 rounded border border-[#333333]">
              Hours
            </span>
          </div>

          <div className="h-52 w-full">
            {chartData.length === 0 ? (
              <div className="h-full flex items-center justify-center text-xs text-[#666666]">
                No workload data for the coming week
              </div>
            ) : (
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={chartData} margin={{ top: 10, right: 10, left: -25, bottom: 0 }}>
                  <CartesianGrid strokeDasharray="2 2" stroke="#2a2a2a" vertical={false} />
                  <XAxis dataKey="name" stroke="#666666" fontSize={10} tickLine={false} />
                  <YAxis stroke="#666666" fontSize={10} tickLine={false} />
                  <Tooltip
                    contentStyle={{ backgroundColor: '#202020', borderColor: '#333333', borderRadius: '6px' }}
                    itemStyle={{ color: '#d4d4d4', fontSize: '11px' }}
                    labelStyle={{ color: '#ebebeb', fontWeight: 'bold', fontSize: '11px' }}
                  />
                  <Bar dataKey="hours" name="Workload (Hours)" fill="#3b82f6" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            )}
          </div>
        </div>

        {/* Today Focus Schedule */}
        <div className="notion-card p-4 flex flex-col">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-xs font-semibold text-[#ebebeb]">Today's Focus</h2>
            <span className="text-[11px] text-[#888888]">{todayBlocks.length} blocks</span>
          </div>

          <div className="space-y-2 flex-1 overflow-y-auto max-h-52">
            {todayBlocks.length === 0 ? (
              <p className="text-xs text-[#666666] text-center py-10">
                No focus blocks scheduled for today.
              </p>
            ) : (
              todayBlocks.map((block) => (
                <div
                  key={block.id}
                  className="p-2.5 bg-[#252525] border border-[#2f2f2f] rounded-md flex items-center justify-between gap-2"
                >
                  <div className="min-w-0">
                    <p className="text-xs font-medium text-[#ebebeb] truncate">{block.title}</p>
                    <p className="text-[11px] text-[#888888] mt-0.5 flex items-center gap-1">
                      <Clock className="w-3 h-3 text-[#666666]" />
                      {format(new Date(block.starts_at), 'hh:mm a')} - {format(new Date(block.ends_at), 'hh:mm a')}
                    </p>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      </div>

      {/* Task Sections: Overdue & Today */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Overdue Tasks */}
        <div className="notion-card p-4">
          <div className="flex items-center justify-between pb-2 mb-2 border-b border-[#2e2e2e]">
            <div className="flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-[#e57373]" />
              <h3 className="text-xs font-semibold text-[#ebebeb]">Overdue</h3>
            </div>
            <span className="text-[11px] font-mono text-[#888888]">{overdueTasks.length}</span>
          </div>

          <div className="divide-y divide-[#2a2a2a]/60">
            {overdueTasks.length === 0 ? (
              <p className="text-xs text-[#666666] py-6 text-center">No overdue tasks.</p>
            ) : (
              overdueTasks.map((t) => (
                <div
                  key={t.id}
                  onClick={() => {
                    setSelectedTask(t);
                    setIsTaskModalOpen(true);
                  }}
                  className="py-2 px-1 flex items-center justify-between gap-2 hover:bg-[#272727] rounded transition-colors cursor-pointer"
                >
                  <div className="flex items-center gap-2.5 min-w-0">
                    <button
                      onClick={(e) => toggleTaskStatus(t, e)}
                      className="w-4 h-4 rounded border border-[#444444] hover:border-[#666666] flex items-center justify-center transition-colors shrink-0"
                    >
                      {t.status === 'done' && <Check className="w-3 h-3 text-[#5cb87a]" />}
                    </button>
                    <span className="text-xs text-[#cccccc] truncate font-medium">
                      {t.title}
                    </span>
                  </div>
                  <div className="flex items-center gap-1.5 shrink-0">
                    <PriorityBadge priority={t.priority} />
                    <span className="text-[10px] text-[#e57373] font-mono">
                      {t.due_at || t.due_date ? format(new Date((t.due_at || t.due_date)!), 'MMM d') : ''}
                    </span>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Due Today */}
        <div className="notion-card p-4">
          <div className="flex items-center justify-between pb-2 mb-2 border-b border-[#2e2e2e]">
            <div className="flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-[#70a5e8]" />
              <h3 className="text-xs font-semibold text-[#ebebeb]">Today's List</h3>
            </div>
            <span className="text-[11px] font-mono text-[#888888]">{todayTasks.length}</span>
          </div>

          <div className="divide-y divide-[#2a2a2a]/60">
            {todayTasks.length === 0 ? (
              <p className="text-xs text-[#666666] py-6 text-center">No tasks scheduled for today.</p>
            ) : (
              todayTasks.map((t) => (
                <div
                  key={t.id}
                  onClick={() => {
                    setSelectedTask(t);
                    setIsTaskModalOpen(true);
                  }}
                  className="py-2 px-1 flex items-center justify-between gap-2 hover:bg-[#272727] rounded transition-colors cursor-pointer"
                >
                  <div className="flex items-center gap-2.5 min-w-0">
                    <button
                      onClick={(e) => toggleTaskStatus(t, e)}
                      className="w-4 h-4 rounded border border-[#444444] hover:border-[#666666] flex items-center justify-center transition-colors shrink-0"
                    >
                      {t.status === 'done' && <Check className="w-3 h-3 text-[#5cb87a]" />}
                    </button>
                    <span className="text-xs text-[#cccccc] truncate font-medium">
                      {t.title}
                    </span>
                  </div>
                  <div className="flex items-center gap-1.5 shrink-0">
                    <PriorityBadge priority={t.priority} />
                    <StatusBadge status={t.status} />
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      </div>

      <TaskModal
        isOpen={isTaskModalOpen}
        task={selectedTask}
        onClose={() => {
          setIsTaskModalOpen(false);
          setSelectedTask(null);
        }}
        onSuccess={() => {
          setIsTaskModalOpen(false);
          setSelectedTask(null);
          fetchDashboardData();
        }}
      />
    </div>
  );
};
