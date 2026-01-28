#!/bin/bash
# =============================================================================
# DivineSense 阿里云 2C2G 一键安装脚本
# =============================================================================
#
# 使用方式:
#   curl -fsSL https://raw.githubusercontent.com/hrygo/divinesense/main/deploy/aliyun/install.sh | bash
#
# 或者下载后执行:
#   chmod +x install.sh && ./install.sh
#
# 功能:
#   - 自动安装 Docker + Docker Compose
#   - 配置国内镜像加速
#   - 下载 DivineSense 预构建镜像
#   - 初始化 PostgreSQL + pgvector
#   - 启动完整服务
#
# =============================================================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# 配置
INSTALL_DIR="/opt/divinesense"
REPO_URL="https://github.com/hrygo/divinesense.git"
BRANCH="${BRANCH:-main}"
BACKUP_DIR="${INSTALL_DIR}/backups"

# 日志函数
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step() { echo -e "${CYAN}[STEP]${NC} $1"; }

# 打印 Banner
print_banner() {
    echo ""
    echo -e "${CYAN}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${NC}  ${GREEN}DivineSense 一键部署脚本${NC}                                    ${CYAN}║${NC}"
    echo -e "${CYAN}║${NC}  ${GREEN}适用于阿里云 2C2G 服务器${NC}                                   ${CYAN}║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

# 检查是否为 root
check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_warn "建议使用 root 用户运行此脚本"
        log_info "当前用户: $USER"
        read -p "是否继续? (y/n): " confirm
        if [ "$confirm" != "y" ]; then
            exit 1
        fi
    fi
}

# 检测系统
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
        OS_VERSION=$VERSION_ID
    else
        log_error "无法检测操作系统"
        exit 1
    fi

    log_info "检测到系统: $OS $OS_VERSION"

    case "$OS" in
        alpine|arch|manjaro)
            PKG_MANAGER="apk"
            ;;
        debian|ubuntu|linuxmint)
            PKG_MANAGER="apt"
            ;;
        centos|rhel|fedora|rocky|almalinux)
            PKG_MANAGER="yum"
            ;;
        *)
            log_error "不支持的操作系统: $OS"
            exit 1
            ;;
    esac

    log_success "包管理器: $PKG_MANAGER"
}

# 安装 Docker
install_docker() {
    log_step "安装 Docker..."

    if command -v docker &>/dev/null; then
        log_success "Docker 已安装: $(docker --version)"
        return 0
    fi

    case "$PKG_MANAGER" in
        apt)
            curl -fsSL https://get.docker.com | sh
            ;;
        yum)
            curl -fsSL https://get.docker.com | sh
            ;;
        apk)
            apk add docker docker-cli-compose
            ;;
    esac

    # 启动 Docker
    systemctl enable docker
    systemctl start docker

    log_success "Docker 安装完成"
}

# 安装 Docker Compose
install_docker_compose() {
    log_step "检查 Docker Compose..."

    if docker compose version &>/dev/null; then
        log_success "Docker Compose 已安装"
        return 0
    fi

    log_info "安装 Docker Compose..."
    curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    chmod +x /usr/local/bin/docker-compose

    log_success "Docker Compose 安装完成"
}

# 配置镜像加速
setup_docker_mirror() {
    log_step "配置 Docker 镜像加速..."

    local docker_config_dir="/etc/docker"
    local daemon_config="$docker_config_dir/daemon.json"

    mkdir -p "$docker_config_dir"

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

    # 重启 Docker
    systemctl restart docker

    log_success "镜像加速配置完成"
}

# 生成随机密码
generate_password() {
    openssl rand -base64 16 | tr -d '/+=' | head -c 20
}

# 获取服务器 IP
get_server_ip() {
    # 尝试多种方式获取 IP
    local ip=$(curl -s -4 ifconfig.me 2>/dev/null)
    if [ -z "$ip" ]; then
        ip=$(curl -s -4 icanhazip.com 2>/dev/null)
    fi
    if [ -z "$ip" ]; then
        ip=$(curl -s -4 ipinfo.io/ip 2>/dev/null)
    fi
    if [ -z "$ip" ]; then
        ip=$(hostname -I | awk '{print $1}')
    fi
    echo "$ip"
}

