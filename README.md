# kb-chat-flow — 支持工作流配置的知识库对话机器人

基于知识库 + 多智能体工作流的 AI 对话机器人系统。可独立部署使用，也可作为客服机器人等场景的后端。

## 主要功能

### 💬 智能对话
- 多轮对话，支持会话历史上下文
- SSE 流式输出，打字机效果
- 知识库增强回答：自动检索相关文档片段注入上下文
- FAQ 优先匹配：用户问题先在 FAQ 库中精确匹配，命中率高的问题直接返回，未命中再走 RAG

### 🔀 工作流编排（可配置）
- 可视化设计对话流程：管理员在后台配置工作流，无需改代码
- 意图分类：配置意图类别和关键词，LLM 自动将用户问题分类（如：紧急、账单、业务办理、维修、综合咨询）
- 条件路由：不同意图走不同的处理分支、挂载不同的 AI Agent
- 多 Agent 串行执行：每个 Agent 完成子任务后，结果自动传递给下游节点
- 最终聚合：所有分支结果汇聚，由最终节点生成统一回复

### 🤖 AI Agent 管理
- 创建和管理多个 AI 智能体，每个可独立配置：
  - 系统提示词（设定角色、语气、回答范围）
  - 模型选择（不同 Agent 可用不同模型，如快速/强大）
  - 温度、Top-P、Max Tokens 等参数
  - 关联的知识库（每个 Agent 只检索自己相关的资料）
- Agent 可复用在多个工作流中

### 📚 知识库管理
- 创建多个知识库，支持公开/私有
- 上传文档自动处理：文本提取 → 智能切分 → 向量化 → 入库
- 支持 PDF、Word、TXT 等常见格式
- 向量检索 + Rerank 重排序，提升召回准确率
- 知识库内搜索预览

### ❓ FAQ 知识库
- 批量导入常见问答对（一题多问法 + 答案）
- 语义 + 关键词混合匹配
- 可设置置信度阈值：高分直接返回、低分走 RAG 兜底
- 支持 Excel 批量上传

### 👥 用户与权限
- 四种角色：管理员、客服座席、普通用户、API 用户
- 登录认证 / 登出
- 管理员专有后台：系统配置、用户管理、Agent 管理、工作流管理、FAQ 管理
- 用户自助：修改密码、管理 API Token、查看调用记录

### 🔌 API Token
- 用户可自行生成/查看 API Token
- Token 用于外部系统调用聊天接口
- 可设置过期时间
- 完整的调用日志记录

### 📊 管理后台
- 系统配置在线修改（LLM 接口、Embedding 接口、Rerank 接口等）
- 提示词模板在线编辑
- 用户管理（创建、删除、重置密码）
- Agent 管理（增删改查）
- 工作流管理（节点编排、意图分类器配置、条件路由）

### 🗄️ 存储后端灵活切换
- 元数据存储：SQLite（默认，零配置）或 MySQL（生产环境）
- 向量存储：本地（默认，零配置）、Milvus、Qdrant
- 按需组合，开发用 SQLite+本地即可，生产切 MySQL+Milvus

---

## 快速开始

### 1. 准备配置文件

```bash
cp cfg.yml.template cfg.yml     # 编辑 cfg.yml，填入 API 密钥等
cp cfg.db.template cfg.db       # 数据库模板
```

### 2. 启动

```bash
./kb-chat-flow
```

### 3. 访问

| 页面 | 地址 |
|---|---|
| 聊天界面 | http://localhost:19007 |
| 知识库管理 | http://localhost:19007/vdb/idx |
| 管理后台 | http://localhost:19007/admin/config |

---

## 配置说明

`cfg.yml` 中的主要配置项：

```yaml
server:
  port: 19007       # 服务端口
  debug: true       # 调试模式

sys:
  name: 对话机器人    # 系统名称
  auth: false       # 是否启用登录认证
  api_auth: false   # 是否启用 API 认证

store:
  backend: "sqlite"  # 元数据存储: sqlite | mysql

vector:
  backend: "local"   # 向量存储: local | milvus | qdrant
```

> LLM API、Embedding API 等密钥类配置请在启动后通过管理后台在线修改，无需写入配置文件。

---

## Docker 部署

```bash
docker build -t kb-chat-flow .
docker run -d -p 19007:19007 -v ./cfg.yml:/app/cfg.yml -v ./cfg.db:/app/cfg.db kb-chat-flow
```
