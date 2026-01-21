#!/bin/bash

# 日程查询诊断工具
# 用于诊断"1月21日有哪些事？"为何返回"暂无日程"

echo "========================================="
echo "🔍 日程查询诊断工具"
echo "========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# ============================================================
# 1. 检查服务状态
# ============================================================
echo "📋 检查服务状态..."
if pgrep -f "memos" > /dev/null; then
    echo -e "${GREEN}✅ Memos 服务正在运行${NC}"
else
    echo -e "${RED}❌ Memos 服务未运行${NC}"
    echo "请先启动服务: make start"
    exit 1
fi

# ============================================================
# 2. 检查数据库连接
# ============================================================
echo ""
echo "📋 检查数据库连接..."
DB_CONTAINER="memos-postgres"
if docker ps | grep -q $DB_CONTAINER; then
    echo -e "${GREEN}✅ 数据库容器正在运行${NC}"

    # 查询1月21日的日程
    echo ""
    echo "📅 查询 2026-01-21 的日程..."
    SCHEDULE_COUNT=$(docker exec $DB_CONTAINER psql -U memos -d memos -t -c \
        "SELECT COUNT(*) FROM schedule
        WHERE start_ts >= extract(epoch from '2026-01-21'::timestamp)
        AND start_ts < extract(epoch from '2026-01-22'::timestamp);" 2>/dev/null | tr -d ' ')

    if [ ! -z "$SCHEDULE_COUNT" ]; then
        if [ "$SCHEDULE_COUNT" -eq "0" ]; then
            echo -e "${YELLOW}⚠️  数据库中没有 2026-01-21 的日程${NC}"
            echo ""
            echo "建议："
            echo "1. 创建一些测试日程"
            echo "2. 或者查询其他日期（如今天、明天）"
        else
            echo -e "${GREEN}✅ 找到 $SCHEDULE_COUNT 条日程${NC}"
            echo ""
            echo "日程详情："
            docker exec $DB_CONTAINER psql -U memos -d memos -c \
                "SELECT id, title,
                to_timestamp(start_ts) as scheduled_time
                FROM schedule
                WHERE start_ts >= extract(epoch from '2026-01-21'::timestamp)
                AND start_ts < extract(epoch from '2026-01-22'::timestamp)
                ORDER BY start_ts;" 2>/dev/null
        fi
    else
        echo -e "${RED}❌ 数据库查询失败${NC}"
    fi
else
    echo -e "${RED}❌ 数据库容器未运行${NC}"
fi

# ============================================================
# 3. 检查代码版本
# ============================================================
echo ""
echo "📋 检查代码版本..."
if grep -q "解析具体日期" server/queryengine/query_router.go 2>/dev/null; then
    echo -e "${GREEN}✅ 代码已包含日期解析功能${NC}"

    # 检查是否重新编译
    BINARY_TIME=$(stat -f "%Sm" -t "%Y-%m-%d %H:%M:%S" memos 2>/dev/null || stat -c "%y" memos 2>/dev/null | cut -d'.' -f1)
    SOURCE_TIME=$(stat -f "%Sm" -t "%Y-%m-%d %H:%M:%S" server/queryengine/query_router.go 2>/dev/null || stat -c "%y" server/queryengine/query_router.go 2>/dev/null | cut -d'.' -f1)

    echo "   源码修改时间: $SOURCE_TIME"
    echo "   二进制编译时间: $BINARY_TIME"

    if [ "$BINARY_TIME" \< "$SOURCE_TIME" ]; then
        echo -e "${RED}❌ 二进制文件过期，需要重新编译！${NC}"
        echo ""
        echo "请执行："
        echo "  make stop"
        echo "  go build ./cmd/memos/..."
        echo "  make start"
    else
        echo -e "${GREEN}✅ 二进制文件是最新的${NC}"
    fi
else
    echo -e "${RED}❌ 代码未更新，请拉取最新代码${NC}"
fi

# ============================================================
# 4. 检查日志中的日期解析
# ============================================================
echo ""
echo "📋 检查日志中的日期解析..."
echo "提示：请发送查询'1月21日有哪些事？'，然后查看以下日志："
echo ""
echo "  make logs backend | grep -E 'QueryRouting|TimeRange|1月21'"

# ============================================================
# 总结
# ============================================================
echo ""
echo "========================================="
echo "📊 诊断总结"
echo "========================================="
echo ""
echo "可能的解决方案："
echo ""
echo "1. ${YELLOW}重新编译和部署${NC}（最常见）"
echo "   make stop"
echo "   go build ./cmd/memos/..."
echo "   make start"
echo ""
echo "2. ${YELLOW}验证数据库中确实有日程${NC}"
echo "   docker exec -it memos-postgres psql -U memos -d memos"
echo "   SELECT * FROM schedule WHERE start_ts >= ...;"
echo ""
echo "3. ${YELLOW}查看日志确认日期解析生效${NC}"
echo "   make logs backend | grep QueryRouting"
echo ""
echo "========================================="
