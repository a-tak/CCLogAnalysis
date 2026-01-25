-- Migration 005: Add File Modification Time Tracking
-- Purpose: Add file_mod_time to sessions and last_scan_time to projects for incremental scanning

-- ============================================================
-- Step 1: Recreate sessions table with file_mod_time
-- ============================================================

CREATE TABLE IF NOT EXISTS sessions_new (
    id TEXT PRIMARY KEY,
    project_id INTEGER NOT NULL,
    git_branch TEXT NOT NULL,
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    duration_seconds INTEGER NOT NULL,
    total_input_tokens INTEGER NOT NULL DEFAULT 0,
    total_output_tokens INTEGER NOT NULL DEFAULT 0,
    total_cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
    total_cache_read_tokens INTEGER NOT NULL DEFAULT 0,
    error_count INTEGER NOT NULL DEFAULT 0,
    first_user_message TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    file_mod_time TEXT,  -- 追加: ファイルの最終更新日時（RFC3339形式）

    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

-- Step 2: Copy existing data from old sessions table
INSERT INTO sessions_new (
    id, project_id, git_branch, start_time, end_time, duration_seconds,
    total_input_tokens, total_output_tokens, total_cache_creation_tokens, total_cache_read_tokens,
    error_count, first_user_message, created_at, updated_at, file_mod_time
)
SELECT
    id, project_id, git_branch, start_time, end_time, duration_seconds,
    total_input_tokens, total_output_tokens, total_cache_creation_tokens, total_cache_read_tokens,
    error_count, first_user_message, created_at, updated_at, NULL as file_mod_time
FROM sessions;

-- Step 3: Drop old sessions table
DROP TABLE sessions;

-- Step 4: Rename new table
ALTER TABLE sessions_new RENAME TO sessions;

-- Step 5: Recreate sessions indexes
CREATE INDEX IF NOT EXISTS idx_sessions_project_id ON sessions(project_id);
CREATE INDEX IF NOT EXISTS idx_sessions_start_time ON sessions(start_time DESC);
CREATE INDEX IF NOT EXISTS idx_sessions_git_branch ON sessions(git_branch);
CREATE INDEX IF NOT EXISTS idx_sessions_project_start ON sessions(project_id, start_time DESC);
CREATE INDEX IF NOT EXISTS idx_sessions_file_mod_time ON sessions(file_mod_time);  -- 新規インデックス

-- Step 6: Recreate sessions trigger
CREATE TRIGGER IF NOT EXISTS update_sessions_timestamp
AFTER UPDATE ON sessions
BEGIN
    UPDATE sessions SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- ============================================================
-- Step 7: Recreate projects table with last_scan_time
-- ============================================================

CREATE TABLE IF NOT EXISTS projects_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    decoded_path TEXT NOT NULL,
    git_root TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_scan_time TEXT  -- 追加: 最終スキャン日時（RFC3339形式）
);

-- Step 8: Copy existing data from old projects table
INSERT INTO projects_new (
    id, name, decoded_path, git_root, created_at, updated_at, last_scan_time
)
SELECT
    id, name, decoded_path, git_root, created_at, updated_at, NULL as last_scan_time
FROM projects;

-- Step 9: Drop old projects table
DROP TABLE projects;

-- Step 10: Rename new table
ALTER TABLE projects_new RENAME TO projects;

-- Step 11: Recreate projects trigger
CREATE TRIGGER IF NOT EXISTS update_projects_timestamp
AFTER UPDATE ON projects
BEGIN
    UPDATE projects SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- Step 12: Create index for last_scan_time
CREATE INDEX IF NOT EXISTS idx_projects_last_scan_time ON projects(last_scan_time);
