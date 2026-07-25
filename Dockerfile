# ============================================================
# Stage 1: Ubuntu 环境编译 Go 二进制
# ============================================================
FROM ubuntu:24.04 AS builder

# 安装编译工具链
RUN apt-get update && apt-get install -y --no-install-recommends \
    wget ca-certificates git \
    && rm -rf /var/lib/apt/lists/*

# 安装 Go（版本通过 BUILDARG 传入，默认 1.26.0）
ARG GO_VERSION=1.26.0
RUN wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -O /tmp/go.tar.gz && \
    tar -C /usr/local -xzf /tmp/go.tar.gz && \
    rm /tmp/go.tar.gz
ENV PATH=/usr/local/go/bin:$PATH
ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /build

# 先复制依赖描述文件，利用 Docker 缓存层
COPY go.mod go.sum ./

# 复制本地 replace 的依赖包
COPY deps/gin /deps/gin
COPY deps/milvus-sdk-go /deps/milvus-sdk-go

# 下载依赖
RUN go mod download

# 复制源码
COPY main.go ./
COPY internal/ ./internal/

# 纯静态编译
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o go_to_chat .

# ============================================================
# Stage 2: Alpine 轻量运行时
# ============================================================
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

WORKDIR /opt/csm

# 从构建阶段拷贝二进制
COPY --from=builder /build/go_to_chat .

# 拷贝静态资源和配置模板
COPY web/ ./web/
COPY cfg.yml.template ./cfg.yml.template

# 创建数据目录
RUN mkdir -p upload_doc vdb

EXPOSE 19007

CMD ["./go_to_chat"]
