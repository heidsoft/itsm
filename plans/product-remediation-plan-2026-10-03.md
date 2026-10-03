# 产品完善计划（2026-10-03）

> Status: draft，待拍板。
> 来源：[`docs/review/product-convergence-initialization-permission-review-2026-10-03.md`](../docs/review/product-convergence-initialization-permission-review-2026-10-03.md)（三轮实测审查，全部条目带 `file:line` 证据）。
> 产品依据：`AGENTS.md` 的 Product Direction / 强制规则；`ROADMAP.md` v1.6.x 收敛项与 Always-On Tracks。
> 本文的性质：**把审查发现派发到既有规划主线上，并给出批次、退出标准和需要拍板的决策点。** 本文不新增实现细节，凡涉及具体修法一律回到审查报告对应小节。

---

## 0. 与既有规划主线的关系（本文不另起第四条主线）

`plans/product-feature-completeness-plan-2026-10-01.md:§0` 已经立过规矩：仓里有多条在跑的规划主线时，新发现应当**归位**而不是**另开**。本轮 46 + 25 + 12 条发现全部可以落到既有主线上，只有权限层的两个 P0 需要新开一个批次编号。

| 审查发现 | 归属主线 | 归位理由 |
|:--|:--|:--|
| §1 P0-2　两套权限权威不一致 / 播种让角色变弱 | **`ROADMAP.md:103` 授权平面收敛 → 新开批次 6** | 批次 1–5 已落地的最后一条正是"admin/technician DBOnly 空集修复 + seeder 同义动作奇偶补齐"。P0-2 是**同一缺陷类的未完部分**（当时只对齐了 2/9 个角色），不是新问题 |
| §1 P0-1　注册接口可无认证自选 `super_admin` | **`ROADMAP.md:234` Always-On Tracks → Security** | 未认证提权属于常驻安全轨，不属于任何版本收敛批次 |
| §2 初始化与种子载体 12 条 | **`plans/production-data-initialization-blueprint.md` Step 1 / Step 4** | Step 1 = "冻结初始化契约并清除生产 P0 风险"，Step 4 = "收敛身份、RBAC、ACL 与菜单"。种子载体带毒、init_admin.sh 口令重置落 Step 1；双权限权威、死授权角色键落 Step 4 |
| §3.2 业务实现 5 条（邮件未接线、空成功、fire-and-forget、乐观锁可绕过、硬编码 tenant 1） | **`plans/product-feature-completeness-plan-2026-10-01.md` 批次 1「消灭假成功」+ 批次 2「契约与安全」** | 该计划批次 1 的定义就是"消灭假成功"，#4 #5 是同类；#6 #7 #8 违反 AGENTS.md 强制规则，属批次 2 |
| §3.1 API 契约 3 条 | **同上，批次 2「契约与安全」** | 列表信封棘轮已在推进（50 → 25 条），继续按棘轮逐条删即可 |
| §3.3 UI 交互 / §3.4 门禁与 CI | **`plans/scope-convergence-plan-2026-09-28.md` B0「恢复门禁可信」** | B0 的定义就是"让绿重新等于测过"；`lint:antd` 覆盖 3/9、`ga-gate` 空转、C.7.1 粒度不足都是 B0 范畴 |
| §4.1 可达的假实现 3 处 | **`plans/edge-feature-stability-audit-2026-10-02.md` 第三节「假成功与未实现被当成可用能力卖」** | 同一节已有 5 条同类证据，本轮 3 条是增量 |
| §4.2 孤儿组件中的假实现 7 处 + 55 个孤儿文件 / 17,987 行 | **同上，第四节「前端平行死实现」** | 该节已列 8 个死模块。本轮 55 个文件是**同一现象的更大口径**，需先做 §1 的术语校准再合并计数 |
| §4.3 门禁盲区 2 条 | **`scope-convergence-plan` B0** | C.7.1 看不见文件级孤儿，属门禁粒度问题 |

**结论：本轮不需要新计划文件承载实现，只需要（a）ROADMAP 开一个批次 6，（b）把上面 8 行派发下去，（c）拍板 §3 的 4 个决策点。**

---

## 1. 术语校准：两种"空壳"必须先分开，否则两份计划会互相打脸

`plans/scope-convergence-plan-2026-09-28.md:53-56` 有一条已生效的修正：

> ⚠️ 第二处修正：空壳结论作废，B1 整批撤销 …… 逐文件复核后确认**全部是误判，真实空壳 = 0**

本轮审查报告 §4.2 却说有 **55 个孤儿组件 / 17,987 行不可达代码**。两个结论**都对，但口径不同**，必须在派发前写清楚，否则后来者会以为其中一份在撒谎：

| 口径 | 定义 | 结论 | 出处 |
|:--|:--|:--|:--|
| **A：菜单可达空壳** | 菜单上有入口、页面存在，但后端不通 / 点了就坏 | **0**（原判 10 模块 / 15 页面全部误判，已作废） | `scope-convergence-plan:53-56,145` |
| **B：文件级不可达** | 文件存在于仓库，但没有任何生产路由能到达它 | **55 个组件 / 17,987 行**（另有 `edge-feature-stability-audit:第四节` 已列的 8 个死 API 模块） | 审查报告 §4.2 |

