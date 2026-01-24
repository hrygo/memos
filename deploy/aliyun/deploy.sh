#!/bin/bash
# =============================================================================
# Memos 单机部署脚本 (2C2G 环境)
# =============================================================================
#
# 使用方式:
#   ./deploy.sh [命令]
#
# 命令:
#   build   - 构建生产镜像
#   deploy  - 部署到生产环境 (首次安装)
#   pull    - 拉取预构建镜像 (替代 build)
#   upgrade - 执行数据库升级 (增量迁移)
#   restart - 重启服务
#   stop    - 停止服务
#   logs    - 查看日志
#   status  - 查看状态
#   backup  - 备份数据库
#   restore - 恢复数据库
#   version - 查看当前版本
#   setup   - 配置 Docker 镜像加速 (阿里云/国内)
#
# 环境变量 (.env.prod):
#   USER_IMAGE       - 使用预构建镜像 (跳过 build)
#   POSTGRES_IMAGE   - 自定义 PG 镜像 (默认: pgvector/pgvector:pg16)
#
# =============================================================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 配置
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
COMPOSE_FILE="${PROJECT_ROOT}/docker/compose/prod.yml"
ENV_FILE="${SCRIPT_DIR}/.env.prod"
IMAGE_NAME="memos"
IMAGE_TAG="${IMAGE_NAME}:latest"
BACKUP_DIR="${SCRIPT_DIR}/backups"
MIGRATIONS_DIR="${PROJECT_ROOT}/store/migration/postgres"
VERSION_FILE="${MIGRATIONS_DIR}/VERSION"

# 日志函数
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 检查环境
check_env() {
    if [ ! -f "${ENV_FILE}" ]; then
        log_error "环境配置文件不存在: ${ENV_FILE}"
        log_info "请先创建配置文件: cp .env.prod.example .env.prod"
        exit 1
    fi

    # 检查密码是否已修改
    if grep -q "your_secure_password_here\|your-server-ip" "${ENV_FILE}"; then
        log_error "请先修改 .env.prod 中的默认配置"
        exit 1
    fi

    log_success "环境检查通过"
}

# 检查 Docker
check_docker() {
    if ! command -v docker &>/dev/null; then
        log_error "Docker 未安装"
        log_info "安装命令: curl -fsSL https://get.docker.com | sh"
        exit 1
    fi

    if ! docker info &>/dev/null; then
        log_error "Docker 未运行，请先启动 Docker"
        exit 1
    fi

    if ! command -v docker-compose &>/dev/null && ! docker compose version &>/dev/null; then
        log_error "Docker Compose 未安装"
        exit 1
    fi

    log_success "Docker 检查通过"
}

# 配置 Docker 镜像加速 (阿里云/国内)
setup_docker_mirror() {
    log_info "配置 Docker 镜像加速..."

    local docker_config_dir="$HOME/.docker"
    local daemon_config="$docker_config_dir/daemon.json"

    mkdir -p "$docker_config_dir"

    # 备份现有配置
    if [ -f "$daemon_config" ]; then
        cp "$daemon_config" "${daemon_config}.backup.$(date +%Y%m%d%H%M%S)"
        log_info "已备份现有配置"
    fi

    # 写入镜像配置
    cat > "$daemon_config" << 'EOF'
{
  "registry-mirrors": [
    "https://docker.m.daocloud.io",
    "https://dockerproxy.com",
    "https://docker.mirrors.ustc.edu.cn",
    "https://docker.nju.edu.cn"
  ],
  "max-concurrent-downloads": 10,
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "100m",
    "max-file": "3"
  }
}
EOF

    log_success "镜像配置已写入: $daemon_config"
    log_warn "请重启 Docker 服务使配置生效:"
    log_warn "  sudo systemctl restart docker"
    log_warn "  或"
    log_warn "  sudo service docker restart"
}

# 获取 docker-compose 命令
get_compose_cmd() {
    if docker compose version &>/dev/null; then
        echo "docker compose"
    else
        echo "docker-compose"
    fi
}

