package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"modelrun/backend/internal/catalog"
	"modelrun/backend/internal/domain"

	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	mu   sync.RWMutex
	path string
	db   *sql.DB
	data domain.Data
}

func New(path string) (*Store, error) {
	if path == "" {
		path = "data/modelrun.db"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	st := &Store{path: path, db: db}
	if err := st.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}

	data, err := st.load()
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	if isEmpty(data) {
		if legacy, ok := loadLegacyJSON(path); ok {
			data = legacy
			if err := st.save(data); err != nil {
				_ = db.Close()
				return nil, err
			}
		}
	}
	ensureSlices(&data)
	if catalog.EnsureDefaults(&data) {
		if err := st.save(data); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	st.data = data

	return st, nil
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Snapshot() domain.Data {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneData(s.data)
}

func (s *Store) Update(fn func(*domain.Data) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := cloneData(s.data)
	if err := fn(&next); err != nil {
		return err
	}
	ensureSlices(&next)
	if err := s.save(next); err != nil {
		return err
	}
	s.data = next

	return nil
}

func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS documents (
  kind TEXT NOT NULL,
  id TEXT NOT NULL,
  payload TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (kind, id)
);
CREATE INDEX IF NOT EXISTS idx_documents_kind ON documents(kind);
`)
	return err
}

func (s *Store) load() (domain.Data, error) {
	var data domain.Data
	var err error

	if data.Projects, err = loadCollection[domain.Project](s.db, "projects"); err != nil {
		return data, err
	}
	if data.Servers, err = loadCollection[domain.ServerConfig](s.db, "servers"); err != nil {
		return data, err
	}
	if data.JumpHosts, err = loadCollection[domain.JumpHost](s.db, "jump_hosts"); err != nil {
		return data, err
	}
	if data.Models, err = loadCollection[domain.ModelConfig](s.db, "models"); err != nil {
		return data, err
	}
	if data.Deployments, err = loadCollection[domain.DeploymentConfig](s.db, "deployments"); err != nil {
		return data, err
	}
	if data.Tasks, err = loadCollection[domain.DeploymentTask](s.db, "tasks"); err != nil {
		return data, err
	}
	if data.RemoteTasks, err = loadCollection[domain.RemoteTask](s.db, "remote_tasks"); err != nil {
		return data, err
	}
	if data.ActionTemplates, err = loadCollection[domain.ActionTemplate](s.db, "action_templates"); err != nil {
		return data, err
	}
	if data.BootstrapConfigs, err = loadCollection[domain.BootstrapConfig](s.db, "bootstrap_configs"); err != nil {
		return data, err
	}
	if data.Logs, err = loadCollection[domain.DeploymentLog](s.db, "logs"); err != nil {
		return data, err
	}
	ensureSlices(&data)

	return data, nil
}

func (s *Store) save(data domain.Data) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec("DELETE FROM documents"); err != nil {
		return err
	}

	if err = saveCollection(tx, "projects", data.Projects, func(v domain.Project, _ int) string { return v.ID }); err != nil {
		return err
	}
	if err = saveCollection(tx, "servers", data.Servers, func(v domain.ServerConfig, _ int) string { return v.ID }); err != nil {
		return err
	}
	if err = saveCollection(tx, "jump_hosts", data.JumpHosts, func(v domain.JumpHost, _ int) string { return v.ID }); err != nil {
		return err
	}
	if err = saveCollection(tx, "models", data.Models, func(v domain.ModelConfig, _ int) string { return v.ID }); err != nil {
		return err
	}
	if err = saveCollection(tx, "deployments", data.Deployments, func(v domain.DeploymentConfig, _ int) string { return v.ID }); err != nil {
		return err
	}
	if err = saveCollection(tx, "tasks", data.Tasks, func(v domain.DeploymentTask, _ int) string { return v.ID }); err != nil {
		return err
	}
	if err = saveCollection(tx, "remote_tasks", data.RemoteTasks, func(v domain.RemoteTask, _ int) string { return v.ID }); err != nil {
		return err
	}
	if err = saveCollection(tx, "action_templates", data.ActionTemplates, func(v domain.ActionTemplate, _ int) string { return v.ID }); err != nil {
		return err
	}
	if err = saveCollection(tx, "bootstrap_configs", data.BootstrapConfigs, func(v domain.BootstrapConfig, _ int) string { return v.ID }); err != nil {
		return err
	}
	if err = saveCollection(tx, "logs", data.Logs, func(_ domain.DeploymentLog, i int) string { return fmt.Sprintf("%08d", i) }); err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

func loadCollection[T any](db *sql.DB, kind string) ([]T, error) {
	rows, err := db.Query("SELECT payload FROM documents WHERE kind = ? ORDER BY id", kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []T{}
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var item T
		if err := json.Unmarshal([]byte(raw), &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func saveCollection[T any](tx *sql.Tx, kind string, items []T, idFn func(T, int) string) error {
	stmt, err := tx.Prepare("INSERT INTO documents(kind, id, payload, updated_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, item := range items {
		id := idFn(item, i)
		if id == "" {
			id = fmt.Sprintf("%08d", i)
		}
		raw, err := json.Marshal(item)
		if err != nil {
			return err
		}
		if _, err := stmt.Exec(kind, id, string(raw)); err != nil {
			return err
		}
	}

	return nil
}

func loadLegacyJSON(path string) (domain.Data, bool) {
	legacyPath := filepath.Join(filepath.Dir(path), "modelrun.json")
	raw, err := os.ReadFile(legacyPath)
	if err != nil || len(raw) == 0 {
		return domain.Data{}, false
	}

	var data domain.Data
	if err := json.Unmarshal(raw, &data); err != nil {
		return domain.Data{}, false
	}
	ensureSlices(&data)

	return data, true
}

func isEmpty(data domain.Data) bool {
	return len(data.Projects) == 0 &&
		len(data.Servers) == 0 &&
		len(data.JumpHosts) == 0 &&
		len(data.Models) == 0 &&
		len(data.Deployments) == 0 &&
		len(data.Tasks) == 0 &&
		len(data.RemoteTasks) == 0 &&
		len(data.ActionTemplates) == 0 &&
		len(data.BootstrapConfigs) == 0 &&
		len(data.Logs) == 0
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
	if data.RemoteTasks == nil {
		data.RemoteTasks = []domain.RemoteTask{}
	}
	if data.ActionTemplates == nil {
		data.ActionTemplates = []domain.ActionTemplate{}
	}
	if data.BootstrapConfigs == nil {
		data.BootstrapConfigs = []domain.BootstrapConfig{}
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
