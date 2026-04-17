import React, { useEffect, useMemo, useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import {
  Activity,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  Cpu,
  Edit3,
  Filter,
  HardDrive,
  Network,
  Plus,
  RefreshCw,
  Search,
  Server,
  ShieldCheck,
  Thermometer,
  Trash2,
  XCircle,
  Zap,
} from 'lucide-react';
import { useAppStore } from '../store';
import { GPUInfo, ServerConfig } from '../types';

type ServerTestResponse = {
  success: boolean;
  message: string;
  gpuInfo?: GPUInfo[];
  driverVersion?: string;
  cudaVersion?: string;
  dockerVersion?: string;
  server?: ServerConfig;
};

type NetdataStatusResponse = {
  endpoint: string;
  reachable: boolean;
  message: string;
  hostname?: string;
  version?: string;
  dashboardPath: string;
  server?: ServerConfig;
};

const emptyForm: Partial<ServerConfig> = {
  name: '',
  host: '',
  sshPort: 22,
  username: 'root',
  authType: 'password',
  password: '',
  privateKey: '',
  isJumpHost: false,
  useJumpHost: false,
  jumpHostId: '',
  npuExporterEndpoint: '',
  netdataEndpoint: 'http://127.0.0.1:19999',
};

const requestJSON = async <T,>(path: string, init?: RequestInit): Promise<T> => {
  const response = await fetch(path, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers || {}),
    },
  });
  const text = await response.text();
  const payload = text ? JSON.parse(text) : null;
  if (!response.ok) {
    throw new Error(payload?.error || response.statusText || 'request failed');
  }
  return payload as T;
};

const formatMB = (value?: number) => {
  const mb = value || 0;
  if (mb >= 1024) return `${(mb / 1024).toFixed(1)} GB`;
  return `${mb.toFixed(0)} MB`;
};

const acceleratorName = (gpu: GPUInfo) => (gpu.type === 'npu' ? `NPU ${gpu.index}` : `GPU ${gpu.index}`);

const buildServerPayload = (
  formData: Partial<ServerConfig>,
  projectId: string,
  existing?: ServerConfig | null
): ServerConfig => {
  const useJumpHost = Boolean(formData.useJumpHost);
  const authType = formData.authType || 'password';

  return {
    id: existing?.id || Date.now().toString(),
    projectId: existing?.projectId || projectId,
    name: formData.name?.trim() || '未命名服务器',
    host: formData.host?.trim() || '',
    sshPort: Number(formData.sshPort) || 22,
    username: formData.username?.trim() || 'root',
    authType,
    password: formData.password || undefined,
    privateKey: formData.privateKey || undefined,
    isJumpHost: Boolean(formData.isJumpHost),
    useJumpHost,
    jumpHostId: useJumpHost ? formData.jumpHostId : undefined,
    gpuInfo: existing?.gpuInfo,
    driverVersion: existing?.driverVersion,
    cudaVersion: existing?.cudaVersion,
    dockerVersion: existing?.dockerVersion,
    npuExporterEndpoint: formData.npuExporterEndpoint || existing?.npuExporterEndpoint,
    npuExporterStatus: existing?.npuExporterStatus,
    npuExporterLastCheck: existing?.npuExporterLastCheck,
    netdataEndpoint: formData.netdataEndpoint || existing?.netdataEndpoint || 'http://127.0.0.1:19999',
    netdataStatus: existing?.netdataStatus,
    netdataLastCheck: existing?.netdataLastCheck,
    lastCheck: existing?.lastCheck,
    status: existing?.status || 'offline',
  };
};

