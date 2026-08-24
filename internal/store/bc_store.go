package store

import (
	"database/sql"

	"task189-cfdbc/internal/model"
)

// BCStore 边界条件持久化。
type BCStore struct{ db *DB }

// NewBCStore 构造条件存储。
func NewBCStore(d *DB) *BCStore { return &BCStore{db: d} }

// Insert 写入条件（版本从 1 开始）。
func (s *BCStore) Insert(bc *model.BC) error {
	_, err := s.db.conn.Exec(
		`INSERT INTO boundary_conditions (id,face_id,region_id,type,unit,value,secondary,status,version,created_at,updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		bc.ID, bc.FaceID, bc.RegionID, bc.Type, bc.Unit, bc.Value, bc.Secondary,
		bc.Status, bc.Version, bc.CreatedAt, bc.UpdatedAt)
	return err
}

// Get 按 ID 读取条件。
func (s *BCStore) Get(id string) (*model.BC, error) {
	row := s.db.conn.QueryRow(
		`SELECT id,face_id,region_id,type,unit,value,secondary,status,version,created_at,updated_at
		 FROM boundary_conditions WHERE id=?`, id)
	var bc model.BC
	if err := row.Scan(&bc.ID, &bc.FaceID, &bc.RegionID, &bc.Type, &bc.Unit, &bc.Value,
		&bc.Secondary, &bc.Status, &bc.Version, &bc.CreatedAt, &bc.UpdatedAt); err != nil {
		return nil, err
	}
	return &bc, nil
}

// ListByFace 按面列出条件。
func (s *BCStore) ListByFace(faceID string) ([]*model.BC, error) {
	rows, err := s.db.conn.Query(
		`SELECT id,face_id,region_id,type,unit,value,secondary,status,version,created_at,updated_at
		 FROM boundary_conditions WHERE face_id=? ORDER BY created_at`, faceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBCs(rows)
}

// ListByRegion 按区域列出条件。
func (s *BCStore) ListByRegion(regionID string) ([]*model.BC, error) {
	rows, err := s.db.conn.Query(
		`SELECT id,face_id,region_id,type,unit,value,secondary,status,version,created_at,updated_at
		 FROM boundary_conditions WHERE region_id=? ORDER BY created_at`, regionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBCs(rows)
}

// ListAll 列出全部条件。
func (s *BCStore) ListAll() ([]*model.BC, error) {
	rows, err := s.db.conn.Query(
		`SELECT id,face_id,region_id,type,unit,value,secondary,status,version,created_at,updated_at
		 FROM boundary_conditions ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBCs(rows)
}

// UpdateStatusAndVersion 更新条件状态与版本（乐观锁：仅当版本匹配）。
func (s *BCStore) UpdateStatusAndVersion(id string, status model.BCStatus, expectVersion int) (int, error) {
	res, err := s.db.conn.Exec(
		`UPDATE boundary_conditions SET status=?, version=version+1, updated_at=? WHERE id=? AND version=?`,
		status, now(), id, expectVersion)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return 0, model.NewConflict("condition %s version changed by concurrent write", id)
	}
	return expectVersion + 1, nil
}

// UpdateValue 更新条件数值（乐观锁）。
func (s *BCStore) UpdateValue(id string, value, secondary float64, expectVersion int) (int, error) {
	return s.UpdateValueAndStatus(id, value, secondary, expectVersion, "")
}

// UpdateValueAndStatus updates the numeric payload and, when status is
// non-empty, the derived status in one optimistic-lock operation.
func (s *BCStore) UpdateValueAndStatus(id string, value, secondary float64, expectVersion int, status model.BCStatus) (int, error) {
	query := `UPDATE boundary_conditions SET value=?, secondary=?, updated_at=?, version=version+1`
	args := []any{value, secondary, now()}
	if status != "" {
		query += `, status=?`
		args = append(args, status)
	}
	query += ` WHERE id=? AND version=?`
	args = append(args, id, expectVersion)
	res, err := s.db.conn.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return 0, model.NewConflict("condition %s version changed by concurrent write", id)
	}
	return expectVersion + 1, nil
}

// CountByType 按类型统计条件数量。
func (s *BCStore) CountByType() (map[model.BCType]int, error) {
	rows, err := s.db.conn.Query(`SELECT type, COUNT(*) FROM boundary_conditions GROUP BY type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[model.BCType]int{}
	for rows.Next() {
		var t model.BCType
		var n int
		if err := rows.Scan(&t, &n); err != nil {
			return nil, err
		}
		out[t] = n
	}
	return out, rows.Err()
}

func scanBCs(rows *sql.Rows) ([]*model.BC, error) {
	var out []*model.BC
	for rows.Next() {
		var bc model.BC
		if err := rows.Scan(&bc.ID, &bc.FaceID, &bc.RegionID, &bc.Type, &bc.Unit, &bc.Value,
			&bc.Secondary, &bc.Status, &bc.Version, &bc.CreatedAt, &bc.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &bc)
	}
	return out, rows.Err()
}
