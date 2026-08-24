// Package service 编排网格、物理、条件、守恒、审计与发布逻辑，
// 负责把 HTTP 请求翻译为持久化 + 纯逻辑的组合调用。
package service

import (
	"time"

	"task189-cfdbc/internal/mesh"
	"task189-cfdbc/internal/model"
	"task189-cfdbc/internal/store"
)

// RegionService 区域与面编排。
type RegionService struct {
	regions *store.RegionStore
	faces   *store.FaceStore
	gen     *store.IDGen
	now     func() string
}

// NewRegionService 构造区域服务。
func NewRegionService(db *store.DB) *RegionService {
	return &RegionService{
		regions: store.NewRegionStore(db),
		faces:   store.NewFaceStore(db),
		gen:     store.NewIDGen("reg"),
		now:     func() string { return time.Now().UTC().Format(time.RFC3339) },
	}
}

// Create 登记网格区域并计算空拓扑哈希。
func (s *RegionService) Create(in mesh.RegionInput) (*model.Region, error) {
	if in.Name == "" {
		return nil, model.NewInvalid("region name required")
	}
	if in.Dimension != 2 && in.Dimension != 3 {
		return nil, model.NewInvalid("dimension must be 2 or 3")
	}
	if in.CellCount <= 0 {
		return nil, model.NewInvalid("cell_count must be positive")
	}
	ts := s.now()
	r := &model.Region{
		ID:          s.gen.Next(),
		Name:        in.Name,
		Dimension:   in.Dimension,
		CellCount:   in.CellCount,
		Status:      model.RegionImported,
		MeshHash:    "",
		CreatedAt:   ts,
		UpdatedAt:   ts,
		Description: in.Description,
	}
	if err := s.regions.Insert(r); err != nil {
		return nil, err
	}
	return r, nil
}

// Get 读取区域。
func (s *RegionService) Get(id string) (*model.Region, error) {
	r, err := s.regions.Get(id)
	if err != nil {
		return nil, model.NewNotFound("region", id)
	}
	return r, nil
}

// List 列出区域。
func (s *RegionService) List() ([]*model.Region, error) { return s.regions.List() }

// ImportFaces 导入面集合：分类 + 连通性检测 + 拓扑哈希 + 幂等。
// 相同网格哈希（同名面集合）导入直接返回既有区域信息（幂等）。
func (s *RegionService) ImportFaces(regionID string, inputs []mesh.FaceInput) (*model.Region, []*model.Face, error) {
	r, err := s.regions.Get(regionID)
	if err != nil {
		return nil, nil, model.NewNotFound("region", regionID)
	}
	if r.Status == model.RegionSealed {
		return nil, nil, model.NewConflict("region %s sealed, cannot import faces", regionID)
	}
	if len(inputs) == 0 {
		return nil, nil, model.NewInvalid("faces input empty")
	}
	ts := s.now()
	faces := make([]*model.Face, 0, len(inputs))
	for _, in := range inputs {
		if in.Name == "" {
			return nil, nil, model.NewInvalid("face name required")
		}
		kind := model.FaceKind(in.Kind)
		if kind != model.KindOuter && kind != model.KindInterface {
			return nil, nil, model.NewInvalid("face kind must be outer or interface")
		}
		f := &model.Face{
			ID:             s.gen.Next(),
			RegionID:       regionID,
			Name:           in.Name,
			Kind:           kind,
			Area:           in.Area,
			NormalX:        in.NormalX,
			NormalY:        in.NormalY,
			NormalZ:        in.NormalZ,
			NodeCount:      in.NodeCount,
			NeighborRegion: in.NeighborRegion,
			NeighborFace:   in.NeighborFace,
			CreatedAt:      ts,
		}
		faces = append(faces, f)
	}

	// 分类：退化/外露/耦合 + 重复面检测
	mesh.Classify(faces)

	// 连通性检测：耦合面必须有对侧
	conn := mesh.CheckConnectivity(r, faces)
	hash := mesh.HashRegion(regionID, faces)

	// 幂等：区域一旦已有拓扑，任何再次导入都必须拒绝——既不能原地
	// 替换（哈希不同），也不能重复落库（哈希相同）。相同面集合再次导入
	// 属幂等重复，直接拒绝，数据库中的面数量不增加。
	if r.MeshHash != "" {
		if r.MeshHash == hash {
			return nil, nil, model.NewConflict("region %s already has identical mesh hash %s; re-importing the same face set is not permitted", regionID, r.MeshHash)
		}
		return nil, nil, model.NewConflict("region %s already has mesh hash %s, import would change topology; derive a new region instead", regionID, r.MeshHash)
	}

	// 落库
	for _, f := range faces {
		if err := s.faces.Insert(f); err != nil {
			return nil, nil, err
		}
	}
	if err := s.regions.UpdateFaceCount(regionID, len(faces), hash); err != nil {
		return nil, nil, err
	}
	// 区域状态推进：连通 → connected；孤立（耦合面无对侧）→ isolated
	status := model.RegionConnected
	if !conn.Connected {
		status = model.RegionIsolated
	}
	if err := s.regions.UpdateStatus(regionID, status); err != nil {
		return nil, nil, err
	}
	r.MeshHash = hash
	r.FaceCount = len(faces)
	r.Status = status
	r.UpdatedAt = ts
	return r, faces, nil
}

// ListFaces 列出区域面。
func (s *RegionService) ListFaces(regionID string) ([]*model.Face, error) {
	if _, err := s.regions.Get(regionID); err != nil {
		return nil, model.NewNotFound("region", regionID)
	}
	return s.faces.ListByRegion(regionID)
}

// GetFace 读取面。
func (s *RegionService) GetFace(id string) (*model.Face, error) {
	f, err := s.faces.Get(id)
	if err != nil {
		return nil, model.NewNotFound("face", id)
	}
	return f, nil
}

// Seal 封存区域：进入 sealed 终态，禁止再改拓扑。
func (s *RegionService) Seal(id string) (*model.Region, error) {
	r, err := s.regions.Get(id)
	if err != nil {
		return nil, model.NewNotFound("region", id)
	}
	if r.Status == model.RegionSealed {
		return r, nil // 幂等
	}
	if !mesh.CanSeal(r.Status, r.FaceCount) {
		return nil, model.NewConflict("region %s must have an imported topology before sealing", id)
	}
	ts := s.now()
	if err := s.regions.UpdateStatus(id, model.RegionSealed, ts); err != nil {
		return nil, err
	}
	r.Status = model.RegionSealed
	r.SealedAt = ts
	r.UpdatedAt = ts
	return r, nil
}

// RegionByHash 按哈希查找区域（幂等导入复用）。
func (s *RegionService) RegionByHash(hash string) (*model.Region, error) {
	regions, err := s.regions.List()
	if err != nil {
		return nil, err
	}
	for _, r := range regions {
		if r.MeshHash == hash {
			return r, nil
		}
	}
	return nil, model.NewNotFound("region", hash)
}
