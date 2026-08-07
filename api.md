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
{"status":"ok","token":"eyJ...","user":{"uid":1,"user_name":"admin","role":1,"note":"内置管理员"}}
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
{"data":{"uid":1,"user_name":"admin","role":1,"note":""}}
```

### 查询在线座席

`GET /api/agents`

```bash
curl <HOST>/api/agents -H "Authorization: Bearer <TOKEN>"
```

```json
{"data":[{"user_name":"person0","login_at":"2026-08-07T10:30:00Z"}]}
```

---

## 2. 对话

### 查询历史

`GET /api/chat/history`

```bash
curl <HOST>/api/chat/history -H "Authorization: Bearer <TOKEN>"
```

```json
{"data":[{"role":"user","content":"你好"},{"role":"assistant","content":"您好！请问有什么可以帮您？"}]}
```

> 每个用户按 `uid` 独立存储，最多保留最近 5 轮（10 条）。

### 发送消息

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

## 3. 管理配置

### 获取配置

`GET /api/config`

```bash
curl <HOST>/api/config -H "Authorization: Bearer <TOKEN>"
```

```json
{
  "data": {
    "sys": {"name":"对话机器人","auth":"false","api_auth":"true","work_mode":0,"default_workflow_id":0},
    "api": {"llm_api_uri":"https://...","llm_api_key":"sk-...","llm_model_name":"gpt-4"},
    "prompt": {"chat_msg":"你是专业的对话机器人..."},
    "kb": {"chunk_size":300,"chunk_overlap":80,"top_k":3,"score_threshold":0.1,"rerank_enabled":false,"rerank_retrieve_n":15},
    "llm": {"temperature":0.7,"top_p":0.9,"max_tokens":2048},
    "faq": {"match_threshold":0.85}
  }
}
```

### 更新配置

`PUT /api/config`

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

`POST /api/config/test-models`

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

## 4. 管理知识库

### 查询我的知识库

`GET /api/vdb`

```bash
curl <HOST>/api/vdb -H "Authorization: Bearer <TOKEN>"
```

```json
{"data":[{"id":1,"name":"燃气知识库","uid":"admin","is_public":0,"is_default":1,"created_at":"..."}]}
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
{"data":{"id":3}}
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
{"data":{"id":1}}
```

### 搜索知识库

`POST /api/vdb/search`

```bash
curl -X POST <HOST>/api/vdb/search \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"query":"燃气费怎么算","vdb_ids":[1]}'
```

```json
{"data":[{"id":"doc_1","content":"阶梯气价...","score":0.95,"source":"燃气价格表.txt","vdb_id":1}]}
```

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

### 查询知识库绑定

`GET /api/vdb/bindings`

```bash
curl <HOST>/api/vdb/bindings -H "Authorization: Bearer <TOKEN>"
```

```json
{"data":{"billing":[1,2],"repair":[3],"faq":[1,3]}}
```

### 保存知识库绑定

`PUT /api/vdb/bindings`

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

## 5. 管理 FAQ

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

### 下载 FAQ 模板

`GET /api/faq/template`

```bash
curl <HOST>/api/faq/template -H "Authorization: Bearer <TOKEN>" -o faq_template.txt
```

### 创建 FAQ

`POST /api/faq`

```bash
curl -X POST <HOST>/api/faq \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"answer":"营业时间周一至周五 8:00-18:00","questions":["营业时间","几点开门"]}'
```

```json
{"status":"ok","id":1}
```

### 上传 FAQ 文件

`POST /api/faq/upload`

```bash
curl -X POST <HOST>/api/faq/upload \
  -H "Authorization: Bearer <TOKEN>" \
  -F "file=@faq.txt"
