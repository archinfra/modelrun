package collect

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"modelrun/backend/internal/domain"

	"golang.org/x/crypto/ssh"
)

type SSHConfig struct {
	ID         string
	Name       string
	Host       string
	SSHPort    int
	Username   string
	AuthType   string
	Password   string
	PrivateKey string
}

type Snapshot struct {
	Message       string
	Accelerators  []domain.GPUInfo
	Resources     domain.ServerResource
	DriverVersion string
	CUDAVersion   string
	DockerVersion string
}

type Collector struct {
	timeout time.Duration
}

func New() *Collector {
	return &Collector{timeout: 8 * time.Second}
}

func FromServer(server domain.ServerConfig) SSHConfig {
	return SSHConfig{
		ID:         server.ID,
		Name:       server.Name,
		Host:       server.Host,
		SSHPort:    server.SSHPort,
		Username:   server.Username,
		AuthType:   server.AuthType,
		Password:   server.Password,
		PrivateKey: server.PrivateKey,
	}
}

func FromJumpHost(host domain.JumpHost) SSHConfig {
	return SSHConfig{
		ID:         host.ID,
		Name:       host.Name,
		Host:       host.Host,
		SSHPort:    host.SSHPort,
		Username:   host.Username,
		AuthType:   host.AuthType,
		Password:   host.Password,
		PrivateKey: host.PrivateKey,
	}
}

func IsMockServer(server domain.ServerConfig) bool {
	return os.Getenv("MODELRUN_FAKE_CONNECT") == "1" || strings.HasPrefix(server.Host, "mock")
}

func MockSnapshot(server domain.ServerConfig) Snapshot {
	accelerators := MockAccelerators(server.ID)
	return Snapshot{
		Message:       "mock connection succeeded",
		Accelerators:  accelerators,
		Resources:     MockResources(accelerators),
		DriverVersion: "535.104.05",
		CUDAVersion:   "12.2",
		DockerVersion: "24.0.7",
	}
}

func MockAccelerators(seed string) []domain.GPUInfo {
	if len(seed)%2 == 0 {
		return []domain.GPUInfo{
			{Index: 0, Type: "gpu", Name: "NVIDIA A100 80GB", MemoryTotal: 81920, MemoryUsed: 24576, MemoryFree: 57344, Utilization: 45, Temperature: 72, PowerDraw: 285, PowerLimit: 400},
			{Index: 1, Type: "gpu", Name: "NVIDIA A100 80GB", MemoryTotal: 81920, MemoryUsed: 18432, MemoryFree: 63488, Utilization: 32, Temperature: 68, PowerDraw: 240, PowerLimit: 400},
		}
	}
	return []domain.GPUInfo{
		{Index: 0, Type: "npu", Name: "Ascend 910B", MemoryTotal: 65536, MemoryUsed: 8192, MemoryFree: 57344, Utilization: 28, Temperature: 65, PowerDraw: 180, Health: "OK"},
	}
}

func MockResources(accelerators []domain.GPUInfo) domain.ServerResource {
	var resource domain.ServerResource
	resource.CPU.Cores = maxInt(8, len(accelerators)*16)
	resource.CPU.Usage = 22.5
	resource.Memory.Total = int64(resource.CPU.Cores) * 4096
	resource.Memory.Used = resource.Memory.Total / 3
	resource.Memory.Free = resource.Memory.Total - resource.Memory.Used
	resource.Disk.Total = 4 * 1024 * 1024
	resource.Disk.Used = resource.Disk.Total / 2
	resource.Disk.Free = resource.Disk.Total - resource.Disk.Used
	resource.Network.RXSpeed = 0
	resource.Network.TXSpeed = 0
	return resource
}

func (c *Collector) Collect(server domain.ServerConfig, jump *SSHConfig) (Snapshot, error) {
	if IsMockServer(server) {
		return MockSnapshot(server), nil
	}

	client, closeFn, err := c.dial(FromServer(server), jump)
	if err != nil {
		return Snapshot{}, err
	}
	defer closeFn()

	accelerators, driver, cuda, err := collectAccelerators(client, server.NPUExporterEndpoint)
	if err != nil {
		return Snapshot{}, err
	}
	resources, err := collectResources(client)
	if err != nil {
		return Snapshot{}, err
	}
	dockerVersion := collectDockerVersion(client)

	return Snapshot{
		Message:       connectionMessage(jump),
		Accelerators:  accelerators,
		Resources:     resources,
		DriverVersion: driver,
		CUDAVersion:   cuda,
		DockerVersion: dockerVersion,
	}, nil
}

