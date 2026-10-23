# itsm-cli

ITSM 命令行工具，双表面设计：

- **TUI（交互模式）**：面向人类操作员，Ink/React 渲染的表格、详情和流式反馈。
- **Agent 模式（headless）**：面向 AI Agent，`--json` 切换 —— 每次调用输出**恰好一个 JSON 信封**并以**语义化 exit code** 结束。

两个表面共享同一个 HTTP 传输层（`src/lib/api-client.ts`），不存在两套 API 调用实现。

## Agent 模式

```bash
# 冷启动自检：一次拿到「连得上吗 / 我是谁 / 我能读什么」
itsm --json doctor

# 能力发现：先看有什么命令
itsm --json commands                 # 命令目录（按 domain 分组）
itsm --json manifest                 # 协议清单：exit code、环境变量、输出控制

# 输入契约：调用前先问需要哪些参数（不用猜、不用靠 400 试错）
itsm --json schema ticket.create                 # 必填参数 + 枚举值 + body 提示
itsm --json schema ticket --as detail            # 整个 domain 的完整参数
itsm --json schema --as tools --domain ticket    # function-calling 定义（OpenAI/Anthropic 可用）

# 基础冒烟（无需登录）
itsm --json infra health
itsm --json infra version
itsm --json infra ready

# 登录（三选一）
itsm --json auth login --username admin --password '...'   # 写入 ~/.itsm/credentials
ITSM_TOKEN=<jwt> itsm --json ticket get 42                 # 静态注入，无需登录文件
ITSM_USERNAME=admin ITSM_PASSWORD='...' itsm --json ticket list   # 自动登录（内存）

# 核心 CRUD 与生命周期
itsm --json ticket create '打印机故障' '3楼打印机卡纸无法打印' --priority high
itsm --json ticket list --status open --page 1
itsm --json ticket list --limit 5 --fields id,ticketNumber,status,priority   # 控制上下文体积
itsm --json ticket list --format ndjson --limit 100                          # 大列表流式消费
itsm --json ticket comments add 42 --content '已联系供应商'
itsm --json ticket workflow-accept 42 --comment '接受处理'
itsm --json ticket workflow-approve 42 --approvalId 7 --action approve

# 事件：new → acknowledged → in_progress → resolved → closed（跨状态跳转会 409）
itsm --json incident create '核心交换机丢包' --priority critical
itsm --json incident acknowledge 3
itsm --json incident update 3 --data '{"status":"in_progress","version":0}'   # 无 in-progress 专命令
itsm --json incident resolve 3 --resolution '更换光模块'
itsm --json incident close 3 --closeNotes '已恢复，观察 24h' --yes            # closeNotes 必填

# 问题：三个必填字段给扁平参数，不用手写 JSON
itsm --json problem create '月度对账偶发超时' --description '每月1日批量对账在22:00后偶发超时，需定位根因。' --priority medium

# BPMN 引擎
itsm --json workflow tasks
itsm --json workflow decision <taskId> --action approve --comment '批准'

# 逃生舱：命令面没覆盖的端点
itsm --json infra api GET /sla/stats
itsm --json infra api POST /knowledge/search --data '{"query":"打印机","pageSize":5}'
```

> **多命令场景必须用 `auth login` 落盘那种方式。** 后端登录限流 10 次/分钟/IP（业务码 2003），
> `ITSM_USERNAME/ITSM_PASSWORD` 和 `ITSM_TOKEN` 之外的每进程登录，跑十来条命令就会被 403 挡住。
> `auth login` 写入 `~/.itsm/credentials`（目录 0700 / 文件 0600）后，后续命令零登录开销。
> 令牌生命周期：access 15 分钟、refresh 7 天；请求 401 时 CLI 自动调 `/api/v1/auth/refresh`
> 静默续期并重放一次（refresh token 单次使用，rotation 后的新值会写回凭据）。
> 自动登录失败不再被吞成「缺少认证token」——错误会带真实原因（如 `auto-login failed: HTTP 403 / code 2003`）与绕行建议。
> 任意写命令加 `--dry-run` 可预演：只回显 `{command, risk, endpoints}`，不发请求。

### 退出码协议

| Exit | 含义 | Agent 动作 |
|------|------|-----------|
| `0` | 成功 | 读 `data` |
| `1` | 网络不可达 | 可重试 |
| `2` | 用法错误（未知命令/缺参数） | 修正输入后重试 |
| `3` | 未认证 | `auth login` / 设置 `ITSM_TOKEN` |
| `4` | 无权限 | 报告缺哪个 `resource:action`，找管理员 |
| `5` | 后端拒绝（校验/404/冲突/5xx，看 `error.kind`） | 按 kind 处理 |
| `6` | CLI 内部错误 | 提 issue |
| `10` | **confirmation_required** | 展示风险 → 用户同意后**同命令追加 `--yes`** 重试 |

