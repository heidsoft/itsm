# 菜单 IA 与权限门控整改复盘（2026-10-26）

> Status: current

> **时间**：2026-10-26
> **范围**：`output/prod-design-improvements-2026-10-26.md` §四 遗留 6 项 + 前端按钮级门控
> **验证方式**：生产栈真实浏览器（agent-browser）+ API/DB 交叉验证 + 临时角色账号
> **产出**：v1.6.35 → v1.6.36 → v1.6.37 三次部署，全部 healthcheck 通过

---

## TL;DR

六个「等产品拍板」的遗留项在一轮里全部定了方向并落地，`pkg/menubaseline` 单测、`authz` 契约、
seeder 守卫、预检表生成、tsc 全绿——**但真实浏览器一跑，仍然抓到 4 个单测覆盖不到的缺陷**：

| # | 缺陷 | 性质 | 为什么单测没抓到 |
|---|------|------|-----------------|
| 1 | `AdminRouteGuard` 只查 admin 桶，14 个 `/admin/*` 配置页对非管理员误判 403 | 真实功能缺陷 | 测试数据把目标路径放在 admin 桶，从未构造「路径在主桶父级下」 |
| 2 | `AssetList` 空态写死 `showAction: true`，只读角色仍看到「新增资产」 | 门控被绕过 | 门控写在页面 `primaryAction`，空态是第二个入口 |
| 3 | `ServiceRequestList` 同型缺陷 | 门控被绕过 | 同上 |
| 4 | 菜单改「客户与合同」「我的待办」，页面 H1 仍是「邮件智能报障」「审批中心」 | 一致性缺陷 | 改名只动了菜单单一源，页面标题是第二个真源 |

结论：**数据层一致 ≠ 用户看到的一致**。菜单/权限这类「声明式配置 + 渲染期过滤」的系统，
必须在真实浏览器里按角色走一遍，否则校验的是配置而不是产品。

---

## 一、测试方法

生产栈（`docker-compose.prod.yml`，nginx + backend + frontend + postgres）上做四层交叉验证：

| 层 | 手段 | 能证明什么 | 不能证明什么 |
|---|------|-----------|-------------|
| L1 单元/契约 | `go test`（menubaseline/authz/seeder/router/middleware）+ `jest` | 基线与角色绑定自洽 | 用户实际看到什么 |
| L2 数据层 | `psql` 查 `menus` / `role_permissions` | 播种是否真的落库 | 菜单过滤后的形状 |
| L3 接口层 | `curl /auth/menus`、`/reports/*`、`/system/vector-store` | 服务端过滤结果、接口鉴权 | 前端渲染与路由守卫 |
| L4 浏览器 | `agent-browser` 按角色登录逐页走查 | 渲染、按钮可见性、守卫、标题 | — |

**关键动作：造角色。** 生产库原本只有 1 个 `super_admin`，无法验证「按角色可见性」。
经 `POST /api/v1/users` 建两个临时账号（`qa_enduser` / `qa_itadmin`），走完再删。
没有这一步，第 1、2、3、4 号缺陷全都测不出来——`super_admin` 对每个页面都有权限，
是所有权限缺陷的「完美掩盖器」。

> 门禁备注：docs-gate C.4 会把上句的「完美」报为「无 revision 锚点的发布声明」，
> 属修辞误报（advisory 级，不阻断）——此处的「完美掩盖器」是描述性说法，
> 不是对某个 release/commit 的断言。

> 附带发现：写操作接口要求 CSRF Double-Submit（`csrf_token` cookie + `X-CSRF-Token` 头），
> 先 `GET /api/v1/csrf-token` 取 token 再带头发请求。脚本化验证时容易只踩「缺少认证token」，
> 实际是 CSRF 层拦下的。

---

## 二、测试矩阵与结果

### 2.1 菜单 IA（L3 + L4）

