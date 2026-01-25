# Gitワークツリー対応実装プラン

## 概要

現在、セッション単位の集計は完了していますが、以下の機能がまだありません：
1. **プロジェクト単位の集計**（複数セッションを横断した統計）
2. **Git Root検出**（ワークツリーの親リポジトリを特定）
3. **プロジェクトグループ**（Git Rootが同じプロジェクトをまとめて分析）

このプランでは、段階的に上記の機能を実装します。

---

## 実装ステップ

### Step 1: プロジェクト単位の集計機能

**目的**: 1つのプロジェクト内の全セッションを集計し、統計情報とトレンドを可視化する

#### データベース層

**新規ファイル**: `internal/db/project_stats.go`

実装する関数：
- `GetProjectStats(projectID)` - プロジェクト全体の統計（総セッション数、総トークン数、平均、エラー率など）
- `GetBranchStats(projectID)` - ブランチ別の統計
- `GetTimeSeriesStats(projectID, period, limit)` - 時系列推移（日別/週別/月別）

**SQLクエリ例**（プロジェクト統計）:
```sql
SELECT
    COUNT(*) as total_sessions,
    SUM(total_input_tokens) as total_input_tokens,
    SUM(total_output_tokens) as total_output_tokens,
    AVG(total_input_tokens + total_output_tokens) as avg_tokens,
    MIN(start_time) as first_session,
    MAX(end_time) as last_session,
    CAST(SUM(CASE WHEN error_count > 0 THEN 1 ELSE 0 END) AS REAL) / COUNT(*) as error_rate
FROM sessions
WHERE project_id = ?
```

**既存インデックス活用**: `idx_sessions_project_start` で十分カバーできるため、新規インデックスは不要

#### API層

**新規エンドポイント**:
- `GET /api/projects/{name}/stats` - プロジェクト統計
- `GET /api/projects/{name}/timeline?period={day|week|month}` - 時系列推移

**新規ファイル**: `internal/api/handlers_project_stats.go`

**修正ファイル**:
- `internal/api/types.go` - レスポンス型追加（`ProjectStatsResponse`, `TimeSeriesResponse` など）
- `internal/api/service_db.go` - サービスメソッド追加
- `internal/api/router.go` - ルーティング追加

#### フロントエンド

**新規ファイル**: `web/src/pages/ProjectDetailPage.tsx`

主要機能：
- プロジェクト基本情報表示
- トークン使用量サマリー（総計・平均・エラー率）
- ブランチ別統計テーブル
- 時系列推移グラフ（Recharts使用）
  - 日別/週別/月別切り替え
  - セッション数推移
  - トークン使用量推移

**修正ファイル**:
- `web/src/lib/api/types.ts` - 型定義追加
- `web/src/lib/api/client.ts` - API関数追加
- `web/src/pages/ProjectsPage.tsx` - プロジェクト名をクリックすると詳細ページに遷移
- `web/src/App.tsx` - ルート追加（`/projects/:projectName`）

#### テスト戦略

- `internal/db/project_stats_test.go` - データベース層テスト（正常系、エッジケース）
- `internal/api/handlers_project_stats_test.go` - APIハンドラーテスト（HTTPステータス、レスポンス構造）
- テストデータ: 複数セッション（異なる日付・ブランチ・エラー状態）を作成

---

### Step 2: Git Root 検出機能

**目的**: プロジェクトのGit Rootを検出し、ワークツリーと親リポジトリの関連を把握する

#### Git Root検出ロジック

**新規パッケージ**: `internal/gitutil/gitutil.go`

実装する関数：
- `DetectGitRoot(projectPath)` - `.git` ディレクトリまたは `.git` ファイルを解析してGit Rootを返す
  - **通常のリポジトリ**: `.git/` ディレクトリが存在 → そのパスを返す
  - **ワークツリー**: `.git` ファイル（`gitdir: /path/to/repo.git/worktrees/name`） → 親リポジトリのパスを抽出
  - **Git管理外**: `.git` が存在しない → 空文字列を返す（エラーではない）

**ワークツリーの `.git` ファイル例**:
```
gitdir: /path/to/repo.git/worktrees/feature-a
```
→ `/path/to/repo.git` を抽出

#### データベース同期処理の拡張

**修正ファイル**: `internal/db/sync.go` の `syncProjectInternal()` 関数

変更内容：
- プロジェクト作成時にGit Rootを検出
- Git Root検出成功 → `CreateProjectWithGitRoot()` で保存
- Git Root検出失敗 → エラーとせず、`git_root` を null で保存（警告ログ出力）
- 既存プロジェクトで `git_root` が未設定の場合、次回同期時に自動更新

#### テスト戦略