两者的交集才是真正需要决策的部分：**口径 B 里的文件如果含假实现，修它是纯浪费**（审查报告 §4.3 #25 的流程发现）。所以派发顺序是：

> **可达性 → 真伪**，不是 **真伪 → 可达性**。

具体动作见 §2 R5。

另外必须记录在案：`check-scope-creep.sh:24-34` 自己就写着第一版守卫产出 10 模块 / 15 页面全是假阳性，教训是"静态检测结论在下判断前必须逐文件复核"。本轮 R5 的删除清单**必须逐文件复核后才允许删**，不得凭扫描器输出直接 `git rm`。

---

## 2. 批次 R1–R8

批次编号用 R（remediation），避免与 `scope-convergence` 的 B0–B6、`product-feature-completeness` 的批次 0–4、`授权平面收敛` 的批次 1–5 撞号。**R 编号只用于本文的排序，实际执行时各条目回到自己的主线批次里。**

### R1　未认证提权收口（P0-1）→ ROADMAP Security 轨

| 项 | 内容 |
|:--|:--|
| 目标 | 消除唯一一条"未认证即可拿到全系统权限"的路径 |
| 动作 | ① `role` 移出注册 DTO，或收敛为常量 `end_user`；② `tenantCode` 不接受请求体，只从认证上下文/域名解析；③ 把 `super_admin` 短路移到 DBOnly 门禁**之后**（`middleware/smart_permission.go:117-122` vs `:145-150`） |
| 退出标准 | 新增回归测试：注册请求体带 `role: "super_admin"` 时，落库角色必须是 `end_user`，且该账号打 `/api/v1/dashboard` 与任一 admin 路由返回 403。测试必须打真实 `SetupRoutes`，不得只测 handler 函数 |
| 为什么排第一 | 改动小、无产品语义争议、不需要拍板（`role` 本来就不该由客户端选）。而 R2 需要拍板，不能被 R2 阻塞 |
| 归属 | `ROADMAP.md:234` Security 轨；同步登记到 `production-data-initialization-blueprint.md` Step 1 |

> **状态回写（2026-10-03，已落地）**
>
> - **代码**：后端 `19500ae6`——`RegisterRequest` 移除 `role`/`tenantCode`，目标租户服务端单源解析（唯一活跃租户自动落、多活跃租户 fail-closed、零活跃租户报"系统尚未初始化"），落库角色固定 `user.RoleEndUser`；前端 `9a5b3f3a`——注册页移除角色选择器，请求体不再携带 `role`/`tenantCode`，契约测试断言请求体不含这两个字段，并清理 6 个 i18n 孤儿键。说明：后端提交由并发会话代为提交（本计划执行者的已验证暂存内容被并入其提交，提交信息与内容相符，按共享工作区纪律报告坐标、不另造重复提交）；前端提交为执行者本人。
> - **动作③不执行，实测为无操作**：三处 `super_admin` 短路（`middleware/rbac.go` 的 `AuthorizeResourceForRole`、同文件 `hasPermission`、`middleware/smart_permission.go` 的 `SmartCheckPermission`）中，后两处在 DBOnly 分支上最终委托给第一处，移动短路位置不改变任何判定结果；而彻底移除短路会让 DBOnly 生产库的 `super_admin` 被锁死——其 Role 行存在即视为"已配置"，空权限集按显式收回处理返回 403，正是 2026-09-17 P0（合法管理员被锁死）缺陷类的复现。super_admin 的显式出口问题归入 R2/N5 授权平面收敛，不在 R1 偷渡。
> - **测试证据**（均在落库前实测为绿）：路由级回归打在真实 `SetupRoutes` 上——多活跃租户下带 `role: "super_admin"` 的攻击 payload 返回 401、业务码非 0、用户表零落库；单租户部署下同 payload 落库角色为 `end_user`，真实登录拿到 cookie 后访问 `GET /api/v1/admin/operations/commands` 返回 403（若提权成立，super_admin 短路会直接放行，该断言即失败）。后端 `go test ./handlers/auth/... ./router/...` ok、`go build ./...`、`go vet`、gofumpt 干净；前端 jest auth-service 30/30、api-contract 3/3、`npm run type-check` 干净。
> - **退出标准修正一处**：原标准要求 dashboard 与 admin 路由都返回 403，实测 `middleware.RolePermissions["end_user"]` 本就包含 `dashboard:read`，dashboard 返回 200 是合法行为而非提权；判别断言以 admin 路由 403 为准。
> - **文档同步**：`docs/api-reference.md` 注册节（移除 `role` 示例 + 说明角色与租户由服务端决定）随 `7146337f` 落库，CHANGELOG P0-1 条目随 `e83e5016` 落库（v1.6.11 发布整理；两笔均为并发会话代为提交，注册节文本为本文执行者所写）。标签 `v1.6.11` 经 `git tag --contains 19500ae6` 验证已包含后端修复提交。
> - **归属修正**：原表写"同步登记到 blueprint Step 1"，重读 `production-data-initialization-blueprint.md` Step 1 后确认其任务面是 Seeder/演示数据/固定密码，与注册入口契约无交集，不做强行登记；ROADMAP Security 轨为程序级清单（扫描/威胁建模/渗透测试）而非漏洞修复台账，P0-1 的权威记录在 CHANGELOG 与 api-reference。此条按本文 §6 纪律（引用坐标动手前重测）执行。
> - **UPGRADE.md 不需要更新**：历史字段为静默忽略而非报错，属非破坏性契约变更，按 AGENTS.md 规则仅破坏性变更触发 UPGRADE.md。