| 用例 | 期望 | 实测 |
|------|------|------|
| 顶级菜单名 | 「客户与合同」取代「邮件报障」 | ✅ 侧边栏 + API 双证 |
| 值班排班归属 | 父级 `/incidents` | ✅ 展开「事件管理」见「新建事件 / 值班排班」 |
| 审批入口唯一 | 只有 `/approvals`（我的待办），无「待我审批」「审批中心」 | ✅ 库内 `/approvals/pending` 已删，侧边栏仅「我的待办」 |
| 审批链命名 | 「审批链配置」 | ✅ |
| 系统管理码 | `system_config:read/write` | ✅ 库内 `/admin` = `system_config:read`，`/admin/menus` = `system_config:write` |
| 报表中心码 | `report:read` | ✅ 菜单与 `/reports/*` 同码 |
| 菜单总数 | 104 → 103 | ✅ |

### 2.2 角色可见性（L4，真实登录）

| 角色 | 侧边栏顶级项 | 报表中心 | 系统管理 | 我的待办 |
|------|------------|---------|---------|---------|
| `super_admin` | 29 | ✅ | ✅ | ✅ |
| `it_admin` | 23 | ✅ | ✅（6 个子项，见下） | ✅ |
| `end_user` | 11 | ❌ | ❌ | ✅ |

**「系统管理可见但子项被裁剪」不是漏洞，是 `backfillParentMenus` 的设计行为**：
`it_admin` 不持 `system_config:read`（分组自身码），但持有 `user:read` / `group:read` /
`department:read` / `team:read` / `bpmn:read`，这些子页需要父级挂载点，于是分组被回填。
实测它只看到 6 个可达子项，`权限管理`（role:write）、`菜单管理`（system_config:write）、
`租户管理`（tenant:read）均被过滤——**分组可见 ≠ 越权**。

> 这一条差点被我误判成缺陷：只看 DB 的 `role_permissions` 会得出「it_admin 不该看到系统管理」，
> 但正确的判断依据是「过滤后还剩什么」。

### 2.3 页面可达性（L4）

| 页面 | admin | it_admin | 判定 |
|------|-------|----------|------|
| `/approvals` | ✅ | ✅ | 不再是 redirect 壳 |
| `/email-intake` | ✅ | — | — |
| `/reports` | ✅ | ✅ | 菜单与接口同码 |
| `/admin/vector-store` | ✅ | — | 改码前是「菜单可见、接口 403」 |
| `/admin/sla-templates` | ✅ | ✅ | **守卫双桶修复后可达**（修复前 it_admin 403） |
| `/admin/cmdb-types` | ✅ | ✅ | — |
| `/admin/users` | ✅ | ✅ | — |
| `/admin/approval-chains` | ✅ | ❌ 403 | 正确：it_admin 无 `approval:write` |
| `/admin/menus` | ✅ | ❌ 403 | 正确：无 `system_config:write` |

### 2.4 按钮级门控（L4）

| 页面 | 按钮 | admin | it_admin | end_user | 期望 |
|------|------|-------|----------|----------|------|
| `/tickets` | 新建工单 | ✅ | ✅ | ✅ | 三者均持 `ticket:write`/`ticket:create` ✅ |
| `/incidents` | 新建事件 | ✅ | ✅ | ❌ | ✅ |
| `/problems` | 新建问题 | ✅ | ✅ | ❌ | ✅ |
| `/changes` | 新建变更 | ✅ | ❌ | ❌ | ✅（it_admin 只有 `change:read`） |
| `/knowledge` | 新建文章 | ✅ | ❌ | ❌ | ✅ |
| `/assets` | 新增资产 | ✅ | ⚠️ 修复前可见 | ⚠️ 修复前可见 | **缺陷 #2** |
| `/cmdb/cis` | 录入资产 | ✅ | ✅ | ❌ | ✅ |

### 2.5 后端兜底鉴权（L3）

`end_user` 直接打接口全部 403（`code 2003`）：`/reports/tickets`、`/system/vector-store`、
`/users`、`/audit-logs`。**菜单隐藏不是授权**——即使绕过前端直接访问 URL，
数据接口仍逐请求把关。`/reports` 页面 `end_user` 能打开（无路由守卫），但页面数据全 403，
属「丑但安全」，列入遗留。

---

