// Package store 提供 SQLite 持久化：建表迁移与各实体的读写。
// 使用 modernc.org/sqlite 纯 Go 驱动，CGO 无关，离线可构建。
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// DB 封装 SQLite 连接。
type DB struct {
	conn *sql.DB
	path string
}

// Open 打开（必要时创建）SQLite 数据库并执行迁移。
func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return nil, fmt.Errorf("mkdir: %w", err)
	}
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	conn.SetMaxOpenConns(4) // allow concurrent callers
	db := &DB{conn: conn, path: path}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, err
	}
	return db, nil
}

// Close 关闭连接。
func (d *DB) Close() error { return d.conn.Close() }

// Path 返回数据库文件路径。
func (d *DB) Path() string { return d.path }

// Ping 检查连接健康。
func (d *DB) Ping() error { return d.conn.Ping() }

// ErrNoRows 空结果。
var ErrNoRows = sql.ErrNoRows

// IsNoRows 判断是否空结果错误。
func IsNoRows(err error) bool { return errors.Is(err, sql.ErrNoRows) }

func now() string { return time.Now().UTC().Format(time.RFC3339) }
