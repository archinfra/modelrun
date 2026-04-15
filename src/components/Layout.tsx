import React, { useState } from 'react';
import { motion } from 'framer-motion';
import { Link, useLocation } from 'react-router-dom';
import {
  LayoutDashboard,
  Database,
  Server,
  Rocket,
  List,
  FolderKanban,
  ChevronDown,
  Plus,
  Settings,
  Bell,
  Search,
  Menu,
  X,
} from 'lucide-react';
import { useAppStore } from '../store';

interface LayoutProps {
  children: React.ReactNode;
}

const menuItems = [
  { path: '/', label: '项目看板', icon: LayoutDashboard },
  { path: '/models', label: '模型仓库', icon: Database },
  { path: '/servers', label: '服务器', icon: Server },
  { path: '/wizard', label: '部署向导', icon: Rocket },
  { path: '/deployments', label: '部署列表', icon: List },
];

const projectColors = [
  'bg-blue-500',
  'bg-emerald-500',
  'bg-purple-500',
  'bg-orange-500',
  'bg-pink-500',
  'bg-cyan-500',
];

export const Layout: React.FC<LayoutProps> = ({ children }) => {
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [showProjectMenu, setShowProjectMenu] = useState(false);
  const location = useLocation();
  const { projects, currentProjectId, setCurrentProject, addProject } = useAppStore();

  const currentProject = projects.find((p) => p.id === currentProjectId);

  const handleCreateProject = () => {
    const newProject = {
      id: Date.now().toString(),
      name: `新项目 ${projects.length + 1}`,
      description: '',
      color: projectColors[projects.length % projectColors.length],
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      serverIds: [],
    };
    addProject(newProject);
    setCurrentProject(newProject.id);
    setShowProjectMenu(false);
  };

  return (
    <div className="min-h-screen bg-slate-50">
      {/* Sidebar */}
      <aside
        className={`fixed left-0 top-0 h-full bg-white border-r border-slate-200 transition-all duration-300 z-50 ${
          sidebarOpen ? 'w-72' : 'w-0 overflow-hidden'
        }`}
      >
        <div className="p-6">
          {/* Logo */}
          <div className="flex items-center gap-3 mb-8">
            <div className="w-10 h-10 bg-gradient-to-br from-blue-600 to-indigo-600 rounded-xl flex items-center justify-center shadow-lg shadow-blue-500/20">
              <Rocket className="w-5 h-5 text-white" />
            </div>
            <div>
              <h1 className="text-lg font-bold text-slate-900">ModelDeploy</h1>
              <p className="text-xs text-slate-500">AI 模型部署平台</p>
            </div>
          </div>

          {/* Project Selector */}
          <div className="mb-6">
            <label className="text-xs font-medium text-slate-500 uppercase tracking-wider mb-2 block">
              当前项目
            </label>
            <div className="relative">
              <button
                onClick={() => setShowProjectMenu(!showProjectMenu)}
                className="w-full flex items-center gap-3 px-4 py-3 bg-slate-100 hover:bg-slate-200 rounded-xl transition-colors"
              >
                <div
                  className={`w-3 h-3 rounded-full ${
                    currentProject?.color || 'bg-slate-400'
                  }`}
                />
                <span className="flex-1 text-left font-medium text-slate-700 truncate">
                  {currentProject?.name || '选择项目'}
                </span>
                <ChevronDown className="w-4 h-4 text-slate-400" />
              </button>

              {showProjectMenu && (
                <motion.div
                  initial={{ opacity: 0, y: -10 }}
                  animate={{ opacity: 1, y: 0 }}
                  className="absolute top-full left-0 right-0 mt-2 bg-white border border-slate-200 rounded-xl shadow-xl z-50"
                >
                  <div className="p-2 max-h-64 overflow-y-auto">
                    {projects.map((project) => (
                      <button
                        key={project.id}
                        onClick={() => {
                          setCurrentProject(project.id);
                          setShowProjectMenu(false);
                        }}
                        className="w-full flex items-center gap-3 px-3 py-2 hover:bg-slate-50 rounded-lg transition-colors"
                      >
                        <div className={`w-3 h-3 rounded-full ${project.color}`} />
                        <span className="text-sm text-slate-700">{project.name}</span>
                        {currentProjectId === project.id && (
                          <div className="ml-auto w-2 h-2 bg-blue-500 rounded-full" />
                        )}
                      </button>
                    ))}
                    <div className="border-t border-slate-100 mt-2 pt-2">
                      <button
                        onClick={handleCreateProject}
                        className="w-full flex items-center gap-3 px-3 py-2 text-blue-600 hover:bg-blue-50 rounded-lg transition-colors"
                      >
                        <Plus className="w-4 h-4" />
                        <span className="text-sm font-medium">创建新项目</span>
                      </button>
                    </div>
                  </div>
                </motion.div>
              )}
            </div>
          </div>

          {/* Navigation */}
          <nav className="space-y-1">
            {menuItems.map((item) => {
              const isActive = location.pathname === item.path;
              return (
                <Link
                  key={item.path}
                  to={item.path}
                  className={`flex items-center gap-3 px-4 py-3 rounded-xl transition-all ${
                    isActive
                      ? 'bg-blue-600 text-white shadow-lg shadow-blue-500/25'
                      : 'text-slate-600 hover:bg-slate-100'
                  }`}
                >
                  <item.icon className="w-5 h-5" />
                  <span className="font-medium">{item.label}</span>
                </Link>
              );
            })}
          </nav>
        </div>

        {/* Bottom Actions */}
        <div className="absolute bottom-0 left-0 right-0 p-4 border-t border-slate-200">
          <button className="w-full flex items-center gap-3 px-4 py-3 text-slate-600 hover:bg-slate-100 rounded-xl transition-colors">
            <Settings className="w-5 h-5" />
            <span className="font-medium">设置</span>
          </button>
        </div>
      </aside>

      {/* Main Content */}
      <div className={`transition-all duration-300 ${sidebarOpen ? 'ml-72' : 'ml-0'}`}>
        {/* Header */}
        <header className="sticky top-0 z-40 bg-white/80 backdrop-blur-md border-b border-slate-200">
          <div className="flex items-center justify-between px-6 py-4">
            <div className="flex items-center gap-4">
              <button
                onClick={() => setSidebarOpen(!sidebarOpen)}
                className="p-2 hover:bg-slate-100 rounded-lg transition-colors"
              >
                {sidebarOpen ? (
                  <X className="w-5 h-5 text-slate-600" />
                ) : (
                  <Menu className="w-5 h-5 text-slate-600" />
                )}
              </button>
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
                <input
                  type="text"
                  placeholder="搜索..."
                  className="pl-10 pr-4 py-2 bg-slate-100 border-0 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 w-64"
                />
              </div>
            </div>

            <div className="flex items-center gap-3">
              <button className="p-2 hover:bg-slate-100 rounded-lg transition-colors relative">
                <Bell className="w-5 h-5 text-slate-600" />
                <span className="absolute top-1 right-1 w-2 h-2 bg-red-500 rounded-full" />
              </button>
              <div className="w-9 h-9 bg-gradient-to-br from-blue-500 to-purple-500 rounded-full flex items-center justify-center text-white font-medium text-sm">
                A
              </div>
            </div>
          </div>
        </header>

        {/* Page Content */}
        <main className="p-6">
          <motion.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
          >
            {children}
          </motion.div>
        </main>
      </div>
    </div>
  );
};
