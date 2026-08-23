package httpapi

import (
	"net/http"
)

// handleBuildPackage 构建求解前置包。
func (a *API) handleBuildPackage(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	_ = decodeJSON(w, r, &in)
	p, err := a.packages.Build(in.Name)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

// handleDerivePackage 派生新包（指定基线包）。
func (a *API) handleDerivePackage(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name   string `json:"name"`
		BaseID string `json:"base_id"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	p, err := a.packages.Derive(in.Name, in.BaseID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

// handleListPackages 列出包。
func (a *API) handleListPackages(w http.ResponseWriter, r *http.Request) {
	list, err := a.packages.List()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleGetPackage 读取包。
func (a *API) handleGetPackage(w http.ResponseWriter, r *http.Request) {
	p, err := a.packages.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// handlePublishPackage 发布包。
func (a *API) handlePublishPackage(w http.ResponseWriter, r *http.Request) {
	p, err := a.packages.Publish(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// handlePackageDiff 计算包差异（?against=<from_id>）。
func (a *API) handlePackageDiff(w http.ResponseWriter, r *http.Request) {
	toID := r.PathValue("id")
	fromID := r.URL.Query().Get("against")
	if fromID == "" {
		writeJSON(w, http.StatusBadRequest, errResp{Error: "query param 'against' required"})
		return
	}
	d, err := a.packages.Diff(fromID, toID)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}
