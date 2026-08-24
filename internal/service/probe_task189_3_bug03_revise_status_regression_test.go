package service

import (
	"testing"

	"task189-cfdbc/internal/conditions"
	"task189-cfdbc/internal/mesh"
	"task189-cfdbc/internal/model"
	"task189-cfdbc/internal/store"
)

func TestTask189Bug03_RevisionPersistsConflictStatus(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	regions := NewRegionService(db)
	conds := NewConditionService(db)
	region, err := regions.Create(mesh.RegionInput{Name: "r", Dimension: 3, CellCount: 10})
	if err != nil {
		t.Fatal(err)
	}
	_, faces, err := regions.ImportFaces(region.ID, []mesh.FaceInput{{Name: "in", Kind: "outer", Area: 1, NodeCount: 4}})
	if err != nil {
		t.Fatal(err)
	}
	bc, err := conds.Assign(conditions.BCInput{FaceID: faces[0].ID, Type: model.BCInletMassFlow, Unit: model.UnitSI, Value: 1})
	if err != nil {
		t.Fatal(err)
	}
	revised, err := conds.Revise(bc.ID, -2, 0, bc.Version)
	if err != nil {
		t.Fatal(err)
	}
	if revised.Status != model.BCStatusConflicting {
		t.Fatalf("revision response status = %q, want conflicting", revised.Status)
	}
	stored, err := conds.Get(bc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != model.BCStatusConflicting {
		t.Fatalf("persisted revision status = %q, want conflicting", stored.Status)
	}
}