# 创建安装目录
create_install_dir() {
    log_step "创建安装目录..."

    mkdir -p "$INSTALL_DIR"
    mkdir -p "$BACKUP_DIR"

    log_success "安装目录: $INSTALL_DIR"
}

# 克隆仓库
clone_repo() {
    log_step "下载 DivineSense 部署文件..."

    cd "$INSTALL_DIR"

    if [ -d ".git" ]; then
        log_info "更新现有仓库..."
        git pull origin "$BRANCH"
    else
        # 检查是否安装了 git
        if ! command -v git &>/dev/null; then
            log_info "安装 git..."
            case "$PKG_MANAGER" in
                apt)
                    apt-get update && apt-get install -y git
                    ;;
                yum)
                    yum install -y git
                    ;;
                apk)
                    apk add git
                    ;;
            esac
        fi

        log_info "克隆仓库..."
        git clone -b "$BRANCH" --depth 1 "$REPO_URL" "$INSTALL_DIR"
    fi

    log_success "仓库下载完成"
}

# 生成配置文件
generate_env_file() {
    log_step "生成配置文件..."

    local env_file="$INSTALL_DIR/.env.prod"
    local db_password=$(generate_password)
    local server_ip=$(get_server_ip)

    if [ -f "$env_file" ]; then
        log_warn "配置文件已存在，跳过生成"
        return 0
    fi

    cat > "$env_file" << EOF
# DivineSense 生产环境配置
# 生成时间: $(date)

# =============================================================================
# 服务配置
# =============================================================================

DIVINESENSE_PORT=5230
TZ=Asia/Shanghai
DIVINESENSE_INSTANCE_URL=http://${server_ip}:5230

# =============================================================================
# PostgreSQL 配置
# =============================================================================

POSTGRES_DB=divinesense
POSTGRES_USER=divinesense
POSTGRES_PASSWORD=${db_password}

# 数据库外部访问 (可选，取消注释后可通过 25432 端口连接)
# POSTGRES_PORT_MAPPING=127.0.0.1:25432:5432

# =============================================================================
# AI 功能配置
# =============================================================================

DIVINESENSE_AI_ENABLED=true

# SiliconFlow API (向量/重排/意图分类)
# 获取地址: https://cloud.siliconflow.cn/account/ak
DIVINESENSE_AI_SILICONFLOW_API_KEY=sk-your-siliconflow-key

# DeepSeek API (对话 LLM)
# 获取地址: https://platform.deepseek.com/api_keys
DIVINESENSE_AI_DEEPSEEK_API_KEY=sk-your-deepseek-key

# =============================================================================
# Docker 镜像配置
# =============================================================================

# 使用预构建镜像 (推荐)
USER_IMAGE=ghcr.io/hrygo/divinesense:latest

# PostgreSQL 镜像
POSTGRES_IMAGE=pgvector/pgvector:pg16
EOF

    log_success "配置文件已生成: $env_file"
    log_warn "数据库密码: ${db_password}"
    log_warn "请妥善保管密码！"

    # 保存密码到单独文件
    echo "$db_password" > "$INSTALL_DIR/.db_password"
    chmod 600 "$INSTALL_DIR/.db_password"
}

# 拉取镜像
pull_images() {
    log_step "拉取 Docker 镜像..."

    cd "$INSTALL_DIR"

    # 拉取 PostgreSQL
    log_info "拉取 PostgreSQL + pgvector..."
    docker pull pgvector/pgvector:pg16

    # 拉取 DivineSense
    log_info "拉取 DivineSense..."
    docker pull ghcr.io/hrygo/divinesense:latest

    # 标记为本地镜像名
    docker tag ghcr.io/hrygo/divinesense:latest divinesense:latest

    log_success "镜像拉取完成"
}

