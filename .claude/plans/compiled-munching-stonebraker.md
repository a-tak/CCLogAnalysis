# プラン: グループ一覧のフィルタリング実装

**日付**: 2026-01-25
**ブランチ**: project-summary

## 背景

Git Worktreeプロジェクトのグルーピング機能において、以下の問題が発生している：

- 削除されたワークツリー（ディレクトリが存在しない）は、git_root検出ができず `git_root=NULL` のスタンドアロングループとして作成される
- これらのグループは、実際には別のワークツリーグループ（`git_root != NULL`）のメンバーになっているにもかかわらず、グループ一覧に表示されてしまう
- 結果として、グループ一覧に100件ものグループが表示され、UIが見づらい

**例**：
- voxmentグループ（`git_root={git-root-path}`）が存在
- voxmentグループには7つのメンバープロジェクトが含まれる（voxment本体 + 現存する6つのワークツリー）
- しかし、削除済みワークツリー（90件以上）も個別グループとして一覧に表示されている

## 要件

グループ一覧表示時に、以下のルールでフィルタリングする：

1. **`git_root != NULL` のグループ**: 常に表示（ワークツリーグループ）
2. **`git_root = NULL` かつ、どのワークツリーグループのメンバーにもなっていない**: 表示（独立プロジェクト）
3. **`git_root = NULL` かつ、ワークツリーグループのメンバーになっている**: **非表示**（削除されたワークツリー）

**重要**: 非表示にするのはグループ一覧のみ。各グループの詳細ページでは、削除済みワークツリーも引き続きメンバーとして表示される。

## 実装アプローチ

### 選択した手法: SQLベースのフィルタリング + Goでの軽量処理

**理由**:
- データ件数が多い（100件以上）ため、SQLでフィルタリングする方が効率的
- 既存コードパターン（JOIN多用）との一貫性
- テスタビリティ: DBレイヤーとAPIレイヤーで責務が明確に分離

### アーキテクチャ

```
┌─────────────────────────────────────────────────────┐
│ API Layer (service_db.go)                          │
│  ListProjectGroups()                                │
│  ├─ db.ListProjectGroups()          ← 全グループ取得│
│  ├─ db.GetStandaloneGroupsInWorktreeGroups()       │
│  │    ↑ 除外対象のグループID取得                    │
│  └─ フィルタリング処理 (mapで除外判定)              │
└─────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────┐
│ DB Layer (project_groups.go)                       │
│  GetStandaloneGroupsInWorktreeGroups()             │
│  ↑ SQL JOIN で除外対象を特定                        │
└─────────────────────────────────────────────────────┘
```

## 実装ステップ

### Step 1: DBレイヤー - 新メソッド追加

**ファイル**: `internal/db/project_groups.go`

**追加内容**: `GetStandaloneGroupsInWorktreeGroups()` メソッド

```go
// GetStandaloneGroupsInWorktreeGroups returns IDs of standalone groups
// that are members of worktree groups (should be hidden in group list)
func (db *DB) GetStandaloneGroupsInWorktreeGroups() ([]int64, error) {
    query := `
        SELECT DISTINCT pg.id
        FROM project_groups pg
        INNER JOIN project_group_mappings pgm ON pg.id = pgm.group_id
        INNER JOIN projects p ON pgm.project_id = p.id
        INNER JOIN project_group_mappings pgm2 ON p.id = pgm2.project_id
        INNER JOIN project_groups pg2 ON pgm2.group_id = pg2.id
        WHERE pg.git_root IS NULL
          AND pg2.git_root IS NOT NULL
    `

    rows, err := db.conn.Query(query)
    if err != nil {
        return nil, fmt.Errorf("failed to query standalone groups in worktree groups: %w", err)
    }
    defer rows.Close()

    var ids []int64
    for rows.Next() {
        var id int64
        if err := rows.Scan(&id); err != nil {
            return nil, fmt.Errorf("failed to scan group id: %w", err)
        }
        ids = append(ids, id)
    }

    if err = rows.Err(); err != nil {
        return nil, fmt.Errorf("error iterating group id rows: %w", err)
    }

    return ids, nil
}
```

