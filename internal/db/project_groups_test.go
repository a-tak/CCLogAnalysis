package db

import (
	"testing"
)

// stringPtr はテスト用のヘルパー関数で、文字列のポインタを返す
func stringPtr(s string) *string {
	return &s
}

func TestCreateProjectGroup(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	t.Run("プロジェクトグループを作成できる", func(t *testing.T) {
		groupID, err := db.CreateProjectGroup("my-repo", stringPtr("/path/to/repo.git"))
		if err != nil {
			t.Fatalf("CreateProjectGroup failed: %v", err)
		}

		if groupID == 0 {
			t.Error("Expected non-zero group ID")
		}

		// 作成されたグループを取得して確認
		group, err := db.GetProjectGroupByID(groupID)
		if err != nil {
			t.Fatalf("GetProjectGroupByID failed: %v", err)
		}

		if group.Name != "my-repo" {
			t.Errorf("Expected name 'my-repo', got '%s'", group.Name)
		}
		if group.GitRoot == nil || *group.GitRoot != "/path/to/repo.git" {
			t.Errorf("Expected git_root '/path/to/repo.git', got '%v'", group.GitRoot)
		}
	})

	t.Run("git_rootがNULLのグループを作成できる", func(t *testing.T) {
		// git_rootがNULLのグループを作成
		groupID, err := db.CreateProjectGroup("no-git-group", nil)
		if err != nil {
			t.Fatalf("CreateProjectGroup failed: %v", err)
		}

		if groupID == 0 {
			t.Error("Expected non-zero group ID")
		}

		// 作成されたグループを取得して確認
		group, err := db.GetProjectGroupByID(groupID)
		if err != nil {
			t.Fatalf("GetProjectGroupByID failed: %v", err)
		}

		if group.Name != "no-git-group" {
			t.Errorf("Expected name 'no-git-group', got '%s'", group.Name)
		}
		if group.GitRoot != nil {
			t.Errorf("Expected git_root to be nil, got '%v'", group.GitRoot)
		}
	})
}

func TestGetProjectGroupByGitRoot(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	t.Run("git_rootからグループを取得できる", func(t *testing.T) {
		// グループを作成
		_, err := db.CreateProjectGroup("test-group", stringPtr("/path/to/test.git"))
		if err != nil {
			t.Fatalf("CreateProjectGroup failed: %v", err)
		}

		// git_rootから取得
		group, err := db.GetProjectGroupByGitRoot("/path/to/test.git")
		if err != nil {
			t.Fatalf("GetProjectGroupByGitRoot failed: %v", err)
		}

		if group.Name != "test-group" {
			t.Errorf("Expected name 'test-group', got '%s'", group.Name)
		}
	})

	t.Run("存在しないgit_rootでエラーを返す", func(t *testing.T) {
		_, err := db.GetProjectGroupByGitRoot("/nonexistent/path")
		if err == nil {
			t.Error("Expected error for nonexistent git_root, got nil")
		}
	})
}

func TestListProjectGroups(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	t.Run("全グループを一覧取得できる", func(t *testing.T) {
		// 複数のグループを作成
		_, err := db.CreateProjectGroup("group-1", stringPtr("/path/to/repo1.git"))
		if err != nil {
			t.Fatalf("CreateProjectGroup failed: %v", err)
		}
		_, err = db.CreateProjectGroup("group-2", stringPtr("/path/to/repo2.git"))
		if err != nil {
			t.Fatalf("CreateProjectGroup failed: %v", err)
		}

		// 一覧を取得
		groups, err := db.ListProjectGroups()
		if err != nil {
			t.Fatalf("ListProjectGroups failed: %v", err)
		}

		if len(groups) != 2 {
			t.Fatalf("Expected 2 groups, got %d", len(groups))
		}
	})
}

func TestAddProjectToGroup(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	t.Run("プロジェクトをグループに追加できる", func(t *testing.T) {
		// プロジェクトを作成
		projectID, err := db.CreateProject("test-project", "/path/to/project")
		if err != nil {
			t.Fatalf("CreateProject failed: %v", err)
		}

		// グループを作成
		groupID, err := db.CreateProjectGroup("test-group", stringPtr("/path/to/repo.git"))
		if err != nil {
			t.Fatalf("CreateProjectGroup failed: %v", err)
		}

		// プロジェクトをグループに追加
		err = db.AddProjectToGroup(projectID, groupID)
		if err != nil {
			t.Fatalf("AddProjectToGroup failed: %v", err)
		}

		// グループのプロジェクト一覧を取得
		projects, err := db.GetProjectsByGroupID(groupID)
		if err != nil {
			t.Fatalf("GetProjectsByGroupID failed: %v", err)
		}

		if len(projects) != 1 {
			t.Fatalf("Expected 1 project, got %d", len(projects))
		}
		if projects[0].ID != projectID {
			t.Errorf("Expected project ID %d, got %d", projectID, projects[0].ID)
		}
	})

	t.Run("同じプロジェクトを重複追加できない", func(t *testing.T) {
		// プロジェクトを作成
		projectID, err := db.CreateProject("duplicate-project", "/path/to/duplicate")
		if err != nil {
			t.Fatalf("CreateProject failed: %v", err)
		}

		// グループを作成
		groupID, err := db.CreateProjectGroup("duplicate-group", stringPtr("/path/to/dup.git"))
		if err != nil {
			t.Fatalf("CreateProjectGroup failed: %v", err)
		}

		// 1回目の追加
		err = db.AddProjectToGroup(projectID, groupID)
		if err != nil {
			t.Fatalf("First AddProjectToGroup failed: %v", err)
		}

		// 2回目の追加（重複）
		err = db.AddProjectToGroup(projectID, groupID)
		if err == nil {
			t.Error("Expected error for duplicate project-group mapping, got nil")
		}
	})
}

