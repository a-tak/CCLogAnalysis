package db

import (
	"testing"
	"time"

	"github.com/a-tak/ccloganalysis/internal/parser"
)

// createTestSession creates a sample parser.Session for testing
// suffix is used to make UUIDs unique across different test cases
func createTestSession(suffix string) *parser.Session {
	now := time.Now().Truncate(time.Second)
	startTime := now.Add(-1 * time.Hour)
	endTime := now

	sessionID := "test-session-" + suffix
	gitBranch := "main"
	projectPath := "/path/to/test-project"

	// ログエントリ作成
	parentUUID := "parent-uuid-" + suffix
	entries := []parser.LogEntry{
		{
			Type:       "user",
			Timestamp:  startTime,
			SessionID:  sessionID,
			UUID:       "entry-uuid-1-" + suffix,
			ParentUUID: nil,
			Cwd:        projectPath,
			Version:    "1.0.0",
			GitBranch:  gitBranch,
			Message: &parser.Message{
				Model: "claude-sonnet-4-5",
				ID:    "msg-1",
				Role:  "user",
				Content: []parser.Content{
					{
						Type: "text",
						Text: "Hello, Claude!",
					},
				},
			},
			RequestID: "req-1",
		},
		{
			Type:       "assistant",
			Timestamp:  startTime.Add(5 * time.Second),
			SessionID:  sessionID,
			UUID:       "entry-uuid-2-" + suffix,
			ParentUUID: &parentUUID,
			Cwd:        projectPath,
			Version:    "1.0.0",
			GitBranch:  gitBranch,
			Message: &parser.Message{
				Model: "claude-sonnet-4-5",
				ID:    "msg-2",
				Role:  "assistant",
				Content: []parser.Content{
					{
						Type: "text",
						Text: "こんにちは！何かお手伝いできることはありますか？",
					},
					{
						Type:  "tool_use",
						ID:    "tool-1",
						Name:  "read_file",
						Input: map[string]interface{}{"path": "/test/file.go"},
					},
				},
				Usage: &parser.Usage{
					InputTokens:              100,
					OutputTokens:             50,
					CacheCreationInputTokens: 200,
					CacheReadInputTokens:     150,
				},
			},
			RequestID: "req-2",
		},
	}

	// ツール呼び出し
	toolCalls := []parser.ToolCall{
		{
			Timestamp: startTime.Add(6 * time.Second),
			Name:      "read_file",
			Input:     map[string]interface{}{"path": "/test/file.go"},
			IsError:   false,
			Result:    "package main\n\nfunc main() {}",
		},
		{
			Timestamp: startTime.Add(10 * time.Second),
			Name:      "write_file",
			Input:     map[string]interface{}{"path": "/test/output.txt", "content": "test"},
			IsError:   true,
			Result:    "Error: permission denied",
		},
	}

	return &parser.Session{
		ID:          sessionID,
		ProjectPath: projectPath,
		GitBranch:   gitBranch,
		StartTime:   startTime,
		EndTime:     endTime,
		Entries:     entries,
		TotalTokens: parser.TokenSummary{
			InputTokens:              100,
			OutputTokens:             50,
			CacheCreationInputTokens: 200,
			CacheReadInputTokens:     150,
		},
		ModelUsage: map[string]parser.TokenSummary{
			"claude-sonnet-4-5": {
				InputTokens:              100,
				OutputTokens:             50,
				CacheCreationInputTokens: 200,
				CacheReadInputTokens:     150,
			},
		},
		ToolCalls:  toolCalls,
		ErrorCount: 1,
	}
}

