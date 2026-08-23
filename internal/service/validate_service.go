package service

import (
	"fmt"
	"time"

	"task189-cfdbc/internal/audit"
	"task189-cfdbc/internal/conditions"
	"task189-cfdbc/internal/conservation"
	"task189-cfdbc/internal/model"
	"task189-cfdbc/internal/physics"
	"task189-cfdbc/internal/store"
)

// ValidateService 一致性校验编排：覆盖、守恒、可解性 → 问题清单 → 结论。
type ValidateService struct {
	faces  *store.FaceStore
	bcs    *store.BCStore
	runs   *store.RunStore
	models *store.ModelStore
	gen    *store.IDGen
	db     *store.DB
	now    func() string
}

// NewValidateService 构造校验服务。
func NewValidateService(db *store.DB) *ValidateService {
	return &ValidateService{
		faces:  store.NewFaceStore(db),
		bcs:    store.NewBCStore(db),
		runs:   store.NewRunStore(db),
		models: store.NewModelStore(db),
		gen:    store.NewIDGen("run"),
		db:     db,
		now:    func() string { return time.Now().UTC().Format(time.RFC3339) },
	}
}

// Run 执行一次完整一致性校验，返回运行记录与问题清单。
func (s *ValidateService) Run() (*model.ValidationRun, []*model.Issue, error) {
	m, err := s.models.GetActive()
	if err != nil {
		return nil, nil, model.NewConflict("no active physics model; activate one before validation")
	}
	allFaces, err := s.faces.ListAll()
	if err != nil {
		return nil, nil, err
	}
	allBCs, err := s.bcs.ListAll()
	if err != nil {
		return nil, nil, err
	}

	configVersion, err := s.db.NextConfigVersion()
	if err != nil {
		return nil, nil, err
	}
	runID := s.gen.Next()
	b := audit.NewRunBuilder(runID, m.ID, configVersion-1)

	// 面索引 + 条件按面聚合
	facesByRegion := map[string][]*model.Face{}
	bcByFace := map[string][]*model.BC{}
	for _, f := range allFaces {
		facesByRegion[f.RegionID] = append(facesByRegion[f.RegionID], f)
	}
	for _, bc := range allBCs {
		bcByFace[bc.FaceID] = append(bcByFace[bc.FaceID], bc)
	}

	// 1) 重复面 / 退化面（error）
	for _, f := range allFaces {
		switch f.Status {
		case model.FaceDuplicate:
			b.Add(physics.IssueDuplicateFace, model.SeverityError, f.RegionID, f.ID,
				"duplicate face "+f.Name+" is duplicate of "+f.DuplicateOf)
		case model.FaceDegenerate:
			b.Add(physics.IssueDegenerateFace, model.SeverityError, f.RegionID, f.ID,
				"degenerate face "+f.Name+": "+f.DegenerateCause)
		}
	}

	// 2) 覆盖性检查：每个外露面恰有一种适用条件
	for regionID, fs := range facesByRegion {
		reg := facesByRegion[regionID] // 仅用于区域级统计
		_ = reg
		cov := conditions.CheckCoverage(fs, bcByFace)
		for _, fid := range cov.MissingFaces {
			b.Add(physics.IssueMissingBC, model.SeverityError, regionID, fid,
				"exposed face has no applicable boundary condition")
		}
		for _, fid := range cov.Conflicting {
			b.Add(physics.IssueConflictingBC, model.SeverityError, regionID, fid,
				"exposed face has more than one applicable condition")
		}
	}

	// 3) 耦合面守恒：双侧条件存在 + 单位一致 + 规格一致
	allPairs := conditions.CollectInterfaces(allFaces, bcByFace)
	for _, p := range allPairs {
		pair := conservation.FromBCs(p.Face, p.SideA, p.SideB)
		res := conservation.CheckInterface(pair, physics.BalanceTolerance)
		if p.SideA == nil || p.SideB == nil {
			side := "a"
			if p.SideA != nil {
				side = "b"
			}
			b.Add(physics.IssueMissingBC, model.SeverityError, p.Face.RegionID, p.Face.ID,
				"interface face missing side "+side+" condition")
			continue
		}
		if !res.UnitOK {
			b.Add(physics.IssueUnitMismatch, model.SeverityError, p.Face.RegionID, p.Face.ID,
				"interface sides use different unit systems")
		}
		if !res.Conserved {
			b.Add(physics.IssueUnconservedInterface, model.SeverityError, p.Face.RegionID, p.Face.ID,
				fmt.Sprintf("interface side specs mismatch (A=%.4g, B=%.4g)", res.SideASpec, res.SideBSpec))
		}
	}

	// 4) 入口/出口总量守恒（可压缩/不可压缩均适用）
	bal := conservation.CheckFlowBalance(allBCs, physics.BalanceTolerance)
	if !bal.Balanced {
		b.Add(physics.IssueFlowImbalance, model.SeverityError, "", "",
			fmt.Sprintf("inflow/outflow imbalance: inflow=%.4g outflow=%.4g rel_err=%.2f%%",
				bal.Inflow, bal.Outflow, bal.RelErr*100))
	}

	// 5) 参考压力可解性：不可压缩模型必须有参考压力条件
	hasApprovedRef := false
	for _, bc := range allBCs {
		if bc.Type == model.BCReferencePressure && bc.Status == model.BCStatusApproved {
			hasApprovedRef = true
			break
		}
	}
	ref := conservation.CheckReferencePressure(m, hasApprovedRef)
	if ref.Missing {
		b.Add(physics.IssueMissingRefPressure, model.SeverityError, "", "",
			"incompressible model requires an approved reference pressure condition")
	}

	// 结论归纳
	issues := b.Issues()
	errCount, warnCount := b.Counts()
	result := audit.Classify(issues)

	run := &model.ValidationRun{
		ID:            runID,
		ModelID:       m.ID,
		ConfigVersion: configVersion - 1,
		Result:        result,
		ErrorCount:    errCount,
		WarningCount:  warnCount,
		CreatedAt:     s.now(),
	}
	if err := s.runs.InsertRun(run); err != nil {
		return nil, nil, err
	}
	for _, iss := range issues {
		iss.CreatedAt = run.CreatedAt
		if err := s.runs.InsertIssue(iss); err != nil {
			return nil, nil, err
		}
	}
	return run, issues, nil
}

// GetRun 读取运行记录。
func (s *ValidateService) GetRun(id string) (*model.ValidationRun, error) {
	r, err := s.runs.GetRun(id)
	if err != nil {
		return nil, model.NewNotFound("validation_run", id)
	}
	return r, nil
}

// ListRuns 列出运行记录。
func (s *ValidateService) ListRuns(limit int) ([]*model.ValidationRun, error) {
	return s.runs.ListRuns(limit)
}

// Issues 列出问题。
func (s *ValidateService) Issues(configVersion, limit int) ([]*model.Issue, error) {
	return s.runs.ListIssues(configVersion, limit)
}

// Issue 读取问题。
func (s *ValidateService) Issue(id string) (*model.Issue, error) {
	i, err := s.runs.GetIssue(id)
	if err != nil {
		return nil, model.NewNotFound("issue", id)
	}
	return i, nil
}

// IssuesByRun 读取运行的问题。
func (s *ValidateService) IssuesByRun(runID string) ([]*model.Issue, error) {
	return s.runs.ListIssuesByRun(runID)
}
