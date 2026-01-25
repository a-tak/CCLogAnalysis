# グループ別のグラフ表示機能追加 実装プラン

**作成日**: 2026-01-25
**対象ブランチ**: add-group-graphs

---

## 概要

プロジェクトグループ詳細ページ（GroupDetailPage）にタイムラインチャートを追加し、グループ内の全プロジェクトのトークン使用量推移をグラフで可視化する。

**目標**: ProjectDetailPageと同様のタイムラインチャートをGroupDetailPageに実装する。

---

## 現状分析

### 実装済み
- ✅ プロジェクトグループ機能（データベース、API、フロントエンド）
- ✅ グループ統計（数値のサマリーカード）
- ✅ ProjectDetailPage のタイムラインチャート
- ✅ TimeSeriesResponse 型定義

### 未実装（今回追加）
- ❌ グループ別タイムライン統計のデータベースクエリ
- ❌ グループ別タイムラインAPIエンドポイント
- ❌ GroupDetailPage のグラフコンポーネント

---

## 実装方針

既存の実装パターンを踏襲:
- **参考**: `GetTimeSeriesStats` (internal/db/project_stats.go:185-273)
- **参考**: ProjectDetailPage のタイムラインチャート (web/src/pages/ProjectDetailPage.tsx:172-233)
- **アプローチ**: project_group_mappings テーブルと JOIN してグループ内全プロジェクトを集計

---

## 実装手順（TDD準拠）

### Phase 1: データベース層

#### ファイル: `internal/db/group_stats.go`

**実装内容**:
```go
// GetGroupTimeSeriesStats retrieves time-series statistics for a project group
func (db *DB) GetGroupTimeSeriesStats(groupID int64, period string, limit int) ([]TimeSeriesStats, error)
```

**SQLクエリ**:
```sql
SELECT
    STRFTIME('%Y-%m-%d', s.start_time) as period_group,
    MIN(DATE(s.start_time)) as period_start,
    MAX(DATE(s.start_time)) as period_end,
    COUNT(*) as session_count,
    COALESCE(SUM(s.total_input_tokens), 0) as total_input_tokens,
    COALESCE(SUM(s.total_output_tokens), 0) as total_output_tokens,
    COALESCE(SUM(s.total_cache_creation_tokens), 0) as total_cache_creation_tokens,
    COALESCE(SUM(s.total_cache_read_tokens), 0) as total_cache_read_tokens
FROM project_group_mappings pgm
INNER JOIN projects p ON pgm.project_id = p.id
INNER JOIN sessions s ON p.id = s.project_id
WHERE pgm.group_id = ?
GROUP BY period_group
ORDER BY period_start DESC
LIMIT ?
```

**TDD手順**:
1. **Red**: `internal/db/group_stats_test.go` にテスト作成
   - `TestGetGroupTimeSeriesStats` - 正常系
   - `TestGetGroupTimeSeriesStatsPeriods` - day/week/month切り替え
   - `TestGetGroupTimeSeriesStatsNoSessions` - セッション0件
2. **Green**: `GetGroupTimeSeriesStats` 実装
3. **Refactor**: エラーハンドリング改善

---

### Phase 2: API層

#### ファイル: `internal/api/service_db.go`

**実装内容**:
```go
// GetProjectGroupTimeline returns time-series statistics for a project group
func (s *DatabaseSessionService) GetProjectGroupTimeline(groupID int64, period string, limit int) (*TimeSeriesResponse, error)
```

- `GetProjectTimeline` (service_db.go:300-343) と同じパターン
- グループIDで存在確認
- `db.GetGroupTimeSeriesStats` を呼び出し
- `TimeSeriesResponse` に変換

#### ファイル: `internal/api/types.go`

**変更内容**:
- `SessionService` インターフェースに新メソッド追加:
```go
GetProjectGroupTimeline(groupID int64, period string, limit int) (*TimeSeriesResponse, error)
```

