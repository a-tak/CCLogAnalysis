---
model: claude-haiku-4-5
description: 現在の変更ファイルをコードレビューエージェントで自動レビューします
allowed-tools:
  - Bash
  - Task
  - AskUserQuestion
  - Edit
  - Read
---

# self-review

現在の変更ファイルをコードレビューエージェントとCodex MCPで並列レビューします。P0/P1/Critical項目を修正後、自動で再レビュー（最大3回）。

## 目的

コミット前に変更ファイルの品質をチェックし、プロジェクト固有のルールやコーディング規約の違反を検出します。

## 実行内容

1. `git status` で変更ファイルを自動検出
2. **並列実行**: コードレビューエージェント（`.claude/agents/code-reviewer.md`）とCodex MCPで同時レビュー
3. 両方のレビュー結果を統合して表示
4. P0/P1/Critical項目があれば、修正を実施
5. 修正後、自動で再レビュー（最大3回まで繰り返し、並列実行）
6. P0/P1/Criticalがゼロになったらコミット可能と通知

## 実行フロー

### ステップ1: 変更ファイルを検出

ブランチ全体の変更を検出します（mainブランチからの差分）：

```bash
git diff --name-only origin/main...HEAD
```

### ステップ2: セルフレビュー & Codex MCPレビューを並列実行（初回）

🚨 **最重要**: 以下の2つのTaskを**必ず同一メッセージ内で並列実行**してください。

**ブランチ全体の変更**をcode-reviewerエージェントとCodex MCPで並列レビューします。

#### Task 1: code-reviewer エージェント（プロジェクト固有ルール）

```
Task(
  subagent_type="code-reviewer",
  description="コードレビュー実施",
  prompt="以下の変更ファイルをレビューしてください。`.claude/rules/`配下のルール、プロジェクトガイドライン（CLAUDE.md）、一般開発ルールに従って、優先度付き結果（P0/P1/P2/P3）で報告してください。

## 変更ファイル（mainブランチからの全変更）
<git diff --name-only origin/main...HEAD の結果を列挙>

## 重点チェック項目
1. テスト駆動開発（TDD）規約への準拠
2. エラーハンドリングの適切性
3. 個人情報の取り扱い（プロジェクトはパブリック公開予定）
4. コードの明確性と保守性
5. テストカバレッジの妥当性"
)
```

#### Task 2: Codex MCP 総合レビュー

```
Task(
  subagent_type="general-purpose",
  description="Codex総合レビュー",
  prompt="""以下の変更をCodexで総合的にレビューしてください。

## 変更ファイル（mainブランチからの全変更）
<git diff --name-only origin/main...HEAD の結果を列挙>

アーキテクチャ原則、コーディング規約、潜在的な問題について総合的にレビューをお願いします。
重要度の高い指摘には「Critical」、中程度には「Important」、軽微なものには「Minor」のラベルを付けてください。"""
)
```

### ステップ3: 両方のレビュー結果を統合して表示

エージェントから返された2つのレビュー結果を統合して表示します。

```
========================================
📋 セルフレビュー結果
========================================

## code-reviewer レビュー結果

### P0 (Blocker) - 必須修正
<P0項目の詳細>

### P1 (Critical) - 重要
<P1項目の詳細>

### P2 (Major) - 推奨
<P2項目の詳細>

### P3 (Minor) - 改善提案
<P3項目の詳細>

[📊 code-reviewer サマリー]
- P0 (Blocker): <数>件 🔴
- P1 (Critical): <数>件 ⚠️
- P2 (Major): <数>件 ℹ️
- P3 (Minor): <数>件 💡

---

## Codex MCP レビュー結果

### Critical
<Critical項目の詳細>

### Important
<Important項目の詳細>

### Minor
<Minor項目の詳細>

[📊 Codex MCP サマリー]
- Critical: <数>件 🔴
- Important: <数>件 ⚠️
- Minor: <数>件 💡

---

## 統合プライオリティ

1. **P0 & Critical（最優先、必ず修正）**
2. **P1 & Important（推奨修正）**
3. P2/P3 & Minor（余裕があれば修正）

[✅ 良い点]
<両方のレビューから得られた良い点>
```

