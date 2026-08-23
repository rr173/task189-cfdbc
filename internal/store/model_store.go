package store

import (
	"database/sql"

	"task189-cfdbc/internal/model"
)

// ModelStore 物理模型持久化。
type ModelStore struct{ db *DB }

// NewModelStore 构造模型存储。
func NewModelStore(d *DB) *ModelStore { return &ModelStore{db: d} }

// Insert 写入模型。
func (s *ModelStore) Insert(m *model.PhysicsModel) error {
	_, err := s.db.conn.Exec(
		`INSERT INTO physics_models (id,name,flow_type,viscous,reference_pressure_required,
			reference_pressure_face_id,default_unit,active,created_at)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		m.ID, m.Name, m.FlowType, boolInt(m.Viscous), boolInt(m.ReferencePressureRequired),
		nullStr(m.ReferencePressureFaceID), m.DefaultUnit, boolInt(m.Active), m.CreatedAt)
	return err
}

// Get 按 ID 读取模型。
func (s *ModelStore) Get(id string) (*model.PhysicsModel, error) {
	row := s.db.conn.QueryRow(
		`SELECT id,name,flow_type,viscous,reference_pressure_required,reference_pressure_face_id,default_unit,active,created_at
		 FROM physics_models WHERE id=?`, id)
	var m model.PhysicsModel
	var refFace sql.NullString
	var visc, refReq, act int
	if err := row.Scan(&m.ID, &m.Name, &m.FlowType, &visc, &refReq, &refFace, &m.DefaultUnit, &act, &m.CreatedAt); err != nil {
		return nil, err
	}
	m.Viscous, m.ReferencePressureRequired, m.Active = visc != 0, refReq != 0, act != 0
	m.ReferencePressureFaceID = refFace.String
	return &m, nil
}

// List 列出全部模型。
func (s *ModelStore) List() ([]*model.PhysicsModel, error) {
	rows, err := s.db.conn.Query(
		`SELECT id,name,flow_type,viscous,reference_pressure_required,reference_pressure_face_id,default_unit,active,created_at
		 FROM physics_models ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.PhysicsModel
	for rows.Next() {
		var m model.PhysicsModel
		var refFace sql.NullString
		var visc, refReq, act int
		if err := rows.Scan(&m.ID, &m.Name, &m.FlowType, &visc, &refReq, &refFace, &m.DefaultUnit, &act, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.Viscous, m.ReferencePressureRequired, m.Active = visc != 0, refReq != 0, act != 0
		m.ReferencePressureFaceID = refFace.String
		out = append(out, &m)
	}
	return out, rows.Err()
}

// SetActive 设置激活模型（同一时刻仅一个激活）。
func (s *ModelStore) SetActive(id string, active bool) error {
	tx, err := s.db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if active {
		if _, err := tx.Exec(`UPDATE physics_models SET active=0`); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`UPDATE physics_models SET active=? WHERE id=?`, boolInt(active), id); err != nil {
		return err
	}
	return tx.Commit()
}

// GetActive 读取当前激活模型。
func (s *ModelStore) GetActive() (*model.PhysicsModel, error) {
	row := s.db.conn.QueryRow(
		`SELECT id,name,flow_type,viscous,reference_pressure_required,reference_pressure_face_id,default_unit,active,created_at
		 FROM physics_models WHERE active=1`)
	var m model.PhysicsModel
	var refFace sql.NullString
	var visc, refReq, act int
	if err := row.Scan(&m.ID, &m.Name, &m.FlowType, &visc, &refReq, &refFace, &m.DefaultUnit, &act, &m.CreatedAt); err != nil {
		return nil, err
	}
	m.Viscous, m.ReferencePressureRequired, m.Active = visc != 0, refReq != 0, act != 0
	m.ReferencePressureFaceID = refFace.String
	return &m, nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
