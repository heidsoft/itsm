# 产品收敛与初始化权限审查（2026-10-03）

> 审查人：Qoder agent（产品 + 架构视角）
> 审查方式：**全部结论基于当日源码实测与差集计算**，不复述文档、提交说明或门禁的"绿"。
> 审查范围：三轮递进 —— ①产品状态收敛（API / 业务实现 / UI 交互 / 边缘功能）②前端"长文案假实现"③系统数据初始化方式 / 默认菜单 / 角色 / 权限是否合理。
> 审查前提：**只读**。未启动服务，未连接任何数据库（含只读连接），未修改任何业务代码。
> 证据保留度：本文件收录三轮中**仍保有 `file:line` 级证据**的条目。第一轮原始清单为 46 项，此处收录其中可复核的核心项；完整逐条枚举见会话记录。

---

## 0. 总体判断

三句话：

1. **最严重的两个问题都在权限层，且都属于"dev 测不出来、只在生产暴露"的家族。** 一条是未认证即可自选 `super_admin`；一条是两套权限权威对同一角色的定义不一致，导致**播种反而让角色变弱**。
2. **门禁骨架是真实生效的，但有三处"绿"是假保证。** `lint:antd` 只覆盖 9 类禁用 API 中的 3 类；`ga-gate` 在启动核心栈时死掉，唯一的全栈 E2E 从未运行；C.7.1 的粒度看不见 55 个孤儿组件。
3. **初始化架构本身在这一档产品里属偏上水平**，码内单一事实来源 + 逐组件事务账本 + fencing + 尊重运维改动的前向修复都做对了。问题不在骨架，在权限语义有两份定义且只对齐了 2/9 个角色。

**建议的第一个动作不是修功能，是修 P0-1 与 P0-2，并把守卫的检查范围扩到全部词表角色让 CI 红出真实差集。**

---

## 1. P0 —— 必须在下一批之前处理

### P0-1　注册接口可无认证自选 `super_admin`，短路发生在 DBOnly 门禁之前

完整链路，逐跳实测：

| # | 位置 | 事实 |
|:--|:--|:--|
| 1 | `itsm-backend/router/router.go:310` | `public.POST("/auth/register", middleware.LoginRateLimiter(), config.AuthHandler.Register)` —— 位于"无需认证的账号自助端点"组，仅有限流 |
| 2 | `itsm-backend/dto/auth_dto.go:95-96` | `Role string \`json:"role" binding:"omitempty"\``、`TenantCode string \`json:"tenantCode,omitempty"\`` —— 均可从请求体绑定，**无白名单** |
| 3 | `itsm-backend/handlers/auth/service.go:147-151` | `SetRole(user.Role(req.Role))`；唯一兜底是空值 → `end_user` |
| 4 | `itsm-backend/ent/user/user.go:288-289` | `RoleSuperAdmin = "super_admin"`、`RoleAdmin = "admin"` 均为合法枚举，通过 `RoleValidator`（10 个枚举值，`:281-303`） |
| 5 | `itsm-backend/middleware/rbac.go:1204-1231` | 登录后 JWT 的 role 进入 `GetContextRoles` |
| 6 | **`itsm-backend/middleware/smart_permission.go:117-122`** | `for _, r := range roles { if r == "super_admin" { return true } }` —— 在 L1/L2/L3 之前，**也在 `:148` 的 DBOnly fail-closed 门禁之前** |

`smart_permission.go:145-150` 的 Phase-1 修复正确关闭了 L4 硬编码兜底（注释明确写了"DBOnly 下硬编码完全不参与授权"），但 **`:117` 的直通未受模式约束**，DBOnly 拦不住它。同类不受模式约束的直通另有两处：`rbac.go:1181`（`AuthorizeResource`）、`rbac.go:1196`（`AuthorizeResourceForRole`）。

**租户也由攻击者选择**：`handlers/auth/service.go:126-142` 从请求体 `tenantCode` 取租户；不传时若恰有 1 个 active 租户则**自动加入**，否则报"请指定要加入的租户(tenantCode)"。单租户私有化部署正好落在自动加入分支。

**前端把角色自选做成了正常功能**：`itsm-frontend/src/app/(auth)/register/page.tsx:258-290` 有一个 `required` 的角色下拉，四个硬编码选项 `developer`/开发人员、`manager`/项目经理、`admin`/系统管理员、`user`/普通用户；`:70` 为提交入口。而 API 接受的范围远大于 UI（含 `super_admin`、`sysadmin`、`it_admin`、`security_admin`、`audit_admin`）。**UI 限制不是安全边界。**

**生产可达性**：`docker-compose.prod.yml:243` 后端绑 `127.0.0.1:8090`，但 `:376` 注明"routes API traffic through nginx (browser → nginx:80 → backend:8090)"；仓库内无 nginx 规则拦 `/auth/register`。

**测试覆盖为零**：在 `handlers/auth/` 与 `router/` 的测试中 grep `super_admin|RoleAdmin` × `regist` → 空。`router/auth_handler_routes_test.go:45-65` 只断言 200 + `common.SuccessCode`，请求体不含 role；`:57-65` 另有跨租户切换测试（说明租户隔离被想到了，提权没有）。`router/route_permission_guard_test.go:32` 把该路由登记为"注册入口，登录前无角色"的豁免项。

**修复方向（需决策）**：`role` 从注册 DTO 移除、或收敛为 `end_user` 单值；`tenantCode` 不接受请求体。这是全部发现中**唯一一条未认证即可拿到全系统权限**的路径。

---

### P0-2　两套权限权威对"角色能做什么"定义不一致；**播种让角色变弱**

#### (a) 差集实测

以 python 解析源码计算（非估算）：

| 量 | 实测值 | 来源 |
|:--|:--|:--|
| catalog 权限码总数 | **160**（无重复） | `internal/authz/catalog.go` `Definitions()`，位置式字面量 `{"ticket:read", "查看工单", "ticket", "read", "..."}` |
| `allPermissionCodes()` | **129** | `internal/authz/roles.go:388-440` |
| `sysadmin` 有效 DB 码集（含同义补齐 + `task:admin`） | **132** | `roles.go:24` = `allPermissionCodes()`；补齐规则见 `:314-352`、`:305-311` |
| **catalog 有、sysadmin 没有** | **28** | 差集 |
| 反向悬挂（sysadmin 有、catalog 没有） | **0** | 差集 —— 这一侧守卫有效 |

