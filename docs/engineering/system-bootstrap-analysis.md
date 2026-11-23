# CloudMesh / ITSM 系统初始化与基线现状审查

> 阶段：第一阶段（只读审查）  
> 日期：2026-11-21  
> 范围：Ent Schema、Migration、SQL Seed、初始化代码、菜单/权限/角色/用户、系统配置、URL 配置、配置与环境变量、Docker/部署/CI/CD  
> 纪律：本轮**未修改任何现有实现**。仅新增本文档；Ent Schema、生成代码、迁移、基线数据、门禁基线均未改动。

## 0. 审查快照与边界

- 会话账本：`2026-11-21-cloudmesh-bootstrap-design`，进入时 `HEAD=ec0c922c04e6`，`origin/main=89ded99a8be1`。
- 工作树进入时有 57 个 WIP 文件（暂存 0 / 修改 37 / 未跟踪 20）；本轮没有触碰这些文件。
- 审查方法：从生产入口反向追踪调用链，按资源类型核对权威数据源、稳定键、唯一约束、幂等写法和覆盖行为；对关键结论做定点代码复读。
- 与已有设计的关系：`plans/production-data-initialization-blueprint.md` 是本仓库已有的权威蓝图，本文档只补充「当前代码实际做到哪一步」的证据，不替代、不重写蓝图。

### 结论摘要

仓库并非从零开始：已有版本化迁移账本、初始化运行账本、组件租约/心跳/fencing、`Plan/Apply/Verify` 接口和只读租户审计。真正阻断目标架构的是五点：

1. **字段级所有权缺失**：绝大多数基线行没有 `source_type` / `managed_fields` / `local_modified_at`，无法区分系统托管字段与管理员自定义字段。
2. **协调语义不完整**：播种以 `Exist → Create` 为主，权限绑定「只增不减」，没有 `create/update/skip/deprecate` 五态计划，也没有三方合并。
3. **升级会覆盖管理员改动**：角色名/描述、菜单名/图标/排序/权限码/父级、Skill 的 `enabled` 都会在重跑或被新 checksum 触发时被覆盖。
4. **漂移检测只验存在性**：`VerifyTenantBaseline` 只检查期望 code/path 是否存在与权限是否包含，字段被改不会报漂移。
5. **部署链存在错线**：生产 compose 的 init 容器与 K8s 迁移 Job 不一致；`systemUrl` 等 URL 未走已有 SSRF 校验；`smtpPassword` 落在普通配置表。

蓝图 §6.1 的 24 项生产发布门禁当前全部为 `[ ]`（`plans/production-data-initialization-blueprint.md:547-570`）。

---

## 1. 现有初始化入口

### 1.1 入口矩阵

| 入口 | 触发方式 | 实际行为 | 证据 |
|:--|:--|:--|:--|
| Web 进程 | `./main`，无 `ITSM_BOOTSTRAP_ONLY` | `NewApplication().Run()`；**显式拒绝** `ITSM_AUTO_MIGRATE` / `ITSM_AUTO_SEED` | `itsm-backend/main.go:30-41`；`internal/bootstrap/app.go:1604-1617` |
| Bootstrap Job | `ITSM_BOOTSTRAP_ONLY=true` | `RunInitialization()`：结构迁移 + 版本化 SQL 迁移 + 租户护栏 + 7 组件初始化 + 运营索引收敛 | `main.go:35-38`；`app.go:1802-1827`、`:1619-1712` |
| 初始化 CLI | `cmd/initialize`，子命令 `plan/apply/retry/status/verify/audit-tenants` | 与 Bootstrap Job 共用 `ProductionAssembly`；`plan` 与 `audit-tenants` 明确只读 | `cmd/initialize/main.go:65-135` |
| 租户开通 | `handlers/tenant/initialization.go:177` / `cmd/provision_tenant` | 在单事务内对目标租户跑全部 7 组件，再验证并写模板版本标记 | `pkg/seeder/tenant_provisioner.go:27-100`、`:146-166` |
| 构建标签工具 | `go build -tags migrate ./cmd/migrate/main.go` | 生产镜像产出独立 `migrate-tool`；**不是** `./main migrate` | `itsm-backend/Dockerfile.prod:71-79` |

