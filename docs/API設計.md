# API設計書

## 概要

Claude Code ログ解析アプリケーションのREST API仕様書です。

---

## ベースURL

```
http://localhost:8080/api
```

---

## エンドポイント一覧

### 1. ヘルスチェック

サーバーの稼働状態を確認します。

**エンドポイント**: `GET /health`

**レスポンス**:
```json
{
  "status": "ok"
}
```

**ステータスコード**:
- `200 OK`: 正常

---

### 2. プロジェクト一覧取得

Claude Codeのプロジェクト一覧を取得します。

**エンドポイント**: `GET /projects`

**レスポンス**:
```json
{
  "projects": [
    {
      "name": "project-folder-name",
      "decodedPath": "/path/to/project",
      "sessionCount": 10
    }
  ]
}
```

**フィールド説明**:
- `name`: プロジェクトフォルダ名（エンコード済み）
- `decodedPath`: デコードされたプロジェクトパス
- `sessionCount`: セッション数

**ステータスコード**:
- `200 OK`: 正常
- `500 Internal Server Error`: サーバーエラー

---

### 3. セッション一覧取得

セッション一覧を取得します。

**エンドポイント**: `GET /sessions`

**クエリパラメータ**:
- `project` (optional): プロジェクト名でフィルタ

**レスポンス**:
```json
{
  "sessions": [
    {
      "id": "uuid-session-id",
      "projectName": "project-folder-name",
      "gitBranch": "main",
      "startTime": "2026-01-24T03:24:10.137Z",
      "endTime": "2026-01-24T03:30:00.000Z",
      "totalTokens": 500,
      "errorCount": 0
    }
  ]
}
```

**フィールド説明**:
- `id`: セッションID（UUID）
- `projectName`: プロジェクト名
- `gitBranch`: Gitブランチ名
- `startTime`: セッション開始時刻（ISO 8601形式）
- `endTime`: セッション終了時刻（ISO 8601形式）
- `totalTokens`: 合計トークン数（入力+出力）
- `errorCount`: エラー発生回数

**ステータスコード**:
- `200 OK`: 正常
- `500 Internal Server Error`: サーバーエラー

---

### 4. セッション詳細取得

特定セッションの詳細情報を取得します。

**エンドポイント**: `GET /sessions/{project}/{id}`

**パスパラメータ**:
- `project`: プロジェクト名
- `id`: セッションID

**レスポンス**:
```json
{
  "id": "uuid-session-id",
  "projectName": "project-folder-name",
  "projectPath": "/Users/user/projects/my-project",
  "gitBranch": "main",
  "startTime": "2026-01-24T03:24:10.137Z",
  "endTime": "2026-01-24T03:30:00.000Z",
  "duration": "5m 50s",
  "totalTokens": {
    "inputTokens": 300,
    "outputTokens": 200,
    "cacheCreationInputTokens": 500,
    "cacheReadInputTokens": 600,
    "totalTokens": 500
  },
  "modelUsage": [
    {
      "model": "claude-sonnet-4-20250514",
      "tokens": {
        "inputTokens": 200,
        "outputTokens": 150,
        "cacheCreationInputTokens": 500,
        "cacheReadInputTokens": 400,
        "totalTokens": 350
      }
    }
  ],
  "toolCalls": [
    {
      "timestamp": "2026-01-24T03:24:30.000Z",
      "name": "Bash",
      "input": {
        "command": "ls -la"
      },
      "isError": false
    }
  ],
  "messages": [
    {
      "type": "user",
      "timestamp": "2026-01-24T03:24:15.000Z",
      "content": [...]
    },
    {
      "type": "assistant",
      "timestamp": "2026-01-24T03:24:20.000Z",
      "model": "claude-sonnet-4-20250514",
      "content": [...]
    }
  ],
  "errorCount": 0
}
```

**フィールド説明**:

#### トークンサマリー
- `inputTokens`: 入力トークン数
- `outputTokens`: 出力トークン数
- `cacheCreationInputTokens`: キャッシュ作成トークン数
- `cacheReadInputTokens`: キャッシュ読取トークン数
- `totalTokens`: 合計トークン数（入力+出力）

#### モデル使用量
- `model`: モデル名（例: claude-sonnet-4-20250514）
- `tokens`: モデルごとのトークン使用量

#### ツール呼び出し
- `timestamp`: 呼び出し時刻
- `name`: ツール名（Bash, Read, Edit など）
- `input`: ツールへの入力パラメータ
- `isError`: エラーかどうか

#### メッセージ
- `type`: メッセージタイプ（user / assistant）
- `timestamp`: 送信時刻
- `model`: 使用モデル（assistantの場合）
- `content`: メッセージ内容（配列）

**ステータスコード**:
- `200 OK`: 正常
- `404 Not Found`: セッションが見つからない
- `500 Internal Server Error`: サーバーエラー

---

### 5. ログ解析実行

ログファイルを解析します。

**エンドポイント**: `POST /analyze`

**リクエストボディ** (optional):
```json
{
  "projectNames": ["project-1", "project-2"],
  "force": false
}
```

**フィールド説明**:
- `projectNames` (optional): 解析対象プロジェクト名の配列（省略時は全プロジェクト）
- `force` (optional): 強制再解析フラグ

**レスポンス**:
```json
{
  "status": "completed",
  "sessionsFound": 100,
  "sessionsParsed": 98,
  "errorCount": 2,
  "message": "Analysis completed successfully"
}
```

**フィールド説明**:
- `status`: 解析ステータス（completed / error）
- `sessionsFound`: 発見したセッション数
- `sessionsParsed`: 解析成功したセッション数
- `errorCount`: エラー発生数
- `message`: メッセージ（エラー時）

**ステータスコード**:
- `200 OK`: 正常
- `400 Bad Request`: リクエストが不正
- `500 Internal Server Error`: サーバーエラー

---

## エラーレスポンス

全エンドポイントで共通のエラーレスポンス形式：

```json
{
  "error": "error_code",
  "message": "詳細なエラーメッセージ"
}
```

**エラーコード**:
- `internal_error`: サーバー内部エラー
- `not_found`: リソースが見つからない
- `invalid_request`: リクエストが不正

---

## CORS

開発時は全オリジンを許可（予定）。

本番環境では適切なオリジン制限を設定。

---

## レート制限

現時点では未実装。

将来的に必要に応じて実装予定。
