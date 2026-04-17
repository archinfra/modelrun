package runtimefiles

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type StepMeta struct {
	DeploymentID   string `json:"deploymentId"`
	ServerID       string `json:"serverId"`
	StepID         string `json:"stepId"`
	StepName       string `json:"stepName"`
	Description    string `json:"description,omitempty"`
	CommandPreview string `json:"commandPreview,omitempty"`
	Command        string `json:"command,omitempty"`
	StartedAt      string `json:"startedAt,omitempty"`
}

type Manager struct {
	baseDir string
	mu      sync.Mutex
}

func New(baseDir string) (*Manager, error) {
	if strings.TrimSpace(baseDir) == "" {
		baseDir = filepath.Join("data", "runs")
	}
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}
	return &Manager{baseDir: baseDir}, nil
}

func (m *Manager) BaseDir() string {
	if m == nil {
		return ""
	}
	return m.baseDir
}

func (m *Manager) ResetDeployment(deploymentID string) error {
	if m == nil || strings.TrimSpace(deploymentID) == "" {
		return nil
	}
	target := m.deploymentDir(deploymentID)
	if err := os.RemoveAll(target); err != nil {
		return err
	}
	return os.MkdirAll(target, 0755)
}

func (m *Manager) WriteStepMeta(meta StepMeta) error {
	if m == nil {
		return nil
	}
	path := m.stepMetaPath(meta.DeploymentID, meta.ServerID, meta.StepID)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0644)
}

func (m *Manager) AppendStepLines(deploymentID, serverID, stepID string, lines []string) error {
	if m == nil || len(lines) == 0 {
		return nil
	}
	path := m.stepLogPath(deploymentID, serverID, stepID)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	for _, line := range lines {
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			continue
		}
		if _, err := file.WriteString(line + "\n"); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) ReadStepTail(deploymentID, serverID, stepID string, limit int) []string {
	if m == nil {
		return nil
	}
	if limit <= 0 {
		limit = 200
	}
	raw, err := os.ReadFile(m.stepLogPath(deploymentID, serverID, stepID))
	if err != nil || len(raw) == 0 {
		return nil
	}
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		filtered = append(filtered, line)
	}
	if len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}
	return filtered
}

func (m *Manager) deploymentDir(deploymentID string) string {
	return filepath.Join(m.baseDir, sanitizePathSegment(deploymentID))
}

func (m *Manager) stepDir(deploymentID, serverID, stepID string) string {
	return filepath.Join(m.deploymentDir(deploymentID), sanitizePathSegment(serverID), sanitizePathSegment(stepID))
}

func (m *Manager) stepLogPath(deploymentID, serverID, stepID string) string {
	return filepath.Join(m.stepDir(deploymentID, serverID, stepID), "output.log")
}

func (m *Manager) stepMetaPath(deploymentID, serverID, stepID string) string {
	return filepath.Join(m.stepDir(deploymentID, serverID, stepID), "meta.json")
}

func sanitizePathSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	return replacer.Replace(value)
}
