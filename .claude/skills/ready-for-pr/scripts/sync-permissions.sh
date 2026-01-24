#!/bin/bash
# sync-permissions.sh
# ローカル設定ファイル (.claude/settings.local.json) の許可コマンドを
# 共有設定ファイル (.claude/settings.json) に統合する

set -e

SHARED_SETTINGS=".claude/settings.json"
LOCAL_SETTINGS=".claude/settings.local.json"

# ワークツリー/ユーザー固有パスを除外するフィルタ関数
# 以下のパターンを除外:
# - /Users/<username>/... (macOSユーザー固有)
# - /home/<username>/... (Linuxユーザー固有)
# - $HOME/... (ホームディレクトリ参照)
# - C:\Users\... (Windowsユーザー固有)
# - export PATH=... (環境変数設定)
# - 条件分岐断片 (if, then, else, fi, do)
filter_worktree_specific() {
  jq '[.[] | select(
    # ユーザー固有パスを除外
    (test("/Users/[^/]+/") | not) and
    (test("/home/[^/]+/") | not) and
    (test("[$]HOME/") | not) and
    (test("C:[\\\\]Users[\\\\]") | not) and
    # 環境変数設定を除外
    (test("^Bash[(]export PATH=") | not) and
    # 条件分岐断片を除外（意味のない設定）
    (test("^Bash[(]if ") | not) and
    (test("^Bash[(]then[)]") | not) and
    (test("^Bash[(]else[)]") | not) and
    (test("^Bash[(]fi[)]") | not) and
    (test("^Bash[(]do ") | not)
  )]'
}

echo "📋 許可コマンドの統合を開始します..."

# 1. settings.local.jsonが存在しない場合はスキップ
if [ ! -f "$LOCAL_SETTINGS" ]; then
  echo "ℹ️  ローカル設定ファイルが存在しません - スキップします"
  exit 0
fi

# 2. settings.jsonが存在しない場合はエラー
if [ ! -f "$SHARED_SETTINGS" ]; then
  echo "❌ 共有設定ファイル ($SHARED_SETTINGS) が見つかりません"
  exit 1
fi

# 3. jqがインストールされているか確認
if ! command -v jq &> /dev/null; then
  echo "❌ jqコマンドが見つかりません。インストールしてください: sudo apt install jq"
  exit 1
fi

# 4. mainブランチの最新settings.jsonを取得
echo "🔍 mainブランチの最新設定を取得中..."
git fetch origin main --quiet 2>/dev/null || true
MAIN_SETTINGS=$(git show origin/main:.claude/settings.json 2>/dev/null || echo "{}")

# 5. 統合前のローカル許可コマンド数をカウント
LOCAL_COUNT=$(jq '.permissions.allow // [] | length' "$LOCAL_SETTINGS" 2>/dev/null || echo "0")

if [ "$LOCAL_COUNT" -eq 0 ]; then
  echo "ℹ️  ローカル設定に許可コマンドがありません - スキップします"
  exit 0
fi

echo "📦 ローカル設定から ${LOCAL_COUNT} 件の許可コマンドを検出"

# 6. ローカル設定からワークツリー固有パスを除外
echo "🔍 ワークツリー固有パスをフィルタリング中..."
FILTERED_LOCAL=$(jq '.permissions.allow // []' "$LOCAL_SETTINGS" | filter_worktree_specific)
FILTERED_COUNT=$(echo "$FILTERED_LOCAL" | jq 'length')
EXCLUDED_COUNT=$((LOCAL_COUNT - FILTERED_COUNT))

if [ "$EXCLUDED_COUNT" -gt 0 ]; then
  echo "   - ${EXCLUDED_COUNT} 件のワークツリー固有コマンドを除外"
fi

# 7. jqで3つの設定をマージ
# - mainブランチの設定
# - 現在のブランチの共有設定
# - ローカル設定（フィルタ済み）
echo "🔧 設定をマージ中..."
jq -s '
  (.[0].permissions.allow // []) as $main |
  (.[1].permissions.allow // []) as $current |
  (.[2]) as $local |
  ($main + $current + $local | unique | sort) as $merged |
  .[1] | .permissions.allow = $merged
' <(echo "$MAIN_SETTINGS") "$SHARED_SETTINGS" <(echo "$FILTERED_LOCAL") > "${SHARED_SETTINGS}.tmp"

# 8. 変更があればsettings.jsonを更新
if ! cmp -s "$SHARED_SETTINGS" "${SHARED_SETTINGS}.tmp"; then
  # 追加された許可コマンドを表示
  BEFORE_COUNT=$(jq '.permissions.allow // [] | length' "$SHARED_SETTINGS")
  AFTER_COUNT=$(jq '.permissions.allow // [] | length' "${SHARED_SETTINGS}.tmp")
  ADDED_COUNT=$((AFTER_COUNT - BEFORE_COUNT))

  mv "${SHARED_SETTINGS}.tmp" "$SHARED_SETTINGS"
  echo "✅ ローカル許可コマンドを共有設定に統合しました"
  echo "   - 統合前: ${BEFORE_COUNT} 件"
  echo "   - 統合後: ${AFTER_COUNT} 件"
  echo "   - 新規追加: ${ADDED_COUNT} 件"

  git add "$SHARED_SETTINGS"

  # 9. (オプション) settings.local.jsonをクリア
  # 統合後はローカル設定の許可コマンドをクリアしても良い
  # ユーザーが手動で削除する方が安全なため、コメントアウト
  # rm "$LOCAL_SETTINGS"
  # echo "ℹ️  ローカル設定をクリアしました"
else
  rm "${SHARED_SETTINGS}.tmp"
  echo "ℹ️  許可コマンドに新規追加はありません"
fi

echo "✨ 許可コマンドの統合が完了しました"
