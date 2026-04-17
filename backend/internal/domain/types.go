package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

type Project struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Color       string   `json:"color"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
	ServerIDs   []string `json:"serverIds"`
}

type ServerConfig struct {
	ID                   string    `json:"id"`
	ProjectID            string    `json:"projectId"`
	Name                 string    `json:"name"`
	Host                 string    `json:"host"`
	SSHPort              int       `json:"sshPort"`
	Username             string    `json:"username"`
	AuthType             string    `json:"authType"`
	Password             string    `json:"password,omitempty"`
	PrivateKey           string    `json:"privateKey,omitempty"`
	IsJumpHost           bool      `json:"isJumpHost,omitempty"`
	UseJumpHost          bool      `json:"useJumpHost"`
	JumpHostID           string    `json:"jumpHostId,omitempty"`
	GPUInfo              []GPUInfo `json:"gpuInfo,omitempty"`
	DriverVersion        string    `json:"driverVersion,omitempty"`
	CUDAVersion          string    `json:"cudaVersion,omitempty"`
	DockerVersion        string    `json:"dockerVersion,omitempty"`
	NPUExporterEndpoint  string    `json:"npuExporterEndpoint,omitempty"`
	NPUExporterStatus    string    `json:"npuExporterStatus,omitempty"`
	NPUExporterLastCheck string    `json:"npuExporterLastCheck,omitempty"`
	NetdataEndpoint      string    `json:"netdataEndpoint,omitempty"`
	NetdataStatus        string    `json:"netdataStatus,omitempty"`
	NetdataLastCheck     string    `json:"netdataLastCheck,omitempty"`
	LastCheck            string    `json:"lastCheck,omitempty"`
	Status               string    `json:"status"`
}

type GPUInfo struct {
	Index       int     `json:"index"`
	Type        string  `json:"type,omitempty"`
	Name        string  `json:"name"`
	MemoryTotal int64   `json:"memoryTotal"`
	MemoryUsed  int64   `json:"memoryUsed"`
	MemoryFree  int64   `json:"memoryFree"`
	Utilization float64 `json:"utilization"`
	Temperature float64 `json:"temperature"`
	PowerDraw   float64 `json:"powerDraw"`
	PowerLimit  float64 `json:"powerLimit"`
	Health      string  `json:"health,omitempty"`
	LogicID     int     `json:"logicId,omitempty"`
	ChipID      int     `json:"chipId,omitempty"`
}

type JumpHost struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Host       string `json:"host"`
	SSHPort    int    `json:"sshPort"`
	Username   string `json:"username"`
	AuthType   string `json:"authType"`
	Password   string `json:"password,omitempty"`
	PrivateKey string `json:"privateKey,omitempty"`
}

type ModelConfig struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Source       string      `json:"source"`
	ModelID      string      `json:"modelId"`
	LocalPath    string      `json:"localPath,omitempty"`
	Revision     string      `json:"revision,omitempty"`
	Size         int64       `json:"size,omitempty"`
	Format       string      `json:"format,omitempty"`
	Parameters   string      `json:"parameters,omitempty"`
	Quantization string      `json:"quantization,omitempty"`
	Files        []ModelFile `json:"files,omitempty"`
}

type ModelFile struct {
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Path     string `json:"path"`
	Checksum string `json:"checksum,omitempty"`
}

type DockerConfig struct {
	Image           string            `json:"image"`
	Registry        string            `json:"registry,omitempty"`
	Tag             string            `json:"tag"`
	GPUDevices      string            `json:"gpuDevices"`
	ShmSize         string            `json:"shmSize"`
	EnvironmentVars map[string]string `json:"environmentVars"`
	Volumes         []VolumeMount     `json:"volumes"`
	Network         string            `json:"network,omitempty"`
	IPC             string            `json:"ipc,omitempty"`
	Privileged      bool              `json:"privileged,omitempty"`
	Runtime         string            `json:"runtime,omitempty"`
}

type VolumeMount struct {
	Host      string `json:"host"`
	Container string `json:"container"`
}

type VLLMParams struct {
	TensorParallelSize   int     `json:"tensorParallelSize"`
	PipelineParallelSize int     `json:"pipelineParallelSize"`
	MaxModelLen          int     `json:"maxModelLen"`
	GPUMemoryUtilization float64 `json:"gpuMemoryUtilization"`
	Quantization         string  `json:"quantization,omitempty"`
	Dtype                string  `json:"dtype"`
	TrustRemoteCode      bool    `json:"trustRemoteCode"`
	EnablePrefixCaching  bool    `json:"enablePrefixCaching"`
	EnableExpertParallel bool    `json:"enableExpertParallel,omitempty"`
	MaxNumSeqs           int     `json:"maxNumSeqs"`
	MaxNumBatchedTokens  int     `json:"maxNumBatchedTokens"`
	SwapSpace            int     `json:"swapSpace,omitempty"`
	EnforceEager         bool    `json:"enforceEager,omitempty"`
	EnableChunkedPrefill bool    `json:"enableChunkedPrefill,omitempty"`
	SpeculativeModel     string  `json:"speculativeModel,omitempty"`
	NumSpeculativeTokens int     `json:"numSpeculativeTokens,omitempty"`
}

type DeploymentServerOverride struct {
	ServerID       string   `json:"serverId"`
	NodeIP         string   `json:"nodeIp,omitempty"`
	VisibleDevices string   `json:"visibleDevices,omitempty"`
	RayStartArgs   []string `json:"rayStartArgs,omitempty"`
}

type DeploymentRayConfig struct {
	Enabled        bool   `json:"enabled"`
	HeadServerID   string `json:"headServerId,omitempty"`
	NICName        string `json:"nicName,omitempty"`
	Port           int    `json:"port,omitempty"`
	DashboardPort  int    `json:"dashboardPort,omitempty"`
	VisibleDevices string `json:"visibleDevices,omitempty"`
}

type DeploymentRuntimeConfig struct {
	ContainerName     string   `json:"containerName,omitempty"`
	WorkDir           string   `json:"workDir,omitempty"`
	ModelDir          string   `json:"modelDir,omitempty"`
	CacheDir          string   `json:"cacheDir,omitempty"`
	SharedCacheDir    string   `json:"sharedCacheDir,omitempty"`
	EnableAutoRestart bool     `json:"enableAutoRestart"`
	ExtraArgs         []string `json:"extraArgs,omitempty"`
}

type DeploymentConfig struct {
	ID              string                     `json:"id"`
	Name            string                     `json:"name"`
	Status          string                     `json:"status"`
	Framework       string                     `json:"framework,omitempty"`
	Model           ModelConfig                `json:"model"`
	Docker          DockerConfig               `json:"docker"`
	VLLM            VLLMParams                 `json:"vllm"`
	Ray             DeploymentRayConfig        `json:"ray,omitempty"`
	Runtime         DeploymentRuntimeConfig    `json:"runtime,omitempty"`
	ServerOverrides []DeploymentServerOverride `json:"serverOverrides,omitempty"`
	Servers         []string                   `json:"servers"`
	APIPort         int                        `json:"apiPort"`
	CreatedAt       string                     `json:"createdAt"`
	UpdatedAt       string                     `json:"updatedAt"`
	Endpoints       []DeploymentEndpoint       `json:"endpoints,omitempty"`
	Metrics         *DeploymentMetrics         `json:"metrics,omitempty"`
}

type DeploymentEndpoint struct {
	ServerID string  `json:"serverId"`
	URL      string  `json:"url"`
	Status   string  `json:"status"`
	Latency  float64 `json:"latency,omitempty"`
}

type DeploymentMetrics struct {
	TotalRequests     int64   `json:"totalRequests"`
	AvgLatency        float64 `json:"avgLatency"`
	TokensPerSecond   float64 `json:"tokensPerSecond"`
	GPUUtilization    float64 `json:"gpuUtilization"`
	MemoryUtilization float64 `json:"memoryUtilization"`
}

type DeploymentStep struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	CommandPreview string   `json:"commandPreview,omitempty"`
	Optional       bool     `json:"optional,omitempty"`
	AutoManaged    bool     `json:"autoManaged,omitempty"`
	Status         string   `json:"status"`
	Progress       int      `json:"progress"`
	Logs           []string `json:"logs"`
	StartTime      string   `json:"startTime,omitempty"`
	EndTime        string   `json:"endTime,omitempty"`
}

type DeploymentTask struct {
	ID              string           `json:"id"`
	DeploymentID    string           `json:"deploymentId"`
	ServerID        string           `json:"serverId"`
	Steps           []DeploymentStep `json:"steps"`
	CurrentStep     int              `json:"currentStep"`
	OverallProgress int              `json:"overallProgress"`
}

type RemoteTask struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Description    string            `json:"description,omitempty"`
	ProjectID      string            `json:"projectId,omitempty"`
	Scope          string            `json:"scope"`
	Status         string            `json:"status"`
	ExecutionType  string            `json:"executionType"`
	CommandPreview string            `json:"commandPreview,omitempty"`
	ScriptURL      string            `json:"scriptUrl,omitempty"`
	ScriptArgs     string            `json:"scriptArgs,omitempty"`
	PresetID       string            `json:"presetId,omitempty"`
	PresetArgs     map[string]string `json:"presetArgs,omitempty"`
	ServerIDs      []string          `json:"serverIds"`
	Runs           []RemoteTaskRun   `json:"runs"`
	CreatedAt      string            `json:"createdAt"`
	StartedAt      string            `json:"startedAt,omitempty"`
	FinishedAt     string            `json:"finishedAt,omitempty"`
}

type RemoteTaskRun struct {
	ServerID   string `json:"serverId"`
	ServerName string `json:"serverName,omitempty"`
	Status     string `json:"status"`
	Command    string `json:"command,omitempty"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
	ExitCode   int    `json:"exitCode,omitempty"`
	StartedAt  string `json:"startedAt,omitempty"`
	FinishedAt string `json:"finishedAt,omitempty"`
}

