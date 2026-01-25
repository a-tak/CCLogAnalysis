#!/bin/bash
set -e

PID_FILE=".claude/skills/server-management/.server.pid"
HEALTH_URL="http://localhost:8080/api/health"
MODE="${1:-dev}"

# 1. 既存プロセスチェック
if [ -f "$PID_FILE" ]; then
  PID=$(cat "$PID_FILE")
  if kill -0 "$PID" 2>/dev/null; then
    echo "❌ エラー: サーバーは既に起動中です（PID: $PID）"
    exit 1
  else
    echo "⚠️  警告: PIDファイルが残っていましたが、プロセスは終了しています"
    rm "$PID_FILE"
  fi
fi

# 2. モード別起動
if [ "$MODE" = "dev" ]; then
  echo "🔧 開発モードでサーバーを起動します..."
  export PORT=8080
  export ENABLE_CORS=true
  export ENABLE_FILE_WATCH=true
  export FILE_WATCH_INTERVAL=15
  export FILE_WATCH_DEBOUNCE=5

  nohup make dev > .claude/skills/server-management/server.log 2>&1 &
  PID=$!

elif [ "$MODE" = "prod" ]; then
  echo "🚀 本番モードでサーバーを起動します..."
  make build
  nohup ./bin/ccloganalysis > .claude/skills/server-management/server.log 2>&1 &
  PID=$!
else
  echo "❌ エラー: 不明なモード '$MODE'"
  echo "使用法: $0 [dev|prod]"
  exit 1
fi

# 3. PID記録
echo "$PID" > "$PID_FILE"
echo "📝 PIDファイルに記録しました: $PID"

# 4. ヘルスチェック（最大30秒待機）
echo "🔍 サーバーの起動を確認中..."
RETRY_COUNT=0
MAX_RETRIES=30

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
  if curl -s -f "$HEALTH_URL" > /dev/null 2>&1; then
    echo "✅ サーバーが正常に起動しました"
    echo "   URL: http://localhost:8080"
    echo "   PID: $PID"
    echo "   ログ: .claude/skills/server-management/server.log"
    exit 0
  fi

  sleep 1
  RETRY_COUNT=$((RETRY_COUNT + 1))
done

echo "❌ エラー: サーバーの起動に失敗しました"
echo "   ログを確認してください: .claude/skills/server-management/server.log"
exit 1
