# 菜单与路由一致性审计

> 基准 commit：`b3bfd748604041d5082a320ed6d5c447221ac76a`

## 1. 生产调用链（已确认接线）

`GET /api/v1/auth/menus`（`router/common_system_routes.go:37`）→ `RBACHandler.GetUserMenus` → `menuServiceImpl.GetUserMenus` → `filterMenusByPermission` / `backfillParentMenus` / `buildMenuTree` / `filterMSPMenus`（`handlers/rbac/menu_service_impl.go:299-335,490-577,664-763`）→ `useUserMenusQuery`（`src/lib/hooks/useUserMenusQuery.ts:33-41`）→ `Sidebar.tsx:109-186`。

播种：`pkg/seeder/seeder.go:775,1930-2017`（平台租户）；新租户 `tenant_provisioner.go:27-99`。
重命名/改父级的修复入口 `InitMenusFromBaseline`（`menu_service_impl.go:828-956`）**只有** `POST /api/v1/menus/init` 一个调用方（`handler.go:655`），没有启动期调用者。

基线总数 **101**（29 一级 + 72 二级），与 `initialization_verify_cold_test.go:308` 棘轮一致。

## 2. 结构检查结果（全部通过）

- 重复路径 0、重复名称 0；101 条路径全部有对应 `page.tsx`。
- 孤儿父节点 0、层级 >2 为 0；不同名称指向同一 URL 为 0（运行时还有 `Sidebar.tsx:145-160` 去重）。
- 没有重复的 AI 分类入口：三个 AI 能力页已归到业务父节点，`menu-config.ts:48-51` 用 `ai` capability key。

## 3. 发现项

```yaml
id: AUDIT-010
priority: P1
category: menu/permission
status: confirmed
title: /tickets/types 菜单权限码与列表 API 权限码不一致
file: pkg/menubaseline/baseline.go:86
evidence: 菜单 ticket_type:read；router/ticket_type_routes.go:12 与 rbac_precheck_gen.go:458 为 ticket:read；roles.go:51-65 的 service_catalog_admin 有前者无后者
reproduction: 以 service_catalog_admin 登录，侧栏出现"工单类型"，GET /api/v1/ticket-types 返回 403
impact: 负责工单类型/模板/分类配置的角色无法打开自己的配置页
root_cause: 路由声明未随码空间拆分更新（历史只修了 /ticket-type-presets）
fix: 把 GET /ticket-types 与 /ticket-types/:id 改为 ticket_type:read，重生成 rbac_precheck_gen.go；不要给角色加 ticket:read（会扩大工单访问面）
acceptance: 角色无 ticket:read 但有 ticket_type:read 时 GET 200；无权限仍 403；baseline_guard_test 增加菜单码=路由码断言
```

```yaml
id: AUDIT-011
priority: P1
category: menu/permission
status: confirmed
title: /service-requests 与 /my-requests 用 ticket:read，数据源要求 service_request:read
file: pkg/menubaseline/baseline.go:28,30
evidence: rbac_precheck_gen.go:407,413 为 service_request:read；service-requests/page.tsx:47-48、my-requests/page.tsx:150 调用对应 API；roles.go:51-65
impact: 服务请求归口角色有全部 API 权限但没有导航入口
fix: 基线码改为 service_request:read（upsertMenu 会更新 DB 行的 permission_code）
acceptance: service_catalog_admin 收到两个菜单，problem_manager（无 service_request:read）收不到
```

```yaml
id: AUDIT-012
priority: P1
category: menu/permission
status: confirmed
title: /workflow/audit 菜单用 audit:read，唯一数据源用 bpmn:read
file: pkg/menubaseline/baseline.go:201
evidence: rbac_precheck_gen.go:141 为 bpmn:read；workflow/audit/page.tsx:68 → bpmn-dashboard-api.ts:179；同族 workflow 菜单 2026-11-17 已迁移 bpmn:*，此条遗漏
impact: agent/technician/it_admin 能调 API 却在侧栏看不到操作日志；而所有 audit:read 持有者本身也有 bpmn:read，无补偿安全收益
fix: 菜单码改为 bpmn:read（不影响 /api/v1/audit-logs 的 audit:read 门禁）
```

