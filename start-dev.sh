#!/bin/bash
# 启动 Memos 开发服务器（使用 PostgreSQL）

# 加载 .env 文件
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
    echo "已加载 .env 配置"
    echo "  - DRIVER: $MEMOS_DRIVER"
    echo "  - DSN: ${MEMOS_DSN:0:50}..."
else
    echo "警告: .env 文件不存在，使用默认配置"
    export MEMOS_DRIVER=postgres
    export MEMOS_DSN="postgres://memos:memos@localhost:25432/memos?sslmode=disable"
fi

# 启动服务器
exec go run ./cmd/memos --mode dev --port 28081
