import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  CheckCircle2,
  CircleDashed,
  Play,
  RefreshCw,
  RotateCcw,
  Settings2,
  Wand2,
  XCircle,
} from 'lucide-react';
import { requestJSON } from '../lib/api';
import { useAppStore } from '../store';
import { DeploymentConfig, DeploymentStep, DeploymentTask, ModelConfig, PipelineTemplate, ServerConfig } from '../types';

type DraftServerOverride = {
  nodeIp: string;
  visibleDevices: string;
  rayStartArgsText: string;
};

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
  rayPort: number;
  rayDashboardPort: number;
  rayVisibleDevices: string;
  workDir: string;
  modelDir: string;
  cacheDir: string;
  sharedCacheDir: string;
  tensorParallelSize: number;
  pipelineParallelSize: number;
  maxModelLen: number;
  gpuMemoryUtilization: number;
  dtype: string;
  quantization: string;
  trustRemoteCode: boolean;
  enablePrefixCaching: boolean;
  enableExpertParallel: boolean;
  maxNumSeqs: number;
  maxNumBatchedTokens: number;
  extraArgsText: string;
  serverOverrides: Record<string, DraftServerOverride>;
};

const statusLabels: Record<DeploymentConfig['status'], string> = {
  draft: '草稿',
  deploying: '部署中',
  running: '运行中',
  failed: '失败',
  stopped: '已停止',
};

const createDraft = (): DraftState => ({
  name: '',
  framework: 'vllm-ascend',
  modelMode: 'existing',
  selectedModelId: '',
  model: { source: 'modelscope', modelId: '', revision: '', localPath: '' },
  dockerImage: '',
  dockerTag: '',
  apiPort: 8000,
  serverIds: [],
  rayEnabled: false,
  rayHeadServerId: '',
  rayNICName: '',
  rayPort: 6379,
  rayDashboardPort: 8265,
  rayVisibleDevices: '',
  workDir: '',
  modelDir: '',
  cacheDir: '',
  sharedCacheDir: '',
  tensorParallelSize: 1,
  pipelineParallelSize: 1,
  maxModelLen: 4096,
  gpuMemoryUtilization: 0.9,
  dtype: 'auto',
  quantization: '',
  trustRemoteCode: true,
  enablePrefixCaching: true,
  enableExpertParallel: false,
  maxNumSeqs: 256,
  maxNumBatchedTokens: 8192,
  extraArgsText: '',
  serverOverrides: {},
});

const applyTemplateDefaults = (draft: DraftState, template: PipelineTemplate): DraftState => ({
  ...draft,
  framework: template.framework,
  dockerImage: template.defaultDocker.image,
  dockerTag: template.defaultDocker.tag,
  apiPort: template.defaultPort,
  rayEnabled: template.defaultRay.enabled || false,
  rayPort: template.defaultRay.port || 6379,
  rayDashboardPort: template.defaultRay.dashboardPort || 8265,
  rayVisibleDevices: template.defaultRay.visibleDevices || '',
  workDir: template.defaultRuntime.workDir || '',
  modelDir: template.defaultRuntime.modelDir || '',
  cacheDir: template.defaultRuntime.cacheDir || '',
  sharedCacheDir: template.defaultRuntime.sharedCacheDir || '',
  tensorParallelSize: template.defaultVllm.tensorParallelSize || 1,
  pipelineParallelSize: template.defaultVllm.pipelineParallelSize || 1,
  maxModelLen: template.defaultVllm.maxModelLen || 4096,
  gpuMemoryUtilization: template.defaultVllm.gpuMemoryUtilization || 0.9,
  dtype: template.defaultVllm.dtype || 'auto',
  quantization: template.defaultVllm.quantization || '',
  trustRemoteCode: template.defaultVllm.trustRemoteCode || false,
  enablePrefixCaching: template.defaultVllm.enablePrefixCaching || false,
  enableExpertParallel: template.defaultVllm.enableExpertParallel || false,
  maxNumSeqs: template.defaultVllm.maxNumSeqs || 256,
  maxNumBatchedTokens: template.defaultVllm.maxNumBatchedTokens || 8192,
});

const countAccelerators = (server: ServerConfig) => {
  const devices = server.gpuInfo || [];
  const npu = devices.filter((item) => item.type === 'npu').length;
  return npu || devices.length;
};

const isQwen397BaseModel = (value: string) => /^Qwen\/Qwen3\.5-397B-A17B$/i.test(value.trim());

const makeOverride = (server?: ServerConfig, current?: Partial<DraftServerOverride>): DraftServerOverride => ({
  nodeIp: current?.nodeIp ?? server?.host ?? '',
  visibleDevices: current?.visibleDevices ?? '',
  rayStartArgsText: current?.rayStartArgsText ?? '',
});

