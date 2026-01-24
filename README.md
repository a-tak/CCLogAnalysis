# Claude Code ログ解析アプリケーション

Claude Codeのログを解析して、トークン使用量やモデル別の使用状況を可視化するツールです。

## 概要

Claude Codeに対する各種調整（モデル選択、プロンプト設定など）の効果を、ログ解析を通じて定量的に評価・可視化します。

## 技術スタック

### バックエンド
- **言語**: Go 1.21+
- **HTTPサーバー**: Go標準ライブラリ
- **データベース**: SQLite（go-sqlite3）
- **データモード**: SQLiteデータベース

### フロントエンド
- **フレームワーク**: React + TypeScript
- **スタイリング**: Tailwind CSS + shadcn/ui
- **グラフ**: Recharts
- **ビルドツール**: Vite

### 配布
- 1バイナリ配布（Go embedでReactビルド成果物を埋め込み）
- 対応OS: Windows + Mac

## プロジェクト構成

```
CCLogAnalysis/
├── cmd/
│   └── server/
│       └── main.go              # サーバーエントリポイント
├── internal/
│   ├── api/                     # REST APIハンドラー
│   │   ├── router.go
│   │   ├── router_test.go
│   │   ├── service_db.go        # データベース版Service
│   │   ├── service_db_test.go
│   │   └── types.go
│   ├── parser/                  # JSONLログパーサー
│   │   ├── parser.go
│   │   ├── parser_test.go
│   │   ├── types.go
│   │   └── testdata/
│   ├── db/                      # SQLiteデータベース層
│   │   ├── schema.sql           # スキーマ定義
│   │   ├── db.go                # DB接続管理
│   │   ├── projects.go          # プロジェクトCRUD
│   │   ├── sessions.go          # セッションCRUD
│   │   ├── sync.go              # ファイルシステム同期
│   │   └── *_test.go            # テストファイル
│   ├── static/                  # 埋め込み静的ファイル
│   ├── analyzer/                # 集計ロジック（予定）
│   └── config/                  # 設定管理（予定）
├── web/                         # Reactプロジェクト
│   ├── src/
│   └── package.json
├── docs/                        # ドキュメント
│   ├── 要件.md
│   ├── ログフォーマット調査.md
│   └── 技術スタック.md
├── go.mod
└── README.md
```

## セットアップ

### 必要な環境

- Go 1.21+
- Node.js 20+
- npm または pnpm

### インストール

1. リポジトリをクローン

```bash
git clone https://github.com/{username}/CCLogAnalysis.git
cd CCLogAnalysis
```

2. Go依存関係のインストール

```bash
go mod download
```

3. React依存関係のインストール

```bash
cd web
npm install
```

## ビルド

### 本番用ビルド

```bash
make build
```

これにより：
1. Reactフロントエンドがビルドされ `web/dist/` に出力
2. ビルド成果物が `internal/static/dist/` にコピー
3. Goバイナリにフロントエンドが埋め込まれる
4. `bin/ccloganalysis` が生成される

### 実行

```bash
# デフォルトDBパス使用（実行ファイルと同じディレクトリ）
./bin/ccloganalysis

# カスタムDBパス指定
DB_PATH=/path/to/custom.db ./bin/ccloganalysis
```

サーバーは http://localhost:8080 で起動し、フロントエンドとAPIの両方を提供します。

**データベース:**
- デフォルトパス: `bin/ccloganalysis.db`（実行ファイルと同じディレクトリ）
- 初回起動時に自動的にClaudeプロジェクトのログを同期します

ブラウザで http://localhost:8080 にアクセスするとReact UIが表示されます。

### クリーンビルド

```bash
make clean
make build
```

## 開発

### 開発モード

開発時はフロントエンドとバックエンドを別々に起動することを推奨：

```bash
# ターミナル1: バックエンド（CORS有効）
make dev
# または
ENABLE_CORS=true go run ./cmd/server/main.go

# ターミナル2: フロントエンド（HMR有効）
cd web && npm run dev
```

この方式では：
- バックエンド: http://localhost:8080
- フロントエンド: http://localhost:5173（Vite開発サーバー）

フロントエンド開発時はViteの高速なHMR（Hot Module Replacement）が利用できます。

### バックエンド開発

サーバーを起動（フロントエンド埋め込み版）：

```bash
make build
./bin/ccloganalysis
```

テストを実行：

```bash
go test ./... -v
# または
make test
```

### フロントエンド開発

```bash
cd web
npm run dev
```

### Makeコマンド一覧

```bash
make help
```

利用可能なコマンド：
- `make build` - フロントエンド→バックエンドの順でビルド
- `make build-frontend` - フロントエンドのみビルド
- `make build-backend` - バックエンドのみビルド
- `make test` - 全テスト実行
- `make clean` - ビルド成果物の削除
- `make dev` - 開発サーバー起動（CORS有効）
- `make run` - ビルド後に実行
- `make help` - ヘルプ表示

## 環境変数

サーバーの動作を環境変数で制御できます。

### データベース設定

| 変数名 | 説明 | デフォルト値 | 例 |
|--------|------|--------------|-----|
| `DB_PATH` | データベースファイルのパス | `bin/ccloganalysis.db`（実行ファイルと同じディレクトリ） | `/path/to/custom.db` |

### サーバー設定

