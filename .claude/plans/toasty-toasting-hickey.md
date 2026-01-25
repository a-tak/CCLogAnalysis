# 会話履歴UI表示仕様変更

**作成日**: 2026-01-25

## 概要

セッション詳細ページでの会話履歴表示について、以下の2点を改善します：

1. **メッセージの左右配置を逆転**：ユーザーメッセージを右寄せ、アシスタントメッセージを左寄せに変更（一般的なチャットUIの慣例に合わせる）
2. **ツール出力を常に展開状態で表示**：tool_use（ツール呼び出し）とtool_result（ツール結果）の両方をデフォルトで展開

## 変更対象ファイル

以下の3ファイル、合計3行のみを変更します：

### 1. MessageItem.tsx - メッセージ配置の逆転

**ファイル**: `web/src/components/conversation/MessageItem.tsx`

**変更箇所**: 20行目

```diff
  return (
    <div className={cn(
      "flex",
-     isUser ? "justify-start" : "justify-end"
+     isUser ? "justify-end" : "justify-start"
    )}>
```

**変更内容**:
- ユーザーメッセージ: 左寄せ（`justify-start`） → 右寄せ（`justify-end`）
- アシスタントメッセージ: 右寄せ（`justify-end`） → 左寄せ（`justify-start`）

### 2. ToolUseBlock.tsx - ツール呼び出しを展開状態に

**ファイル**: `web/src/components/conversation/ToolUseBlock.tsx`

**変更箇所**: 9行目

```diff
export function ToolUseBlock({ toolUse }: ToolUseBlockProps) {
-  const [expanded, setExpanded] = useState(false)
+  const [expanded, setExpanded] = useState(true)
```

**変更内容**:
- デフォルト: 閉じた状態 → 開いた状態

### 3. ToolResultBlock.tsx - ツール結果を常に展開状態に

**ファイル**: `web/src/components/conversation/ToolResultBlock.tsx`

**変更箇所**: 10行目

```diff
export function ToolResultBlock({ toolResult }: ToolResultBlockProps) {
-  const [expanded, setExpanded] = useState(toolResult.is_error || false)
+  const [expanded, setExpanded] = useState(true)
  const isError = toolResult.is_error || false
```

**変更内容**:
- デフォルト: エラー時のみ展開 → 常に展開
- エラー時の色分けロジック（11行目以降）は維持

## 影響範囲

### 変更されない部分

- **トグル機能**: ユーザーが手動で展開/折りたたみする機能は維持
- **エラー表示**: エラー時の赤色表示（border-destructive, bg-destructive/10）は維持
- **背景色**: ユーザー/アシスタントの背景色（bg-muted / bg-primary/10）は維持
- **バックエンド**: API、型定義、データベースは一切変更なし

### ユーザーへの影響

**ポジティブ**:
- 一般的なチャットUIの慣例に沿った配置（自分=右、相手=左）
- ツール情報が最初から見える（手動で開く手間が不要）

**注意点**:
- 長いセッションでは縦スクロール量が増加する
- 既存ユーザーは左右が逆になることに最初は違和感を感じる可能性

## 実装手順

### ステップ1: MessageItem.tsxの修正

```bash
# 1. ファイルを開く
code web/src/components/conversation/MessageItem.tsx
```

20行目を以下のように変更:
```typescript
isUser ? "justify-end" : "justify-start"
```

### ステップ2: ToolUseBlock.tsxの修正

```bash
# 2. ファイルを開く
code web/src/components/conversation/ToolUseBlock.tsx
```

9行目を以下のように変更:
```typescript
const [expanded, setExpanded] = useState(true)
```

### ステップ3: ToolResultBlock.tsxの修正

```bash
# 3. ファイルを開く
code web/src/components/conversation/ToolResultBlock.tsx
```

10行目を以下のように変更:
```typescript
const [expanded, setExpanded] = useState(true)
```

## 検証方法

### 1. 開発サーバーの起動

```bash
cd web
npm run dev
```

### 2. セッション詳細ページでの確認

1. ブラウザで `http://localhost:5173` を開く
2. 任意のセッションの詳細ページを開く
3. 以下を確認:

**メッセージ配置**:
- [ ] ユーザーメッセージが右寄せになっている
- [ ] アシスタントメッセージが左寄せになっている
- [ ] 背景色が適切に表示されている（ユーザー=グレー、アシスタント=薄いプライマリカラー）

**ツール展開状態**:
- [ ] ツール呼び出し（Tool: {name}）がデフォルトで展開されている
- [ ] ツール呼び出しのInputが表示されている
- [ ] ツール結果（Tool Result）がデフォルトで展開されている
- [ ] ツール結果の内容が表示されている
- [ ] エラー時のツール結果が赤色で表示されている

**インタラクション**:
- [ ] ツール呼び出しのトグルボタンをクリックして折りたたみができる
- [ ] 折りたたんだ状態から再度展開できる
- [ ] ツール結果のトグルボタンも同様に動作する
- [ ] 矢印アイコンが正しく変化する（▼展開時 / ▶折りたたみ時）

### 3. 異なるブラウザでの確認

- [ ] Chrome
- [ ] Firefox
- [ ] Safari（macOSの場合）

## テクニカルノート

### なぜこの変更で十分か

- **Reactの状態管理**: useStateの初期値を変更するだけで、既存のsetExpanded, トグルロジックは全て動作する
- **CSSクラス**: Tailwind CSSのjustify-start/justify-endを入れ替えるだけで、レイアウトが逆転する
- **型定義**: APIレスポンスやContentBlock型に変更がないため、型安全性は維持される

### 今後の改善案

1. **アニメーション追加**: 展開/折りたたみ時のスムーズなトランジション
2. **設定の永続化**: LocalStorageでデフォルト展開状態をユーザーが選択可能に
3. **テストの追加**: React Testing Libraryでコンポーネントテストを作成

## コミット

変更が確認できたら、以下のメッセージでコミット:

```bash
git add web/src/components/conversation/MessageItem.tsx \
        web/src/components/conversation/ToolUseBlock.tsx \
        web/src/components/conversation/ToolResultBlock.tsx

git commit -m "feat: 会話履歴のUI表示を改善

- ユーザーメッセージを右寄せ、アシスタントメッセージを左寄せに変更
- ツール呼び出しと結果をデフォルトで展開状態に変更
- トグル機能は維持

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

## 完了条件

- [ ] 3ファイルの変更が完了
- [ ] 開発サーバーで動作確認が完了
- [ ] 全ての検証項目がパス
- [ ] コミットが完了
