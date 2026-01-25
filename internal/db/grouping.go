package db

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
)

// SyncProjectGroups automatically creates/updates project groups based on git_root
func (db *DB) SyncProjectGroups() error {
	// 全プロジェクトを取得
	projects, err := db.ListProjects()
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	// Git Rootごとにプロジェクトをグループ化
	groupMap := make(map[string][]*ProjectRow)
	standaloneProjects := []*ProjectRow{}

	for _, project := range projects {
		if project.GitRoot != nil && *project.GitRoot != "" {
			// Git Rootがある場合はグループ化
			gitRoot := *project.GitRoot
			groupMap[gitRoot] = append(groupMap[gitRoot], project)
		} else {
			// Git Rootがない場合は独立グループとして扱う
			standaloneProjects = append(standaloneProjects, project)
		}
	}

	// 各Git Rootに対してグループを作成・更新
	for gitRoot, projectsInGroup := range groupMap {
		// グループが既に存在するか確認
		group, err := db.GetProjectGroupByGitRoot(gitRoot)
		if err != nil {
			// グループが存在しない場合は新規作成
			groupName := generateGroupName(gitRoot)
			gitRootPtr := &gitRoot
			groupID, err := db.CreateProjectGroup(groupName, gitRootPtr)
			if err != nil {
				log.Printf("Warning: failed to create project group for git_root %s: %v", gitRoot, err)
				continue
			}

			// 新規作成したグループを取得
			group, err = db.GetProjectGroupByID(groupID)
			if err != nil {
				log.Printf("Warning: failed to get created project group: %v", err)
				continue
			}
		}

		// グループにプロジェクトを追加（重複は無視）
		for _, project := range projectsInGroup {
			err := db.AddProjectToGroup(project.ID, group.ID)
			if err != nil {
				// 重複エラーは無視（既に追加済み）
				if !strings.Contains(err.Error(), "UNIQUE constraint failed") {
					log.Printf("Warning: failed to add project %s to group %s: %v", project.Name, group.Name, err)
				}
			}
		}
	}

	// Git Rootがないプロジェクトを独立グループとして作成
	for _, project := range standaloneProjects {
		groupName := project.Name
		// グループが既に存在するか確認
		group, err := db.GetProjectGroupByName(groupName)
		if err != nil {
			// グループが存在しない場合は新規作成（git_root = NULL）
			groupID, err := db.CreateProjectGroup(groupName, nil)
			if err != nil {
				log.Printf("Warning: failed to create standalone group for project %s: %v", project.Name, err)
				continue
			}

			// 新規作成したグループを取得
			group, err = db.GetProjectGroupByID(groupID)
			if err != nil {
				log.Printf("Warning: failed to get created standalone group: %v", err)
				continue
			}
		}

		// プロジェクトをグループに追加（重複は無視）
		err = db.AddProjectToGroup(project.ID, group.ID)
		if err != nil {
			// 重複エラーは無視（既に追加済み）
			if !strings.Contains(err.Error(), "UNIQUE constraint failed") {
				log.Printf("Warning: failed to add project %s to standalone group: %v", project.Name, err)
			}
		}
	}

	return nil
}

// generateGroupName generates a group name from a git root path
// Example: "/path/to/repo.git" -> "repo"
func generateGroupName(gitRoot string) string {
	if gitRoot == "" {
		return "unknown"
	}

	// パスの最後の要素を取得
	baseName := filepath.Base(gitRoot)

	// .gitサフィックスを除去
	if strings.HasSuffix(baseName, ".git") {
		baseName = strings.TrimSuffix(baseName, ".git")
	}

	// 空の場合は"unknown"を返す
	if baseName == "" || baseName == "." || baseName == "/" {
		return "unknown"
	}

	return baseName
}
