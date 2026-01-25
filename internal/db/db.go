package db

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

//go:embed migrations/003_project_groups.sql
var migration003SQL string

//go:embed migrations/004_nullable_git_root.sql
var migration004SQL string

// DB wraps the SQLite database connection
type DB struct {
	conn *sql.DB
}

// NewDB creates a new database connection and initializes the schema
func NewDB(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// SQLiteのパフォーマンス設定
	// 1接続推奨（SQLiteは並行書き込みに弱い）
	conn.SetMaxOpenConns(1)

	// 外部キー制約を有効化
	_, err = conn.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	db := &DB{conn: conn}

	// スキーマの適用
	if err := db.Migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// Migrate applies the database schema and migrations
func (db *DB) Migrate() error {
	// 基本スキーマを適用
	_, err := db.conn.Exec(schemaSQL)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	// マイグレーションを適用
	err = db.runMigrations()
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// runMigrations executes all pending migrations
func (db *DB) runMigrations() error {
	// マイグレーションテーブルを作成
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// マイグレーション003を実行
	err = db.applyMigration("003", migration003SQL)
	if err != nil {
		return fmt.Errorf("failed to apply migration 003: %w", err)
	}

	// マイグレーション004を実行
	err = db.applyMigration("004", migration004SQL)
	if err != nil {
		return fmt.Errorf("failed to apply migration 004: %w", err)
	}

	return nil
}

// applyMigration applies a single migration if not already applied
func (db *DB) applyMigration(version string, sql string) error {
	// マイグレーションが既に適用されているかチェック
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check migration status: %w", err)
	}

	if count > 0 {
		// 既に適用済み
		return nil
	}

	// マイグレーションを実行
	_, err = db.conn.Exec(sql)
	if err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// マイグレーション記録を追加
	_, err = db.conn.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return nil
}