type RemoteTaskPreset struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Fields      []RemoteTaskPresetField `json:"fields,omitempty"`
}

type RemoteTaskPresetField struct {
	Key          string `json:"key"`
	Label        string `json:"label"`
	Description  string `json:"description,omitempty"`
	Required     bool   `json:"required,omitempty"`
	DefaultValue string `json:"defaultValue,omitempty"`
	Placeholder  string `json:"placeholder,omitempty"`
}

type ActionTemplate struct {
	ID                 string                `json:"id"`
	Name               string                `json:"name"`
	Description        string                `json:"description"`
	Category           string                `json:"category,omitempty"`
	BuiltIn            bool                  `json:"builtIn,omitempty"`
	ExecutionType      string                `json:"executionType"`
	CommandTemplate    string                `json:"commandTemplate,omitempty"`
	ScriptURL          string                `json:"scriptUrl,omitempty"`
	ScriptArgsTemplate string                `json:"scriptArgsTemplate,omitempty"`
	Fields             []ActionTemplateField `json:"fields,omitempty"`
	Tags               []string              `json:"tags,omitempty"`
	CreatedAt          string                `json:"createdAt"`
	UpdatedAt          string                `json:"updatedAt"`
}

type ActionTemplateField struct {
	Key          string `json:"key"`
	Label        string `json:"label"`
	Description  string `json:"description,omitempty"`
	Required     bool   `json:"required,omitempty"`
	DefaultValue string `json:"defaultValue,omitempty"`
	Placeholder  string `json:"placeholder,omitempty"`
}

