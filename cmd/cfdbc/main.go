// Command cfdbc 是计算流体网格边界条件一致性服务入口。
//
// 用法：
//
//	cfdbc --addr :8080 --db data.db          # 启动 HTTP 服务
//	cfdbc --smoke-test --db /tmp/x.db       # 执行端到端自检（不驻留）
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"task189-cfdbc/internal/conditions"
	"task189-cfdbc/internal/httpapi"
	"task189-cfdbc/internal/mesh"
	"task189-cfdbc/internal/model"
	"task189-cfdbc/internal/physics"
	"task189-cfdbc/internal/service"
	"task189-cfdbc/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "cfdbc.db", "SQLite database path")
	smoke := flag.Bool("smoke-test", false, "run end-to-end smoke test then exit")
	flag.Parse()

	if *smoke {
		if err := runSmoke(*dbPath); err != nil {
			log.Fatalf("smoke test failed: %v", err)
		}
		fmt.Println("smoke test passed: mesh boundary condition consistency service OK")
		return
	}

	db, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	api := buildAPI(db)
	log.Printf("cfdbc listening on %s (db=%s)", *addr, *dbPath)
	if err := http.ListenAndServe(*addr, api.Router()); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// buildAPI 装配全部服务与 HTTP 路由。
func buildAPI(db *store.DB) *httpapi.API {
	return httpapi.New(
		service.NewRegionService(db),
		service.NewModelService(db),
		service.NewConditionService(db),
		service.NewValidateService(db),
		service.NewPackageService(db),
		service.NewStatsService(db),
	)
}

// must 断言错误为 nil（smoke 内部使用）。
func must(err error) {
	if err != nil {
		panic(err)
	}
}

