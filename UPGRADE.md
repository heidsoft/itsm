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