# 加载环境变量
load_env() {
    # 读取数据库配置 (安全的变量解析，避免 eval 注入)
    # 使用 tr 和 xargs 清理控制字符和前后空格
    POSTGRES_DB=$(grep "^POSTGRES_DB=" "${ENV_FILE}" 2>/dev/null | cut -d'=' -f2- | tr -d '\n\r' | xargs || echo "memos")
    POSTGRES_USER=$(grep "^POSTGRES_USER=" "${ENV_FILE}" 2>/dev/null | cut -d'=' -f2- | tr -d '\n\r' | xargs || echo "memos")
    POSTGRES_PASSWORD=$(grep "^POSTGRES_PASSWORD=" "${ENV_FILE}" 2>/dev/null | cut -d'=' -f2- | tr -d '\n\r' | xargs)
    # 读取自定义镜像配置
    USER_IMAGE=$(grep "^USER_IMAGE=" "${ENV_FILE}" 2>/dev/null | cut -d'=' -f2- | tr -d '\n\r' | xargs)
    POSTGRES_IMAGE=$(grep "^POSTGRES_IMAGE=" "${ENV_FILE}" 2>/dev/null | cut -d'=' -f2- | tr -d '\n\r' | xargs || echo "pgvector/pgvector:pg16")

    # 默认值
    POSTGRES_DB=${POSTGRES_DB:-memos}
    POSTGRES_USER=${POSTGRES_USER:-memos}
    POSTGRES_PASSWORD=${POSTGRES_PASSWORD:-}
    USER_IMAGE=${USER_IMAGE:-}
    POSTGRES_IMAGE=${POSTGRES_IMAGE:-pgvector/pgvector:pg16}
}

# 获取当前数据库版本
get_db_version() {
    local compose=$(get_compose_cmd)
    load_env

    $compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" exec -T postgres psql -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" -t -c \
        "SELECT value FROM system_setting WHERE name = 'schema_version';" 2>/dev/null | xargs || echo "none"
}

# 获取代码版本
get_code_version() {
    if [ -f "${VERSION_FILE}" ]; then
        cat "${VERSION_FILE}"
    else
        # 从代码中获取版本
        grep -oP '(?<=Version = ")[^"]+' "${PROJECT_ROOT}/internal/version/version.go" 2>/dev/null | xargs || echo "unknown"
    fi
}

# 构建镜像
build_image() {
    log_info "开始构建生产镜像..."

    cd "${PROJECT_ROOT}"

    # 检查构建依赖
    if ! command -v go &>/dev/null; then
        log_error "Go 未安装，无法构建镜像"
        log_info "替代方案: 使用预构建镜像"
        log_info "  在 .env.prod 中设置: USER_IMAGE=ghcr.io/usememos/memos:latest"
        exit 1
    fi

    if ! command -v pnpm &>/dev/null && ! command -v npm &>/dev/null; then
        log_error "pnpm/npm 未安装，无法构建前端"
        log_info "安装命令: npm install -g pnpm"
        exit 1
    fi

    # 检查前端
    if [ ! -d "web/node_modules" ]; then
        log_info "安装前端依赖..."
        cd web && pnpm install && cd ..
    fi

    # 构建前端
    log_info "构建前端资源..."
    cd web
    pnpm release
    cd ..

    # 构建 Docker 镜像
    log_info "构建 Docker 镜像..."
    docker build -t ${IMAGE_TAG} -f docker/Dockerfile .

    log_success "镜像构建完成: ${IMAGE_TAG}"
}

# 拉取预构建镜像
pull_image() {
    load_env
    local target_image="${USER_IMAGE:-ghcr.io/usememos/memos:latest}"

    log_info "拉取预构建镜像: ${target_image}"

    if docker pull "${target_image}"; then
        # 标记为本地镜像名
        docker tag "${target_image}" "${IMAGE_TAG}"
        log_success "镜像拉取完成: ${IMAGE_TAG}"
    else
        log_error "镜像拉取失败"
        log_info "请检查网络或配置 Docker 镜像加速: $0 setup"
        exit 1
    fi
}

# 部署服务
deploy() {
    log_info "=========================================="
    log_info "Memos 单机部署 (2C2G)"
    log_info "=========================================="

    check_env
    check_docker

    local compose=$(get_compose_cmd)

    # 确保镜像是最新的
    log_info "检查镜像..."
    load_env
    if ! docker image inspect ${IMAGE_TAG} &>/dev/null; then
        if [ -n "${USER_IMAGE}" ]; then
            log_warn "使用预构建镜像: ${USER_IMAGE}"
            pull_image
        else
            log_warn "镜像不存在，开始构建..."
            log_info "提示: 可使用预构建镜像跳过构建，在 .env.prod 设置 USER_IMAGE"
            build_image
        fi
    fi

    # 创建备份目录
    mkdir -p "${BACKUP_DIR}"

    # 检查是否已部署
    if docker ps -a --format '{{.Names}}' | grep -q "^memos-postgres$"; then
        log_warn "检测到已存在的服务，请使用 'upgrade' 进行升级"
        log_info "如需重新部署，请先运行: $0 stop"
        exit 1
    fi

    # 启动服务
    log_info "启动服务 (PostgreSQL + Memos)..."
    $compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" up -d

    # 等待服务启动
    log_info "等待服务启动..."
    sleep 10

    # 等待 Memos 就绪
    log_info "等待 Memos 启动..."
    local max_wait=60
    local waited=0
    while [ $waited -lt $max_wait ]; do
        if docker exec memos sh -c "cat < /dev/null > /dev/tcp/127.0.0.1/5230" 2>/dev/null; then
            break
        fi
        sleep 2
        waited=$((waited + 2))
    done

    # 检查服务状态
    if $compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" ps | grep -q "Up"; then
        echo ""
        log_success "=========================================="
        log_success "部署成功！"
        log_success "=========================================="
        echo ""
        log_info "数据库版本: $(get_db_version)"
        log_info "代码版本: $(get_code_version)"
        log_info "访问地址: http://localhost:5230"
        log_info "查看日志: $0 logs"
        log_info "查看状态: $0 status"
        echo ""
    else
        log_error "服务启动失败，请检查日志"
        exit 1
    fi
}

