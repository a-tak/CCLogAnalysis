# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Windows/Mac向けバイナリパッケージング対応
- GoReleaserによる自動リリース
- GitHub Actionsワークフロー（タグプッシュで自動リリース）

### Changed
- SQLiteドライバーを`modernc.org/sqlite`に変更（CGO依存を削除）
- クロスプラットフォームビルドがシンプルになった

### Improved
- Pure Go実装により、ビルドプロセスが簡素化された
- 依存関係が削減され、保守性が向上した

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
