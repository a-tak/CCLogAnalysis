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

# 2. PID読み取り
PID=$(cat "$PID_FILE")
echo "📝 PIDファイルから読み取り: $PID"

# 3. プロセス存在確認
if ! kill -0 "$PID" 2>/dev/null; then
  echo "⚠️  警告: プロセス $PID は既に終了しています"
  rm "$PID_FILE"
  echo "✅ PIDファイルをクリーンアップしました"
  exit 0
fi

# 4. Graceful Shutdown（SIGTERM）
echo "🛑 サーバーを停止中... (PID: $PID)"
kill -TERM "$PID"

# 5. 最大10秒待機
WAIT_COUNT=0
while [ $WAIT_COUNT -lt $SIGTERM_TIMEOUT ]; do
  if ! kill -0 "$PID" 2>/dev/null; then
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
kill -KILL "$PID" 2>/dev/null || true
sleep 1

if ! kill -0 "$PID" 2>/dev/null; then
  echo "✅ サーバーを強制終了しました"
  rm "$PID_FILE"
  exit 0
else
  echo "❌ エラー: サーバーの停止に失敗しました"
  exit 1
fi
