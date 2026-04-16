import React, { useEffect, useMemo, useState } from 'react';
import {
  CheckCircle2,
  CircleDashed,
  Play,
  RefreshCw,
  Server,
  Settings2,
  XCircle,
} from 'lucide-react';
import { requestJSON } from '../lib/api';
import { useAppStore } from '../store';
import {
  DeploymentConfig,
  DeploymentTask,
  ModelConfig,
  PipelineTemplate,
  ServerConfig,
} from '../types';

type DraftState = {
  name: string;
  framework: string;
  modelMode: 'existing' | 'custom';
  selectedModelId: string;
  model: Partial<ModelConfig>;
  dockerImage: string;
  dockerTag: string;
  apiPort: number;
  serverIds: string[];
  rayEnabled: boolean;
  rayHeadServerId: string;
  rayNICName: string;
  workDir: string;
  modelDir: string;
  cacheDir: string;
  sharedCacheDir: string;
  extraArgsText: string;
};

const createDraft = (): DraftState => ({
  name: '',
  framework: 'vllm-ascend',
  modelMode: 'existing',
  selectedModelId: '',
  model: { source: 'modelscope', modelId: '', revision: 'main', localPath: '' },
  dockerImage: '',
  dockerTag: '',
  apiPort: 8000,
  serverIds: [],
  rayEnabled: false,
  rayHeadServerId: '',
  rayNICName: '',
  workDir: '',
  modelDir: '',
  cacheDir: '',
  sharedCacheDir: '',
  extraArgsText: '',
});

