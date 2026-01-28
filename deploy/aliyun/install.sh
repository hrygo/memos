#!/bin/bash
# =============================================================================
# DivineSense 阿里云 2C2G 一键安装脚本 v2.0
# =============================================================================
#
# 使用方式:
#   curl -fsSL https://raw.githubusercontent.com/hrygo/divinesense/main/deploy/aliyun/install.sh | bash
#
# 支持系统:
#   - 阿里云 Linux 2/3
#   - CentOS 7/8
#   - Rocky Linux 8/9
#   - Ubuntu 18.04+
#   - Debian 10+
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
INSTALL_DIR="${INSTALL_DIR:-/opt/divinesense}"
REPO_URL="${REPO_URL:-https://github.com/hrygo/divinesense.git}"
BRANCH="${BRANCH:-main}"
BACKUP_DIR="${INSTALL_DIR}/backups"

# 系统要求
MIN_RAM_MB=1800
MIN_DISK_MB=4096
DOCKER_VERSION="25.0.3"

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
    echo -e "${CYAN}║${NC}  ${GREEN}DivineSense 一键部署脚本 v2.0${NC}                             ${CYAN}║${NC}"
    echo -e "${CYAN}║${NC}  ${GREEN}适用于阿里云 2C2G 服务器${NC}                                   ${CYAN}║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

# 检查系统资源
check_system_resources() {
    log_step "检查系统资源..."

    # 检查内存
    local total_mem_kb=$(grep MemTotal /proc/meminfo 2>/dev/null | awk '{print $2}')
    local total_mem_mb=$((total_mem_kb / 1024))

    if [ "$total_mem_mb" -lt "$MIN_RAM_MB" ]; then
        log_error "内存不足: 需要 ${MIN_RAM_MB}MB，当前 ${total_mem_mb}MB"
        log_info "建议升级配置或使用 swap 空间"
        return 1
    fi
    log_success "内存检查通过: ${total_mem_mb}MB"

    # 检查磁盘空间
    local available_disk_mb=$(df -m / | awk 'NR==2 {print $4}')

    if [ "$available_disk_mb" -lt "$MIN_DISK_MB" ]; then
        log_error "磁盘空间不足: 需要 ${MIN_DISK_MB}MB，当前可用 ${available_disk_mb}MB"
        return 1
    fi
    log_success "磁盘检查通过: ${available_disk_mb}MB 可用"

    return 0
}

# 检测系统
detect_os() {
    log_step "检测操作系统..."

    # 检查阿里云 Linux
    if [ -f /etc/aliyun-release ]; then
        OS="aliyun"
        . /etc/aliyun-release 2>/dev/null || true
        OS_VERSION="${VERSION_ID:-unknown}"
        PKG_MANAGER="yum"
        log_info "检测到阿里云 Linux: $OS_VERSION"
        return 0
    fi

    # 检查标准 os-release
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
        OS_VERSION=$VERSION_ID
    else
        # 兼容老系统
        if [ -f /etc/redhat-release ]; then
            OS="centos"
            OS_VERSION=$(rpm -qf /etc/redhat-release --queryformat '%{VERSION}' | cut -d. -f1)
        elif [ -f /etc/debian_version ]; then
            OS="debian"
            OS_VERSION=$(cat /etc/debian_version)
        else
            log_error "无法检测操作系统"
            exit 1
        fi
    fi

    case "$OS" in
        alpine|arch|manjaro)
            PKG_MANAGER="apk"
            ;;
        debian|ubuntu|linuxmint)
            PKG_MANAGER="apt"
            ;;
        centos|rhel|fedora|rocky|almalinux|aliyun)
            PKG_MANAGER="yum"
            ;;
        *)
            log_error "不支持的操作系统: $OS"
            log_info "支持的系统: 阿里云 Linux, CentOS, Rocky, Debian, Ubuntu"
            exit 1
            ;;
    esac

    log_success "系统: $OS $OS_VERSION | 包管理器: $PKG_MANAGER"
}

# 检查是否为 root
check_root() {
    if [ "$EUID" -ne 0 ]; then
        log_error "此脚本需要 root 权限运行"
        log_info "请使用: sudo $0"
        exit 1
    fi
}

# 安装基础工具
install_base_tools() {
    log_step "安装基础工具..."

    local tools="curl git openssl"

    case "$PKG_MANAGER" in
        apt)
            export DEBIAN_FRONTEND=noninteractive
            apt-get update -qq
            apt-get install -y -qq $tools 2>/dev/null || true
            ;;
        yum)
            yum install -y -q $tools 2>/dev/null || true
            ;;
        apk)
            apk add --no-cache $tools
            ;;
    esac

    log_success "基础工具已安装"
}

