# セッション0件問題の解決プラン

## 問題の概要

`/Users/{username}/.claude/projects/-Users-{username}-Documents-GitHub-CCLogAnalysis` ディレクトリには66個のセッションファイル（.jsonl）が存在しているのに、アプリケーションでは0件と表示される。

## 根本原因

調査の結果、以下の問題が判明：

1. **エラーの隠蔽**: プロジェクトがDBに登録されていない場合、エラーを返さずに空リストを返している
   - `internal/api/service_db.go` ListSessions() 行83-86

2. **初期同期のサイレントエラー**: 初期同期が失敗しても警告出力のみで継続
   - `internal/api/service_db.go` autoSyncIfNeeded() 行34-48

3. **ログ不足**: 同期処理の各ステップで詳細なログが不足しており、問題の診断が困難

## 実装方針

段階的に実装し、各フェーズでTDDを徹底します：

### Phase 1: 診断機能の実装（最優先）

**目的**: 現在の問題を特定できるようにする

#### 1.1 ログ機能の実装

**新規ファイル**: `internal/logger/logger.go`

構造化ログを提供：
- ログレベル（DEBUG, INFO, WARN, ERROR）
- 環境変数 `LOG_LEVEL` で制御（デフォルト: INFO）
- 標準出力への出力
- プロジェクト名、セッション数などのコンテキスト情報を含む

**テスト**: `internal/logger/logger_test.go`
- ログレベルのフィルタリング
- 構造化ログのフォーマット確認

#### 1.2 同期処理への詳細ログ追加

**変更ファイル**: `internal/db/sync.go`

追加するログポイント：
- SyncAll開始時: プロジェクト数
- 各プロジェクト処理: 開始/完了、セッション数
- 各セッション処理: パース成功/失敗、DB挿入成功/失敗
- エラー発生時: 詳細なエラーメッセージ（プロジェクト名、セッションID含む）

**テストの更新**: `internal/db/sync_test.go`
- ログ出力の確認（テスト用ログバッファを使用）

#### 1.3 診断エンドポイントの追加

**新規ファイル**: `internal/api/debug.go`

エンドポイント:
- `GET /api/debug/status`
  - DB内のプロジェクト数
  - DB内のセッション数
  - ファイルシステム上のプロジェクト数
  - 初期同期の状態（成功/失敗）
  - 最後の同期エラー（あれば）

**テスト**: `internal/api/debug_test.go`

**変更ファイル**: `internal/api/router.go`
- 診断エンドポイントのルーティング追加（行111付近）

**変更ファイル**: `internal/api/service_db.go`
- `syncError error` フィールドを追加して初期同期エラーを保持
- 診断情報を取得するメソッド追加

### Phase 2: エラーハンドリングの改善

**目的**: エラーをユーザーに可視化する

#### 2.1 ListSessions のエラー改善

**変更ファイル**: `internal/api/service_db.go` (行83-86)

**現在のコード**:
```go
project, err := s.db.GetProjectByName(projectName)
if err != nil {
    // プロジェクトが存在しない場合は空のリストを返す
    return []SessionSummary{}, nil
}
```

**改善後**:
- プロジェクトが見つからない場合、ログに警告を出力
- レスポンスに警告メッセージを含める（空リストは返すが、なぜ空なのかを示す）
- または、HTTPステータスコード404を返す（API設計の変更が必要）

**テストの更新**: `internal/api/service_db_test.go`
- プロジェクト不存在時の挙動確認

#### 2.2 初期同期エラーの可視化

**変更ファイル**: `internal/api/service_db.go` (行34-48)

**改善内容**:
- `DatabaseSessionService` 構造体に `syncError error` フィールド追加
- `autoSyncIfNeeded()` でエラーが発生した場合、`syncError` に保存
- `GetSyncStatus()` メソッドを追加して、初期同期の状態を取得可能に

**テストの追加**: `internal/api/service_db_test.go`
- 初期同期失敗時の挙動確認
- `GetSyncStatus()` の動作確認

#### 2.3 SyncResult へのエラー詳細追加

**変更ファイル**: `internal/db/sync.go` (行11-17)

**現在の構造体**:
```go
type SyncResult struct {
    ProjectsProcessed int
    SessionsFound     int
    SessionsSynced    int
    SessionsSkipped   int
    ErrorCount        int
}
```

**改善後**:
```go
type SyncResult struct {
    ProjectsProcessed int
    SessionsFound     int
    SessionsSynced    int
    SessionsSkipped   int
    ErrorCount        int
    Errors            []string // エラー詳細のリスト
}
```

各エラーは「プロジェクト名: エラー内容」または「プロジェクト名/セッションID: エラー内容」の形式で記録

**テストの更新**: `internal/db/sync_test.go`
- エラー詳細の記録確認

