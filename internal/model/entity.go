// Package model 定义计算流体网格边界条件一致性服务的领域实体、
// 状态机枚举与共享错误。所有业务包依赖本包，本包不依赖其它内部包。
package model

import (
	"fmt"
	"math"
)

// RegionStatus 网格区域状态机：
// imported（待导入）→ connected（连通）/ isolated（孤立）→ sealed（已封存）。
type RegionStatus string

const (
	RegionImported  RegionStatus = "imported"
	RegionConnected RegionStatus = "connected"
	RegionIsolated  RegionStatus = "isolated"
	RegionSealed    RegionStatus = "sealed"
)

// IsKnownBCType reports whether a boundary condition type is part of the
// supported solver vocabulary.
func IsKnownBCType(t BCType) bool {
	switch t {
	case BCInletVelocity, BCInletMassFlow, BCInletTotalPress,
		BCOutletPressure, BCOutletMassFlow, BCOutletFree,
		BCWallNoSlip, BCWallSlip, BCSymmetry,
		BCInterfaceSideA, BCInterfaceSideB, BCReferencePressure:
		return true
	default:
		return false
	}
}

// IsFiniteBoundaryValue rejects NaN and both infinities before a condition
// enters the persisted configuration.
func IsFiniteBoundaryValue(v float64) bool {
	return !math.IsNaN(v)
}

// FaceStatus 面状态机：
// unclassified（未分类）→ exposed（外露）/ coupled（耦合）/ duplicate（重复）/ degenerate（退化）。
type FaceStatus string

const (
	FaceUnclassified FaceStatus = "unclassified"
	FaceExposed      FaceStatus = "exposed"
	FaceCoupled      FaceStatus = "coupled"
	FaceDuplicate    FaceStatus = "duplicate"
	FaceDegenerate   FaceStatus = "degenerate"
)

// FaceKind 面的物理角色，决定可配的边界条件类型。
type FaceKind string

const (
	KindOuter     FaceKind = "outer"     // 外露面：入口/出口/壁面/对称面
	KindInterface FaceKind = "interface" // 耦合面：需要双侧守恒
)

// BCStatus 边界条件状态机：
// draft（草案）→ applicable（适用）/ conflicting（冲突）/ missing（缺失）→ approved（已批准）。
type BCStatus string

const (
	BCStatusDraft       BCStatus = "draft"
	BCStatusApplicable  BCStatus = "applicable"
	BCStatusConflicting BCStatus = "conflicting"
	BCStatusMissing     BCStatus = "missing"
	BCStatusApproved    BCStatus = "approved"
)

// PackageStatus 求解前置包状态机：
// building（构建中）→ solvable（可求解）/ underconstrained（欠约束）/ overconstrained（过约束）→ published（已发布）。
type PackageStatus string

const (
	PkgBuilding         PackageStatus = "building"
	PkgSolvable         PackageStatus = "solvable"
	PkgUnderconstrained PackageStatus = "underconstrained"
	PkgOverconstrained  PackageStatus = "overconstrained"
	PkgPublished        PackageStatus = "published"
)

// FlowType 流动模型类型：可压缩 / 不可压缩。
type FlowType string

const (
	FlowCompressible   FlowType = "compressible"
	FlowIncompressible FlowType = "incompressible"
)

// BCType 边界条件类型。
type BCType string

const (
	BCInletVelocity     BCType = "inlet_velocity"     // 速度入口
	BCInletMassFlow     BCType = "inlet_mass_flow"    // 质量流量入口
	BCInletTotalPress   BCType = "inlet_total_press"  // 总压入口
	BCOutletPressure    BCType = "outlet_pressure"    // 压力出口
	BCOutletMassFlow    BCType = "outlet_mass_flow"   // 质量流量出口
	BCOutletFree        BCType = "outlet_free"        // 自由出口
	BCWallNoSlip        BCType = "wall_no_slip"       // 无滑移壁面
	BCWallSlip          BCType = "wall_slip"          // 滑移壁面
	BCSymmetry          BCType = "symmetry"           // 对称面
	BCInterfaceSideA    BCType = "interface_side_a"   // 耦合面 A 侧
	BCInterfaceSideB    BCType = "interface_side_b"   // 耦合面 B 侧
	BCReferencePressure BCType = "reference_pressure" // 参考压力
)

// UnitSystem 单位制。
type UnitSystem string

const (
	UnitSI  UnitSystem = "si"  // 米-千克-秒
	UnitCGS UnitSystem = "cgs" // 厘米-克-秒
)

// Severity 问题严重度。
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Region 网格区域摘要：导入时登记边界框、单元规模与网格哈希，
// 哈希用于相同网格幂等导入。
type Region struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Dimension   int          `json:"dimension"` // 2 或 3
	CellCount   int          `json:"cell_count"`
	FaceCount   int          `json:"face_count"`
	MeshHash    string       `json:"mesh_hash"` // 区域拓扑指纹
	Status      RegionStatus `json:"status"`
	CreatedAt   string       `json:"created_at"`
	UpdatedAt   string       `json:"updated_at"`
	SealedAt    string       `json:"sealed_at,omitempty"`
	Description string       `json:"description,omitempty"`
}

