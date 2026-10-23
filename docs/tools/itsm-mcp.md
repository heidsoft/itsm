# ITSM MCP Server（itsm-mcp）

`itsm-mcp` 把 ITSM 的命令面暴露成 **MCP（Model Context Protocol）** 工具，使 WorkBuddy、Claude Desktop、Cursor 等任意 MCP 客户端可以直接查询与操作本实例：工单、事件、问题、变更、发布、服务请求、CMDB、知识库、SLA、BPMN 工作流，共 268 条命令。

它同时是「用 Agent 完善产品」的入口：客户端不仅能调 API，还能读 `ROADMAP.md`、`CHANGELOG.md`、API 参考与架构文档（MCP resources），从而在有产品上下文的前提下提出改动，而不是另造一套并行机制。

> 结论先说：**只读能用、写操作有闸、零依赖部署**。写操作的风险闸与 CLI 的 `exit 10` 是同一套判定，不会因为换了入口就绕过去。

---

## 1. 快速接入

### 1.1 WorkBuddy

编辑 `~/.workbuddy/mcp.json`：

```json
{
  "mcpServers": {
    "itsm": {
      "command": "/usr/local/bin/node",
      "args": ["/path/to/itsm/itsm-cli/bin/itsm-mcp.js"],
      "env": {
        "ITSM_BASE_URL": "http://localhost:8090",
        "ITSM_REPO_ROOT": "/path/to/itsm"
      }
    }
  }
}
```

保存后到「连接器管理 → 自定义」里 **信任（Trust）** 该 server 才会生效（不会自动激活）。

### 1.2 Claude Desktop / 其他客户端

配置结构一致：

```json
{
  "mcpServers": {
    "itsm": {
      "command": "node",
      "args": ["/path/to/itsm/itsm-cli/bin/itsm-mcp.js"],
      "env": { "ITSM_BASE_URL": "https://itsm.example.com" }
    }
  }
}
```

### 1.3 手工验证（不装任何客户端）

```bash
printf '%s\n' \
'{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{}}}' \
'{"jsonrpc":"2.0","id":2,"method":"tools/list"}' \
| node itsm-cli/bin/itsm-mcp.js
```

一条命令列全部工具名：

```bash
node itsm-cli/bin/itsm-mcp.js --describe
```

---

## 2. 工具面：三层，而不是 268 个工具

把 268 条命令平铺成 268 个工具会淹没模型：`ticket.update` 与 `ticket.subtasks.update` 之类的近邻工具会互相干扰，`tools/list` 本身就要吃掉上万 token。因此分三层：

| 层 | 内容 | 使用场景 |
|:---|:---|:---|
| **精选工具**（20 个） | `itsm_ticket_list` / `itsm_ticket_get` / `itsm_ticket_create` / `itsm_ticket_update` / `itsm_ticket_comment_add` / `itsm_ticket_transition` / `itsm_incident_*` / `itsm_change_list` / `itsm_change_approvals` / `itsm_cmdb_search` / `itsm_cmdb_impact` / `itsm_knowledge_search` / `itsm_sla_stats` / `itsm_workflow_tasks` / `itsm_whoami` / `itsm_health` / `itsm_search` | 高频路径，带完整 JSON Schema，一次调用到位 |
| **发现三件套** | `itsm_discover` → `itsm_describe` → `itsm_call` | 精选工具未覆盖的动作；参数不靠猜 |
| **其余 245 条** | 经 `itsm_call` 用命令名执行 | 审批链、标签、关联、版本、审计、连接器等 |

少数工具会在端点语义之外补一层默认值，让工具名和实际返回一致：`itsm_workflow_tasks` 默认只返待处理（`status=created`），因为 `GET /bpmn/tasks` 不传 `status` 时连已完成的任务一起返回；显式传 `status` 仍以调用方的值为准，走 `itsm_call` 则完全不过滤。

