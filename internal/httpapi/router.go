// Package httpapi 提供 RESTful HTTP 接口，路由前缀 /api。
// 使用 Go 1.22+ 增强 ServeMux（方法 + 路径参数）。
package httpapi

import (
	"encoding/json"
	"log"
	"net/http"

	"task189-cfdbc/internal/service"
)

// API 聚合服务依赖。
type API struct {
	regions  *service.RegionService
	models   *service.ModelService
	conds    *service.ConditionService
	validate *service.ValidateService
	packages *service.PackageService
	stats    *service.StatsService
}

// New 构造 API。
func New(
	regions *service.RegionService,
	models *service.ModelService,
	conds *service.ConditionService,
	validate *service.ValidateService,
	packages *service.PackageService,
	stats *service.StatsService,
) *API {
	return &API{
		regions:  regions,
		models:   models,
		conds:    conds,
		validate: validate,
		packages: packages,
		stats:    stats,
	}
}

// Router 装配全部路由。
func (a *API) Router() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", a.handleHealth)
	mux.HandleFunc("GET /api/stats", a.handleStats)

	// 区域
	mux.HandleFunc("POST /api/regions", a.handleCreateRegion)
	mux.HandleFunc("GET /api/regions", a.handleListRegions)
	mux.HandleFunc("GET /api/regions/{id}", a.handleGetRegion)
	mux.HandleFunc("POST /api/regions/{id}/faces", a.handleImportFaces)
	mux.HandleFunc("GET /api/regions/{id}/faces", a.handleListRegionFaces)
	mux.HandleFunc("POST /api/regions/{id}/seal", a.handleSealRegion)

	// 面
	mux.HandleFunc("GET /api/faces/{id}", a.handleGetFace)
	mux.HandleFunc("GET /api/faces/{id}/conditions", a.handleFaceConditions)

	// 物理模型
	mux.HandleFunc("POST /api/models", a.handleCreateModel)
	mux.HandleFunc("GET /api/models", a.handleListModels)
	mux.HandleFunc("GET /api/models/active", a.handleGetActiveModel)
	mux.HandleFunc("GET /api/models/{id}", a.handleGetModel)
	mux.HandleFunc("POST /api/models/{id}/activate", a.handleActivateModel)

	// 边界条件
	mux.HandleFunc("POST /api/conditions", a.handleAssignCondition)
	mux.HandleFunc("GET /api/conditions", a.handleListConditions)
	mux.HandleFunc("GET /api/conditions/{id}", a.handleGetCondition)
	mux.HandleFunc("POST /api/conditions/{id}/approve", a.handleApproveCondition)
	mux.HandleFunc("POST /api/conditions/{id}/revise", a.handleReviseCondition)

	// 校验
	mux.HandleFunc("POST /api/validate", a.handleValidate)
	mux.HandleFunc("GET /api/runs", a.handleListRuns)
	mux.HandleFunc("GET /api/runs/{id}", a.handleGetRun)
	mux.HandleFunc("GET /api/runs/{id}/issues", a.handleRunIssues)
	mux.HandleFunc("GET /api/issues", a.handleListIssues)
	mux.HandleFunc("GET /api/issues/{id}", a.handleGetIssue)

	// 前置包
	mux.HandleFunc("POST /api/packages", a.handleBuildPackage)
	mux.HandleFunc("POST /api/packages/derive", a.handleDerivePackage)
	mux.HandleFunc("GET /api/packages", a.handleListPackages)
	mux.HandleFunc("GET /api/packages/{id}", a.handleGetPackage)
	mux.HandleFunc("POST /api/packages/{id}/publish", a.handlePublishPackage)
	mux.HandleFunc("GET /api/packages/{id}/diff", a.handlePackageDiff)

	return a.logging(mux)
}

func (a *API) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// writeJSON 写 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// errResp 错误响应体。
type errResp struct {
	Error string `json:"error"`
}

// writeErr 把内部错误映射为 HTTP 状态码。
func writeErr(w http.ResponseWriter, err error) {
	switch e := err.(type) {
	case interface{ StatusCode() int }:
		writeJSON(w, e.StatusCode(), errResp{Error: err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, errResp{Error: err.Error()})
	}
}

// decodeJSON 解析请求体。
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeJSON(w, http.StatusBadRequest, errResp{Error: "invalid json: " + err.Error()})
		return false
	}
	return true
}
