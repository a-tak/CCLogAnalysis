-- Migration 003: Project Groups Schema Update
-- Purpose: Add UNIQUE constraint to git_root and updated_at column to project_groups table

-- Step 1: Create new project_groups table with UNIQUE constraint and updated_at
CREATE TABLE IF NOT EXISTS project_groups_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    git_root TEXT NOT NULL UNIQUE,  -- Add UNIQUE constraint
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP  -- Add updated_at column
);

-- Step 2: Copy existing data from old table (if any)
INSERT INTO project_groups_new (id, name, git_root, created_at, updated_at)
SELECT id, name, git_root, created_at, created_at as updated_at
FROM project_groups;

-- Step 3: Drop old table
DROP TABLE IF EXISTS project_groups;

-- Step 4: Rename new table
ALTER TABLE project_groups_new RENAME TO project_groups;

-- Step 5: Recreate project_group_mappings table (foreign key references)
-- Note: project_group_mappings already has correct schema in schema.sql

-- Step 6: Add updated_at trigger for project_groups
CREATE TRIGGER IF NOT EXISTS update_project_groups_timestamp
AFTER UPDATE ON project_groups
BEGIN
    UPDATE project_groups SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- Step 7: Create indexes
CREATE INDEX IF NOT EXISTS idx_project_group_mappings_group ON project_group_mappings(group_id);
CREATE INDEX IF NOT EXISTS idx_project_groups_git_root ON project_groups(git_root);
