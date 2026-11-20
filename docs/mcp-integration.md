# MCP 接入指南（CloudMesh ITSM）

> 状态：可用（能力底座已交付并通过端到端验证）；对外口径「持续更新」，未宣称 GA。
> 协议版本：MCP `2025-06-18`　端点：`POST /api/v1/mcp`　传输：JSON-RPC 2.0 over HTTP

本指南说明如何让外部 Agent 客户端（Claude Desktop / Claude Code / 第三方编排器）连接 CloudMesh ITSM 的 MCP 通道，发现并调用 ITSM 运维能力。

---

## 1. 架构前提：MCP 是协议出口，不是第二套权限体系

MCP 通道**不新增业务能力**，只把 ITSM 既有能力换一种协议形状暴露出去：

| 环节 | 复用的既有实现 | 判定职责 |
|---|---|---|
| 工具面 | `internal/services.ToolRegistry` | 工具清单与入参 schema |
| 执行链 | `ai.Service.ExecuteTool` | Gate 2 角色 RBAC + Gate 2.5 主体零信任 + 写工具审批 + 审计落库 |
| 交付门禁 | `marketplace.SkillInvocationEntitlement` | 目录上架 + 租户安装才可见/可调 |

因此接入方**不需要理解内部实现**，只需满足认证与授权两条前提。

---

## 2. 前置条件

- 已部署的 CloudMesh ITSM（后端可达，端口/域名按你的部署定）
- 一个**有效租户**（`tenant_id > 0`）
- `JWT_SECRET` 环境变量（签发令牌用，须与后端一致）
- 客户端支持 MCP 的 HTTP 传输并可配置自定义请求头

---

## 3. 签发访问令牌

### 3.1 人类主体（推荐用于桌面客户端调试）

人类主体走正常登录流程，用 `POST /api/v1/auth/login` 拿 access token，调用时放 `Authorization` 头：

```
Authorization: Bearer <access_token>
```

角色须持 `mcp:read` 权限码（`sysadmin` / `agent` / `audit_admin` 默认持有；`end_user` 不持有）。

### 3.2 服务主体（推荐用于无人值守的 Agent 服务）

服务主体是**进程身份**而非登录用户，JWT 里 `user_id` 恒为 0，权限来自签发时固定的只读载体角色 `audit_admin`。

```bash
cd itsm-backend
export JWT_SECRET='<与后端一致的 JWT_SECRET>'
TOKEN=$(go run ./cmd/service-token-gen \
  -tenant-id 1 \
  -username svc-mcp-client \
  -ttl 8760h)
```

签发工具的硬约束（防止误配提权）：

- `-tenant-id` 必须为正数（令牌绑定固定租户，**禁自报**）
- 载体角色固定为 `audit_admin`（全只读），传入其他角色会被拒绝
- `JWT_SECRET` 缺失直接退出
- stdout **只有令牌本身**，便于 shell 注入；诊断信息走 stderr

### 3.3 放行服务主体调用具体工具（必做）

服务主体默认**零权限**：运行期策略的 `allowedPrincipals` 恒为 `["human"]`，即 AI/服务主体对所有权限码默认无权。

调用某个工具，需要管理员在权限策略里把该工具对应的 `resource:action` 显式放行给 `service`：

```
# 例：放行 incident:read 给 service 主体
# 通过管理端「权限码运行期策略与 AI 主体授权」界面操作：
#   选中 incident:read → 主体授权 → 勾选 service
```

**放行粒度是「按工具码」而非「按通道」**：可以只放行只读工具（`incident:read`、`cmdb:read`），写工具（`cmdb:write` 等）仍保持拒绝。写工具即使放行，调用时仍会进入人工审批流。

未放行时，调用返回 `isError=true`，错误信息形如：

```
主体 service 未获授权执行 incident:read，请检查该权限码的 allowedPrincipals 策略
```

> 🔴 **改完策略不是立刻生效**：权限判定有两层 5 分钟 TTL 缓存
> （`middleware` 的角色权限缓存 + `internal/authz` 的运行期策略缓存）。
> 改完 `allowedPrincipals` 后**最长需等 5 分钟**才会生效，表现为「明明放行了
> 还是被拒」。这不是 bug，是缓存窗口；排障时先确认是否刚改过策略。
>
> 🔴 **存量部署必须先播种 `mcp:read`**：本权限码是 2026-11-18 新增的，
> 而生产 compose 的 `ITSM_AUTO_SEED=false` —— seeder 不在每次启动跑，
> **新码不会自动进入已存在的 `permissions` 表**。未播种时端点虽已注册，
> 但任何角色都会被判 403「权限不足」（DB 里查不到该码）。
> 升级存量部署须执行迁移：
>
> ```bash
> psql "$DSN" -v ON_ERROR_STOP=1 -f itsm-backend/migrations/20261118_mcp_read_permission.sql
> ```
>
> 该迁移幂等，按租户插入权限码并给 `agent`/`audit_admin` 建立关联。

