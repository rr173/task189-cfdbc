package service

import (
	"math"
	"testing"

	"task189-cfdbc/internal/conditions"
	"task189-cfdbc/internal/mesh"
	"task189-cfdbc/internal/model"
	"task189-cfdbc/internal/store"
)

func TestTask189Bug01_InfiniteBoundaryValueIsRejected(t *testing.T) {
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
	_, faces, err := regions.ImportFaces(region.ID, []mesh.FaceInput{{Name: "wall", Kind: "outer", Area: 1, NodeCount: 4}})
	if err != nil {
		t.Fatal(err)
	}
	bc, err := conds.Assign(conditions.BCInput{FaceID: faces[0].ID, Type: model.BCWallNoSlip, Unit: model.UnitSI, Value: math.Inf(1)})
	if err != nil {
		t.Fatal(err)
	}
	if bc.Status != model.BCStatusConflicting {
		panic("infinite boundary value was accepted")
	}
}