func TestCreateSession(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	t.Run("正常にセッションを作成できる", func(t *testing.T) {
		// テストプロジェクト作成
		projectName := "test-project"
		decodedPath := "/path/to/test-project"
		_, err := db.CreateProject(projectName, decodedPath)
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		// テストセッション作成
		session := createTestSession("1")

		// セッション保存
		err = db.CreateSession(session, projectName)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// セッションが保存されたことを確認
		var count int
		err = db.conn.QueryRow("SELECT COUNT(*) FROM sessions WHERE id = ?", session.ID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query session: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected 1 session, got %d", count)
		}
	})

	t.Run("セッションのトークン集計が正しく保存される", func(t *testing.T) {
		projectName := "token-test-project"
		decodedPath := "/path/to/token-test-project"
		_, err := db.CreateProject(projectName, decodedPath)
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		session := createTestSession("token")

		err = db.CreateSession(session, projectName)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// トークン集計を確認
		var inputTokens, outputTokens, cacheCreation, cacheRead int
		query := `
			SELECT total_input_tokens, total_output_tokens,
			       total_cache_creation_tokens, total_cache_read_tokens
			FROM sessions WHERE id = ?
		`
		err = db.conn.QueryRow(query, session.ID).Scan(
			&inputTokens, &outputTokens, &cacheCreation, &cacheRead,
		)
		if err != nil {
			t.Fatalf("Failed to query tokens: %v", err)
		}

		if inputTokens != session.TotalTokens.InputTokens {
			t.Errorf("Expected input_tokens=%d, got %d", session.TotalTokens.InputTokens, inputTokens)
		}
		if outputTokens != session.TotalTokens.OutputTokens {
			t.Errorf("Expected output_tokens=%d, got %d", session.TotalTokens.OutputTokens, outputTokens)
		}
		if cacheCreation != session.TotalTokens.CacheCreationInputTokens {
			t.Errorf("Expected cache_creation=%d, got %d", session.TotalTokens.CacheCreationInputTokens, cacheCreation)
		}
		if cacheRead != session.TotalTokens.CacheReadInputTokens {
			t.Errorf("Expected cache_read=%d, got %d", session.TotalTokens.CacheReadInputTokens, cacheRead)
		}
	})

	t.Run("モデル使用量が正しく保存される", func(t *testing.T) {
		projectName := "model-test-project"
		decodedPath := "/path/to/model-test-project"
		_, err := db.CreateProject(projectName, decodedPath)
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		session := createTestSession("model")

		err = db.CreateSession(session, projectName)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// モデル使用量を確認
		var count int
		err = db.conn.QueryRow("SELECT COUNT(*) FROM model_usage WHERE session_id = ?", session.ID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query model usage: %v", err)
		}

		expectedCount := len(session.ModelUsage)
		if count != expectedCount {
			t.Errorf("Expected %d model usage entries, got %d", expectedCount, count)
		}

		// 各モデルのトークン数を確認
		for model, tokens := range session.ModelUsage {
			var inputTokens, outputTokens int
			query := "SELECT input_tokens, output_tokens FROM model_usage WHERE session_id = ? AND model = ?"
			err = db.conn.QueryRow(query, session.ID, model).Scan(&inputTokens, &outputTokens)
			if err != nil {
				t.Errorf("Failed to query model %s: %v", model, err)
				continue
			}

			if inputTokens != tokens.InputTokens {
				t.Errorf("Model %s: expected input_tokens=%d, got %d", model, tokens.InputTokens, inputTokens)
			}
			if outputTokens != tokens.OutputTokens {
				t.Errorf("Model %s: expected output_tokens=%d, got %d", model, tokens.OutputTokens, outputTokens)
			}
		}
	})

	t.Run("ログエントリが正しく保存される", func(t *testing.T) {
		projectName := "entry-test-project"
		decodedPath := "/path/to/entry-test-project"
		_, err := db.CreateProject(projectName, decodedPath)
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		session := createTestSession("entry")

		err = db.CreateSession(session, projectName)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// ログエントリ数を確認
		var count int
		err = db.conn.QueryRow("SELECT COUNT(*) FROM log_entries WHERE session_id = ?", session.ID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query log entries: %v", err)
		}

		expectedCount := len(session.Entries)
		if count != expectedCount {
			t.Errorf("Expected %d log entries, got %d", expectedCount, count)
		}
	})

	t.Run("メッセージが正しく保存される", func(t *testing.T) {
		projectName := "message-test-project"
		decodedPath := "/path/to/message-test-project"
		_, err := db.CreateProject(projectName, decodedPath)
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		session := createTestSession("message")

		err = db.CreateSession(session, projectName)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// メッセージ数を確認（メッセージを持つエントリの数）
		var count int
		err = db.conn.QueryRow("SELECT COUNT(*) FROM messages WHERE log_entry_id IN (SELECT id FROM log_entries WHERE session_id = ?)", session.ID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query messages: %v", err)
		}

		// createTestSession()では2つのエントリ、両方にメッセージがある
		expectedCount := 2
		if count != expectedCount {
			t.Errorf("Expected %d messages, got %d", expectedCount, count)
		}
	})

	t.Run("ツール呼び出しが正しく保存される", func(t *testing.T) {
		projectName := "tool-test-project"
		decodedPath := "/path/to/tool-test-project"
		_, err := db.CreateProject(projectName, decodedPath)
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		session := createTestSession("tool")

		err = db.CreateSession(session, projectName)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// ツール呼び出し数を確認
		var count int
		err = db.conn.QueryRow("SELECT COUNT(*) FROM tool_calls WHERE session_id = ?", session.ID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query tool calls: %v", err)
		}

		expectedCount := len(session.ToolCalls)
		if count != expectedCount {
			t.Errorf("Expected %d tool calls, got %d", expectedCount, count)
		}

		// エラーのあるツール呼び出しを確認
		var errorCount int
		err = db.conn.QueryRow("SELECT COUNT(*) FROM tool_calls WHERE session_id = ? AND is_error = 1", session.ID).Scan(&errorCount)
		if err != nil {
			t.Fatalf("Failed to query error tool calls: %v", err)
		}

		expectedErrorCount := 1
		if errorCount != expectedErrorCount {
			t.Errorf("Expected %d error tool calls, got %d", expectedErrorCount, errorCount)
		}
	})

	t.Run("存在しないプロジェクト名でエラーを返す", func(t *testing.T) {
		session := createTestSession("invalid")

		err := db.CreateSession(session, "non-existent-project")
		if err == nil {
			t.Error("Expected error for non-existent project, got nil")
		}
	})

	t.Run("トランザクション失敗時にロールバックされる", func(t *testing.T) {
		projectName := "rollback-test-project"
		decodedPath := "/path/to/rollback-test-project"
		_, err := db.CreateProject(projectName, decodedPath)
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		session := createTestSession("rollback")

		// 1回目の保存（成功するはず）
		err = db.CreateSession(session, projectName)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// 2回目の保存（同じIDで失敗するはず）
		err = db.CreateSession(session, projectName)
		if err == nil {
			t.Error("Expected error for duplicate session ID, got nil")
		}

		// 最初の保存だけが残っていることを確認
		var count int
		err = db.conn.QueryRow("SELECT COUNT(*) FROM sessions WHERE id = ?", session.ID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query sessions: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected 1 session after rollback, got %d", count)
		}
	})
}