### R2　双权限权威对齐（P0-2）→ 授权平面收敛 批次 6

分三步，**第一步不改行为，只让 CI 红出真实差集**：

**R2-a　扩大守卫范围（无争议，可立即做）**

`pkg/seeder/role_permission_guard_test.go:60` 的 `parityRoles` 当前只有 `{domainrole.Admin, domainrole.Technician}`。扩到**全部词表角色**（`middleware.RolePermissions` 的 13 个键 ∩ `authz.BuiltinRolePermissionCodes()` 的 32 个键）。

- 退出标准：守卫在扩范围后**必须是红的**（否则说明差集已被别处修掉，需重新取证）。红出来的差集清单要原样贴进本文 §3 N4 的决策材料。
- 注意：这一步会让 CI 变红，**必须先与 §3 N4 的拍板同批进行**，或者以 `t.Skip` + 明确 TODO 的形式落地并在同一提交里登记待拍板事项，不得留一个长期红的 CI。

> **状态（2026-10-03，已落地，与差集登记同一提交）**：守卫按「词表角色 ∩ `middleware.RolePermissions` 同名键」结构性扩围，不再手工维护角色清单。实测口径：`middleware.RolePermissions` 共 **9 个键**（`rbac.go:65`：super_admin/sysadmin/admin/manager/agent/technician/security/end_user/msp_viewer），与 `domainrole.All` 同名的 **7 个**（`it_admin`、`security_admin` 无 middleware 条目；middleware 的 `security`、`msp_viewer` 不在领域词表），扣除 super_admin 的 `Login ["*"]` 旁路后实际校验 **6 个**——硬档 admin/technician（差异即红，实测 0 缺失）+ 待拍板档 manager/agent/sysadmin/end_user（子测试 `t.Skip` 登记真实差集，共 50 对）。本文上文的"13 个键 ∩ 32 个键"为过时数字，以本实测为准。差集清单原样登记在 §3 N4。

**R2-b　修掉错误前提（无争议）**

`middleware/rbac.go:1004-1007` 的注释断言：

> sysadmin 不再短路：DBOnly 模式下其权限来自 role_permissions 播种数据（seeder.go allPermissionCodes() 全量授权），权限收回/降级因此可生效。

而 `allPermissionCodes()` 返回 **129** 个码，catalog 有 **160** 个，差 **28** 个码，且这 28 个码全部只出现在 `internal/authz/roles.go:232-262` 的 `admin` 块里。前提为假 → 注释必须改，或者代码必须补齐到真。**这条不需要产品拍板，因为它是"注释撒谎"，不是"语义选择"。**

> **状态（2026-10-03）：已落地（选"改注释"，最小改动）。** 复测快照：`allPermissionCodes()` 实测 129 码（uniq 129，off-catalog 0），catalog 160 码中 **31** 码不在其内——本文上文的"差 28 码"为过时数字，差集随 catalog 演化漂移，以注释内快照日期为准。已核实判定链 `hasPermission → AuthorizeResourceForRole → loadPermissionsByMode`：DBOnly configured 态确实尊重 role_permissions（空集=显式撤销），"收回/降级机制上可生效"为真；为假的是"全量授权"与 "seeder.go" 文件引用（函数实际位于 `internal/authz/roles.go:388`）。`rbac.go:1004-1009` 注释已按实测改写：保留机制层为真的断言，标注码集覆盖缺口（129/160、不随 catalog 自动跟上），指向 §3 N4/N5 拍板。补齐代码（授予 31 个缺失码）属语义选择，不属本条。

**R2-c　产品决策后实施（见 §3 N4）**

无论选扩权还是收权，退出标准都包含：`sysadmin` 与 `end_user` 打 `/api/v1/dashboard` 的跨角色回归测试，且测试在修复前必须失败。

**R2-d　解除自愈阻塞（独立于 N4，可并行）**

`pkg/seeder/seeder.go:2068-2083` 会在每次播种时 `Delete().Where(RoleIDEQ, TenantIDEQ, PermissionIDIn(managed...), PermissionIDNotIn(permIDs...))` —— **运维在数据库里手工补的授权，下一次播种就被静默撤销**。在 N4 拍板前，这一条会让任何手工热修失效。

- 动作：把 managed 集合的收敛改成"只增不减 + 显式退役清单"，或者要求删除动作走 `initialization_installations` 账本记录并输出 diff 日志。
- 归属：`production-data-initialization-blueprint.md` Step 4。

### R3　种子载体去毒 → 初始化蓝图 Step 1