const ServerCard: React.FC<{
  server: ServerConfig;
  jumpName?: string;
  isChecking: boolean;
  onProbe: (server: ServerConfig) => void;
  onRefreshNetdata: (server: ServerConfig) => void;
  onInstallNetdata: (server: ServerConfig) => void;
  onEdit: (server: ServerConfig) => void;
  onDelete: (id: string) => void;
}> = ({ server, jumpName, isChecking, onProbe, onRefreshNetdata, onInstallNetdata, onEdit, onDelete }) => {
  const [expanded, setExpanded] = useState(false);
  const accelerators = server.gpuInfo || [];
  const gpuCount = accelerators.filter((gpu) => gpu.type !== 'npu').length;
  const npuCount = accelerators.filter((gpu) => gpu.type === 'npu').length;
  const totalMemory = accelerators.reduce((acc, gpu) => acc + gpu.memoryTotal, 0);
  const averageTemperature =
    accelerators.length > 0
      ? Math.round(accelerators.reduce((acc, gpu) => acc + gpu.temperature, 0) / accelerators.length)
      : null;
  const averageUtilization =
    accelerators.length > 0
      ? Math.round(accelerators.reduce((acc, gpu) => acc + gpu.utilization, 0) / accelerators.length)
      : null;

  return (
    <motion.div
      layout
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className="bg-white rounded-2xl border border-slate-200 overflow-hidden hover:shadow-lg transition-shadow"
    >
      <div className="p-5">
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-center gap-4 min-w-0">
            <div
              className={`w-12 h-12 rounded-xl flex items-center justify-center ${
                server.status === 'online'
                  ? 'bg-emerald-100 text-emerald-600'
                  : server.status === 'checking'
                  ? 'bg-blue-100 text-blue-600'
                  : 'bg-red-100 text-red-600'
              }`}
            >
              <Server className="w-6 h-6" />
            </div>
            <div className="min-w-0">
              <h3 className="font-semibold text-slate-900 text-lg truncate">{server.name}</h3>
              <div className="flex flex-wrap items-center gap-2 mt-1">
                <span className="text-sm text-slate-500">{server.host}</span>
                <span className="text-slate-300">|</span>
                <span className="text-sm text-slate-500">SSH {server.sshPort}</span>
                <span
                  className={`px-2 py-0.5 rounded-full text-xs font-medium ${
                    server.status === 'online'
                      ? 'bg-emerald-100 text-emerald-700'
                      : server.status === 'checking'
                      ? 'bg-blue-100 text-blue-700'
                      : 'bg-red-100 text-red-700'
                  }`}
                >
                  {server.status === 'online' ? '??' : server.status === 'checking' ? '???' : '??'}
                </span>
                {server.isJumpHost && (
                  <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-indigo-50 text-indigo-700">
                    ???
                  </span>
                )}
                {server.useJumpHost && (
                  <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-slate-100 text-slate-600">
                    ? {jumpName || '???'} ??
                  </span>
                )}
                <span
                  className={`px-2 py-0.5 rounded-full text-xs font-medium ${
                    server.netdataStatus === 'online'
                      ? 'bg-emerald-100 text-emerald-700'
                      : server.netdataStatus === 'offline'
                      ? 'bg-amber-100 text-amber-700'
                      : 'bg-slate-100 text-slate-600'
                  }`}
                >
                  Netdata {server.netdataStatus === 'online' ? '??' : server.netdataStatus === 'offline' ? '??' : '???'}
                </span>
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            <button
              onClick={() => onProbe(server)}
              disabled={isChecking}
              className="flex items-center gap-1.5 px-3 py-2 bg-slate-900 hover:bg-slate-800 disabled:bg-slate-300 text-white rounded-lg text-sm font-medium transition-colors"
            >
              <RefreshCw className={`w-4 h-4 ${isChecking ? 'animate-spin' : ''}`} />
              SSH ??
            </button>
            <button
              onClick={() => onInstallNetdata(server)}
              className="flex items-center gap-1.5 px-3 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors"
            >
              <Activity className="w-4 h-4" />
              ?? Netdata
            </button>
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
              <span>???</span>
            </div>
            <div className="text-lg font-semibold text-slate-900">
              {accelerators.length > 0 ? `${gpuCount} GPU / ${npuCount} NPU` : '???'}
            </div>
          </div>
          <div className="bg-slate-50 rounded-xl p-3">
            <div className="flex items-center gap-2 text-slate-500 text-sm mb-1">
              <HardDrive className="w-4 h-4" />
              <span>?? / HBM</span>
            </div>
            <div className="text-lg font-semibold text-slate-900">
              {accelerators.length > 0 ? formatMB(totalMemory) : '???'}
            </div>
          </div>
          <div className="bg-slate-50 rounded-xl p-3">
            <div className="flex items-center gap-2 text-slate-500 text-sm mb-1">
              <Thermometer className="w-4 h-4" />
              <span>??</span>
            </div>
            <div className="text-lg font-semibold text-slate-900">
              {averageTemperature === null ? '???' : `${averageTemperature}?C`}
            </div>
          </div>
          <div className="bg-slate-50 rounded-xl p-3">
            <div className="flex items-center gap-2 text-slate-500 text-sm mb-1">
              <Activity className="w-4 h-4" />
              <span>???</span>
            </div>
            <div className="text-lg font-semibold text-slate-900">
              {averageUtilization === null ? '???' : `${averageUtilization}%`}
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
              <div className="flex flex-wrap items-center gap-6 text-sm text-slate-500">
                <span>??: {server.driverVersion || '???'}</span>
                <span>CUDA: {server.cudaVersion || '???'}</span>
                <span>Docker: {server.dockerVersion || '???'}</span>
                <span>
                  NPU Exporter: {server.npuExporterStatus || '???'}
                  {server.npuExporterEndpoint ? ` (${server.npuExporterEndpoint})` : ''}
                </span>
                <span>
                  Netdata: {server.netdataStatus || '???'}
                  {server.netdataEndpoint ? ` (${server.netdataEndpoint})` : ''}
                </span>
                <span>????: {server.lastCheck ? new Date(server.lastCheck).toLocaleString() : '???'}</span>
              </div>

              <div className="space-y-3">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <h4 className="font-medium text-slate-900">Netdata ??</h4>
                    <p className="text-sm text-slate-500">??????????? Netdata ????????</p>
                  </div>
                  <div className="flex flex-wrap gap-3">
                    <button
                      onClick={() => onRefreshNetdata(server)}
                      className="inline-flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-700 hover:bg-slate-50 transition-colors"
                    >
                      <RefreshCw className="w-4 h-4" />
                      ?? Netdata ??
                    </button>
                    <a
                      href={`/api/servers/${server.id}/netdata/dashboard/v1/`}
                      target="_blank"
                      rel="noreferrer"
                      className="inline-flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-700 hover:bg-slate-50 transition-colors"
                    >
                      <Activity className="w-4 h-4" />
                      ?????
                    </a>
                  </div>
                </div>

                <div className="rounded-2xl border border-slate-200 overflow-hidden bg-slate-950">
                  {server.netdataStatus === 'online' ? (
                    <iframe
                      key={`${server.id}-${server.netdataLastCheck || 'dashboard'}`}
                      src={`/api/servers/${server.id}/netdata/dashboard/v1/`}
                      title={`netdata-${server.name}`}
                      className="w-full h-[640px] border-0 bg-white"
                    />
                  ) : (
                    <div className="p-5 text-sm text-slate-300">
                      Netdata ???????????? Netdata??????????????????????
                    </div>
                  )}
                </div>
              </div>

              <div className="space-y-3">
                <h4 className="font-medium text-slate-900">?????</h4>
                {accelerators.length === 0 ? (
                  <div className="border border-dashed border-slate-200 rounded-xl p-5 text-sm text-slate-500">
                    ???????? GPU / NPU ??????SSH ?????????? SSH ???????????????
                  </div>
                ) : (
                  accelerators.map((gpu) => {
                    const memoryPercent = gpu.memoryTotal > 0 ? (gpu.memoryUsed / gpu.memoryTotal) * 100 : 0;
                    const powerText = gpu.powerLimit > 0 ? `${gpu.powerDraw}/${gpu.powerLimit}W` : `${gpu.powerDraw}W`;

                    return (
                      <div key={`${gpu.type || 'gpu'}-${gpu.index}`} className="bg-slate-50 rounded-xl p-4 space-y-3">
                        <div className="flex items-center justify-between gap-4">
                          <div className="flex items-center gap-3 min-w-0">
                            <Cpu className="w-5 h-5 text-blue-500" />
                            <span className="font-medium text-slate-900 truncate">
                              {acceleratorName(gpu)} - {gpu.name}
                            </span>
                            {gpu.health && (
                              <span className="px-2 py-0.5 rounded-full bg-white text-slate-500 text-xs">
                                {gpu.health}
                              </span>
                            )}
                          </div>
                          <div className="flex items-center gap-4 text-sm shrink-0">
                            <span className="flex items-center gap-1 text-slate-500">
                              <Thermometer className="w-4 h-4 text-orange-400" />
                              {gpu.temperature}?C
                            </span>
                            <span className="flex items-center gap-1 text-slate-500">
                              <Zap className="w-4 h-4 text-yellow-400" />
                              {powerText}
                            </span>
                          </div>
                        </div>

                        <div className="grid grid-cols-2 gap-4">
                          <div>
                            <div className="flex justify-between text-sm text-slate-500 mb-1">
                              <span>{gpu.type === 'npu' ? 'HBM ??' : '????'}</span>
                              <span className="font-mono">
                                {formatMB(gpu.memoryUsed)} / {formatMB(gpu.memoryTotal)}
                              </span>
                            </div>
                            <div className="h-2 bg-slate-200 rounded-full overflow-hidden">
                              <div
                                className={`h-full rounded-full ${
                                  memoryPercent > 80
                                    ? 'bg-red-500'
                                    : memoryPercent > 50
                                    ? 'bg-yellow-500'
                                    : 'bg-emerald-500'
                                }`}
                                style={{ width: `${Math.min(memoryPercent, 100)}%` }}
                              />
                            </div>
                          </div>
                          <div>
                            <div className="flex justify-between text-sm text-slate-500 mb-1">
                              <span>???</span>
                              <span className="font-mono">{gpu.utilization}%</span>
                            </div>
                            <div className="h-2 bg-slate-200 rounded-full overflow-hidden">
                              <div
                                className="h-full bg-blue-500 rounded-full"
                                style={{ width: `${Math.min(gpu.utilization, 100)}%` }}
                              />
                            </div>
                          </div>
                        </div>
                      </div>
                    );
                  })
                )}
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  );
};

