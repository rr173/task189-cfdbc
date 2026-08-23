package httpapi

import (
	"net/http"
	"strconv"
)

// handleValidate 执行一致性校验。
func (a *API) handleValidate(w http.ResponseWriter, r *http.Request) {
	run, issues, err := a.validate.Run()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"run":    run,
		"issues": issues,
	})
}

// handleListRuns 列出校验运行（?limit=）。
func (a *API) handleListRuns(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	list, err := a.validate.ListRuns(limit)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleGetRun 读取校验运行。
func (a *API) handleGetRun(w http.ResponseWriter, r *http.Request) {
	run, err := a.validate.GetRun(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

// handleRunIssues 读取运行的问题。
func (a *API) handleRunIssues(w http.ResponseWriter, r *http.Request) {
	list, err := a.validate.IssuesByRun(r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}
