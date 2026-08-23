package service

import (
	"testing"

	"task189-cfdbc/internal/mesh"
	"task189-cfdbc/internal/model"
	"task189-cfdbc/internal/store"
)

func TestTask189Bug06_ImportedRegionCannotBeSealed(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	regions := NewRegionService(db)
	region, err := regions.Create(mesh.RegionInput{Name: "empty", Dimension: 3, CellCount: 10})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := regions.Seal(region.ID); err == nil {
		t.Fatal("imported region was sealed")
	}
	stored, err := regions.Get(region.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != model.RegionImported {
		t.Fatalf("region status after rejected seal = %q, want imported", stored.Status)
	}
}
