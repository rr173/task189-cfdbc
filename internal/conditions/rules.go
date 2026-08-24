// Package conditions 校验边界条件的适用性：每个外露面恰有一种
// 适用条件、耦合面两侧成对、类型与面角色匹配、单位一致。
package conditions

import (
	"task189-cfdbc/internal/model"
)

// BCInput 边界条件创建输入。
type BCInput struct {
	FaceID    string           `json:"face_id"`
	RegionID  string           `json:"region_id"`
	Type      model.BCType     `json:"type"`
	Unit      model.UnitSystem `json:"unit"`
	Value     float64          `json:"value"`
	Secondary float64          `json:"secondary,omitempty"`
}

// FaceRoleAllowed 判断条件类型是否可用于该面角色。
// 外露面允许入口/出口/壁面/对称/参考压力；耦合面只允许 interface 两侧。
func FaceRoleAllowed(f *model.Face, t model.BCType) bool {
	if f.Kind == model.KindInterface {
		return t == model.BCInterfaceSideA || t == model.BCInterfaceSideB
	}
	switch t {
	case model.BCInterfaceSideA, model.BCInterfaceSideB:
		return false
	}
	return true
}

// SideForType 返回 interface 条件所属侧（A/B）。
func SideForType(t model.BCType) (string, bool) {
	switch t {
	case model.BCInterfaceSideA:
		return "a", true
	case model.BCInterfaceSideB:
		return "b", true
	}
	return "", false
}

// Assess 评估单条条件的适用状态。规则：
//  1. 类型与面角色不匹配 → conflicting；
//  2. 数值非法（非有限值、面积为正时需要面积>0 之外的约束）→ conflicting；
//  3. 否则 applicable。
func Assess(f *model.Face, bc *model.BC) {
	if !model.IsKnownBCType(bc.Type) {
		bc.Status = model.BCStatusConflicting
		return
	}
	if !FaceRoleAllowed(f, bc.Type) {
		bc.Status = model.BCStatusConflicting
		return
	}
	if isBadValue(bc.Type, bc.Value) {
		bc.Status = model.BCStatusConflicting
		return
	}
	bc.Status = model.BCStatusApplicable
}

// isBadValue 判定条件数值是否非法（→ conflicting）。
// 非有限值（NaN/Inf）非法；入口/出口质量流量为负在物理上无意义
// （流向反转应由入口/出口类型表达，而非负流量值），同样视为非法。
func isBadValue(t model.BCType, v float64) bool {
	if !model.IsFiniteBoundaryValue(v) {
		return true
	}
	switch t {
	case model.BCInletMassFlow, model.BCOutletMassFlow:
		return v < 0
	default:
		return false
	}
}

// HasExistingCondition 判断面是否已存在适用/已批准的覆盖条件。
// 参考压力是全局标定，不占用面的覆盖名额；耦合面条件由配对逻辑处理。
// 用于分配时立即标记覆盖冲突（同一外露面两种条件）。
func HasExistingCondition(bcs []*model.BC) bool {
	for _, bc := range bcs {
		if bc.Type == model.BCReferencePressure {
			continue
		}
		if bc.Status == model.BCStatusApplicable || bc.Status == model.BCStatusApproved {
			return true
		}
	}
	return false
}

// IsCoveringType 判断条件类型是否参与“覆盖名额”计算。
// 参考压力为全局标定，不参与；interface 类型由配对逻辑处理。
func IsCoveringType(t model.BCType) bool {
	return t != model.BCReferencePressure && t != model.BCInterfaceSideA && t != model.BCInterfaceSideB
}

// CoverageCheck 覆盖性检查结果：返回缺失（无条件的 exposed 面）与
// 冲突（同一 exposed 面多于一条条件）列表。
type CoverageCheck struct {
	MissingFaces []string
	Conflicting  []string
}

// CheckCoverage 对区域内的面执行覆盖检查：
// 每个 exposed 面必须恰有一条适用条件。
func CheckCoverage(faces []*model.Face, bcByFace map[string][]*model.BC) CoverageCheck {
	var out CoverageCheck
	for _, f := range faces {
		if f.Status != model.FaceExposed {
			continue
		}
		bcs := bcByFace[f.ID]
		applicable := 0
		for _, bc := range bcs {
			if !IsCoveringType(bc.Type) {
				continue
			}
			if bc.Status == model.BCStatusApplicable || bc.Status == model.BCStatusApproved {
				applicable++
			}
		}
		switch {
		case applicable == 0:
			out.MissingFaces = append(out.MissingFaces, f.ID)
		case applicable > 1:
			out.Conflicting = append(out.Conflicting, f.ID)
		}
	}
	return out
}

// InterfacePair 耦合面双侧条件：SideA 属于本面（或对侧面），SideB 反之。
type InterfacePair struct {
	Face  *model.Face
	SideA *model.BC
	SideB *model.BC
}

// isSideA 判断条件是否为 interface 的 A 侧。
func isSideA(t model.BCType) bool { return t == model.BCInterfaceSideA }

// CollectInterfaces 收集耦合面及其 A/B 侧条件。
// 配对规则：耦合面按邻接关系配对——面 f 的对侧面为
// （f.NeighborRegion 内，ID 或名称匹配 f.NeighborFace 的面）。
// 本面条件为 side_a 时对侧条件应为 side_b，反之亦然；缺侧时对应指针为 nil。
func CollectInterfaces(faces []*model.Face, bcByFace map[string][]*model.BC) []InterfacePair {
	faceByID := make(map[string]*model.Face, len(faces))
	for _, f := range faces {
		faceByID[f.ID] = f
	}
	sideOf := func(f *model.Face) *model.BC {
		for _, bc := range bcByFace[f.ID] {
			if _, ok := SideForType(bc.Type); ok {
				return bc
			}
		}
		return nil
	}

	var pairs []InterfacePair
	paired := map[string]bool{}
	for _, f := range faces {
		if f.Kind != model.KindInterface || f.Status == model.FaceDuplicate || paired[f.ID] {
			continue
		}
		// 对侧面：同区域的 ID/名称匹配；找不到则回退到任意区域的名称匹配
		var nb *model.Face
		if f.NeighborRegion != "" {
			for _, g := range faces {
				if g.RegionID == f.NeighborRegion && (g.ID == f.NeighborFace || g.Name == f.NeighborFace) {
					nb = g
					break
				}
			}
		}
		if nb == nil {
			nb = faceByID[f.NeighborFace]
		}
		if nb != nil {
			paired[nb.ID] = true
		}

		self := sideOf(f)
		var other *model.BC
		if nb != nil {
			other = sideOf(nb)
		}
		pair := InterfacePair{Face: f}
		if self != nil && isSideA(self.Type) {
			pair.SideA, pair.SideB = self, other
		} else {
			pair.SideA, pair.SideB = other, self
		}
		pairs = append(pairs, pair)
	}
	return pairs
}

// UnitMismatch 检查同一耦合面两侧单位是否一致。
func UnitMismatch(a, b *model.BC) bool {
	if a == nil || b == nil {
		return false
	}
	return a.Unit != b.Unit
}
