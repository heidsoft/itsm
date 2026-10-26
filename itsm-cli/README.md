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

# 登录（密码只从 stdin 或 0600 文件读，永不进入 argv）
read -rs ITSM_PW && printf %s "$ITSM_PW" | itsm --json auth login --username admin --password-stdin   # 写入 ~/.itsm/credentials
itsm --json auth login --username admin --password-file ~/.itsm/admin.pw   # 文件必须 chmod 600
ITSM_TOKEN_FILE=~/.itsm/token itsm --json ticket get 42                   # 预发令牌，0600 文件
ITSM_USERNAME=admin ITSM_PASSWORD_FILE=~/.itsm/admin.pw itsm --json ticket list   # 自动登录（内存，不落盘）

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

### 认证与凭据安全

- **密码/令牌不进 argv**：`--password`、`--token` 会被显式拒绝（exit 2），并提示安全通道。可用
  `--password-stdin` / `--password-file`、`--token-stdin`、`--reset-token-stdin` / `--reset-token-file`，
  或 `ITSM_PASSWORD_FILE`、`ITSM_TOKEN_FILE`、`ITSM_RESET_TOKEN_FILE`。
- **TUI 登录改为遮蔽输入**：`itsm login -u admin` 交互输入密码，或 `printf %s "$PW" | itsm login -u admin`。
  旧写法 `-p <password>` 已拒绝，避免 shell history 与 `ps` 泄漏。
- **凭据文件防篡改**：目录 0700、文件 0600；同目录临时文件 + `rename` 原子写；拒绝写入 symlink；
  读取时校验 owner，发现权限过宽自动收紧；`ITSM_CREDENTIALS_FILE` 可指向 CI 的临时路径。
  `auth status` 的 `credentialStore` 会报告 path/mode/owner/symlink 是否安全。
- **明文 HTTP 默认拒绝**：向非 loopback 的 `http://` 发送密码或令牌会以 exit 2 失败；
  loopback 例外用于本地开发，遗留内网需显式 `ITSM_ALLOW_INSECURE_HTTP=1`。URL 内嵌 userinfo 一律拒绝。
- **令牌不出现在任何输出**：`auth login/refresh/status/logout/switch-tenant` 均不返回
  `accessToken` / `refreshToken`，错误体会统一脱敏。
- **登出会吊销服务端会话**：`auth logout` 先带 `Cookie: refresh_token=...` 调后端吊销，再删除本地文件；
  仅清本地用 `--local-only`。`auth refresh` 要求后端轮换出新 refresh token，否则判定失败。

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
   `--data <json>` → body。300+ 命令零手工维护全覆盖。
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
| `ITSM_TOKEN` / `ITSM_TOKEN_FILE` | Agent 身份静态注入；文件版必须是 0600 |
| `ITSM_USERNAME` + `ITSM_PASSWORD_FILE` | 无 token 时的自动登录（进程内存，不落盘；优先文件而非明文变量） |
| `ITSM_CREDENTIALS_FILE` | 覆盖凭据库路径（CI 挂载临时卷时使用） |
| `ITSM_RESET_TOKEN_FILE` | 密码重置令牌文件（0600），替代 argv `--token` |
| `ITSM_ALLOW_INSECURE_HTTP` | 设为 `1` 才允许向非 loopback 的明文 HTTP 发送凭据（遗留内网，不推荐） |
| `ITSM_CLI_AGENT_NO_GUARD` | 设为 `1` 关闭 stdout 守卫（仅调试用，会允许 stray log 污染协议流） |

认证链的优先级：`--base-url` > 环境变量 > `~/.itsm/credentials`。

> 注意：后端登录接口把 JWT 放在 **HttpOnly cookie**（`access_token`/`refresh_token`）里，
> 响应体无 token 字段。CLI 从 `Set-Cookie` 提取后写入 credentials 或内存。

## MCP 服务（itsm-mcp）

同一份命令注册表还以 **MCP**（Model Context Protocol，stdio + JSON-RPC 2.0）暴露给图形化宿主
（WorkBuddy / Claude Desktop / Cursor 等），让没有 shell 的客户端也能查询与操作本实例：

```bash
node bin/itsm-mcp.js             # 进入协议循环，零 npm 依赖（dist 已入库，干净克隆直接可跑）
node bin/itsm-mcp.js --describe  # 只打印工具面，不进协议循环
```

