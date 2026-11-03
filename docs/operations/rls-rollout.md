# RLS 上线 Runbook：off → shadow → enforce

> 适用版本：v1.6.x 起。代码事实源：`itsm-backend/database/rls/`、
> `itsm-backend/migrations/20261031_rls_*`。

本文只回答一件事：**怎样把 PostgreSQL 行级隔离从"策略已就绪"变成"真的会拦截跨租户查询"，以及为什么当前不能直接改 `RLS_MODE=enforce`。**

---

## 1. 三个档位到底做什么

| 档位 | 应用侧行为 | 数据库是否过滤行 |
|:---|:---|:---|
| `off`（默认） | 不注入会话变量 | 否 |
| `shadow` | 注入 `app.current_tenant`；缺租户上下文的查询打 WARN | 否（仅观察） |
| `enforce` | 注入变量；缺租户上下文直接拒绝执行 | **取决于连接角色**（见 §3） |

`enforce` 不等于"隔离已生效"。驱动只负责注入变量和 fail-close；PostgreSQL 是否按策略过滤，取决于**连接角色**和**策略是否 FORCE**。

---

## 2. 为什么 enforce 会拒绝启动

`RLS_MODE=enforce` 时，后端启动会执行就绪探测（`database/rls/readiness.go`），任一条件不满足即**拒绝启动**，而不是告警继续：

| 未就绪原因 | 含义 | 修法 |
|:---|:---|:---|
| 连接角色是超级用户 | PostgreSQL 对超级用户无条件绕过策略，`FORCE` 也无效 | 改用 `itsm_app` 连接 |
| 连接角色带 `BYPASSRLS` | 策略对它不生效 | 改用普通应用角色；`BYPASSRLS` 只留给迁移/后台任务 |
| `public` 没有任何表启用 RLS | 策略层为空 | 重放 009/019/20261028 迁移 |
| 有表启用 RLS 但没有 policy | 默认拒绝所有行，业务大面积 0 行 | 执行 `20261031_rls_policy_alignment.sql` |
| 表归当前角色所有且未 `FORCE` | 表 owner 绕过非 FORCE 策略 | 改用非 owner 角色，或 `ALTER TABLE ... FORCE ROW LEVEL SECURITY` |

拒绝启动是刻意的：显式要求隔离却无法隔离时继续运行，等于对外承诺一个不存在的边界，而且会让 `enforce_applied` 计数看起来像"隔离生效"的证据。

---

## 3. 当前的真实阻塞点（必须先读）

**应用目前只有一个数据库连接池，`DB_APP_ROLE_USER` / `DB_ADMIN_ROLE_USER` 尚未接线**（`config.DatabaseConfig.AppDSN()` / `AdminDSN()` 已定义但无调用方）。也就是说：

- 把 `DB_USER` 换成 `itsm_app` → 请求路径能享受 RLS，但**启动时的建表/索引/迁移也会以 `itsm_app` 执行**，它会因为缺少 DDL 权限而失败；
- 保持 `DB_USER` 为部署 owner → RLS 永远不生效。

因此 `enforce` 上线的前置代码工作是**双连接池**：请求路径走 `itsm_app`，迁移/种子/后台任务走 `itsm_admin`。在此之前：

- 生产/K8s 清单保持 `RLS_MODE=shadow`（`deploy/k8s/01-configmap.yaml` 已改）；
- `enforce` 只会让你更快发现"配置了但做不到"，不会带来隔离。

> 这是本 Runbook 唯一需要写代码的前置项。角色、策略、谓词、账本都已经就绪。

---

## 4. 角色准备（带外，一次性）

角色由版本化迁移 `itsm-backend/migrations/20261031_rls_roles_bootstrap.sql` 自动创建，**默认 `NOLOGIN` 且不写口令**：

```bash
# 1) 跑迁移（应用启动或 migrate 命令会自动执行；也可手工）
psql -U <owner> -d <db> -v ON_ERROR_STOP=1 -1 \
  -f itsm-backend/migrations/20261031_rls_roles_bootstrap.sql

# 2) 带外设置登录口令（口令永远不进仓库）
psql -U <owner> -d <db> -c "ALTER ROLE itsm_app   LOGIN PASSWORD '<strong-password>';"
psql -U <owner> -d <db> -c "ALTER ROLE itsm_admin LOGIN PASSWORD '<strong-password>';"

# 3) 校验属性
psql -U <owner> -d <db> -c \
  "SELECT rolname, rolsuper, rolbypassrls, rolcanlogin
     FROM pg_roles WHERE rolname IN ('itsm_app','itsm_admin');"
# 期望：itsm_app  superuser=f bypassrls=f
#       itsm_admin superuser=f bypassrls=t
```

