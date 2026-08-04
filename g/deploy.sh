#!/bin/bash
# ============================================================
# deploy.sh — 启动 kb-chat-flow Docker 容器（给运维用）
#
# 用法:
#   ./deploy.sh                  # 默认 tag: latest
#   ./deploy.sh v1.0.0           # 指定版本
#
# 说明:
#   - 必须解压到 kb-chat-flow/ 目录下再运行（与镜像内路径/挂载对应）
# ============================================================

set -euo pipefail

# ---- 配置 ----
IMAGE_NAME="${IMAGE_NAME:-kb-chat-flow}"
CONTAINER_NAME="${CONTAINER_NAME:-kb-chat-flow}"
TAG="${1:-latest}"
FULL_IMAGE="${IMAGE_NAME}:${TAG}"
PORT="${PORT:-19007}"

# ---- 颜色 ----
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
info()  { echo -e "${GREEN}[INFO]${NC}  $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
err()   { echo -e "${RED}[ERR]${NC}   $*"; }

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

# ---- 检查 docker ----
if ! command -v docker &>/dev/null; then
    err "docker 未安装"
    exit 1
fi

# ---- 初始化配置文件 ----
CONFIG_COPIED=false

if [[ ! -f cfg.yml ]]; then
    if [[ -f cfg.yml.template ]]; then
        cp cfg.yml.template cfg.yml
        info "已从 cfg.yml.template 创建 cfg.yml，请按需修改"
        CONFIG_COPIED=true
    else
        err "cfg.yml.template 不存在，无法创建 cfg.yml"
        exit 1
    fi
fi

if [[ ! -f cfg.db ]]; then
    if [[ -f cfg.db.template ]]; then
        cp cfg.db.template cfg.db
        info "已从 cfg.db.template 创建 cfg.db"
    else
        warn "cfg.db.template 不存在，将创建空 cfg.db"
        touch cfg.db
    fi
fi

if $CONFIG_COPIED; then
    warn "============================================"
    warn "  首次启动：请先编辑 cfg.yml 后再重新执行"
    warn "  vim cfg.yml"
    warn "============================================"
    exit 0
fi

# ---- 创建数据目录 ----
mkdir -p log upload_doc vdb dt

# ---- 停旧容器 ----
if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    warn "容器 ${CONTAINER_NAME} 已存在，正在移除..."
    docker stop "${CONTAINER_NAME}" 2>/dev/null || true
    docker rm "${CONTAINER_NAME}" 2>/dev/null || true
fi

# ---- 拉取镜像 ----
if ! docker images --format '{{.Repository}}:{{.Tag}}' | grep -q "^${FULL_IMAGE}$"; then
    warn "本地无镜像 ${FULL_IMAGE}，请先执行 build.sh 构建，或手动 docker pull"
    info "尝试启动已有镜像..."
fi

# ---- 启动 ----
info "启动容器: ${CONTAINER_NAME}"
info "  镜像:   ${FULL_IMAGE}"
info "  端口:   ${PORT}"

docker run -d \
    --name "${CONTAINER_NAME}" \
    --restart unless-stopped \
    -p "${PORT}:19007" \
    -v "${SCRIPT_DIR}/cfg.yml:/opt/csm/cfg.yml" \
    -v "${SCRIPT_DIR}/cfg.db:/opt/csm/cfg.db" \
    -v "${SCRIPT_DIR}/log:/opt/csm/log" \
    -v "${SCRIPT_DIR}/upload_doc:/opt/csm/upload_doc" \
    -v "${SCRIPT_DIR}/vdb:/opt/csm/vdb" \
    -v "${SCRIPT_DIR}/dt:/opt/csm/dt" \
    "${FULL_IMAGE}"

# ---- 检查启动状态 ----
sleep 2
if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    info "容器启动成功"
    echo ""
    info "查看日志:  docker logs -f ${CONTAINER_NAME}"
    info "访问地址:  http://localhost:${PORT}"
else
    err "容器启动失败，查看日志:"
    docker logs "${CONTAINER_NAME}" 2>/dev/null || true
    exit 1
fi
