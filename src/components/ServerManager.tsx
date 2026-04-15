import React, { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import {
  Server,
  Plus,
  Trash2,
  Edit3,
  CheckCircle2,
  XCircle,
  Cpu,
  HardDrive,
  Thermometer,
  Zap,
  Activity,
  ChevronDown,
  ChevronRight,
  Search,
  Filter,
} from 'lucide-react';
import { useAppStore } from '../store';
import { ServerConfig, GPUInfo } from '../types';

const mockGPU: GPUInfo[] = [
  {
    index: 0,
    name: 'NVIDIA A100 80GB',
    memoryTotal: 81920,
    memoryUsed: 24576,
    memoryFree: 57344,
    utilization: 45,
    temperature: 72,
    powerDraw: 285,
    powerLimit: 400,
  },
  {
    index: 1,
    name: 'NVIDIA A100 80GB',
    memoryTotal: 81920,
    memoryUsed: 18432,
    memoryFree: 63488,
    utilization: 32,
    temperature: 68,
    powerDraw: 240,
    powerLimit: 400,
  },
];

const ServerCard: React.FC<{
  server: ServerConfig;
  onEdit: (server: ServerConfig) => void;
  onDelete: (id: string) => void;
}> = ({ server, onEdit, onDelete }) => {
  const [expanded, setExpanded] = useState(false);
  const gpuInfo = server.gpuInfo || mockGPU;

  return (
    <motion.div
      layout
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className="bg-white rounded-2xl border border-slate-200 overflow-hidden hover:shadow-lg transition-shadow"
    >
      <div className="p-5">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-4">
            <div
              className={`w-12 h-12 rounded-xl flex items-center justify-center ${
                server.status === 'online'
                  ? 'bg-emerald-100 text-emerald-600'
                  : 'bg-red-100 text-red-600'
              }`}
            >
              <Server className="w-6 h-6" />
            </div>
            <div>
              <h3 className="font-semibold text-slate-900 text-lg">{server.name}</h3>
              <div className="flex items-center gap-2 mt-1">
                <span className="text-sm text-slate-500">{server.host}</span>
                <span className="text-slate-300">·</span>
                <span className="text-sm text-slate-500">SSH {server.sshPort}</span>
                <span className="text-slate-300">·</span>
                <span
                  className={`px-2 py-0.5 rounded-full text-xs font-medium ${
                    server.status === 'online'
                      ? 'bg-emerald-100 text-emerald-700'
                      : 'bg-red-100 text-red-700'
                  }`}
                >
                  {server.status === 'online' ? '在线' : '离线'}
                </span>
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setExpanded(!expanded)}
              className="p-2 hover:bg-slate-100 rounded-lg transition-colors"
            >
              {expanded ? (
                <ChevronDown className="w-5 h-5 text-slate-400" />
              ) : (
                <ChevronRight className="w-5 h-5 text-slate-400" />
              )}
            </button>
            <button
              onClick={() => onEdit(server)}
              className="p-2 hover:bg-blue-50 text-slate-400 hover:text-blue-600 rounded-lg transition-colors"
            >
              <Edit3 className="w-5 h-5" />
            </button>
            <button
              onClick={() => onDelete(server.id)}
              className="p-2 hover:bg-red-50 text-slate-400 hover:text-red-600 rounded-lg transition-colors"
            >
              <Trash2 className="w-5 h-5" />
            </button>
          </div>
        </div>

        <div className="grid grid-cols-4 gap-4 mt-5">
          <div className="bg-slate-50 rounded-xl p-3">
            <div className="flex items-center gap-2 text-slate-500 text-sm mb-1">
              <Cpu className="w-4 h-4" />
              <span>GPU</span>
            </div>
            <div className="text-lg font-semibold text-slate-900">
              {gpuInfo.length} 个
            </div>
          </div>
          <div className="bg-slate-50 rounded-xl p-3">
            <div className="flex items-center gap-2 text-slate-500 text-sm mb-1">
              <HardDrive className="w-4 h-4" />
              <span>显存</span>
            </div>
            <div className="text-lg font-semibold text-slate-900">
              {(gpuInfo.reduce((acc, g) => acc + g.memoryTotal, 0) / 1024).toFixed(0)} GB
            </div>
          </div>
          <div className="bg-slate-50 rounded-xl p-3">
            <div className="flex items-center gap-2 text-slate-500 text-sm mb-1">
              <Thermometer className="w-4 h-4" />
              <span>温度</span>
            </div>
            <div className="text-lg font-semibold text-slate-900">
              {Math.round(gpuInfo.reduce((acc, g) => acc + g.temperature, 0) / gpuInfo.length)}°C
            </div>
          </div>
          <div className="bg-slate-50 rounded-xl p-3">
            <div className="flex items-center gap-2 text-slate-500 text-sm mb-1">
              <Activity className="w-4 h-4" />
              <span>利用率</span>
            </div>
            <div className="text-lg font-semibold text-slate-900">
              {Math.round(gpuInfo.reduce((acc, g) => acc + g.utilization, 0) / gpuInfo.length)}%
            </div>
          </div>
        </div>
      </div>

      <AnimatePresence>
        {expanded && (
          <motion.div
            initial={{ height: 0 }}
            animate={{ height: 'auto' }}
            exit={{ height: 0 }}
            className="border-t border-slate-200"
          >
            <div className="p-5 space-y-4">
              <div className="flex items-center gap-6 text-sm text-slate-500">
                <span>驱动版本: {server.driverVersion || '535.104.05'}</span>
                <span>CUDA: {server.cudaVersion || '12.2'}</span>
                <span>Docker: {server.dockerVersion || '24.0.7'}</span>
              </div>

              <div className="space-y-3">
                <h4 className="font-medium text-slate-900">GPU 详情</h4>
                {gpuInfo.map((gpu) => (
                  <div
                    key={gpu.index}
                    className="bg-slate-50 rounded-xl p-4 space-y-3"
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <Cpu className="w-5 h-5 text-blue-500" />
                        <span className="font-medium text-slate-900">
                          GPU {gpu.index} - {gpu.name}
                        </span>
                      </div>
                      <div className="flex items-center gap-4 text-sm">
                        <span className="flex items-center gap-1 text-slate-500">
                          <Thermometer className="w-4 h-4 text-orange-400" />
                          {gpu.temperature}°C
                        </span>
                        <span className="flex items-center gap-1 text-slate-500">
                          <Zap className="w-4 h-4 text-yellow-400" />
                          {gpu.powerDraw}/{gpu.powerLimit}W
                        </span>
                      </div>
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <div className="flex justify-between text-sm text-slate-500 mb-1">
                          <span>显存使用</span>
                          <span className="font-mono">
                            {(gpu.memoryUsed / 1024).toFixed(1)} / {(gpu.memoryTotal / 1024).toFixed(0)} GB
                          </span>
                        </div>
                        <div className="h-2 bg-slate-200 rounded-full overflow-hidden">
                          <div
                            className={`h-full rounded-full ${
                              gpu.memoryUsed / gpu.memoryTotal > 0.8
                                ? 'bg-red-500'
                                : gpu.memoryUsed / gpu.memoryTotal > 0.5
                                ? 'bg-yellow-500'
                                : 'bg-emerald-500'
                            }`}
                            style={{ width: `${(gpu.memoryUsed / gpu.memoryTotal) * 100}%` }}
                          />
                        </div>
                      </div>
                      <div>
                        <div className="flex justify-between text-sm text-slate-500 mb-1">
                          <span>利用率</span>
                          <span className="font-mono">{gpu.utilization}%</span>
                        </div>
                        <div className="h-2 bg-slate-200 rounded-full overflow-hidden">
                          <div
                            className="h-full bg-blue-500 rounded-full"
                            style={{ width: `${gpu.utilization}%` }}
                          />
                        </div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  );
};

export const ServerManager: React.FC = () => {
  const { projects, currentProjectId, servers, addServer, removeServer } = useAppStore();
  const [showAddModal, setShowAddModal] = useState(false);
  const [editingServer, setEditingServer] = useState<ServerConfig | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [formData, setFormData] = useState<Partial<ServerConfig>>({
    name: '',
    host: '',
    sshPort: 22,
    username: 'root',
    authType: 'password',
    password: '',
    useJumpHost: false,
  });

  const currentProject = projects.find((p) => p.id === currentProjectId);
  const projectServers = servers.filter(
    (s) =>
      s.projectId === currentProjectId &&
      (s.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        s.host.toLowerCase().includes(searchQuery.toLowerCase()))
  );

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (editingServer) {
      // Update existing
    } else {
      const newServer: ServerConfig = {
        id: Date.now().toString(),
        projectId: currentProjectId || '',
        name: formData.name || '',
        host: formData.host || '',
        sshPort: formData.sshPort || 22,
        username: formData.username || 'root',
        authType: formData.authType || 'password',
        password: formData.password,
        useJumpHost: formData.useJumpHost || false,
        status: 'offline',
      };
      addServer(newServer);
    }
    setShowAddModal(false);
    setEditingServer(null);
    setFormData({
      name: '',
      host: '',
      sshPort: 22,
      username: 'root',
      authType: 'password',
      password: '',
      useJumpHost: false,
    });
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">服务器管理</h1>
          <p className="text-slate-500 mt-1">
            {currentProject?.name} · {projectServers.length} 台服务器
          </p>
        </div>
        <button
          onClick={() => setShowAddModal(true)}
          className="flex items-center gap-2 px-5 py-2.5 bg-blue-600 hover:bg-blue-700 text-white rounded-xl font-medium transition-colors shadow-lg shadow-blue-500/25"
        >
          <Plus className="w-5 h-5" />
          添加服务器
        </button>
      </div>

      {/* Search & Filter */}
      <div className="flex items-center gap-4">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400" />
          <input
            type="text"
            placeholder="搜索服务器..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full pl-10 pr-4 py-2.5 bg-white border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>
        <button className="flex items-center gap-2 px-4 py-2.5 border border-slate-200 rounded-xl text-slate-600 hover:bg-slate-50 transition-colors">
          <Filter className="w-4 h-4" />
          筛选
        </button>
      </div>

      {/* Server Grid */}
      <div className="grid gap-4">
        {projectServers.length === 0 ? (
          <div className="text-center py-16 bg-white rounded-2xl border border-slate-200 border-dashed">
            <Server className="w-16 h-16 text-slate-300 mx-auto mb-4" />
            <h3 className="text-lg font-medium text-slate-900 mb-1">暂无服务器</h3>
            <p className="text-slate-500 mb-4">添加服务器到当前项目</p>
            <button
              onClick={() => setShowAddModal(true)}
              className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors"
            >
              立即添加
            </button>
          </div>
        ) : (
          projectServers.map((server) => (
            <ServerCard
              key={server.id}
              server={server}
              onEdit={(s) => {
                setEditingServer(s);
                setFormData(s);
                setShowAddModal(true);
              }}
              onDelete={removeServer}
            />
          ))
        )}
      </div>

      {/* Add/Edit Modal */}
      <AnimatePresence>
        {showAddModal && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
            onClick={() => setShowAddModal(false)}
          >
            <motion.div
              initial={{ scale: 0.95, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.95, opacity: 0 }}
              className="bg-white rounded-2xl p-6 w-full max-w-lg"
              onClick={(e) => e.stopPropagation()}
            >
              <h2 className="text-xl font-bold text-slate-900 mb-6">
                {editingServer ? '编辑服务器' : '添加服务器'}
              </h2>
              <form onSubmit={handleSubmit} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    服务器名称
                  </label>
                  <input
                    type="text"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    className="w-full px-4 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500"
                    placeholder="例如: GPU-Node-01"
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-slate-700 mb-1">
                      主机地址
                    </label>
                    <input
                      type="text"
                      value={formData.host}
                      onChange={(e) => setFormData({ ...formData, host: e.target.value })}
                      className="w-full px-4 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500"
                      placeholder="192.168.1.100"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-slate-700 mb-1">
                      SSH 端口
                    </label>
                    <input
                      type="number"
                      value={formData.sshPort}
                      onChange={(e) => setFormData({ ...formData, sshPort: parseInt(e.target.value) })}
                      className="w-full px-4 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500"
                    />
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    用户名
                  </label>
                  <input
                    type="text"
                    value={formData.username}
                    onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                    className="w-full px-4 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    密码
                  </label>
                  <input
                    type="password"
                    value={formData.password}
                    onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                    className="w-full px-4 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500"
                  />
                </div>
                <div className="flex gap-3 pt-4">
                  <button
                    type="button"
                    onClick={() => setShowAddModal(false)}
                    className="flex-1 py-2.5 border border-slate-200 rounded-xl text-slate-600 hover:bg-slate-50 font-medium transition-colors"
                  >
                    取消
                  </button>
                  <button
                    type="submit"
                    className="flex-1 py-2.5 bg-blue-600 hover:bg-blue-700 text-white rounded-xl font-medium transition-colors"
                  >
                    {editingServer ? '保存' : '添加'}
                  </button>
                </div>
              </form>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};