| 项 | 内容 |
|:--|:--|
| 动作 | ① **删** `itsm-backend/config/seed/seed_data.sql`（38,991 B，全仓引用 0，内含 catalog 外的权限码 `admin:write` / `change:manage` / `tenant:manage`，一条 `psql -f` 就能污染 RBAC）；② `demo.json` 移出生产镜像 —— `itsm-backend/Dockerfile:69` 与 `Dockerfile.prod:100` 都是 `COPY config/seed ./config/seed`（整目录），改为只 COPY 生产需要的文件，或把 demo 段挪到 dev 镜像专用目录；③ 修 `itsm-backend/init_admin.sh`：去掉重跑时的口令重置与 `role='admin'` 强制降级（`:35-43`，副作用是已提为 `super_admin` 的账号跑一次脚本就被降级）、改掉 `psql -v admin_pw=` 的进程列表暴露（`:20`）、去掉硬编码 id `'admin-001'`（`:48`）与 `admin@itsm.com`（`:50`） |
| 退出标准 | 生产镜像内 `ls config/seed` 不含 `demo.json` 与 `seed_data.sql`（用 `docker run --rm <prod-image> ls` 实测，不看 Dockerfile 推断）；`init_admin.sh` 重跑两次后 `role` 不变、`password_hash` 不变；新增一条守卫：`config/seed/**` 内出现的权限码必须 ⊆ `authz.Definitions()` |
| 依赖 | 无。删除死文件是纯收益，可与 R1 并行 |

**⚠️ 本批次不继承审查报告的一条错误结论。** 报告 §2 #3 曾写"`make dev-seed-demo` 在 Makefile 中不存在"，**这是错的**，该 target 存在于 `Makefile:240-249`、已列入 `.PHONY`（`:2`）、并有 `ROADMAP.md:94` 与 `README.md:142` 的文档。报告已就地划掉并记入 §6 #9。R3 的动作 ② 因此**不是"删掉一个不存在的入口"，而是"把一个真实存在的 dev 入口的产物移出生产镜像"**——`make dev-seed-demo` 本身要保留。

> **状态（2026-10-03）：已落地，退出标准全部实测通过。**
> - 动作①：`config/seed/seed_data.sql`（38,991 B）已删除；目录现存 `default.json` + `demo.json`。
> - 动作②：`Dockerfile` 与 `Dockerfile.prod` 的 `COPY config/seed ./config/seed` 改为只 COPY `config/seed/default.json`（注释标明 demo.json 属 `make dev-seed-demo` 开发载体，不进生产镜像）。生产镜像内 `ls config/seed` 以 `docker run --rm` 实测为准（结果见本批次提交说明）。
> - 动作③：`init_admin.sh` 重写——重跑改为幂等跳过（不重置口令、不调整角色，已提为 super_admin 的账号不再被降级）；口令/邮箱只经 stdin 与 `PGPASSWORD` 环境变量进入 psql，不再走 `psql -v` 进程列表；硬编码 id `'admin-001'` 与 `admin@itsm.com` 删除（id 由数据库自增，email 可用 `ADMIN_EMAIL` 覆盖，默认 `admin@example.com` 与 Go seeder 基线一致）；默认租户缺失时 fail closed（`RAISE EXCEPTION`），不再回退 `tenant_id=1`。
> - 守卫：新增 `pkg/seeder/seed_carrier_guard_test.go`——扫描 `config/seed/` 全部文件（不限扩展名，防 seed_data.sql 式再引入），凡 `resource:action` 形状 token 必须 ⊆ `permissionDefinitions()`；基线 0 命中，负样例探针（`admin:write`）实测命中变红。
> - 实测记录：演练库（一次性 postgres:16-alpine 容器，非业务库）首跑创建 admin（role=admin、tenant 取自 `tenants.code='default'`）；运维提权+改口令后重跑，`role` 仍为 `super_admin`、`password_hash` 与运维改后一致；无默认租户库重跑 exit 3 并报「默认租户不存在，不做租户回退」，0 行写入。
> - 观察项（不在本批次扩权）：脚本只写 `users.role` 不写 `user_roles` 边；`MigrateUserRolesBackfill` 在每次 seeder Apply 时回填，下次服务启动自愈。
> - 备注：`CHANGELOG.md [Unreleased]` 的 R3 条目因该文件正处于并行会话在途修改中，另行落账，不与本批次提交混入。

### R4　门禁补口 → scope-convergence B0

| 缺口 | 动作 | 退出标准 |
|:--|:--|:--|
| `tools/check-antd-legacy.sh:18-22` 只覆盖 9 类禁用 API 中的 3 类 | 补 `visible=`、`destroyOnClose`、`bodyStyle`、`overlay`、`dropdownRender`、`onDropdownVisibleChange` 六类 | 每类各造一个违规样例，脚本必须命中；再跑全仓，命中数登记为基线，只许降不许升 |
| `ga-gate` 在 "Start core stack" 处死掉 → 唯一的全栈 E2E 从未运行 | 先修启动步骤，让 E2E 真跑；跑不起来就必须**把该 job 标红或删掉**，不得留一个空转的绿 | CI 日志里能看到 E2E 用例数 > 0 |
| C.7.1（`check-scope-creep.sh:24-58`）是模块级粒度，看不见文件级孤儿 | 增加文件级孤儿检测：`src/components/**` 与 `src/lib/api/**` 中不被任何 `src/app/**` 页面（含其传递引用）到达的文件 | 见 R5：先出清单 + 逐文件复核，**守卫第一版设为 advisory**，等 R5 清完再转 blocking。理由就是 `check-scope-creep.sh:33` 自己写的教训——静态检测假阳性会逼人删掉正确的东西 |
| 补：六类 antd 缺口之外，`lint:antd` 需要覆盖"项目自有组件的同名 prop" | 检测必须区分组件归属，不得机械全局替换（AGENTS.md 已有此规则） | 自有组件用 `visible` 不误报 |

