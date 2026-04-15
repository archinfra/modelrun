package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"modelrun/backend/internal/domain"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	mu   sync.RWMutex
	path string
	data domain.Data
}

func New(path string) (*Store, error) {
	if path == "" {
		path = "data/modelrun.json"
	}

	st := &Store{path: path}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			st.data = domain.Data{}
			ensureSlices(&st.data)
			return st, st.saveLocked()
		}
		return nil, err
	}

	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &st.data); err != nil {
			return nil, err
		}
	}
	ensureSlices(&st.data)

	return st, nil
}

func (s *Store) Snapshot() domain.Data {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneData(s.data)
}

func (s *Store) Update(fn func(*domain.Data) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := fn(&s.data); err != nil {
		return err
	}
	ensureSlices(&s.data)

	return s.saveLocked()
}

func (s *Store) saveLocked() error {
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, append(raw, '\n'), 0644)
}

func ensureSlices(data *domain.Data) {
	if data.Projects == nil {
		data.Projects = []domain.Project{}
	}
	if data.Servers == nil {
		data.Servers = []domain.ServerConfig{}
	}
	if data.JumpHosts == nil {
		data.JumpHosts = []domain.JumpHost{}
	}
	if data.Models == nil {
		data.Models = []domain.ModelConfig{}
	}
	if data.Deployments == nil {
		data.Deployments = []domain.DeploymentConfig{}
	}
	if data.Tasks == nil {
		data.Tasks = []domain.DeploymentTask{}
	}
	if data.Logs == nil {
		data.Logs = []domain.DeploymentLog{}
	}
}

func cloneData(data domain.Data) domain.Data {
	raw, err := json.Marshal(data)
	if err != nil {
		return domain.Data{}
	}

	var cloned domain.Data
	if err := json.Unmarshal(raw, &cloned); err != nil {
		return domain.Data{}
	}
	ensureSlices(&cloned)

	return cloned
}