// Face 面摘要：所属区域、物理角色、形态状态、面积、法向与邻接关系。
// 耦合面记录对面区域与对面面 ID，用于双侧守恒检查。
type Face struct {
	ID              string     `json:"id"`
	RegionID        string     `json:"region_id"`
	Name            string     `json:"name"`
	Kind            FaceKind   `json:"kind"`
	Status          FaceStatus `json:"status"`
	Area            float64    `json:"area"`
	NormalX         float64    `json:"normal_x"`
	NormalY         float64    `json:"normal_y"`
	NormalZ         float64    `json:"normal_z"`
	NodeCount       int        `json:"node_count"`
	NeighborRegion  string     `json:"neighbor_region,omitempty"`  // 耦合面对侧区域
	NeighborFace    string     `json:"neighbor_face,omitempty"`    // 耦合面对侧面
	DuplicateOf     string     `json:"duplicate_of,omitempty"`     // 重复面对应原面
	DegenerateCause string     `json:"degenerate_cause,omitempty"` // 退化原因
	CreatedAt       string     `json:"created_at"`
}

// PhysicsModel 物理模型：决定校验规则（参考压力是否必须、粘性是否参与守恒）。
type PhysicsModel struct {
	ID                        string     `json:"id"`
	Name                      string     `json:"name"`
	FlowType                  FlowType   `json:"flow_type"`
	Viscous                   bool       `json:"viscous"`
	ReferencePressureRequired bool       `json:"reference_pressure_required"`
	ReferencePressureFaceID   string     `json:"reference_pressure_face_id,omitempty"`
	DefaultUnit               UnitSystem `json:"default_unit"`
	Active                    bool       `json:"active"`
	CreatedAt                 string     `json:"created_at"`
}

// BC 边界条件：绑定到面，携带类型、单位与参数（速度/压力/流量等）。
type BC struct {
	ID        string     `json:"id"`
	FaceID    string     `json:"face_id"`
	RegionID  string     `json:"region_id"`
	Type      BCType     `json:"type"`
	Unit      UnitSystem `json:"unit"`
	Value     float64    `json:"value"`     // 主要数值（速度/压力/质量流量）
	Secondary float64    `json:"secondary"` // 辅助数值（如总压下的温度）
	Status    BCStatus   `json:"status"`
	Version   int        `json:"version"` // 乐观锁版本，并发改写检测
	CreatedAt string     `json:"created_at"`
	UpdatedAt string     `json:"updated_at"`
}

// ValidationRun 一次一致性校验运行。
type ValidationRun struct {
	ID            string `json:"id"`
	ModelID       string `json:"model_id"`
	ConfigVersion int    `json:"config_version"` // 条件配置版本（每次条件变更 +1）
	Result        string `json:"result"`         // solvable / underconstrained / overconstrained
	ErrorCount    int    `json:"error_count"`
	WarningCount  int    `json:"warning_count"`
	CreatedAt     string `json:"created_at"`
}

// Issue 校验发现的问题定位。
type Issue struct {
	ID            string   `json:"id"`
	RunID         string   `json:"run_id"`
	ConfigVersion int      `json:"config_version"`
	Type          string   `json:"type"`
	Severity      Severity `json:"severity"`
	RegionID      string   `json:"region_id,omitempty"`
	FaceID        string   `json:"face_id,omitempty"`
	Message       string   `json:"message"`
	CreatedAt     string   `json:"created_at"`
}

// SolverPackage 求解前置包：发布后不可改写，修改配置须派生新包。
type SolverPackage struct {
	ID                string        `json:"id"`
	Name              string        `json:"name"`
	ConfigVersion     int           `json:"config_version"`
	RegionHash        string        `json:"region_hash"`
	ConditionsVersion int           `json:"conditions_version"`
	ModelID           string        `json:"model_id"`
	Snapshot          string        `json:"snapshot"` // JSON 快照
	Status            PackageStatus `json:"status"`
	CreatedAt         string        `json:"created_at"`
	PublishedAt       string        `json:"published_at,omitempty"`
}

// ErrNotFound 表示目标实体不存在。
type ErrNotFound struct{ Kind, ID string }

func (e *ErrNotFound) Error() string { return fmt.Sprintf("%s %s not found", e.Kind, e.ID) }

// StatusCode 映射为 HTTP 404。
func (e *ErrNotFound) StatusCode() int { return 404 }

// ErrConflict 表示状态机/幂等/版本冲突。
type ErrConflict struct{ Message string }

func (e *ErrConflict) Error() string { return fmt.Sprintf("conflict: %s", e.Message) }

// StatusCode 映射为 HTTP 409。
func (e *ErrConflict) StatusCode() int { return 409 }

// ErrInvalid 表示输入不合法。
type ErrInvalid struct{ Message string }

func (e *ErrInvalid) Error() string { return fmt.Sprintf("invalid: %s", e.Message) }

// StatusCode 映射为 HTTP 400。
func (e *ErrInvalid) StatusCode() int { return 400 }

// NewNotFound 构造实体不存在错误。
func NewNotFound(kind, id string) error { return &ErrNotFound{Kind: kind, ID: id} }

// NewConflict 构造冲突错误。
func NewConflict(format string, args ...any) error {
	return &ErrConflict{Message: fmt.Sprintf(format, args...)}
}

// NewInvalid 构造非法输入错误。
func NewInvalid(format string, args ...any) error {
	return &ErrInvalid{Message: fmt.Sprintf(format, args...)}
}