### 1.2 Bootstrap Job 的真实执行顺序

1. `InitDatabaseWithRLS()`：`app.go:1802-1827`。
2. `InitializeStorage()` 以 `tenantctx.SystemContext(ctx, "bootstrap:initialize_storage", reason)` 进入，理由为 "schema migration and default seed at process boot"：`app.go:1619-1622`。
3. 迁移前准备（仅 `AutoMigrate=true`）：角色权限租户字段回填、CMDB 模型、工单表单字段、工作流模板 IDENTITY、审批决策索引：`app.go:1624-1643`。
4. Ent DDL：`client.Schema.Create(ctx)`：`app.go:1644`。
5. 版本化 SQL 迁移：`EnsureMigrationsTable → RunMigrations(PostSchema) → FilesystemMigrations → AdoptUnrecorded → MergeWithRegistered → RunMigrations(merged)`：`app.go:1647-1653`、`:1747-1787`。
6. 租户治理门禁（prod fatal / dev warn）：`app.go:1659`、`:1796-1800`。
7. 若 `AutoSeed=true`：检查是否需要首管 → 弱口令风险闸门 → `seeder.NewProductionAssembly` → `PlatformRequest("bootstrap-job", release, platformOnly=true)` → `Engine.Apply`：`app.go:1663-1692`。
8. 运营索引与索引收敛：`app.go:1694-1709`。
9. 实例启动只重跑租户护栏，不再迁移/播种：`app.go:1847-1865`。

### 1.3 初始化引擎的现有能力（应复用，不重写）

- 组件 DAG 与 `Plan/Apply/Verify` 接口：`internal/initialization/engine.go:47-53`；7 个生产组件见 `pkg/seeder/initialization_adapter.go:50-58`。
- 每次组件执行：获取租约 → 起 attempt → 心跳 → 开组件事务 → `Apply` → `Complete` → `Commit`，失败时 fenced 记录并回滚：`engine.go:141-223`。
- 租约实现用 `ON CONFLICT ... fencing_token+1 WHERE lease expired`，心跳丢失后旧执行者被 fence：`internal/initialization/sql_store.go:112-160`、`:162-179`、`:257-287`。
- 清单 checksum 由 JSON payload + 代码内定义共同计算：`pkg/seeder/manifest_digest.go:14-145`；组件 checksum 在 `initialization_adapter.go:71-82` 注入。
- 组件账本表：`initialization_installations/runs/component_attempts/managed_records`，含 `installed_version`、`source_checksum`、`stable_key_aliases`：`migration/migrations.go:447-535`。

**边界**：租约只覆盖步骤 7 的组件 `Apply`；步骤 3–6、8–9 在租约之前/之外执行，多实例并发跑 bootstrap job 仍可能在 DDL 与索引收敛阶段竞争。

---

## 2. 基线来源清单

| 资源 | 权威来源 | 稳定键 | 系统/自定义标记 | 重跑行为 |
|:--|:--|:--|:--|:--|
| 菜单 | `pkg/menubaseline.MenuDefinitions()` → `seeder.go:1930-1990` | `(tenant_id, path)` | 无 `is_system` | `upsertMenu` 覆盖 name/icon/sort/permission_code/parent；**保留** `is_visible`/`is_enabled` |
| 权限码 | `internal/authz.Definitions()` → `seeder.go:1858-1915` | `code`（租户内） | 信封内层 `isSystem` | 存在则跳过；策略信封一次性迁移 |
| 角色 | `BuiltinRoles()` / `PracticeRoles()` → `seeder.go:1252-1281` | `(tenant_id, code)` | `role.is_system` 仅角色有 | 覆盖 name/description/is_system |
| 角色授权 | `internal/authz.BuiltinRolePermissionCodes()` → `seeder.go:2161-2239` | `(role_id, permission_id)` | 无 | **只增不减**；唯一收缩通道是 retired 清单 |
| 首管 | `ADMIN_PASSWORD` bcrypt 建 admin/super_admin；或一次性 bootstrap token | `(tenant_id, username)` | `user.is_bootstrap_admin` | 存在即跳过，不重置密码 |
| 系统配置 | `handlers/systemconfig/service.go:249-306` 的 24 个硬编码键 | `(tenant_id, key)` | 无 | 列表查询在 `total==0` 时懒加载；已存在则跳过 |
| URL 配置 | 同上 `systemUrl`；SSO/connector 各自持有 URL | key / provider key | 无 | 只补缺 |
| 工单类型 | `pkg/seeder/manifest_definitions.go:36-55` | `(tenant_id, code)` | 无 | 存在即跳过 |
| 工作流模板 / BPMN | `manifest_definitions.go:60-74` + 内嵌 `templates/*.bpmn` | `(tenant_id, key, version)` | 无 `is_system` | 存在即跳过，仅修旧占位 XML |
| Skill | `handlers/ai/builtin_skills.go:18-53`，Web 启动时持久化 | `skill_definitions(tenant_id=0, code)` | `version` + `checksum` + 内置能力前缀 | checksum 变化即更新整行，**包括 `enabled`** |
| CMDB 云模板 | `config/cmdb/cloud_service_templates.json` | 模板 key | 无 | 有独立 reconcile 实现，可复用 |
| 手工 SQL | `sql/init_data.sql`、`seed_test_data.sql` 等 | 无 | 无 | psql 手工执行，硬编码 `tenant_id=1` |

