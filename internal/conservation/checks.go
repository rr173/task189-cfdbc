// Package conservation 校验守恒与可解性：耦合面双侧守恒、
// 入口/出口总量约束、参考压力可解性。本包只做纯逻辑。
package conservation

import (
	"task189-cfdbc/internal/model"
	"task189-cfdbc/internal/physics"
)

// InterfaceResult 单个耦合面的守恒结论。
type InterfaceResult struct {
	FaceID    string
	Conserved bool
	UnitOK    bool
	SideASpec float64
	SideBSpec float64
	Reason    string
}

// CheckInterface 校验耦合面双侧守恒：两侧必须都存在适用条件、
// 单位一致、且 A/B 侧的数值规格（换算到 SI）一致。
// tolerance 为相对容差。
func CheckInterface(pair interfacePair, tolerance float64) InterfaceResult {
	r := InterfaceResult{FaceID: pair.face.ID, Conserved: true}
	if pair.sideA == nil || pair.sideB == nil {
		r.Conserved = false
		r.Reason = "one side missing condition"
		return r
	}
	if physics.CompatibleUnits(pair.sideA.Unit, pair.sideB.Unit) {
		r.UnitOK = true
	} else {
		r.UnitOK = false
		r.Conserved = false
		r.Reason = "unit mismatch"
		return r
	}
	a := physics.ToSIValue(pair.sideA.Value, pair.sideA.Unit, "velocity")
	b := physics.ToSIValue(pair.sideB.Value, pair.sideB.Unit, "velocity")
	r.SideASpec = a
	r.SideBSpec = b
	if !approxEqual(a, b, tolerance) {
		r.Conserved = false
		r.Reason = "side spec mismatch"
	}
	return r
}

// interfacePair 隐藏依赖：从 conditions.InterfacePair 适配。
// 为避免包间循环依赖，conservation 自行定义最小接口。
type interfacePair struct {
	face  *model.Face
	sideA *model.BC
	sideB *model.BC
}

// FromBCs 由两侧条件构造守恒检查输入。
func FromBCs(face *model.Face, sideA, sideB *model.BC) interfacePair {
	return interfacePair{face: face, sideA: sideA, sideB: sideB}
}

func approxEqual(a, b, tol float64) bool {
	denom := max(abs(a), abs(b))
	if denom == 0 {
		return a == b
	}
	return abs(a-b)/denom <= tol
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// FlowBalance 入口/出口总量校验结果。
type FlowBalance struct {
	Inflow   float64
	Outflow  float64
	RelErr   float64
	Balanced bool
}

// CheckFlowBalance 汇总所有质量流量条件的换算后总量并判定守恒。
// 速度/压力条件量纲不同不可直接相加，由存在性与可解性检查覆盖；
// 总量守恒仅针对入口/出口质量流量（单位换算到 kg/s 后比较）。
func CheckFlowBalance(bcs []*model.BC, tolerance float64) FlowBalance {
	var inflow, outflow float64
	for _, bc := range bcs {
		switch bc.Type {
		case model.BCInletMassFlow:
			inflow += physics.ToSIValue(bc.Value, bc.Unit, "mass_flow")
		case model.BCOutletMassFlow:
			outflow += physics.ToSIValue(bc.Value, bc.Unit, "mass_flow")
		}
	}
	relErr, imbalanced := physics.Imbalanced(inflow, outflow, tolerance)
	return FlowBalance{
		Inflow: inflow, Outflow: outflow,
		RelErr: relErr, Balanced: !imbalanced,
	}
}

// RefPressureCheck 参考压力可解性：不可压缩模型且存在多个压力出口时
// 必须配置参考压力，否则压力场不可唯一确定。
type RefPressureCheck struct {
	Required  bool
	Fulfilled bool
	Missing   bool
}

// CheckReferencePressure 判断参考压力是否满足。
// 模型要求（不可压缩）时必须有已批准的 reference_pressure 条件。
func CheckReferencePressure(m *model.PhysicsModel, hasApprovedRef bool) RefPressureCheck {
	required := m.ReferencePressureRequired
	if !required {
		return RefPressureCheck{Required: false, Fulfilled: true}
	}
	return RefPressureCheck{
		Required:  true,
		Fulfilled: hasApprovedRef,
		Missing:   !hasApprovedRef,
	}
}
