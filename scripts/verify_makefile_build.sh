#!/bin/bash

# 验证 Makefile 的 start 和 restart 是否会自动 build

echo "========================================="
echo "🔍 验证 Makefile 自动编译功能"
echo "========================================="
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "📋 检查 Makefile 配置..."
echo ""

# 检查 start 目标
if grep -q "^start: build" Makefile; then
    echo -e "${GREEN}✅ start 目标依赖于 build${NC}"
    echo "   这意味着 'make start' 会先自动编译"
else
    echo -e "${RED}❌ start 目标没有依赖于 build${NC}"
    exit 1
fi

echo ""

# 检查 restart 目标
if grep -q "^restart: build" Makefile; then
    echo -e "${GREEN}✅ restart 目标依赖于 build${NC}"
    echo "   这意味着 'make restart' 会先自动编译"
else
    echo -e "${RED}❌ restart 目标没有依赖于 build${NC}"
    exit 1
fi

echo ""
echo "========================================="
echo "📊 依赖关系分析"
echo "========================================="
echo ""

echo "当执行 'make start' 时："
echo "  1. Make 检测到 start 依赖于 build"
echo "  2. Make 先执行 build 目标"
echo "  3. Make 再执行 start 的命令（dev.sh start）"
echo ""

echo "当执行 'make restart' 时："
echo "  1. Make 检测到 restart 依赖于 build"
echo "  2. Make 先执行 build 目标"
echo "  3. Make 再执行 restart 的命令（dev.sh restart）"
echo ""

echo "========================================="
echo "✅ 验证通过！"
echo "========================================="
echo ""
echo "现在可以使用以下命令："
echo ""
echo -e "  ${GREEN}make start${NC}    # 启动服务（自动编译最新版本）"
echo -e "  ${GREEN}make restart${NC}  # 重启服务（自动编译最新版本）"
echo -e "  ${GREEN}make stop${NC}     # 停止服务（不编译）"
echo ""
echo "提示：如果只编译不启动，使用："
echo -e "  ${YELLOW}make build${NC}"
echo ""