- `internal/gitutil/gitutil_test.go`
  - 通常のGitリポジトリでテスト
  - Gitワークツリーでテスト（`.git` ファイルをパース）
  - Git管理外プロジェクトでテスト（空文字列を返す）
  - 不正な `.git` ファイル形式でエラーテスト
- `internal/db/sync_test.go` に追加
  - Git Root検出・保存のテスト
  - 既存プロジェクトの更新テスト

**testdata準備**:
- `testdata/normal-repo/.git/` ディレクトリ
- `testdata/worktree/.git` ファイル（内容: `gitdir: /path/to/repo.git/worktrees/worktree`）
- `testdata/no-git/` ディレクトリ（`.git` なし）

---

### Step 3: プロジェクトグループ機能

**目的**: Git Rootが同じプロジェクトを自動的にグループ化し、グループ単位で統計を集計する

#### データベーススキーマ調整

**既存スキーマ** (`internal/db/schema.sql` L147-160):
```sql
CREATE TABLE IF NOT EXISTS project_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    git_root TEXT NOT NULL,  -- UNIQUE制約なし
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    -- updated_at カラムなし
);
```

**調整内容**（マイグレーション実装が必要）:
1. `project_groups.git_root` に **UNIQUE制約** を追加（同じGit Rootのグループは1つだけ）
2. `project_groups.updated_at` カラムを追加
3. `updated_at` 自動更新トリガーを追加
4. インデックス追加:
   - `idx_project_group_mappings_group ON project_group_mappings(group_id)`
   - `idx_project_groups_git_root ON project_groups(git_root)`

**マイグレーションスクリプト**: `internal/db/migrations/003_project_groups.sql`（新規作成）

SQLiteは既存テーブルに制約を追加できないため、テーブルを再作成する：
1. `project_groups_new` テーブルを作成（UNIQUE制約、updated_atカラム付き）
2. 既存データを移行
3. 旧テーブルを削除、新テーブルをリネーム
4. トリガー・インデックス追加

#### データベース層実装

**新規ファイル**: `internal/db/project_groups.go`

CRUD操作：
- `CreateProjectGroup(name, gitRoot)` - グループ作成
- `GetProjectGroupByGitRoot(gitRoot)` - Git Rootからグループ取得
- `GetProjectGroupByID(id)` - IDからグループ取得
- `ListProjectGroups()` - グループ一覧
- `AddProjectToGroup(projectID, groupID)` - プロジェクトをグループに追加
- `GetProjectsByGroupID(groupID)` - グループ内のプロジェクト一覧
- `GetGroupWithProjects(groupID)` - グループとメンバープロジェクトを取得
- `DeleteProjectGroup(id)` - グループ削除（CASCADE削除でmappingsも自動削除）

**新規ファイル**: `internal/db/grouping.go`

自動グループ化ロジック：
- `SyncProjectGroups()` - プロジェクトグループを自動生成・同期
  1. 全プロジェクトを取得
  2. Git Rootごとにグループ化（map[gitRoot][]*ProjectRow）
  3. 各Git Rootに対してグループを作成（存在しない場合）
  4. グループにプロジェクトを追加
- `generateGroupName(gitRoot)` - Git Rootからグループ名を生成（例: `/path/to/repo.git` → `repo`）

**新規ファイル**: `internal/db/group_stats.go`

グループ統計集計：
- `GetGroupStats(groupID)` - グループ全体の統計（総プロジェクト数、総セッション数、総トークン数、平均、エラー率など）

**SQLクエリ例**（グループ統計）:
```sql
SELECT
    COUNT(DISTINCT p.id) as total_projects,
    COUNT(s.id) as total_sessions,
    SUM(s.total_input_tokens) as total_input_tokens,
    SUM(s.total_output_tokens) as total_output_tokens,
    AVG(s.total_input_tokens + s.total_output_tokens) as avg_tokens,
    MIN(s.start_time) as first_session,
    MAX(s.end_time) as last_session,
    CAST(SUM(CASE WHEN s.error_count > 0 THEN 1 ELSE 0 END) AS REAL) / COUNT(s.id) as error_rate
FROM project_group_mappings pgm
INNER JOIN projects p ON pgm.project_id = p.id
LEFT JOIN sessions s ON p.id = s.project_id
WHERE pgm.group_id = ?
```

#### 同期処理の拡張

**修正ファイル**: `internal/db/sync.go` の `SyncAll()` 関数

変更内容：
- 全プロジェクトの同期後に `db.SyncProjectGroups()` を呼び出す
- エラーが発生しても警告ログを出力して続行（同期処理全体は失敗させない）

#### API層

**新規エンドポイント**:
- `GET /api/groups` - プロジェクトグループ一覧
- `GET /api/groups/{id}` - グループ詳細（配下のプロジェクト一覧含む）
- `GET /api/groups/{id}/stats` - グループ単位の統計

**新規ファイル**: `internal/api/handlers_groups.go`

