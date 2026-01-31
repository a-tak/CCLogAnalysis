#!/bin/bash
set -e

# リリース作成スクリプト
# 使用方法: ./create-release.sh <version> <release-notes>
#
# 引数:
#   version: リリースバージョン（例: 0.4.3）
#   release-notes: リリースノート（複数行可）

VERSION="$1"
RELEASE_NOTES="$2"

# 引数チェック
if [ -z "$VERSION" ]; then
    echo "❌ エラー: バージョン番号が指定されていません"
    echo "使用方法: $0 <version> <release-notes>"
    exit 1
fi

if [ -z "$RELEASE_NOTES" ]; then
    echo "❌ エラー: リリースノートが指定されていません"
    echo "使用方法: $0 <version> <release-notes>"
    exit 1
fi

echo "📦 リリース作成を開始します"
echo "   バージョン: v$VERSION"
echo ""

# タグが既に存在するかチェック
if git rev-parse "v$VERSION" >/dev/null 2>&1; then
    echo "❌ エラー: タグ v$VERSION は既に存在します"
    exit 1
fi

# 1. CHANGELOGをコミット
echo "📝 CHANGELOGをコミット中..."
git add CHANGELOG.md
git commit -m "$(cat <<EOF
chore: CHANGELOG更新 - v${VERSION}リリース準備

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"
echo "✅ CHANGELOGコミット完了"
echo ""

# 2. タグ作成
echo "🏷️  タグを作成中..."
git tag "v$VERSION"
echo "✅ タグ v$VERSION を作成しました"
echo ""

# 3. Push
echo "⬆️  リモートにpush中..."
git push origin main
git push origin "v$VERSION"
echo "✅ Push完了"
echo ""

# 4. GitHubリリース作成
echo "🚀 GitHubリリースを作成中..."
RELEASE_URL=$(gh release create "v$VERSION" \
    --title "v$VERSION" \
    --notes "$RELEASE_NOTES")
echo "✅ GitHubリリース作成完了"
echo ""

echo "==========================================
✅ リリース v$VERSION を作成しました
=========================================="
echo ""
echo "🔗 リリースURL: $RELEASE_URL"