# 部署服务
deploy_services() {
    log_step "部署服务..."

    cd "$INSTALL_DIR"

    # 使用 docker compose 启动服务
    if docker compose version &>/dev/null; then
        docker compose -f docker/compose/prod.yml --env-file .env.prod up -d
    else
        docker-compose -f docker/compose/prod.yml --env-file .env.prod up -d
    fi

    log_success "服务启动完成"
}

# 等待服务就绪
wait_for_service() {
    log_step "等待服务启动..."

    local max_wait=60
    local waited=0

    while [ $waited -lt $max_wait ]; do
        if docker exec divinesense sh -c "cat < /dev/null > /dev/tcp/127.0.0.1/5230" 2>/dev/null; then
            log_success "DivineSense 已就绪"
            return 0
        fi
        sleep 2
        waited=$((waited + 2))
        echo -n "."
    done

    log_warn "服务启动可能需要更长时间，请手动检查"
}

# 显示部署结果
show_result() {
    local server_ip=$(get_server_ip)

    echo ""
    echo -e "${GREEN}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║${NC}  ${GREEN}部署完成！${NC}                                                  ${GREEN}║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${CYAN}访问信息:${NC}"
    echo -e "  URL:      ${YELLOW}http://${server_ip}:5230${NC}"
    echo ""
    echo -e "${CYAN}重要文件:${NC}"
    echo -e "  配置文件: ${INSTALL_DIR}/.env.prod"
    echo -e "  数据库密码: ${INSTALL_DIR}/.db_password"
    echo -e "  备份目录: ${BACKUP_DIR}"
    echo ""
    echo -e "${CYAN}常用命令:${NC}"
    echo -e "  查看状态: ${YELLOW}cd ${INSTALL_DIR} && ./deploy.sh status${NC}"
    echo -e "  查看日志: ${YELLOW}cd ${INSTALL_DIR} && ./deploy.sh logs${NC}"
    echo -e "  重启服务: ${YELLOW}cd ${INSTALL_DIR} && ./deploy.sh restart${NC}"
    echo -e "  备份数据: ${YELLOW}cd ${INSTALL_DIR} && ./deploy.sh backup${NC}"
    echo ""
    echo -e "${YELLOW}⚠️  下一步:${NC}"
    echo -e "  1. 配置 AI API Keys: ${YELLOW}vi ${INSTALL_DIR}/.env.prod${NC}"
    echo -e "  2. 重启服务: ${YELLOW}cd ${INSTALL_DIR} && ./deploy.sh restart${NC}"
    echo ""
}

# 配置防火墙
configure_firewall() {
    log_step "配置防火墙..."

    if command -v ufw &>/dev/null; then
        ufw allow 5230/tcp
        log_success "UFW 防火墙规则已添加"
    elif command -v firewall-cmd &>/dev/null; then
        firewall-cmd --permanent --add-port=5230/tcp
        firewall-cmd --reload 2>/dev/null || true
        log_success "firewalld 防火墙规则已添加"
    else
        log_info "未检测到防火墙，请手动开放 5230 端口"
    fi
}

# 配置定时备份
setup_cron_backup() {
    log_step "配置定时备份..."

    local cron_file="/etc/cron.d/divinesense-backup"

    cat > "$cron_file" << EOF
# DivineSense 每日自动备份
0 2 * * * root cd ${INSTALL_DIR} && ./deploy.sh backup && ./deploy.sh cleanup > /dev/null 2>&1
EOF

    chmod 644 "$cron_file"

    log_success "定时备份已配置 (每天凌晨 2 点)"
}

# 主函数
main() {
    print_banner

    # 交互式配置
    read -p "是否继续安装? (y/n): " confirm
    if [ "$confirm" != "y" ]; then
        log_info "安装已取消"
        exit 0
    fi

    check_root
    detect_os
    install_docker
    install_docker_compose
    setup_docker_mirror
    create_install_dir
    clone_repo
    generate_env_file
    pull_images
    deploy_services
    wait_for_service
    configure_firewall
    setup_cron_backup
    show_result
}

# 运行主函数
main "$@"