#### ファイル: `internal/api/handlers_groups.go`

**実装内容**:
```go
// getGroupTimelineHandler returns time-series statistics for a project group
func (h *Handler) getGroupTimelineHandler(w http.ResponseWriter, r *http.Request)
```

- `getProjectTimelineHandler` と同じパターン
- クエリパラメータ: `period` (day/week/month), `limit` (default: 30)
- エラーレスポンス: 400, 404, 500

#### ファイル: `internal/api/router.go`

**変更内容**:
```go
mux.HandleFunc("GET /api/groups/{id}/timeline", h.getGroupTimelineHandler)
```

**TDD手順**:
1. **Red**: `internal/api/handlers_groups_test.go` にテスト作成
   - `TestGetGroupTimelineHandler` - 正常系
   - `TestGetGroupTimelineHandlerWithPeriod` - periodパラメータ
   - `TestGetGroupTimelineHandlerInvalidGroupID` - 無効ID (400)
   - `TestGetGroupTimelineHandlerNotFound` - 存在しないグループ (404)
2. **Green**: API層の実装
3. **Refactor**: エラーハンドリング改善

---

### Phase 3: フロントエンド層

#### ファイル: `web/src/lib/api/client.ts`

**実装内容**:
```typescript
// Get project group timeline
async getProjectGroupTimeline(
  groupId: number,
  period: 'day' | 'week' | 'month' = 'day',
  limit = 30
): Promise<TimeSeriesResponse> {
  return fetchApi<TimeSeriesResponse>(
    `/groups/${groupId}/timeline?period=${period}&limit=${limit}`
  )
}
```

#### ファイル: `web/src/pages/GroupDetailPage.tsx`

**変更内容**:

1. **State追加**:
```typescript
const [timeline, setTimeline] = useState<TimeSeriesResponse | null>(null)
const [period, setPeriod] = useState<'day' | 'week' | 'month'>('day')
```

2. **データ取得** (useEffect内):
```typescript
const [groupData, statsData, timelineData] = await Promise.all([
  api.getProjectGroup(groupId),
  api.getProjectGroupStats(groupId),
  api.getProjectGroupTimeline(groupId, period, 30),
])
setTimeline(timelineData)
```

3. **タイムラインチャート追加** (統計カードの後):
- ProjectDetailPage.tsx (172-233行) のコードをコピー
- `api.getProjectGroupTimeline` に変更
- グラフUI/UXは完全に同一

**インポート追加**:
```typescript
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'
```

---

## クリティカルファイル

実装において最も重要なファイル:

1. **`internal/db/group_stats.go`** - グループタイムライン統計の実装
2. **`internal/api/handlers_groups.go`** - グループタイムラインエンドポイント
3. **`web/src/pages/GroupDetailPage.tsx`** - フロントエンドのグラフコンポーネント

参考実装:
- **`internal/db/project_stats.go`** - GetTimeSeriesStats (185-273行)
- **`web/src/pages/ProjectDetailPage.tsx`** - タイムラインチャート (172-233行)

---

## テスト計画

### データベース層テスト
- ✅ 複数プロジェクト・複数日付のデータ集計
- ✅ 期間パラメータ（day/week/month）の切り替え
- ✅ 境界値（セッション0件、プロジェクト0件）
- ✅ エラーケース（無効な期間パラメータ）

### API層テスト
- ✅ 正常系（200 OK、正しいJSONレスポンス）
- ✅ パラメータ（period/limitのデフォルト値と指定値）
- ✅ エラー（400 Bad Request、404 Not Found）
- ✅ Content-Type: application/json

### フロントエンドテスト（手動）
- ✅ グラフ描画（データが正しく表示される）
- ✅ 期間切り替え（タブクリックで期間が変更される）
- ✅ レスポンシブ（画面サイズに応じて適切に表示される）
- ✅ エラー表示（データなし時のメッセージ表示）

---

