# go_to_chat — Go 语言 AI 知识库问答系统

基于知识库的 AI 智能问答系统，支持 LLM API、Embedding API、Milvus 向量数据库，通过 YAML 配置文件进行配置。

## 项目结构

```
go_to_chat/
├── main.go                         # 入口文件
├── cfg.yml                         # 运行配置（从 cfg.yml.template 复制）
├── cfg.yml.template                # 配置模板
├── README.md
├── go.mod / go.sum
├── internal/
│   ├── config/config.go            # YAML 配置加载
│   ├── model/types.go              # 所有数据类型定义
│   ├── store/sqlite.go             # SQLite 元数据存储（KB/文件信息）
│   ├── llm/client.go               # LLM 客户端（OpenAI 兼容，支持流式）
│   ├── embedding/client.go         # Embedding 客户端（OpenAI 兼容）
│   ├── vdb/
│   │   ├── interface.go            # 向量存储接口
│   │   ├── milvus.go               # Milvus 远程实现 + 工厂函数
│   │   └── local.go                # 本地嵌入式向量存储（默认）
│   ├── kb/manager.go               # 知识库管理 / 文件处理 / 检索
│   ├── session/manager.go          # 会话历史管理（内存）
│   └── handler/
│       ├── handler.go              # 处理器聚合
│       ├── page.go                 # 页面渲染
│       ├── chat.go                 # 聊天 API（SSE 流式）
│       └── vdb.go                  # 知识库管理 API
├── web/
│   ├── templates/
│   │   ├── index.html              # 聊天界面
│   │   └── vdb.html                # 知识库管理界面
│   └── static/
│       ├── css/style.css           # 所有样式
│       ├── js/app.js               # 聊天交互逻辑
│       ├── js/vdb.js               # 知识库管理交互
│       └── lib/                    # 第三方库（marked, purify, font-awesome）
└── upload_doc/                     # 上传文档存储目录
```

## 依赖项目

### Go 依赖包

| 包 | 用途 | GitHub 地址 |
|---|---|---|
| `github.com/gin-gonic/gin` | Web 框架 | https://github.com/gin-gonic/gin |
| `gopkg.in/yaml.v3` | YAML 配置解析 | https://github.com/go-yaml/yaml |
| `github.com/mattn/go-sqlite3` | SQLite 驱动（CGo，链接系统 libsqlite3） | https://github.com/mattn/go-sqlite3 |
| `github.com/milvus-io/milvus-sdk-go/v2` | Milvus Go SDK（可选，远程模式需要） | https://github.com/milvus-io/milvus-sdk-go |

### 系统依赖

| 依赖 | 用途 | 安装命令 |
|---|---|---|
| Go 1.21+ | 编译运行 | `sudo apt install golang-go` 或从 https://go.dev/dl/ 下载 |
| libsqlite3-dev | SQLite C 库（编译 go-sqlite3 需要） | `sudo apt install libsqlite3-dev` |

### 前端第三方库

已内嵌在 `web/static/lib/` 中，无需额外安装：

| 文件 | 用途 |
|---|---|
| `marked.min.js` | Markdown 渲染 |
| `purify.min.js` | HTML 净化（防 XSS） |
| `font-awesome.all.min.css` | 图标字体 |

## 开发环境搭建

### 1. 安装 Go

```bash
# Ubuntu/Debian
sudo apt install golang-go

# 或从官网下载最新版
wget https://go.dev/dl/go1.23.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# 验证
go version
```

### 2. 安装 SQLite 开发库

```bash
sudo apt install libsqlite3-dev
```

### 3. 配置 Go 代理（中国大陆用户）

由于国内访问 GitHub 可能受限，需要设置 Go 代理：

```bash
go env -w GOPROXY=https://goproxy.cn,direct
```

验证：

```bash
go env GOPROXY
# 输出: https://goproxy.cn,direct
```

> `goproxy.cn` 会缓存 GitHub 上的 Go 包，避免直接访问 GitHub 超时。

### 4. 解决 GitHub 无法访问的问题

如果 `goproxy.cn` 也没有缓存某些包（如 Milvus SDK），可以手动下载到本地后使用 `replace` 指令：

```bash
# 克隆需要的项目到本地
cd ~/workspace
git clone https://github.com/gin-gonic/gin.git
git clone https://github.com/milvus-io/milvus-sdk-go.git

# 在 go.mod 中添加本地路径映射
cd go_to_chat
go mod edit -replace github.com/gin-gonic/gin=/home/rd/workspace/gin
go mod edit -replace github.com/milvus-io/milvus-sdk-go/v2=/home/rd/workspace/milvus-sdk-go
```

> 如果你能直接访问 GitHub，不需要上面的步骤，`go mod tidy` 会自动下载。

### 5. 下载依赖

```bash
cd go_to_chat
go env -w GOPROXY=https://goproxy.cn,direct   # 国内用户先设置代理
go mod tidy
```

## 配置

### 复制配置模板