### Phase 3: 同期処理の堅牢化（オプション）

**目的**: 根本的な問題を防ぐ

#### 3.1 プロジェクト登録の検証強化

**変更ファイル**: `internal/db/sync.go` (行67-85)

**改善内容**:
- プロジェクト作成後、実際にDBから取得できるか確認
- 取得できない場合、詳細なエラーメッセージをログ出力
- `SyncResult.Errors` にエラーを追加

#### 3.2 パーサーのエラーハンドリング強化

**変更ファイル**: `internal/parser/parser.go` (行50-64)

**改善内容**:
- ディレクトリアクセスエラーの詳細化（権限エラー、存在しないなど）
- エラーメッセージに具体的なパスを含める（個人情報は一般化）

**テストの追加**: `internal/parser/parser_test.go`
- 読み取り不可ディレクトリのテスト
- 存在しないディレクトリのテスト

## 実装の優先順位

### 1. 最優先（即座に実装）
- Phase 1-1: ログ機能
- Phase 1-2: 同期処理へのログ追加
- Phase 1-3: 診断エンドポイント

→ ユーザーが問題を診断できるようになる

### 2. 次に優先（診断後すぐ）
- Phase 2: エラーハンドリング改善

→ エラーが可視化され、問題が明確になる

### 3. その後実装（オプション）
- Phase 3: 同期処理の堅牢化

→ 再発防止

## 重要なファイル

### 新規作成
- `internal/logger/logger.go` - ログ機能
- `internal/logger/logger_test.go` - ログ機能のテスト
- `internal/api/debug.go` - 診断エンドポイント
- `internal/api/debug_test.go` - 診断エンドポイントのテスト

### 変更が必要
- `internal/db/sync.go` - 詳細ログ追加、エラー詳細記録
- `internal/db/sync_test.go` - テスト更新
- `internal/api/service_db.go` - エラーハンドリング改善、診断機能追加
- `internal/api/service_db_test.go` - テスト更新
- `internal/api/router.go` - 診断エンドポイントのルーティング
- `internal/parser/parser.go` - エラーメッセージ改善（オプション）
- `internal/parser/parser_test.go` - テスト追加（オプション）

## 検証方法

### 開発中の検証

1. **ログ機能のテスト**
   ```bash
   go test ./internal/logger/...
   ```

2. **同期処理のテスト**
   ```bash
   LOG_LEVEL=DEBUG go test ./internal/db/... -v
   ```

3. **診断エンドポイントのテスト**
   ```bash
   go test ./internal/api/... -v
   ```

### 統合テスト

1. **サーバー起動**
   ```bash
   LOG_LEVEL=DEBUG go run cmd/server/main.go
   ```

2. **診断エンドポイント確認**
   ```bash
   curl http://localhost:3001/api/debug/status
   ```

   期待される出力:
   ```json
   {
     "db_projects": 0,
     "db_sessions": 0,
     "fs_projects": 10,
     "sync_status": "failed",
     "sync_error": "failed to sync project -Users-{username}-Documents-GitHub-CCLogAnalysis: ..."
   }
   ```

3. **ログ確認**
   - 初期同期時のログを確認
   - プロジェクト検出のログ
   - セッションファイル検出のログ
   - エラー発生箇所のログ

4. **問題解決後の確認**
   ```bash
   curl http://localhost:3001/api/sessions?project=-Users-{username}-Documents-GitHub-CCLogAnalysis
   ```

   期待される結果: 66件のセッションが返される

## 成功条件

- [ ] ログ機能が実装され、すべてのテストがパス
- [ ] 同期処理に詳細ログが追加され、テストがパス
- [ ] 診断エンドポイントが実装され、テストがパス
- [ ] ListSessions のエラーハンドリングが改善され、テストがパス
- [ ] 初期同期エラーが可視化され、テストがパス
- [ ] SyncResult にエラー詳細が含まれ、テストがパス
- [ ] サーバー起動時のログで問題箇所が特定できる
- [ ] 診断エンドポイントで問題の原因が特定できる
- [ ] 問題が解決され、66件のセッションが正しく表示される

## 注意事項

### TDD の徹底
- すべての新機能・変更は、テストを先に書く
- Red（失敗）→ Green（成功）→ Refactor（リファクタリング）のサイクルを守る

### 個人情報の保護
- ログメッセージやエラーメッセージには個人情報を含めない
- パスは `{username}`, `{project-name}` で一般化
- テストデータも同様に一般化

### 後方互換性
- 既存のAPIレスポンスフォーマットは維持
- 新しいフィールドは追加のみ
- エンドポイントのURLパスは変更しない

### パフォーマンス
- ログ出力のオーバーヘッドを最小化（DEBUGレベルのみ詳細ログ）
- 診断エンドポイントは軽量に保つ
