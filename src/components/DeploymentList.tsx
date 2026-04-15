import React, { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import {
  Play, Square, Trash2, Terminal, Activity,
  Cpu, HardDrive, Zap, Globe, ChevronDown, ChevronRight,
  Clock, AlertCircle, CheckCircle2, XCircle, Loader2
} from 'lucide-react';
import { useAppStore } from '../store';

const StatusBadge = ({ status }: { status: string }) => {
  const configs = {
    running: { icon: Activity, color: 'text-emerald-400', bg: 'bg-emerald-500/10', border: 'border-emerald-500/30' },
    deploying: { icon: Loader2, color: 'text-blue-400', bg: 'bg-blue-500/10', border: 'border-blue-500/30' },
    failed: { icon: XCircle, color: 'text-red-400', bg: 'bg-red-500/10', border: 'border-red-500/30' },
    stopped: { icon: Square, color: 'text-amber-400', bg: 'bg-amber-500/10', border: 'border-amber-500/30' },
    draft: { icon: Clock, color: 'text-slate-400', bg: 'bg-slate-500/10', border: 'border-slate-500/30' }
  };
  const cfg = configs[status as keyof typeof configs] || configs.draft;
  const Icon = cfg.icon;
  return (
    <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded text-xs font-medium ${cfg.bg} ${cfg.color} border ${cfg.border}`}>
      <Icon className={`w-3.5 h-3.5 ${status === 'deploying' ? 'animate-spin' : ''}`} />
      {status.toUpperCase()}
    </span>
  );
};

const MetricCard = ({ label, value, unit, icon: Icon, color }: any) => (
  <div className="bg-slate-900/50 border border-slate-700/50 rounded p-3">
    <div className="flex items-center gap-2 text-slate-500 text-xs mb-1">
      <Icon className={`w-3.5 h-3.5 ${color}`} />
      {label}
    </div>
    <div className="text-slate-200 font-mono text-lg">
      {value}<span className="text-slate-500 text-sm ml-1">{unit}</span>
    </div>
  </div>
);

const EndpointRow = ({ endpoint }: { endpoint: any }) => (
  <div className="flex items-center justify-between py-2 px-3 bg-slate-800/30 rounded border border-slate-700/30">
    <div className="flex items-center gap-3">
      <div className={`w-2 h-2 rounded-full ${endpoint.status === 'healthy' ? 'bg-emerald-400' : 'bg-red-400'}`} />
      <code className="text-xs text-slate-400 font-mono">{endpoint.serverId}</code>
      <span className="text-xs text-slate-500">→</span>
      <code className="text-xs text-blue-400 font-mono">{endpoint.url}</code>
    </div>
    <div className="flex items-center gap-4 text-xs">
      <span className="text-slate-500">Latency: <span className="text-slate-300 font-mono">{endpoint.latency}ms</span></span>
      <span className={`px-2 py-0.5 rounded ${endpoint.status === 'healthy' ? 'bg-emerald-500/20 text-emerald-400' : 'bg-red-500/20 text-red-400'}`}>
        {endpoint.status}
      </span>
    </div>
  </div>
);

const DeploymentCard = ({ deployment, isExpanded, onToggle }: any) => {
  const { updateDeployment, removeDeployment } = useAppStore();

  return (
    <motion.div
      layout
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      className="bg-slate-800/40 border border-slate-700/50 rounded-lg overflow-hidden"
    >
      <div
        className="p-4 cursor-pointer hover:bg-slate-800/60 transition-colors"
        onClick={onToggle}
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <button className="text-slate-500 hover:text-slate-300">
              {isExpanded ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
            </button>
            <div>
              <div className="flex items-center gap-3">
                <h3 className="text-slate-200 font-medium">{deployment.name}</h3>
                <StatusBadge status={deployment.status} />
              </div>
              <div className="flex items-center gap-4 mt-1.5 text-xs text-slate-500">
                <span className="flex items-center gap-1"><Cpu className="w-3 h-3" /> {deployment.model.name}</span>
                <span className="flex items-center gap-1"><Globe className="w-3 h-3" /> Port {deployment.apiPort}</span>
                <span className="flex items-center gap-1"><HardDrive className="w-3 h-3" /> {deployment.servers.length} nodes</span>
              </div>
            </div>
          </div>

          <div className="flex items-center gap-2">
            {deployment.status === 'stopped' && (
              <button
                onClick={(e) => { e.stopPropagation(); updateDeployment(deployment.id, { status: 'running' }); }}
                className="p-2 rounded bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20 transition-colors"
              >
                <Play className="w-4 h-4" />
              </button>
            )}
            {deployment.status === 'running' && (
              <button
                onClick={(e) => { e.stopPropagation(); updateDeployment(deployment.id, { status: 'stopped' }); }}
                className="p-2 rounded bg-amber-500/10 text-amber-400 hover:bg-amber-500/20 transition-colors"
              >
                <Square className="w-4 h-4" />
              </button>
            )}
            <button
              onClick={(e) => { e.stopPropagation(); }}
              className="p-2 rounded bg-slate-700/50 text-slate-400 hover:bg-slate-700 hover:text-slate-200 transition-colors"
            >
              <Terminal className="w-4 h-4" />
            </button>
            <button
              onClick={(e) => { e.stopPropagation(); removeDeployment(deployment.id); }}
              className="p-2 rounded bg-red-500/10 text-red-400 hover:bg-red-500/20 transition-colors"
            >
              <Trash2 className="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>

      <AnimatePresence>
        {isExpanded && (
          <motion.div
            initial={{ height: 0 }}
            animate={{ height: 'auto' }}
            exit={{ height: 0 }}
            className="border-t border-slate-700/50"
          >
            <div className="p-4 space-y-4">
              {deployment.metrics && (
                <div className="grid grid-cols-4 gap-3">
                  <MetricCard label="RPS" value={deployment.metrics.tokensPerSecond} unit="tok/s" icon={Zap} color="text-yellow-400" />
                  <MetricCard label="Latency" value={deployment.metrics.avgLatency} unit="ms" icon={Activity} color="text-blue-400" />
                  <MetricCard label="GPU" value={deployment.metrics.gpuUtilization} unit="%" icon={Cpu} color="text-emerald-400" />
                  <MetricCard label="Memory" value={deployment.metrics.memoryUtilization} unit="%" icon={HardDrive} color="text-purple-400" />
                </div>
              )}

              {deployment.endpoints && (
                <div className="space-y-2">
                  <h4 className="text-xs font-medium text-slate-500 uppercase tracking-wider">Endpoints</h4>
                  {deployment.endpoints.map((ep: any) => <EndpointRow key={ep.serverId} endpoint={ep} />)}
                </div>
              )}

              <div className="grid grid-cols-2 gap-4 text-xs">
                <div className="space-y-1">
                  <span className="text-slate-500">Docker Image</span>
                  <code className="text-slate-300 font-mono">{deployment.docker.image}:{deployment.docker.tag}</code>
                </div>
                <div className="space-y-1">
                  <span className="text-slate-500">vLLM Config</span>
                  <code className="text-slate-300 font-mono">TP={deployment.vllm.tensorParallelSize} PP={deployment.vllm.pipelineParallelSize}</code>
                </div>
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  );
};

export const DeploymentList: React.FC = () => {
  const { deployments } = useAppStore();
  const [expandedId, setExpandedId] = useState<string | null>(null);

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-xl font-bold text-slate-100">Deployments</h2>
          <p className="text-sm text-slate-500 mt-1">{deployments.length} active deployments</p>
        </div>
        <div className="flex items-center gap-2 text-xs text-slate-500">
          <span className="flex items-center gap-1"><div className="w-2 h-2 rounded-full bg-emerald-400" /> Running</span>
          <span className="flex items-center gap-1"><div className="w-2 h-2 rounded-full bg-blue-400" /> Deploying</span>
          <span className="flex items-center gap-1"><div className="w-2 h-2 rounded-full bg-red-400" /> Failed</span>
        </div>
      </div>

      <div className="space-y-3">
        {deployments.map((d) => (
          <DeploymentCard
            key={d.id}
            deployment={d}
            isExpanded={expandedId === d.id}
            onToggle={() => setExpandedId(expandedId === d.id ? null : d.id)}
          />
        ))}
      </div>

      {deployments.length === 0 && (
        <div className="text-center py-16 border border-slate-700/50 border-dashed rounded-lg">
          <Terminal className="w-12 h-12 text-slate-600 mx-auto mb-3" />
          <p className="text-slate-500">No deployments found</p>
          <p className="text-xs text-slate-600 mt-1">Create a new deployment from the wizard</p>
        </div>
      )}
    </div>
  );
};