```bash
cp cfg.yml.template cfg.yml
```

### 配置说明

```yaml
server:
  port: 19007       # 服务端口
  debug: true       # 调试模式（生产环境设为 false）

sys:
  name: 知识库问答     # 页面标题
  auth: false       # 是否启用认证

api:
  # LLM API 配置（OpenAI 兼容接口）
  # DeepSeek 示例: https://api.deepseek.com/v1
  # OpenAI 示例: https://api.openai.com/v1
  llm_api_uri: https://api.deepseek.com/v1
  llm_api_key: sk-你的API密钥
  llm_model_name: deepseek-chat

  # Embedding API 配置（OpenAI 兼容接口）
  # 阿里云 DashScope 示例: https://dashscope.aliyuncs.com/compatible-mode/v1
  embedding_api_uri: https://api.openai.com/v1
  embedding_api_key: sk-你的API密钥
  embedding_model_name: text-embedding-3-small

# Milvus 向量数据库配置
milvus:
  # 远程 Milvus 服务地址，例如: http://localhost:19530
  # 留空则使用本地嵌入式向量存储（无需安装任何东西）
  uri: ""
  token: ""         # Milvus 认证 token（可选）

# 提示词模板
prompts:
  chat_msg: |
    你是专业的知识库问答助手...
    可用变量: {context}, {history}, {question}, {cur_date}, {cur_week}
```

## 编译

### 普通编译

```bash
# 使用系统 libsqlite3（推荐，编译快）
CGO_ENABLED=1 go build -o csm_app .

# 减小二进制体积（去掉调试信息）
CGO_ENABLED=1 go build -ldflags="-s -w" -o csm_app .
```

### 纯 Go 编译（无需安装 libsqlite3-dev）

如果不想安装 SQLite 系统库，可以换用纯 Go 的 SQLite 实现（首次编译较慢）：

```bash
# 1. 修改 internal/store/sqlite.go:
#    将 import 改为 _ "modernc.org/sqlite"
#    将 sql.Open("sqlite3", ...) 改为 sql.Open("sqlite", ...)

# 2. 编译
CGO_ENABLED=0 go build -o csm_app .
```

### 编译说明

| 选项 | 说明 |
|---|---|
| `CGO_ENABLED=1` | 启用 CGo，链接系统 C 库（libsqlite3）。编译快，但需要安装 `libsqlite3-dev` |
| `-ldflags="-s -w"` | 去掉符号表和调试信息，大约瘦身 30% |

## 运行

```bash
# 确保 cfg.yml 已配置好 API 密钥
./csm_app
```

启动后访问：

| 页面 | 地址 |
|---|---|
| 聊天界面 | http://localhost:19007 |
| 知识库管理 | http://localhost:19007/vdb/idx |

## 向量数据库说明

### 默认：本地嵌入式存储

当 `milvus.uri` 为空时，系统使用内置的本地向量存储。数据保存在 `./vdb/vdb_idx_{uid}_{kb_id}/vectors.json`，无需安装任何额外服务。

- 使用余弦相似度进行向量检索
- 数据以 JSON 格式持久化到磁盘
- 适合开发环境和数据量较小的场景

### 远程模式：Milvus

当 `milvus.uri` 配置为远程服务地址时，系统使用 Milvus Go SDK 连接远程 Milvus 服务。

1. **安装 Milvus Standalone**

   ```bash
   # Docker 方式
   docker run -d --name milvus-standalone \
     -p 19530:19530 -p 9091:9091 \
     milvusdb/milvus:latest
   ```

2. **配置 cfg.yml**

   ```yaml
   milvus:
     uri: http://localhost:19530
     token: ""
   ```

3. **Python 用户也可直接用 Milvus Lite**

   ```bash
   pip install pymilvus
   # Milvus Lite 会自动嵌入到 Python 进程中运行
   ```

## API 路由

### 页面

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/` | 聊天界面 |
| GET | `/vdb/idx` | 知识库管理界面 |

### 聊天

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/chat` | 发送消息（SSE 流式返回） |
| POST | `/chat/clear` | 清空会话 |

### 知识库管理

| 方法 | 路径 | 参数 | 说明 |
|---|---|---|---|
| POST | `/vdb/my/list` | uid | 我的知识库列表 |
| POST | `/vdb/create` | uid, name, is_public | 创建知识库 |
| POST | `/vdb/delete` | uid, id | 删除知识库 |
| POST | `/vdb/set/default` | uid, id | 设为默认知识库 |
| POST | `/vdb/file/list` | uid, vdb_id | 文件列表 |
| POST | `/vdb/upload` | uid, vdb_id, file | 上传文档 |
| POST | `/vdb/file/delete` | uid, file_id | 删除文件 |
| POST | `/vdb/process/info` | file_id | 处理进度查询 |
| POST | `/vdb/search` | uid, vdb_id, query | 知识库内搜索 |
| POST | `/vdb/pub/list` | uid | 公开知识库列表 |
