---
paths:
  - "internal/api/**/*.go"
---

# API Development Rules

## 責務

- REST APIエンドポイントの提供
- リクエスト/レスポンスの処理
- ビジネスロジックの実装（serviceレイヤー）

## テスト駆動開発（TDD）

- 各ハンドラーにはテストを必須で作成
- モックサービスを使用してハンドラーをテスト
- HTTPステータスコードを正しくテスト
- レスポンスボディの構造をテスト

## エンドポイント設計

- RESTful設計に従う
- 全エンドポイントは `/api` プレフィックスを使用
- バージョニングが必要な場合は `/api/v1` の形式
- エラーレスポンスは統一形式を使用

## エラーハンドリング

- 適切なHTTPステータスコードを返す
  - `200 OK`: 成功
  - `400 Bad Request`: リクエストが不正
  - `404 Not Found`: リソースが見つからない
  - `500 Internal Server Error`: サーバーエラー
- エラーレスポンスは `ErrorResponse` 型を使用
- `Content-Type: application/json` を必ず設定

## 依存関係

- `internal/parser` は使用可能
- データベース層が実装されたら `internal/db` を使用
- Go標準ライブラリのHTTPサーバーを使用（外部フレームワーク不要）
