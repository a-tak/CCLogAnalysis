---
model: claude-sonnet-4-5
name: issue
description: 現在のブランチ名からissue番号を抽出し、GitHubのissue情報を分析してプラン立案します
allowed-tools:
  - Bash(gh:*)
  - Bash(git:*)
  - EnterPlanMode
argument-hint: "[issue-number]"
---

# issue

## Context

**現在のブランチ名からissue番号を抽出し、GitHubのissue情報を分析してプラン立案するスキルです。**

このスキルは、ブランチ名の先頭にあるissue番号を自動検出し、GitHubからissue情報を取得して、自動的にプラン立案モードに入ります。

## スキル実行モデル

このスキルは **動的生成モデル** で実行されます：

- **Claude Agent が SKILL.md の指示を読み取り、bash コマンドを動的生成**
- 静的なスクリプトファイル（`.sh`）は不要
- Agent が実行時に適切なコマンドを組み立てて Bash ツールで実行
- 引数解析ロジックも Agent が SKILL.md の記述に従って生成

**利点:**
- スキル定義（SKILL.md）の更新だけで動作変更可能
- スクリプトファイルのメンテナンス負担が小さい
- Agent が文脈に応じて最適なコマンドを生成

## Your task

現在のブランチからissue番号を抽出し、GitHubのissue情報を分析してプラン立案を開始してください。

**重要: すべてのステップをユーザーに確認せず自動的に実行してください。途中で停止したり、確認を求めたりしないでください。**

### 引数の解析

以下の2つのパターンをサポートしています:

#### パターン1: 引数でissue番号を指定
```bash
/issue 123
```

この場合、指定されたissue番号を使用します。

#### パターン2: 引数なし（デフォルト）
```bash
/issue
```

この場合、現在のブランチ名からissue番号を自動抽出します。

**引数解析ロジック:**
```bash
# 引数から issue 番号を取得
ISSUE_NUMBER="$ARGUMENTS"

# 引数がない場合は、現在のブランチ名から抽出
if [ -z "$ISSUE_NUMBER" ]; then
  CURRENT_BRANCH=$(git branch --show-current)
  # ブランチ名の先頭から数字を抽出（例: "4-fix-file-real-time-monitoring" → "4"）
  ISSUE_NUMBER=$(echo "$CURRENT_BRANCH" | grep -o '^[0-9]\+')
fi

# issue番号が取得できない場合はエラー
if [ -z "$ISSUE_NUMBER" ]; then
  echo "❌ エラー: issue番号を取得できませんでした"
  echo ""
  echo "使用方法:"
  echo "  1. issue番号を指定: /issue 123"
  echo "  2. ブランチ名から自動検出: /issue (ブランチ名が '数字-説明' の形式の場合)"
  echo ""
  echo "現在のブランチ: $CURRENT_BRANCH"
  exit 1
fi
```

### 処理フロー

以下の3つのステップを順番に実行してください：

#### Step 1: GitHub CLI認証確認

```bash
echo "=========================================="
echo "Step 1: GitHub CLI認証確認"
echo "=========================================="
echo ""

# GitHub CLI 認証状態を確認
if ! gh auth status &>/dev/null; then
  echo "❌ GitHub CLIが認証されていません"
  echo ""
  echo "認証を行ってください:"
  echo "  gh auth login"
  echo ""
  exit 1
fi

echo "✅ GitHub CLI認証済み"
echo ""
```

**成功時**: Step 2へ進む
**未認証時**: エラーメッセージを表示して中断

---

#### Step 2: Issue情報取得

```bash
echo "=========================================="
echo "Step 2: Issue情報取得 (#$ISSUE_NUMBER)"
echo "=========================================="
echo ""

# Issue情報をJSON形式で取得
ISSUE_JSON=$(gh issue view "$ISSUE_NUMBER" --json number,title,body,labels,state 2>&1)
ISSUE_STATUS=$?

if [ $ISSUE_STATUS -ne 0 ]; then
  echo "❌ Issue #$ISSUE_NUMBER の取得に失敗しました"
  echo ""
  echo "$ISSUE_JSON"
  echo ""
  exit 1
fi

# Issue情報を整形して表示
echo "📋 Issue情報:"
echo "$ISSUE_JSON" | jq -r '"  番号: #\(.number)\n  タイトル: \(.title)\n  状態: \(.state)\n  ラベル: \(.labels | map(.name) | join(", "))"'
echo ""
echo "詳細:"
echo "$ISSUE_JSON" | jq -r '.body'
echo ""
echo "✅ Issue情報を取得しました"
echo ""
```

