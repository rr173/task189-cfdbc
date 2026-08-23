package mesh

import (
	"testing"

	"task189-cfdbc/internal/model"
)

func TestApplyFaceStatusDegenerate(t *testing.T) {
	f := &model.Face{Kind: model.KindOuter, NodeCount: 2, Area: 0.1}
	ApplyFaceStatus(f)
	if f.Status != model.FaceDegenerate {
		t.Fatalf("expected degenerate, got %s", f.Status)
	}
}

func TestApplyFaceStatusExposedAndCoupled(t *testing.T) {
	f1 := &model.Face{Kind: model.KindOuter, NodeCount: 4, Area: 1.0}
	ApplyFaceStatus(f1)
	if f1.Status != model.FaceExposed {
		t.Fatalf("expected exposed, got %s", f1.Status)
	}
	f2 := &model.Face{Kind: model.KindInterface, NodeCount: 4, Area: 1.0}
	ApplyFaceStatus(f2)
	if f2.Status != model.FaceCoupled {
		t.Fatalf("expected coupled, got %s", f2.Status)
	}
}

func TestDetectDuplicates(t *testing.T) {
	faces := []*model.Face{
		{ID: "f1", Name: "in", Area: 1.0, NormalX: 1, NodeCount: 4},
		{ID: "f2", Name: "in-copy", Area: 1.0, NormalX: 1, NodeCount: 4},
	}
	changed := DetectDuplicates(faces)
	if len(changed) != 1 || changed[0] != "f2" {
		t.Fatalf("expected f2 duplicated, got %v", changed)
	}
	if faces[1].Status != model.FaceDuplicate || faces[1].DuplicateOf != "f1" {
		t.Fatalf("expected f2 marked duplicate of f1, got %+v", faces[1])
	}
}

func TestHashRegionStable(t *testing.T) {
	a := []*model.Face{{ID: "f1", Name: "in"}, {ID: "f2", Name: "out"}}
	b := []*model.Face{{ID: "f3", Name: "out"}, {ID: "f4", Name: "in"}}
	ha, hb := HashRegion("r1", a), HashRegion("r1", b)
	if ha != hb {
		t.Fatalf("hash should be order-independent: %s vs %s", ha, hb)
	}
}

func TestCheckConnectivityOrphan(t *testing.T) {
	r := &model.Region{ID: "r1"}
	faces := []*model.Face{
		{ID: "f1", Kind: model.KindInterface, NeighborRegion: "", NeighborFace: ""},
	}
	c := CheckConnectivity(r, faces)
	if c.Connected {
		t.Fatal("expected isolated region")
	}
	if len(c.OrphanFaces) != 1 {
		t.Fatalf("expected 1 orphan face, got %v", c.OrphanFaces)
	}
}

func TestFaceFingerprintDistinct(t *testing.T) {
	f1 := &model.Face{Area: 1.0, NormalX: 1, NodeCount: 4}
	f2 := &model.Face{Area: 2.0, NormalX: 1, NodeCount: 4}
	if FaceFingerprint(f1) == FaceFingerprint(f2) {
		t.Fatal("expected different fingerprints for different geometry")
	}
}