### exit 10 确认门禁（飞书 CLI 模式）

删除/终止/回滚等高风险写操作不会直接执行：

```bash
$ itsm --json ticket delete 42
{"ok":false,"error":{"kind":"confirmation_required","hint":"... retry with --yes","risk":{...}}}   # exit 10
$ itsm --json ticket delete 42 --yes    # 用户同意后执行
```

Agent 收到 exit 10 **禁止静默重试**；必须先向用户展示 `risk` 内容获得明确同意。

### 输出信封

```json
{ "ok": true, "data": { ...后端 data 原样透传... } }
{ "ok": false, "error": { "kind": "forbidden", "message": "...", "httpStatus": 403, "retryable": false, "hint": "..." } }
```

- 成功时 `data` 是后端 `{code,message,data}` 信封中 `data` 字段的原样 JSON（CLI 不做字段猜测或二次包装）。
- 失败时 `kind` ∈ `network | usage | not_authenticated | forbidden | not_found | conflict | validation | unavailable | confirmation_required | internal`。
- 供应商/SQL 原始错误文本在 CLI 侧已脱敏（`sanitizeProviderMessage`）。
- stdout 只承载这一个 JSON 文档：agent 模式下 `console.log/info/debug` 全部重定向到 stderr，
  避免任何调试输出或第三方库日志污染协议流。

### 输出体积控制（token 预算）

列表接口一次回几十条完整对象，直接进 context 会烧掉大量 token。三个 flag 在协议层做裁剪，
不需要写 jq：

```bash
itsm --json ticket list --limit 5                          # 只保留前 5 条
itsm --json ticket list --fields id,ticketNumber,status    # 只保留指定字段
itsm --json ticket list --fields id,assignee.name          # 支持点号路径
itsm --json ticket list --format ndjson --limit 100        # 一行一条，末尾一行 {_meta}
```

裁剪后信封会多一个 `meta`，明确告诉 agent 看到的是不是全量：

```json
{"ok":true,"data":{"items":[...5...],"total":137,...},
 "meta":{"returned":5,"total":137,"truncated":true,"projected":["id","status"]}}
```

`--fields` 请求的字段若不存在，会显式给 `null`（而不是悄悄缺失），避免 agent 把「字段没返回」
误判成「值为空」。只作用于列表载荷，单对象响应不会被裁剪——若对非列表载荷传了这两个 flag，
信封会带 `meta.shapingSkipped` 说明未生效，而不是假装已裁剪。

### 输入契约发现（schema）

Agent 最大的浪费是猜参数。`schema` 从命令注册表 + 后端 DTO 校验标签派生，无需手写第二份文档：

```bash
itsm --json schema                          # 全量摘要：命令 / 风险 / 必填参数（~15KB）
itsm --json schema ticket                   # 某个 domain 的完整参数（含类型、枚举、默认值）
itsm --json schema ticket.create            # 单命令精确契约
itsm --json schema --as tools --domain cmdb # function-calling 工具定义
```

参数来源有两层：

1. **usage 字符串解析**：`<title>` → 必填 positional，`[--status s]` → 可选 flag，
   `--data <json>` → body。190+ 命令零手工维护全覆盖。
2. **后端 `oneof=` 枚举**：`scripts/gen-enums.mjs` 扫描 `itsm-backend` 的 DTO，
   只导出**跨域语义一致**的字段（如 `severity`/`impact`/`priority`）。
   `status`/`type`/`action` 这类每个域含义不同的字段刻意不导出——**宁可没有，也不给错的值**。

`--as tools` 的每个工具都带 `x-itsm` 映射（`command`、positional 顺序、`bodyFlag`、`endpoints`），
宿主可以把模型参数还原成 CLI 调用；高风险工具会在 description 里标注 `[HIGH RISK: requires --yes]`。

### 冷启动自检（doctor）

```bash
itsm --json doctor              # 连通性 + 版本 + 身份 + 7 个领域的读权限探针（并行）
itsm --json doctor --skipProbes # 只看连通性与身份
```

返回 `verdict` ∈ `healthy | degraded | unreachable | unauthenticated`，每个失败检查带
`kind` 与 `hint`；诊断本身是结果，因此**能产出报告时总是 exit 0**。
`degraded` 表示身份有效但某些领域被 RBAC 拒绝——agent 应读失败项再决定是否继续。