迁移会同时授予 `itsm_app` 对 `public` 现有表/序列的 CRUD，并为**当前执行者未来创建**的对象设置默认权限。因此迁移必须以表 owner 身份执行。

---

## 5. 策略姿态校验

```bash
# 5.1 每张 RLS 表都应有 tenant_isolation_<table>，且谓词走 helper
psql -U <owner> -d <db> -c "
  SELECT c.relname,
         c.relrowsecurity AS rls,
         c.relforcerowsecurity AS force,
         p.policyname,
         p.qual
  FROM pg_class c
  LEFT JOIN pg_policies p
    ON p.schemaname='public' AND p.tablename=c.relname
  WHERE c.relkind='r' AND c.relnamespace='public'::regnamespace
    AND c.relrowsecurity
  ORDER BY 1;"
```

任何 `qual` 不是 `tenant_id = get_current_tenant_id()` 的策略都应由
`20261031_rls_policy_alignment.sql` 统一；该迁移末尾会自检，漏表直接失败。

### 真实拦截验证（不需要应用）

```sql
-- 以 itsm_app 身份验证跨租户不可见
SET ROLE itsm_app;
SELECT set_config('app.current_tenant', '1', false);
SELECT count(*) FROM <some_tenant_table> WHERE tenant_id = 1;  -- > 0
SELECT count(*) FROM <some_tenant_table> WHERE tenant_id = 2;  -- 0
SELECT count(*) FROM <some_tenant_table>;                      -- 只含 tenant 1
RESET ROLE;
```

如果第 3 条仍返回其它租户的行，说明当前角色是 owner/超级用户/带 BYPASSRLS，或策略没生效。

---

## 6. 切换流程

1. **保持 `RLS_MODE=shadow` 运行一个完整业务周期**，观察日志中
   `rls: shadow mode is observable but enforce would NOT be effective yet`：
   - 出现该 WARN 说明就绪探测未通过，按 `reasons` 逐条修；
   - 未出现说明数据库姿态已就绪，但**仍不能切 enforce**，直到 §3 的双连接池落地。
2. **完成双连接池**：请求路径用 `itsm_app`，迁移/种子/后台任务用 `itsm_admin`。
3. **在预发环境切 `enforce`**，跑核心回归：
   - 跨租户直查 ID 必须返回"不存在"；
   - 列表/导出/仪表盘不得少行；
   - 后台任务（SLA 扫描、BPMN 推进、通知投递）必须用 `itsm_admin` 或显式 system context，否则会被策略挡住。
4. **生产切 `enforce`**：改环境变量并重启；启动失败即表示就绪探测未通过，按日志 `reasons` 处理，**不要**为了"先起来"把模式改回 off 而不记录原因。
5. **回滚**：把 `RLS_MODE` 改回 `shadow` 并重启即可。策略本身不需要删除。

---

## 7. 不要做的事

- **不要**直接执行 `database/rls/migrations/002_pilot_policies.sql`。它是早期 FORCE 试点（仅 `changes`/`vectors`，策略名无表名后缀），只用于 `tests/multi_tenant` 的范围棘轮与历史验证；生产姿态由 009/019/20261028/20261031 定义。执行它会在同一张表上叠加第二条策略，并让 `20261031_rls_policy_alignment.sql` 的规范化结果失真。
- **不要**把 `itsm_app` 设为 `SUPERUSER` 或 `BYPASSRLS` 来"绕过权限问题"。那会让 RLS 静默失效，而 readiness 探测会拒绝启动。
- **不要**在 `enforce` 下让后台任务复用请求路径连接。跨租户系统任务必须走 `itsm_admin` 或显式 system context，并留审计。
- **不要**手工编辑 `schema_migrations` 里的 checksum。迁移内容变更必须新增迁移，不能改历史文件。

---

## 8. 相关文件

| 文件 | 作用 |
|:---|:---|
| `itsm-backend/database/rls/driver.go` | Ent 驱动装饰器；`ParseMode` 严格失败关闭；会话变量注入 |
| `itsm-backend/database/rls/readiness.go` | enforce 就绪探测（五类 unready） |
| `itsm-backend/database/rls/rls.go` | 连接级 `set_config` / `DISCARD ALL` |
| `itsm-backend/migrations/20261031_rls_policy_alignment.sql` | 策略命名/谓词统一 + 自检 |
| `itsm-backend/migrations/20261031_rls_roles_bootstrap.sql` | 双角色引导（NOLOGIN、无口令） |
| `itsm-backend/migration/migrations.go` | 009/019：helper 与首批策略（权威内联流，禁改） |
| `itsm-backend/migrations/20261028_rls_core_tables.sql` | 5 张事务主表补齐策略 |