| 変数名 | 説明 | デフォルト値 | 例 |
|--------|------|--------------|-----|
| `PORT` | サーバーポート番号 | `8080` | `3000` |
| `CLAUDE_PROJECTS_DIR` | Claudeプロジェクトディレクトリ | `~/.claude/projects` | `/custom/path/to/projects` |
| `ENABLE_CORS` | CORS有効化（開発用） | `false` | `true` |

### 使用例

```bash
# カスタムポートで起動
PORT=3000 ./bin/ccloganalysis

# 開発モード（CORS有効）
ENABLE_CORS=true go run ./cmd/server/main.go

# カスタムプロジェクトディレクトリとDB
CLAUDE_PROJECTS_DIR=/custom/path DB_PATH=/custom/db.sqlite ./bin/ccloganalysis
```

## API エンドポイント

### 実装済み

| エンドポイント | メソッド | 説明 |
|---------------|---------|------|
| `/api/health` | GET | ヘルスチェック |
| `/api/projects` | GET | プロジェクト一覧を取得 |
| `/api/sessions` | GET | セッション一覧を取得（クエリパラメータ: `project`） |
| `/api/sessions/{project}/{id}` | GET | セッション詳細を取得 |
| `/api/analyze` | POST | ログ解析を実行（DB版：同期実行） |

### 使用例

```bash
# ヘルスチェック
curl http://localhost:8080/api/health

# プロジェクト一覧
curl http://localhost:8080/api/projects

# セッション一覧（全プロジェクト）
curl http://localhost:8080/api/sessions

# セッション一覧（特定プロジェクト）
curl "http://localhost:8080/api/sessions?project=my-project"

# セッション詳細
curl http://localhost:8080/api/sessions/my-project/session-id

# ログ解析・同期（DB版のみ）
# 全プロジェクトを同期
curl -X POST http://localhost:8080/api/analyze

# 特定プロジェクトを同期
curl -X POST http://localhost:8080/api/analyze \
  -H "Content-Type: application/json" \
  -d '{"projectNames":["my-project"]}'
```

## ログの場所

Claude Codeのログは以下の場所に保存されています：

- **macOS/Linux**: `~/.claude/projects/`
- **Windows**: `C:\Users\{username}\.claude\projects\`

各プロジェクトごとにフォルダが作成され、セッションIDの`.jsonl`ファイルとして保存されます。

## データベース版について

### 概要

データベース版では、ファイルシステム上のJSONLログをSQLiteデータベースに同期して管理します。

**利点**:
- 高速なクエリ実行
- 複雑な集計・分析が可能
- 将来的な拡張機能（統計、エラーパターン検出など）の基盤

すべての操作はSQLiteデータベースを通じて実行されます。

### データベーススキーマ

**Phase 1（実装済み）**:
- `projects`: プロジェクト情報
- `sessions`: セッション基本情報とトークン集計
- `model_usage`: モデル別トークン使用量
- `log_entries`: ログエントリ
- `messages`: メッセージ本文（検索用）
- `tool_calls`: ツール呼び出し履歴

**Phase 2（スキーマのみ、実装は将来）**:
- `project_groups`, `project_group_mappings`: Gitワークツリー対応
- `error_patterns`, `error_occurrences`: エラーパターン検出
- `period_statistics`: 期間別統計キャッシュ

### 同期機能

初回起動時、データベースが空の場合は自動的にログを同期します。

手動で再同期したい場合は `/api/analyze` エンドポイントを使用できます：

```bash
# サーバー起動
./bin/ccloganalysis

# ログを手動で再同期（全プロジェクト）
curl -X POST http://localhost:8080/api/analyze
```

**同期の挙動**:
- 既存のセッションは自動的にスキップ（重複なし）
- 新しいセッションのみがDBに追加される
- エラーが発生しても処理は継続される

### データベースファイルの場所

デフォルトのデータベースパス:
- **macOS/Linux**: `~/.claude/ccloganalysis.db`
- **Windows**: `C:\Users\{username}\.claude\ccloganalysis.db`

環境変数 `DB_PATH` でカスタマイズ可能です。

## 開発状況

### Phase 1 (MVP) - ✅ 完了

- [x] プロジェクト初期化
- [x] JSONLログパーサー実装
- [x] REST API実装
  - [x] プロジェクト一覧
  - [x] セッション一覧
  - [x] セッション詳細
  - [x] 解析実行
- [x] テストコード作成（50+件）
- [x] React UI基本実装
  - [x] プロジェクト一覧ページ
  - [x] セッション一覧ページ
  - [x] セッション詳細ページ
  - [x] 会話履歴の閲覧
  - [x] トークン使用量の可視化（グラフ）
- [x] Go embedでフロントエンド統合（1バイナリ配布）
- [x] **SQLiteデータベース層実装**
  - [x] スキーマ定義（Phase 1 & 2）
  - [x] プロジェクト・セッションCRUD
  - [x] ファイルシステム→DB同期機能
  - [x] 環境変数による切り替え機能
  - [x] 包括的なテストカバレッジ

### Phase 2 - 予定

- [ ] エラーパターン検出（DB基盤は実装済み）
- [ ] 期間別統計（DB基盤は実装済み）
- [ ] プロジェクトグループ機能（DB基盤は実装済み）
- [ ] メッセージ全文検索（FTS）
- [ ] 要約機能（Claude API連携）
- [ ] エクスポート機能

## ライセンス

（未定）

## 貢献

（未定）

## 関連ドキュメント

- [要件定義](docs/要件.md)
- [ログフォーマット調査](docs/ログフォーマット調査.md)
- [技術スタック](docs/技術スタック.md)
