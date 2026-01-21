#!/bin/bash

# 日程智能体测试脚本
# 用于快速验证 Schedule Agent 功能

set -e

# 加载 .env 文件
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
elif [ -f ../.env ]; then
    export $(grep -v '^#' ../.env | xargs)
fi

echo "========================================="
echo "  日程智能体 - 快速测试脚本"
echo "========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 检查必要的环境变量
echo -e "${YELLOW}检查环境配置...${NC}"

if [ -z "$MEMOS_AI_ENABLED" ]; then
    echo -e "${RED}错误: MEMOS_AI_ENABLED 未设置${NC}"
    echo "请在 .env 文件中设置: MEMOS_AI_ENABLED=true"
    exit 1
fi

if [ -z "$MEMOS_AI_LLM_PROVIDER" ]; then
    echo -e "${RED}错误: MEMOS_AI_LLM_PROVIDER 未设置${NC}"
    echo "请在 .env 文件中配置 LLM provider"
    exit 1
fi

echo -e "${GREEN}✓ 环境配置检查通过${NC}"
echo ""

# 检查服务是否运行
echo -e "${YELLOW}检查服务状态...${NC}"

if ! curl -s http://localhost:28081 > /dev/null; then
    echo -e "${RED}错误: 后端服务未运行${NC}"
    echo "请先启动服务: make start"
    exit 1
fi

echo -e "${GREEN}✓ 后端服务正在运行${NC}"
echo ""

# 如果没有 token，提示用户
if [ -z "$MEMOS_TEST_TOKEN" ]; then
    echo -e "${YELLOW}=========================================${NC}"
    echo -e "${YELLOW}需要认证 Token${NC}"
    echo -e "${YELLOW}=========================================${NC}"
    echo ""
    echo "请按以下步骤获取 token："
    echo ""
    echo "1. 登录或注册账户："
    echo "   curl -X POST http://localhost:28081/api/v1/auth/signin \\"
    echo "     -H 'Content-Type: application/json' \\"
    echo "     -d '{\"username\":\"your_username\",\"password\":\"your_password\"}'"
    echo ""
    echo "2. 从响应中复制 access_token"
    echo ""
    echo "3. 设置环境变量："
    echo "   export MEMOS_TEST_TOKEN=your_token_here"
    echo ""
    echo "或者直接在下面输入 token："
    read -p "请输入 token: " MEMOS_TEST_TOKEN
    echo ""
fi

# 测试函数
test_schedule_query() {
    echo -e "${YELLOW}测试 1: 查询日程${NC}"
    echo "输入: 查看明天的安排"
    echo ""

    response=$(curl -s -X POST "http://localhost:28081/api/v1/ai/chat" \
        -H "Authorization: Bearer $MEMOS_TEST_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{
            "message": "查看明天有什么安排",
            "user_timezone": "Asia/Shanghai"
        }')

    echo "响应:"
    echo "$response" | jq -r '.content' 2>/dev/null || echo "$response"
    echo ""
    echo -e "${GREEN}✓ 查询测试完成${NC}"
    echo ""
}

test_schedule_create() {
    echo -e "${YELLOW}测试 2: 创建日程${NC}"
    echo "输入: 后天上午10点开个产品会"
    echo ""

    response=$(curl -s -X POST "http://localhost:28081/api/v1/ai/chat" \
        -H "Authorization: Bearer $MEMOS_TEST_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{
            "message": "后天上午10点开个产品讨论会",
            "user_timezone": "Asia/Shanghai"
        }')

    echo "响应:"
    echo "$response" | jq -r '.content' 2>/dev/null || echo "$response"
    echo ""
    echo -e "${GREEN}✓ 创建测试完成${NC}"
    echo ""
}

test_schedule_weekly() {
    echo -e "${YELLOW}测试 3: 查询本周日程${NC}"
    echo "输入: 本周有哪些日程"
    echo ""

    response=$(curl -s -X POST "http://localhost:28081/api/v1/ai/chat" \
        -H "Authorization: Bearer $MEMOS_TEST_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{
            "message": "本周有哪些日程安排？",
            "user_timezone": "Asia/Shanghai"
        }')

    echo "响应:"
    echo "$response" | jq -r '.content' 2>/dev/null || echo "$response"
    echo ""
    echo -e "${GREEN}✓ 本周查询测试完成${NC}"
    echo ""
}

# 主菜单
show_menu() {
    echo ""
    echo -e "${YELLOW}=========================================${NC}"
    echo -e "${YELLOW}  请选择测试项目${NC}"
    echo -e "${YELLOW}=========================================${NC}"
    echo "1. 查询明天的日程"
    echo "2. 创建新日程（后天上午10点）"
    echo "3. 查询本周日程"
    echo "4. 运行所有测试"
    echo "5. 退出"
    echo ""
    read -p "请输入选项 (1-5): " choice
    echo ""

    case $choice in
        1)
            test_schedule_query
            ;;
        2)
            test_schedule_create
            ;;
        3)
            test_schedule_weekly
            ;;
        4)
            test_schedule_query
            test_schedule_create
            test_schedule_weekly
            echo ""
            echo -e "${GREEN}=========================================${NC}"
            echo -e "${GREEN}  所有测试完成！${NC}"
            echo -e "${GREEN}=========================================${NC}"
            ;;
        5)
            echo "退出测试"
            exit 0
            ;;
        *)
            echo -e "${RED}无效选项，请重新选择${NC}"
            ;;
    esac
}

# 主循环
while true; do
    show_menu
done
