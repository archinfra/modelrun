import React, { useEffect, useState } from 'react';
import { Activity, Play, RefreshCw, Square, Trash2 } from 'lucide-react';
import { requestJSON } from '../lib/api';
import { DeploymentConfig, DeploymentTask } from '../types';

export const DeploymentList: React.FC = () => {
  const [deployments, setDeployments] = useState<DeploymentConfig[]>([]);
  const [tasks, setTasks] = useState<DeploymentTask[]>([]);
  const [selectedId, setSelectedId] = useState('');
  const [error, setError] = useState('');

  const load = async () => {
    const deploymentItems = await requestJSON<DeploymentConfig[]>('/api/deployments');
    setDeployments(deploymentItems || []);
    if (selectedId) {
      const taskItems = await requestJSON<DeploymentTask[]>(`/api/tasks?deploymentId=${encodeURIComponent(selectedId)}`);
      setTasks(taskItems || []);
    }
  };

  useEffect(() => {
    void load().catch((err) => setError(err instanceof Error ? err.message : 'Failed to load deployments'));
    const timer = window.setInterval(() => void load().catch(() => undefined), 3000);
    return () => window.clearInterval(timer);
  }, [selectedId]);

  const runAction = async (deployment: DeploymentConfig, action: 'start' | 'stop' | 'delete') => {
    try {
      if (action === 'delete') {
        await requestJSON<void>(`/api/deployments/${deployment.id}`, { method: 'DELETE' });
      } else {
        await requestJSON(`/api/deployments/${deployment.id}/${action}`, { method: 'POST' });
      }
      if (selectedId === deployment.id && action === 'delete') {
        setSelectedId('');
        setTasks([]);
      }
      await load();
      setError('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Operation failed');
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Deployments</h1>
          <p className="text-slate-500 mt-1">展示后端真实部署记录和当前流水线任务状态。</p>
        </div>
        <button onClick={() => void load()} className="px-4 py-2.5 bg-white border border-slate-200 rounded-xl text-slate-700 hover:bg-slate-50">
          <span className="inline-flex items-center gap-2"><RefreshCw className="w-4 h-4" />刷新</span>
        </button>
      </div>

      {error && <div className="rounded-2xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div>}

      <div className="grid gap-6 xl:grid-cols-[0.95fr,1.05fr]">
        <section className="bg-white border border-slate-200 rounded-2xl p-6">
          <div className="space-y-3">
            {deployments.map((deployment) => (
              <button
                key={deployment.id}
                onClick={() => setSelectedId(deployment.id)}
                className={`w-full rounded-2xl border p-4 text-left ${selectedId === deployment.id ? 'border-blue-500 bg-blue-50' : 'border-slate-200 bg-slate-50 hover:bg-white'}`}
              >
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="font-medium text-slate-900">{deployment.name}</div>
                    <div className="text-sm text-slate-500 mt-1">
                      {deployment.framework} | {deployment.model.name} | {deployment.servers.length} servers
                    </div>
                  </div>
                  <span className="text-xs text-slate-600">{deployment.status}</span>
                </div>
                <div className="flex gap-2 mt-4">
                  <button type="button" onClick={(event) => { event.stopPropagation(); void runAction(deployment, 'start'); }} className="px-3 py-2 rounded-xl bg-emerald-50 text-emerald-700 border border-emerald-200">
                    <span className="inline-flex items-center gap-2"><Play className="w-4 h-4" />启动</span>
                  </button>
                  <button type="button" onClick={(event) => { event.stopPropagation(); void runAction(deployment, 'stop'); }} className="px-3 py-2 rounded-xl bg-amber-50 text-amber-700 border border-amber-200">
                    <span className="inline-flex items-center gap-2"><Square className="w-4 h-4" />停止</span>
                  </button>
                  <button type="button" onClick={(event) => { event.stopPropagation(); void runAction(deployment, 'delete'); }} className="px-3 py-2 rounded-xl bg-red-50 text-red-700 border border-red-200">
                    <span className="inline-flex items-center gap-2"><Trash2 className="w-4 h-4" />删除</span>
                  </button>
                </div>
              </button>
            ))}
          </div>
        </section>

        <section className="bg-white border border-slate-200 rounded-2xl p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 rounded-xl bg-slate-100 text-slate-700 flex items-center justify-center">
              <Activity className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-lg font-semibold text-slate-900">Pipeline Tasks</h2>
              <p className="text-sm text-slate-500">{selectedId ? `${tasks.length} server task(s)` : 'Select a deployment'}</p>
            </div>
          </div>
          <div className="space-y-4">
            {tasks.map((task) => (
              <div key={task.id} className="rounded-2xl border border-slate-200 bg-slate-50 p-4">
                <div className="font-medium text-slate-900">{task.serverId}</div>
                <div className="text-sm text-slate-500 mt-1">{task.overallProgress}%</div>
                <div className="mt-3 space-y-2">
                  {task.steps.map((step) => (
                    <div key={step.id} className="rounded-xl border border-slate-200 bg-white px-3 py-2">
                      <div className="flex items-center justify-between gap-3">
                        <span className="text-sm font-medium text-slate-900">{step.name}</span>
                        <span className="text-xs text-slate-500">{step.status}</span>
                      </div>
                      {step.commandPreview && <pre className="mt-2 text-xs bg-slate-900 text-slate-100 rounded-lg p-2 overflow-x-auto whitespace-pre-wrap break-all">{step.commandPreview}</pre>}
                      {step.logs?.length ? <pre className="mt-2 text-xs bg-slate-900 text-slate-100 rounded-lg p-2 overflow-x-auto whitespace-pre-wrap break-all max-h-48">{step.logs.join('\n')}</pre> : null}
                    </div>
                  ))}
                </div>
              </div>
            ))}
            {!tasks.length && <div className="text-sm text-slate-500">当前没有任务日志。</div>}
          </div>
        </section>
      </div>
    </div>
  );
};