`config/seed/default.json`（147 行）与 `demo.json` 提供部门/团队/SLA/审批流/流程绑定/工单视图/CI 类型等；checksum 计算覆盖 JSON 与代码定义（`manifest_digest.go:14-145`）。构建标签 `seed_demo` 的演示播种不在生产镜像默认流程中。

---

## 3. 数据关联

- **租户 → 角色 → 权限**：`roles(tenant_id, code)` 与 `permissions(tenant_id, code)`；`role_permissions(role_id, permission_id, tenant_id)` 是显式关联表。`middleware/rbac_precheck_gen.go` 是路由声明生成的预检表，不是权限注册源；权威源是 `internal/authz`。生产库补了 `uq_roles_tenant_code`、`uq_permissions_tenant_code`、`uq_rp_role_permission`（`migrations/20261027_prod_schema_guards.sql:32-47`）。
- **菜单 ↔ 权限**：`menus.permission_code` 逻辑关联权限码，角色不直连菜单；按 `permission_code` 过滤可见性（`handlers/rbac/menu_service_impl.go`）。菜单业务唯一键 `(tenant_id, path)` 由迁移文件创建，因 entc 对自关联 edge + 组合唯一索引会组合爆炸而未写在 schema 中（`ent/schema/menu.go:64-68`、`migrations/20261011_add_menus_tenant_path_unique.sql:31`）。
- **用户 ↔ 角色**：`user.role` 是主角色，另有 `user_roles` 多角色边；存量回填由 `MigrateUserRolesBackfill` 负责（`seeder.go:1358-1364`）。
- **系统配置 / URL**：`system_configs(tenant_id, key, category)`；`tenant.bootstrap.version.<id>` 也是 system_config 行，用于标记租户模板版本（`tenant_provisioner.go:146-159`）。
- **流程 / 模板**：`process_definitions(tenant_id, key, version)` 与 `workflow_templates(tenant_id, key, version)` 都有版本维度；`process_definition.is_latest` 标记当前版本。
- **账本 ↔ 业务行**：`initialization_managed_records` 保存稳定键别名与源校验和，但绝大多数业务行本身不携带来源字段，因此账本与行之间会漂移。
- **租户边界**：`middleware/tenant.go:179-183` 注入 tenant context；`common/tenantctx/system.go:39-46` 的 `SystemContext` 允许迁移/播种合法跨租户写并打 zap 日志；`database/rls/driver.go:308-321` 对 bypass 放行、对缺失 tenant fail-closed。Ent 写守卫在 `database/security.go:101-149`。

---

## 4. 重复执行风险

