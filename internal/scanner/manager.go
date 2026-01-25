package scanner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/a-tak/ccloganalysis/internal/db"
	"github.com/a-tak/ccloganalysis/internal/parser"
)

// ScanStatus represents the current status of a scan operation
type ScanStatus string

const (
	ScanStatusIdle      ScanStatus = "idle"      // スキャン未実行
	ScanStatusRunning   ScanStatus = "running"   // スキャン実行中
	ScanStatusCompleted ScanStatus = "completed" // スキャン完了
	ScanStatusFailed    ScanStatus = "failed"    // スキャン失敗
)

// ScanProgress represents the progress of a scan operation
type ScanProgress struct {
	Status            ScanStatus
	ProjectsProcessed int
	SessionsFound     int
	SessionsSynced    int
	ErrorCount        int
	StartedAt         time.Time
	CompletedAt       *time.Time
	LastError         string
}

// ScanManager manages the lifecycle and state of scan operations
type ScanManager struct {
	db       *db.DB
	parser   *parser.Parser
	progress *ScanProgress
	mu       sync.RWMutex
	cancelFn context.CancelFunc
	wg       sync.WaitGroup
}

// NewScanManager creates a new ScanManager instance
func NewScanManager(database *db.DB, p *parser.Parser) *ScanManager {
	return &ScanManager{
		db:     database,
		parser: p,
		progress: &ScanProgress{
			Status: ScanStatusIdle,
		},
	}
}

// StartInitialScan starts the initial scan operation asynchronously
func (m *ScanManager) StartInitialScan(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 既にスキャンが実行中の場合はエラー
	if m.progress.Status == ScanStatusRunning {
		return fmt.Errorf("scan is already running")
	}

	// スキャン状態を初期化
	m.progress = &ScanProgress{
		Status:    ScanStatusRunning,
		StartedAt: time.Now(),
	}

	// キャンセル可能なコンテキストを作成
	scanCtx, cancel := context.WithCancel(ctx)
	m.cancelFn = cancel

	// goroutineでスキャンを実行
	m.wg.Add(1)
	go m.runScan(scanCtx)

	return nil
}

// GetProgress returns the current scan progress (thread-safe)
func (m *ScanManager) GetProgress() ScanProgress {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// コピーを返す（構造体のコピー）
	progress := *m.progress

	return progress
}

// Stop stops the scan operation gracefully
func (m *ScanManager) Stop() {
	m.mu.Lock()
	if m.cancelFn != nil {
		m.cancelFn()
	}
	m.mu.Unlock()

	// スキャンの完了を待つ
	m.wg.Wait()
}

// runScan executes the scan operation (runs in goroutine)
func (m *ScanManager) runScan(ctx context.Context) {
	defer m.wg.Done()

	// スキャン実行
	result, err := db.SyncAll(m.db, m.parser)

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	if err != nil {
		// エラー時
		m.progress.Status = ScanStatusFailed
		m.progress.LastError = err.Error()
		m.progress.CompletedAt = &now
		fmt.Printf("Scan failed: %v\n", err)
		return
	}

	// 成功時
	m.progress.Status = ScanStatusCompleted
	m.progress.ProjectsProcessed = result.ProjectsProcessed
	m.progress.SessionsFound = result.SessionsFound
	m.progress.SessionsSynced = result.SessionsSynced
	m.progress.ErrorCount = result.ErrorCount
	m.progress.CompletedAt = &now

	if result.ErrorCount > 0 {
		m.progress.LastError = fmt.Sprintf("%d errors occurred during scan", result.ErrorCount)
	}

	fmt.Printf("Scan completed: %d projects, %d sessions synced\n",
		result.ProjectsProcessed, result.SessionsSynced)
}
