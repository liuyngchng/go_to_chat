# Chat2KB API 接口文档

> Go (`g/`)、Java (`j/`)、Python (`llm_agent/apps/csm/`) 三版本共用。
>
> 示例中 `<HOST>` = `http://localhost:19007`，`<TOKEN>` 通过登录接口获取。

---

## 概述

- **Base URL**: `<HOST>/api/`
- **Content-Type**: `application/json`
- **认证**: `Authorization: Bearer <TOKEN>`
- **成功**: `{"data": ...}` 或 `{"status": "ok"}`
- **失败**: `{"error": "..."}`

---

## 1. 认证

### 登录

```bash
curl -X POST <HOST>/api/login \
  -H "Content-Type: application/json" \
  -d '{"user_name":"admin","password":"admin"}'
```

```json
{"status":"ok","token":"eyJ...","user_name":"admin","role":1}
```

### 登出

`POST /api/logout`

```bash
curl -X POST <HOST>/api/logout \
  -H "Authorization: Bearer <TOKEN>"
```

```json
{"status":"ok"}
```

### 获取当前用户

`GET /api/me`

```bash
curl <HOST>/api/me -H "Authorization: Bearer <TOKEN>"
```

```json
{"user_name":"admin","role":1}
```

### 查询在线座席

`GET /api/agents`

```bash
curl <HOST>/api/agents -H "Authorization: Bearer <TOKEN>"
```

```json
{"agents":[{"user_name":"person0","login_time":"2026-08-07T10:30:00Z","note":"内置客服座席"}]}
```

---

## 2. 对话

### 发送消息（SSE 流式）

`POST /api/chat`

根据 `sys.work_mode` 自动路由：`0`=知识库问答，`1`=CSM 硬编码工作流，`2`=动态工作流。

```bash
curl -X POST <HOST>/api/chat \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"msg":"你好，请问营业时间？"}'
```

```
// SSE 流 (text/event-stream)
data:                         ← 初始化
data: [步骤 1/3] 意图分类: faq  ← work_mode=1 时的进度
data: 营业时间为周一至周五...     ← LLM 流式输出
data: [DONE]                   ← 结束
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| msg | string | ✅ | 用户消息 |
| uid | string |   | 会话标识；当 `api_auth=false` 时生效，可管理多会话 |

> **UID 行为**：`api_auth=true` 时强制使用 token 解析的 uid；`api_auth=false` 时优先使用请求中的 `uid`，未传则 fallback 为 token uid。

### 发送消息（同步/非流式）

`POST /api/chat/sync`

与流式接口共享同一套工作模式路由逻辑，返回 JSON 而非 SSE。

```bash
curl -X POST <HOST>/api/chat/sync \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"msg":"你好，请问营业时间？"}'
```

```json
{"answer":"营业时间为周一至周五 8:00-18:00，周六 9:00-12:00。","source":"kb"}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| answer | string | 完整回复内容 |
| source | string | 来源: `"faq"` / `"kb"` / `"csm"` / `"dynamic"` |
| score | float | 仅 FAQ 匹配时返回，匹配分数 |

### 查询历史

`GET /api/chat/history`

```bash
curl <HOST>/api/chat/history -H "Authorization: Bearer <TOKEN>"
```

```json
{"data":[{"role":"user","content":"你好"},{"role":"assistant","content":"您好！请问有什么可以帮您？"}]}
```

> 每个用户按 `uid` 独立存储，最多保留最近 5 轮（10 条）。服务重启后自动从数据库恢复。

### 清空会话

`POST /api/chat/clear`

```bash
curl -X POST <HOST>/api/chat/clear \
  -H "Authorization: Bearer <TOKEN>"
```

```json
{"status":"ok"}
```

---

## 3. 健康检查 & 服务信息

### 健康检查

`GET /health`

免认证，用于负载均衡器和监控探活。

```bash
curl <HOST>/health
```

```json
{"status":"ok"}
```

### 服务信息

`GET /api/info`

```bash
curl <HOST>/api/info -H "Authorization: Bearer <TOKEN>"
```