28 个缺失码：`alert:read/write`、`alerts:read/write`、`dashboard:read/admin`、`incident:admin`、`incident:force-update`、`knowledge:admin`、`notification:create/read/write`、`permission:read`、`project:delete`、`role:delete`、`service_request:approve`、`survey:read`、`system_config:read/write`、`tenant:read/write`、`ticket_category:write`、`ticket_tag:write`、`ticket_template:write`、`ticket_type:write/create/update/delete`。

全部 28 个**只出现在 `admin` 一个角色的 DB 码集里**（`roles.go:232-262`）。

#### (b) 这 28 个码确实在路由上被强制

字面量 grep `"system_config:read"` 在 `router/`、`middleware/`、`handlers/` 中 **0 命中** —— 但这是**假阴性**：`middleware/rbac_precheck_gen.go` 把 Resource 与 Action 分开存储：

```go
"/api/v1/system-configs":  {Resource: "system_config", Action: "read"},   // :382
"/api/v1/dashboard":       {Resource: "dashboard",     Action: "read"},
"/api/v1/admin/tenants":   {Resource: "tenant",        Action: "read"},
```

且不止预检映射，路由上有**内联强制**：`router/dashboard_routes.go:141-237` 共 31 处 `middleware.RequirePermission("dashboard", "read")`（`:141`、`:142`、`:143`、`:149`、`:155`、`:171`、`:190`、`:196`、`:203`、`:205`、`:207`、`:209-211`、`:221`、`:231-237`）。

判定链实测：`RequirePermission`（`rbac.go:906`，决策在 `:951`）→ `AuthorizeResource`（`:1176`）→ `loadPermissionsByMode`（`:1054`）→ DBOnly `configured` 态**只返回 DB 码集**（`:1059-1061`）→ `checkPermissionMatch`（`:1119-1145`）无通配可用。

按 `Resource:Action` 组合统计受影响路由模式：

| 权限码 | 受影响路由模式数 |
|:--|--:|
| `dashboard:read` | 31 |
| `system_config:read` | 19 |
| `system_config:write` | 14 |
| `notification:write` | 8 |
| `dashboard:admin` | 9 |
| `notification:read` | 6 |
| `tenant:read` | 5 |
| `notification:create` | 4 |
| `survey:read` | 4 |
| `tenant:write` | 4 |
| `service_request:approve` | 2 |
| `permission:read` / `role:delete` | 各 1 |
| **合计（去重）** | **89 / 799** |

受影响路由含 `/api/v1/dashboard`、`/api/v1/dashboard/overview`、`/api/v1/system-configs*`、`/api/v1/configs*`、`/api/v1/menus*`、`/api/v1/admin/tenants`、`/api/v1/tickets/assignment-rules*`、`/api/v1/tickets/automation-rules*`、`/api/v1/tickets/*/auto-assign`、`/api/v1/system/config`、`/api/v1/tickets/assign-recommendations/*`。

#### (c) 谁持有 `dashboard:read`

遍历 `BuiltinRolePermissionCodes()` 全部 **32** 个角色键（含 `allExcept` / `allPermissionCodes` 派生与同义补齐后的有效集）：

```
dashboard:read      DB 播种持有者(1): ['admin']
system_config:read  DB 播种持有者(1): ['admin']
tenant:read         DB 播种持有者(1): ['admin']
permission:read     DB 播种持有者(1): ['admin']
role:delete         DB 播种持有者(1): ['admin']
survey:read         DB 播种持有者(0): []        ← catalog.go:229-231 注明有意不授予任何角色
notification:read   DB 播种持有者(5): ['admin','agent','end_user','security','technician']
```

对照硬编码兜底 `middleware/rbac.go:70`（`super_admin` 的同款通配在 `:67`）：

```go
"sysadmin": {
    {Resource: "*", Action: "*"}, // 系统管理员拥有所有权限
},
```

逐角色比对"硬编码允许 vs DB 播种拒绝"：

| 权限码 | 硬编码 ALLOW 但 DB 播种 DENY 的角色 |
|:--|:--|
| `dashboard:read` | `agent`、`end_user`、`manager`、`security`、`super_admin`、`sysadmin` |
| `system_config:read` | `end_user`、`super_admin`、`sysadmin` |
| `notification:read` | `manager`、`super_admin`、`sysadmin` |
| `tenant:read` | `super_admin`、`sysadmin` |

**反直觉结果**：`loadPermissionsByMode`（`rbac.go:1056-1071`）在 DBOnly 下区分三态 —— `configured` 只返回 DB 码集，`unconfigured` 才回落硬编码。角色**被播种了权限行 → 判定 configured → 失去硬编码本来给它的权限**。

> 播种让角色变弱；不播种反而更强。

`BuiltinRoles()`（`seeder.go:1183-1195`）播种的 9 个词表角色中，生产 DBOnly 下能打开仪表盘的只有 `admin` 与 `super_admin`（后者靠 `rbac.go:1181` 短路，不经 DB）。`manager`、`agent`、`technician`、`end_user`、`sysadmin`、`it_admin`、`security_admin` 登录后落地页 `/dashboard` → **403**。

#### (d) 代码注释把这个前提写成了断言

`middleware/rbac.go:1004-1007`（`hasPermission`）：

```go
// 仅 super_admin 硬编码直通（与 smart_permission.go checkRolePermissionFromDB 语义统一）。
// sysadmin 不再短路：DBOnly 模式下其权限来自 role_permissions 播种数据
// （seeder.go allPermissionCodes() 全量授权），权限收回/降级因此可生效。
```

"`allPermissionCodes()` 全量授权"这个前提是**假的**：129 ≠ 160。有人基于它主动移除了 sysadmin 短路，而该假设从未被验证。

#### (e) 守卫为什么看不见