## 三、四个缺陷的根因与修法

### 缺陷 1：`AdminRouteGuard` 只查 admin 桶

菜单桶按**根祖先**分桶，不按自身 path 前缀：`/admin/cmdb-types` 挂在「配置管理」、
`/admin/sla-templates` 挂在「SLA 管理」、`/admin/approval-chains` 挂在「工作流」，
这些都进 main 桶。守卫却只 `menuContainsPath(menus.admin, pathname)`，
于是持 `sla:write` 的 `it_admin` 点「SLA 模板」时，路径不在他可见的 admin 桶里 → 403。

此前用 `ADMIN_ROUTE_PERMISSIONS` 逐页打补丁，只补了 `cmdb-types` 一个，其余 13 个漏着。

**修法**：两桶任一命中即放行。补 2 例回归测试（主桶父级下放行、两桶皆无仍 403）。

### 缺陷 2/3：空态入口绕过门控

列表页的「新建」有两个入口：页面 `primaryAction`（已门控）和表格 `emptyText` 里的
`LoadingEmptyError.empty.showAction`（写死 `true`）。`end_user` 在 `/assets` 页头看不到按钮，
空态里却能点到「新增资产」，点进去 POST 403。

**修法**：`showAction` 改为由 `usePermissions` 推导（`AssetList`、`ServiceRequestList`）。
新增静态守卫 `src/components/ui/__tests__/empty-action-gating.test.ts`：
全源码禁止字面量 `showAction: true`，并做阳性对照防止守卫被清空后静默通过。

### 缺陷 4：菜单名与页面 H1 不一致

改名只动了 `pkg/menubaseline`（菜单单一源），页面标题是**第二个真源**：
`/approvals` 的 `approvalCenter.title` 仍是「审批中心」，`/email-intake` 的 H1 仍是「邮件智能报障」。
用户点「我的待办」进来看到「审批中心」，导航与内容对不上。

**修法**：页面标题对齐菜单名（`我的待办` / `My To-dos`、`客户与合同`），
原语义降级到副标题，信息不丢。

---

## 四、修复后复测（v1.6.37，生产栈）

四个缺陷修完重新构建部署，用同一套走查脚本回归，并做「反向验证」确认没有过度收紧：

| 用例 | 修复前 | 修复后（v1.6.37） |
|------|--------|------------------|
| end_user `/assets` 空态 | 有「新增资产」（点进去 403） | **无** ✅ |
| admin `/assets` | 有 | 有 ✅（未过度收紧） |
| it_admin `/admin/sla-templates` | 403 | **200 正常渲染** ✅ |
| it_admin `/admin/menus` | 403 | 403 ✅（确实无 `system_config:write`，不该放开） |
| `/approvals` 页面标题 | 审批中心 | **我的待办** ✅（面包屑同步） |
| `/email-intake` 页面标题 | 邮件智能报障 | **客户与合同** ✅（副标题保留 AI 邮件报障语义） |
| admin `/service-requests` 空态 | 有「新建服务请求」 | 有 ✅ |

截图留档：`output/qa-2026-10-26/`（侧边栏、我的待办、客户与合同）。

**反向验证是这套走查里最容易漏的一步。** 门控只往「收紧」方向改时，如果不同时验证
「该看到的还看得到」，很容易把管理员也一起关在门外——`AdminRouteGuard` 那次改动就同时
断言了「两桶皆无 → 仍 403」，避免把守卫改成「永远放行」。

---

## 五、沉淀：6 条经验

1. **权限类改动必须造「低权角色」再验。** 只有一个 `super_admin` 的环境，
   所有「看不到」类缺陷都不可见。测试脚本里应把「建临时角色 → 走查 → 删除」做成标准步骤。

2. **门控要按「入口种类」清点，不能按页面清点。** 一个列表页至少有三类写入口：
   页头 `primaryAction`、空态 `actionText`、行内操作列。只改页头等于只堵了一个洞。
   → 已落守卫：`empty-action-gating.test.ts`。

