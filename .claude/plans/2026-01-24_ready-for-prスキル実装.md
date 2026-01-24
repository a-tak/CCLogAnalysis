# ready-for-pr スキル移植実装プラン

## 概要

別プロジェクトの ready-for-pr スキルを CCLogAnalysis プロジェクトに移植します。
PR作成前の最終チェック（mainマージ・ビルド・テスト・セルフレビュー）を自動化するスキルです。

## ユーザー要件確認結果

- ✅ /self-review スキルも一緒に実装する
- ✅ ESLintエラーは ready-for-pr 実装前に修正する
- ✅ テスト成功フラグ（.git/.test-passed）は今回実装しない
- ✅ 最小限の移植（MVP）で実装する

## 実装方針

### 採用アプローチ

**動的生成型モデル**（移植元と同じ）
- SKILL.md に詳細な処理フローを記述
- Claude Agent が SKILL.md を読み取り、bash コマンドを動的生成・実行
- 補助スクリプトは最小限

### 移植範囲

**実装する機能（6ステップ）:**
1. mainブランチマージ（コンフリクト検出・中止機能付き）
2. ビルド（`make build`）
3. Linter チェック（ESLint）
4. ユニットテスト（`make test`、`--skip-unit-tests` オプション対応）
5. セルフレビュー（`/self-review`）
6. 完了報告

**スキップする機能:**
- 許可コマンド統合（sync-permissions.sh）- CCLogAnalysis では不要
- テスト成功フラグ（.git/.test-passed）- git-create-pr スキル未実装のため将来拡張
- コード簡素化提案（code-simplifier）- 将来拡張
- インストゥルメンティッドテスト - Android 固有機能のため不要

## 実装ステップ

### Phase 1: 事前準備 - ESLintエラーの修正

**対象ファイル:**
- `web/` 配下の TypeScript/React ファイル（既存のESLintエラー2件）

**作業内容:**
1. `cd web && npm run lint` でエラー箇所を特定
2. ESLint エラーを修正
3. 修正確認（`npm run lint` でエラーゼロ）
4. コミット（`fix: ESLintエラーを修正`）

**検証方法:**
```bash
cd web && npm run lint  # エラーが0件であることを確認
```

---

### Phase 2: /self-review スキル実装（最小限）

**目的:**
ready-for-pr の Step 5 で使用する /self-review スキルを実装します。

**実装範囲:**
- コード変更の自動レビュー
- 重大度別の指摘（P0: Blocker、P1: Critical、P2: Major、P3: Minor）
- P0/P1 検出時の処理中断

**ファイル構成:**
```
.claude/skills/self-review/
└── SKILL.md
```

**SKILL.md の構造:**
```markdown
---
model: claude-sonnet-4-5
name: self-review
description: コード変更の自動レビュー
allowed-tools:
  - Bash
  - Read
  - Grep
---

# self-review

## Context
mainブランチからの差分をレビューし、問題を指摘します。

## Your task
1. `git diff origin/main...HEAD` で変更を取得
2. 変更内容をレビュー
3. 重大度別に指摘を分類（P0/P1/P2/P3）
4. レビュー結果を表示

## レビュー基準
- P0（Blocker）: セキュリティ脆弱性、致命的バグ
- P1（Critical）: バグ、重要なベストプラクティス違反
- P2（Major）: コード品質問題、改善推奨
- P3（Minor）: スタイル、軽微な改善提案

## 出力フォーマット
指摘がある場合、重大度別に表示。
P0/P1がある場合は **修正必須** と明示。
```

**検証方法:**
```bash
/self-review  # コマンドが正常に動作し、レビュー結果が表示されることを確認
```

---

### Phase 3: ready-for-pr スキル実装（MVP版）

**ファイル構成:**
```
.claude/skills/ready-for-pr/
└── SKILL.md
```

**SKILL.md の構造:**

