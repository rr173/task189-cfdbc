package service

import (
	"testing"

	"task189-cfdbc/internal/conditions"
	"task189-cfdbc/internal/model"
	"task189-cfdbc/internal/store"
)

// newTestDB 打开一个内存 SQLite 并迁移建表，供条件服务测试使用。
// modernc.org/sqlite 的 :memory: 数据库随连接存活；Open 将连接数限制为 1，
// 故测试期间数据持续可见。
func newTestDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// seedFace 登记一个区域并导入一个外露面，返回该面。
func seedFace(t *testing.T, db *store.DB) *model.Face {
	t.Helper()
	regions := store.NewRegionStore(db)
	faces := store.NewFaceStore(db)
	r := &model.Region{
		ID:        "reg-1",
		Name:      "duct",
		Dimension: 3,
		CellCount: 10,
		Status:    model.RegionConnected,
		MeshHash:  "h",
		CreatedAt: "2026-01-01T00:00:00Z",
		UpdatedAt: "2026-01-01T00:00:00Z",
	}
	if err := regions.Insert(r); err != nil {
		t.Fatalf("insert region: %v", err)
	}
	f := &model.Face{
		ID:        "face-in",
		RegionID:  r.ID,
		Name:      "face-in",
		Kind:      model.KindOuter,
		Status:    model.FaceExposed,
		Area:      0.25,
		NodeCount: 4,
		CreatedAt: "2026-01-01T00:00:00Z",
	}
	if err := faces.Insert(f); err != nil {
		t.Fatalf("insert face: %v", err)
	}
	return f
}

// TestReviseNegativeInletMassFlowToConflicting 修订入口质量流量为负数后，
// 返回结果与数据库持久化状态都必须变为 conflicting，不能继续 applicable。
func TestReviseNegativeInletMassFlowToConflicting(t *testing.T) {
	db := newTestDB(t)
	face := seedFace(t, db)
	conds := NewConditionService(db)

	// 先分配一个合法的正入口质量流量 → applicable
	bc, err := conds.Assign(conditions.BCInput{
		FaceID: face.ID, Type: model.BCInletMassFlow, Unit: model.UnitSI, Value: 1.5,
	})
	if err != nil {
		t.Fatalf("assign inlet: %v", err)
	}
	if bc.Status != model.BCStatusApplicable {
		t.Fatalf("expected initial applicable, got %s", bc.Status)
	}

	// 修订为负数 → 返回值须为 conflicting
	rev, err := conds.Revise(bc.ID, -0.7, 0, bc.Version)
	if err != nil {
		t.Fatalf("revise to negative: %v", err)
	}
	if rev.Status != model.BCStatusConflicting {
		t.Fatalf("expected revised status conflicting, got %s", rev.Status)
	}
	if rev.Value != -0.7 {
		t.Fatalf("expected revised value -0.7, got %v", rev.Value)
	}

	// 数据库中持久化状态也必须为 conflicting
	reloaded, err := conds.Get(bc.ID)
	if err != nil {
		t.Fatalf("get revised bc: %v", err)
	}
	if reloaded.Status != model.BCStatusConflicting {
		t.Fatalf("expected persisted status conflicting, got %s", reloaded.Status)
	}
	if reloaded.Value != -0.7 {
		t.Fatalf("expected persisted value -0.7, got %v", reloaded.Value)
	}
}

// TestAssignNegativeInletMassFlowConflicting 分配阶段即拒绝负入口质量流量。
func TestAssignNegativeInletMassFlowConflicting(t *testing.T) {
	db := newTestDB(t)
	face := seedFace(t, db)
	conds := NewConditionService(db)

	bc, err := conds.Assign(conditions.BCInput{
		FaceID: face.ID, Type: model.BCInletMassFlow, Unit: model.UnitSI, Value: -1.0,
	})
	if err != nil {
		t.Fatalf("assign negative inlet: %v", err)
	}
	if bc.Status != model.BCStatusConflicting {
		t.Fatalf("expected conflicting for negative inlet mass flow, got %s", bc.Status)
	}
}

// TestReviseOutletNegativeConflicting 出口质量流量修订为负同样须转为 conflicting。
func TestReviseOutletNegativeConflicting(t *testing.T) {
	db := newTestDB(t)
	face := seedFace(t, db)
	conds := NewConditionService(db)

	bc, err := conds.Assign(conditions.BCInput{
		FaceID: face.ID, Type: model.BCOutletMassFlow, Unit: model.UnitSI, Value: 2.0,
	})
	if err != nil {
		t.Fatalf("assign outlet: %v", err)
	}
	rev, err := conds.Revise(bc.ID, -3.0, 0, bc.Version)
	if err != nil {
		t.Fatalf("revise outlet negative: %v", err)
	}
	if rev.Status != model.BCStatusConflicting {
		t.Fatalf("expected conflicting, got %s", rev.Status)
	}
}
