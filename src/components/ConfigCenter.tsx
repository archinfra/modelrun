import React, { useEffect, useMemo, useState } from 'react';
import {
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  Database,
  FileJson,
  FileText,
  Pencil,
  Plus,
  RefreshCw,
  Settings2,
  ShieldAlert,
  Trash2,
  Wrench,
} from 'lucide-react';
import { requestJSON } from '../lib/api';
import { ActionTemplate, BackendLogEntry, BootstrapConfig, PipelineStepTemplate, SystemStatus } from '../types';

type ActionTemplateDraft = Partial<ActionTemplate> & {
  fieldsText: string;
  tagsText: string;
};

type BootstrapDraft = Partial<BootstrapConfig> & {
  defaultArgsText: string;
};

type PipelineStepDraft = Partial<PipelineStepTemplate> & {
  detailsText: string;
};

type ConfigTab = 'actions' | 'bootstrap' | 'pipeline' | 'logs' | 'status';

const emptyActionDraft = (): ActionTemplateDraft => ({
  name: '',
  description: '',
  category: 'runtime',
  executionType: 'command',
  commandTemplate: '',
  scriptUrl: '',
  scriptArgsTemplate: '',
  builtIn: false,
  fieldsText: '[]',
  tagsText: '',
});

const emptyBootstrapDraft = (): BootstrapDraft => ({
  name: '',
  description: '',
  serviceType: '',
  category: 'runtime',
  actionTemplateId: '',
  endpoint: '',
  port: 0,
  builtIn: false,
  defaultArgsText: '{}',
});

const emptyPipelineStepDraft = (): PipelineStepDraft => ({
  framework: 'vllm-ascend',
  stepId: 'custom_step',
  name: '',
  description: '',
  commandTemplate: '',
  previewTemplate: '',
  builtIn: false,
  detailsText: '[]',
});

const stepTemplatePlaceholders = [
  '{{checkModelTargetCommand}} / {{checkModelTargetPreview}}',
  '{{prepareModelFetcherCommand}} / {{prepareModelFetcherPreview}}',
  '{{syncModelCommand}} / {{syncModelPreview}}',
  '{{pullImageCommand}} / {{pullImagePreview}}',
  '{{launchRuntimeCommand}} / {{launchRuntimePreview}}',
  '{{verifyServiceCommand}} / {{verifyServicePreview}}',
  '{{imageRef}} / {{imageRefQuoted}}',
  '{{containerName}} / {{containerNameQuoted}}',
  '{{modelHostPath}} / {{modelHostPathQuoted}}',
  '{{workDir}} / {{workDirQuoted}}',
  '{{cacheDir}} / {{cacheDirQuoted}}',
  '{{serverHost}} / {{serverHostQuoted}}',
  '{{modelId}} / {{modelIdQuoted}}',
  '{{apiPort}} / {{apiPortQuoted}}',
];

const parseJSONText = <T,>(value: string, fallback: T): T => {
  if (!value.trim()) return fallback;
  return JSON.parse(value) as T;
};

const statusPill = (enabled: boolean) =>
  enabled
    ? 'bg-amber-100 text-amber-700 border-amber-200'
    : 'bg-emerald-100 text-emerald-700 border-emerald-200';

