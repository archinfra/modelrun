import React from 'react';
import { motion } from 'framer-motion';
import { Activity, HardDrive, Thermometer, Zap } from 'lucide-react';
import { GPUInfo } from '../types';

interface GPUCardProps {
  gpu: GPUInfo;
  index: number;
}

const formatMB = (value: number) => {
  if (value >= 1024) return `${(value / 1024).toFixed(1)} GB`;
  return `${value.toFixed(0)} MB`;
};

export const GPUCard: React.FC<GPUCardProps> = ({ gpu, index }) => {
  const memoryPercent = gpu.memoryTotal > 0 ? (gpu.memoryUsed / gpu.memoryTotal) * 100 : 0;
  const powerPercent = gpu.powerLimit > 0 ? (gpu.powerDraw / gpu.powerLimit) * 100 : 0;
  const label = gpu.type === 'npu' ? 'NPU' : 'GPU';

  const getTempColor = (temp: number) => {
    if (temp < 60) return 'text-green-400';
    if (temp < 80) return 'text-yellow-400';
    return 'text-red-400';
  };

  const getUtilColor = (util: number) => {
    if (util < 30) return 'text-green-400';
    if (util < 70) return 'text-yellow-400';
    return 'text-red-400';
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay: index * 0.1 }}
      className="bg-slate-900/80 border border-slate-700 rounded-lg p-4 font-mono text-sm"
    >
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2 min-w-0">
          <span className="text-slate-500">{label} {gpu.index}</span>
          <span className="text-slate-300 font-medium truncate">{gpu.name}</span>
          {gpu.health && <span className="text-xs text-slate-500">{gpu.health}</span>}
        </div>
        <div className={`flex items-center gap-1 ${getTempColor(gpu.temperature)}`}>
          <Thermometer className="w-3 h-3" />
          <span>{gpu.temperature}°C</span>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3 mb-3">
        <div className="bg-slate-800/50 rounded p-2">
          <div className="flex items-center gap-1 text-slate-500 mb-1">
            <HardDrive className="w-3 h-3" />
            <span>{gpu.type === 'npu' ? 'HBM' : '显存'}</span>
          </div>
          <div className="text-slate-300">
            {formatMB(gpu.memoryUsed)} / {formatMB(gpu.memoryTotal)}
          </div>
          <div className="w-full h-1 bg-slate-700 rounded mt-1">
            <div
              className={`h-full rounded ${memoryPercent > 80 ? 'bg-red-500' : memoryPercent > 50 ? 'bg-yellow-500' : 'bg-green-500'}`}
              style={{ width: `${Math.min(memoryPercent, 100)}%` }}
            />
          </div>
        </div>

        <div className="bg-slate-800/50 rounded p-2">
          <div className="flex items-center gap-1 text-slate-500 mb-1">
            <Zap className="w-3 h-3" />
            <span>功耗</span>
          </div>
          <div className="text-slate-300">
            {gpu.powerDraw.toFixed(1)} / {gpu.powerLimit.toFixed(1)} W
          </div>
          <div className="w-full h-1 bg-slate-700 rounded mt-1">
            <div
              className="h-full rounded bg-blue-500"
              style={{ width: `${Math.min(powerPercent, 100)}%` }}
            />
          </div>
        </div>
      </div>

      <div className="flex items-center justify-between">
        <div className={`flex items-center gap-1 ${getUtilColor(gpu.utilization)}`}>
          <Activity className="w-3 h-3" />
          <span>利用率 {gpu.utilization}%</span>
        </div>
        <div className="text-slate-500">
          空闲 {formatMB(gpu.memoryFree)}
        </div>
      </div>
    </motion.div>
  );
};
