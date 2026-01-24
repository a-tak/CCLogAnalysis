---
paths:
  - "web/**/*.{ts,tsx,jsx}"
---

# React Development Rules

## 技術スタック

- React + TypeScript
- Tailwind CSS + shadcn/ui
- Recharts（グラフ表示）
- Vite（ビルドツール）

## コンポーネント設計

- 関数コンポーネントとHooksを使用
- 全てのPropsに TypeScript型定義を含める
- Atomic Design パターンに従う
  - Atoms: ボタン、入力フィールドなど
  - Molecules: フォームグループ、カードなど
  - Organisms: ヘッダー、セッションリストなど
  - Templates: ページレイアウト
  - Pages: 実際のページ

## スタイリング

- Tailwind CSSのユーティリティクラスを使用
- カスタムコンポーネントはshadcn/uiを優先
- インラインスタイルは避ける
- レスポンシブデザインを考慮

## 状態管理

- ローカル状態: `useState`, `useReducer`
- グローバル状態: React Context（必要に応じて）
- サーバー状態: TanStack Query（React Query）導入予定

## API通信

- `fetch` APIを使用
- エラーハンドリングを適切に行う
- ローディング状態を表示
- 型安全なAPI通信（TypeScript型定義）

## テスト

- React Testing Libraryを使用（導入予定）
- ユーザーの視点でテストを書く
- 実装の詳細ではなく振る舞いをテスト

## 依存関係

- 外部パッケージは最小限に
- 新しいパッケージ追加時は理由を明確に