```text
itsm_discover { query: "审批" }        # 中文可检索，按命令名命中优先排序
itsm_describe { command: "change.approvals" }
itsm_call     { command: "change.approvals", params: { id: "CHG-1" } }
```

工具 schema **由 CLI 的命令注册表派生**（`itsm-cli/src/agent/spec.ts` 的 `CommandSpec.input`），不是第二份手写表。因此新增命令、改参数、改枚举会自动出现在 MCP 里，不会与 CLI 契约漂移。

---

## 3. 安全模型

| 机制 | 行为 | 为什么这样做 |
|:---|:---|:---|
| **风险闸** | `high-risk-write`（删除/终止/回滚/批量/重放等）在 `confirm` 缺失时不执行，返回 `confirmation_required` + 端点/用法/是否不可逆 | 与 CLI `exit 10` 同一判定源；「高风险」不因换入口而降级 |
| **不自动重试** | 写操作永不由服务端重试 | 后端没有幂等键，重试等于重复写入 |
| **工具注解** | `readOnlyHint` / `destructiveHint` / `idempotentHint` | 客户端可据此决定是否要二次确认 |
| **凭据来源** | `ITSM_TOKEN[_FILE]` → `ITSM_USERNAME`+`ITSM_PASSWORD[_FILE]` → `~/.itsm/credentials` | 与 `itsm` CLI 完全一致，无需二次登录 |
| **传输策略** | 明文 HTTP 只允许 loopback（或显式 `ITSM_ALLOW_INSECURE_HTTP=1`） | bearer token 不该穿过不可信网络 |
| **输出纯净** | `console.*` 全部重定向到 stderr，stdout 只承载 JSON-RPC 帧 | 一行 stray log 就会让客户端解析失败 |
| **失败即分类** | `forbidden` / `not_found` / `conflict` / `validation` / `network` 原样透传并附下一步提示 | 避免模型把 403 当 404 追、或对 409 盲目重试 |

列表调用默认只回 50 条，`meta` 给出 `total` / `returned` / `truncated`；需要更多用 `limit`、`page`、`pageSize` 分页取。一次性拉全量是最快的上下文爆掉方式。

---

## 4. 资源与提示词

**Resources**（只读上下文，不消耗工具调用）

| URI | 内容 |
|:---|:---|
| `itsm://capabilities` | 实时能力清单：按域统计命令数与读写比例、当前 `baseURL` / 身份来源 / 仓库根目录 |
| `itsm://docs/roadmap` | `ROADMAP.md`：已落地与下一阶段收敛项 |
| `itsm://docs/changelog` | `CHANGELOG.md`：判断「设计如此」还是「回归」 |
| `itsm://docs/api-reference` | `docs/api-reference.md`：HTTP 契约 |
| `itsm://docs/architecture` | `docs/architecture/overview.md`：系统边界与分层规则 |

文档资源从仓库读取（`ITSM_REPO_ROOT` 可覆盖，默认自动向上查找），单文件上限 64KB，超出部分明确标注已截断——不会把 400KB 的 CHANGELOG 灌进上下文。

**Prompts**（把「怎么做」写进协议，而不是每次靠人重复嘱咐）

| 名称 | 作用 |
|:---|:---|
| `triage_my_queue` | 按 SLA 风险梳理待办（工作流任务 + 工单 + 事件），产出处置顺序 |
| `incident_postmortem` | 从事件本体、关联告警、CMDB 影响面、知识库产出可评审复盘 |
| `change_risk_review` | 审批前把变更内容、审批链状态、影响面、历史依据放在一起评审 |

三条提示词都明确要求「读真实数据、区分事实与推测、不自行审批」，属于流程约束而不是文案模板。

---

## 5. 协议实现范围