func TestGetSession(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	t.Run("正常にセッションを取得できる", func(t *testing.T) {
		// テストプロジェクトとセッション作成
		projectName := "get-session-project"
		decodedPath := "/path/to/get-session-project"
		_, err := db.CreateProject(projectName, decodedPath)
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		originalSession := createTestSession("get")

		err = db.CreateSession(originalSession, projectName)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// セッション取得
		retrievedSession, err := db.GetSession(originalSession.ID)
		if err != nil {
			t.Fatalf("Failed to get session: %v", err)
		}

		// セッション情報を検証
		if retrievedSession.ID != originalSession.ID {
			t.Errorf("Expected ID=%s, got %s", originalSession.ID, retrievedSession.ID)
		}
		if retrievedSession.GitBranch != originalSession.GitBranch {
			t.Errorf("Expected GitBranch=%s, got %s", originalSession.GitBranch, retrievedSession.GitBranch)
		}
		if !retrievedSession.StartTime.Equal(originalSession.StartTime) {
			t.Errorf("Expected StartTime=%v, got %v", originalSession.StartTime, retrievedSession.StartTime)
		}
		if !retrievedSession.EndTime.Equal(originalSession.EndTime) {
			t.Errorf("Expected EndTime=%v, got %v", originalSession.EndTime, retrievedSession.EndTime)
		}

		// トークン集計を検証
		if retrievedSession.TotalTokens.InputTokens != originalSession.TotalTokens.InputTokens {
			t.Errorf("Expected InputTokens=%d, got %d",
				originalSession.TotalTokens.InputTokens, retrievedSession.TotalTokens.InputTokens)
		}

		// モデル使用量を検証
		if len(retrievedSession.ModelUsage) != len(originalSession.ModelUsage) {
			t.Errorf("Expected %d models, got %d",
				len(originalSession.ModelUsage), len(retrievedSession.ModelUsage))
		}

		// ログエントリを検証
		if len(retrievedSession.Entries) != len(originalSession.Entries) {
			t.Errorf("Expected %d entries, got %d",
				len(originalSession.Entries), len(retrievedSession.Entries))
		}

		// ツール呼び出しを検証
		if len(retrievedSession.ToolCalls) != len(originalSession.ToolCalls) {
			t.Errorf("Expected %d tool calls, got %d",
				len(originalSession.ToolCalls), len(retrievedSession.ToolCalls))
		}

		// エラーカウントを検証
		if retrievedSession.ErrorCount != originalSession.ErrorCount {
			t.Errorf("Expected ErrorCount=%d, got %d",
				originalSession.ErrorCount, retrievedSession.ErrorCount)
		}
	})

	t.Run("存在しないセッションIDでエラーを返す", func(t *testing.T) {
		_, err := db.GetSession("non-existent-session-id")
		if err == nil {
			t.Error("Expected error for non-existent session, got nil")
		}
	})
}

