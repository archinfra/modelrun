import React, { useMemo, useState } from 'react';
import { motion } from 'framer-motion';
import {
  CheckCircle2,
  Cpu,
  Edit3,
  FolderKanban,
  Play,
  Plus,
  Rocket,
  Search,
  Server,
  Trash2,
  X,
} from 'lucide-react';
import { useAppStore } from '../store';
import { Project } from '../types';

const projectColors = [
  'bg-blue-500',
  'bg-emerald-500',
  'bg-purple-500',
  'bg-orange-500',
  'bg-pink-500',
  'bg-cyan-500',
];

const emptyForm = {
  name: '',
  description: '',
  color: projectColors[0],
};

type ProjectForm = typeof emptyForm;

export const ProjectManager: React.FC = () => {
  const {
    projects,
    currentProjectId,
    servers,
    deployments,
    addProject,
    updateProject,
    removeProject,
    setCurrentProject,
  } = useAppStore();

  const [query, setQuery] = useState('');
  const [editingProject, setEditingProject] = useState<Project | null>(null);
  const [form, setForm] = useState<ProjectForm>(emptyForm);

  const filteredProjects = useMemo(() => {
    const keyword = query.trim().toLowerCase();
    if (!keyword) return projects;
    return projects.filter((project) =>
      `${project.name} ${project.description}`.toLowerCase().includes(keyword)
    );
  }, [projects, query]);

  const openCreate = () => {
    setEditingProject(null);
    setForm({
      ...emptyForm,
      color: projectColors[projects.length % projectColors.length],
    });
  };

  const openEdit = (project: Project) => {
    setEditingProject(project);
    setForm({
      name: project.name,
      description: project.description,
      color: project.color || projectColors[0],
    });
  };

  const saveProject = (event: React.FormEvent) => {
    event.preventDefault();
    const now = new Date().toISOString();

    if (editingProject) {
      updateProject(editingProject.id, {
        ...form,
        updatedAt: now,
      });
      setEditingProject(null);
      setForm(emptyForm);
      return;
    }

    const project: Project = {
      id: Date.now().toString(),
      name: form.name.trim() || `项目 ${projects.length + 1}`,
      description: form.description.trim(),
      color: form.color,
      createdAt: now,
      updatedAt: now,
      serverIds: [],
    };
    addProject(project);
    setCurrentProject(project.id);
    setForm({
      ...emptyForm,
      color: projectColors[(projects.length + 1) % projectColors.length],
    });
  };

  const deleteProject = (project: Project) => {
    const confirmed = window.confirm(`删除项目「${project.name}」？关联服务器和部署也会从当前工作区移除。`);
    if (!confirmed) return;
    removeProject(project.id);
  };

  const getProjectStats = (project: Project) => {
    const projectServers = servers.filter((server) => server.projectId === project.id);
    const projectDeployments = deployments.filter((deployment) =>
      projectServers.some((server) => deployment.servers.includes(server.id))
    );

    return {
      servers: projectServers.length,
      gpus: projectServers.reduce((total, server) => total + (server.gpuInfo?.length || 0), 0),
      deployments: projectDeployments.length,
      running: projectDeployments.filter((deployment) => deployment.status === 'running').length,
    };
  };

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-3">
            <FolderKanban className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-slate-900">项目管理</h1>
          </div>
          <p className="text-slate-500 mt-2">
            按项目隔离服务器、部署和资源视图，切换项目后看板和向导会跟随当前项目工作。
          </p>
        </div>
        <button
          onClick={openCreate}
          className="flex items-center gap-2 px-5 py-2.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors"
        >
          <Plus className="w-5 h-5" />
          新建项目
        </button>
      </div>

      <div className="grid grid-cols-[minmax(0,1fr)_360px] gap-6">
        <div className="space-y-4">
          <div className="relative max-w-md">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400" />
            <input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="搜索项目名称或描述"
              className="w-full pl-10 pr-4 py-2.5 bg-white border border-slate-200 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>

          {filteredProjects.length === 0 ? (
            <div className="bg-white border border-dashed border-slate-200 rounded-lg py-16 text-center">
              <FolderKanban className="w-14 h-14 text-slate-300 mx-auto mb-4" />
              <h3 className="text-lg font-semibold text-slate-900">还没有项目</h3>
              <p className="text-slate-500 mt-1">先建一个项目，再把服务器和部署放进去。</p>
              <button
                onClick={openCreate}
                className="mt-5 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors"
              >
                创建第一个项目
              </button>
            </div>
          ) : (
            <div className="grid gap-4">
              {filteredProjects.map((project) => {
                const stats = getProjectStats(project);
                const active = currentProjectId === project.id;

                return (
                  <motion.div
                    key={project.id}
                    layout
                    initial={{ opacity: 0, y: 12 }}
                    animate={{ opacity: 1, y: 0 }}
                    className={`bg-white border rounded-lg p-5 transition-colors ${
                      active ? 'border-blue-500 shadow-lg shadow-blue-500/10' : 'border-slate-200'
                    }`}
                  >
                    <div className="flex items-start justify-between gap-4">
                      <div className="flex items-start gap-4 min-w-0">
                        <div className={`w-4 h-4 rounded-full mt-1 ${project.color || 'bg-slate-400'}`} />
                        <div className="min-w-0">
                          <div className="flex items-center gap-2">
                            <h2 className="text-lg font-semibold text-slate-900 truncate">
                              {project.name}
                            </h2>
                            {active && (
                              <span className="inline-flex items-center gap-1 px-2 py-0.5 bg-blue-50 text-blue-700 rounded text-xs font-medium">
                                <CheckCircle2 className="w-3 h-3" />
                                当前项目
                              </span>
                            )}
                          </div>
                          <p className="text-sm text-slate-500 mt-1 line-clamp-2">
                            {project.description || '暂无描述'}
                          </p>
                          <p className="text-xs text-slate-400 mt-2">
                            更新于 {new Date(project.updatedAt).toLocaleString()}
                          </p>
                        </div>
                      </div>

                      <div className="flex items-center gap-2">
                        {!active && (
                          <button
                            onClick={() => setCurrentProject(project.id)}
                            className="px-3 py-1.5 bg-slate-100 hover:bg-blue-50 text-slate-700 hover:text-blue-700 rounded-lg text-sm font-medium transition-colors"
                          >
                            切换
                          </button>
                        )}
                        <button
                          onClick={() => openEdit(project)}
                          className="p-2 hover:bg-slate-100 text-slate-500 rounded-lg transition-colors"
                        >
                          <Edit3 className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => deleteProject(project)}
                          className="p-2 hover:bg-red-50 text-slate-500 hover:text-red-600 rounded-lg transition-colors"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </div>

                    <div className="grid grid-cols-4 gap-3 mt-5">
                      <ProjectStat icon={Server} label="服务器" value={stats.servers} />
                      <ProjectStat icon={Cpu} label="GPU" value={stats.gpus} />
                      <ProjectStat icon={Rocket} label="部署" value={stats.deployments} />
                      <ProjectStat icon={Play} label="运行中" value={stats.running} />
                    </div>
                  </motion.div>
                );
              })}
            </div>
          )}
        </div>

        <form onSubmit={saveProject} className="bg-white border border-slate-200 rounded-lg p-5 h-fit sticky top-24">
          <div className="flex items-center justify-between mb-5">
            <h2 className="text-lg font-semibold text-slate-900">
              {editingProject ? '编辑项目' : '新建项目'}
            </h2>
            {editingProject && (
              <button
                type="button"
                onClick={() => {
                  setEditingProject(null);
                  setForm(emptyForm);
                }}
                className="p-1.5 hover:bg-slate-100 rounded-lg text-slate-500"
              >
                <X className="w-4 h-4" />
              </button>
            )}
          </div>

          <div className="space-y-4">
            <label className="block">
              <span className="block text-sm font-medium text-slate-700 mb-1">项目名称</span>
              <input
                value={form.name}
                onChange={(event) => setForm({ ...form, name: event.target.value })}
                placeholder="例如：推理生产集群"
                className="w-full px-3 py-2 border border-slate-200 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </label>

            <label className="block">
              <span className="block text-sm font-medium text-slate-700 mb-1">项目描述</span>
              <textarea
                value={form.description}
                onChange={(event) => setForm({ ...form, description: event.target.value })}
                placeholder="这个项目的用途、环境或归属团队"
                rows={4}
                className="w-full px-3 py-2 border border-slate-200 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none"
              />
            </label>

            <div>
              <span className="block text-sm font-medium text-slate-700 mb-2">项目颜色</span>
              <div className="flex flex-wrap gap-2">
                {projectColors.map((color) => (
                  <button
                    key={color}
                    type="button"
                    onClick={() => setForm({ ...form, color })}
                    className={`w-8 h-8 rounded-lg ${color} ${
                      form.color === color ? 'ring-2 ring-offset-2 ring-slate-900' : ''
                    }`}
                    aria-label={color}
                  />
                ))}
              </div>
            </div>
          </div>

          <button
            type="submit"
            className="w-full mt-6 py-2.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors"
          >
            {editingProject ? '保存项目' : '创建项目'}
          </button>
        </form>
      </div>
    </div>
  );
};

const ProjectStat = ({
  icon: Icon,
  label,
  value,
}: {
  icon: React.ElementType;
  label: string;
  value: number;
}) => (
  <div className="bg-slate-50 rounded-lg p-3">
    <div className="flex items-center gap-2 text-xs text-slate-500">
      <Icon className="w-4 h-4" />
      {label}
    </div>
    <div className="text-xl font-semibold text-slate-900 mt-1">{value}</div>
  </div>
);