| 方法 | 状态 |
|:---|:---|
| `initialize` / `notifications/initialized` | ✅ 版本协商（`2025-06-18` / `2025-03-26` / `2024-11-05`） |
| `tools/list` / `tools/call` | ✅ 含 `structuredContent` + 注解 |
| `resources/list` / `resources/templates/list` / `resources/read` | ✅ 未知 URI 返回 `-32002` |
| `prompts/list` / `prompts/get` | ✅ 缺必填参数返回 `-32602` |
| `ping` / `logging/setLevel` / `completion/complete` | ✅ |
| `sampling/*` / `roots/*` / `elicitation/*` | ➖ 未实现（不需要向客户端反问） |

**为什么不用官方 SDK**：`itsm-cli/dist` 已入库，`bin/itsm.js` 从干净克隆直接可跑（agent 分支零 npm 依赖）。引入 `@modelcontextprotocol/sdk` 会让 MCP 成为唯一需要 `node_modules` 的入口，也让协议行为绑在第三方发版节奏上。协议本身很小：换行分隔的 JSON-RPC，加上十余个方法。读取端同时兼容 LSP 风格的 `Content-Length` 分帧，避免客户端换实现就挂住。

**退出语义**：stdin 关闭不等于工作结束——客户端可能发一批请求立刻关管道。服务端会等所有已接受的请求答完再退出（5 秒上限），否则最后几条调用会被静默丢弃。

---

## 6. 排障

| 现象 | 处置 |
|:---|:---|
| 工具调用返回 `not_authenticated` | 先 `itsm --json doctor` 看身份；再 `itsm auth login` 一次（落盘 0600 并自动续期）。**不要**每条命令都带 `ITSM_USERNAME/PASSWORD`，会撞 10 次/分钟/IP 的登录限流（业务码 2003） |
| `forbidden` | 身份有效但缺权限：让管理员补对应的 `resource:action` 授权，换个身份重试没有意义。注意 prod 权限缓存 TTL 5 分钟，改完权限要让 backend 重启才立刻生效 |
| 连不上后端 | `ITSM_BASE_URL` **不能**带 `/api/v1`；prod 走 nginx 时是 `http://localhost`，直连后端是 `http://localhost:8090` |
| 客户端看不到工具 | 自定义 MCP 需要手动 Trust；`--describe` 能列工具说明二进制本身没问题 |
| 需要看协议细节 | `ITSM_MCP_DEBUG=1`，日志走 stderr，不会污染 stdout |

---

## 7. 验证与回归

```bash
cd itsm-cli
npm run build
node --test tests/mcp-protocol.test.mjs   # 18 例：握手/工具面/风险闸/裁剪/资源/提示词/分帧容错/排空
node scripts/mcp-smoke.mjs                # 打真实实例的只读冒烟（不会写数据）
node scripts/mcp-smoke.mjs --offline       # 只跑本地协议断言
```

`tests/mcp-protocol.test.mjs` 覆盖的硬约束：

- 高风险写**未发请求**（用桩后端计数断言 `DELETE /cmdb/cis/7` 命中 0 次），`confirm: true` 后才恰好 1 次；
- 未声明参数在**发请求前**被拒，并列出可用参数名；
- 列表默认裁剪到 50 条且 `meta.total` 保留全量口径；
- stdout 逐行可解析——出现非 JSON 帧直接判失败；
- 畸形帧返回 `-32700` 后服务仍可继续应答；
- 一批请求 + 立即关 stdin，每条都得到应答。

---

## 8. 已知边界（后续路线）

- **只有 stdio 传输**。远程/多客户端共享需要 streamable HTTP + OAuth，当前不做（本地 MCP 已是 WorkBuddy 的主场景）。
- **身份是进程级**，不按 MCP 会话切换租户。多租户场景请为不同租户各配一个 server 实例（不同 `ITSM_BASE_URL` + 凭据）。
- **无细粒度工具授权**：`tools/list` 按 registry 静态输出，不按调用者权限裁剪。看不到权限的调用会以 `forbidden` 失败（fail-closed），而不是从列表里消失。
- **写操作无幂等键**，因此服务端不重试；需要「至少一次」语义的批量操作应在客户端编排。
