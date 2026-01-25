---
model: claude-haiku-4-5
name: ready-for-pr
description: PR作成準備（mainマージ・ビルド・テスト・セルフレビュー）
allowed-tools:
  - Bash
  - Read
  - Task
argument-hint: "[--skip-unit-tests]"
---

# ready-for-pr

## Context

**PR作成準備を自動化するスキルです。**

このスキルは、PR作成前の最終チェック（mainブランチマージ、ビルド、テスト、セルフレビュー）を一括実行します。

## スキル実行モデル

このスキルは **動的生成モデル** で実行されます：

- **Claude Agent が SKILL.md の指示を読み取り、bash コマンドを動的生成**
- 静的なスクリプトファイル（`.sh`）は最小限（sync-permissions.sh のみ）
- Agent が実行時に適切なコマンドを組み立てて Bash ツールで実行
- 引数解析ロジックも Agent が SKILL.md の記述に従って生成

**利点:**
- スキル定義（SKILL.md）の更新だけで動作変更可能
- スクリプトファイルのメンテナンス負担が小さい
- Agent が文脈に応じて最適なコマンドを生成

## Your task

PR作成準備を自動的に実行し、すべてのチェックが完了したら完了報告を表示してください。

**重要: すべてのステップをユーザーに確認せず自動的に実行してください。途中で停止したり、確認を求めたりしないでください。**

### 引数の解析

以下の引数をサポートしています:

- `--skip-unit-tests`: ユニットテスト（Step 5）をスキップします

**引数解析ロジック:**
```bash
SKIP_UNIT_TESTS=false

# 渡された引数を解析
for arg in "$@"; do
  case "$arg" in
    --skip-unit-tests)
      SKIP_UNIT_TESTS=true
      ;;
  esac
done
```

### 処理フロー

以下の7つのステップを順番に実行してください：

#### Step 1: mainブランチマージ

```bash
echo "=========================================="
echo "Step 1: mainブランチマージ"
echo "=========================================="
echo ""

git fetch origin main
git merge origin/main --no-edit
```

**成功時**: Step 2へ進む
**コンフリクト発生時**: 以下の処理を実施
1. コンフリクトファイル一覧を取得（`git diff --name-only --diff-filter=U`）
2. `git merge --abort` でマージを中止
3. エラーメッセージを表示（下記参照）

**エラーメッセージ**:
```
⚠️ mainブランチとのコンフリクトが発生しました

コンフリクトファイル:
<`git diff --name-only --diff-filter=U` の出力（abort前に取得）>

手動で解決してください:
1. git fetch origin main
2. git merge origin/main
3. コンフリクトを手動解決
4. git add <解決済みファイル>
5. git commit
6. 再度 /ready-for-pr を実行
```

---

#### Step 2: 許可コマンドの統合

```bash
echo "=========================================="
echo "Step 2: 許可コマンドの統合"
echo "=========================================="
echo ""

./.claude/skills/ready-for-pr/scripts/sync-permissions.sh || true
echo ""
```

**処理内容**:
- ローカル設定ファイル（`.claude/settings.local.json`）の許可コマンドを共有設定（`.claude/settings.json`）に統合
- 各Worktreeで個別に許可したコマンドを、プロジェクト全体で共有できるようにする
- 失敗してもエラーにせず、警告表示のみで続行（`|| true`）

**成功時**: 統合結果を表示してStep 3へ進む
**ローカル設定なし**: "ローカル設定ファイルが存在しません"と表示してStep 3へ進む
**失敗時**: 警告表示のみでStep 3へ進む（処理を中断しない）

---

#### Step 3: 最終ビルド

```bash
echo "=========================================="
echo "Step 3: 最終ビルド"
echo "=========================================="
echo ""

make build
```

**成功時**: Step 4へ進む
**失敗時**: エラー報告して中断

**エラーメッセージ**:
```
⚠️ ビルド失敗: mainマージ後にビルドエラーが発生しました

エラー詳細:
<ビルドエラーメッセージ>

推奨対処:
1. フロントエンドエラーの場合: cd web && npm run build
2. バックエンドエラーの場合: go build -o bin/ccloganalysis ./cmd/server
3. 修正完了後、再度 /ready-for-pr を実行
```

