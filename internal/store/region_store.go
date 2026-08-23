package store

import (
	"database/sql"

	"task189-cfdbc/internal/model"
)

// RegionStore 区域持久化。
type RegionStore struct{ db *DB }

// NewRegionStore 构造区域存储。
func NewRegionStore(d *DB) *RegionStore { return &RegionStore{db: d} }

// Insert 写入区域。
func (s *RegionStore) Insert(r *model.Region) error {
	_, err := s.db.conn.Exec(
		`INSERT INTO regions (id,name,dimension,cell_count,face_count,mesh_hash,status,created_at,updated_at,sealed_at,description)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		r.ID, r.Name, r.Dimension, r.CellCount, r.FaceCount, r.MeshHash, r.Status,
		r.CreatedAt, r.UpdatedAt, nullStr(r.SealedAt), r.Description)
	return err
}

// Get 按 ID 读取区域。
func (s *RegionStore) Get(id string) (*model.Region, error) {
	row := s.db.conn.QueryRow(
		`SELECT id,name,dimension,cell_count,face_count,mesh_hash,status,created_at,updated_at,sealed_at,description
		 FROM regions WHERE id=?`, id)
	var r model.Region
	var sealed sql.NullString
	if err := row.Scan(&r.ID, &r.Name, &r.Dimension, &r.CellCount, &r.FaceCount,
		&r.MeshHash, &r.Status, &r.CreatedAt, &r.UpdatedAt, &sealed, &r.Description); err != nil {
		return nil, err
	}
	r.SealedAt = sealed.String
	return &r, nil
}

// List 列出全部区域（按创建时间倒序）。
func (s *RegionStore) List() ([]*model.Region, error) {
	rows, err := s.db.conn.Query(
		`SELECT id,name,dimension,cell_count,face_count,mesh_hash,status,created_at,updated_at,sealed_at,description
		 FROM regions ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Region
	for rows.Next() {
		var r model.Region
		var sealed sql.NullString
		if err := rows.Scan(&r.ID, &r.Name, &r.Dimension, &r.CellCount, &r.FaceCount,
			&r.MeshHash, &r.Status, &r.CreatedAt, &r.UpdatedAt, &sealed, &r.Description); err != nil {
			return nil, err
		}
		r.SealedAt = sealed.String
		out = append(out, &r)
	}
	return out, rows.Err()
}

// UpdateStatus 更新状态与更新时间。
func (s *RegionStore) UpdateStatus(id string, status model.RegionStatus, extra ...string) error {
	sealedAt := ""
	if len(extra) > 0 {
		sealedAt = extra[0]
	}
	_, err := s.db.conn.Exec(
		`UPDATE regions SET status=?, updated_at=?, sealed_at=COALESCE(?,sealed_at) WHERE id=?`,
		status, now(), nullStr(sealedAt), id)
	return err
}

// UpdateFaceCount 更新面计数与拓扑哈希。
func (s *RegionStore) UpdateFaceCount(id string, faceCount int, hash string) error {
	_, err := s.db.conn.Exec(
		`UPDATE regions SET face_count=?, mesh_hash=?, updated_at=? WHERE id=?`,
		faceCount, hash, now(), id)
	return err
}

// CountByStatus 按状态统计区域数量。
func (s *RegionStore) CountByStatus() (map[model.RegionStatus]int, error) {
	rows, err := s.db.conn.Query(`SELECT status, COUNT(*) FROM regions GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[model.RegionStatus]int{}
	for rows.Next() {
		var st model.RegionStatus
		var n int
		if err := rows.Scan(&st, &n); err != nil {
			return nil, err
		}
		out[st] = n
	}
	return out, rows.Err()
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