`pkg/seeder/role_permission_guard_test.go` 锁三条契约，注释 `:19-21` 明确写着第三条是为了"防双权威表漂移——本 P0 的根因类别"：

- 契约 1（`:39-42`）：词表角色码集非空 → sysadmin 有 132 个，**过**
- 契约 2（`:45-52`）：码 ⊆ `permissionDefinitions()` → 悬挂为 0，**过**
- 契约 3（`:59-77`）：与 `middleware.RolePermissions` 硬编码全等 —— `parityRoles` **只有 `admin` 和 `technician`**

守卫精准识别了缺陷类别，然后把检查范围限定在 **2/9** 个词表角色上，恰好漏掉唯一使用 `*:*` 通配的 `sysadmin`，也漏掉 `dashboard:read` 这类只授给 `admin` 的码。`:54-58` 的注释解释了缩小范围的理由（"end_user/agent/manager 等角色的 DB 码集是 2026-09-07 有意裁剪的子集……不应在本守卫中静默扩大权限"）—— 理由本身成立，但结果是**裁剪幅度从未被量化，也没人知道裁掉了登录后第一屏**。

#### (f) 手工修复会被静默撤销

`pkg/seeder/seeder.go:2068-2083`（`seedRolePermissions`）：

```go
s.client.RolePermission.Delete().Where(
    rolepermission.RoleIDEQ(r.ID),
    rolepermission.TenantIDEQ(t.ID),
    rolepermission.PermissionIDIn(managedPermissionIDs...),
    rolepermission.PermissionIDNotIn(permIDs...),
)
```

运维若在 DB 里给 `end_user` 补 `dashboard:read` 救急，**下一次初始化会把它删掉**。菜单与部门的"reconcile 尊重运维改动"性质（`seeder.go:1839-1858` 更新时确实不碰 `is_visible`/`is_enabled`；`:1010-1014` 部门不覆写名称）在**角色权限上不成立**。这一条必须写进运维手册。

#### (g) 测试覆盖

`router/`、`handlers/dashboard/`、`tests/` 下含 `dashboard` 的测试文件：**0 个**。非 admin 角色访问仪表盘这条路径无任何回归。

#### (h) 修复方向（需产品决策，不该由审查者代选）

1. 先把契约 3 的 `parityRoles` 从 2 个扩到全部词表角色，让 CI 红出**真实差集**（不改行为，只暴露）。
2. 再二选一：**扩权**（给 `sysadmin`/`manager`/`agent`/`end_user` 等补 `dashboard:read` 等码）或 **收权**（把 `dashboard:read` 从路由上摘掉，改为"认证即可"）。前者放大能力边界，后者改变产品语义。
3. 无论选哪个，都要补 `sysadmin`/`end_user` 打 `/api/v1/dashboard` 的跨角色回归测试。

---

## 2. P1 —— 初始化与种子载体

| # | 问题 | 证据 |
|:--|:--|:--|
| 1 | **`itsm-backend/config/seed/seed_data.sql`（38,991 B）死文件但带毒**：全仓引用 **0**（`*.go`/`*.yml`/`*.yaml`/`*.sh`/`Dockerfile*`/`Makefile` 全部实测无命中，唯一提及在 `.workbuddy/memory/2026-09-22.md:136`），却被整目录打进两个镜像；内含 catalog 中**不存在**的权限码 `admin:write`（L153/L334）、`change:manage`（L188）、`tenant:manage`（L342），以及自己的 admin/role INSERT。一条 `psql -f` 就能污染 RBAC | `itsm-backend/Dockerfile:69`、`itsm-backend/Dockerfile.prod:100`（均为 `COPY config/seed ./config/seed`，整目录） |
| 2 | **`demo.json` 是虚构业务数据**（8 incidents / 2 problems / 3 changes / 5 KB 文章），AGENTS.md 明禁"写入伪造客户业务数据"。它随 `config/seed` 整目录进入**生产镜像**，需要 `ITSM_SEED_CONFIG` + build tag `seed_demo` 双门禁才会生效；仓库内**受支持的入口是 dev 专用**的 `make dev-seed-demo`，其 DB 坐标硬编码为 `localhost:55432` / `itsm_user` / `itsm`（不会指向生产库）。风险是"文件在生产镜像里、只差一个环境变量"，不是"有一条受支持的生产路径" | `pkg/seeder/seed_demo.go:1-2`（build tag）；`Makefile:240-249`；`docker-compose.standalone.yml` 与两个 Dockerfile 均未设 `ITSM_SEED_CONFIG` |
| 3 | ~~`seed_demo.go:9` 引用的 `make dev-seed-demo` 在 Makefile 中不存在~~ —— **本条已撤销，是错的**。该 target 存在于 `Makefile:240-249`、已列入 `.PHONY`（`:2`）、并在 `ROADMAP.md:94` 与 `README.md:142` 有文档。此条原样采纳了子代理结论而未自行复核，见 §6 #9 | `Makefile:2`、`:240-249` |
| 4 | **`mergeSeedConfig` 是整段替换**：客户 JSON 若覆写 `roles` 段，新装环境会静默丢掉其余 17 个默认角色 | `pkg/seeder/seeder.go:469-543`（历史结论复核仍成立） |
| 5 | **`loadSeedConfig` 用 cwd 相对路径**：换 WORKDIR 就换客户拿到的数据 | `pkg/seeder/seeder.go:441-444` |
| 6 | **无首次登录强制改密**：全仓 grep `must_change_password` / `MustChangePassword` / `password_changed_at` / `force_password` = 0（`ent/schema/`、`dto/`）。且 **`itsm-backend/init_admin.sh` 重跑会重置已存在 admin 的口令并强制改角色**：`:35-43` `IF EXISTS(... username='admin') THEN UPDATE users SET password_hash=..., role='admin'`，RAISE NOTICE 明写"密码已重置"。副作用：一个已被提为 `super_admin` 的账号跑一次脚本就被**降级回 `admin`**；`:48` 硬编码 id `'admin-001'`、`:50` 硬编码 `admin@itsm.com`；`:20` 经 `psql -v admin_pw=...` 传口令，**在进程列表中可见** | `itsm-backend/init_admin.sh:11`、`:20`、`:35-43`、`:48-50`（注意：不在 `scripts/` 下，本轮已更正路径） |
| 7 | **租户开通不写 `initialization_installations` 账本**：`tenant_provisioner.go`（202 行）内 grep `installation`/`ledger`/`fencing` 仅命中 `:143` 的**注释**，两个调用方（`handlers/tenant/initialization.go:177`、`cmd/provision_tenant/main.go:55`）也无账本写入，`handlers/tenant/*.go` 中 `initialization_installations` 命中 0。改用 SystemConfig 版本标记 + 实时校验（`VerifyTenantBaseline`）。整个 DAG 在一个事务里（原子性强于 `Engine.Apply` 的逐组件事务），但**没有 fencing**，同一租户并发开通无保护。**注释与事实不一致**：`:143` 写"the initialization ledger stays the source of truth"，而租户域根本没有账本行 | `pkg/seeder/tenant_provisioner.go:22-27`、`:94-99`、`:143-150`（已知偏差，本轮复核仍在，且注释误导为新增发现） |
| 8 | **4 个角色键无对应 Role 行** → 死授权：`rd_manager`、`developer`、`qa_engineer`、`team_lead`（`security` 同类）。32 个 map 键 vs 24 个播种角色。改这些键不产生任何效果，是配置漂移陷阱 | `internal/authz/roles.go` vs `seeder.go:1183-1243` |
| 9 | **路由权限守卫只覆盖写路由**，带 15 条有理由的豁免白名单（login、webhooks、bootstrap create-admin、logout、switch-tenant、ws）；**GET 路由不在守卫范围内**，读侧只靠预检映射 | `router/route_permission_guard_test.go:27-49` |
| 10 | **`seedGroups` 仍是 create-only 逐条跳过**：组改名/改描述永不前向修复 → BPMN `candidateGroups` 会静默失效 | `pkg/seeder/seeder.go:1143-1176` |
| 11 | **`ENV` 与 `SERVER_ENV` 两条解析路径**：`configurePermissionMode` 读 `ENV`，另一处读 `SERVER_ENV` 并回落 `ENV`。今天 fail-safe，但很脆 | `internal/bootstrap/app.go:347`、`:1283-1290`、`:1565-1567` |
| 12 | **注释与代码不一致（小）**：`seeder.go:1204-1216` 注释说"config 优先"，代码在冲突时保留 **builtin** 条目 | 实测 |

