#!/bin/bash
set -e

PID_FILE=".claude/skills/server-management/.server.pid"
MODE="${1:-dev}"

# 1. 既存プロセスチェック
if [ -f "$PID_FILE" ]; then
  read EXISTING_PID EXISTING_PORT < "$PID_FILE"
  if kill -0 "$EXISTING_PID" 2>/dev/null; then
    echo "❌ エラー: サーバーは既に起動中です（PID: $EXISTING_PID, ポート: $EXISTING_PORT）"
    exit 1
  else
    echo "⚠️  警告: PIDファイルが残っていましたが、プロセスは終了しています"
    rm "$PID_FILE"
  fi
fi

# 2. 空きポートを探す（8080から8089まで）
echo "🔍 空きポートを検索中..."
PORT=""
for port in {8080..8089}; do
  if ! lsof -i :$port -t >/dev/null 2>&1; then
    PORT=$port
    echo "✅ ポート $PORT が利用可能です"
    break
  fi
done

if [ -z "$PORT" ]; then
  echo "❌ エラー: ポート 8080-8089 はすべて使用中です"
  exit 1
fi

# 3. モード別環境変数設定
if [ "$MODE" = "dev" ]; then
  echo "🔧 開発モードでサーバーを起動します..."
  export PORT=$PORT
  export ENABLE_CORS=true
  export ENABLE_FILE_WATCH=true
  export FILE_WATCH_INTERVAL=15
  export FILE_WATCH_DEBOUNCE=5
elif [ "$MODE" = "prod" ]; then
  echo "🚀 本番モードでサーバーを起動します..."
  export PORT=$PORT
else
  echo "❌ エラー: 不明なモード '$MODE'"
  echo "使用法: $0 [dev|prod]"
  exit 1
fi

# 4. ビルド & 起動
make build
nohup ./bin/ccloganalysis > .claude/skills/server-management/server.log 2>&1 &
PID=$!

# 5. PIDとポートを記録
echo "$PID $PORT" > "$PID_FILE"
echo "📝 PIDファイルに記録しました: PID=$PID, PORT=$PORT"

# 6. ヘルスチェック（最大30秒待機）
echo "🔍 サーバーの起動を確認中..."
HEALTH_URL="http://localhost:$PORT/api/health"
RETRY_COUNT=0
MAX_RETRIES=30

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
  if curl -s -f "$HEALTH_URL" > /dev/null 2>&1; then
    echo "✅ サーバーが正常に起動しました"
    echo "   URL: http://localhost:$PORT"
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
