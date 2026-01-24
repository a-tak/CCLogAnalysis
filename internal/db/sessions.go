package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/a-tak/ccloganalysis/internal/parser"
)

// SessionRow represents a row in the sessions table
type SessionRow struct {
	ID                      string
	ProjectID               int64
	GitBranch               string
	StartTime               time.Time
	EndTime                 time.Time
	DurationSeconds         int
	TotalInputTokens        int
	TotalOutputTokens       int
	TotalCacheCreationTokens int
	TotalCacheReadTokens    int
	ErrorCount              int
	FirstUserMessage        string
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// CreateSession creates a new session and all related data in a transaction
func (db *DB) CreateSession(session *parser.Session, projectName string) error {
	// トランザクション開始
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // エラー時は自動ロールバック

	// プロジェクトIDを取得
	var projectID int64
	err = tx.QueryRow("SELECT id FROM projects WHERE name = ?", projectName).Scan(&projectID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("project not found: %s", projectName)
	}
	if err != nil {
		return fmt.Errorf("failed to get project ID: %w", err)
	}

	// First user messageを計算
	firstUserMessage := calculateFirstUserMessage(session)

	// セッション挿入
	durationSeconds := int(session.EndTime.Sub(session.StartTime).Seconds())
	sessionQuery := `
		INSERT INTO sessions (
			id, project_id, git_branch, start_time, end_time, duration_seconds,
			total_input_tokens, total_output_tokens,
			total_cache_creation_tokens, total_cache_read_tokens,
			error_count, first_user_message
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = tx.Exec(sessionQuery,
		session.ID, projectID, session.GitBranch,
		session.StartTime, session.EndTime, durationSeconds,
		session.TotalTokens.InputTokens, session.TotalTokens.OutputTokens,
		session.TotalTokens.CacheCreationInputTokens, session.TotalTokens.CacheReadInputTokens,
		session.ErrorCount,
		firstUserMessage,
	)
	if err != nil {
		return fmt.Errorf("failed to insert session: %w", err)
	}

	// モデル使用量挿入
	modelUsageQuery := `
		INSERT INTO model_usage (
			session_id, model, input_tokens, output_tokens,
			cache_creation_tokens, cache_read_tokens
		) VALUES (?, ?, ?, ?, ?, ?)
	`
	modelStmt, err := tx.Prepare(modelUsageQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare model usage statement: %w", err)
	}
	defer modelStmt.Close()

	for model, tokens := range session.ModelUsage {
		_, err = modelStmt.Exec(
			session.ID, model,
			tokens.InputTokens, tokens.OutputTokens,
			tokens.CacheCreationInputTokens, tokens.CacheReadInputTokens,
		)
		if err != nil {
			return fmt.Errorf("failed to insert model usage for %s: %w", model, err)
		}
	}

	// ログエントリ挿入
	logEntryQuery := `
		INSERT INTO log_entries (
			session_id, uuid, parent_uuid, entry_type, timestamp,
			cwd, version, request_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	logStmt, err := tx.Prepare(logEntryQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare log entry statement: %w", err)
	}
	defer logStmt.Close()

	// メッセージ挿入用
	messageQuery := `
		INSERT INTO messages (
			log_entry_id, model, role, content_text, content_json
		) VALUES (?, ?, ?, ?, ?)
	`
	msgStmt, err := tx.Prepare(messageQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare message statement: %w", err)
	}
	defer msgStmt.Close()

	for _, entry := range session.Entries {
		result, err := logStmt.Exec(
			session.ID, entry.UUID, entry.ParentUUID, entry.Type, entry.Timestamp,
			entry.Cwd, entry.Version, entry.RequestID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert log entry %s: %w", entry.UUID, err)
		}

		// メッセージがあれば挿入
		if entry.Message != nil {
			logEntryID, err := result.LastInsertId()
			if err != nil {
				return fmt.Errorf("failed to get log entry ID: %w", err)
			}

			// Content配列をJSONにシリアライズ
			contentJSON, err := json.Marshal(entry.Message.Content)
			if err != nil {
				return fmt.Errorf("failed to marshal message content: %w", err)
			}

			// テキストコンテンツを抽出（検索用）
			contentText := extractTextFromContent(entry.Message.Content)

			_, err = msgStmt.Exec(
				logEntryID, entry.Message.Model, entry.Message.Role,
				contentText, string(contentJSON),
			)
			if err != nil {
				return fmt.Errorf("failed to insert message for entry %s: %w", entry.UUID, err)
			}
		}
	}

	// ツール呼び出し挿入
	toolCallQuery := `
		INSERT INTO tool_calls (
			session_id, timestamp, tool_name, input_json, is_error, result_text
		) VALUES (?, ?, ?, ?, ?, ?)
	`
	toolStmt, err := tx.Prepare(toolCallQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare tool call statement: %w", err)
	}
	defer toolStmt.Close()

	for _, toolCall := range session.ToolCalls {
		// InputをJSONにシリアライズ
		inputJSON, err := json.Marshal(toolCall.Input)
		if err != nil {
			return fmt.Errorf("failed to marshal tool input: %w", err)
		}

		_, err = toolStmt.Exec(
			session.ID, toolCall.Timestamp, toolCall.Name,
			string(inputJSON), toolCall.IsError, toolCall.Result,
		)
		if err != nil {
			return fmt.Errorf("failed to insert tool call %s: %w", toolCall.Name, err)
		}
	}

	// トランザクションコミット
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// extractTextFromContent extracts text content from Content array for search
func extractTextFromContent(contents []parser.Content) string {
	var text string
	for i, content := range contents {
		if content.Type == "text" && content.Text != "" {
			if i > 0 {
				text += "\n"
			}
			text += content.Text
		}
	}
	return text
}

// calculateFirstUserMessage calculates the first user message for session list display
// It handles the Warmup message skip logic
func calculateFirstUserMessage(session *parser.Session) string {
	var firstUserMsg, secondUserMsg, firstAssistantMsg string

	userMsgCount := 0
	assistantMsgCount := 0

	for _, entry := range session.Entries {
		if entry.Type == "user" && entry.Message != nil {
			userMsgCount++
			if userMsgCount == 1 {
				firstUserMsg = extractTextFromContent(entry.Message.Content)
			} else if userMsgCount == 2 {
				secondUserMsg = extractTextFromContent(entry.Message.Content)
			}
		}

		if entry.Type == "assistant" && entry.Message != nil && assistantMsgCount == 0 {
			assistantMsgCount++
			firstAssistantMsg = extractTextFromContent(entry.Message.Content)
		}

		// 必要な情報が揃ったら早期終了
		if userMsgCount >= 2 && assistantMsgCount >= 1 {
			break
		}
	}

	// Warmup処理ロジック
	if firstUserMsg == "Warmup" {
		if secondUserMsg != "" {
			return truncate(secondUserMsg, 100)
		} else if firstAssistantMsg != "" {
			return truncate(firstAssistantMsg, 100)
		}
	}

	return truncate(firstUserMsg, 100)
}

// truncate truncates a string to maxLen characters (rune-based for Japanese support)
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}

// GetSession retrieves a session and all related data
func (db *DB) GetSession(sessionID string) (*parser.Session, error) {
	// セッション基本情報取得
	sessionQuery := `
		SELECT s.id, p.decoded_path, s.git_branch, s.start_time, s.end_time,
		       s.total_input_tokens, s.total_output_tokens,
		       s.total_cache_creation_tokens, s.total_cache_read_tokens,
		       s.error_count
		FROM sessions s
		JOIN projects p ON s.project_id = p.id
		WHERE s.id = ?
	`
	var session parser.Session
	var projectPath string
	err := db.conn.QueryRow(sessionQuery, sessionID).Scan(
		&session.ID, &projectPath, &session.GitBranch,
		&session.StartTime, &session.EndTime,
		&session.TotalTokens.InputTokens, &session.TotalTokens.OutputTokens,
		&session.TotalTokens.CacheCreationInputTokens, &session.TotalTokens.CacheReadInputTokens,
		&session.ErrorCount,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query session: %w", err)
	}
	session.ProjectPath = projectPath

	// モデル使用量取得
	modelUsageQuery := `
		SELECT model, input_tokens, output_tokens,
		       cache_creation_tokens, cache_read_tokens
		FROM model_usage
		WHERE session_id = ?
	`
	modelRows, err := db.conn.Query(modelUsageQuery, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query model usage: %w", err)
	}
	defer modelRows.Close()

	session.ModelUsage = make(map[string]parser.TokenSummary)
	for modelRows.Next() {
		var model string
		var tokens parser.TokenSummary
		err = modelRows.Scan(
			&model,
			&tokens.InputTokens, &tokens.OutputTokens,
			&tokens.CacheCreationInputTokens, &tokens.CacheReadInputTokens,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan model usage: %w", err)
		}
		session.ModelUsage[model] = tokens
	}

	// ログエントリとメッセージ取得
	entryQuery := `
		SELECT le.id, le.uuid, le.parent_uuid, le.entry_type, le.timestamp,
		       le.cwd, le.version, le.request_id,
		       m.model, m.role, m.content_json
		FROM log_entries le
		LEFT JOIN messages m ON le.id = m.log_entry_id
		WHERE le.session_id = ?
		ORDER BY le.timestamp
	`
	entryRows, err := db.conn.Query(entryQuery, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query log entries: %w", err)
	}
	defer entryRows.Close()

	session.Entries = []parser.LogEntry{}
	for entryRows.Next() {
		var entry parser.LogEntry
		var entryID int64
		var model, role sql.NullString
		var contentJSON sql.NullString

		err = entryRows.Scan(
			&entryID, &entry.UUID, &entry.ParentUUID, &entry.Type, &entry.Timestamp,
			&entry.Cwd, &entry.Version, &entry.RequestID,
			&model, &role, &contentJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan log entry: %w", err)
		}

		// セッション情報を設定
		entry.SessionID = sessionID
		entry.GitBranch = session.GitBranch

		// メッセージがあれば復元
		if contentJSON.Valid {
			entry.Message = &parser.Message{
				Model: model.String,
				Role:  role.String,
			}

			// Content配列をデシリアライズ
			err = json.Unmarshal([]byte(contentJSON.String), &entry.Message.Content)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal message content: %w", err)
			}
		}

		session.Entries = append(session.Entries, entry)
	}

	// ツール呼び出し取得
	toolCallQuery := `
		SELECT timestamp, tool_name, input_json, is_error, result_text
		FROM tool_calls
		WHERE session_id = ?
		ORDER BY timestamp
	`
	toolRows, err := db.conn.Query(toolCallQuery, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tool calls: %w", err)
	}
	defer toolRows.Close()

	session.ToolCalls = []parser.ToolCall{}
	for toolRows.Next() {
		var toolCall parser.ToolCall
		var inputJSON string

		err = toolRows.Scan(
			&toolCall.Timestamp, &toolCall.Name, &inputJSON,
			&toolCall.IsError, &toolCall.Result,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tool call: %w", err)
		}

		// Inputをデシリアライズ
		err = json.Unmarshal([]byte(inputJSON), &toolCall.Input)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal tool input: %w", err)
		}

		session.ToolCalls = append(session.ToolCalls, toolCall)
	}

	return &session, nil
}

