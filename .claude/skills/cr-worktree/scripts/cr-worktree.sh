#!/bin/bash
#
# ワークツリー作成スクリプト（git worktree版）
#
# このスクリプトはgit worktreeコマンドを使用してワークツリーを作成し、
# Claude Codeを起動します。
#
# 使用方法:
#   .claude/skills/cr-worktree/scripts/cr-worktree.sh <ブランチ名> [--from-current] [--with-issue-command]
#
# 引数:
#   <ブランチ名>           ブランチ名（例: feature/new-feature, 571-improve-cr-worktree）
#   --from-current         現在のブランチから分岐（デフォルトはmainから分岐）
#   --with-issue-command   Issue番号経由の呼び出し（Claude Codeで自動的に /issue コマンドを実行）
#
# 注意:
#   - Issue番号や説明文を直接渡すことはできません
#   - Issue番号経由での使用は /cr-worktree スラッシュコマンドを使用してください
#

set -e  # エラー時に即座に停止

# メインブランチ同期関数
# mainブランチ（またはmaster）がリモートより古い場合、自動的に同期する
sync_main_branch_with_remote() {
  echo "🔄 メインブランチの同期状態を確認中..."
  echo ""

  # メインブランチ名とリポジトリパスの検出
  local WORKTREE_INFO=$(git worktree list | grep -E '\[(main|master)\]' | head -1)

  if [ -z "$WORKTREE_INFO" ]; then
    echo "❌ エラー: メインブランチ（main または master）が見つかりません"
    echo "   git worktree listでメインブランチが検出できませんでした"
    exit 1
  fi

  local MAIN_REPO_ROOT=$(echo "$WORKTREE_INFO" | awk '{print $1}')
  local MAIN_BRANCH=$(echo "$WORKTREE_INFO" | sed 's/.*\[\(.*\)\]/\1/')

  echo "📍 メインリポジトリ: $MAIN_REPO_ROOT"
  echo "🌿 メインブランチ: $MAIN_BRANCH"
  echo ""

  # リモートの最新情報を取得
  echo "🔄 リモートの最新情報を取得中..."
  FETCH_OUTPUT=$(git -C "$MAIN_REPO_ROOT" fetch origin 2>&1)
  FETCH_EXIT_CODE=$?

  if [ $FETCH_EXIT_CODE -ne 0 ]; then
    echo "⚠️  警告: リモートの最新情報取得に失敗しました"
    echo "   ネットワーク接続を確認してください"
    echo ""
    echo "エラー詳細:"
    echo "$FETCH_OUTPUT"
    echo ""
    echo "処理を続行しますか？ (古いメインブランチから作成される可能性があります)"
    read -p "続行する場合は 'y' を入力: " CONFIRM
    if [ "$CONFIRM" != "y" ]; then
      exit 1
    fi
    echo ""
    return 0  # ユーザーが続行を選択した場合は処理を継続
  fi

  echo "✅ リモート情報取得完了"
  echo ""

  # ローカルとリモートのコミットハッシュを取得
  local LOCAL_HASH=$(git -C "$MAIN_REPO_ROOT" rev-parse $MAIN_BRANCH)
  local REMOTE_HASH=$(git -C "$MAIN_REPO_ROOT" rev-parse origin/$MAIN_BRANCH)

  # デバッグ: ハッシュを表示
  echo "🔍 ブランチの状態:"
  echo "   ローカル: ${LOCAL_HASH:0:8}"
  echo "   リモート: ${REMOTE_HASH:0:8}"
  echo ""

  # 差分チェック
  if [ "$LOCAL_HASH" = "$REMOTE_HASH" ]; then
    echo "✅ メインブランチは最新です"
    echo ""
    return 0
  fi

  echo "🔍 ローカルとリモートに差分があります"
  echo "   ローカル: ${LOCAL_HASH:0:8}"
  echo "   リモート: ${REMOTE_HASH:0:8}"
  echo ""

  # ケース1: ローカルが古い（リモートに新しいコミットがある）
  if git -C "$MAIN_REPO_ROOT" merge-base --is-ancestor $MAIN_BRANCH origin/$MAIN_BRANCH 2>/dev/null; then
    echo "🔄 メインブランチがリモートより古いため、同期します..."
    echo ""

    # 未コミット変更のチェック
    if ! git -C "$MAIN_REPO_ROOT" diff-index --quiet HEAD -- 2>/dev/null; then
      echo "❌ エラー: メインブランチに未コミットの変更があります"
      echo "   変更をコミットまたはstashしてから再実行してください"
      echo ""
      exit 1
    fi

    # Fast-forwardマージ
    echo "🔧 Fast-forwardマージを実行中..."
    if git -C "$MAIN_REPO_ROOT" merge --ff-only origin/$MAIN_BRANCH 2>/dev/null; then
      echo "✅ メインブランチを最新に更新しました"
      echo ""
      return 0
    else
      echo "❌ エラー: メインブランチの更新に失敗しました"
      echo "   Fast-forwardマージができませんでした"
      echo ""
      exit 1
    fi
  fi

  # ケース2: ローカルが新しい（リモートにpushされていないコミットがある）
  if git -C "$MAIN_REPO_ROOT" merge-base --is-ancestor origin/$MAIN_BRANCH $MAIN_BRANCH 2>/dev/null; then
    echo "⚠️  警告: メインブランチにリモートにpushされていないコミットがあります"
    echo "   これは通常の状態ではありません（mainブランチは直接コミットしない運用）"
    echo ""
    echo "   ローカルのコミット: ${LOCAL_HASH:0:8}"
    echo "   リモートのコミット: ${REMOTE_HASH:0:8}"
    echo ""
    echo "処理を続行しますか？"
    read -p "続行する場合は 'y' を入力: " CONFIRM
    if [ "$CONFIRM" != "y" ]; then
      exit 1
    fi
    echo ""
    return 0
  fi

  # ケース3: 分岐している（コンフリクトの可能性）
  echo "❌ エラー: メインブランチがリモートと分岐しています"
  echo "   ローカル: ${LOCAL_HASH:0:8}"
  echo "   リモート: ${REMOTE_HASH:0:8}"
  echo ""
  echo "   以下のいずれかの操作が必要です:"
  echo "   1. メインブランチでgit pullを実行してマージ"
  echo "   2. メインブランチをリセットしてリモートと同期: git reset --hard origin/$MAIN_BRANCH"
  echo ""
  exit 1
}

