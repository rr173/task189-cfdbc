package store

import (
	"database/sql"

	"task189-cfdbc/internal/model"
)

// FaceStore 面持久化。
type FaceStore struct{ db *DB }

// NewFaceStore 构造面存储。
func NewFaceStore(d *DB) *FaceStore { return &FaceStore{db: d} }

// Insert 写入面。
func (s *FaceStore) Insert(f *model.Face) error {
	_, err := s.db.conn.Exec(
		`INSERT INTO faces (id,region_id,name,kind,status,area,normal_x,normal_y,normal_z,node_count,
			neighbor_region,neighbor_face,duplicate_of,degenerate_cause,created_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		f.ID, f.RegionID, f.Name, f.Kind, f.Status, f.Area, f.NormalX, f.NormalY, f.NormalZ, f.NodeCount,
		nullStr(f.NeighborRegion), nullStr(f.NeighborFace), nullStr(f.DuplicateOf), nullStr(f.DegenerateCause), f.CreatedAt)
	return err
}

// Get 按 ID 读取面。
func (s *FaceStore) Get(id string) (*model.Face, error) {
	row := s.db.conn.QueryRow(
		`SELECT id,region_id,name,kind,status,area,normal_x,normal_y,normal_z,node_count,
			neighbor_region,neighbor_face,duplicate_of,degenerate_cause,created_at
		 FROM faces WHERE id=?`, id)
	var f model.Face
	var nr, nf, dup, dg sql.NullString
	if err := row.Scan(&f.ID, &f.RegionID, &f.Name, &f.Kind, &f.Status, &f.Area,
		&f.NormalX, &f.NormalY, &f.NormalZ, &f.NodeCount, &nr, &nf, &dup, &dg, &f.CreatedAt); err != nil {
		return nil, err
	}
	f.NeighborRegion, f.NeighborFace, f.DuplicateOf, f.DegenerateCause = nr.String, nf.String, dup.String, dg.String
	return &f, nil
}

// ListByRegion 列出区域全部面。
func (s *FaceStore) ListByRegion(regionID string) ([]*model.Face, error) {
	rows, err := s.db.conn.Query(
		`SELECT id,region_id,name,kind,status,area,normal_x,normal_y,normal_z,node_count,
			neighbor_region,neighbor_face,duplicate_of,degenerate_cause,created_at
		 FROM faces WHERE region_id=? ORDER BY name`, regionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFaces(rows)
}

// ListAll 列出全部面。
func (s *FaceStore) ListAll() ([]*model.Face, error) {
	rows, err := s.db.conn.Query(
		`SELECT id,region_id,name,kind,status,area,normal_x,normal_y,normal_z,node_count,
			neighbor_region,neighbor_face,duplicate_of,degenerate_cause,created_at FROM faces`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFaces(rows)
}

// CountByRegion returns the persisted face count used by idempotent imports.
func (s *FaceStore) CountByRegion(regionID string) (int, error) {
	var count int
	if err := s.db.conn.QueryRow(`SELECT COUNT(*) FROM faces WHERE region_id=?`, regionID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// UpdateStatus 更新面状态。
func (s *FaceStore) UpdateStatus(id string, status model.FaceStatus, duplicateOf, degenerate string) error {
	_, err := s.db.conn.Exec(
		`UPDATE faces SET status=?, duplicate_of=COALESCE(?,duplicate_of), degenerate_cause=COALESCE(?,degenerate_cause) WHERE id=?`,
		status, nullStr(duplicateOf), nullStr(degenerate), id)
	return err
}

func scanFaces(rows *sql.Rows) ([]*model.Face, error) {
	var out []*model.Face
	for rows.Next() {
		var f model.Face
		var nr, nf, dup, dg sql.NullString
		if err := rows.Scan(&f.ID, &f.RegionID, &f.Name, &f.Kind, &f.Status, &f.Area,
			&f.NormalX, &f.NormalY, &f.NormalZ, &f.NodeCount, &nr, &nf, &dup, &dg, &f.CreatedAt); err != nil {
			return nil, err
		}
		f.NeighborRegion, f.NeighborFace, f.DuplicateOf, f.DegenerateCause = nr.String, nf.String, dup.String, dg.String
		out = append(out, &f)
	}
	return out, rows.Err()
}