**修正ファイル**:
- `internal/api/types.go` - レスポンス型追加（`ProjectGroupResponse`, `ProjectGroupStatsResponse` など）
- `internal/api/service_db.go` - サービスメソッド追加
- `internal/api/router.go` - ルーティング追加

#### フロントエンド

**修正ファイル**: `web/src/pages/ProjectsPage.tsx`

変更内容：
- プロジェクト一覧に**グループ表示**を追加
- 表示パターン:
  - タブ切り替え（「全プロジェクト」「グループ別」）
  - グループ別表示: グループカード → 配下のプロジェクト一覧

**新規ファイル**: `web/src/pages/GroupDetailPage.tsx`

主要機能：
- グループ基本情報表示（グループ名、Git Root）
- グループ全体の統計サマリー（総プロジェクト数、総トークン数など）
- 配下プロジェクト一覧
- グループ全体のトークン使用量推移グラフ

**修正ファイル**:
- `web/src/lib/api/types.ts` - 型定義追加
- `web/src/lib/api/client.ts` - API関数追加
- `web/src/App.tsx` - ルート追加（`/groups/:groupId`）

#### テスト戦略

- `internal/db/project_groups_test.go`
  - CRUD操作のテスト（作成、取得、一覧、削除）
  - UNIQUE制約のテスト（同じgit_rootで重複作成できない）
  - CASCADE削除のテスト
- `internal/db/grouping_test.go`
  - 自動グループ化のテスト（Git Rootが同じプロジェクトがグループ化される）
  - グループ名生成のテスト
  - 既存グループに新規プロジェクトが追加されるテスト
- `internal/api/handlers_groups_test.go`
  - APIハンドラーテスト（HTTPステータス、レスポンス構造）

---

## 重要なファイル一覧

### Step 1: プロジェクト単位の集計機能

**バックエンド（新規作成）**:
- `internal/db/project_stats.go` - プロジェクト統計集計の中核ロジック
- `internal/db/project_stats_test.go` - テスト
- `internal/api/handlers_project_stats.go` - プロジェクト統計APIハンドラー
- `internal/api/handlers_project_stats_test.go` - テスト

**バックエンド（修正）**:
- `internal/api/types.go` - レスポンス型追加
- `internal/api/service_db.go` - サービスメソッド追加
- `internal/api/router.go` - ルーティング追加

**フロントエンド（新規作成）**:
- `web/src/pages/ProjectDetailPage.tsx` - プロジェクト詳細ページ

**フロントエンド（修正）**:
- `web/src/lib/api/types.ts` - 型定義追加
- `web/src/lib/api/client.ts` - API関数追加
- `web/src/pages/ProjectsPage.tsx` - プロジェクト名をリンク化
- `web/src/App.tsx` - ルート追加

### Step 2: Git Root 検出機能

**バックエンド（新規作成）**:
- `internal/gitutil/gitutil.go` - Git Root検出ロジック
- `internal/gitutil/gitutil_test.go` - テスト
- `internal/gitutil/testdata/` - テストデータ（通常リポジトリ、ワークツリー、Git管理外）

**バックエンド（修正）**:
- `internal/db/sync.go` - 同期処理にGit Root検出を統合
- `internal/db/sync_test.go` - テスト追加

### Step 3: プロジェクトグループ機能

**バックエンド（新規作成）**:
- `internal/db/migrations/003_project_groups.sql` - マイグレーションスクリプト
- `internal/db/project_groups.go` - プロジェクトグループのCRUD操作
- `internal/db/project_groups_test.go` - テスト
- `internal/db/grouping.go` - 自動グループ化ロジック
- `internal/db/grouping_test.go` - テスト
- `internal/db/group_stats.go` - グループ統計集計
- `internal/db/group_stats_test.go` - テスト
- `internal/api/handlers_groups.go` - グループAPIハンドラー
- `internal/api/handlers_groups_test.go` - テスト

**バックエンド（修正）**:
- `internal/db/schema.sql` - プロジェクトグループテーブルのスキーマ調整（コメント更新）
- `internal/db/db.go` - マイグレーション実行ロジック追加
- `internal/db/sync.go` - SyncAll()にプロジェクトグループ同期を追加
- `internal/api/types.go` - レスポンス型追加
- `internal/api/service_db.go` - サービスメソッド追加
- `internal/api/router.go` - ルーティング追加

**フロントエンド（新規作成）**:
- `web/src/pages/GroupDetailPage.tsx` - グループ詳細ページ

**フロントエンド（修正）**:
- `web/src/pages/ProjectsPage.tsx` - グループ表示を追加
- `web/src/lib/api/types.ts` - 型定義追加
- `web/src/lib/api/client.ts` - API関数追加
- `web/src/App.tsx` - ルート追加

---

## 検証方法

### Step 1: プロジェクト単位の集計機能