---

## 3. P1 —— API 契约与业务实现（第一轮）

### 3.1 API 契约

| # | 问题 | 证据 |
|:--|:--|:--|
| 1 | **列表响应用"领域名集合键"而非 `items`**（如 `assets`、`records`），违反 AGENTS.md 强制的统一分页信封。已建棘轮基线锁存量、禁止增长。**存量正在被并发会话收敛：本轮审查开始时为 50 条，落盘时实测已降至 25 条**（`envelopeBaseline` 位于 `:61-87`，首条 `asset_dto.go\|AssetListResponse\|assets`，末条 `ticket_workflow_dto.go\|TicketCCListResponse\|records`）；相关提交 `fbf4eeb9`、`42f11fe7` | `itsm-backend/tests/contract/list_envelope_ratchet_test.go:59-87`，注释"每收敛一个都应删除条目并配前端契约更新" |
| 2 | **第二套成功标识**：`gin.H{"success": true}` 出现在 4 处，违反"业务成功不得返回 `success: true`" | `router/dashboard_routes.go:147`、`:153`、`:188`、`:269` |
| 3 | **能力开关与后端路由脱钩**：`DISABLED_API_CONTRACTS` **5 条**（change-api、knowledge-base-api、notification-preference-api、ticket-relations-api、ticket-root-cause-api）全部指向**未注册**的后端路由；且 10 个能力开关中 **7 个零消费者引用**（实测 `aiKnowledgeSearch`、`knowledgeAdvancedActions`、`changeClassification`、`advancedProblemActions`、`advancedTicketRelations`、`rootCauseWorkflowActions`、`workflowAnalytics` 均为 0；仅 `mspAllocationHistory`、`notificationTemplateManagement`、`notificationChannelManagement` 各 2 处）。文件内 `:62-72` 的 NOTE 记录了此前已清理过一批豁免，说明这条清单**在持续被修，但剩余 5 条与其对应开关仍未接线** | `itsm-frontend/src/config/product-capabilities.ts:6-39`（开关）、`:56-61`（豁免） |

### 3.2 业务实现

| # | 问题 | 证据 |
|:--|:--|:--|
| 4 | **忘记密码永不发邮件，但 API 返回成功文案**：`NewEmailService` 与两个 `SetEmailService` 方法的**非测试调用方为 0**，因此 `if s.emailService != nil` 永假；token 已创建、邮件从未发出，用户看到"如果该邮箱已注册，我们将发送密码重置链接"。典型的"未配置伪装成空成功" | `service/email_service.go:53-54`；`handlers/auth/service.go:36`、`:170-184`；`service/ticket_notification_service.go:51-53` |
| 5 | **`SimilarIncidents` 三处空成功**：nil 依赖、embed 错误、向量检索错误**都**返回 `[]map[string]any{}, nil`，正是 AGENTS.md 禁止的"把依赖不可用伪装成空结果"；且**零调用方** | `service/ai_services.go:1-30` |
| 6 | **提交后 fire-and-forget goroutine**：`go s.executeRules(...)` 在事务提交后启动，违反"禁止 fire-and-forget goroutine，必须使用事务型 durable command/outbox"的强制规则 | `handlers/incident/service.go:195-206` |
| 7 | **省略 `version` 可绕过乐观锁**：先取 `current.Version` 再 `if req.Version > 0` 覆盖 —— 客户端不传 version 时退化为 read-then-unconditional-write，正是规范点名禁止的形态 | `service/ticket_service.go:1098-1112` |
| 8 | **硬编码 `tenant_id = 1`**：无 API key 的 dev 分支与有效 key 分支**都**写 `c.Set("tenant_id", 1)` + `TenantContext{TenantID: 1}`，违反"缺少租户上下文必须 fail closed，禁止回退到 tenant 1" | `cmd/cmdb/main.go:110-130` |

