package service

import (
	"errors"
	"time"

	"task189-cfdbc/internal/audit"
	"task189-cfdbc/internal/model"
	"task189-cfdbc/internal/release"
	"task189-cfdbc/internal/store"
)

// PackageService 求解前置包编排：构建（绑定拓扑哈希+条件版本+快照）、
// 发布（不可改写）、派生（修订网格/条件 → 新包）、差异。
type PackageService struct {
	pkgs   *store.PackageStore
	faces  *store.FaceStore
	bcs    *store.BCStore
	runs   *store.RunStore
	models *store.ModelStore
	gen    *store.IDGen
	db     *store.DB
	now    func() string
}

// NewPackageService 构造包服务。
func NewPackageService(db *store.DB) *PackageService {
	return &PackageService{
		pkgs:   store.NewPackageStore(db),
		faces:  store.NewFaceStore(db),
		bcs:    store.NewBCStore(db),
		runs:   store.NewRunStore(db),
		models: store.NewModelStore(db),
		gen:    store.NewIDGen("pkg"),
		db:     db,
		now:    func() string { return time.Now().UTC().Format(time.RFC3339) },
	}
}

// Build 构建求解前置包：绑定当前拓扑哈希、条件版本与快照。
func (s *PackageService) Build(name string) (*model.SolverPackage, error) {
	m, err := s.models.GetActive()
	if err != nil {
		return nil, model.NewConflict("no active physics model; activate one before building package")
	}
	allFaces, err := s.faces.ListAll()
	if err != nil {
		return nil, err
	}
	allBCs, err := s.bcs.ListAll()
	if err != nil {
		return nil, err
	}
	if len(allFaces) == 0 {
		return nil, model.NewInvalid("no faces imported; build after importing mesh")
	}

	regionHash := ""
	regions, rerr := store.NewRegionStore(s.db).List()
	if rerr != nil {
		return nil, rerr
	}
	if len(regions) > 0 {
		regionHash = regions[0].MeshHash
	}
	configVersion, cerr := s.db.NextConfigVersion()
	if cerr != nil {
		return nil, cerr
	}
	condsVersion, verr := s.db.NextConditionsVersion()
	if verr != nil {
		return nil, verr
	}
	if name == "" {
		name = "package-" + time.Now().Format("20060102T150405")
	}
	snap := release.NewSnapshot(regionHash, m.ID, configVersion-1, condsVersion,
		derefFaces(allFaces), derefBCs(allBCs))
	raw, serr := snap.Encode()
	if serr != nil {
		return nil, model.NewInvalid("snapshot encode failed: %v", serr)
	}
	p := &model.SolverPackage{
		ID:                s.gen.Next(),
		Name:              name,
		ConfigVersion:     configVersion - 1,
		RegionHash:        regionHash,
		ConditionsVersion: condsVersion,
		ModelID:           m.ID,
		Snapshot:          raw,
		Status:            model.PkgBuilding,
		CreatedAt:         s.now(),
	}
	if err := s.pkgs.Insert(p); err != nil {
		return nil, err
	}
	return p, nil
}

// Publish 发布包：building → published。发布后禁止直接改写。
func (s *PackageService) Publish(id string) (*model.SolverPackage, error) {
	p, err := s.pkgs.Get(id)
	if err != nil {
		return nil, model.NewNotFound("solver_package", id)
	}
	if p.Status != model.PkgBuilding {
		return nil, model.NewConflict("package %s not in building state (status=%s)", id, p.Status)
	}
	lastRun, err := s.runs.LatestForModel(p.ModelID)
	if err != nil {
		if errors.Is(err, store.ErrNoRows) {
			return nil, model.NewConflict("package %s has no validation run", id)
		}
		return nil, err
	}
	if lastRun.ConfigVersion < p.ConfigVersion || !audit.Publishable(lastRun.Result) {
		return nil, model.NewConflict("package %s is not backed by a solvable validation", id)
	}
	if err := s.pkgs.UpdateStatus(id, model.PkgPublished); err != nil {
		return nil, err
	}
	p.Status = model.PkgPublished
	p.PublishedAt = s.now()
	return p, nil
}

// List 列出包。
func (s *PackageService) List() ([]*model.SolverPackage, error) { return s.pkgs.List() }

// Get 读取包。
func (s *PackageService) Get(id string) (*model.SolverPackage, error) {
	p, err := s.pkgs.Get(id)
	if err != nil {
		return nil, model.NewNotFound("solver_package", id)
	}
	return p, nil
}

// Diff 计算两个包快照差异。
func (s *PackageService) Diff(fromID, toID string) (*release.Diff, error) {
	from, err := s.pkgs.Get(fromID)
	if err != nil {
		return nil, model.NewNotFound("solver_package", fromID)
	}
	to, err := s.pkgs.Get(toID)
	if err != nil {
		return nil, model.NewNotFound("solver_package", toID)
	}
	fromSnap, err1 := release.DecodeSnapshot(from.Snapshot)
	if err1 != nil {
		return nil, model.NewInvalid("decode from snapshot: %v", err1)
	}
	toSnap, err2 := release.DecodeSnapshot(to.Snapshot)
	if err2 != nil {
		return nil, model.NewInvalid("decode to snapshot: %v", err2)
	}
	d := release.DiffPackages(fromSnap, toSnap, from.ID, to.ID)
	return &d, nil
}

// derefFaces 把面指针切片转为值切片（快照不可变拷贝）。
func derefFaces(fs []*model.Face) []model.Face {
	out := make([]model.Face, len(fs))
	for i, f := range fs {
		out[i] = *f
	}
	return out
}

// derefBCs 把条件指针切片转为值切片。
func derefBCs(bcs []*model.BC) []model.BC {
	out := make([]model.BC, len(bcs))
	for i, b := range bcs {
		out[i] = *b
	}
	return out
}

// Derive 派生新包：修订网格（新区域哈希）或条件变更后创建新包。
// 已发布包为不可变基线；派生包保留对新基线的引用语义（快照独立）。
// 只有 published 状态的包可作为派生基线；building 及其他状态必须被拒绝。
func (s *PackageService) Derive(name, baseID string) (*model.SolverPackage, error) {
	base, err := s.pkgs.Get(baseID)
	if err != nil {
		return nil, model.NewNotFound("solver_package", baseID)
	}
	if !release.CanDerive(base) {
		return nil, model.NewConflict("package %s must be published before deriving (status=%s)", baseID, base.Status)
	}
	return s.Build(name)
}
