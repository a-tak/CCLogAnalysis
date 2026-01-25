# cr-worktreeスキル拡張: WIPドキュメント自動生成機能

**作業日**: 2026-01-25

---

## 概要

cr-worktreeスキルに、説明文入力時のWIPドキュメント自動生成機能を追加します。

### 背景

現在のcr-worktreeスキルには3つのパターンがあります：
1. **Issue番号** → GitHub Issue情報取得、ブランチ名自動生成、`/issue`コマンド実行
2. **ブランチ名** → ブランチ名そのまま使用、Claude起動のみ
3. **説明文** → ブランチ名自動生成、Claude起動のみ（**WIPドキュメントなし**）

パターン3（説明文入力）の場合、新しいターミナルで参照できるWIPドキュメントがないため、作業内容の記録・共有が不便です。

### 目的

説明文から自動的にWIPドキュメントを生成し、新しいworktree内の`docs/WIP/`に配置することで、Claude起動時に即座に参照可能にします。

---

## 要件

### 機能要件

1. **対象パターン**: パターン3（説明文入力）のみ
2. **生成タイミング**: worktree作成・環境整備完了後、Claude起動前
3. **配置場所**: 新しいworktree内の`docs/WIP/YYYY-MM-DD_タスク名.md`
4. **git管理**: 未追跡ファイルとして残す（自動コミットしない）
5. **既存フォーマット準拠**: 既存WIPドキュメントの形式に従う

### 非機能要件

1. **エラーハンドリング**: WIPドキュメント生成失敗時も処理続行（警告のみ）
2. **パフォーマンス**: 処理時間への影響は1秒未満（無視できる）
3. **後方互換性**: パターン1, 2には影響なし

---

## 実装内容

### Phase 1: WIPドキュメント生成機能実装（`cr-worktree.sh`）

#### 1.1 WIPテンプレート生成関数の追加

**場所**: `.claude/skills/cr-worktree/scripts/cr-worktree.sh`（新規関数）

```bash
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
  sed -i '' "s/\${TASK_DESCRIPTION}/${TASK_DESCRIPTION}/g" "$WIP_FILE"
  sed -i '' "s/\${TODAY}/${TODAY}/g" "$WIP_FILE"

  if [ $? -eq 0 ]; then
    echo "✅ WIPドキュメント生成完了: $WIP_FILE"
    return 0
  else
    echo "⚠️  警告: WIPドキュメント生成に失敗しました"
    return 1
  fi
}
```

**注意点**:
- heredoc内で変数展開を防ぐため、`<<'EOF'`を使用（クォート付き）
- `sed`コマンドで後から変数を置換
- macOSの`sed`は`-i ''`が必要（バックアップなしの上書き）

#### 1.2 スクリプト引数の拡張

**場所**: `.claude/skills/cr-worktree/scripts/cr-worktree.sh`（引数処理部分）

```bash
# 引数の取得
ARG="$1"
FROM_CURRENT=false
WITH_ISSUE_COMMAND=false
WITH_DESCRIPTION=""  # 追加: 説明文を保持

# オプション解析
shift  # 最初の引数（ブランチ名）をスキップ
for arg in "$@"; do
  case $arg in
    --from-current)
      FROM_CURRENT=true
      ;;
    --with-issue-command)
      WITH_ISSUE_COMMAND=true
      ;;
    --with-description=*)  # 追加
      WITH_DESCRIPTION="${arg#*=}"
      ;;
  esac
done
```

#### 1.3 WIPドキュメント生成の呼び出し

**場所**: `.claude/skills/cr-worktree/scripts/cr-worktree.sh`（319行目付近、環境整備完了後）

```bash
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
```

---

### Phase 2: SKILL.md のフロー修正

**場所**: `.claude/skills/cr-worktree/SKILL.md`（175-199行）

#### パターン3の処理手順更新

**現在**:
```markdown
### パターン3: 説明文が指定された場合

**処理手順:**

1. ブランチ名自動生成 → 説明文を英語に変換 + kebab-case化
2. `FROM_CURRENT` フラグを確認
3. スクリプト実行 → `.claude/skills/cr-worktree/scripts/cr-worktree.sh <生成したブランチ名> [--from-current]`
```

**修正後**:
```markdown
### パターン3: 説明文が指定された場合

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
```