# WIPドキュメント生成関数
# 引数:
#   $1: 説明文（タスク名）
#   $2: ワークツリーパス
generate_wip_document() {
  local TASK_DESCRIPTION="$1"
  local WORKTREE_PATH="$2"
  local WIP_DIR="$WORKTREE_PATH/docs/WIP"

  # 現在の日付取得
  local TODAY=$(date +"%Y-%m-%d")

  # ファイル名用のタスク名（50文字制限）
  local TASK_NAME_SHORT="${TASK_DESCRIPTION:0:50}"

  # ファイル名から無効な文字（スラッシュなど）を削除
  TASK_NAME_SHORT="${TASK_NAME_SHORT//\//}"

  # WIPファイル名
  local WIP_FILE="$WIP_DIR/${TODAY}_${TASK_NAME_SHORT}.md"

  # WIPディレクトリ存在確認
  if [ ! -d "$WIP_DIR" ]; then
    echo "⚠️  警告: WIPディレクトリが見つかりません: $WIP_DIR"
    echo "   WIPドキュメント生成をスキップします"
    return 1
  fi

  # WIPドキュメント生成（既存フォーマット準拠）
  cat > "$WIP_FILE" <<'EOF'
# ${TASK_DESCRIPTION}

**作業日**: ${TODAY}

---

## 概要

${TASK_DESCRIPTION}

---

## 実装内容

### Phase 1: 実装レイヤー

（ここに実装内容を記載）

---

## 完了チェックリスト

- [ ] 実装完了
- [ ] テスト作成・パス
- [ ] ドキュメント更新
- [ ] コミット実施

---

## 次のステップ

（次に実施すべき作業を記載）

---

## 参考情報

（関連ドキュメント、参考URLなど）

---

**このドキュメントは /cr-worktree スキルで自動生成されました。**
**作業内容に応じて適宜更新してください。**
EOF

  # 変数展開（heredoc内の変数を実際の値に置換）
  # sed の置換コマンドで | を区切り文字として使用（/ が含まれる説明文に対応）
  sed -i '' "s|\${TASK_DESCRIPTION}|${TASK_DESCRIPTION}|g" "$WIP_FILE"
  sed -i '' "s|\${TODAY}|${TODAY}|g" "$WIP_FILE"

  if [ $? -eq 0 ]; then
    echo "✅ WIPドキュメント生成完了: $WIP_FILE"
    return 0
  else
    echo "⚠️  警告: WIPドキュメント生成に失敗しました"
    return 1
  fi
}