## 検証方法（エンドツーエンド）

### 1. サーバー起動
```bash
/server-management start dev
```

### 2. ブラウザで確認
1. プロジェクト一覧ページ → 「グループ別」タブ
2. グループカードをクリック → グループ詳細ページ
3. タイムラインチャートが表示されることを確認
4. 期間切り替え（日別/週別/月別）の動作確認

### 3. APIエンドポイントテスト
```bash
# グループ一覧を取得してグループIDを確認
curl http://localhost:8080/api/groups

# グループタイムラインを取得
curl "http://localhost:8080/api/groups/1/timeline?period=day&limit=30"
```

---

## エラーハンドリング

### データベース層
- 無効な `period` → `fmt.Errorf("invalid period: %s", period)`
- グループIDが存在しない → SQLクエリ結果が空（エラーではない）
- 日時パースエラー → `fmt.Errorf("failed to parse period start: %w", err)`

### API層
- グループID解析失敗 → 400 Bad Request
- グループが存在しない → 404 Not Found
- データベースエラー → 500 Internal Server Error
- 全て `ErrorResponse` 型を使用

### フロントエンド層
- API呼び出し失敗 → エラー表示
- データなし → 「データがありません」メッセージ
- ローディング状態 → スピナー表示

---

## コミット戦略

### Commit 1: データベース層
```
feat: グループタイムライン統計のデータベース層を実装

- GetGroupTimeSeriesStats メソッドを追加
- 複数プロジェクトのセッションを期間ごとに集計
- テストケース追加（正常系、境界値、エラー）

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
```

### Commit 2: API層
```
feat: グループタイムライン統計のAPIエンドポイントを実装

- GET /api/groups/{id}/timeline エンドポイント追加
- GetProjectGroupTimeline サービスメソッド実装
- ハンドラーテスト追加

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
```

### Commit 3: フロントエンド層
```
feat: グループ詳細ページにタイムラインチャートを追加

- getProjectGroupTimeline メソッドをAPIクライアントに追加
- GroupDetailPage にグラフコンポーネント実装
- 期間切り替え機能（日別/週別/月別）

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
```

---

## ドキュメント更新

### `docs/API設計.md`
新規エンドポイントを追加:
```markdown
#### GET /api/groups/{id}/timeline

グループのタイムライン統計を取得

**クエリパラメータ**:
- `period` (string, optional): 集計期間 ("day" | "week" | "month", default: "day")
- `limit` (number, optional): 最大データポイント数 (default: 30)

**レスポンス**: `TimeSeriesResponse`
```

### `docs/WIP/2026-01-25_グループ別のグラフ表示機能追加.md`
実装完了後、完了チェックリストを更新

---

## リスクと緩和策

### リスク
- **低リスク**: 既存パターンの踏襲なので、実装の不確実性は低い
- **注意点**: JOINクエリのパフォーマンス（大量データ時）

### 緩和策
- LIMIT句でデータ量を制限（デフォルト30件）
- インデックスは既に存在（project_group_mappings.group_id）

---

## 見積もり

### 実装時間
- データベース層: 1-2時間（テスト含む）
- API層: 1-2時間（テスト含む）
- フロントエンド層: 1時間
- 統合テスト・調整: 1時間
- **合計**: 4-6時間

### 複雑度
- **低**: 既存パターンの踏襲、新規設計不要
- **テストカバレッジ**: 高（TDD準拠）

---

## 成功基準

### 機能要件
- ✅ グループ詳細ページにタイムラインチャートが表示される
- ✅ 期間切り替え（日別/週別/月別）が動作する
- ✅ グループ内の全プロジェクトのセッションが集計される
- ✅ データがない場合に適切なメッセージが表示される

### 非機能要件
- ✅ 全てのテストがパスする
- ✅ レスポンシブデザインが適用される
- ✅ エラーハンドリングが適切に実装される
- ✅ 既存のコードスタイルに従っている
