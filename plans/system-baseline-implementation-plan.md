# CloudMesh 系统基线改造 · 阶段 3–6 实施计划

> 状态：**待确认，未执行**。本文档不含任何已落地的代码或迁移。
> 前置：`docs/engineering/system-bootstrap-analysis.md`（阶段 1 现状审查）、`docs/engineering/system-baseline-design.md`（阶段 2 目标架构）。
> 纪律：按任务约束 9，**先出实施计划，确认后才动数据库结构与数据迁移**。本文档落地后立即停止，不做 Ent Schema、不生成代码、不写迁移、不跑回填。

---

## 0. 结论先行

现有仓库**不是从零开始**。`internal/initialization/engine.go` 已经实现了本任务要求的大部分工程能力：

| 任务要求 | 现有实现 | 位置 |
|:--|:--|:--|
| 幂等 | 组件级 `(scope,component)` 唯一行 + checksum 比对 | `sql_store.go:112-160` |
| 事务 | `BeginComponent → Apply → Complete → Commit`，失败 `Rollback` | `engine.go:192-219` |
| 并发锁 | `ON CONFLICT ... fencing_token+1 WHERE lease_expires_at < NOW()` | `sql_store.go:124-141` |
| 依赖排序 | Kahn 拓扑排序 + 环检测 + 未知依赖报错 | `engine.go:260-300` |
| 错误记录 | `initialization_runs` / `initialization_component_attempts` | `migrations.go:470-506` |
| 失败恢复 | `FailComponent` fenced 写 + `retry` 子命令 | `sql_store.go:289-313` |

因此 **第四阶段不需新建引擎**，只需在 `Initializer.Plan` 的五态语义、`Apply` 的字段级合并、以及 CLI 表现层上扩展。这条判断直接决定了实施量：**新增 6 张表 + 扩展 3 个接口 + 1 个 CLI 命令组**，而不是重写初始化框架。

真正缺失且必须补的四项：

1. **字段级所有权**——`initialization_managed_records` 表已存在（`migrations.go:508-525`，含 `ownership_mode`/`managed_fields`/`local_modified_at`/`stable_key_aliases`），但全仓 **零消费方**（`grep` 仅命中建表语句、`tenant_guard.go` 白名单、`migrator_test` 表名断言）。这是与 `dry_run` 同类的"字段即声明"假闭环。
2. **五态协调语义**——现有 `Action{Type,SourceKey,Summary}`（`engine.go:36-40`）只有 `reconcile` 一种 Type，无 `create/update/skip/deprecate` 区分，也无 diff 载体。
3. **三方合并**——播种普遍是 `Exist → Create` 或无条件 `Set*`，没有"上次系统值 vs 当前值"比较。
4. **审计 diff**——`runs`/`attempts` 有版本、执行人、fencing、结果摘要，但无 before/after 字段值、无 reason、无 dry-run 标记。

### 0.1 审查补强：一处已记录但风险未被点明的事实

阶段 1 文档 §3 记录了「`tenant.bootstrap.version.<id>` 也是 system_config 行」，但没有推出它的边界含义。两条记账实际是分裂的：

| 对象 | 版本记账位置 | 并发保护 |
|:--|:--|:--|
| 平台基线（`scope=platform,0`） | `initialization_installations` 账本（`sql_store.go`） | 租约 + 心跳 + fencing |
| 业务租户（`scope=tenant,id`） | `system_configs` 的 `tenant.bootstrap.version.<id>` 行（`tenant_provisioner.go:146-166`） | **无租约**，只有单事务 |

`ProvisionTenant` 代码注释明确写着 "it does not write to the initialization_installations ledger (that ledger covers platform bootstrap)"（`tenant_provisioner.go:144-145`）。后果：

