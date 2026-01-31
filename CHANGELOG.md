# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

---

## [0.4.3] - 2026-01-31

### Fixed
- グラフの時系列方向を完全修正（Issue #21）
  - 全ページ（Projects、ProjectDetail、GroupDetail）でソート順を昇順に統一
  - 不正な日付（0001-01-01）をグラフから除外
  - テストケースも正しい順序に修正

### Changed
- 不要なドキュメントファイルとプランファイルを削除
- READMEの更新

---

## [0.4.2] - 2026-01-29

### Fixed
- ProjectsPageのグラフの時系列方向を修正（Issue #19）
  - 左が古い日付、右が新しい日付の正しい順序に修正

### Added
- ProjectsPageのテストスイート追加（全50テスト）

---

## [0.4.1] - 2026-01-29

### Fixed
- 日付ドリルダウンの無限リクエスト問題を修正（Issue #16）
- ProjectsPageでも日付ドリルダウンの無限リクエスト問題を修正
- settings.jsonの危険なプロセス管理コマンドを削除

---

## [0.4.0] - 2026-01-29

### Added
- プロジェクト表示名の改善 - エンコード済みフォルダ名の代わりにわかりやすいフォルダ名を表示
- セッションのcwdからプロジェクト表示名を自動取得
- グループ表示名をgit_rootから自動取得

### Changed
- displayName生成ロジックを共通化してエラーハンドリングを改善

### Fixed
- 不安定なタイミング依存テストを削除し、テストの信頼性を向上

---

## [0.3.1] - 2026-01-29

### Fixed
- 個人情報保護対策 - パブリック公開前の個人情報を一般化
- 初回スキャン時のlastScanTimeを記録してポーリング時のスキップ判定を機能させる
- セッションの不要な再同期を防止してログ出力を削減
- ファイルウォッチャーのシャットダウンハング問題を解決し、ログ表示を改善
- 不要なセッション更新とシャットダウン遅延を改善
- HTTPサーバーのグレースフルシャットダウンを実装
- SQLiteのDSN設定を改善してデータベース破損エラーを解決
- 初回同期完了後にファイルウォッチャーを起動
- ログ過剰出力を削減 - fmt.Printf削除とLogger統一
- package-lock.jsonを更新してGoReleaserビルドエラーを修正

### Added
- デバッグAPIにスキャンマネージャーの進捗情報を追加

### Changed
- セッション詳細から日付フィルタを削除、セッション一覧に日付範囲フィルタを追加

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