**成功時**: Step 3へ進む（Issue情報をメモリに保持）
**失敗時**: エラーメッセージを表示して中断

---

#### Step 3: プラン立案モード開始

**この後、EnterPlanMode ツールを使用してプラン立案モードに入ってください。**

```bash
echo "=========================================="
echo "Step 3: プラン立案開始"
echo "=========================================="
echo ""
```

**EnterPlanMode 呼び出し後の指示:**

プラン立案モードに入ったら、Step 2で取得したissue情報に基づいて以下を実施してください：

1. **Issue要件の分析**:
   - Issueのタイトルと本文を詳細に分析
   - 実装すべき内容を明確化
   - 関連するラベル・優先度を考慮

2. **コードベース調査**:
   - 関連するファイルやモジュールを探索（Glob, Grep, Read ツールを使用）
   - 既存の実装パターンを確認
   - 影響範囲を特定

3. **プラン作成**:
   - 実装手順を具体的にリストアップ
   - ファイル変更箇所を明確化
   - テスト計画を含める
   - リスクと対処法を記載

4. **プランファイル作成**:
   - `.claude/plans/YYYY-MM-DD_<タスク名>.md` の形式で作成
   - プランの内容を記述
   - プランファイル名は `plans.md` ルールに従うこと

5. **ExitPlanMode**:
   - プランが完成したら ExitPlanMode ツールを使用
   - ユーザーにプラン承認を求める

---

## 使用例

### パターン1: 引数でissue番号を指定

```bash
/issue 123
```

実行されるフロー:
1. issue番号: `123`
2. GitHub CLI認証確認
3. `gh issue view 123` で情報取得
4. プラン立案モード開始

### パターン2: ブランチ名から自動検出（推奨）

```bash
# ブランチ名: 4-fix-file-real-time-monitoring
/issue
```

実行されるフロー:
1. 現在のブランチ名を取得: `4-fix-file-real-time-monitoring`
2. issue番号を抽出: `4`
3. GitHub CLI認証確認
4. `gh issue view 4` で情報取得
5. プラン立案モード開始

---

## エラーハンドリング

| エラー | 対処方法 |
|-------|---------|
| GitHub CLI未認証 | `gh auth login` を案内 |
| issue番号が取得できない | ブランチ名を確認、または引数で issue 番号を指定 |
| issue が存在しない | issue 番号を確認 |

---

## 重要な注意事項

### 1. ブランチ名の命名規則

このスキルは以下のブランチ名形式を前提としています:

```
<issue番号>-<説明>
```

例:
- `4-fix-file-real-time-monitoring`
- `123-add-new-feature`
- `999-refactor-api`

### 2. cr-worktree スキルとの連携

`/cr-worktree` スキルでは、issue番号を指定すると自動的に `/issue` コマンドを実行します:

```bash
/cr-worktree 123
```

この場合:
1. ワークツリー作成
2. 環境整備
3. 新しいTerminalウィンドウでClaude Code起動
4. **自動的に `/issue` コマンド実行**

### 3. プラン立案モードでの動作

プラン立案モード（EnterPlanMode）に入った後は、以下のツールを積極的に活用してください：

- **Glob**: ファイル検索（`**/*.go`, `**/*.tsx` など）
- **Grep**: コード検索（関数名、変数名など）
- **Read**: ファイル内容の確認
- **Task**: 複雑な調査タスク（Explore agent）

### 4. Agent の役割

このスキルは動的生成モデルで実行されるため、Agent が SKILL.md の指示に従ってコマンドを生成します。
スクリプトファイルは不要で、Agent が動的に実行します。
