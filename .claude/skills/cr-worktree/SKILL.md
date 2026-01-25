---
model: claude-haiku-4-5
allowed-tools: Bash(.claude/skills/cr-worktree/scripts/cr-worktree.sh:*), Bash(.claude/skills/cr-worktree/scripts/update-issue-status.sh:*), Bash(gh:*), Bash(jq:*), Bash(git:*)
argument-hint: [issue-number|branch-name|description] [--from-current]
description: Issue番号やブランチ名を指定してワークツリーを作成し、Claude Codeを起動して/issueコマンドを自動実行します
---

**`.claude/skills/cr-worktree/scripts/cr-worktree.sh`**を使用してワークツリーを作成し、**新しいTerminalウィンドウで**Claude Codeを起動して開発を開始してください。

---

## 📋 クイックリファレンス

### 引数パターンの判定フロー

スキル実行時に渡された `$ARGUMENTS` を以下の順序で判定してください：

1. **オプション分離**: `--from-current` を先に分離 → `FROM_CURRENT` フラグに保持
2. **パターン判定**（正規表現、以下の順序で評価）:
   - **パターン1（Issue番号）**: `^[0-9]+$` → GitHub CLIでIssue取得 → ブランチ名決定 → スクリプト実行
   - **パターン2（ブランチ名）**: `^[a-zA-Z0-9/_-]+$` → スクリプト実行
   - **パターン3（説明文）**: 上記以外 → ブランチ名自動生成 → スクリプト実行

### 判定例

| 入力 | 判定結果 | 処理内容 |
|------|---------|---------|
| `1032` | パターン1 | Issue #1032取得 → ブランチ名決定 → `.claude/skills/cr-worktree/scripts/cr-worktree.sh <branch> --with-issue-command` |
| `1032-fix-bug` | パターン2 | `.claude/skills/cr-worktree/scripts/cr-worktree.sh 1032-fix-bug` |
| `feature/new-feature` | パターン2 | `.claude/skills/cr-worktree/scripts/cr-worktree.sh feature/new-feature` |
| `ログパーサーのバグ修正` | パターン3 | ブランチ名自動生成 → `.claude/skills/cr-worktree/scripts/cr-worktree.sh <generated-branch>` |
| `1032 --from-current` | パターン1 + オプション | `.claude/skills/cr-worktree/scripts/cr-worktree.sh <branch> --from-current --with-issue-command` |

### 重要なポイント

- **引数取得**: `$ARGUMENTS` 環境変数から取得（スキル実行時のみ利用可能）
- **判定順序**: 必ず パターン1 → パターン2 → パターン3 の順で評価
- **エラー時**: 引数が空の場合、使用例付きエラーメッセージを表示
- **GitHub CLI**: Issue番号指定時は認証確認が必要

---

## アーキテクチャ

**git worktree addコマンドを使用したワークツリー作成方式：**

- **`.claude/skills/cr-worktree/scripts/cr-worktree.sh`**: ワークツリー作成とClaude Code起動の実装（osascript使用）
- **`.claude/skills/cr-worktree/scripts/update-issue-status.sh`**: GitHub Projects ステータス更新
- **`./SKILL.md`**: ブランチ名決定ロジックのみ（Claude Codeの役割）

### 動作フロー

1. **スラッシュコマンド**: Issue番号/説明文からブランチ名を決定（Claude Code、Haikuモデル）
2. **`cr-worktree.sh`**:
   - メインブランチ同期確認
   - `git worktree add`でワークツリー作成
   - 環境整備（Go依存関係、Node.js依存関係、テスト実行）
   - **osascriptで新しいTerminalウィンドウを起動**
   - ワークツリーディレクトリに移動
   - Claude Code起動（Issue番号経由の場合は`/issue`コマンド自動実行）

---

## 処理の実装

**重要**: このスキルは Claude Code（AI）が実行します。クイックリファレンスを参照して処理してください。

### 実装の基本フロー

1. **`$ARGUMENTS` から引数を取得** → 空の場合はエラー
2. **オプション分離** → `--from-current` を先に分離して `FROM_CURRENT` フラグに保持
3. **パターン判定** → 正規表現で パターン1 → パターン2 → パターン3 の順で評価
4. **該当パターンの処理を実行** → 詳細は下記の各パターンを参照

**エラーハンドリング:**
- 引数が空 → 使用例付きエラーメッセージ表示
- GitHub CLI未認証（Issue番号指定時）→ `gh auth login` を案内
- Issueステータス更新失敗 → 警告表示して続行

---

### パターン1: Issue番号が指定された場合

**判定条件**: メイン部分が純粋な数字（正規表現: `^[0-9]+$`）

**処理手順:**

1. GitHub CLI認証確認 → 未認証の場合は `gh auth login` を案内
2. Issue情報取得 → `gh issue view <Issue番号> --json title`
3. Issueステータス更新 → `.claude/skills/cr-worktree/scripts/update-issue-status.sh <Issue番号> "In progress"`（失敗しても続行）
4. ブランチ名決定 → Issueタイトルを英語に変換 + kebab-case化 + Issue番号プレフィックス
5. スクリプト実行 → `.claude/skills/cr-worktree/scripts/cr-worktree.sh <ブランチ名> [--from-current] --with-issue-command`

**実装例:**

