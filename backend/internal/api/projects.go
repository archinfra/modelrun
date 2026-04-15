package api

import (
	"encoding/json"
	"net/http"

	"modelrun/backend/internal/domain"
	"modelrun/backend/internal/store"
)

func (a *API) handleProjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, a.store.Snapshot().Projects)
	case http.MethodPost:
		var project domain.Project
		if err := readJSON(r, &project); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		defaultProject(&project)

		if err := a.store.Update(func(data *domain.Data) error {
			data.Projects = append(data.Projects, project)
			return nil
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusCreated, project)
	default:
		methodNotAllowed(w)
	}
}

func (a *API) handleProject(w http.ResponseWriter, r *http.Request) {
	id, rest := pathParts(r.URL.Path, "/api/projects/")
	if id == "" || len(rest) != 0 {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		data := a.store.Snapshot()
		idx := findProject(data.Projects, id)
		if idx < 0 {
			writeError(w, http.StatusNotFound, store.ErrNotFound)
			return
		}
		writeJSON(w, http.StatusOK, data.Projects[idx])
	case http.MethodPut, http.MethodPatch:
		var patch map[string]json.RawMessage
		if err := readJSON(r, &patch); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		var project domain.Project
		err := a.store.Update(func(data *domain.Data) error {
			idx := findProject(data.Projects, id)
			if idx < 0 {
				return store.ErrNotFound
			}
			updated, err := mergeJSON(data.Projects[idx], patch)
			if err != nil {
				return err
			}
			updated.ID = data.Projects[idx].ID
			if updated.CreatedAt == "" {
				updated.CreatedAt = data.Projects[idx].CreatedAt
			}
			updated.UpdatedAt = domain.Now()
			data.Projects[idx] = updated
			project = updated
			return nil
		})
		if err != nil {
			writeStoreError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, project)
	case http.MethodDelete:
		err := a.store.Update(func(data *domain.Data) error {
			idx := findProject(data.Projects, id)
			if idx < 0 {
				return store.ErrNotFound
			}
			data.Projects = append(data.Projects[:idx], data.Projects[idx+1:]...)
			data.Servers = filterServersByProject(data.Servers, id)
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

func findProject(projects []domain.Project, id string) int {
	for i, project := range projects {
		if project.ID == id {
			return i
		}
	}
	return -1
}

func filterServersByProject(servers []domain.ServerConfig, projectID string) []domain.ServerConfig {
	filtered := servers[:0]
	for _, server := range servers {
		if server.ProjectID != projectID {
			filtered = append(filtered, server)
		}
	}
	return filtered
}
