package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/a-tak/ccloganalysis/internal/db"
	"github.com/a-tak/ccloganalysis/internal/parser"
)

func TestDebugStatusHandler(t *testing.T) {
	t.Run("正常時のステータスを返す", func(t *testing.T) {
		// テスト用のDBとパーサーをセットアップ
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")
		database, err := db.NewDB(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer database.Close()

		// テスト用のClaudeプロジェクトディレクトリを作成
		claudeDir := filepath.Join(tmpDir, ".claude", "projects")
		err = os.MkdirAll(filepath.Join(claudeDir, "test-project-1"), 0755)
		if err != nil {
			t.Fatalf("Failed to create test project dir: %v", err)
		}
		err = os.MkdirAll(filepath.Join(claudeDir, "test-project-2"), 0755)
		if err != nil {
			t.Fatalf("Failed to create test project dir: %v", err)
		}

		p := parser.NewParser(claudeDir)

		// サービスを作成（初期同期なし）
		service := &DatabaseSessionService{
			db:     database,
			parser: p,
		}

		// プロジェクトとセッションをDBに追加
		_, err = database.CreateProject("test-project-1", "/path/to/project1")
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		// HTTPリクエストを作成
		req := httptest.NewRequest(http.MethodGet, "/api/debug/status", nil)
		rec := httptest.NewRecorder()

		// ハンドラーを実行
		handler := DebugStatusHandler(service)
		handler.ServeHTTP(rec, req)

		// ステータスコードを確認
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		// レスポンスボディを解析
		var response DebugStatusResponse
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// レスポンスの内容を確認
		if response.DBProjects != 1 {
			t.Errorf("Expected 1 DB project, got %d", response.DBProjects)
		}
		if response.DBSessions != 0 {
			t.Errorf("Expected 0 DB sessions, got %d", response.DBSessions)
		}
		if response.FSProjects != 2 {
			t.Errorf("Expected 2 filesystem projects, got %d", response.FSProjects)
		}
		if response.SyncStatus != "not_synced" && response.SyncStatus != "success" {
			t.Errorf("Expected sync status 'not_synced' or 'success', got '%s'", response.SyncStatus)
		}
	})

	t.Run("初期同期エラーの情報を返す", func(t *testing.T) {
		// テスト用のDBをセットアップ
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")
		database, err := db.NewDB(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer database.Close()

		// 存在しないディレクトリを指定してパーサーを作成（同期が失敗する）
		claudeDir := filepath.Join(tmpDir, "nonexistent", ".claude", "projects")
		p := parser.NewParser(claudeDir)

		// サービスを作成（初期同期を試みるがエラーになる）
		service := &DatabaseSessionService{
			db:        database,
			parser:    p,
			syncError: errors.New("test sync error"),
		}

		// HTTPリクエストを作成
		req := httptest.NewRequest(http.MethodGet, "/api/debug/status", nil)
		rec := httptest.NewRecorder()

		// ハンドラーを実行
		handler := DebugStatusHandler(service)
		handler.ServeHTTP(rec, req)

		// ステータスコードを確認
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		// レスポンスボディを解析
		var response DebugStatusResponse
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// 同期エラーが記録されていることを確認
		if response.SyncStatus != "failed" {
			t.Errorf("Expected sync status 'failed', got '%s'", response.SyncStatus)
		}
		if response.SyncError == "" {
			t.Error("Expected sync error message, got empty string")
		}
		if response.SyncError != "test sync error" {
			t.Errorf("Expected sync error 'test sync error', got '%s'", response.SyncError)
		}
	})

	t.Run("パーサーがnilの場合", func(t *testing.T) {
		// テスト用のDBをセットアップ
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")
		database, err := db.NewDB(dbPath)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer database.Close()

		// パーサーなしでサービスを作成
		service := &DatabaseSessionService{
			db:     database,
			parser: nil,
		}

		// HTTPリクエストを作成
		req := httptest.NewRequest(http.MethodGet, "/api/debug/status", nil)
		rec := httptest.NewRecorder()

		// ハンドラーを実行
		handler := DebugStatusHandler(service)
		handler.ServeHTTP(rec, req)

		// ステータスコードを確認
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		// レスポンスボディを解析
		var response DebugStatusResponse
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// FSProjectsが0であることを確認
		if response.FSProjects != 0 {
			t.Errorf("Expected 0 filesystem projects (parser is nil), got %d", response.FSProjects)
		}
	})
}