#### Frontmatter
```yaml
---
model: claude-haiku-4-5
name: ready-for-pr
description: PR作成準備（mainマージ・ビルド・テスト・セルフレビュー）
allowed-tools:
  - Bash
  - Read
  - Skill
argument-hint: "[--skip-unit-tests]"
---
```

#### セクション構成

**1. Context**
- スキルの概要
- 動的生成モデルの説明

**2. Your task**
- メイン処理フロー
- 引数解析ロジック（`--skip-unit-tests`）
- 6つのステップの実行

**3. 処理フロー詳細**

##### Step 1: mainブランチマージ
```bash
git fetch origin main
git merge origin/main --no-edit
```

**成功時**: Step 2へ進む
**コンフリクト発生時**:
1. コンフリクトファイル一覧を取得（`git diff --name-only --diff-filter=U`）
2. `git merge --abort` でマージを中止
3. エラーメッセージを表示して中断

##### Step 2: ビルド
```bash
make build
```

**成功時**: Step 3へ進む
**失敗時**: エラー詳細を表示して中断

**エラーメッセージ例:**
```
⚠️ ビルド失敗: mainマージ後にビルドエラーが発生しました

エラー詳細:
<ビルドエラーメッセージ>

推奨対処:
1. フロントエンドエラーの場合: cd web && npm run build
2. バックエンドエラーの場合: go build -o bin/ccloganalysis ./cmd/server
3. 修正完了後、再度 /ready-for-pr を実行
```

##### Step 3: Linter チェック（ESLint）
```bash
cd web && npm run lint
```

**成功時**: Step 4へ進む
**失敗時**: エラー詳細を表示して中断

##### Step 4: ユニットテスト
```bash
# --skip-unit-tests が false の場合のみ実行
if [ "$SKIP_UNIT_TESTS" = false ]; then
  make test
fi
```

**成功時**: Step 5へ進む
**失敗時**: エラー詳細を表示して中断
**スキップ時**: スキップメッセージを表示して Step 5へ進む

##### Step 5: セルフレビュー
```bash
/self-review
```

**P0/P1検出時**: 修正を指示して中断
**P2/P3のみまたは指摘なし**: Step 6へ進む

##### Step 6: 完了報告
```
✅ PR作成準備が完了しました！

チェック完了項目:
- ✅ mainブランチマージ
- ✅ ビルド成功
- ✅ Linterチェック成功（ESLint）
- ✅ ユニットテスト成功（XX tests passed）
- ✅ セルフレビューP0/P1なし

次のステップ:
手動でPRを作成してください:
  git push origin <branch-name>
  gh pr create
```

**4. エラーハンドリング**
- 各ステップでのエラー時の対処方法を記述
- 繰り返し実行可能な設計（べき等性）

**5. 重要な注意事項**
- 自動実行（ユーザー確認なし）
- Agent の役割分離

---

### Phase 4: 動作確認とテスト

**テストケース:**

1. **正常系**: 全ステップが成功する場合
   ```bash
   /ready-for-pr
   ```
   期待結果: Step 1 → 6 まで全て成功し、完了報告が表示される

2. **テストスキップ**: `--skip-unit-tests` オプション
   ```bash
   /ready-for-pr --skip-unit-tests
   ```
   期待結果: Step 4 がスキップされ、他のステップは実行される

3. **エラーケース - mainマージでコンフリクト**
   - 事前条件: mainブランチと競合する変更を作成
   - 期待結果: コンフリクトファイル一覧を表示し、`git merge --abort` 後に中断

4. **エラーケース - ビルド失敗**
   - 事前条件: ビルドエラーを含むコードを作成
   - 期待結果: エラー詳細を表示して中断

5. **エラーケース - Linterエラー**
   - 事前条件: ESLintエラーを含むコードを作成
   - 期待結果: Linterエラーを表示して中断

6. **エラーケース - テスト失敗**
   - 事前条件: テストが失敗するコードを作成
   - 期待結果: テストエラーを表示して中断

