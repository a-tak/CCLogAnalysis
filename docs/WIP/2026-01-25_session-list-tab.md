# WIP: プロジェクト詳細ページへのセッション一覧追加

**日付**: 2026-01-25
**セッション**: （次回作業）

## 現在の状態

### ✅ 完了済み
- ✅ Git Root検出機能の修正（2026-01-24）
  - コミット `bb22e0b`: Git Root検出を実際の作業ディレクトリで実行
- ✅ AvgTokens型エラー修正（2026-01-24）
  - コミット `0cf5ffe`: AvgTokensをfloat64型に修正
- ✅ プロジェクト一覧が正しく表示される（106個のプロジェクト）
- ✅ プロジェクト詳細ページの統計・グラフ表示

### ⚠️ 現在の問題
- プロジェクト詳細ページにセッション一覧が表示されない
  - ProjectDetailPageは統計とグラフのみ
  - SessionsPageは別ページで実装済み
- ユーザーがプロジェクト詳細で統計とセッション一覧の両方を見たい

### 💡 解決策
- mainブランチのセッションリストパフォーマンス改善をマージ
- ProjectDetailPageにタブUI追加（統計/セッション一覧）
- ページネーション実装（20件/ページ）

---

## mainブランチのパフォーマンス改善内容

### コミット情報
- **コミット**: `46945da feat: セッションリストパフォーマンス改善`
- **日付**: 2026-01-24 20:45

### 改善内容
**問題点**:
- 大量セッション（757件）で2秒以上の遅延
- 複雑なサブクエリが O(n × 3) で実行

**解決策**:
- `first_user_message`をDB事前計算方式に変更
- セッション作成時に計算してカラムに格納
- リスト表示時はシンプルなSELECT

**効果**:
- 2秒以上 → 0.5秒未満（75%以上の改善）

### 変更ファイル
- `internal/db/schema.sql` - `first_user_message`カラム追加
- `internal/db/sessions.go` - 計算ロジック追加、CreateSession/ListSessions修正
- `internal/db/sessions_test.go` - 6つの新規テスト追加

---

## 実装プラン概要

### 作業ステップ

#### Step 1: mainブランチマージ
```bash
git fetch origin
git merge origin/main
```

**予想されるコンフリクト**:
- `internal/db/schema.sql`
- `internal/db/sessions.go`
- `internal/db/sessions_test.go`

**解決方針**: mainの変更を優先、project-summaryの独自変更（Git Root検出）を保持

#### Step 2: データベースマイグレーション
```bash
# 開発環境: 再作成
rm bin/ccloganalysis.db
./bin/ccloganalysis

# または本番環境: マイグレーション
sqlite3 bin/ccloganalysis.db "ALTER TABLE sessions ADD COLUMN first_user_message TEXT DEFAULT '';"
```

#### Step 3: フロントエンド実装

**新規コンポーネント**:
1. `web/src/components/SessionListTab.tsx` - セッション一覧表示
2. `web/src/components/Pagination.tsx` - ページャーUI

**変更ファイル**:
1. `web/src/pages/ProjectDetailPage.tsx` - タブUI追加

**実装内容**:
- 統計タブ: 既存の統計カード + グラフ
- セッション一覧タブ: SessionsPageの実装を参考にセッションテーブル表示
- ページネーション: クライアント側で20件/ページ

#### Step 4: テスト・動作確認
1. バックエンドテスト全パス確認
2. フロントエンドビルド
3. ブラウザで動作確認
4. パフォーマンステスト（757件セッション）

#### Step 5: コミット
```
feat: プロジェクト詳細ページにセッション一覧タブを追加

mainブランチのセッションリストパフォーマンス改善をマージし、
プロジェクト詳細ページにセッション一覧タブを追加。
```

---

## 技術的な詳細

### UI設計

**タブ構成**:
```
┌─────────────────────────────────────┐
│ プロジェクト詳細: {project-name}      │
├─────────────────────────────────────┤
│ [統計・グラフ] [セッション一覧]        │ ← タブ
├─────────────────────────────────────┤
│ 統計タブの場合:                       │
│ - 統計カード × 4                      │
│ - トークン使用量グラフ                │
│                                       │
│ セッション一覧タブの場合:             │
│ - セッションテーブル                  │
│ - ページネーション                    │
└─────────────────────────────────────┘
```