| 等级 | 风险 | 证据 | 重复执行的后果 |
|:--|:--|:--|:--|
| P0 | 角色字段覆盖 | `seeder.go:1327-1336` | 管理员改过的角色名/描述被基线重置 |
| P0 | 菜单字段覆盖 | `seeder.go:1968-1992` | 管理员改过的菜单名/图标/排序/权限码/父级被重置；仅 visible/enabled 保留 |
| P0 | Skill 整行 upsert | `handlers/skill/persistence.go:104-137` | 代码 checksum 变化时连 `enabled` 一起覆盖 |
| P0 | 权限自动扩张 | `seeder.go:2161-2239` | 新增内置权限码会自动写入所有匹配角色的授权行 |
| P1 | `Exist → Create` 非原子 | `seeder.go:828`、`:836`、`:2536-2555` | 并发下无 `ON CONFLICT` 保护，靠唯一索引报错而非幂等成功 |
| P1 | 首管硬编码租户 | `seeder.go:968`、`:1020` 使用 `tenant.CodeEQ("default")`，未走 `baselineTenant()` | 当前靠 `installsPlatformTenant()` 守卫；未来调用点移出守卫即串租户 |
| P1 | 模板版本查询缺租户谓词 | `tenant_provisioner.go:148-150` | 按 key 查询未带 `TenantIDEQ` |
| P1 | 迁移早于租约 | `app.go:1624-1653` vs `engine.go:141-223` | 多实例 bootstrap 仍可能在 DDL/索引阶段竞争 |
| P2 | 稳定键护栏缺口 | `permission_definition.go:35`、`role_permission.go:32`、`endpoint_acl.go:63` 写成 `Index()` 而非 `Indexes()` | Ent 不创建这些唯一索引；部分由 SQL 迁移兜底；`endpoint_ac_ls` 无读取方，旧 ACL 迁移已退役（`migrations/20260501_rbac_endpoint_acls.sql:1-24`） |

已具备的保护：首管密码存在即跳过且弱口令被闸门拒绝（`seeder.go:1019-1068`）；角色授权有 retired 收缩清单（`seeder.go:2244-2272`）；迁移 checksum 不一致直接失败（`migration/migrator.go:99-124`）；Web 进程拒绝 DDL/Seed（`app.go:1607-1617`）。

---

## 5. 升级风险

1. **无三方合并**：蓝图要求「旧系统值 / 租户当前值 / 新系统值」三方合并并输出 `create/update/revoke/conflict/orphan`（蓝图 `:104-115`）；现有 `Action{Type,SourceKey,Summary}` 无这些语义（`engine.go:36-40`）。
2. **漂移只验存在性**：`VerifyTenantBaseline` 只验 code/path 存在与权限子集（`tenant_provisioner.go:110-140`）；`AuditTenantBaselines` 是内存报告，不落库（`baseline_audit.go:21-62`）。
3. **审计无 diff**：`initialization_runs/attempts` 有版本、执行人、fencing、结果摘要，但没有前后值、`reason`、`release` 与 dry-run 结果；系统 bypass 只有 zap 日志。
4. **双轨 seed 源**：版本化引擎与无版本的手工 SQL 并存；`sql/init_data.sql` 自述执行方式为 psql（`:2`），且 Go 代码无引用。旧 RBAC ACL 迁移已退役但仍留在目录里。
5. **两版本升级矩阵缺失**：蓝图 §6.1 要求「最近两个版本升级测试通过」（`:565`），仓库中未见对应测试。
6. **RLS 生产默认关闭**：`docker-compose.prod.yml:144` 设 `RLS_MODE=off`；`config/config.go:302-310` 默认也是 off，三档 `off/shadow/enforce` 的生效仍需单独验证。
7. **K8s 迁移 Job 错线**：`deploy/k8s/22-migrate-job.yaml:15-22` 用 `command: ["./main"]` + `args: ["migrate"]` + `ITSM_PROCESS_MODE=migrate`，但 `main.go:30-41` 只判断 `ITSM_BOOTSTRAP_ONLY`；生产镜像真正的迁移工具是 `/out/migrate-tool`（`Dockerfile.prod:71-79`）。该 Job 实际会走 Web 启动路径，不会执行迁移。

---

## 6. 配置、环境变量与部署/CI

### 6.1 关键环境变量

