package mesh

import (
	"testing"

	"task189-cfdbc/internal/model"
)

func TestTask189Bug07_IncompleteInterfaceNeighborIsIsolated(t *testing.T) {
	region := &model.Region{ID: "r1"}
	faces := []*model.Face{{
		ID: "f1", RegionID: "r1", Kind: model.KindInterface,
		NeighborRegion: "r2", NeighborFace: "",
	}}
	result := CheckConnectivity(region, faces)
	if result.Connected {
		t.Fatal("interface with missing neighbor face was classified connected")
	}
	if len(result.OrphanFaces) != 1 || result.OrphanFaces[0] != "f1" {
		t.Fatalf("orphan faces = %v, want [f1]", result.OrphanFaces)
	}
}
