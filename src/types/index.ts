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
  maxNumSeqs: number;
  maxNumBatchedTokens: number;
  swapSpace?: number;
  enforceEager?: boolean;
  enableChunkedPrefill?: boolean;
  speculativeModel?: string;
  numSpeculativeTokens?: number;
}

export interface DeploymentConfig {
  id: string;
  name: string;
  status: 'draft' | 'deploying' | 'running' | 'failed' | 'stopped';
  model: ModelConfig;
  docker: DockerConfig;
  vllm: VLLMParams;
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
  status: 'pending' | 'running' | 'completed' | 'failed';
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
