#!/bin/bash
#
# GitHub ProjectsでIssueのステータスを更新するスクリプト
#
# 使用方法:
#   ./scripts/update-issue-status.sh <issue-number> <status>
#
# 引数:
#   issue-number: Issue番号
#   status: 更新後のステータス（"Statement", "Backlog", "Ready", "In progress", "In review", "Done"）
#
# 例:
#   ./scripts/update-issue-status.sh 123 "In progress"
#

# Note: set -e を使用しない理由:
# - GraphQL API失敗時（行182-203）にフォールバック処理（gh project item-edit）を実行するため
# - 複数の認証/API呼び出しが失敗しても、可能な限り処理を継続してユーザーに情報を提供するため
#
# 終了コード:
#   0: 成功 または ソフトエラー（認証なし、Issue未関連付けなど、呼び出し元が処理を継続すべき）
#   1: クリティカルエラー（引数不足、リポジトリ情報取得失敗など、処理を中断すべき）
#   2: リカバラブルエラー（ステータス更新失敗したが手動対応可能）

# 引数チェック
if [ $# -ne 2 ]; then
  echo "❌ エラー: 引数が不足しています"
  echo ""
  echo "使用方法:"
  echo "  $0 <issue-number> <status>"
  echo ""
  echo "ステータス:"
  echo "  - Statement"
  echo "  - Backlog"
  echo "  - Ready"
  echo "  - In progress"
  echo "  - In review"
  echo "  - Done"
  echo ""
  echo "例:"
  echo "  $0 123 \"In progress\""
  exit 1
fi

ISSUE_NUMBER="$1"
NEW_STATUS="$2"

# Issue番号の検証（数値のみを許可）
if ! [[ "$ISSUE_NUMBER" =~ ^[0-9]+$ ]]; then
  echo "❌ エラー: 無効なIssue番号 '$ISSUE_NUMBER'"
  echo "   Issue番号は数値のみを指定してください"
  exit 1
fi

# ステータスの検証
# Note: GitHub Projectsの実際のステータス名は "In progress" (小文字p)
VALID_STATUSES=("Statement" "Backlog" "Ready" "In progress" "In review" "Done")
if [[ ! " ${VALID_STATUSES[@]} " =~ " ${NEW_STATUS} " ]]; then
  echo "❌ エラー: 無効なステータス '$NEW_STATUS'"
  echo ""
  echo "有効なステータス:"
  printf "  - %s\n" "${VALID_STATUSES[@]}"
  exit 1
fi

echo "📋 Issue #${ISSUE_NUMBER} のステータスを更新中..."
echo "   新しいステータス: ${NEW_STATUS}"

# GitHub CLIの認証チェック
if ! gh auth status &> /dev/null; then
  echo "⚠️  警告: GitHub CLIの認証が必要です"
  echo "   実行してください: gh auth login"
  echo "   処理を続行できません"
  exit 0  # ソフトエラー: 呼び出し元が処理を継続
fi

# jqの存在チェック
if ! command -v jq &> /dev/null; then
  echo "❌ エラー: jqがインストールされていません"
  echo "   jqはJSON処理に必要です"
  echo "   インストール方法:"
  echo "   - macOS: brew install jq"
  echo "   - Ubuntu/Debian: sudo apt-get install jq"
  echo "   - Windows (Git Bash): choco install jq"
  exit 1  # クリティカルエラー
fi

# リポジトリ情報を取得（HTTPS/SSH両方に対応）
if ! REPO_URL=$(git config --get remote.origin.url 2>&1); then
  echo "❌ エラー: Gitリポジトリ情報を取得できませんでした"
  echo "   このスクリプトはGitリポジトリ内で実行する必要があります"
  echo "   詳細: $REPO_URL"
  exit 1  # クリティカルエラー
fi

# HTTPS形式: https://github.com/owner/repo.git
# SSH形式: git@github.com:owner/repo.git
if [[ "$REPO_URL" =~ github\.com[:/]([^/]+)/([^/]+)(\.git)?$ ]]; then
  REPO_OWNER="${BASH_REMATCH[1]}"
  REPO_NAME="${BASH_REMATCH[2]%.git}"  # .gitを除去
else
  echo "❌ エラー: GitHub リポジトリではありません"
  echo "   URL: $REPO_URL"
  echo "   このスクリプトはGitHub リポジトリで使用する必要があります"
  exit 1  # クリティカルエラー
fi

echo "   リポジトリ: ${REPO_OWNER}/${REPO_NAME}"

# Issueの存在確認とProject情報取得（GraphQL APIを直接使用）
ISSUE_DATA=$(gh api graphql -f query='
  query($owner: String!, $repo: String!, $number: Int!) {
    repository(owner: $owner, name: $repo) {
      issue(number: $number) {
        number
        title
        projectItems(first: 5) {
          nodes {
            id
            project {
              id
              title
            }
          }
        }
      }
    }
  }
' -f owner="$REPO_OWNER" -f repo="$REPO_NAME" -F number="$ISSUE_NUMBER" 2>&1) || true

if [ -z "$ISSUE_DATA" ] || ! echo "$ISSUE_DATA" | jq -e '.data.repository.issue.number' &> /dev/null; then
  echo "⚠️  警告: Issue #${ISSUE_NUMBER} が見つかりません"
  echo "   詳細: $(echo "$ISSUE_DATA" | head -1)"
  echo "   処理を続行できません"
  exit 0
fi

# jqを1回だけ実行して必要な値を全て取得（パフォーマンス最適化）
# タイトルにタブやスペースが含まれる場合も安全に処理するため、改行区切りで取得
IFS=$'\n' read -r -d '' ISSUE_TITLE PROJECT_COUNT PROJECT_ITEM_ID PROJECT_ID PROJECT_TITLE <<< "$(echo "$ISSUE_DATA" | jq -r '
  .data.repository.issue |
  [
    .title,
    (.projectItems.nodes | length),
    (.projectItems.nodes[0].id // ""),
    (.projectItems.nodes[0].project.id // ""),
    (.projectItems.nodes[0].project.title // "")
  ] | .[]
')" || true

echo "   Issue: #${ISSUE_NUMBER} - ${ISSUE_TITLE}"

# ProjectItems の存在確認
if [ "$PROJECT_COUNT" = "0" ] || [ -z "$PROJECT_ITEM_ID" ] || [ "$PROJECT_ITEM_ID" = "null" ]; then
  echo "⚠️  警告: このIssueはどのProjectにも関連付けられていません"
  echo "   ステータスを更新するには、まずIssueをProjectに追加してください"
  exit 0
fi

# 複数のProjectに関連付けられている場合は警告
if [ "$PROJECT_COUNT" -gt 1 ]; then
  echo "ℹ️  情報: このIssueは ${PROJECT_COUNT} 個のProjectに関連付けられています"
  echo "   最初のProject（${PROJECT_TITLE}）のステータスを更新します"
fi

echo "   Project Item ID: ${PROJECT_ITEM_ID}"
echo "   Project: ${PROJECT_TITLE}"
echo ""
echo "🔄 ステータスを更新中..."

# Projectのフィールド情報を取得（StatusフィールドのIDと選択肢IDを取得）
PROJECT_FIELDS=$(gh api graphql -f query="
  query(\$projectId: ID!) {
    node(id: \$projectId) {
      ... on ProjectV2 {
        fields(first: 20) {
          nodes {
            ... on ProjectV2SingleSelectField {
              id
              name
              options {
                id
                name
              }
            }
          }
        }
      }
    }
  }
" -f projectId="$PROJECT_ID" 2>&1) || true

# フィールド情報取得の成功確認
if [ -z "$PROJECT_FIELDS" ] || ! echo "$PROJECT_FIELDS" | jq -e '.data.node.fields.nodes' &> /dev/null; then
  echo "⚠️  警告: Projectフィールド情報を取得できませんでした"
  echo "   詳細: $(echo "$PROJECT_FIELDS" | head -1)"
  echo "   gh project item-edit で試行します..."

  # フォールバック: gh project item-edit コマンドを使用
  FALLBACK_RESULT=$(gh project item-edit --id "$PROJECT_ITEM_ID" --field-name "Status" --single-select-option-name "$NEW_STATUS" 2>&1)
  FALLBACK_EXIT_CODE=$?

  if [ $FALLBACK_EXIT_CODE -eq 0 ]; then
    echo "✅ ステータスを '${NEW_STATUS}' に更新しました（フォールバック方式）"
    exit 0
  fi

  # フォールバック失敗時のエラー詳細
  echo "   フォールバック失敗:"
  echo "$FALLBACK_RESULT" | head -2 | sed 's/^/   /'

  echo ""
  echo "⚠️  警告: 自動ステータス更新に失敗しました"
  echo ""
  echo "📝 手動でステータスを更新してください:"
  echo "   1. GitHubでIssue #${ISSUE_NUMBER} を開く"
  echo "   2. Projectsタブでステータスを '${NEW_STATUS}' に変更"
  exit 2  # リカバラブルエラー: ステータス更新失敗したが手動対応可能
fi

# StatusフィールドのIDと選択肢IDを1回のjqで取得（パフォーマンス最適化）
IFS=$'\n' read -r -d '' STATUS_FIELD_ID STATUS_OPTION_ID <<< "$(echo "$PROJECT_FIELDS" | jq -r --arg status "$NEW_STATUS" '
  (.data.node.fields.nodes[] | select(.name == "Status")) as $statusField |
  [
    ($statusField.id // ""),
    ($statusField.options[] | select(.name == $status) | .id // "")
  ] | .[]
')" || true

if [ -z "$STATUS_FIELD_ID" ] || [ "$STATUS_FIELD_ID" = "null" ]; then
  echo "⚠️  警告: Statusフィールドが見つかりません"
  echo "   利用可能なフィールド:"
  echo "$PROJECT_FIELDS" | jq -r '.data.node.fields.nodes[].name' | sed 's/^/   - /'
  exit 1
fi

if [ -z "$STATUS_OPTION_ID" ] || [ "$STATUS_OPTION_ID" = "null" ]; then
  echo "⚠️  警告: ステータス '$NEW_STATUS' が見つかりません"
  echo "   利用可能なステータス:"
  echo "$PROJECT_FIELDS" | jq -r '.data.node.fields.nodes[] | select(.name == "Status") | .options[].name' | sed 's/^/   - /'
  exit 1
fi

echo "   Status Field ID: ${STATUS_FIELD_ID}"
echo "   Status Option ID: ${STATUS_OPTION_ID}"

# GraphQL APIを使用してステータスを更新
set +e  # 一時的にエラー検出を無効化
UPDATE_RESULT=$(gh api graphql -f query='
  mutation($projectId: ID!, $itemId: ID!, $fieldId: ID!, $optionId: String!) {
    updateProjectV2ItemFieldValue(input: {
      projectId: $projectId
      itemId: $itemId
      fieldId: $fieldId
      value: {
        singleSelectOptionId: $optionId
      }
    }) {
      projectV2Item {
        id
      }
    }
  }
' -f projectId="$PROJECT_ID" -f itemId="$PROJECT_ITEM_ID" -f fieldId="$STATUS_FIELD_ID" -f optionId="$STATUS_OPTION_ID" 2>&1)
UPDATE_EXIT_CODE=$?
set -e  # エラー検出を再度有効化（影響範囲を限定）

# JSON構造で成功を判定（終了コードは参考情報のみ）
if echo "$UPDATE_RESULT" | jq -e '.data.updateProjectV2ItemFieldValue.projectV2Item.id' &> /dev/null; then
  echo "✅ ステータスを '${NEW_STATUS}' に更新しました"
  exit 0
fi

# すべての方法が失敗した場合
echo ""
echo "⚠️  警告: 自動ステータス更新に失敗しました"
echo "   エラー詳細:"
echo "$UPDATE_RESULT" | head -3 | sed 's/^/   /'
echo ""
echo "📝 手動でステータスを更新してください:"
echo "   1. GitHubでIssue #${ISSUE_NUMBER} を開く"
echo "   2. Projectsタブでステータスを '${NEW_STATUS}' に変更"
echo ""
echo "または、以下のコマンドを試してください:"
echo "   gh project item-edit --id $PROJECT_ITEM_ID --field-name Status --single-select-option-id <option-id>"
echo ""

exit 2  # リカバラブルエラー: ステータス更新失敗したが手動対応可能