### 业务流程验收（flow）

单条命令只能证明某个端点可用，不能证明「工单能走完生命周期」。`flow` 把跨端点流程写成可执行验收场景，
复用同一个 `ApiClient` 调真实后端，输出机器可判定的 verdict：

```bash
itsm --json flow list                       # 场景目录
itsm --json flow show ticket-lifecycle      # 运行前先看步骤/端点/断言/负向用例
itsm --json flow run ticket-lifecycle       # 执行；默认 teardown 清理自建数据
itsm --json flow run change-lifecycle --keep --report /tmp/change.json
itsm --json flow run ticket-lifecycle --var assigneeId=7   # 覆盖场景变量
```

verdict 三态：

| verdict | 含义 | exit |
|---------|------|------|
| `passed` | 所有正向断言与负向断言都成立 | 0 |
| `failed` | 产品行为与契约不符（含「负向用例没有被拒绝」） | 5 |
| `blocked` | 环境/治理前置不满足（未绑定 BPMN、需要第二评审人、缺少目录项），既不能算通过也不能算产品失败 | 5 |

- **失败即停**：后续步骤依赖前一步的 ID，继续跑只会产生噪声；`teardown` 始终执行（反序），
  失败只记 `warnings`，不会掩盖真正的失败点。
- `context` 是本次创建对象的 ID，`runTag`（`agent-YYYYMMDDHHMMSS-xxxx`）嵌入标题，便于追溯；
  `--keep` 跳过清理，`--report <path>` 落盘完整证据。
- 场景声明的 endpoint 会并入 `flow.run` 的命令契约，因此 `check:route-drift` 同样覆盖流程用到的路由。
- **负向用例是一等公民**：非法状态迁移、跨租户过期资源、缺失 CI 必须以声明的 `kind`/`httpStatus`
  被拒绝。负向步骤收到 500（而不是 404/409）判 `failed`——那说明错误分类坏了，不能降级成 `blocked`。
- 只写自建数据：场景只创建/删除自己带 `runTag` 的对象，不修改既有业务数据。

当前 `blocked` 的两种典型原因都是真实环境约束，不是 CLI 缺陷：知识发布要求「作者不能自审」，
需要第二个评审身份；服务请求审批要求目录项绑定 BPMN 流程。

### 超时、重试与幂等边界

| Flag | 默认 | 行为 |
|------|------|------|
| `--timeout <ms>` | 30000 | 单请求硬超时，超时 → exit 1，hint 指路 `--timeout`/`doctor` |
| `--retries <n>` | 2 | 仅对**幂等**请求（GET/HEAD）自动重试；429/5xx 尊重 `Retry-After` |
| `--no-retry` | — | 关闭自动重试 |

写请求**永不自动重试**：后端没有 HTTP 层幂等键，盲目重放可能重复创建。
写超时后应先读一次状态（`ticket get`/`list`）确认是否已生效，再决定是否重发。

### 环境变量

| 变量 | 作用 |
|------|------|
| `ITSM_BASE_URL` | 覆盖后端地址（默认 `http://localhost:8090`） |
| `ITSM_TOKEN` | Agent 身份静态注入（优先级最高） |
| `ITSM_USERNAME` / `ITSM_PASSWORD` | 无 token 时的自动登录（进程内存，不落盘） |
| `ITSM_CLI_AGENT_NO_GUARD` | 设为 `1` 关闭 stdout 守卫（仅调试用，会允许 stray log 污染协议流） |

认证链的优先级：`--base-url` > 环境变量 > `~/.itsm/credentials`。

> 注意：后端登录接口把 JWT 放在 **HttpOnly cookie**（`access_token`/`refresh_token`）里，
> 响应体无 token 字段。CLI 从 `Set-Cookie` 提取后写入 credentials 或内存。

## 命令面速览

完整目录以 `itsm --json commands` 为准（208 命令，直接从代码注册表生成）。主要 domain：

