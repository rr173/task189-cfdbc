package service

import (
	"time"

	"task189-cfdbc/internal/conditions"
	"task189-cfdbc/internal/model"
	"task189-cfdbc/internal/physics"
	"task189-cfdbc/internal/store"
)

// ModelService 物理模型编排。
type ModelService struct {
	models *store.ModelStore
	gen    *store.IDGen
	now    func() string
}

// NewModelService 构造模型服务。
func NewModelService(db *store.DB) *ModelService {
	return &ModelService{
		models: store.NewModelStore(db),
		gen:    store.NewIDGen("mdl"),
		now:    func() string { return time.Now().UTC().Format(time.RFC3339) },
	}
}

// Create 创建物理模型。
func (s *ModelService) Create(in physics.ModelInput) (*model.PhysicsModel, error) {
	m, err := physics.NewModel(in)
	if err != nil {
		return nil, err
	}
	m.ID = s.gen.Next()
	m.CreatedAt = s.now()
	if err := s.models.Insert(m); err != nil {
		return nil, err
	}
	return m, nil
}

// List 列出模型。
func (s *ModelService) List() ([]*model.PhysicsModel, error) { return s.models.List() }

// Get 读取模型。
func (s *ModelService) Get(id string) (*model.PhysicsModel, error) {
	m, err := s.models.Get(id)
	if err != nil {
		return nil, model.NewNotFound("physics_model", id)
	}
	return m, nil
}

// Activate 激活模型（同时取消其它模型激活）。
func (s *ModelService) Activate(id string) (*model.PhysicsModel, error) {
	if _, err := s.models.Get(id); err != nil {
		return nil, model.NewNotFound("physics_model", id)
	}
	if err := s.models.SetActive(id, true); err != nil {
		return nil, err
	}
	return s.models.Get(id)
}

// GetActive 读取激活模型。
func (s *ModelService) GetActive() (*model.PhysicsModel, error) {
	return s.models.GetActive()
}

// ConditionService 边界条件编排：分配、评估适用性、批准（乐观锁）。
type ConditionService struct {
	bcs   *store.BCStore
	faces *store.FaceStore
	gen   *store.IDGen
	db    *store.DB
	now   func() string
}

// NewConditionService 构造条件服务。
func NewConditionService(db *store.DB) *ConditionService {
	return &ConditionService{
		bcs:   store.NewBCStore(db),
		faces: store.NewFaceStore(db),
		gen:   store.NewIDGen("bc"),
		db:    db,
		now:   func() string { return time.Now().UTC().Format(time.RFC3339) },
	}
}

// Assign 分配边界条件到面：检查角色匹配 + 数值合法 → applicable/conflicting。
func (s *ConditionService) Assign(in conditions.BCInput) (*model.BC, error) {
	if in.FaceID == "" {
		return nil, model.NewInvalid("face_id required")
	}
	f, err := s.faces.Get(in.FaceID)
	if err != nil {
		return nil, model.NewNotFound("face", in.FaceID)
	}
	if in.Unit != model.UnitSI && in.Unit != model.UnitCGS {
		return nil, model.NewInvalid("unit must be si or cgs")
	}
	if !model.IsKnownBCType(in.Type) {
		return nil, model.NewInvalid("unknown boundary condition type")
	}
	ts := s.now()
	bc := &model.BC{
		ID:        s.gen.Next(),
		FaceID:    f.ID,
		RegionID:  f.RegionID,
		Type:      in.Type,
		Unit:      in.Unit,
		Value:     in.Value,
		Secondary: in.Secondary,
		Status:    model.BCStatusDraft,
		Version:   1,
		CreatedAt: ts,
		UpdatedAt: ts,
	}
	// 评估适用性（类型-角色 + 数值合法性）
	conditions.Assess(f, bc)
	// 覆盖冲突：同一外露面已有适用条件时，新条件立即标记 conflicting。
	// 参考压力为全局标定，不参与覆盖名额，可挂在已有条件面上。
	if bc.Type != model.BCReferencePressure {
		if existing, err := s.bcs.ListByFace(f.ID); err == nil && conditions.HasExistingCondition(existing) {
			bc.Status = model.BCStatusConflicting
		}
	}
	if err := s.bcs.Insert(bc); err != nil {
		return nil, err
	}
	return bc, nil
}

// Approve 批准条件：仅 applicable 可批准为 approved（乐观锁）。
func (s *ConditionService) Approve(id string) (*model.BC, error) {
	bc, err := s.bcs.Get(id)
	if err != nil {
		return nil, model.NewNotFound("boundary_condition", id)
	}
	if bc.Status != model.BCStatusApplicable {
		return nil, model.NewConflict("condition %s not applicable (status=%s)", id, bc.Status)
	}
	v, err := s.bcs.UpdateStatusAndVersion(id, model.BCStatusApproved, bc.Version)
	if err != nil {
		return nil, err
	}
	bc.Status = model.BCStatusApproved
	bc.Version = v
	return bc, nil
}

// Revise 修订条件数值（乐观锁：并发版本冲突检测）。
func (s *ConditionService) Revise(id string, value, secondary float64, expectVersion int) (*model.BC, error) {
	bc, err := s.bcs.Get(id)
	if err != nil {
		return nil, model.NewNotFound("boundary_condition", id)
	}
	if bc.Status == model.BCStatusApproved {
		return nil, model.NewConflict("condition %s approved, cannot revise; create a new condition", id)
	}
	v, err := s.bcs.UpdateValueAndStatus(id, value, secondary, expectVersion, "")
	if err != nil {
		return nil, err
	}
	bc.Value, bc.Secondary, bc.Version = value, secondary, v
	// 修订后重新评估适用性
	f, ferr := s.faces.Get(bc.FaceID)
	if ferr == nil {
		conditions.Assess(f, bc)
		if bc.Status == model.BCStatusConflicting {
			if _, err := s.bcs.UpdateValueAndStatus(id, value, secondary, v, bc.Status); err != nil {
				return nil, err
			}
			bc.Version = v + 1
		}
	}
	return bc, nil
}

// List 列出条件。
func (s *ConditionService) List() ([]*model.BC, error) { return s.bcs.ListAll() }

// ListByFace 列出面的条件。
func (s *ConditionService) ListByFace(faceID string) ([]*model.BC, error) {
	return s.bcs.ListByFace(faceID)
}

// Get 读取条件。
func (s *ConditionService) Get(id string) (*model.BC, error) {
	bc, err := s.bcs.Get(id)
	if err != nil {
		return nil, model.NewNotFound("boundary_condition", id)
	}
	return bc, nil
}
