-- CCLogAnalysis Database Schema
-- Phase 1: 基本機能用テーブル
-- Phase 2: 将来拡張用テーブル（スキーマのみ作成、実装は将来）

-- ============================================================
-- Phase 1: 基本機能用テーブル
-- ============================================================

-- プロジェクトテーブル
CREATE TABLE IF NOT EXISTS projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,            -- エンコード済みフォルダ名
    decoded_path TEXT NOT NULL,           -- デコード済みパス
    git_root TEXT,                        -- Gitルートパス
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- セッションテーブル
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,                  -- UUIDセッションID
    project_id INTEGER NOT NULL,
    git_branch TEXT NOT NULL,
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    duration_seconds INTEGER NOT NULL,

    -- トークン集計（非正規化：頻繁にアクセスされる）
    total_input_tokens INTEGER NOT NULL DEFAULT 0,
    total_output_tokens INTEGER NOT NULL DEFAULT 0,
    total_cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
    total_cache_read_tokens INTEGER NOT NULL DEFAULT 0,

    error_count INTEGER NOT NULL DEFAULT 0,
    first_user_message TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

-- モデル使用量テーブル
CREATE TABLE IF NOT EXISTS model_usage (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    model TEXT NOT NULL,
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
    cache_read_tokens INTEGER NOT NULL DEFAULT 0,

    UNIQUE(session_id, model),
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

-- ログエントリテーブル
CREATE TABLE IF NOT EXISTS log_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    uuid TEXT NOT NULL,                   -- エントリのUUID
    parent_uuid TEXT,                     -- 親エントリのUUID
    entry_type TEXT NOT NULL,             -- 'user', 'assistant', 'queue-operation'
    timestamp DATETIME NOT NULL,
    cwd TEXT,
    version TEXT,
    request_id TEXT,

    UNIQUE(session_id, uuid),             -- セッション内でUUIDがユニーク
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

-- メッセージテーブル（メッセージ本文検索対応）
CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    log_entry_id INTEGER NOT NULL,
    model TEXT,
    role TEXT NOT NULL,                   -- 'user', 'assistant'
    content_text TEXT,                    -- テキストコンテンツ（検索用）
    content_json TEXT NOT NULL,           -- Content配列全体（JSON）

    FOREIGN KEY (log_entry_id) REFERENCES log_entries(id) ON DELETE CASCADE
);

-- ツール呼び出しテーブル
CREATE TABLE IF NOT EXISTS tool_calls (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    log_entry_id INTEGER,
    timestamp DATETIME NOT NULL,
    tool_name TEXT NOT NULL,
    input_json TEXT,                      -- interface{}をJSON化
    is_error BOOLEAN NOT NULL DEFAULT 0,
    result_text TEXT,

    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE,
    FOREIGN KEY (log_entry_id) REFERENCES log_entries(id) ON DELETE SET NULL
);

-- ============================================================
-- Phase 1: インデックス
-- ============================================================

-- sessions
CREATE INDEX IF NOT EXISTS idx_sessions_project_id ON sessions(project_id);
CREATE INDEX IF NOT EXISTS idx_sessions_start_time ON sessions(start_time DESC);
CREATE INDEX IF NOT EXISTS idx_sessions_git_branch ON sessions(git_branch);
CREATE INDEX IF NOT EXISTS idx_sessions_project_start ON sessions(project_id, start_time DESC);

-- model_usage
CREATE INDEX IF NOT EXISTS idx_model_usage_session ON model_usage(session_id);
CREATE INDEX IF NOT EXISTS idx_model_usage_model ON model_usage(model);

-- log_entries
CREATE INDEX IF NOT EXISTS idx_log_entries_session ON log_entries(session_id);
CREATE INDEX IF NOT EXISTS idx_log_entries_timestamp ON log_entries(timestamp);
CREATE INDEX IF NOT EXISTS idx_log_entries_type ON log_entries(entry_type);

-- messages
CREATE INDEX IF NOT EXISTS idx_messages_log_entry ON messages(log_entry_id);
CREATE INDEX IF NOT EXISTS idx_messages_role ON messages(role);

-- tool_calls
CREATE INDEX IF NOT EXISTS idx_tool_calls_session ON tool_calls(session_id);
CREATE INDEX IF NOT EXISTS idx_tool_calls_timestamp ON tool_calls(timestamp);
CREATE INDEX IF NOT EXISTS idx_tool_calls_tool_name ON tool_calls(tool_name);
CREATE INDEX IF NOT EXISTS idx_tool_calls_is_error ON tool_calls(is_error);

-- ============================================================
-- Phase 1: トリガー（updated_at自動更新）
-- ============================================================

CREATE TRIGGER IF NOT EXISTS update_projects_timestamp
AFTER UPDATE ON projects
BEGIN
    UPDATE projects SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_sessions_timestamp
AFTER UPDATE ON sessions
BEGIN
    UPDATE sessions SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- ============================================================
-- Phase 2: 将来拡張用テーブル（スキーマのみ、実装は将来）
-- ============================================================

-- プロジェクトグループテーブル（Gitワークツリー対応）
CREATE TABLE IF NOT EXISTS project_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    git_root TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS project_group_mappings (
    project_id INTEGER NOT NULL,
    group_id INTEGER NOT NULL,
    PRIMARY KEY (project_id, group_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (group_id) REFERENCES project_groups(id) ON DELETE CASCADE
);

-- エラーパターンテーブル（エラーパターン検出）
CREATE TABLE IF NOT EXISTS error_patterns (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pattern_hash TEXT NOT NULL UNIQUE,
    tool_name TEXT NOT NULL,
    error_message TEXT NOT NULL,
    occurrence_count INTEGER NOT NULL DEFAULT 1,
    first_seen DATETIME NOT NULL,
    last_seen DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS error_occurrences (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pattern_id INTEGER NOT NULL,
    tool_call_id INTEGER NOT NULL,
    session_id TEXT NOT NULL,
    occurred_at DATETIME NOT NULL,

    FOREIGN KEY (pattern_id) REFERENCES error_patterns(id) ON DELETE CASCADE,
    FOREIGN KEY (tool_call_id) REFERENCES tool_calls(id) ON DELETE CASCADE,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

-- 期間別統計キャッシュテーブル
CREATE TABLE IF NOT EXISTS period_statistics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    period_type TEXT NOT NULL,            -- 'day', 'week', 'month'
    period_start DATETIME NOT NULL,
    period_end DATETIME NOT NULL,
    project_id INTEGER,                   -- NULLの場合は全プロジェクト

    session_count INTEGER NOT NULL DEFAULT 0,
    total_input_tokens INTEGER NOT NULL DEFAULT 0,
    total_output_tokens INTEGER NOT NULL DEFAULT 0,
    total_cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
    total_cache_read_tokens INTEGER NOT NULL DEFAULT 0,

    model_stats_json TEXT,                -- [{model, tokens}, ...]
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(period_type, period_start, project_id)
);

-- ============================================================
-- Phase 2: インデックス
-- ============================================================

CREATE INDEX IF NOT EXISTS idx_error_patterns_tool ON error_patterns(tool_name);
CREATE INDEX IF NOT EXISTS idx_error_occurrences_pattern ON error_occurrences(pattern_id);
CREATE INDEX IF NOT EXISTS idx_error_occurrences_session ON error_occurrences(session_id);
CREATE INDEX IF NOT EXISTS idx_period_stats_type_start ON period_statistics(period_type, period_start);
CREATE INDEX IF NOT EXISTS idx_period_stats_project ON period_statistics(project_id);