**SQLロジック解説**:
- `pg`: スタンドアロングループ（`git_root IS NULL`）
- `pgm`: そのグループのメンバーとなるプロジェクトのマッピング
- `p`: プロジェクト本体
- `pgm2`: そのプロジェクトが属する別グループのマッピング
- `pg2`: 別グループ（ワークツリーグループ、`git_root IS NOT NULL`）

### Step 2: APIレイヤー - フィルタリング実装

**ファイル**: `internal/api/service_db.go`

**修正対象**: `ListProjectGroups()` メソッド（行337-371）

**修正内容**:

```go
func (s *DatabaseSessionService) ListProjectGroups() ([]ProjectGroupResponse, error) {
    // 1. 全グループを取得
    groupRows, err := s.db.ListProjectGroups()
    if err != nil {
        return nil, fmt.Errorf("failed to list project groups: %w", err)
    }

    // 2. 除外対象のグループIDを取得
    hiddenGroupIDs, err := s.db.GetStandaloneGroupsInWorktreeGroups()
    if err != nil {
        return nil, fmt.Errorf("failed to get hidden group ids: %w", err)
    }

    // 3. 除外対象をmapに変換（O(1)検索用）
    hiddenMap := make(map[int64]bool)
    for _, id := range hiddenGroupIDs {
        hiddenMap[id] = true
    }

    // 4. フィルタリング
    groups := make([]ProjectGroupResponse, 0, len(groupRows))
    for _, row := range groupRows {
        // 除外対象でない場合のみ追加
        if !hiddenMap[row.ID] {
            groups = append(groups, ProjectGroupResponse{
                ID:        row.ID,
                Name:      row.Name,
                GitRoot:   row.GitRoot,
                CreatedAt: row.CreatedAt,
                UpdatedAt: row.UpdatedAt,
            })
        }
    }

    // 5. ソート（既存ロジック - 変更なし）
    sort.Slice(groups, func(i, j int) bool {
        iHasGitRoot := groups[i].GitRoot != nil
        jHasGitRoot := groups[j].GitRoot != nil

        if iHasGitRoot != jHasGitRoot {
            return iHasGitRoot
        }

        return groups[i].Name < groups[j].Name
    })

    return groups, nil
}
```

### Step 3: テスト実装

**ファイル**: `internal/db/project_groups_test.go`

**追加テスト**:

```go
func TestGetStandaloneGroupsInWorktreeGroups(t *testing.T) {
    db, _ := setupTestDB(t)
    defer db.Close()

    t.Run("git_root=NULL かつ ワークツリーグループのメンバー → IDが返る", func(t *testing.T) {
        // Setup:
        // 1. ワークツリーグループ作成
        worktreeGroupID, _ := db.CreateProjectGroup("voxment", stringPtr("/path/to/voxment"))

        // 2. スタンドアロングループ作成（削除済みワークツリー想定）
        standaloneGroupID, _ := db.CreateProjectGroup("voxment-worktrees-deleted", nil)

        // 3. プロジェクト作成
        projectID, _ := db.CreateProject("deleted-worktree-project", "/path/to/deleted")

        // 4. プロジェクトを両グループに追加
        db.AddProjectToGroup(projectID, worktreeGroupID)
        db.AddProjectToGroup(projectID, standaloneGroupID)

        // Execute
        hiddenIDs, err := db.GetStandaloneGroupsInWorktreeGroups()

        // Assert
        if err != nil {
            t.Fatalf("GetStandaloneGroupsInWorktreeGroups failed: %v", err)
        }
        if len(hiddenIDs) != 1 {
            t.Errorf("Expected 1 hidden group, got %d", len(hiddenIDs))
        }
        if hiddenIDs[0] != standaloneGroupID {
            t.Errorf("Expected standalone group ID %d, got %d", standaloneGroupID, hiddenIDs[0])
        }
    })

    t.Run("git_root=NULL かつ 独立プロジェクト → IDが返らない", func(t *testing.T) {
        // Setup: スタンドアロングループ（ワークツリーグループのメンバーではない）
        standaloneGroupID, _ := db.CreateProjectGroup("independent-project", nil)
        projectID, _ := db.CreateProject("independent", "/path/to/independent")
        db.AddProjectToGroup(projectID, standaloneGroupID)

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
        worktreeGroupID, _ := db.CreateProjectGroup("worktree-group", stringPtr("/path/to/repo"))
        projectID, _ := db.CreateProject("project", "/path/to/project")
        db.AddProjectToGroup(projectID, worktreeGroupID)

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
```