# 升级服务
upgrade() {
    log_info "=========================================="
    log_info "Memos 升级"
    log_info "=========================================="

    check_env
    check_docker

    local compose=$(get_compose_cmd)
    local db_version=$(get_db_version)
    local code_version=$(get_code_version)

    log_info "当前数据库版本: ${db_version}"
    log_info "当前代码版本: ${code_version}"

    if [ "${db_version}" = "none" ]; then
        log_error "数据库未初始化，请先运行: $0 deploy"
        exit 1
    fi

    if [ "${db_version}" = "${code_version}" ]; then
        log_info "数据库已是最新版本，仅更新服务镜像"
        # 仅更新镜像并重启
        build_image
        $compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" up -d
        log_success "服务已更新"
        return 0
    fi

    # 备份当前数据库
    log_warn "升级前自动备份数据库..."
    if ! backup_auto; then
        log_error "备份失败，终止升级"
        exit 1
    fi

    # 检查是否有迁移脚本
    local migration_files=$(ls "${MIGRATIONS_DIR}"/V*.sql 2>/dev/null | sort -V || true)
    if [ -z "${migration_files}" ]; then
        log_info "没有找到迁移脚本，仅更新服务版本"
    else
        log_info "发现迁移脚本，准备执行..."
        for migration_file in ${migration_files}; do
            log_info "执行: $(basename "${migration_file}")"
            if $compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" exec -T postgres psql -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" -f - < "${migration_file}"; then
                log_success "  迁移成功"
            else
                log_error "  迁移失败，请检查日志"
                exit 1
            fi
        done
    fi

    # 更新镜像
    log_info "更新服务镜像..."
    build_image

    # 重启服务
    log_info "重启服务..."
    $compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" up -d

    # 等待服务就绪
    sleep 5

    local new_db_version=$(get_db_version)
    log_success "升级完成！数据库版本: ${new_db_version}"
}

# 重启服务
restart() {
    log_info "重启服务..."
    local compose=$(get_compose_cmd)
    $compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" restart
    log_success "服务已重启"
}

# 停止服务
stop() {
    log_info "停止服务..."
    local compose=$(get_compose_cmd)
    $compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" down
    log_success "服务已停止"
}

# 查看日志
logs() {
    local compose=$(get_compose_cmd)
    local service="$2"

    if [ -n "$service" ]; then
        $compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" logs -f --tail=100 "$service"
    else
        $compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" logs -f --tail=100
    fi
}

# 查看状态
status() {
    local compose=$(get_compose_cmd)
    echo ""
    echo "=== 服务状态 ==="
    $compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" ps
    echo ""
    echo "=== 版本信息 ==="
    echo "数据库版本: $(get_db_version)"
    echo "代码版本: $(get_code_version)"
    echo ""
    echo "=== 资源使用 ==="
    docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}" memos-postgres memos 2>/dev/null || echo "服务未运行"
    echo ""
}

# 备份数据库
backup() {
    local compose=$(get_compose_cmd)
    local backup_file="${BACKUP_DIR}/memos-backup-$(date +%Y%m%d-%H%M%S).sql.gz"

    mkdir -p "${BACKUP_DIR}"

    log_info "备份数据库到: ${backup_file}"

    load_env
    if $compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" exec -T postgres pg_dump -U "${POSTGRES_USER}" "${POSTGRES_DB}" 2>&1 | gzip > "${backup_file}"; then
        # 验证备份文件 (使用 wc -c 兼容所有平台)
        if [ -f "${backup_file}" ] && [ $(wc -c < "${backup_file}" 2>/dev/null || echo 0) -gt 0 ]; then
            log_success "备份完成: ${backup_file}"
            # 显示备份文件大小
            local size=$(du -h "${backup_file}" | cut -f1)
            log_info "备份大小: ${size}"
        else
            log_error "备份文件为空或无效"
            rm -f "${backup_file}"
            exit 1
        fi
    else
        log_error "备份失败"
        exit 1
    fi
}