# 安装 Docker
install_docker() {
    log_step "安装 Docker ${DOCKER_VERSION}..."

    if command -v docker &>/dev/null; then
        local installed_version=$(docker --version | grep -oP '\d+\.\d+\.\d+' || echo "unknown")
        log_success "Docker 已安装: $installed_version"
        return 0
    fi

    case "$PKG_MANAGER" in
        apt)
            # Ubuntu/Debian
            if [ ! -f /usr/share/keyrings/docker-archive-keyring.gpg ]; then
                install -m 0755 -d /etc/apt/keyrings
                curl -fsSL https://download.docker.com/linux/${OS}/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
                chmod a+r /etc/apt/keyrings/docker.gpg

                echo \
                  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/${OS} \
                  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
                  tee /etc/apt/sources.list.d/docker.list > /dev/null

                apt-get update -qq
            fi
            apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-buildx-plugin
            ;;
        yum)
            # CentOS/RHEL/Aliyun Linux
            yum install -y -q yum-utils
            yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
            yum install -y -q docker-ce docker-ce-cli containerd.io docker-buildx-plugin
            ;;
        apk)
            apk add docker docker-cli-compose
            ;;
    esac

    # 启动 Docker
    systemctl enable docker 2>/dev/null || true
    systemctl start docker

    log_success "Docker 安装完成"
}

# 安装 Docker Compose
install_docker_compose() {
    log_step "安装 Docker Compose..."

    if docker compose version &>/dev/null; then
        local compose_version=$(docker compose version --short)
        log_success "Docker Compose 已安装: $compose_version"
        return 0
    fi

    # Docker Compose v2 随 Docker 已安装
    if ! docker compose version &>/dev/null; then
        log_info "安装独立 docker-compose..."
        curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
        chmod +x /usr/local/bin/docker-compose
    fi

    log_success "Docker Compose 已就绪"
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
    systemctl restart docker 2>/dev/null || service docker restart 2>/dev/null || true

    log_success "镜像加速已配置"
}

# 生成随机密码
generate_password() {
    if command -v openssl &>/dev/null; then
        openssl rand -base16 16 | tr -d '/+='
    else
        tr -dc A-Za-z0-9 </dev/urandom | head -c 20
    fi
}

# 获取服务器 IP
get_server_ip() {
    # 阿里云元数据服务（最快）
    local ip=$(curl -s --connect-timeout 1 http://100.100.100.200/latest/meta-data/network/interfaces/macs/ 2>/dev/null | head -1 | \
              xargs -I {} curl -s http://100.100.100.200/latest/meta-data/network/interfaces/{}/ipv4/primary-ip-address 2>/dev/null)

    if [ -z "$ip" ]; then
        ip=$(curl -s --connect-timeout 3 -4 ifconfig.me 2>/dev/null)
    fi
    if [ -z "$ip" ]; then
        ip=$(curl -s --connect-timeout 3 -4 icanhazip.com 2>/dev/null)
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
        git pull origin "$BRANCH" 2>/dev/null || true
    else
        # 检查 git
        if ! command -v git &>/dev/null; then
            log_info "安装 git..."
            case "$PKG_MANAGER" in
                apt) apt-get install -y -qq git ;;
                yum) yum install -y -q git ;;
                apk) apk add git ;;
            esac
        fi

        log_info "克隆仓库..."
        git clone -b "$BRANCH" --depth 1 "$REPO_URL" "$INSTALL_DIR" 2>/dev/null || {
            log_error "Git 克隆失败，尝试下载发布包..."
            # 如果 git 失败，下载预打包的发布文件
            wget -O /tmp/divinesense.tar.gz "https://github.com/hrygo/divinesense/archive/refs/heads/main.tar.gz" 2>/dev/null || {
                log_error "下载失败，请检查网络连接"
                exit 1
            }
            tar -xzf /tmp/divinesense.tar.gz -C "$INSTALL_DIR" --strip-components=1
            rm -f /tmp/divinesense.tar.gz
        }
    fi

    log_success "部署文件已下载"
}

