package db

import (
	"database/sql"
	"fmt"
	"time"
)

// ProjectGroupRow represents a row in the project_groups table
type ProjectGroupRow struct {
	ID        int64
	Name      string
	GitRoot   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CreateProjectGroup creates a new project group
func (db *DB) CreateProjectGroup(name, gitRoot string) (int64, error) {
	if name == "" {
		return 0, fmt.Errorf("group name cannot be empty")
	}
	if gitRoot == "" {
		return 0, fmt.Errorf("git root cannot be empty")
	}

	query := `
		INSERT INTO project_groups (name, git_root)
		VALUES (?, ?)
	`
	result, err := db.conn.Exec(query, name, gitRoot)
	if err != nil {
		return 0, fmt.Errorf("failed to insert project group: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return id, nil
}

// GetProjectGroupByGitRoot retrieves a project group by git root
func (db *DB) GetProjectGroupByGitRoot(gitRoot string) (*ProjectGroupRow, error) {
	query := `
		SELECT id, name, git_root, created_at, updated_at
		FROM project_groups
		WHERE git_root = ?
	`
	var group ProjectGroupRow
	err := db.conn.QueryRow(query, gitRoot).Scan(
		&group.ID,
		&group.Name,
		&group.GitRoot,
		&group.CreatedAt,
		&group.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project group not found for git_root: %s", gitRoot)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query project group: %w", err)
	}

	return &group, nil
}

// GetProjectGroupByID retrieves a project group by ID
func (db *DB) GetProjectGroupByID(id int64) (*ProjectGroupRow, error) {
	query := `
		SELECT id, name, git_root, created_at, updated_at
		FROM project_groups
		WHERE id = ?
	`
	var group ProjectGroupRow
	err := db.conn.QueryRow(query, id).Scan(
		&group.ID,
		&group.Name,
		&group.GitRoot,
		&group.CreatedAt,
		&group.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project group not found: id=%d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query project group: %w", err)
	}

	return &group, nil
}

// ListProjectGroups retrieves all project groups
func (db *DB) ListProjectGroups() ([]*ProjectGroupRow, error) {
	query := `
		SELECT id, name, git_root, created_at, updated_at
		FROM project_groups
		ORDER BY name
	`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query project groups: %w", err)
	}
	defer rows.Close()

	var groups []*ProjectGroupRow
	for rows.Next() {
		var group ProjectGroupRow
		err := rows.Scan(
			&group.ID,
			&group.Name,
			&group.GitRoot,
			&group.CreatedAt,
			&group.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project group row: %w", err)
		}
		groups = append(groups, &group)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating project group rows: %w", err)
	}

	return groups, nil
}

// AddProjectToGroup adds a project to a group
func (db *DB) AddProjectToGroup(projectID, groupID int64) error {
	query := `
		INSERT INTO project_group_mappings (project_id, group_id)
		VALUES (?, ?)
	`
	_, err := db.conn.Exec(query, projectID, groupID)
	if err != nil {
		return fmt.Errorf("failed to add project to group: %w", err)
	}

	return nil
}

// GetProjectsByGroupID retrieves all projects in a group
func (db *DB) GetProjectsByGroupID(groupID int64) ([]*ProjectRow, error) {
	query := `
		SELECT p.id, p.name, p.decoded_path, p.git_root, p.created_at, p.updated_at
		FROM projects p
		INNER JOIN project_group_mappings pgm ON p.id = pgm.project_id
		WHERE pgm.group_id = ?
		ORDER BY p.name
	`
	rows, err := db.conn.Query(query, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to query projects by group: %w", err)
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

// GetGroupWithProjects retrieves a group and its member projects
func (db *DB) GetGroupWithProjects(groupID int64) (*ProjectGroupRow, []*ProjectRow, error) {
	// グループを取得
	group, err := db.GetProjectGroupByID(groupID)
	if err != nil {
		return nil, nil, err
	}

	// プロジェクト一覧を取得
	projects, err := db.GetProjectsByGroupID(groupID)
	if err != nil {
		return nil, nil, err
	}

	return group, projects, nil
}

// DeleteProjectGroup deletes a project group (CASCADE deletes mappings)
func (db *DB) DeleteProjectGroup(id int64) error {
	query := `DELETE FROM project_groups WHERE id = ?`
	_, err := db.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete project group: %w", err)
	}

	return nil
}