---

## 4. 配置客户端

### 4.1 Claude Desktop / Claude Code（stdio 转发）

多数桌面客户端以 stdio 方式启动 MCP server，需要用本地转发进程把 stdio 转成 HTTP。示例（以 `mcp-remote` 为例）：

```json
{
  "mcpServers": {
    "cloudmesh-itsm": {
      "command": "npx",
      "args": [
        "-y", "mcp-remote",
        "https://itsm.example.com/api/v1/mcp",
        "--header",
        "Authorization: Bearer <TOKEN>"
      ]
    }
  }
}
```

### 4.2 直接用 HTTP（自研客户端）

```bash
curl -sS -X POST https://itsm.example.com/api/v1/mcp \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
      "protocolVersion": "2025-06-18",
      "capabilities": {},
      "clientInfo": {"name": "my-agent", "version": "1.0.0"}
    }
  }'
```

---

## 5. 协议交互

### 5.1 生命周期

```
initialize  →  （客户端发 notifications/initialized）  →  tools/list / tools/call
```

版本协商：客户端报自己支持的 `protocolVersion`；服务端不支持时回自己支持的版本，由客户端决定是否断开。

### 5.2 列出工具面

```json
{"jsonrpc":"2.0","id":2,"method":"tools/list"}
```

响应：

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "tools": [
      {
        "name": "get_incident_stats",
        "description": "获取当前租户的事件统计",
        "inputSchema": {"type": "object", "properties": {}},
        "annotations": {"readOnlyHint": true, "destructiveHint": false}
      }
    ]
  }
}
```

**工具面按租户动态收敛**：目录未上架或租户未安装的工具既不出现，直接点名调用也会被拒。

### 5.3 调用工具

```json
{"jsonrpc":"2.0","id":3,"method":"tools/call",
 "params":{"name":"get_incident_stats","arguments":{}}}
```

响应（成功）：

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [{"type": "text", "text": "{\n  \"open\": 3\n}"}],
    "structuredContent": {"open": 3}
  }
}
```

响应（写工具：**已受理、待人工审批，尚未执行**）：

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [{"type": "text", "text": "status=pending_approval\ntool=create_ticket\ninvocation_id=42\nmessage=该工具会产生业务副作用，已提交人工审批，尚未执行。审批通过后由服务端执行器落地。\nnext=请勿声称操作已完成；用 invocation_id 查询审批结果，approved 后才代表已执行。"}],
    "_meta": {"invocationId": 42, "status": "pending_approval", "pollHint": "GET /api/v1/agent/tools/42"}
  }
}
```

> **写工具的回执不是执行结果。** 写工具恒进人工审批流，`tools/call` 返回的是「受理」回执：
> - 正文带 `status=pending_approval`，且 `_meta.status` 同值——客户端据此判定「未执行」，不得宣告操作完成；
> - **不填 `structuredContent`**：工具声明的 `outputSchema` 描述真实产出，待审批不是产出；
> - **不是 `isError`**：受理属正常业务流，标成错误会诱发客户端重试或告警；
> - 用 `_meta.invocationId` 或 `_meta.pollHint` 查询审批进展；最终状态是 `approved`（已批准，副作用随后由执行器落地）或 `rejected`（已驳回，不会执行）。
>
> 部分主体（如 `agent` / `service`）的审批强度由运行期策略决定：高风险工具可能需要**两名不同审批人**依次批准，首票只进入 `awaiting_second`，仍不放行。发起人不得自批。

响应（业务失败，如权限拒绝）：

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [{"type": "text", "text": "主体 service 未获授权执行 incident:read..."}],
    "isError": true
  }
}
```

> 协议错误（未知方法、参数非法）用 JSON-RPC `error` 对象表达；**业务失败（权限拒绝、工具报错）用 `isError: true`**——这是 MCP 规范对两类失败的区分。
>
> HTTP 状态码恒为 200：失败信息在 body 里。客户端不要只看 HTTP 码。

---

## 6. 安全说明

| 项 | 行为 |
|---|---|
| 认证 | `Authorization: Bearer`；未认证 401 |
| 通道准入 | 路由级 `mcp:read`；无此权限 403 |
| 主体准入 | 运行期策略 `allowedPrincipals`（默认仅 human） |
| 动作权限 | 执行链按工具 `resource:action` 判定（Gate 2） |
| 租户隔离 | 工具面与执行均按 `tenant_id` 收敛，令牌绑定固定租户 |
| 写操作 | 恒进人工审批流，不自动执行；`tools/call` 返回 `status=pending_approval` 受理回执（非执行结果） |
| 审计 | 调用落 `ToolInvocation`（含角色快照与审批状态） |
| CSRF | Bearer 认证自动豁免（无浏览器 cookie 语义）；cookie 会话仍需 CSRF token |
| 传输 | 生产须走 HTTPS |

