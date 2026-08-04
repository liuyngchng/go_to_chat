# ============================================================
# kb-chat-flow 运行时镜像 (Alpine)
# 二进制在宿主机 (Ubuntu) 上编译，此处只打包运行
# ============================================================
FROM alpine:3.21

# 国内 Alpine 镜像源（加速 apk 安装）
RUN echo "https://mirrors.aliyun.com/alpine/v3.21/main" > /etc/apk/repositories && \
    echo "https://mirrors.aliyun.com/alpine/v3.21/community" >> /etc/apk/repositories

RUN apk add --no-cache \
        ca-certificates tzdata \
        fasttext \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

ENV LANG=C.UTF-8 \
    LC_ALL=C.UTF-8

WORKDIR /opt/csm

COPY kb-chat-flow .
COPY cfg.yml.template ./cfg.yml.template
COPY dt ./dt

RUN mkdir -p log upload_doc vdb dt

EXPOSE 19007

CMD ["./kb-chat-flow"]