```yaml
id: AUDIT-013
priority: P1
category: menu/migration
status: partial
title: 存量租户的 AI 能力归位与"超时升级"重命名依赖管理员手动调用
file: migrations/20261121_retire_ai_center_menu.sql:11-13；migrations/20261121_merge_escalation_menus.sql:4-5
evidence: 两个迁移只按精确路径禁用退役行；注释声称"启动期 InitMenusFromBaseline"，但全仓只有 handler.go:655 一个调用方；升级租户的三个 AI 子页因父行被禁用且未改父级而从菜单消失
impact: 升级租户丢失三处真实 AI 能力入口，命名与基线不一致
fix: 在两个迁移中按 (tenant_id, path) 用关联子查询修正 name/parent_id，保持幂等、无硬编码 id；或实现显式租户 reconcile 步骤
acceptance: 两个租户的升级库执行 up SQL 后，子页 parent 指向业务父页、名称与基线一致；重复执行无变化；_down 可回滚
```

```yaml
id: AUDIT-014
priority: P2
category: menu/retirement
status: confirmed
title: 遗留 /ai/chat/list 别名单可让已退役入口复活
file: migrations/20261121_retire_ai_center_menu.sql:18-21；seeder.go:2064；Sidebar.tsx:69
evidence: 迁移只禁用精确 /ai/chat；前端会把 /ai/chat/list 归一化为 /ai/chat；seed 的路径修复仍可能把Enabled 行改名到退役路径
fix: 迁移 WHERE 加 OR path='/ai/chat/list'；seed 路径修复改为删除而非改名
acceptance: 两租户各插入 /ai/chat 与 /ai/chat/list 行，up SQL 后两行均 disabled，行数不变
```

```yaml
id: AUDIT-015
priority: P1
category: docs/drift
status: confirmed
title: docs/product/frontend-menu-and-features.md 停留在 102 条菜单与已退役入口
file: docs/product/frontend-menu-and-features.md:3,16-17,78,182,264,595
evidence: 文档写 102 条、/ai/chat 一级、/approvals/pending、service:read/workflow:read/system:write、/templates→template；HEAD 实际 101 条、AI 中心退役、/approvals、service_catalog:read/bpmn:read/system_config:read、/tickets/templates
fix: 从 MenuDefinitions() 与 capabilityPathRules 重新生成表格，更新 Last synced 与退役标记；加 docs-gate 禁止退役字符串
```

```yaml
id: AUDIT-016
priority: P2
category: test/fixture
status: confirmed
title: cmdb-commercial-menu.test.ts 自喂 fixture 与基线矛盾
file: src/components/layout/sidebar/__tests__/cmdb-commercial-menu.test.ts:27-66
evidence: 测试 mock 仅 4 个 CMDB 子菜单并断言云账号/云资源/对账不存在；基线 baseline.go:146-160 已包含它们；测试恒绿
fix: 改为读取后端基线（参考 icon-map-parity.test.ts）或删除过时的精确列表断言
```

```yaml
id: AUDIT-017
priority: P2
category: test/guard
status: confirmed
title: 缺少"菜单路径→页面存在"与"菜单码→页面主 API 码"的 CI 守卫
evidence: 现有守卫只覆盖码存在性、命名、图标、URL 形状；api-contract 只校验 URL；scripts/acceptance/menu-feature-audit.py 为人工且用 admin 登录，看不到 AUDIT-010/011/012
fix: 新增前端源码扫描测试（枚举 page.tsx 对基线路径）+ Go 表驱动测试（13 条抽样菜单的主 API 码）
acceptance: 删除一个 page.tsx 或翻转一条期望码后新守卫必须变红
```

## 4. 已确认的历史整改（不要重复）

工单仪表盘退役（`20261101_*` + 测试）、AI 中心退役与三能力归位、升级矩阵合并/「超时升级」命名、审批中心去重、`bpmn:*` 对齐、系统配置单轨码、图标一致性、父子循环与 `(tenant,path)` 重复守卫，均已在 HEAD 落地并有守卫。本次发现的 AUDIT-013/014 是这些整改的**存量租户收尾缺口**，不是重做。
