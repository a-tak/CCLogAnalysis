# Claude Code ログ解析アプリケーション

Claude Codeのログを解析して、トークン使用量やモデル別の使用状況を可視化するツールです。

## 概要

Claude Codeに対する各種調整（モデル選択、プロンプト設定など）の効果を、ログ解析を通じて定量的に評価・可視化します。

## バイナリダウンロード

最新リリースは [GitHub Releases](https://github.com/{username}/{project-name}/releases) からダウンロードできます。

### 対応プラットフォーム

- **Windows**: amd64
- **macOS**: Intel (amd64), Apple Silicon (arm64)

### インストール

1. 最新リリースから環境に合ったファイルをダウンロード
2. アーカイブを解凍
3. 実行ファイルをパスの通ったディレクトリに配置（オプション）

### 実行

```bash
# macOS / Linux
./ccloganalysis

# Windows
ccloganalysis.exe
```

ブラウザで http://localhost:8080 にアクセスすると、React UIが表示されます。

## 技術スタック

### バックエンド
- **言語**: Go 1.21+
- **HTTPサーバー**: Go標準ライブラリ
- **データベース**: SQLite（modernc.org/sqlite - Pure Go、cgo不要）
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
{project-name}/
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
git clone https://github.com/{username}/{project-name}.git
cd {project-name}
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

#### server-managementスキル（推奨）

Claude Code から以下のコマンドでサーバーを管理できます：

```bash
# 開発モードで起動（バックグラウンド）
/server-management start dev

# 開発モードで起動（フォアグラウンド、別ターミナル推奨）
/server-management start dev --foreground

# 本番モードで起動
/server-management start prod

# サーバー状態確認
/server-management status

# サーバー停止
/server-management stop
```

**利点:**
- 空きポート自動検出（8080-8089）
- PIDファイルによる確実なプロセス管理
- Graceful shutdown対応
- ビルド & 起動を自動実行
- フォアグラウンドモード対応（別ターミナルで起動可能）
- 状態確認コマンドで起動状態を簡単に確認

**フォアグラウンドモード:**
- ログが直接ターミナルに表示される
- Ctrl+C で安全に停止できる
- 別ターミナルでの実行を推奨

#### 手動起動

```bash
# 推奨: bin/ディレクトリのデータベースを使用
DB_PATH=./bin/ccloganalysis.db ./bin/ccloganalysis

# または、実行ファイルと同じディレクトリのDBを使用（デフォルト）
./bin/ccloganalysis
```

**アクセス:**
- ブラウザで http://localhost:8080 にアクセスするとReact UIが表示されます
- server-managementで起動時は、ログに表示されるURLを使用してください

**データベース:**
- 推奨パス: `bin/ccloganalysis.db`
- デフォルト: 実行ファイルと同じディレクトリの `ccloganalysis.db`
- 初回起動時に自動的にClaudeプロジェクトのログを同期します

⚠️ **重要**: 複数の実行ファイルがある場合、それぞれが別のデータベースを使用します。server-managementスキルの使用を推奨します。

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

### サーバー管理スクリプト

Claude Codeからサーバーの起動・停止・状態確認を簡単に行うためのスクリプトを提供しています。

**起動:**
```bash
# バックグラウンドモード（開発）
.claude/skills/server-management/scripts/start-server.sh dev

# フォアグラウンドモード（開発、別ターミナル推奨）
.claude/skills/server-management/scripts/start-server.sh dev --foreground

# 本番モード
.claude/skills/server-management/scripts/start-server.sh prod
```

**状態確認:**
```bash
.claude/skills/server-management/scripts/status-server.sh
```

**停止:**
```bash
.claude/skills/server-management/scripts/stop-server.sh
```

**ログ確認（バックグラウンドモード時）:**
```bash
tail -f .claude/skills/server-management/server.log
```

**開発モードのデフォルト環境変数:**
- `PORT=8080`
- `ENABLE_CORS=true`
- `ENABLE_FILE_WATCH=true`
- `FILE_WATCH_INTERVAL=15`
- `FILE_WATCH_DEBOUNCE=5`

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

### ワークツリー管理

`/cr-worktree` スキルを使用して、効率的に新しい機能開発やバグ修正を開始できます。

**基本的な使い方:**

```bash
# Issue番号から自動的にブランチを作成（Issueステータスも自動更新）
/cr-worktree 123

# ブランチ名を直接指定
/cr-worktree feature/new-feature

# 説明文から自動的にブランチ名を生成
/cr-worktree "ログパーサーのバグ修正"

# 現在のブランチから分岐
/cr-worktree 123 --from-current
```

**何が起こるか:**

1. メインブランチの同期確認（最新のコードを取得）
2. ワークツリーの作成（`CCLogAnalysis.worktrees/<branch-name>/`）
3. 環境整備:
   - Go依存関係のインストール（`go mod download`）
   - Node.js依存関係のインストール（`npm ci`）
   - テスト実行（`make test`）
4. 新しいターミナルウィンドウでClaude Code起動

**Issue番号指定の利点:**

- GitHub Issueから自動的にブランチ名を決定
- Issueステータスを自動的に "In progress" に更新
- Claude Code起動時に `/issue` コマンドを自動実行

**ワークツリーの場所:**

```
{home-directory}/Documents/GitHub/
├── {project-name}/              # メインリポジトリ
└── {project-name}.worktrees/    # ワークツリー用ディレクトリ
    ├── feature-xyz/
    ├── fix-abc/
    └── 123-issue-title/
```

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
| `LOG_LEVEL` | ログレベル（DEBUG, INFO, WARN, ERROR） | `INFO` | `DEBUG` |

**ログレベルについて**:
- `DEBUG`: 詳細なデバッグ情報を出力（開発時推奨）
- `INFO`: 通常の動作情報を出力（本番環境推奨）
- `WARN`: 警告以上を出力
- `ERROR`: エラーのみを出力

### ファイル監視設定

| 変数名 | 説明 | デフォルト値 | 範囲 | 例 |
|--------|------|--------------|------|-----|
| `ENABLE_FILE_WATCH` | ファイル監視機能の有効化 | `false` | `true`/`false` | `true` |
| `FILE_WATCH_INTERVAL` | スキャン間隔（秒） | `15` | 5～3600 | `30` |
| `FILE_WATCH_DEBOUNCE` | デバウンス時間（秒） | `5` | 1～60 | `10` |

**ファイル監視機能について**:
- 新しいログファイル（`.jsonl`）が追加されると、自動的にデータベースに同期されます
- スキャン間隔で定期的にファイルシステムをチェックします
- デバウンス時間により、短時間の連続同期を抑制して負荷を軽減します
- デフォルトは無効です（既存動作への影響を最小化）

### 使用例

```bash
# カスタムポートで起動
PORT=3000 ./bin/ccloganalysis

# 開発モード（CORS有効）
ENABLE_CORS=true go run ./cmd/server/main.go

# カスタムプロジェクトディレクトリとDB
CLAUDE_PROJECTS_DIR=/custom/path DB_PATH=/custom/db.sqlite ./bin/ccloganalysis

# ファイル監視を有効化（15秒ごとにスキャン）
ENABLE_FILE_WATCH=true ./bin/ccloganalysis

# ファイル監視を有効化（カスタム間隔: 30秒ごと、デバウンス: 10秒）
ENABLE_FILE_WATCH=true FILE_WATCH_INTERVAL=30 FILE_WATCH_DEBOUNCE=10 ./bin/ccloganalysis
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

### データの永続性

**重要**: 一度データベースに取り込まれたデータは、自動的に削除されることはありません。

**具体的な挙動**:

1. **ワークツリーが削除された場合**
   - ディスクからワークツリーのディレクトリを削除しても、データベース内のデータは保持されます
   - 削除されたワークツリーのセッション履歴も引き続き閲覧可能です
   - グループ一覧では、名前に"worktree"を含むグループは自動的に非表示になります
   - グループ詳細ページでは、削除済みワークツリーのプロジェクトも表示されます

2. **ログファイルが削除された場合**
   - Claude Codeのログファイル（`.jsonl`）が日数経過で削除されても、データベース内のデータは影響を受けません
   - すでに取り込まれたセッションは永続的に保持されます

3. **データのクリーンアップ**
   - 現在、自動クリーンアップ機能はありません
   - 不要なデータを削除したい場合は、手動でデータベースファイルを削除して再構築してください

**メリット**:
- 過去のすべてのセッション履歴を保持できる
- 削除されたワークツリーのデータも分析可能
- ログファイルのローテーション後もデータが残る

**注意点**:
- データベースサイズは時間とともに増加し続けます
- 定期的にデータベースファイルを確認し、必要に応じて再構築することを推奨します

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