func TestGetGroupWithProjects(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	t.Run("グループと配下のプロジェクトを取得できる", func(t *testing.T) {
		// グループを作成
		groupID, err := db.CreateProjectGroup("my-group", stringPtr("/path/to/repo.git"))
		if err != nil {
			t.Fatalf("CreateProjectGroup failed: %v", err)
		}

		// プロジェクトを作成してグループに追加
		project1ID, err := db.CreateProject("project-1", "/path/to/project1")
		if err != nil {
			t.Fatalf("CreateProject failed: %v", err)
		}
		project2ID, err := db.CreateProject("project-2", "/path/to/project2")
		if err != nil {
			t.Fatalf("CreateProject failed: %v", err)
		}

		err = db.AddProjectToGroup(project1ID, groupID)
		if err != nil {
			t.Fatalf("AddProjectToGroup failed: %v", err)
		}
		err = db.AddProjectToGroup(project2ID, groupID)
		if err != nil {
			t.Fatalf("AddProjectToGroup failed: %v", err)
		}

		// グループとプロジェクトを取得
		group, projects, err := db.GetGroupWithProjects(groupID)
		if err != nil {
			t.Fatalf("GetGroupWithProjects failed: %v", err)
		}

		if group.Name != "my-group" {
			t.Errorf("Expected group name 'my-group', got '%s'", group.Name)
		}

		if len(projects) != 2 {
			t.Fatalf("Expected 2 projects, got %d", len(projects))
		}
	})
}

func TestDeleteProjectGroup(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	t.Run("グループを削除できる", func(t *testing.T) {
		// グループを作成
		groupID, err := db.CreateProjectGroup("delete-group", stringPtr("/path/to/delete.git"))
		if err != nil {
			t.Fatalf("CreateProjectGroup failed: %v", err)
		}

		// プロジェクトを作成してグループに追加
		projectID, err := db.CreateProject("delete-project", "/path/to/delete-project")
		if err != nil {
			t.Fatalf("CreateProject failed: %v", err)
		}
		err = db.AddProjectToGroup(projectID, groupID)
		if err != nil {
			t.Fatalf("AddProjectToGroup failed: %v", err)
		}

		// グループを削除
		err = db.DeleteProjectGroup(groupID)
		if err != nil {
			t.Fatalf("DeleteProjectGroup failed: %v", err)
		}

		// グループが削除されたことを確認
		_, err = db.GetProjectGroupByID(groupID)
		if err == nil {
			t.Error("Expected error for deleted group, got nil")
		}

		// マッピングも削除されたことを確認（CASCADE削除）
		projects, err := db.GetProjectsByGroupID(groupID)
		if err != nil {
			t.Fatalf("GetProjectsByGroupID failed: %v", err)
		}
		if len(projects) != 0 {
			t.Errorf("Expected 0 projects after group deletion, got %d", len(projects))
		}
	})
}