7. **エラーケース - セルフレビューでP0/P1検出**
   - 事前条件: セキュリティ脆弱性や重大なバグを含むコードを作成
   - 期待結果: P0/P1指摘を表示して中断

---

### Phase 5: ドキュメント整備

**作成するドキュメント:**

1. **`.claude/skills/ready-for-pr/README.md`** - 使用方法
   - スキルの概要
   - 使用例
   - 引数一覧
   - トラブルシューティング

2. **WIPドキュメントの更新** - 進捗記録
   - `docs/WIP/YYYY-MM-DD.md` に実装完了を記録
   - 実装内容と動作確認結果を記載

3. **CLAUDE.md の確認** - プロジェクト情報の整合性確認
   - スキル情報の追加（必要に応じて）

---

## 重要ファイル一覧

### 新規作成ファイル
- `.claude/skills/self-review/SKILL.md`
- `.claude/skills/ready-for-pr/SKILL.md`
- `.claude/skills/ready-for-pr/README.md`（ドキュメント）

### 修正が必要なファイル
- `web/` 配下の TypeScript/React ファイル（ESLintエラー修正）

### 参照するファイル
- 移植元プロジェクトの ready-for-pr スキル（SKILL.md）
- `Makefile`（ビルド・テストコマンド）
- `web/package.json`（ESLintコマンド）

---

## 実装の注意点

### 1. CCLogAnalysis 固有のカスタマイズ

| 項目 | 移植元 | CCLogAnalysis |
|------|---------|---------------|
| ビルド | `./gradlew build` | `make build` |
| テスト | `./gradlew test` | `make test` |
| Linter | `ktlint` + `detekt` | `npm run lint` (ESLint) |
| 技術スタック | Kotlin/Android | Go + React/TypeScript |

### 2. テストの前提条件

- `make test` はフロントエンドビルドが必須
- Step 2 でビルド完了後、Step 4 でテスト実行

### 3. エラーハンドリング

- 各ステップでのエラーは **即座に中断**
- エラーメッセージには **具体的な対処方法** を含める
- 繰り返し実行可能な設計（ステップの途中から再実行可能）

### 4. Agent 実行モデル

- **動的生成型**: SKILL.md の記述に従って Agent がコマンドを生成
- スクリプトファイルは不要（移植元の sync-permissions.sh も今回は移植しない）
- Agent が文脈に応じて最適なコマンドを実行

---

## 将来の拡張計画（今回は実装しない）

1. **テスト成功フラグ**（`.git/.test-passed`）
   - git-create-pr スキル実装時に追加

2. **許可コマンド統合**（sync-permissions.sh）
   - settings.json に permissions 設定が追加された場合に実装

3. **Go Linter**（golangci-lint）
   - 静的解析ツール導入後に Step 3 に追加

4. **E2Eテスト**（Playwright）
   - E2Eテスト実装後に新しいステップとして追加

---

## 検証方法

### 最終検証（全機能テスト）

```bash
# 1. ESLintエラーがないことを確認
cd web && npm run lint

# 2. /self-review スキルの動作確認
/self-review

# 3. ready-for-pr スキルの正常系テスト
/ready-for-pr

# 4. テストスキップオプションの確認
/ready-for-pr --skip-unit-tests
```

### 期待される結果

- ESLint: エラー0件
- /self-review: レビュー結果が表示される（P0/P1がないこと）
- /ready-for-pr: 全ステップが成功し、完了報告が表示される
- `--skip-unit-tests`: Step 4 がスキップされ、他は正常実行

---

## タスク完了の定義

- ✅ Phase 1: ESLintエラー修正完了
- ✅ Phase 2: /self-review スキル実装完了、動作確認OK
- ✅ Phase 3: ready-for-pr スキル実装完了、MVP機能動作確認OK
- ✅ Phase 4: 全テストケースでの動作確認OK
- ✅ Phase 5: ドキュメント整備完了
- ✅ コミット作成（日本語コミットメッセージ）