type BootstrapConfig struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	ServiceType      string            `json:"serviceType"`
	Category         string            `json:"category,omitempty"`
	BuiltIn          bool              `json:"builtIn,omitempty"`
	ActionTemplateID string            `json:"actionTemplateId"`
	DefaultArgs      map[string]string `json:"defaultArgs,omitempty"`
	Endpoint         string            `json:"endpoint,omitempty"`
	Port             int               `json:"port,omitempty"`
	CreatedAt        string            `json:"createdAt"`
	UpdatedAt        string            `json:"updatedAt"`
}

type PipelineTemplate struct {
	ID             string                  `json:"id"`
	Name           string                  `json:"name"`
	Framework      string                  `json:"framework"`
	Description    string                  `json:"description"`
	SupportsRay    bool                    `json:"supportsRay,omitempty"`
	DefaultPort    int                     `json:"defaultPort"`
	DefaultDocker  DockerConfig            `json:"defaultDocker"`
	DefaultVLLM    VLLMParams              `json:"defaultVllm"`
	DefaultRay     DeploymentRayConfig     `json:"defaultRay"`
	DefaultRuntime DeploymentRuntimeConfig `json:"defaultRuntime"`
	Steps          []PipelineTemplateStep  `json:"steps"`
}

type PipelineTemplateStep struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Optional    bool     `json:"optional,omitempty"`
	AutoManaged bool     `json:"autoManaged,omitempty"`
	Details     []string `json:"details,omitempty"`
}

