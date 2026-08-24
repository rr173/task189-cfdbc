package conditions

import (
	"testing"

	"task189-cfdbc/internal/model"
)

func TestFaceRoleAllowed(t *testing.T) {
	outer := &model.Face{Kind: model.KindOuter}
	if !FaceRoleAllowed(outer, model.BCInletMassFlow) {
		t.Fatal("outer face should allow inlet")
	}
	if FaceRoleAllowed(outer, model.BCInterfaceSideA) {
		t.Fatal("outer face should reject interface side")
	}
	iface := &model.Face{Kind: model.KindInterface}
	if !FaceRoleAllowed(iface, model.BCInterfaceSideA) {
		t.Fatal("interface face should allow side a")
	}
	if FaceRoleAllowed(iface, model.BCInletMassFlow) {
		t.Fatal("interface face should reject inlet")
	}
}

func TestAssessConflicting(t *testing.T) {
	f := &model.Face{Kind: model.KindOuter}
	bc := &model.BC{Type: model.BCInterfaceSideA, Value: 1.0} // 类型与面不匹配
	Assess(f, bc)
	if bc.Status != model.BCStatusConflicting {
		t.Fatalf("expected conflicting, got %s", bc.Status)
	}
}

func TestAssessApplicable(t *testing.T) {
	f := &model.Face{Kind: model.KindOuter}
	bc := &model.BC{Type: model.BCWallNoSlip, Value: 0}
	Assess(f, bc)
	if bc.Status != model.BCStatusApplicable {
		t.Fatalf("expected applicable, got %s", bc.Status)
	}
}

func TestCheckCoverageMissingAndConflict(t *testing.T) {
	faces := []*model.Face{
		{ID: "f1", Status: model.FaceExposed},
		{ID: "f2", Status: model.FaceExposed},
	}
	bcByFace := map[string][]*model.BC{
		"f1": {{Type: model.BCWallNoSlip, Status: model.BCStatusApplicable},
			{Type: model.BCInletMassFlow, Status: model.BCStatusApplicable}}, // 冲突
		"f2": {}, // 缺失
	}
	cov := CheckCoverage(faces, bcByFace)
	if len(cov.MissingFaces) != 1 || cov.MissingFaces[0] != "f2" {
		t.Fatalf("expected f2 missing, got %v", cov.MissingFaces)
	}
	if len(cov.Conflicting) != 1 || cov.Conflicting[0] != "f1" {
		t.Fatalf("expected f1 conflicting, got %v", cov.Conflicting)
	}
}

func TestHasInterfaceSideDuplicate(t *testing.T) {
	existing := []*model.BC{
		{Type: model.BCInterfaceSideA, Status: model.BCStatusApplicable},
	}
	// 同侧再次分配 → 已存在 → 应标记冲突
	if !HasInterfaceSide(existing, model.BCInterfaceSideA) {
		t.Fatal("expected existing side a to block a second side a assignment")
	}
	// 对侧分配 → 不冲突（由守恒检查配对）
	if HasInterfaceSide(existing, model.BCInterfaceSideB) {
		t.Fatal("existing side a should not block side b assignment")
	}
	// 已批准的同侧也占用名额
	approved := []*model.BC{
		{Type: model.BCInterfaceSideB, Status: model.BCStatusApproved},
	}
	if !HasInterfaceSide(approved, model.BCInterfaceSideB) {
		t.Fatal("expected approved side b to block a second side b assignment")
	}
	// conflicting/draft 的同侧不占用名额（仅 applicable/approved 算数）
	stale := []*model.BC{
		{Type: model.BCInterfaceSideA, Status: model.BCStatusConflicting},
	}
	if HasInterfaceSide(stale, model.BCInterfaceSideA) {
		t.Fatal("conflicting side should not block reassignment")
	}
	// 非耦合面条件不算同侧占用
	mixed := []*model.BC{
		{Type: model.BCInletMassFlow, Status: model.BCStatusApplicable},
	}
	if HasInterfaceSide(mixed, model.BCInterfaceSideA) {
		t.Fatal("non-interface condition should not count as side occupancy")
	}
}

func TestCollectInterfacesPairAcrossRegions(t *testing.T) {
	faces := []*model.Face{
		{ID: "f-a", RegionID: "r1", Name: "coup", Kind: model.KindInterface, NeighborRegion: "r2", NeighborFace: "j"},
		{ID: "f-b", RegionID: "r2", Name: "j", Kind: model.KindInterface, NeighborRegion: "r1", NeighborFace: "coup"},
	}
	bcByFace := map[string][]*model.BC{
		"f-a": {{Type: model.BCInterfaceSideA, Unit: model.UnitSI, Value: 1.5}},
		"f-b": {{Type: model.BCInterfaceSideB, Unit: model.UnitSI, Value: 1.5}},
	}
	pairs := CollectInterfaces(faces, bcByFace)
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	p := pairs[0]
	if p.SideA == nil || p.SideB == nil {
		t.Fatal("expected both sides present")
	}
	if p.SideA.FaceID != "f-a" && p.SideA.FaceID != "" {
		t.Fatalf("unexpected side a: %+v", p.SideA)
	}
}

func TestCollectInterfacesMissingSide(t *testing.T) {
	faces := []*model.Face{
		{ID: "f-a", RegionID: "r1", Name: "coup", Kind: model.KindInterface, NeighborRegion: "r2", NeighborFace: "j"},
		{ID: "f-b", RegionID: "r2", Name: "j", Kind: model.KindInterface, NeighborRegion: "r1", NeighborFace: "coup"},
	}
	bcByFace := map[string][]*model.BC{
		"f-a": {{Type: model.BCInterfaceSideA, Unit: model.UnitSI, Value: 1.5}},
	}
	pairs := CollectInterfaces(faces, bcByFace)
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	if pairs[0].SideB != nil {
		t.Fatal("expected missing side b")
	}
}