func (c *Collector) Accelerators(server domain.ServerConfig, jump *SSHConfig) ([]domain.GPUInfo, error) {
	if IsMockServer(server) {
		return MockAccelerators(server.ID), nil
	}

	client, closeFn, err := c.dial(FromServer(server), jump)
	if err != nil {
		return nil, err
	}
	defer closeFn()

	accelerators, _, _, err := collectAccelerators(client, server.NPUExporterEndpoint)
	return accelerators, err
}

func (c *Collector) Resources(server domain.ServerConfig, jump *SSHConfig) (domain.ServerResource, error) {
	if IsMockServer(server) {
		return MockResources(server.GPUInfo), nil
	}

	client, closeFn, err := c.dial(FromServer(server), jump)
	if err != nil {
		return domain.ServerResource{}, err
	}
	defer closeFn()

	return collectResources(client)
}

func (c *Collector) dial(target SSHConfig, jump *SSHConfig) (*ssh.Client, func(), error) {
	if target.Host == "" {
		return nil, nil, errors.New("server host is empty")
	}

	targetConfig, err := sshClientConfig(target, c.timeout)
	if err != nil {
		return nil, nil, fmt.Errorf("target ssh config: %w", err)
	}
	targetAddr := net.JoinHostPort(target.Host, strconv.Itoa(defaultPort(target.SSHPort)))

	if jump == nil {
		client, err := ssh.Dial("tcp", targetAddr, targetConfig)
		if err != nil {
			return nil, nil, fmt.Errorf("ssh connect %s: %w", targetAddr, err)
		}
		return client, func() { _ = client.Close() }, nil
	}

	jumpConfig, err := sshClientConfig(*jump, c.timeout)
	if err != nil {
		return nil, nil, fmt.Errorf("jump ssh config: %w", err)
	}
	jumpAddr := net.JoinHostPort(jump.Host, strconv.Itoa(defaultPort(jump.SSHPort)))
	jumpClient, err := ssh.Dial("tcp", jumpAddr, jumpConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("ssh connect jump host %s: %w", jumpAddr, err)
	}

	conn, err := jumpClient.Dial("tcp", targetAddr)
	if err != nil {
		_ = jumpClient.Close()
		return nil, nil, fmt.Errorf("jump host dial target %s: %w", targetAddr, err)
	}

	clientConn, chans, reqs, err := ssh.NewClientConn(conn, targetAddr, targetConfig)
	if err != nil {
		_ = conn.Close()
		_ = jumpClient.Close()
		return nil, nil, fmt.Errorf("ssh handshake target %s via jump host: %w", targetAddr, err)
	}

	client := ssh.NewClient(clientConn, chans, reqs)
	return client, func() {
		_ = client.Close()
		_ = jumpClient.Close()
	}, nil
}

func sshClientConfig(endpoint SSHConfig, timeout time.Duration) (*ssh.ClientConfig, error) {
	auth := []ssh.AuthMethod{}
	if strings.TrimSpace(endpoint.PrivateKey) != "" {
		signer, err := ssh.ParsePrivateKey([]byte(endpoint.PrivateKey))
		if err != nil && endpoint.Password != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(endpoint.PrivateKey), []byte(endpoint.Password))
		}
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}
	if endpoint.Password != "" {
		auth = append(auth, ssh.Password(endpoint.Password))
		auth = append(auth, ssh.KeyboardInteractive(func(_ string, _ string, questions []string, _ []bool) ([]string, error) {
			answers := make([]string, len(questions))
			for i := range answers {
				answers[i] = endpoint.Password
			}
			return answers, nil
		}))
	}
	if len(auth) == 0 {
		return nil, errors.New("password or privateKey is required")
	}
	username := endpoint.Username
	if username == "" {
		username = "root"
	}

	return &ssh.ClientConfig{
		User:            username,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}, nil
}

func run(client sshRunner, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(command); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return strings.TrimSpace(stdout.String()), errors.New(message)
	}

	return strings.TrimSpace(stdout.String()), nil
}

func hasCommand(client *ssh.Client, command string) bool {
	_, err := run(client, "command -v "+shellQuote(command)+" >/dev/null 2>&1")
	return err == nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func defaultPort(port int) int {
	if port == 0 {
		return 22
	}
	return port
}

func connectionMessage(jump *SSHConfig) string {
	if jump == nil {
		return "ssh connection succeeded"
	}
	return fmt.Sprintf("ssh connection succeeded via jump host %s", jump.Name)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