- **并发风险**：两个 `ProvisionTenant` 对同一租户并发执行时无 fencing 互斥，靠 PostgreSQL 行锁与唯一索引撞库兜底。撞库的表现是唯一约束报错，而非 `ErrLeaseHeld` 这样的可辨识信号——运维看到的是"开租户失败"，无从判断是并发冲突。
- **状态不可查**：`cloudmesh system status` 用 `store.Status` 只查 `scope=platform,0`，业务租户的基线状态**在统一视图里看不到**，只能逐租户调 `VerifyTenantBaseline`。
- **两个版本源会漂移**：`system_configs` 行可被用户改（它是普通配置表一行），而 `initialization_installations.installed_version` 不可被业务接口改。

**阶段 3/4 必须统一**：`BootstrapState` + `SystemBaselineRecord` 的定位就是把两条记账收敛到一张可按 scope 查询的表，同时**保留** `initialization_installations` 的租约语义（租约 SQL 不动）。租户开通补上 `AcquireLease(scope=tenant,id,component)`，让并发冲突返回 `ErrLeaseHeld` 而非唯一约束报错。

---

## 1. 阶段 3：Ent Schema 与迁移（待审批）

### 1.1 六个 Schema 的落地顺序

严格按依赖顺序，每一步单独可编译、单独可回滚：

| 序 | Schema | 表名 | 依赖 | 说明 |
|:--:|:--|:--|:--|:--|
| 1 | `SystemBaseline` | `system_baselines` | 无 | 声明式清单版本头，`(scope_type, scope_id, version)` 唯一 |
| 2 | `SystemBaselineItem` | `system_baseline_items` | 1 | 条目与稳定键，`(baseline_id, resource_type, stable_key)` 唯一 |
| 3 | `SystemBaselineRecord` | `system_baseline_records` | 1 | 每 scope/component 已应用状态与 `last_applied_hash` |
| 4 | `BootstrapState` | `bootstrap_states` | 无 | 阶段快照投影，**不替代** `initialization_installations` 的租约 |
| 5 | `ConfigOverride` | `config_overrides` | 无 | 租户/用户覆盖值，含 `secret_ref` |
| 6 | `ConfigAuditEvent` | `config_audit_events` | 无 | 追加写审计，`run_id` 关联既有 `initialization_runs` |

### 1.2 关键设计约束（避免重犯已知缺陷）

- **唯一索引写 `Indexes()` 而非 `Index()`**。仓库已有三处拼错（`permission_definition.go:35`、`role_permission.go:32`、`endpoint_acl.go:63`），Ent 静默忽略、靠 SQL 迁移兜底。新表不允许再犯；同时把这三处拼写修正列入同批（`role_permissions` 已有迁移 `20261027_prod_schema_guards.sql` 兜底，风险低）。
- **菜单表继续不声明组合唯一索引**。`ent/schema/menu.go:64-68` 有明确记录：entc v0.14.6 对 Menu 自关联 edge + 组合唯一索引会组合爆炸（CPU 1200% 不终止，两次复现）。`(tenant_id, path)` 由 `migrations/20261011_add_menus_tenant_path_unique.sql` 承担。
- **不复制执行历史**。`SystemBaselineRecord` 只存"期望状态 + 所有权"，不存 lease/attempt；执行历史继续由 `initialization_*` 三表负责，二者以 `run_id` 关联。
- **`BootstrapState` 不接管租约**。租约的 fencing 语义在 `sql_store.go` 的 SQL 条件更新里，搬到 Ent 会丢掉 `WHERE lease_expires_at > clock_timestamp()` 的原子性。`BootstrapState` 只做可查询投影。

### 1.3 迁移计划（人工审批点 ①）

迁移文件命名遵循仓库既有约定 `<YYYYMMDD>_<topic>.sql` + `_down.sql`：

| 文件 | 内容 | 回滚 |
|:--|:--|:--|
| `20261122_create_system_baseline.sql` | 6 张表 DDL + 索引 + 注释 | `_down.sql` DROP 6 表 |
| `20261122_fix_index_declarations.sql` | 三处 `Index()` → `Indexes()` 修正（**只改 schema 声明，索引本体由既有迁移保证**） | 无需回滚（声明层修正） |
| `20261122_backfill_baseline_v1.sql` | 把现有 `menubaseline` / `authz` 清单序列化为 `system_baselines` + `system_baseline_items` 的 `v1` 记录 | `_down.sql` 按 `version='v1'` 删除 |

