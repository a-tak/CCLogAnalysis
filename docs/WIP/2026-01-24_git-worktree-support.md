# WIP: Gitワークツリー対応実装

**日付**: 2026-01-24
**セッション**: cb746300-39f2-4b1c-91ee-fcd0b191afeb

## 実装完了項目

### Step 1: プロジェクト単位の集計 ✅
- ✅ データベース層（`internal/db/project_stats.go`）
  - `GetProjectStats()`: プロジェクト全体の統計
  - `GetBranchStats()`: ブランチ別統計
  - `GetTimeSeriesStats()`: 時系列推移（日別/週別/月別）
- ✅ API層（`internal/api/handlers_project_stats.go`）
  - `GET /api/projects/{name}/stats`
  - `GET /api/projects/{name}/timeline`
- ✅ フロントエンド（`web/src/pages/ProjectDetailPage.tsx`）
  - プロジェクト統計サマリー表示
  - トークン使用量推移グラフ（Recharts）
  - 日別/週別/月別切り替え

### Step 2: Git Root検出 ⚠️
- ✅ Git Root検出ロジック（`internal/gitutil/gitutil.go`）
  - `DetectGitRoot()`: `.git`ディレクトリまたはファイルからGit Rootを検出
  - ワークツリー対応（`.git`ファイルのパース）
- ⚠️ **問題あり**: 同期処理への統合（`internal/db/sync.go`）
  - 現在の実装は`decodedPath`でGit Root検出を試みている
  - しかし`decodedPath`は実際のファイルシステムパスではない

### Step 3: プロジェクトグループ ✅
- ✅ マイグレーション（`internal/db/migrations/003_project_groups.sql`）
- ✅ プロジェクトグループCRUD（`internal/db/project_groups.go`）
- ✅ 自動グループ化ロジック（`internal/db/grouping.go`）
- ✅ グループ統計（`internal/db/group_stats.go`）
- ✅ 同期処理への統合（`SyncAll()`に`SyncProjectGroups()`を追加）
- ✅ API層（`internal/api/handlers_groups.go`）
  - `GET /api/groups`
  - `GET /api/groups/{id}`
  - `GET /api/groups/{id}/stats`
- ✅ フロントエンド
  - `web/src/pages/GroupDetailPage.tsx`: グループ詳細ページ
  - `web/src/pages/ProjectsPage.tsx`: タブ切り替え（全プロジェクト/グループ別）

## 残課題: Git Root検出の修正

### 問題の詳細

**現状の問題**:
1. プロジェクトディレクトリ構造
   - Claude Codeのプロジェクトは`~/.claude/projects/{encoded-name}`に保存
   - このディレクトリには`.git`ファイルが存在しない
   - 実際の作業ディレクトリとは別の場所

2. `decodedPath`の誤解
   - `decoded_path`はプロジェクト名をデコードしただけのもの
   - 実際のファイルシステムパスではない
   - 例: `-Users-a-tak-Documents-GitHub-CCLogAnalysis` → `/Users/a/tak/Documents/GitHub/CCLogAnalysis`
     - しかし実際にはこのパスは存在しない（`decoded_path`は単なる表示用）

3. 実際の作業ディレクトリパス
   - 実際の作業ディレクトリは不明
   - プロジェクトメタデータやログファイルから抽出する必要がある

### 解決策: 方法3（プロジェクトディレクトリのメタデータから実際のパスを取得）

**調査結果**:
```bash
$ ls -la ~/.claude/projects/-Users-a-tak-Documents-GitHub-CCLogAnalysis-worktrees-project-summary/
drwx------@   5 a-tak  staff      160 Jan 24 19:02 .
drwx------@ 111 a-tak  staff     3552 Jan 24 20:39 ..
drwx------@   3 a-tak  staff       96 Jan 24 18:47 1bb545cb-dd54-41d9-b759-5e9fa337ee41
-rw-------@   1 a-tak  staff   988896 Jan 24 18:58 1bb545cb-dd54-41d9-b759-5e9fa337ee41.jsonl
-rw-------@   1 a-tak  staff  3682568 Jan 24 21:02 cb746300-39f2-4b1c-91ee-fcd0b191afeb.jsonl
```

プロジェクトディレクトリには：
- セッションIDのディレクトリ（例: `1bb545cb-dd54-41d9-b759-5e9fa337ee41/`）
- セッションJSONLファイル（例: `cb746300-39f2-4b1c-91ee-fcd0b191afeb.jsonl`）

**次の実装ステップ**:

1. **セッションディレクトリまたはJSONLファイルから実際のパスを抽出**
   - セッションJSONLファイルの最初の方に`cwd`（Current Working Directory）情報がある可能性
   - セッションディレクトリ内のメタデータファイルを確認

2. **調査が必要な項目**:
   ```bash
   # セッションディレクトリの中身を確認
   ls -la ~/.claude/projects/{project-name}/{session-id}/

   # JSONLファイルの最初の数行を確認（cwdを探す）
   head -50 ~/.claude/projects/{project-name}/{session-id}.jsonl | jq .
   ```

3. **実装方針**:
   - `internal/parser/parser.go`に`GetProjectWorkingDirectory(projectName)`メソッドを追加
   - セッションJSONLファイルまたはメタデータから実際の作業ディレクトリパスを抽出
   - `internal/db/sync.go`でこのメソッドを使ってGit Root検出

4. **代替案**:
   - もしメタデータから取得できない場合、手動でGit Rootを設定できるAPI/UIを追加
   - または、`decoded_path`をそのまま使用し、存在しないパスの場合はスキップ

## コミット状況