### 3.3 UI 交互

| # | 问题 | 证据 |
|:--|:--|:--|
| 9 | **风险过滤器是 no-op**：`type: undefined, // risk在API中可能对应type，这里先不处理` —— 用户选择风险等级后结果不变，且注释暴露了未解决的契约猜测 | `itsm-frontend/src/components/change/ChangeList.tsx:70-105` |
| 10 | **`lint:antd` 只覆盖 9 类禁用 API 中的 3 类**：`Space direction=`、`<Tabs.TabPane`、`Form.(Input\|TextArea\|Select\|DatePicker\|Radio\|Checkbox)`。`visible=`/`destroyOnClose`/`bodyStyle`/`overlay`/`dropdownRender`/`onDropdownVisibleChange` 六类**无门禁** | `itsm-frontend/tools/check-antd-legacy.sh:18-22` |

### 3.4 门禁与 CI

| # | 问题 | 证据 |
|:--|:--|:--|
| 11 | **`ga-gate` 在 "Start core stack" 处死掉** → 仓库中**唯一的全栈 E2E 从未真正运行**。它的"绿"是空转的绿 | 实测 CI 日志 |
| 12 | **CI 当前为红，但与本次审查的代码无关**：本地有 **20 个未推送提交**，远端跑的是旧代码。审查结论一律以本地实测为准 | `git log origin/main..HEAD` |
| 13 | 正面结论：`cd itsm-backend && go test ./...` 实测 **exit 0 / 83 ok / 0 FAIL / 198 no-test-files**（无并发噪声） | 实测 |

---

## 4. 前端"长文案假实现"（第二轮）

### 4.1 可达的假实现（生产路由上真实存在）

| # | 位置 | 事实 |
|:--|:--|:--|
| 14 | `itsm-frontend/src/components/ticket/TicketAdvancedSearch.tsx`（可达于 `src/app/(main)/tickets/page.tsx:10,12,256`） | `:125-133` 预置模板"分配给我且未完成的工单"里写 `assigneeId: 1, // 当前用户ID` —— **硬编码 1 冒充当前用户**；`:174` `savedSearches` 是 useState；`:327`、`:335-349` `saveSearch` 只 `setSavedSearches` 然后 `message.success('搜索条件已保存')`；`:403`、`:442-453` 同类。**整个文件零 `@/lib/api` 导入** —— 刷新即丢，且"分配给我"实际查的是用户 1 |
| 15 | `itsm-frontend/src/app/agent-ops-demo/page.tsx`（216 行，102 条长中文文案） | `:28-70` 硬编码 `evidence`（"checkout-api P95 延迟超过 2.5s"、"错误率 18.7%"、"匹配 INC-2025-0817，相似度 92%"）与 `guardrails`（"SRE Copilot · on behalf of 刘洋"、"风险等级 L3 · 必须人工批准"、"单次授权 · 10 分钟"）；整个"批准 → 执行"流程是 `useState<'pending'\|'approved'\|'executed'>`；"执行成功 连接池已恢复"是一个 `useMemo` 字符串。**不在菜单，但是存活的生产路由** —— 文案的专业度会让访客误判产品能力 |
| 16 | `itsm-frontend/src/components/change/ChangeList.tsx:70-105` | 见 §3.3 #9 —— 真实组件里的**部分假**：列表数据真，风险过滤假 |

### 4.2 孤儿组件中的假实现（55 个组件 / 17,987 行不可达代码中已取证的 7 处）

| # | 位置 | 事实 |
|:--|:--|:--|
| 17 | `src/app/(main)/admin/system-config/EnhancedSystemConfig.tsx`（1014 行，孤儿） | 系统状态部分是真的，但 `:202`、`:224-277` `// 模拟日志数据` + `setLogs` 硬编码行（admin 登录 192.168.1.100 2026-07-04）；`:742-800` `// 模拟备份数据` + `setBackups`（`full_backup_20260704` 2.5 GB completed）；`:995`、`:1000` **"开始备份"按钮无 onClick** |
| 18 | `src/components/business/hooks/useSatisfactionData.ts`（33 行，**零消费者**） | `// Simulate data fetching` + `setTimeout(1000)` → 硬编码 `{overall:4.2, responseTime:4.0, resolutionQuality:4.3, communication:4.1, totalResponses:1250}` |
| 19 | `src/components/business/NotificationCenter.tsx:273-281`（1057 行，孤儿） | `handleTestChannel` = 1 秒 `setTimeout` 后**永远** `channelTestSuccess`；catch 分支不可达 |
| 20 | `src/components/business/TicketCategoryExport.tsx:125-175`、`:300-325`（458 行，孤儿） | `// 模拟导出进度` `setInterval` 每 200 ms +10% 到 90%；真实数据来自 `ticketCategoryService.getCategoryTree()`。**注释撒谎而代码没有**：`exportToExcel` 注释称"文件扩展名为.xlsx"，代码实际产出 `.csv` 并 `message.info('Excel导出功能需要安装xlsx库，当前使用CSV格式')` |
| 21 | `src/components/business/TicketTemplate.tsx:95-120`、`:220-240`（635 行，孤儿） | 全套 mock CRUD：`// 模拟API调用` 1 秒延迟 + `mockTemplates`（"网络故障标准模板"）；复制/删除/更新/创建**全部只弹成功提示** |
| 22 | `src/components/business/TicketMultiLevelApproval.tsx:299-305`（767 行，孤儿） | `// 注意：删除工作流API尚未实现` + `antMessage.success('工作流已删除（模拟）')` |
| 23 | `src/components/business/KnowledgeIntegration.tsx:122`、`:130`、`:167-175`、`:215-231`（637 行，孤儿） | 搜索与推荐是真的，但"关联文章"是 `// 模拟关联API调用` + 本地对象（`associatedBy: '当前用户'`）+ 成功提示 |

### 4.3 门禁为什么看不见这些

