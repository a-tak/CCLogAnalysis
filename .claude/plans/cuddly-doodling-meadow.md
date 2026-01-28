# ログ過剰出力削減の実装計画

## 問題の概要

現在、15秒ごとにバックエンド側で大量のログが出力されている。

### 根本原因

1. **フロントエンド側**：3つのポーリングフック（各15秒間隔）が並行稼働
   - `useProjectsPolling.ts` - 全プロジェクト一覧取得
   - `useProjectDetailPolling.ts` - プロジェクト詳細取得
   - `useSessionsPolling.ts` - セッション一覧取得

2. **バックエンド側**：多層的なログ出力（Logger + fmt.Printf）
   - `/internal/watcher/watcher.go` - ファイルウォッチャー
   - `/internal/api/service_db.go` - APIサービス層
   - `/internal/parser/parser.go` - パーサー層

## 実装方針

ユーザー要望：
- ポーリング間隔：15秒のまま（変更なし）
- ログレベル：INFOのまま

→ **バックエンド側のfmt.Printfをすべて削除または Logger に統一する**

---

## 実装内容

### Phase 1: fmt.Printfの削除または Logger 統一

#### 1-1. `/internal/watcher/watcher.go`

**変更内容**：
- L57: `fmt.Println("File watcher started")` → **削除**（main.goで既に起動メッセージあり）
- L78: `fmt.Println("File watcher stopped")` → **削除**（main.goで既にシャットダウンメッセージあり）
- L96: `fmt.Printf("File watcher: sync failed: %v\n", err)` → **削除**（エラーは上位で処理される）
- L131-132: `fmt.Printf("File watcher: synced %d sessions from %d projects\n", ...)` → **削除**（変更があった場合のみ出力されているが、通常運用では不要）

**理由**：
- ファイルウォッチャーは15秒ごとに動作するため、ログ出力が過剰になる
- 起動・停止メッセージはmain.goで既に出力されている
- エラーはDB層で適切に処理されるため、ここでの出力は不要

#### 1-2. `/internal/api/service_db.go`

**変更内容**：
- L36-52: `autoSyncIfNeeded()` 関数 → **完全削除**（既に使われていない）
- L412: `fmt.Printf("Warning: failed to count sessions for project %s: %v\n", ...)` → **Loggerに変更**

```go
// 変更前
fmt.Printf("Warning: failed to count sessions for project %s: %v\n", row.Name, err)

// 変更後
s.logger.WarnWithContext("Failed to count sessions for project", map[string]interface{}{
    "project": row.Name,
    "error":   err.Error(),
})
```

**理由**：
- `autoSyncIfNeeded`は既にコメントで「使われていない」と記載されている
- L412の警告は重要なので、Loggerで適切に出力する

#### 1-3. `/internal/parser/parser.go`

**変更内容**：
- L148: `fmt.Printf("Warning: failed to parse line %d: %v\n", ...)` → **削除**
- L230: `fmt.Printf("Warning: failed to parse session %s: %v\n", ...)` → **削除**

**理由**：
- パースエラーは`scanner.Err()`で既にエラーとして返される
- 15秒ごとのポーリングで同じ警告が繰り返し出力される可能性が高い
- エラーは上位で処理されるため、ここでの出力は不要

### Phase 2: ドキュメント更新

#### 2-1. `README.md`

**変更内容**：
環境変数セクションに`LOG_LEVEL`の説明を追加

```markdown
## 環境変数

- `LOG_LEVEL`: ログレベル（DEBUG, INFO, WARN, ERROR）デフォルト: INFO
  - 開発時は `LOG_LEVEL=DEBUG` でより詳細なログを表示
  - 本番環境では INFO 推奨
```

**理由**：
- デバッグ時に詳細なログを見たい場合、環境変数で制御できることを明示

---

## 変更するファイル

### 最優先（Phase 1）

1. `/internal/watcher/watcher.go`
   - L57, 78, 96, 131-132のfmt.Printf削除

2. `/internal/api/service_db.go`
   - L36-52の`autoSyncIfNeeded`削除
   - L412のfmt.PrintfをLoggerに変更

3. `/internal/parser/parser.go`
   - L148, 230のfmt.Printf削除

### 補助（Phase 2）

4. `README.md`
   - 環境変数セクションに`LOG_LEVEL`の説明追加

---

## テスト方法

### 1. 通常起動（INFOレベル）

```bash
cd /Users/a-tak/Documents/GitHub/CCLogAnalysis.worktrees/reduce-excessive-logging
go run cmd/server/main.go
```

**確認ポイント**：
- 起動メッセージが表示されること
- ポーリング時に過剰なログが出ていないこと
- エラー時は適切にログ出力されること（service_db.goのL412）

### 2. デバッグモード（DEBUGレベル）

```bash
LOG_LEVEL=DEBUG go run cmd/server/main.go
```

**確認ポイント**：
- 詳細なデバッグ情報が表示されること（DB層のDEBUGログなど）

### 3. ファイルウォッチャーの動作確認

1. サーバーを起動
2. 新しいセッションファイルを追加
3. 15秒後（またはdebounce時間後）に同期が実行されることを確認
4. 変更がない場合はログが出ないことを確認

### 4. フロントエンドのポーリング確認

1. ブラウザの開発者ツールでNetworkタブを開く
2. 15秒ごとにAPIリクエストが送信されることを確認
3. サーバーログで過剰なリクエストログがないことを確認

---

## 実装優先順位

### Phase 1（最優先 - 即座に削減効果）

1. `fmt.Printf`の削除または Logger 統一
2. 廃止コード（`autoSyncIfNeeded`）の削除

### Phase 2（ドキュメント整備）

1. README.mdに`LOG_LEVEL`環境変数の説明追加

---

## Critical Files for Implementation

- `/Users/a-tak/Documents/GitHub/CCLogAnalysis.worktrees/reduce-excessive-logging/internal/watcher/watcher.go`
- `/Users/a-tak/Documents/GitHub/CCLogAnalysis.worktrees/reduce-excessive-logging/internal/api/service_db.go`
- `/Users/a-tak/Documents/GitHub/CCLogAnalysis.worktrees/reduce-excessive-logging/internal/parser/parser.go`
- `/Users/a-tak/Documents/GitHub/CCLogAnalysis.worktrees/reduce-excessive-logging/README.md`
