#!/bin/bash

# リポジトリルートを取得
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../" && pwd)"
PID_FILE="$REPO_ROOT/.claude/skills/server-management/.server.pid"

# PIDファイルをチェック
if [ ! -f "$PID_FILE" ]; then
    echo "❌ サーバーは起動していません（PIDファイルが見つかりません）"
    exit 1
fi

# PIDとポート番号を読み取り
PID=$(cut -d: -f1 "$PID_FILE" 2>/dev/null || echo "")
PORT=$(cut -d: -f2 "$PID_FILE" 2>/dev/null || echo "")

if [ -z "$PID" ]; then
    echo "❌ 無効なPIDファイルです"
    rm -f "$PID_FILE"
    exit 1
fi

echo "🛑 サーバーを停止しています (PID: $PID)..."

# プロセスの確認
if ! kill -0 "$PID" 2>/dev/null; then
    echo "⚠️  プロセスが既に終了しています"
    rm -f "$PID_FILE"
    exit 0
fi

# Graceful Shutdown（SIGTERM送信）
kill -TERM "$PID" 2>/dev/null || true

# 最大10秒待機
WAIT_COUNT=0
MAX_WAIT=10

while [ $WAIT_COUNT -lt $MAX_WAIT ]; do
    if ! kill -0 "$PID" 2>/dev/null; then
        echo "✅ サーバーが正常に停止しました"
        break
    fi
    WAIT_COUNT=$((WAIT_COUNT + 1))
    sleep 1
done

# 仍然起動している場合は強制終了
if kill -0 "$PID" 2>/dev/null; then
    echo "⚠️  タイムアウト、サーバーを強制終了します..."
    kill -KILL "$PID" 2>/dev/null || true
    sleep 1
fi

# ポートを使用している残存プロセスも確実に終了
if [ -n "$PORT" ]; then
    REMAINING_PIDS=$(lsof -Pi :$PORT -sTCP:LISTEN -t 2>/dev/null || echo "")
    if [ -n "$REMAINING_PIDS" ]; then
        echo "⚠️  ポート $PORT を使用しているプロセスを終了します..."
        echo "$REMAINING_PIDS" | xargs kill -9 2>/dev/null || true
    fi
fi

# PIDファイルを削除
rm -f "$PID_FILE"

echo "✅ サーバーの停止処理が完了しました"
exit 0