| # | 盲区 | 机制 |
|:--|:--|:--|
| 24 | **C.7.1 看不见 55 个孤儿组件** | `scripts/docs-gate/check-scope-creep.sh:24-58` 的扫描范围是 `src/app/(main)/<module>` 下的页面**加上它们引用的 `@/components/*`**。孤儿组件不被任何页面引用 → 天然在范围外。粒度是模块级，不是文件级 |
| 25 | **流程发现：修复投入被花在死代码上** | 上述 7 处假实现全部位于孤儿文件。任何"修好这个假实现"的工时都是纯浪费 —— 正确动作是先删或先接线，再谈修复。**审查顺序应当是"可达性 → 真伪"，不是"真伪 → 可达性"** |

### 4.4 C.7.1 自身的历史教训（值得保留在案）

`check-scope-creep.sh:24-34` 记录了第一版产出 10 个 shell 模块 / 15 个页面，**全部是假阳性**（漏了 `.tsx` 扩展名；API 信号只认 `lib/api`，不认 `lib/services/*-service.ts`；未排除兼容跳转页）。`:33` 的教训写得很准：

> 静态检测结论在下判断前必须逐文件复核，否则守卫会逼人删掉正确的东西

本轮我在这条上犯了同类错误（见 §6）。

---

## 5. 确实做对的部分（与问题同权重列出）

**权限与初始化**

- **`internal/authz/catalog.go` 是真正的单一事实来源**：包文档 `:1-50` 记录了权限码曾散落 4 处、此包治本；`Definitions()` 是改码唯一入口，附三条规则（独立资源面 / ≥1 路由引用 / ≥1 角色绑定，否则死码）；配 codegen（`cmd/authz-gen`、`cmd/authz-codegen`）+ 两个守卫（`role_permission_guard_test.go`、`router/permission_code_catalog_guard_test.go`）。**悬挂码实测为 0。**
- **生产模式确实是 DBOnly**：`internal/bootstrap/app.go:1283-1290` 只对 development/dev/test/local 放宽到 Fallback，其余 DBOnly；结构体默认值也是 DBOnly（`middleware/rbac.go:465`，注释"企业级交付：仅使用数据库权限，支持多租户差异化"）。L4 硬编码兜底在 DBOnly 下被正确关闭（`smart_permission.go:145-150`）。
- **默认口令守卫在生产真的会跑**：`internal/bootstrap/default_credentials_guard.go:118-145` 查 `ENV` 后回落 `DEPLOYMENT_MODE ∈ {production,prod,private,saas,saas_msp}`；`docker-compose.prod.yml:110,185,326` 与 `.env.prod:28` 均设 `private` → **命中**。
- **菜单基线干净**：`pkg/menubaseline/baseline.go`（157 行）**97 条**菜单，48 个非空权限码**全部存在于 catalog**（python 差集 = 0），仅 `/dashboard` 有意留空（任何登录用户可见）。`seeder.go:1839-1858` 更新时**不碰 `is_visible`/`is_enabled`** → 运维的隐藏选择能活过重装。
- **菜单不是授权点**（符合"隐藏菜单不等于授权"）：路由各自带 `RequirePermission`/预检；菜单过滤（`service/menu_service.go:296-331`）读同一份 DB 权限并遵守 DBOnly。
- **`system-baseline-<tenant>` 账号确实不可登录**：密码哈希 `"!"`（永不匹配 bcrypt）+ `Active=false`（`pkg/seeder/baseline_creator.go:19`、`:42-44`）；且被排除在租户删除的用户检查外（`service/tenant_service.go:298`）。
- **默认路径无虚构业务数据**：`config/seed/default.json` 四类业务数组为 0；`getProductDefaultConfig()` 强制清空（`seeder.go:460-467`）；`pkg/seeder/business_records.go:28-34` 空集直接 return，文件头 `:1-13` 明确写了这条设计约束。**符合 AGENTS.md。**
- **平台级初始化的可靠性是真的**：`internal/initialization` `Engine` 逐组件 lease + fencing token + 心跳，apply/verify/账本 attempt 在**同一物理事务**内，失败向上传播 fail-closed（`engine.go:141-223`；`pkg/seeder/initialization_adapter.go:257-302`）。digest 覆盖 JSON manifest + 码内权限/菜单/角色/组/授权定义 + **两套 BPMN embed**（`manifest_digest.go:16-128`）。
- **逐条目 reconcile 已成主流**：departments/teams/ticket-types/SLA/menus 均已从"有任意行就整段跳过"改为逐条目；本轮**未发现**残留的整段跳过（`seedGroups` 除外，见 §2 #10）。
- **租户唯一键已收敛**：`migrations/20260923_tenant_scope_baseline_unique_keys.sql` 修了 `ticket_categories.code`、`tags.code`、`process_deployments.deployment_id`，带重复预检。`marketplace_items.name` 有意保持全局（`initialization_adapter.go:200-215`）。
- **新租户不复制 default 租户**：`tenant_provisioner.go:22-26` 明确走审计过的组件 DAG 而非克隆线上数据。
- **运维审计入口存在**：`initialize -action audit-tenants`（`cmd/initialize/main.go:135-146`、`pkg/seeder/baseline_audit.go`，只读且有写拦截测试锁住）+ 租户初始化状态 API（`handlers/tenant/initialization.go`）。

**门禁**

- Gate C.6 / C.7 本地实测全绿、0 FAIL / 0 WAIVED；C.6.4 棘轮未涨。
- `list_envelope_ratchet_test.go` 的棘轮思路正确：**先锁存量、禁止增长、每收敛一个删一条**，而不是一次性改 50 处。
- `route_permission_guard_test.go` 的 15 条豁免**逐条带理由**，不是空白名单。

---

## 6. 审查者自己的差点搞错（与发现同等重要）

按 `feedback-measured-review-over-documents` 的要求，本轮与前两轮的近似失误一并记录：

