import React, { useEffect, useMemo, useState } from 'react';
import { motion } from 'framer-motion';
import { Check, Cpu, Server, Thermometer, Zap } from 'lucide-react';
import { useAppStore } from '../store';

export const ServerWizard: React.FC = () => {
  const { wizard, updateWizardConfig, servers, currentProjectId } = useAppStore();
  const [selected, setSelected] = useState<string[]>(wizard.config.servers || []);
  const [expanded, setExpanded] = useState<string | null>(null);

  const projectServers = useMemo(
    () => servers.filter((server) => server.projectId === currentProjectId),
    [servers, currentProjectId]
  );

  useEffect(() => {
    const validIds = new Set(projectServers.map((server) => server.id));
    const nextSelected = selected.filter((serverId) => validIds.has(serverId));
    if (nextSelected.length !== selected.length) {
      setSelected(nextSelected);
      updateWizardConfig({ servers: nextSelected });
    }
  }, [projectServers, selected, updateWizardConfig]);

  const toggleServer = (id: string) => {
    const next = selected.includes(id) ? selected.filter((serverId) => serverId !== id) : [...selected, id];
    setSelected(next);
    updateWizardConfig({ servers: next });
  };

  const formatMB = (mb: number) => {
    if (mb >= 1024) return `${(mb / 1024).toFixed(1)} GB`;
    return `${mb} MB`;
  };

  const onlineCount = projectServers.filter((server) => server.status === 'online').length;
  const acceleratorCount = projectServers.reduce((acc, server) => acc + (server.gpuInfo?.length || 0), 0);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-xs text-slate-500 font-mono">
          <span className="text-green-400">●</span> {onlineCount} ONLINE
          <span className="text-slate-600">|</span>
          <span className="text-blue-400">●</span> {acceleratorCount} ACCELERATORS
        </div>
        <div className="text-xs text-slate-500 font-mono">
          来自服务器管理的真实节点
        </div>
      </div>

      <div className="grid gap-3">
        {projectServers.length === 0 ? (
          <div className="border border-dashed border-slate-700 rounded-lg p-6 text-center text-slate-500">
            当前项目还没有服务器。请先到“服务器管理”添加节点并采集信息。
          </div>
        ) : (
          projectServers.map((server) => {
            const accelerators = server.gpuInfo || [];
            const npuCount = accelerators.filter((gpu) => gpu.type === 'npu').length;
            const gpuCount = accelerators.length - npuCount;

            return (
              <motion.div
                key={server.id}
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                className={`border rounded-lg overflow-hidden transition-all ${
                  selected.includes(server.id) ? 'border-green-500/50 bg-green-500/5' : 'border-slate-700 bg-slate-900/50'
                }`}
              >
                <div
                  className="flex items-center gap-3 p-3 cursor-pointer"
                  onClick={() => toggleServer(server.id)}
                >
                  <div className={`w-4 h-4 rounded border flex items-center justify-center ${
                    selected.includes(server.id) ? 'bg-green-500 border-green-500' : 'border-slate-600'
                  }`}>
                    {selected.includes(server.id) && <Check className="w-3 h-3 text-black" />}
                  </div>
                  <Server className="w-4 h-4 text-slate-400" />
                  <div className="flex-1 min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="text-sm font-mono text-slate-200 truncate">{server.name}</span>
                      <span className="text-xs text-slate-500 font-mono">{server.host}</span>
                      <span className="px-1.5 py-0.5 bg-slate-800 text-slate-400 text-[10px] rounded font-mono">
                        {gpuCount} GPU / {npuCount} NPU
                      </span>
                      <span className={`px-1.5 py-0.5 text-[10px] rounded font-mono ${
                        server.status === 'online' ? 'bg-green-500/10 text-green-400' : 'bg-red-500/10 text-red-400'
                      }`}>
                        {server.status.toUpperCase()}
                      </span>
                    </div>
                  </div>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      setExpanded(expanded === server.id ? null : server.id);
                    }}
                    className="text-xs text-slate-500 hover:text-slate-300"
                  >
                    {expanded === server.id ? 'COLLAPSE' : 'EXPAND'}
                  </button>
                </div>

                {expanded === server.id && (
                  <motion.div initial={{ height: 0 }} animate={{ height: 'auto' }} className="border-t border-slate-700/50">
                    <div className="p-3 space-y-3">
                      <div className="flex flex-wrap gap-4 text-xs font-mono text-slate-500">
                        <span>Driver: {server.driverVersion || '-'}</span>
                        <span>CUDA: {server.cudaVersion || '-'}</span>
                        <span>Docker: {server.dockerVersion || '-'}</span>
                        {server.useJumpHost && <span>Jump: {server.jumpHostId}</span>}
                      </div>

                      <div className="grid gap-2">
                        {accelerators.length === 0 ? (
                          <div className="bg-slate-950 rounded p-3 text-xs text-slate-500">
                            尚未采集 GPU/NPU 信息。
                          </div>
                        ) : (
                          accelerators.map((gpu) => {
                            const memoryPercent = gpu.memoryTotal > 0 ? (gpu.memoryUsed / gpu.memoryTotal) * 100 : 0;

                            return (
                              <div key={`${gpu.type || 'gpu'}-${gpu.index}`} className="bg-slate-950 rounded p-2.5">
                                <div className="flex items-center justify-between mb-2">
                                  <div className="flex items-center gap-2 min-w-0">
                                    <Cpu className="w-3 h-3 text-blue-400" />
                                    <span className="text-xs font-mono text-slate-300">
                                      {gpu.type === 'npu' ? 'NPU' : 'GPU'}-{gpu.index}
                                    </span>
                                    <span className="text-xs text-slate-500 truncate">{gpu.name}</span>
                                  </div>
                                  <div className="flex items-center gap-3 text-xs">
                                    <span className="flex items-center gap-1 text-slate-400">
                                      <Thermometer className="w-3 h-3" /> {gpu.temperature}°C
                                    </span>
                                    <span className="flex items-center gap-1 text-slate-400">
                                      <Zap className="w-3 h-3" /> {gpu.powerDraw}W
                                    </span>
                                  </div>
                                </div>

                                <div className="space-y-1.5">
                                  <div className="flex items-center justify-between text-[10px] text-slate-500">
                                    <span>MEMORY</span>
                                    <span className="font-mono">{formatMB(gpu.memoryUsed)} / {formatMB(gpu.memoryTotal)}</span>
                                  </div>
                                  <div className="h-1.5 bg-slate-800 rounded-full overflow-hidden">
                                    <div
                                      className="h-full bg-blue-500 rounded-full"
                                      style={{ width: `${Math.min(memoryPercent, 100)}%` }}
                                    />
                                  </div>

                                  <div className="flex items-center justify-between text-[10px] text-slate-500">
                                    <span>UTILIZATION</span>
                                    <span className="font-mono">{gpu.utilization}%</span>
                                  </div>
                                  <div className="h-1.5 bg-slate-800 rounded-full overflow-hidden">
                                    <div
                                      className={`h-full rounded-full ${
                                        gpu.utilization > 80 ? 'bg-red-500' : gpu.utilization > 50 ? 'bg-yellow-500' : 'bg-green-500'
                                      }`}
                                      style={{ width: `${Math.min(gpu.utilization, 100)}%` }}
                                    />
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
              </motion.div>
            );
          })
        )}
      </div>

      {selected.length > 0 && (
        <div className="flex items-center gap-2 p-2 bg-green-500/10 border border-green-500/30 rounded text-xs text-green-400 font-mono">
          <Check className="w-3 h-3" />
          {selected.length} NODES SELECTED
        </div>
      )}
    </div>
  );
};
