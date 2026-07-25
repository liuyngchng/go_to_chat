#!/bin/bash
# ============================================================
# build.sh — 构建 go_to_chat Docker 镜像
#
# 用法:
#   ./build.sh                  # 默认 tag: go_to_chat:latest
#   ./build.sh v1.0.0           # 指定版本
#   ./build.sh v1.0.0 --push    # 构建并推送
#
# 多阶段构建:
#   Stage 1: Ubuntu 24.04 编译 Go 二进制
#   Stage 2: Alpine 3.21 运行时镜像
#
# 处理 go.mod 中本地 replace 依赖:
#   github.com/gin-gonic/gin           -> /home/rd/workspace/gin
#   github.com/milvus-io/milvus-sdk-go -> /home/rd/workspace/milvus-sdk-go
# ============================================================

set -euo pipefail

# ---- 配置 ----
IMAGE_NAME="${IMAGE_NAME:-go_to_chat}"
REGISTRY="${REGISTRY:-}"                        # 镜像仓库地址，如 registry.cn-hangzhou.aliyuncs.com/your-ns
GO_VERSION="${GO_VERSION:-1.26.0}"
DOCKERFILE="${DOCKERFILE:-Dockerfile}"

# 本地 replace 依赖路径（相对宿主机）
LOCAL_DEPS=(
    "/home/rd/workspace/gin"
    "/home/rd/workspace/milvus-sdk-go"
)

# ---- 颜色输出 ----
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
info()  { echo -e "${GREEN}[INFO]${NC}  $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
err()   { echo -e "${RED}[ERR]${NC}   $*"; }

# ---- 解析参数 ----
TAG=""
PUSH=false
while [[ $# -gt 0 ]]; do
    case "$1" in
        --push) PUSH=true ;;
        -h|--help)
            echo "用法: $0 [<tag>] [--push]"
            echo ""
            echo "  <tag>      镜像标签，默认 latest"
            echo "  --push     构建后推送到仓库"
            echo ""
            echo "环境变量:"
            echo "  IMAGE_NAME   镜像名，默认 go_to_chat"
            echo "  REGISTRY     仓库地址，如 registry.cn-hangzhou.aliyuncs.com/my-ns"
            echo "  GO_VERSION   Go 版本，默认 1.26.0"
            exit 0
            ;;
        *) TAG="$1" ;;
    esac
    shift
done

TAG="${TAG:-latest}"

if [[ -n "$REGISTRY" ]]; then
    FULL_IMAGE="${REGISTRY}/${IMAGE_NAME}:${TAG}"
else
    FULL_IMAGE="${IMAGE_NAME}:${TAG}"
fi

# ---- 检查前提 ----
if ! command -v docker &>/dev/null; then
    err "docker 未安装或不在 PATH 中"
    exit 1
fi

for dep in "${LOCAL_DEPS[@]}"; do
    if [[ ! -d "$dep" ]]; then
        err "本地依赖目录不存在: $dep"
        exit 1
    fi
done

# ---- 准备临时构建上下文 ----
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BUILD_DIR="$(mktemp -d -t go_to_chat_build_XXXXXX)"
trap "rm -rf $BUILD_DIR" EXIT

info "准备构建上下文: $BUILD_DIR"

# 1. 复制项目源码（排除不需要的文件）
rsync -a \
    --exclude='.git' \
    --exclude='go_to_chat' \
    --exclude='cfg.db' \
    --exclude='cfg.db.template' \
    --exclude='app.log' \
    --exclude='upload_doc' \
    --exclude='vdb' \
    --exclude='.claude' \
    "$SCRIPT_DIR/" "$BUILD_DIR/"

# 2. 复制本地 replace 依赖到 deps/ 目录
mkdir -p "$BUILD_DIR/deps"
cp -r /home/rd/workspace/gin "$BUILD_DIR/deps/gin"
cp -r /home/rd/workspace/milvus-sdk-go "$BUILD_DIR/deps/milvus-sdk-go"

# 3. 修改 go.mod 中的 replace 路径 → 容器内路径
sed -i 's|replace github.com/gin-gonic/gin => /home/rd/workspace/gin|replace github.com/gin-gonic/gin => /deps/gin|' "$BUILD_DIR/go.mod"
sed -i 's|replace github.com/milvus-io/milvus-sdk-go/v2 => /home/rd/workspace/milvus-sdk-go|replace github.com/milvus-io/milvus-sdk-go/v2 => /deps/milvus-sdk-go|' "$BUILD_DIR/go.mod"

info "go.mod replace 路径已调整:"
grep 'replace' "$BUILD_DIR/go.mod" || true

# ---- 构建 Docker 镜像 ----
info "开始构建镜像: $FULL_IMAGE"
info "Go 版本: $GO_VERSION"

docker build \
    --build-arg GO_VERSION="$GO_VERSION" \
    -t "$FULL_IMAGE" \
    -f "$BUILD_DIR/$DOCKERFILE" \
    "$BUILD_DIR"

info "镜像构建完成: $FULL_IMAGE"

# ---- 可选推送 ----
if $PUSH; then
    if [[ -z "$REGISTRY" ]]; then
        err "推送需要设置 REGISTRY 环境变量"
        exit 1
    fi
    info "推送镜像: $FULL_IMAGE"
    docker push "$FULL_IMAGE"
    info "推送完成"
fi

# ---- 镜像信息 ----
echo ""
info "========== 镜像信息 =========="
docker images "$FULL_IMAGE" --format "table {{.Repository}}:{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"
echo ""
info "启动命令:"
echo "  docker run -d --name go_to_chat -p 19007:19007 \\"
echo "    -v \$(pwd)/cfg.yml:/opt/csm/cfg.yml \\"
echo "    -v \$(pwd)/upload_doc:/opt/csm/upload_doc \\"
echo "    -v \$(pwd)/vdb:/opt/csm/vdb \\"
echo "    $FULL_IMAGE"
