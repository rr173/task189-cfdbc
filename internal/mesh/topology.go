// Package mesh 管理网格区域与面的登记、分类、拓扑邻接、
// 连通性检测与拓扑哈希。本包只做纯逻辑，不依赖存储。
package mesh

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"task189-cfdbc/internal/model"
)

// FaceInput 面导入输入。
type FaceInput struct {
	Name      string  `json:"name"`
	Kind      string  `json:"kind"` // outer / interface
	Area      float64 `json:"area"`
	NormalX   float64 `json:"normal_x"`
	NormalY   float64 `json:"normal_y"`
	NormalZ   float64 `json:"normal_z"`
	NodeCount int     `json:"node_count"`
	// 耦合面：对侧区域与面对侧 ID
	NeighborRegion string `json:"neighbor_region,omitempty"`
	NeighborFace   string `json:"neighbor_face,omitempty"`
}

// RegionInput 区域登记输入。
type RegionInput struct {
	Name        string `json:"name"`
	Dimension   int    `json:"dimension"`
	CellCount   int    `json:"cell_count"`
	Description string `json:"description,omitempty"`
}

// ApplyFaceStatus 根据分类结果派生面的状态。
// 分类规则：外露面 → exposed；耦合面 → coupled；
// 面积非法或节点数过少 → degenerate；与已有面同指纹 → duplicate。
func ApplyFaceStatus(f *model.Face) {
	switch {
	case f.NodeCount < 3 || f.Area <= 0:
		f.Status = model.FaceDegenerate
		if f.NodeCount < 3 {
			f.DegenerateCause = "node_count_lt_3"
		} else {
			f.DegenerateCause = "non_positive_area"
		}
	case f.Kind == model.KindInterface:
		f.Status = model.FaceCoupled
	default:
		f.Status = model.FaceExposed
	}
}

// FaceFingerprint 计算面的拓扑指纹（几何特征：面积、法向、节点数），
// 用于重复面检测。名称不参与——几何相同即视为重复面。
func FaceFingerprint(f *model.Face) string {
	return fingerprint(f.Area, f.NormalX, f.NormalY, f.NormalZ, f.NodeCount)
}

func fingerprint(parts ...any) string {
	var sb strings.Builder
	for _, p := range parts {
		fmt.Fprintf(&sb, "%v|", p)
	}
	sum := sha256.Sum256([]byte(sb.String()))
	return hex.EncodeToString(sum[:12])
}

// DetectDuplicates 在给定区域内标记重复面：同指纹仅保留第一个，
// 其余标记为 duplicate 并记录 duplicate_of。
func DetectDuplicates(faces []*model.Face) []string {
	seen := make(map[string]string, len(faces))
	var changed []string
	for _, f := range faces {
		fp := FaceFingerprint(f)
		if orig, ok := seen[fp]; ok {
			f.Status = model.FaceDuplicate
			f.DuplicateOf = orig
			changed = append(changed, f.ID)
			continue
		}
		seen[fp] = f.ID
	}
	return changed
}

// HashRegion 计算区域拓扑哈希：对面集合按名称排序后取指纹串再哈希。
// 相同拓扑幂等——相同面集合导入得到相同哈希。
func HashRegion(regionID string, faces []*model.Face) string {
	parts := make([]string, 0, len(faces))
	for _, f := range faces {
		parts = append(parts, f.Name)
	}
	sort.Strings(parts)
	return fingerprint(regionID, strings.Join(parts, ","), len(faces))
}

// CanSeal reports whether an imported topology has reached a terminally
// reviewable state.
func CanSeal(status model.RegionStatus, faceCount int) bool {
	return faceCount > 0 && (status == model.RegionConnected || status == model.RegionIsolated)
}

// Classify 执行一次完整分类：先按几何退化，再按角色，
// 最后做重复面检测，返回被改变状态的面 ID 列表。
func Classify(faces []*model.Face) []string {
	var changed []string
	for _, f := range faces {
		before := f.Status
		ApplyFaceStatus(f)
		if before != f.Status {
			changed = append(changed, f.ID)
		}
	}
	changed = append(changed, DetectDuplicates(faces)...)
	return changed
}

// Connectivity 连通性检测结果。
type Connectivity struct {
	RegionID    string
	Connected   bool
	OrphanFaces []string // 无邻接的耦合面
	Reason      string
}

// CheckConnectivity 检查区域连通性：耦合面必须存在对侧区域，
// 外露面不得声称有对侧。返回是否连通及孤立原因。
func CheckConnectivity(r *model.Region, faces []*model.Face) Connectivity {
	c := Connectivity{RegionID: r.ID, Connected: true}
	for _, f := range faces {
		switch f.Kind {
		case model.KindInterface:
			if f.NeighborRegion == "" || f.NeighborFace == "" {
				c.Connected = false
				c.OrphanFaces = append(c.OrphanFaces, f.ID)
			}
		default:
			if f.NeighborRegion != "" || f.NeighborFace != "" {
				c.Connected = false
				c.OrphanFaces = append(c.OrphanFaces, f.ID)
				c.Reason = "outer face declares neighbor"
			}
		}
	}
	if !c.Connected && c.Reason == "" {
		c.Reason = "interface face missing neighbor"
	}
	return c
}

// RegionSnapshot 区域拓扑摘要（用于前置包快照）。
type RegionSnapshot struct {
	Region model.Region `json:"region"`
	Faces  []model.Face `json:"faces"`
}

// BuildRegionSnapshot 构造不可变区域快照。
func BuildRegionSnapshot(r *model.Region, faces []*model.Face) RegionSnapshot {
	copied := make([]model.Face, len(faces))
	for i, f := range faces {
		copied[i] = *f
	}
	return RegionSnapshot{Region: *r, Faces: copied}
}

// FacesByKind 按角色统计面数量。
func FacesByKind(faces []*model.Face) map[model.FaceKind]int {
	out := map[model.FaceKind]int{}
	for _, f := range faces {
		out[f.Kind]++
	}
	return out
}
