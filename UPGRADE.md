# 升级指南 (UPGRADE)

本指南面向从 **v1.6.x** 升级到最新版本（含开源就绪加固）的自部署用户，覆盖破坏性变更、环境变量变更、数据库迁移、部署与回滚。

> 当前稳定版本：`1.6.20`（见 `CHANGELOG.md`；生产部署镜像标签见 `.env.prod` 的 `VERSION`，已与发布版本对齐）。
> 本文档同时适用于 Docker Compose 与私有化（k8s / 二进制）部署。

---

## 0. 升级前检查清单

- [ ] **完整备份数据库**（PostgreSQL `pg_dump`）与 `uploads/`、`.env.prod`。
- [ ] 确认 `DB_PASSWORD` / `JWT_SECRET` / `ADMIN_PASSWORD` 均为强随机值（≥ 32 字符）。
- [ ] 记录当前运行的镜像标签（如 `ghcr.io/heidsoft/itsm-backend:1.6.9`）。
- [ ] 确认维护窗口：迁移期间建议停止写入或切换到只读。
- [ ] 外部集成方已知悉 [破坏性变更](#1-破坏性变更)（camelCase 字段、Envelope 变更）。

---

## 1. 破坏性变更

以下变更来自 `CHANGELOG.md` 的 `Unreleased` 段，集成方必须适配：

### 1.1 API 响应字段改为 camelCase
Ticket / Incident / SLA / BPMN 响应不再暴露 snake_case 字段：

| 旧（snake_case，已移除） | 新（camelCase） |
| --- | --- |
| `deleted_count` | `deletedCount` |
| `assigned_count` | `assignedCount` |
| `page_size` | `pageSize` |
| `workflow_steps` | `workflowSteps` |
| `avg_resolution_time` | `avgResolutionTime` |

前端 `src/lib/api/http-client.ts` 的 `toCamelCase` 兼容层保留一个版本作为防御，下个次要版本移除。

### 1.2 SLA / BPMN 查询参数改为 camelCase
- `GET /api/v1/sla-policies/match`：`ticket_type` → `ticketType`，`customer_tier` → `customerTier`
- `GET /api/v1/sla-policies/compliance-rate`：`start_date` / `end_date` → `startDate` / `endDate`
- BPMN 监控端点：`time_range` / `start_time` / `end_time` → `timeRange` / `startTime` / `endTime`

### 1.3 SLA 模板与 BPMN 监控端点改用标准信封
`controller/sla_template_controller.go`、`controller/bpmn_monitoring_controller.go` 现在返回
`{ code, message, data }`（通过 `common.Success` / `common.Fail`）。旧的 `{"message":..., "data":...}`
信封已移除；HTTP 状态码语义保持不变（400→`ParamErrorCode`，404→`NotFoundCode`，500→`InternalErrorCode`）。

### 1.4 Ant Design `direction` prop 移除
五个页面的 `Space` / `Steps` 组件改用 `orientation="vertical"`（Ant Design v6 API）。
CI 守卫 `itsm-frontend/tools/check-antd-direction.sh` 会在 `direction="vertical"` 重现时构建失败。

### 1.5 内置角色 admin / technician 权限种子补齐（2026-09-17，权限扩大提示）
`BuiltinRoles()` 种子会将 `users.role` 词表角色写入 `roles` 表（DBOnly 权威态判定为
configured），但角色权限映射此前缺少 `admin` / `technician` 条目，导致这两个角色的
`role_permissions` 为空集——DBOnly 语义下空集=显式撤销，`users.role=admin|technician`
的用户除个别未挂权限中间件的路由外全部 403。

本次补齐：
- `pkg/seeder`：`builtinRolePermissionCodes()` 补 `admin`（91 码，与硬编码兜底全等）
  / `technician`（16 码）条目；`permissionDefinitions()` 补 25 个缺失权限码。
- `handlers/sla_template`：`/sla/templates` 三个路由补挂 `RequirePermission`（此前裸奔）。
- 守卫测试 `pkg/seeder/role_permission_guard_test.go`：词表角色空集、码空间漂移、
  admin/technician 与硬编码兜底不一致，三者任一出现即 CI 红。

**自部署用户注意**：升级后重启时 seeder 会自动为已存在的 admin / technician 角色行补建
权限行（幂等，只增不删）。这是对角色设计语义（"租户内系统管理"/"二线技术处理"）的恢复，
属**权限扩大**——如你曾依赖"空集拒绝"的行为或自行配置过这两个角色，请升级后审计
`roles` 表权限绑定。

### 1.6 任务面与流程面分权：technician 权限收窄 / 任务操作纳入权限码（2026-09-17）
授权平面审计发现 `/bpmn/*` 与 `/process-trigger/*`、`/process-bindings/*` 等**写路由未挂权限
中间件**，仅靠 `ResourceActionMap` 的粗粒度路径预检兜底，而 `bpmn:write` 同时授予了
`technician`（rank=2 二线技术员）——实测该角色可创建/发布 BPMN 流程定义、启动/挂起流程实例、
触发任意流程、改写流程绑定，且可对**他人**任务提交决策。同时 `/bpmn/tasks*` 路由引用的
`task:read` / `task:admin` / `task:update` 权限码在码空间中从未定义，导致 DBOnly 权威态下
**审批中心对所有角色（除 super_admin）不可用**——典型「写通读堵」。

本次变更分两部分，方向相反，请注意：

**（A）权限收窄（需评估是否影响你的自定义角色）**
- `technician` 剥离 `bpmn:write`：不再能设计/发布/克隆/启停流程定义、启停/挂起流程实例、
  触发流程或改写流程绑定。流程写权限现由 `bpmn:write` / `bpmn:delete` 独占（种子中仅 `admin`、
  以及经 `allPermissionCodes()` 的 `sysadmin`/`it_director`/`ops_director` 持有）。
- 若你的租户曾把 `technician` 当作「流程维护者」使用，请在升级前为该角色补授 `bpmn:write`，
  或改用 `admin` / 自建流程管理员角色。
- 同时 `standard-changes`、`known-errors`、`escalation-matrices`、`a2ui`、`surveys`、
  `marketplace` 等域的写路由补挂了权限门（此前裸奔或仅靠预检兜底）。

**（B）权限扩大（恢复既有设计意图）**
- 新增 `task:read` / `task:update`：授予**除 guest 外的全部内置角色**。理由：任何角色都可能被
  流程指派任务（变更 CAB 评审、服务请求审批、工单流转），而粒度控制不在角色层——
  `GET /bpmn/tasks` 只返回本人/候选任务，`task:update` 的每一步（认领态区分、自审批防护、
  委托/加签目标校验）由 handler 层 `authorizeTaskActor` 二次收口。
- 新增 `task:admin`：仅授予 13 个管理/监督角色（`admin`/`sysadmin`/`it_director`/`ops_director`/
  `manager`/`dept_manager`/`team_lead`/`sd_manager`/`ops_manager`/`change_manager`/`it_admin`/
  `security_admin`/`audit_admin`），用于跨用户全量任务视图 `GET /bpmn/tasks/all`。
- 新增 `marketplace:read` / `marketplace:write` / `survey:read` 码定义，**未授予任何角色**
  （仅供后续批次开放，当前不产生任何访问变化）。

**升级注意**：`pkg/seeder` 的 `builtinRolePermissionCodes()` 会在启动时幂等地把上述码写入
`role_permissions`（只增不删、按受管权限集替换）。变更后请重启 backend 以刷新权限缓存

### 1.7 登录/刷新响应令牌收敛（2026-09-20，安全加固）
`POST /api/v1/auth/login` 和 `POST /api/v1/auth/refresh`（含遗留 `/api/v1/refresh-token`）
的 JSON 响应不再返回 `accessToken` 和 `refreshToken` 字段，改为仅通过 HttpOnly cookie 下发
（`access_token` 15 分钟，`refresh_token` 7 天）。响应 `data` 仅保留 `user` 上下文。

**影响范围**：
- 前端从 `response.data.accessToken` 读取令牌的代码将失效，必须改为依赖 HttpOnly cookie
  自动携带（浏览器行为，无需手动处理）
- API 集成方若通过 JSON 解析令牌，需改为 cookie 模式或联系后端获取替代方案
- CSRF 保护已启用，写操作需携带 CSRF token（从 `GET /api/v1/csrf-token` 获取）

**升级注意**：此变更为安全加固，防止 XSS 攻击窃取令牌。升级后所有客户端必须适配
cookie 模式，否则将无法完成认证。
（TTL 5 分钟且无失效端点）。升级后建议用 `admin` 与 `technician` 两个账号各取一个写端点
（如 `POST /api/v1/bpmn/process-definitions`）与一个任务端点（`GET /api/v1/bpmn/tasks`）
复验预期：admin 全通、technician 写 403 / 任务 200。

---

### 1.7 权限码词表统一 + 预检映射全量对齐（2026-09-17，权限扩大提示）
批次 2/3 治本批次落地后，**大量此前对普通管理角色不可达的管理面恢复可用**，属于权限扩大，
私有化部署升级后建议复核各角色的实际可达面是否符合最小权限预期。

**权限变化摘要**（DBOnly configured 态，种子收敛式写入，`itsm-init` 重跑即生效）：
- **CMDB 全域恢复可达**：`cmdb_ci*`/`cmdb_cloud_*`/`cloud_*` 等 15 族路由码收敛到既有 `cmdb:*`，
  持有 `cmdb:read/write/delete` 的角色即刻恢复配置项/CI 类型/关系/标签/视图/导入导出/云资源纳管全量访问。
- **admin 能力补齐**：新增 `ticket:{assign,escalate,export}`、`change:{approve,rollback}`、
  `release:{approve,rollback}`、`service_request:approve`、`incident:delete`、`knowledge:delete`、
  `ticket:{create,update}`（此前 admin 缺 ticket:update/create，**更新工单会 403**）。
- **write⇒同义细分动作奇偶补齐**（全角色）：ticket/ticket_category/ticket_tag/ticket_template 的
  write 持有者自动获得 create/update；department write 持有者获得 create/update/delete；
  notification write 持有者获得 create；system write 持有者获得 read。
- **新增码**：`tenant:{read,write}`（授予 admin/sysadmin）。
- **AI 端点预检放宽**：`/ai/chat`、`/ai/triage`、`/ai/rag/search`、`/ai/predictions`、
  `/ai/*/analyze`、`/agent/tools/execute`、`/cmdb/cis/search`、`/tickets/prediction/*`、
  `/sla/monitor*` 等 18 条 POST 端点预检对齐路由声明的 read——持有对应 read 码的角色
  （如 technician/agent 的 ai:read）此前被预检层误拒，现在可用。若需限制 AI 消耗，
  请用限流/配额机制而非 RBAC。
- **预检映射对齐**：`/vendors /surveys /connectors /applications /reports /menus /tenants /cloud`
  等路径的 403「推断失配」全部消除；`/audit-logs` 资源名错（audit_logs→audit）修复。

**升级操作**：无手工步骤，重建 `itsm-init` 镜像并 `up -d` 即可（seeder 收敛式写入会
补齐新码、删除已收敛的旧映射行）。验证：admin 登录后访问 `/cmdb/cis`、`/audit-logs`、
`/tenants` 应 200。

---

### 1.8 列表响应统一为 `{items,total,page,pageSize,totalPages}`（2026-10-02，破坏性）
`common.ListResponse` 此前会在 `items` 之外**再返回一份同数组的领域名别名**（按切片元素类型
反射推断，例如 `tickets`、`incidents`、`changes`、`articles`），并把同一组分页事实再嵌套一个
`pagination` 对象。同一接口因此有两套真相，消费方读到哪一套取决于代码写的先后。现在
`data` 只输出五个平铺键，别名与嵌套 `pagination` 一律删除。

**旧 → 新**：

```jsonc
// 旧（GET /api/v1/changes）
{ "code": 0, "data": {
    "items": [ … ], "changes": [ … ],           // 同一份数组返回两遍
    "total": 42, "page": 1, "pageSize": 10, "totalPages": 5,
    "pagination": { "total": 42, "page": 1, "pageSize": 10, "totalPages": 5, "hasNext": true, "hasPrev": false }
}}

// 新
{ "code": 0, "message": "success", "data": {
    "items": [ … ], "total": 42, "page": 1, "pageSize": 10, "totalPages": 5 }}
```

**集成方必须改的三件事**：
1. 集合一律读 `data.items`；不要再读领域名顶层键，也不要写 `items ?? tickets ?? []` 这类
   多字段 fallback（AGENTS.md 已禁止，现在也没有第二份数据可供 fallback）。
2. 分页元数据只读平铺的 `data.total` / `data.page` / `data.pageSize` / `data.totalPages`；
   `data.pagination` 不再返回。
3. 请求分页参数只发 `page` 与 `pageSize`。本次同时把事件域读侧从 `size` 改成 `pageSize`
   （`GET /api/v1/incidents`、`GET /api/v1/incidents/alerts/active`），继续发 `size` 会被忽略并
   回退默认 10 条——`TestList_SizeParamIsNotContract` 锁死了这一点。

**本次已切到标准信封的生产入口**：`GET /api/v1/tickets`、`GET /api/v1/incidents`、
`GET /api/v1/incidents/alerts/active`（补上此前缺失的 `totalPages`）、`GET /api/v1/changes`、
`GET /api/v1/knowledge/articles`、`GET /api/v1/ticket-types`，以及所有走
`common.SuccessWithPagination` / `common.NewListResponse` 的接口。`GET /api/v1/bpmn/process-definitions`、
`GET /api/v1/bpmn/process-instances` 的 `pagination.total` 读取点同步改为 `total`。

**尚未收敛（按棘轮基线登记，不构成本次破坏）**：`dto/*.go` 里 30 个手写 List 结构仍返回领域名
集合键（`roles`、`menus`、`tenants`、`catalogs`、`allocations`、`notifications` 等），9 个 CMDB/服务目录/
通知信封仍用 `size` 代替 `pageSize`，11 个信封仍缺 `page`/`pageSize`/`totalPages` 中的若干键。
清单与收敛方法见 `itsm-backend/tests/contract/list_envelope_ratchet_test.go`——三条双向棘轮会让
新增违规和"收敛后忘记收口"都构建失败，因此这些存量形状只减不增。

**验证**：
```bash
cd itsm-backend && go test ./tests/contract/... -run TestListEnvelope
cd itsm-backend && go test ./handlers/incident/... -run 'TestList_CanonicalEnvelope|TestList_SizeParamIsNotContract'
cd itsm-backend && go test ./handlers/knowledge/... -run TestListArticles_CanonicalEnvelope
```

### 1.9 会话真相统一到后端（2026-10-02，破坏性：续签与登出语义）

三处对外行为变化，API 集成方与自研客户端必须适配：

1. **`POST /api/v1/auth/logout` 不再要求认证凭证。** 此前它挂在鉴权中间件之后，access token
   过期（15 分钟）后必然 401，浏览器留下 7 天的 `refresh_token` cookie 被自动续签，登出形同失效。
   现在无条件清除两类 cookie 并吊销请求中携带的 access/refresh token；吊销存储不可用时返回
   HTTP 503 / 业务码 5003（`会话凭证已清除，但服务端吊销未完成`），重复登出幂等返回 200。
2. **refresh token 改为单次使用（含未配置 Redis 的部署）。** 每次
   `POST /api/v1/auth/refresh` 原子认领旧凭证并下发新凭证，重放旧值返回 401 / 业务码 2001。
   旧版本只有配置了 Redis 才生效，`REDIS_HOST` 为空时黑名单检查被静默跳过，同一枚 refresh token
   可无限重放。客户端必须持久化**新签发的 cookie**而不是继续复用旧值；多副本部署若要吊销状态
   跨副本共享，必须配置 `REDIS_HOST`（未配置时启动日志会显式告警，吊销仅覆盖处理该请求的副本）。
3. **改密/停用/降权现在真正吊销存量 refresh token。** 此前该路径调用从未被装配构造的
   `TokenBlacklistService`，属于死代码；`MinIssuedAt` 的 TTL 也只有 1 小时，短于 7 天 refresh
   生命周期。现在 TTL 为 8 天且刷新链路会消费该约束。

**新增端点**：`GET /api/v1/auth/session` → `{user, tenants, expiresIn}`，前端唯一的会话真相来源；
`POST /api/v1/auth/login` 与 refresh 响应新增 `expiresIn`（服务端时钟的 access token 剩余秒数）。
令牌值仍只在 HttpOnly cookie 中，响应体不返回。

**验证**：
```bash
cd itsm-backend && go test ./router/... -run TestSetupRoutes_\(AuthCookieOnlyResponses\|SessionTruthAndLogout\)
cd itsm-backend && go test ./middleware/... -run 'TokenRevocation|InvalidateUserTokens'
```

**前端（同批改动）**：登录态的唯一来源改为 `GET /api/v1/auth/session`（`src/lib/api/session-api.ts`），
以下本地推断入口已删除，自研页面若曾复用需改为读会话端点：

| 删除 | 原用途 |
| --- | --- |
| JS 写入的 `auth-token` 标记 cookie 与 `token-storage.isAuthenticated()` | 用「JS 能看见的 cookie 存在」代替后端结论 |
| `AuthService.setTokens/getAccessToken/getRefreshToken/getToken/clearTokens` | httpOnly 下恒为 null 的凭证读写 helper |
| 废弃客户端 `authApi.refreshToken()/validateToken()` 与 legacy `/api/v1/refresh-token` 前端调用 | 第二套续签与探活 |
| `lib/auth/jwt-decoder.ts`、`components/layout/RouteGuard.tsx`、`components/providers/Providers.tsx` | 按 JWT 形状判定登录；挂在永不为真的 `getToken()` 上的守卫 |
| 登录请求与登录页的 `rememberMe` | 后端从不读该字段，勾选框不改变服务端窗口 |
| `useAuthStore` 的 `token` 字段与 `notificationWS.connect(userId, token)` 的 token 形参 | httpOnly 下 JS 从不持有凭证；通知页曾按 `user?.id && token` 给 WebSocket 设门禁，那个 `token` 恒为 `undefined`，连接因此静默失败 |

`auth-storage` 的持久化改为显式 `merge`：只恢复 `currentTenant`，旧版本写入的 `user`/`isAuthenticated`/`token`
不再被浅合并复活。自研代码若曾读 `useAuthStore.getState().token`，改为直接发起请求（凭证由浏览器 cookie
携带）或读会话端点。

续签改由后端 `expiresIn` 驱动并**强制单飞**：refresh token 单次可用，并行续签会让后到的请求拿旧凭证
认领失败，把有效会话判定为过期。登出请求带 `keepalive`，调用方随后整页跳转也不会取消吊销请求；
服务端吊销失败不再沉默——本地状态清空，同时在控制台留下 `revoked=false` 的原因。

### 1.10 通知列表信封与查询参数收敛（2026-10-03，破坏性）

§1.8 登记为债务的 `notifications` 集合键在通知域落地，两个列表端点各自只有一个真相：

- `GET /api/v1/notifications`：`data` 从 `{notifications,total,page,size}` 改为
  `{items,total,page,pageSize,totalPages}`（该端点实测真的做 `Count` + `Offset/Limit`，
  所以补齐三键而不是降级）。查询参数只认 `page`/`pageSize`/`type`/`read`；此前的
  `size`、`is_read` 不再被读取，`pageSize` 越界或缺省回落到 20（上限 100），`page`
  缺省或非法回落到 1。`userId`/`tenantId` 改为只来自认证上下文，请求里自报的同名
  查询参数不再参与绑定。
- `GET /api/v1/tickets/{id}/notifications`：集合键 `notifications` → `items`，形状保持
  诚实的 `{items,total}`（服务层无 `Offset/Limit`，按 §「不分页的列表」不得伪造分页键）。

```jsonc
// 旧（GET /api/v1/notifications）
{ "code": 0, "data": { "notifications": [ … ], "total": 42, "page": 1, "size": 20 } }

// 新
{ "code": 0, "message": "success", "data": {
    "items": [ … ], "total": 42, "page": 1, "pageSize": 20, "totalPages": 3 } }
```

**集成方必须改的三件事**：读 `data.items` 而非 `data.notifications`；请求分页发
`pageSize` 而非 `size`、已读过滤发 `read` 而非 `is_read`；不要再尝试用查询参数
`userId`/`tenantId` 指定他人（此前该参数会被绑定，现在忽略——通知只会返回认证用户自己的）。

### 1.11 变更列表分页夹紧与风险等级参数名单一化（2026-10-03，破坏性）

`GET /api/v1/changes` 此前用裸 `strconv.Atoi` 读查询参数并忽略错误，值原样传进 Ent 的
`Offset/Limit`，而响应的分页元数据在 `SuccessWithPagination` 里另走一遍
`ValidatePagination`——同一个非法入参在 SQL 侧和声明侧有两个答案。属于 §1.10 之前同一批
未修的缺陷类。现在 HTTP 入口统一走 `common.GetPaginationFromQuery`，与通知、变更 PIR
列表共用同一套夹紧规则。

| 查询入参 | 旧行为 | 新行为 |
| --- | --- | --- |
| `pageSize` 缺省或非法 | 默认 10 | **默认 20** |
| `pageSize=0` | SQL 不加 `LIMIT`，**整表返回**，同时响应写 `pageSize:10` | 夹紧为默认页长 20，响应与行数一致 |
| `page=0` / 负数 | SQL 收到负 `OFFSET`（Postgres 报错） | 回落第 1 页 |
| `pageSize=5000` | SQL 用 5000、响应声明 100（两套夹紧） | 回落默认页长 20（`GetPaginationFromQuery` 只采纳 `(0,100]`，**不是夹到 100**） |
| `risk_level=high` | 生效 | **不再读取**，只认 `riskLevel` |

`risk_level` 的移除不是因为有人用它——实测前端从未发送过该名字（`change-api.ts` 一直发
`riskLevel`）——而是「先读 snake_case、空则读 camelCase」属于 AGENTS.md 禁止的多字段兼容，
保留会让两套参数名长期共存且无法判定哪套是契约。

列表排序同时补了 `ID` 并列键：`created_at` 非唯一，同一秒创建的变更在页边界的归属此前不确定。

**集成方必须改的两件事**：
1. 需要整表或大页时显式发 `pageSize`（上限 100），不要依赖 `pageSize=0`「取全部」——
   该写法此前会返回整个租户的变更表，现在返回 20 条。
2. 风险等级过滤只发 `riskLevel`。同时注意 `type`、`priority` 两个查询参数后端**从不识别**
   （静默忽略，前端表单和 mock 却在按它们过滤）；该缺口已登记为待收敛债务，不要当成可用契约。

**验证**：
```bash
cd itsm-backend && go test ./router/ -run TestChangeListRoute
cd itsm-frontend && npx jest --runTestsByPath src/lib/api/__tests__/change-api.test.ts
```

### 1.12 问题列表信封、分页夹紧与查询参数收敛（2026-10-03，破坏性）

`GET /api/v1/problems` 此前有**三套**页长规则同时生效：handler 把绑定到的原值直接下传，
repository 自己按 `page<1→1、size<1→10、size>200→200` 截断，响应末尾再手写一遍
`page<1→1、pageSize<1→10` 和一个没有上限的除法，把未夹紧的原值回显出去。与 §1.10、
§1.11 是同一个缺陷类，但这里的后果是**静默丢数据**而不是多取：205 条数据配 `pageSize=250`
时 SQL 只返回 200 条，响应却声明 `pageSize:250、totalPages:1`——按响应声明翻页的调用方翻完
第 1 页就停，第 201~205 条在任何一页都拿不到。现在与通知、变更列表共用
`common.GetPaginationFromQuery` 单点夹紧。

| 项目 | 旧行为 | 新行为 |
| --- | --- | --- |
| 响应集合键 | `data.problems` | **`data.items`**（标准五键信封 `{items,total,page,pageSize,totalPages}`） |
| `pageSize` 缺省或非法 | 声明侧默认 10 | **默认 20** |
| `pageSize=0` | 走 repository 的 `size<1→10`，与声明侧 10 恰好一致但无保证 | 夹紧为默认页长 20，SQL 与响应同源 |
| `page=0` / 负数 | SQL 侧与响应侧各自兜到第 1 页（同一规则两处兜底，谁先改谁分叉） | 由 HTTP 入口单点回落第 1 页 |
| `pageSize=300` | SQL 截断为 200、响应声明 300（两个真相） | 回落默认页长 20（只采纳 `(0,100]`，**不是夹到 100**） |
| `sortBy`/`sortOrder`/`dateFrom`/`dateTo` | DTO 声明，但 handler 与 repository 从不读取 | **从请求 DTO 删除**，未识别参数一律忽略 |

排序固定为 `created_at DESC, id ASC`：`created_at` 非唯一列，此前同一秒创建的问题在页边界的
归属不确定，补 `ID` 兜底成全序。

**集成方必须改的两件事**：
1. 读列表用 `data.items`，不要再读 `data.problems`；需要整表或大页时显式发 `pageSize`
   （上限 100），`pageSize=0` 与越界值现在都表示「用默认页长」而不是「取全部」。
2. 删除对 `sortBy`/`sortOrder`/`dateFrom`/`dateTo` 的依赖。这四个参数此前就不生效，任何按
   它们排序或按时间过滤的集成逻辑本来已经是错的，只是没有报错。

**验证**：
```bash
cd itsm-backend && go test ./router/ -run TestProblemListRouteEnvelopeAndPagination
cd itsm-frontend && npx jest --runTestsByPath src/lib/api/__tests__/problem-api.test.ts
```

### 1.13 服务请求列表信封与分页夹紧收敛（2026-10-03，破坏性）

`GET /api/v1/service-requests`、`GET /api/v1/service-requests/me` 与
`GET /api/v1/service-requests/approvals/pending` 原先有**三套**页长规则同时生效：请求 DTO 的
`binding:"omitempty,min=1,max=100"`、handler 里手写的 `Page==0→1 / PageSize==0→10`、
`handlers/service_request/repository_impl.go` 私有的 `page<1→1、PageSize<1→10、>100→100`；
响应末尾再手拼一遍 `map[string]interface{}` 和 `(total+pageSize-1)/pageSize`。
前端 `src/lib/api/service-request-api.ts` 声明的响应形状是**第四套**
`{requests,total,page,size}`，靠 `normalizeList` 的 `raw.requests || raw.items || []`、
`raw.size || raw.pageSize || requests.length` 猜后端到底返回什么，`total` 缺失时静默变成
「当前页长度」。后端 JSON 键名这次**没有变化**（两个 handler 本来就手拼 `items/total/page/
pageSize/totalPages`），变化的是夹紧归属、默认页长与越界值的 HTTP 结果。

| 项目 | 旧行为 | 新行为 |
| --- | --- | --- |
| 缺省 `pageSize` | handler 私有默认 **10**，`totalPages` 按 10 算 | **20**（`common.GetPaginationFromQuery`） |
| `pageSize=0` | 绑定 `min=1` 先失败 → **HTTP 400 / code 1001** | **HTTP 200 / code 0**，回落默认页长 20，且不退化成不加 `LIMIT` |
| `pageSize=5000` | 绑定 `max=100` 失败 → **HTTP 400 / code 1001** | **HTTP 200 / code 0**，回落 **20**（只采纳 `(0,100]`，**不是夹到 100**），SQL 与声明同源 |
| `page=0` / `page=-1` | 绑定 `min=1` 失败 → **HTTP 400 / code 1001** | 由 HTTP 入口单点回落第 1 页，负 `OFFSET` 不出现 |
| 分页夹紧归属 | DTO binding + handler + repository 三处 | HTTP 入口 `common.GetPaginationFromQuery` 单点；repository 保留一道 `common.ValidatePagination` 只服务**非 HTTP 调用方** |
| `size`/`limit`/`sortBy` | DTO 从未声明，前端 `normalizeList` 却按 `size` 优先读 | 前端只读 `items/total/page/pageSize/totalPages`，别名读取全部删除 |
| `userId` | 旧 handler 读 `req.UserID`（该字段无 `form` tag，实测从不生效） | 从请求 DTO 删除；操作者身份只取认证上下文，`/me` 由路径或 `scope=me` 判定 |
| `status`（待审批收件箱） | 绑定但从不参与查询 | **收件箱不再绑定任何查询过滤**，条件只由审批记录的 pending 与认证上下文决定 |
| 排序 | `created_at DESC`（非唯一列，同秒入库的请求单在页边界归属不确定） | `created_at DESC, id ASC` |

**集成方必须改的两件事**：
1. 按「缺省即 20 条」核对分页预期；`pageSize` 越界或为 0 不再返回 400/1001，而是
   **HTTP 200 + code 0 + 页长 20**，把 400 当成「参数非法」信号来处理的那段逻辑需要重看。
   想要大页请显式发 `pageSize`（上限 100）；越界值现在表示「用默认页长」，不表示「取全部」。
2. 前端调用方删除对 `requests` / `size` 的读取。`src/types/biz/service-request.ts`（第三份
   服务请求实体声明，含 `ServiceRequestListResponse{requests,total,page,size}`）已随本批删除，
   唯一类型来源是 `src/lib/api/service-request-api.ts`。

**验证**：
```bash
cd itsm-backend && go test ./router/ -run TestServiceRequestListRouteEnvelopeAndPagination
cd itsm-frontend && npx jest --runTestsByPath src/lib/api/__tests__/service-request-api.test.ts src/lib/api/__tests__/service-catalog-api.test.ts
```

### 1.14 CMDB 列表信封、页长单一所有者与 `size` 别名移除（2026-10-03，破坏性）

受影响端点（`/api/v1/configuration-items/*` 兼容别名同批生效，两者是同一 handler）：

| 端点 | 集合键 | 信封来源 DTO |
| --- | --- | --- |
| `GET /api/v1/cmdb/cis` | `items`（未变） | `dto.CIListResponse` |
| `GET /api/v1/cmdb/ci-types` | `items`（未变） | `dto.CITypeListResponse` |
| `GET /api/v1/cmdb/tags` | `items`（未变） | `dto.CITagListResponse` |
| `GET /api/v1/cmdb/cis/{id}/history` | `items`（未变） | `dto.CIHistoryListResponse` |
| `GET /api/v1/cmdb/views` | `items`（未变） | `dto.ListResponse[CISavedView]` |
| `GET /api/v1/cmdb/import` | `items`（未变） | `dto.ListResponse[ImportCIResult]` |
| `GET /api/v1/cmdb/export` | `items`（未变） | `dto.ListResponse[ExportCIResult]` |

这 7 个端点**实测都真的分页**（`Count` + `Offset/Limit`），响应却只回 `{items,total,page,size}`：
没有 `totalPages`，调用方无法核对是否还有下一页，把当前页当成该租户的全部 CI / CI 类型 / 历史。
`size` 同时是请求侧的分页别名（`dto.ListCIRequest` 的 `form:"size"`）。JSON 键名变化只有
`size`→`pageSize` 与新增 `totalPages`，`items` 从未改过名。

| 项目 | 旧行为 | 新行为 |
| --- | --- | --- |
| 响应分页键 | `page` + `size`，无 `totalPages` | `page` + `pageSize` + `totalPages`（`common.NewPaginationResponse` 单点算出） |
| 请求页长键 | `size`（`pageSize` 不被读取） | **只有 `pageSize`**，`size`/`limit`/`offset` 一律不参与分页决策 |
| `GET /cmdb/cis` 越界页长 | `binding:"min=1,max=200"` → **HTTP 400 / code 1001**（`size=0` 因 `omitempty` 绕过校验，`Limit(0)` 在 Ent 等于不加 LIMIT，**返回整表**） | **HTTP 200 / code 0**，回落默认页长 20 |
| 其余 6 个端点越界页长 | handler 用裸 `strconv.Atoi`，**完全不夹紧**：`size=300` 原样下传取 300 条；`size=abc` 解析失败变 0 → `Limit(0)` **返回整表** | 统一 `common.GetPaginationFromQuery`：缺省 1/20，只采纳 `(0,100]`，越界回落 **20** |
| 默认页长 | 各处 `DefaultQuery("size","20")`，无单一所有者 | 20（平台值） |
| `include_public`（`/cmdb/views`） | 查询参数 `include_public` | **`includePublic`**（camelCase 契约；旧名实测只有 swagger 提到，前端与测试零调用） |
| CI 列表排序 | 无 `sortBy` 时**完全没有 `ORDER BY`**，翻页顺序由数据库返回顺序决定，同一条可能重复或漏出 | 固定 `created_at DESC, id ASC`；类型/标签/历史/视图/导入导出同样补 `id` 并列键 |
| CI 导出记录数 | 任务内 `PageSize: 10000` 一次性下传，被服务层静默夹成 **20 条** | 按 `MaxPageSize` 逐页读到 10000 上限，导出范围与用户勾选一致 |
| `GET /cmdb/ontology` 的 `ciTypes` | 索要 `pageSize=500`（把「要整表」藏在越界页长里），超过 500 个类型静默截断 | 逐页读全该租户活跃类型 |

**集成方必须改的三件事**：
1. 读响应的地方把 `data.size` 改成 `data.pageSize`，并可直接使用新增的 `data.totalPages`。
2. 请求侧删除 `size`，改发 `pageSize`。想要大页必须显式发 `pageSize`（上限 100）；
   **越界值现在意味着「用默认页长 20」，不再意味着「取全部」**，也不会再返回 400/1001。
   原先依赖 `size=500`/`size=10000` 一次取全量的调用方会静默只拿 20 条，必须改成翻页读取
   （前端已提供 `CMDBApi.getAllCIs(params, maxRecords)` / `getCITypes()` 作为参考实现）。
3. `/cmdb/views` 的 `include_public` 改 `includePublic`。

**验证**：
```bash
cd itsm-backend && go test ./router/ -run TestCMDBListRoutesEnvelopeAndPagination
cd itsm-frontend && npx jest --runTestsByPath src/lib/api/__tests__/cmdb-api.test.ts src/lib/api/__tests__/cmdb-relationship.test.ts
```

### 1.15 服务目录列表信封与 `size` 别名移除（2026-10-03，破坏性）

受影响端点（4 个 URL 共用同一个 `Handler.List`，`/search` 共用 `Handler.Search`）：

| 端点 | 集合键 | 信封来源 DTO |
| --- | --- | --- |
| `GET /api/v1/service-catalog` | `items`（原 `catalogs`） | `dto.ServiceCatalogListResponse` |
| `GET /api/v1/service-catalogs` | `items`（原 `catalogs`） | `dto.ServiceCatalogListResponse` |
| `GET /api/v1/service-catalog-services` | `items`（原 `catalogs`） | `dto.ServiceCatalogListResponse` |
| `GET /api/v1/service-catalogs/search` | `items`（原 `catalogs`） | `dto.ServiceCatalogListResponse` |

这是列表信封棘轮里**最后一条 `size` 别名基线**，删除后 `envelopeAliasBaseline` 为空集。
四个服务目录列表端点实测都做 `Count` + `Offset/Limit` 真分页，响应却只回
`{catalogs,total,page,size}`：领域名集合键 + `size` 别名 + 没有 `totalPages`。
页长同时有 4 个所有者（实测互不一致）：

| 项目 | 旧行为（120 条种子实测） | 新行为 |
| --- | --- | --- |
| 响应集合键 | `catalogs` | **`items`** |
| 响应分页键 | `page` + `size`，无 `totalPages` | `page` + `pageSize` + `totalPages`（`common.NewPaginationResponse` 单点算出） |
| 请求页长键 | 只认 `size`（`dto.GetServiceCatalogsRequest` 的 `form:"size"`） | **只有 `pageSize`**，`size` 完全不参与分页决策 |
| `?size=1000` | 通过 `binding:"max=1000"`，再被 handler 静默夹成 **100 条** | HTTP 200 / code 0，回落默认页长 **20** |
| `?size=1001` | `binding` 拒绝 → **业务 code 1001 参数错误** | HTTP 200 / code 0，回落默认页长 **20** |
| 缺省页长 | `Handler.List` 10、`Service.List` 10、`Service.Search` 20，三处各写一遍 | 20（平台值，单点） |
| `GET /service-catalogs/search` 翻页 | `Page`/`Size` 写死 `1`/`20`，**第二页永远拿不到数据** | 与 `List` 共用同一入口所有者，`page`/`pageSize` 真正生效 |
| 空结果 | `var responses []dto.ServiceCatalogResponse` → `"catalogs": null` | `"items": []` |
| 排序 | `created_at DESC`（同秒数据页边界不确定） | `created_at DESC, id ASC` |
| 非 HTTP 调用方零值分页 | `Offset(0)+Limit(0)`，Ent 的 `Limit(0)` 等于不加 LIMIT → **整表读出** | `EntRepository.List/Search` 用 `common.ValidatePagination` 兜底（缺省 10） |

**集成方必须改的三件事**：
1. 读响应的地方把 `data.catalogs` 改成 `data.items`、`data.size` 改成 `data.pageSize`，并可直接使用新增的 `data.totalPages`。
2. 请求侧删除 `size`，改发 `pageSize`（上限 100）。**越界值现在意味着「用默认页长 20」，
   不再意味着「取全部」，也不会再返回 1001**；原先发 `size=1000` 想一次取全的调用方
   （旧实现其实只拿到 100 条）必须改成翻页读取。前端已提供
   `ServiceCatalogApi.getAllServices(params, maxRecords)`，`exportCatalog` 已改为翻页取全并在
   超过 5000 条时显式报错，而不是静默导出半截 CSV。
3. 关键词搜索走 `GET /api/v1/service-catalogs/search?q=`（现在支持 `page`/`pageSize`）。
   列表端点仍不接受 `search`/`q`，前端 `getServices({search})` 只做当前页过滤并如实回显
   后端 `total`，服务端 search 属于账本 E4-24。

**验证**：
```bash
cd itsm-backend && go test ./router/ -run TestServiceCatalogListRoutesEnvelopeAndPagination
cd itsm-backend && go test ./handlers/service_catalog/ ./tests/contract/
cd itsm-frontend && npx jest --runTestsByPath src/lib/api/__tests__/service-catalog-api.test.ts
```

### 1.16 资产/许可证列表信封收敛与第三个页长所有者删除（2026-10-04，破坏性）

| 端点 | 旧响应 | 新响应 |
| --- | --- | --- |
| `GET /api/v1/assets` | `{total, assets, page, pageSize, totalPages}` | `{items, total, page, pageSize, totalPages}` |
| `GET /api/v1/licenses` | `{total, licenses}`（只有两键，无分页回显） | `{items, total, page, pageSize, totalPages}` |

这两个端点原先有**三套**页长真相同时生效，且 `licenses` 那条会静默读整表：

| 项目 | 旧行为 | 新行为 |
| --- | --- | --- |
| `assets` 缺省页长 | handler `DefaultQuery("pageSize","10")` → **10**，服务层再判 `<1 或 >100 → 20` | **20**（`common.GetPaginationFromQuery` 单点） |
| `assets ?pageSize=150` | handler 自称允许到 200 并放行，服务层判 `>100` 静默改成 **20** | HTTP 200 / code 0，回落默认页长 **20**（只采纳 `(0,100]`） |
| `assets ?pageSize=201` / `?page=abc` | handler 手写 `strconv.Atoi` 失败 → **HTTP 400 / code 4000** | HTTP 200 / code 0，回落 1/20，不再用 400 表达「参数写错」 |
| `licenses ?page=abc` | handler 用 `page, _ := strconv.Atoi(...)` **吞掉错误**传 0；服务层条件是 `if page > 0 && pageSize > 0` 才加 `Offset/Limit` → 条件为假，**不带 `LIMIT` 把该租户许可证整表读出** | 无条件分页，非法页码回落 1，页长回落 20 |
| `licenses` 非 HTTP 调用方 | 传 0/0 即等价于「不加 LIMIT」 | `common.ValidatePagination` 兜底（缺省 1/10） |
| 排序 | `created_at DESC`（非唯一列，同秒数据页边界归属不确定） | `created_at DESC, id ASC` |
| 空结果 | `"assets": null` / `"licenses": null` | `"items": []` |

同一批删除三条**零引用死声明**（只被棘轮基线和自身测试引用，不是 HTTP 契约的一部分）：
`dto.ListParticipantsResponse` 与其元素类型 `dto.ArticleParticipantResponse`（知识文章参与者：
没有注册路由、没有 service 方法、前端无调用方），以及泛型 `dto.MSPReportListResponse[T]`
（活的 MSP 报表信封是别的结构体）。按 E4-3 口径直接删除而非改名保留。

**集成方必须改的两件事**：
1. 把 `data.assets` / `data.licenses` 改成 `data.items`，`licenses` 端点现在开始回显
   `page`/`pageSize`/`totalPages`，可以直接用它判断是否还有下一页。
2. 原先依赖「`pageSize` 越大取得越多」或「非法参数返回 400」的调用方需要重看：越界与非法值
   现在都是 HTTP 200 + code 0 + 默认页长 20，想取全量必须翻页。
   前端 `AssetList`/`LicenseList` 已改为直接读五键信封，并删掉 `|| []`、`|| 0` 兜底
   （`total: 0` 是合法值，兜底会把「无数据」与「请求失败」混为一谈）。

**验证**：
```bash
cd itsm-backend && go test ./router/ -run "TestAssetAndLicenseListRoutesEnvelopeAndPagination|TestAssetListDTOKeys"
cd itsm-backend && go test ./service/ -run TestAsset && go test ./tests/contract/
cd itsm-frontend && npx jest --runTestsByPath src/lib/api/__tests__/asset-api.test.ts
```

### 1.17 工单规则与附件三个列表的集合键改为 `items`（2026-10-04，破坏性）

| 端点 | 旧响应 | 新响应 |
| --- | --- | --- |
| `GET /api/v1/tickets/assignment-rules` | `{rules, total}` | `{items, total}` |
| `GET /api/v1/tickets/automation-rules` | `{rules, total}` | `{items, total}` |
| `GET /api/v1/tickets/{id}/attachments` | `{attachments, total}` | `{items, total}` |

这三个端点实测**不分页**：service 整表取回（`Where(tenant_id)` + `All(ctx)`，无 `Count`、无
`Offset/Limit`），handler 里 `Total: len(items)`。因此本批**只**收敛集合键，保留诚实的两键形状，
**没有**补 `page`/`pageSize`/`totalPages`——给不存在的分页协议补键等于用响应伪造契约
（AGENTS「能力状态与失败语义」，`docs/api-reference.md`「不分页的列表」）。

其他行为不变：排序仍是规则 `priority DESC, created_at DESC`、附件 `created_at DESC`；
查询参数仍然一个都没有（发送 `page`/`pageSize` 不报错也不生效）；空结果一直是 `[]`
（service 用 `make(..., 0, len)`）。权限与租户边界不变：规则列表要 `system_config:read`，
附件列表要 `ticket:read` 且调用者须是工单相关方或管理角色，跨租户工单回 404/`4004`。

**同一批删除两条并行声明**（`GET /api/v1/tickets/{id}/attachments` 此前有**三份**前端声明）：
`TicketApi.getTicketAttachments()`（声明 `{attachments,...}`，收敛后必然恒错，零生产调用方）与
`ticketService.getTicketAttachments()`（声明返回**裸数组** `TicketAttachment[]`，后端从不这样返回，
零调用方）。唯一真相是 `TicketAttachmentApi.listAttachments()`。按 E4-3 口径删除而非改名保留。

**集成方必须改的一件事**：把 `data.rules` / `data.attachments` 改成 `data.items`。
如果此前用 `pageSize` 想少取一点，注意这三个端点后端不做分页，返回的是该租户/该工单的全量。

**验证**：
```bash
cd itsm-backend && go test ./router/ -run "TestTicketRuleAndAttachmentListRoutesEnvelope|TestTicketAttachmentDownloadRoute"
cd itsm-backend && go test ./tests/contract/ -run TestListEnvelope
cd itsm-frontend && npx jest --runTestsByPath src/lib/api/__tests__/ticket-attachment-api.test.ts \
  src/lib/api/__tests__/ticket-assignment-api.test.ts src/lib/api/__tests__/ticket-automation-rule-api.test.ts
```

### 1.18 工单抄送两个列表的集合键改为 `items`，姓名键是 `fullName`（2026-10-04，破坏性）

| 端点 | 旧响应 | 新响应 |
| --- | --- | --- |
| `GET /api/v1/tickets/cc/my` | `{records, total}` | `{items, total}` |
| `GET /api/v1/tickets/{id}/cc` | `{records, total}` | `{items, total}` |

两个端点实测**不分页**（`service/ticket_workflow_service.go:339/351` 都是
`Where(tenant_id)` + `All(ctx)`、`Total=len(records)`），所以同样只收敛集合键，
保留诚实的两键形状，**不补** `page`/`pageSize`/`totalPages`。

**同时修一处从未生效的字段读取**：`item.user` / `item.addedBy` 是后端
`dto.WorkflowUserInfo`，姓名键一直是 `fullName`，**从未发出 `name`**。前端类型和
「我的抄送」页面读的是 `user.name`，因此这两列在所有历史版本里都落到 `username`。
本次把前端类型改为复用 `WorkflowUserInfo`，页面改读 `fullName`。
集成方若也按 `user.name` 取姓名，需要同步改为 `user.fullName`。

其他行为变化只有一处：两条列表的排序从 `added_at DESC` 改为
`added_at DESC, id ASC`。此前同一时刻抄送多人时顺序由数据库随意决定，现在是确定的
（自增 id 升序），重复请求不会漂。

权限与租户边界不变：两端都要 `workflow:read`，`cc/my` 只返回收件人为当前用户的记录，
`{id}/cc` 需要调用者是工单申请人/受理人/审批人/被抄送人或管理角色。**已知缺陷**：跨租户
探测 `{id}/cc` 时 service 返回的是 `NotFoundCode`，但 handler 用 `common.FailWithErr`
透传，实测固定成 HTTP 500 + `5001`（把「工单不存在」伪装成内部错误）。这与全站 286 处
`common.Fail(..., err.Error())` 一起登记为 E4-29，等 E4-5 错误分类收敛统一处理；
本次未改语义，只保证零泄漏。

**同一批删除一处死声明**：`src/types/ticket-workflow.ts` 的 `TicketCC` 接口零消费方，
且缺 `ticketNumber/title/status/priority` 四个字段（不是后端形状），按 E4-3 口径删除而
不是改名保留。唯一真相是 `src/lib/api/ticket-api.ts` 的 `TicketCCRecord`。

**验证**：
```bash
cd itsm-backend && go test ./router/ -run TestTicketCCListRoutesEnvelope
cd itsm-backend && go test ./tests/contract/ -run TestListEnvelope
cd itsm-frontend && npx jest --runTestsByPath src/lib/api/__tests__/ticket-api.test.ts
```

### 1.19 分配推荐列表的集合键改为 `items`，item 字段名与后端 DTO 对齐（2026-10-04，破坏性）

| 端点 | 旧响应 | 新响应 |
| --- | --- | --- |
| `GET /api/v1/tickets/assign-recommendations/{id}` | `{recommendations, total}` | `{items, total}` |

该端点实测**不分页**：`service/ticket_assignment_smart_service.go:116` 把租户内全部可用
用户排序后整表返回，handler 写 `Total=len(items)`，因此只收敛集合键并保留诚实的两键，
**不补** `page`/`pageSize`/`totalPages`。

item 的字段一直是后端 `dto.AssignmentRecommendation` 的形状，本次未改：
`{userId, username, name, email, score, reason, workload, skills?, categories?}`。
前端 `src/lib/api/ticket-assignment-api.ts` 此前的类型是**凭空发明**的
`userName/userEmail/userAvatar/factors{skillMatch,workload,historySuccess,availability}`，
后端从未发出任何一个键，消费方按它取值只会拿到 `undefined`；现已逐字段对齐后端 DTO。

**同一批删掉三处自我冗余的兜底**（同文件）：`normalizeRule` 里四处
`x ?? x`、`testRule` 请求体里两处 `data.ruleId ?? data.ruleId`，以及
`normalizeAutoAssign` 给 `assignmentType` 编造的 `'smart'` 默认值。后端实测只发
`auto` / `rule` / `ticket_type_rule` / `manual`（`service/ticket_assignment_service.go:110/138/435`、
`service/ticket_assignment_smart_service.go:112/239`），`'smart'` 是不存在的取值；
`AutoAssignResponse.assignmentType` 的联合类型已改为这四个真实值。

**已知缺陷（本片未改语义）**：跨租户工单推荐实测是 HTTP 500 + `5001`（service 返回
`ticket not found` 后被 `common.FailWithErr` 固定映射成内部错误，`common/response.go:183`），
语义应为 404 + `4004`；与全站 286 处 `common.Fail(..., err.Error())` 一并登记 E4-29，
等 E4-5 错误分类收敛统一裁定。测试已锁定「失败响应绝不携带 `items` 载荷、不泄漏对方
租户用户名」这一半。另有一处待裁：`assign-recommendations/{id}` 与 `{id}/auto-assign`
在前端**零生产调用方**（只有 API client 自身与它的单测），是否退役或补 UI 交 E5 裁决。

**验证**：
```bash
cd itsm-backend && go test ./router/ -run TestTicketAssignRecommendationsRouteEnvelope
cd itsm-backend && go test ./tests/contract/ -run TestListEnvelope
cd itsm-frontend && npx jest --runTestsByPath src/lib/api/__tests__/ticket-assignment-api.test.ts
```

### 1.20 业务拒绝不再伪装成服务故障：错误分类收敛到单一分类器（2026-10-04，破坏性）

本节只改**响应语义**，不改任何字段名、集合键或分页键。

**旧行为**：`common.FailWithErr`、`common.RespondError` 与 `ErrorHandler` 中间件各自持有
一套错误→状态的映射，其中 `FailWithErr` 无条件写 `Fail(c, 5001, publicMsg)`，也就是
HTTP **500**：service 层明确返回的 `common.BusinessError(NotFoundCode)`、`ForbiddenCode`、
`ConflictCode` 业务码在 handler 边界被整条丢弃。结果是「工单不存在」「你没权限看」
「版本冲突」在客户端、前端拦截器和监控里全部长成服务端故障。

**新行为**：`code ⇄ HTTP status` 只有一张登记表（`common/response.go` 的
`codeStatusPairs`），错误分类只有一个入口（`common.classifyError`，经 `errors.As` 识别
`AppError`/`BusinessError`/`VersionConflictError`），发响应只有一个落点（`common.writeClassified`）。
`Fail`、`FailWithData`、`FailWithErr`、`RespondError`、`ErrorHandler` 全部读这三件东西。

调用方会观察到的变化：

| 场景 | 旧响应 | 新响应 |
| --- | --- | --- |
| service 返回 `NotFoundCode` 业务错误 | `500` / `5001` | `404` / `4004` |
| service 返回 `ForbiddenCode` 业务错误 | `500` / `5001` | `403` / `2003` |
| service 返回 `ConflictCode` 业务错误 | `500` / `5001` | `409` / `4090` |
| service 返回 `UnprocessableEntityCode` | `500` / `5001` | `422` / `4220` |
| `AppError{HTTPStatus: 503}`（依赖未接线/不可用） | `500` / `5001` | `503` / `5003` |
| `VersionConflictError`（乐观锁冲突） | `409` / `4090`，`data` 为空 | `409` / `4090`，`data` 为 `{resourceId, currentVersion, serverVersion}` |
| 驱动/SQL/provider 原始错误（非业务错误） | `500` / `5001` + 安全文案 | 不变：仍 `500` / `5001` |

**已在本片锁定精确 status 的真实入口**（其余 292 处 `FailWithErr` 调用点无需改动即继承
正确映射，因为收敛发生在 helper 内部而不是逐个 handler）：

- `POST /api/v1/tickets/workflow/accept`：工单不存在/跨租户 → `404` / `4004`
- `GET /api/v1/tickets/cc/my`、`GET /api/v1/tickets/:id/cc`：跨租户 → `404` / `4004`；
  同租户但非发起人/受理人/审批人/抄送人 → `403` / `2003`
- `POST /api/v1/tickets/:id/auto-assign`、`GET /api/v1/tickets/assign-recommendations/:id`：
  工单不存在与跨租户返回**同一个** `404` / `4004`，探测方无法据此确认工单存在于别的租户
  （这修正了 §1.19 末尾登记的「已知缺陷」）

**不变的部分**（避免误判成更大范围的重构）：

- 公共文案的所有权仍按入口点区分：`FailWithErr` 继续用 handler 传入的 `publicMsg`
  （领域错误消息可能是英文，直接透出会破坏中文界面文案）；`RespondError` 透出领域消息。
- `Fail`/`FailWithData` 对未登记业务码仍然回落 HTTP 200，新码必须显式登记，不靠 default 蒙混。
- 响应信封结构 `{code, message, data}` 未变；`data` 只在版本冲突这一种情况下新增载荷。

**集成方需要检查的**：任何把「HTTP 500」当作「该端点服务端坏了」来告警或兜底的客户端，
现在会收到 404/403/409/422/503。按 HTTP 语义分流的逻辑（4xx 不重试、5xx 才重试）在
本片之后才真正成立；此前把用户探测记成服务端故障的告警面板需要重新设阈值。

**防新增机制**：`itsm-backend/tests/contract/error_leak_ratchet_test.go` 冻结了
「把 `err.Error()` 当公共消息写进响应」的存量面（2026-10-04 实测 **421 处 / 37 个文件**，
§1.21 已把该基线收敛到 418 处 / 36 个文件，
前三名占近半数：`handlers/cmdb/production_service.go` 73、`handlers/bpmn/workflow.go` 67、
`handlers/notification/handler.go` 29）。计数上升即红；下降也必须先把基线改小并写进
CHANGELOG，不允许悄悄少几处却没人知道。重新采集：
`cd itsm-backend && ERROR_LEAK_RATCHET_DUMP=1 go test ./tests/contract/ -run TestErrorLeakRatchet -v`。

**验证**：
```bash
cd itsm-backend && go test ./common/ -run 'TestFailWithErr|TestRespondError|TestFail_Unauthorized|TestFailWithData'
cd itsm-backend && go test ./handlers/ticket_workflow/ -run TestHandler_AcceptTicket
cd itsm-backend && go test ./router/ -run 'TestTicketCCListRoutesEnvelope|TestTicketRuleAttachmentListRoutes'
cd itsm-backend && go test ./tests/contract/ -run TestErrorLeakRatchet
```

### 1.21 工单关联查询端点：跨租户读取通道关闭带来的响应变更（2026-10-04）

**受影响端点**（三条，均为 `itsm-backend/router/ticket_routes.go` 注册、前端 `ticket-relations-api.ts` 之外的真实入口）：

- `GET /api/v1/tickets/:id/relations`
- `GET /api/v1/tickets/:id/relations/stats`
- `GET /api/v1/tickets/:id/configuration-items`

**变更前实际行为。** `service/ticketassociationservice.go` 定位工单只用
`Where(ticket.ID(ticketID))`，没有任何租户谓词；`getChildrenTree` 进一步用
`Where(ticket.ParentTicketID(ticketID))` 做**全表扫描**。也就是说，只要另一租户的工单把
`parent_ticket_id` 指向本租户的工单 ID（该列是普通 int，没有跨租户外键约束保证），子工单的
`title`、`description`、`priority`、`status`、`assignedTo` 就会经由 `buildTicketResponse`
出现在本租户的响应里；`related_tickets` 自关联 M2M 与 `configuration_items` 边是同一形态的
第二、第三条读取通道。同时三条 handler 用 `common.Fail(c, common.InternalErrorCode,
err.Error())` 回应 service 错误，把「工单不存在」和「Ent 内部错误」一起写成 HTTP 500 /
业务码 5001，并把原始错误串透出。

**变更后行为。**

| 场景 | 变更前 | 变更后 |
| --- | --- | --- |
| 工单属于当前租户 | 200 / 0 | 200 / 0（不变） |
| 工单属于当前租户、无关联 | `{"childrenTree":null,"parentChain":null,"relatedTickets":[]}` | 三个集合都是 `[]` |
| 工单不存在 | 500 / 5001 + 原始 Ent 文本 | 404 / 4004 `工单不存在` |
| 工单属于其他租户 | 200 / 0 + **对方工单内容** | 404 / 4004，且响应体与「不存在」**逐字节相同** |
| `:id` 非法 | 400 / 1001 `invalid ticket id` | 400 / 1001 `无效的工单ID` |
| 缺少租户上下文 | 按无租户读取执行 | 401 / 2001（认证上下文解析失败即拒绝） |

逐字节相同是刻意的：如果跨租户返回 403 而不存在返回 404，调用方就能用状态码差异枚举其他
租户的工单 ID。回归测试把这条写成了 `assert.Equal(t, string(nfBody), string(body))`，不是
只比较状态码。

**集成方需要检查的**：

1. 任何依赖「工单不存在返回 500」的兜底或告警必须改判 404；这类请求现在根本不是服务端故障。
2. 任何此前从 `/relations` 拿到过跨租户子工单的部署，说明存在**脏数据**（`parent_ticket_id`
   跨租户指向）。修复后这些行不再出现在响应里，但库里仍然错着；清理属于存量数据修复，需按
   「停用不删、先给引用计数」另行约定，本片没有改动任何数据。
3. 父链遍历新增 64 层上限与环检测，超深/成环的存量数据不再导致无限递归或长时间请求。
4. `data: []` / 全 0 统计并不总是「确实没有关联」：handler 里还有一条 `else` 分支在关联服务
   未装配时返回假成功空结果，`stats` 有三个键恒为 0。已在 `docs/api-reference.md` 如实申报，
   收敛排期记为 E4-35b，不要把它当成可用能力。

**删掉的公共方法**（避免有人继续以为它们可用）：`GetTicketWithAssociations`、`GetTicketHierarchy`
与 `TicketHierarchy` 类型，全仓库零调用方，属于把跨租户读取通道固化下来的死门面。

**防新增机制**：本片把 `router/ticket_routes.go` 的 3 处 `common.Fail(c, InternalErrorCode,
err.Error())` 泄漏点改为 `common.RespondError`，使 `tests/contract/error_leak_ratchet_test.go`
的基线从 §1.20 记录的 421 处 / 37 个文件收敛到 **418 处 / 36 个文件**（该文件键被删除）。
这是棘轮第一次向下收敛——它本来在计数下降时也会红，必须同批把基线改小并写进文档，
不允许「悄悄少了几处」。
新增回归：`itsm-backend/router/ticket_relations_tenant_scope_route_test.go`（5 叶，打在
`SetupRoutes` 真实注册上）+ `service/ticketassociationservice_test.go` 的
`TestTicketAssociationService_RejectsForeignTenantTicketReads`。负证明实测（HEAD 实现 + 本片
新夹具，用 `git worktree` 在 HEAD 上单独跑）：5 叶里 4 叶转红——同租户正例拿到对方子工单、
跨租户三端点 200 且响应含 `PROBE-*` 探针行、不存在工单 500/`5001`、空关联实测
`{"childrenTree":null,"parentChain":null,"relatedTickets":[]}`；只有「非法 ID 400/`1001`」
锁的是既有行为因此常绿。修复后 5 叶全绿。

**验证**：
```bash
cd itsm-backend && go test ./service/ -run TestTicketAssociationService
cd itsm-backend && go test ./router/ -run 'TestTicketAssociationRoutes|TestTicketRelationsRoutesTenantScope'
cd itsm-backend && go test ./tests/contract/ -run TestErrorLeakRatchet
```

### 1.22 云账号/云服务/云资源只保留 `/api/v1/cmdb/cloud-*` 一套表面（2026-10-04，破坏性）

**删除的端点**（15 条，`itsm-backend/router/cloud_routes.go` 整个文件、`itsm-backend/handlers/cloud/`
包与 `itsm-backend/service/cloud_service.go` 一并删除）：

- `GET|POST /api/v1/cloud/accounts`、`GET|PUT|DELETE /api/v1/cloud/accounts/:id`
- `GET|POST /api/v1/cloud/services`、`GET|PUT|DELETE /api/v1/cloud/services/:id`
- `GET|POST /api/v1/cloud/resources`、`GET|PUT|DELETE /api/v1/cloud/resources/:id`

**为什么删。** 这 15 条与 `router/cmdb_routes.go` 里已注册的 `/api/v1/cmdb/cloud-accounts`、
`/api/v1/cmdb/cloud-services`、`/api/v1/cmdb/cloud-resources`（同样 15 条、同一组
`RequirePermission("cmdb", …)`）覆盖的是**同一个用例**，却各有独立的 handler、独立的 service、
独立的仓储查询与独立的 DTO。AGENTS.md 规定「一个用例只能有一个业务规则所有者」，两套并存
意味着同一个云资源列表在两条路径上有不同的租户口径、页长口径与错误口径，而调用方无从知道
自己在用哪一套。实测引用计数：前端 `src/` 对 `/api/v1/cloud/*` **零调用**（`CMDBApi` 自始只用
`/api/v1/cmdb/cloud-*`），`service.NewCloudService` 的构造点只有 `internal/bootstrap` 里那一条
`CloudHandler` 装配；因此删除的是无人使用的第二套实现，不是能力。

**迁移方式**（绝大多数调用只需换 URL 前缀）：

| 旧 | 新 |
| --- | --- |
| `GET /api/v1/cloud/accounts` | `GET /api/v1/cmdb/cloud-accounts` |
| `GET/PUT/DELETE /api/v1/cloud/accounts/:id` | `…/api/v1/cmdb/cloud-accounts/:id` |
| `GET /api/v1/cloud/services` | `GET /api/v1/cmdb/cloud-services` |
| `GET/PUT/DELETE /api/v1/cloud/services/:id` | `…/api/v1/cmdb/cloud-services/:id` |
| `GET /api/v1/cloud/resources` | `GET /api/v1/cmdb/cloud-resources` |
| `GET/PUT/DELETE /api/v1/cloud/resources/:id` | `…/api/v1/cmdb/cloud-resources/:id` |

请求体的 camelCase 字段名在两套表面上一致（`provider`/`accountId`/`accountName`/
`credentialRef`/`regionWhitelist`/`isActive`；云服务为 `serviceCode`/`serviceName`/
`resourceTypeCode`/`resourceTypeName`/`attributeSchema`/`parentId` 等），因此写入侧不需要改字段。

**响应侧的实际变化**（活表面自身被本片修掉的真缺陷，按端点列出）：

| 场景 | 变更前 | 变更后 |
| --- | --- | --- |
| `GET /cmdb/cloud-resources` 信封 | `{items,total,page,size}`，`q.All(ctx)` 整表读取、无 `Count`、无排序 | `{items,total,page,pageSize,totalPages}`，`Count` + `updated_at DESC, id ASC` |
| `GET /cmdb/cloud-resources` 页长 | `strconv.Atoi` 原值下传：`pageSize=0` → Ent 不加 LIMIT 返回整表，`page=-1` → 负 OFFSET，`abc` → 静默当 0 | `cmdbPagination` 单一所有者：缺省 1/20，越界或非数字回落 20 |
| `GET /cmdb/cloud-resources` 过滤 | `service_id`（snake_case）与 `serviceId` 同时接受；`accountId` 是无人读取的幽灵参数 | 只认 `serviceId`/`cloudAccountId`；`serviceId=abc` 返回 400 / code 1001 |
| `GET /cmdb/cloud-accounts`、`GET /cmdb/cloud-services` | 返回 `{items,total}`，但被前端当成可分页列表用 `Array.isArray ? … : items ?? data` 猜测形状 | 仍是 `{items,total}` 两键，并在文档里写明**不分页**（这两个端点是 CI 表单/云资源页的选择器数据源，整份返回）；前端的形状猜测已删除 |
| 三个列表的租户 | `c.GetInt("tenant_id")`：缺上下文返回 **0**，于是去查「租户 0」的数据 | `handlerctx.ResolveTenantID`：缺上下文 401、MSP 越权 403，fail closed |
| 三个列表的错误 | `common.InternalError(… + err.Error())` 把底层串透出 | `common.RespondError` / `common.ParamErrorWithErr`，原始串只进日志 |
| 云账号响应凭据 | 已只外露 `hasCredential` | 不变，且守卫搬进路由级契约测试（响应体断言不含 `credentialRef` 与密钥明文） |

**需要检查的集成方**：

1. 外部脚本/Webhook 若曾调用 `/api/v1/cloud/*`，现在得到 **404**（该组路由不再注册），请改前缀。
2. 若曾用 `?size=200` 或 `?limit=200` 一次性拉全云资源：`size`/`limit`/`offset` 都不参与分页，
   越界页长回落 20。需要整份数据请按页翻到底（前端参考实现：`CMDBApi.getAllCloudResources
   (params, maxRecords)`、`useAllCloudResourcesQuery`）。
3. 若曾用 `?service_id=3` 过滤云资源：别名已删除，请改 `serviceId=3`；否则该条件被忽略并返回
   该租户全部资源（而不是报错），这是移除别名后唯一的行为差异。
4. 写入侧一处**校验强度差异**需如实说明：被删表面在 `provider` 上带
   `oneof=aliyun tencent huawei aws azure onprem`，活表面的 `dto.CloudAccountRequest` /
   `dto.CloudServiceRequest` 只有 `required`。迁移到活表面后枚举校验变弱了，已登记为台账
   E4-20b 遗留项。**该差异已在 §1.23 补齐**（写入与查询两侧现在都认这六个规范值），
   因此原本合法的别名写入现在会变成 400，请按 §1.23 的口径核对存量数据与调用方。

**ACL/预检同步**：`middleware/rbac_precheck_gen.go`（`go run ./cmd/authz-gen` 重新生成，diff 为
纯删除 15 条）与手工维护的 `middleware/precheck_fallback.go`（删除 7 条 `/api/v1/cloud*` 兜底项，
`TestRoutePrecheckAlignment` 不会报告死兜底项，必须手删）一并收敛；`docs/acl-manifest.yaml` 与
`itsm-backend/docs/swagger.{json,yaml}` 已重新生成，不再声明这 15 条。

**防新增机制**：`itsm-backend/router/cmdb_cloud_list_route_test.go` 打在
`SetupRoutes` → `SetupCMDBRoutes` → `RequirePermission("cmdb", …)` 的真实注册上，解码目标为
测试本地 wire 结构（不引用 `dto`，因此修复前也能编译，证明的是运行时行为）。除信封/页长/别名/
租户断言外还锁了两条容易被忽略的路径：租户 B 用 `provider=aliyun` 过滤必须得到 `[]` 而不是跨
租户命中租户 A 的资源（`provider` 走云账号边），以及旧 `/api/v1/cloud/{accounts,services,
resources}` 三路径必须真实 404。
`tests/contract/error_leak_ratchet_test.go` 基线因此从 **418 处 / 36 个文件**收敛到
**415 处 / 36 个文件**（`handlers/cmdb/handler.go` 5 → 2）。
**负证明**：HEAD 实现 + 本片新夹具（`git worktree` 单独跑）修复前转红的正是这些断言——默认页长
返回全部 25 条（整表读取）、`page=-1` 得到负偏移页、`service_id` 别名仍然生效、云资源响应缺
`pageSize`/`totalPages` 键、旧路径 200 而非 404；修复后全绿。

**验证**：
```bash
cd itsm-backend && go test ./router/ -run 'TestCMDBCloud'
cd itsm-backend && go test ./handlers/cmdb/ ./dto/ ./middleware/ ./tests/contract/
cd itsm-frontend && npm run type-check && npm run lint:antd
cd itsm-frontend && npx jest src/lib/api/__tests__/cmdb-api.test.ts src/lib/__tests__/api-contract.test.ts
```

### 1.23 `/api/v1/cmdb/cloud-*` 补齐租户 fail-closed、404 语义与 provider 枚举（2026-10-04，破坏性）

§1.22 把云表面收敛成一套之后，活表面自身还留着三类问题；本片按台账 E4-20b/E4-20c 收口，
以下变更对**已在调用 `/api/v1/cmdb/cloud-*` 的集成方**是可见的。

**行为变更**（逐条实测，变更前取值来自 `HEAD` 的实现）：

| 场景 | 变更前 | 变更后 |
| --- | --- | --- |
| `provider` 写入（`POST`/`PUT` 云账号、云服务、云资源） | 只有 `required`，`alibaba`/`qcloud`/`private` 这类别名照样 200 落库，到发现阶段才表现为「没有适配器」 | `required,oneof=aliyun tencent huawei aws azure onprem`，别名一律 400 / code `1001`（绑定失败，公共文案 `Invalid request body`） |
| `provider` 查询（三个列表） | 无校验，拼错即静默返回空列表 | `omitempty,oneof=…`，省略=不过滤，别名 400 / code `1001` |
| `GET …/{id}` 命中他租户行 | `ent: *_not found` 未分类 → 500 / code `5001` + 「操作失败」 | 404 / code `4004`，文案按资源固定（`cloud service not found` 等） |
| `DELETE …/{id}` 命中 0 行 | 仓储只返回 `error`，0 行也是 `nil` → **200 报删除成功**，对方的行原封不动 | 返回受影响行数，0 行 → 404 / `4004` |
| `PUT …/{id}` 命中他租户行 | `UpdateOneID` 的 `ErrNotFound` 未分类 → 500 / `5001` | 404 / `4004`，且已实测对方的行保持原值 |
| 云资源写入引用他租户的 `cloudAccountId`/`serviceId` | 只按 ID 写入，不做归属校验（跨租户关系注入） | 各自返回 404 / `4004`；账号与服务厂商矛盾返回 400 / `4000` |
| CMDB 其余读取端点（17 处） | `c.GetInt("tenant_id")`，缺上下文退化为「租户 0」查询 | 全部改 `handlerctx.ResolveTenantID`，缺上下文 401 / `2001` fail closed |
| `GET /api/v1/cmdb/capabilities` 缺租户 | 该域自定义 401 / `2002`「缺少租户认证上下文」 | 与其他 CMDB 端点一致：401 / `2001`「租户信息缺失/租户上下文缺失」 |
| CMDB handler 的错误映射 | 域内私有 `failCMDBError` 用 `switch appErr.Code` 复刻一份「领域错误 → 状态码 + 文案」，是 §1.20 单一分类器之外的第二套真相 | 删除该函数，15 条云/发现端点统一走 `common.RespondError`；`errorLeakBaseline` 中 `handlers/cmdb/handler.go` 由 2 清零，棘轮基线 **415 处 / 36 文件 → 413 处 / 35 文件** |
| Swagger | 15 条活路由零覆盖（注解原本只写在已删除的 `handlers/cloud` 上） | 15 条全部有 `@Router`/`@Success`/`@Failure` 与响应模型；`docs/swagger.{json,yaml,go}` 重生成后 157→163 paths、216→231 operations、新增 11 个 definition、删除 0 个 |

**需要检查的集成方**：

1. 若曾用别名 `provider`（`alibaba`、`alicloud`、`qcloud`、`huaweicloud`、`tencentcloud`、`amazon`、
   `private`）写入或过滤，请改为六个规范值，否则现在是 400。别名只在
   `service/cloud.NormalizeProvider` 的适配器边界归一化，不再是 API 取值。
2. 存量数据库里若有别名厂商名，读取路径不受影响（查询不带 `provider` 过滤时仍返回该行），
   但**再次保存**会被拒绝。是否需要一次性数据订正请单独拍板；本片不动存量数据。
3. 若集成方曾把「跨租户 500」或「删除返回 200」当作轮询/幂等依据，需要改成按 404 处理。
   跨租户与真实不存在现在是同一份 404 报文，无法再据此区分。
4. 前端 `CMDBApi.getCloudServices('alibaba')` 这类调用（此前只存在于测试夹具）现在会得到 400，
   夹具已同步为 `aliyun`。

**未收口项（本片只登记，不改）**：`ListCloudAccounts`/`ListCloudServices` 仍是无上限整表读取，
「选择器端点」要不要资源上限或改真分页需产品拍板；`GET /api/v1/cmdb/discovery/sources`、
`/discovery/results` 返回裸数组（无信封键），后者的过滤参数仍绑定 snake_case `job_id`；
绑定失败的公共文案在仓库里仍是 `Invalid request body` 与 `请求参数错误` 两套并存。

**验证**：
```bash
cd itsm-backend && go test ./router/ -run 'TestCMDBCloud'
cd itsm-backend && go test ./handlers/cmdb/ ./dto/ ./tests/contract/ -count=1
cd itsm-frontend && npm run type-check && npm run lint:antd
cd itsm-frontend && npx jest src/lib/api/__tests__/cmdb-api.test.ts
```

契约测试：`router/cmdb_cloud_authz_route_test.go`（新增，打在真实 `SetupRoutes` +
`RequirePermission("cmdb", …)` + `TenantMiddleware` 上）。缺租户上下文那一组断言的是**链路级**
401/`2001`——`middleware.TenantMiddleware` 先于任何 handler 中止，因此 handler 内的
`ResolveTenantID` 分支在路由级不可达，该分支由 `handlers/cmdb` 的单元测试覆盖，
本片不用它伪造「回归」。

### 1.24 事件列表与活跃告警的页长规则收敛到 `common` 单一所有者（2026-10-04，破坏性）

台账 E4-9b。`GET /api/v1/incidents` 和 `GET /api/v1/incidents/alerts/active` 此前各自解析
`page`/`pageSize`，与平台规则（`common.GetPaginationFromQuery`：缺省 1/20，只采纳 `(0,100]`，
越界或非数字**回落缺省页长**而非夹到上限）不一致。

**行为变更**（变更前取值来自 `HEAD` 的实现，逐例实测）：

| 请求 | 变更前 | 变更后 |
| --- | --- | --- |
| `GET /api/v1/incidents`（不带分页参数） | 10 条 | 20 条（平台缺省） |
| `pageSize=abc` / `0` / 负数 | `strconv.Atoi` 的错误被丢弃 ⇒ `Limit(0)`，Ent 在该分支**不加 `LIMIT`** ⇒ 整租户表一次返回（实测 105 条种子全量返回，响应还回显 `pageSize:0`） | 回落 20，且响应回显的就是实际采纳值 |
| `pageSize=500` | 不设上限，SQL 真去取 500 条 | 回落 20 |
| `page=0` / `page=-5` | 算出负 `OFFSET`；SQLite 当 0 吞掉，**Postgres 按规范报 `OFFSET must not be negative` ⇒ 500** | 回落第 1 页 |
| `GET /api/v1/incidents/alerts/active`（不带分页参数） | 10 条（handler 与 service **各写一份** `1/10/100` 字面量，同一规则两处真相） | 20 条，两层都只读 `common` 的常量 |
| `…alerts/active?pageSize=5000` | 夹到 100（handler 侧夹紧，service 侧再夹一次） | 回落 20（平台语义） |

`docs/api-reference.md` 事件段原先写的「默认 10，上限 200」与两份实现都不符（代码里从未出现
200 这个上限），已按实测改写；swagger 注解的 `default(10)` 同批改为 `default(1)` /
`default(20)`，`docs/swagger.{go,json,yaml}` 已在同一提交重生成。

**需要检查的集成方**：

1. 依赖「不传 `pageSize` 就是 10 条」的调用方现在会拿到 20 条。要固定页长请显式传 `pageSize`。
2. 若曾用 `pageSize` 传一个足够大的值来「一次拉完」，该写法现在失效（回落到 20）；
   请改为按 `page` 翻页，页长上限是 100。
3. 非数字页长在旧实现里是整表读取通道，修复后同一请求只返回 20 条。若之前有集成方在
   依赖这个错误行为做全量导出，需要改成分页拉取。

**未收口项（本片只登记，不改）**：`common.GetPaginationFromQuery`（越界回落 20，26 个非测试调用点）
与 `common.ValidatePagination`（越界夹到 100，19 个调用点）仍是两套夹紧语义，归一要一次性核对
全部既有端点与前端分页预期，属独立批次；`handlers/`/`controller/` 内仍有 11 处自建
`DefaultQuery("pageSize", <字面量>)`（分布在 8 个文件），本片新建了双向棘轮
`tests/contract/page_size_owner_ratchet_test.go` 把它们钉在基线内、新增即红，按域逐片收敛。

**验证**：
```bash
cd itsm-backend && go test ./handlers/incident/ -run 'TestList_|TestActiveAlerts_' -count=1
cd itsm-backend && go test ./tests/contract/ -run 'TestPageSizeSingleOwnerRatchet' -count=1
cd itsm-frontend && npm run type-check && npm run test:unit -- --runTestsByPath src/lib/__tests__/api-contract.test.ts
```

回归形态说明：`handlers/incident/list_pagination_contract_test.go`（7 例）锁缺省页长、整表读取
通道、非数字/零/负值回落、100 边界与回显一致性；`handlers/incident/alert_pagination_route_test.go`
（1 例 6 子例）打在真实 `IncidentHandler` + 真实 `IncidentAlertingService` 装配上（本域路由未走
`router.SetupRoutes`，夹具自行注册该路径并注入租户上下文，因此证明的是 handler+service 两层
只剩一套规则，不是路由级装配）。**负证明**：把 `handlers/incident/handler.go` 与
`service/incident_alerting_service.go` 还原到 `HEAD` 后，10 条断言转红（最直观一条是
`should have 20 item(s), but has 105`）；另有 3 例修复前后皆绿，锁的是既有正确行为。
前端 `incident-api.ts` 同批删除对信封键的 `?? ` 猜测与自算 `totalPages`，改为直接透传
`PaginationResponse<Incident>`，因此后端缺省页长的变化会立刻在契约测试里可见，而不是被前端兜底掩盖。

### 1.25 `GET /api/v1/ai/audit-logs` 的页长与信封收敛（2026-10-04，破坏性）

AI 审计日志端点的页长规则原先有三处真相，且响应信封只有四键：

| 输入 | 修复前实测 | 修复后 |
|:---|:---|:---|
| 不带 `pageSize` | handler 缺省 20、service 缺省也是 20（巧合一致） | 20（`common.DefaultPageSize`） |
| `pageSize=150` | 落在 service 自建的 `auditMaxPageSize=200` 上界内 ⇒ 一次返回 130 条并回显 `pageSize:150` | 20（平台只采纳 `(0,100]`） |
| `pageSize=300` | service 把越界值改回 20，**信封却回显 300** ⇒ 客户端按 300 条/页去翻页 | 20，且回显 20 |
| `pageSize=abc` / `0` | handler 回落 20 | 20 |
| `page=-3` / `abc` | handler 回落 1 | 1 |
| 响应 `data` 键 | `{items,total,page,pageSize}`（无 `totalPages`） | `{items,total,page,pageSize,totalPages}` |
| 缺租户上下文 | 200 长度之外还出现**拼接的两个 JSON 文档**：`{"code":2001,…}{"code":2002,"message":"未授权访问"}` | 单个 401/`2001` 文档 |

最后一条不是分页问题而是响应完整性问题：`handlerctx.ResolveTenantID` 在拒绝时已经写过一份
401 并终止请求（`middleware.AbortIfTenantError` 必然写响应），handler 紧接着又 `common.Fail` 写第二份。
`net/http` 只会把第二份的 status 记成 superfluous WriteHeader，**两份 JSON 字节都会发给客户端**，
任何严格 JSON 解析都会失败。本批把这一处（`handlers/ai/handler.go:620`）的重复写删掉；
同一形态在全仓 `handlers/` 修复后仍剩 **24 处 / 6 个文件**（`ticket_type` 10、`ai` 4、`operations` 4、
`ticket_rating` 3、`timer` 3、`skill` 1），登记为台账 E4-46 交后续统一收敛，本批未动。

**集成检查**：

1. 依赖 `pageSize` 传大值「一次拉完」的调用失效（现回落 20），需改为按 `page` 翻页，上限 100。
2. 直接解析 `data` 四键的客户端需接受新增的 `totalPages`；集合键名 `items` 未变。
3. 非 HTTP 调用方（`service.AITelemetryService.ListAuditLogs`）的页长语义随之改变：`pageSize>100`
   现在是**夹到 100**（`common.ValidatePagination`），而不是旧实现的「>200 才回落 20、
   (100,200] 照原值执行」。

**验证**：
```bash
cd itsm-backend && go test ./tests/contract/ -run TestAIAuditLogs -count=1
cd itsm-frontend && npm run type-check && npx jest src/lib/api/__tests__/ai-api.test.ts
```

回归形态说明：`tests/contract/ai_audit_logs_envelope_test.go` 8 例（回显一例覆盖 `pageSize=5000`、
`pageSize=abc`、`pageSize=0`、`page=-3` 四组输入）打在按 `router/ai_routes.go:37` 与
`internal/bootstrap/app.go:944` 复现的注册与装配上（`RequirePermission` 不在信封契约范围内，夹具自行注入租户上下文），
锁五键精确集合、缺省页长 20、越界页长无法越过平台边界、回显等于实际采纳值、第二页余数、
租户收敛、缺租户只写一份响应、空结果序列化 `[]`。**负证明**：在 HEAD 的独立工作区里跑同一份夹具
（只拷入本批新增测试，实现全部是修复前形态），8 例中 **7 例转红**，实测报文分别是
`should have 20 item(s), but has 130`（`pageSize=150` 整段读出）、回显 `pageSize` 为 `150`/`5000`
（信封声明与真实读取分家）、`data` 键集合缺 `totalPages`（含空结果一例），以及缺租户响应体
`{"code":2001,"message":"租户上下文缺失"}{"code":2002,"message":"未授权访问"}` 解析失败
（`invalid character '{' after top-level value`）；唯一修复前后皆绿的一例锁的是既有的租户收敛行为。
前端 `AIAuditLogsResponse` 同批改为复用平台 `PaginationResponse<AIAuditEntry>`，
`ai/audit/page.tsx` 删掉 `res.items ?? []`、`res.page || page`、`res.total ?? 0` 三处兜底。

### 1.26 BPMN 审计日志与流程实例列表的页长收敛（2026-10-04，破坏性）

`GET /api/v1/bpmn/dashboard/audit-logs`、`GET /api/v1/bpmn/monitoring/audit-logs` 与
`GET /api/v1/bpmn/monitoring/instances/status` 的页长原先有两套 HTTP 解析加两套 service 规则，
其中 service 的分页分支是条件式的（`if req.Page > 0 && req.PageSize > 0 { Offset/Limit }`），
条件不成立时**整条 `LIMIT`/`OFFSET` 子句消失**。HEAD 实测（单租户 25 条种子）：

| 输入 | 修复前实测 | 修复后 |
|:---|:---|:---|
| 不带 `page`/`pageSize` | dashboard 20（`==0` 补默认）、monitoring 20 | 20（`common.DefaultPageSize`） |
| `pageSize=150` / `5000` | 按原值进 `LIMIT`，25 条一次返回；信封却由 `common.SuccessWithList` 内的另一套夹紧回显 `pageSize:100`、`totalPages:1` | 20，且回显 20 |
| `pageSize=-5` | dashboard 只判 `==0`，负值原样下传 ⇒ 分页分支不成立 ⇒ **该租户审计整表返回**（实测 25 条） | 20 |
| `pageSize=abc` | dashboard 回落 20；monitoring 的 `DefaultQuery` + 丢弃 `Atoi` 错误得到 **0** ⇒ 同样整表返回 | 20 |
| `page=abc`（`pageSize=10`） | dashboard 解析失败即保留缺省、回落第 1 页；monitoring 的 `DefaultQuery` + 丢弃错误得到 **0** ⇒ 分页分支不成立 ⇒ 整表返回 25 条 | 第 1 页、10 条 |
| `page=-3`（`pageSize=10`） | dashboard 只补 `==0`，负页码原样下传 ⇒ 分页分支不成立 ⇒ 整表返回 25 条 | 第 1 页、10 条 |
| 非 HTTP 调用方传零值 | `ListProcessInstancesStatus` 自建缺省 1/20 且**无上界**；审计查询走「条件不成立即不分页」 | `common.ValidatePagination`（`page<=0→1`、`pageSize<=0→20`、`>100→100`），任何取值都带 `LIMIT` |

**集成检查**：

1. 用 `pageSize=5000` 之类越界值表达「一次拉完」的调用现在拿到 20 条，需按 `page` 翻页（上限 100）。
2. 依赖「不传分页参数即返回全部」的调用行为改变：过去是整表读取，现在返回第 1 页 20 条，
   需要全量请显式逐页读取或走导出类端点。
3. 三个端点的响应键集合本就已是标准五键（`items,total,page,pageSize,totalPages`），未改名；
   变化只在 `page`/`pageSize`/`totalPages` 的**取值**与 `items` 的行数。

**验证**：
```bash
cd itsm-backend && go test ./handlers/bpmn/ -count=1
```

回归形态说明：`handlers/bpmn/audit_pagination_contract_test.go`（6 例）打在
`DashboardHandler.RegisterRoutes` + `MonitoringHandler.RegisterRoutes` 的真实注册上，锁默认页长 20、
越界与非法页长一律回落且只回 20 条、非法页码回落第 1 页、末页余数如实 5 条、
越界页长下租户 2 仍只见自己的 3 条，以及回显页长等于实际进入 `LIMIT` 的值。
**负证明**：在 HEAD 的独立工作区只拷入这两份测试文件（实现全是修复前形态）跑同一夹具，
6 例转红 3 例，实测报文为 `should have 20 item(s), but has 25`（`pageSize=abc/-5/150/5000`、
`page=abc&pageSize=10`、`page=-3&pageSize=10` 各自的整表读数）与
`expected: 20 / actual: 100`（`pageSize` 回显）、`expected: 2 / actual: 1`（`totalPages`）；
修复前后皆绿的 3 例锁的是既有默认页长、末页余数与租户收敛行为。

同批实测登记而未改的三项（台账 **E4-48**，已于次日收口，见 §1.29）：dashboard 与 monitoring 的 `audit-logs`
是同一用例的两套 HTTP 表面（后者委托到同一个 `BPMNAuditService.QueryAuditLogs`）；
两条路由注册时都没有 `RequirePermission`，只做认证与租户范围收敛；
`dashboard.go` 直接序列化 Ent 模型，响应项的键是 `process_instance_key`、`activity_id`、
`tenant_id` 这类 snake_case 且把租户 ID 本身发给前端（违反 camelCase 与 DTO 规则）。
本批只收页长，未触碰表面归属、鉴权声明与 DTO 序列化。

### 1.27 `handlerctx.Resolve*` 拒绝后不再补写第二份响应（2026-10-04）

E4-45 在 `GET /api/v1/ai/audit-logs` 上实测到一个响应完整性缺陷：
`handlerctx.ResolveTenantID(c)` 失败时委托 `middleware.AbortIfTenantError`
（`middleware/msp_tenant_resolver.go`），后者对 401/403/500 三种情形都**已写体并 abort**；
但 handler 形如：

```go
tenantID, ok := handlerctx.ResolveTenantID(c)
if !ok {
    common.Fail(c, common.UnauthorizedCode, "未授权访问")
    return
}
```

实际上是**两次写响应**——`common.Fail` 写第二份 `{code:2002,...}` 拼接在已写体之后，
客户端按 `{code,message,data}` 严格解析时报 `invalid character '{' after top-level value`。

成功路径完全正常，所以此前从未被报成 bug——但任何按平台信封解析的集成方在这 24 个端点的
**鉴权失败路径**上都会拿到无法解析的响应。

**变更范围**（24 处 / 6 文件，仅删除双写，不改文案、不改 status、不改业务规则）：

| 文件 | 删除的 `!ok` 写响应行 |
|:---|:---|
| `itsm-backend/handlers/ticket_type/handler.go` | 9 |
| `itsm-backend/handlers/ai/handler.go` | 4 |
| `itsm-backend/handlers/operations/handler.go` | 4 |
| `itsm-backend/handlers/ticket_rating/handler.go` | 3 |
| `itsm-backend/handlers/timer/handler.go` | 3 |
| `itsm-backend/handlers/skill/handler.go` | 1 |

全部统一收敛为：

```go
tenantID, ok := handlerctx.ResolveTenantID(c)
if !ok {
    return
}
```

并在 `common/handlerctx/handlerctx.go` 的 `ResolveTenantID` / `RequireTenantID` 函数注释里
写明「失败时 `middleware.AbortIfTenantError` **已写响应并 abort**；调用方在 `!ok` 分支
**禁止**再调用 `common.Fail` / `common.FailWithErr` / `common.ParamError` / `c.JSON` /
`c.AbortWithStatusJSON`，只能 `return`」。

**集成检查**：

1. 任何抓 `*gin.Context` 自渲染的中间件若在 `handlerctx.Resolve*` 之后又读 `c.Writer.Written()`
   来判定「是否写过响应」：现在更可靠——失败时一定 `Written() > 0`，成功时一定为 0。
2. 跨域统一文案（`未授权访问` vs `租户上下文缺失` vs E4-40③ 的 `Invalid request body`/
   `请求参数错误`）是同一类问题，但**本批不动**——只删除双写、不改文案。集成方在
   401/403 路径上仍可能看到两种文案，需等后续单独批次统一。
3. 鉴权失败时实际响应的 HTTP status / 业务 code 全部由 `middleware.AbortIfTenantError` 决定
   （401 / code 2001、403 / code 2003、500 / code 5000），本批不引入新 status / 新 code。

**验证**：

```bash
cd itsm-backend && go test ./tests/contract/ ./common/... ./handlers/... ./service/ -count=1 -run 'HandlerCtx|PageSize|ListEnvelope|AuditLogs|Pagination'
```

回归 `tests/contract/handlerctx_no_double_write_test.go`（双向 AST 棘轮）：
所有 `handlers/*.go` 里 `tenantID, ok := handlerctx.Resolve*` / `userID, ok := handlerctx.Resolve*` /
`tenantID, ok := handlerctx.Require*` 形态的 `!ok` 分支必须只含 `return`、不得包含
`common.Fail` / `common.FailWithErr` / `common.ParamError` / `c.JSON` / `c.AbortWithStatusJSON`
等任何再写响应调用；同时校验 `handlerctx.Resolve*` / `Require*` 自身的契约注释 token
（防止未来给 `Resolve*` 增一个不写响应的返回值时这条注释被悄悄删除）。

**负证明**：把任一文件改回 `common.Fail`（或任何再写响应调用）即红。

**未触碰**（实测登记、本批范围外）：

- `handlers/sla_template/handler.go:32-34` 仍以 `common.Fail(..., AuthFailedCode, ...)` 写第二份（1 处未收口）。
- `handlers/dashboard_handler.go:528-530` 的 `!ok` 分支体不是裸 `return` 而是给本地变量赋空值（按上下文判断是否构成双写，需读全函数后才能定）。

### 1.28 流程实例端点统一以 `processInstanceId`（PI-* 业务键）寻址（2026-10-05，破坏性）

`/api/v1/bpmn/process-instances/:id` 家族此前存在**双寻址键**缺陷：同族六个端点里只有
`GET` 详情把 `:id` 当**数字 Ent 主键**（`strconv.Atoi` 后按 `ID` 查），而
`suspend` / `resume` / `terminate` / `variables` / `approval-history` 与
`monitoring/instances/:instanceId/timeline` 全部按 **PI-* 业务键**（`process_instance_id` 列，
schema 全局唯一）寻址。调用方拿到列表响应里的 `instanceId` 再去打 GET 详情必然 404——
这正是 ga-gate 全链路 E2E（`ticket-type-full-chain.spec.ts` 轮询实例状态至 completed）
连续红的根因。

**变更**：

1. `service.BPMNProcessInstanceService.GetProcessInstance` / `GetProcessInstanceHistory`
   改按 PI 业务键 + 租户谓词查询。**数字 Ent ID 从此不是任何端点的寻址键**，用数字打
   `GET /process-instances/:id` 或 `/:id/approval-history` 返回 404（fail-closed，不再
   猜测调用方意图）。
2. `POST /api/v1/bpmn/process-instances`（启动流程）的响应从直接序列化 Ent 模型改为
   `dto.BPMNProcessInstanceResponse`——响应键从 snake_case（`process_instance_id`）修正为
   camelCase（`instanceId`），与 GET/List 同族契约一致。
3. 前端 `WorkflowApi.getInstances` / `getInstance` / `startWorkflow` 的实例标识只读
   `item.instanceId`（删除 `instanceId || id` 多字段兜底）；审批中心待办的流程实例深链
   从数字 `task.processInstanceId` 改为 `task.processInstanceKey`（PI）。

**集成检查**：

- 任何外部集成若曾依赖「GET 详情可用数字 ID 命中」：改为使用列表/创建响应中的
  `instanceId`（PI 键）。响应 DTO 的 `id` 字段仍是数字 Ent ID，仅作展示与关联用途。
- 解析 `POST /process-instances` 响应的代码需从 snake_case 键改为 camelCase 键。

**验证**：

```bash
cd itsm-backend && go test ./service/ ./handlers/bpmn/ ./tests/contract/ -count=1
```

回归锁：`service/bpmn_process_instance_get_test.go`（PI 命中 / 跨租户拒绝 / **数字 ID
必须 404**）、`service/bpmn_process_instance_history_test.go`（approval-history 同键）。
负证明：把查询改回按 `ID` 即红。

### 1.29 BPMN 流程审计收敛为单一表面 + 权限门 + camelCase DTO（2026-10-05，破坏性）

台账 E4-48（§1.26 同批登记）的收口。此前「流程审计日志读取」同时存在两套已注册 HTTP
表面、两条都没有 `RequirePermission`，dashboard 那条还把 Ent 模型直接序列化给前端：

- `GET /api/v1/bpmn/dashboard/audit-logs`：响应项键为 snake_case（`process_instance_key`、
  `activity_id`、`tenant_id`），并把租户 ID 本身发给前端；
- `GET /api/v1/bpmn/monitoring/audit-logs`：委托到同一个 `BPMNAuditService.QueryAuditLogs`，
  但把行降维成 `resourceType/resourceId` 弱结构；
- 任何已认证且拿到租户上下文的用户都能读全量流程审计；
- 唯一生产消费方 `/workflow/audit` 页面按 camelCase 类型读取，「流程实例/活动/操作人/受理人」
  四列恒空，只能靠 `normalizeAuditLog` 兜底，并额外在查询串里自报 `tenantId`。

**变更**：

1. **唯一表面**是 `GET /api/v1/bpmn/dashboard/audit-logs`（及 `audit-logs/user/:userId`）。
   `GET /api/v1/bpmn/monitoring/audit-logs` 路由、`MonitoringHandler.GetAuditLogs`、
   `BPMNMonitoringService.GetAuditLogs` 与 `AuditLogRequest` 全部删除，调用返回 404。
2. **权限门**：两条 dashboard 审计路由挂 `RequirePermission("bpmn","read")`，
   `rbac_precheck_gen.go` 同批再生成。无 `bpmn:read` 的角色从「任意已认证用户可读」变为
   403 / code 2003；缺认证上下文 fail-closed 401 / code 2001。
3. **DTO**：响应项改为 `dto.ProcessAuditLogResponse`，键集合恰好为 camelCase
   （`processInstanceKey`、`activityId`、`userName`、`assigneeId`、`durationMs` 等 21 键），
   **不再包含 `tenant_id`/`tenantId`**。
4. 死路由 `GET /api/v1/bpmn/dashboard/audit-logs/timeline` 一并删除（注册的参数名与 handler
   读取的 `process_instance_key` 从不匹配，任何调用恒返回 1001，零生产消费方）；
   时间线唯一表面为 `GET /api/v1/bpmn/monitoring/instances/:instanceId/timeline`。
5. 前端 `ProcessAuditLog` 类型删去 `tenantId`，`QueryAuditLogsRequest` 删去 `tenantId?`，
   `getUserActivity` 不再发送 `tenantId` 查询参数；`/workflow/audit` 页面删除
   `normalizeAuditLog` 同键自我复制的兜底与「无 tenantId 就报错」的本地门禁。

**集成检查**：

- 解析 `dashboard/audit-logs` 响应的代码需从 snake_case 键改为 camelCase 键；
  依赖响应中 `tenant_id` 的调用方需改为使用自身认证上下文的租户。
- 调用 `monitoring/audit-logs` 的集成需切换到 `dashboard/audit-logs`（参数集不变：
  `processInstanceId`/`processDefinitionKey`/`action`/`userId`/`activityType`/
  `startTime`/`endTime`/`page`/`pageSize`，时间仍按 `2006-01-02` 日期解析、非法值静默忽略）。
- 以 `msp_viewer` 等非 BPMN 角色轮询审计接口的自动化任务将开始收到 403，需为角色补
  `bpmn:read` 权限或改用有权限的服务账号。

**验证**：

```bash
cd itsm-backend && go test ./handlers/bpmn/ ./service/ ./tests/contract/ ./middleware/ -count=1
```

回归锁：`handlers/bpmn/audit_single_surface_contract_test.go`（重复表面 404 / camelCase
精确键集合且无 tenantId / 权限门 200-403-401 三态）、
`handlers/bpmn/audit_pagination_contract_test.go`（页长与租户收敛，打在真实注册路由上）。

### 1.30 SLA / Skill / 工作台列表页长收敛到单一所有者（2026-10-05，破坏性）

台账 E4-47①–④。`/api/v1/sla/definitions`、`/api/v1/sla/violations`、
`/api/v1/sla/performance`、`/api/v1/sla/alert-history`、`/api/v1/skills`、
`/api/v1/admin/skills`、`/api/v1/workbench` 的页长此前分散在 4 套私有实现里：
sla handler 缺省 10/20 不一且接受 `size` 别名、**没有任何上界**（`pageSize=5000`
原样进 `LIMIT`；`alert-history` 的 `pageSize=abc` 解析成 0 后 `LIMIT` 子句整体消失，
一次写错分页参数即读穿该租户全表）；sla service 又私有夹紧 `>200 → 200`（200 不是
平台值）；skill 私有把 `>100` 截断成 100；workbench 手抄一份 `(0,100]` 判定。

**变更**：

1. 七个端点的 HTTP 入口统一走 `common.GetPaginationFromQuery`：缺省 `page=1`、
   `pageSize=20`，只采纳 `[1,100]`，越界（`150`/`5000`）、非数字（`abc`）、`0`/负数
   一律**回落 20**（不是夹紧），页码下界 1；响应回显即真正下传给 `Offset/Limit` 的值。
2. `GET /api/v1/sla/definitions` 缺省页长由 **10 改为 20**；`size` 查询参数在
   definitions/violations 上**不再被接受**（此前与 `pageSize` 双轨）。
3. `GET /api/v1/sla/performance` 的非法 `page`/`pageSize`（非数字、0、负数）此前返回
   400/1001，现回落缺省 1/20（与平台其它列表端点一致）；`pageSize` 在 `(100,200]`
   此前按原值执行、`>200` 执行 200，现一律回落 20。
4. `GET /api/v1/skills` 与 `/api/v1/admin/skills` 的 `pageSize>100` 此前夹到 100，
   现回落 20。
5. `GET /api/v1/workbench` 行为语义不变（越界本就回落缺省），收敛的是所有权。
6. 信封：sla 三个列表端点（definitions/violations/alert-history）从四键手拼改为
   平台五键 `{items,total,page,pageSize,totalPages}`（新增 `totalPages`）；
   violations/alert-history 空结果此前序列化为 `items: null`，现为 `items: []`。

**集成检查**：

- 仍发送 `size` 的调用方需改为 `pageSize`（仓库内前端实测无此发送点，历史调用面为
  `handlers/sla/handler_test.go`，已同批改走 `pageSize`）。
- 依赖「`/sla/performance` 写错页长返回 400」做输入校验的调用方需改用显式校验；
  现在该请求会成功并返回缺省页。
- 期望 `/sla/definitions` 每页 10 条的调用方需显式传 `pageSize=10`。

**验证**：

```bash
cd itsm-backend && go test ./handlers/sla/ ./handlers/skill/ ./handlers/workbench/ ./tests/contract/ -count=1
```

回归锁：`handlers/sla/pagination_contract_test.go`（缺省 1/20、越界/非数字/零负回落且
回显即下传值、`size` 别名失效、末页余数、performance 非法页长不再 400）、
`handlers/skill/handler_test.go` 与 `handlers/workbench/handler_test.go` 各新增页长回落例；
`tests/contract/page_size_owner_ratchet_test.go` 基线同批 17→11 行。

### 1.31 内置角色权限码集扩权 + 移除双权威硬编码兜底表（2026-10-05，N4/N5 拍板）

`plans/product-remediation-plan-2026-10-03.md` §3 N4（选"扩权"）与 N5（选"移除"）落地。

**变更内容**

1. **N4 扩权**：`authz.BuiltinRolePermissionCodes()` 按 R2-a 实测差集补齐
   `manager`（26 码，含 `dashboard:read`/`release:*`/`group:*`/`org:*` 等经理全域视角）、
   `agent`（7 码：`notification:write`、`dashboard:read`、`service_catalog:read`、
   `change:write`、`alert:read`、`group:read`、`bpmn:read`）、
   `end_user`（16 码，ITIL 读面 + 仪表盘/通知/AI 基线）。播种契约为「只增不减」，
   存量库下次启动自动补齐 `role_permissions` 行，无需手工 SQL。
2. **N5 双权威移除**：`middleware.RolePermissions` 硬编码表删除。DBOnly `unconfigured`
   兜底、`HardcodeOnly`/`Merge`/`Fallback` 模式、菜单构建与 `/auth/me` permissions 展示面
   统一改为 `authz.RolePermissionDefaults()`——内置角色直接派生自播种码集本身，
   兜底与播种结构性同源，不可能再漂移。
3. **拼写收敛**：`agent` 不再授予 `alerts:read`（路由预检实测零条 `alerts:*` 声明，
   统一为 `alert:read`）。`security`（legacy users.role 存量值）与 `msp_*` 五个运行时
   角色的默认集在 authz 侧显式保留，行为不变。

**行为变化与升级检查**

- `/auth/me` 与菜单可见性中，manager/agent/end_user 的权限码数组将按扩权结果变多；
  依赖 permissions 数组做按钮门控的前端定制逻辑需回归验证（本项目前端按后端返回渲染，预期无需改动）。
- `sysadmin` 在 `unconfigured` 兜底态的授权从 `*:*` 通配收敛为枚举码集（与 `configured`
  态 DB 实际授权一致）。全新安装若 RBAC 播种失败，sysadmin 不再自动获得码空间之外的能力——
  属预期收紧；播种失败应看启动日志 `seedRolePermissions` 告警。
- 守卫变化：`pkg/seeder/role_permission_guard_test.go` 的 R2-a `t.Skipf` 待拍板档退役，
  替换为「N4 扩权回归锁」+「单一真源锁」（兜底默认 ⊆ DB 码集，双权威回潮即 CI 红）。

### 1.32 users.username 改为按租户组合唯一（2026-10-06，含 schema 迁移）

修复「第二个租户开不出管理员」。产品基线要求每个租户的管理员都叫 `admin`，而
`users.username` 原本是字段级全局唯一键，跨租户必然冲突。

**变更内容**

1. `ent/schema/user.go`：`username` 去掉字段级 `.Unique()`，新增
   `Indexes().Unique().Fields("tenant_id", "username")`；`email` 保持全局唯一。
2. 新增迁移 `itsm-backend/migrations/20261006_tenant_scope_user_username_unique.sql`
   （配套 `_down.sql`）：先按 `(tenant_id, username)` 检测存量重名，有重名直接
   `RAISE EXCEPTION`（ERRCODE `unique_violation`）中止，再 drop 旧全局唯一键
   `users_username_key` 并建 `user_tenant_id_username`。
3. `service/user_service.go`：`CreateUser` / `UpdateUser` 的用户名查重加上
   `tenant_id` 谓词；邮箱查重保持全局（与找回密码按 email 定位一致）。
4. `handlers/common`：登录必须落在唯一账号上。`tenantCode` 解析失败不再降级为
   跨租户用户名匹配，按凭证错误拒绝；用户名跨租户重名且未指定租户时返回
   「该用户名在多个租户下存在，请选择租户后登录」（业务码 1001 参数错误），
   不再返回「用户名或密码错误」。

**升级影响**

- 迁移由 bootstrap job 自动执行（`ITSM_BOOTSTRAP_ONLY=true ITSM_AUTO_MIGRATE=true`），
  无需手工命令；执行顺序是先 ent 建组合索引、后本迁移删旧全局键，两者都幂等。
- 存量库前提（已在演练副本实测）：`(tenant_id, username)`、全局 `username`、
  全局 `email` 三项重复均为 0 行，因此不需要消解数据。若你的库检测出重名，
  迁移会明确报错并指出组数，此时先改名再升级。
- 登录接口对外契约新增一种失败消息；前端登录页目前只提交用户名/口令，
  单租户与「用户名恰好唯一」的部署行为不变。多租户 SaaS 要在登录页补租户选择器，
  否则重名账号（如各租户的 `admin`）会收到上面的消歧提示。
- 回滚只在前滚不可行时使用：`_down.sql` 会先检测全局重名，跨租户同名 `admin`
  是本次放宽后的合法状态，因此回滚几乎必然被拒绝——正确处置是前滚修复。

**验证**

```bash
cd itsm-backend
# 迁移在真实 PostgreSQL 上的三条证明（旧形状复现冲突 / 重复检测中止 / 与 ent 生成索引同名）
docker run -d --rm --name itsm-drill-uq -e POSTGRES_USER=drill -e POSTGRES_PASSWORD=drill \
  -e POSTGRES_DB=itsm_init_test_users --tmpfs /var/lib/postgresql/data:rw,mode=1777 \
  -p 127.0.0.1:5499:5432 pgvector/pgvector:pg17
ITSM_INITIALIZATION_TEST_DSN="postgres://drill:drill@127.0.0.1:5499/itsm_init_test_users?sslmode=disable" \
  go test ./migration/ -run UserUsername -v
go test ./handlers/common/ -run TestLogin -v
```

### 1.33 角色词表重设与 legacy 角色退役（2026-10-08，破坏性：角色取值 + 数据清理）

角色=权限包，不是岗位。此前 `users.role` 枚举、`roles` 表种子、`internal/authz` 兜底表、
内置审批组与 BPMN 模板各自持有一套词汇，交集很小：审批派单指向 `l1_support` / `it_director`
这类库里查不到人的角色码，`security` 这类 legacy 值又一路泄漏到界面。现在词表集中在
`itsm-backend/domain/role`，其余各层由守卫强制对齐。

**新的封闭词表**

| 层 | 取值 | 承载 |
| --- | --- | --- |
| 主角色 | `super_admin` `admin` `sysadmin` `security_admin` `audit_admin` `it_admin` `manager` `agent` `technician` `end_user` | `users.role` 单字段（一人一个），同时是 `roles` 表的 `is_system` 种子 |
| ITIL 实践角色 | `change_manager` `problem_manager` `knowledge_manager` `cmdb_admin` `service_catalog_admin` | 仅 `roles` + `user_roles` 边，可叠加 |
| MSP 协作角色 | `msp_viewer` `msp_tech` `msp_specialist` `msp_manager` `msp_admin` | 同上 |

`sysadmin` / `security_admin` / `audit_admin` 是等保「三员分立」的三个互斥管理角色。

**退役值**：`security` `it_director` `ops_director` `ops_manager` `ops_engineer` `dba`
`network_eng` `sd_manager` `l1_support` `l2_support` `l3_expert` `rd_manager` `developer`
`qa_engineer` `dept_manager` `team_lead` `guest`。岗位/团队差异请改用部门、团队或租户自建角色。

**升级影响**

- `POST/PUT /api/v1/users` 的 `role` 只接受上表主角色（外加前端别名 `user`，服务端归一为
  `end_user`）。传入退役值现在返回参数错误，不再落库。
- 迁移 `itsm-backend/migrations/20261008_retire_legacy_roles.sql` 由新版本镜像启动时的
  AutoMigrate 自动应用（`migrations/` 目录即真相，且 AutoMigrate 早于 AutoSeed，
  所以删除的退役角色不会被旧种子复活）。它做三件事：
  1. `users.role='security'` 改为 `'security_admin'`（语义等价，权限面不缩小）；
  2. 已部署流程定义 XML 的角色指派位按映射改写：`l1_support`→`agent`，
     `l2_support`/`dba`→`technician`，`l3_expert`/`it_director`/`ops_director`→`it_admin`，
     `sd_manager`/`ops_manager`/`rd_manager`→`manager`；
  3. `roles` 表里的退役角色在**没有任何成员**时连同其 `role_permissions` 一并删除；
     仍有成员的角色会被保留并 `RAISE WARNING` 点名，等你先迁走成员再重跑，避免把人悬空。
- 迁移 `itsm-backend/migrations/20261009_drop_orphan_ai_analysis_result.sql` 同批自动应用，
  下线单数孤儿表 `ai_analysis_result`（Ent 实际写复数 `ai_analysis_results`）。表内有数据即
  `RAISE EXCEPTION` 中止，不静默删。
- **清空演示业务数据的运维陷阱**：若你也用 `TRUNCATE ... RESTART IDENTITY` 清掉工单/事件/变更等
  业务行，主键会从 1 重新分配，而 `operational_commands` 的幂等键形如 `ticket:1:rules:create`。
  只要旧聚合的命令历史还留着，新建第一条工单就会撞唯一约束、返回 `code 5001 创建工单失败`。
  清理时必须一并删除聚合已不存在的命令行（按 `aggregate_type` + `aggregate_id` 回查父表，
  保留聚合仍存活的行），而不是关掉幂等约束。
- **不可逆**：本迁移删除角色行与授权行，不提供 `_down.sql`。执行前必须 `pg_dump`；
  脚本末尾带「定义仍含已映射退役码即 `RAISE EXCEPTION`」的收尾不变量，且可重复执行。
- 若你的自建角色、外部集成或自动化脚本按退役角色码过滤用户/审批人，需在升级前改为新码。
  连接器回调、Webhook payload 中的历史角色字符串不受影响（只是文本）。
- 历史条目 §1.5、§1.6 里出现的 `it_director` / `ops_director` / `dept_manager` / `team_lead` /
  `sd_manager` / `ops_manager` / `guest` 是当时的权限授予清单，仅作记录，现已不是合法角色。

**验证**

```bash
cd itsm-backend
# 词表跨层契约（集合相等 + 退役码零出现，覆盖枚举/DTO/authz/种子/BPMN/前端）
go test ./domain/role/ -run TestContract -v
# 迁移在隔离 PostgreSQL 上的演练：改值、映射改写、无成员才删、幂等重跑
go test ./pkg/seeder/ ./internal/authz/ ./tests/parity/ -count=1
```

### 1.34 SLA 定义响应改为投影后的 camelCase 配置对象（2026-10-15，破坏性：字段删除 + 类型变更）

`SLADefinitionResponse` 过去把数据库里的两个 JSON 列原样透出：`businessHours` 与 `escalationRules`
都是 snake_case 键的自由 map（`time_zone`、`work_days`、`is_24_7`、`levels[].threshold_minutes`、
`notify_roles`）。前端因此只能在页面里各自猜键名，`/admin/sla-definitions` 的服务窗口列被硬编码成
「7×24」，升级规则列直接 `JSON.stringify` 整个 map。现在投影收敛到 DTO 层：

| 旧响应 | 新响应 |
| --- | --- |
| `businessHours`：`{"time_zone":"Asia/Shanghai","is_24_7":true,...}` 原始 map | `businessHours`：`{timeZone,alwaysOn,workDays[],startTime,endTime}`，配置为空时该字段缺省 |
| `escalationRules`：`{"levels":[{"level":1,"threshold_minutes":15,"notify_roles":["manager"]}]}` | `escalationLevels`：`[{level,thresholdMinutes,notifyRoles[]}]`，**`escalationRules` 字段已删除** |
| `conditions` 原始 map | 不变（仍是原始 map，管理页不编辑） |

**升级影响**

- 任何按 `escalationRules.levels[].threshold_minutes` 读取的集成方需改为
  `escalationLevels[].thresholdMinutes`；`businessHours.is_24_7` 改为 `businessHours.alwaysOn`。
- `escalationLevels` 只收录类型正确的层级，条数可能少于原配置；`businessHours` 缺省不等于
  `alwaysOn: false`（前者是「没配」，后者是「配了但不是 7×24」），展示层需区分。
- 纯 API 边界变更：**无数据库迁移、无数据改写**，`sla_definitions` 的列与存量 JSON 保持原样；
  回滚只需退回旧镜像。
- 创建接口 `POST /api/v1/sla/definitions` 的 `isActive` 是非指针 `bool`，不传即按 `false` 落成停用
  状态。这不是本次新增的破坏，但页面/脚本创建 SLA 时应显式提交该字段。
- `GET /api/v1/sla/definitions` 只解析 `page`/`pageSize`（上限 100），不支持按名称或启用状态过滤；
  需要全量请按 `total` 翻页，不要假定一次请求返回全部。

**验证**

```bash
cd itsm-backend && go test ./dto/ ./handlers/sla/ -count=1
cd itsm-frontend && npx jest --coverage=false --testPathPattern "sla-definitions|sla-api|sla-format"
```

### 1.35 SLA 模板响应改为投影配置对象，并新增安装状态字段（2026-10-16，破坏性：类型变更 + 字段新增 + 状态码变更）

`/api/v1/sla/templates`（含 `/{key}` 与 `/{key}/install`）过去把目录里的 JSON 配置原样透出，安装状态则完全由前端
本地点击维护。现在模板与已安装 SLA 定义共用 §1.34 的投影函数，并由服务端返回安装事实。

| 旧响应 | 新响应 |
| --- | --- |
| `businessHours`：`{"time_zone":...,"is_24_7":true}` 原始 map | `businessHours`：`{timeZone,alwaysOn,workDays[],startTime,endTime}`，配置为空时字段缺省 |
| `escalationRules`：`{"levels":[{"threshold_minutes":15,"notify_roles":[...]}]}` | `escalationLevels`：`[{level,thresholdMinutes,notifyRoles[]}]`，**`escalationRules` 字段已删除** |
| 无安装状态 | 新增 `installed`（必填布尔）、`slaDefinitionId`、`installedAt`（已安装时才有值） |
| 未知 `key`、缺租户上下文一律 HTTP 400 | 未知 `key` → HTTP 404 / code 4004；缺租户上下文 → HTTP 401 / code 2001（fail-closed，不再返回「谁都没装过」的列表） |

**升级影响**

- 前端与集成方需按新字段读取：`escalationRules.levels[].threshold_minutes` → `escalationLevels[].thresholdMinutes`，
  `businessHours.is_24_7` → `businessHours.alwaysOn`；安装状态请读 `installed`/`slaDefinitionId`，不要再自行记忆点击结果。
- 列表响应仍是 `{items,total}`（目录固定、无分页参数），安装响应 `{templateKey,slaDefinitionId,created,wasAlreadyExist,message}`
  未变更；`created: false, wasAlreadyExist: true` 表示幂等命中，不是失败。
- **无数据库迁移、无数据改写**。安装识别改用 `sla_definitions.conditions.templateKey` 标记，并兼容标记引入前
  写入的描述约定 `[Template] <key> - …`：存量安装仍能被判为已安装，不会重复创建；改名不影响识别（此前按名称
  前缀判断的版本，改名会让安装记录消失）。若管理员此前改过描述的 `[Template] ` 开头，该条会被视为未安装
  （再点安装会新建一份），需人工确认后删除旧记录。
- 已知边界：安装幂等互斥只在单副本进程内。多副本部署下并发安装同一模板理论上仍可产生两条定义，彻底去重需要
  `sla_definitions` 上的 `(tenant_id, template_key)` 唯一约束（尚未引入）。
- 模板的 `escalationLevels` 只是随定义写入的参考配置，当前没有运行时消费者；升级动作在「升级规则」
  `/admin/escalation-rules` 与「升级矩阵」`/admin/escalation-matrices` 配置。
- 纯 API 边界变更，回滚只需退回旧镜像（旧镜像会重新透出原始 map，新写入的 `templateKey` 标记对旧代码无害）。

**验证**

```bash
cd itsm-backend && go test ./service/ ./handlers/sla_template/ -count=1
cd itsm-frontend && npx jest --coverage=false --testPathPattern "sla-templates|sla-template-api|sla-format"
```

### 1.36 团队列表改为标准列表信封，成员关系升为一等接口（2026-10-16，破坏性：响应结构变更 + 状态码变更）

`GET /api/v1/org/teams`（以及别名 `GET /api/v1/teams`）此前直接把数组塞进 `data`，成员关系只在库里、
接口层完全不可见，前端团队管理页的成员头像与「团队成员总数」因此恒为空。现在列表走统一信封，
成员读写成为独立端点。

| 旧 | 新 |
| --- | --- |
| `data: [{id,name,code,...}]` 裸数组，无分页、无总数 | `data: {items,total,page,pageSize,totalPages}` |
| 团队对象没有成员（部分集成方自行猜 `edges.users`） | `members: [{id,username,name,email}]`，无成员时为 `[]` |
| 无搜索参数，前端拿全量数组本地过滤 | `GET /api/v1/org/teams?search=` 在服务端匹配 `name`/`code`/`description` |
| 成员只能通过未注册的 `POST /api/v1/org/teams/members`（恒 404） | `GET/POST/DELETE /api/v1/org/teams/{id}/members`，写操作 `data` 回带服务端成员列表 |
| `PUT /api/v1/org/teams/{id}` 用非指针字段，传 `""` 清空描述/传 `0` 解除负责人都是空操作 | `description`/`managerId` 为可选指针：未传保持原值，显式 `""` 清空描述、`0` 解除负责人 |
| 未知团队 DELETE 返回成功 | HTTP 404 / code 4004 |
| 缺少租户上下文时按 0 号租户查询（空列表） | HTTP 401 / code 2001（fail-closed） |

**升级影响**

- 读取团队列表的集成方需解一层 `data.items`；分页由服务端负责，`total` 是过滤后的全量条数，
  `pageSize` 超过 100 会被回落到 100（不再是「有多少返回多少」）。
- 成员是 **1:N**（`users.team_users` 外键）：一个用户同一时刻只属于一个团队。把已在别队的用户
  加入新团队返回 HTTP 409 / code 4090，消息里带上其归属团队名；重复添加同一成员返回 200 且不会
  产生第二份关系；移除非成员返回 404。跨租户或已停用的用户作为成员/负责人写入返回 400 / code 4000
  （领域层 `AppError` 路径，不是参数层的 1001）。
- `team:write` 此前在任何角色条目里都不存在，DBOnly 权限模式（企业交付默认）下租户管理员访问
  团队管理恒为 403。现已补进 `admin` 角色；自定义角色若也需要管理团队，需自行授予 `team:read`/`team:write`。
- **无数据库迁移**：成员关系使用既有的 `users.team_users` 列，未新增表或约束。想要「一个用户属于多个团队」
  需要引入 M2M 关联表与迁移，本次未做。
- 前端 `CommonApi.getTeams()`（只返回别名列表、无成员）已删除，统一改用 `teamService`；
  `service/team_service.go` 这条与 `handlers/common` 重复的实现一并移除，后端团队用例只有一个所有者。

**验证**

```bash
cd itsm-backend && go test ./handlers/common/ ./router/ ./internal/authz/ -count=1
cd itsm-frontend && npx jest --coverage=false --testPathPattern "team-service|admin/teams|api-contract"
```

### 1.37 菜单接口的清空语义、上级校验与删除状态码（2026-10-16，破坏性：状态码变更 + 请求语义变更）

`GET /api/v1/menus` 的响应结构没有变化（一直是标准列表信封），但管理页 `/admin/menus` 读的是不存在的
`data.menus`，整表渲染为空、四张统计卡恒为 0。修复前端的同时，把同一组端点上三个会造成数据损坏的
写语义补齐了。

| 旧 | 新 |
| --- | --- |
| `PUT /api/v1/menus/{id}` 只按指针判断，页面无从表达「移除上级菜单」，提交后上级不变（静默失效） | `parentId: 0` 清为**顶级菜单**（写 NULL）；未传保持原上级 |
| `icon`/`permissionCode`/`description` 无法清除 | 显式 `""` 清除；未传保持原值 |
| `parentId` 可指向其他租户的菜单、自身或自己的后代，落库成功后整棵子树从页面上消失 | 写入前校验同租户、非自身、非后代，违反返回 HTTP 400 / code 4000 且不留脏数据 |
| `DELETE /api/v1/menus/{id}` 仍有子菜单时 HTTP 500 / code 5001「删除菜单失败」 | HTTP 409 / code 4090「请先删除子菜单」，父子行都不受影响 |
| `DELETE /api/v1/menus/{id}` 对不存在或跨租户的 id 返回 HTTP 200 / code 0（什么也没删却报成功） | HTTP 404 / code 4004「资源不存在或已被删除」，与读/改同口径 |

**升级影响**

- 依赖「删除任意 id 都成功」的脚本需要按 404 处理已不存在的菜单；需要先清理子树的脚本现在会收到 409，
  应改为自底向上删除或先提示用户。
- 「清空」与「不传」从此是两件事：集成方要清空字段必须显式发送空值（`parentId: 0`、`icon: ""`），
  省略键表示保持不变。前端管理页已按全量提交实现该约定。
- **清除 `permissionCode` 是放权操作**：该字段为空的菜单在侧边栏过滤里视为「无权限要求」，对租户内所有
  登录用户可见。前端保存表单会显式提交空串，因此操作者清空权限码时必须自行确认影响面。
- `GET /api/v1/menus` 一次返回当前租户的**完整**菜单集合，`page` 恒为 1、`pageSize` 恒等于 `total`，
  不接受分页参数；管理页的搜索与树形组装是在这份完整集合上做的本地视图，不是本地分页。
- **无数据库迁移**：上级关系沿用 `menus.parent_id`（可空）与既有 `(tenant_id, path)` 组合唯一索引。

**验证**

```bash
cd itsm-backend && go test ./service/ ./handlers/rbac/ -count=1
cd itsm-frontend && npx jest --coverage=false --testPathPattern "admin/menus|menu-api|api-contract"
```

### 1.38 工单分类「关联工作流」开始真正参与建单选路（2026-10-16，破坏性：写入口状态码变更 + 路由行为变更）

`ticket_categories.workflow_definition_key` 此前只写库、不生效：建单与流程启动代码从未读取它。
管理员在 `/admin/ticket-categories` 选了工作流会收到「保存成功」，新工单却仍走默认流程。
现在该绑定成为建单选路的一级，并对写入增加了可解析性校验。

**建单流程 Key 解析优先级**（`POST /api/v1/tickets`，请求未显式传 `workflowDefinitionKey` 时）

| 顺序 | 来源 | 说明 |
| --- | --- | --- |
| 1 | 请求体 `workflowDefinitionKey` | 最高优先，不再被工单类型静默覆盖 |
| 2 | 工单类型 `ticket_types.workflow_definition_key` | 工单类型已配置时以类型为准，分类配置让位 |
| 3 | 工单分类 `ticket_categories.workflow_definition_key` | 本次新增；分类由 `categoryId` 或分类名解析而来 |
| 4 | 流程绑定表 `process_bindings`（按工单类型匹配） | 兜底命中 `ticket_general_flow` 且工单为 `high`/`urgent` 时改走 `ticket_urgent_flow` |
| 5 | 命令侧最终兜底 `ticket_general_flow` | 解析器不可用或返回空时 |

| 旧 | 新 |
| --- | --- |
| `POST`/`PUT /api/v1/ticket-categories` 接受任意 `workflowDefinitionKey`，不校验是否存在 | 必须是**本租户存在且已激活**的流程定义，否则 HTTP 400 / code 4000「工作流不存在、未激活或不属于当前租户」，行不受影响 |
| 传其他租户的流程 Key 会落库并在启动流程时报「流程定义 X 不存在」，命令反复重试 | 写入即拒绝；已落库的历史脏绑定在消费端回退路由表与内置兜底并告警，不再卡住工单生命周期 |
| 校验失败响应统一为「操作失败」，管理员无从判断原因 | 分类写端点改走 `common.RespondError`，返回分类后的安全文案 |

**升级影响**

- **存量分类绑定会立即改变新工单的流程走向**。升级后请核对：
  `SELECT id, code, workflow_definition_key FROM ticket_categories WHERE workflow_definition_key <> '';`
  若某条绑定本意是「占位」或指向已停用流程，请在管理页清空（提交 `workflowDefinitionKey: ""`）或改绑，
  否则该分类下、且工单类型未配置工作流的新工单会走这条流程。
- 写入校验只作用于新的 `POST`/`PUT`；历史数据不会自动清理，由消费端兜底保证不卡死。
- 分类改绑**不会**重启或迁移已在运行的流程实例，只影响后续新工单。
- **无数据库迁移**：沿用 `ticket_categories.workflow_definition_key`（可空，max 128）。

**验证**

```bash
cd itsm-backend && go test ./service/ ./handlers/ticket_category/ -count=1
```

### 1.39 内置角色权限码变更，需要重跑 bootstrap job（2026-10-16，破坏性：路由下线 + 角色授权增减）

本批三处权限面调整都只改「内置角色 → 权限码」权威源（`internal/authz/roles.go`），
**不会随服务重启自动落到已有安装**，必须重放初始化：

```bash
ITSM_BOOTSTRAP_ONLY=true ITSM_AUTO_SEED=true ./itsm-backend
```

| 变更 | 影响 |
| --- | --- |
| `manager` 新增 `ticket_category:read` | 部门经理此前持 `ticket:write`（派生 `create`/`update`）与 `ticket:assign`，能建单却读不到 `GET /api/v1/ticket-categories`，建单页分类下拉为空。重放后该角色可见并可选分类 |
| `GET /api/v1/system-configs/init`（别名 `GET /api/v1/configs/init`）下线 | 该路由是按 `system_config:read` 守卫的**写**入口，挂在 GET 上；默认配置本就由 `ListConfigs` 懒初始化。外部脚本若曾调用它需改为直接读取配置 |
| `end_user` 撤回 `system_config:read` | 读接口整表返回 SMTP、密码策略、2FA 等租户级配置，而最终用户无任何前端入口。撤回由 `RetiredRolePermissionCodes()` 承载并计入 `identity-rbac` 校验和，因此重放 job 才会真正回收既有授权行 |

授权播种是「只增不减」语义：新增码在重放后立刻生效，撤回码只在退役清单登记后才回收。
未重放前的表现与升级前一致，不会间歇性 403。

**验证**

```bash
cd itsm-backend && go test ./internal/authz/ ./tests/parity/ ./pkg/seeder/ -count=1
```

### 1.40 审批待办菜单改挂 `task:read`（2026-10-16，破坏性：菜单可见性变更，需重放 §1.39 的 bootstrap job）

「审批中心」/「待我审批」两条菜单此前挂 `approval:read`，而页面实际只消费
`GET /api/v1/bpmn/tasks`（`task:read`）与工单/变更/服务请求的 pending 列表，
隔离栈实测该页**没有任何 `approval:*` 调用**。`approval:read` 只授给
change_manager / it_admin / service_catalog_admin / 三员等配置角色，
`agent`、`manager`、`end_user`、`technician` 这些真正的待办处理人不持有它；
叠加 `/approvals` 挂在 `/workflow` 下（`buildMenuTree` 把 `/workflow` 前缀归入 admin 桶，
前端只对管理角色渲染该桶），结果是**处理人在整个侧边栏里没有任何审批入口**，
只能手敲 URL，而该页面对他们返回 200 且正常渲染。

| 菜单 | 旧权限码 | 新权限码 |
| --- | --- | --- |
| `/approvals/pending`「待我审批」（顶级，main 桶） | `approval:read` | `task:read` |
| `/approvals`「审批中心」（`/workflow` 子项，admin 桶） | `approval:read` | `task:read` |

**不会随服务重启自动生效**：`permission_code` 落在 `menus` 表，需要重放初始化
（`pkg/seeder` `upsertMenu` 与 `service/menu_service.go` `upsertBaselineMenu` 都会 UPDATE 已有行）：

```bash
ITSM_BOOTSTRAP_ONLY=true ITSM_AUTO_SEED=true ./itsm-backend
```

**升级影响**

- 重放后 `agent` / `manager` / `end_user` / `technician` 的主导航新增「待我审批」一项；
  该菜单是纯展示层，接口仍逐请求按 `task:read`/`task:update` 与归属校验把关。
- 管理角色无变化：`/approvals` 原码 `approval:read` 与新码 `task:read` 这批角色都持有。
- 若某租户在菜单管理端手工改过这两条的 `permission_code`，重放会覆盖回基线值，需重新配置。
- **未动**的部分：审批记录与流程定义配置面（`GET /api/v1/approvals` → `ApprovalHandler.ListWorkflows`）
  仍要求 `approval:read`；本次只改菜单归属，不给任何角色补授权限。
- **无数据库迁移**：只更新 `menus.permission_code` 两行。

**验证**

```bash
cd itsm-backend && go test ./pkg/menubaseline/ ./pkg/seeder/ -count=1
```

### 1.41 自动化/手工分配规则改用自有权限码，分配推荐并入 `ticket:assign`（2026-10-17，破坏性：权限码变更）

授权平面第二轮收敛的修正：10-17 的功能点台账批次发现 09-17 批次（§1.7）把
`assignment_rule` / `automation_rule` 收敛进 `system_config:*` 过度了——这两组是
分配/自动化的规则配置面而非系统配置面，把只读规则浏览授给 `system_config:read` 会
连带放开其它系统配置读写排查通道。改为自有码：

| 接口组 | 旧权限码 | 新权限码 |
| --- | --- | --- |
| `/tickets/automation-rules`（6 个端点：CRUD + test） | `system_config:read` / `system_config:write` | `automation_rule:read` / `automation_rule:write` |
| `/tickets/assignment-rules`（含 test 共 7 个端点） | `system_config:read` / `system_config:write` | `assignment_rule:read` / `assignment_rule:write` |
| `POST /tickets/{id}/auto-assign`、`GET /tickets/assign-recommendations/{id}` | `system_config:write` / `system_config:read` | `ticket:assign`（与单工单手工分配同码） |

**升级影响**：自定义角色此前按 `system_config:*` 授权的规则浏览/编辑将 403，
需在角色管理里改为对应自有码；`super_admin` 通过快捷分支不受影响。
无 schema/迁移变更（权限码全部命中 `seeder.AllDefinedPermissionCodes()`，由
`permission_code_catalog_guard` 守卫）。



### 1.42 同域错误形态不一致补齐：not-found/越权/非法迁移不再伪装成 500/400（2026-10-18，破坏性：状态码变更）

本节只改**响应状态码与业务码**，不改任何字段、路由或分页契约。§1.20 建立了单一
分类器，但仍有多个同域路径残留裸 `fmt.Errorf` / 统一 `Fail`，表现为同一个
API 里「相邻函数行为不一致」。本批以真实入口 HTTP 回归钉住后逐一纠正，
调用方若一直按 400/500 重试逻辑编码（例如把 500 当作可重试的暂时性故障），
需按新语义识别 404/403/409。

| 接口 | 触发条件 | 旧行为 | 新行为 |
| --- | --- | --- | --- |
| `POST /service-requests/{id}/provision` | 工单不存在 / 跨租户 | 400/`4000` | 404/`4004` |
| `POST /service-requests/{id}/provision` | 状态非 `security_approved` | 500/`5001`（裸 error 兜底） | 409/`4090` |
| `POST /provisioning-tasks/{id}/execute` | 任务不存在 / 跨租户 | 500/`5001` | 404/`4004` |
| `POST /tickets/{id}/comments`、`GET /tickets/{id}/comments` | 工单不存在 / 跨租户 | 500/`5001` | 404/`4004` |
| `POST /tickets/{id}/comments` | 内容为空 / 提及用户无效 / 附件不属于该单 | 500/`5001` | 400/`1001`–`1002` |
| `PUT /tickets/{id}/comments/{cid}`、`DELETE …/{cid}` | 非作者/处理人 | 500/`5001` | 403/`2003` |
| `DELETE /tickets/{id}/attachments/{aid}` | 非上传人/处理人/申请人 | 500/`5001` | 403/`2003` |
| `DELETE /tickets/{id}/attachments/{aid}` | 附件不存在 / 跨租户 | 500/`5001` | 404/`4004` |
| `POST /tickets/{id}/notifications` | 工单不存在 / 跨租户（直发与 outbox 双路径） | 500/`5001` | 404/`4004` |

`POST /tickets/{id}/notifications` 同时收紧**收件人校验**：直发路径与 outbox
入箱路径的收件人必须属于本租户（此前收件人存在性校验不带 `tenant_id` 谓词，
工单租户可为其它租户的真身用户创建通知行 / 入箱投递命令——跨租户引用注入）。
跨租户/不存在收件人按「用户不存在」静默跳过（`code:0` + 零副作用末尾），
与幽灵 `userId` 在响应上不可区分。

存在性防泄漏不变：跨租户与「资源不存在」均为同一个 404/`4004` 错误形态。
无 schema/迁移变更，无需执行额外命令。

**验证**

```bash
cd itsm-backend && go test -count=1 \
  ./handlers/provisioning/ ./handlers/ticket_comment/ ./handlers/ticket_notification/ \
  ./handlers/ticket_view/ ./router/ ./service/
```

### 1.43 附件响应不再暴露服务端磁盘路径 `filePath`（2026-10-18，破坏性：字段删除）

B7 测试侦查发现 `TicketAttachmentResponse` 把 `attachment.file_path`（服务端
磁盘/存储路径，如 `uploads/tickets/<id>_<ts>_<name>`）原样放进 API 响应——
部署拓扑与存储布局属于服务端实现细节，进入客户端契约是信息泄漏面，前端
三个类型声明也无任何组件消费该字段（死字段）。

**变更**：`{items,total}` 条目与单条上传响应删除 `filePath`；下载入口唯一
真相仍是推导的 `fileUrl = /api/v1/tickets/{id}/attachments/{attachmentId}`。
无迁移（列保留在 ent schema，磁盘文件生命周期不受影响）。

**升级影响**：下游调用方若读取 `filePath` 需改用 `fileName` + `fileUrl`。
mapper 侧契约守卫：`dto/ticket_attachment_dto_test.go` 断言响应不含
`filePath`；入口侧由 `handlers/ticket_attachment/cross_tenant_http_test.go`
的列表正控制锁定。

### 1.44 资产/许可证写面 not-found 与席位冲突不再伪装成成功/500（2026-10-18，破坏性：状态码变更）

与 §1.42 同型的残留采集：`handlers/asset` 的写面在跨租户/不存在时因忽略
条件操作影响行数而静默假成功，席位冲突则以裸 error 落入 500 兜底。本批
以真实入口 HTTP 回归（`handlers/asset/cross_tenant_http_test.go`，含
license 双面）钉住后统一为 sentinel 错误语义。

| 接口 | 触发条件 | 旧行为 | 新行为 |
| --- | --- | --- | --- |
| `DELETE /assets/{id}` | 资产不存在 / 跨租户 | 200/`0` 静默假成功（影响行数被忽略） | 404/`4004` |
| `PUT /assets/{id}/assign` | 资产不存在 / 跨租户 | 200/`0` + `data:null` 静默假成功 | 404/`4004` |
| `PUT /assets/{id}/retire` | 资产不存在 / 跨租户 | 200/`0` + `data:null` 静默假成功 | 404/`4004` |
| `DELETE /licenses/{id}` | 许可证不存在 / 跨租户 | 200/`0` 静默假成功 | 404/`4004` |
| `PUT /licenses/{id}/assign` | 许可证不存在 / 跨租户 | 200/`0` + `data:null` 静默假成功 | 404/`4004` |
| `PUT /licenses/{id}/assign` | 席位不足 | 500/`5001`（裸 error 兜底） | 409/`4090` |
| `PUT /licenses/{id}/assign` | 并发扣减未命中（条件 UPDATE 保护） | 500/`5001` | 409/`4090` |

`PUT /licenses/{id}/assign` 同时收紧**被分配用户的租户归属**：`userIds`
只保留本租户真实存在的用户 id，跨租户/无效 id 静默跳过（不写
`license.Users`、不扣席位），与 §1.42 通知收件人语义一致——此前对端租户
真身用户 id 可被写进本租许可证的用户名单（跨租户引用注入）并占用席位。

存在性防泄漏不变：跨租户与「资源不存在」同为 404/`4004`。读面
`Get/Update/UpdateStatus`（asset 与 license 两侧）此前已正确返回
404，本批未改动。无 schema/迁移变更。

**验证**

```bash
cd itsm-backend && go test -count=1 \
  ./handlers/asset/ ./service/ ./router/
```

### 1.45 分配规则与变更评审写面错误形态不再伪装成成功/500（2026-10-18，破坏性：状态码变更）

与 §1.42/§1.44 同型收敛：`handlers/assignment_smart`（分配规则写面）与
`handlers/change_review`（评审组名册）在跨租户/不存在/重复时，此前把状态
码伪装成 200 假成功或 500/`5001` 兜底。

| 接口 | 旧行为 | 新行为 | 不变量 |
|:---|:---|:---|:---|
| `DELETE /tickets/assignment-rules/{id}` | 条件删除忽略 0 行 → 永远 200（跨租户探测面） | 404/`4004` | 不存在/跨租户/重放同掩码；对端行零副作用 |
| `POST /tickets/assignment-rules/test`（ruleId 跨租户） | 裸 error → 500/`5001` | 404/`4004` | 与 `GET` 同域口径（`ErrAssignmentRuleNotFound`） |
| `POST /tickets/assignment-rules/test`（ticketId 跨租户） | **无租户谓词**：他租工单被读取做匹配并回显 `assignedTo` | 404/`4004` | 他租工单零读取（`ErrTestTicketNotFound`） |
| `POST /change-review/members`（userId 跨租户/不存在） | 裸 error → 500/`5001` | 400/`1001` | 引用型资源缺失，与工单评论「提及用户无效」同口径 |
| `POST /change-review/members`（同 user+type 重复） | 裸 error → 500/`5001` | 409/`4090` | 业务冲突语义 |
| `PUT/DELETE /change-review/members/{他租id}` | 裸 error → 500/`5001` | 404/`4004` | 跨租户与不存在同掩码 |
| `GET /change-review/members`（空名册） | `data: null` | `data: []` | 列表契约恒为数组 |

防御纵深：`GET /change-review/members` 读用户详情补租户谓词，历史脏数据
（成员行指向他租用户）在列表中静默跳过。规则面权限码仍为 §1.41 的
`assignment_rule:{read,write}`；动作面（auto-assign/assign-recommendations/
test）归 `ticket:assign`。升级矩阵 HTTP 面（`/escalation-matrices`，sla:read/write）
为纯回归锁定，无语义变更。无 schema/迁移变更。

**验证**

```bash
cd itsm-backend && go test -count=1 \
  ./handlers/assignment_smart/ ./handlers/change_review/ \
  ./handlers/escalation_matrix/ ./service/ ./router/
```


### 1.46 BPMN 列表过滤/分页首次真正生效，实例生命周期与定义删除改用冲突语义（2026-10-18，破坏性：响应结果与状态码变更）

`ListProcessInstancesRequest` 等 BPMN 请求结构只有 `json` 标签，而 handler 用的是
`ShouldBindQuery`——Gin 按 Go 字段名匹配查询参数，camelCase 参数因此静默丢空，
service 里写好的过滤谓词和 `Offset/Limit` 是死代码。补齐 `form` 标签后，**同一批
请求会第一次真的返回被过滤/翻页的结果**；调用方若此前依赖"过滤参数不生效"来拿全量，
需显式不带参数请求。身份与关系键保持不绑定（`form:"-"`）：`tenantId`、`userId`、
`assignee`、`candidateGroups` 只认认证上下文，`processInstanceId`（数值外键）与
`startDate`/`endDate`（wire 格式未收敛）继续忽略，避免把租户变成可覆盖参数、
或把时间过滤变成 400。

| 接口 | 旧行为 | 新行为 | 不变量 |
|:---|:---|:---|:---|
| `GET /bpmn/process-instances`（`processDefinitionKey`/`status`/`businessKey`/`page`/`pageSize`） | 参数恒被丢弃：返回全量首页，翻页重复 | 过滤与分页真正生效 | `tenantId` 仍来自认证上下文 |
| `GET /bpmn/process-definitions`（`key`/`category`/`isActive`） | 同上 | 同上 | 同上 |
| `GET /bpmn/tasks`（`status`/`processDefinitionKey`/`page`/`pageSize`） | 同上 | 同上 | `assignee`/`candidateGroups` 不可由请求注入 |
| `PUT /bpmn/process-instances/{id}/suspend\|resume\|terminate` | 不校验源状态：对 `completed`/`terminated` 实例照样写库、写审计、返回 200 | 非法迁移 **409/`4090`** 且实例状态原样留存；实例不存在/跨租户 **404/`4004`** | 合法迁移收敛为单一状态机表：`suspend←running`、`resume←suspended`、`terminate←running\|suspended` |
| `DELETE /bpmn/process-definitions/{key}`（该版本有实例） | 裸 `fmt.Errorf` → 500/`5001`，看不出是业务拒绝 | 有活跃实例（running/suspended）**409/`4090`**；仅有历史实例 **409/`4090`** 并给出可执行指引 | 无实例版本正常删除；跨租户按 not-found。`process_instances.process_definition_id` 是必填外键，**有历史实例的版本仍删不掉**（归档/软删属 schema 与产品决策） |
| `POST /tickets/analytics/export`（前端调用形状） | 前端把 `dateFrom/dateTo/format` 全塞 query、请求体为空；后端 `ShouldBindJSON` 读 `timeRange`（必填、恰好两个 `YYYY-MM-DD`）→ 必然 400，页面再统一压成"导出失败，请重试" | 请求体带 `timeRange/dimensions/metrics/chartType/filters`，`format` 留在查询参数；页面透出服务端真实文案 | 后端契约未变，属前端调用方修复 |
| 内置流程定义条件表达式 | 读 `variables['approval_result']`，而引擎三条写入路径都写驼峰 → 条件恒 false，审批驳回不走驳回分支且无任何报错 | 模板文件改为 `approvalResult`；存量 `process_definitions.bpmn_xml` 由 `migrations/20261018_fix_bpmn_approval_result_variable.sql` 改写 | 该迁移属磁盘自发现流，版本日期晚于收养 cutoff（`20260908`），下次后端启动即自动执行；幂等（重复命中 0 行），残留则 `RAISE EXCEPTION` fail closed |

本批无新增表结构变更。因该迁移会改写流程定义 XML 文本，升级前建议对目标库做一次备份。

**验证**

```bash
cd itsm-backend && go test -count=1 \
  ./handlers/bpmn/ ./service/ ./migration/ ./internal/bootstrap/
cd itsm-frontend && npx jest \
  src/lib/services/__tests__/analytics-service.test.ts \
  src/lib/api/__tests__/ticket-analytics-api.test.ts \
  src/lib/__tests__/api-contract.test.ts
```

### 1.47 基础管理域写面错误形态与幂等收敛（projects / ticket-tags / ticket-rating / surveys / timers）（2026-10-18，破坏性：状态码变更）

与 §1.42/§1.44/§1.45 同型收敛：项目管理、工单标签、工单评分、满意度调查与
BPMN 计时器各读/写面此前把「不存在/跨租户/业务违规/重复操作」伪装成 200 假
成功或 500/`5001` 兜底。

| 接口 | 旧行为 | 新行为 | 不变量 |
|:---|:---|:---|:---|
| `POST /projects`（managerId/departmentId 他租） | **无引用校验**：他租引用静默落库 | 400/`1001` | 引用型资源缺失与工单评论「提及用户无效」同口径 |
| `POST /projects`（未传 managerId/departmentId） | 显式 `Set 0` → FK 严格环境 500/`5001` | 200（字段留空） | Optional 字段 0 = 未指定 |
| `DELETE /projects/{id}`（重放/跨租户） | 200 假成功或裸「项目不存在: id=..」→ 500 | 404/`4004` | 条件删除验 affected rows；id 不再进错误文本 |
| `GET/PUT /projects` 响应 | 直返 ent 模型（snake_case 泄漏） | DTO（camelCase） | Handler 不直返 ent 模型 |
| `DELETE /ticket-tags/{id}`（重放/跨租户） | 0 行静默 nil → 200 假成功 | 404/`4004` | 条件删除验 affected rows（sentinel `ErrTicketTagNotFound`） |
| `POST /tickets/{id}/tags`（重复提交同 tagIds） | M2M UNIQUE 冲突 → 500/`5001` | 200 + 幂等 | 绑定幂等：已绑定标签跳过，不重复落库 |
| `POST /tickets/{id}/rating`（待解决工单） | 裸 error → 500/`5001` | 400/`1001` | 业务状态违规不伪装成内部故障 |
| `POST /tickets/{id}/rating`（非申请人） | 裸 error → 500/`5001` | 403/`2003` | 域内授权违规 |
| `POST /tickets/{id}/rating`（已评过/并发二次） | check-then-write 竞态可静默覆盖 + 裸 error → 500 | 409/`4090` | 条件更新（rating IS NULL 或 =0）+ affected rows；重放/竞态不覆盖第一次评分 |
| `POST /surveys/responses`（surveyId/ticketId 他租） | **无引用校验**：他租 survey 引用静默落库 | 400/`1001` | 响应 survey/ticket 边必须本租；`respondentId` 写入认证用户（此前硬编码 0） |
| `GET /surveys/{id}/analytics`、`GET /surveys/{id}/responses`（他租/不存在 survey） | 200 + 空统计/空列表（存在性探测面） | 404/`4004` | 读统计/响应前先验 survey 归属 |
| `GET /surveys` 等响应 | 直返 ent 模型（snake_case 泄漏） | DTO（camelCase） | Handler 不直返 ent 模型 |
| `GET /timers/{timerId}`（不存在） | ent NotFound 上抛 → 500/`5001` | 404/`4004` | 空态与跨租户同形掩码（notFoundError） |
| `GET /system/vector-store` | 无变更 | （纯回归锁定） | 未配置=unconfigured、非法 env=error 降级、settings 脱敏 |

未被本批修而明确不动的：`GET /timers/{timerId}` 对跨租户 timerID 保持读后掩码
（timerID 为 BPMN 生成的全局唯一字符串，与不存在同形 404）；`Tag`（系统预置）
与 `TicketTag`（工单关联）双实体并存、`project.code` 全局唯一不按租户组合——
归属裁决记为待办。无 schema/迁移变更。

**验证**

```bash
cd itsm-backend && go test -count=1 \
  ./handlers/project/ ./handlers/survey/ ./handlers/ticket_rating/ \
  ./handlers/ticket_tag/ ./handlers/timer/ ./handlers/vector_store/ \
  ./handlers/prediction/ ./service/ ./router/
```

### 1.48 自助注册与密码找回的失败原因不再统一压成 401（2026-10-18，破坏性：状态码与业务码变更）

`POST /auth/register` 与 `POST /auth/forgot-password` 此前把所有失败交给
`AuthFailedWithErr`，一律返回 **401/`2001`「认证失败」**：用户名被占用、邮箱被占用、
部署只有一个活跃租户的 fail-closed 拒绝、邮件服务未配置，对端看起来完全一样，
新用户只会反复重试同一个表单。现按错误性质分级：

| 场景 | 旧响应 | 新响应 | 不变量 |
|:---|:---|:---|:---|
| 用户名/邮箱已被注册 | 401/`2001` | **409/`4090`** + `用户名已被注册`/`邮箱已被注册` | 冲突不创建任何用户行 |
| 多活跃租户 / 无活跃租户 | 401/`2001` | **503/`5003`** + 自助注册未开放 / 尚未初始化 | 部署状态不得伪装成用户输入错误 |
| 密码找回：邮件服务未配置 | 401/`2001` | **503/`5003`** + `密码重置功能未启用：邮件服务未配置` | 能力未配置属 operational unavailable |
| 请求体校验失败 | 400/`1001` | 不变 | Gin binding 规则未动（用户名 3–20 位字母数字、密码 ≥8 位） |
| 存储层故障 | 401/`2001` | **500/`5001`** | 未分类故障仍按内部错误处理 |

`POST /auth/reset-password` 与重置 token 校验接口本批**未改**：其 service 返回的是
非类型化 error，直接换成 `RespondError` 会把「token 无效」也映射成 500，需要先给
这两个方法补业务错误分类（已记待办）。

前端可见行为（非接口契约）：注册向导第二步提交现在携带第一步的账号与密码；用户名
规则与后端 binding 一致，非法字符在本地即被拦住；`AuthService` 会解析非 2xx 的统一
响应信封，把服务端 `message` 原样显示（登录限流的 `retryAfterSeconds` 也由此带出）。

**验证**

```bash
cd itsm-backend && go test -count=1 ./handlers/auth/ ./router/ -run 'Auth|Register'
cd itsm-frontend && npx jest wizard-submit auth-service --coverage=false
```

### 1.49 CSRF token 响应字段改为 `csrfToken`（2026-10-19，破坏性：字段改名）

`GET /api/v1/csrf-token` 响应体的 `data.csrf_token` 改为 **`data.csrfToken`**（camelCase，
与 API 边界字段命名规范一致），旧字段已移除，不做双字段兼容。`csrf_token` 这一
snake_case 名称仅保留给 HttpOnly cookie 名与 form 字段名（传输层标识，不属于 JSON 契约）。

受影响场景与迁移方式：

| 调用方 | 旧读取 | 新读取 |
|:---|:---|:---|
| curl / jq 脚本 | `.data.csrf_token` | `.data.csrfToken` |
| 前端 `security.csrfProtection` | `data.data?.csrf_token`（已同步修改） | `data.data?.csrfToken` |
| 自定义集成 | — | 同上改名即可 |

请求侧不变：写请求仍携带 `X-CSRF-Token` 头，每次成功写请求后 token 轮换需重新获取。

CHANGELOG / README / API 参考（`docs/api-reference.md` 新增「CSRF Token」小节）已同步。

**验证**

```bash
cd itsm-backend && go test -count=1 ./middleware/ -run 'TestCSRF'
cd itsm-frontend && npx jest src/lib/__tests__/security.test.ts src/lib/api/__tests__/auth-api.test.ts --coverage=false
```

### 1.50 工单流转路由的权限码从 `workflow:*` 改为 `ticket:*`（2026-10-20，破坏性：授权口径变更）

`/api/v1/tickets/workflow/*` 与 `/api/v1/tickets/{id}/workflow*`、`/api/v1/tickets/{id}/cc`、
`/api/v1/tickets/cc/my` 这 16 条路由此前由 `RequirePermission("workflow", "read"|"write")` 守护。
`workflow:read`/`workflow:write` 在权限目录里的语义是**流程定义管理**（同一码还守着
`POST /bpmn/ai/generate` 与 `/bpmn/ai/templates/*/publish`），而这几条路由的动作对象是某一张工单。
两套语义共用一个码，实测（v1.6.21 prod）结果是双向失效：

- `agent`/`manager` 不持 `workflow:*`，工单详情流转面板与批准/接单/拒绝/解决**全部 403/`2003`**，
  但界面照常渲染按钮；反而 `end_user` 因持有 `workflow:read` 能看到自己工单的进度。
- 要给一线角色开结单能力，就必须连带授予流程模板发布权。

现按工单动作归位：

| 路由 | 旧权限码 | 新权限码 |
|:---|:---|:---|
| `GET /tickets/{id}/workflow/state`、`state-v2`、`workflow-history`、`workflow_records`、`{id}/cc`、`/cc/my` | `workflow:read` | `ticket:read` |
| `POST /tickets/workflow/accept`、`forward`、`cc` | `workflow:write` | `ticket:assign` |
| `POST /tickets/workflow/approve`、`reject` | `workflow:write` | `ticket:approve`（新增权限码） |
| `POST /tickets/workflow/resolve` | `workflow:write` | `ticket:resolve` |
| `POST /tickets/workflow/close`、`reopen` | `workflow:write` | `ticket:close` |
| `POST /tickets/workflow/withdraw` | `workflow:write` | `ticket:update` |

对自部署方的影响：

- **内置角色无需手工干预**：`agent`/`manager`/`technician`/`it_admin` 的基线已补齐对应 `ticket:*` 码，
  角色权限播种只增不减，升级重启后自动收敛。
- **租户自建角色按 `workflow:*` 授权的会断**：例如某租户给自定义"服务台组"只发了 `workflow:write`，
  升级后该组不能再操作工单流转，需改授 `ticket:assign`/`ticket:approve`/`ticket:resolve`/`ticket:close`。
- **两处刻意的能力收回**：`change_manager` 此前借 `workflow:write` 可操作工单流转，现按角色职责不再授予；
  `service_catalog_admin` 此前借 `workflow:read` 可读工单流转进度，现无 `ticket:read` 故不可读。
  需要保留的租户可手工授予对应 `ticket:*` 码。
- `workflow:read`/`workflow:write` 现在只表示流程定义管理，与工单数据面完全解耦。

**验证**

```bash
cd itsm-backend && go test -count=1 ./middleware/ -run 'TestTicketWorkflowRoutes|TestTicketFlowRoles'
cd itsm-frontend && npx jest src/components/ticket/__tests__/TicketDetailActionPermissions.test.tsx --coverage=false
```

### 1.51 工作台 KPI 改为按登录者数据范围统计，并取消捏造环比（2026-10-21，行为变更）

`GET /api/v1/dashboard/overview`、`/kpi-metrics` 的工单类指标此前只按租户统计，与坐席实际能看到的
工单集合不是同一范围；同时环比字段是推导出来的假值——上期窗口没有样本时会被折算成 `+100%`，
`overdue-tickets` 的变化值甚至写死为 `3`。接口字段是新增（向后兼容），但**显示的数字会变**：

- `agent`/`manager`/`technician` 等一线角色升级后，工作台工单数会**下降**到本人创建或受理的集合，
  与 `/tickets` 列表、`/tickets/stats` 完全一致（同一 `datascope.ForTicketReader` 判定）。
  管理员与租户管理员仍为租户全量。需要给一线角色看全租户口径时，调整该角色的数据范围而不是改前端。
- KPI DTO 新增 `comparable`（是否有可比上期基线）与 `scope`（`tenant` / `own`）。
  `comparable:false` 时 `change` 恒为 `0`、`trend`/`changeType` 恒为 `stable`。
- 待处理、处理中、超时是时点存量，库里没有上月末快照，**恒不再显示环比**；
  自建看板如果直接消费这些字段，请按 `comparable` 分支渲染，不要继续用 `change !== undefined` 判断。
- 图表区（趋势、团队负载、事件分布、高峰时段）本期仍为租户级口径，前端已在工作台显式标注。

**验证**

```bash
cd itsm-backend && go test -count=1 ./service/ -run 'TestDashboardService_KPI|TestDashboardRepository_KPI'
cd itsm-backend && go test -count=1 ./handlers/ -run TestDashboard
cd itsm-frontend && npx jest --coverage=false --testPathPattern "dashboard/__tests__/(KPICards|useDashboardData)"
```

### 1.52 工单模板配置面：`it_admin` 补授权，`isActive`/`workflowSteps` 真正生效（2026-10-21，行为变更）

`/tickets/templates` 页此前有两类「点了没反应」：

- **权限**：`it_admin` 只有 `ticket_template:read`，能看不能改——「配置模板」这项职责落在只读权限面上，
  保存必然 403。现补 `ticket_template:create`/`ticket_template:update`（仍不含 `delete`，抹掉存量模板留给
  租户管理员与目录管理员）。内置角色升级后由权限播种自动收敛；**租户自建角色**若也勾了模板读，需要手工补写码。
- **模板状态开关与工作流步骤**：创建/更新请求在 handler 重组时只带 6 个字段，`isActive` 与 `workflowSteps`
  被丢弃，服务层又把创建的 `isActive` 写死 `true`、停用只能靠一个 `json:"-"` 的内部标志。因此表单里的
  「模板状态」提交停用仍得到启用模板，复制一个停用模板会得到启用副本，前端类型声明的 `workflowSteps`
  永远存不进库。现在 `dto.TicketTemplate.isActive` 是指针：不传按启用创建（保持历史默认），显式 `false`
  真实落库，PUT 与 `PATCH /templates/{id}/status` 两条停用路径结果一致；`workflowSteps` 会持久化并在响应回显。
- **列表全量**：`GET /api/v1/tickets/templates` 的服务端实现在无分页信封的情况下写死 `PageSize=100`，
  第 101 个模板既不出现也不体现在 `total` 上。现在按契约真实返回全量（模板是配置数据，租户级量级可控）。

**验证**

```bash
cd itsm-backend && go test -count=1 ./middleware/ -run TestTicketTemplate
cd itsm-backend && go test -count=1 ./handlers/ticket/ -run TestTemplateRoutes
```

### 1.53 工单响应的人名改为服务端投影，删除从未赋值的 `requester`/`assignee` 对象（2026-10-21，破坏性：字段删除 + 字段新增）

`dto.TicketResponse` 声明过 `requester`/`assignee` 两个用户对象，但 15 个响应出口从未给它们赋值，
所以这个字段在真实响应里恒为 `null`（`omitempty` 下直接消失）。前端拿不到名字，只能另开
`GET /api/v1/users?pageSize=100` 全量通讯录去猜，或者把已派工单渲染成「未分配」。

**变更**

- 删除 `TicketResponse.requester` / `TicketResponse.assignee` 与 `dto.UserBasicInfo`（连带删除唯一写它们的
  `ToTicketResponseWithUsers`）。工单接口不再返回任何用户对象，也不返回 `username`/`email`/`phone`/`role`。
- 新增 `TicketResponse.requesterName` / `assigneeName`：由 `handlers/ticket` 的 `GetUserNamesByIDs`
  按**当前租户**批量投影（单条一次、一页一次，ID 去重，只 `SELECT id, name`）。查询失败只降级为不显示姓名，
  不影响工单本身可读。
- 无数据库迁移。姓名来源仍是 `users.name`（`NotEmpty`，不存在空显示名的存量行）。

**升级影响**

- 外部调用方若按 `requester.name` / `assignee.username` 取值，需改读 `requesterName` / `assigneeName`。
- **跨租户与已删除的用户 ID 不会返回姓名**：`assigneeId` 照常返回，`assigneeName` 缺省。前端统一用
  `displayUserName(name, id, emptyLabel)`（`src/lib/formatters.ts`）：有姓名显示姓名，只有 ID 显示 `用户 #<id>`，
  两者都没有才显示「未分配」。不要把「只有 ID」折叠成「未分配」——那是把数据缺失冒充成业务状态。
- 需要用户目录（邮箱、部门、可选处理人列表）请显式走 `GET /api/v1/users`，工单接口不再顺带外泄。
  工单详情页已改为只在持有 `ticket:assign` 且真的打开分配/抄送弹窗时才拉通讯录。

**验证**

```bash
cd itsm-backend && go test -count=1 ./handlers/ticket/ -run TestTicket
cd itsm-frontend && npx jest --coverage=false --runInBand TicketDetailUserNames
```

已知遗留：MSP 客户工单列表接口尚未接入同一投影，前端已如实显示 `用户 #id`（见 CHANGELOG「工单列表/详情
把已派工单显示成未分配」条目的遗留说明）。

### 1.54 问题 SLA 与评论从 503 unready 转真实契约（2026-10-21，行为变更：503→200 + 新表）

`GET /api/v1/problems/:id/sla` 与 `GET/POST /api/v1/problems/:id/comments` 此前恒定返回 `503/5003`
（unready），这是能力契约明写的「唯一阻断 GA 的项」。本次接入真实实现：

**SLA**

- 问题创建即按 `service_type="problem"` + 优先级匹配 SLA 定义，写入 `sla_states`
  （`aggregate_type="problem"`，与工单/事件同一引擎同一多态表）；问题进入 `resolved`/`closed`
  时打解决时间戳；SLA monitor 周期扫描与违规创建覆盖 problem。
- **`bound=false` 契约**：租户无 problem 类型定义或优先级无匹配时问题不绑定 SLA，
  响应 `{bound:false, status:"unbound"}`。与工单不同——工单无匹配时回落隐藏默认 60/480 分钟，
  问题域有意不套用，避免升级后存量问题瞬间全部违规。调用方必须把 `bound=false` 与
  `status="active"` 区分开。
- `Store.RecordResolution` 语义修正：`resolved_at` 已存在的记录（迟到解决）不再被改写为 `met`，
  违规事实不因迟到解决被抹除。

**评论**

- 新表 `problem_comments`（迁移 `20261021_create_problem_comments.sql`），`GET` 分页时间正序、
  `POST` 建评（`isInternal`/`mentions`/`attachments`）。无 update/delete（与 incident 评论同一取舍）。
- 跨租户 `problem_id` 统一 `404`/`4004`，不暴露 ID 可枚举信号。
- `userName` 服务端投影（与工单同一口径）。

**升级影响**

- 升级前集成方若按 503 契约做了显式降级，现在会收到 `200`——请改按 `bound`/`status` 字段判断。
- 需要数据库迁移：`20261021_create_problem_comments.sql`（Ent 自动迁移同样会建表）。
- 想让问题绑定 SLA 的租户，需在 SLA 定义中配置 `service_type="problem"` 的定义并按优先级匹配。

**验证**

```bash
cd itsm-backend && go test -count=1 ./handlers/problem/ ./service/sla/
```

### 1.56 问题→知识沉淀闭环（2026-10-22，新端点 + 迁移）

**新增端点**

- `POST /api/v1/problems/:id/knowledge`：把问题沉淀为知识草稿。请求体全字段可选（`{title?, category?, tags?}`），空值用问题单内容兜底；产物为 `draft`，发布须过知识评审流。权限 `knowledge:write`（动作语义是「创建知识」）。
- `GET /api/v1/problems/:id/knowledge`：反查该问题沉淀的知识文章（含草稿，创建时间倒序）。权限 `knowledge:read`。
- 质量门：问题既无根因也无解决方案时沉淀 `422`/`4220`（无复用价值的空壳文章会污染 RAG 语料）。
- 跨租户 `problem_id` 在两个方向统一 `404`/`4004`；知识仓储未装配时沉淀 `503`/`5003`。
- `GET /problem-investigation/problems/:id/summary` 的 `KnowledgeArticles` 字段自此有真实数据（此前恒为空数组）。

**迁移**：`20261022_knowledge_source_problem.sql` —— `knowledge_articles` 增补可空列 `source_problem_id` + 索引 `(tenant_id, source_problem_id)`，幂等可重复执行。裸 ID 引用而非外键：问题删除不级联删知识（沉淀产物是独立资产）。

**消费方注意**：知识文章响应新增 `sourceProblemId` 字段（`omitempty`）；沉淀不是发布——需要对外可见仍要走既有评审端点（`/knowledge/articles/:id/review/submit` → `approve` → `publish`）。

**验证**

```bash
cd itsm-backend && go test -count=1 ./router/ -run TestProblemKnowledgeRouteContract
```

### 1.55 预览域收口批次：服务请求 SLA、变更提交门禁、企微/钉钉连接器转正（2026-10-21，行为变更）

**服务请求 SLA（新端点 + 新写入路径）**

- 服务请求创建即按 `service_type="service_request"` 写入 `sla_states`（与工单/事件/问题同一引擎）。服务请求无 priority 字段，SLA 定义按 service_type 不带优先级匹配；**无匹配定义不绑定**（不套用隐藏默认值），`GET /api/v1/service-requests/:id/sla` 的 `bound=false` 如实回答。
- not-found 契约与该域一致：HTTP 200 + 业务码 404、无 data（不暴露存在性）。
- 想让服务请求计时：在 SLA 定义中配置 `service_type="service_request"` 的定义即可，monitor 自动纳入扫描。

**变更提交门禁（行为变更：此前可提交的现在会被拦）**

- 影响摘要成为提交审批的后端门禁：受影响 CI 命中关键 CI、或关联未关闭事件时，**风险等级必须显式设为 `high`** 才能提交（422）。此前影响摘要只在前端提示，low/medium 风险也能直接提交。
- 进入 `scheduled` 状态时复查排期窗口冲突：审批后修改排期时间到与活跃变更同 CI 时间重叠的窗口会被 422 拦截。
- 两道门禁查询失败均降级放行（best-effort）但留 Warn 日志；被拦截的提交会给出具体补齐指引。

**企微/钉钉连接器转正（Disabled→Pilot）**

- capability 矩阵不再把两渠道标为「未包含在商业发布」；部署就绪改由环境变量申报：`ITSM_WECOM_READY=true` / `ITSM_DINGTALK_READY=true`（与 `ITSM_FEISHU_READY` 同款门控）。未设置时矩阵显示「部署依赖未就绪」，代码能力不隐藏。
- 企微配置：`corp_id` + `corp_secret` + `agent_id` + 回调 `callback_token`/`encoding_aes_key`；钉钉配置：`app_key` + `app_secret` + `agent_id`（加签机器人另需 secret）。渠道诊断 `GET /notifications/channels` 可直接查看两渠道装配与健康状态。

**验证**

```bash
cd itsm-backend && go test -count=1 ./handlers/service_request/ ./handlers/change/ ./handlers/capability/ ./router/ -run "SLA|ImpactGate|Capability|Connector"
```

### 1.56 交付任务失败恢复：执行权守卫、重试与人工接管（2026-10-21，行为变更 + 新表字段）

**行为变更：已完成任务不能再执行**

- `POST /api/v1/provisioning-tasks/:id/execute` 对已 `succeeded` 的任务返回 **409/`4090`「该交付任务已完成，不能重复执行」**（此前会重新跑一遍 provider，等于重复创建云资源）。对 `running` 任务返回 409「正在执行中」。
- 集成方若把「重复调用 execute」当作幂等重试手段，请改用 `retry`（见下）。两者语义不同：`execute` 是执行一次，`retry` 是重新排队。

**新端点**

- `POST /api/v1/provisioning-tasks/:id/retry`（`service_request:write`，请求体 `{reason?}`）：把 `failed` 任务重新排队，返回 `status=pending`；受 `maxAttempts`（5）约束；每次尝试独立幂等键。同时把服务请求从 `failed` 拉回 `provisioning`。
- `POST /api/v1/provisioning-tasks/:id/resolve`（`service_request:write`，请求体 `{outcome, note}`）：人工接管。`outcome` 仅 `delivered|failed`，`note` **必填**（判定依据，用于审计）。仅 `failed`/`pending` 可判定。

**新增字段**

- `provisioning_tasks`：`attempt_count`（默认 0）、`last_attempt_at`、`resolved_by_user_id`、`resolution_note`。全部可空或带默认值，**存量行无需回填**。
- 任务响应新增 `attemptCount`/`maxAttempts`/`lastAttemptAt`/`resolvedByUserId`/`resolutionNote`。前端据此决定是否显示「重试」按钮。

**升级影响**

- 需要数据库迁移：`20261021_provisioning_task_recovery.sql`（Ent 自动迁移同样会加列）。
- 存量 `running` 任务若 `last_attempt_at` 为空，会被租约判定视为「无人持有」并允许重新领取——这是有意的：升级前崩溃留下的僵尸任务必须能自愈。升级前请确认没有正在执行的交付。
- 存量 `failed` 任务 `attempt_count=0`，升级后仍可重试。

**验证**

```bash
cd itsm-backend && go test -count=1 ./service/ -run "Provisioning|ExecuteTask|RetryTask|ResolveTask" && \
  go test -count=1 ./handlers/provisioning/
```


### 1.57 知识发布门禁：发布必须过评审（2026-10-22，行为变更：未过评审 publish 由 200 变 409）

**行为变更：`POST /knowledge/articles/:id/publish` 增加评审门禁**

- `review_status != approved` 的文章调用 publish 返回 **409/`4090`「文章尚未通过评审，不能发布」**。此前该端点直接写 `is_published=true`，等于给评审流开了一条旁路：作者可以绕过「不能自审」（`applyReview` 拒绝作者批自己的稿）把自己写的草稿直接放进 RAG——空壳内容进语料正是 422 质量门要防的事。
- 正确路径：`POST .../review/submit`（作者送审）→ 由**其他**评审人 `POST .../review/approve`（通过即自动发布，并在**同一事务**内入向量索引）。
- `unpublish` 不设门槛（撤下内容永远是安全方向）；`approved` 内容可 unpublish 后再 publish 恢复。
- 已发布文章的重复 publish 改为幂等：不再重复产生向量索引任务。

**集成方影响**

- 若有自动化脚本或外部集成直接调用 publish 去发布草稿，升级后会收到 409，需改为 submit→approve 流程。
- 前端详情页同步调整：草稿展示「提交评审」，`under_review` 只给禁用提示，`approved` 且未发布才给「发布文章」；知识库此前缺少的「作者送审」入口一并补上（审批页只有 approve/reject）。

**验证**

```bash
cd itsm-backend && go test -count=1 ./handlers/knowledge/ -run "PublishRequiresApprovedReview|ApprovePublishes|UnpublishAlwaysAllowed|DraftNotRetrievable"
cd itsm-frontend && npx jest --coverage=false --runInBand ArticleDetail
```

### 1.58 变更审批会签门禁 + 终态收口原子化（2026-10-22，行为变更 + 事务语义修正）

**行为变更一：`POST /changes/{id}/approve` 不再"一票即批准"**

会签（`parallel`/`all`）、N of M 阈值与多级审批链的条件此前只写在**无路由**的 legacy 入口里，
生产路径（`TransitionStatus`）在完成 BPMN 待办后直接把状态 CAS 成 `approved`。结果是：
两人会签的变更，第一位审批人批准后整单即进入 `approved`，第二位审批人的待办形同虚设。

现在按审批链求值：

| 场景 | 变更前 | 变更后 |
|---|---|---|
| 会签 2 人，只批 1 票 | `200` + `approved`（缺陷） | `200` + 仍为 `pending`，该票已写入审计表 |
| 会签 2 人，2 票齐 | `200` + `approved` | `200` + `approved` |
| 3 人链阈值 2，只批 1 票 | `200` + `approved`（缺陷） | `200` + 仍为 `pending` |
| 多级链只批第 1 级 | `200` + `approved`（缺陷） | `200` + 仍为 `pending` |
| 链上已有驳回票，再调 approve | `200` + `approved` | `409`（`ErrInvalidTransition`） |

**集成方影响**：若自动化脚本以"调一次 approve 就拿到 `approved`"为前提，会签/多级链下会
拿到 `200` + `pending`。请读取响应体 `data.status` 判定，不要只看 HTTP 码；未达阈值时不要重试
（重复审批会因"不是待办审批人"返回 403）。

**行为变更二：终态流转的审批链收口进同一事务**

`reject` / `complete` / `cancel` / `rollback` / `close` 在**同一事务**内把残留 `pending`
审批链节点置为 `obsolete`。此前收口在 `commit` 之后用全局 rawDB 执行、失败只记日志，会留下
"变更已终态、审批链仍 pending"的半写状态（审批人待办里永远挂着已终结的单）。现在收口失败
整体回滚并返回错误——不会再出现状态成功、链未收口的部分成功。

**行为变更三：删除无路由的 legacy 审批写入口**

`Service.ProcessApproval` 与 `Repository.UpdateApprovalRecord` 已删除（零调用点、无路由）。
它绕过 BPMN 桥接、把"改审批记录 / 改变更状态 / 收口审批链"分成三段各自成事务。
若你方的二次开发代码调用了这两个 Go 方法，需改为调用 `Service.TransitionStatus`。

**已知边界**：变更未绑定运行中流程实例时（纯业务回退），审批决定不落
`ProcessApprovalDecision`，会签无法求值。该边界在 **1.59** 收紧为失败关闭——
不再"按单票批准"，本批只保证主路径正确，回退路径的缺口见下一节。

**验证**

```bash
cd itsm-backend && go test -count=1 ./handlers/change/ -run "TransitionStatus_ParallelQuorum|TransitionStatus_ThresholdNotMet|TransitionStatus_RejectedDecisionBlocksApprove|TransitionStatus_Reject_ClosesChainsSameTx|TransitionStatus_ChainCloseFailure|EvaluateApprovalQuorum"
```

### 1.59 会签门禁补齐回退路径：缺流程模板时改为失败关闭（2026-10-22，行为变更：未集齐由 200+approved 变 409）

1.58 修好了"有流程实例"的主路径，但留了一个同源缺口：变更**没有绑定运行中的流程实例**
时（`approvalBridge` 为 nil，或桥接返回 `handled=false`），审批决定无处落库，
会签票无法累积，代码退回"桥接后无条件 CAS `approved`"。效果与 1.58 修复前一样——
会签链在缺少流程模板的租户上仍被一个人批掉，只是原因从"逻辑缺失"换成了"配置缺失"。

现在该路径按「当前这一票」求值，不够就失败关闭：

| 场景（均无运行中流程实例） | 变更前 | 变更后 |
|---|---|---|
| 会签 2 人，批 1 票 | `200` + `approved`（缺陷） | `409`（`ErrQuorumRequiresWorkflow`） |
| 多级链只批第 1 级 | `200` + `approved`（缺陷） | `409` |
| 单审批人链 | `200` + `approved` | `200` + `approved`（不变） |
| 或签 2 候选取 1 票 | `200` + `approved` | `200` + `approved`（不变） |
| 未配置审批链（预授权/紧急） | `200` + `approved` | `200` + `approved`（不变） |

**为什么不补写决策行**：`ProcessApprovalDecision` 的 `process_instance_id` /
`process_task_id` / `process_node_key` / `task_id` 都是流程事实，回退路径里没有这些东西，
凭空造一个等于往不可变审计表写假账，事后无法与真实流程对账。

**升级动作**：需要会签 / N of M / 多级审批的租户，必须为对应变更类型配置审批流程模板；
否则审批人调用 `POST /changes/{id}/approve` 会拿到 `409` 与可行动提示（提示文案包含
"配置审批流程模板"）。只做单级单审批的租户无需任何动作。

**验证**

```bash
cd itsm-backend && go test -count=1 ./handlers/change/ -run "QuorumWithoutWorkflowInstance|SingleVoteSufficientWithoutInstance|QuorumWithoutEntClient|NoBoundInstanceFallsBack"
```

### 1.60 变更实施结果留痕：新增 `/fail`，失败与回滚必须写原因（2026-10-22，新端点 + 行为变更 + 新列）

**背景**：状态机一直允许 `in_progress|failed → failed`，`GET /changes/stats` 也有 `failed`
计数，但**没有任何 API 入口**能把变更推进到 `failed`。于是实施失败的变更只能被报成
`completed`（"成功率"虚高）或直接 `rolled_back`（失败原因与回滚原因混为一谈）。
同时 `TransitionStatus` 的 `reason` 只被审批桥接消费，非审批流转的理由**完全不落库**——
审计只能看到"状态从 in_progress 变成 rolled_back"，看不到为什么。

**变更一：新增 `POST /api/v1/changes/{id}/fail`**（权限码 `change:write`）

`failed` 不是死胡同：`failed → scheduled`（重新排期）、`failed → rolled_back`、`failed → cancelled`
都是合法迁移。`GET /changes/stats` 的 `failed` 计数从"永远为 0"变成真实来源。

**变更二：`fail` / `rollback` 必须填写原因**

| 场景 | 变更前 | 变更后 |
|---|---|---|
| `POST /changes/{id}/fail` | 端点不存在 | `200`，状态 `failed` + `outcomeNote` |
| `POST /changes/{id}/fail` 不带原因 | — | `400`（`1002`）+ 指出缺原因及其用途 |
| `POST /changes/{id}/rollback` 不带原因 | `200`，原因丢弃 | `400`（`1002`） |
| `POST /changes/{id}/complete` 带原因 | 原因丢弃 | 写入 `outcomeNote`（可选） |

请求体字段 `reason` 与 `comment` 等价（`comment` 优先，空则取 `reason`）。

**变更三：`changes` 新增列 `outcome_note`，创建 PIR 收紧状态门禁**

- 迁移 `itsm-backend/migrations/20261022_change_outcome_note.sql`（`ADD COLUMN IF NOT EXISTS`，幂等）。
  auto-migrate 会自动加列；只跑版本化迁移的库请执行该文件。
- `GET /changes/{id}` 响应新增 `outcomeNote`；`outcome_note` 与状态在**同一条 UPDATE** 落库。
- `POST /changes/{id}/pir` 此前对任何状态开放（草稿也能建 PIR，把"实施成功率"指标搞乱），
  现在只接受 `completed` / `rolled_back` / `failed`，其余返回 `409` +「尚未实施完成」。

**集成方影响**：脚本若调用 `/rollback` 不带原因会开始收到 `400`；把回滚原因放进
`{"comment": "..."}` 同样生效。以 `completed` 记录实施失败的脚本应改调 `/fail`——
否则失败率指标与 PIR 结论仍然是错的。

**验证**

```bash
cd itsm-backend && go test -count=1 ./handlers/change/ -run "FailRequiresReason|RollbackRequiresReason|FailRecordsOutcomeNote|RollbackRecordsOutcomeNote|FailedCanRescheduleOrRollback"
cd itsm-backend && go test -count=1 ./service/ -run "CreatePIR_"
```

### 1.61 变更回滚清单：失败可回退从「一次状态翻转」变成「逐项确认」（2026-10-22，新端点 + 新表 + 新门禁）

**背景**：能力契约链路 C（受控变更）的最后一条上线门禁是「失败可回退」。此前它只落在一次
状态翻转上——`rollback_plan` 是一段自由文本，提交门禁只校验非空；写了几步、走到第几步、
每步成没成，系统一概不知。审计只能看到「状态从 in_progress 变成 rolled_back」，而真实事故里
最难回答的正是：回滚到第几步了？哪一步没回成？谁确认的？PIR 的「影响是否已复原」因此只能靠记忆填。

**变更一：新增 `change_rollback_steps` 表（一行一步）**

迁移 `itsm-backend/migrations/20261022_change_rollback_steps.sql`（`CREATE TABLE IF NOT EXISTS`，幂等）。
每步带 `status`（`pending`/`done`/`skipped`/`failed`）、`note`、`resolved_by`、`resolved_at`，
并有 `(change_id, step_no)` 唯一索引与 `(tenant_id, change_id, step_no)` 查询索引。

> 不复用 007 迁移预建的 `change_rollback_plans.rollback_steps`（JSONB）：逐项确认需要每步独立的
> 状态/备注/确认人/时间，并要求按行做租户谓词更新与并发 CAS。JSONB 数组只能整段覆盖，
> 并发确认会互相踩踏，也拿不到「谁在什么时候确认了第 3 步」这一行级事实。
> （`change_rollback_plans` / `change_rollback_executions` 至今零 Go 引用的空表，本版仍未使用。）

**变更二：新增三个端点**

| 端点 | 权限码 | 说明 |
|---|---|---|
| `GET /changes/{id}/rollback-steps` | `change:read` | 按 `stepNo` 升序返回清单 |
| `PUT /changes/{id}/rollback-steps` | `change:write` | 覆盖式编排，`{"steps":["摘流量","回滚镜像"]}` |
| `POST /changes/{id}/rollback-steps/{stepId}` | `change:rollback` | `{"status":"done\|skipped\|failed\|pending","note":"..."}` |

`GET /changes/{id}` 响应新增 `rollbackSteps`（`resolved` 为派生字段）。`status=pending` 表示撤销误点。

**变更三：`rolled_back` 增加门禁（行为变更）**

| 场景 | 变更前 | 变更后 |
|---|---|---|
| 清单存在且仍有 `pending` 步骤时回滚 | `200`（无证据也照样"已回滚"） | `422`（`ErrRollbackStepsUnresolved`），文案点出还差哪几步，状态不动 |
| 清单全部已确认（含 `failed`） | `200` | `200`（`failed` 也算确认：回滚失败是真实结果） |
| 没有清单 | `200` | `200`——应急回滚不能被「你还没编排清单」挡住 |
| 清单读取失败 | `200` | 拒绝（失败关闭：读不到就无法证明已走完，而回滚不可逆） |
| 实施开始后改清单 | — | `409`（`ErrRollbackChecklistLocked`）：否则「清空清单→点回滚」即可绕过门禁 |
| 重复确认同一步 | — | `409`，文案说明已被谁于何时确认，不覆盖前人结论 |

**集成方影响**：已有变更都没有清单，回滚行为不变。若脚本先 `PUT /rollback-steps` 再回滚，
必须先把每步确认完，否则会收到 `422`；不想留证据的应急场景不要创建清单。

**刻意不做的事**：不提供「由 ITSM 执行回滚脚本」。回滚多在生产高危路径（数据库恢复、流量切换），
代跑脚本的风险远大于收益；本版承诺的是「可编排、可逐项确认、可审计」，执行仍由人或既定 runbook 完成。

**验证**

```bash
cd itsm-backend && go test -count=1 ./handlers/change/ -run "RollbackBlockedByPendingStep|RollbackAllowedWhenAllStepsResolved|RollbackWithoutChecklistStillAllowed|RollbackChecklistReadFailureFailsClosed|ReplaceRollbackSteps|ResolveRollbackStep|RollbackStepRepository|ValidateRollbackChecklist"
cd itsm-frontend && npx jest src/components/change/__tests__/ChangeDetailChecklist.test.tsx
```

注入回归：临时摘掉 `TransitionStatus` 里的 `validateRollbackChecklist` 调用，
`RollbackBlockedByPendingStep` 与 `RollbackChecklistReadFailureFailsClosed` 均转红（断言"不得标记已回滚"），
恢复后全绿——测试锁定的是门禁本身，不是管道。

### 1.62 连接器健康面跨租户收口（2026-10-22，行为变更：响应形状 + 探测范围）

**缺陷**：`GET /connectors/health` 用的是 `Manager.HealthCheckAll`。任何持 `connector:read` 的租户都能：

1. 读到**全部租户**连接器的 `name/provider/ok`（响应以 `tenantID/name/provider` 为键）；
2. 借一次请求让服务端去**拨号**别家租户的 SMTP / 飞书 / 钉钉端点（健康检查是真实拨号，5 秒超时）——
   等于给普通租户一个跨租户连通性探针。

同一缺陷也在 `GET /connectors`、`GET /connectors/configs`、`GET /connectors/lifecycle` 与
`POST /connectors/configs` 的响应里（这几处输出自身按租户过滤，但**拨号**是全租户的）。

修法早已写好却从未接线：`Manager.HealthCheckTenant` 带着"不能借健康检查探测别家端点"的注释，
`Handler.TenantHealth` 也做了租户过滤，两者都是**零路由引用**的死代码。

**行为变更**

| 项 | 变更前 | 变更后 |
|---|---|---|
| 探测范围 | 全租户实例 | 仅认证租户实例（`HealthCheckTenant`） |
| 响应形状 | `map["<tenantId>/<name>/<provider>"] → {ok,...}` | `{tenantId, items:[{name, provider, ok, latencyMs, message, checkedAt, extra}], checkedAt}` |
| 排序 | map 无序 | 按 `name`、`provider` 升序（便于对比/快照） |
| 其他四个读面 | `HealthCheckAll` | `HealthCheckTenant`（响应形状不变） |

**顺带修掉的前端缺陷**：管理页取健康状态写的是 `health[`${0}/${name}/${provider}`]`，租户号被硬编码成 `0`
（实际是 `1`），所以键永远匹配不上 —— **健康状态列从未显示过真实结果**，一直回落到"已启用"的绿标。
现在按 `name`/`provider` 定位，实例异常会显示红标。

**集成方影响**：直接解析 `map` 形态的脚本会取不到值（本来也只能靠猜租户号）。
改读 `data.items[]`，用 `name` + `provider` 匹配即可；只关心"某个连接器通不通"的调用方也可以直接读
`GET /connectors`（`installed`/`enabled`/`healthy` 字段一直是对的）。

**验证**

```bash
cd itsm-backend && go test -count=1 ./handlers/connector/ -run "Health_OnlyProbesCurrentTenant|ConnectorReadEndpoints_OnlyProbeCurrentTenant|Health_ItemsSortedAndIdentifiable"
cd itsm-frontend && npx jest --coverage=false 'app/\(main\)/admin/connectors'
```

负证明：`health_scope_test.go` 用计数桩连接器，租户 1 发请求时断言租户 2 的 `HealthCheck` 调用次数为 `0`
——把 `HealthCheckTenant` 换回 `HealthCheckAll` 该用例立刻转红；前端用例在旧取值方式下也会红（状态列恒为"已启用"）。

### 1.63 连接器凭据轮换接线（2026-10-22，新端点 + 语义修正）

`Handler.RotateSecret` 的实现在仓库里躺了很久——**没有任何路由指向它**，是个只能读不能调的
死代码，契约里「租户密钥轮换」因此一直挂在"待验收"。本版把它接成端点：

```http
POST /api/v1/connectors/configs/{name}/rotate-secret
{ "newCredentials": { "appSecret": "new-secret" } }
```

权限码 `connector:write`；成功写 `audit_log`（`resource=connector`、`action=rotate_secret`）。

**语义修正（三处，都是原实现的坑）**

| 项 | 变更前 | 变更后 |
|---|---|---|
| `gracePeriodHours` | 请求参数 + 写进 `settings.grace_period_at`，但**全仓没有任何代码读它** —— 假的回滚窗口 | **移除**。明确语义：旧凭据立即失效，不存在新旧并行窗口（三家 IM 收发验签都只读一份凭据） |
| `settings` 合并 | 直接改 `manager` 内实例共享的 map → 未落库的 `rotated_at` 进了运行态 | 复制后写入（`cloneSettings`） |
| 落库失败 | `manager.Revoke(cfg)` → 连接器直接不可用 | `manager.Provision(previous)` 恢复旧配置，返回 `5xx` 并提示"已恢复原配置" |
| 初始化失败 | `RespondError` → 500，文案是套话 | `400` + "新凭据无法初始化，已保留原凭据"，并把错误文案里回显的凭据值脱敏成 `******` |
| 审计写入 | `RequestBody` 含 `gracePeriodHours` | 只含连接器/provider/操作人/时间/凭据**键名**；审计失败以 error 留痕但不回滚轮换 |

**集成方影响**：请求体不再接受 `gracePeriodHours`（传了会被忽略，不报错）；响应字段由
`{name, gracePeriodAt, gracePeriodHours, rotated}` 变为 `{name, provider, rotated, rotatedAt, note}`。
该端点此前无路由，无存量调用方。

**操作建议**：轮换前先在对方平台保留旧密钥（飞书/企微/钉钉控制台可重新启用），
因为它会立即失效；需要回退时用旧凭据调一次 `POST /connectors/configs` 覆盖回来。

**验证**

```bash
cd itsm-backend && go test -count=1 ./handlers/connector/ -run "RotateSecret" \
  && go test -count=1 ./router/ -run "ConnectorRotateSecretRouteRegistered|ConnectorRoutes_PermissionPrecheckResolves"
```

覆盖：成功轮换（运行态 + 库 + 审计三方一致、响应与审计都不回显凭据）、空凭据 400、
不存在 404、Init 失败保留原凭据、落库失败恢复运行态且实例存活、无审计器仍成功、审计失败不回滚。
负证明：注释掉路由注册行 → `TestConnectorRotateSecretRouteRegistered` 转红（"credential rotation is unreachable"）。

### 1.64 连接器市场安装闭环：记录状态不再代替运行态（2026-10-22，行为变更：安装/配置/卸载语义）

连接器类商品（`type=connector`）安装时要做两件事：写 `tenant_installations` 记录，
把实例注册进 `connector.Manager`。旧实现只保证第一件——`provisionConnector` 的错误被
`logger.Warnw` 吞掉后照样把状态置 `active`：

```go
// 旧实现（已删除）
if err := s.provisionConnector(...); err != nil {
    s.logger.Warnw("Connector provisioning failed during install", ...)
    // 不阻塞安装流程 ← 记录成了 active，运行态里根本没有这个实例
}
installation = ...SetStatus(StatusActive).Save(ctx)
```

后果是故障现场与管理界面相反：市场页显示「已安装」，而发送消息、回调、通知投递全部
失败，排障方向被误导。同一个病还有三处变体：配置更新（成功落库后才 provision，失败
只 Warn）、卸载（`Revoke` 失败只 Warn，记录照样 uninstalled）、卸载后重装（复用历史
记录时**完全不 provision**，直接置 active）。

**现在的语义**

| 动作 | 变更前 | 变更后 |
|---|---|---|
| 安装 | provision 失败 → `active`（静默） | `200` + `status=failed` + `error_message`（真实原因），响应 message 提示补配置 |
| 重复安装（记录 `failed`/`installing`） | 幂等短路，返回那条永远起不来的记录 | **视为重试**，重新注册，成功即 `active` |
| 卸载后重装 | 复用历史记录并直接 `active`，不注册 | 复用记录但重置为 `installing` 后走统一注册路径 |
| 更新配置 | 先落库再 provision，失败只 Warn → 「库是新凭据、内存是旧凭据」 | **先 provision 成功再落库**；失败 `409`，库内保留上一份可用配置 |
| 更新配置（记录原为 `failed`） | 状态不动，永远 failed | 注册成功即转 `active` 并清空 `error_message`（补配置即闭环） |
| 卸载 | `Revoke` 失败只 Warn，记录照样 `uninstalled` | 按 `name` 复查运行态，残留即 `409` 且**记录保持原状态** |

**为什么安装失败返回 200 而不是 4xx**：安装记录确实被创建了，`409`/`500` 会让人以为
什么都没发生。返回 `200 + failed` 让前端能显示「已安装但未就绪 + 原因」，安装痕迹也不丢。

**集成方影响**：`POST /marketplace/items/{id}/install` 的响应 `status` 现在可能为
`failed`（此前恒为 `active`），调用方**必须检查** `status`/`errorMessage`，不能只看 HTTP 码。
`PUT /marketplace/installations/{id}/config` 与 `POST .../uninstall` 新增 `409` 响应。

**验证**

```bash
cd itsm-backend && go test -count=1 ./service/marketplace/ ./handlers/marketplace/
```

覆盖：缺凭据安装 → `failed` 且运行态无实例、成功安装 → `active` 且有实例、`failed`
状态重复安装 → 重试成功且复用同一记录、补配置 → `failed` 转 `active`、配置更新失败 →
库保留旧凭据且旧实例存活、正常卸载 → 实例消失、残留实例卸载 → `409` 且记录不变、
卸载后重装 → 仍走注册（缺凭据时为 `failed`），以及 handler 层两例（`200 + failed` payload、
配置失败 `409` 文案含「已保留原配置」）。

负证明（三处逐一注入并恢复）：把安装失败改回「Warn 后照样置 active」→ 两个安装用例转红；
把配置更新改回静默 → 配置用例转红（`expected ErrMarketplaceConnectorNotReady, got <nil>`）；
去掉卸载后的运行态复查 → 残留用例转红；删掉 handler 的「未就绪」分支 → 用例转红并暴露
旧行为正是 `500 {"code":5001,"message":"安装组件失败"}`。

### 1.65 入站回调去重收口：飞书的重放防护其实没接线（2026-10-22，行为变更：回调去重 + 失败可重投）

飞书 webhook 的去重一直用 Handler 私有的 `map[string]time.Time`，而 `connector.InboundDedup`
（持久化到 `connector_inbound_dedup` 表）虽然被 `SetInboundDedup` 注入，**从未被读取过**——
与健康面、密钥轮换同一个病：能力写了、没接线。企微/钉钉早已用持久化版本，飞书是唯一例外。

后果：进程重启后 nonce 表清零，多副本部署时各存一份，同一事件会被重复处理；而飞书的
`task_event` 是全仓唯一会真正改写业务数据的入站路径（同步任务），重复执行就是重复建/改任务。

顺带发现两处配套缺陷：

| 缺陷 | 变更前 | 变更后 |
|---|---|---|
| 过期行覆盖 | schema 注释承诺「`expires_at` 后允许覆盖」，但唯一约束不会随时间消失，只能靠 `CleanupExpired` —— 而它**零生产调用点**（只有单测调）。清理一旦停摆，同一 `event_id` 永久判重复 | 命中唯一约束时先删掉已过期的同一行再重试一次；不依赖后台任务健康度 |
| 表只增不减 | `CleanupExpired` 无调用方 | 后台任务每 10 分钟清理（`connector-inbound-dedup-cleanup`） |
| 失败重投丢失 | 登记行后若业务处理失败，行仍在 → 平台重投被判「重复」直接 200，事件永久丢失（平台看到的是重复，不会再来第三次） | 新增 `InboundDedup.Forget`：处理失败释放该行再返回 `5xx`，平台重投能重新接管 |

**现在飞书回调的语义**：验签 → ±5 分钟时间窗 → `(tenant_id, "feishu", timestamp:nonce)`
登记去重（持久化）→ 处理 → 成功写 `ok`/`ignored`，失败释放。去重存储不可用时 `500`
fail-closed（放过去重等于允许同一事件被二次执行）。

**集成方影响**：飞书 webhook 在去重存储不可用时从「静默放行」变为 `500`——这是刻意的，
飞书会重试。去重行为对外不可见（重复请求仍然返回成功响应，只是不重复执行业务）。

**验证**

```bash
cd itsm-backend && go test -count=1 ./connector/ ./handlers/feishu/ ./router/
```

覆盖：跨进程重启仍判重复（新 Handler + 新去重器、共享同一 DB——内存 map 做不到这一点）、
租户隔离、失败释放后重投可重新接管、`response_status` 从 `processing` 变为 `ok`、
过期行在清理任务没跑过时也能被覆盖（旧实现必然判重复）。

负证明：删掉 `inboundDedup` 分支退回内存 map → 三个持久化用例全部转红
（"persistent dedup must survive a process restart; the in-memory map cannot"）；
去掉过期覆盖分支 → 过期用例转红。既有 `router` 安全契约（同一 nonce 第二次必须 403）
保持通过——去重仍在 URL Verification 之前，所有回调（含探活）都纳入一次性 nonce。

### 1.66 死信人工重放：从「只有 curl」到产品面，同时堵住重复重试（2026-10-22，行为变更：重试幂等 + 新页面）

命令总线的重放（`/admin/operations/commands/{id}/replay`、`/bulk-replay`）与投递重试
（`/notification-deliveries/{id}/retry`）两个入口后端一直存在，但产品里唯一可点的
「重放」是租户初始化抽屉里的那个按钮——它只覆盖「产品基线安装」这一类命令。
BPMN 服务任务死信、通知投递命令死信、失败投递记录，运维都只能 curl。

这一批补两件事：

**1）新增运维面「管理 → 死信与重放」（`/admin/operations`）**

| Tab | 数据源 | 动作 | 幂等语义 |
|---|---|---|---|
| 命令死信 | `GET /admin/operations/commands?status=dead_letter` | 单条重放 / 按筛选批量重放 | 保留原 `idempotency_key` 与 `fencing_token`，worker 仍按原身份去重，不会与历史成功的业务写入重复 |
| 失败投递 | `GET /notification-deliveries?status=failed` | 单条重试 | 以**新幂等键**入箱，会真实再发一次消息 |

页面顶部共用同一份 `summary`（`deadLetter` / `pending` / `processing` / `stuckLeases`）——
「积压」和「死信」是两种完全不同的处置，混在一起看会导错方向。

**2）投递重试不再幂等漏洞（行为变更）**

`failed` 投递记录刻意保持 `failed`（失败证据可追溯），所以界面上的「重试」按钮永远可点。
此前每次点击都会以新幂等键入一条新命令，每条都真实发送一次：

- 运维排障时连点两下 → 收件人收到两条一模一样的通知；
- 重试成功后记录仍是 `failed`，界面上看不出「已经重试过了」，还会再点第三次。

现在按「重试命令谱系」判定（重试命令的幂等键是 `<原键>:retry:<纳秒>`）：

| 上一次重试的状态 | 结果 |
|---|---|
| `pending` / `processing` | **409**「该投递已有在途重试（命令 #N），请等待结果后再决定是否需要再试」 |
| `succeeded` | **409**「该投递已重试成功（命令 #N），无需再次重试」 |
| `dead_letter` / `cancelled` | 放行——确实没送出去，必须允许再试 |

判定用 `IdempotencyKeyHasPrefix` 查命令表，**并且必须再用精确前缀复核一次**：前缀谓词走的是
SQL `LIKE`，幂等键里常见的 `_` 会被当通配符，`ticket_42_created:retry:` 会命中
`ticketX42Xcreated:retry:`，把无关投递判成「已有在途重试」——那会让这条失败投递永远重试不了。

不新增投递状态字段：命令表已经完整记录重试谱系，给 `notification_deliveries` 加「已重试」
状态会污染投递状态机（`pending/sent/failed` 是投递事实，不是操作历史）。

**集成方影响**：对同一条失败投递重复调用 `retry` 现在返回 `409`（此前每次都返回 `200` 并真实发送）。
需要「无论如何再发一次」的场景请先确认收件目标已修复，再单独发通知，不要指望绕过幂等守卫。

**验证**

```bash
cd itsm-backend && go test -count=1 ./handlers/notification/ ./router/
cd itsm-frontend && npx tsc --noEmit
```

新增 4 例：连点两次第二下 409 且只产生一条重试命令、上一次重试成功后 409、上一次重试
`dead_letter` 时放行、`LIKE` 通配误命中不误判。负证明：把守卫条件短路（`false && blocked`）
→ 前两例转红。

### 1.67 限流与「资源不存在」的响应码修正：403/200 变成 429/404（2026-10-22，行为变更：错误码 + HTTP 状态）

`common.Fail(c, code, msg)` 的第二个参数是**业务码**，HTTP 状态由 `codeStatusPairs` 推导；
未登记的业务码兜底为 HTTP 200。有三处代码直接把 HTTP 状态码当作业务码传入，造成协议层假成功：

| 场景 | 之前 | 现在 |
|---|---|---|
| 登录/普通限流 | 登录 `403` + `2003`（权限不足），普通限流 `200` + `429` | `429` + `4290`，响应含 `retryAfter` |
| `GET /service-requests/:id`（不存在/跨租户） | `200` + `404` | `404` + `4004` |
| `GET /service-requests/:id/sla`（不存在/跨租户） | `200` + `404` | `404` + `4004` |
| 未知 CI 建关系 / 重复关系 / 自环 / 成环 | `500` + `5001`「创建CI关系失败」 | `404` + `4004` / `409` + `4090` / `400` + `1002` / `409` + `4090` |

**集成方影响**

- 依赖 `code === 2003` 判断限流的客户端必须改判 `code === 4290`（或 `HTTP 429`）。
  语义也随之变化：`2003` 表示"没有权限，重试无用"，`4290` 表示"稍后重试"——
  响应仍带 `retryAfter` 秒数。
- 依赖"HTTP 200 + `code:404`"判断服务请求不存在的客户端需同时接受 `HTTP 404`；
  `itsm-cli` 已按业务码统一分类，两种历史形态都能识别为 `not_found`。
- CI 关系创建的失败分支从统一 500 拆分为 400/404/409，错误兜底文案不再是唯一线索。

**验证**

```bash
cd itsm-backend && go test -count=1 ./common/ ./service/ ./handlers/service_request/ ./handlers/ticket/
cd itsm-cli && npm run check
```

### 1.68 变更域第二写路径清零 + 服务目录项绑定校验对称化（2026-10-23，行为变更：无效绑定从静默保存变报错）

**变更域（内部收敛，无 API/迁移/配置变化）**：`ProcessApproval` 删除后遗留的复核器
`checkAndTransitionChange` 整体删除——它自带「读状态 → 独立 CAS」的第二条状态写路径，
无事务、无 BPMN 桥接、不收口审批链；会签/阈值/并发守卫测试全部改走 `TransitionStatus`
生产入口。无事务退化路径（mock/legacy 装配才可达）的终态流转把状态与结果说明合并为
单条条件 UPDATE。自部署用户无感知。

**服务目录项绑定校验对称化（自部署用户需关注）**：`ServiceCatalogItem` 的
`sla_id` 此前**从未被校验**——可以指向不存在的 SLA；Update 可把 `approval_chain_id`/
`sla_id` 改成任意整数、`requires_approval` 置 true 可不携带审批链。这些「配置成功、
消费必失败」的假绑定现在会在保存时报错：

- `sla_id`：必须存在且属于当前租户（否则报 `sla definition not found in tenant`）；
- `approval_chain_id`：必须存在、属于当前租户且为 active；
- `requires_approval=true` 时审批链必填（Update 按合并后的生效值判定，未携带的
  字段沿用库中现值——已有合法绑定的条目改名不受影响）。

此前保存成功的非法绑定不会在升级时被清理，需要自查：

```sql
SELECT id, name, sla_id FROM service_catalog_items
WHERE sla_id > 0 AND sla_id NOT IN (SELECT id FROM sla_definitions);
```

（审批链同理。）查出的脏数据修复方式：更新为合法 ID，或把 `sla_id`/`approval_chain_id`
清零（ent 的 Optional int 字段 0 表示未绑定）。

**验证**

```bash
cd itsm-backend && go test -count=1 ./service/ -run 'Binding' && go test -count=1 ./handlers/change/
```

### 1.69 管理者统一服务质量视图：`GET /api/v1/reports/service-quality`（2026-10-23，新增端点 + 口径版本契约）

报表域预览收口：把散在工作台 KPI、`/reports/sla`、`/reports/incidents`、各单域报表与
analytics 工单统计里的局部指标聚合成一次请求的管理视图。权限 `dashboard:read`（报表
资源统一收敛在 dashboard），无迁移、无配置变更。

**响应结构**（`common.Response.data` 为 `service.ServiceQualityReport`）：

- `metricVersion` / `metricVersions`：**口径版本契约**。任何指标的计算语义变更
  （统计窗口、过滤条件、合规率分母、环比基线）都必须递增对应版本号；下游管理层
  月报与外部对账脚本应以版本号感知口径变化，禁止凭数字猜语义。首版 `1.0`。
- `scope`：KPI 行级口径标注（`tenant` / `own`），与工作台 `/dashboard/overview`
  同一规则（`datascope.ForTicketReader`）——管理者看全租户，坐席看本人。
- `kpi` / `sla` / `incidentDistribution` / `satisfaction` / `ticketTrend` /
  `ticketStats` 六块；空集合恒为 `[]` 而非 `null`。
- `days`：统计窗口，默认 30、钳制 [1, 90]。`ticketStats` 块在 analytics 未装配时
  显式缺省——可见降级，不冒充全量。

**给下游消费方**：如果你此前在前端把多个报表端点的数字自己拼成管理视图，迁移到本
端点后请记录首次接入时的 `metricVersion`；后续升级 CHANGELOG 出现「报表口径版本
递增」字样时，按 UPGRADE 对应小节核对数字含义是否变化。

**导出审计说明**：`POST /api/v1/analytics/export` 的审计由全局审计中间件承载
（`export` 动作 + 请求体摘要入 `audit_logs`），此前契约里「导出审计缺失」的表述
经核实为过度报告，本版已如实收敛。

**验证**

```bash
cd itsm-backend && go test -count=1 ./service/ -run 'TestServiceQualityReport' \
  && go test -count=1 ./handlers/ -run 'TestParseQualityWindow|TestGetServiceQualityUnwired'
```

### 1.70 AI 分诊服务端底账 + 置信度契约收紧（2026-10-23，行为变更：`POST /ai/audit` 越界 confidence 返回 400）

AI 域预览收口（置信度强制化 + 反馈收集）：AI 建议的采纳/拒绝反馈从「前端自报」改为
「服务端底账」，置信度口径统一 0-1 并在入口强制校验。

**分诊 `POST /api/v1/ai/triage` 响应新增字段**（向后兼容，纯新增）：

- `promptVersion`：口径版本，当前 `triage-v1`（分词语义/类别枚举/置信度归一化变更必须递增）。
- `model`：决策链路溯源（`guidance` / `llm` / `keyword`），是「哪条路径给出的建议」，
  不是具体模型名（模型名随配置漂移，不适合做契约）。
- `auditId`：服务端决策底账行 ID。服务端在产出建议时自动写一条 `item_type='ai_audit'`
  行（溯源字段由服务端写入，`accepted=NULL` 未决策）。前端采纳/拒绝改为
  `PATCH /api/v1/ai/audit-logs/{auditId}` 翻转底账——旧后端无 `auditId` 时前端自动
  回退旧的 `POST /ai/audit` 自报路径。底账写失败不阻塞建议返回。

**行为变更（集成方影响）**：`POST /ai/audit` 的 `confidence` 必须落在 `[0,1]`
（`0`/`1` 边界合法），越界返回 400 / `1002`，此前是静默 clamp 到 score 0-100。
非法值进入 evaluator 的置信度校准分桶会把整个桶污染成"低置信"，因此不再容忍。

**顺带修复**：分诊此前调 `Suggest`（内部硬编码 `tenant=1`），Guidance sidecar 拿到
的租户上下文是错的，现改用租户版 `SuggestForTenant`。工单 RCA 响应补
`promptVersion`（`ticket-rca-v1`，与 `incident-rca-v1` 同一契约）；RCA/事件分析的
`ai_analysis_results` 持久化记录此前 `confidence_score` 恒空，现携带真实置信度。

**验证**

```bash
cd itsm-backend && go test -count=1 ./handlers/ai/ ./service/ -run 'TestTriage|TestRecordAudit_Confidence|TestAIResultTrace'
cd itsm-frontend && npx tsc --noEmit
```

### 1.71 云发现 Job 闭环：`POST /cmdb/discovery/jobs` 从 fail-closed 转正（2026-10-23，新增端点 + 行为变更）

CMDB 云发现此前只有来源 CRUD 与结果读面，`POST /discovery/jobs` 刻意返回 503
（「禁止创建永远停在 queued 的假任务」）。本版打通完整执行链：

- **创建即入队**：来源/账号校验后，`discovery_jobs` 落库与出箱命令
  `cmdb.cloud_discovery.run` 入队**同一事务**——不存在没有命令的僵尸任务。
  携带 `idempotencyKey` 重复创建返回 409；不携带则每次唯一键（手动重跑不受限）。
- **Job 级执行器**（唯一生产路径）：认领（租约 + fencing token）→ 按来源
  `serviceCodes × regions` 逐 scope 拉取 → 与 `cloud_resources` 按 `identity_hash`
  diff → `discovery_results` 落证据 → 资源写回并维护 `first_seen_at/last_seen_at/
  missing_count`（连续缺失达 `staleThreshold` 置 `stale`）→ 终态 + summary +
  来源 `last_success_at`。取消在 scope 边界生效。
- **新读/操作面**：`GET /discovery/jobs`（分页/按状态过滤）、`GET /discovery/jobs/{id}`、
  `POST .../cancel`、`POST .../retry`（failed/partial_failed → 派生新任务，
  `parentJobId` 溯源）。租约字段不外露。
- **旧路径移除**：`Runner.RunAll`（全账号直写 CI、绕过 jobs/results/CloudResource
  三张表、忽略命令 payload）及其 worker 已删除，`cmdb.cloud_discovery.run` 命令
  改由 job 级执行器消费（payload 缺 `discoveryJobId` 的历史命令会进死信，可查明后重放）。
- **当前 provider 面**：阿里云 ECS（capability 端点如实标注）。`schedule` 字段
  仍无 cron 消费者（保持既有状态，未伪造成已支持）。

**验证**

```bash
cd itsm-backend && go test -count=1 ./handlers/cmdb/ ./router/
```

## 2. 环境变量变更

本次升级**移除了多个"幽灵配置项"**（在示例文件中声明但代码/Compose 从不读取，用户配置了也不生效），并修正了一个 Grafana 密码安全缺陷。

### 2.1 已移除的变量（从 `.env*` 示例中删除）
如你的 `.env` 仍设置了以下变量，它们现在**无任何效果**，可安全删除：

| 变量 | 原位置 | 说明 |
| --- | --- | --- |
| `SLA_CHECK_INTERVAL` | `.env.example` / `.env.dev.example` | SLA 扫描间隔实际硬编码在服务内 |
| `ESCALATION_CHECK_INTERVAL` | `.env.example` / `.env.dev.example` | 升级检查间隔硬编码在服务内 |
| `EMBEDDING_PIPELINE_INTERVAL` | 全部示例 | 知识库 RAG 未读取该 env |
| `EMBEDDING_BATCH_SIZE` | 全部示例 | 同上 |
| `EMBEDDING_FULL_PASS_SIZE` | 全部示例 | 同上 |
| `MINIO_PORT` | `.env.example` / `.env.dev.example` | MinIO 未随默认部署启动（见 2.3） |
| `GRAFANA_PASSWORD` | 全部示例 | **已废弃**（见 2.2） |
| `ENABLE_METRICS` | `.env.prod.example` | 代码实际读取 `ITSM_ENABLE_PUBLIC_METRICS`；Compose 中残留的 `- ENABLE_METRICS=true` 已删除（死配置） |

### 2.2 新增 / 修正：`GRAFANA_ADMIN_PASSWORD`（安全修复）
原 `GRAFANA_PASSWORD` 从未被 Grafana 读取（Grafana 实际读 `GF_SECURITY_ADMIN_PASSWORD`），导致 Grafana 管理员密码静默回退到弱默认 `admin123`。

- 生产示例新增 `GRAFANA_ADMIN_PASSWORD=`（建议强密码）。
- `scripts/deploy-prod.sh` 现在生成并写入 `GRAFANA_ADMIN_PASSWORD`，而不是无效的 `GRAFANA_PASSWORD`。
- monitoring 堆栈通过 `monitoring/docker-compose.monitoring.yml` 的
  `GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_ADMIN_PASSWORD:-admin123}` 读取。

> 如果你之前依赖 `GRAFANA_PASSWORD`，请改为设置 `GRAFANA_ADMIN_PASSWORD`，否则 Grafana 仍为默认密码。

### 2.3 `SERVER_PORT` 仍有效（保留）
`SERVER_PORT` 通过 `itsm-backend/config.yaml.example` 的 `server.port: ${SERVER_PORT:8080}` 被后端消费，**保留**，请勿删除。`config.yaml` 由后端 `config.LoadConfig()` 加载并做环境变量替换。

---

## 3. 数据库迁移

后端通过 Ent 自动迁移（`client.Schema.Create()`）应用 schema 变更，**无需手写 SQL**。

- 生产常驻后端建议 `ITSM_AUTO_MIGRATE=false`，由一次性 `itsm-init` 任务执行迁移。
- 手动触发迁移：
  ```bash
  # 安全：仅创建/更新表结构，绝不 DROP DATABASE
  make db-reset        # 交互确认后执行迁移；CI 可用 DB_RESET_CONFIRM=reset 跳过确认
  ```
  > 旧 `make db-migrate` 已被替换为安全的 `make db-reset`。不要再使用名称含 `db-migrate` 的旧脚本——历史上曾有同名脚本执行 `DROP DATABASE`。

- 升级前务必 `pg_dump` 备份；迁移不可逆时可用备份恢复。

---

## 4. Docker Compose 部署

### 4.1 生产部署
```bash
cp .env.prod.example .env.prod
# 编辑 .env.prod：DB_PASSWORD / JWT_SECRET / ADMIN_PASSWORD / CORS_ALLOWED_ORIGINS / GRAFANA_ADMIN_PASSWORD
./scripts/deploy-prod.sh init     # 生成强密钥并写入 .env.prod（chmod 600）
./scripts/deploy-prod.sh deploy   # 拉起全部服务（含 nginx 与 itsm-worker）
```

`deploy-prod.sh deploy` 现已包含：
- 启动 `nginx` 与 `itsm-worker`（历史版本漏起，导致静态资源 502 / 异步任务不执行）。
- `nginx` 健康检查（`:80`）+ `itsm-worker` 存活检查。
- 登录冒烟测试读取 `.env.prod` 的 `ADMIN_PASSWORD`（不再硬编码 `admin123`）。

### 4.2 MinIO 为可选（storage profile）
后端与前端当前**不使用** MinIO/S3。MinIO 仅在 `docker-compose.prod.yml` 的 `storage` profile 下启动：
```bash
docker compose --env-file .env.prod -f docker-compose.prod.yml --profile storage up -d minio
```
常规部署不需要 MinIO，`MINIO_*` 均可留空。

### 4.3 开发部署
```bash
cp .env.dev.example .env
docker compose -f docker-compose.dev.yml up -d
```
`docker-compose.dev.yml` 的核心服务已移出 `dev` profile，默认即可启动；`--profile dev` 仍向后兼容。

---

## 5. 镜像标签与版本固定

- 生产环境**固定具体标签**（如 `:1.6.9`），不要使用 `:latest`，便于回滚。
- 镜像：`ghcr.io/heidsoft/itsm-backend`、`ghcr.io/heidsoft/itsm-frontend`、`ghcr.io/heidsoft/itsm-init`。
- 升级时前后端标签保持一致，避免 DTO 契约错配（见 1.1）。

---

## 6. 升级后验证

```bash
# 1. 服务健康
curl -f http://localhost:80/healthz        # 经 nginx 探活
curl -f http://localhost:8090/health        # 后端探活（如暴露）

# 2. 登录冒烟（deploy 脚本已内置；手动复验）
curl -c cookies.txt -X POST http://localhost/api/v1/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"<你的ADMIN_PASSWORD>"}'

# 3. 网关/前端资源
curl -fI http://localhost/                    # 返回 200，且引用 /_next/ 资源

# 4. Grafana 密码生效
# 用 .env.prod 中的 GRAFANA_ADMIN_PASSWORD 登录 http://localhost:3000
```

---

## 7. 回滚

1. 停止新版本：`docker compose --env-file .env.prod -f docker-compose.prod.yml down`。
2. 如需回滚 schema 变更：从升级前 `pg_dump` 备份恢复数据库（Ent 自动迁移**不提供** down migration，务必依赖备份）。
3. 拉起旧标签镜像：`docker compose ... up -d` 配合旧 `:1.6.x` 镜像。

---

## 8. 常见问题

**Q: 升级后 Grafana 仍用 `admin123` 能登录？**
A: 你设置的是旧的 `GRAFANA_PASSWORD`。改为设置 `GRAFANA_ADMIN_PASSWORD`（或重新跑 `deploy-prod.sh init`）。

**Q: 前端报 DTO 字段不存在？**
A: 集成方/前端缓存了 snake_case 字段（见 1.1）。清缓存并升级前端到同版本；临时兼容层 `toCamelCase` 仅保留一个版本。

**Q: `make db-migrate` 报错？**
A: 该目标已重命名为安全的 `make db-reset`（见 3）。

**Q: MinIO 启动失败？**
A: 常规部署不需要 MinIO。仅在使用对象存储时加 `--profile storage`（见 4.2）。
