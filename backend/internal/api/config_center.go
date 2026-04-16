package api

import (
	"encoding/json"
	"net/http"
	"sort"

	"modelrun/backend/internal/domain"
	"modelrun/backend/internal/store"
)

func (a *API) handleActionTemplates(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		data := a.store.Snapshot()
		items := append([]domain.ActionTemplate{}, data.ActionTemplates...)
		sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var item domain.ActionTemplate
		if err := readJSON(r, &item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		defaultActionTemplate(&item)
		if err := a.store.Update(func(data *domain.Data) error {
			data.ActionTemplates = append(data.ActionTemplates, item)
			return nil
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, item)
	default:
		methodNotAllowed(w)
	}
}

func (a *API) handleActionTemplate(w http.ResponseWriter, r *http.Request) {
	id, rest := pathParts(r.URL.Path, "/api/action-templates/")
	if id == "" || len(rest) != 0 {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		data := a.store.Snapshot()
		idx := findActionTemplate(data.ActionTemplates, id)
		if idx < 0 {
			writeError(w, http.StatusNotFound, store.ErrNotFound)
			return
		}
		writeJSON(w, http.StatusOK, data.ActionTemplates[idx])
	case http.MethodPut, http.MethodPatch:
		var patch map[string]json.RawMessage
		if err := readJSON(r, &patch); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		var item domain.ActionTemplate
		err := a.store.Update(func(data *domain.Data) error {
			idx := findActionTemplate(data.ActionTemplates, id)
			if idx < 0 {
				return store.ErrNotFound
			}
			updated, err := mergeJSON(data.ActionTemplates[idx], patch)
			if err != nil {
				return err
			}
			updated.ID = data.ActionTemplates[idx].ID
			if updated.CreatedAt == "" {
				updated.CreatedAt = data.ActionTemplates[idx].CreatedAt
			}
			defaultActionTemplate(&updated)
			data.ActionTemplates[idx] = updated
			item = updated
			return nil
		})
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		err := a.store.Update(func(data *domain.Data) error {
			idx := findActionTemplate(data.ActionTemplates, id)
			if idx < 0 {
				return store.ErrNotFound
			}
			data.ActionTemplates = append(data.ActionTemplates[:idx], data.ActionTemplates[idx+1:]...)
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

func (a *API) handleBootstrapConfigs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		data := a.store.Snapshot()
		items := append([]domain.BootstrapConfig{}, data.BootstrapConfigs...)
		sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var item domain.BootstrapConfig
		if err := readJSON(r, &item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		defaultBootstrapConfig(&item)
		if err := a.store.Update(func(data *domain.Data) error {
			data.BootstrapConfigs = append(data.BootstrapConfigs, item)
			return nil
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, item)
	default:
		methodNotAllowed(w)
	}
}

func (a *API) handleBootstrapConfig(w http.ResponseWriter, r *http.Request) {
	id, rest := pathParts(r.URL.Path, "/api/bootstrap-configs/")
	if id == "" || len(rest) != 0 {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		data := a.store.Snapshot()
		idx := findBootstrapConfig(data.BootstrapConfigs, id)
		if idx < 0 {
			writeError(w, http.StatusNotFound, store.ErrNotFound)
			return
		}
		writeJSON(w, http.StatusOK, data.BootstrapConfigs[idx])
	case http.MethodPut, http.MethodPatch:
		var patch map[string]json.RawMessage
		if err := readJSON(r, &patch); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		var item domain.BootstrapConfig
		err := a.store.Update(func(data *domain.Data) error {
			idx := findBootstrapConfig(data.BootstrapConfigs, id)
			if idx < 0 {
				return store.ErrNotFound
			}
			updated, err := mergeJSON(data.BootstrapConfigs[idx], patch)
			if err != nil {
				return err
			}
			updated.ID = data.BootstrapConfigs[idx].ID
			if updated.CreatedAt == "" {
				updated.CreatedAt = data.BootstrapConfigs[idx].CreatedAt
			}
			defaultBootstrapConfig(&updated)
			data.BootstrapConfigs[idx] = updated
			item = updated
			return nil
		})
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		err := a.store.Update(func(data *domain.Data) error {
			idx := findBootstrapConfig(data.BootstrapConfigs, id)
			if idx < 0 {
				return store.ErrNotFound
			}
			data.BootstrapConfigs = append(data.BootstrapConfigs[:idx], data.BootstrapConfigs[idx+1:]...)
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

func (a *API) handlePipelineStepTemplates(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		data := a.store.Snapshot()
		items := append([]domain.PipelineStepTemplate{}, data.PipelineSteps...)
		sort.Slice(items, func(i, j int) bool {
			if items[i].Framework == items[j].Framework {
				return items[i].StepID < items[j].StepID
			}
			return items[i].Framework < items[j].Framework
		})
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var item domain.PipelineStepTemplate
		if err := readJSON(r, &item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		defaultPipelineStepTemplate(&item)
		if err := a.store.Update(func(data *domain.Data) error {
			data.PipelineSteps = append(data.PipelineSteps, item)
			return nil
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, item)
	default:
		methodNotAllowed(w)
	}
}

func (a *API) handlePipelineStepTemplate(w http.ResponseWriter, r *http.Request) {
	id, rest := pathParts(r.URL.Path, "/api/pipeline-step-templates/")
	if id == "" || len(rest) != 0 {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		data := a.store.Snapshot()
		idx := findPipelineStepTemplate(data.PipelineSteps, id)
		if idx < 0 {
			writeError(w, http.StatusNotFound, store.ErrNotFound)
			return
		}
		writeJSON(w, http.StatusOK, data.PipelineSteps[idx])
	case http.MethodPut, http.MethodPatch:
		var patch map[string]json.RawMessage
		if err := readJSON(r, &patch); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		var item domain.PipelineStepTemplate
		err := a.store.Update(func(data *domain.Data) error {
			idx := findPipelineStepTemplate(data.PipelineSteps, id)
			if idx < 0 {
				return store.ErrNotFound
			}
			updated, err := mergeJSON(data.PipelineSteps[idx], patch)
			if err != nil {
				return err
			}
			updated.ID = data.PipelineSteps[idx].ID
			if updated.CreatedAt == "" {
				updated.CreatedAt = data.PipelineSteps[idx].CreatedAt
			}
			defaultPipelineStepTemplate(&updated)
			data.PipelineSteps[idx] = updated
			item = updated
			return nil
		})
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		err := a.store.Update(func(data *domain.Data) error {
			idx := findPipelineStepTemplate(data.PipelineSteps, id)
			if idx < 0 {
				return store.ErrNotFound
			}
			data.PipelineSteps = append(data.PipelineSteps[:idx], data.PipelineSteps[idx+1:]...)
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

func defaultActionTemplate(item *domain.ActionTemplate) {
	now := domain.Now()
	if item.ID == "" {
		item.ID = domain.NewID("action")
	}
	if item.Name == "" {
		item.Name = "New Action"
	}
	if item.ExecutionType == "" {
		item.ExecutionType = "command"
	}
	if item.Fields == nil {
		item.Fields = []domain.ActionTemplateField{}
	}
	if item.Tags == nil {
		item.Tags = []string{}
	}
	if item.CreatedAt == "" {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
}

func defaultBootstrapConfig(item *domain.BootstrapConfig) {
	now := domain.Now()
	if item.ID == "" {
		item.ID = domain.NewID("bootstrap")
	}
	if item.Name == "" {
		item.Name = "New Bootstrap Config"
	}
	if item.DefaultArgs == nil {
		item.DefaultArgs = map[string]string{}
	}
	if item.CreatedAt == "" {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
}

func defaultPipelineStepTemplate(item *domain.PipelineStepTemplate) {
	now := domain.Now()
	if item.ID == "" {
		item.ID = domain.NewID("pipeline_step")
	}
	if item.Name == "" {
		item.Name = "New Pipeline Step"
	}
	if item.Framework == "" {
		item.Framework = "vllm-ascend"
	}
	if item.StepID == "" {
		item.StepID = "custom_step"
	}
	if item.Details == nil {
		item.Details = []string{}
	}
	if item.CreatedAt == "" {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
}

func findActionTemplate(items []domain.ActionTemplate, id string) int {
	for i, item := range items {
		if item.ID == id {
			return i
		}
	}
	return -1
}

func findBootstrapConfig(items []domain.BootstrapConfig, id string) int {
	for i, item := range items {
		if item.ID == id {
			return i
		}
	}
	return -1
}

func findPipelineStepTemplate(items []domain.PipelineStepTemplate, id string) int {
	for i, item := range items {
		if item.ID == id {
			return i
		}
	}
	return -1
}
