package db

import (
	"database/sql"
	"fmt"
	"time"
)

// ProjectRow represents a row in the projects table
type ProjectRow struct {
	ID          int64
	Name        string
	DecodedPath string
	GitRoot     *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CreateProject creates a new project without git root
func (db *DB) CreateProject(name, decodedPath string) (int64, error) {
	if name == "" {
		return 0, fmt.Errorf("project name cannot be empty")
	}
	if decodedPath == "" {
		return 0, fmt.Errorf("decoded path cannot be empty")
	}

	query := `
		INSERT INTO projects (name, decoded_path)
		VALUES (?, ?)
	`
	result, err := db.conn.Exec(query, name, decodedPath)
	if err != nil {
		return 0, fmt.Errorf("failed to insert project: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return id, nil
}

// CreateProjectWithGitRoot creates a new project with git root
func (db *DB) CreateProjectWithGitRoot(name, decodedPath, gitRoot string) (int64, error) {
	if name == "" {
		return 0, fmt.Errorf("project name cannot be empty")
	}
	if decodedPath == "" {
		return 0, fmt.Errorf("decoded path cannot be empty")
	}

	query := `
		INSERT INTO projects (name, decoded_path, git_root)
		VALUES (?, ?, ?)
	`
	result, err := db.conn.Exec(query, name, decodedPath, gitRoot)
	if err != nil {
		return 0, fmt.Errorf("failed to insert project: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return id, nil
}

// GetProjectByName retrieves a project by name
func (db *DB) GetProjectByName(name string) (*ProjectRow, error) {
	query := `
		SELECT id, name, decoded_path, git_root, created_at, updated_at
		FROM projects
		WHERE name = ?
	`
	var project ProjectRow
	err := db.conn.QueryRow(query, name).Scan(
		&project.ID,
		&project.Name,
		&project.DecodedPath,
		&project.GitRoot,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query project: %w", err)
	}

	return &project, nil
}

// GetProjectByID retrieves a project by ID
func (db *DB) GetProjectByID(id int64) (*ProjectRow, error) {
	query := `
		SELECT id, name, decoded_path, git_root, created_at, updated_at
		FROM projects
		WHERE id = ?
	`
	var project ProjectRow
	err := db.conn.QueryRow(query, id).Scan(
		&project.ID,
		&project.Name,
		&project.DecodedPath,
		&project.GitRoot,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project not found: id=%d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query project: %w", err)
	}

	return &project, nil
}

// ListProjects retrieves all projects ordered by name
func (db *DB) ListProjects() ([]*ProjectRow, error) {
	query := `
		SELECT id, name, decoded_path, git_root, created_at, updated_at
		FROM projects
		ORDER BY name
	`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query projects: %w", err)
	}
	defer rows.Close()

	var projects []*ProjectRow
	for rows.Next() {
		var project ProjectRow
		err := rows.Scan(
			&project.ID,
			&project.Name,
			&project.DecodedPath,
			&project.GitRoot,
			&project.CreatedAt,
			&project.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project row: %w", err)
		}
		projects = append(projects, &project)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating project rows: %w", err)
	}

	return projects, nil
}

// UpdateProjectGitRoot updates the git root of a project
func (db *DB) UpdateProjectGitRoot(id int64, gitRoot string) error {
	query := `
		UPDATE projects
		SET git_root = ?
		WHERE id = ?
	`
	_, err := db.conn.Exec(query, gitRoot, id)
	if err != nil {
		return fmt.Errorf("failed to update project git root: %w", err)
	}

	return nil
}

// DeleteProject deletes a project by ID
func (db *DB) DeleteProject(id int64) error {
	query := `DELETE FROM projects WHERE id = ?`
	_, err := db.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	return nil
}

// UpdateProjectLastScanTime updates the last scan time for a project
func (db *DB) UpdateProjectLastScanTime(projectID int64, scanTime time.Time) error {
	query := `UPDATE projects SET last_scan_time = ? WHERE id = ?`
	_, err := db.conn.Exec(query, scanTime.Format(time.RFC3339), projectID)
	if err != nil {
		return fmt.Errorf("failed to update last scan time: %w", err)
	}
	return nil
}

// GetProjectLastScanTime returns the last scan time for a project
func (db *DB) GetProjectLastScanTime(projectID int64) (*time.Time, error) {
	var scanTimeStr sql.NullString
	query := `SELECT last_scan_time FROM projects WHERE id = ?`
	err := db.conn.QueryRow(query, projectID).Scan(&scanTimeStr)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project not found: id=%d", projectID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get last scan time: %w", err)
	}

	if !scanTimeStr.Valid {
		return nil, nil
	}

	scanTime, err := time.Parse(time.RFC3339, scanTimeStr.String)
	if err != nil {
		return nil, fmt.Errorf("failed to parse last scan time: %w", err)
	}

	return &scanTime, nil
}

// GetProjectWorkingDirectory returns the working directory from the most recent session
func (db *DB) GetProjectWorkingDirectory(projectID int64) (string, error) {
	// 最新セッションのIDを取得
	var sessionID string
	sessionQuery := `
		SELECT id
		FROM sessions
		WHERE project_id = ?
		ORDER BY start_time DESC
		LIMIT 1
	`
	err := db.conn.QueryRow(sessionQuery, projectID).Scan(&sessionID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("no sessions found for project")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get latest session: %w", err)
	}

	// 最新セッションのログエントリからcwdを取得
	var cwd string
	cwdQuery := `
		SELECT cwd
		FROM log_entries
		WHERE session_id = ? AND cwd IS NOT NULL AND cwd != ''
		LIMIT 1
	`
	err = db.conn.QueryRow(cwdQuery, sessionID).Scan(&cwd)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("no cwd found in session")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get cwd: %w", err)
	}

	return cwd, nil
}