const deploymentToDraft = (deployment: DeploymentConfig, models: ModelConfig[], servers: ServerConfig[]): DraftState => {
  const matchedModel = models.find((item) => item.id === deployment.model.id);
  const overrides = Object.fromEntries(
    deployment.servers.map((serverId) => {
      const server = servers.find((item) => item.id === serverId);
      const override = deployment.serverOverrides?.find((item) => item.serverId === serverId);
      return [serverId, makeOverride(server, {
        nodeIp: override?.nodeIp || server?.host || '',
        visibleDevices: override?.visibleDevices || '',
        rayStartArgsText: (override?.rayStartArgs || []).join('\n'),
      })];
    })
  );

  return {
    name: deployment.name,
    framework: deployment.framework || 'vllm-ascend',
    modelMode: matchedModel ? 'existing' : 'custom',
    selectedModelId: matchedModel?.id || '',
    model: matchedModel || deployment.model,
    dockerImage: deployment.docker.image,
    dockerTag: deployment.docker.tag,
    apiPort: deployment.apiPort,
    serverIds: deployment.servers,
    rayEnabled: deployment.ray?.enabled || false,
    rayHeadServerId: deployment.ray?.headServerId || deployment.servers[0] || '',
    rayNICName: deployment.ray?.nicName || '',
    rayPort: deployment.ray?.port || 6379,
    rayDashboardPort: deployment.ray?.dashboardPort || 8265,
    rayVisibleDevices: deployment.ray?.visibleDevices || '',
    workDir: deployment.runtime?.workDir || '',
    modelDir: deployment.runtime?.modelDir || '',
    cacheDir: deployment.runtime?.cacheDir || '',
    sharedCacheDir: deployment.runtime?.sharedCacheDir || '',
    tensorParallelSize: deployment.vllm.tensorParallelSize || 1,
    pipelineParallelSize: deployment.vllm.pipelineParallelSize || 1,
    maxModelLen: deployment.vllm.maxModelLen || 4096,
    gpuMemoryUtilization: deployment.vllm.gpuMemoryUtilization || 0.9,
    dtype: deployment.vllm.dtype || 'auto',
    quantization: deployment.vllm.quantization || '',
    trustRemoteCode: deployment.vllm.trustRemoteCode || false,
    enablePrefixCaching: deployment.vllm.enablePrefixCaching || false,
    enableExpertParallel: deployment.vllm.enableExpertParallel || false,
    maxNumSeqs: deployment.vllm.maxNumSeqs || 256,
    maxNumBatchedTokens: deployment.vllm.maxNumBatchedTokens || 8192,
    extraArgsText: (deployment.runtime?.extraArgs || []).join('\n'),
    serverOverrides: overrides,
  };
};

type DeploymentRealtimeMessage = {
  type?: string;
  deploymentId?: string;
  data?: {
    serverId?: string;
    stepId?: string;
    progress?: number;
    status?: DeploymentStep['status'] | DeploymentConfig['status'];
    overallProgress?: number;
    endpoints?: DeploymentConfig['endpoints'];
    lines?: string[];
  };
};

const mergeDeploymentStatus = (
  items: DeploymentConfig[],
  deploymentId: string,
  patch: { status?: DeploymentConfig['status']; endpoints?: DeploymentConfig['endpoints'] }
) =>
  items.map((deployment) =>
    deployment.id === deploymentId
      ? {
          ...deployment,
          status: patch.status || deployment.status,
          endpoints: patch.endpoints || deployment.endpoints,
        }
      : deployment
  );

const mergeTaskProgress = (
  items: DeploymentTask[],
  serverId: string,
  stepId: string,
  patch: { progress?: number; status?: DeploymentStep['status']; overallProgress?: number }
) =>
  items.map((task) => {
    if (task.serverId !== serverId) return task;
    let changed = false;
    const steps = task.steps.map((step) => {
      if (step.id !== stepId) return step;
      changed = true;
      return {
        ...step,
        progress: typeof patch.progress === 'number' ? patch.progress : step.progress,
        status: patch.status || step.status,
      };
    });
    if (!changed) return task;
    return {
      ...task,
      steps,
      overallProgress: typeof patch.overallProgress === 'number' ? patch.overallProgress : task.overallProgress,
    };
  });

