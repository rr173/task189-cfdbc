// Package audit 维护校验问题清单：校验运行的结果归类、
// 问题定位与严重度统计。
package audit

import (
	"sort"

	"task189-cfdbc/internal/model"
)

// Result 校验总体结论。
const (
	ResultSolvable         = "solvable"
	ResultUnderconstrained = "underconstrained"
	ResultOverconstrained  = "overconstrained"
)

// RunBuilder 在内存中累积一次校验运行的问题，最终落库。
type RunBuilder struct {
	ModelID       string
	ConfigVersion int
	issues        []*model.Issue
	nextID        int
	runID         string
}

// NewRunBuilder 创建问题清单构建器。
func NewRunBuilder(runID, modelID string, configVersion int) *RunBuilder {
	return &RunBuilder{runID: runID, ModelID: modelID, ConfigVersion: configVersion}
}

// Add 追加一条问题。
func (b *RunBuilder) Add(issueType string, sev model.Severity, regionID, faceID, msg string) {
	b.nextID++
	b.issues = append(b.issues, &model.Issue{
		ID:            b.runID + "-" + itoa(b.nextID),
		RunID:         b.runID,
		ConfigVersion: b.ConfigVersion,
		Type:          issueType,
		Severity:      sev,
		RegionID:      regionID,
		FaceID:        faceID,
		Message:       msg,
	})
}

// Issues 返回已收集的问题（按严重度降序、类型排序）。
func (b *RunBuilder) Issues() []*model.Issue {
	out := make([]*model.Issue, len(b.issues))
	copy(out, b.issues)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Severity != out[j].Severity {
			return sevRank(out[i].Severity) < sevRank(out[j].Severity)
		}
		return out[i].Type < out[j].Type
	})
	return out
}

// Counts 统计错误与警告数量。
func (b *RunBuilder) Counts() (errors, warnings int) {
	for _, i := range b.issues {
		if i.Severity == model.SeverityError {
			errors++
		} else {
			warnings++
		}
	}
	return
}

// Result 根据问题严重度归纳运行结论：
// 有 error → underconstrained 或 overconstrained（由调用方按类型细分）；
// 无 error 有 warning → overconstrained；
// 无问题 → solvable。
func Result(errors, warnings int) string {
	if errors > 0 {
		return ResultUnderconstrained
	}
	if warnings > 0 {
		return ResultOverconstrained
	}
	return ResultSolvable
}

// Classify 按问题类型归纳结论：
// 缺失类（missing_bc / missing_reference_pressure / region_isolated）→ underconstrained；
// 冲突/失衡类（conflicting_bc / unconserved_interface / flow_imbalance /
// duplicate_face / degenerate_face / unit_mismatch）→ overconstrained。
func Classify(issues []*model.Issue) string {
	hasError := false
	for _, i := range issues {
		if i.Severity != model.SeverityError {
			continue
		}
		hasError = true
		switch i.Type {
		case "missing_bc", "missing_reference_pressure", "region_isolated":
			return ResultUnderconstrained
		}
	}
	if hasError {
		return ResultOverconstrained
	}
	return ResultSolvable
}

// Publishable reports whether a validation result is safe to hand to a
// solver. Only a clean run can become a published package baseline.
func Publishable(result string) bool { return result == ResultSolvable }

func sevRank(s model.Severity) int {
	if s == model.SeverityError {
		return 0
	}
	return 1
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [8]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