// runSmoke 端到端自检：真实走完“登记 → 导入 → 模型 → 条件 → 校验 →
// 发布 → 派生 → 重启恢复”闭环，并断言关键不变量。
func runSmoke(dbPath string) error {
	if dbPath == "" || dbPath == ":memory:" {
		dbPath = filepath.Join(os.TempDir(), "cfdbc-smoke.db")
	}
	_ = os.Remove(dbPath)

	// --- 阶段 1：创建数据库并登记区域 ---
	db, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	regions := service.NewRegionService(db)
	models := service.NewModelService(db)
	conds := service.NewConditionService(db)
	validate := service.NewValidateService(db)
	packages := service.NewPackageService(db)

	// 1) 登记入口区域（三维、1 入口 + 1 出口 + 2 壁面 + 1 耦合面）
	r1, err := regions.Create(mesh.RegionInput{Name: "duct-main", Dimension: 3, CellCount: 120000, Description: "main duct mesh"})
	if err != nil {
		return fmt.Errorf("create region1: %w", err)
	}
	// 2) 登记耦合对侧区域
	r2, err := regions.Create(mesh.RegionInput{Name: "duct-side", Dimension: 3, CellCount: 40000, Description: "side chamber mesh"})
	if err != nil {
		return fmt.Errorf("create region2: %w", err)
	}

	// 3) 导入区域1面：外露入口/出口/壁面 + 耦合面（对侧区域2 face-j）
	_, faces1, err := regions.ImportFaces(r1.ID, []mesh.FaceInput{
		{Name: "face-in", Kind: "outer", Area: 0.25, NormalX: -1, NodeCount: 4},
		{Name: "face-out", Kind: "outer", Area: 0.25, NormalX: 1, NodeCount: 4},
		{Name: "face-wall-a", Kind: "outer", Area: 1.0, NormalY: 1, NodeCount: 4},
		{Name: "face-wall-b", Kind: "outer", Area: 1.0, NormalY: -1, NodeCount: 4},
		{Name: "face-coup", Kind: "interface", Area: 0.16, NormalZ: 1, NodeCount: 4,
			NeighborRegion: r2.ID, NeighborFace: "face-j"},
	})
	if err != nil {
		return fmt.Errorf("import faces region1: %w", err)
	}
	// 4) 导入区域2面：耦合面 + 出口
	_, faces2, err := regions.ImportFaces(r2.ID, []mesh.FaceInput{
		{Name: "face-j", Kind: "interface", Area: 0.16, NormalZ: -1, NodeCount: 4,
			NeighborRegion: r1.ID, NeighborFace: "face-coup"},
		{Name: "face-out2", Kind: "outer", Area: 0.09, NormalX: 1, NodeCount: 4},
	})
	if err != nil {
		return fmt.Errorf("import faces region2: %w", err)
	}

	// 5) 创建并激活不可压缩模型（要求参考压力）
	m, err := models.Create(physics.ModelInput{Name: "incompressible-duct", FlowType: model.FlowIncompressible, Viscous: true, DefaultUnit: model.UnitSI})
	if err != nil {
		return fmt.Errorf("create model: %w", err)
	}
	if _, err := models.Activate(m.ID); err != nil {
		return fmt.Errorf("activate model: %w", err)
	}

	// 6) 配置边界条件：入口质量流量、出口质量流量、壁面无滑移、区域2压力出口
	inlet, err := conds.Assign(conditions.BCInput{FaceID: faceID(faces1, "face-in"), Type: model.BCInletMassFlow, Unit: model.UnitSI, Value: 1.5})
	if err != nil {
		return fmt.Errorf("assign inlet: %w", err)
	}
	outlet, err := conds.Assign(conditions.BCInput{FaceID: faceID(faces1, "face-out"), Type: model.BCOutletMassFlow, Unit: model.UnitSI, Value: 1.5})
	if err != nil {
		return fmt.Errorf("assign outlet: %w", err)
	}
	for _, f := range faces1 {
		if f.Name == "face-wall-a" || f.Name == "face-wall-b" {
			if _, err := conds.Assign(conditions.BCInput{FaceID: f.ID, Type: model.BCWallNoSlip, Unit: model.UnitSI}); err != nil {
				return fmt.Errorf("assign wall: %w", err)
			}
		}
	}
	if _, err := conds.Assign(conditions.BCInput{FaceID: faceID(faces2, "face-out2"), Type: model.BCOutletPressure, Unit: model.UnitSI, Value: 0}); err != nil {
		return fmt.Errorf("assign outlet2: %w", err)
	}

	// 7) 批准入口/出口条件
	for _, id := range []string{inlet.ID, outlet.ID} {
		if _, err := conds.Approve(id); err != nil {
			return fmt.Errorf("approve: %w", err)
		}
	}

	// 8) 校验：耦合面只配了一侧 → 欠约束
	run1, issues1, err := validate.Run()
	if err != nil {
		return fmt.Errorf("validate run1: %w", err)
	}
	if run1.Result != "underconstrained" {
		return fmt.Errorf("expected underconstrained (interface one side), got %s", run1.Result)
	}
	if !hasIssue(issues1, "missing_bc", faceID(faces1, "face-coup")) {
		return fmt.Errorf("expected missing_bc on coupled face face-coup, got %+v", issues1)
	}

	// 9) 补齐耦合面 A 侧；同时给同一外露面 face-in 再配一个 wall → 冲突检测
	coupSideA, err := conds.Assign(conditions.BCInput{FaceID: faceID(faces1, "face-coup"), Type: model.BCInterfaceSideA, Unit: model.UnitSI, Value: 1.5})
	if err != nil {
		return fmt.Errorf("assign coup side a: %w", err)
	}
	conflict, err := conds.Assign(conditions.BCInput{FaceID: faceID(faces1, "face-in"), Type: model.BCWallNoSlip, Unit: model.UnitSI})
	if err != nil {
		return fmt.Errorf("assign conflict wall: %w", err)
	}
	if conflict.Status != model.BCStatusConflicting {
		return fmt.Errorf("expected conflicting status for double-bc face, got %s", conflict.Status)
	}

	// 10) 参考压力缺失 → 仍欠约束
	run2, _, err := validate.Run()
	if err != nil {
		return fmt.Errorf("validate run2: %w", err)
	}
	if run2.Result != "underconstrained" {
		return fmt.Errorf("expected underconstrained (missing ref pressure), got %s", run2.Result)
	}

	// 11) 添加并批准参考压力；撤销冲突条件（修订为 0 不再是墙）
	ref, err := conds.Assign(conditions.BCInput{FaceID: faceID(faces1, "face-out"), Type: model.BCReferencePressure, Unit: model.UnitSI, Value: 101325})
	if err != nil {
		return fmt.Errorf("assign ref pressure: %w", err)
	}
	if _, err := conds.Approve(ref.ID); err != nil {
		return fmt.Errorf("approve ref: %w", err)
	}
	conflictFresh, err := conds.Get(conflict.ID)
	if err != nil {
		return fmt.Errorf("get conflict bc: %w", err)
	}
	if _, err := conds.Revise(conflict.ID, 0, 0, conflictFresh.Version); err != nil {
		return fmt.Errorf("revise conflict: %w", err)
	}

	// 12) 仍欠约束（耦合面 B 侧未配）
	run3, _, err := validate.Run()
	if err != nil {
		return fmt.Errorf("validate run3: %w", err)
	}
	if run3.Result != "underconstrained" {
		return fmt.Errorf("expected underconstrained (coup side b), got %s", run3.Result)
	}

	// 13) 配置耦合面 B 侧并批准双侧
	sideB, err := conds.Assign(conditions.BCInput{FaceID: faceID(faces2, "face-j"), Type: model.BCInterfaceSideB, Unit: model.UnitSI, Value: 1.5})
	if err != nil {
		return fmt.Errorf("assign coup side b: %w", err)
	}
	if _, err := conds.Approve(sideB.ID); err != nil {
		return fmt.Errorf("approve side b: %w", err)
	}
	if _, err := conds.Approve(coupSideA.ID); err != nil {
		return fmt.Errorf("approve side a: %w", err)
	}

	// 14) 全部补齐 → 可求解
	run4, issues4, err := validate.Run()
	if err != nil {
		return fmt.Errorf("validate run4: %w", err)
	}
	if run4.Result != "solvable" {
		return fmt.Errorf("expected solvable, got %s (issues=%d)", run4.Result, len(issues4))
	}

	// 15) 发布前置包；已发布包不可直接改写 → 派生新包
	pkg, err := packages.Build("duct-case-v1")
	if err != nil {
		return fmt.Errorf("build package: %w", err)
	}
	if _, err := packages.Publish(pkg.ID); err != nil {
		return fmt.Errorf("publish package: %w", err)
	}
	derived, err := packages.Derive("duct-case-v2", pkg.ID)
	if err != nil {
		return fmt.Errorf("derive package: %w", err)
	}
	if derived.ID == pkg.ID {
		return fmt.Errorf("expected derived package to differ from base")
	}
	diff, err := packages.Diff(pkg.ID, derived.ID)
	if err != nil {
		return fmt.Errorf("diff packages: %w", err)
	}

	// --- 阶段 2：关闭并重新打开同一数据库，验证持久化与重启恢复 ---
	if err := db.Close(); err != nil {
		return fmt.Errorf("close db: %w", err)
	}
	db2, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("reopen db: %w", err)
	}
	defer db2.Close()

	regions2 := service.NewRegionService(db2)
	validate2 := service.NewValidateService(db2)
	packages2 := service.NewPackageService(db2)
	stats2 := service.NewStatsService(db2)

	// 幂等：相同网格哈希导入被拒绝（拓扑已存在）。先用与原区域1完全相同
	// 的面集合再次导入——必须被拒绝，且面数量不能增加。
	r1Before, err := regions2.Get(r1.ID)
	if err != nil {
		return fmt.Errorf("get r1 before re-import: %w", err)
	}
	faceCountBefore := r1Before.FaceCount
	_, _, err = regions2.ImportFaces(r1.ID, []mesh.FaceInput{
		{Name: "face-in", Kind: "outer", Area: 0.25, NormalX: -1, NodeCount: 4},
		{Name: "face-out", Kind: "outer", Area: 0.25, NormalX: 1, NodeCount: 4},
		{Name: "face-wall-a", Kind: "outer", Area: 1.0, NormalY: 1, NodeCount: 4},
		{Name: "face-wall-b", Kind: "outer", Area: 1.0, NormalY: -1, NodeCount: 4},
		{Name: "face-coup", Kind: "interface", Area: 0.16, NormalZ: 1, NodeCount: 4,
			NeighborRegion: r2.ID, NeighborFace: "face-j"},
	})
	if err == nil {
		return fmt.Errorf("expected identical face set re-import to be rejected")
	}
	r1After, err := regions2.Get(r1.ID)
	if err != nil {
		return fmt.Errorf("get r1 after re-import: %w", err)
	}
	if r1After.FaceCount != faceCountBefore {
		return fmt.Errorf("identical re-import must not change face count: before=%d after=%d", faceCountBefore, r1After.FaceCount)
	}

	// 运行记录与包状态持久化
	lastRun, err := validate2.GetRun(run4.ID)
	if err != nil {
		return fmt.Errorf("get run after reopen: %w", err)
	}
	if lastRun.Result != "solvable" {
		return fmt.Errorf("run result changed after reopen: %s", lastRun.Result)
	}
	reloaded, err := packages2.Get(pkg.ID)
	if err != nil {
		return fmt.Errorf("get pkg after reopen: %w", err)
	}
	if reloaded.Status != model.PkgPublished {
		return fmt.Errorf("pkg status changed after reopen: %s", reloaded.Status)
	}

	// 统计一致性
	st, err := stats2.Collect()
	if err != nil {
		return fmt.Errorf("collect stats: %w", err)
	}
	if st.Regions[string(model.RegionConnected)] < 1 {
		return fmt.Errorf("expected connected regions in stats, got %+v", st.Regions)
	}
	if st.Packages[string(model.PkgPublished)] < 1 {
		return fmt.Errorf("expected published package in stats, got %+v", st.Packages)
	}

	fmt.Printf("smoke ok: runs=%d issues=%d packages=%d diff=%s\n",
		len(mustNoErr(validate2.ListRuns(10))), len(mustNoErr(validate2.Issues(0, 100))), len(mustNoErr(packages2.List())), diff.Summary)
	return nil
}

// faceID 按名称查找面 ID。
func faceID(faces []*model.Face, name string) string {
	for _, f := range faces {
		if f.Name == name {
			return f.ID
		}
	}
	return ""
}

// hasIssue 判断问题清单中是否含指定类型且命中指定面。
func hasIssue(issues []*model.Issue, issueType, faceID string) bool {
	for _, i := range issues {
		if i.Type == issueType && (faceID == "" || i.FaceID == faceID) {
			return true
		}
	}
	return false
}

// mustNoErr 解包 (T, error)。
func mustNoErr[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
