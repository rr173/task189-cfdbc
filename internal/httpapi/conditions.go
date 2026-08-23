package httpapi

import (
	"net/http"

	"task189-cfdbc/internal/conditions"
)

// handleAssignCondition 分配边界条件。
func (a *API) handleAssignCondition(w http.ResponseWriter, r *http.Request) {
	var in conditions.BCInput
	if !decodeJSON(w, r, &in) {
		return
	}
	bc, err := a.conds.Assign(in)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, bc)
}

// handleListConditions 列出条件。
func (a *API) handleListConditions(w http.ResponseWriter, r *http.Request) {
	list, err := a.conds.List()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleGetCondition 读取条件。
func (a *API) handleGetCondition(w http.ResponseWriter, r *http.Request) {
	bc, err := a.conds.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bc)
}

// handleApproveCondition 批准条件。
func (a *API) handleApproveCondition(w http.ResponseWriter, r *http.Request) {
	bc, err := a.conds.Approve(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bc)
}

// handleReviseCondition 修订条件（乐观锁）。
func (a *API) handleReviseCondition(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Value     float64 `json:"value"`
		Secondary float64 `json:"secondary"`
		Version   int     `json:"version"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	bc, err := a.conds.Revise(r.PathValue("id"), in.Value, in.Secondary, in.Version)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bc)
}