| # | 差点得出的错误结论 | 实际情况 | 教训 |
|:--|:--|:--|:--|
| 1 | **`catalog=0`** → "sysadmin 拥有全部权限，P0-2 不成立" | 用 `Code:\s*"..."` 匹配 `catalog.go` 返回 0；该文件用**位置式**结构体字面量 `{"ticket:read", "查看工单", ...}`。改正后得到 160 | **0 结果是正则错了的信号，不是事实**。任何"空集"结论都必须先证明提取器能提取到东西 |
| 2 | **"28 个码无路由引用 → 纯死码，无运行时影响"** | zsh 下未加引号的 `--include=*.go` 直接 `no matches found`，20 个码全部输出 `0 route refs` | 命令报错和"结果为 0"必须区分；stderr 不能吞 |
| 3 | **同上，即使修好引号仍是 0 命中** | `rbac_precheck_gen.go` 把 `Resource` 与 `Action` **分开存**，字面量 `"system_config:read"` 天然不存在。逐行读生成文件才发现 89 个路由受影响 | **这条差点把真 P0 降级成死码**。搜索关键词要匹配数据的存储形态，不是匹配概念的书写形态 |
| 4 | `visible=` 单行正则返回 0 → 差点判定子代理"8 处 antd 违规"为假 | 改用普通 `grep -rn '\bvisible='` 找到 9 处；再查组件归属，**9 处全是项目自有组件**（`WorkflowNewModal`、`CreateServiceModal`、`ApprovalChainModal`、`SLAViolationDetailModal`、`TicketTypeFormModal`），AGENTS.md 明确允许 → 真实 antd 违规数为 **0** | 子代理的**分类**错了，但我的第一版正则也错了。两边都要复核 |
| 5 | 把 `app/(main)/dashboard/page.tsx`（669 行、92 条长中文、无 `@/lib/api` 导入）列为头号假实现嫌疑 | 它导入**路由本地 hook** `./hooks/useDashboardData`，后者真的调 `DashboardAPI.getOverview()` + `ticketService.getTicketStats()` | 启发式漏了 AGENTS.md 推荐的"路由本地 hook"形态。同一类假阳性也命中了薄包装页 |
| 6 | `grep -c 'Permission:\s*"'` 在 `menubaseline/baseline.go` 返回 0 → 差点断言"菜单不带权限绑定" | 字段名是 **`PermissionCode:`**。改正后：97 条菜单、1 条有意留空、0 条缺字段 | 同 #1 |
| 7 | 第一次 `go test ./...` 判定失败 | exit 1 来自我尾部 `grep -cE '^FAIL'` 匹配 0 行；且过滤器 `grep -vE '^(ok\|---\|\?)'` 会**吃掉 `--- FAIL` 行**。重跑并干净计数：exit 0 / 83 ok / 0 FAIL / 198 no-test-files | 不要用管道尾部的 grep 退出码代表被测命令的结果 |
| 8 | 报告初稿写"50 处列表信封违规，收敛尚未开始" | 落盘前逐条复核引用时发现：并发会话已把它收敛到 **25 条**，且 `envelopeBaseline` 从 `:40-117` 移到了 `:61-87`（提交 `fbf4eeb9`、`42f11fe7`）。同一轮复核还纠正了 `rbac.go` 通配的行号（`:69` → `:70`） | **共享工作区里，引用会在你写报告的过程中失效。** 落盘前必须对每条 `file:line` 重跑一次核对；这也正是"记忆/结论会过期，用前先验"的同一纪律 |
| 9 | 报告初稿写"`make dev-seed-demo` 在 Makefile 中不存在（0 命中）" | **完全错。** 该 target 在 `Makefile:240-249`、已列入 `.PHONY`（`:2`）、`ROADMAP.md:94` 标为已落地、`README.md:142` 有用法。这条是**原样采纳子代理结论而未自行复核**的结果 | 子代理的"某物不存在"属于**否定性断言**，比肯定性断言更容易错（一次 grep 的路径、cwd 或 glob 写错就会得到 0 命中）。凡是"不存在/0 引用/死文件"类结论，必须自己重跑一次才写进报告。同轮还纠正了两处路径错误：`init_admin.sh` 在 `itsm-backend/` 而非 `scripts/`；Dockerfile 引用需带 `itsm-backend/` 前缀 |

**本轮落盘前的引用复核结果**：后端 16 处、前端 15 处 `file:line` 引用**逐条重跑核对**，除上表 #8 的三项外全部仍然成立（`dashboard_routes.go:147/153/188/269`、`ticket_service.go:1107`、`incident/service.go:203`、`cmd/cmdb/main.go:118/124`、`smart_permission.go:117-119`、`router.go:310`、`auth_dto.go:95`、`auth/service.go:151/179`、`seeder.go:2079`、`role_permission_guard_test.go:60`、`app.go:1286`、`seed_data.sql:153/334`、`check-antd-legacy.sh:18`、`ChangeList.tsx:81`、`TicketAdvancedSearch.tsx:131/348`、`tickets/page.tsx:10/12/256`、`agent-ops-demo/page.tsx:33/36/86`、`EnhancedSystemConfig.tsx:252`、`useSatisfactionData.ts:18`、`NotificationCenter.tsx:276`、`TicketCategoryExport.tsx:134`、`TicketTemplate.tsx:104/228`、`TicketMultiLevelApproval.tsx:302`、`KnowledgeIntegration.tsx:217`、`check-scope-creep.sh:33`）。"7/10 能力开关零消费者"亦重跑确认为 7。

**对子代理结论的修正**（本轮第 5 个子代理）：方向正确、两处数字不准 —— 缺失码是 **28** 不是 31；角色键 **32** 个但播种角色 **24** 个。其风险排序第 1 条（sysadmin 缺码 → DBOnly 403）结论成立，但给出的机制（内联 `RequirePermission`）不完整，真实机制是**预检映射的 Resource/Action 组合 + 内联强制两者都有**，且影响面远超 sysadmin（`dashboard:read` 只授给 `admin`）。

---

## 7. 静态无法确定的部分（不猜）

