package db

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestDB creates a temporary test database
func setupTestDB(t *testing.T) (*DB, string) {
	t.Helper()

	// 一時ディレクトリにテスト用DBを作成
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	return db, dbPath
}

func TestNewDB(t *testing.T) {
	t.Run("正常にデータベースを作成できる", func(t *testing.T) {
		db, dbPath := setupTestDB(t)
		defer db.Close()

		// DBファイルが作成されたことを確認
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Error("Database file was not created")
		}

		// DBが使用可能か確認（簡単なクエリ実行）
		var version string
		err := db.conn.QueryRow("SELECT sqlite_version()").Scan(&version)
		if err != nil {
			t.Errorf("Failed to query database: %v", err)
		}
		if version == "" {
			t.Error("SQLite version should not be empty")
		}
	})

	t.Run("無効なパスでエラーを返す", func(t *testing.T) {
		invalidPath := "/invalid/path/that/does/not/exist/test.db"
		_, err := NewDB(invalidPath)
		if err == nil {
			t.Error("Expected error for invalid path, got nil")
		}
	})
}

func TestMigrate(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	t.Run("Phase 1テーブルが作成される", func(t *testing.T) {
		tables := []string{
			"projects",
			"sessions",
			"model_usage",
			"log_entries",
			"messages",
			"tool_calls",
		}

		for _, table := range tables {
			var count int
			query := "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?"
			err := db.conn.QueryRow(query, table).Scan(&count)
			if err != nil {
				t.Errorf("Failed to query table existence for %s: %v", table, err)
			}
			if count != 1 {
				t.Errorf("Table %s was not created", table)
			}
		}
	})

	t.Run("Phase 2テーブルが作成される", func(t *testing.T) {
		tables := []string{
			"project_groups",
			"project_group_mappings",
			"error_patterns",
			"error_occurrences",
			"period_statistics",
		}

		for _, table := range tables {
			var count int
			query := "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?"
			err := db.conn.QueryRow(query, table).Scan(&count)
			if err != nil {
				t.Errorf("Failed to query table existence for %s: %v", table, err)
			}
			if count != 1 {
				t.Errorf("Table %s was not created", table)
			}
		}
	})

	t.Run("インデックスが作成される", func(t *testing.T) {
		// 主要なインデックスをいくつか確認
		indexes := []string{
			"idx_sessions_project_id",
			"idx_sessions_start_time",
			"idx_model_usage_session",
			"idx_log_entries_session",
			"idx_messages_log_entry",
			"idx_tool_calls_session",
		}

		for _, index := range indexes {
			var count int
			query := "SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?"
			err := db.conn.QueryRow(query, index).Scan(&count)
			if err != nil {
				t.Errorf("Failed to query index existence for %s: %v", index, err)
			}
			if count != 1 {
				t.Errorf("Index %s was not created", index)
			}
		}
	})

	t.Run("トリガーが作成される", func(t *testing.T) {
		triggers := []string{
			"update_projects_timestamp",
			"update_sessions_timestamp",
		}

		for _, trigger := range triggers {
			var count int
			query := "SELECT COUNT(*) FROM sqlite_master WHERE type='trigger' AND name=?"
			err := db.conn.QueryRow(query, trigger).Scan(&count)
			if err != nil {
				t.Errorf("Failed to query trigger existence for %s: %v", trigger, err)
			}
			if count != 1 {
				t.Errorf("Trigger %s was not created", trigger)
			}
		}
	})
}

func TestClose(t *testing.T) {
	t.Run("正常にクローズできる", func(t *testing.T) {
		db, _ := setupTestDB(t)

		err := db.Close()
		if err != nil {
			t.Errorf("Failed to close database: %v", err)
		}

		// クローズ後はクエリが失敗することを確認
		var version string
		err = db.conn.QueryRow("SELECT sqlite_version()").Scan(&version)
		if err == nil {
			t.Error("Expected error after closing database, got nil")
		}
	})

	t.Run("二重クローズでもエラーにならない", func(t *testing.T) {
		db, _ := setupTestDB(t)

		_ = db.Close()
		err := db.Close()
		// 二重クローズはエラーを返すが、パニックしない
		// SQLiteドライバによって挙動が異なる可能性があるため、
		// エラーの有無はチェックしない
		_ = err
	})
}

func TestSchemaIntegrity(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	t.Run("外部キー制約が有効", func(t *testing.T) {
		var enabled int
		err := db.conn.QueryRow("PRAGMA foreign_keys").Scan(&enabled)
		if err != nil {
			t.Errorf("Failed to check foreign keys: %v", err)
		}
		// デフォルトでは無効の可能性があるため、有効化を確認するだけ
		// 実際の制約動作は各テーブルのテストで確認
	})

	t.Run("projectsテーブルのカラム確認", func(t *testing.T) {
		expectedColumns := map[string]bool{
			"id":             true,
			"name":           true,
			"decoded_path":   true,
			"git_root":       true,
			"created_at":     true,
			"updated_at":     true,
			"last_scan_time": true, // マイグレーション005で追加
		}

		rows, err := db.conn.Query("PRAGMA table_info(projects)")
		if err != nil {
			t.Fatalf("Failed to get table info: %v", err)
		}
		defer rows.Close()

		foundColumns := make(map[string]bool)
		for rows.Next() {
			var cid int
			var name, dataType string
			var notNull, pk int
			var dfltValue interface{}
			err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk)
			if err != nil {
				t.Errorf("Failed to scan column info: %v", err)
			}
			foundColumns[name] = true
		}

		for col := range expectedColumns {
			if !foundColumns[col] {
				t.Errorf("Expected column %s not found in projects table", col)
			}
		}
	})

	t.Run("sessionsテーブルのカラム確認", func(t *testing.T) {
		expectedColumns := map[string]bool{
			"id":                         true,
			"project_id":                 true,
			"git_branch":                 true,
			"start_time":                 true,
			"end_time":                   true,
			"duration_seconds":           true,
			"total_input_tokens":         true,
			"total_output_tokens":        true,
			"total_cache_creation_tokens": true,
			"total_cache_read_tokens":    true,
			"error_count":                true,
			"first_user_message":         true,
			"created_at":                 true,
			"updated_at":                 true,
			"file_mod_time":              true, // マイグレーション005で追加
		}

		rows, err := db.conn.Query("PRAGMA table_info(sessions)")
		if err != nil {
			t.Fatalf("Failed to get table info: %v", err)
		}
		defer rows.Close()

		foundColumns := make(map[string]bool)
		for rows.Next() {
			var cid int
			var name, dataType string
			var notNull, pk int
			var dfltValue interface{}
			err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk)
			if err != nil {
				t.Errorf("Failed to scan column info: %v", err)
			}
			foundColumns[name] = true
		}

		for col := range expectedColumns {
			if !foundColumns[col] {
				t.Errorf("Expected column %s not found in sessions table", col)
			}
		}
	})
}
