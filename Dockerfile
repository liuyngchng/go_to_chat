FROM ubuntu:24.04

# 创建运行目录
WORKDIR /opt/csm

# 拷贝二进制和静态资源
COPY go_to_chat .
COPY web/ ./web/
COPY cfg.yml.template ./cfg.yml.template

# 创建数据目录
RUN mkdir -p upload_doc vdb

EXPOSE 19007

CMD ["./go_to_chat"]
