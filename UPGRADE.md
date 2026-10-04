# 升级指南 (UPGRADE)

本指南面向从 **v1.6.x** 升级到最新版本（含开源就绪加固）的自部署用户，覆盖破坏性变更、环境变量变更、数据库迁移、部署与回滚。

> 当前稳定版本：`1.6.9`（见 `package.json` / `CHANGELOG.md`）。
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
