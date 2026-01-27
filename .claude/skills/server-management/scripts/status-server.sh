#!/bin/bash

# リポジトリルートを取得
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
PID_FILE="$REPO_ROOT/.claude/skills/server-management/.server.pid"
LOG_FILE="$REPO_ROOT/server.log"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 CCLogAnalysis サーバー状態"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# PIDファイルをチェック
if [ ! -f "$PID_FILE" ]; then
    echo "状態: ⚫ 停止中"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    exit 0
fi

# PIDとポート番号を読み取り
PID=$(cut -d: -f1 "$PID_FILE" 2>/dev/null || echo "")
PORT=$(cut -d: -f2 "$PID_FILE" 2>/dev/null || echo "")

if [ -z "$PID" ]; then
    echo "状態: ❌ エラー（無効なPIDファイル）"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    exit 1
fi

# プロセスの確認
if ! kill -0 "$PID" 2>/dev/null; then
    echo "状態: ⚠️  異常（PIDファイルは存在しますが、プロセスが見つかりません）"
    echo "PID: $PID (プロセス不在)"
    echo ""
    echo "💡 ヒント: '/server-management stop' で状態をクリアしてください"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    exit 0
fi

# 起動中
echo "状態: ✅ 起動中"
echo "PID: $PID"
echo "ポート: $PORT"
echo "URL: http://localhost:$PORT"
echo ""

# ヘルスチェック
if curl -s "http://localhost:$PORT/api/health" > /dev/null 2>&1; then
    echo "ヘルスチェック: ✅ 正常"
else
    echo "ヘルスチェック: ⚠️  応答なし（起動中の可能性があります）"
fi

# ログファイルの確認
if [ -f "$LOG_FILE" ]; then
    echo "ログファイル: $LOG_FILE"
    LOG_SIZE=$(du -h "$LOG_FILE" | cut -f1)
    echo "ログサイズ: $LOG_SIZE"
fi

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
