package service

import (
	"path/filepath"
	"testing"

	"task189-cfdbc/internal/mesh"
	"task189-cfdbc/internal/model"
	"task189-cfdbc/internal/store"
)

// newTestDB 构造一个临时 SQLite 库用于服务层测试。
func newTestDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// TestSealRejectsImportedRegion 断言刚登记、尚未导入任何面的区域不能封存，
// 且封存被拒绝后区域状态保持 imported 不变（sealed_at 也为空）。
func TestSealRejectsImportedRegion(t *testing.T) {
	svc := NewRegionService(newTestDB(t))
	r, err := svc.Create(mesh.RegionInput{Name: "empty", Dimension: 3, CellCount: 100})
	if err != nil {
		t.Fatalf("create region: %v", err)
	}
	if r.Status != model.RegionImported {
		t.Fatalf("expected imported after create, got %s", r.Status)
	}

	if _, err := svc.Seal(r.ID); err == nil {
		t.Fatal("expected seal to be rejected for imported region, got nil error")
	}

	// 封存被拒绝后状态必须保持 imported。
	reloaded, err := svc.Get(r.ID)
	if err != nil {
		t.Fatalf("get region: %v", err)
	}
	if reloaded.Status != model.RegionImported {
		t.Fatalf("expected status to remain imported after rejected seal, got %s", reloaded.Status)
	}
	if reloaded.SealedAt != "" {
		t.Fatalf("expected sealed_at to stay empty after rejected seal, got %s", reloaded.SealedAt)
	}
}

// TestSealAcceptsImportedFacesRegion 断言导入面（进入 connected）后可以正常封存。
func TestSealAcceptsImportedFacesRegion(t *testing.T) {
	svc := NewRegionService(newTestDB(t))
	r, err := svc.Create(mesh.RegionInput{Name: "duct", Dimension: 3, CellCount: 100})
	if err != nil {
		t.Fatalf("create region: %v", err)
	}
	if _, _, err := svc.ImportFaces(r.ID, []mesh.FaceInput{
		{Name: "in", Kind: "outer", Area: 0.25, NormalX: -1, NodeCount: 4},
		{Name: "out", Kind: "outer", Area: 0.25, NormalX: 1, NodeCount: 4},
	}); err != nil {
		t.Fatalf("import faces: %v", err)
	}
	sealed, err := svc.Seal(r.ID)
	if err != nil {
		t.Fatalf("seal connected region: %v", err)
	}
	if sealed.Status != model.RegionSealed {
		t.Fatalf("expected sealed, got %s", sealed.Status)
	}
	if sealed.SealedAt == "" {
		t.Fatal("expected sealed_at set after sealing")
	}
}

// TestSealIdempotent 断言对已封存区域再次封存为幂等成功，不报错。
func TestSealIdempotent(t *testing.T) {
	svc := NewRegionService(newTestDB(t))
	r, err := svc.Create(mesh.RegionInput{Name: "duct2", Dimension: 3, CellCount: 100})
	if err != nil {
		t.Fatalf("create region: %v", err)
	}
	if _, _, err := svc.ImportFaces(r.ID, []mesh.FaceInput{
		{Name: "in", Kind: "outer", Area: 0.25, NormalX: -1, NodeCount: 4},
	}); err != nil {
		t.Fatalf("import faces: %v", err)
	}
	if _, err := svc.Seal(r.ID); err != nil {
		t.Fatalf("first seal: %v", err)
	}
	if _, err := svc.Seal(r.ID); err != nil {
		t.Fatalf("idempotent seal: %v", err)
	}
}
