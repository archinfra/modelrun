import React, { useState } from 'react';
import { motion } from 'framer-motion';
import { Server, Plus, Check, Cpu, HardDrive, Thermometer, Zap } from 'lucide-react';
import { useAppStore } from '../store';
import { ServerConfig } from '../types';

const mockServers: ServerConfig[] = [
  {
    id: '1',
    projectId: 'default',
    name: 'gpu-node-01',
    host: '192.168.1.101',
    sshPort: 22,
    username: 'root',
    authType: 'key',
    useJumpHost: false,
    status: 'online',
    driverVersion: '535.104.05',
    cudaVersion: '12.2',
    dockerVersion: '24.0.7',
    gpuInfo: [
      { index: 0, name: 'NVIDIA A100 80GB', memoryTotal: 81920, memoryUsed: 24576, memoryFree: 57344, utilization: 45, temperature: 72, powerDraw: 285, powerLimit: 400 },
      { index: 1, name: 'NVIDIA A100 80GB', memoryTotal: 81920, memoryUsed: 18432, memoryFree: 63488, utilization: 32, temperature: 68, powerDraw: 240, powerLimit: 400 },
    ],
  },
  {
    id: '2',
    projectId: 'default',
    name: 'gpu-node-02',
    host: '192.168.1.102',
    sshPort: 22,
    username: 'root',
    authType: 'key',
    useJumpHost: false,
    status: 'online',
    driverVersion: '535.104.05',
    cudaVersion: '12.2',
    dockerVersion: '24.0.7',
    gpuInfo: [
      { index: 0, name: 'NVIDIA A100 40GB', memoryTotal: 40960, memoryUsed: 8192, memoryFree: 32768, utilization: 28, temperature: 65, powerDraw: 180, powerLimit: 250 },
    ],
  },
];

export const ServerWizard: React.FC = () => {
  const { wizard, updateWizardConfig } = useAppStore();
  const [selected, setSelected] = useState<string[]>(wizard.config.servers || []);
  const [expanded, setExpanded] = useState<string | null>(null);

  const toggleServer = (id: string) => {
    const next = selected.includes(id) ? selected.filter(s => s !== id) : [...selected, id];
    setSelected(next);
    updateWizardConfig({ servers: next });
  };

  const formatBytes = (mb: number) => {
    if (mb >= 1024) return `${(mb / 1024).toFixed(1)} GB`;
    return `${mb} MB`;
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-xs text-slate-500 font-mono">
          <span className="text-green-400">●</span> {mockServers.filter(s => s.status === 'online').length} ONLINE
          <span className="text-slate-600">|</span>
          <span className="text-blue-400">◆</span> {mockServers.reduce((acc, s) => acc + (s.gpuInfo?.length || 0), 0)} GPUs
        </div>
        <button className="flex items-center gap-1 px-3 py-1.5 bg-slate-800 hover:bg-slate-700 border border-slate-700 rounded text-xs text-slate-300 transition-colors">
          <Plus className="w-3 h-3" /> ADD NODE
        </button>
      </div>

      <div className="grid gap-3">
        {mockServers.map((server) => (
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
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-mono text-slate-200">{server.name}</span>
                  <span className="text-xs text-slate-500 font-mono">{server.host}</span>
                  <span className="px-1.5 py-0.5 bg-slate-800 text-slate-400 text-[10px] rounded font-mono">{server.gpuInfo?.length || 0}x GPU</span>
                </div>
              </div>
              <button
                onClick={(e) => { e.stopPropagation(); setExpanded(expanded === server.id ? null : server.id); }}
                className="text-xs text-slate-500 hover:text-slate-300"
              >
                {expanded === server.id ? 'COLLAPSE' : 'EXPAND'}
              </button>
            </div>

            {expanded === server.id && (
              <motion.div initial={{ height: 0 }} animate={{ height: 'auto' }} className="border-t border-slate-700/50">
                <div className="p-3 space-y-3">
                  <div className="flex gap-4 text-xs font-mono text-slate-500">
                    <span>Driver: {server.driverVersion}</span>
                    <span>CUDA: {server.cudaVersion}</span>
                    <span>Docker: {server.dockerVersion}</span>
                  </div>

                  <div className="grid gap-2">
                    {server.gpuInfo?.map((gpu) => (
                      <div key={gpu.index} className="bg-slate-950 rounded p-2.5">
                        <div className="flex items-center justify-between mb-2">
                          <div className="flex items-center gap-2">
                            <Cpu className="w-3 h-3 text-blue-400" />
                            <span className="text-xs font-mono text-slate-300">GPU-{gpu.index}</span>
                            <span className="text-xs text-slate-500">{gpu.name}</span>
                          </div>
                          <div className="flex items-center gap-3 text-xs">
                            <span className="flex items-center gap-1 text-slate-400">
                              <Thermometer className="w-3 h-3" /> {gpu.temperature}°C
                            </span>
                            <span className="flex items-center gap-1 text-slate-400">
                              <Zap className="w-3 h-3" /> {gpu.powerDraw}/{gpu.powerLimit}W
                            </span>
                          </div>
                        </div>

                        <div className="space-y-1.5">
                          <div className="flex items-center justify-between text-[10px] text-slate-500">
                            <span>MEMORY</span>
                            <span className="font-mono">{formatBytes(gpu.memoryUsed)} / {formatBytes(gpu.memoryTotal)}</span>
                          </div>
                          <div className="h-1.5 bg-slate-800 rounded-full overflow-hidden">
                            <div
                              className="h-full bg-gradient-to-r from-blue-500 to-cyan-400 rounded-full"
                              style={{ width: `${(gpu.memoryUsed / gpu.memoryTotal) * 100}%` }}
                            />
                          </div>

                          <div className="flex items-center justify-between text-[10px] text-slate-500">
                            <span>UTILIZATION</span>
                            <span className="font-mono">{gpu.utilization}%</span>
                          </div>
                          <div className="h-1.5 bg-slate-800 rounded-full overflow-hidden">
                            <div
                              className={`h-full rounded-full ${gpu.utilization > 80 ? 'bg-red-500' : gpu.utilization > 50 ? 'bg-yellow-500' : 'bg-green-500'}`}
                              style={{ width: `${gpu.utilization}%` }}
                            />
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </motion.div>
            )}
          </motion.div>
        ))}
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
