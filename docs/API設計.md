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
      "errorCount": 0,
      "firstUserMessage": "セッションリストですが、現在はセッションIDだけでは内容がわからないため、開始時間以外にセッションを選択する基準がないです。セッションの最初の会話が少しリスト..."
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
- `firstUserMessage`: 最初のユーザーメッセージ（100文字まで、それ以上は切り詰め）

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

---

## グループ関連エンドポイント

### 12. プロジェクトグループ一覧取得

プロジェクトグループの一覧を取得します。

**エンドポイント**: `GET /groups`

**レスポンス**:
```json
{
  "groups": [
    {
      "id": 1,
      "name": "CCLogAnalysis",
      "gitRoot": "/Users/{username}/Documents/GitHub/CCLogAnalysis",
      "createdAt": "2026-01-25T12:21:39Z",
      "updatedAt": "2026-01-25T12:21:39Z"
    }
  ]
}
```

**フィールド説明**:
- `id`: グループID
- `name`: グループ名
- `gitRoot`: Gitリポジトリのルートパス
- `createdAt`: グループ作成時刻（ISO 8601形式）
- `updatedAt`: グループ更新時刻（ISO 8601形式）

**ステータスコード**:
- `200 OK`: 正常
- `500 Internal Server Error`: サーバーエラー

---

### 13. プロジェクトグループ詳細取得

グループに属するプロジェクト一覧を含む詳細情報を取得します。

**エンドポイント**: `GET /groups/{id}`

**パスパラメータ**:
- `id`: グループID

**レスポンス**:
```json
{
  "id": 1,
  "name": "CCLogAnalysis",
  "gitRoot": "/Users/{username}/Documents/GitHub/CCLogAnalysis",
  "createdAt": "2026-01-25T12:21:39Z",
  "updatedAt": "2026-01-25T12:21:39Z",
  "projects": [
    {
      "name": "project-1",
      "decodedPath": "/Users/{username}/Documents/GitHub/project1",
      "sessionCount": 53
    }
  ]
}
```

**ステータスコード**:
- `200 OK`: 正常
- `400 Bad Request`: グループIDが不正
- `404 Not Found`: グループが見つからない
- `500 Internal Server Error`: サーバーエラー

---

### 14. プロジェクトグループ統計取得

グループ内の全プロジェクトの統計情報を取得します。

**エンドポイント**: `GET /groups/{id}/stats`

**パスパラメータ**:
- `id`: グループID

**レスポンス**:
```json
{
  "totalProjects": 4,
  "totalSessions": 75,
  "totalInputTokens": 273957,
  "totalOutputTokens": 23199,
  "totalCacheCreationTokens": 20543766,
  "totalCacheReadTokens": 418417015,
  "avgTokens": 3962.08,
  "firstSession": "2026-01-20T10:00:00Z",
  "lastSession": "2026-01-25T15:30:00Z",
  "errorRate": 0.573
}
```

**フィールド説明**:
- `totalProjects`: グループに属するプロジェクト数
- `totalSessions`: グループ全体のセッション数
- `totalInputTokens`: 全体の入力トークン数
- `totalOutputTokens`: 全体の出力トークン数
- `totalCacheCreationTokens`: キャッシュ作成トークン数
- `totalCacheReadTokens`: キャッシュ読み取りトークン数
- `avgTokens`: セッション当たりの平均トークン数
- `firstSession`: 最初のセッション開始時刻
- `lastSession`: 最後のセッション終了時刻
- `errorRate`: エラー発生率

**ステータスコード**:
- `200 OK`: 正常
- `400 Bad Request`: グループIDが不正
- `404 Not Found`: グループが見つからない
- `500 Internal Server Error`: サーバーエラー

---

### 15. プロジェクトグループタイムライン統計取得

グループ内の全プロジェクトの時系列統計を取得します。

**エンドポイント**: `GET /groups/{id}/timeline`

**パスパラメータ**:
- `id`: グループID

**クエリパラメータ**:
- `period` (optional): 集計期間 ("day" | "week" | "month", default: "day")
- `limit` (optional): 取得するデータポイント数 (default: 30)

**レスポンス**:
```json
{
  "period": "day",
  "data": [
    {
      "periodStart": "2026-01-25T00:00:00Z",
      "periodEnd": "2026-01-25T00:00:00Z",
      "sessionCount": 27,
      "totalInputTokens": 144778,
      "totalOutputTokens": 12082,
      "totalCacheCreationTokens": 20543766,
      "totalCacheReadTokens": 418417015,
      "totalTokens": 156860
    },
    {
      "periodStart": "2026-01-24T00:00:00Z",
      "periodEnd": "2026-01-24T00:00:00Z",
      "sessionCount": 47,
      "totalInputTokens": 129179,
      "totalOutputTokens": 11117,
      "totalCacheCreationTokens": 19731593,
      "totalCacheReadTokens": 370148894,
      "totalTokens": 140296
    }
  ]
}
```

**フィールド説明**:
- `period`: 集計期間
- `data`: 時系列データ配列
  - `periodStart`: 期間開始日（その期間の最初のセッションの日付）
  - `periodEnd`: 期間終了日（その期間の最後のセッションの日付）
  - `sessionCount`: その期間のセッション数
  - `totalInputTokens`: その期間の入力トークン合計
  - `totalOutputTokens`: その期間の出力トークン合計
  - `totalCacheCreationTokens`: その期間のキャッシュ作成トークン合計
  - `totalCacheReadTokens`: その期間のキャッシュ読み取りトークン合計
  - `totalTokens`: その期間の総トークン数（入力+出力）

**ステータスコード**:
- `200 OK`: 正常
- `400 Bad Request`: グループIDが不正、またはperiodパラメータが不正
- `404 Not Found`: グループが見つからない
- `500 Internal Server Error`: サーバーエラー

---

将来的に必要に応じて実装予定。