**回填策略**：`backfill_baseline_v1` 只读现有代码清单写表，**不触碰任何业务行**（菜单/权限/角色表零写入）。该迁移可重复执行（`ON CONFLICT DO NOTHING`），且在 `dry_run` 下不执行。

**审批要求**：三份迁移必须由数据库负责人 + 发布负责人签字后才进入执行。签字前不进 `AutoMigrate` 路径。

### 1.4 空卷首装门禁前置检查

阶段 3 完成必须通过：

- `make entgen-check`（CI `backend-ci.yml:357-392`，只在 `ent/schema/**` 变更时跑）
- `fresh-install-gate.yml`（空卷首装，`MIN_MIGRATIONS=50`）
- `go test ./migration/ -count=1`（含 `fresh_install_reference_test.go` 的迁移版本引用锁）

---

## 2. 阶段 4：Reconciliation Engine

### 2.1 接口扩展（不新建引擎）

扩展现有 `Initializer` 契约，保持 `Plan/Apply/Verify` 三方法不变（四条通道已依赖它）：

```go
// internal/initialization/engine.go 扩展现有结构，不新增 interface
type Action struct {
    Type      string  // create|update|skip|deprecate（替代单一 "reconcile"）
    SourceKey string
    Resource  string  // menu|permission|role_grant|config|...
    Summary   string
    Before    map[string]any  // 新增：字段级旧值（nil 表示新建）
    After     map[string]any  // 新增：字段级目标值
    Conflict  string          // 新增：非空即冲突，组件 fail-closed
}
```

**向后兼容**：`Before`/`After`/`Conflict` 为零值时行为与现在完全一致，旧组件无需改。

### 2.2 三方合并算法

```
for each item in registry:
    cur  = read current row by stable_key
    want = registry.value
    last = record.last_applied_values[stable_key]   # 系统上次写入的值

    if cur == nil:                      -> create
    else for each field in item.managed_fields:
        if cur[field] == last[field]:   -> 可安全 update 到 want[field]
        elif want[field] != last[field]:-> conflict（系统要改，但用户也改过）
        else:                           -> skip（用户改过，系统无新意）
    non-managed fields                  -> 永不写入，永不删除
```

落点：`pkg/seeder` 内新增 `reconcile.go`，由各组件 `Plan` 调用，不改 7 个组件的 `apply` 主体。

### 2.3 CLI（第四阶段交付）

新建 `cmd/cloudmesh/`（**唯一新增可执行入口**），复用 `cmd/initialize` 的 `ProductionAssembly`：

| 命令 | 行为 | 数据库写入 |
|:--|:--|:--|
| `cloudmesh bootstrap` | 首装或版本推进；委托 `assembly.Engine.Apply` | 是（受审批标记约束） |
| `cloudmesh system reconcile --dry-run` | `Engine.Plan` + Drift Detector，输出五态 JSON | **零** |
| `cloudmesh system reconcile --apply` | `Engine.Apply`，按 DAG/事务/租约 | 是 |
| `cloudmesh system status` | `store.Status` + `BootstrapState` | 零 |
| `cloudmesh system drift` | 只读漂移报告 | 零 |

**dry-run 零写入的强制手段**（这条最容易假实现）：`--dry-run` 分支**不调用 `BeginRun`**。现有 `Engine.Apply` 第一步就是 `store.BeginRun`，写 `initialization_runs`。因此 dry-run 必须走独立的 `Engine.Plan` 路径，并加测试断言 `initialization_runs` 行数不变、`config_audit_events` 仅有 `dry_run=true` 行。

### 2.4 禁止物理删除的强制手段

- `deprecate` 只做三件事：`deprecated=true`、`deprecated_since=NOW()`、进审批队列。**无 DELETE 语句**。
- 加守卫测试：全仓 `grep -r "Delete()" pkg/seeder/ internal/initialization/` 命中即失败（豁免显式退役清单 `applyRetiredRolePermissions`，定义在 `seeder.go:2247`，其行为由 `role_permission_guard_test.go` 独立测试覆盖）。

