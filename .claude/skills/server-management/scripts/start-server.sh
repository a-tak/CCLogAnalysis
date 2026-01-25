#!/bin/bash

set -e

# パラメータ解析
MODE="${1:-dev}"

# リポジトリルートを取得
# スクリプト位置: investigate-session-pickup-issue/.claude/skills/server-management/scripts/
# リポジトリルート: investigate-session-pickup-issue/
# したがって: .. => server-management
#             ../.. => skills
#             ../../.. => .claude
#             ../../../.. => investigate-session-pickup-issue
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"
SKILL_DIR="$SCRIPT_DIR/.."
PID_FILE="$SKILL_DIR/.server.pid"
LOG_FILE="$SKILL_DIR/server.log"

# 既存プロセスをチェック
if [ -f "$PID_FILE" ]; then
    OLD_PID=$(cut -d: -f1 "$PID_FILE" 2>/dev/null || echo "")
    if [ -n "$OLD_PID" ] && kill -0 "$OLD_PID" 2>/dev/null; then
        echo "❌ サーバーは既に起動しています (PID: $OLD_PID)"
        exit 1
    fi
fi

# 空きポートを検索
echo "🔍 空きポートを検索中..."
PORT=""
for p in {8080..8089}; do
    if ! lsof -Pi :$p -sTCP:LISTEN -t >/dev/null 2>&1; then
        PORT=$p
        break
    fi
done

if [ -z "$PORT" ]; then
    echo "❌ 空きポートが見つかりません（8080-8089を確認してください）"
    exit 1
fi

echo "✅ ポート $PORT が利用可能です"

# フロントエンドをビルド
echo "🔧 フロントエンドをビルド中..."
cd "$REPO_ROOT/web"
npm run build > /dev/null 2>&1 || {
    echo "❌ フロントエンドのビルドに失敗しました"
    exit 2
}

# バックエンド用のビルトファイルをコピー
mkdir -p "$REPO_ROOT/internal/static/dist"
cp -r "$REPO_ROOT/web/dist"/* "$REPO_ROOT/internal/static/dist/" 2>/dev/null || true

# サーバーをビルド
echo "🔧 サーバーをビルド中..."
cd "$REPO_ROOT"
go build -o ".server_bin" cmd/server/main.go > /dev/null 2>&1 || {
    echo "❌ サーバーのビルドに失敗しました"
    exit 2
}

# 環境変数を設定
export PORT=$PORT
export DB_PATH="$REPO_ROOT/bin/ccloganalysis.db"
case "$MODE" in
    dev)
        export LOG_LEVEL="DEBUG"
        export SKIP_INITIAL_SYNC="1"
        echo "🔧 開発モード(LOG_LEVEL=DEBUG, 初回同期スキップ)でサーバーを起動します..."
        ;;
    prod)
        export LOG_LEVEL="INFO"
        echo "🔧 本番モード(LOG_LEVEL=INFO)でサーバーを起動します..."
        ;;
    *)
        echo "❌ 不正なモード: $MODE (dev または prod を指定してください)"
        exit 1
        ;;
esac

# サーバーをバックグラウンド起動
"$REPO_ROOT/.server_bin" > "$LOG_FILE" 2>&1 &
SERVER_PID=$!

# PIDとポート番号を保存
echo "$SERVER_PID:$PORT" > "$PID_FILE"

# ヘルスチェック（最大30秒待機）
echo "⏳ サーバーのヘルスチェック中..."
HEALTH_CHECK_COUNT=0
MAX_ATTEMPTS=30

while [ $HEALTH_CHECK_COUNT -lt $MAX_ATTEMPTS ]; do
    if curl -s "http://localhost:$PORT/api/health" > /dev/null 2>&1; then
        echo "✅ サーバーが起動しました"
        echo "📍 URL: http://localhost:$PORT"
        echo "📝 ログファイル: $LOG_FILE"
        exit 0
    fi

    # プロセスが生きているかチェック
    if ! kill -0 "$SERVER_PID" 2>/dev/null; then
        echo "❌ サーバープロセスが異常終了しました"
        cat "$LOG_FILE" | tail -20
        exit 1
    fi

    HEALTH_CHECK_COUNT=$((HEALTH_CHECK_COUNT + 1))
    sleep 1
done

echo "❌ ヘルスチェックがタイムアウトしました"
kill "$SERVER_PID" 2>/dev/null || true
exit 1
