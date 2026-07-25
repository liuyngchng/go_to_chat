# go_to_chat — Java 版 AI 知识库问答系统

基于知识库的 AI 智能问答系统，使用 **Netty** HTTP 服务器，支持 LLM API（OpenAI 兼容）、Embedding API、Milvus 向量数据库，通过 YAML 配置文件进行配置。

Go 版本见 `../go/`，本项目为功能完全对标的 Java 实现。

## 项目结构

```
java/
├── pom.xml                                  # Maven 配置
├── Dockerfile                               # 多阶段构建 (Maven → JRE Alpine)
├── build.sh                                 # Docker 构建脚本
├── README.md
├── src/main/java/com/gotochat/
│   ├── Main.java                            # 入口，手动 DI 组装所有组件
│   ├── config/
│   │   └── AppConfig.java                   # YAML 配置加载 (Jackson YAML)
│   ├── model/                               # 数据模型 (12 个 POJO)
│   │   ├── Config.java                      #   配置聚合
│   │   ├── ServerConfig.java                #   服务器配置
│   │   ├── SysConfig.java                   #   系统配置
│   │   ├── APIConfig.java                   #   API 配置
│   │   ├── MilvusConfig.java                #   Milvus 配置
│   │   ├── PromptsConfig.java               #   提示词配置
│   │   ├── VdbInfo.java                     #   知识库元数据
│   │   ├── VdbFileInfo.java                 #   文件信息
│   │   ├── ChatMessage.java                 #   聊天消息
│   │   ├── ChatRequest.java                 #   聊天请求
│   │   ├── ChatHistory.java                 #   会话历史
│   │   ├── ChatCompletionRequest.java       #   OpenAI 兼容请求
│   │   ├── ChatCompletionMsg.java           #   消息体
│   │   ├── ChatCompletionChunk.java         #   流式响应片段
│   │   ├── EmbeddingRequest.java            #   向量化请求
│   │   ├── EmbeddingResponse.java           #   向量化响应
│   │   ├── SearchResult.java                #   检索结果
│   │   └── VectorRecord.java                #   向量记录
│   ├── store/
│   │   └── SQLiteStore.java                 # SQLite 元数据存储 (Druid 连接池)
│   ├── llm/
│   │   └── LLMClient.java                   # LLM 客户端 (SSE 流式)
│   ├── embedding/
│   │   └── EmbeddingClient.java             # Embedding 客户端
│   ├── vdb/
│   │   ├── VectorStore.java                 # 向量存储接口
│   │   ├── LocalVectorStore.java            # 本地 SQLite 向量存储
│   │   ├── MilvusVectorStore.java           # Milvus 远程实现
│   │   └── VectorStoreFactory.java          # 工厂: 自动选择本地/远程
│   ├── kb/
│   │   ├── KnowledgeBaseManager.java        # 知识库管理
│   │   └── TextExtractor.java               # PDF/DOCX/XLSX 文本提取
│   ├── session/
│   │   └── SessionManager.java              # 会话管理 (内存)
│   └── server/
│       ├── HttpServer.java                  # Netty 启动 + 静态文件 + 工具方法
│       ├── Router.java                      # 简单路由表
│       ├── Handler.java                     # 函数式接口
│       ├── PageHandler.java                 # HTML 页面渲染
│       ├── ChatHandler.java                 # 聊天 API (SSE)
│       └── VdbHandler.java                  # 知识库管理 API (14 个接口)
└── src/main/resources/
    ├── log4j2.xml                           # Log4j2 配置
    ├── cfg.yml.template                     # 配置模板
    ├── templates/
    │   ├── index.html                       # 聊天界面
    │   └── vdb.html                         # 知识库管理界面
    └── static/
        ├── css/style.css
        ├── js/app.js
        ├── js/vdb.js
        ├── lib/                             # marked, purify, font-awesome
        └── webfonts/
```

## 技术栈

| 组件 | 依赖 | 版本 |
|------|------|------|
| HTTP 服务器 | Netty | 4.1 |
| 数据库 | SQLite JDBC + Druid 连接池 | 3.47 / 1.2 |
| 配置解析 | Jackson YAML | 2.18 |
| JSON | Jackson | 2.18 |
| 日志 | Log4j2 | 2.24 |
| PDF 解析 | Apache PDFBox | 3.0 |
| DOCX/XLSX 解析 | Apache POI | 5.4 |
| 向量数据库 | Milvus Java SDK (可选) | 2.5 |
| JDK | Eclipse Temurin | 17 LTS |
| 构建 | Maven | 3.9+ |

## 开发环境搭建

### 1. 安装 JDK 17

