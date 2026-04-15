# ModelDeploy API 文档

## 概述

ModelDeploy 是一个模型推理部署工具，提供 Web 界面管理 vLLM/MindIE 模型的分布式部署。

## 架构

```
┌─────────────┐     HTTP/WebSocket     ┌─────────────┐     SSH/Exec      ┌─────────────┐
│   Web UI    │ ◄────────────────────► │   Backend   │ ◄───────────────► │ GPU Servers │
│  (React)    │                        │  (Node.js)  │                   │  (Docker)   │
└─────────────┘                        └─────────────┘                   └─────────────┘
```

## 前端接口

### 1. REST API 接口

#### 服务器管理

```typescript
// 获取服务器列表
GET /api/servers
Response: ServerConfig[]

// 添加服务器
POST /api/servers
Body: {
  name: string;
  host: string;
  sshPort: number;
  username: string;
  authType: 'password' | 'key';
  password?: string;
  privateKey?: string;
  useJumpHost?: boolean;
  jumpHostId?: string;
}

// 测试服务器连接
POST /api/servers/:id/test
Response: {
  success: boolean;
  message: string;
  gpuInfo?: GPUInfo[];
  driverVersion?: string;
  cudaVersion?: string;
}

// 获取服务器资源
GET /api/servers/:id/resources
Response: ServerResource

// 获取服务器GPU信息
GET /api/servers/:id/gpu
Response: GPUInfo[]
```

#### 模型管理

```typescript
// 获取模型列表
GET /api/models
Response: ModelConfig[]

// 添加模型
POST /api/models
Body: {
  name: string;
  source: 'modelscope' | 'huggingface' | 'local';
  modelId?: string;
  localPath?: string;
  revision?: string;
}

// 扫描本地模型
POST /api/models/scan
Body: { path: string }
Response: ModelFile[]

// 获取模型详情
GET /api/models/:id
Response: ModelConfig & { files: ModelFile[] }

// 从 ModelScope 搜索模型
GET /api/models/search?source=modelscope&q=llama
Response: { id: string; name: string; downloads: number }[]
```

#### 部署管理

```typescript
// 获取部署列表
GET /api/deployments
Response: DeploymentConfig[]

// 创建部署
POST /api/deployments
Body: {
  name: string;
  modelId: string;
  serverIds: string[];
  docker: DockerConfig;
  vllm: VLLMParams;
  apiPort: number;
}

// 启动部署
POST /api/deployments/:id/start
Response: { taskId: string }

// 停止部署
POST /api/deployments/:id/stop
Response: { success: boolean }

// 删除部署
DELETE /api/deployments/:id

// 获取部署日志
GET /api/deployments/:id/logs?serverId=&stepId=
Response: DeploymentLog[]

// 获取部署指标
GET /api/deployments/:id/metrics
Response: DeploymentMetrics
```

### 2. WebSocket 接口

```typescript
// 连接地址
ws://host/ws

// 订阅部署任务
{ type: 'subscribe', deploymentId: string }

// 接收消息
interface WebSocketMessage {
  type: 'log' | 'progress' | 'status' | 'metric';
  deploymentId: string;
  data: any;
}

// 日志消息
{
  type: 'log',
  deploymentId: 'xxx',
  data: {
    timestamp: string;
    level: 'info' | 'warn' | 'error';
    message: string;
    serverId: string;
  }
}

// 进度消息
{
  type: 'progress',
  deploymentId: 'xxx',
  data: {
    serverId: string;
    stepId: string;
    progress: number;
    status: 'pending' | 'running' | 'completed' | 'failed';
  }
}

// 状态消息
{
  type: 'status',
  deploymentId: 'xxx',
  data: {
    status: 'draft' | 'deploying' | 'running' | 'failed' | 'stopped';
    endpoints: DeploymentEndpoint[];
  }
}
```

## 后端实现指南

### 技术栈

- **Runtime**: Node.js 18+ / Python 3.10+
- **Framework**: Express.js / FastAPI
- **SSH**: node-ssh / paramiko
- **Docker**: dockerode / docker SDK
- **WebSocket**: ws / socket.io

### 核心模块

#### 1. SSH 连接管理器

```typescript
class SSHManager {
  // 建立 SSH 连接
  async connect(config: ServerConfig): Promise<SSHClient>;

  // 通过跳板机连接
  async connectViaJumpHost(
    target: ServerConfig,
    jumpHost: JumpHost
  ): Promise<SSHClient>;

  // 执行命令
  async exec(command: string): Promise<{ stdout: string; stderr: string }>;

  // 上传文件
  async uploadFile(local: string, remote: string): Promise<void>;

  // 下载文件
  async downloadFile(remote: string, local: string): Promise<void>;
}
```

#### 2. GPU 信息采集

```typescript
class GPUCollector {
  // 获取 NVIDIA GPU 信息
  async getNvidiaInfo(ssh: SSHClient): Promise<GPUInfo[]> {
    const cmd = 'nvidia-smi --query-gpu=index,name,memory.total,memory.used,memory.free,utilization.gpu,temperature.gpu,power.draw,power.limit --format=csv,noheader,nounits';
    // 解析 CSV 输出
  }

  // 获取昇腾 NPU 信息
  async getNPUInfo(ssh: SSHClient): Promise<NPUInfo[]> {
    const cmd = 'npu-smi info -t';
    // 解析输出
  }
}
```