---

## 3. 阶段 5：安全约束落地对照

| # | 约束 | 现状 | 需补动作 |
|:--:|:--|:--|:--|
| 1 | 首管密码不入仓库 | ✅ 已合规。`ADMIN_PASSWORD` 环境变量 + 弱口令闸门（`seeder.go:1019-1068`） | 无 |
| 2 | Token 一次性/有效期 | 部分。`bootstrap_token.go` 有生成/消费分离 | 补 TTL 强制校验 + 重放测试；`BOOTSTRAP_TOKEN_ENABLED=1` 时 `verifyIdentityRBAC` 跳过 admin 校验（`initialization_adapter.go:364`），需补替代断言 |
| 3 | 权限定义与授权分离 | ✅ 已合规。`internal/authz` 只定义码，授权只写 `role_permissions` | 无 |
| 4 | 新增权限不自动授权 | ❌ **不合规**。`seedRolePermissions` 对 `authz.BuiltinRolePermissionCodes()` 全量匹配角色并自动写入（`seeder.go:2209-2236`） | 改为：新增码只进"待授权清单"，需显式发布审批 |
| 5 | 自定义角色不被覆盖 | 部分。`is_system` 字段存在，但 `seedRoles` 对匹配 code 的角色无条件 `SetName/SetDescription/SetIsSystem`（`seeder.go:1331-1339`） | 按 `is_system` 分流：`is_system=false` 的角色不参与 upsert |
| 6 | Secret 不进普通配置表 | ❌ **不合规**。`smtpPassword` 是 `system_configs` 普通键（`handlers/systemconfig/service.go:277`） | 迁 `ConfigOverride.secret_ref`；保留键名但值改为引用形式 |
| 7 | 外部 URL SSRF | 部分。`validateWebhookURL` 已拒绝私网/回环/链路本地（`service/bpmn/webhook_handler.go:69-92`） | `systemUrl` 未走该校验；需统一下沉到 Config Resolver |
| 8 | 多租户强制隔离 | ⚠️ 部分。RLS 默认 `off`（`config/config.go:303-310`、`docker-compose.prod.yml:144`） | 基线表 `tenant_id` 必填 + 阶段 3 起在 enforce 下跑门禁 |
| 9 | 配置变更留操作人审计 | 部分。`runs/attempts` 有执行人，但无字段 diff | `ConfigAuditEvent` 补 `before/after/reason/release` |

**约束 4 是本次改造的核心安全项**。当前行为：代码里新增一个权限码 → 下次播种自动授予所有匹配角色 → 等于"系统升级即扩权"。这与任务约束 4 直接冲突。修正方案：

```go
// 现状（seeder.go:2209-2236）：全量自动写
for _, code := range codes { /* 直接 Create role_permission */ }

// 目标：区分「既有码」与「新增码」
for _, code := range codes {
    if s.isExistingGrant(roleID, code) { continue }        // 幂等跳过
    if !s.isApprovedNewGrant(role.Code, code) {            // 新增码需审批
        s.pendingGrants = append(s.pendingGrants, ...)     // 进待授权清单
        continue
    }
    // 显式批准后才写
}
```

审批清单载体：`ConfigAuditEvent`（`action=new_grant_pending`）+ CLI `cloudmesh system grants --pending`。**不引入新表**。

---

## 4. 阶段 6：13 类测试矩阵

