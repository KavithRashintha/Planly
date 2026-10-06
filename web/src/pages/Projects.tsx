import React, { useState, useEffect, useCallback } from 'react';
import { Plus, Folder, Edit2, Trash2, Check, Loader2 } from 'lucide-react';
import { api } from '../api/client';
import { Project, Task } from '../types';
import { ProjectModal } from '../components/ProjectModal';
import { Link } from 'react-router-dom';

export const Projects: React.FC = () => {
  const [projects, setProjects] = useState<Project[]>([]);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingProject, setEditingProject] = useState<Project | null>(null);

  const fetchProjectsData = useCallback(async () => {
    setLoading(true);
    try {
      const [projRes, tasksRes] = await Promise.all([
        api.get<Project[]>('/projects'),
        api.get<Task[]>('/tasks?limit=100').catch(() => ({ data: [] })),
      ]);

      const loadedTasks = Array.isArray(tasksRes.data) ? tasksRes.data : [];
      setTasks(loadedTasks);

      const projectsList = Array.isArray(projRes.data) ? projRes.data : [];
      const enhanced = projectsList.map((p) => {
        const projTasks = loadedTasks.filter((t) => t.project_id === p.id);
        const completed = projTasks.filter((t) => t.status === 'done').length;
        return {
          ...p,
          task_count: projTasks.length,
          completed_count: completed,
        };
      });

      setProjects(enhanced);
    } catch (err) {
      console.error('Failed to load projects:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchProjectsData();
  }, [fetchProjectsData]);

  const deleteProject = async (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    if (!window.confirm('Delete this project? Associated tasks will remain unassigned.')) return;
    try {
      await api.delete(`/projects/${id}`);
      fetchProjectsData();
    } catch (err) {
      console.error('Failed to delete project:', err);
    }
  };

  return (
    <div className="max-w-6xl mx-auto space-y-6 pb-16 font-sans">
      <div className="space-y-1">
        <div className="text-3xl select-none">📁</div>
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold tracking-tight text-[#ebebeb]">
            Projects
          </h1>
          <button
            onClick={() => {
              setEditingProject(null);
              setIsModalOpen(true);
            }}
            className="flex items-center gap-1.5 bg-[#2383e2] hover:bg-[#1d72c5] text-white text-xs font-medium px-3 py-1.5 rounded-md transition-colors cursor-pointer"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>New Project</span>
          </button>
        </div>
      </div>

      {loading ? (
        <div className="py-20 flex items-center justify-center">
          <Loader2 className="w-5 h-5 text-[#888888] animate-spin" />
        </div>
      ) : projects.length === 0 ? (
        <div className="notion-card p-12 text-center">
          <Folder className="w-8 h-8 text-[#555555] mx-auto mb-2" />
          <h3 className="text-xs font-semibold text-[#ebebeb]">No projects yet</h3>
          <p className="text-xs text-[#888888] mt-1 max-w-sm mx-auto">
            Group your daily work into Notion-style project workspaces.
          </p>
          <button
            onClick={() => setIsModalOpen(true)}
            className="mt-3 px-3 py-1.5 bg-[#2b2b2b] hover:bg-[#333333] text-[#ebebeb] text-xs font-medium rounded-md transition-colors"
          >
            Create Project
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {projects.map((p) => {
            const taskTotal = p.task_count || 0;
            const completed = p.completed_count || 0;
            const percentage = taskTotal > 0 ? Math.round((completed / taskTotal) * 100) : 0;

            return (
              <div
                key={p.id}
                className="notion-card p-4 flex flex-col justify-between group"
              >
                <div>
                  <div className="flex items-center justify-between mb-3">
                    <div className="flex items-center gap-2">
                      <span
                        className="w-2.5 h-2.5 rounded-full"
                        style={{ backgroundColor: p.color }}
                      />
                      <h3 className="text-sm font-semibold text-[#ebebeb]">
                        {p.name}
                      </h3>
                    </div>

                    <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                      <button
                        onClick={() => {
                          setEditingProject(p);
                          setIsModalOpen(true);
                        }}
                        className="p-1 text-[#888888] hover:text-[#ebebeb] rounded hover:bg-[#2e2e2e]"
                        title="Edit"
                      >
                        <Edit2 className="w-3 h-3" />
                      </button>
                      <button
                        onClick={(e) => deleteProject(p.id, e)}
                        className="p-1 text-[#888888] hover:text-[#e57373] rounded hover:bg-[#2e2e2e]"
                        title="Delete"
                      >
                        <Trash2 className="w-3 h-3" />
                      </button>
                    </div>
                  </div>

                  {/* Progress bar */}
                  <div className="space-y-1 mb-4">
                    <div className="flex justify-between text-[11px] text-[#888888]">
                      <span>Progress</span>
                      <span>{percentage}%</span>
                    </div>
                    <div className="w-full bg-[#2a2a2a] rounded-full h-1.5 overflow-hidden">
                      <div
                        className="h-1.5 rounded-full transition-all duration-300"
                        style={{
                          width: `${percentage}%`,
                          backgroundColor: p.color || '#2383e2',
                        }}
                      />
                    </div>
                  </div>
                </div>

                <div className="pt-3 border-t border-[#2a2a2a] flex items-center justify-between text-[11px] text-[#888888]">
                  <span>{completed}/{taskTotal} completed</span>
                  <Link
                    to={`/tasks?project_id=${p.id}`}
                    className="text-[#2383e2] hover:underline font-medium"
                  >
                    View tasks &rarr;
                  </Link>
                </div>
              </div>
            );
          })}
        </div>
      )}

      <ProjectModal
        isOpen={isModalOpen}
        project={editingProject}
        onClose={() => {
          setIsModalOpen(false);
          setEditingProject(null);
        }}
        onSuccess={() => {
          setIsModalOpen(false);
          setEditingProject(null);
          fetchProjectsData();
        }}
      />
    </div>
  );
};