```json
{
  "name": "对话机器人",
  "version": "1.0.0",
  "work_mode": 0,
  "vector_backend": "local",
  "store_backend": "sqlite",
  "supported_file_types": ["txt", "md", "pdf", "docx", "xlsx"],
  "api_auth_enabled": true
}
```

---

## 4. 管理配置

### 获取配置

`GET /api/config`

```bash
curl <HOST>/api/config -H "Authorization: Bearer <TOKEN>"
```

```json
{
  "data": {
    "sys": {"name":"对话机器人","auth":"false","api_auth":"true","work_mode":0,"default_workflow_id":0},
    "api": {"llm_api_uri":"https://...","llm_api_key":"sk-...","llm_model_name":"gpt-4","embedding_api_uri":"...","embedding_api_key":"...","embedding_model_name":"...","rerank_api_uri":"...","rerank_api_key":"...","rerank_model_name":"..."},
    "prompt": {"chat_msg":"你是专业的对话机器人..."},
    "kb": {"chunk_size":300,"chunk_overlap":80,"top_k":3,"score_threshold":0.1,"rerank_enabled":false,"rerank_retrieve_n":15},
    "llm": {"temperature":0.7,"top_p":0.9,"max_tokens":2048},
    "faq": {"match_threshold":0.85}
  }
}
```

### 更新配置

`PUT /api/config` ⚠️ 仅管理员

```bash
curl -X PUT <HOST>/api/config \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "sys": {"name":"对话机器人","api_auth":"true","work_mode":0,"default_workflow_id":0},
    "api": {"llm_api_uri":"https://api.openai.com/v1","llm_api_key":"sk-xxx","llm_model_name":"gpt-4"},
    "prompt": {"chat_msg":"你是专业的对话机器人..."},
    "kb": {"chunk_size":300,"chunk_overlap":80,"top_k":3,"score_threshold":0.1},
    "llm": {"temperature":0.7,"top_p":0.9,"max_tokens":2048},
    "faq": {"match_threshold":0.85}
  }'
```

```json
{"status":"ok"}
```

> 只传需要修改的字段。`sys.auth` 仅从 `cfg.yml` 读取，不可通过 API 修改。

### 测试模型连接

`POST /api/config/test-models` ⚠️ 仅管理员

```bash
curl -X POST <HOST>/api/config/test-models \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "llm_api_uri":"https://api.openai.com/v1",
    "llm_api_key":"sk-xxx",
    "llm_model_name":"gpt-4",
    "embedding_api_uri":"https://api.openai.com/v1",
    "embedding_api_key":"sk-xxx",
    "embedding_model_name":"text-embedding-3-small",
    "rerank_api_uri":"https://api.jina.ai/v1",
    "rerank_api_key":"jina-xxx",
    "rerank_model_name":"jina-reranker-v2"
  }'
```

```json
{
  "results": [
    {"name":"LLM 对话模型","ok":true,"message":"连接成功","elapsed_ms":234},
    {"name":"Embedding 向量模型","ok":true,"message":"连接成功 (dim=1536)","elapsed_ms":120},
    {"name":"Rerank 重排序模型","ok":true,"message":"连接成功","elapsed_ms":98}
  ],
  "all_ok": true
}
```

---

## 5. 管理知识库

### 查询我的知识库

`GET /api/vdb`

```bash
curl <HOST>/api/vdb -H "Authorization: Bearer <TOKEN>"
```

```json
{"data":[{"id":1,"name":"燃气知识库","uid":"admin","is_public":false,"is_default":true,"create_time":"..."}]}
```

### 查询公共知识库

`GET /api/vdb/pub`

```bash
curl <HOST>/api/vdb/pub -H "Authorization: Bearer <TOKEN>"
```

### 创建知识库

`POST /api/vdb`

```bash
curl -X POST <HOST>/api/vdb \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"name":"新知识库","is_public":false}'
```

```json
{"status":"ok","id":3}
```

### 删除知识库

`DELETE /api/vdb/:id`

```bash
curl -X DELETE <HOST>/api/vdb/3 -H "Authorization: Bearer <TOKEN>"
```

```json
{"status":"ok"}
```

### 设为默认

`PUT /api/vdb/:id/default`