func TestListSessions(t *testing.T) {
	t.Run("空のリストを返す", func(t *testing.T) {
		db, _ := setupTestDB(t)
		defer db.Close()

		rows, err := db.ListSessions(nil, 10, 0)
		if err != nil {
			t.Fatalf("Failed to list sessions: %v", err)
		}

		if len(rows) != 0 {
			t.Errorf("Expected empty list, got %d sessions", len(rows))
		}
	})

	t.Run("複数のセッションを取得できる", func(t *testing.T) {
		db, _ := setupTestDB(t)
		defer db.Close()

		projectName := "list-test-project"
		decodedPath := "/path/to/list-test-project"
		projectID, err := db.CreateProject(projectName, decodedPath)
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		// 3つのセッションを作成
		for i := 0; i < 3; i++ {
			session := createTestSession("list-" + string(rune('1'+i)))

			err = db.CreateSession(session, projectName)
			if err != nil {
				t.Fatalf("Failed to create session %d: %v", i, err)
			}
		}

		// セッション一覧取得
		rows, err := db.ListSessions(nil, 10, 0)
		if err != nil {
			t.Fatalf("Failed to list sessions: %v", err)
		}

		if len(rows) != 3 {
			t.Errorf("Expected 3 sessions, got %d", len(rows))
		}

		// プロジェクトIDでフィルタリング
		rows, err = db.ListSessions(&projectID, 10, 0)
		if err != nil {
			t.Fatalf("Failed to list sessions by project: %v", err)
		}

		if len(rows) != 3 {
			t.Errorf("Expected 3 sessions for project, got %d", len(rows))
		}
	})

	t.Run("limitとoffsetが正しく機能する", func(t *testing.T) {
		db, _ := setupTestDB(t)
		defer db.Close()

		projectName := "pagination-test-project"
		decodedPath := "/path/to/pagination-test-project"
		_, err := db.CreateProject(projectName, decodedPath)
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		// 5つのセッションを作成
		for i := 0; i < 5; i++ {
			session := createTestSession("pagination-" + string(rune('1'+i)))

			err = db.CreateSession(session, projectName)
			if err != nil {
				t.Fatalf("Failed to create session %d: %v", i, err)
			}
		}

		// limit=2で取得
		rows, err := db.ListSessions(nil, 2, 0)
		if err != nil {
			t.Fatalf("Failed to list sessions: %v", err)
		}

		if len(rows) != 2 {
			t.Errorf("Expected 2 sessions with limit=2, got %d", len(rows))
		}

		// offset=2で取得
		rows, err = db.ListSessions(nil, 10, 2)
		if err != nil {
			t.Fatalf("Failed to list sessions: %v", err)
		}

		if len(rows) != 3 { // 5 - 2 = 3
			t.Errorf("Expected 3 sessions with offset=2, got %d", len(rows))
		}
	})

	t.Run("最初のユーザーメッセージを取得できる", func(t *testing.T) {
		db, _ := setupTestDB(t)
		defer db.Close()

		projectName := "first-message-project"
		decodedPath := "/path/to/first-message-project"
		_, err := db.CreateProject(projectName, decodedPath)
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		// セッション作成
		session := createTestSession("first-msg")
		err = db.CreateSession(session, projectName)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// セッション一覧取得
		rows, err := db.ListSessions(nil, 10, 0)
		if err != nil {
			t.Fatalf("Failed to list sessions: %v", err)
		}

		if len(rows) == 0 {
			t.Fatal("Expected at least 1 session, got 0")
		}

		// 最初のユーザーメッセージが取得されていることを確認
		// createTestSession()の最初のエントリは"Hello, Claude!"
		expectedMessage := "Hello, Claude!"
		if rows[0].FirstUserMessage != expectedMessage {
			t.Errorf("Expected FirstUserMessage=%q, got %q", expectedMessage, rows[0].FirstUserMessage)
		}
	})

	t.Run("100文字超のメッセージが切り詰められる", func(t *testing.T) {
		db, _ := setupTestDB(t)
		defer db.Close()

		projectName := "long-message-project"
		decodedPath := "/path/to/long-message-project"
		_, err := db.CreateProject(projectName, decodedPath)
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		// 長いメッセージを持つセッションを作成
		session := createTestSession("long-msg")
		// 最初のエントリのメッセージを長い文字列に置き換え
		longMessage := "これは非常に長いメッセージです。" + string(make([]byte, 150)) // 150バイト以上の文字列
		for i := range longMessage[30:] {
			longMessage = longMessage[:30+i] + "あ"
		}
		session.Entries[0].Message.Content[0].Text = longMessage

		err = db.CreateSession(session, projectName)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// セッション一覧取得
		rows, err := db.ListSessions(nil, 10, 0)
		if err != nil {
			t.Fatalf("Failed to list sessions: %v", err)
		}

		if len(rows) == 0 {
			t.Fatal("Expected at least 1 session, got 0")
		}

		// メッセージが100文字に切り詰められていることを確認
		messageLen := len([]rune(rows[0].FirstUserMessage))
		if messageLen > 100 {
			t.Errorf("Expected FirstUserMessage length <= 100, got %d", messageLen)
		}
	})

	t.Run("メッセージがないセッションで空文字列を返す", func(t *testing.T) {
		db, _ := setupTestDB(t)
		defer db.Close()

		projectName := "no-message-project"
		decodedPath := "/path/to/no-message-project"
		_, err := db.CreateProject(projectName, decodedPath)
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		// メッセージなしのセッションを作成
		session := createTestSession("no-msg")
		session.Entries = []parser.LogEntry{} // メッセージを削除

		err = db.CreateSession(session, projectName)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// セッション一覧取得
		rows, err := db.ListSessions(nil, 10, 0)
		if err != nil {
			t.Fatalf("Failed to list sessions: %v", err)
		}

		if len(rows) == 0 {
			t.Fatal("Expected at least 1 session, got 0")
		}

		// メッセージがない場合は空文字列
		if rows[0].FirstUserMessage != "" {
			t.Errorf("Expected empty FirstUserMessage, got %q", rows[0].FirstUserMessage)
		}
	})

	t.Run("アシスタントメッセージは除外される", func(t *testing.T) {
		db, _ := setupTestDB(t)
		defer db.Close()

		projectName := "assistant-first-project"
		decodedPath := "/path/to/assistant-first-project"
		_, err := db.CreateProject(projectName, decodedPath)
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		// アシスタントメッセージが最初のセッションを作成
		session := createTestSession("assistant-first")
		// エントリの順序を逆にして、assistantが最初になるようにする
		session.Entries[0], session.Entries[1] = session.Entries[1], session.Entries[0]
		// タイムスタンプも調整
		session.Entries[0].Timestamp = session.StartTime
		session.Entries[1].Timestamp = session.StartTime.Add(5 * time.Second)

		err = db.CreateSession(session, projectName)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// セッション一覧取得
		rows, err := db.ListSessions(nil, 10, 0)
		if err != nil {
			t.Fatalf("Failed to list sessions: %v", err)
		}

		if len(rows) == 0 {
			t.Fatal("Expected at least 1 session, got 0")
		}

		// ユーザーメッセージ("Hello, Claude!")が取得されることを確認
		expectedMessage := "Hello, Claude!"
		if rows[0].FirstUserMessage != expectedMessage {
			t.Errorf("Expected FirstUserMessage=%q, got %q", expectedMessage, rows[0].FirstUserMessage)
		}
	})

	t.Run("最初のメッセージがWarmupの場合はスキップされる", func(t *testing.T) {
		db, _ := setupTestDB(t)
		defer db.Close()

		projectName := "warmup-skip-project"
		decodedPath := "/path/to/warmup-skip-project"
		_, err := db.CreateProject(projectName, decodedPath)
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		// Warmupが最初で、その後に実際のメッセージがあるセッションを作成
		session := createTestSession("warmup-skip")
		session.Entries = []parser.LogEntry{
			{
				Type:      "user",
				Timestamp: session.StartTime,
				SessionID: session.ID,
				UUID:      "entry-uuid-1-warmup-skip",
				Cwd:       "/path/to/test-project",
				GitBranch: "main",
				Message: &parser.Message{
					Model: "claude-sonnet-4-5",
					Role:  "user",
					Content: []parser.Content{
						{Type: "text", Text: "Warmup"},
					},
				},
			},
			{
				Type:      "user",
				Timestamp: session.StartTime.Add(10 * time.Second),
				SessionID: session.ID,
				UUID:      "entry-uuid-2-warmup-skip",
				Cwd:       "/path/to/test-project",
				GitBranch: "main",
				Message: &parser.Message{
					Model: "claude-sonnet-4-5",
					Role:  "user",
					Content: []parser.Content{
						{Type: "text", Text: "実際の質問内容"},
					},
				},
			},
		}

		err = db.CreateSession(session, projectName)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// セッション一覧取得
		rows, err := db.ListSessions(nil, 10, 0)
		if err != nil {
			t.Fatalf("Failed to list sessions: %v", err)
		}

		if len(rows) == 0 {
			t.Fatal("Expected at least 1 session, got 0")
		}

		// Warmupがスキップされて、2番目のメッセージが取得されることを確認
		expectedMessage := "実際の質問内容"
		if rows[0].FirstUserMessage != expectedMessage {
			t.Errorf("Expected FirstUserMessage=%q, got %q", expectedMessage, rows[0].FirstUserMessage)
		}
	})

	t.Run("Warmupしかない場合はアシスタントの応答を返す", func(t *testing.T) {
		db, _ := setupTestDB(t)
		defer db.Close()

		projectName := "warmup-only-project"
		decodedPath := "/path/to/warmup-only-project"
		_, err := db.CreateProject(projectName, decodedPath)
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		// Warmupしかないセッションを作成（アシスタント応答あり）
		session := createTestSession("warmup-only")
		session.Entries = []parser.LogEntry{
			{
				Type:      "user",
				Timestamp: session.StartTime,
				SessionID: session.ID,
				UUID:      "entry-uuid-1-warmup-only",
				Cwd:       "/path/to/test-project",
				GitBranch: "main",
				Message: &parser.Message{
					Model: "claude-sonnet-4-5",
					Role:  "user",
					Content: []parser.Content{
						{Type: "text", Text: "Warmup"},
					},
				},
			},
			{
				Type:      "assistant",
				Timestamp: session.StartTime.Add(time.Second),
				SessionID: session.ID,
				UUID:      "entry-uuid-2-warmup-only",
				Cwd:       "/path/to/test-project",
				GitBranch: "main",
				Message: &parser.Message{
					Model: "claude-sonnet-4-5",
					Role:  "assistant",
					Content: []parser.Content{
						{Type: "text", Text: "I understand this is a warmup request. I'm ready to help with your project."},
					},
				},
			},
		}

		err = db.CreateSession(session, projectName)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// セッション一覧取得
		rows, err := db.ListSessions(nil, 10, 0)
		if err != nil {
			t.Fatalf("Failed to list sessions: %v", err)
		}

		if len(rows) == 0 {
			t.Fatal("Expected at least 1 session, got 0")
		}

		// Warmupしかない場合はアシスタントの応答を返す
		expectedMessage := "I understand this is a warmup request. I'm ready to help with your project."
		if rows[0].FirstUserMessage != expectedMessage {
			t.Errorf("Expected FirstUserMessage=%q, got %q", expectedMessage, rows[0].FirstUserMessage)
		}
	})
}