### R5　孤儿文件清理 → edge-feature-stability-audit 第四节

**顺序强制：先定可达性，再谈真伪。**

1. 生成 55 个文件的清单（组件 + `lib/api` 客户端 + `lib/services`），每条附"最近一次 git 提交"与"是否被 barrel 文件（`index.ts`）引住"。
2. **逐文件人工复核**（不得凭扫描器输出直接删）。三类结论：
   - **确认死** → `git rm`，同一提交里更新 `edge-feature-stability-audit` 第四节的计数；
   - **应当接线**（功能是真的、只是没挂页面）→ 转为 `product-feature-completeness-plan` 的候选需求，标注需要产品拍板；
   - **误判**（扫描器漏了动态 import / 路由约定 / barrel）→ 记入清单并修扫描器。
3. 清完之后，**再**处理审查报告 §4.2 的 7 处假实现 —— 其中位于已删文件里的自动消失，位于"应当接线"文件里的才值得修。
4. 同步处理 §4.1 的 3 处**可达**假实现（这三处不受清理顺序影响，可直接修）：
   - `components/ticket/TicketAdvancedSearch.tsx:125-133` 硬编码 `assigneeId: 1` 冒充当前用户 → 改为从认证上下文取；`:174`、`:327`、`:335-349`、`:403`、`:442-453` 的 `savedSearches` 是 useState + `message.success('搜索条件已保存')`，整个文件零 `@/lib/api` 导入 → 要么接后端保存，要么把"保存搜索条件"按钮**摘掉**，不得留一个点了会骗人的按钮；
   - `src/app/agent-ops-demo/page.tsx`（216 行 / 102 条长中文文案 / 硬编码 evidence 与 guardrails）→ 见 §3 N6；
   - `components/change/ChangeList.tsx:70-105` 风险过滤器 no-op（注释 `// risk在API中可能对应type，这里先不处理`）→ 修契约或摘掉过滤器，二者择一，不得留一个改了不生效的下拉框。

### R6　假成功与强制规则违规 → product-feature-completeness 批次 1 + 批次 2

| # | 项 | 动作 | 归属 |
|:--|:--|:--|:--|
| 1 | 忘记密码永不发邮件但返回成功文案（`service/email_service.go:53-54`；`handlers/auth/service.go:36,170-184`） | 未配置邮件服务时返回 **operational unavailable**（AGENTS.md：缺少 connector 配置属 operational unavailable，不得返回空成功），前端文案改为"密码重置功能未启用" | 批次 1 |
| 2 | `service/ai_services.go:1-30` `SimilarIncidents` 三处返回 `[]map[string]any{}, nil`（nil 依赖 / embed 错误 / 向量检索错误），且**零调用方** | 零调用方 → 先判死活；若活则改为返回 degraded 状态 + 错误分类 | 批次 1 |
| 3 | `handlers/incident/service.go:195-206` `go s.executeRules(...)` 事务提交后 fire-and-forget | 改走 `commandbus.EnqueueTx`，与业务写入同事务 | 批次 2（违反强制规则） |
| 4 | `service/ticket_service.go:1098-1112` 省略 `version` 可绕过乐观锁 | 去掉 `if req.Version > 0` 的退化分支；`version` 缺失按 validation error 拒绝 | 批次 2（违反强制规则） |
| 5 | `cmd/cmdb/main.go:110-130` 两个分支都写 `tenant_id = 1` | fail closed：无租户上下文直接拒绝启动 / 拒绝请求 | 批次 2（违反强制规则） |
| 6 | `router/dashboard_routes.go:147,153,188,269` 四处 `gin.H{"success": true}` | 改为统一 `{code:0,message,data}` | 批次 2 |
| 7 | `src/config/product-capabilities.ts`：`DISABLED_API_CONTRACTS` 5 条全部指向未注册路由；10 个开关中 7 个零消费者 | 逐条判定：路由该注册的注册，开关该删的删。**capability flag 不得用来掩盖后端缺口** | 批次 2 |
| 8 | 列表信封棘轮（`tests/contract/list_envelope_ratchet_test.go`） | 按棘轮逐条删，每条同步前端契约。**这是一个正在被并发会话移动的靶子**：审查报告写作期间它是单列表 25 条，本文起草时已实测为**三个列表**——`envelopeBaseline`（`:61`，23 条）、`envelopeKeyBaseline`（`:101`，5 条）、`envelopeAliasBaseline`（`:116`，6 条）。**不要照抄本文的任何数字**，动手前重新 `awk` 计数 | 批次 2 |