# 自动备份 (用于升级前)
backup_auto() {
    local compose=$(get_compose_cmd)
    local backup_file="${BACKUP_DIR}/memos-backup-pre-upgrade-$(date +%Y%m%d-%H%M%S).sql.gz"

    mkdir -p "${BACKUP_DIR}"

    load_env
    if $compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" exec -T postgres pg_dump -U "${POSTGRES_USER}" "${POSTGRES_DB}" 2>&1 | gzip > "${backup_file}"; then
        # 验证备份文件 (使用 wc -c 兼容所有平台)
        if [ -f "${backup_file}" ] && [ $(wc -c < "${backup_file}" 2>/dev/null || echo 0) -gt 0 ]; then
            log_success "自动备份完成: ${backup_file}"
            return 0
        fi
    fi
    log_error "自动备份失败"
    return 1
}

# 恢复数据库
restore() {
    local compose=$(get_compose_cmd)
    local backup_file="$2"

    if [ -z "${backup_file}" ]; then
        log_error "请指定备份文件"
        echo "用法: $0 restore <backup-file>"
        exit 1
    fi

    if [ ! -f "${backup_file}" ]; then
        log_error "备份文件不存在: ${backup_file}"
        exit 1
    fi

    log_warn "即将恢复数据库，现有数据将被覆盖！"
    read -p "确认继续? (yes/no): " confirm

    if [ "${confirm}" != "yes" ]; then
        log_info "已取消"
        exit 0
    fi

    log_info "恢复数据库..."

    load_env
    gunzip < "${backup_file}" | $compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" exec -T postgres psql -U "${POSTGRES_USER}" "${POSTGRES_DB}"

    log_success "恢复完成，请重启服务: $0 restart"
}

# 查看版本
version() {
    echo ""
    echo "=== 版本信息 ==="
    echo "数据库版本: $(get_db_version)"
    echo "代码版本: $(get_code_version)"
    echo ""
}

# 清理旧备份
cleanup() {
    log_info "清理 7 天前的备份..."
    find "${BACKUP_DIR}" -name "memos-backup-*.sql.gz" -mtime +7 -delete
    log_success "清理完成"
}

# 主函数
main() {
    case "${1:-deploy}" in
        build)
            build_image
            ;;
        deploy)
            deploy
            ;;
        pull)
            pull_image
            ;;
        setup)
            setup_docker_mirror
            ;;
        upgrade)
            upgrade
            ;;
        restart)
            restart
            ;;
        stop)
            stop
            ;;
        logs)
            logs "$@"
            ;;
        status)
            status
            ;;
        backup)
            backup
            ;;
        restore)
            restore "$@"
            ;;
        version)
            version
            ;;
        cleanup)
            cleanup
            ;;
        *)
            echo "用法: $0 [命令] [参数]"
            echo ""
            echo "命令:"
            echo "  setup     - 配置 Docker 镜像加速 (国内推荐)"
            echo "  pull      - 拉取预构建镜像 (替代 build)"
            echo "  build     - 构建生产镜像"
            echo "  deploy    - 部署到生产环境 (首次安装)"
            echo "  upgrade   - 执行数据库升级"
            echo "  restart   - 重启服务"
            echo "  stop      - 停止服务"
            echo "  logs      - 查看日志 [服务名]"
            echo "  status    - 查看状态"
            echo "  backup    - 备份数据库"
            echo "  restore   - 恢复数据库 <备份文件>"
            echo "  version   - 查看版本信息"
            echo "  cleanup   - 清理 7 天前的备份"
            echo ""
            echo "一键部署流程:"
            echo "  1. cp .env.prod.example .env.prod && vi .env.prod"
            echo "  2. $0 setup       # 配置镜像加速 (可选)"
            echo "  3. $0 deploy      # 首次部署"
            echo ""
            echo "使用预构建镜像 (无需 Go/Node.js):"
            echo "  1. 在 .env.prod 设置: USER_IMAGE=ghcr.io/usememos/memos:latest"
            echo "  2. $0 deploy"
            echo ""
            echo "示例:"
            echo "  $0 deploy              # 首次部署"
            echo "  $0 upgrade             # 升级版本"
            echo "  $0 logs                # 查看所有日志"
            echo "  $0 logs postgres       # 查看数据库日志"
            echo "  $0 backup              # 备份数据库"
            echo "  $0 restore backups/xxx.sql.gz"
            exit 1
            ;;
    esac
}

main "$@"