```

```json
{"status":"ok","count":15}
```

### 更新 FAQ

`PUT /api/faq/:id`

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

`DELETE /api/faq/:id`

```bash
curl -X DELETE <HOST>/api/faq/1 -H "Authorization: Bearer <TOKEN>"
```

```json
{"status":"ok"}
```

### 清空 FAQ

`DELETE /api/faq`

```bash
curl -X DELETE <HOST>/api/faq -H "Authorization: Bearer <TOKEN>"
```

```json
{"status":"ok"}
```

---

## 6. 管理用户

### 查询用户列表

`GET /api/users`

```bash
curl <HOST>/api/users -H "Authorization: Bearer <TOKEN>"
```

```json
{"data":[{"uid":1,"user_name":"admin","role":1,"note":"内置管理员"}]}
```

### 创建用户

`POST /api/users`

```bash
curl -X POST <HOST>/api/users \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"user_name":"new_user","password":"123456","role":0,"note":""}'
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

`DELETE /api/users/:name`

```bash
curl -X DELETE <HOST>/api/users/new_user -H "Authorization: Bearer <TOKEN>"
```

```json
{"status":"ok"}
```

### 重置密码

`PUT /api/users/:name/reset-pwd`

```bash
curl -X PUT <HOST>/api/users/admin/reset-pwd -H "Authorization: Bearer <TOKEN>"
```

```json
{"status":"ok","new_password":"admin"}
```

---

## 7. 用户自助

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
{"data":[{"id":1,"token_preview":"eyJ...XXX","expires_at":"2026-09-06T...","created_at":"..."}]}
```

### 生成 Token

`POST /api/user/token`

```bash
curl -X POST <HOST>/api/user/token \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"hours":720}'
```

```json
{"token":"eyJhbGciOi...","token_preview":"eyJhbG...xxx","expires_at":"2026-09-06T10:00:00Z"}
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

## 8. 管理 Agent

### 查询公开 Agent

`GET /api/ai-agents/public`

```bash
curl <HOST>/api/ai-agents/public -H "Authorization: Bearer <TOKEN>"
```

```json
{"data":[{"id":1,"name":"通用客服","description":"默认智能体","model_name":"gpt-4"}]}
```

### 查询全部 Agent

`GET /api/ai-agents`

```bash
curl <HOST>/api/ai-agents -H "Authorization: Bearer <TOKEN>"
```

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

## 9. 管理工作流

### 查询公开工作流

`GET /api/workflows`

```bash
curl <HOST>/api/workflows -H "Authorization: Bearer <TOKEN>"
```

```json
{"data":[{"id":1,"name":"燃气客服工作流","description":"处理燃气客服咨询"}]}
```

### 创建工作流

`POST /api/workflows`

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
      {"id":"classify","type":"classify","agent_id":0,"next":["bill","busi","repair","faq"]},
      {"id":"bill","type":"agent","agent_id":1,"next":[],"condition":"billing","final":true},
      {"id":"busi","type":"agent","agent_id":1,"next":[],"condition":"business","final":true},
      {"id":"repair","type":"agent","agent_id":1,"next":[],"condition":"repair","final":true},
      {"id":"faq","type":"agent","agent_id":1,"next":[],"final":true}
    ]
  }'
```

```json
{"status":"ok","id":1}
```

### 查询工作流详情

`GET /api/workflows/:id`

```bash
curl <HOST>/api/workflows/1 -H "Authorization: Bearer <TOKEN>"
```

### 更新工作流

`PUT /api/workflows/:id`

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

`DELETE /api/workflows/:id`

```bash
curl -X DELETE <HOST>/api/workflows/2 -H "Authorization: Bearer <TOKEN>"
```

```json
{"status":"ok"}
```

---

## 10. 测试意图分类

`POST /api/classifier/test`

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

---

## 11. 查询系统变量

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

| role | 说明 |
|------|------|
| 0 | 普通用户 |
| 1 | 管理员 |
| 2 | 客服座席 |
| 3 | API 用户 |

### 工作模式

| work_mode | 路由逻辑 |
|-----------|---------|
| 0 | FAQ 匹配 → 知识库检索 → LLM |
| 1 | 意图分类 → 按意图路由 → 检索(可选) → LLM |
| 2 | 从 `workflow_def` 表加载 DAG 配置执行 |

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
  | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
```