1. **単体テスト**: `go test ./internal/db ./internal/api`
2. **APIテスト**:
   ```bash
   # プロジェクト統計
   curl http://localhost:8080/api/projects/{project-name}/stats

   # 時系列統計（日別）
   curl "http://localhost:8080/api/projects/{project-name}/timeline?period=day"
   ```
3. **UI確認**: ブラウザでプロジェクト一覧からプロジェクトをクリックし、詳細ページが表示されることを確認
4. **グラフ確認**: Rechartsのグラフが正しくレンダリングされ、データが表示されることを確認

### Step 2: Git Root 検出機能

1. **単体テスト**: `go test ./internal/gitutil ./internal/db`
2. **Git Root検出確認**:
   - 通常のGitリポジトリ（`.git/` ディレクトリ）で `~/.claude/projects/` 配下にプロジェクトを配置
   - Gitワークツリー（`.git` ファイル）でプロジェクトを配置
   - Git管理外のプロジェクトを配置
   - サーバーを起動 → 自動同期でGit Rootが正しく検出されることを確認
3. **データベース確認**:
   ```sql
   SELECT name, decoded_path, git_root FROM projects;
   ```
   各プロジェクトの `git_root` カラムが正しく設定されていることを確認

### Step 3: プロジェクトグループ機能

1. **単体テスト**: `go test ./internal/db ./internal/api`
2. **APIテスト**:
   ```bash
   # グループ一覧
   curl http://localhost:8080/api/groups

   # グループ詳細
   curl http://localhost:8080/api/groups/1

   # グループ統計
   curl http://localhost:8080/api/groups/1/stats
   ```
3. **自動グループ化確認**:
   - 同じリポジトリの複数ワークツリーを `~/.claude/projects/` に配置
     - 例: `my-repo/`, `my-repo-feature-a/`, `my-repo-feature-b/`
   - サーバーを起動 → 自動同期でグループが作成されることを確認
4. **データベース確認**:
   ```sql
   SELECT * FROM project_groups;
   SELECT * FROM project_group_mappings;
   ```
   Git Rootが同じプロジェクトが同じグループに属していることを確認
5. **UI確認**: ブラウザでプロジェクト一覧を開き、グループ表示が正しく機能することを確認

---

## 実装上の注意事項

### TDD（テスト駆動開発）

- **必須**: すべての新機能は「テスト→実装→リファクタリング」のサイクルで開発
- Red（失敗するテストを書く）→ Green（テストを通す）→ Refactor（リファクタリング）

### トランザクション

- 複数テーブルの更新は1トランザクション内で実行
- エラー時は自動ロールバック（`defer tx.Rollback()`）
- 成功時に明示的にコミット

### パフォーマンス

- 集計クエリが重くならないよう、既存インデックスを活用
- 時系列統計はLIMITで取得件数を制限（デフォルト30件）
- 将来的には `period_statistics` テーブル（キャッシュ）を活用可能

### エラーハンドリング

- Git Root検出失敗はエラーとせず、警告ログを出力して続行
- プロジェクトグループ同期失敗も警告ログを出力して続行
- APIエンドポイントでは適切なHTTPステータスコードを返す
  - 200 OK: 成功
  - 400 Bad Request: リクエストが不正
  - 404 Not Found: リソースが見つからない
  - 500 Internal Server Error: サーバーエラー

### 個人情報保護

- ドキュメント・コード例では実際のパスを `{username}` などで一般化
- テストデータでも実際のユーザー名・プロジェクト名を含めない

---

## ドキュメント更新

実装完了後に以下のドキュメントを更新：

1. **`docs/API設計.md`** - 新規エンドポイントを追加
2. **`docs/要件.md`** - Phase 2 の進捗を更新
3. **`README.md`** - 新機能の説明を追加
4. **新規ドキュメント**: `docs/プロジェクトグループ機能.md` - 詳細仕様を記載

---

## リスクと対策

| リスク | 対策 |
|--------|------|
| Git Root検出の失敗 | エラーとせず、git_rootをnullで保存。将来的に手動グループ作成機能を追加 |
| パフォーマンス低下 | インデックス活用、LIMIT設定、将来的にキャッシュテーブル活用 |
| 既存データとの整合性 | マイグレーションで既存データ保持、トランザクション内で更新 |
| Windowsパスの互換性 | `filepath` パッケージ使用、Windows環境でテスト |

---

## まとめ

このプランにより、以下が実現できます：

1. **プロジェクト単位の集計** - セッションを横断した統計とトレンド可視化
2. **Git Root検出** - ワークツリーと親リポジトリの関連把握
3. **プロジェクトグループ** - 関連プロジェクトをまとめて分析

各ステップは独立して実装可能なため、段階的に進められます。TDDに従い、テストを書きながら確実に実装していきます。