| # | 用例 | 断言要点 | 现有可复用夹具 |
|:--:|:--|:--|:--|
| 1 | 首次初始化 | 7 组件全 succeed；`system_baselines` 有 v1 行 | `fresh-install-gate.yml` |
| 2 | 重复初始化 | 第二次 run 零 `create`；`installations.installed_version` 不变 | `fence_test.go::TestPostgresEngineRepeatedSuccessIsIdempotentAndScopeIsolated`（**现成**） |
| 3 | 同版本重复协调 | 全部 `skip`；`config_audit_events` 无 `update` | 新增（在 #2 基础上加五态断言） |
| 4 | 新版本加菜单 | 仅新增行为 `create`；存量行 `skip` | `menus_reconcile_test.go` |
| 5 | 新版本加权限 | 权限行 `create`；**角色授权不自动写**，进待授权清单（约束 4） | `role_permissions_reconcile_test.go` |
| 6 | 管理员改系统默认菜单 | 改 name → 漂移 `field_modified` + `skip`，不覆盖 | `seeder_test.go` |
| 7 | 租户自定义角色 | `is_system=false` 角色重跑后 name/description 不变 | `role_permission_guard_test.go` |
| 8 | 用户自定义系统配置 | `ConfigOverride` 存在时基线不覆盖 | 新增 |
| 9 | 环境 URL 不同 | `systemUrl` 在 prod/dev 不同时不被基线重置；SSRF 拒绝 `127.0.0.1` | `webhook_handler` 校验复用 |
| 10 | 并发执行协调 | 两 goroutine 同时 `AcquireLease`，一个成功一个 `ErrLeaseHeld` | `fence_test.go::TestPostgresEngineTakeoverRollsBackStaleBusinessWrite`（**现成**） |
| 11 | 中途失败恢复 | Apply 注入错误 → `attempts.status=failed` → `retry` 后 succeed | `fence_test.go::TestPostgresEngineFailureRollsBackBusinessAndSuccessLedger`（**现成**） |
| 12 | 多租户隔离 | 租户 A 的协调不影响 B；RLS enforce 下跨租户查询返空 | `tenant_scope_test.go`、`fence_test.go::…ScopeIsolated` 段 |
| 13 | Dry Run 零写入 | `initialization_runs` 行数前后相等；`config_audit_events` 只有 `dry_run=true` | 新增 |

**复用比例**：13 条中 8 条可直接复用或扩展既有夹具（`fence_test.go` 的 7 个 Postgres 用例已覆盖接管、过期 fence、失败回滚、幂等、scope 隔离、取消关闭 run），真正需从零写的只有 #3、#8、#13 三条。这是"不重复造轮子"的直接体现。

**测试纪律**（沿用仓库既有判据）：

- 内存 SQLite 与 PostgreSQL 行为不同（`ORDER BY` 方言、`ON CONFLICT`）。租约/fencing 测试必须用真实 PostgreSQL（`fence_test.go` 已如此）。
- `go test -count=1` 禁止缓存命中。
- 破坏性用例（如 deprecate）必须自建探针对象，不拿种子数据当靶子。

---

## 5. 分阶段交付与门禁

| 阶段 | 交付物 | 门禁 | 人工审批 |
|:--|:--|:--|:--|
| 1 | `docs/engineering/system-bootstrap-analysis.md` | 无（只读） | ✅ 已完成 |
| 2 | `docs/engineering/system-baseline-design.md` | 无（只读） | ✅ 已完成 |
| 3 | 6 Schema + 3 迁移 + 生成代码 | `entgen-check`、`fresh-install-gate`、`go test ./migration/` | **① 迁移计划签字** |
| 4 | Reconciliation + 5 CLI | 13 类测试中 1/2/3/4/6/10/11/13 | **② dry-run 报告复核** |
| 5 | 9 条安全约束 | 逐条测试或审计证据 | **③ 约束 4/6/7 改动签字** |
| 6 | 全量测试矩阵 | 13 条全绿 | **④ 生产前演练** |

**不批量删除历史基线数据**：`backfill_baseline_v1` 只写新表；`system_baselines` 多版本并存，旧版本行永久保留。
**不重建生产数据库**：所有变更走前滚迁移 + `_down.sql`，不依赖清库。
**保留技术栈**：Go + Ent + PostgreSQL，不引入新框架。

---

## 6. 待用户确认的决策点

按约束 9，以下四项需明确后才能进入阶段 3：