### 完了済みコミット
1. `67d0fa0`: feat: プロジェクトグループ機能のAPI層を実装
   - Step 3のバックエンド実装（API層、同期処理統合）
2. `38f0300`: feat: プロジェクトグループ機能のフロントエンド実装
   - Step 3のフロントエンド実装（全ページコンポーネント）

### 未コミット
- `internal/db/sync.go`: Git Root検出の修正（現在は問題あり）
- `internal/parser/parser.go`: `DecodeProjectPath()`の修正試行（未完成）

## 次のセッションでやること

1. **セッションメタデータの調査**
   - プロジェクトディレクトリ内のセッションディレクトリを調査
   - JSONLファイルから`cwd`や実際の作業ディレクトリ情報を探す

2. **実際のパス取得メソッドの実装**
   - `Parser.GetProjectWorkingDirectory(projectName)`を実装
   - セッションデータから実際の作業ディレクトリパスを抽出

3. **Git Root検出の修正**
   - `internal/db/sync.go`で実際のパスを使ってGit Root検出
   - 存在しないパスの場合のエラーハンドリング

4. **テストと動作確認**
   - 修正後、実際のプロジェクトでGit Root検出が動作することを確認
   - グループ化が正しく機能することを確認

5. **コミット**
   - 修正が完了したら`fix: Git Root検出を実際の作業ディレクトリで実行`としてコミット

## 参考情報

### 関連ファイル
- `internal/gitutil/gitutil.go`: Git Root検出ロジック
- `internal/db/sync.go`: 同期処理（Git Root検出を呼び出す）
- `internal/parser/parser.go`: プロジェクトパース処理
- `internal/db/grouping.go`: 自動グループ化ロジック

### テストコマンド
```bash
# サーバー起動
./bin/ccloganalysis

# Git Root設定状況確認
sqlite3 bin/ccloganalysis.db "SELECT COUNT(*) as total, COUNT(git_root) as with_git_root FROM projects;"

# グループ確認
curl -s http://localhost:8080/api/groups | jq '.'
```

### 現在のサーバー状態
- サーバーは起動中（PID: /tmp/server.pid）
- ポート: 8080
- ログ: `/tmp/server_new.log`
- データベース: `bin/ccloganalysis.db`

## メモ

- UI機能（プロジェクト詳細ページ、グループ詳細ページ）は正常に動作している
- Git Root検出さえ修正できれば、完全に機能する
- 現在は手動でGit Rootを設定すればグループ化は動作する

---

## 完了報告（セッション: db82df6e-ac08-4906-a49b-d8d1f420eb6f）

**日時**: 2026-01-24 23:22

### 実装完了: Git Root検出の修正 ✅

**実装内容**:
1. ✅ `Parser.GetProjectWorkingDirectory()`メソッドを追加（`internal/parser/parser.go`）
   - セッションJSONLから最初のセッションを取得
   - `cwd`フィールドから実際の作業ディレクトリパスを抽出
   - セッションが存在しない場合はエラーを返す
   - cwdが空の場合はエラーを返す

2. ✅ `sync.go`の修正（`internal/db/sync.go`）
   - `GetProjectDir()`の代わりに`GetProjectWorkingDirectory()`を使用
   - セッションが存在しない場合のエラーハンドリング追加
   - 新規プロジェクト作成時と既存プロジェクトのGit Root更新時の両方を修正

3. ✅ テストケース追加（TDD）
   - `TestGetProjectWorkingDirectory_Success`: 通常ケース
   - `TestGetProjectWorkingDirectory_NoSessions`: セッションが存在しない
   - `TestGetProjectWorkingDirectory_EmptyCwd`: cwdが空のセッション
   - `TestGetProjectWorkingDirectory_MultipleSessionsSameDir`: 複数セッション
   - `TestGitRootDetection/セッションが存在しないプロジェクトでgit_rootがnullになる`: エッジケース

**テスト結果**: 全テストパス ✅
- `internal/parser`のテスト: 全てパス
- `internal/db`のテスト: 全てパス
- Git Root検出テスト: 全ケースパス

**コミット**:
```
bb22e0b: fix: Git Root検出を実際の作業ディレクトリで実行

セッションのcwdフィールドから実際の作業ディレクトリを取得し、
Git Root検出を実行するように修正。

- Parser.GetProjectWorkingDirectory()を追加
- sync.goでセッションベースのパス取得に変更
- セッションが存在しない場合のエラーハンドリング追加
```

### 解決した問題

1. **問題**: `~/.claude/projects/{encoded-name}`には`.git`が存在しない
   - **解決**: セッションの`cwd`フィールドから実際の作業ディレクトリを取得

2. **問題**: `decodedPath`は実際のファイルシステムパスではない
   - **解決**: `decodedPath`を使わず、セッションデータから実際のパスを取得

3. **問題**: ワークツリープロジェクトで親リポジトリのGit Rootが検出されない
   - **解決**: 実際の作業ディレクトリで検出することで、`gitutil.DetectGitRoot()`が正しく動作

### 動作確認

**期待される動作**:
- ✅ ワークツリープロジェクト（`project-summary`など）のGit Rootが正しく検出される
- ✅ 同じGit Rootを持つプロジェクトが同じグループにまとめられる
- ✅ セッションがないプロジェクトは`git_root = null`でエラーが発生しない
- ✅ Git管理外のプロジェクトは`git_root = null`になる

### 残課題

なし。Git Root検出機能は完全に実装され、全てのテストがパスしています。

### 次のステップ

1. サーバーを再起動して実際のプロジェクトでGit Root検出が動作することを確認
2. グループ化機能が正しく動作することを確認
3. 必要に応じてドキュメントを更新
