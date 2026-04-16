export interface Project {
  id: string;
  name: string;
  description: string;
  color: string;
  createdAt: string;
  updatedAt: string;
  serverIds: string[];
}

export interface ServerConfig {
  id: string;
  projectId: string;
  name: string;
  host: string;
  sshPort: number;
  username: string;
  authType: 'password' | 'key';
  password?: string;
  privateKey?: string;
  isJumpHost?: boolean;
  useJumpHost: boolean;
  jumpHostId?: string;
  gpuInfo?: GPUInfo[];
  driverVersion?: string;
  cudaVersion?: string;
  dockerVersion?: string;
  npuExporterEndpoint?: string;
  npuExporterStatus?: string;
  npuExporterLastCheck?: string;
  lastCheck?: string;
  status: 'online' | 'offline' | 'checking';
}

export interface GPUInfo {
  index: number;
  type?: 'gpu' | 'npu';
  name: string;
  memoryTotal: number;
  memoryUsed: number;
  memoryFree: number;
  utilization: number;
  temperature: number;
  powerDraw: number;
  powerLimit: number;
  health?: string;
  logicId?: number;
  chipId?: number;
}

export interface JumpHost {
  id: string;
  name: string;
  host: string;
  sshPort: number;
  username: string;
  authType: 'password' | 'key';
  password?: string;
  privateKey?: string;
}

export interface ModelConfig {
  id: string;
  name: string;
  source: 'modelscope' | 'huggingface' | 'local';
  modelId: string;
  localPath?: string;
  revision?: string;
  size?: number;
  format?: string;
  parameters?: string;
  quantization?: string;
  files?: ModelFile[];
}

export interface ModelFile {
  name: string;
  size: number;
  path: string;
  checksum?: string;
}

export interface DockerConfig {
  image: string;
  registry?: string;
  tag: string;
  gpuDevices: string;
  shmSize: string;
  environmentVars: Record<string, string>;
  volumes: Array<{ host: string; container: string }>;
  network?: string;
  ipc?: string;
  privileged?: boolean;
  runtime?: string;
}

export interface VLLMParams {
  tensorParallelSize: number;
  pipelineParallelSize: number;
  maxModelLen: number;
  gpuMemoryUtilization: number;
  quantization?: string;
  dtype: string;
  trustRemoteCode: boolean;
  enablePrefixCaching: boolean;
  enableExpertParallel?: boolean;
  maxNumSeqs: number;
  maxNumBatchedTokens: number;
  swapSpace?: number;
  enforceEager?: boolean;
  enableChunkedPrefill?: boolean;
  speculativeModel?: string;
  numSpeculativeTokens?: number;
}

export interface DeploymentServerOverride {
  serverId: string;
  nodeIp?: string;
  visibleDevices?: string;
  rayStartArgs?: string[];
}

export interface DeploymentRayConfig {
  enabled: boolean;
  headServerId?: string;
  nicName?: string;
  port?: number;
  dashboardPort?: number;
  visibleDevices?: string;
}

export interface DeploymentRuntimeConfig {
  containerName?: string;
  workDir?: string;
  modelDir?: string;
  cacheDir?: string;
  sharedCacheDir?: string;
  enableAutoRestart?: boolean;
  extraArgs?: string[];
}

export interface DeploymentConfig {
  id: string;
  name: string;
  status: 'draft' | 'deploying' | 'running' | 'failed' | 'stopped';
  framework?: 'tei' | 'vllm-ascend' | 'mindie' | string;
  model: ModelConfig;
  docker: DockerConfig;
  vllm: VLLMParams;
  ray?: DeploymentRayConfig;
  runtime?: DeploymentRuntimeConfig;
  serverOverrides?: DeploymentServerOverride[];
  servers: string[];
  apiPort: number;
  createdAt: string;
  updatedAt: string;
  endpoints?: DeploymentEndpoint[];
  metrics?: DeploymentMetrics;
}

export interface DeploymentEndpoint {
  serverId: string;
  url: string;
  status: 'healthy' | 'unhealthy' | 'unknown';
  latency?: number;
}

export interface DeploymentMetrics {
  totalRequests: number;
  avgLatency: number;
  tokensPerSecond: number;
  gpuUtilization: number;
  memoryUtilization: number;
}

export interface DeploymentStep {
  id: string;
  name: string;
  description: string;
  commandPreview?: string;
  optional?: boolean;
  autoManaged?: boolean;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'stopped';
  progress: number;
  logs: string[];
  startTime?: string;
  endTime?: string;
}

