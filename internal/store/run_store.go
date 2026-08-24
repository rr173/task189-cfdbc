package store

import (
	"database/sql"

	"task189-cfdbc/internal/model"
)

// RunStore 校验运行与问题持久化。
type RunStore struct{ db *DB }

// NewRunStore 构造运行存储。
func NewRunStore(d *DB) *RunStore { return &RunStore{db: d} }

// InsertRun 写入校验运行。
func (s *RunStore) InsertRun(r *model.ValidationRun) error {
	_, err := s.db.conn.Exec(
		`INSERT INTO validation_runs (id,model_id,config_version,result,error_count,warning_count,created_at)
		 VALUES (?,?,?,?,?,?,?)`,
		r.ID, r.ModelID, r.ConfigVersion, r.Result, r.ErrorCount, r.WarningCount, r.CreatedAt)
	return err
}

// GetRun 读取校验运行。
func (s *RunStore) GetRun(id string) (*model.ValidationRun, error) {
	row := s.db.conn.QueryRow(
		`SELECT id,model_id,config_version,result,error_count,warning_count,created_at
		 FROM validation_runs WHERE id=?`, id)
	var r model.ValidationRun
	if err := row.Scan(&r.ID, &r.ModelID, &r.ConfigVersion, &r.Result, &r.ErrorCount, &r.WarningCount, &r.CreatedAt); err != nil {
		return nil, err
	}
	return &r, nil
}

// ListRuns 列出校验运行（倒序）。
func (s *RunStore) ListRuns(limit int) ([]*model.ValidationRun, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.conn.Query(
		`SELECT id,model_id,config_version,result,error_count,warning_count,created_at
		 FROM validation_runs ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.ValidationRun
	for rows.Next() {
		var r model.ValidationRun
		if err := rows.Scan(&r.ID, &r.ModelID, &r.ConfigVersion, &r.Result, &r.ErrorCount, &r.WarningCount, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}

// LatestForModel returns the most recent persisted validation for a model.
// When no run exists it returns an error wrapping sql.ErrNoRows so callers
// can distinguish "no validation yet" from a genuine lookup failure.
func (s *RunStore) LatestForModel(modelID string) (*model.ValidationRun, error) {
	row := s.db.conn.QueryRow(
		`SELECT id,model_id,config_version,result,error_count,warning_count,created_at
		 FROM validation_runs WHERE model_id=? ORDER BY created_at DESC, rowid DESC LIMIT 1`, modelID)
	var r model.ValidationRun
	if err := row.Scan(&r.ID, &r.ModelID, &r.ConfigVersion, &r.Result, &r.ErrorCount, &r.WarningCount, &r.CreatedAt); err != nil {
		return nil, err
	}
	return &r, nil
}

// InsertIssue 写入一条问题。
func (s *RunStore) InsertIssue(i *model.Issue) error {
	_, err := s.db.conn.Exec(
		`INSERT INTO validation_issues (id,run_id,config_version,type,severity,region_id,face_id,message,created_at)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		i.ID, i.RunID, i.ConfigVersion, i.Type, i.Severity,
		nullStr(i.RegionID), nullStr(i.FaceID), i.Message, i.CreatedAt)
	return err
}

// ListIssuesByRun 列出一次运行的问题。
func (s *RunStore) ListIssuesByRun(runID string) ([]*model.Issue, error) {
	rows, err := s.db.conn.Query(
		`SELECT id,run_id,config_version,type,severity,region_id,face_id,message,created_at
		 FROM validation_issues WHERE run_id=? ORDER BY severity, type`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIssues(rows)
}

// ListIssues 列出最近问题（可选按配置版本过滤）。
func (s *RunStore) ListIssues(configVersion int, limit int) ([]*model.Issue, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := `SELECT id,run_id,config_version,type,severity,region_id,face_id,message,created_at
		FROM validation_issues`
	args := []any{}
	if configVersion > 0 {
		query += ` WHERE config_version=?`
		args = append(args, configVersion)
	}
	query += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIssues(rows)
}

// GetIssue 读取单条问题。
func (s *RunStore) GetIssue(id string) (*model.Issue, error) {
	row := s.db.conn.QueryRow(
		`SELECT id,run_id,config_version,type,severity,region_id,face_id,message,created_at
		 FROM validation_issues WHERE id=?`, id)
	return scanIssue(row)
}

func scanIssues(rows *sql.Rows) ([]*model.Issue, error) {
	var out []*model.Issue
	for rows.Next() {
		var i model.Issue
		var regionID, faceID sql.NullString
		if err := rows.Scan(&i.ID, &i.RunID, &i.ConfigVersion, &i.Type, &i.Severity,
			&regionID, &faceID, &i.Message, &i.CreatedAt); err != nil {
			return nil, err
		}
		i.RegionID, i.FaceID = regionID.String, faceID.String
		out = append(out, &i)
	}
	return out, rows.Err()
}

func scanIssue(row *sql.Row) (*model.Issue, error) {
	var i model.Issue
	var regionID, faceID sql.NullString
	if err := row.Scan(&i.ID, &i.RunID, &i.ConfigVersion, &i.Type, &i.Severity,
		&regionID, &faceID, &i.Message, &i.CreatedAt); err != nil {
		return nil, err
	}
	i.RegionID, i.FaceID = regionID.String, faceID.String
	return &i, nil
}