### ページネーション戦略

**クライアント側ページネーション** (初期実装):
- 全セッションを一度に取得
- フロントエンドで20件ずつ分割表示
- 利点: シンプル、ページ切り替え高速
- mainの改善により全件取得でも0.5秒未満

**将来の改善案** (サーバー側ページネーション):
- API: `/api/sessions?project=xxx&limit=20&offset=0`
- 1000件以上のプロジェクトが増えたら検討

### パフォーマンス考慮

**初期ロード**:
- 757件で0.5秒未満（mainの改善効果）
- クライアント側ページネーションでも十分高速

**メモリ使用量**:
- 757セッション × 約200バイト/セッション = 約150KB
- ブラウザメモリ的に問題なし

---

## Critical Files

### マージ対象（mainブランチ）
1. `internal/db/schema.sql` - スキーマ変更
2. `internal/db/sessions.go` - ロジック変更
3. `internal/db/sessions_test.go` - テスト追加

### 変更ファイル
4. `web/src/pages/ProjectDetailPage.tsx` - タブUI追加

### 新規ファイル
5. `web/src/components/SessionListTab.tsx` - セッション一覧コンポーネント
6. `web/src/components/Pagination.tsx` - ページャーコンポーネント

---

## エッジケース処理

| ケース | 処理 |
|--------|------|
| セッション0件 | 「セッションが見つかりません」表示 |
| 1ページのみ | ページネーション非表示 |
| ローディング中 | スピナー表示 |
| エラー発生 | エラーメッセージ表示 |
| プロジェクト名不正 | 404またはエラー |

---

## 次のセッションでやること

### Phase 1: mainブランチマージ（30分）
1. `git fetch origin && git merge origin/main`
2. コンフリクト解消
   - `schema.sql`: mainの変更を採用
   - `sessions.go`: mainの変更 + project-summaryの独自変更をマージ
   - `sessions_test.go`: 両方のテスト保持
3. `go test ./...`でテスト全パス確認

### Phase 2: データベースマイグレーション（5分）
1. `rm bin/ccloganalysis.db`
2. サーバー起動で新規スキーマ作成

### Phase 3: フロントエンド実装（60分）
1. `SessionListTab.tsx`作成
   - SessionsPageの実装を参考
   - ページネーション追加
2. `Pagination.tsx`作成
   - シンプルなページャーUI
3. `ProjectDetailPage.tsx`修正
   - タブUI追加
   - 既存の統計部分を`<TabsContent>`で囲む

### Phase 4: テスト・動作確認（30分）
1. `npm run build`
2. サーバー起動
3. ブラウザで確認
   - タブ切り替え動作
   - セッション一覧表示
   - ページネーション動作
4. パフォーマンステスト（voxmentプロジェクト: 757件）

### Phase 5: コミット（10分）
1. Git commit
2. WIPドキュメント更新（完了報告）

**推定合計時間**: 約2時間15分

---

## 参考情報

### 関連ドキュメント
- プランファイル: `.claude/plans/shiny-jingling-beaver.md`
- mainブランチ設計ドキュメント: `docs/WIP/2026-01-24_セッションリストパフォーマンス改善.md`

### 関連コミット
- `bb22e0b`: Git Root検出修正
- `0cf5ffe`: AvgTokens型修正
- `46945da` (main): セッションリストパフォーマンス改善

### テストコマンド
```bash
# バックエンド
go test ./internal/db/... -v

# フロントエンド
cd web && npm run build

# 統合テスト
./bin/ccloganalysis
open http://localhost:8080/projects/-Users-a-tak-Documents-GitHub-voxment
```

---

## メモ

- mainブランチのパフォーマンス改善が非常に効果的（75%以上の高速化）
- クライアント側ページネーションで十分なパフォーマンス
- SessionsPageの実装を再利用できるため、実装は比較的シンプル
- タブUIにより、統計とセッション一覧を自然に切り替え可能
