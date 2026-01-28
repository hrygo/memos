#!/bin/bash
# DivineSense 数据库迁移脚本
# 从 memos 数据库迁移到 divinesense 数据库

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 默认配置
OLD_DB_NAME="${OLD_DB_NAME:-memos}"
NEW_DB_NAME="${NEW_DB_NAME:-divinesense}"
DB_USER="${DB_USER:-memos}"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
BACKUP_DIR="${BACKUP_DIR:-/tmp/divinesense_migration}"

# 函数：打印信息
info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# 函数：检查命令是否存在
check_command() {
    if ! command -v $1 &> /dev/null; then
        error "$1 命令不存在，请先安装"
    fi
}

# 函数：创建备份目录
create_backup_dir() {
    mkdir -p "$BACKUP_DIR"
    info "备份目录: $BACKUP_DIR"
}

# 函数：备份现有数据库
backup_database() {
    info "正在备份数据库 $OLD_DB_NAME..."

    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="$BACKUP_DIR/${OLD_DB_NAME}_full_${timestamp}.sql"
    local schema_file="$BACKUP_DIR/${OLD_DB_NAME}_schema_${timestamp}.sql"
    local data_file="$BACKUP_DIR/${OLD_DB_NAME}_data_${timestamp}.sql"

    # 备份完整数据库
    pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" \
        --format=plain --no-owner --no-acl \
        "$OLD_DB_NAME" > "$backup_file" || error "备份失败"

    # 只备份 schema
    pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" \
        --schema-only --no-owner --no-acl \
        "$OLD_DB_NAME" > "$schema_file" || error "schema 备份失败"

    # 只备份数据
    pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" \
        --data-only --no-owner --no-acl \
        "$OLD_DB_NAME" > "$data_file" || error "数据备份失败"

    info "备份完成:"
    info "  - 完整备份: $backup_file"
    info "  - Schema:   $schema_file"
    info "  - 数据:     $data_file"
}

# 函数：创建新数据库
create_new_database() {
    info "正在创建新数据库 $NEW_DB_NAME..."

    # 删除已存在的数据库
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres \
        -c "DROP DATABASE IF EXISTS $NEW_DB_NAME;" 2>/dev/null || true

    # 创建新数据库
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres \
        -c "CREATE DATABASE $NEW_DB_NAME OWNER $DB_USER;" || error "创建数据库失败"

    # 安装 pgvector 扩展
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$NEW_DB_NAME" \
        -c "CREATE EXTENSION IF NOT EXISTS vector;" || warn "vector 扩展安装失败"

    info "新数据库创建完成"
}

# 函数：迁移数据
migrate_data() {
    info "正在迁移数据..."

    # 恢复数据到新数据库
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$NEW_DB_NAME" \
        < "$BACKUP_DIR/${OLD_DB_NAME}_full_$(ls -t "$BACKUP_DIR"/${OLD_DB_NAME}_full_*.sql | head -1 | xargs -n1 basename | grep -oP '\d{8}_\d{6}')" \
        || error "数据迁移失败"

    info "数据迁移完成"
}

# 函数：验证迁移
verify_migration() {
    info "正在验证迁移..."

    # 检查表数量
    local table_count=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$NEW_DB_NAME" \
        -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE';")

    info "新数据库表数量: $table_count"

    if [ "$table_count" -lt 10 ]; then
        error "迁移验证失败: 表数量太少"
    fi
}

# 函数：更新环境配置
update_env_config() {
    info "请更新您的 .env 文件:"
    echo ""
    echo "DIVINESENSE_DSN=\"host=$DB_HOST port=$DB_PORT user=$DB_USER password=<your_password> dbname=$NEW_DB_NAME sslmode=disable\""
    echo ""
}

# 主函数
main() {
    info "开始 DivineSense 数据库迁移..."
    echo ""

    # 检查必要的命令
    check_command pg_dump
    check_command psql

    # 提示用户输入密码（如果需要）
    if [ -z "$PGPASSWORD" ]; then
        warn "请设置 PGPASSWORD 环境变量或使用 ~/.pgpass 文件"
    fi

    # 执行迁移步骤
    create_backup_dir
    backup_database
    create_new_database
    migrate_data
    verify_migration

    echo ""
    info "=== 迁移完成 ==="
    update_env_config
}

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        --old-db)
            OLD_DB_NAME="$2"
            shift 2
            ;;
        --new-db)
            NEW_DB_NAME="$2"
            shift 2
            ;;
        --user)
            DB_USER="$2"
            shift 2
            ;;
        --host)
            DB_HOST="$2"
            shift 2
            ;;
        --port)
            DB_PORT="$2"
            shift 2
            ;;
        --backup-dir)
            BACKUP_DIR="$2"
            shift 2
            ;;
        --help)
            echo "用法: $0 [选项]"
            echo ""
            echo "选项:"
            echo "  --old-db NAME      旧数据库名称 (默认: memos)"
            echo "  --new-db NAME      新数据库名称 (默认: divinesense)"
            echo "  --user NAME        数据库用户 (默认: memos)"
            echo "  --host HOST        数据库主机 (默认: localhost)"
            echo "  --port PORT        数据库端口 (默认: 5432)"
            echo "  --backup-dir DIR   备份目录 (默认: /tmp/divinesense_migration)"
            echo "  --help             显示此帮助信息"
            exit 0
            ;;
        *)
            error "未知选项: $1"
            ;;
    esac
done

# 运行主函数
main
