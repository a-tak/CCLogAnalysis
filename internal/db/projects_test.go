package db

import (
	"testing"
	"time"
)

func TestCreateProject(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	t.Run("正常にプロジェクトを作成できる", func(t *testing.T) {
		name := "my-project"
		decodedPath := "/path/to/my-project"

		id, err := db.CreateProject(name, decodedPath)
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		if id <= 0 {
			t.Error("Expected positive project ID")
		}
	})

	t.Run("gitRootを含むプロジェクトを作成できる", func(t *testing.T) {
		name := "project-with-git"
		decodedPath := "/path/to/project-with-git"
		gitRoot := "/path/to/project-with-git/.git"

		id, err := db.CreateProjectWithGitRoot(name, decodedPath, gitRoot)
		if err != nil {
			t.Fatalf("Failed to create project with git root: %v", err)
		}

		if id <= 0 {
			t.Error("Expected positive project ID")
		}

		// 作成したプロジェクトを取得して確認
		project, err := db.GetProjectByName(name)
		if err != nil {
			t.Fatalf("Failed to get project: %v", err)
		}

		if project.GitRoot == nil {
			t.Error("Expected git_root to be set")
		} else if *project.GitRoot != gitRoot {
			t.Errorf("Expected git_root=%s, got %s", gitRoot, *project.GitRoot)
		}
	})

	t.Run("同じ名前のプロジェクトは作成できない", func(t *testing.T) {
		name := "duplicate-project"
		decodedPath := "/path/to/duplicate-project"

		// 1回目の作成
		_, err := db.CreateProject(name, decodedPath)
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		// 2回目の作成（重複）
		_, err = db.CreateProject(name, decodedPath)
		if err == nil {
			t.Error("Expected error for duplicate project name, got nil")
		}
	})

	t.Run("空の名前でエラーを返す", func(t *testing.T) {
		_, err := db.CreateProject("", "/path/to/project")
		if err == nil {
			t.Error("Expected error for empty project name, got nil")
		}
	})

	t.Run("空のパスでエラーを返す", func(t *testing.T) {
		_, err := db.CreateProject("test-project", "")
		if err == nil {
			t.Error("Expected error for empty decoded path, got nil")
		}
	})
}

func TestGetProjectByName(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	t.Run("正常にプロジェクトを取得できる", func(t *testing.T) {
		name := "get-test-project"
		decodedPath := "/path/to/get-test-project"

		// プロジェクト作成
		createdID, err := db.CreateProject(name, decodedPath)
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		// プロジェクト取得
		project, err := db.GetProjectByName(name)
		if err != nil {
			t.Fatalf("Failed to get project: %v", err)
		}

		if project.ID != createdID {
			t.Errorf("Expected ID=%d, got %d", createdID, project.ID)
		}
		if project.Name != name {
			t.Errorf("Expected Name=%s, got %s", name, project.Name)
		}
		if project.DecodedPath != decodedPath {
			t.Errorf("Expected DecodedPath=%s, got %s", decodedPath, project.DecodedPath)
		}
		if project.GitRoot != nil {
			t.Errorf("Expected GitRoot=nil, got %s", *project.GitRoot)
		}
		if project.CreatedAt.IsZero() {
			t.Error("Expected CreatedAt to be set")
		}
		if project.UpdatedAt.IsZero() {
			t.Error("Expected UpdatedAt to be set")
		}
	})

	t.Run("存在しないプロジェクトでエラーを返す", func(t *testing.T) {
		_, err := db.GetProjectByName("non-existent-project")
		if err == nil {
			t.Error("Expected error for non-existent project, got nil")
		}
	})
}

func TestGetProjectByID(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	t.Run("正常にプロジェクトを取得できる", func(t *testing.T) {
		name := "id-test-project"
		decodedPath := "/path/to/id-test-project"

		// プロジェクト作成
		createdID, err := db.CreateProject(name, decodedPath)
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		// プロジェクト取得
		project, err := db.GetProjectByID(createdID)
		if err != nil {
			t.Fatalf("Failed to get project: %v", err)
		}

		if project.ID != createdID {
			t.Errorf("Expected ID=%d, got %d", createdID, project.ID)
		}
		if project.Name != name {
			t.Errorf("Expected Name=%s, got %s", name, project.Name)
		}
	})

	t.Run("存在しないIDでエラーを返す", func(t *testing.T) {
		_, err := db.GetProjectByID(99999)
		if err == nil {
			t.Error("Expected error for non-existent project ID, got nil")
		}
	})
}