type PipelineStepTemplate struct {
	ID              string   `json:"id"`
	Framework       string   `json:"framework"`
	StepID          string   `json:"stepId"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Optional        bool     `json:"optional,omitempty"`
	AutoManaged     bool     `json:"autoManaged,omitempty"`
	BuiltIn         bool     `json:"builtIn,omitempty"`
	CommandTemplate string   `json:"commandTemplate"`
	PreviewTemplate string   `json:"previewTemplate,omitempty"`
	Details         []string `json:"details,omitempty"`
	CreatedAt       string   `json:"createdAt"`
	UpdatedAt       string   `json:"updatedAt"`
}

type ServerResource struct {
	CPU struct {
		Cores int     `json:"cores"`
		Usage float64 `json:"usage"`
	} `json:"cpu"`
	Memory struct {
		Total int64 `json:"total"`
		Used  int64 `json:"used"`
		Free  int64 `json:"free"`
	} `json:"memory"`
	Disk struct {
		Total int64 `json:"total"`
		Used  int64 `json:"used"`
		Free  int64 `json:"free"`
	} `json:"disk"`
	Network struct {
		RXSpeed float64 `json:"rxSpeed"`
		TXSpeed float64 `json:"txSpeed"`
	} `json:"network"`
}

type DeploymentLog struct {
	Timestamp    string `json:"timestamp"`
	Level        string `json:"level"`
	Message      string `json:"message"`
	DeploymentID string `json:"deploymentId,omitempty"`
	ServerID     string `json:"serverId,omitempty"`
	StepID       string `json:"stepId,omitempty"`
}

type Data struct {
	Projects         []Project              `json:"projects"`
	Servers          []ServerConfig         `json:"servers"`
	JumpHosts        []JumpHost             `json:"jumpHosts"`
	Models           []ModelConfig          `json:"models"`
	Deployments      []DeploymentConfig     `json:"deployments"`
	Tasks            []DeploymentTask       `json:"tasks"`
	RemoteTasks      []RemoteTask           `json:"remoteTasks"`
	ActionTemplates  []ActionTemplate       `json:"actionTemplates"`
	BootstrapConfigs []BootstrapConfig      `json:"bootstrapConfigs"`
	PipelineSteps    []PipelineStepTemplate `json:"pipelineSteps"`
	Logs             []DeploymentLog        `json:"logs"`
}

func NewID(prefix string) string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err == nil {
		return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(b[:]))
	}
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

func Now() string {
	return time.Now().UTC().Format(time.RFC3339)
}