```bash
curl -X PUT <HOST>/api/vdb/1/default -H "Authorization: Bearer <TOKEN>"
```

```json
{"status":"ok"}
```

### 查询文件列表

`GET /api/vdb/:id/files`

```bash
curl <HOST>/api/vdb/1/files -H "Authorization: Bearer <TOKEN>"
```

```json
{"data":[{"id":1,"name":"faq.txt","percent":100,"process_info":"处理完成","created_at":"..."}]}
```

### 上传文件

`POST /api/vdb/:id/upload`

```bash
curl -X POST <HOST>/api/vdb/1/upload \
  -H "Authorization: Bearer <TOKEN>" \
  -F "file=@/path/to/faq.txt"
```

```json
{"status":"ok","file":{"id":1}}
```

支持格式: `txt`, `md`, `pdf`, `docx`, `xlsx`

### 搜索知识库

`POST /api/vdb/search`

支持三种搜索模式：

**模式 1 — 搜索多个知识库**（推荐）：

```bash
curl -X POST <HOST>/api/vdb/search \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"query":"燃气费怎么算","vdb_ids":[1,2]}'
```

**模式 2 — 搜索单个知识库**（兼容旧版）：

```bash
curl -X POST <HOST>/api/vdb/search \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"query":"燃气费怎么算","vdb_id":1}'
```

**模式 3 — 不指定知识库，搜索所有可访问的**：

```bash
curl -X POST <HOST>/api/vdb/search \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"query":"燃气费怎么算"}'
```

```json
{"data":"[燃气知识库]\n阶梯气价...\n"}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| query | string | ✅ | 搜索关键词 |
| vdb_ids | int[] |   | 多个知识库 ID，优先级最高 |
| vdb_id | int |   | 单个知识库 ID（兼容旧版） |

> 优先级: `vdb_ids` > `vdb_id` > 搜索所有可访问知识库（我的 + 公开）

### 查询处理进度

`GET /api/vdb/file/:id/progress`

```bash
curl <HOST>/api/vdb/file/1/progress -H "Authorization: Bearer <TOKEN>"
```

```json
{"data":{"percent":75,"process_info":"正在向量化..."}}
```

### 删除文件

`DELETE /api/vdb/file/:id`

```bash
curl -X DELETE <HOST>/api/vdb/file/1 -H "Authorization: Bearer <TOKEN>"
```

```json
{"status":"ok"}
```

### 下载文件

`GET /api/vdb/file/:id/download`

```bash
curl <HOST>/api/vdb/file/1/download -H "Authorization: Bearer <TOKEN>" -o myfile.txt
```

> 仅文件上传者可下载。

### 查询文件分块

`GET /api/vdb/file/:id/chunks`

查看文档被切分和向量化后的所有文本块。

```bash
curl <HOST>/api/vdb/file/1/chunks -H "Authorization: Bearer <TOKEN>"
```

```json
{"data":[{"id":"faq.txt_chunk_0","content":"营业时间 周一至周五...","metadata":{"source":"/abs/path/to/faq.txt"},"score":0}]}
```

> 当前仅 `local` 向量后端支持，Milvus/Qdrant 返回空数组。

### 查询知识库绑定

`GET /api/vdb/bindings` ⚠️ 仅管理员

```bash
curl <HOST>/api/vdb/bindings -H "Authorization: Bearer <TOKEN>"
```

```json
{"data":{"billing":[1,2],"repair":[3],"faq":[1,3]}}
```

### 保存知识库绑定

`PUT /api/vdb/bindings` ⚠️ 仅管理员

```bash
curl -X PUT <HOST>/api/vdb/bindings \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"billing":[1,2],"repair":[3],"faq":[1,3]}'
```

```json
{"status":"ok"}
```

> 保存后 CSM 引擎热加载生效，无需重启。

---

## 6. 管理 FAQ

### 查询 FAQ 列表

`GET /api/faq`

```bash
curl <HOST>/api/faq -H "Authorization: Bearer <TOKEN>"
```

```json
{
  "data": [{
    "id":1,
    "answer":"营业时间周一至周五 8:00-18:00",
    "source_file":"faq.txt",
    "created_at":"...",
    "questions":[{"id":1,"question":"营业时间"},{"id":2,"question":"几点开门"}]
  }]
}
```

### FAQ 匹配

`POST /api/faq/match`

独立匹配接口，不经过 LLM，直接返回最匹配的 FAQ 答案。

```bash
curl -X POST <HOST>/api/faq/match \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"query":"营业时间"}'
```

```json
{"answer":"营业时间周一至周五 8:00-18:00","score":0.92,"matched":true}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| answer | string | 匹配到的答案，未匹配为空字符串 |
| score | float | 余弦相似度分数 (0~1) |
| matched | bool | 是否达到阈值匹配成功 |

