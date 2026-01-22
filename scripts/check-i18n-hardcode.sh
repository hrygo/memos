#!/bin/bash
# 硬编码文本检查脚本 - 检测前端代码中的硬编码文本

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
WEB_SRC="$PROJECT_ROOT/web/src"

echo "🔍 Checking for hardcoded text in frontend code..."
echo ""

# 允许的常见单词（小写）
ALLOWED_WORDS=(
    "and" "or" "the" "a" "an" "of" "to" "in" "is" "for" "with" "as" "by"
    "at" "on" "off" "up" "down" "left" "right" "in" "out"
    "from" "over" "under" "via" "use" "new" "old" "more" "less"
    "true" "false" "null" "undefined" "NaN" "Infinity"
    "src" "href" "alt" "id" "class" "type" "name" "value" "placeholder"
    "localhost" "http" "https" "www" "com" "org" "io" "app" "api"
    "div" "span" "button" "input" "form" "link" "nav" "header" "footer"
    "props" "state" "data" "ref" "key" "children" "className"
    "const" "let" "var" "function" "return" "import" "export" "default"
    "react" "typescript" "javascript" "css" "html" "json" "xml" "svg"
    "lc" "pi" "bi" "lu" "lucide" "react" "next" "tailwind" "vscode"
    "x" "y" "z" "w" "h" "mx" "my" "mt" "mb" "ml" "mr" "px" "py" "pt" "pb" "pl" "pr"
    "xs" "sm" "md" "lg" "xl" "2xl" "flex" "grid" "block" "inline"
    "slate" "zinc" "neutral" "stone" "red" "orange" "amber" "yellow"
    "lime" "green" "emerald" "teal" "cyan" "sky" "blue" "indigo" "violet"
    "purple" "fuchsia" "pink" "rose"
    "dark" "light" "hover" "focus" "active" "disabled"
    # 日期和数字相关
    "mon" "tue" "wed" "thu" "fri" "sat" "sun"
    "jan" "feb" "mar" "apr" "may" "jun" "jul" "aug" "sep" "oct" "nov" "dec"
    "yyyy" "mm" "dd" "hh" "ii" "ss"
)

# 构建允许单词的 grep 模式
ALLOWED_PATTERN=$(IFS="|"; echo "${ALLOWED_WORDS[*]}")

# 查找可疑的硬编码文本
# 排除：注释、console.log、已有的 t() 调用、纯标签
find "$WEB_SRC" -name "*.tsx" -o -name "*.ts" | while read -r file; do
    # 检查 JSX 中的硬编码文本 (>2 个单词，包含字母)
    grep -n '>' "$file" | \
        grep -E '>[A-Z][a-zA-Z]{2,}' | \
        grep -vE '(t\(|useTranslate|//|/\*|TODO|FIXME|NOTE|XXX)' | \
        grep -vE "($ALLOWED_PATTERN)" | \
        head -5
done | head -20

echo ""
echo "💡 Tips:"
echo "  - Use t('your.key') for all user-facing text"
echo "  - Add the key to both en.json and zh-Hans.json"
echo "  - Run 'make check-i18n' to verify key synchronization"