### ステップ4: P0/P1/Critical項目があれば修正を実施

P0/P1項目（code-reviewer）またはCritical項目（Codex MCP）が検出された場合、以下の処理を実施：

1. **AskUserQuestion でどの項目を修正するか確認**

```
AskUserQuestion(
  questions: [{
    question: "以下のP0/P1/Critical項目を修正しますか？",
    header: "修正対象",
    multiSelect: true,
    options: [
      { label: "[code-reviewer] P0 #1: 個人情報漏洩", description: "..." },
      { label: "[code-reviewer] P1 #2: エラーハンドリング欠如", description: "..." },
      { label: "[Codex] Critical #1: セキュリティ脆弱性", description: "..." },
      ...
    ]
  }]
)
```

2. **選択された項目の修正を実施**

ユーザーが選択した項目を Edit ツールで修正します。

3. **テスト実行**

```bash
make test
```

テストが失敗した場合は、修正を取り消してユーザーに報告します。

### ステップ5: 修正後に再レビュー（自動、並列実行）

修正後、再度 `git diff --name-only origin/main...HEAD` でブランチ全体の変更ファイルを検出し、**ステップ2と同様にcode-reviewerエージェントとCodex MCPレビューを並列実行**します。

**再レビューループ制御:**
- 最大再試行回数: 3回まで
- ループ終了条件: P0/P1/Critical項目がゼロになる
- 無限ループ防止: 3回の再レビュー後もP0/P1/Criticalが残る場合は、残課題を報告してユーザーに判断を委ねる

### ステップ6: 最終判定

- **P0/P1/Criticalがゼロ**: コミット可能と通知 ✅
- **最大再試行回数到達**: 残課題を報告してユーザーに判断を委ねる
- **P2/P3/Important/Minorのみの場合**: コミット可能と通知（余裕があれば修正を推奨）
- **問題なしの場合**: 品質良好と通知

## 出力例

### P0/P1/Criticalがゼロの場合

```
========================================
✅ セルフレビュー完了
========================================

P0/P1/Criticalの問題は検出されませんでした。
コミット可能です。

[📊 code-reviewer サマリー]
- P0 (Blocker): 0件
- P1 (Critical): 0件
- P2 (Major): 2件
- P3 (Minor): 3件

[📊 Codex MCP サマリー]
- Critical: 0件
- Important: 1件
- Minor: 2件

💡 P2/P3/Important/Minorの改善提案がありますが、必須ではありません。
   余裕があれば対応を検討してください。

次のステップ:
  git add .
  git commit -m "..."
```

### P0/P1/Criticalが残る場合

```
========================================
⚠️ セルフレビュー: 修正が必要です
========================================

P0/P1/Critical項目が検出されました。
修正してから再度 /self-review を実行してください。

[📊 code-reviewer サマリー]
- P0 (Blocker): 1件 🔴
- P1 (Critical): 2件 ⚠️

[📊 Codex MCP サマリー]
- Critical: 1件 🔴

[💡 優先対応項目]
1. [code-reviewer] P0 #1: 個人情報漏洩
2. [code-reviewer] P1 #2: エラーハンドリング欠如
3. [Codex] Critical #1: セキュリティ脆弱性

推奨対処:
1. 指摘項目を修正
2. 修正完了後、再度 /self-review を実行
```

## 推奨される使用タイミング

### ケース1: 新機能実装後

```bash
# 実装完了
make test

# セルフレビュー（P0/P1項目を自動修正→再レビュー）
/self-review

# P0/P1がゼロになったらコミット可能と通知される
git add .
git commit -m "feat: 新機能追加"
```

### ケース2: バグ修正後

```bash
# バグ修正完了
make test

# セルフレビュー（P0/P1項目を自動修正→再レビュー）
/self-review

# P0/P1がゼロになったらコミット可能と通知される
git add .
git commit -m "fix: バグ修正"
```

## 参照ドキュメント

- **[.claude/agents/code-reviewer.md](.claude/agents/code-reviewer.md)**: コードレビューエージェントの詳細
- **[CLAUDE.md](../../CLAUDE.md)**: プロジェクトガイドライン
- **[.claude/rules/general.md](../rules/general.md)**: 一般開発ルール
