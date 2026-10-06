import React, { useState, useEffect } from 'react';
import { X, AlertCircle } from 'lucide-react';
import { api } from '../api/client';
import { Project } from '../types';

interface ProjectModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
  project?: Project | null;
}

const COLOR_PRESETS = [
  '#2383e2', // Blue
  '#529e72', // Green
  '#c77d48', // Orange
  '#be5258', // Red
  '#9d68d3', // Purple
  '#ba8569', // Brown
  '#999999', // Gray
];

export const ProjectModal: React.FC<ProjectModalProps> = ({
  isOpen,
  onClose,
  onSuccess,
  project,
}) => {
  const [name, setName] = useState('');
  const [color, setColor] = useState('#2383e2');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      setError(null);
      if (project) {
        setName(project.name);
        setColor(project.color || '#2383e2');
      } else {
        setName('');
        setColor('#2383e2');
      }
    }
  }, [isOpen, project]);

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) {
      setError('Project name is required');
      return;
    }

    setLoading(true);
    setError(null);

    const payload = {
      name: name.trim(),
      color,
    };

    try {
      if (project) {
        await api.put(`/projects/${project.id}`, payload);
      } else {
        await api.post('/projects', payload);
      }
      onSuccess();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Failed to save project';
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/65 backdrop-blur-[2px] animate-in fade-in duration-100 font-sans">
      <div className="bg-[#202020] border border-[#333333] rounded-lg w-full max-w-sm shadow-2xl overflow-hidden flex flex-col text-xs">
        <div className="px-5 py-3 border-b border-[#2b2b2b] flex items-center justify-between">
          <span className="font-semibold text-[#ebebeb] text-sm">
            {project ? 'Edit Project' : 'New Project'}
          </span>
          <button
            onClick={onClose}
            className="text-[#888888] hover:text-[#ebebeb] p-1 rounded hover:bg-[#2b2b2b] transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-5 space-y-4">
          {error && (
            <div className="flex items-center gap-2 p-2.5 bg-[#3d1f22] border border-[#592b30] rounded text-[#e57373]">
              <AlertCircle className="w-4 h-4 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          <div>
            <label className="block text-[#888888] mb-1">
              Project Name
            </label>
            <input
              type="text"
              required
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. Thesis, Sprint 4, Reading"
              className="w-full bg-[#1b1b1b] border border-[#2e2e2e] focus:border-[#2383e2] rounded px-3 py-1.5 text-[#ebebeb] focus:outline-none"
            />
          </div>

          <div>
            <label className="block text-[#888888] mb-1.5">
              Color Tag
            </label>
            <div className="flex items-center gap-2">
              {COLOR_PRESETS.map((c) => (
                <button
                  type="button"
                  key={c}
                  onClick={() => setColor(c)}
                  className={`w-5 h-5 rounded-full transition-transform ${
                    color === c ? 'scale-125 ring-2 ring-white/60' : 'hover:scale-110'
                  }`}
                  style={{ backgroundColor: c }}
                />
              ))}
            </div>
          </div>

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
              {loading ? 'Saving...' : project ? 'Update' : 'Create'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
