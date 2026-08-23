package service

import (
	"testing"

	"task189-cfdbc/internal/mesh"
	"task189-cfdbc/internal/store"
)

func TestTask189Bug04_ReimportingSameMeshIsRejected(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	regions := NewRegionService(db)
	region, err := regions.Create(mesh.RegionInput{Name: "r", Dimension: 3, CellCount: 10})
	if err != nil {
		t.Fatal(err)
	}
	inputs := []mesh.FaceInput{{Name: "wall", Kind: "outer", Area: 1, NodeCount: 4}}
	if _, _, err := regions.ImportFaces(region.ID, inputs); err != nil {
		t.Fatal(err)
	}
	if _, _, err := regions.ImportFaces(region.ID, inputs); err == nil {
		t.Fatal("same mesh was imported twice")
	}
	faces, err := regions.ListFaces(region.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(faces) != 1 {
		t.Fatalf("face count after duplicate import = %d, want 1", len(faces))
	}
}