| 变量 | 作用 | 证据 |
|:--|:--|:--|
| `ITSM_BOOTSTRAP_ONLY` | 选择 bootstrap job 而非 Web 进程 | `main.go:35-38` |
| `ITSM_AUTO_MIGRATE` / `ITSM_AUTO_SEED` | 控制 DDL 与组件播种；Web 启动时两者为真即报错 | `config.go:294-295`；`app.go:1607-1617` |
| `BOOTSTRAP_TOKEN_ENABLED` | `0` 走 `ADMIN_PASSWORD` 一次性建管；`1` 走 token 生成/消费两步 | `docker-compose.prod.yml:132-143`；`cmd/initialize/main.go:65-109` |
| `ADMIN_PASSWORD` | 首管密码，存在即跳过、弱口令拒绝 | `seeder.go:1019-1068` |
| `ITSM_RELEASE_VERSION` | 进入初始化请求的发布版本 | `app.go:1683` |
| `RLS_MODE` / `RLS_TENANT_VAR_NAME` | RLS 三档与 GUC 变量名，默认 `off` / `app.current_tenant` | `config.go:302-310` |
| `DEPLOYMENT_MODE` | private/saas/saas_msp 能力裁剪 | `main.go:33` |

### 6.2 Docker 与部署

- `docker-compose.prod.yml:108-155` 定义 `itsm-init`：一次性跑初始化，带 `ADMIN_PASSWORD`、`ITSM_BOOTSTRAP_ONLY=true`、`ITSM_AUTO_MIGRATE=true`、`ITSM_AUTO_SEED=true`、`BOOTSTRAP_TOKEN_ENABLED=0`、`RLS_MODE=off`。
- `itsm-backend/Dockerfile.prod:62-79` 在同一镜像内构建 `/out/main` 与 `/out/migrate-tool`；两者入口不同，K8s Job 未使用后者（见 §5.7）。
- `docker-compose.dev.yml` / `deploy/docker-compose.saas.yml` 沿用同样的 init 模式；另有 `docker-compose.fresh-gate.yml` 专供空卷首装门禁。
- `ITSM_PROCESS_MODE=migrate` 在 K8s Job 中出现，但 `main.go` 未读取该变量——属于配置与入口不一致。

### 6.3 CI/CD

| 工作流 | 覆盖内容 | 对本改造的意义 |
|:--|:--|:--|
| `fresh-install-gate.yml:1-8,44-65` | 空卷首装：迁移入账、BPMN 模板部署、流程绑定零悬空 key；`MIN_MIGRATIONS=50` | 新基线必须继续通过空卷首装 |
| `rls-golden-path.yml:98-104` | `go run -tags migrate .` 后再验证 RLS 角色 | 可作为 RLS enforce 下初始化的基线夹具 |
| `backend-ci.yml:357-392` | `make entgen-check` 全图生成差分门禁；只 diff `ent/schema/**` | 新增 Ent Schema 必须重生成；改动 `go.mod`/生成器不会触发该门禁 |
| `docs-gate.yml:176-199` | C.1–C.6 hard，覆盖 CHANGELOG/ROADMAP 与产品面棘轮 | 新增资源面需同步基线/文档，不能靠临时放宽 |
| `ga-gate.yml` / `security.yml` / `api-contract-check.yml` | 整栈可服务、安全扫描、API 契约 | 阶段 4 CLI 与阶段 5 安全约束的回归面 |

---

## 7. 证据索引（关键路径）

- 入口与顺序：`itsm-backend/main.go:30-41`；`internal/bootstrap/app.go:1604-1617,1619-1712,1802-1827`；`cmd/initialize/main.go:80-128`。
- 引擎与账本：`internal/initialization/engine.go:36-53,124-223`；`internal/initialization/sql_store.go:112-160`；`migration/migrations.go:447-535`。
- 迁移：`migration/migrator.go:50-124`；`migrations/20261027_prod_schema_guards.sql:32-47,65-66`；`migrations/20261011_add_menus_tenant_path_unique.sql:31`。
- 播种覆盖：`pkg/seeder/seeder.go:1019-1068,1327-1336,1968-1992,2161-2239,2383-2412`；`handlers/skill/persistence.go:104-137`；`handlers/systemconfig/service.go:104-109,249-306`。
- 漂移与审计：`pkg/seeder/baseline_audit.go:11-62`；`pkg/seeder/tenant_provisioner.go:110-166`；`common/tenantctx/system.go:39-46`。
- 蓝图与门禁：`plans/production-data-initialization-blueprint.md:76-128,543-570`。

> 本轮到此为止：未修改实现，未生成 Ent 代码，未执行迁移，未改任何基线或门禁文件。
