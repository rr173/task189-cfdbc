package httpapi

import (
	"net/http"

	"task189-cfdbc/internal/physics"
)

// handleCreateModel 创建物理模型。
func (a *API) handleCreateModel(w http.ResponseWriter, r *http.Request) {
	var in physics.ModelInput
	if !decodeJSON(w, r, &in) {
		return
	}
	m, err := a.models.Create(in)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

// handleListModels 列出模型。
func (a *API) handleListModels(w http.ResponseWriter, r *http.Request) {
	list, err := a.models.List()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleGetModel 读取模型。
func (a *API) handleGetModel(w http.ResponseWriter, r *http.Request) {
	m, err := a.models.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// handleGetActiveModel 读取激活模型。
func (a *API) handleGetActiveModel(w http.ResponseWriter, r *http.Request) {
	m, err := a.models.GetActive()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// handleActivateModel 激活模型。
func (a *API) handleActivateModel(w http.ResponseWriter, r *http.Request) {
	m, err := a.models.Activate(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}
