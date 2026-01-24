# プロジェクト詳細ページへのセッション一覧追加プラン

**日付**: 2026-01-25
**タスク**: ProjectDetailPageにセッション一覧タブを追加
**ゴール**: 統計・グラフとセッション一覧の両方を1つのページで確認できるようにする

## 背景

### 現状
- **ProjectDetailPage**: 統計サマリーとトークン使用量グラフのみ表示
- **SessionsPage**: セッション一覧は別ページで実装済み
- **ユーザー要望**: プロジェクト詳細ページでセッションも見たい

### mainブランチの改善内容
- コミット `46945da`: セッションリストパフォーマンス大幅改善
  - 2秒以上 → 0.5秒未満
  - `first_user_message`をDB事前計算方式に変更
  - 大量セッション（757件など）対応

### 実装方針（ユーザー確認済み）
1. ✅ **mainブランチを先にマージ** - パフォーマンス改善を取り込む
2. ✅ **タブで分離** - 統計タブとセッション一覧タブ
3. ✅ **ページネーション実装** - 初期表示20-50件、ページャーで追加読み込み

---

## 実装ステップ

### Step 1: mainブランチのマージ

**目的**: セッションリストパフォーマンス改善を取り込む

**作業**:
```bash
# 現在のブランチ: project-summary
git fetch origin
git merge origin/main
```

**コンフリクト予測**:
- `internal/db/schema.sql` - sessionsテーブルに`first_user_message`カラム追加
- `internal/db/sessions.go` - CreateSession、ListSessionsの変更
- `internal/db/sessions_test.go` - 新規テスト追加

**解決方針**:
- mainの変更を優先的に取り込む
- project-summaryブランチの独自変更（Git Root検出など）を保持

**検証**:
```bash
# マージ後のテスト実行
go test ./...
cd web && npm run build
```

### Step 2: データベースマイグレーション

**目的**: mainブランチのスキーマ変更を既存DBに適用

**作業**:
1. 既存データベースをバックアップ
   ```bash
   cp bin/ccloganalysis.db bin/ccloganalysis.db.backup
   ```

2. マイグレーション実行または再作成
   ```bash
   # オプション1: 再作成（開発環境推奨）
   rm bin/ccloganalysis.db
   ./bin/ccloganalysis  # 新規スキーマで作成

   # オプション2: マイグレーション（本番環境向け）
   sqlite3 bin/ccloganalysis.db "ALTER TABLE sessions ADD COLUMN first_user_message TEXT DEFAULT '';"
   ```

3. 既存セッションの`first_user_message`を再計算
   - CreateSessionで自動計算されるため、新規セッションは自動的に設定される
   - 既存セッションは空文字列のまま（リスト表示時に影響は少ない）

### Step 3: ProjectDetailPageにタブUIを追加

**ファイル**: `web/src/pages/ProjectDetailPage.tsx`

**現在の構造**:
```tsx
<div>
  <h1>プロジェクト詳細</h1>
  <div>統計カード × 4</div>
  <div>グラフ</div>
</div>
```

**新しい構造**:
```tsx
<div>
  <h1>プロジェクト詳細</h1>
  <Tabs defaultValue="stats">
    <TabsList>
      <TabsTrigger value="stats">統計・グラフ</TabsTrigger>
      <TabsTrigger value="sessions">セッション一覧</TabsTrigger>
    </TabsList>

    <TabsContent value="stats">
      <div>統計カード × 4</div>
      <div>グラフ</div>
    </TabsContent>

    <TabsContent value="sessions">
      <SessionListTab projectName={projectName} />
    </TabsContent>
  </Tabs>
</div>
```

**変更箇所**:
- L51-223: 既存の統計・グラフ部分を`<TabsContent value="stats">`で囲む
- 新規: `<TabsContent value="sessions">`を追加
- Tabsコンポーネントは既にインポート済み（L5）