## エッジケース処理

| ケース | 動作 |
|--------|------|
| グループがメンバーを持たない（git_root=NULL） | 表示（独立プロジェクト） |
| グループがメンバーを持たない（git_root!=NULL） | 表示（ワークツリーグループ） |
| 1つのプロジェクトが複数グループに所属 | ワークツリーグループのメンバーなら非表示 |
| `GetStandaloneGroupsInWorktreeGroups()` が空スライスを返す | 全グループ表示（正常動作） |

## パフォーマンス

- **SQLクエリ**: 2回（全グループ取得 + 除外対象ID取得）
- **計算量**: O(n + m)（n=マッピング件数、m=グループ件数）
- **推定実行時間**: 100件規模で数ミリ秒
- **インデックス**: `project_group_mappings` の PRIMARY KEY を活用

## Critical Files

以下のファイルを修正・追加する：

1. **`internal/db/project_groups.go`**
   - `GetStandaloneGroupsInWorktreeGroups()` メソッドを追加
   - 除外対象のグループIDを取得するSQLクエリを実装

2. **`internal/api/service_db.go`**
   - `ListProjectGroups()` メソッドを修正
   - フィルタリングロジックを追加

3. **`internal/db/project_groups_test.go`**
   - `TestGetStandaloneGroupsInWorktreeGroups()` を追加
   - 3つのエッジケースをカバー

## 検証手順

### 1. ユニットテスト

```bash
go test ./internal/db -v -run TestGetStandaloneGroupsInWorktreeGroups
go test ./internal/api -v
```

すべてのテストがパスすることを確認。

### 2. 統合テスト

```bash
# データベースを削除して再同期
rm ccloganalysis.db
.claude/skills/server-management/scripts/start-server.sh dev
```

サーバーログでエラーがないことを確認。

### 3. Playwright MCPで動作確認

1. ブラウザで http://localhost:8080 を開く
2. 「グループ別」タブをクリック
3. 以下を確認：
   - **git_rootを持つグループ（CCLogAnalysis, open-claude-code, voxment, thedotmack）が先頭に表示**
   - **削除済みワークツリーのグループが非表示**（100件から大幅に減少）
   - **独立プロジェクト（`{encoded-standalone-project}` など）が表示**
4. voxmentグループをクリック
5. **削除済みワークツリーがメンバーとして表示されることを確認**

### 4. API直接確認

```bash
# グループ一覧を取得
curl -s http://localhost:8080/api/groups | jq '.groups | length'
# 期待値: 100件から大幅に減少（10-20件程度）

# git_rootを持つグループが先頭に来ることを確認
curl -s http://localhost:8080/api/groups | jq '.groups[:5] | .[] | {name, gitRoot}'
# 期待値: CCLogAnalysis, open-claude-code, thedotmack, voxment

# voxmentグループの詳細を確認
curl -s http://localhost:8080/api/groups/3 | jq '.projects | length'
# 期待値: 7件（削除済みワークツリーも含む）
```

## ロールバック計画

問題が発生した場合：

1. **即座にロールバック**:
   ```bash
   git revert <commit-hash>
   .claude/skills/server-management/scripts/stop-server.sh
   .claude/skills/server-management/scripts/start-server.sh dev
   ```

2. **ホットフィックス**（フィルタリング無効化）:
   ```go
   // service_db.go の ListProjectGroups() を修正
   // hiddenGroupIDs, err := s.db.GetStandaloneGroupsInWorktreeGroups()
   hiddenGroupIDs := []int64{}  // 空スライスで全グループ表示
   ```

## 成功基準

- [ ] ユニットテストがすべてパス
- [ ] 既存テストが壊れていない
- [ ] グループ一覧のグループ数が大幅に減少（100件 → 10-20件程度）
- [ ] git_rootを持つグループが先頭に表示
- [ ] 独立プロジェクトが表示される
- [ ] グループ詳細ページで削除済みワークツリーが引き続き表示される
- [ ] API応答時間が許容範囲内（< 100ms）
