# Claude Code ログ解析アプリケーション

Claude Codeのログを解析して、トークン使用量やモデル別の使用状況を可視化するツールです。

## 概要

Claude Codeに対する各種調整（モデル選択、プロンプト設定など）の効果を、ログ解析を通じて定量的に評価・可視化します。

## 技術スタック

### バックエンド
- **言語**: Go 1.21+
- **HTTPサーバー**: Go標準ライブラリ
- **データベース**: SQLite（予定）

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
│   │   ├── service.go
│   │   └── types.go
│   ├── parser/                  # JSONLログパーサー
│   │   ├── parser.go
│   │   ├── parser_test.go
│   │   ├── types.go
│   │   └── testdata/
│   ├── analyzer/                # 集計ロジック（予定）
│   ├── db/                      # データベース操作（予定）
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
./bin/ccloganalysis
```

サーバーは http://localhost:8080 で起動し、フロントエンドとAPIの両方を提供します。

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

## API エンドポイント

### 実装済み

| エンドポイント | メソッド | 説明 |
|---------------|---------|------|
| `/api/health` | GET | ヘルスチェック |
| `/api/projects` | GET | プロジェクト一覧を取得 |
| `/api/sessions` | GET | セッション一覧を取得（クエリパラメータ: `project`） |
| `/api/sessions/{project}/{id}` | GET | セッション詳細を取得 |
| `/api/analyze` | POST | ログ解析を実行 |

### 使用例

```bash
# ヘルスチェック
curl http://localhost:8080/api/health

# プロジェクト一覧
curl http://localhost:8080/api/projects

# セッション一覧（特定プロジェクト）
curl "http://localhost:8080/api/sessions?project=my-project"

# セッション詳細
curl http://localhost:8080/api/sessions/my-project/session-id
```

## ログの場所

Claude Codeのログは以下の場所に保存されています：

- **macOS/Linux**: `~/.claude/projects/`
- **Windows**: `C:\Users\{username}\.claude\projects\`

各プロジェクトごとにフォルダが作成され、セッションIDの`.jsonl`ファイルとして保存されます。

## 開発状況

### Phase 1 (MVP) - 進行中

- [x] プロジェクト初期化
- [x] JSONLログパーサー実装
- [x] REST API実装
  - [x] プロジェクト一覧
  - [x] セッション一覧
  - [x] セッション詳細
  - [x] 解析実行
- [x] テストコード作成（24件）
- [x] React UI基本実装
  - [x] プロジェクト一覧ページ
  - [x] セッション一覧ページ
  - [x] セッション詳細ページ
- [x] Go embedでフロントエンド統合（1バイナリ配布）
- [ ] 会話履歴の閲覧（詳細表示）
- [ ] トークン使用量の可視化（グラフ）
- [ ] モデル別使用量のグラフ

### Phase 2 - 予定

- [ ] エラーパターン検出
- [ ] 要約機能（Claude API連携）
- [ ] 期間別推移グラフ
- [ ] エクスポート機能

## ライセンス

（未定）

## 貢献

（未定）

## 関連ドキュメント

- [要件定義](docs/要件.md)
- [ログフォーマット調査](docs/ログフォーマット調査.md)
- [技術スタック](docs/技術スタック.md)
