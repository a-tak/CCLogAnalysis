-- Migration: 004_nullable_git_root.sql
-- Description: project_groups.git_root を NULL可能に変更し、UNIQUE制約を削除
-- SQLiteはALTER TABLE ... MODIFY COLUMNをサポートしないため、テーブルを再作成

-- 1. 新しいテーブル構造で作成
CREATE TABLE project_groups_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    git_root TEXT,  -- NOT NULL制約とUNIQUE制約を削除（NULL可能に）
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 2. 既存データをコピー
INSERT INTO project_groups_new (id, name, git_root, created_at, updated_at)
SELECT id, name, git_root, created_at, updated_at FROM project_groups;

-- 3. 元のテーブルを削除
DROP TABLE project_groups;

-- 4. テーブル名を変更
ALTER TABLE project_groups_new RENAME TO project_groups;

-- 5. トリガーを再作成
CREATE TRIGGER IF NOT EXISTS update_project_groups_timestamp
AFTER UPDATE ON project_groups
BEGIN
    UPDATE project_groups SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- 6. インデックスを再作成（git_rootにはインデックスのみ、UNIQUE制約なし）
CREATE INDEX IF NOT EXISTS idx_project_groups_git_root ON project_groups(git_root);
