package db

import (
	"testing"
)

func TestSyncProjectGroups(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	t.Run("同じGit Rootのプロジェクトが自動的にグループ化される", func(t *testing.T) {
		// 同じGit Rootを持つ複数のプロジェクトを作成
		gitRoot := "/path/to/repo.git"
		_, err := db.CreateProjectWithGitRoot("project-1", "/path/to/project1", gitRoot)
		if err != nil {
			t.Fatalf("CreateProjectWithGitRoot failed: %v", err)
		}
		_, err = db.CreateProjectWithGitRoot("project-2", "/path/to/project2", gitRoot)
		if err != nil {
			t.Fatalf("CreateProjectWithGitRoot failed: %v", err)
		}
		_, err = db.CreateProjectWithGitRoot("project-3", "/path/to/project3", gitRoot)
		if err != nil {
			t.Fatalf("CreateProjectWithGitRoot failed: %v", err)
		}

		// 別のGit Rootのプロジェクトも作成
		anotherGitRoot := "/path/to/another.git"
		_, err = db.CreateProjectWithGitRoot("another-project", "/path/to/another", anotherGitRoot)
		if err != nil {
			t.Fatalf("CreateProjectWithGitRoot failed: %v", err)
		}

		// 自動グループ化を実行
		err = db.SyncProjectGroups()
		if err != nil {
			t.Fatalf("SyncProjectGroups failed: %v", err)
		}

		// グループが2つ作成されたことを確認
		groups, err := db.ListProjectGroups()
		if err != nil {
			t.Fatalf("ListProjectGroups failed: %v", err)
		}
		if len(groups) != 2 {
			t.Fatalf("Expected 2 groups, got %d", len(groups))
		}

		// 最初のGit Rootのグループを確認
		group1, err := db.GetProjectGroupByGitRoot(gitRoot)
		if err != nil {
			t.Fatalf("GetProjectGroupByGitRoot failed: %v", err)
		}

		// グループに3つのプロジェクトが含まれることを確認
		projects, err := db.GetProjectsByGroupID(group1.ID)
		if err != nil {
			t.Fatalf("GetProjectsByGroupID failed: %v", err)
		}
		if len(projects) != 3 {
			t.Errorf("Expected 3 projects in group, got %d", len(projects))
		}

		// 2つ目のGit Rootのグループを確認
		group2, err := db.GetProjectGroupByGitRoot(anotherGitRoot)
		if err != nil {
			t.Fatalf("GetProjectGroupByGitRoot failed: %v", err)
		}

		// グループに1つのプロジェクトが含まれることを確認
		projects2, err := db.GetProjectsByGroupID(group2.ID)
		if err != nil {
			t.Fatalf("GetProjectsByGroupID failed: %v", err)
		}
		if len(projects2) != 1 {
			t.Errorf("Expected 1 project in group, got %d", len(projects2))
		}
	})

	t.Run("Git Rootがnullのプロジェクトはグループ化されない", func(t *testing.T) {
		newDB, _ := setupTestDB(t)
		defer newDB.Close()

		// Git Root未設定のプロジェクトを作成
		_, err := newDB.CreateProject("no-git-project", "/path/to/no-git")
		if err != nil {
			t.Fatalf("CreateProject failed: %v", err)
		}

		// 自動グループ化を実行
		err = newDB.SyncProjectGroups()
		if err != nil {
			t.Fatalf("SyncProjectGroups failed: %v", err)
		}

		// グループが作成されないことを確認
		groups, err := newDB.ListProjectGroups()
		if err != nil {
			t.Fatalf("ListProjectGroups failed: %v", err)
		}
		if len(groups) != 0 {
			t.Errorf("Expected 0 groups, got %d", len(groups))
		}
	})

	t.Run("既存グループに新しいプロジェクトが追加される", func(t *testing.T) {
		newDB, _ := setupTestDB(t)
		defer newDB.Close()

		gitRoot := "/path/to/existing.git"

		// 最初のプロジェクトを作成してグループ化
		_, err := newDB.CreateProjectWithGitRoot("first-project", "/path/to/first", gitRoot)
		if err != nil {
			t.Fatalf("CreateProjectWithGitRoot failed: %v", err)
		}

		err = newDB.SyncProjectGroups()
		if err != nil {
			t.Fatalf("First SyncProjectGroups failed: %v", err)
		}

		// グループを取得
		group, err := newDB.GetProjectGroupByGitRoot(gitRoot)
		if err != nil {
			t.Fatalf("GetProjectGroupByGitRoot failed: %v", err)
		}

		// 新しいプロジェクトを追加
		_, err = newDB.CreateProjectWithGitRoot("second-project", "/path/to/second", gitRoot)
		if err != nil {
			t.Fatalf("CreateProjectWithGitRoot failed: %v", err)
		}

		// 再度グループ化を実行
		err = newDB.SyncProjectGroups()
		if err != nil {
			t.Fatalf("Second SyncProjectGroups failed: %v", err)
		}

		// 同じグループに2つのプロジェクトが含まれることを確認
		projects, err := newDB.GetProjectsByGroupID(group.ID)
		if err != nil {
			t.Fatalf("GetProjectsByGroupID failed: %v", err)
		}
		if len(projects) != 2 {
			t.Errorf("Expected 2 projects in group, got %d", len(projects))
		}

		// グループが1つだけであることを確認（重複作成されない）
		groups, err := newDB.ListProjectGroups()
		if err != nil {
			t.Fatalf("ListProjectGroups failed: %v", err)
		}
		if len(groups) != 1 {
			t.Errorf("Expected 1 group, got %d", len(groups))
		}
	})

	t.Run("重複マッピングが作成されない", func(t *testing.T) {
		newDB, _ := setupTestDB(t)
		defer newDB.Close()

		gitRoot := "/path/to/duplicate-check.git"

		// プロジェクトを作成してグループ化
		_, err := newDB.CreateProjectWithGitRoot("dup-project", "/path/to/dup", gitRoot)
		if err != nil {
			t.Fatalf("CreateProjectWithGitRoot failed: %v", err)
		}

		err = newDB.SyncProjectGroups()
		if err != nil {
			t.Fatalf("First SyncProjectGroups failed: %v", err)
		}

		// 同じプロジェクトで再度グループ化を実行
		err = newDB.SyncProjectGroups()
		if err != nil {
			t.Fatalf("Second SyncProjectGroups failed: %v", err)
		}

		// グループを取得
		group, err := newDB.GetProjectGroupByGitRoot(gitRoot)
		if err != nil {
			t.Fatalf("GetProjectGroupByGitRoot failed: %v", err)
		}

		// プロジェクトが1つだけであることを確認
		projects, err := newDB.GetProjectsByGroupID(group.ID)
		if err != nil {
			t.Fatalf("GetProjectsByGroupID failed: %v", err)
		}
		if len(projects) != 1 {
			t.Errorf("Expected 1 project in group (no duplicates), got %d", len(projects))
		}
	})
}

func TestGenerateGroupName(t *testing.T) {
	tests := []struct {
		name         string
		gitRoot      string
		expectedName string
	}{
		{
			name:         "標準的な.gitパス",
			gitRoot:      "/path/to/repo.git",
			expectedName: "repo",
		},
		{
			name:         ".gitなしのパス",
			gitRoot:      "/path/to/myproject",
			expectedName: "myproject",
		},
		{
			name:         "ルートディレクトリ",
			gitRoot:      "/repo.git",
			expectedName: "repo",
		},
		{
			name:         "複雑なパス",
			gitRoot:      "/home/user/projects/my-awesome-project.git",
			expectedName: "my-awesome-project",
		},
		{
			name:         "空のパス",
			gitRoot:      "",
			expectedName: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateGroupName(tt.gitRoot)
			if result != tt.expectedName {
				t.Errorf("Expected group name '%s', got '%s'", tt.expectedName, result)
			}
		})
	}
}