---

#### Step 4: Linterチェック（ESLint）

```bash
echo "=========================================="
echo "Step 4: Linterチェック（ESLint）"
echo "=========================================="
echo ""

# 変更されたフロントエンドファイルを検出（mainブランチからの全変更）
CHANGED_FRONTEND_FILES=$(git diff --name-only origin/main...HEAD | grep "^web/" || true)

if [ -z "$CHANGED_FRONTEND_FILES" ]; then
  echo "ℹ️ フロントエンドファイルの変更がありません。Linterをスキップします。"
  echo ""
else
  echo "📝 変更されたフロントエンドファイル:"
  echo "$CHANGED_FRONTEND_FILES"
  echo ""

  echo "🔧 ESLintチェック実行中..."
  if cd web && npm run lint; then
    cd ..
    echo "✅ ESLint: 問題なし"
  else
    cd ..
    echo "❌ ESLint: エラー検出"
    echo ""
    echo "推奨対処:"
    echo "1. cd web && npm run lint でエラー確認"
    echo "2. エラーを修正"
    echo "3. 修正完了後、再度 /ready-for-pr を実行"
    exit 1
  fi
  echo ""
fi
```

**成功時**: Step 4.5へ進む
**失敗時**: エラー報告して中断

---

#### Step 4.5: コード簡素化提案（公式code-simplifierプラグイン）

**目的**: Linterチェック後に、コードの簡素化・明確化を提案

**前提条件**: 公式code-simplifierプラグインがインストールされていること
```bash
claude plugin install code-simplifier
```

**実装方法**: このステップはClaude Agentが動的に実行します。以下の処理を実施してください：

```bash
echo "=========================================="
echo "Step 4.5: コード簡素化提案（公式プラグイン）"
echo "=========================================="
echo ""

# 変更されたファイルを検出（mainブランチからの全変更）
CHANGED_FILES=$(git diff --name-only origin/main...HEAD || true)

if [ -z "$CHANGED_FILES" ]; then
  echo "ℹ️ ファイルの変更がありません。コード簡素化をスキップします。"
  echo ""
else
  echo "📝 変更されたファイル:"
  echo "$CHANGED_FILES"
  echo ""
  echo "🔍 コード簡素化提案を実行中（公式code-simplifierプラグイン）..."
  echo ""
fi
```

**bashコマンド実行後、変更されたファイルがある場合のみ、以下のTaskを呼び出してください：**

**実装形式**: code-simplifierエージェントを呼び出してコード簡素化提案を実行します。

```
Task(
  subagent_type="code-simplifier",
  description="コード簡素化提案",
  prompt="mainブランチからの変更ファイルについて、コードの簡素化提案を行ってください。

## 重点観点
- コードの明確化・一貫性・保守性の向上
- 関数分割、ネスト削減、命名改善の具体的提案
- TypeScript/React、Goのベストプラクティスに従った改善提案

## 出力
検出された改善機会と推奨される変更を報告してください。"
)
```

**エージェント呼び出し後の処理:**

```bash
# エージェント実行結果を確認
echo ""
echo "✅ コード簡素化提案が完了しました（公式code-simplifierプラグイン）"
echo ""
echo "💡 提案内容を確認して、必要に応じて修正を検討してください"
echo "   （提案のみで、PR作成を阻止しません）"
echo ""
```

**成功時**: Step 5へ進む
**失敗時**: 警告表示のみで継続（PR作成を阻止しない）

**重要な注意事項:**
- code-simplifierエージェントは**提案のみ**のため、PR作成を阻止しない
- エージェント呼び出しに失敗した場合は、警告表示のみで継続
- セルフレビュー（Step 6）で再度チェックされる

---

#### Step 5: 最終テスト（ユニットテスト）

**SKIP_UNIT_TESTS が true の場合:**
```bash
echo "=========================================="
echo "Step 5: ユニットテスト"
echo "=========================================="
echo ""
echo "⏭️  ユニットテスト実行をスキップします..."
echo ""
```