### Step 4: SessionListTabコンポーネントの作成

**ファイル**: `web/src/components/SessionListTab.tsx` (新規作成)

**機能**:
1. セッション一覧の取得（`api.getSessions(projectName)`）
2. ページネーション（初期表示: 20件）
3. テーブル表示（SessionsPageの実装を参考）

**実装内容**:
```tsx
interface SessionListTabProps {
  projectName: string
}

export function SessionListTab({ projectName }: SessionListTabProps) {
  const [sessions, setSessions] = useState<Session[]>([])
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const pageSize = 20

  useEffect(() => {
    // セッション取得
    api.getSessions(projectName).then(...)
  }, [projectName])

  // ページネーション計算
  const startIndex = (page - 1) * pageSize
  const endIndex = startIndex + pageSize
  const displayedSessions = sessions.slice(startIndex, endIndex)

  return (
    <Card>
      <CardHeader>
        <CardTitle>セッション一覧</CardTitle>
        <CardDescription>{sessions.length} 件のセッション</CardDescription>
      </CardHeader>
      <CardContent>
        <Table>
          {/* SessionsPageの実装を参考 */}
        </Table>
        <Pagination>
          {/* ページャー */}
        </Pagination>
      </CardContent>
    </Card>
  )
}
```

**参考実装**: `web/src/pages/SessionsPage.tsx` (L1-120)
- テーブル構造をコピー
- カラム: Session ID、Branch、Start Time、Tokens、Errors

### Step 5: ページネーションコンポーネントの追加

**ファイル**: `web/src/components/Pagination.tsx` (新規作成)

**機能**:
- 前ページ/次ページボタン
- ページ番号表示
- 全ページ数表示

**実装**:
```tsx
interface PaginationProps {
  currentPage: number
  totalPages: number
  onPageChange: (page: number) => void
}

export function Pagination({ currentPage, totalPages, onPageChange }: PaginationProps) {
  return (
    <div className="flex items-center justify-between mt-4">
      <Button
        variant="outline"
        onClick={() => onPageChange(currentPage - 1)}
        disabled={currentPage === 1}
      >
        前へ
      </Button>
      <span className="text-sm text-muted-foreground">
        {currentPage} / {totalPages} ページ
      </span>
      <Button
        variant="outline"
        onClick={() => onPageChange(currentPage + 1)}
        disabled={currentPage === totalPages}
      >
        次へ
      </Button>
    </div>
  )
}
```

### Step 6: テスト・動作確認

**確認項目**:
1. ✅ mainブランチのマージが成功（テスト全パス）
2. ✅ データベースマイグレーション完了
3. ✅ ProjectDetailPageでタブ切り替えが動作
4. ✅ 統計タブで既存の統計・グラフが表示
5. ✅ セッション一覧タブでセッションが表示
6. ✅ ページネーションで20件ずつ表示
7. ✅ パフォーマンス: 大量セッションでも0.5秒未満で表示

**テストコマンド**:
```bash
# バックエンドテスト
go test ./...

# フロントエンドビルド
cd web && npm run build

# サーバー起動
./bin/ccloganalysis

# ブラウザで確認
# http://localhost:8080/projects/{project-name}
```

### Step 7: コミット

**コミットメッセージ**:
```
feat: プロジェクト詳細ページにセッション一覧タブを追加

mainブランチのセッションリストパフォーマンス改善をマージし、
プロジェクト詳細ページにセッション一覧タブを追加。

変更内容:
- mainブランチのマージ（セッションリストパフォーマンス改善）
- ProjectDetailPageにタブUI追加（統計/セッション一覧）
- SessionListTabコンポーネント新規作成
- ページネーション実装（20件/ページ）

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
```

---

## Critical Files

### 変更ファイル
1. **`web/src/pages/ProjectDetailPage.tsx`**
   - タブUI追加
   - 統計部分を`<TabsContent>`で囲む
   - セッション一覧タブを追加