1. **是否接受"并存"方案**：`SystemBaselineRecord` 与 `initialization_installations` 长期并存（前者存期望状态与所有权，后者存执行历史与租约），还是合并为单表？设计文档按"并存但单一写入方"处理，合并方案会改动 `sql_store.go` 全部租约 SQL，风险显著更高。
2. **约束 4 的审批载体**：新增权限授权改为"进 `ConfigAuditEvent` 待授权清单 + CLI 查询"，不引入新表。是否接受？
3. **`smtpPassword` 迁移方式**：保留键名、值改为 `secret_ref:ENV:SMTP_PASSWORD` 形式，还是新增独立键？前者前端表单需小改（显示引用而非明文）。
4. **K8s 迁移 Job 错线**（`deploy/k8s/22-migrate-job.yaml` 用 `./main migrate`，而 `main.go` 只认 `ITSM_BOOTSTRAP_ONLY`，镜像真实迁移工具是 `/out/migrate-tool`）：建议单独提交修复，不混入基线改造。

---

## 7. 明确不做的事

- 不重写 `internal/initialization/engine.go`（租约/fencing/事务/DAG 已正确实现）。
- 不新建独立执行引擎或事件驱动协调（会与 `ExecuteTool` + commandbus + BPMN 三条既有语义竞争）。
- 不改 `pkg/seeder` 的 7 组件 `apply` 主体（只在 `Plan` 层加五态，`Apply` 层加字段合并）。
- 不删除 `sql/init_data.sql` 等历史手工脚本（属历史治理，非本次范围）。
- 不动 `system_configs` 现有 24 键的语义，只加 `ConfigOverride` 覆盖层。
- **不代调既有红**：见 §8。

---

## 8. 本轮验证结果与既有红披露

### 8.1 已执行的验证（真实结果，未编造）

| 命令 | 结果 |
|:--|:--|
| `go vet ./pkg/seeder/ ./internal/initialization/` | exit 0，无输出 |
| `go test ./internal/initialization/ -count=1` | `ok itsm-backend/internal/initialization 0.689s` |
| `go test ./migration/ -count=1` | `ok itsm-backend/migration 0.757s` |
| `go test ./pkg/seeder/ -count=1` | **FAIL**（见 §8.2，既有红） |

### 8.2 既有红：`TestColdVerifyExpectationsAreDeterministicAndIndependent`

失败断言（`pkg/seeder/initialization_verify_cold_test.go:303-304`）：

```
require.Len(t, menuDefinitions(), 102)
require.Len(t, menus, 102)
```

实际 `menuDefinitions()` 返回 **101** 条。

**归因证据（判定为本批之外引入）**：

| 证据 | 结果 |
|:--|:--|
| `git archive HEAD itsm-backend/pkg/menubaseline` 后统计 `{Name:` | **102**（HEAD 自洽） |
| 当前工作树 `grep -c "{Name:" baseline.go` | **101** |
| `git diff HEAD -- baseline.go` | 并行会话删除了一级菜单 `{Name: "AI 中心", Path: "/ai/chat"}`，并把三个子菜单重挂到业务域 |
| 断言文件本身 | `initialization_verify_cold_test.go` **未被修改**（`git diff --stat` 无记录） |

结论：HEAD 版本（102 菜单 + 102 断言）自洽且通过；当前树的失败来自 `2026-11-21-ai-center-rehome` 会话删除一级菜单后**未同步更新断言数量**。这是该会话的收尾缺口（同会话已认领 `pkg/menubaseline/baseline.go` 与 `baseline_guard_test.go`，但漏了这一个断言文件）。

**本轮不代为修复**，理由：该断言属 `2026-11-21-ai-center-rehome` 的改动面，跨会话修改会造成认领冲突；且本轮任务是只读审查 + 设计 + 计划，不涉及 `pkg/seeder` 测试。已在下方作为交接项记录。

### 8.3 交接项（给 `2026-11-21-ai-center-rehome` 会话）

`pkg/seeder/initialization_verify_cold_test.go:303-304` 的 `102` 需改为 `101`，并在注释中补一行说明"2026-11-21 AI 中心一级菜单下线，102→101"。同一文件 `:296-302` 的历史注释链也需追加该条，否则下一个改动者会再次困惑。

---

> 本文档为**计划**，未执行任何数据库结构或数据变更。确认后再进入阶段 3。

