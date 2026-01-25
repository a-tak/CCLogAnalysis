#!/bin/bash
set -e

PID_FILE=".claude/skills/server-management/.server.pid"
SIGTERM_TIMEOUT=10

# 1. PIDファイルチェック
if [ ! -f "$PID_FILE" ]; then
  echo "⚠️  警告: PIDファイルが見つかりません"
  echo "   サーバーはスクリプト以外の方法で起動された可能性があります"
  echo "   手動でプロセスを確認してください: ps aux | grep ccloganalysis"
  exit 1
fi

# 2. PIDとポートを読み取り
read PID PORT < "$PID_FILE"
echo "📝 PIDファイルから読み取り: PID=$PID, PORT=$PORT"

# 3. プロセス存在確認
if ! kill -0 "$PID" 2>/dev/null; then
  echo "⚠️  警告: プロセス $PID は既に終了しています"
  rm "$PID_FILE"
  echo "✅ PIDファイルをクリーンアップしました"
  exit 0
fi

# 4. Graceful Shutdown（SIGTERM）
echo "🛑 サーバーを停止中... (PID: $PID, ポート: $PORT)"
kill -TERM $PID 2>/dev/null || true

# 5. 最大10秒待機
WAIT_COUNT=0
while [ $WAIT_COUNT -lt $SIGTERM_TIMEOUT ]; do
  # プロセスとポートの両方をチェック
  if ! kill -0 "$PID" 2>/dev/null && ! lsof -i :$PORT -t >/dev/null 2>&1; then
    echo "✅ サーバーが正常に停止しました"
    rm "$PID_FILE"
    exit 0
  fi

  sleep 1
  WAIT_COUNT=$((WAIT_COUNT + 1))
  echo -n "."
done

echo ""
echo "⚠️  警告: Graceful shutdownがタイムアウトしました"
echo "   強制終了します..."

# 6. 強制終了（SIGKILL）
# プロセスを強制終了
kill -KILL $PID 2>/dev/null || true

# ポートを使用しているプロセスも確実に終了
PORT_PIDS=$(lsof -i :$PORT -t 2>/dev/null || true)
if [ -n "$PORT_PIDS" ]; then
  echo "   ポート $PORT を使用している残存プロセスを終了します: $PORT_PIDS"
  for PORT_PID in $PORT_PIDS; do
    kill -KILL $PORT_PID 2>/dev/null || true
  done
fi

sleep 1

# ポートが解放されたか確認
if ! lsof -i :$PORT -t >/dev/null 2>&1; then
  echo "✅ サーバーを強制終了しました"
  rm "$PID_FILE"
  exit 0
else
  echo "❌ エラー: サーバーの停止に失敗しました"
  echo "   残存プロセス:"
  lsof -i :$PORT 2>/dev/null || true
  exit 1
fi