### 下载 FAQ 模板

`GET /api/faq/template`

```bash
curl <HOST>/api/faq/template -H "Authorization: Bearer <TOKEN>" -o faq_template.txt
```

### 创建 FAQ

`POST /api/faq` ⚠️ 仅管理员

```bash
curl -X POST <HOST>/api/faq \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"answer":"营业时间周一至周五 8:00-18:00","questions":["营业时间","几点开门"]}'
```

```json
{"status":"ok"}
```

### 上传 FAQ 文件

`POST /api/faq/upload` ⚠️ 仅管理员

```bash
curl -X POST <HOST>/api/faq/upload \
  -H "Authorization: Bearer <TOKEN>" \
  -F "file=@faq.txt"
```

```json
{"status":"ok","created":15,"total":15}
```

### 更新 FAQ

`PUT /api/faq/:id` ⚠️ 仅管理员

```bash
curl -X PUT <HOST>/api/faq/1 \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"answer":"新的回复内容","questions":["新问题1","新问题2"]}'
```

```json
{"status":"ok"}
```

### 删除 FAQ

`DELETE /api/faq/:id` ⚠️ 仅管理员

```bash
curl -X DELETE <HOST>/api/faq/1 -H "Authorization: Bearer <TOKEN>"
```

```json
{"status":"ok"}
```

### 清空 FAQ

`DELETE /api/faq` ⚠️ 仅管理员

```bash
curl -X DELETE <HOST>/api/faq -H "Authorization: Bearer <TOKEN>"
```

```json
{"status":"ok"}
```

---

## 7. 管理用户

### 查询用户列表

`GET /api/users` ⚠️ 仅管理员

```bash
curl <HOST>/api/users -H "Authorization: Bearer <TOKEN>"
```

```json
{"data":[{"uid":1,"user_name":"admin","role":1,"note":"内置管理员"}]}
```

### 创建用户

`POST /api/users` ⚠️ 仅管理员

```bash
curl -X POST <HOST>/api/users \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"user_name":"new_user","user_pwd":"123456","role":0,"note":""}'
```

```json
{"status":"ok"}
```

| role | 说明 |
|------|------|
| 0 | 普通用户 |
| 1 | 管理员 |
| 2 | 客服座席 |
| 3 | API 用户 |

### 删除用户

`DELETE /api/users/:name` ⚠️ 仅管理员

```bash
curl -X DELETE <HOST>/api/users/new_user -H "Authorization: Bearer <TOKEN>"
```

```json
{"status":"ok"}
```

### 重置密码

`PUT /api/users/:name/reset-pwd` ⚠️ 仅管理员

```bash
curl -X PUT <HOST>/api/users/admin/reset-pwd \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"user_pwd":"admin"}'
```

```json
{"status":"ok"}
```

---

## 8. 用户自助

### 修改密码

`PUT /api/user/password`

```bash
curl -X PUT <HOST>/api/user/password \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"old_pwd":"admin","new_pwd":"newpass123"}'
```

```json
{"status":"ok"}
```

### 查询我的 Token

`GET /api/user/tokens`

```bash
curl <HOST>/api/user/tokens -H "Authorization: Bearer <TOKEN>"
```

```json
{"data":[{"id":1,"token_preview":"eyJ...XXX","expires_at":"2026-09-06T...","create_time":"..."}]}
```

### 生成 Token

`POST /api/user/token`

```bash
curl -X POST <HOST>/api/user/token \
  -H "Authorization: Bearer <TOKEN>"
```

