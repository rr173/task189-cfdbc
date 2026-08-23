package httpapi

import (
	"net/http"
	"strconv"

	"task189-cfdbc/internal/mesh"
)

// handleCreateRegion 登记网格区域。
func (a *API) handleCreateRegion(w http.ResponseWriter, r *http.Request) {
	var in mesh.RegionInput
	if !decodeJSON(w, r, &in) {
		return
	}
	reg, err := a.regions.Create(in)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, reg)
}

// handleListRegions 列出区域。
func (a *API) handleListRegions(w http.ResponseWriter, r *http.Request) {
	list, err := a.regions.List()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleGetRegion 读取区域。
func (a *API) handleGetRegion(w http.ResponseWriter, r *http.Request) {
	reg, err := a.regions.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, reg)
}

// handleImportFaces 导入面集合。
func (a *API) handleImportFaces(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Faces []mesh.FaceInput `json:"faces"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	reg, faces, err := a.regions.ImportFaces(r.PathValue("id"), in.Faces)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"region": reg,
		"faces":  faces,
	})
}

// handleListRegionFaces 列出区域面。
func (a *API) handleListRegionFaces(w http.ResponseWriter, r *http.Request) {
	faces, err := a.regions.ListFaces(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, faces)
}

// handleSealRegion 封存区域。
func (a *API) handleSealRegion(w http.ResponseWriter, r *http.Request) {
	reg, err := a.regions.Seal(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, reg)
}

// handleGetFace 读取面。
func (a *API) handleGetFace(w http.ResponseWriter, r *http.Request) {
	f, err := a.regions.GetFace(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, f)
}

// handleFaceConditions 列出面的条件。
func (a *API) handleFaceConditions(w http.ResponseWriter, r *http.Request) {
	list, err := a.conds.ListByFace(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleListIssues 列出问题（支持 ?config_version= 与 ?limit=）。
func (a *API) handleListIssues(w http.ResponseWriter, r *http.Request) {
	cv, _ := strconv.Atoi(r.URL.Query().Get("config_version"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	list, err := a.validate.Issues(cv, limit)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleGetIssue 读取问题。
func (a *API) handleGetIssue(w http.ResponseWriter, r *http.Request) {
	iss, err := a.validate.Issue(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, iss)
}
