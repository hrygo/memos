#!/bin/bash
# i18n 检查脚本 - 验证 en.json 和 zh-Hans.json 的 key 同步

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
EN_LOCALE="$PROJECT_ROOT/web/src/locales/en.json"
ZH_LOCALE="$PROJECT_ROOT/web/src/locales/zh-Hans.json"

echo "🔍 Checking i18n keys synchronization..."
echo ""

# 检查文件是否存在
if [ ! -f "$EN_LOCALE" ]; then
    echo "❌ Error: $EN_LOCALE not found"
    exit 1
fi

if [ ! -f "$ZH_LOCALE" ]; then
    echo "❌ Error: $ZH_LOCALE not found"
    exit 1
fi

# 使用 Node.js 提取 JSON 中的所有 key
extract_keys() {
    node -e "
        const fs = require('fs');
        const data = JSON.parse(fs.readFileSync('$1', 'utf8'));

        function extractKeys(obj, prefix = '') {
            let keys = [];
            for (const key in obj) {
                const fullKey = prefix ? \`\${prefix}.\${key}\` : key;
                if (typeof obj[key] === 'object' && obj[key] !== null && !Array.isArray(obj[key])) {
                    keys = keys.concat(extractKeys(obj[key], fullKey));
                } else {
                    keys.push(fullKey);
                }
            }
            return keys;
        }

        const keys = extractKeys(data);
        keys.forEach(k => console.log(k));
    "
}

# 提取所有 key
EN_KEYS=$(extract_keys "$EN_LOCALE" | sort)
ZH_KEYS=$(extract_keys "$ZH_LOCALE" | sort)

# 只在 en 中的 key
ONLY_IN_EN=$(comm -23 <(echo "$EN_KEYS") <(echo "$ZH_KEYS"))

# 只在 zh 中的 key
ONLY_IN_ZH=$(comm -13 <(echo "$EN_KEYS") <(echo "$ZH_KEYS"))

# 统计
TOTAL_EN=$(echo "$EN_KEYS" | wc -l | tr -d ' ')
TOTAL_ZH=$(echo "$ZH_KEYS" | wc -l | tr -d ' ')

# 输出结果
echo "📊 Statistics:"
echo "  en.json keys:      $TOTAL_EN"
echo "  zh-Hans.json keys: $TOTAL_ZH"
echo ""

# 检查差异
HAS_ERROR=0

if [ -n "$ONLY_IN_EN" ]; then
    echo "❌ Keys only in en.json (missing in zh-Hans.json):"
    echo "$ONLY_IN_EN" | while read -r key; do
        echo "   - $key"
    done
    echo ""
    HAS_ERROR=1
fi

if [ -n "$ONLY_IN_ZH" ]; then
    echo "⚠️  Keys only in zh-Hans.json (missing in en.json):"
    echo "$ONLY_IN_ZH" | while read -r key; do
        echo "   - $key"
    done
    echo ""
    HAS_ERROR=1
fi

if [ $HAS_ERROR -eq 0 ]; then
    echo "✅ All i18n keys are synchronized!"
    exit 0
else
    echo ""
    echo "❌ i18n check failed. Please ensure en.json and zh-Hans.json have matching keys."
    exit 1
fi