# 引数の取得
ARG="$1"
FROM_CURRENT=false
WITH_ISSUE_COMMAND=false
WITH_DESCRIPTION=""

# オプション解析
for arg in "$@"; do
  case $arg in
    --from-current)
      FROM_CURRENT=true
      ;;
    --with-issue-command)
      WITH_ISSUE_COMMAND=true
      ;;
    --with-description=*)
      WITH_DESCRIPTION="${arg#*=}"
      ;;
  esac
done

# 引数がない場合はエラーメッセージを表示
if [ -z "$ARG" ]; then
  echo "❌ エラー: 引数が指定されていません"
  echo ""
  echo "使用方法:"
  echo "  .claude/skills/cr-worktree/scripts/cr-worktree.sh <ブランチ名> [--from-current] [--with-issue-command]"
  echo ""
  echo "例:"
  echo "  .claude/skills/cr-worktree/scripts/cr-worktree.sh feature/new-feature"
  echo "  .claude/skills/cr-worktree/scripts/cr-worktree.sh 571-improve-cr-worktree --from-current"
  echo "  .claude/skills/cr-worktree/scripts/cr-worktree.sh 571-improve-cr-worktree --with-issue-command"
  echo ""
  echo "注意:"
  echo "  - Issue番号や説明文を直接渡すことはできません"
  echo "  - Issue番号経由での使用は /cr-worktree スラッシュコマンドを使用してください"
  exit 1
fi

# Issue番号かどうかをチェック
if [[ "$ARG" =~ ^[0-9]+$ ]]; then
  # パターン1: Issue番号の場合
  # 設計判断: Issue番号からブランチ名を決定するにはClaude Codeが必要なため、
  # このスクリプトは直接Issue番号を受け付けず、スラッシュコマンド経由での使用を前提とする。
  # 副作用（Issueステータス更新）を起こさないよう、早期にバリデーションして終了する。

  echo "❌ エラー: このスクリプトは直接Issue番号では実行できません"
  echo ""
  echo "理由: Issue番号からブランチ名を決定するには、Claude Codeが必要です。"
  echo "      このスクリプトはブランチ名を引数として受け取る設計です。"
  echo ""
  echo "代わりに以下のスラッシュコマンドを使用してください:"
  echo "  /cr-worktree $ARG"
  echo ""
  echo "スラッシュコマンドは以下の処理を自動実行します:"
  echo "  1. GitHubからIssue #$ARG の情報を取得"
  echo "  2. Issueステータスを 'In progress' に更新"
  echo "  3. Claude CodeがIssueタイトルから自動的にブランチ名を決定"
  echo "  4. このスクリプトを適切なブランチ名で呼び出し"
  exit 1

