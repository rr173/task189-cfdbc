package service

import (
	"testing"

	"task189-cfdbc/internal/mesh"
	"task189-cfdbc/internal/model"
	"task189-cfdbc/internal/physics"
	"task189-cfdbc/internal/store"
)

func TestTask189Bug08_DeriveRequiresPublishedBase(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	regions := NewRegionService(db)
	models := NewModelService(db)
	packages := NewPackageService(db)
	region, err := regions.Create(mesh.RegionInput{Name: "r", Dimension: 3, CellCount: 10})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := regions.ImportFaces(region.ID, []mesh.FaceInput{{Name: "wall", Kind: "outer", Area: 1, NodeCount: 4}}); err != nil {
		t.Fatal(err)
	}
	m, err := models.Create(physics.ModelInput{Name: "compressible", FlowType: model.FlowCompressible, DefaultUnit: model.UnitSI})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := models.Activate(m.ID); err != nil {
		t.Fatal(err)
	}
	base, err := packages.Build("building-base")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := packages.Derive("child", base.ID); err == nil {
		t.Fatal("derived package from non-published base")
	}
}
