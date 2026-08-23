// Package physics 解释物理模型：可压缩/不可压缩流动、粘性、
// 参考压力要求与单位制换算。本包只做纯逻辑。
package physics

import (
	"math"

	"task189-cfdbc/internal/model"
)

// ModelInput 物理模型创建输入。
type ModelInput struct {
	Name    string          `json:"name"`
	FlowType model.FlowType `json:"flow_type"`
	Viscous bool            `json:"viscous"`
	DefaultUnit model.UnitSystem `json:"default_unit"`
}

// NewModel 构造物理模型：不可压缩流动必须要求参考压力，否则不需要。
func NewModel(in ModelInput) (*model.PhysicsModel, error) {
	if in.Name == "" {
		return nil, model.NewInvalid("model name required")
	}
	if in.FlowType != model.FlowCompressible && in.FlowType != model.FlowIncompressible {
		return nil, model.NewInvalid("flow_type must be compressible or incompressible")
	}
	if in.DefaultUnit != model.UnitSI && in.DefaultUnit != model.UnitCGS {
		return nil, model.NewInvalid("default_unit must be si or cgs")
	}
	refRequired := in.FlowType == model.FlowIncompressible
	return &model.PhysicsModel{
		FlowType:                  in.FlowType,
		Name:                      in.Name,
		Viscous:                   in.Viscous,
		DefaultUnit:               in.DefaultUnit,
		ReferencePressureRequired: refRequired,
		Active:                    false,
	}, nil
}

// PressureFactor 单位制到帕斯卡的换算系数。
func PressureFactor(u model.UnitSystem) float64 {
	if u == model.UnitCGS {
		return 0.1 // 1 dyn/cm^2 = 0.1 Pa
	}
	return 1.0
}

// MassFlowFactor 单位制到千克/秒的换算系数（CGS 为克/秒）。
func MassFlowFactor(u model.UnitSystem) float64 {
	if u == model.UnitCGS {
		return 0.001
	}
	return 1.0
}

// VelocityFactor 单位制到米/秒的换算系数（CGS 为厘米/秒）。
func VelocityFactor(u model.UnitSystem) float64 {
	if u == model.UnitCGS {
		return 0.01
	}
	return 1.0
}

// ToSIValue 把给定单位下的数值换算为 SI 基准值。
// kind 取 pressure / mass_flow / velocity。
func ToSIValue(value float64, unit model.UnitSystem, kind string) float64 {
	switch kind {
	case "pressure":
		return value * PressureFactor(unit)
	case "mass_flow":
		return value * MassFlowFactor(unit)
	case "velocity":
		return value * VelocityFactor(unit)
	default:
		return value
	}
}

// ReferencePressureFulfilled 判断参考压力要求是否满足：
// 模型要求参考压力时，必须存在一条已批准且类型为 reference_pressure 的条件。
func ReferencePressureFulfilled(m *model.PhysicsModel, hasApprovedRef bool) bool {
	if !m.ReferencePressureRequired {
		return true
	}
	return hasApprovedRef
}

// ConservativeBCType 判断条件类型是否参与守恒总量计算。
// 入口/出口类参与，壁面/对称/参考压力不参与。
func ConservativeBCType(t model.BCType) bool {
	switch t {
	case model.BCInletVelocity, model.BCInletMassFlow, model.BCInletTotalPress,
		model.BCOutletPressure, model.BCOutletFree:
		return true
	default:
		return false
	}
}

// InflowType 判断是否为入口条件。
func InflowType(t model.BCType) bool {
	switch t {
	case model.BCInletVelocity, model.BCInletMassFlow, model.BCInletTotalPress:
		return true
	}
	return false
}

// OutflowType 判断是否为出口条件。
func OutflowType(t model.BCType) bool {
	switch t {
	case model.BCOutletPressure, model.BCOutletMassFlow, model.BCOutletFree:
		return true
	}
	return false
}

// CompatibleUnits 判断两个单位制是否一致（一致才可守恒比较）。
func CompatibleUnits(a, b model.UnitSystem) bool { return a == b }

// BalanceTolerance 默认守恒容差（相对误差 1%）。
const BalanceTolerance = 0.01

// Imbalanced 判断入口与出口总量是否在容差内守恒。
// 返回相对误差；误差超过容差视为不平衡。
func Imbalanced(inflow, outflow, tolerance float64) (relErr float64, imbalanced bool) {
	denom := math.Max(math.Abs(inflow), math.Abs(outflow))
	if denom == 0 {
		return 0, inflow != outflow
	}
	relErr = math.Abs(inflow-outflow) / denom
	return relErr, relErr > tolerance
}

// IssueType 常量：校验问题类型标识。
const (
	IssueMissingBC            = "missing_bc"
	IssueConflictingBC        = "conflicting_bc"
	IssueUnconservedInterface = "unconserved_interface"
	IssueFlowImbalance        = "flow_imbalance"
	IssueMissingRefPressure   = "missing_reference_pressure"
	IssueDuplicateFace        = "duplicate_face"
	IssueDegenerateFace       = "degenerate_face"
	IssueUnitMismatch         = "unit_mismatch"
	IssueRegionIsolated       = "region_isolated"
)