elif [[ "$ARG" =~ ^[a-zA-Z0-9/_-]+$ ]]; then
  # パターン2-3: ブランチ名の場合（英数字とハイフン、スラッシュ、アンダースコアのみ）
  BRANCH_NAME="$ARG"
  echo "🌿 ブランチ名: $BRANCH_NAME"
  echo ""

  # --from-currentオプションが指定されていない場合のみメインブランチ同期チェック
  if [ "$FROM_CURRENT" = false ]; then
    sync_main_branch_with_remote
  fi

  # メインリポジトリの検出
  echo "🔧 メインリポジトリを検出中..."
  WORKTREE_INFO=$(git worktree list | grep -E '\[(main|master)\]' | head -1)

  if [ -z "$WORKTREE_INFO" ]; then
    echo "❌ エラー: メインリポジトリが見つかりません"
    echo "   git worktree listでメインブランチが検出できませんでした"
    exit 1
  fi

  MAIN_REPO_ROOT=$(echo "$WORKTREE_INFO" | awk '{print $1}')
  MAIN_BRANCH=$(echo "$WORKTREE_INFO" | sed 's/.*\[\(.*\)\]/\1/')

  echo "✅ メインリポジトリ検出: $MAIN_REPO_ROOT"
  echo "   メインブランチ: $MAIN_BRANCH"
  echo ""

  # ワークツリーパスの決定
  WORKTREE_BASE="$(dirname "$MAIN_REPO_ROOT")/$(basename "$MAIN_REPO_ROOT").worktrees"
  WORKTREE_PATH="$WORKTREE_BASE/$BRANCH_NAME"

  echo "📂 ワークツリーパス: $WORKTREE_PATH"
  echo ""

  # ブランチ名重複チェック
  if git show-ref --verify --quiet "refs/heads/$BRANCH_NAME"; then
    echo "❌ エラー: ブランチ '$BRANCH_NAME' は既に存在します"
    echo ""
    echo "既存のブランチを使用する場合:"
    echo "  cd $WORKTREE_BASE"
    echo "  # 既存のワークツリーディレクトリを確認"
    echo ""
    exit 1
  fi

  # ワークツリーディレクトリ重複チェック
  if [ -d "$WORKTREE_PATH" ]; then
    echo "❌ エラー: ワークツリーディレクトリ '$WORKTREE_PATH' は既に存在します"
    echo ""
    echo "ディレクトリを削除してから再実行してください:"
    echo "  rm -rf \"$WORKTREE_PATH\""
    echo ""
    exit 1
  fi

  # ワークツリーベースディレクトリ作成
  mkdir -p "$WORKTREE_BASE"

  # 分岐元ブランチの決定
  BASE_BRANCH="$MAIN_BRANCH"
  if [ "$FROM_CURRENT" = true ]; then
    CURRENT_BRANCH=$(git branch --show-current)
    if [ -n "$CURRENT_BRANCH" ]; then
      BASE_BRANCH="$CURRENT_BRANCH"
      echo "📍 現在のブランチ ($CURRENT_BRANCH) から分岐します"
      echo ""
    else
      echo "⚠️  現在のブランチを取得できませんでした。$MAIN_BRANCH から分岐します。"
      echo ""
    fi
  fi

  # ワークツリー作成
  echo "🔧 ワークツリーを作成中..."
  if ! git worktree add -b "$BRANCH_NAME" "$WORKTREE_PATH" "$BASE_BRANCH"; then
    echo "❌ エラー: ワークツリーの作成に失敗しました"
    # 作成途中のディレクトリをクリーンアップ
    if [ -d "$WORKTREE_PATH" ]; then
      echo "   作成途中のディレクトリを削除中..."
      rm -rf "$WORKTREE_PATH"
    fi
    exit 1
  fi
  echo "✅ ワークツリー作成完了"
  echo ""

  # ワークツリーディレクトリに移動
  cd "$WORKTREE_PATH"

  # 環境整備
  echo "🔧 環境整備を開始します..."
  echo ""

  # 1. Go依存関係インストール
  echo "📦 Go依存関係をインストール中..."
  if ! go mod download; then
    echo "❌ エラー: Go依存関係のインストールに失敗しました"
    exit 1
  fi
  echo "✅ Go依存関係インストール完了"
  echo ""

  # 2. Node.js依存関係インストール
  echo "📦 Node.js依存関係をインストール中..."
  cd web
  if ! npm ci; then
    echo "⚠️  警告: npm ci に失敗しました。npm install を試行します..."
    if ! npm install; then
      echo "❌ エラー: Node.js依存関係のインストールに失敗しました"
      exit 1
    fi
  fi
  cd ..
  echo "✅ Node.js依存関係インストール完了"
  echo ""

  # 2.5. Reactアプリのビルド
  echo "🏗️  Reactアプリをビルド中..."
  cd web
  if ! npm run build; then
    echo "❌ エラー: Reactアプリのビルドに失敗しました"
    exit 1
  fi
  cd ..
  echo "✅ Reactアプリのビルド完了"
  echo ""

  echo "✅ 環境整備完了"
  echo ""

  # 説明文経由の場合、WIPドキュメント生成
  if [ -n "$WITH_DESCRIPTION" ]; then
    echo "📝 WIPドキュメントを生成中..."
    echo ""

    if generate_wip_document "$WITH_DESCRIPTION" "$WORKTREE_PATH"; then
      echo ""
    else
      echo "   （WIPドキュメント生成に失敗しましたが、処理を続行します）"
      echo ""
    fi
  fi

  # Claude Code起動
  echo "🚀 Claude Codeを新しいターミナルウィンドウで起動中..."
  echo "📌 ウィンドウタイトル: CCLogAnalysis: $BRANCH_NAME"
  echo ""

  # macOS判定
  if [[ "$OSTYPE" != "darwin"* ]]; then
    echo "⚠️  警告: 新しいターミナルウィンドウの起動はmacOSでのみ利用可能です"
    echo ""
    echo "現在のOS: $OSTYPE"
    echo ""
    echo "手動でClaude Codeを起動する場合:"
    echo "  cd $WORKTREE_PATH"
    echo "  claude"
    echo ""
  else
    # Ghostty検出
    TERMINAL_APP="Terminal"
    if [ -d "/Applications/Ghostty.app" ]; then
      TERMINAL_APP="Ghostty"
      echo "✅ Ghosttyを検出しました"
    else
      echo "✅ Terminal.appを使用します（Ghostty未検出）"
    fi
    echo ""

    # Ghostty優先、失敗時はTerminal.appへフォールバック
    # 引数:
    #   $1: ワークツリーパス
    #   $2: ブランチ名
    #   $3: Claudeコマンド（"claude" または "claude '/issue'"）
    #   $4: ターミナルアプリ（"Ghostty" または "Terminal"）
    # 戻り値:
    #   0: 起動成功
    #   1: 起動失敗
    launch_claude_in_terminal() {
      local WORKTREE_PATH="$1"
      local BRANCH_NAME="$2"
      local CLAUDE_COMMAND="$3"
      local TERMINAL_APP="$4"

      if [ "$TERMINAL_APP" = "Ghostty" ]; then
        # Ghostty起動（ログインシェルを明示してPATHを読み込む）
        # cdコマンドを明示的に実行してワークツリーに移動
        if open -na Ghostty --args \
          --title="CCLogAnalysis: $BRANCH_NAME" \
          -e "$SHELL" -l -c "cd '$WORKTREE_PATH' && $CLAUDE_COMMAND" &> /dev/null
        then
          echo "✅ Claude Codeを起動しました（Ghostty）"
          echo ""
          echo "新しいGhosttyウィンドウでClaude Codeが起動しています。"
          echo ""
          return 0
        else
          echo "⚠️  警告: Ghosttyの起動に失敗しました。Terminal.appで起動します..."
          echo ""
        fi
      fi

      # フォールバック: Terminal.app（ログインシェルを明示）
      if osascript <<EOF &> /dev/null
tell application "Terminal"
    do script "$SHELL -l -c \"cd '$WORKTREE_PATH' && $CLAUDE_COMMAND\""
    set custom title of front window to "CCLogAnalysis: $BRANCH_NAME"
end tell
EOF
      then
        echo "✅ Claude Codeを起動しました（Terminal.app）"
        echo ""
        echo "新しいTerminalウィンドウでClaude Codeが起動しています。"
        echo ""
        return 0
      else
        echo "⚠️  警告: Claude Codeの起動に失敗しました"
        echo ""
        echo "手動で起動してください:"
        echo "  cd $WORKTREE_PATH"
        echo "  $CLAUDE_COMMAND"
        echo ""
        return 1
      fi
    }

    # claudeコマンドの存在確認
    if ! command -v claude &> /dev/null; then
      echo "⚠️  警告: claudeコマンドが見つかりませんでした"
      echo ""
      echo "Claude Codeがインストールされていないか、PATHに含まれていない可能性があります。"
      echo "手動でClaude Codeを起動してください:"
      echo "  cd $WORKTREE_PATH"
      echo "  claude"
      echo ""
    else
      # Issue番号経由の場合は /issue コマンドを自動実行
      if [ "$WITH_ISSUE_COMMAND" = true ]; then
        echo "📋 Issue番号経由のため、/issue コマンドを自動実行します"
        launch_claude_in_terminal "$WORKTREE_PATH" "$BRANCH_NAME" "claude '/issue'" "$TERMINAL_APP"
      else
        # 通常起動
        echo "📋 Claude Codeを起動します"
        launch_claude_in_terminal "$WORKTREE_PATH" "$BRANCH_NAME" "claude" "$TERMINAL_APP"
      fi
    fi
  fi

else
  # パターン4: 説明文の場合
  echo "❌ エラー: このスクリプトは直接説明文では実行できません"
  echo ""
  echo "代わりに以下のスラッシュコマンドを使用してください:"
  echo "  /cr-worktree \"$ARG\""
  echo ""
  echo "スラッシュコマンドはClaude Codeが説明文から自動的にブランチ名を決定します。"
  exit 1
fi

echo ""
echo "=========================================="
echo "✅ ワークツリー作成・Claude Code起動完了"
echo "=========================================="
echo ""
echo "🌿 ブランチ: $BRANCH_NAME"
echo ""
