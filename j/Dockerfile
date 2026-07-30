# ============================================================
# Stage 1: Maven 编译
# ============================================================
FROM maven:3.9-eclipse-temurin-17 AS builder

WORKDIR /build

# 先复制 pom.xml，利用 Docker 缓存层下载依赖
COPY pom.xml .
RUN mvn dependency:go-offline -B

# 复制源码
COPY src/ ./src/

# 编译打包
RUN mvn package -DskipTests -B

# ============================================================
# Stage 2: JRE 运行时
# ============================================================
FROM eclipse-temurin:17-jre-alpine

RUN apk add --no-cache tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

WORKDIR /opt/csm

# 从构建阶段拷贝 jar
COPY --from=builder /build/target/go-to-chat-1.0.0.jar ./go-to-chat.jar

# 拷贝配置模板
COPY src/main/resources/cfg.yml.template ./cfg.yml.template

# 创建数据目录
RUN mkdir -p upload_doc vdb

EXPOSE 19007

CMD ["java", "-jar", "go-to-chat.jar"]
