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
