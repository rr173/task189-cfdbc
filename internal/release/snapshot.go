// Package release 管理求解前置包：发布快照、条件版本绑定、
// 包间差异与派生。发布后不可改写，修改条件派生新包。
package release

import (
	"encoding/json"
	"sort"

	"task189-cfdbc/internal/model"
)

// Snapshot 求解前置包快照：冻结区域拓扑 + 物理模型 + 条件版本。
type Snapshot struct {
	RegionHash        string       `json:"region_hash"`
	ConfigVersion     int          `json:"config_version"`
	ConditionsVersion int          `json:"conditions_version"`
	ModelID           string       `json:"model_id"`
	Faces             []model.Face `json:"faces"`
	Conditions        []model.BC   `json:"conditions"`
}

// NewSnapshot 构造快照（拷贝输入防止外部修改）。
func NewSnapshot(regionHash, modelID string, configVersion, conditionsVersion int,
	faces []model.Face, bcs []model.BC) Snapshot {
	fs := make([]model.Face, len(faces))
	copy(fs, faces)
	cs := make([]model.BC, len(bcs))
	copy(cs, bcs)
	return Snapshot{
		RegionHash:        regionHash,
		ModelID:           modelID,
		ConfigVersion:     configVersion,
		ConditionsVersion: conditionsVersion,
		Faces:             fs,
		Conditions:        cs,
	}
}

// Encode 序列化快照为 JSON 字符串。
func (s Snapshot) Encode() (string, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// DecodeSnapshot 反序列化快照。
func DecodeSnapshot(raw string) (Snapshot, error) {
	var s Snapshot
	err := json.Unmarshal([]byte(raw), &s)
	return s, err
}

// Diff 前置包差异：比较两个包的快照，输出变更摘要。
type Diff struct {
	FromPackageID          string   `json:"from_package_id"`
	ToPackageID            string   `json:"to_package_id"`
	RegionHashChanged      bool     `json:"region_hash_changed"`
	ConfigVersionDelta     int      `json:"config_version_delta"`
	ConditionsVersionDelta int      `json:"conditions_version_delta"`
	AddedFaces             []string `json:"added_faces"`
	RemovedFaces           []string `json:"removed_faces"`
	ChangedConditions      []string `json:"changed_conditions"`
	Summary                string   `json:"summary"`
}

// DiffPackages 计算两个包的快照差异。若 To 为空快照则视为同包全量对照。
func DiffPackages(from, to Snapshot, fromID, toID string) Diff {
	d := Diff{FromPackageID: fromID, ToPackageID: toID}
	d.RegionHashChanged = from.RegionHash != to.RegionHash
	d.ConfigVersionDelta = to.ConfigVersion - from.ConfigVersion
	d.ConditionsVersionDelta = to.ConditionsVersion - from.ConditionsVersion

	faceSet := func(s Snapshot) map[string]model.Face {
		m := make(map[string]model.Face, len(s.Faces))
		for _, f := range s.Faces {
			m[f.Name] = f
		}
		return m
	}
	fromF, toF := faceSet(from), faceSet(to)
	for name := range toF {
		if _, ok := fromF[name]; !ok {
			d.AddedFaces = append(d.AddedFaces, name)
		}
	}
	for name := range fromF {
		if _, ok := toF[name]; !ok {
			d.RemovedFaces = append(d.RemovedFaces, name)
		}
	}
	sort.Strings(d.AddedFaces)
	sort.Strings(d.RemovedFaces)

	condKey := func(s Snapshot) map[string]model.BC {
		m := make(map[string]model.BC, len(s.Conditions))
		for _, c := range s.Conditions {
			m[c.FaceID] = c
		}
		return m
	}
	fromC, toC := condKey(from), condKey(to)
	for faceID, tc := range toC {
		fc, ok := fromC[faceID]
		if !ok || fc.Value != tc.Value || fc.Type != tc.Type || fc.Unit != tc.Unit {
			d.ChangedConditions = append(d.ChangedConditions, faceID)
		}
	}
	sort.Strings(d.ChangedConditions)

	switch {
	case d.RegionHashChanged:
		d.Summary = "拓扑变更：区域网格哈希不同，需重新校验并派生新包"
	case len(d.ChangedConditions) > 0:
		d.Summary = "条件变更：共享拓扑，条件版本已前进，发布为新包"
	default:
		d.Summary = "无实质差异"
	}
	return d
}

// Published 判断包是否已发布（发布后不可直接改写）。
func Published(p *model.SolverPackage) bool { return p.Status == model.PkgPublished }

// CanDerive restricts package branching to an immutable published baseline.
func CanDerive(p *model.SolverPackage) bool { return p != nil && Published(p) }