export const ConfigCenter: React.FC = () => {
  const [systemStatus, setSystemStatus] = useState<SystemStatus | null>(null);
  const [actions, setActions] = useState<ActionTemplate[]>([]);
  const [bootstraps, setBootstraps] = useState<BootstrapConfig[]>([]);
  const [pipelineSteps, setPipelineSteps] = useState<PipelineStepTemplate[]>([]);
  const [logs, setLogs] = useState<BackendLogEntry[]>([]);
  const [activeTab, setActiveTab] = useState<ConfigTab>('actions');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [expandedActionId, setExpandedActionId] = useState<string | null>(null);
  const [expandedBootstrapId, setExpandedBootstrapId] = useState<string | null>(null);
  const [expandedPipelineStepId, setExpandedPipelineStepId] = useState<string | null>(null);
  const [showActionModal, setShowActionModal] = useState(false);
  const [showBootstrapModal, setShowBootstrapModal] = useState(false);
  const [showPipelineStepModal, setShowPipelineStepModal] = useState(false);
  const [editingAction, setEditingAction] = useState<ActionTemplate | null>(null);
  const [editingBootstrap, setEditingBootstrap] = useState<BootstrapConfig | null>(null);
  const [editingPipelineStep, setEditingPipelineStep] = useState<PipelineStepTemplate | null>(null);
  const [actionDraft, setActionDraft] = useState<ActionTemplateDraft>(emptyActionDraft);
  const [bootstrapDraft, setBootstrapDraft] = useState<BootstrapDraft>(emptyBootstrapDraft);
  const [pipelineStepDraft, setPipelineStepDraft] = useState<PipelineStepDraft>(emptyPipelineStepDraft);

  const loadBase = async () => {
    const [status, actionItems, bootstrapItems, pipelineStepItems] = await Promise.all([
      requestJSON<SystemStatus>('/api/system/status'),
      requestJSON<ActionTemplate[]>('/api/action-templates'),
      requestJSON<BootstrapConfig[]>('/api/bootstrap-configs'),
      requestJSON<PipelineStepTemplate[]>('/api/pipeline-step-templates'),
    ]);
    setSystemStatus(status);
    setActions(Array.isArray(actionItems) ? actionItems : []);
    setBootstraps(Array.isArray(bootstrapItems) ? bootstrapItems : []);
    setPipelineSteps(Array.isArray(pipelineStepItems) ? pipelineStepItems : []);
  };

  const loadLogs = async () => {
    const logItems = await requestJSON<BackendLogEntry[]>('/api/backend-logs?limit=200');
    setLogs(Array.isArray(logItems) ? logItems : []);
  };

  useEffect(() => {
    let active = true;

    loadBase()
      .then(() => {
        if (!active) return;
        setError('');
      })
      .catch((err) => {
        if (!active) return;
        setError(err instanceof Error ? err.message : 'Failed to load config center data');
      })
      .finally(() => {
        if (active) {
          setLoading(false);
        }
      });

    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    if (activeTab !== 'logs') return;
    void loadLogs().catch(() => undefined);
    const timer = window.setInterval(() => {
      void loadLogs().catch(() => undefined);
    }, 5000);
    return () => window.clearInterval(timer);
  }, [activeTab]);

  const actionOptions = useMemo(
    () => actions.map((action) => ({ value: action.id, label: action.name })),
    [actions]
  );
  const actionNameById = useMemo(
    () => Object.fromEntries(actions.map((action) => [action.id, action.name])),
    [actions]
  );
  const reversedLogs = useMemo(() => logs.slice().reverse(), [logs]);
  const tabs: Array<{ id: ConfigTab; label: string; count?: number }> = useMemo(
    () => [
      { id: 'actions', label: '动作模板', count: actions.length },
      { id: 'bootstrap', label: '服务初始化', count: bootstraps.length },
      { id: 'pipeline', label: '流水线脚本', count: pipelineSteps.length },
      { id: 'logs', label: '后端日志', count: logs.length },
      { id: 'status', label: '数据状态' },
    ],
    [actions.length, bootstraps.length, logs.length, pipelineSteps.length]
  );

  const openCreateAction = () => {
    setEditingAction(null);
    setActionDraft(emptyActionDraft());
    setShowActionModal(true);
  };

  const openEditAction = (action: ActionTemplate) => {
    setEditingAction(action);
    setActionDraft({
      ...action,
      fieldsText: JSON.stringify(action.fields || [], null, 2),
      tagsText: (action.tags || []).join(', '),
    });
    setShowActionModal(true);
  };

  const openCreateBootstrap = () => {
    setEditingBootstrap(null);
    setBootstrapDraft({
      ...emptyBootstrapDraft(),
      actionTemplateId: actionOptions[0]?.value || '',
    });
    setShowBootstrapModal(true);
  };

  const openEditBootstrap = (item: BootstrapConfig) => {
    setEditingBootstrap(item);
    setBootstrapDraft({
      ...item,
      defaultArgsText: JSON.stringify(item.defaultArgs || {}, null, 2),
    });
    setShowBootstrapModal(true);
  };

  const openCreatePipelineStep = () => {
    setEditingPipelineStep(null);
    setPipelineStepDraft(emptyPipelineStepDraft());
    setShowPipelineStepModal(true);
  };

  const openEditPipelineStep = (item: PipelineStepTemplate) => {
    setEditingPipelineStep(item);
    setPipelineStepDraft({
      ...item,
      detailsText: JSON.stringify(item.details || [], null, 2),
    });
    setShowPipelineStepModal(true);
  };

  const handleSaveAction = async (event: React.FormEvent) => {
    event.preventDefault();
    try {
      const payload = {
        name: actionDraft.name?.trim(),
        description: actionDraft.description?.trim(),
        category: actionDraft.category?.trim(),
        executionType: actionDraft.executionType,
        commandTemplate: actionDraft.commandTemplate || '',
        scriptUrl: actionDraft.scriptUrl || '',
        scriptArgsTemplate: actionDraft.scriptArgsTemplate || '',
        builtIn: Boolean(actionDraft.builtIn),
        fields: parseJSONText(actionDraft.fieldsText, []),
        tags: actionDraft.tagsText
          .split(',')
          .map((item) => item.trim())
          .filter(Boolean),
      };

      if (editingAction) {
        await requestJSON<ActionTemplate>(`/api/action-templates/${editingAction.id}`, {
          method: 'PATCH',
          body: JSON.stringify(payload),
        });
        setNotice(`已更新动作模板 ${payload.name}`);
      } else {
        await requestJSON<ActionTemplate>('/api/action-templates', {
          method: 'POST',
          body: JSON.stringify(payload),
        });
        setNotice(`已新增动作模板 ${payload.name}`);
      }

      setShowActionModal(false);
      await loadBase();
      setError('');
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存动作模板失败');
    }
  };

  const handleSaveBootstrap = async (event: React.FormEvent) => {
    event.preventDefault();
    try {
      const payload = {
        name: bootstrapDraft.name?.trim(),
        description: bootstrapDraft.description?.trim(),
        serviceType: bootstrapDraft.serviceType?.trim(),
        category: bootstrapDraft.category?.trim(),
        actionTemplateId: bootstrapDraft.actionTemplateId,
        endpoint: bootstrapDraft.endpoint?.trim(),
        port: bootstrapDraft.port || 0,
        builtIn: Boolean(bootstrapDraft.builtIn),
        defaultArgs: parseJSONText(bootstrapDraft.defaultArgsText, {}),
      };

      if (editingBootstrap) {
        await requestJSON<BootstrapConfig>(`/api/bootstrap-configs/${editingBootstrap.id}`, {
          method: 'PATCH',
          body: JSON.stringify(payload),
        });
        setNotice(`已更新服务初始化项 ${payload.name}`);
      } else {
        await requestJSON<BootstrapConfig>('/api/bootstrap-configs', {
          method: 'POST',
          body: JSON.stringify(payload),
        });
        setNotice(`已新增服务初始化项 ${payload.name}`);
      }

      setShowBootstrapModal(false);
      await loadBase();
      setError('');
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存服务初始化项失败');
    }
  };

  const handleDeleteAction = async (item: ActionTemplate) => {
    if (!window.confirm(`删除动作模板“${item.name}”？`)) return;
    try {
      await requestJSON<void>(`/api/action-templates/${item.id}`, { method: 'DELETE' });
      setNotice(`已删除动作模板 ${item.name}`);
      await loadBase();
      setError('');
    } catch (err) {
      setError(err instanceof Error ? err.message : '删除动作模板失败');
    }
  };

  const handleDeleteBootstrap = async (item: BootstrapConfig) => {
    if (!window.confirm(`删除服务初始化项“${item.name}”？`)) return;
    try {
      await requestJSON<void>(`/api/bootstrap-configs/${item.id}`, { method: 'DELETE' });
      setNotice(`已删除服务初始化项 ${item.name}`);
      await loadBase();
      setError('');
    } catch (err) {
      setError(err instanceof Error ? err.message : '删除服务初始化项失败');
    }
  };

  const handleSavePipelineStep = async (event: React.FormEvent) => {
    event.preventDefault();
    try {
      const payload = {
        framework: pipelineStepDraft.framework?.trim(),
        stepId: pipelineStepDraft.stepId?.trim(),
        name: pipelineStepDraft.name?.trim(),
        description: pipelineStepDraft.description?.trim(),
        commandTemplate: pipelineStepDraft.commandTemplate || '',
        previewTemplate: pipelineStepDraft.previewTemplate || '',
        builtIn: Boolean(pipelineStepDraft.builtIn),
        optional: Boolean(pipelineStepDraft.optional),
        autoManaged: Boolean(pipelineStepDraft.autoManaged),
        details: parseJSONText(pipelineStepDraft.detailsText, []),
      };

      if (editingPipelineStep) {
        await requestJSON<PipelineStepTemplate>(`/api/pipeline-step-templates/${editingPipelineStep.id}`, {
          method: 'PATCH',
          body: JSON.stringify(payload),
        });
        setNotice(`已更新流水线步骤脚本 ${payload.framework}/${payload.stepId}`);
      } else {
        await requestJSON<PipelineStepTemplate>('/api/pipeline-step-templates', {
          method: 'POST',
          body: JSON.stringify(payload),
        });
        setNotice(`已新增流水线步骤脚本 ${payload.framework}/${payload.stepId}`);
      }

      setShowPipelineStepModal(false);
      await loadBase();
      setError('');
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存流水线步骤脚本失败');
    }
  };

  const handleDeletePipelineStep = async (item: PipelineStepTemplate) => {
    if (!window.confirm(`删除流水线步骤脚本“${item.framework}/${item.stepId}”吗？`)) return;
    try {
      await requestJSON<void>(`/api/pipeline-step-templates/${item.id}`, { method: 'DELETE' });
      setNotice(`已删除流水线步骤脚本 ${item.framework}/${item.stepId}`);
      await loadBase();
      setError('');
    } catch (err) {
      setError(err instanceof Error ? err.message : '删除流水线步骤脚本失败');
    }
  };

  return (
    <div className="space-y-8">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Config Center</h1>
          <p className="text-slate-500 mt-1">统一查看真实数据来源、默认服务初始化项、以及可编辑的内置动作模板。</p>
        </div>
        <button
          onClick={() => {
            setLoading(true);
            Promise.all([loadBase(), activeTab === 'logs' ? loadLogs() : Promise.resolve()])
              .then(() => setError(''))
              .catch((err) => setError(err instanceof Error ? err.message : '刷新失败'))
              .finally(() => setLoading(false));
          }}
          className="flex items-center gap-2 px-4 py-2.5 bg-white border border-slate-200 rounded-xl text-slate-700 hover:bg-slate-50 transition-colors"
        >
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          刷新
        </button>
      </div>

      {(notice || error) && (
        <div
          className={`rounded-2xl border px-4 py-3 text-sm ${
            error ? 'border-red-200 bg-red-50 text-red-700' : 'border-emerald-200 bg-emerald-50 text-emerald-700'
          }`}
        >
          {error || notice}
        </div>
      )}

      <div className="flex flex-wrap gap-2">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            type="button"
            onClick={() => setActiveTab(tab.id)}
            className={`inline-flex items-center gap-2 rounded-lg border px-4 py-2 text-sm transition-colors ${
              activeTab === tab.id
                ? 'border-blue-500 bg-blue-50 text-blue-700'
                : 'border-slate-200 bg-white text-slate-600 hover:bg-slate-50'
            }`}
          >
            <span>{tab.label}</span>
            {typeof tab.count === 'number' ? <span className="rounded bg-white/80 px-2 py-0.5 text-xs">{tab.count}</span> : null}
          </button>
        ))}
      </div>

      {activeTab === 'status' && systemStatus && (
        <section className="grid gap-4 lg:grid-cols-[1.1fr,0.9fr]">
          <div className="bg-white border border-slate-200 rounded-2xl p-6">
            <div className="flex items-center gap-3 mb-4">
              <Database className="w-5 h-5 text-blue-600" />
              <h2 className="text-lg font-semibold text-slate-900">Data Source Status</h2>
            </div>
            <div className="space-y-3 text-sm">
              <div className="flex items-center justify-between gap-3">
                <span className="text-slate-500">Storage driver</span>
                <span className="font-medium text-slate-900">{systemStatus.storage.driver}</span>
              </div>
              <div className="flex items-center justify-between gap-3">
                <span className="text-slate-500">SQLite path</span>
                <code className="text-xs bg-slate-100 px-2 py-1 rounded-lg text-slate-700">{systemStatus.storage.path}</code>
              </div>
              <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3 mt-4">
                {Object.entries(systemStatus.storage.counts).map(([key, value]) => (
                  <div key={key} className="rounded-2xl bg-slate-50 border border-slate-200 p-4">
                    <div className="text-sm text-slate-500">{key}</div>
                    <div className="text-2xl font-semibold text-slate-900 mt-1">{value}</div>
                  </div>
                ))}
              </div>
            </div>
          </div>

          <div className="bg-white border border-slate-200 rounded-2xl p-6 space-y-4">
            <div className="flex items-center gap-3">
              <ShieldAlert className="w-5 h-5 text-amber-600" />
              <h2 className="text-lg font-semibold text-slate-900">Mock & Demo</h2>
            </div>
            <div className="rounded-2xl border border-slate-200 p-4">
              <div className="flex items-center justify-between gap-3">
                <div>
                  <div className="font-medium text-slate-900">Remote collect mock</div>
                  <div className="text-sm text-slate-500 mt-1">{systemStatus.mock.description}</div>
                </div>
                <span className={`px-3 py-1 rounded-full border text-xs font-medium ${statusPill(systemStatus.mock.fakeConnectEnabled)}`}>
                  {systemStatus.mock.fakeConnectEnabled ? 'enabled' : 'disabled'}
                </span>
              </div>
              <div className="text-xs text-slate-500 mt-3">
                关闭 mock 时，确保后端运行环境里不要设置 <code>{systemStatus.mock.toggleEnv}</code>。
              </div>
            </div>

            <div className="space-y-3">
              {systemStatus.demoFeatures.map((item) => (
                <div key={item.key} className="rounded-2xl border border-slate-200 p-4">
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <div className="font-medium text-slate-900">{item.name}</div>
                      <div className="text-sm text-slate-500 mt-1">{item.description}</div>
                    </div>
                    <span className={`px-3 py-1 rounded-full border text-xs font-medium ${item.persisted ? 'bg-emerald-100 text-emerald-700 border-emerald-200' : 'bg-amber-100 text-amber-700 border-amber-200'}`}>
                      {item.persisted ? 'persisted' : 'demo'}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </section>
      )}

      {activeTab === 'bootstrap' && <section className="bg-white border border-slate-200 rounded-2xl p-6 space-y-4">
        <div className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <Settings2 className="w-5 h-5 text-indigo-600" />
            <div>
              <h2 className="text-lg font-semibold text-slate-900">Service Bootstrap</h2>
              <p className="text-sm text-slate-500">node_exporter、npu_exporter、modelscope CLI 之类的服务初始化项。</p>
            </div>
          </div>
          <button
            onClick={openCreateBootstrap}
            className="flex items-center gap-2 px-4 py-2.5 bg-indigo-600 text-white rounded-xl hover:bg-indigo-700 transition-colors"
          >
            <Plus className="w-4 h-4" />
            新增初始化项
          </button>
        </div>

        <div className="space-y-3">
          {bootstraps.map((item) => {
            const expanded = expandedBootstrapId === item.id;
            const actionName = actionNameById[item.actionTemplateId] || item.actionTemplateId;
            return (
              <div key={item.id} className="border border-slate-200 rounded-2xl overflow-hidden">
                <button
                  type="button"
                  onClick={() => setExpandedBootstrapId(expanded ? null : item.id)}
                  className="w-full px-5 py-4 flex items-start justify-between gap-4 hover:bg-slate-50 transition-colors"
                >
                  <div className="text-left min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-medium text-slate-900">{item.name}</span>
                      {item.builtIn && (
                        <span className="px-2.5 py-1 rounded-full bg-slate-100 text-slate-600 text-xs">built-in</span>
                      )}
                      <span className="px-2.5 py-1 rounded-full bg-blue-50 text-blue-700 text-xs">{item.serviceType}</span>
                    </div>
                    <div className="text-sm text-slate-500 mt-2">{item.description}</div>
                  </div>
                  {expanded ? <ChevronDown className="w-5 h-5 text-slate-400" /> : <ChevronRight className="w-5 h-5 text-slate-400" />}
                </button>
                {expanded && (
                  <div className="border-t border-slate-200 px-5 py-4 bg-slate-50 space-y-3 text-sm">
                    <div className="grid gap-3 md:grid-cols-2">
                      <KV label="Action template" value={actionName} />
                      <KV label="Endpoint" value={item.endpoint || '-'} />
                      <KV label="Port" value={item.port ? `${item.port}` : '-'} />
                      <KV label="Category" value={item.category || '-'} />
                    </div>
                    <pre className="text-xs bg-slate-900 text-slate-100 rounded-xl p-4 overflow-x-auto">{JSON.stringify(item.defaultArgs || {}, null, 2)}</pre>
                    <div className="flex gap-3">
                      <button
                        onClick={() => openEditBootstrap(item)}
                        className="inline-flex items-center gap-2 px-3 py-2 bg-white border border-slate-200 rounded-xl text-slate-700 hover:bg-slate-100 transition-colors"
                      >
                        <Pencil className="w-4 h-4" />
                        编辑
                      </button>
                      <button
                        onClick={() => handleDeleteBootstrap(item)}
                        className="inline-flex items-center gap-2 px-3 py-2 bg-white border border-red-200 rounded-xl text-red-700 hover:bg-red-50 transition-colors"
                      >
                        <Trash2 className="w-4 h-4" />
                        删除
                      </button>
                    </div>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </section>}

      {activeTab === 'pipeline' && <section className="bg-white border border-slate-200 rounded-2xl p-6 space-y-4">
        <div className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <FileText className="w-5 h-5 text-violet-600" />
            <div>
              <h2 className="text-lg font-semibold text-slate-900">Pipeline Step Scripts</h2>
              <p className="text-sm text-slate-500">部署步骤不再写死在后端代码里。这里可以直接编辑每一步真正执行的脚本模板。</p>
            </div>
          </div>
          <button
            onClick={openCreatePipelineStep}
            className="flex items-center gap-2 px-4 py-2.5 bg-violet-600 text-white rounded-xl hover:bg-violet-700 transition-colors"
          >
            <Plus className="w-4 h-4" />
            新增步骤脚本
          </button>
        </div>

        <div className="rounded-2xl border border-slate-200 bg-slate-50 p-4 text-sm text-slate-600">
          <div className="font-medium text-slate-900 mb-2">可用占位符</div>
          <div className="flex flex-wrap gap-2">
            {stepTemplatePlaceholders.map((item) => (
              <code key={item} className="rounded-lg bg-white px-2 py-1 text-xs text-slate-700 border border-slate-200">
                {item}
              </code>
            ))}
          </div>
        </div>

        <div className="space-y-3">
          {pipelineSteps.map((item) => {
            const expanded = expandedPipelineStepId === item.id;
            return (
              <div key={item.id} className="border border-slate-200 rounded-2xl overflow-hidden">
                <button
                  type="button"
                  onClick={() => setExpandedPipelineStepId(expanded ? null : item.id)}
                  className="w-full px-5 py-4 flex items-start justify-between gap-4 hover:bg-slate-50 transition-colors"
                >
                  <div className="text-left min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-medium text-slate-900">{item.name}</span>
                      {item.builtIn && <span className="px-2.5 py-1 rounded-full bg-slate-100 text-slate-600 text-xs">built-in</span>}
                      <span className="px-2.5 py-1 rounded-full bg-violet-50 text-violet-700 text-xs">{item.framework}</span>
                      <span className="px-2.5 py-1 rounded-full bg-slate-100 text-slate-600 text-xs">{item.stepId}</span>
                    </div>
                    <div className="text-sm text-slate-500 mt-2">{item.description}</div>
                  </div>
                  {expanded ? <ChevronDown className="w-5 h-5 text-slate-400" /> : <ChevronRight className="w-5 h-5 text-slate-400" />}
                </button>
                {expanded && (
                  <div className="border-t border-slate-200 px-5 py-4 bg-slate-50 space-y-3">
                    <div className="grid gap-3 md:grid-cols-2 text-sm">
                      <KV label="Framework" value={item.framework} />
                      <KV label="Step ID" value={item.stepId} />
                      <KV label="Auto Managed" value={item.autoManaged ? 'yes' : 'no'} />
                      <KV label="Optional" value={item.optional ? 'yes' : 'no'} />
                    </div>
                    {item.details?.length ? (
                      <div className="rounded-2xl border border-slate-200 bg-white p-4">
                        <div className="text-sm font-medium text-slate-900 mb-2">Step Details</div>
                        <pre className="text-xs bg-slate-900 text-slate-100 rounded-xl p-4 overflow-x-auto whitespace-pre-wrap break-all">{JSON.stringify(item.details, null, 2)}</pre>
                      </div>
                    ) : null}
                    <div className="grid gap-3 lg:grid-cols-2">
                      <div className="rounded-2xl border border-slate-200 bg-white p-4">
                        <div className="text-sm font-medium text-slate-900 mb-2">Preview Template</div>
                        <pre className="text-xs bg-slate-900 text-slate-100 rounded-xl p-4 overflow-x-auto whitespace-pre-wrap break-all">{item.previewTemplate || item.commandTemplate}</pre>
                      </div>
                      <div className="rounded-2xl border border-slate-200 bg-white p-4">
                        <div className="text-sm font-medium text-slate-900 mb-2">Command Template</div>
                        <pre className="text-xs bg-slate-900 text-slate-100 rounded-xl p-4 overflow-x-auto whitespace-pre-wrap break-all">{item.commandTemplate}</pre>
                      </div>
                    </div>
                    <div className="flex gap-3">
                      <button
                        onClick={() => openEditPipelineStep(item)}
                        className="inline-flex items-center gap-2 px-3 py-2 bg-white border border-slate-200 rounded-xl text-slate-700 hover:bg-slate-100 transition-colors"
                      >
                        <Pencil className="w-4 h-4" />
                        编辑
                      </button>
                      <button
                        onClick={() => handleDeletePipelineStep(item)}
                        className="inline-flex items-center gap-2 px-3 py-2 bg-white border border-red-200 rounded-xl text-red-700 hover:bg-red-50 transition-colors"
                      >
                        <Trash2 className="w-4 h-4" />
                        删除
                      </button>
                    </div>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </section>}

      {activeTab === 'actions' && <section className="bg-white border border-slate-200 rounded-2xl p-6 space-y-4">
        <div className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <Wrench className="w-5 h-5 text-blue-600" />
            <div>
              <h2 className="text-lg font-semibold text-slate-900">Action Templates</h2>
              <p className="text-sm text-slate-500">远程任务和服务初始化可以共用的内置动作模板。</p>
            </div>
          </div>
          <button
            onClick={openCreateAction}
            className="flex items-center gap-2 px-4 py-2.5 bg-blue-600 text-white rounded-xl hover:bg-blue-700 transition-colors"
          >
            <Plus className="w-4 h-4" />
            新增动作模板
          </button>
        </div>

        <div className="space-y-3">
          {actions.map((item) => {
            const expanded = expandedActionId === item.id;
            return (
              <div key={item.id} className="border border-slate-200 rounded-2xl overflow-hidden">
                <button
                  type="button"
                  onClick={() => setExpandedActionId(expanded ? null : item.id)}
                  className="w-full px-5 py-4 flex items-start justify-between gap-4 hover:bg-slate-50 transition-colors"
                >
                  <div className="text-left min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-medium text-slate-900">{item.name}</span>
                      {item.builtIn && (
                        <span className="px-2.5 py-1 rounded-full bg-slate-100 text-slate-600 text-xs">built-in</span>
                      )}
                      <span className="px-2.5 py-1 rounded-full bg-indigo-50 text-indigo-700 text-xs">{item.executionType}</span>
                    </div>
                    <div className="text-sm text-slate-500 mt-2">{item.description}</div>
                  </div>
                  {expanded ? <ChevronDown className="w-5 h-5 text-slate-400" /> : <ChevronRight className="w-5 h-5 text-slate-400" />}
                </button>
                {expanded && (
                  <div className="border-t border-slate-200 px-5 py-4 bg-slate-50 space-y-3">
                    <div className="grid gap-3 md:grid-cols-2 text-sm">
                      <KV label="Category" value={item.category || '-'} />
                      <KV label="Tags" value={(item.tags || []).join(', ') || '-'} />
                    </div>
                    <pre className="text-xs bg-slate-900 text-slate-100 rounded-xl p-4 overflow-x-auto whitespace-pre-wrap break-all">
                      {item.executionType === 'command' ? item.commandTemplate || '' : item.scriptUrl || ''}
                    </pre>
                    {item.fields?.length ? (
                      <div className="rounded-2xl border border-slate-200 bg-white p-4">
                        <div className="flex items-center gap-2 text-sm font-medium text-slate-900 mb-3">
                          <FileJson className="w-4 h-4 text-slate-500" />
                          参数字段
                        </div>
                        <pre className="text-xs bg-slate-900 text-slate-100 rounded-xl p-4 overflow-x-auto">{JSON.stringify(item.fields, null, 2)}</pre>
                      </div>
                    ) : null}
                    <div className="flex gap-3">
                      <button
                        onClick={() => openEditAction(item)}
                        className="inline-flex items-center gap-2 px-3 py-2 bg-white border border-slate-200 rounded-xl text-slate-700 hover:bg-slate-100 transition-colors"
                      >
                        <Pencil className="w-4 h-4" />
                        编辑
                      </button>
                      <button
                        onClick={() => handleDeleteAction(item)}
                        className="inline-flex items-center gap-2 px-3 py-2 bg-white border border-red-200 rounded-xl text-red-700 hover:bg-red-50 transition-colors"
                      >
                        <Trash2 className="w-4 h-4" />
                        删除
                      </button>
                    </div>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </section>}

      {activeTab === 'logs' && <section className="bg-white border border-slate-200 rounded-2xl p-6 space-y-4">
        <div className="flex items-center gap-3">
          <Database className="w-5 h-5 text-slate-700" />
          <div>
            <h2 className="text-lg font-semibold text-slate-900">Backend Logs</h2>
            <p className="text-sm text-slate-500">查看最近 200 条 Go 后端运行日志。部署步骤的实时输出仍然在部署面板里查看。</p>
          </div>
        </div>
        <div className="rounded-2xl border border-slate-200 bg-slate-950 text-slate-100 p-4 max-h-[28rem] overflow-y-auto font-mono text-xs space-y-2">
          {logs.length === 0 ? (
            <div className="text-slate-400">当前还没有后端日志。</div>
          ) : (
            reversedLogs.map((item, index) => (
              <div key={`${item.timestamp}-${index}`} className="break-all">
                <span className="text-slate-500">[{item.timestamp}]</span>{' '}
                <span className={item.level === 'error' ? 'text-red-300' : item.level === 'warn' ? 'text-amber-300' : 'text-emerald-300'}>
                  {item.level.toUpperCase()}
                </span>{' '}
                {item.component ? <span className="text-blue-300">{item.component}</span> : null}
                <span className="text-slate-100"> {item.message}</span>
              </div>
            ))
          )}
        </div>
      </section>}

      {showActionModal && (
        <Modal title={editingAction ? '编辑动作模板' : '新增动作模板'} onClose={() => setShowActionModal(false)}>
          <form onSubmit={handleSaveAction} className="space-y-4">
            <div className="grid gap-4 md:grid-cols-2">
              <Input label="名称" value={actionDraft.name || ''} onChange={(value) => setActionDraft((current) => ({ ...current, name: value }))} />
              <Input label="分类" value={actionDraft.category || ''} onChange={(value) => setActionDraft((current) => ({ ...current, category: value }))} />
            </div>
            <Textarea label="描述" value={actionDraft.description || ''} onChange={(value) => setActionDraft((current) => ({ ...current, description: value }))} rows={3} />
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1">执行类型</label>
              <select
                value={actionDraft.executionType || 'command'}
                onChange={(event) => setActionDraft((current) => ({ ...current, executionType: event.target.value }))}
                className="w-full px-4 py-2 border border-slate-200 rounded-xl bg-white focus:ring-2 focus:ring-blue-500"
              >
                <option value="command">command</option>
                <option value="script_url">script_url</option>
              </select>
            </div>
            {actionDraft.executionType === 'script_url' ? (
              <>
                <Textarea label="Script URL 模板" value={actionDraft.scriptUrl || ''} onChange={(value) => setActionDraft((current) => ({ ...current, scriptUrl: value }))} rows={3} />
                <Textarea label="Script Args 模板" value={actionDraft.scriptArgsTemplate || ''} onChange={(value) => setActionDraft((current) => ({ ...current, scriptArgsTemplate: value }))} rows={3} />
              </>
            ) : (
              <Textarea label="命令模板" value={actionDraft.commandTemplate || ''} onChange={(value) => setActionDraft((current) => ({ ...current, commandTemplate: value }))} rows={8} />
            )}
            <Textarea label="字段定义 JSON" value={actionDraft.fieldsText} onChange={(value) => setActionDraft((current) => ({ ...current, fieldsText: value }))} rows={6} />
            <Input label="标签（逗号分隔）" value={actionDraft.tagsText} onChange={(value) => setActionDraft((current) => ({ ...current, tagsText: value }))} />
            <label className="flex items-center gap-3 text-sm text-slate-700">
              <input
                type="checkbox"
                checked={Boolean(actionDraft.builtIn)}
                onChange={(event) => setActionDraft((current) => ({ ...current, builtIn: event.target.checked }))}
              />
              标记为 built-in
            </label>
            <ModalActions onClose={() => setShowActionModal(false)} />
          </form>
        </Modal>
      )}

      {showPipelineStepModal && (
        <Modal title={editingPipelineStep ? '编辑流水线步骤脚本' : '新增流水线步骤脚本'} onClose={() => setShowPipelineStepModal(false)}>
          <form onSubmit={handleSavePipelineStep} className="space-y-4">
            <div className="grid gap-4 md:grid-cols-2">
              <Input label="Framework" value={pipelineStepDraft.framework || ''} onChange={(value) => setPipelineStepDraft((current) => ({ ...current, framework: value }))} />
              <Input label="Step ID" value={pipelineStepDraft.stepId || ''} onChange={(value) => setPipelineStepDraft((current) => ({ ...current, stepId: value }))} />
            </div>
            <div className="grid gap-4 md:grid-cols-2">
              <Input label="名称" value={pipelineStepDraft.name || ''} onChange={(value) => setPipelineStepDraft((current) => ({ ...current, name: value }))} />
              <Input label="预览模板" value={pipelineStepDraft.previewTemplate || ''} onChange={(value) => setPipelineStepDraft((current) => ({ ...current, previewTemplate: value }))} />
            </div>
            <Textarea label="描述" value={pipelineStepDraft.description || ''} onChange={(value) => setPipelineStepDraft((current) => ({ ...current, description: value }))} rows={3} />
            <Textarea label="命令模板" value={pipelineStepDraft.commandTemplate || ''} onChange={(value) => setPipelineStepDraft((current) => ({ ...current, commandTemplate: value }))} rows={10} />
            <Textarea label="细节 JSON" value={pipelineStepDraft.detailsText} onChange={(value) => setPipelineStepDraft((current) => ({ ...current, detailsText: value }))} rows={5} />
            <div className="rounded-2xl border border-slate-200 bg-slate-50 p-4 text-sm text-slate-600">
              <div className="font-medium text-slate-900 mb-2">常用占位符</div>
              <div className="flex flex-wrap gap-2">
                {stepTemplatePlaceholders.map((item) => (
                  <code key={item} className="rounded-lg bg-white px-2 py-1 text-xs text-slate-700 border border-slate-200">
                    {item}
                  </code>
                ))}
              </div>
            </div>
            <div className="grid gap-3 md:grid-cols-3">
              <label className="flex items-center gap-3 text-sm text-slate-700">
                <input
                  type="checkbox"
                  checked={Boolean(pipelineStepDraft.optional)}
                  onChange={(event) => setPipelineStepDraft((current) => ({ ...current, optional: event.target.checked }))}
                />
                可选步骤
              </label>
              <label className="flex items-center gap-3 text-sm text-slate-700">
                <input
                  type="checkbox"
                  checked={Boolean(pipelineStepDraft.autoManaged)}
                  onChange={(event) => setPipelineStepDraft((current) => ({ ...current, autoManaged: event.target.checked }))}
                />
                自动托管
              </label>
              <label className="flex items-center gap-3 text-sm text-slate-700">
                <input
                  type="checkbox"
                  checked={Boolean(pipelineStepDraft.builtIn)}
                  onChange={(event) => setPipelineStepDraft((current) => ({ ...current, builtIn: event.target.checked }))}
                />
                标记为 built-in
              </label>
            </div>
            <ModalActions onClose={() => setShowPipelineStepModal(false)} />
          </form>
        </Modal>
      )}

      {showBootstrapModal && (
        <Modal title={editingBootstrap ? '编辑服务初始化项' : '新增服务初始化项'} onClose={() => setShowBootstrapModal(false)}>
          <form onSubmit={handleSaveBootstrap} className="space-y-4">
            <div className="grid gap-4 md:grid-cols-2">
              <Input label="名称" value={bootstrapDraft.name || ''} onChange={(value) => setBootstrapDraft((current) => ({ ...current, name: value }))} />
              <Input label="服务类型" value={bootstrapDraft.serviceType || ''} onChange={(value) => setBootstrapDraft((current) => ({ ...current, serviceType: value }))} />
            </div>
            <Textarea label="描述" value={bootstrapDraft.description || ''} onChange={(value) => setBootstrapDraft((current) => ({ ...current, description: value }))} rows={3} />
            <div className="grid gap-4 md:grid-cols-2">
              <Input label="分类" value={bootstrapDraft.category || ''} onChange={(value) => setBootstrapDraft((current) => ({ ...current, category: value }))} />
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">关联动作模板</label>
                <select
                  value={bootstrapDraft.actionTemplateId || ''}
                  onChange={(event) => setBootstrapDraft((current) => ({ ...current, actionTemplateId: event.target.value }))}
                  className="w-full px-4 py-2 border border-slate-200 rounded-xl bg-white focus:ring-2 focus:ring-blue-500"
                >
                  {actionOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              </div>
            </div>
            <div className="grid gap-4 md:grid-cols-2">
              <Input label="Endpoint" value={bootstrapDraft.endpoint || ''} onChange={(value) => setBootstrapDraft((current) => ({ ...current, endpoint: value }))} />
              <Input
                label="Port"
                type="number"
                value={`${bootstrapDraft.port || 0}`}
                onChange={(value) => setBootstrapDraft((current) => ({ ...current, port: Number(value) || 0 }))}
              />
            </div>
            <Textarea label="默认参数 JSON" value={bootstrapDraft.defaultArgsText} onChange={(value) => setBootstrapDraft((current) => ({ ...current, defaultArgsText: value }))} rows={6} />
            <label className="flex items-center gap-3 text-sm text-slate-700">
              <input
                type="checkbox"
                checked={Boolean(bootstrapDraft.builtIn)}
                onChange={(event) => setBootstrapDraft((current) => ({ ...current, builtIn: event.target.checked }))}
              />
              标记为 built-in
            </label>
            <ModalActions onClose={() => setShowBootstrapModal(false)} />
          </form>
        </Modal>
      )}
    </div>
  );
};

const KV: React.FC<{ label: string; value: string }> = ({ label, value }) => (
  <div className="rounded-2xl border border-slate-200 bg-white px-4 py-3">
    <div className="text-slate-500 text-xs uppercase tracking-wide">{label}</div>
    <div className="text-slate-900 mt-1 break-all">{value}</div>
  </div>
);

const Modal: React.FC<{ title: string; children: React.ReactNode; onClose: () => void }> = ({ title, children, onClose }) => (
  <div className="fixed inset-0 bg-black/50 z-50 p-4 flex items-center justify-center" onClick={onClose}>
    <div className="w-full max-w-3xl max-h-[90vh] overflow-y-auto rounded-3xl bg-white p-6" onClick={(event) => event.stopPropagation()}>
      <div className="flex items-center justify-between gap-4 mb-6">
        <div className="flex items-center gap-3">
          <CheckCircle2 className="w-5 h-5 text-blue-600" />
          <h3 className="text-xl font-semibold text-slate-900">{title}</h3>
        </div>
        <button onClick={onClose} className="text-slate-500 hover:text-slate-700">
          关闭
        </button>
      </div>
      {children}
    </div>
  </div>
);

const ModalActions: React.FC<{ onClose: () => void }> = ({ onClose }) => (
  <div className="flex gap-3 pt-2">
    <button type="button" onClick={onClose} className="flex-1 px-4 py-2.5 border border-slate-200 rounded-xl text-slate-700 hover:bg-slate-50">
      取消
    </button>
    <button type="submit" className="flex-1 px-4 py-2.5 bg-blue-600 text-white rounded-xl hover:bg-blue-700">
      保存
    </button>
  </div>
);

const Input: React.FC<{
  label: string;
  value: string;
  onChange: (value: string) => void;
  type?: string;
}> = ({ label, value, onChange, type = 'text' }) => (
  <div>
    <label className="block text-sm font-medium text-slate-700 mb-1">{label}</label>
    <input
      type={type}
      value={value}
      onChange={(event) => onChange(event.target.value)}
      className="w-full px-4 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500"
    />
  </div>
);

const Textarea: React.FC<{
  label: string;
  value: string;
  onChange: (value: string) => void;
  rows?: number;
}> = ({ label, value, onChange, rows = 4 }) => (
  <div>
    <label className="block text-sm font-medium text-slate-700 mb-1">{label}</label>
    <textarea
      rows={rows}
      value={value}
      onChange={(event) => onChange(event.target.value)}
      className="w-full px-4 py-2 border border-slate-200 rounded-xl font-mono text-sm focus:ring-2 focus:ring-blue-500"
    />
  </div>
);