`infra`（健康/版本/能力/全局搜索/原始API/**doctor 自检**） · `auth`（login/status/switch-tenant） · `ticket`（CRUD + workflow 状态机 + 评论 + AI） · `incident` · `problem`（含 known-errors） · `change`（全生命周期） · `release` · `request`/`catalog`（服务请求） · `cmdb`（CI/关系/拓扑/影响分析） · `knowledge` · `sla` · `workflow`（BPMN） · `notification`（含投递审计） · `connector`（集成市场） · `admin`（用户/租户/RBAC/审计/MSP）

`itsm --json doctor` 是 `infra.doctor` 的别名，注册表仍是唯一事实来源。

## 维护约定

### 一条命令跑完全部检查

```bash
cd itsm-cli && npm run check
# = typecheck + 枚举表同步 + 路由漂移 + 对外流程验收 + 25 条协议/流程回归测试
```

### 路由漂移检查

CLI 端点清单必须与后端 `router/` + `handlers/*/RegisterRoutes` 逐一对应：

```bash
cd itsm-cli && npm run check:route-drift
```

后端是 URL 契约的唯一事实来源；该脚本静态扫描后端全部路由注册（~900 条），
任何 CLI 引用了未注册的端点都会 fail。改路由时两边在同一变更里更新。

### 枚举表同步

```bash
cd itsm-cli && npm run gen:enums     # 从后端 DTO oneof= 重新生成
cd itsm-cli && npm run check:enums   # CI：表过期则 fail
```

生产物是 `src/agent/enums.generated.ts`（勿手改）。白名单只收录跨域语义一致的字段；
`status`/`type`/`action`/`provider`/`channel` 因每个域含义不同而被排除。

### 协议回归测试

```bash
cd itsm-cli && npm test
```

`tests/agent-protocol.test.mjs` 用**进程内 stub 后端**跑真实 `bin/itsm.js`，断言：
单一 JSON 信封、`--limit/--fields/--format ndjson` 裁剪与 `meta`、exit 0/1/2/3/4/5/10 全映射、
超时、幂等 GET 重试、高风险写无 `--yes` 时**零请求**、`--dry-run` 不发请求、
`--help` 不被风险门禁拦截、`schema` 必填参数与枚举、`doctor` 报告。

`tests/flow-runner.test.mjs` 用合成场景驱动流程 runner，锁定判定语义：capture 上下文、
失败即停、teardown 必执行且反序、`expectError` 只接受声明的拒绝、负向步骤收到 5xx 判 `failed`
而非 `blocked`、`blockedWhen` 降级、`--keep` 跳过清理、teardown 失败只产生 warning。

### Agent 表面冒烟检查

```bash
cd itsm-cli && npm run smoke:agent              # 本地协议 + 在线探活
cd itsm-cli && npm run smoke:agent -- --offline # 只跑不依赖后端的协议断言
```

锁定外部 agent 依赖的契约：`manifest` 协议字段、命令目录域数、未知命令 exit 2、
`--help` 不被 exit 10 风险门禁拦截（且 usage 里有必填参数）、`problem create` 本地校验、
`--dry-run` 预演、高风险写无 `--yes` 必 exit 10、`infra health` 可达、有凭据时认证读可用。
改了 `--json` 协议、风险门禁或命令必填参数时先跑它。

### 启动开销

`bin/itsm.js` 在 `--json` 分支**不导入** Ink/React/meow，只加载 `dist/agent/runner.js`。
早期版本顶层静态导入 TUI 依赖，单次 agent 调用约 4.7s；现在约 1.0s（本机 `node -e 0` 即 ~0.7s）。
新增 agent 依赖时不要在 `bin/itsm.js` 顶部静态导入——会直接吃掉这个优化。

### 新增命令

1. 在 `src/lib/api-client.ts` 加传输方法（路径必须是静态字面量）。
2. 在 `src/agent/commands/<domain>.ts` 注册 `cmd(...)`（name、summary、usage、endpoints、run）。
3. `risk` 由动词自动推导（`delete/cancel/terminate/...` → high-risk-write，自动获得 exit 10 门禁）。
4. 参数契约从 `usage` 自动派生；需要枚举/默认值/示例时传第 7 个参数 `meta`（`params`/`examples`/`bodyFields`）。
   枚举若属于后端 `oneof=` 且跨域一致，优先走 `gen:enums`，不要手写。
5. 请求体字段以 `itsm-backend/handlers/<domain>/` 的 DTO JSON tag 为准（camelCase）。
6. **有必填 body 的 POST，即使命令无参数也要发 `{}`**：后端用 `ShouldBindJSON` 绑定，空 body 会 EOF → 400「请求参数错误」（`incident close` 曾因此恒 400）。
7. 新增 domain 或命令后跑 `npm run check`（路由漂移 + 枚举同步 + 协议测试）。
8. 新增业务流程验收：在 `src/agent/flow/scenarios/<domain>.ts` 定义步骤（正向断言 + 负向 `expectError`），
   在 `src/agent/flow/registry.ts` 注册；治理/环境限制用 `blockedWhen` 显式表达，不要把它写成 `expectError` 绕过检查。