```bash
# Ubuntu/Debian
sudo apt install openjdk-17-jdk

# 或从 Adoptium 下载
wget https://packages.adoptium.net/artifactory/deb/pool/main/t/temurin-17-jdk_17.0.13_amd64.deb
sudo dpkg -i temurin-17-jdk_17.0.13_amd64.deb

# 验证
java -version
# 输出: openjdk version "17.0.x" ...
```

### 2. 安装 Maven

```bash
# Ubuntu/Debian
sudo apt install maven

# 或手动安装
wget https://dlcdn.apache.org/maven/maven-3/3.9.9/binaries/apache-maven-3.9.9-bin.tar.gz
sudo tar -C /usr/local -xzf apache-maven-3.9.9-bin.tar.gz
echo 'export PATH=/usr/local/apache-maven-3.9.9/bin:$PATH' >> ~/.bashrc
source ~/.bashrc

# 验证
mvn --version
```

### 3. Maven 镜像

`pom.xml` 已配置阿里云 Maven 仓库，无需额外配置，`mvn` 命令直接走阿里云。

## 配置

### 复制配置模板

```bash
cp src/main/resources/cfg.yml.template cfg.yml
```

### 配置说明

```yaml
server:
  port: 19007       # 服务端口
  debug: true       # 调试模式 (生产环境设为 false)

sys:
  name: 对话机器人    # 页面标题
  auth: false       # 是否启用认证

api:
  # LLM API 配置 (OpenAI 兼容接口)
  llm_api_uri: https://api.deepseek.com/v1
  llm_api_key: sk-你的API密钥
  llm_model_name: deepseek-chat

  # Embedding API 配置 (OpenAI 兼容接口)
  embedding_api_uri: https://api.openai.com/v1
  embedding_api_key: sk-你的API密钥
  embedding_model_name: text-embedding-3-small

# Milvus 向量数据库配置
milvus:
  uri: ""           # 远程 Milvus 地址 (留空使用本地 SQLite 向量存储)
  token: ""         # Milvus 认证 token (可选)

# 提示词模板
prompts:
  chat_msg: |
    你是专业的知识库问答助手...
    可用变量: {context}, {history}, {question}, {cur_date}, {cur_week}
```

## 编译

```bash
cd java

# 下载依赖
mvn dependency:resolve

# 编译打包 (生成 target/go-to-chat-1.0.0.jar)
mvn package -DskipTests

# 跳过测试快速编译
mvn package -DskipTests -q
```

## 运行

```bash
# 确保 cfg.yml 和 cfg.db 在项目根目录
java -jar target/go-to-chat-1.0.0.jar
```

启动后访问：

| 页面 | 地址 |
|------|------|
| 聊天界面 | http://localhost:19007 |
| 知识库管理 | http://localhost:19007/vdb/idx |

## API 路由

### 页面

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/` | 聊天界面 |
| GET | `/vdb/idx` | 知识库管理界面 |

### 聊天

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/chat` | 发送消息 (SSE 流式返回) |
| POST | `/chat/clear` | 清空会话 |

### 知识库管理

| 方法 | 路径 | 参数 | 说明 |
|------|------|------|------|
| POST | `/vdb/my/list` | uid | 我的知识库列表 |
| POST | `/vdb/create` | uid, name, is_public | 创建知识库 |
| POST | `/vdb/delete` | uid, id | 删除知识库 |
| POST | `/vdb/set/default` | uid, id | 设为默认知识库 |
| POST | `/vdb/file/list` | uid, vdb_id | 文件列表 |
| POST | `/vdb/upload` | uid, vdb_id, file | 上传文档 (multipart) |
| POST | `/vdb/file/delete` | uid, file_id | 删除文件 |
| POST | `/vdb/process/info` | file_id | 处理进度查询 |
| POST | `/vdb/search` | uid, vdb_id, query | 知识库内搜索 |
| POST | `/vdb/pub/list` | uid | 公开知识库列表 |

## Docker 构建

```bash
# 构建镜像
./build.sh

# 指定版本号
./build.sh v1.0.0

# 构建并推送
REGISTRY=registry.cn-hangzhou.aliyuncs.com/my-ns ./build.sh v1.0.0 --push

# 运行容器
docker run -d --name go_to_chat -p 19007:19007 \
  -v $(pwd)/cfg.yml:/opt/csm/cfg.yml \
  -v $(pwd)/upload_doc:/opt/csm/upload_doc \
  -v $(pwd)/vdb:/opt/csm/vdb \
  go_to_chat:latest
```

## 向量数据库说明

### 默认：本地 SQLite 存储

当 `milvus.uri` 为空时，使用内置本地向量存储。数据保存在 `./vdb/vectors.db`，向量以 BLOB 格式存储，内存缓存加速检索，余弦相似度计算。

### 远程模式：Milvus

配置 `milvus.uri` 为远程服务地址时，使用 Milvus Java SDK 连接远程服务：

```yaml
milvus:
  uri: http://localhost:19530
  token: ""
```