```bash
# Issue番号: 1032
# Issueタイトル: "cr-worktreeスキルで引数を指定しているのに..."
# 生成ブランチ名: 1032-fix-cr-worktree-arguments

.claude/skills/cr-worktree/scripts/cr-worktree.sh 1032-fix-cr-worktree-arguments --with-issue-command

# オプション付きの場合:
.claude/skills/cr-worktree/scripts/cr-worktree.sh 1032-fix-cr-worktree-arguments --from-current --with-issue-command
```

<details>
<summary>📖 詳細な実装ステップ（クリックして展開）</summary>

**ステップ1: 引数の取得と検証**
- `$ARGUMENTS` から値を取得
- 値が空の場合、使用例付きエラーメッセージを表示して終了

**ステップ2: GitHub CLI認証確認**
- `gh auth status` で確認
- 未認証の場合: `gh auth login` を案内

**ステップ3: デバッグ情報の表示**
```text
🔍 デバッグ情報:
  ARGUMENTS変数: [取得した値]
  判定パターン: Issue番号
```

**ステップ4: Issue情報取得**
```bash
gh issue view <Issue番号> --json title
```

**ステップ5: Issueステータス更新**
```bash
.claude/skills/cr-worktree/scripts/update-issue-status.sh <Issue番号> "In progress"
```

**ステップ6: ブランチ名決定**
- Issueタイトルを英語に変換（AIの判断）
- kebab-case形式に変換
- Issue番号をプレフィックスとして追加

**ステップ7: スクリプト実行**
```bash
.claude/skills/cr-worktree/scripts/cr-worktree.sh <ブランチ名> --with-issue-command
# または
.claude/skills/cr-worktree/scripts/cr-worktree.sh <ブランチ名> --from-current --with-issue-command
```

</details>

---

### パターン2: ブランチ名が指定された場合

**判定条件**: メイン部分が英数字+記号（正規表現: `^[a-zA-Z0-9/_-]+$`）

**注意**: `1032-fix-bug` のような数字で始まるブランチ名も、パターン1にマッチしないためパターン2として処理されます。

**処理手順:**

1. `FROM_CURRENT` フラグを確認
2. スクリプト実行 → `.claude/skills/cr-worktree/scripts/cr-worktree.sh <ブランチ名> [--from-current]`

**実装例:**

```bash
# ブランチ名指定
.claude/skills/cr-worktree/scripts/cr-worktree.sh feature/new-feature

# オプション付き
.claude/skills/cr-worktree/scripts/cr-worktree.sh feature/new-feature --from-current

# 数字で始まるブランチ名もOK
.claude/skills/cr-worktree/scripts/cr-worktree.sh 1032-fix-bug
```

---

### パターン3: 説明文が指定された場合

**判定条件**: メイン部分がパターン1、パターン2のどちらにも該当しない（日本語含む自由形式）

**処理手順:**

1. ブランチ名自動生成 → 説明文を英語に変換 + kebab-case化
2. `FROM_CURRENT` フラグを確認
3. スクリプト実行 → `.claude/skills/cr-worktree/scripts/cr-worktree.sh <生成したブランチ名> [--from-current] --with-description="<説明文>"`
4. WIPドキュメント自動生成 → worktree内の`docs/WIP/`に配置

**実装例:**

```bash
# 説明文: "ログパーサーのバグ修正"
# 生成ブランチ名: fix-log-parser-bug

.claude/skills/cr-worktree/scripts/cr-worktree.sh fix-log-parser-bug --with-description="ログパーサーのバグ修正"

# オプション付き
.claude/skills/cr-worktree/scripts/cr-worktree.sh fix-log-parser-bug --from-current --with-description="ログパーサーのバグ修正"
```

**WIPドキュメント仕様:**
- ファイル名: `docs/WIP/YYYY-MM-DD_<説明文50文字>.md`
- 既存のWIPドキュメント形式に準拠
- 自動生成後は未追跡ファイルとして残る（git add不要）
- Claude起動時に即座に参照可能

---

## 使用例

```bash
# Issue番号指定 → /issueコマンドを自動実行
/cr-worktree 123

# ブランチ名指定
/cr-worktree feature/new-feature

# 説明文から自動生成
/cr-worktree "ログパーサーのバグ修正"

# 現在のブランチから分岐
/cr-worktree 123 --from-current
```

## 実行される内部コマンド

```bash
# Issue番号指定時（例: /cr-worktree 123）
.claude/skills/cr-worktree/scripts/cr-worktree.sh 123-fix-log-parser --with-issue-command
  → メインブランチ同期確認
  → git worktree add -b 123-fix-log-parser <ワークツリーパス> main
  → 環境整備（go mod download、npm ci、make test）
  → osascript（新しいTerminalウィンドウを起動）
  → cd <ワークツリーパス>
  → claude '/issue'

# ブランチ名指定時（例: /cr-worktree feature/new-feature）
.claude/skills/cr-worktree/scripts/cr-worktree.sh feature/new-feature
  → メインブランチ同期確認
  → git worktree add -b feature/new-feature <ワークツリーパス> main
  → 環境整備（go mod download、npm ci、make test）
  → osascript（新しいTerminalウィンドウを起動）
  → cd <ワークツリーパス>
  → claude
```

**重要な仕様:**

- ✅ **追加**: git worktree addコマンドでワークツリー作成
- ✅ **追加**: 環境整備（Go + React）
- ✅ **追加**: osascriptで新しいTerminalウィンドウを起動
- ✅ **追加**: `claude '/issue'` - Issue番号経由の場合に/issueコマンドを自動実行（引数として直接渡す）
