package service

import (
	"testing"

	"task189-cfdbc/internal/conditions"
	"task189-cfdbc/internal/mesh"
	"task189-cfdbc/internal/model"
	"task189-cfdbc/internal/store"
)

func TestTask189Bug09_DuplicateInterfaceSideIsConflicting(t *testing.T) {
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
	_, faces, err := regions.ImportFaces(region.ID, []mesh.FaceInput{{Name: "couple", Kind: "interface", Area: 1, NodeCount: 4, NeighborRegion: "r2", NeighborFace: "other"}})
	if err != nil {
		t.Fatal(err)
	}
	in := conditions.BCInput{FaceID: faces[0].ID, Type: model.BCInterfaceSideA, Unit: model.UnitSI, Value: 1}
	if _, err := conds.Assign(in); err != nil {
		t.Fatal(err)
	}
	second, err := conds.Assign(in)
	if err != nil {
		t.Fatal(err)
	}
	if second.Status != model.BCStatusConflicting {
		t.Fatalf("duplicate interface side status = %q, want conflicting", second.Status)
	}
}
