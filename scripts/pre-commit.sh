#!/bin/bash
# Pre-commit hook - 在提交前检查代码质量

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "🔍 Running pre-commit checks..."
echo ""

# 获取暂存的文件
STAGED_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep -E '\.(go|tsx?|json)$' || true)

if [ -z "$STAGED_FILES" ]; then
    echo "No relevant files staged. Skipping checks."
    exit 0
fi

echo "📁 Staged files:"
echo "$STAGED_FILES" | sed 's/^/  - /'
echo ""

# 检查是否有 Go 文件变更
if echo "$STAGED_FILES" | grep -q '\.go$'; then
    echo "🔧 Checking Go files..."
    go build ./... || {
        echo "❌ Go build failed. Aborting commit."
        exit 1
    }
    echo "✅ Go build OK"
    echo ""
fi

# 检查是否有 locale 文件变更
if echo "$STAGED_FILES" | grep -q 'locales/en.json\|locales/zh-Hans.json'; then
    echo "🌍 Checking i18n..."
    "$SCRIPT_DIR/check-i18n.sh" || {
        echo "❌ i18n check failed. Aborting commit."
        echo "   Please ensure en.json and zh-Hans.json have matching keys."
        exit 1
    }
    echo ""
fi

# 检查是否有前端文件变更
if echo "$STAGED_FILES" | grep -qE '\.(tsx?|ts)$'; then
    echo "⚛️  Checking frontend files..."
    cd web
    if ! pnpm lint --no-fix 2>/dev/null; then
        echo "⚠️  Frontend lint has issues. Consider running 'pnpm lint:fix'"
    fi
    cd ..
    echo ""
fi

echo "✅ Pre-commit checks passed!"
exit 0