#### 3. Docker 控制器

```typescript
class DockerController {
  // 拉取镜像
  async pullImage(image: string, registry?: string): Promise<void>;

  // 创建并启动容器
  async createContainer(config: ContainerConfig): Promise<string>;

  // 停止容器
  async stopContainer(containerId: string): Promise<void>;

  // 获取容器日志
  async getLogs(containerId: string, tail?: number): Promise<string>;

  // 检查容器健康状态
  async healthCheck(containerId: string): Promise<boolean>;
}
```

#### 4. 部署执行器

```typescript
class DeploymentExecutor {
  // 执行部署步骤
  async execute(deployment: DeploymentConfig): Promise<void> {
    for (const serverId of deployment.servers) {
      await this.deployToServer(deployment, serverId);
    }
  }

  // 单服务器部署流程
  private async deployToServer(
    deployment: DeploymentConfig,
    serverId: string
  ): Promise<void> {
    // 1. 检查环境
    await this.checkEnvironment(serverId);

    // 2. 拉取镜像
    await this.pullImage(serverId, deployment.docker);

    // 3. 准备模型
    await this.prepareModel(serverId, deployment.model);

    // 4. 启动容器
    await this.startContainer(serverId, deployment);

    // 5. 健康检查
    await this.healthCheck(serverId, deployment);
  }
}
```

#### 5. 模型下载器

```typescript
class ModelDownloader {
  // 从 ModelScope 下载
  async downloadFromModelScope(
    modelId: string,
    revision: string,
    targetPath: string
  ): Promise<void> {
    // 使用 modelscope SDK 或 git lfs
  }

  // 从 HuggingFace 下载
  async downloadFromHuggingFace(
    modelId: string,
    revision: string,
    targetPath: string
  ): Promise<void> {
    // 使用 huggingface-cli 或 git lfs
  }

  // 计算模型大小
  async calculateSize(path: string): Promise<number>;
}
```

### 部署流程

```
1. 环境检查
   ├── 检查 Docker 是否安装
   ├── 检查 NVIDIA Docker 运行时
   ├── 检查 GPU 驱动版本
   └── 检查磁盘空间

2. 镜像准备
   ├── 登录镜像仓库（如需要）
   ├── 拉取指定镜像
   └── 验证镜像完整性

3. 模型准备
   ├── 检查本地模型是否存在
   ├── 如不存在，从源下载
   ├── 验证模型文件完整性
   └── 计算模型大小和格式

4. 容器启动
   ├── 生成 Docker 运行参数
   ├── 挂载数据卷
   ├── 设置环境变量
   ├── 启动容器
   └── 等待服务就绪

5. 健康检查
   ├── 检查容器运行状态
   ├── 检查 API 端口可访问
   ├── 发送测试请求
   └── 更新部署状态
```

### 数据存储

```typescript
// 使用 SQLite/PostgreSQL 存储
interface DatabaseSchema {
  servers: ServerConfig[];
  jump_hosts: JumpHost[];
  models: ModelConfig[];
  deployments: DeploymentConfig[];
  deployment_logs: DeploymentLog[];
  deployment_tasks: DeploymentTask[];
}
```

### 安全配置

1. **SSH 密钥管理**: 使用加密存储私钥
2. **密码加密**: 使用 AES-256 加密存储密码
3. **API 认证**: JWT Token 或 Session
4. **命令注入防护**: 严格校验所有输入参数
5. **权限控制**: 区分管理员和普通用户

### 监控告警

```typescript
interface MonitorConfig {
  // GPU 温度告警阈值
  gpuTempThreshold: number;
  // GPU 内存告警阈值
  gpuMemoryThreshold: number;
  // 服务健康检查间隔
  healthCheckInterval: number;
  // 日志保留天数
  logRetentionDays: number;
}
```

## Remote Task Dispatch

The application now includes a generic remote task dispatch capability for
robots/servers. This is separate from deployment tasks and is designed for
operational actions such as:

- dispatching a raw shell command to one, many, or all robots
- downloading a remote shell script URL and executing it remotely
- running built-in operational presets like Docker-based NPU exporter install

### REST Endpoints

```typescript
GET /api/remote-task-presets
Response: RemoteTaskPreset[]

GET /api/remote-tasks
Response: RemoteTask[]

POST /api/remote-tasks
Body: {
  name?: string;
  description?: string;
  projectId?: string;
  scope: 'all' | 'project' | 'selected';
  executionType: 'command' | 'script_url' | 'preset';
  command?: string;
  scriptUrl?: string;
  scriptArgs?: string;
  presetId?: string;
  presetArgs?: Record<string, string>;
  serverIds?: string[];
}
Response: RemoteTask

GET /api/remote-tasks/:id
Response: RemoteTask
```

### Supported Presets

- `docker_install_npu_exporter`
  Default image: `swr.cn-south-1.myhuaweicloud.com/ascendhub/npu-exporter:v2.0.1`
- `docker_pull_image`
- `docker_restart_service`

### Remote Task Result Model

Each task stores:

- top-level execution metadata and target scope
- a resolved command preview
- per-server run status (`pending`, `running`, `completed`, `failed`)
- per-server stdout/error text and timestamps
