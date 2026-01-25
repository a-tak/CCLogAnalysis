package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/a-tak/ccloganalysis/internal/db"
	"github.com/a-tak/ccloganalysis/internal/parser"
)

func TestNewScanManager(t *testing.T) {
	// テスト用データベースを作成
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer database.Close()

	// テスト用パーサーを作成
	p := parser.NewParser(tmpDir)

	// ScanManagerを作成
	manager := NewScanManager(database, p)

	// 初期状態を確認
	if manager == nil {
		t.Fatal("NewScanManager returned nil")
	}

	progress := manager.GetProgress()
	if progress.Status != ScanStatusIdle {
		t.Errorf("Expected status to be idle, got %s", progress.Status)
	}
}

func TestStartInitialScan(t *testing.T) {
	// テスト用データベースを作成
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer database.Close()

	// テスト用のClaudeプロジェクトディレクトリを作成
	claudeDir := filepath.Join(tmpDir, "claude_projects")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create test claude directory: %v", err)
	}

	// テスト用パーサーを作成
	p := parser.NewParser(claudeDir)

	// ScanManagerを作成
	manager := NewScanManager(database, p)

	// スキャンを開始
	ctx := context.Background()
	err = manager.StartInitialScan(ctx)
	if err != nil {
		t.Fatalf("Failed to start initial scan: %v", err)
	}

	// スキャンが開始されたことを確認
	progress := manager.GetProgress()
	if progress.Status != ScanStatusRunning && progress.Status != ScanStatusCompleted {
		t.Errorf("Expected status to be running or completed, got %s", progress.Status)
	}

	// スキャンの完了を待つ
	manager.Stop()

	// 最終的な状態を確認
	finalProgress := manager.GetProgress()
	if finalProgress.Status != ScanStatusCompleted && finalProgress.Status != ScanStatusFailed {
		t.Errorf("Expected status to be completed or failed, got %s", finalProgress.Status)
	}

	// CompletedAtが設定されていることを確認
	if finalProgress.CompletedAt == nil {
		t.Error("Expected CompletedAt to be set")
	}
}

func TestStartInitialScan_AlreadyRunning(t *testing.T) {
	// テスト用データベースを作成
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer database.Close()

	// テスト用のClaudeプロジェクトディレクトリを作成
	claudeDir := filepath.Join(tmpDir, "claude_projects")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create test claude directory: %v", err)
	}

	// テスト用パーサーを作成
	p := parser.NewParser(claudeDir)

	// ScanManagerを作成
	manager := NewScanManager(database, p)

	// 最初のスキャンを開始
	ctx := context.Background()
	err = manager.StartInitialScan(ctx)
	if err != nil {
		t.Fatalf("Failed to start initial scan: %v", err)
	}

	// 2回目のスキャンを開始（エラーになるはず）
	err = manager.StartInitialScan(ctx)
	if err == nil {
		t.Error("Expected error when starting scan while already running, got nil")
	}

	// クリーンアップ
	manager.Stop()
}

func TestGetProgress_ThreadSafe(t *testing.T) {
	// テスト用データベースを作成
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer database.Close()

	// テスト用のClaudeプロジェクトディレクトリを作成
	claudeDir := filepath.Join(tmpDir, "claude_projects")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create test claude directory: %v", err)
	}

	// テスト用パーサーを作成
	p := parser.NewParser(claudeDir)

	// ScanManagerを作成
	manager := NewScanManager(database, p)

	// スキャンを開始
	ctx := context.Background()
	err = manager.StartInitialScan(ctx)
	if err != nil {
		t.Fatalf("Failed to start initial scan: %v", err)
	}

	// 複数のgoroutineから同時にGetProgressを呼び出す
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				progress := manager.GetProgress()
				// パニックしないことを確認
				_ = progress.Status
			}
			done <- true
		}()
	}

	// 全てのgoroutineが完了するのを待つ
	for i := 0; i < 10; i++ {
		<-done
	}

	// クリーンアップ
	manager.Stop()
}

func TestStop(t *testing.T) {
	// テスト用データベースを作成
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer database.Close()

	// テスト用のClaudeプロジェクトディレクトリを作成
	claudeDir := filepath.Join(tmpDir, "claude_projects")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("Failed to create test claude directory: %v", err)
	}

	// テスト用パーサーを作成
	p := parser.NewParser(claudeDir)

	// ScanManagerを作成
	manager := NewScanManager(database, p)

	// スキャンを開始
	ctx := context.Background()
	err = manager.StartInitialScan(ctx)
	if err != nil {
		t.Fatalf("Failed to start initial scan: %v", err)
	}

	// Stopを呼び出し
	startTime := time.Now()
	manager.Stop()
	elapsed := time.Since(startTime)

	// Stopが適切な時間内に完了することを確認（タイムアウト: 5秒）
	if elapsed > 5*time.Second {
		t.Errorf("Stop took too long: %v", elapsed)
	}

	// スキャンが完了または失敗状態になっていることを確認
	progress := manager.GetProgress()
	if progress.Status != ScanStatusCompleted && progress.Status != ScanStatusFailed {
		t.Errorf("Expected status to be completed or failed after Stop, got %s", progress.Status)
	}
}