**⚠️ R6 的并发碰撞提示。** 本文起草时工作区有他人在途改动，其中 `itsm-frontend/src/components/business/NotificationCenter.tsx` 既在 R5 的孤儿清单里（审查报告 §4.2 #19：`handleTestChannel` 是 1 秒 `setTimeout` 后永远成功），又正被 notification 域的并发会话修改。**R5 不得在该文件在途期间对它做删除判定**，须等其落地后重新取证可达性——并发会话可能正是在给它接线。同类需先 `git status` 确认归属的还有 `docs/api-reference.md`、`docs/testing/static-analysis-gates.md`（R8 会改这两处）。

### R7　初始化语义修补 → 初始化蓝图 Step 1 / Step 4

| # | 项 | 动作 |
|:--|:--|:--|
| 1 | `mergeSeedConfig`（`pkg/seeder/seeder.go:469-543`）是整段替换：客户 JSON 覆写 `roles` 段会静默丢掉其余 17 个默认角色 | 改为按 key 合并，或至少在整段替换时输出"将丢弃 N 个内置条目"的显式警告日志 + 文档 |
| 2 | `loadSeedConfig`（`:441-444`）用 cwd 相对路径 | 改为可注入的绝对路径 / 嵌入 FS，并在启动日志打印实际解析到的路径 |
| 3 | `seeder.go:1204-1216` 注释说"config 优先"，代码 `if seen[r.Code] { continue }` 保留 **builtin** | 改注释或改代码，二者必须一致（同 R2-b，属"注释撒谎"类，不需拍板） |
| 4 | `seedGroups`（`:1143-1176`）create-only，组改名永不前向修复 → BPMN `candidateGroups` 静默失效 | 按 code 做 upsert（改名/改描述属产品基线，应前向修复；成员归属属运维选择，不得覆写） |
| 5 | `tenant_provisioner.go` 不写 `initialization_installations` 账本，且 `:143` 注释写"the initialization ledger stays the source of truth"而租户域无账本行；整个 DAG 单事务但**无 fencing** | 二选一：写账本 + 加 fencing，或改注释并说明"租户开通走 SystemConfig 版本标记 + `VerifyTenantReadiness`，不走账本"。**注释与事实不一致本身是缺陷** |
| 6 | 4 个角色键（`rd_manager`、`developer`、`qa_engineer`、`team_lead`，`security` 同类）无对应 Role 行 → 死授权；32 个 map 键 vs 24 个播种角色 | 加守卫：`BuiltinRolePermissionCodes()` 的键集合必须 ⊆ 实际播种的角色 code 集合；多余的键删掉或补播种 |
| 7 | 无首次登录强制改密（全仓 grep `must_change_password` / `MustChangePassword` / `password_changed_at` / `force_password` = 0） | 见 §3 N7（是否本轮做，需拍板：涉及 schema 变更 + 迁移） |
| 8 | `ENV` 与 `SERVER_ENV` 两条解析路径（`internal/bootstrap/app.go:347,1283-1290,1565-1567`） | 收敛为一条；今天 fail-safe 不代表明天 fail-safe，且权限模式选择依赖它 |

### R8　文档回写（与上述各批同提交，不得留到"下个 PR"）

AGENTS.md 的 Documentation Sync Discipline 是强制的，本轮欠账：

| 文件 | 欠账 | 动作 |
|:--|:--|:--|
| `ROADMAP.md:4` | `Last synced: 2026-09-25`，已过期 8 天 | 每批落地时刷新；本轮首次提交即刷新 |
| `ROADMAP.md:103` | 授权平面收敛只记到批次 1–5 | R2 立项时补"批次 6（进行中）"，落地后改为已落地并附证据 |
| `ROADMAP.md:79-140` | v1.6.x「当前收敛项」6 条未反映本轮 P0 | R1/R2 登记为收敛项 |
| `CHANGELOG.md [Unreleased]` | 无本轮条目 | 每个 user-visible 修复同提交追加 |
| `docs/api-reference.md` | R6 #1（忘记密码语义变化）、R6 #6（success 标识移除）、R6 #8（列表信封）都属契约变更 | 同提交更新 |
| `plans/edge-feature-stability-audit-2026-10-02.md` 第四节 | 死实现计数未含本轮 55 个文件 | R5 完成后合并计数，并注明两种口径（本文 §1） |
| `plans/scope-convergence-plan-2026-09-28.md:53-56` | "真实空壳 = 0" 与本轮"55 个孤儿"表面冲突 | 在该节加一行指向本文 §1 的口径区分，**不改其结论**（它的结论在它的口径下是对的） |

---

## 3. 需要拍板的决策（编号 N4–N7，避开既有 D1–D8 与 N1–N3）

> N1–N3 已被 `product-feature-completeness-plan-2026-10-01.md:162` 占用（N1/N2 已拍板并落地，N3「统一工作台跨人可见范围」仍开放）。D1–D8 属架构主线。

### N4　P0-2 走"扩权"还是"收权"？（阻塞 R2-c，不阻塞 R2-a/b/d）

实测事实：`sysadmin` 在 DB 播种里比硬编码权威**少 28 个码**，其中 `dashboard:read` **只有 `admin` 一个角色持有**，而该码在 89 条路由的预检映射里被强制（`middleware/rbac_precheck_gen.go`，Resource/Action 分开存，所以字面 grep `"system_config:read"` 会返回假零）。

