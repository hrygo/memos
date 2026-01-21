#!/bin/bash

# 测试日程检索功能
# 验证"1月21日有哪些事？"查询能否正确检索到日程

echo "========================================="
echo "🔍 测试日程检索功能"
echo "========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "📋 步骤 1: 查询数据库中的日程..."
echo ""

# 查询 2026-01-21 的日程
SCHEDULES=$(docker exec memos-postgres-dev psql -U memos -d memos -t -c \
  "SELECT id, title, start_ts
   FROM schedule
   WHERE start_ts >= extract(epoch from '2026-01-21'::timestamp)
   AND start_ts < extract(epoch from '2026-01-22'::timestamp)
   ORDER BY start_ts;" 2>/dev/null)

if [ -n "$SCHEDULES" ]; then
    echo -e "${GREEN}✅ 找到以下日程：${NC}"
    echo "$SCHEDULES" | while read -r line; do
        echo "  - $line"
    done
else
    echo -e "${RED}❌ 数据库中没有 2026-01-21 的日程${NC}"
    echo ""
    echo "建议：先创建一些测试日程"
    exit 1
fi

echo ""
echo "📋 步骤 2: 计算时间戳..."
echo ""

# 2026-01-21 00:00:00 UTC 的时间戳
START_TIMESTAMP=$(date -j -f "%Y-%m-%d %H:%M:%S" "2026-01-21 00:00:00" +%s 2>/dev/null || \
               date -d "2026-01-21 00:00:00" +%s 2>/dev/null)

# 2026-01-22 00:00:00 UTC 的时间戳
END_TIMESTAMP=$(date -j -f "%Y-%m-%d %H:%M:%S" "2026-01-22 00:00:00" +%s 2>/dev/null || \
             date -d "2026-01-22 00:00:00" +%s 2>/dev/null)

echo -e "${BLUE}时间范围：${NC}"
echo "  开始: 2026-01-21 00:00:00 UTC (时间戳: $START_TIMESTAMP)"
echo "  结束: 2026-01-22 00:00:00 UTC (时间戳: $END_TIMESTAMP)"
echo ""

echo "📋 步骤 3: 验证日程是否在时间范围内..."
echo ""

MATCHED=0
echo "$SCHEDULES" | while read -r id title start_ts; do
    # 去除空白
    id=$(echo "$id" | xargs)
    title=$(echo "$title" | xargs)
    start_ts=$(echo "$start_ts" | xargs)

    if [ -n "$start_ts" ]; then
        # 检查是否在时间范围内
        if [ "$start_ts" -ge "$START_TIMESTAMP" ] && [ "$start_ts" -lt "$END_TIMESTAMP" ]; then
            echo -e "  ${GREEN}✅${NC} [$id] $title (时间戳: $start_ts)"
            MATCHED=$((MATCHED + 1))
        else
            echo -e "  ${RED}❌${NC} [$id] $title (时间戳: $start_ts, 超出范围)"
        fi
    fi
done

echo ""
echo "📋 步骤 4: 检查后端日志中的路由决策..."
echo ""

echo "请在另一个终端中运行以下命令查看实时日志："
echo ""
echo -e "${YELLOW}make logs backend${NC}"
echo ""
echo "然后在 AI Chat 中查询：${YELLOW}\"1月21日有哪些事？\"${NC}"
echo ""
echo "期望看到的日志："
echo "  [QueryRouting] Strategy: hybrid_with_time_filter (或 schedule_bm25_only)"
echo "  [QueryRouting] TimeRange: 1月21日 (2026-01-21 00:00 to 2026-01-22 00:00)"
echo "  [Retrieval] Found X results"
echo ""

echo "📋 步骤 5: 如果日志中没有路由决策信息..."
echo ""
echo "可能的原因："
echo "  1. Connect RPC 版本没有使用 QueryRouter"
echo "  2. 路由决策被禁用或配置错误"
echo "  3. 日志级别不够，没有打印路由决策"
echo ""
echo "建议："
echo -e "  ${YELLOW}1. 检查后端是否使用 Connect RPC${NC}"
echo -e "  ${YELLOW}2. 确认 QueryRouter 已正确初始化${NC}"
echo -e "  ${YELLOW}3. 查看完整的后端启动日志${NC}"
echo ""

echo "========================================="
echo "📊 诊断完成"
echo "========================================="
echo ""
echo "下一步操作："
echo "  1. 在前端查询 '1月21日有哪些事？'"
echo "  2. 查看后端日志: make logs backend"
echo "  3. 如果仍然没有日程，请提供日志输出"
echo ""
