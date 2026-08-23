package release

import (
	"testing"

	"task189-cfdbc/internal/model"
)

func TestSnapshotEncodeDecode(t *testing.T) {
	s := NewSnapshot("hash-1", "m1", 3, 7,
		[]model.Face{{ID: "f1", Name: "in"}},
		[]model.BC{{ID: "bc1", FaceID: "f1", Type: model.BCInletMassFlow}})
	raw, err := s.Encode()
	if err != nil {
		t.Fatal(err)
	}
	back, err := DecodeSnapshot(raw)
	if err != nil {
		t.Fatal(err)
	}
	if back.RegionHash != "hash-1" || back.ConditionsVersion != 7 {
		t.Fatalf("roundtrip mismatch: %+v", back)
	}
	if len(back.Faces) != 1 || back.Faces[0].Name != "in" {
		t.Fatalf("face roundtrip mismatch: %+v", back.Faces)
	}
}

func TestDiffPackagesConditionChanged(t *testing.T) {
	from := NewSnapshot("h", "m1", 1, 1, nil,
		[]model.BC{{ID: "bc1", FaceID: "f1", Type: model.BCInletMassFlow, Value: 1.0}})
	to := NewSnapshot("h", "m1", 2, 2, nil,
		[]model.BC{{ID: "bc1", FaceID: "f1", Type: model.BCInletMassFlow, Value: 2.0}})
	d := DiffPackages(from, to, "p1", "p2")
	if d.RegionHashChanged {
		t.Fatal("region hash should not change")
	}
	if len(d.ChangedConditions) != 1 || d.ChangedConditions[0] != "f1" {
		t.Fatalf("expected changed condition f1, got %v", d.ChangedConditions)
	}
	if d.Summary != "条件变更：共享拓扑，条件版本已前进，发布为新包" {
		t.Fatalf("unexpected summary: %s", d.Summary)
	}
}

func TestDiffPackagesRegionChanged(t *testing.T) {
	from := NewSnapshot("h1", "m1", 1, 1, []model.Face{{ID: "f1", Name: "a"}}, nil)
	to := NewSnapshot("h2", "m1", 2, 1, []model.Face{{ID: "f1", Name: "a"}, {ID: "f2", Name: "b"}}, nil)
	d := DiffPackages(from, to, "p1", "p2")
	if !d.RegionHashChanged {
		t.Fatal("expected region hash changed")
	}
	if len(d.AddedFaces) != 1 || d.AddedFaces[0] != "b" {
		t.Fatalf("expected added face b, got %v", d.AddedFaces)
	}
}

func TestPublished(t *testing.T) {
	if Published(&model.SolverPackage{Status: model.PkgBuilding}) {
		t.Fatal("building should not be published")
	}
	if !Published(&model.SolverPackage{Status: model.PkgPublished}) {
		t.Fatal("published should be published")
	}
}
