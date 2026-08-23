package store

import (
	"database/sql"

	"task189-cfdbc/internal/model"
)

// PackageStore 求解前置包持久化。
type PackageStore struct{ db *DB }

// NewPackageStore 构造包存储。
func NewPackageStore(d *DB) *PackageStore { return &PackageStore{db: d} }

// Insert 写入包（默认 building 状态）。
func (s *PackageStore) Insert(p *model.SolverPackage) error {
	_, err := s.db.conn.Exec(
		`INSERT INTO solver_packages (id,name,config_version,region_hash,conditions_version,model_id,snapshot,status,created_at,published_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		p.ID, p.Name, p.ConfigVersion, p.RegionHash, p.ConditionsVersion, p.ModelID,
		p.Snapshot, p.Status, p.CreatedAt, nullStr(p.PublishedAt))
	return err
}

// Get 按 ID 读取包。
func (s *PackageStore) Get(id string) (*model.SolverPackage, error) {
	row := s.db.conn.QueryRow(
		`SELECT id,name,config_version,region_hash,conditions_version,model_id,snapshot,status,created_at,published_at
		 FROM solver_packages WHERE id=?`, id)
	var p model.SolverPackage
	var pub sql.NullString
	if err := row.Scan(&p.ID, &p.Name, &p.ConfigVersion, &p.RegionHash, &p.ConditionsVersion,
		&p.ModelID, &p.Snapshot, &p.Status, &p.CreatedAt, &pub); err != nil {
		return nil, err
	}
	p.PublishedAt = pub.String
	return &p, nil
}

// List 列出全部包（倒序）。
func (s *PackageStore) List() ([]*model.SolverPackage, error) {
	rows, err := s.db.conn.Query(
		`SELECT id,name,config_version,region_hash,conditions_version,model_id,snapshot,status,created_at,published_at
		 FROM solver_packages ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.SolverPackage
	for rows.Next() {
		var p model.SolverPackage
		var pub sql.NullString
		if err := rows.Scan(&p.ID, &p.Name, &p.ConfigVersion, &p.RegionHash, &p.ConditionsVersion,
			&p.ModelID, &p.Snapshot, &p.Status, &p.CreatedAt, &pub); err != nil {
			return nil, err
		}
		p.PublishedAt = pub.String
		out = append(out, &p)
	}
	return out, rows.Err()
}

// UpdateStatus 更新包状态。
func (s *PackageStore) UpdateStatus(id string, status model.PackageStatus) error {
	var pub any
	if status == model.PkgPublished {
		pub = now()
	}
	_, err := s.db.conn.Exec(
		`UPDATE solver_packages SET status=?, published_at=COALESCE(?,published_at) WHERE id=?`,
		status, pub, id)
	return err
}

// LatestForRegion 返回指定区域哈希最新创建的包（用于派生）。
func (s *PackageStore) LatestForRegion(regionHash string) (*model.SolverPackage, error) {
	row := s.db.conn.QueryRow(
		`SELECT id,name,config_version,region_hash,conditions_version,model_id,snapshot,status,created_at,published_at
		 FROM solver_packages WHERE region_hash=? ORDER BY created_at DESC LIMIT 1`, regionHash)
	var p model.SolverPackage
	var pub sql.NullString
	if err := row.Scan(&p.ID, &p.Name, &p.ConfigVersion, &p.RegionHash, &p.ConditionsVersion,
		&p.ModelID, &p.Snapshot, &p.Status, &p.CreatedAt, &pub); err != nil {
		return nil, err
	}
	p.PublishedAt = pub.String
	return &p, nil
}

// CountByStatus 按状态统计包数量。
func (s *PackageStore) CountByStatus() (map[model.PackageStatus]int, error) {
	rows, err := s.db.conn.Query(`SELECT status, COUNT(*) FROM solver_packages GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[model.PackageStatus]int{}
	for rows.Next() {
		var st model.PackageStatus
		var n int
		if err := rows.Scan(&st, &n); err != nil {
			return nil, err
		}
		out[st] = n
	}
	return out, rows.Err()
}