- **运行时是否真的 403** —— 未起服务、未连库，§1 P0-2 全部是源码级推导。**需要在演练库上用一个 `end_user` 与一个 `sysadmin` 账号实测确认**（按 `initialization-drill-database-safety` 的既定演练配方，禁碰 `itsm`/`itsm_prod`）。
- `admin` 角色持有 `tenant:write`（`roles.go:243`，catalog `:236-237` 定义为"创建、更新、停用租户"）**是否在 handler 层被限定为平台级** —— 未验证。这是潜在的租户越权面。
- 预检映射（799 条）是否覆盖**每一个**已注册 GET 路由 —— 未做全量比对。
- 任何环境的真实 DB 状态；`PermissionConfig.Mode` 是否会在启动后被改写。
- `migrations/20260923` 之外的唯一约束全量清单。
- 既有记忆中两个未关闭的 P2 是否仍存在：`prepareProcessApprovalDecisionIndexMigration` 每次启动删掉 Ent 自己的非唯一索引并打假日志"legacy unique index dropped"；`audit-tenants` 不区分软删除租户（e2e 残留污染报告）。**本轮未复核。**

---

## 8. 建议处理顺序

| 批次 | 内容 | 为什么是这个顺序 |
|:--|:--|:--|
| **第 1 批** | P0-1 注册提权（`role` 移出注册 DTO 或收敛为 `end_user`；`tenantCode` 不接受请求体）+ 补跨角色提权回归测试 | 唯一一条**未认证**即可拿到全系统权限的路径；改动小、无产品语义争议 |
| **第 2 批** | P0-2 第一步：守卫契约 3 的 `parityRoles` 扩到全部词表角色 | **只暴露不改行为**，让 CI 红出真实差集，为后续决策提供数据 |
| **第 3 批** | P0-2 第二步：产品决策"扩权 vs 收权"，然后补 `sysadmin`/`end_user` 打 `/api/v1/dashboard` 的回归 | 需要产品拍板，不该由审查者代选 |
| **第 4 批** | 删 `itsm-backend/config/seed/seed_data.sql`（死文件、含 catalog 外的码）；把 `demo.json` 移出生产镜像的 `COPY config/seed`（改为 dev 镜像专用或 `.dockerignore`）；修 `itsm-backend/init_admin.sh` 的口令重置 + 角色强制降级 + `psql -v` 口令暴露 | 死文件带毒，删除是纯收益；init_admin.sh 三条都是运维误操作即可触发的 |
| **第 5 批** | 前端：先删/接线 55 个孤儿组件（17,987 行），**再**谈修其中的假实现 | §4.3 #25：否则工时花在死代码上 |
| **第 6 批** | 门禁补口：`check-antd-legacy.sh` 补齐 6 类；修 `ga-gate` 的 "Start core stack"；C.7.1 增加文件级孤儿检测 | 让"绿"重新等于"测过" |
| **第 7 批** | §3 的业务实现项（邮件未接线、空成功、fire-and-forget、乐观锁可绕过、硬编码 tenant 1） | 各自独立，可并行；其中 #6 #7 #8 违反 AGENTS.md 强制规则 |
| **第 8 批** | §3.1 的列表信封收敛（**落盘时剩 25 条**，按棘轮逐条删；已在推进中） | 量大但机械，且已有棘轮保护不会恶化 |

---

## 9. 设计判断：初始化方式 / 默认菜单 / 角色 / 权限"是否合理"

**骨架合理，权限层的两个权威没对齐，而这正好是最不该出错的地方。**

初始化架构在这一档产品里属偏上水平：码内单一事实来源 + CI 守卫、逐组件事务账本 + fencing、尊重运维改动的前向修复、默认不造假业务数据、无默认口令、租户开通不复制线上数据。这些都是需要刻意设计才有的性质，不是默认能得到的。

不合理的地方集中在**权限语义有两份定义**：

- `middleware.RolePermissions`（硬编码，13 角色，`sysadmin` = `*:*` 通配）
- `authz.BuiltinRolePermissionCodes()`（DB 播种，32 角色键 / 24 播种角色，`sysadmin` = 129 码枚举）

同一个"`sysadmin`"在两套里意思不同。守卫识别了这个风险类别（注释原话"防双权威表漂移——本 P0 的根因类别"），但只对齐了 2 个角色。叠加 DBOnly 的 `configured`/`unconfigured` 三态语义，就产生了"**播种让角色变弱**"这个反直觉结果 —— 而这类缺陷在 dev（Fallback 模式）下**永远测不出来**，只在生产暴露。

这与既有记忆中"只在全新安装暴露的迁移缺陷类"是**同一个家族**：不是逻辑错，是**环境差异把正确性吃掉了**。对这个家族，唯一有效的防御是让守卫在两种模式下都跑，而不是靠人在生产上发现。

菜单设计是这轮里最干净的部分：97 条码内定义、权限码 100% 落在 catalog、reconcile 不覆写运维选择、且菜单明确不是授权点。**这一块不需要动。**

---

## 10. 附：本轮实测命令口径

便于复核与复跑：

```bash
cd itsm-backend

# 权限码差集（catalog vs sysadmin 有效集）—— 解析源码，不连库
#   catalog: 位置式字面量  ^\s*\{"([a-z0-9_]+:[a-z0-9_\-]+)"
#   sysadmin: allPermissionCodes() + synonymParity 派生 + task:admin

# 受影响路由模式（Resource/Action 组合）
grep -n 'Resource: "system_config"' middleware/rbac_precheck_gen.go

# 路由内联强制
grep -n 'RequirePermission("dashboard", "read")' router/dashboard_routes.go

# 死文件引用
grep -rn 'seed_data.sql' --include='*.go' --include='*.yml' --include='*.yaml' \
  --include='*.sh' --include='Dockerfile*' --include='Makefile' .

# 提权测试覆盖
grep -rn 'super_admin\|RoleAdmin' handlers/auth/ router/ --include='*_test.go'

# 后端全量测试（干净计数，不要用管道尾部 grep 的退出码）
go test ./... ; echo "GO_TEST_EXIT=$?"
```

前端：

```bash
cd itsm-frontend
npm run lint:antd          # 只覆盖 3 类
npm run type-check
git diff --unified=0 -- '*.tsx' '*.jsx' | rg '^\+.*<Space\b[^>]*\bdirection\s*='
```

---

*本报告是 2026-10-03 的审查快照。按 `docs/review/README.md` 的治理规则，它不是当前架构或发布事实源；进行新开发前请以当日源码、测试与运行时证据复核。*
