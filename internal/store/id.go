package store

import (
	"strconv"
	"strings"
)

// IDGen 生成简单递增 ID 前缀。
type IDGen struct {
	prefix string
	seq    int
}

// NewIDGen 构造 ID 生成器。
func NewIDGen(prefix string) *IDGen { return &IDGen{prefix: prefix} }

// Next 生成下一个 ID：prefix-<seq>。
func (g *IDGen) Next() string {
	g.seq++
	return g.prefix + "-" + strconv.Itoa(g.seq)
}

// NextConfigVersion 计算下一次条件配置版本：取已有条件最大版本 +1。
// 空库返回 1。
func (d *DB) NextConfigVersion() (int, error) {
	var maxV int
	if err := d.conn.QueryRow(`SELECT COALESCE(MAX(version),0) FROM boundary_conditions`).Scan(&maxV); err != nil {
		return 0, err
	}
	return maxV + 1, nil
}

// NextConditionsVersion 计算下一次条件集合版本：取全部条件数量 / 每面最大版本。
// 用于前置包绑定条件版本。
func (d *DB) NextConditionsVersion() (int, error) {
	var maxV int
	if err := d.conn.QueryRow(`SELECT COALESCE(MAX(version),0) FROM boundary_conditions`).Scan(&maxV); err != nil {
		return 0, err
	}
	return maxV, nil
}

// SanitizeID 清洗 ID 输入：仅允许字母数字与 -_。
func SanitizeID(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