---

## 7. 排障

| 现象 | 原因 | 处理 |
|---|---|---|
| 401「用户不存在」 | service 主体走了用户表查询路径 | 确认用的是 `cmd/service-token-gen` 签发的令牌（非随机串） |
| 401「token已失效」 | 令牌被吊销或过期 | 重新签发 |
| 403 | 角色不持 `mcp:read` | 换角色或给该角色加 `mcp:read` |
| `isError` 且提示未获授权 | 服务主体未被放行该工具码 | 按 §3.3 放行对应 `resource:action` |
| 工具面为空 | 工具未上架或租户未安装 | 在应用市场安装对应组件 |
| 404 | 端点未注册 | 确认 MCP handler 已装配（bootstrap 装配失败会 Fatalw） |

---

## 8. 客户端验收脚本

服务端自测（`go test`）覆盖协议形状，但**真实客户端还有额外要求**——官方 SDK 的
zod schema 会校验 `tools/list` 响应（例如 `outputSchema.type` 必须是字面量
`"object"`），不合规时**整个工具面加载失败**，服务端契约测试发现不了这类问题。

仓库提供用官方 SDK（Claude Desktop / Claude Code 背后的同一套）跑的验收脚本：

```bash
npm i @modelcontextprotocol/sdk@1.21.0
node scripts/acceptance/mcp-client-acceptance.mjs \
  https://itsm.example.com/api/v1/mcp "$TOKEN"
```

覆盖：initialize 握手与版本协商、tools/list 的 SDK schema 校验
（`inputSchema` 与 `outputSchema` 必须 object 型）、tools/call 真实执行、
未知工具的错误语义、ping。退出码 0 表示「真实 MCP 客户端可用」。

> 2026-11-18 首次跑这个脚本时，它抓出过一个真缺陷：`outputSchema.type`
> 为 `"array"` 的工具会让官方 SDK 在 zod 校验阶段拒绝**整个** tools/list
> 响应 —— 不是个别工具失效，是整条通道在真实客户端里不可用。修复方式是把
> 非 object 型结果 schema 包成 `{"type":"object","properties":{"items":...}}`，
> 实际返回的 `structuredContent` 按同一形状给出。

## 9. 当前边界（诚实说明）

- **协议**：实现了 `initialize` / `ping` / `tools/list` / `tools/call`；**未实现** `resources`、`prompts`、`completions`、`notifications`，客户端不应依赖它们。
- **传输**：仅 HTTP POST 单端点；**未实现** SSE 流式推送与 stdio 原生进程模式（需用 `mcp-remote` 之类的转发器）。
- **能力**：工具面等于 ITSM 既有的 AI 工具集，不含 MCP 专属的新工具。
- **未完成**：调用限额/计费读面、管理端可配的服务主体放行界面。
- **已验证**（2026-11-18）：用官方 MCP SDK 1.21.0 跑通完整协议交互（initialize / tools/list 19 个工具全部通过 SDK schema 校验 / tools/call / 未知工具错误语义 / ping），真实实例连生产 Postgres。
- **未验证**：未在 Claude Desktop GUI 里做端到端点击验证；未做 SSE 流式推送与 stdio 原生进程模式（客户端需用 stdio→HTTP 转发器）；未做多租户并发的权限隔离验证。

---

## 10. 相关文件

| 文件 | 作用 |
|---|---|
| `itsm-backend/handlers/mcp/protocol.go` | JSON-RPC 2.0 + MCP 方法集类型定义 |
| `itsm-backend/handlers/mcp/toolprojection.go` | 工具定义 → MCP tool schema 投影 |
| `itsm-backend/handlers/mcp/service.go` | 协议编排 + 交付门禁 + 执行委托 |
| `itsm-backend/handlers/mcp/handler.go` | HTTP 传输层 |
| `itsm-backend/internal/bootstrap/mcp_assembly.go` | 装配入口（生产与测试共用） |
| `itsm-backend/cmd/service-token-gen/` | 服务主体令牌签发 |
| `plans/mcp-subscription-commercialization-design.md` | 托管订阅与套餐权益设计 |
| `scripts/acceptance/mcp-client-acceptance.mjs` | 真实 MCP 客户端验收脚本（官方 SDK） |
| `itsm-backend/migrations/20261118_mcp_read_permission.sql` | 存量部署的 `mcp:read` 权限码播种 |
