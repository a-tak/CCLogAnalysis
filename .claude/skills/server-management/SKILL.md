---
model: claude-haiku-4-5
name: server-management
description: サーバーの起動・停止を管理
allowed-tools:
  - Bash
  - Read
argument-hint: "[start|stop|status] [dev|prod] [--foreground]"
---

# server-management

## Context

**CCLogAnalysis サーバーの起動・停止を管理するスキルです。**

このスキルは、開発中に複数のサーバーインスタンスが必要な場合でも、自動的に空きポートを見つけて起動し、確実に停止することができます。

## スキル実行モデル

このスキルは **スクリプト実行モデル** で動作します：

- **scripts/start-server.sh**: サーバー起動スクリプト
- **scripts/stop-server.sh**: サーバー停止スクリプト
- Agent は適切なスクリプトを Bash ツールで実行

**利点:**
- プロセス管理とPIDファイル管理をスクリプトに集約
- 空きポート自動検出機能により、複数インスタンスの同時実行が可能
- Graceful Shutdown と強制終了の両方をサポート

## 機能

### サーバー起動（start-server.sh）

**使用法:**
```bash
.claude/skills/server-management/scripts/start-server.sh [dev|prod] [--foreground|-f]
```

**オプション:**
- `dev`: 開発モード（デフォルト、LOG_LEVEL=DEBUG、初回同期スキップ）
- `prod`: 本番モード（LOG_LEVEL=INFO）
- `--foreground` / `-f`: フォアグラウンドで起動（別ターミナル推奨）

**動作（バックグラウンドモード）:**
1. 既存プロセスをチェック（起動済みならエラー）
2. ポート 8080-8089 から空きポートを自動検出
3. モードに応じた環境変数を設定
4. サーバーをビルド & バックグラウンド起動
5. PIDとポート番号を `.server.pid` に記録
6. ヘルスチェック（最大30秒待機）
7. 起動成功/失敗を通知

**動作（フォアグラウンドモード）:**
1. ポート検出とビルドは同じ
2. サーバーをフォアグラウンドで起動
3. ログが直接ターミナルに表示される
4. Ctrl+C で安全に停止できる
5. PIDファイルは作成されない（停止スクリプト不要）

**環境変数:**
- `PORT`: 自動検出されたポート番号
- `DB_PATH`: データベースパス
- `LOG_LEVEL`: dev=DEBUG, prod=INFO
- `SKIP_INITIAL_SYNC`: dev=1（初回同期スキップ）

### サーバー停止（stop-server.sh）

**使用法:**
```bash
.claude/skills/server-management/scripts/stop-server.sh
```

**動作:**
1. `.server.pid` からPIDとポート番号を読み取り
2. プロセス存在確認
3. Graceful Shutdown（SIGTERM送信、最大10秒待機）
4. タイムアウト時は強制終了（記録されたPIDのみ）
5. PIDファイル削除
6. 終了確認通知

**安全性:**
- 記録されたPIDのみを対象に終了処理を実行
- 他のプロセスに影響を与えない

### サーバー状態確認（status-server.sh）

**使用法:**
```bash
.claude/skills/server-management/scripts/status-server.sh
```

**表示内容:**
- サーバーの起動状態（起動中/停止中/異常）
- PID、ポート番号、URL
- ヘルスチェック結果
- ログファイルのパスとサイズ

## Your task

ユーザーが指定したアクション（start / stop / status）とオプションに基づいて、適切なスクリプトを実行してください。

### 引数の解析

以下のアクションをサポートしています:

**アクション:**
- `start`: サーバーを起動
- `stop`: サーバーを停止
- `status`: サーバーの状態を確認

**start のオプション:**
- `dev`: 開発モード（デフォルト）
- `prod`: 本番モード
- `--foreground` / `-f`: フォアグラウンドで起動

### 実行例

**開発モードでサーバーを起動（バックグラウンド）:**
```bash
.claude/skills/server-management/scripts/start-server.sh dev
```

**開発モードでサーバーを起動（フォアグラウンド）:**
```bash
.claude/skills/server-management/scripts/start-server.sh dev --foreground
```

**本番モードでサーバーを起動:**
```bash
.claude/skills/server-management/scripts/start-server.sh prod
```

**サーバーを停止:**
```bash
.claude/skills/server-management/scripts/stop-server.sh
```

**サーバーの状態を確認:**
```bash
.claude/skills/server-management/scripts/status-server.sh
```

### 重要な注意事項

**⚠️ 絶対にやってはいけないこと:**
- スキルを使わずに直接 `kill` コマンドを実行する
- `lsof` でポートを検索してプロセスを終了する
- PIDファイル以外の方法でプロセスを特定する

**✅ 正しい方法:**
- **必ず** このスキルのスクリプトを使用する
- サーバーの状態確認は `status` コマンドを使う
- フォアグラウンド起動時は、ユーザーに別ターミナルでの実行を推奨する

## ファイル構成

```
.claude/skills/server-management/
├── SKILL.md                      # このファイル
├── scripts/
│   ├── start-server.sh          # サーバー起動スクリプト
│   ├── stop-server.sh           # サーバー停止スクリプト
│   └── status-server.sh         # サーバー状態確認スクリプト
└── .server.pid                  # PIDとポート番号（自動生成、.gitignore対象）

プロジェクトルート:
└── server.log                   # サーバーログ（自動生成、.gitignore対象）
```

## 注意事項

- PIDファイル（`.server.pid`）は自動生成され、Git管理外です
- ポート 8080-8089 の範囲で空きポートを自動検出します
- すべてのポートが使用中の場合はエラーになります
- 開発モード・本番モード共にビルド済みバイナリを使用します（PID管理の確実性のため）
- フォアグラウンドモードでは、PIDファイルは作成されません（Ctrl+Cで停止）
- フォアグラウンドモードは別ターミナルでの実行を推奨します