工具面分两层：20 个精选工具（工单读写与状态机流转、事件分诊、变更审批链、CMDB 影响分析、
知识检索、SLA 概览、我的待办、身份自检）+ `itsm_discover` → `itsm_describe` → `itsm_call`
覆盖全部 302 条命令。**schema 由 `CommandSpec.input` 派生**（不新增第二份端点表），
风险门与 `exit 10` 同源：高风险写缺 `confirm` 时**不发请求**，且永不自动重试。
列表默认裁剪 50 条并给出 `total`/`truncated`；resources 提供实时能力清单与 4 篇产品文档，
prompts 提供队列梳理 / 事件复盘 / 变更风险评审三条操作规程。

```bash
node --test tests/mcp-protocol.test.mjs   # 18 例协议回归（桩后端，含"高风险写未发请求"）
node scripts/mcp-smoke.mjs                # 打真实实例的只读冒烟；--offline 只跑本地断言
```

接入配置、安全模型与排障见 [docs/tools/itsm-mcp.md](../docs/tools/itsm-mcp.md)。

## 命令面速览

完整目录以 `itsm --json commands` 为准（332 命令，直接从代码注册表生成）。主要 domain：

`infra`（健康/版本/能力/全局搜索/原始API/**doctor 自检**） · `auth`（登录/刷新/状态/登出/切租户/注册/密码重置/SSO） · `ticket`（CRUD + workflow 状态机 + 评论 + 子任务 + 关系 + SLA + 评分 + AI） · `incident`（含根因/分类/影响评估/重大事件/事件流/报告） · `problem`（含 known-errors） · `change`（全生命周期 + 审批链 + 风险评估 + CMDB 影响 + 回滚清单 + PIR + 变更日历） · `release` · `request`/`catalog`（服务请求与目录 CRUD） · `cmdb`（CI/关系/拓扑/影响分析/标签/版本回退/CI 类型） · `cloud`（云账号/服务/资源 + 发现源/任务/结果 + **deferred 资源人工补全建账**） · `knowledge` · `sla` · `workflow`（BPMN） · `notification`（含投递审计） · `connector`（集成市场） · `admin`（用户/租户/RBAC/审计/MSP/**可靠副作用命令总线**/定时器）

`itsm --json doctor` 是 `infra.doctor` 的别名，注册表仍是唯一事实来源。

## 维护约定

### 一条命令跑完全部检查

```bash
cd itsm-cli && npm run check
# = typecheck + 枚举表同步 + 路由漂移 + flag/positional 声明审计 + 73 条协议/流程/安全回归测试
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

### 声明与实现一致性

```bash
cd itsm-cli && npm run audit:flags         # implementation 读到的每个 flag 必须在 usage 里声明
cd itsm-cli && npm run audit:positionals   # implementation 读到的每个 positional 必须在 usage 里声明
```

两者都从 `dist/agent/registry.js` 派生，与 `itsm schema` / `itsm_describe` 看到的是同一份契约。
运行时还有第三道闸：未声明的 flag 与多余 positional 一律 exit 2（见 `validateFlags` / `validatePositionals`）。

### 产品功能点审查（对着真机跑）

```bash
cd itsm-cli && npm run audit:product           # 报告写入 ../output/itsm-cli-product-audit-<date>.md
cd itsm-cli && npm run audit:product -- --strict  # 有违规即 exit 1（可接 CI 的夜间任务）
```

只读探测（119 条）覆盖全部领域的 list/get/stats + 负向路径 + 协议断言，逐条比对本仓库自己声明的 API 契约：
字段命名（camelCase / 持久化列名泄漏）、零值时间、列表信封 `{items,total,page,pageSize,totalPages}`、
成功信封必须带 data、列表给出的 id 必须能被详情端点取回、能力禁用与日期格式的语义一致性。
写操作只用 `--dry-run` 与 exit-10 闸门断言，不会对后端产生任何写入。

### 协议回归测试

```bash
cd itsm-cli && npm test
```

`tests/agent-protocol.test.mjs` 用**进程内 stub 后端**跑真实 `bin/itsm.js`，断言：
单一 JSON 信封、`--limit/--fields/--format ndjson` 裁剪与 `meta`、exit 0/1/2/3/4/5/10 全映射、
超时、幂等 GET 重试、高风险写无 `--yes` 时**零请求**、`--dry-run` 不发请求、
`--help` 不被风险门禁拦截、`schema` 必填参数与枚举、`doctor` 报告。

`tests/auth-security.test.mjs` 锁定认证安全不变量：`--password` / `--token` 在 argv 上被显式拒绝且零请求、
密码文件必须 0600、凭据库写入为 0600 且拒绝穿过 symlink、令牌不出现在 stdout、
非 loopback 的明文 HTTP 在发送凭据前以 exit 2 拒绝、`ITSM_ALLOW_INSECURE_HTTP=1` 才放行。

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
