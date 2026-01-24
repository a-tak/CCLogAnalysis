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

## 開発

### バックエンド開発

サーバーを起動：

```bash
go run ./cmd/server/main.go
```

テストを実行：

```bash
go test ./... -v
```

### フロントエンド開発

```bash
cd web
npm run dev
```

### テスト

全テストを実行：

```bash
# バックエンドテスト
go test ./... -v

# フロントエンドテスト（今後実装）
cd web
npm test
```

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
- [x] テストコード作成（13件）
- [ ] React UIの実装
- [ ] 会話履歴の閲覧
- [ ] トークン使用量の可視化
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