func TestGetStandaloneGroupsInWorktreeGroups(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	t.Run("git_root=NULL かつ ワークツリーグループのメンバー → IDが返る", func(t *testing.T) {
		// Setup:
		// 1. ワークツリーグループ作成
		worktreeGroupID, err := db.CreateProjectGroup("voxment", stringPtr("/path/to/voxment"))
		if err != nil {
			t.Fatalf("CreateProjectGroup failed: %v", err)
		}

		// 2. スタンドアロングループ作成（削除済みワークツリー想定）
		standaloneGroupID, err := db.CreateProjectGroup("voxment-worktrees-deleted", nil)
		if err != nil {
			t.Fatalf("CreateProjectGroup failed: %v", err)
		}

		// 3. プロジェクト作成
		projectID, err := db.CreateProject("deleted-worktree-project", "/path/to/deleted")
		if err != nil {
			t.Fatalf("CreateProject failed: %v", err)
		}

		// 4. プロジェクトを両グループに追加
		err = db.AddProjectToGroup(projectID, worktreeGroupID)
		if err != nil {
			t.Fatalf("AddProjectToGroup (worktree) failed: %v", err)
		}
		err = db.AddProjectToGroup(projectID, standaloneGroupID)
		if err != nil {
			t.Fatalf("AddProjectToGroup (standalone) failed: %v", err)
		}

		// Execute
		hiddenIDs, err := db.GetStandaloneGroupsInWorktreeGroups()

		// Assert
		if err != nil {
			t.Fatalf("GetStandaloneGroupsInWorktreeGroups failed: %v", err)
		}
		if len(hiddenIDs) != 1 {
			t.Errorf("Expected 1 hidden group, got %d", len(hiddenIDs))
		}
		if len(hiddenIDs) > 0 && hiddenIDs[0] != standaloneGroupID {
			t.Errorf("Expected standalone group ID %d, got %d", standaloneGroupID, hiddenIDs[0])
		}
	})

	t.Run("git_root=NULL かつ 独立プロジェクト → IDが返らない", func(t *testing.T) {
		// Setup: スタンドアロングループ（ワークツリーグループのメンバーではない）
		standaloneGroupID, err := db.CreateProjectGroup("independent-project", nil)
		if err != nil {
			t.Fatalf("CreateProjectGroup failed: %v", err)
		}
		projectID, err := db.CreateProject("independent", "/path/to/independent")
		if err != nil {
			t.Fatalf("CreateProject failed: %v", err)
		}
		err = db.AddProjectToGroup(projectID, standaloneGroupID)
		if err != nil {
			t.Fatalf("AddProjectToGroup failed: %v", err)
		}

		// Execute
		hiddenIDs, err := db.GetStandaloneGroupsInWorktreeGroups()

		// Assert
		if err != nil {
			t.Fatalf("GetStandaloneGroupsInWorktreeGroups failed: %v", err)
		}
		// standaloneGroupIDは含まれないはず
		for _, id := range hiddenIDs {
			if id == standaloneGroupID {
				t.Error("Independent project group should not be hidden")
			}
		}
	})

	t.Run("git_root!=NULL のグループ → IDが返らない", func(t *testing.T) {
		// Setup: ワークツリーグループ
		worktreeGroupID, err := db.CreateProjectGroup("worktree-group", stringPtr("/path/to/repo"))
		if err != nil {
			t.Fatalf("CreateProjectGroup failed: %v", err)
		}
		projectID, err := db.CreateProject("project", "/path/to/project")
		if err != nil {
			t.Fatalf("CreateProject failed: %v", err)
		}
		err = db.AddProjectToGroup(projectID, worktreeGroupID)
		if err != nil {
			t.Fatalf("AddProjectToGroup failed: %v", err)
		}

		// Execute
		hiddenIDs, err := db.GetStandaloneGroupsInWorktreeGroups()

		// Assert
		if err != nil {
			t.Fatalf("GetStandaloneGroupsInWorktreeGroups failed: %v", err)
		}
		// worktreeGroupIDは含まれないはず
		for _, id := range hiddenIDs {
			if id == worktreeGroupID {
				t.Error("Worktree group should not be hidden")
			}
		}
	})
}

func TestGetStandaloneGroupsWithWorktreeName(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	t.Run("git_root=NULL かつ 名前にworktreeを含む → IDが返る", func(t *testing.T) {
		// Setup: worktreeを含むスタンドアロングループ
		groupID, err := db.CreateProjectGroup("my-repo-worktrees-feature-123", nil)
		if err != nil {
			t.Fatalf("CreateProjectGroup failed: %v", err)
		}

		// Execute
		hiddenIDs, err := db.GetStandaloneGroupsWithWorktreeName()

		// Assert
		if err != nil {
			t.Fatalf("GetStandaloneGroupsWithWorktreeName failed: %v", err)
		}

		found := false
		for _, id := range hiddenIDs {
			if id == groupID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected worktree group to be in hidden list")
		}
	})

	t.Run("git_root=NULL だが 名前にworktreeを含まない → IDが返らない", func(t *testing.T) {
		// Setup: worktreeを含まないスタンドアロングループ
		groupID, err := db.CreateProjectGroup("independent-project", nil)
		if err != nil {
			t.Fatalf("CreateProjectGroup failed: %v", err)
		}

		// Execute
		hiddenIDs, err := db.GetStandaloneGroupsWithWorktreeName()

		// Assert
		if err != nil {
			t.Fatalf("GetStandaloneGroupsWithWorktreeName failed: %v", err)
		}

		for _, id := range hiddenIDs {
			if id == groupID {
				t.Error("Independent project should not be in hidden list")
			}
		}
	})

	t.Run("git_root!=NULL かつ 名前にworktreeを含む → IDが返らない", func(t *testing.T) {
		// Setup: ワークツリーグループ（名前にworktreeを含むが、git_rootがある）
		groupID, err := db.CreateProjectGroup("my-worktrees-repo", stringPtr("/path/to/repo"))
		if err != nil {
			t.Fatalf("CreateProjectGroup failed: %v", err)
		}

		// Execute
		hiddenIDs, err := db.GetStandaloneGroupsWithWorktreeName()

		// Assert
		if err != nil {
			t.Fatalf("GetStandaloneGroupsWithWorktreeName failed: %v", err)
		}

		for _, id := range hiddenIDs {
			if id == groupID {
				t.Error("Worktree group with git_root should not be in hidden list")
			}
		}
	})
}
