---
model: claude-haiku-4-5
name: release
description: GitHubリリースを自動作成。前回リリースからの差分を分析し、CHANGELOGを更新してリリースを作成します。
argument-hint: [patch|minor|major|version]
disable-model-invocation: true
allowed-tools: Bash(git *), Bash(gh *), Edit, Read, Grep
---

# Release スキル

GitHubリリースを自動作成します。前回のリリースからの差分を分析し、CHANGELOGを更新してリリースを作成します。

## 📋 クイックリファレンス

### 引数パターン

| 引数 | 動作 |
|------|------|
| なし | 自動判定（コミット内容からバージョンを決定） |
| `patch` / `minor` / `major` | セマンティックバージョニングに基づいてバージョンアップ |
| `0.5.0` | 指定したバージョン番号を使用 |

### 処理フロー概要

1. **前回リリース確認** → 最新タグを取得
2. **差分確認** → コミット履歴を取得・分析
3. **バージョン決定** → セマンティックバージョニングまたは引数から決定
4. **CHANGELOG更新** → `Edit` ツールで更新
5. **リリース作成** → スクリプト実行

## 処理フロー

このスキルは以下の手順でリリースを作成します：

### ステップ1: 前回リリースからの差分確認

```bash
# 最新タグを取得
LATEST_TAG=$(git tag --sort=-v:refname | head -1)

# 差分コミットを確認
git log ${LATEST_TAG}..HEAD --pretty=format:"%h %s" --reverse
```

- 差分コミットがない場合: エラーメッセージを表示して終了
- 差分がある場合: 次のステップへ

### ステップ2: コミット内容の分析

各コミットの詳細を `git show <commit> --stat` で確認し、以下のカテゴリに分類：

| カテゴリ | 判定基準 | セマンティックバージョニング |
|---------|---------|------------------------------|
| **Fixed** | `fix:` で始まる、バグ修正 | Patch |
| **Added** | `feat:` で始まる、新機能追加 | Minor |
| **Changed** | `chore:`, `refactor:` など | Patch |
| **Security** | セキュリティ関連 | Patch (または Minor) |
| **Breaking** | `BREAKING CHANGE:` | Major |

### ステップ3: バージョン番号の決定

引数に基づいてバージョン番号を決定：

```text
引数なし → コミット内容から自動判定（ステップ2の結果を使用）
"patch" → パッチバージョンアップ（例: 0.4.3 → 0.4.4）
"minor" → マイナーバージョンアップ（例: 0.4.3 → 0.5.0）
"major" → メジャーバージョンアップ（例: 0.4.3 → 1.0.0）
"X.Y.Z" → 指定されたバージョンを使用
```

### ステップ4: CHANGELOG更新

`Edit` ツールで `CHANGELOG.md` を更新：

```markdown
## [Unreleased]

---

## [X.Y.Z] - YYYY-MM-DD

### Fixed

- 修正内容（Issue #XXへのリンク付き）

### Added

- 追加内容

### Changed

- 変更内容

---

## [前回のバージョン] - ...
```

**重要な注意事項:**

- 日付は今日の日付（`YYYY-MM-DD` 形式）を使用
- Issue番号がある場合はリンクを追加（例: `（Issue #21）`）
- コミットメッセージから意味のある説明を抽出

### ステップ5: リリースノートの生成

GitHubリリース用のリリースノートを生成（Markdown形式）：

```markdown
## 🐛 Fixed

- **修正内容の概要** ([#XX](https://github.com/a-tak/CCLogAnalysis/issues/XX))
  - 詳細1
  - 詳細2

## ✨ Added

- **追加内容の概要**
  - 詳細

## 🔧 Changed

- **変更内容の概要**
  - 詳細

---

**Full Changelog**: https://github.com/a-tak/CCLogAnalysis/compare/vX.Y.Z...vA.B.C
```

### ステップ6: スクリプト実行

`scripts/create-release.sh` を実行してリリース作成：

```bash
.claude/skills/release/scripts/create-release.sh <version> "<release-notes>"
```

**引数:**

- `<version>`: バージョン番号（例: `0.4.3`）※ `v` プレフィックスなし
- `"<release-notes>"`: リリースノート（複数行可、ダブルクォートで囲む）

### スクリプトの役割

`scripts/create-release.sh` は以下を実行：

1. CHANGELOGのコミット
2. タグ作成（`v` プレフィックス付き）
3. Push（main + タグ）
4. GitHubリリース作成

## 実装例

```bash
# 引数なしの場合（自動判定）
# 1. 最新タグ: v0.4.2
# 2. 差分コミット: 3件（全て fix:）
# 3. バージョン判定: Patch → 0.4.3
# 4. CHANGELOG更新
# 5. リリースノート生成
# 6. スクリプト実行
.claude/skills/release/scripts/create-release.sh 0.4.3 "$(cat <<'EOF'
## 🐛 Fixed
- **グラフの時系列方向を完全修正** ([#21](https://github.com/a-tak/CCLogAnalysis/issues/21))
  - 全ページでソート順を昇順に統一

---

**Full Changelog**: https://github.com/a-tak/CCLogAnalysis/compare/v0.4.2...v0.4.3
EOF
)"
```

## エラーハンドリング

- 差分コミットがない場合: エラーメッセージを表示して終了
- CHANGELOGが既に更新されている場合: 確認を求める
- GitHub CLI未認証: `gh auth login` を案内
- タグが既に存在する場合: エラーメッセージを表示

## 注意事項

- リリース作成前に必ずテストが全てパスしていることを確認してください
- CHANGELOG.md のフォーマットは Keep a Changelog に準拠しています
- セマンティックバージョニングに従ってバージョン番号を決定します