| 选项 | 含义 | 代价 |
|:--|:--|:--|
| **A：扩权** | 给 `sysadmin` / `manager` / `agent` / `end_user` 等补 `dashboard:read` 等缺失码 | 放大能力边界；需要逐码判断"这个角色真的该看仪表盘吗"，不是一次性补齐 28 个 |
| **B：收权** | 把 `dashboard:read` 从路由上摘掉，改为"认证即可访问" | 改变产品语义（仪表盘不再是受控资源）；但符合直觉，且一次动作解决 89 条路由 |
| **C：混合** | 仪表盘类只读聚合改"认证即可"；其余缺失码按角色逐个补 | 工作量最大，但语义最准 |

**审查者不代选。** 建议 R2-a 先把真实差集红出来，用差集清单作为拍板材料。

**R2-a 实测差集（2026-10-03，守卫红出，原样登记为拍板材料）**

| 角色 | DB 码集缺少的 (resource,action) 对 | 计 |
|:--|:--|:--|
| manager | ticket:assign, ticket:escalate, ticket:export, notification:read, notification:write, incident:write, dashboard:read, cmdb:read, service_catalog:read, sla:read, bpmn:read, release:read, release:write, asset:read, asset:write, license:read, license:write, group:read, group:write, org:read, org:write, project:read, project:write, application:read, application:write, ai:read | 26 |
| agent | notification:write, dashboard:read, service_catalog:read, change:write, alert:read, alerts:read, group:read, bpmn:read | 8 |
| sysadmin | `*:*` —— middleware 条目本身是字面通配符（`rbac.go:65`），DB 侧是 129 个枚举码，无法按对比较 | 表示差异，非真实缺口 |
| end_user | notification:write, dashboard:read, ai:read, ai:write, sla:read, system_config:read, org:read, department:read, cmdb:read, incident:read, change:read, problem:read, bpmn:read, release:read, asset:read, license:read | 15 |
| admin / technician | ——（硬档全绿：DB 码集是硬编码兜底的完整镜像） | 0 |

三条口径说明与附带发现：

1. **差集口径**：上表 = `middleware.RolePermissions[role]` 的 (resource,action) 对 − 该角色 DB 播种码集展开的对，衡量的是"unconfigured 兜底 vs configured 实际授权"的差异，**不是**"路由所需 vs 角色持有"的直接测量。N4 若走收权路线（选项 B/C），应以 `rbac_precheck_gen.go` 的路由预检需求为基准重新计算；本表头部"`sysadmin` 少 28 个码"的实测事实对应的是 DB 播种（129）与 catalog（160）的差，与 middleware 侧通配表示是两件事。
2. **middleware 自身词汇漂移**：agent 条目同时持有 `alert:read` 与 `alerts:read` 两种拼写，而 DB 码集两者皆无——硬编码兜底内部先漂移了。无论 N4 选哪项，这两个拼写需先收敛为一个。
3. **sysadmin 通配表示**：middleware 给 sysadmin 的是 `*:*` 字面量，DB 给的是枚举码。N5 选项 C（codegen 生成 RolePermissions）必须解决这对表达差异，本行是其首个具体样例。

### N5　`middleware.RolePermissions` 这份硬编码权威最终留不留？

它是双权威问题的根因。批次 1–5 已经做到"路由声明成为权限单一真源"（预检映射由 `cmd/authz-gen` 从声明 AST 生成）。顺着这条路，`RolePermissions` 的终局应当是：

| 选项 | 含义 |
|:--|:--|
| **A：保留为 DBOnly `unconfigured` 状态的兜底** | 现状。三态语义（`rbac.go:1054-1071`）依赖它。风险是它必须与 DB 播种**永久保持同步**，而守卫只覆盖 2/9 角色 |
| **B：降级为"仅 dev/test 可见"，生产 unconfigured 直接 fail-closed** | 消除双权威；代价是全新安装若播种失败会直接锁死，需要配套的可观测告警 |
| **C：由 `authz.Definitions()` codegen 生成 `RolePermissions`** | 单一真源 + 保留兜底；需要处理 `*:*` 通配与 129/160 枚举的表达差异 |

### N6　`src/app/agent-ops-demo/page.tsx` 留不留？

216 行、102 条长中文文案、硬编码 evidence（"checkout-api P95 延迟超过 2.5s"、"匹配 INC-2025-0817，相似度 92%"）与 guardrails（"SRE Copilot · on behalf of 刘洋"、"风险等级 L3 · 必须人工批准"、"单次授权 · 10 分钟"），整个"批准 → 执行"流程是 `useState`，"执行成功 连接池已恢复"是一个 `useMemo` 字符串。**不在菜单，但是存活的生产路由。**

| 选项 | 含义 |
|:--|:--|
| **A：删** | 最省事。风险是它可能是给 v1.7 AI 方向做的演示资产 |
| **B：留但加显式 DEMO 水印 + 从生产构建排除** | 保留演示价值，消除"访客误判产品能力"的风险 |
| **C：留并接线** | 变成真功能，工作量最大，且与 ROADMAP v1.7「AI evaluator」方向重叠，应先确认是否属于那条线 |

### N7　首次登录强制改密是否本轮做？

