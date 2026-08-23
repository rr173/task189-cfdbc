package httpapi

import "net/http"

// handleHealth 健康检查。
func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleStats 统计概览。
func (a *API) handleStats(w http.ResponseWriter, r *http.Request) {
	s, err := a.stats.Collect()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}