export const ServerManager: React.FC = () => {
  const {
    projects,
    currentProjectId,
    servers,
    addServer,
    setServers,
    updateServer,
    removeServer,
  } = useAppStore();
  const [showAddModal, setShowAddModal] = useState(false);
  const [editingServer, setEditingServer] = useState<ServerConfig | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [formData, setFormData] = useState<Partial<ServerConfig>>(emptyForm);
  const [checkingId, setCheckingId] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [notice, setNotice] = useState('');
  const [error, setError] = useState('');

  const currentProject = projects.find((p) => p.id === currentProjectId);

  useEffect(() => {
    let active = true;
    requestJSON<ServerConfig[]>('/api/servers')
      .then((remoteServers) => {
        if (!active) return;
        setServers(remoteServers);
        setError('');
      })
      .catch((err) => {
        if (!active) return;
        setError(`后端服务器列表暂时不可用：${err.message}`);
      });
    return () => {
      active = false;
    };
  }, [setServers]);

  const projectServers = useMemo(() => {
    const keyword = searchQuery.trim().toLowerCase();
    return servers.filter((server) => {
      if (server.projectId !== currentProjectId) return false;
      if (!keyword) return true;
      return `${server.name} ${server.host}`.toLowerCase().includes(keyword);
    });
  }, [servers, currentProjectId, searchQuery]);

  const jumpCandidates = useMemo(
    () =>
      servers.filter(
        (server) =>
          server.projectId === currentProjectId &&
          server.isJumpHost &&
          server.id !== editingServer?.id
      ),
    [servers, currentProjectId, editingServer?.id]
  );

  const resetForm = () => {
    setFormData({ ...emptyForm });
    setEditingServer(null);
  };

  const closeModal = () => {
    setShowAddModal(false);
    resetForm();
  };

  const openCreate = () => {
    resetForm();
    setShowAddModal(true);
  };

  const openEdit = (server: ServerConfig) => {
    setEditingServer(server);
    setFormData({
      ...emptyForm,
      ...server,
      password: server.password || '',
      privateKey: server.privateKey || '',
      jumpHostId: server.jumpHostId || '',
      npuExporterEndpoint: server.npuExporterEndpoint || '',
      netdataEndpoint: server.netdataEndpoint || 'http://127.0.0.1:19999',
    });
    setShowAddModal(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!currentProjectId && !editingServer?.projectId) {
      setError('请先创建或选择一个项目。');
      return;
    }
    if (formData.useJumpHost && !formData.jumpHostId) {
      setError('启用跳板机连接时必须选择一台跳板机。');
      return;
    }

    const payload = buildServerPayload(formData, currentProjectId || editingServer?.projectId || '', editingServer);
    setSaving(true);
    setError('');

    try {
      if (editingServer) {
        const updated = await requestJSON<ServerConfig>(`/api/servers/${editingServer.id}`, {
          method: 'PATCH',
          body: JSON.stringify(payload),
        });
        updateServer(editingServer.id, updated);
        setNotice(`已更新服务器 ${updated.name}`);
      } else {
        const created = await requestJSON<ServerConfig>(`/api/servers?projectId=${encodeURIComponent(payload.projectId)}`, {
          method: 'POST',
          body: JSON.stringify(payload),
        });
        addServer(created);
        setNotice(`已添加服务器 ${created.name}`);
      }
      closeModal();
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存服务器失败');
    } finally {
      setSaving(false);
    }
  };

  const handleProbe = async (server: ServerConfig) => {
    setCheckingId(server.id);
    setNotice('');
    setError('');
    updateServer(server.id, { status: 'checking' });

    try {
      const result = await requestJSON<ServerTestResponse>(`/api/servers/${server.id}/test`, {
        method: 'POST',
      });
      const updated = result.server || {
        ...server,
        status: result.success ? 'online' : 'offline',
        gpuInfo: result.gpuInfo,
        driverVersion: result.driverVersion,
        cudaVersion: result.cudaVersion,
        dockerVersion: result.dockerVersion,
        lastCheck: new Date().toISOString(),
      };
      updateServer(server.id, updated);
      if (result.success) {
        setNotice(result.message || '服务器采集完成');
      } else {
        setError(result.message || '服务器采集失败');
      }
    } catch (err) {
      updateServer(server.id, { status: 'offline', lastCheck: new Date().toISOString() });
      setError(err instanceof Error ? err.message : '服务器采集失败');
    } finally {
      setCheckingId(null);
    }
  };

  const handleRefreshNetdata = async (server: ServerConfig) => {
    setNotice('');
    setError('');
    try {
      const result = await requestJSON<NetdataStatusResponse>(`/api/servers/${server.id}/netdata`);
      if (result.server) {
        updateServer(server.id, result.server);
      } else {
        updateServer(server.id, {
          netdataEndpoint: result.endpoint,
          netdataStatus: result.reachable ? 'online' : 'offline',
          netdataLastCheck: new Date().toISOString(),
        });
      }
      setNotice(result.message || 'Netdata 状态已刷新');
    } catch (err) {
      updateServer(server.id, {
        netdataStatus: 'offline',
        netdataLastCheck: new Date().toISOString(),
      });
      setError(err instanceof Error ? err.message : '刷新 Netdata 状态失败');
    }
  };

  const handleInstallNetdata = async (server: ServerConfig) => {
    setNotice('');
    setError('');
    try {
      await requestJSON('/api/remote-tasks', {
        method: 'POST',
        body: JSON.stringify({
          name: `安装 Netdata - ${server.name}`,
          scope: 'selected',
          executionType: 'preset',
          presetId: 'install_netdata',
          serverIds: [server.id],
        }),
      });
      setNotice(`已向 ${server.name} 下发 Netdata 安装任务。安装完成后刷新状态即可查看看板。`);
    } catch (err) {
      setError(err instanceof Error ? err.message : '下发 Netdata 安装任务失败');
    }
  };

  const handleDelete = async (id: string) => {
    const server = servers.find((item) => item.id === id);
    const confirmed = window.confirm(`删除服务器“${server?.name || id}”？`);
    if (!confirmed) return;

    try {
      await requestJSON<void>(`/api/servers/${id}`, { method: 'DELETE' });
      removeServer(id);
      setNotice('服务器已删除');
    } catch (err) {
      setError(err instanceof Error ? err.message : '删除服务器失败');
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">服务器管理</h1>
          <p className="text-slate-500 mt-1">
            {currentProject?.name || '未选择项目'} | {projectServers.length} 台服务器
          </p>
        </div>
        <button
          onClick={openCreate}
          disabled={!currentProjectId}
          className="flex items-center gap-2 px-5 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:bg-slate-300 text-white rounded-xl font-medium transition-colors shadow-lg shadow-blue-500/25"
        >
          <Plus className="w-5 h-5" />
          添加服务器
        </button>
      </div>

      {(notice || error) && (
        <div
          className={`flex items-center gap-2 rounded-xl border px-4 py-3 text-sm ${
            error ? 'border-red-200 bg-red-50 text-red-700' : 'border-emerald-200 bg-emerald-50 text-emerald-700'
          }`}
        >
          {error ? <XCircle className="w-4 h-4" /> : <CheckCircle2 className="w-4 h-4" />}
          <span>{error || notice}</span>
        </div>
      )}

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
        <div className="flex items-center gap-2 px-4 py-2.5 border border-slate-200 rounded-xl text-slate-600 bg-white">
          <Filter className="w-4 h-4" />
          {projectServers.filter((server) => server.isJumpHost).length} 台跳板机
        </div>
      </div>

      <div className="grid gap-4">
        {projectServers.length === 0 ? (
          <div className="text-center py-16 bg-white rounded-2xl border border-slate-200 border-dashed">
            <Server className="w-16 h-16 text-slate-300 mx-auto mb-4" />
            <h3 className="text-lg font-medium text-slate-900 mb-1">暂无服务器</h3>
            <p className="text-slate-500 mb-4">添加服务器后，点击采集信息即可通过 SSH 获取 GPU/NPU 和系统信息。</p>
            <button
              onClick={openCreate}
              disabled={!currentProjectId}
              className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-slate-300 text-white rounded-lg font-medium transition-colors"
            >
              立即添加
            </button>
          </div>
        ) : (
          projectServers.map((server) => (
            <ServerCard
              key={server.id}
              server={server}
              jumpName={servers.find((candidate) => candidate.id === server.jumpHostId)?.name}
              isChecking={checkingId === server.id}
              onProbe={handleProbe}
              onRefreshNetdata={handleRefreshNetdata}
              onInstallNetdata={handleInstallNetdata}
              onEdit={openEdit}
              onDelete={handleDelete}
            />
          ))
        )}
      </div>

      <AnimatePresence>
        {showAddModal && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4"
            onClick={closeModal}
          >
            <motion.div
              initial={{ scale: 0.95, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.95, opacity: 0 }}
              className="bg-white rounded-2xl p-6 w-full max-w-2xl max-h-[90vh] overflow-y-auto"
              onClick={(e) => e.stopPropagation()}
            >
              <h2 className="text-xl font-bold text-slate-900 mb-6">
                {editingServer ? '编辑服务器' : '添加服务器'}
              </h2>
              <form onSubmit={handleSubmit} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">服务器名称</label>
                  <input
                    type="text"
                    required
                    value={formData.name || ''}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    className="w-full px-4 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500"
                    placeholder="例如: npu-node-01"
                  />
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-slate-700 mb-1">主机地址</label>
                    <input
                      type="text"
                      required
                      value={formData.host || ''}
                      onChange={(e) => setFormData({ ...formData, host: e.target.value })}
                      className="w-full px-4 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500"
                      placeholder="公网 IP 或内网 IP"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-slate-700 mb-1">SSH 端口</label>
                    <input
                      type="number"
                      min={1}
                      value={formData.sshPort || 22}
                      onChange={(e) => setFormData({ ...formData, sshPort: Number(e.target.value) || 22 })}
                      className="w-full px-4 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-slate-700 mb-1">用户名</label>
                    <input
                      type="text"
                      value={formData.username || 'root'}
                      onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                      className="w-full px-4 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-slate-700 mb-1">认证方式</label>
                    <select
                      value={formData.authType || 'password'}
                      onChange={(e) =>
                        setFormData({ ...formData, authType: e.target.value as ServerConfig['authType'] })
                      }
                      className="w-full px-4 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500 bg-white"
                    >
                      <option value="password">密码</option>
                      <option value="key">私钥</option>
                    </select>
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    {formData.authType === 'key' ? '私钥口令或密码备用' : '密码'}
                  </label>
                  <input
                    type="password"
                    value={formData.password || ''}
                    onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                    className="w-full px-4 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500"
                  />
                </div>

                {formData.authType === 'key' && (
                  <div>
                    <label className="block text-sm font-medium text-slate-700 mb-1">SSH 私钥</label>
                    <textarea
                      rows={6}
                      value={formData.privateKey || ''}
                      onChange={(e) => setFormData({ ...formData, privateKey: e.target.value })}
                      className="w-full px-4 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500 font-mono text-sm"
                      placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
                    />
                  </div>
                )}

                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    NPU Exporter Metrics 地址（可选）
                  </label>
                  <input
                    type="text"
                    value={formData.npuExporterEndpoint || ''}
                    onChange={(e) => setFormData({ ...formData, npuExporterEndpoint: e.target.value })}
                    className="w-full px-4 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500"
                placeholder="默认 http://127.0.0.1:8082/metrics，系统会自动回退探测 9101"
                  />
                  <p className="text-xs text-slate-500 mt-1">
                    这个地址从目标服务器本机访问；通过跳板机连接时也仍然是目标机本地地址。
                  </p>
                </div>

                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">Netdata ??????</label>
                  <input
                    type="text"
                    value={formData.netdataEndpoint || ''}
                    onChange={(e) => setFormData({ ...formData, netdataEndpoint: e.target.value })}
                    className="w-full px-4 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500"
                    placeholder="?? http://127.0.0.1:19999"
                  />
                  <p className="text-xs text-slate-500 mt-1">
                    ????? SSH ????????? Netdata ?????????????
                  </p>
                </div>

                <div className="grid gap-3 rounded-xl border border-slate-200 p-4">
                  <label className="flex items-start gap-3">
                    <input
                      type="checkbox"
                      checked={Boolean(formData.isJumpHost)}
                      onChange={(e) => setFormData({ ...formData, isJumpHost: e.target.checked })}
                      className="mt-1"
                    />
                    <span>
                      <span className="flex items-center gap-2 text-sm font-medium text-slate-800">
                        <ShieldCheck className="w-4 h-4 text-indigo-500" />
                        这台服务器可作为跳板机
                      </span>
                      <span className="block text-sm text-slate-500 mt-1">
                        后续内网服务器可以通过它建立 SSH 转发。
                      </span>
                    </span>
                  </label>

                  <label className="flex items-start gap-3">
                    <input
                      type="checkbox"
                      checked={Boolean(formData.useJumpHost)}
                      onChange={(e) =>
                        setFormData({
                          ...formData,
                          useJumpHost: e.target.checked,
                          jumpHostId: e.target.checked ? formData.jumpHostId || jumpCandidates[0]?.id || '' : '',
                        })
                      }
                      className="mt-1"
                    />
                    <span>
                      <span className="flex items-center gap-2 text-sm font-medium text-slate-800">
                        <Network className="w-4 h-4 text-blue-500" />
                        通过跳板机连接
                      </span>
                      <span className="block text-sm text-slate-500 mt-1">
                        当前服务器可以填写内网 IP，后端会先连跳板机，再从跳板机连目标 SSH。
                      </span>
                    </span>
                  </label>

                  {formData.useJumpHost && (
                    <div>
                      <label className="block text-sm font-medium text-slate-700 mb-1">选择跳板机</label>
                      <select
                        value={formData.jumpHostId || ''}
                        onChange={(e) => setFormData({ ...formData, jumpHostId: e.target.value })}
                        className="w-full px-4 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500 bg-white"
                      >
                        <option value="">请选择跳板机</option>
                        {jumpCandidates.map((server) => (
                          <option key={server.id} value={server.id}>
                            {server.name} ({server.host})
                          </option>
                        ))}
                      </select>
                      {jumpCandidates.length === 0 && (
                        <p className="text-sm text-red-500 mt-2">
                          当前项目还没有可用跳板机，请先添加一台服务器并勾选“可作为跳板机”。
                        </p>
                      )}
                    </div>
                  )}
                </div>

                <div className="flex gap-3 pt-4">
                  <button
                    type="button"
                    onClick={closeModal}
                    className="flex-1 py-2.5 border border-slate-200 rounded-xl text-slate-600 hover:bg-slate-50 font-medium transition-colors"
                  >
                    取消
                  </button>
                  <button
                    type="submit"
                    disabled={saving}
                    className="flex-1 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:bg-slate-300 text-white rounded-xl font-medium transition-colors"
                  >
                    {saving ? '保存中...' : editingServer ? '保存' : '添加'}
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
