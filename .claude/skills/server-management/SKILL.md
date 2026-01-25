---
model: claude-haiku-4-5
name: server-management
description: サーバーの起動・停止を管理
allowed-tools:
  - Bash
  - Read
argument-hint: "[start|stop] [dev|prod]"
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
.claude/skills/server-management/scripts/start-server.sh [dev|prod]
```

**動作:**
1. 既存プロセスをチェック（起動済みならエラー）
2. ポート 8080-8089 から空きポートを自動検出
3. モードに応じた環境変数を設定
   - **dev**: CORS有効、ファイル監視有効
   - **prod**: デフォルト設定
4. サーバーをビルド & バックグラウンド起動
5. PIDとポート番号を `.server.pid` に記録
6. ヘルスチェック（最大30秒待機）
7. 起動成功/失敗を通知

**環境変数（開発モード）:**
- `PORT`: 自動検出されたポート番号
- `ENABLE_CORS=true`
- `ENABLE_FILE_WATCH=true`
- `FILE_WATCH_INTERVAL=15`
- `FILE_WATCH_DEBOUNCE=5`

### サーバー停止（stop-server.sh）

**使用法:**
```bash
.claude/skills/server-management/scripts/stop-server.sh
```

**動作:**
1. `.server.pid` からPIDとポート番号を読み取り
2. プロセス存在確認
3. Graceful Shutdown（SIGTERM送信、最大10秒待機）
4. タイムアウト時は強制終了（SIGKILL）
5. ポートを使用している残存プロセスも確実に終了
6. PIDファイル削除
7. 終了確認通知

## Your task

ユーザーが指定したアクション（start または stop）とモード（dev または prod）に基づいて、適切なスクリプトを実行してください。

### 引数の解析

以下の引数をサポートしています:

**第1引数:**
- `start`: サーバーを起動
- `stop`: サーバーを停止

**第2引数（start のみ）:**
- `dev`: 開発モード（デフォルト）
- `prod`: 本番モード

### 実行例

**開発モードでサーバーを起動:**
```bash
.claude/skills/server-management/scripts/start-server.sh dev
```

**本番モードでサーバーを起動:**
```bash
.claude/skills/server-management/scripts/start-server.sh prod
```

**サーバーを停止:**
```bash
.claude/skills/server-management/scripts/stop-server.sh
```

## ファイル構成

```
.claude/skills/server-management/
├── SKILL.md                      # このファイル
├── scripts/
│   ├── start-server.sh          # サーバー起動スクリプト
│   └── stop-server.sh           # サーバー停止スクリプト
├── .server.pid                  # PIDとポート番号（自動生成、.gitignore対象）
└── server.log                   # サーバーログ（自動生成、.gitignore対象）
```

## 注意事項

- PIDファイル（`.server.pid`）は自動生成され、Git管理外です
- ポート 8080-8089 の範囲で空きポートを自動検出します
- すべてのポートが使用中の場合はエラーになります
- 開発モード・本番モード共にビルド済みバイナリを使用します（PID管理の確実性のため）