**SKIP_UNIT_TESTS が false の場合（デフォルト）:**
```bash
echo "=========================================="
echo "Step 5: ユニットテスト"
echo "=========================================="
echo ""

# ユニットテスト実行
if make test; then
  echo ""
  echo "✅ ユニットテストが成功しました"
  echo ""
else
  echo ""
  echo "⚠️ テスト失敗: mainマージ後にテストが失敗しました"
  echo ""
  echo "推奨対処:"
  echo "1. make test で再確認"
  echo "2. エラーを修正"
  echo "3. 修正完了後、再度 /ready-for-pr を実行"
  exit 1
fi
```

**成功時**: Step 6へ進む
**失敗時**: エラー詳細を表示して中断

---

#### Step 6: セルフレビュー

```bash
echo "=========================================="
echo "Step 6: セルフレビュー"
echo "=========================================="
echo ""
```

この後、`/self-review` スキルを呼び出してください（Skill ツールを使用）。

**P0/P1なし**: Step 7へ進む
**P0/P1あり**: 修正提案して中断

**エラーメッセージ**:
```
⚠️ セルフレビューで問題が検出されました

P0 (Blocker): <件数>
P1 (Critical): <件数>

推奨対処:
1. 指摘項目を修正
2. /self-review で再確認
3. P0/P1がゼロになったら、再度 /ready-for-pr を実行
```

---

#### Step 7: 完了報告

すべてのステップが成功したら、以下のメッセージを表示してください:

```
=========================================="
✅ PR作成準備が完了しました！
=========================================="

チェック完了項目:
- ✅ mainブランチマージ
- ✅ 許可コマンド統合
- ✅ ビルド成功
- ✅ Linterチェック成功（ESLint）
- ✅ コード簡素化提案完了（公式プラグイン）
- ✅ ユニットテスト成功 [or スキップ]
- ✅ セルフレビューP0/P1なし

💡 コード簡素化提案:
   公式プラグインから返される提案を確認して、実装可能な改善があれば対応してください
   （セルフレビューで検出された改善提案であり、PR作成を阻止しません）

次のステップ:
手動でPRを作成してください:
  git push origin <branch-name>
  gh pr create
```

---

### 使用例

#### 全テスト実行（推奨）

```bash
/ready-for-pr
```

#### ユニットテストをスキップ

```bash
/ready-for-pr --skip-unit-tests
```

---

## エラーハンドリング

| エラー | 対処方法 |
|-------|---------|
| mainマージでコンフリクト | `git merge --abort` → 手動解決を指示 |
| ビルド失敗 | エラー詳細表示 → 修正を指示 |
| Linterエラー（ESLint） | エラー詳細表示 → 修正を指示 |
| テスト失敗 | エラー詳細表示 → 修正を指示 |
| セルフレビューP0/P1 | 指摘項目表示 → 修正を指示 |

すべてのエラーで「修正後に `/ready-for-pr` を再実行」を案内してください。

---

## 重要な注意事項

### 1. 繰り返し実行可能

このコマンドはエラー時に中断し、修正後に再実行することを前提としています。

ステップ途中で失敗した場合:
1. エラーメッセージを確認
2. 修正を実施
3. 再度 `/ready-for-pr` を実行
4. 前回成功したステップはスキップせず、すべてのステップを最初から実行

### 2. セルフレビューの統合

セルフレビュー（`/self-review`）スキルを内部で呼び出します。

P0/P1項目が検出された場合は、PR作成準備を中断し、修正を促してください。

### 3. テストスキップオプションの使用

- `--skip-unit-tests`: アプリ本体に影響のない変更時（スキル・ドキュメント修正など）に有効
- スキップ時も他のチェック（ビルド、Linter、セルフレビュー）は実行されます

### 4. Agent の役割

このスキルは動的生成モデルで実行されるため、Agent が SKILL.md の指示に従ってコマンドを生成します。
スクリプトファイルは sync-permissions.sh のみで、他の処理は Agent が動的に実行します。