export const DeployWizard: React.FC = () => {
  const { currentProjectId, projects } = useAppStore();
  const [templates, setTemplates] = useState<PipelineTemplate[]>([]);
  const [models, setModels] = useState<ModelConfig[]>([]);
  const [servers, setServers] = useState<ServerConfig[]>([]);
  const [deployments, setDeployments] = useState<DeploymentConfig[]>([]);
  const [tasks, setTasks] = useState<DeploymentTask[]>([]);
  const [draft, setDraft] = useState<DraftState>(createDraft);
  const [currentDeploymentId, setCurrentDeploymentId] = useState('');
  const [expandedStepId, setExpandedStepId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');

  const currentProject = projects.find((item) => item.id === currentProjectId);
  const selectedTemplate = useMemo(
    () => templates.find((item) => item.framework === draft.framework || item.id === draft.framework),
    [draft.framework, templates]
  );
  const visibleServers = useMemo(
    () => servers.filter((item) => !currentProjectId || item.projectId === currentProjectId),
    [servers, currentProjectId]
  );
  const selectedModel = useMemo(
    () => models.find((item) => item.id === draft.selectedModelId),
    [draft.selectedModelId, models]
  );

  const loadBase = async () => {
    const [templateItems, modelItems, serverItems, deploymentItems] = await Promise.all([
      requestJSON<PipelineTemplate[]>('/api/pipeline-templates'),
      requestJSON<ModelConfig[]>('/api/models'),
      requestJSON<ServerConfig[]>('/api/servers'),
      requestJSON<DeploymentConfig[]>('/api/deployments'),
    ]);
    setTemplates(templateItems || []);
    setModels(modelItems || []);
    setServers(serverItems || []);
    setDeployments(deploymentItems || []);
  };

  useEffect(() => {
    let active = true;
    loadBase()
      .then(() => active && setError(''))
      .catch((err) => active && setError(err instanceof Error ? err.message : 'Failed to load pipeline data'))
      .finally(() => active && setLoading(false));
    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    if (!selectedTemplate) return;
    setDraft((current) => ({
      ...current,
      dockerImage: selectedTemplate.defaultDocker.image,
      dockerTag: selectedTemplate.defaultDocker.tag,
      apiPort: selectedTemplate.defaultPort,
      workDir: selectedTemplate.defaultRuntime.workDir || '',
      modelDir: selectedTemplate.defaultRuntime.modelDir || '',
      cacheDir: selectedTemplate.defaultRuntime.cacheDir || '',
      sharedCacheDir: selectedTemplate.defaultRuntime.sharedCacheDir || '',
    }));
  }, [selectedTemplate?.framework]);

  useEffect(() => {
    if (!currentDeploymentId) return;
    let active = true;
    const poll = async () => {
      const [deploymentItems, taskItems] = await Promise.all([
        requestJSON<DeploymentConfig[]>('/api/deployments'),
        requestJSON<DeploymentTask[]>(`/api/tasks?deploymentId=${encodeURIComponent(currentDeploymentId)}`),
      ]);
      if (!active) return;
      setDeployments(deploymentItems || []);
      setTasks(taskItems || []);
    };
    void poll();
    const timer = window.setInterval(() => void poll(), 3000);
    return () => {
      active = false;
      window.clearInterval(timer);
    };
  }, [currentDeploymentId]);

  const stepCards = useMemo(() => {
    return (selectedTemplate?.steps || []).map((templateStep) => {
      const related = tasks
        .map((task) => ({ serverId: task.serverId, step: task.steps.find((item) => item.id === templateStep.id) }))
        .filter((item) => item.step);
      const statuses = related.map((item) => item.step?.status || 'pending');
      let status = 'pending';
      if (statuses.includes('failed')) status = 'failed';
      else if (statuses.includes('running')) status = 'running';
      else if (statuses.length && statuses.every((item) => item === 'completed')) status = 'completed';
      const progress = related.length
        ? Math.round(related.reduce((sum, item) => sum + (item.step?.progress || 0), 0) / related.length)
        : 0;
      return { templateStep, related, status, progress };
    });
  }, [selectedTemplate?.steps, tasks]);

  const saveDeployment = async (startNow: boolean) => {
    const model = draft.modelMode === 'existing' ? selectedModel : draft.model;
    if (!selectedTemplate) return setError('请选择推理框架模板。');
    if (!draft.name.trim()) return setError('请填写部署名称。');
    if (!model) return setError('请选择模型。');
    if (draft.serverIds.length === 0) return setError('至少选择一台服务器。');
    if (model.source === 'local' && !model.localPath?.trim()) return setError('本地模型必须填写路径。');
    if (model.source !== 'local' && !model.modelId?.trim()) return setError('远端模型必须填写模型 ID。');

    setSubmitting(true);
    setError('');
    try {
      const created = await requestJSON<DeploymentConfig>('/api/deployments', {
        method: 'POST',
        body: JSON.stringify({
          name: draft.name.trim(),
          framework: draft.framework,
          model,
          docker: { image: draft.dockerImage, tag: draft.dockerTag, gpuDevices: 'all', shmSize: '16g', environmentVars: {}, volumes: [], network: 'host', ipc: 'host', privileged: draft.framework !== 'tei' },
          ray: { enabled: draft.rayEnabled, headServerId: draft.rayHeadServerId, nicName: draft.rayNICName, port: 6379, dashboardPort: 8265 },
          runtime: {
            workDir: draft.workDir.trim(),
            modelDir: draft.modelDir.trim(),
            cacheDir: draft.cacheDir.trim(),
            sharedCacheDir: draft.sharedCacheDir.trim(),
            enableAutoRestart: true,
            extraArgs: draft.extraArgsText.split('\n').map((item) => item.trim()).filter(Boolean),
          },
          vllm: selectedTemplate.defaultVllm,
          servers: draft.serverIds,
          apiPort: draft.apiPort,
        }),
      });
      if (startNow) {
        await requestJSON(`/api/deployments/${created.id}/start`, { method: 'POST' });
        setNotice(`已启动流水线 ${created.name}`);
      } else {
        setNotice(`已保存部署 ${created.name}`);
      }
      setCurrentDeploymentId(created.id);
      await loadBase();
      setTasks(startNow ? await requestJSON<DeploymentTask[]>(`/api/tasks?deploymentId=${encodeURIComponent(created.id)}`) : []);
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存部署失败');
      setNotice('');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="space-y-8">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Pipeline Console</h1>
          <p className="text-slate-500 mt-1">{currentProject?.name || '全局工作区'} | 不再需要下一步，整条部署链路在同一页完成。</p>
        </div>
        <div className="flex items-center gap-3">
          <button onClick={() => { setLoading(true); loadBase().finally(() => setLoading(false)); }} className="px-4 py-2.5 bg-white border border-slate-200 rounded-xl text-slate-700 hover:bg-slate-50">
            <span className="inline-flex items-center gap-2"><RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />刷新</span>
          </button>
          <button onClick={() => void saveDeployment(false)} disabled={submitting} className="px-4 py-2.5 bg-white border border-slate-200 rounded-xl text-slate-700 hover:bg-slate-50">保存草稿</button>
          <button onClick={() => void saveDeployment(true)} disabled={submitting} className="px-4 py-2.5 bg-blue-600 text-white rounded-xl hover:bg-blue-700">
            <span className="inline-flex items-center gap-2"><Play className="w-4 h-4" />启动流水线</span>
          </button>
        </div>
      </div>
      {(notice || error) && (
        <div className={`rounded-2xl border px-4 py-3 text-sm ${error ? 'border-red-200 bg-red-50 text-red-700' : 'border-emerald-200 bg-emerald-50 text-emerald-700'}`}>
          {error || notice}
        </div>
      )}
      <div className="grid gap-6 xl:grid-cols-[1.05fr,0.95fr]">
        <div className="space-y-6">
          <Section title="Framework Presets">
            <div className="grid gap-3 md:grid-cols-3">
              {templates.map((template) => (
                <button key={template.id} onClick={() => setDraft((current) => ({ ...current, framework: template.framework }))} className={`rounded-2xl border p-4 text-left ${draft.framework === template.framework ? 'border-blue-500 bg-blue-50' : 'border-slate-200 bg-slate-50 hover:bg-white'}`}>
                  <div className="font-medium text-slate-900">{template.name}</div>
                  <div className="text-sm text-slate-500 mt-2">{template.description}</div>
                </button>
              ))}
            </div>
          </Section>
          <Section title="Deployment Basics">
            <div className="grid gap-4 md:grid-cols-2">
              <Field label="Deployment name" value={draft.name} onChange={(value) => setDraft((current) => ({ ...current, name: value }))} />
              <Field label="API port" type="number" value={`${draft.apiPort}`} onChange={(value) => setDraft((current) => ({ ...current, apiPort: Number(value) || 0 }))} />
            </div>
          </Section>
          <Section title="Model Source">
            <div className="flex gap-3 mb-4">
              <SmallToggle active={draft.modelMode === 'existing'} onClick={() => setDraft((current) => ({ ...current, modelMode: 'existing' }))} label="Use persisted model" />
              <SmallToggle active={draft.modelMode === 'custom'} onClick={() => setDraft((current) => ({ ...current, modelMode: 'custom' }))} label="Custom model" />
            </div>
            {draft.modelMode === 'existing' ? (
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">Persisted model</label>
                <select value={draft.selectedModelId} onChange={(event) => setDraft((current) => ({ ...current, selectedModelId: event.target.value }))} className="w-full px-4 py-2 border border-slate-200 rounded-xl bg-white">
                  <option value="">Select a model</option>
                  {models.map((model) => (
                    <option key={model.id} value={model.id}>{model.name} | {model.source}</option>
                  ))}
                </select>
              </div>
            ) : (
              <div className="grid gap-4 md:grid-cols-2">
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">Source</label>
                  <select value={draft.model.source || 'modelscope'} onChange={(event) => setDraft((current) => ({ ...current, model: { ...current.model, source: event.target.value as ModelConfig['source'] } }))} className="w-full px-4 py-2 border border-slate-200 rounded-xl bg-white">
                    <option value="modelscope">modelscope</option>
                    <option value="huggingface">huggingface</option>
                    <option value="local">local</option>
                  </select>
                </div>
                <Field label={draft.model.source === 'local' ? 'Local path' : 'Model ID'} value={draft.model.source === 'local' ? draft.model.localPath || '' : draft.model.modelId || ''} onChange={(value) => setDraft((current) => ({ ...current, model: current.model.source === 'local' ? { ...current.model, localPath: value } : { ...current.model, modelId: value } }))} />
              </div>
            )}
          </Section>
          <Section title="Target Servers">
            <div className="grid gap-3 md:grid-cols-2">
              {visibleServers.map((server) => (
                <label key={server.id} className="flex items-start gap-3 rounded-2xl border border-slate-200 p-4 bg-slate-50">
                  <input type="checkbox" checked={draft.serverIds.includes(server.id)} onChange={() => setDraft((current) => ({ ...current, serverIds: current.serverIds.includes(server.id) ? current.serverIds.filter((item) => item !== server.id) : [...current.serverIds, server.id] }))} className="mt-1" />
                  <span>
                    <span className="block font-medium text-slate-900">{server.name}</span>
                    <span className="block text-sm text-slate-500 mt-1">{server.host} | {server.status}</span>
                  </span>
                </label>
              ))}
            </div>
          </Section>
          <Section title="Runtime">
            <div className="grid gap-4 md:grid-cols-2">
              <Field label="Image" value={draft.dockerImage} onChange={(value) => setDraft((current) => ({ ...current, dockerImage: value }))} />
              <Field label="Tag" value={draft.dockerTag} onChange={(value) => setDraft((current) => ({ ...current, dockerTag: value }))} />
            </div>
            {selectedTemplate?.supportsRay && (
              <div className="grid gap-4 md:grid-cols-2 mt-4">
                <SmallToggle active={draft.rayEnabled} onClick={() => setDraft((current) => ({ ...current, rayEnabled: !current.rayEnabled }))} label="Enable Ray bootstrap" />
                <Field label="Ray NIC" value={draft.rayNICName} onChange={(value) => setDraft((current) => ({ ...current, rayNICName: value }))} />
              </div>
            )}
            <div className="grid gap-4 md:grid-cols-2 mt-4">
              <Field label="工作目录" value={draft.workDir} onChange={(value) => setDraft((current) => ({ ...current, workDir: value }))} />
              <Field label="模型目录" value={draft.modelDir} onChange={(value) => setDraft((current) => ({ ...current, modelDir: value }))} />
              <Field label="缓存目录" value={draft.cacheDir} onChange={(value) => setDraft((current) => ({ ...current, cacheDir: value }))} />
              <Field label="共享缓存目录（可选）" value={draft.sharedCacheDir} onChange={(value) => setDraft((current) => ({ ...current, sharedCacheDir: value }))} />
            </div>
            <div className="mt-3 rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600">
              下载模型、创建目录和写启动脚本时，系统会先判断当前 SSH 用户是否对这些目录有写权限；如果没有，再自动尝试使用 `sudo -n`。
            </div>
            <div className="mt-4">
              <label className="block text-sm font-medium text-slate-700 mb-1">Extra args (one per line)</label>
              <textarea rows={4} value={draft.extraArgsText} onChange={(event) => setDraft((current) => ({ ...current, extraArgsText: event.target.value }))} className="w-full px-4 py-2 border border-slate-200 rounded-xl font-mono text-sm" />
            </div>
          </Section>
        </div>
        <div className="space-y-6">
          <Section title="Pipeline Board">
            <div className="grid gap-4">
              {stepCards.map(({ templateStep, related, status, progress }) => (
                <div key={templateStep.id} className={`rounded-2xl border p-4 ${status === 'failed' ? 'border-red-200 bg-red-50' : status === 'running' ? 'border-blue-200 bg-blue-50' : status === 'completed' ? 'border-emerald-200 bg-emerald-50' : 'border-slate-200 bg-slate-50'}`}>
                  <button type="button" onClick={() => setExpandedStepId(expandedStepId === templateStep.id ? null : templateStep.id)} className="w-full text-left">
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <div className="font-medium text-slate-900">{templateStep.name}</div>
                        <div className="text-sm text-slate-500 mt-2">{templateStep.description}</div>
                      </div>
                      <StatusIcon status={status} />
                    </div>
                    <div className="mt-3 h-2 rounded-full bg-white overflow-hidden">
                      <div className="h-full rounded-full bg-slate-700/40" style={{ width: `${progress}%` }} />
                    </div>
                  </button>
                  {expandedStepId === templateStep.id && (
                    <div className="mt-4 pt-4 border-t border-slate-200 space-y-3">
                      {(templateStep.details || []).map((detail) => <div key={detail} className="text-sm text-slate-600">{detail}</div>)}
                      {related.length > 0 ? related.map(({ serverId, step }) => (
                        <div key={`${templateStep.id}-${serverId}`} className="rounded-2xl border border-slate-200 bg-white p-4">
                          <div className="font-medium text-slate-900">{visibleServers.find((item) => item.id === serverId)?.name || serverId}</div>
                          {step?.commandPreview && <pre className="mt-3 text-xs bg-slate-900 text-slate-100 rounded-xl p-3 overflow-x-auto whitespace-pre-wrap break-all">{step.commandPreview}</pre>}
                          <pre className="mt-3 text-xs bg-slate-900 text-slate-100 rounded-xl p-3 overflow-x-auto whitespace-pre-wrap break-all max-h-64">{step?.logs?.join('\n') || 'No logs yet.'}</pre>
                        </div>
                      )) : <div className="text-sm text-slate-500">Start the deployment and this stage will show per-server status, commands, and logs.</div>}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </Section>
          <Section title="Recent Deployments">
            <div className="space-y-3">
              {deployments.slice().sort((a, b) => b.createdAt.localeCompare(a.createdAt)).slice(0, 8).map((deployment) => (
                <button key={deployment.id} onClick={() => { setCurrentDeploymentId(deployment.id); setNotice(`Viewing deployment ${deployment.name}`); }} className={`w-full rounded-2xl border p-4 text-left ${currentDeploymentId === deployment.id ? 'border-blue-500 bg-blue-50' : 'border-slate-200 bg-slate-50 hover:bg-white'}`}>
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <div className="font-medium text-slate-900">{deployment.name}</div>
                      <div className="text-sm text-slate-500 mt-1">{deployment.framework} | {deployment.servers.length} servers | port {deployment.apiPort}</div>
                    </div>
                    <span className="text-xs text-slate-600">{deployment.status}</span>
                  </div>
                </button>
              ))}
            </div>
          </Section>
        </div>
      </div>
    </div>
  );
};

const Section: React.FC<{ title: string; children: React.ReactNode }> = ({ title, children }) => (
  <section className="bg-white border border-slate-200 rounded-2xl p-6">
    <div className="flex items-center gap-3 mb-4">
      <div className="w-10 h-10 rounded-xl bg-slate-100 text-slate-700 flex items-center justify-center"><Settings2 className="w-5 h-5" /></div>
      <h2 className="text-lg font-semibold text-slate-900">{title}</h2>
    </div>
    {children}
  </section>
);

const Field: React.FC<{ label: string; value: string; onChange: (value: string) => void; type?: string }> = ({ label, value, onChange, type = 'text' }) => (
  <div>
    <label className="block text-sm font-medium text-slate-700 mb-1">{label}</label>
    <input type={type} value={value} onChange={(event) => onChange(event.target.value)} className="w-full px-4 py-2 border border-slate-200 rounded-xl" />
  </div>
);

const SmallToggle: React.FC<{ active: boolean; onClick: () => void; label: string }> = ({ active, onClick, label }) => (
  <button type="button" onClick={onClick} className={`px-4 py-2 rounded-xl border text-sm ${active ? 'border-blue-500 bg-blue-50 text-blue-700' : 'border-slate-200 bg-white text-slate-600'}`}>{label}</button>
);

const StatusIcon: React.FC<{ status: string }> = ({ status }) => {
  if (status === 'completed') return <CheckCircle2 className="w-5 h-5 text-emerald-600" />;
  if (status === 'running') return <RefreshCw className="w-5 h-5 text-blue-600 animate-spin" />;
  if (status === 'failed') return <XCircle className="w-5 h-5 text-red-600" />;
  return <CircleDashed className="w-5 h-5 text-slate-400" />;
};