全仓无 `must_change_password` 类字段。R3 修完 `init_admin.sh` 之后，初始口令来源仍然需要一条"首次登录必须改"的闭环才算企业级。代价是 schema 变更 + 迁移 + 前端拦截，且要考虑存量用户（`password_changed_at` 为 NULL 时是否强制）。

| 选项 | 含义 |
|:--|:--|
| **A：本轮做** | 与 R3 同批，一次性把"初始口令"这个面关掉 |
| **B：推到初始化蓝图 Step 4** | 与身份/RBAC 收敛同批，避免本轮范围膨胀 |

---

## 4. 建议执行顺序与并行度

```
可立即并行（不需拍板）        需要拍板后才能动
─────────────────────       ─────────────────
R1  未认证提权收口            R2-c  ← 阻塞于 N4
R2-a 扩守卫（先 Skip 登记）    N5    ← 决定 R2 的终局形态
R2-b 修错误注释/补齐码
R2-d 解除播种自愈阻塞
R3  种子载体去毒
R4  门禁补口（advisory 起步）
R5  孤儿清理（复核后删）       R5 中的"应当接线"类 ← 阻塞于产品需求确认
R6 #1 #2 #3 #4 #5 #6
R7 #1 #2 #3 #4 #5 #6 #8
R8  文档回写（随各批同提交）
                              N6  agent-ops-demo 去留
                              N7  强制改密是否本轮
                              R6 #7 #8（能力开关 / 信封）需逐条判定
```

**第一动作**：R1 + R2-a + R3。三条都不需要拍板，且 R2-a 会产出 N4 的决策材料。

---

## 5. 本轮不做的事（显式列出，避免范围膨胀）

1. **不重构 `controller/` → `handlers/<domain>/` 的迁移。** AGENTS.md 说迁移是 opportunistic 的，本轮没有触发条件。
2. **不动菜单基线。** 审查报告 §9 的结论是"菜单设计是这轮里最干净的部分：97 条码内定义、权限码 100% 落在 catalog、reconcile 不覆写运维选择、且菜单明确不是授权点。**这一块不需要动。**"
3. **不动 `internal/authz/catalog.go` 的三条规则**（≥1 路由引用 / ≥1 角色绑定 / 独立资源面）。悬挂码实测为 0，这套守卫是有效的。
4. **不新增第二套审批引擎、不新增第二套权限模型。** N5 的选项 C 是收敛，不是新增。
5. **不在本轮处理审查报告 §7「静态无法确定的部分」。** 那些条目需要运行时证据，而本轮审查前提是只读、未启动服务、未连接任何数据库。要动它们必须先按 `feedback-real-database-plan-first` 单独给方案并等确认。

---

## 6. 执行时的取证纪律（来自本轮审查自身的教训）

审查报告 §6 记了 9 条"审查者差点搞错"。其中 3 条会直接影响本计划的执行方式，必须写进计划：

| 教训 | 对本计划的约束 |
|:--|:--|
| **0 命中通常是正则错了，不是事实**（catalog 位置字面量、zsh 未引号的 `--include=*.go`、Resource/Action 分开存导致字面 grep 假零） | R2/R5/R6 的"死代码""零引用"结论，**必须换第二种检索方式复核**（读 `rbac_precheck_gen.go` 而不是 grep 字符串；用 `git grep` 而不是 shell glob）。R5 的删除清单尤其如此——删错一个活文件的代价远高于漏删一个死文件 |
| **不得原样采纳子代理结论**（报告 §2 #3 曾误写 `make dev-seed-demo` 不存在，实际在 `Makefile:240-249`；另有两个路径缺 `itsm-backend/` 前缀） | 本计划 §2 R3 已显式标注不继承该错误。后续任何"某文件/target/路由不存在"的结论，**执行者必须自己重跑一次**才允许据此删除或修改 |
| **并发会话会让引用坐标失效**（本轮发生了三次：① 报告写作期间棘轮基线 50 → 25 条、行号 `:40-117` → `:59-87`；② 本文起草期间它又被拆成三个列表、主列表降到 23 条；③ 未推送提交数从 20 涨到 27） | 本计划所有 `file:line` 与计数**在动手前必须重新实测**，本文写的数字只证明"当时是这样"。特别是 R6 #8 的棘轮基线、R5 的 55 个文件清单。工作区当前有 **25 个他人在途文件**（notification 域为主，另有 `docs/api-reference.md`、`docs/testing/static-analysis-gates.md`），提交前必须 `git status` 逐文件确认归属，**不得 `git add -A`**；本地 27 个提交未推送，推送需单独授权 |

---

## 7. 计划变更协议

按 `plans/production-data-initialization-blueprint.md:§8` 与 `plans/README.md` 的既有约定：

- 本文的批次状态变化（立项 / 进行中 / 已落地）必须同步回写 §2 对应行，并附提交号与测试证据。
- 本文与既有主线冲突时，**以既有主线为准**，本文标记 superseded；本文不覆盖任何主线的退出标准。
- 计划中的 checkbox、退出标准或"完成报告"不自动等于功能已发布；判断实现状态必须回到当前源码、迁移、测试和运行时证据（`plans/README.md` 原文约束）。