### 新規ファイル
2. **`web/src/components/SessionListTab.tsx`**
   - セッション一覧表示ロジック
   - ページネーション制御

3. **`web/src/components/Pagination.tsx`**
   - ページャーUI

### マージ対象（mainブランチ）
4. **`internal/db/schema.sql`**
   - `first_user_message`カラム追加

5. **`internal/db/sessions.go`**
   - `calculateFirstUserMessage()`追加
   - `CreateSession()`修正
   - `ListSessions()`簡素化

6. **`internal/db/sessions_test.go`**
   - パフォーマンス改善のテスト追加

---

## パフォーマンス考慮事項

### ページネーション戦略
- **クライアント側ページネーション**: 全セッションを取得し、フロントエンドで分割表示
  - 利点: シンプル、ページ切り替え高速
  - 欠点: 初期ロード時に全件取得（mainの改善で0.5秒未満なので許容範囲）

- **サーバー側ページネーション（将来の改善案）**:
  - API: `/api/sessions?project=xxx&limit=20&offset=0`
  - 利点: 初期ロード高速、メモリ効率良い
  - 欠点: API修正必要、ページ切り替え時にネットワーク通信

### 初期実装
- クライアント側ページネーションで実装
- mainブランチの改善により十分高速（757件で0.5秒未満）
- 将来的にさらに大量（1000件以上）になったらサーバー側対応を検討

---

## エッジケース処理

| ケース | 処理 |
|--------|------|
| セッションが0件 | 「セッションが見つかりません」メッセージ表示 |
| ページ数が1ページのみ | ページネーション非表示 |
| ローディング中 | スピナー表示 |
| エラー発生 | エラーメッセージ表示 |
| プロジェクト名が不正 | 404ページまたはエラー |

---

## 次セッションでの作業フロー

### Phase 1: mainブランチマージ
1. `git fetch origin && git merge origin/main`
2. コンフリクト解消（schema.sql、sessions.go、sessions_test.go）
3. `go test ./...`で全テストパス確認

### Phase 2: データベースマイグレーション
1. `rm bin/ccloganalysis.db`（開発環境）
2. サーバー起動で新規スキーマ作成

### Phase 3: フロントエンド実装
1. `SessionListTab.tsx`作成
2. `Pagination.tsx`作成
3. `ProjectDetailPage.tsx`にタブ追加

### Phase 4: 動作確認・コミット
1. ビルド・起動・ブラウザ確認
2. パフォーマンステスト
3. コミット

---

## 検証方法

### 1. ユニットテスト
```bash
# バックエンド（mainブランチのテストが全パス）
go test ./internal/db/... -v -run TestCalculateFirstUserMessage
go test ./internal/db/... -v -run TestListSessions
```

### 2. 統合テスト
```bash
# サーバー起動
./bin/ccloganalysis

# ブラウザで確認
open http://localhost:8080/projects/-Users-a-tak-Documents-GitHub-CCLogAnalysis-worktrees-project-summary

# タブ切り替え確認
# ページネーション確認（20件以上のプロジェクトで）
```

### 3. パフォーマンステスト
- 757件のセッションがあるプロジェクト（voxment）で確認
- セッション一覧タブの表示速度が0.5秒未満であることを確認
- ページ切り替えが即座に反応することを確認

---

## 期待される結果

### ユーザー体験
- プロジェクト詳細ページで統計とセッション一覧を簡単に切り替え
- 統計タブ: プロジェクト全体の傾向を把握
- セッション一覧タブ: 個別セッションの詳細を確認

### パフォーマンス
- 大量セッション（757件）でも0.5秒未満で表示
- ページネーションにより見やすい20件/ページ表示
- タブ切り替えは瞬時

### 保守性
- SessionsPageの実装を再利用
- コンポーネント分割により責務明確
- mainブランチのパフォーマンス改善を活用