export interface DeploymentTask {
  id: string;
  deploymentId: string;
  serverId: string;
  steps: DeploymentStep[];
  currentStep: number;
  overallProgress: number;
}

export type RemoteTaskScope = 'all' | 'project' | 'selected';
export type RemoteTaskExecutionType = 'command' | 'script_url' | 'preset';
export type RemoteTaskStatus = 'pending' | 'running' | 'completed' | 'failed' | 'partial';
export type RemoteTaskRunStatus = 'pending' | 'running' | 'completed' | 'failed';

export interface RemoteTaskRun {
  serverId: string;
  serverName?: string;
  status: RemoteTaskRunStatus;
  command?: string;
  output?: string;
  error?: string;
  exitCode?: number;
  startedAt?: string;
  finishedAt?: string;
}

export interface RemoteTask {
  id: string;
  name: string;
  description?: string;
  projectId?: string;
  scope: RemoteTaskScope;
  status: RemoteTaskStatus;
  executionType: RemoteTaskExecutionType;
  commandPreview?: string;
  scriptUrl?: string;
  scriptArgs?: string;
  presetId?: string;
  presetArgs?: Record<string, string>;
  serverIds: string[];
  runs: RemoteTaskRun[];
  createdAt: string;
  startedAt?: string;
  finishedAt?: string;
}

export interface RemoteTaskPresetField {
  key: string;
  label: string;
  description?: string;
  required?: boolean;
  defaultValue?: string;
  placeholder?: string;
}

export interface RemoteTaskPreset {
  id: string;
  name: string;
  description: string;
  fields?: RemoteTaskPresetField[];
}

export interface ActionTemplateField {
  key: string;
  label: string;
  description?: string;
  required?: boolean;
  defaultValue?: string;
  placeholder?: string;
}

export interface ActionTemplate {
  id: string;
  name: string;
  description: string;
  category?: string;
  builtIn?: boolean;
  executionType: 'command' | 'script_url' | string;
  commandTemplate?: string;
  scriptUrl?: string;
  scriptArgsTemplate?: string;
  fields?: ActionTemplateField[];
  tags?: string[];
  createdAt: string;
  updatedAt: string;
}

export interface BootstrapConfig {
  id: string;
  name: string;
  description: string;
  serviceType: string;
  category?: string;
  builtIn?: boolean;
  actionTemplateId: string;
  defaultArgs?: Record<string, string>;
  endpoint?: string;
  port?: number;
  createdAt: string;
  updatedAt: string;
}

export interface PipelineTemplateStep {
  id: string;
  name: string;
  description: string;
  optional?: boolean;
  autoManaged?: boolean;
  details?: string[];
}

export interface PipelineTemplate {
  id: string;
  name: string;
  framework: 'tei' | 'vllm-ascend' | 'mindie' | string;
  description: string;
  supportsRay?: boolean;
  defaultPort: number;
  defaultDocker: DockerConfig;
  defaultVllm: VLLMParams;
  defaultRay: DeploymentRayConfig;
  defaultRuntime: DeploymentRuntimeConfig;
  steps: PipelineTemplateStep[];
}

export interface SystemStatus {
  storage: {
    driver: string;
    path: string;
    counts: Record<string, number>;
    persistedCollections: string[];
  };
  mock: {
    fakeConnectEnabled: boolean;
    toggleEnv: string;
    legacyHostPrefixMock: boolean;
    description: string;
  };
  demoFeatures: Array<{
    key: string;
    name: string;
    enabled: boolean;
    persisted: boolean;
    description: string;
  }>;
}

export type WizardStep =
  | 'model'
  | 'docker'
  | 'vllm'
  | 'servers'
  | 'review'
  | 'deploy';

export interface WizardState {
  currentStep: WizardStep;
  completedSteps: WizardStep[];
  config: Partial<DeploymentConfig>;
  projectId?: string;
}

export interface ServerResource {
  cpu: {
    cores: number;
    usage: number;
  };
  memory: {
    total: number;
    used: number;
    free: number;
  };
  disk: {
    total: number;
    used: number;
    free: number;
  };
  network: {
    rxSpeed: number;
    txSpeed: number;
  };
}

export interface DeploymentLog {
  timestamp: string;
  level: 'info' | 'warn' | 'error' | 'debug';
  message: string;
  serverId?: string;
  stepId?: string;
}