```json
{"status":"ok","token":"eyJhbGciOi...","expires_at":"2026-09-06T10:00:00Z"}
```

### 查询调用日志

`GET /api/user/call-logs`

```bash
curl <HOST>/api/user/call-logs -H "Authorization: Bearer <TOKEN>"
```

```json
{"data":[{"id":1,"api_path":"/api/chat","method":"POST","status_code":200,"created_at":"..."}]}
```

---

## 9. 管理 Agent

> 所有认证用户均可读写 Agent。

### 查询全部 Agent

`GET /api/ai-agents`

```bash
curl <HOST>/api/ai-agents -H "Authorization: Bearer <TOKEN>"
```

```json
{"data":[{"id":1,"name":"通用客服","description":"默认智能体","system_prompt":"你是专业的对话机器人...","model_name":"gpt-4","vdb_ids":"[1,2]","created_at":"...","updated_at":"..."}]}
```

### 查询公开 Agent 列表

`GET /api/ai-agents/public`

```bash
curl <HOST>/api/ai-agents/public -H "Authorization: Bearer <TOKEN>"
```

```json
{"data":[{"id":1,"name":"通用客服"}]}
```

> 仅返回 `id` + `name`，供聊天页下拉选择用。

### 创建 Agent

`POST /api/ai-agents`

```bash
curl -X POST <HOST>/api/ai-agents \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"燃气客服",
    "description":"回答燃气相关问题",
    "system_prompt":"你是专业的燃气客服...",
    "model_name":"gpt-4",
    "temperature":0.7,
    "top_p":0.9,
    "max_tokens":2048,
    "vdb_ids":[1,2]
  }'
```

```json
{"status":"ok","id":2}
```

### 查询 Agent 详情

`GET /api/ai-agents/:id`

```bash
curl <HOST>/api/ai-agents/1 -H "Authorization: Bearer <TOKEN>"
```

```json
{"data":{"id":1,"name":"通用客服","description":"...","system_prompt":"...","model_name":"gpt-4","vdb_ids":"[1,2]","created_at":"...","updated_at":"..."}}
```

### 更新 Agent

`PUT /api/ai-agents/:id`

```bash
curl -X PUT <HOST>/api/ai-agents/1 \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"name":"燃气客服 v2","system_prompt":"更新后的提示词..."}'
```

```json
{"status":"ok"}
```

### 删除 Agent

`DELETE /api/ai-agents/:id`

```bash
curl -X DELETE <HOST>/api/ai-agents/2 -H "Authorization: Bearer <TOKEN>"
```

```json
{"status":"ok"}
```

---

## 10. 管理工作流

> 读取（列表/详情）：所有认证用户可访问。创建/更新/删除：仅管理员。

### 查询公开工作流列表

`GET /api/workflows`

```bash
curl <HOST>/api/workflows -H "Authorization: Bearer <TOKEN>"
```

```json
{"data":[{"id":1,"name":"燃气客服工作流","description":"处理燃气客服咨询","classifier":{...},"nodes":[...]}]}
```

### 查询工作流详情

`GET /api/workflows/:id`

所有认证用户均可查看详情（含节点、分类器配置）。

```bash
curl <HOST>/api/workflows/1 -H "Authorization: Bearer <TOKEN>"
```

```json
{"data":{"id":1,"name":"燃气客服工作流","description":"...","classifier":{...},"nodes":[...]}}
```

### 创建工作流

`POST /api/workflows` ⚠️ 仅管理员

```bash
curl -X POST <HOST>/api/workflows \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"燃气客服工作流",
    "description":"处理燃气客服咨询",
    "classifier":{
      "output_var":"intent",
      "prompt":"你是一个燃气客服意图分类器，只输出类别名称。",
      "categories":[
        {"name":"emergency","description":"紧急情况","keywords":["漏气","报警","燃气味"]},
        {"name":"billing","description":"账单查询","keywords":["账单","缴费","欠费"]},
        {"name":"business","description":"业务办理","keywords":["开户","过户","报装"]},
        {"name":"repair","description":"故障维修","keywords":["维修","坏了","打不着火"]},
        {"name":"faq","description":"常见咨询","keywords":["营业时间","电话","地址"]}
      ]
    },
    "nodes":[
      {"id":"bill","agent_id":1,"input_template":"{{sys.user_query}}","output_var":"bill_result","condition":"billing","is_final":true},
      {"id":"faq","agent_id":1,"input_template":"{{sys.user_query}}","output_var":"faq_result","is_final":true}
    ]
  }'
```