---

## 変更対象ファイル

| ファイルパス | 変更内容 | 影響範囲 |
|------------|---------|---------|
| `.claude/skills/cr-worktree/scripts/cr-worktree.sh` | WIPドキュメント生成関数追加、引数処理拡張、生成呼び出し | パターン3のみ |
| `.claude/skills/cr-worktree/SKILL.md` | パターン3の処理手順詳細化、`--with-description`オプション追加 | Claude Agent動作変更 |

---

## テスト計画

### テストケース

| テスト名 | 実行コマンド | 期待結果 |
|---------|------------|---------|
| **基本動作** | `/cr-worktree "セッション詳細画面のUI改善"` | WIPドキュメント生成、Claude起動 |
| **長い説明文** | `/cr-worktree "セッション詳細画面のUI改善とパフォーマンス最適化、およびレスポンシブデザイン対応"` | ファイル名50文字切り詰め、WIPドキュメント生成 |
| **特殊文字** | `/cr-worktree "「ログ解析」バグ修正 #123"` | 特殊文字が適切にエスケープ、WIPドキュメント生成 |
| **--from-current** | `/cr-worktree "テストタスク" --from-current` | 現在のブランチから分岐、WIPドキュメント生成 |
| **パターン1（Issue）** | `/cr-worktree 123` | WIPドキュメント生成なし、`/issue`コマンド実行 |
| **パターン2（ブランチ名）** | `/cr-worktree feature-test` | WIPドキュメント生成なし、Claude起動のみ |

### 検証項目

#### Phase 1: スクリプト動作確認
- [ ] WIPドキュメント生成関数が正常に動作
- [ ] `--with-description`オプションが正しく解析される
- [ ] WIPディレクトリが存在しない場合、警告のみで続行
- [ ] ファイル名が50文字で切り詰められる

#### Phase 2: エンドツーエンドテスト
- [ ] `/cr-worktree "説明文"`でworktree作成
- [ ] 新しいworktree内の`docs/WIP/`にWIPドキュメントが存在
- [ ] WIPドキュメントがgit未追跡ファイルとして存在（`git status`で確認）
- [ ] 新しいターミナルでClaude起動時、WIPドキュメントが参照可能

#### Phase 3: 後方互換性確認
- [ ] パターン1（Issue番号）が正常に動作（WIPドキュメント生成なし）
- [ ] パターン2（ブランチ名）が正常に動作（WIPドキュメント生成なし）

---

## 注意事項

### セキュリティとプライバシー

- **説明文**: ユーザー入力の任意テキストが含まれるため、個人情報に注意
- **パブリック公開前**: CLAUDE.mdのチェックリストに従い、個人情報が含まれていないか確認

### macOS特有の注意点

- **sed コマンド**: macOSの`sed`は`-i ''`が必要（GNU sedとの違い）
- **ターミナルアプリ**: Ghostty → Terminal.appの優先順位で起動

### エラーハンドリング

- **WIPディレクトリが存在しない**: 警告のみ、処理続行
- **ファイル作成権限エラー**: 警告のみ、処理続行
- **説明文が空**: WIPドキュメント生成スキップ（`-n "$WITH_DESCRIPTION"`チェック）

---

## 完了チェックリスト

### 実装
- [ ] `cr-worktree.sh`にWIPドキュメント生成関数追加
- [ ] `cr-worktree.sh`に`--with-description`オプション処理追加
- [ ] `cr-worktree.sh`に生成呼び出しロジック追加
- [ ] `SKILL.md`のパターン3処理手順更新

### テスト
- [ ] 基本動作テスト（説明文入力）
- [ ] 長い説明文テスト（50文字切り詰め確認）
- [ ] 特殊文字テスト
- [ ] パターン1, 2の後方互換性確認

### ドキュメント
- [ ] SKILL.mdの使用例更新
- [ ] WIPドキュメント仕様の記載

---

## 次のステップ

1. `cr-worktree.sh`のWIPドキュメント生成関数実装
2. スクリプト引数処理の拡張
3. SKILL.mdのパターン3処理手順更新
4. テスト実施（基本動作、エッジケース、後方互換性）
5. ドキュメント整備

---

**このプランはPlan modeで作成されました。実装前にユーザー承認が必要です。**