3. **菜单桶不是「路径前缀分组」。** 分桶依据是根祖先，子项 path 与所在桶可以完全无关。
   任何「按 path 前缀判断在哪个桶」的假设都会错。→ 已落守卫：双桶断言。

4. **改名是双源操作。** 菜单名改了，页面标题、面包屑、i18n 文案、文档表格都要跟。
   `menubaseline` 只是导航单一源，不是「产品名词单一源」。

5. **「分组可见但子项被裁」是正常态，别当漏洞修。** `backfillParentMenus` 为了挂载点
   回填父级；判断是否越权要看**过滤后剩下的叶子**，不是父级是否存在。

6. **Docker 编译报「ent 文件语法错误」先怀疑构建缓存。** 本次后端构建出现上百条
   `ent/alert/alert.go:14:2: "context" imported and not used` 之类，而本地
   `go build`、`GOOS=linux` 交叉编译、镜像内文件 md5 全部正常。`docker builder prune -af`
   清掉 24GB 缓存后一次通过——**缓存层不一致，不是代码问题**。
   判断方法：`CACHED` 的层里如果编译错误指向的文件内容与实际不符，就是缓存脏了。

---

## 六、遗留

> 2026-10-27 收口：下表中前两项已闭环（`83f6f1930` / `62087dc9b`），QA 账号一项已判定不处理。

| 项 | 现状 | 处置 |
|---|------|------|
| ~~`/reports` 无路由守卫~~ | ✅ 已闭环：`AdminRouteGuard` 增加 `MENU_GATED_ROOTS`，`/reports` 及子页按根路径判定菜单可见性；补 4 例回归（含「end_user 直连 URL 得 403」） | `83f6f1930` |
| ~~`precheck_fallback.go` 的 `POST /api/v1/reports*` → `dashboard:admin`~~ | ✅ 已闭环：该条与另外 54 条同为死条目，整批删除；新增 `TestPrecheckFallbackPoliciesMatchRegisteredRoutes` 防复发 | `62087dc9b` |
| MSP 五角色 | 已播种（8–13 条码），但生产未开 MSP 模块 | 开 MSP 部署模式后按 provider 角色补一轮走查 |
| 空态门控单测 | 只有静态守卫（禁字面量 `showAction: true`）+ 浏览器走查，无组件级单测 | 若要重构 `LoadingEmptyError`，先补组件测试 |
| 「客户与合同」页仍有「值班配置」Tab | 与「值班排班移入事件管理」的决策擦边：Tab 只放两个 PoC 按钮（新增排班/新增班次），真正可用的排班管理在 `/email-intake/on-call` | 需产品确认：是保留为「合同驱动的排班配置」入口，还是直接删掉消除重复 |
| 临时 QA 账号 | `qa_enduser`(22) / `qa_itadmin`(23) 软删（`active=false`），行仍在 `users` 表 | **判定不硬删**：全库扫过 19 个 FK + 全部 50 个用户引用列，仅 `user_roles`(2 行) 与 `audit_logs`(7 行) 命中；`audit_logs` 既无 FK 也无 username 冗余列，硬删会让这 7 条审计记录指向不存在的用户。软删行本身就是「账号已停用」的正确表达，保留 |

---

## 七、复现步骤

```bash
# 1. 生产栈就位
docker compose -f docker-compose.prod.yml --env-file .env.prod ps

# 2. 取 CSRF + 建临时角色
curl -s -c cj -b cj http://localhost:8090/api/v1/csrf-token
curl -s -b cj -X POST http://localhost:8090/api/v1/users \
  -H 'Content-Type: application/json' -H "X-CSRF-Token: $CSRF" \
  -d '{"username":"qa_enduser","email":"qa_enduser@example.com","name":"QA","password":"QaTemp#2026x9","role":"end_user"}'

# 3. 浏览器按角色走查（关键页）
agent-browser open http://localhost/login      # 登录 qa_enduser
agent-browser snapshot | grep menuitem         # 侧边栏
agent-browser open http://localhost/assets     # 空态是否仍有「新增资产」
agent-browser open http://localhost/admin/sla-templates   # 守卫是否误挡

# 4. 收尾：删除临时账号
```
