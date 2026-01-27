#!/bin/bash

# リポジトリルートを取得
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
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

# まだ起動している場合は強制終了（記録されたPIDのみ）
if kill -0 "$PID" 2>/dev/null; then
    echo "⚠️  タイムアウト、サーバーを強制終了します (PID: $PID)..."
    kill -KILL "$PID" 2>/dev/null || true
    sleep 1

    # 最終確認
    if kill -0 "$PID" 2>/dev/null; then
        echo "❌ サーバーの停止に失敗しました (PID: $PID)"
        echo "⚠️  手動での確認が必要です"
        exit 1
    fi
fi

# PIDファイルを削除
rm -f "$PID_FILE"

echo "✅ サーバーの停止処理が完了しました"
exit 0