```json
{"status":"ok","id":1}
```

### 更新工作流

`PUT /api/workflows/:id` ⚠️ 仅管理员

```bash
curl -X PUT <HOST>/api/workflows/1 \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"name":"燃气客服 v2"}'
```

```json
{"status":"ok"}
```

### 删除工作流

`DELETE /api/workflows/:id` ⚠️ 仅管理员

```bash
curl -X DELETE <HOST>/api/workflows/2 -H "Authorization: Bearer <TOKEN>"
```

```json
{"status":"ok"}
```

---

## 11. 测试意图分类

`POST /api/classifier/test`

所有认证用户可用。

```bash
curl -X POST <HOST>/api/classifier/test \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"workflow_id":1,"text":"燃气费怎么查"}'
```

```json
{
  "tiers": [
    {"name":"关键词匹配","matched":true,"result":"billing","score":1.0,"elapsed_ms":0},
    {"name":"fastText","matched":false,"skipped":true,"elapsed_ms":0},
    {"name":"语义相似度","matched":false,"skipped":true,"elapsed_ms":0},
    {"name":"LLM分类","matched":false,"skipped":true,"elapsed_ms":0}
  ],
  "final":"billing",
  "total_ms":2
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| workflow_id | int | ✅ | 工作流 ID |
| text | string | ✅ | 待分类的用户输入 |

---

## 12. 查询系统变量

`GET /api/system-vars`

```bash
curl <HOST>/api/system-vars -H "Authorization: Bearer <TOKEN>"
```

```json
{
  "data": [
    {"name":"sys.user_query","description":"用户当前问题"},
    {"name":"sys.history","description":"历史对话记录"},
    {"name":"sys.cur_date","description":"当前日期 (YYYY-MM-DD)"},
    {"name":"sys.cur_week","description":"当前星期几（中文）"},
    {"name":"sys.kb_context","description":"知识库检索结果"},
    {"name":"sys.intent","description":"意图分类结果"}
  ]
}
```

---

## 附录

### 角色

| role | 说明 | 权限概述 |
|------|------|---------|
| 0 | 普通用户 | 聊天、查看知识库、查看 FAQ、查看工作流/Agent |
| 1 | 管理员 | 全部权限：配置管理、用户管理、FAQ/工作流/Agent 写操作 |
| 2 | 客服座席 | 同普通用户 + 座席在线状态 |
| 3 | API 用户 | 同普通用户，适合第三方程序调用 |

### 认证行为

| api_auth | 行为 |
|----------|------|
| `true` | 所有 `/api/*` 路由需携带 `Authorization: Bearer <TOKEN>` |
| `false` | `/api/*` 可免 token 访问；聊天接口支持通过 `uid` 字段管理多会话 |

### 工作模式

| work_mode | 路由逻辑 |
|-----------|---------|
| 0 | FAQ 匹配 → 知识库检索 → LLM |
| 1 | 意图分类 → 按意图路由 → 检索(可选) → LLM |
| 2 | 从 `workflow_def` 表加载 DAG 配置执行 |

### API 端点速查

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/health` | 免认证 | 健康检查 |
| POST | `/api/login` | 免认证 | 登录 |
| POST | `/api/logout` | 免认证 | 登出 |
| GET | `/api/me` | 认证 | 当前用户信息 |
| GET | `/api/agents` | 认证 | 在线座席 |
| POST | `/api/chat` | 认证 | SSE 流式聊天 |
| POST | `/api/chat/sync` | 认证 | 同步聊天 |
| GET | `/api/chat/history` | 认证 | 聊天历史 |
| POST | `/api/chat/clear` | 认证 | 清空会话 |
| GET | `/api/info` | 认证 | 服务信息 |
| GET | `/api/config` | 认证 | 读取配置 |
| PUT | `/api/config` | 管理员 | 更新配置 |
| POST | `/api/config/test-models` | 管理员 | 测试模型连接 |
| GET | `/api/vdb` | 认证 | 我的知识库 |
| GET | `/api/vdb/pub` | 认证 | 公开知识库 |
| POST | `/api/vdb` | 认证 | 创建知识库 |
| DELETE | `/api/vdb/:id` | 认证 | 删除知识库 |
| PUT | `/api/vdb/:id/default` | 认证 | 设为默认 |
| GET | `/api/vdb/:id/files` | 认证 | 文件列表 |
| POST | `/api/vdb/:id/upload` | 认证 | 上传文件 |
| POST | `/api/vdb/search` | 认证 | 搜索知识库 |
| GET | `/api/vdb/file/:id/progress` | 认证 | 处理进度 |
| GET | `/api/vdb/file/:id/chunks` | 认证 | 文件分块 |
| GET | `/api/vdb/file/:id/download` | 认证 | 下载文件 |
| DELETE | `/api/vdb/file/:id` | 认证 | 删除文件 |
| GET | `/api/vdb/bindings` | 管理员 | CSM 绑定 |
| PUT | `/api/vdb/bindings` | 管理员 | 保存 CSM 绑定 |
| GET | `/api/faq` | 认证 | FAQ 列表 |
| POST | `/api/faq/match` | 认证 | FAQ 匹配 |
| GET | `/api/faq/template` | 认证 | FAQ 模板 |
| POST | `/api/faq` | 管理员 | 创建 FAQ |
| POST | `/api/faq/upload` | 管理员 | 上传 FAQ |
| PUT | `/api/faq/:id` | 管理员 | 更新 FAQ |
| DELETE | `/api/faq/:id` | 管理员 | 删除 FAQ |
| DELETE | `/api/faq` | 管理员 | 清空 FAQ |
| GET | `/api/users` | 管理员 | 用户列表 |
| POST | `/api/users` | 管理员 | 创建用户 |
| DELETE | `/api/users/:name` | 管理员 | 删除用户 |
| PUT | `/api/users/:name/reset-pwd` | 管理员 | 重置密码 |
| PUT | `/api/user/password` | 认证 | 修改密码 |
| GET | `/api/user/tokens` | 认证 | 我的 Token |
| POST | `/api/user/token` | 认证 | 生成 Token |
| GET | `/api/user/call-logs` | 认证 | 调用日志 |
| GET | `/api/ai-agents` | 认证 | Agent 列表 |
| GET | `/api/ai-agents/public` | 认证 | Agent 公开列表 |
| POST | `/api/ai-agents` | 认证 | 创建 Agent |
| GET | `/api/ai-agents/:id` | 认证 | Agent 详情 |
| PUT | `/api/ai-agents/:id` | 认证 | 更新 Agent |
| DELETE | `/api/ai-agents/:id` | 认证 | 删除 Agent |
| GET | `/api/workflows` | 认证 | 工作流列表 |
| GET | `/api/workflows/:id` | 认证 | 工作流详情 |
| POST | `/api/workflows` | 管理员 | 创建工作流 |
| PUT | `/api/workflows/:id` | 管理员 | 更新工作流 |
| DELETE | `/api/workflows/:id` | 管理员 | 删除工作流 |
| POST | `/api/classifier/test` | 认证 | 测试意图分类 |
| GET | `/api/system-vars` | 认证 | 系统变量 |

### 通用 curl 模板

```bash
# GET
curl <HOST>/api/<path> -H "Authorization: Bearer <TOKEN>"

# POST/PUT JSON
curl -X POST <HOST>/api/<path> \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"key":"value"}'

# 文件上传
curl -X POST <HOST>/api/<path> \
  -H "Authorization: Bearer <TOKEN>" \
  -F "file=@/path/to/file.txt"

# 登录并保存 Token
TOKEN=$(curl -s <HOST>/api/login \
  -H "Content-Type: application/json" \
  -d '{"user_name":"admin","password":"admin"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])")
```