// ListSessions retrieves sessions with optional filtering and pagination
func (db *DB) ListSessions(projectID *int64, limit, offset int) ([]*SessionRow, error) {
	query := `
		SELECT s.id, s.project_id, s.git_branch, s.start_time, s.end_time, s.duration_seconds,
		       s.total_input_tokens, s.total_output_tokens,
		       s.total_cache_creation_tokens, s.total_cache_read_tokens,
		       s.error_count,
		       s.first_user_message,
		       s.created_at, s.updated_at
		FROM sessions s
	`

	var args []interface{}
	if projectID != nil {
		query += " WHERE s.project_id = ?"
		args = append(args, *projectID)
	}

	query += " ORDER BY s.start_time DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*SessionRow
	for rows.Next() {
		var session SessionRow
		err = rows.Scan(
			&session.ID, &session.ProjectID, &session.GitBranch,
			&session.StartTime, &session.EndTime, &session.DurationSeconds,
			&session.TotalInputTokens, &session.TotalOutputTokens,
			&session.TotalCacheCreationTokens, &session.TotalCacheReadTokens,
			&session.ErrorCount, &session.FirstUserMessage,
			&session.CreatedAt, &session.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session row: %w", err)
		}
		sessions = append(sessions, &session)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating session rows: %w", err)
	}

	return sessions, nil
}