# 生成配置文件
generate_env_file() {
    log_step "生成配置文件..."

    local env_file="$INSTALL_DIR/.env.prod"
    local db_password=$(generate_password)
    local server_ip=$(get_server_ip)

    if [ -z "$server_ip" ]; then
        log_warn "无法获取公网 IP，请手动配置 INSTANCE_URL"
        server_ip="your-server-ip"
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

# 数据库外部访问 (可选，需要 pgAdmin/DataGrip 时开启)
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

USER_IMAGE=ghcr.io/hrygo/divinesense:latest
POSTGRES_IMAGE=pgvector/pgvector:pg16
EOF

    log_success "配置文件已生成"
    log_warn "数据库密码: ${db_password}"

    # 保存密码到单独文件
    echo "$db_password" > "$INSTALL_DIR/.db_password"
    chmod 600 "$INSTALL_DIR/.db_password"
    log_info "密码已保存到: $INSTALL_DIR/.db_password"
}

# 拉取镜像
pull_images() {
    log_step "拉取 Docker 镜像..."

    # 拉取 PostgreSQL
    log_info "拉取 PostgreSQL + pgvector..."
    docker pull pgvector/pgvector:pg16 || {
        log_error "镜像拉取失败"
        log_info "尝试配置镜像加速..."
        setup_docker_mirror
        docker pull pgvector/pgvector:pg16
    }

    # 拉取 DivineSense
    log_info "拉取 DivineSense..."
    docker pull ghcr.io/hrygo/divinesense:latest || {
        log_warn "官方镜像可能不存在，跳过..."
    }

    # 标记为本地镜像名
    if docker images | grep -q "ghcr.io/hrygo/divinesense"; then
        docker tag ghcr.io/hrygo/divinesense:latest divinesense:latest 2>/dev/null || true
    fi

    log_success "镜像准备完成"
}

# 部署服务
deploy_services() {
    log_step "部署服务..."

    cd "$INSTALL_DIR"

    # 确保存在必要的目录结构
    mkdir -p "$INSTALL_DIR/docker/compose"

    # 检查 compose 文件
    if [ ! -f "$INSTALL_DIR/docker/compose/prod.yml" ]; then
        log_error "缺少部署文件，请重新运行: clone_repo"
        exit 1
    fi

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

    local max_wait=90
    local waited=0

    while [ $waited -lt $max_wait ]; do
        # 检查 PostgreSQL 是否就绪
        if docker exec divinesense-postgres pg_isready -U divinesense &>/dev/null; then
            log_success "PostgreSQL 已就绪"
        fi

        # 检查 DivineSense 是否就绪
        if docker exec divinesense sh -c "cat < /dev/null > /dev/tcp/127.0.0.1/5230" 2>/dev/null; then
            log_success "DivineSense 已就绪"
            return 0
        fi

        sleep 3
        waited=$((waited + 3))
        echo -n "."
    done

    log_warn "服务可能需要更长时间启动，请检查日志"
    return 0
}

# 配置防火墙
configure_firewall() {
    log_step "配置防火墙..."

    local configured=false

    # UFW (Ubuntu/Debian)
    if command -v ufw &>/dev/null; then
        ufw allow 5230/tcp 2>/dev/null || true
        log_success "UFW 防火墙规则已添加"
        configured=true
    fi

    # firewalld (CentOS/RHEL/Aliyun Linux)
    if command -v firewall-cmd &>/dev/null; then
        if systemctl is-active firewalld &>/dev/null; then
            firewall-cmd --permanent --add-port=5230/tcp 2>/dev/null || true
            firewall-cmd --reload 2>/dev/null || true
            log_success "firewalld 防火墙规则已添加"
            configured=true
        fi
    fi

    # iptables (通用)
    if [ "$configured" = false ]; then
        if command -v iptables &>/dev/null; then
            # 检查是否已有规则
            if ! iptables -C INPUT -p tcp --dport 5230 -j ACCEPT &>/dev/null; then
                iptables -I INPUT -p tcp --dport 5230 -j ACCEPT
                # 保存规则
                if command -v iptables-save &>/dev/null; then
                    iptables-save > /etc/iptables.rules 2>/dev/null || true
                fi
                log_success "iptables 防火墙规则已添加"
            fi
        else
            log_warn "未检测到防火墙，请手动开放 5230 端口"
        fi
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

    # 确保 cron 服务运行
    systemctl enable crond 2>/dev/null || systemctl enable cron 2>/dev/null || true

    log_success "定时备份已配置 (每天凌晨 2 点)"
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

# 主函数
main() {
    print_banner

    # 系统检查
    check_root
    detect_os
    check_system_resources || exit 1
    install_base_tools

    # Docker 安装
    install_docker
    install_docker_compose
    setup_docker_mirror

    # 部署
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
