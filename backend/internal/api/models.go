package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"modelrun/backend/internal/domain"
	"modelrun/backend/internal/store"
)

func (a *API) handleModels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		data := a.store.Snapshot()
		models := data.Models
		if source := r.URL.Query().Get("source"); source != "" {
			models = make([]domain.ModelConfig, 0, len(data.Models))
			for _, model := range data.Models {
				if model.Source == source {
					models = append(models, model)
				}
			}
		}
		writeJSON(w, http.StatusOK, models)
	case http.MethodPost:
		var model domain.ModelConfig
		if err := readJSON(r, &model); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		defaultModel(&model)

		if model.Source == "local" && model.LocalPath != "" && len(model.Files) == 0 {
			if files, err := scanModelFiles(model.LocalPath); err == nil {
				model.Files = files
				model.Size = sumModelSize(files)
			}
		}

		if err := a.store.Update(func(data *domain.Data) error {
			data.Models = append(data.Models, model)
			return nil
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusCreated, model)
	default:
		methodNotAllowed(w)
	}
}

func (a *API) handleModel(w http.ResponseWriter, r *http.Request) {
	id, rest := pathParts(r.URL.Path, "/api/models/")
	if id == "" || len(rest) != 0 {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		data := a.store.Snapshot()
		idx := findModel(data.Models, id)
		if idx < 0 {
			writeError(w, http.StatusNotFound, store.ErrNotFound)
			return
		}
		writeJSON(w, http.StatusOK, data.Models[idx])
	case http.MethodPut, http.MethodPatch:
		var patch map[string]json.RawMessage
		if err := readJSON(r, &patch); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		var model domain.ModelConfig
		err := a.store.Update(func(data *domain.Data) error {
			idx := findModel(data.Models, id)
			if idx < 0 {
				return store.ErrNotFound
			}
			updated, err := mergeJSON(data.Models[idx], patch)
			if err != nil {
				return err
			}
			updated.ID = data.Models[idx].ID
			defaultModel(&updated)
			data.Models[idx] = updated
			model = updated
			return nil
		})
		if err != nil {
			writeStoreError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, model)
	case http.MethodDelete:
		err := a.store.Update(func(data *domain.Data) error {
			idx := findModel(data.Models, id)
			if idx < 0 {
				return store.ErrNotFound
			}
			data.Models = append(data.Models[:idx], data.Models[idx+1:]...)
			return nil
		})
		if err != nil {
			writeStoreError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		methodNotAllowed(w)
	}
}

func (a *API) handleModelScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.Path) == "" {
		writeError(w, http.StatusBadRequest, errors.New("path is required"))
		return
	}

	files, err := scanModelFiles(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, files)
}

func (a *API) handleModelSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	source := r.URL.Query().Get("source")
	q := strings.ToLower(r.URL.Query().Get("q"))
	results := []map[string]any{
		{"id": "qwen/Qwen2-72B-Instruct", "name": "Qwen2-72B-Instruct", "source": "modelscope", "downloads": 1800000},
		{"id": "Qwen/Qwen2.5-7B-Instruct", "name": "Qwen2.5-7B-Instruct", "source": "huggingface", "downloads": 2500000},
		{"id": "deepseek-ai/DeepSeek-R1-Distill-Qwen-32B", "name": "DeepSeek-R1-Distill-Qwen-32B", "source": "huggingface", "downloads": 900000},
		{"id": "LLM-Research/Meta-Llama-3-8B-Instruct", "name": "Meta-Llama-3-8B-Instruct", "source": "modelscope", "downloads": 780000},
	}

	filtered := make([]map[string]any, 0, len(results))
	for _, result := range results {
		resultSource, _ := result["source"].(string)
		name, _ := result["name"].(string)
		id, _ := result["id"].(string)
		if source != "" && resultSource != source {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(name+" "+id), q) {
			continue
		}
		filtered = append(filtered, result)
	}

	writeJSON(w, http.StatusOK, filtered)
}

func scanModelFiles(root string) ([]domain.ModelFile, error) {
	root = filepath.Clean(root)
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []domain.ModelFile{{Name: filepath.Base(root), Path: filepath.Base(root), Size: info.Size()}}, nil
	}

	files := make([]domain.ModelFile, 0)
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, domain.ModelFile{
			Name: filepath.Base(path),
			Path: filepath.ToSlash(rel),
			Size: info.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files, nil
}

func sumModelSize(files []domain.ModelFile) int64 {
	var total int64
	for _, file := range files {
		total += file.Size
	}
	return total
}

func findModel(models []domain.ModelConfig, id string) int {
	for i, model := range models {
		if model.ID == id || model.ModelID == id {
			return i
		}
	}
	return -1
}
