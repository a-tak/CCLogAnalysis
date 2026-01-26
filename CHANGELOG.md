# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

---

## [0.2.0] - 2026-01-26

### Added
- セッション更新機能（セッション中の会話をリアルタイム反映）
- ファイル監視機能（デフォルトで有効化）
- Git Worktree対応（複数ブランチでの同時作業をサポート）
- 増分同期での新規プロジェクトGit Root検出機能
- セッション詳細画面のページング機能
- トークンダッシュボードのドリルダウン機能（全体サマリーグラフ、パンくずナビゲーション）
- グループ別のグラフ表示機能（日別/週別/月別の時系列統計）
- フロントエンド自動更新機能（15秒ごと）
- issueスキル追加
- ready-for-prスキル改善

### Fixed
- ファイル監視のリアルタイム動作の安定化
- セッション詳細表示のパフォーマンス改善
- 初期ローディング実装とuseCallback依存配列の修正

### Improved
- 初期スキャンの非同期化によるパフォーマンス向上
- スキャン最適化実装
- 初期化進捗表示の改善

---

## リリース方法

新しいバージョンをリリースするには：

```bash
# バージョンタグを作成（例: v0.1.0）
git tag -a v0.1.0 -m "First binary release"
git push origin v0.1.0
```

GitHub Actionsが自動的に以下を実行します：
1. Windows/Mac向けバイナリをビルド
2. GitHub Releasesにアップロード
3. チェックサムファイルを生成