const appendTaskStepLogs = (items: DeploymentTask[], serverId: string, stepId: string, lines: string[]) =>
  items.map((task) => {
    if (task.serverId !== serverId) return task;
    let changed = false;
    const steps = task.steps.map((step) => {
      if (step.id !== stepId) return step;
      changed = true;
      return {
        ...step,
        logs: [...(step.logs || []), ...lines],
      };
    });
    return changed ? { ...task, steps } : task;
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
  const [editingDeploymentId, setEditingDeploymentId] = useState('');
  const [expandedStepId, setExpandedStepId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const logRefs = useRef<Record<string, HTMLPreElement | null>>({});

  const currentProject = projects.find((item) => item.id === currentProjectId);
  const selectedTemplate = useMemo(
    () => templates.find((item) => item.framework === draft.framework || item.id === draft.framework),
    [draft.framework, templates]
  );
  const visibleServers = useMemo(
    () => servers.filter((item) => !currentProjectId || item.projectId === currentProjectId),
    [servers, currentProjectId]
  );
  const selectedServers = useMemo(
    () => visibleServers.filter((item) => draft.serverIds.includes(item.id)),
    [draft.serverIds, visibleServers]
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
      .catch((err) => active && setError(err instanceof Error ? err.message : '加载流水线数据失败'))
      .finally(() => active && setLoading(false));
    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    if (!selectedTemplate) return;
    setDraft((current) => {
      if (current.dockerImage || current.workDir || current.name) return current;
      return applyTemplateDefaults(current, selectedTemplate);
    });
  }, [selectedTemplate?.id]);

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
    const timer = window.setInterval(() => void poll(), 10000);
    return () => {
      active = false;
      window.clearInterval(timer);
    };
  }, [currentDeploymentId]);

  useEffect(() => {
    if (!currentDeploymentId) return;

    let active = true;
    let socket: WebSocket | null = null;
    let retryTimer = 0;

    const connect = () => {
      if (!active) return;
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      socket = new WebSocket(`${protocol}//${window.location.host}/ws`);

      socket.addEventListener('open', () => {
        socket?.send(JSON.stringify({ type: 'subscribe', deploymentId: currentDeploymentId }));
      });

      socket.addEventListener('message', (event) => {
        let message: DeploymentRealtimeMessage | null = null;
        try {
          message = JSON.parse(event.data) as DeploymentRealtimeMessage;
        } catch {
          return;
        }
        if (!message || message.deploymentId !== currentDeploymentId) return;
        const data = message.data;

        if (message.type === 'progress' && data?.serverId && data.stepId) {
          setTasks((current) =>
            mergeTaskProgress(current, data.serverId || '', data.stepId || '', {
              progress: data.progress,
              status: data.status as DeploymentStep['status'] | undefined,
              overallProgress: data.overallProgress,
            })
          );
          return;
        }

        if (message.type === 'step_log' && data?.serverId && data.stepId && data.lines?.length) {
          setTasks((current) => appendTaskStepLogs(current, data.serverId || '', data.stepId || '', data.lines || []));
          return;
        }

        if (message.type === 'status') {
          setDeployments((current) =>
            mergeDeploymentStatus(current, currentDeploymentId, {
              status: data?.status as DeploymentConfig['status'] | undefined,
              endpoints: data?.endpoints,
            })
          );
        }
      });

      socket.addEventListener('close', () => {
        if (!active) return;
        retryTimer = window.setTimeout(connect, 1500);
      });
    };

    connect();

    return () => {
      active = false;
      window.clearTimeout(retryTimer);
      if (socket && socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify({ type: 'unsubscribe', deploymentId: currentDeploymentId }));
      }
      socket?.close();
    };
  }, [currentDeploymentId]);

  useEffect(() => {
    if (!expandedStepId) return;
    const raf = window.requestAnimationFrame(() => {
      Object.entries(logRefs.current).forEach(([key, node]) => {
        if (!node || !key.startsWith(`${expandedStepId}:`)) return;
        node.scrollTop = node.scrollHeight;
      });
    });
    return () => window.cancelAnimationFrame(raf);
  }, [expandedStepId, tasks]);

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

  const modelIdentity = useMemo(() => {
    if (draft.modelMode === 'existing') return selectedModel?.modelId || selectedModel?.name || '';
    return draft.model.modelId || draft.model.name || draft.model.localPath || '';
  }, [draft.model.localPath, draft.model.modelId, draft.model.name, draft.modelMode, selectedModel]);

  const acceleratorCounts = useMemo(
    () => selectedServers.map((item) => countAccelerators(item)).filter((item) => item > 0),
    [selectedServers]
  );
  const uniformAcceleratorCount =
    acceleratorCounts.length > 0 && acceleratorCounts.every((item) => item === acceleratorCounts[0]) ? acceleratorCounts[0] : 0;
  const recommendedTP = uniformAcceleratorCount || draft.tensorParallelSize || 1;
  const recommendedPP = Math.max(1, selectedServers.length);

  const normalizeServerSelection = (serverIds: string[], nextOverrides: Record<string, DraftServerOverride>) => {
    const overrides: Record<string, DraftServerOverride> = {};
    serverIds.forEach((serverId) => {
      const server = visibleServers.find((item) => item.id === serverId);
      overrides[serverId] = makeOverride(server, nextOverrides[serverId]);
    });
    return {
      serverOverrides: overrides,
      rayHeadServerId: serverIds.includes(draft.rayHeadServerId) ? draft.rayHeadServerId : serverIds[0] || '',
    };
  };

  const toggleServer = (server: ServerConfig) => {
    setDraft((current) => {
      const exists = current.serverIds.includes(server.id);
      const serverIds = exists ? current.serverIds.filter((item) => item !== server.id) : [...current.serverIds, server.id];
      const nextOverrides = { ...current.serverOverrides, [server.id]: makeOverride(server, current.serverOverrides[server.id]) };
      if (exists) delete nextOverrides[server.id];
      const normalized = normalizeServerSelection(serverIds, nextOverrides);
      return { ...current, serverIds, ...normalized };
    });
  };

  const updateOverride = (serverId: string, patch: Partial<DraftServerOverride>) => {
    setDraft((current) => {
      const server = visibleServers.find((item) => item.id === serverId);
      return {
        ...current,
        serverOverrides: {
          ...current.serverOverrides,
          [serverId]: { ...makeOverride(server, current.serverOverrides[serverId]), ...patch },
        },
      };
    });
  };

  const loadDeploymentIntoEditor = (deployment: DeploymentConfig) => {
    setDraft(deploymentToDraft(deployment, models, visibleServers));
    setEditingDeploymentId(deployment.id);
    setCurrentDeploymentId(deployment.id);
    setNotice(`已载入部署 ${deployment.name}，可以直接修改后重新运行。`);
    setError('');
  };

  const resetEditor = () => {
    const template = templates.find((item) => item.framework === 'vllm-ascend') || templates[0];
    setEditingDeploymentId('');
    setCurrentDeploymentId('');
    setTasks([]);
    setExpandedStepId(null);
    setNotice('');
    setError('');
    setDraft(template ? applyTemplateDefaults(createDraft(), template) : createDraft());
  };

  const applyRayRecommendation = () => {
    setDraft((current) => {
      const nextOverrides = { ...current.serverOverrides };
      selectedServers.forEach((server) => {
        const acceleratorCount = countAccelerators(server);
        nextOverrides[server.id] = {
          ...makeOverride(server, current.serverOverrides[server.id]),
          visibleDevices:
            current.serverOverrides[server.id]?.visibleDevices ||
            (acceleratorCount > 0 ? Array.from({ length: acceleratorCount }, (_, index) => `${index}`).join(',') : ''),
        };
      });
      return {
        ...current,
        rayEnabled: selectedServers.length > 1,
        rayHeadServerId: current.rayHeadServerId || selectedServers[0]?.id || '',
        tensorParallelSize: recommendedTP,
        pipelineParallelSize: recommendedPP,
        enableExpertParallel: isQwen397BaseModel(modelIdentity) || current.enableExpertParallel,
        serverOverrides: nextOverrides,
      };
    });
    setNotice('已套用一组适合作为双机 Ray 起步的参数，后续仍可继续微调。');
    setError('');
  };

  const restartExistingDeployment = async (deployment: DeploymentConfig) => {
    setSubmitting(true);
    setError('');
    try {
      await requestJSON(`/api/deployments/${deployment.id}/start`, { method: 'POST' });
      setCurrentDeploymentId(deployment.id);
      setNotice(`已重新启动流水线 ${deployment.name}`);
      const [deploymentItems, taskItems] = await Promise.all([
        requestJSON<DeploymentConfig[]>('/api/deployments'),
        requestJSON<DeploymentTask[]>(`/api/tasks?deploymentId=${encodeURIComponent(deployment.id)}`),
      ]);
      setDeployments(deploymentItems || []);
      setTasks(taskItems || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : '重新启动流水线失败');
    } finally {
      setSubmitting(false);
    }
  };

  const buildRayPreview = (server: ServerConfig) => {
    if (!draft.rayEnabled) return '未启用 Ray，当前部署会直接在单机容器内执行推理服务。';
    const override = draft.serverOverrides[server.id] || makeOverride(server);
    const head = selectedServers.find((item) => item.id === draft.rayHeadServerId) || selectedServers[0];
    const headOverride = draft.serverOverrides[head?.id || ''];
    const nodeIp = override.nodeIp.trim() || server.host || '127.0.0.1';
    const headNodeIp = headOverride?.nodeIp?.trim() || head?.host || '127.0.0.1';
    const extraArgs = override.rayStartArgsText.split('\n').map((item) => item.trim()).filter(Boolean).join(' ');
    if (head?.id === server.id) {
      return [
        '角色: Ray Head',
        `ray start --head --port ${draft.rayPort || 6379} --dashboard-host 0.0.0.0 --dashboard-port ${draft.rayDashboardPort || 8265} --node-ip-address ${nodeIp}${extraArgs ? ` ${extraArgs}` : ''}`,
        '随后在本机执行: vllm serve /model --distributed-executor-backend ray ...',
      ].join('\n');
    }
    return [
      '角色: Ray Worker',
      `ray start --address ${headNodeIp}:${draft.rayPort || 6379} --node-ip-address ${nodeIp}${extraArgs ? ` ${extraArgs}` : ''}`,
      '当前节点只加入集群并常驻，不会重复执行 vllm serve。',
    ].join('\n');
  };

  const saveDeployment = async (startNow: boolean) => {
    const model = draft.modelMode === 'existing' ? selectedModel : draft.model;
    if (!selectedTemplate) return setError('请先选择一个部署模板。');
    if (!draft.name.trim()) return setError('请填写部署名称。');
    if (!model) return setError('请先选择模型。');
    if (draft.serverIds.length === 0) return setError('至少选择一台服务器。');
    if (draft.rayEnabled && !draft.rayHeadServerId) return setError('启用 Ray 时必须指定 head 节点。');
    if ((model.source || 'local') === 'local' && !model.localPath?.trim()) return setError('本地模型必须填写模型目录。');
    if ((model.source || 'local') !== 'local' && !model.modelId?.trim()) return setError('远端模型必须填写模型 ID。');

    const serverOverrides = draft.serverIds
      .map((serverId) => {
        const override = draft.serverOverrides[serverId];
        if (!override) return null;
        const rayStartArgs = override.rayStartArgsText.split('\n').map((item) => item.trim()).filter(Boolean);
        if (!override.nodeIp.trim() && !override.visibleDevices.trim() && rayStartArgs.length === 0) return null;
        return {
          serverId,
          nodeIp: override.nodeIp.trim(),
          visibleDevices: override.visibleDevices.trim(),
          rayStartArgs,
        };
      })
      .filter(Boolean);

    const payload = {
      name: draft.name.trim(),
      framework: draft.framework,
      model,
      docker: {
        image: draft.dockerImage,
        tag: draft.dockerTag,
        gpuDevices: 'all',
        shmSize: '16g',
        environmentVars: {},
        volumes: [],
        network: 'host',
        ipc: 'host',
        privileged: draft.framework !== 'tei',
      },
      ray: {
        enabled: draft.rayEnabled,
        headServerId: draft.rayHeadServerId,
        nicName: draft.rayNICName.trim(),
        port: draft.rayPort,
        dashboardPort: draft.rayDashboardPort,
        visibleDevices: draft.rayVisibleDevices.trim(),
      },
      runtime: {
        workDir: draft.workDir.trim(),
        modelDir: draft.modelDir.trim(),
        cacheDir: draft.cacheDir.trim(),
        sharedCacheDir: draft.sharedCacheDir.trim(),
        enableAutoRestart: true,
        extraArgs: draft.extraArgsText.split('\n').map((item) => item.trim()).filter(Boolean),
      },
      vllm: {
        tensorParallelSize: draft.tensorParallelSize,
        pipelineParallelSize: draft.pipelineParallelSize,
        maxModelLen: draft.maxModelLen,
        gpuMemoryUtilization: draft.gpuMemoryUtilization,
        quantization: draft.quantization.trim(),
        dtype: draft.dtype.trim() || 'auto',
        trustRemoteCode: draft.trustRemoteCode,
        enablePrefixCaching: draft.enablePrefixCaching,
        enableExpertParallel: draft.enableExpertParallel,
        maxNumSeqs: draft.maxNumSeqs,
        maxNumBatchedTokens: draft.maxNumBatchedTokens,
      },
      serverOverrides,
      servers: draft.serverIds,
      apiPort: draft.apiPort,
    };

    setSubmitting(true);
    setError('');
    try {
      const target = editingDeploymentId
        ? await requestJSON<DeploymentConfig>(`/api/deployments/${editingDeploymentId}`, { method: 'PATCH', body: JSON.stringify(payload) })
        : await requestJSON<DeploymentConfig>('/api/deployments', { method: 'POST', body: JSON.stringify(payload) });
      if (startNow) await requestJSON(`/api/deployments/${target.id}/start`, { method: 'POST' });
      setEditingDeploymentId(target.id);
      setCurrentDeploymentId(target.id);
      setNotice(startNow ? `已启动流水线 ${target.name}` : `已保存部署 ${target.name}`);
      await loadBase();
      setTasks(startNow ? await requestJSON<DeploymentTask[]>(`/api/tasks?deploymentId=${encodeURIComponent(target.id)}`) : []);
    } catch (err) {
      setError(err instanceof Error ? err.message : startNow ? '启动流水线失败' : '保存部署失败');
      setNotice('');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="space-y-8">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">部署流水线</h1>
          <p className="text-slate-500 mt-1">{currentProject?.name || '全局工作区'} | 同一页完成模板选择、Ray 组网、参数调整和失败后的重新执行。</p>
        </div>
        <div className="flex items-center gap-3">
          <button onClick={() => { setLoading(true); loadBase().finally(() => setLoading(false)); }} className="px-4 py-2.5 bg-white border border-slate-200 rounded-xl text-slate-700 hover:bg-slate-50">
            <span className="inline-flex items-center gap-2"><RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />刷新</span>
          </button>
          <button onClick={resetEditor} className="px-4 py-2.5 bg-white border border-slate-200 rounded-xl text-slate-700 hover:bg-slate-50">
            <span className="inline-flex items-center gap-2"><RotateCcw className="w-4 h-4" />新建部署</span>
          </button>
          <button onClick={() => void saveDeployment(false)} disabled={submitting} className="px-4 py-2.5 bg-white border border-slate-200 rounded-xl text-slate-700 hover:bg-slate-50">{editingDeploymentId ? '更新配置' : '保存草稿'}</button>
          <button onClick={() => void saveDeployment(true)} disabled={submitting} className="px-4 py-2.5 bg-blue-600 text-white rounded-xl hover:bg-blue-700">
            <span className="inline-flex items-center gap-2"><Play className="w-4 h-4" />{editingDeploymentId ? '保存并重跑' : '启动流水线'}</span>
          </button>
        </div>
      </div>

      {editingDeploymentId && <div className="rounded-2xl border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-700">当前正在编辑已有部署 {draft.name}。如果上一次失败，可以直接修改参数后点击“保存并重跑”。</div>}
      {(notice || error) && <div className={`rounded-2xl border px-4 py-3 text-sm ${error ? 'border-red-200 bg-red-50 text-red-700' : 'border-emerald-200 bg-emerald-50 text-emerald-700'}`}>{error || notice}</div>}

      <div className="grid gap-6 xl:grid-cols-[1.1fr,0.9fr]">
        <div className="space-y-6">
          <Section title="框架模板">
            <div className="grid gap-3 md:grid-cols-3">
              {templates.map((template) => (
                <button key={template.id} onClick={() => setDraft((current) => applyTemplateDefaults(current, template))} className={`rounded-2xl border p-4 text-left ${draft.framework === template.framework ? 'border-blue-500 bg-blue-50' : 'border-slate-200 bg-slate-50 hover:bg-white'}`}>
                  <div className="font-medium text-slate-900">{template.name}</div>
                  <div className="text-sm text-slate-500 mt-2">{template.description}</div>
                </button>
              ))}
            </div>
          </Section>

          <Section title="基础信息">
            <div className="grid gap-4 md:grid-cols-2">
              <Field label="部署名称" value={draft.name} onChange={(value) => setDraft((current) => ({ ...current, name: value }))} />
              <Field label="API 端口" type="number" value={`${draft.apiPort}`} onChange={(value) => setDraft((current) => ({ ...current, apiPort: Number(value) || 0 }))} />
            </div>
          </Section>

          <Section title="模型来源">
            <div className="flex gap-3 mb-4">
              <SmallToggle active={draft.modelMode === 'existing'} onClick={() => setDraft((current) => ({ ...current, modelMode: 'existing' }))} label="使用已保存模型" />
              <SmallToggle active={draft.modelMode === 'custom'} onClick={() => setDraft((current) => ({ ...current, modelMode: 'custom' }))} label="自定义模型" />
            </div>
            {draft.modelMode === 'existing' ? (
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">模型</label>
                <select value={draft.selectedModelId} onChange={(event) => setDraft((current) => ({ ...current, selectedModelId: event.target.value }))} className="w-full px-4 py-2 border border-slate-200 rounded-xl bg-white">
                  <option value="">请选择模型</option>
                  {models.map((model) => <option key={model.id} value={model.id}>{model.name} | {model.source}</option>)}
                </select>
              </div>
            ) : (
              <div className="grid gap-4 md:grid-cols-2">
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">来源</label>
                  <select value={draft.model.source || 'modelscope'} onChange={(event) => setDraft((current) => ({ ...current, model: { ...current.model, source: event.target.value as ModelConfig['source'] } }))} className="w-full px-4 py-2 border border-slate-200 rounded-xl bg-white">
                    <option value="modelscope">ModelScope</option>
                    <option value="huggingface">Hugging Face</option>
                    <option value="local">本地目录</option>
                  </select>
                </div>
                <Field label={draft.model.source === 'local' ? '本地目录' : '模型 ID'} value={draft.model.source === 'local' ? draft.model.localPath || '' : draft.model.modelId || ''} onChange={(value) => setDraft((current) => ({ ...current, model: current.model.source === 'local' ? { ...current.model, localPath: value } : { ...current.model, modelId: value } }))} />
              </div>
            )}
            {isQwen397BaseModel(modelIdentity) && <div className="mt-4 rounded-2xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-700">当前识别到模型为原始 BF16 版 Qwen/Qwen3.5-397B-A17B。按 vLLM Ascend 文档，更接近 4 台 A2/910B 或 2 台 A3 的资源级别；如果只有两台 910B，更建议改用 `-w8a8` 量化版，再配合 Ray 进行双机部署。</div>}
          </Section>

          <Section title="目标服务器">
            <div className="grid gap-3 md:grid-cols-2">
              {visibleServers.map((server) => (
                <label key={server.id} className="flex items-start gap-3 rounded-2xl border border-slate-200 p-4 bg-slate-50">
                  <input type="checkbox" checked={draft.serverIds.includes(server.id)} onChange={() => toggleServer(server)} className="mt-1" />
                  <span>
                    <span className="block font-medium text-slate-900">{server.name}</span>
                    <span className="block text-sm text-slate-500 mt-1">{server.host} | {server.status}{countAccelerators(server) > 0 ? ` | ${countAccelerators(server)} 张加速卡` : ''}{server.useJumpHost ? ' | 经跳板机' : ''}</span>
                  </span>
                </label>
              ))}
            </div>
          </Section>

          {draft.framework === 'vllm-ascend' && <Section title="vLLM 参数">
            <div className="grid gap-4 md:grid-cols-2">
              <Field label="Tensor Parallel Size" type="number" value={`${draft.tensorParallelSize}`} onChange={(value) => setDraft((current) => ({ ...current, tensorParallelSize: Number(value) || 1 }))} />
              <Field label="Pipeline Parallel Size" type="number" value={`${draft.pipelineParallelSize}`} onChange={(value) => setDraft((current) => ({ ...current, pipelineParallelSize: Number(value) || 1 }))} />
              <Field label="Max Model Len" type="number" value={`${draft.maxModelLen}`} onChange={(value) => setDraft((current) => ({ ...current, maxModelLen: Number(value) || 4096 }))} />
              <Field label="GPU/NPU Memory Utilization" type="number" value={`${draft.gpuMemoryUtilization}`} onChange={(value) => setDraft((current) => ({ ...current, gpuMemoryUtilization: Number(value) || 0.9 }))} />
              <Field label="DType" value={draft.dtype} onChange={(value) => setDraft((current) => ({ ...current, dtype: value }))} />
              <Field label="Quantization" value={draft.quantization} onChange={(value) => setDraft((current) => ({ ...current, quantization: value }))} />
              <Field label="Max Num Seqs" type="number" value={`${draft.maxNumSeqs}`} onChange={(value) => setDraft((current) => ({ ...current, maxNumSeqs: Number(value) || 1 }))} />
              <Field label="Max Num Batched Tokens" type="number" value={`${draft.maxNumBatchedTokens}`} onChange={(value) => setDraft((current) => ({ ...current, maxNumBatchedTokens: Number(value) || 1 }))} />
            </div>
            <div className="flex flex-wrap gap-3 mt-4">
              <SmallToggle active={draft.trustRemoteCode} onClick={() => setDraft((current) => ({ ...current, trustRemoteCode: !current.trustRemoteCode }))} label="信任远端代码" />
              <SmallToggle active={draft.enablePrefixCaching} onClick={() => setDraft((current) => ({ ...current, enablePrefixCaching: !current.enablePrefixCaching }))} label="开启 Prefix Cache" />
              <SmallToggle active={draft.enableExpertParallel} onClick={() => setDraft((current) => ({ ...current, enableExpertParallel: !current.enableExpertParallel }))} label="开启 Expert Parallel" />
            </div>
            <div className="mt-4 rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600">对双机 Ray 来说，常见起步思路是“每台机器卡数 = TP，机器台数 = PP”。例如两台 910B 服务器、每台 8 卡，通常先从 TP=8、PP=2 开始，再看显存和吞吐调整。</div>
          </Section>}

          <Section title="运行时与 Ray">
            <div className="grid gap-4 md:grid-cols-2">
              <Field label="镜像" value={draft.dockerImage} onChange={(value) => setDraft((current) => ({ ...current, dockerImage: value }))} />
              <Field label="Tag" value={draft.dockerTag} onChange={(value) => setDraft((current) => ({ ...current, dockerTag: value }))} />
            </div>

            {selectedTemplate?.supportsRay && <div className="mt-4 space-y-4">
              <div className="flex items-center justify-between gap-3 rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3">
                <div>
                  <div className="font-medium text-slate-900">启用 Ray 集群</div>
                  <div className="text-sm text-slate-500 mt-1">启用后会自动区分 head/worker，下发不同的 ray start 参数。</div>
                </div>
                <SmallToggle active={draft.rayEnabled} onClick={() => setDraft((current) => ({ ...current, rayEnabled: !current.rayEnabled, rayHeadServerId: current.rayHeadServerId || current.serverIds[0] || '' }))} label={draft.rayEnabled ? '已开启' : '未开启'} />
              </div>

              {draft.rayEnabled && <>
                <div className="grid gap-4 md:grid-cols-2">
                  <div>
                    <label className="block text-sm font-medium text-slate-700 mb-1">Ray Head 节点</label>
                    <select value={draft.rayHeadServerId} onChange={(event) => setDraft((current) => ({ ...current, rayHeadServerId: event.target.value }))} className="w-full px-4 py-2 border border-slate-200 rounded-xl bg-white">
                      <option value="">请选择</option>
                      {selectedServers.map((server) => <option key={server.id} value={server.id}>{server.name}</option>)}
                    </select>
                  </div>
                  <Field label="Ray 通信网卡名" value={draft.rayNICName} onChange={(value) => setDraft((current) => ({ ...current, rayNICName: value }))} />
                  <Field label="Ray 端口" type="number" value={`${draft.rayPort}`} onChange={(value) => setDraft((current) => ({ ...current, rayPort: Number(value) || 6379 }))} />
                  <Field label="Ray Dashboard 端口" type="number" value={`${draft.rayDashboardPort}`} onChange={(value) => setDraft((current) => ({ ...current, rayDashboardPort: Number(value) || 8265 }))} />
                  <Field label="全局可见设备" value={draft.rayVisibleDevices} onChange={(value) => setDraft((current) => ({ ...current, rayVisibleDevices: value }))} />
                </div>
                <div className="rounded-2xl border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-700">Ray head 负责建立集群与对外承载 vLLM API；worker 只负责加入集群并提供计算资源。{uniformAcceleratorCount ? ` 当前每台已识别 ${uniformAcceleratorCount} 张加速卡，建议先从 TP=${recommendedTP}、PP=${recommendedPP} 开始。` : ''}</div>
                <div className="flex items-center justify-between gap-3 rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3">
                  <div className="text-sm text-slate-600">针对双机 Ray 自动填入一组起步参数，并为每台服务器补齐 node IP 与可见设备建议值。</div>
                  <button type="button" onClick={applyRayRecommendation} className="px-3 py-2 rounded-xl bg-blue-600 text-white hover:bg-blue-700"><span className="inline-flex items-center gap-2"><Wand2 className="w-4 h-4" />套用建议值</span></button>
                </div>
                <div className="grid gap-4">
                  {selectedServers.map((server) => {
                    const override = draft.serverOverrides[server.id] || makeOverride(server);
                    const isHead = (draft.rayHeadServerId || selectedServers[0]?.id) === server.id;
                    return <div key={server.id} className={`rounded-2xl border p-4 ${isHead ? 'border-blue-200 bg-blue-50' : 'border-slate-200 bg-slate-50'}`}>
                      <div className="flex items-start justify-between gap-3">
                        <div>
                          <div className="font-medium text-slate-900">{server.name} {isHead ? '(Ray Head)' : '(Ray Worker)'}</div>
                          <div className="text-sm text-slate-500 mt-1">{server.host}</div>
                        </div>
                        <span className="text-xs text-slate-600">{server.status}</span>
                      </div>
                      <div className="grid gap-4 md:grid-cols-2 mt-4">
                        <Field label="本机 node IP" value={override.nodeIp} onChange={(value) => updateOverride(server.id, { nodeIp: value })} />
                        <Field label="本机可见设备" value={override.visibleDevices} onChange={(value) => updateOverride(server.id, { visibleDevices: value })} />
                      </div>
                      <div className="mt-4">
                        <label className="block text-sm font-medium text-slate-700 mb-1">Ray 额外参数（每行一个）</label>
                        <textarea rows={3} value={override.rayStartArgsText} onChange={(event) => updateOverride(server.id, { rayStartArgsText: event.target.value })} className="w-full px-4 py-2 border border-slate-200 rounded-xl font-mono text-sm" />
                      </div>
                      <pre className="mt-4 text-xs bg-slate-900 text-slate-100 rounded-xl p-3 overflow-x-auto whitespace-pre-wrap break-all">{buildRayPreview(server)}</pre>
                    </div>;
                  })}
                </div>
              </>}
            </div>}

            <div className="grid gap-4 md:grid-cols-2 mt-4">
              <Field label="工作目录" value={draft.workDir} onChange={(value) => setDraft((current) => ({ ...current, workDir: value }))} />
              <Field label="模型目录" value={draft.modelDir} onChange={(value) => setDraft((current) => ({ ...current, modelDir: value }))} />
              <Field label="缓存目录" value={draft.cacheDir} onChange={(value) => setDraft((current) => ({ ...current, cacheDir: value }))} />
              <Field label="共享缓存目录（可选）" value={draft.sharedCacheDir} onChange={(value) => setDraft((current) => ({ ...current, sharedCacheDir: value }))} />
            </div>
            <div className="mt-3 rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600">模型下载、目录创建和启动脚本写入时，系统会先判断当前 SSH 用户是否对目标目录有写权限；没有时会自动尝试 `sudo -n`。因此建议在这里显式配置一个可写目录，例如 `/data/models`、`/data/cache`。远端模型会自动下载到“模型目录/规范化模型 ID”，例如 `Qwen/Qwen3.5-397B-A17B` 会落到 `/data/models/qwen/qwen3.5-397b-a17b`。</div>
            <div className="mt-4">
              <label className="block text-sm font-medium text-slate-700 mb-1">额外启动参数（每行一个）</label>
              <textarea rows={4} value={draft.extraArgsText} onChange={(event) => setDraft((current) => ({ ...current, extraArgsText: event.target.value }))} className="w-full px-4 py-2 border border-slate-200 rounded-xl font-mono text-sm" />
            </div>
          </Section>
        </div>

        <div className="space-y-6">
          <Section title="流水线看板">
            <div className="grid gap-4">
              {stepCards.map(({ templateStep, related, status, progress }) => <div key={templateStep.id} className={`rounded-2xl border p-4 ${status === 'failed' ? 'border-red-200 bg-red-50' : status === 'running' ? 'border-blue-200 bg-blue-50' : status === 'completed' ? 'border-emerald-200 bg-emerald-50' : 'border-slate-200 bg-slate-50'}`}>
                <button type="button" onClick={() => setExpandedStepId(expandedStepId === templateStep.id ? null : templateStep.id)} className="w-full text-left">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <div className="font-medium text-slate-900">{templateStep.name}</div>
                      <div className="text-sm text-slate-500 mt-2">{templateStep.description}</div>
                    </div>
                    <StatusIcon status={status} />
                  </div>
                  <div className="mt-3 h-2 rounded-full bg-white overflow-hidden"><div className="h-full rounded-full bg-slate-700/40" style={{ width: `${progress}%` }} /></div>
                </button>
                {expandedStepId === templateStep.id && <div className="mt-4 pt-4 border-t border-slate-200 space-y-3">
                  {(templateStep.details || []).map((detail) => <div key={detail} className="text-sm text-slate-600">{detail}</div>)}
                  {related.length > 0 ? related.map(({ serverId, step }) => <div key={`${templateStep.id}-${serverId}`} className="rounded-2xl border border-slate-200 bg-white p-4">
                    <div className="font-medium text-slate-900">{visibleServers.find((item) => item.id === serverId)?.name || serverId}</div>
                    {step?.commandPreview && <pre className="mt-3 text-xs bg-slate-900 text-slate-100 rounded-xl p-3 overflow-x-auto whitespace-pre-wrap break-all">{step.commandPreview}</pre>}
                    <pre ref={(node) => { logRefs.current[`${templateStep.id}:${serverId}`] = node; }} className="mt-3 text-xs bg-slate-900 text-slate-100 rounded-xl p-3 overflow-x-auto overflow-y-auto whitespace-pre-wrap break-all max-h-64">{step?.logs?.join('\n') || '当前还没有日志。'}</pre>
                  </div>) : <div className="text-sm text-slate-500">选择一个部署并启动后，这里会展示每台服务器的执行命令和日志。</div>}
                </div>}
              </div>)}
            </div>
          </Section>

          <Section title="最近部署">
            <div className="space-y-3">
              {deployments.slice().sort((a, b) => b.createdAt.localeCompare(a.createdAt)).slice(0, 8).map((deployment) => <div key={deployment.id} className={`rounded-2xl border p-4 ${currentDeploymentId === deployment.id ? 'border-blue-500 bg-blue-50' : 'border-slate-200 bg-slate-50'}`}>
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="font-medium text-slate-900">{deployment.name}</div>
                    <div className="text-sm text-slate-500 mt-1">{deployment.framework} | {deployment.servers.length} 台服务器 | 端口 {deployment.apiPort}</div>
                  </div>
                  <span className="text-xs text-slate-600">{statusLabels[deployment.status]}</span>
                </div>
                <div className="mt-3 flex flex-wrap gap-2">
                  <button type="button" onClick={() => { setCurrentDeploymentId(deployment.id); setNotice(`正在查看部署 ${deployment.name} 的执行过程。`); setError(''); }} className="px-3 py-2 rounded-xl bg-white border border-slate-200 text-slate-700 hover:bg-slate-50">查看执行</button>
                  <button type="button" onClick={() => loadDeploymentIntoEditor(deployment)} className="px-3 py-2 rounded-xl bg-white border border-slate-200 text-slate-700 hover:bg-slate-50">载入编辑</button>
                  {deployment.status !== 'deploying' && <button type="button" onClick={() => void restartExistingDeployment(deployment)} disabled={submitting} className="px-3 py-2 rounded-xl bg-blue-600 text-white hover:bg-blue-700">直接重跑</button>}
                </div>
              </div>)}
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
