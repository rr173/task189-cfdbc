package store

import "fmt"

// migrate 建表迁移：若表不存在则创建。
func (d *DB) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS regions (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			dimension INTEGER NOT NULL,
			cell_count INTEGER NOT NULL,
			face_count INTEGER NOT NULL,
			mesh_hash TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			sealed_at TEXT,
			description TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS faces (
			id TEXT PRIMARY KEY,
			region_id TEXT NOT NULL,
			name TEXT NOT NULL,
			kind TEXT NOT NULL,
			status TEXT NOT NULL,
			area REAL NOT NULL,
			normal_x REAL NOT NULL,
			normal_y REAL NOT NULL,
			normal_z REAL NOT NULL,
			node_count INTEGER NOT NULL,
			neighbor_region TEXT,
			neighbor_face TEXT,
			duplicate_of TEXT,
			degenerate_cause TEXT,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_faces_region ON faces(region_id)`,
		`CREATE TABLE IF NOT EXISTS physics_models (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			flow_type TEXT NOT NULL,
			viscous INTEGER NOT NULL,
			reference_pressure_required INTEGER NOT NULL,
			reference_pressure_face_id TEXT,
			default_unit TEXT NOT NULL,
			active INTEGER NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS boundary_conditions (
			id TEXT PRIMARY KEY,
			face_id TEXT NOT NULL,
			region_id TEXT NOT NULL,
			type TEXT NOT NULL,
			unit TEXT NOT NULL,
			value REAL NOT NULL,
			secondary REAL NOT NULL,
			status TEXT NOT NULL,
			version INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_bc_face ON boundary_conditions(face_id)`,
		`CREATE TABLE IF NOT EXISTS validation_runs (
			id TEXT PRIMARY KEY,
			model_id TEXT NOT NULL,
			config_version INTEGER NOT NULL,
			result TEXT NOT NULL,
			error_count INTEGER NOT NULL,
			warning_count INTEGER NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS validation_issues (
			id TEXT PRIMARY KEY,
			run_id TEXT NOT NULL,
			config_version INTEGER NOT NULL,
			type TEXT NOT NULL,
			severity TEXT NOT NULL,
			region_id TEXT,
			face_id TEXT,
			message TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_issue_run ON validation_issues(run_id)`,
		`CREATE TABLE IF NOT EXISTS solver_packages (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			config_version INTEGER NOT NULL,
			region_hash TEXT NOT NULL,
			conditions_version INTEGER NOT NULL,
			model_id TEXT NOT NULL,
			snapshot TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			published_at TEXT
		)`,
	}
	for i, s := range stmts {
		if _, err := d.conn.Exec(s); err != nil {
			return fmt.Errorf("migrate stmt %d: %w", i, err)
		}
	}
	return nil
}