func TestListProjects(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	t.Run("空のリストを返す", func(t *testing.T) {
		projects, err := db.ListProjects()
		if err != nil {
			t.Fatalf("Failed to list projects: %v", err)
		}

		if len(projects) != 0 {
			t.Errorf("Expected empty list, got %d projects", len(projects))
		}
	})

	t.Run("複数のプロジェクトを取得できる", func(t *testing.T) {
		// プロジェクト作成
		projects := []struct {
			name        string
			decodedPath string
		}{
			{"project-a", "/path/to/project-a"},
			{"project-b", "/path/to/project-b"},
			{"project-c", "/path/to/project-c"},
		}

		for _, p := range projects {
			_, err := db.CreateProject(p.name, p.decodedPath)
			if err != nil {
				t.Fatalf("Failed to create project %s: %v", p.name, err)
			}
		}

		// プロジェクト一覧取得
		result, err := db.ListProjects()
		if err != nil {
			t.Fatalf("Failed to list projects: %v", err)
		}

		if len(result) != len(projects) {
			t.Errorf("Expected %d projects, got %d", len(projects), len(result))
		}

		// 名前でソートされていることを確認
		for i, p := range result {
			if p.Name != projects[i].name {
				t.Errorf("Expected project[%d].Name=%s, got %s", i, projects[i].name, p.Name)
			}
		}
	})
}

func TestUpdateProject(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	t.Run("プロジェクトを更新できる", func(t *testing.T) {
		name := "update-test-project"
		decodedPath := "/path/to/update-test-project"

		// プロジェクト作成
		id, err := db.CreateProject(name, decodedPath)
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		// 最初のupdated_atを記録
		project1, err := db.GetProjectByID(id)
		if err != nil {
			t.Fatalf("Failed to get project: %v", err)
		}
		firstUpdatedAt := project1.UpdatedAt

		// 少し待ってから更新（タイムスタンプの差を確認するため）
		// SQLiteのCURRENT_TIMESTAMPは秒単位の精度なので1秒以上待つ
		time.Sleep(1100 * time.Millisecond)

		// GitRoot更新
		newGitRoot := "/new/git/root"
		err = db.UpdateProjectGitRoot(id, newGitRoot)
		if err != nil {
			t.Fatalf("Failed to update project: %v", err)
		}

		// 更新されたプロジェクトを取得
		project2, err := db.GetProjectByID(id)
		if err != nil {
			t.Fatalf("Failed to get updated project: %v", err)
		}

		if project2.GitRoot == nil {
			t.Error("Expected git_root to be set")
		} else if *project2.GitRoot != newGitRoot {
			t.Errorf("Expected git_root=%s, got %s", newGitRoot, *project2.GitRoot)
		}

		// updated_atが更新されていることを確認
		if !project2.UpdatedAt.After(firstUpdatedAt) {
			t.Errorf("Expected updated_at to be updated, was %v, now %v", firstUpdatedAt, project2.UpdatedAt)
		}
	})
}

func TestDeleteProject(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	t.Run("プロジェクトを削除できる", func(t *testing.T) {
		name := "delete-test-project"
		decodedPath := "/path/to/delete-test-project"

		// プロジェクト作成
		id, err := db.CreateProject(name, decodedPath)
		if err != nil {
			t.Fatalf("Failed to create project: %v", err)
		}

		// プロジェクト削除
		err = db.DeleteProject(id)
		if err != nil {
			t.Fatalf("Failed to delete project: %v", err)
		}

		// 削除されたことを確認
		_, err = db.GetProjectByID(id)
		if err == nil {
			t.Error("Expected error when getting deleted project, got nil")
		}
	})

	t.Run("存在しないプロジェクトの削除でエラーにならない", func(t *testing.T) {
		err := db.DeleteProject(99999)
		// 存在しないIDの削除はエラーにならない（影響行数0）
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}
