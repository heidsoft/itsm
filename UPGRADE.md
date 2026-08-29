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

---

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
