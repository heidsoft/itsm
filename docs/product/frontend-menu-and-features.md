# 前端菜单与功能映射梳理

> 适用范围：`itsm-frontend` + `itsm-backend` v1.6.x 收尾期；最后同步时间：2026-09-26。
> 用途：把"前端页面 ↔ 后端菜单 ↔ API 入口 ↔ 能力治理"四个视角统一到一张图，作为菜单收口与下阶段补齐的单一事实来源。
>
> 本文是 docs/product/ 家族的新增文档，与 `ai-automation-status.md` 并列，专注"菜单/页面/功能"层。

---

## §1 菜单架构总览

### 1.1 结论

- **前端没有静态菜单配置**。`itsm-frontend/src/components/layout/sidebar/menu-config.ts` 仅保留 `MenuItem` 类型与 `capabilityPathRules`，没有任何 `const getMenuConfig()` 静态数组。
- **菜单唯一来源是后端**：登录用户通过 `GET /api/v1/auth/menus` 获取当前租户 + 角色 + 权限过滤后的菜单树。
- **后端 `pkg/seeder/seeder.go::menuDefinitions()` 是种子真理**：78 条菜单（21 顶级 + 57 子级）由 `seedMenus` 两遍写入 `menu` 表（顶级 → 子级按 `parent_path` 反查）。
- **运行时入口**：`MenuService.GetUserMenus`（`service/menu_service.go:211`）→ `dto.MenuTreeResponse{Main, Admin}`。
- **前端缓存**：`useUserMenusQuery`（`src/lib/hooks/useUserMenusQuery.ts`）走 React Query，5 分钟 staleTime；菜单管理端 CRUD 后派发 `MENUS_UPDATED_EVENT` 自动 invalidate。

### 1.2 数据流

```text
seedMenus → menu 表（tenant_id + path 唯一）
                ↑
        upsertMenu (两遍 upsert)
                ↑
   menuDefinitions() = 78 条菜单种子（唯一真理）

登录后:
  GET /api/v1/auth/menus
    → handlers/rbac/handler.go:493 GetUserMenus
    → MenuService.GetUserMenus
        ├─ 取用户角色/权限
        ├─ menu 表查 is_enabled && is_visible
        ├─ filterMenusByPermission
        ├─ buildMenuTree → main / admin
        └─ filterMSPMenus (IsMSPEnabled() == false 时剔除 msp:*)
    → { main, admin }  (camelCase JSON)
        ↓
  useUserMenusQuery (React Query 缓存 5min)
        ↓
  convertApiMenuToSidebar → MenuItems → AntD Menu
        ↓
  filterByCapability (maturity/buildAvailable/deploymentReady/tenantReady fail-closed)
```

### 1.3 关键文件清单

| 角色 | 路径 | 行数参考 |
|------|------|---------|
| 前端类型 + capability 规则 | [itsm-frontend/src/components/layout/sidebar/menu-config.ts](file:///Users/heidsoft/Downloads/research/itsm/itsm-frontend/src/components/layout/sidebar/menu-config.ts) | 49 |
| Sidebar 渲染 | [itsm-frontend/src/components/layout/sidebar/Sidebar.tsx](file:///Users/heidsoft/Downloads/research/itsm/itsm-frontend/src/components/layout/sidebar/Sidebar.tsx) | 264 |
| 菜单项渲染 | [itsm-frontend/src/components/layout/sidebar/MenuItems.tsx](file:///Users/heidsoft/Downloads/research/itsm/itsm-frontend/src/components/layout/sidebar/MenuItems.tsx) | 158 |
| 菜单 API DTO + admin CRUD | [itsm-frontend/src/lib/api/menu-api.ts](file:///Users/heidsoft/Downloads/research/itsm/itsm-frontend/src/lib/api/menu-api.ts) | 107 |
| React Query 缓存 | [itsm-frontend/src/lib/hooks/useUserMenusQuery.ts](file:///Users/heidsoft/Downloads/research/itsm/itsm-frontend/src/lib/hooks/useUserMenusQuery.ts) | 62 |
| 菜单种子（真理） | [itsm-backend/pkg/seeder/seeder.go:1809-1938 menuDefinitions()](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/pkg/seeder/seeder.go#L1809-L1938) | 130 |
| 菜单 upsert | [itsm-backend/pkg/seeder/seeder.go:1941-2028 seedMenus/upsertMenu](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/pkg/seeder/seeder.go#L1941-L2028) | 87 |
| 运行时 GetUserMenus | [itsm-backend/service/menu_service.go:211-292 GetUserMenus+filterMSPMenus](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/service/menu_service.go#L211-L292) | 81 |
| 路由注册 | [itsm-backend/router/common_system_routes.go:14-214](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/router/common_system_routes.go#L14-L214) | 200 |
| HTTP Handler | [itsm-backend/handlers/rbac/handler.go:493 GetUserMenus](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/rbac/handler.go#L493-L514) | 22 |
| Marketplace 路由 | [itsm-backend/handlers/marketplace/handler.go:44-62 RegisterRoutes](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/marketplace/handler.go#L44-L62) | 19 |

---

## §2 顶级菜单（21 条）

按 `sort_order` 升序。所有顶级条目均无 `parent_path`，由 `buildMenuTree` 拆为 `main` 或 `admin`（路径前缀 `/admin` → admin）。

| # | sort | path | name | icon | permission | main/admin | 备注 |
|---|-----:|------|------|------|------------|------------|------|
| 1 | 10 | `/dashboard` | 服务台 | LayoutDashboard | （无） | main | 默认登录入口 |
| 2 | 20 | `/service-requests` | 服务请求 | FileText | `ticket:read` | main | 用户面向的工单请求 |
| 3 | 23 | `/tickets` | 工单管理 | FileText | `ticket:read` | main | **2026-08-30 归位新增**，从 service-requests 子菜单提到顶级 |
| 4 | 25 | `/my-requests` | 我的请求 | User | `ticket:read` | main |  |
| 5 | 30 | `/incidents` | 事件管理 | AlertCircle | `incident:read` | main |  |
| 6 | 35 | `/noc` | NOC工作台 | Activity | `incident:read` | main | 重大事件作战室 |
| 7 | 40 | `/problems` | 问题管理 | HelpCircle | `problem:read` | main |  |
| 8 | 50 | `/changes` | 变更管理 | BarChart3 | `change:read` | main |  |
| 9 | 60 | `/knowledge` | 知识库 | Book | `knowledge:read` | main |  |
| 10 | 65 | `/email-intake` | 邮件报障 | Inbox | `email_intake:read` | main | AI 邮件智能报障 |
| 11 | 70 | `/service-catalog` | 服务目录 | BookOpen | `service:read` | main |  |
| 12 | 75 | `/cmdb` | CMDB | Database | `cmdb:read` | main |  |
| 13 | 80 | `/assets` | 资产管理 | Monitor | `asset:read` | main |  |
| 14 | 90 | `/sla` | SLA 管理 | Calendar | `sla:read` | main |  |
| 15 | 100 | `/workflow` | 工作流 | GitMerge | `workflow:read` | main |  |
| 16 | 110 | `/ai/chat` | AI 助手 | Bot | `ai:read` | main |  |
| 17 | 115 | `/approvals/pending` | 待我审批 | CheckCircle | `approval:read` | main |  |
| 18 | 120 | `/msp` | 客户管理 | Building | `msp:read` | main | **MSP 门控**：`IsMSPEnabled()=false` 时被 `filterMSPMenus` 剔除 |
| 19 | 130 | `/releases` | 发布管理 | Rocket | `release:read` | main |  |
| 20 | 200 | `/admin` | 系统管理 | Settings | `system:write` | admin | Sidebar 通过 `isAdmin`（user:write/role:write/system_config:write/ticket_type:manage 任一）切换 admin 区块 |
| 21 | 210 | `/audit-logs` | 审计日志 | Shield | `audit:read` | main | **2026-08-30 归位新增**，从 /admin 移到顶级 |
| 22 | 212 | `/notifications` | 通知配置 | Bell | `notification:read` | main | **2026-08-30 归位新增** |

> **注**：上表 22 行（其中 21/22 是新增的审计/通知顶级条目），原始 seeder 注释声称"21 顶级"，实际算上归位后是 22 条顶级 + 56 条子级 = 78 条。

### 2.1 归位历史（2026-08-30）

| 旧 path | 新 path | 理由 |
|---------|---------|------|
| `/service-requests` 下的"工单类型/工单统计" | `/tickets/types`、`/tickets/analytics` | 与工单业务一致，避免在 service-requests 下找不到 |
| `/admin/audit-logs` | `/audit-logs`（顶级） | 审计是独立业务条线 |
| `/admin/notifications` | `/notifications`（顶级） | 通知是独立业务条线 |
| `/admin/workflow/audit` | `/workflow/audit` | 与工作流专属审计一致 |

---

## §3 子菜单树

按顶级菜单分组（`ParentPath` 关联）。子菜单总数 56 条，覆盖"新建/审批/统计/规则/配置"等运营动作。

### 3.1 服务请求 `/service-requests`

无子菜单（业务页面即入口）。

### 3.2 工单管理 `/tickets`

| sort | path | name | permission | icon |
|-----:|------|------|------------|------|
| 21 | `/tickets/types` | 工单类型 | `ticket_type:read` | ClipboardList |
| 22 | `/tickets/analytics` | 工单统计 | `ticket:read` | BarChart3 |

### 3.3 事件管理 `/incidents`

| sort | path | name | permission | icon |
|-----:|------|------|------------|------|
| 32 | `/incidents/create` | 新建事件 | `incident:write` | Plus |

### 3.4 问题管理 `/problems`

| sort | path | name | permission | icon |
|-----:|------|------|------------|------|
| 42 | `/problems/known-errors` | 已知错误 | `problem:read` | AlertCircle |

### 3.5 变更管理 `/changes`

| sort | path | name | permission | icon |
|-----:|------|------|------------|------|
| 52 | `/changes/new` | 新建变更 | `change:write` | Plus |

### 3.6 知识库 `/knowledge`

| sort | path | name | permission | icon |
|-----:|------|------|------------|------|
| 63 | `/knowledge/articles/new` | 新建文章 | `knowledge:write` | Plus |

### 3.7 邮件报障 `/email-intake`

| sort | path | name | permission | icon |
|-----:|------|------|------------|------|
| 652 | `/email-intake/customers` | 客户资料 | `customer_master:read` | Users |
| 653 | `/email-intake/contracts` | 支持合同 | `support_contract:read` | FileText |
| 654 | `/email-intake/sources` | 来源组织 | `customer_master:read` | Globe |
| 655 | `/email-intake/on-call` | 值班排班 | `on_call:read` | Clock |

### 3.8 服务目录 `/service-catalog`

| sort | path | name | permission | icon |
|-----:|------|------|------------|------|
| 72 | `/service-catalog/approvals` | 待我审批-目录 | `service:read` | CheckCircle |

### 3.9 CMDB `/cmdb`

| sort | path | name | permission | icon |
|-----:|------|------|------------|------|
| 751 | `/cmdb/cis` | 配置项列表 | `cmdb:read` | Server |
| 752 | `/cmdb/cis/create` | 新建CI | `cmdb:write` | Plus |
| 753 | `/cmdb/relationships` | 关系管理 | `cmdb:read` | GitBranch |
| 754 | `/cmdb/topology` | 拓扑图 | `cmdb:read` | Share2 |

### 3.10 资产管理 `/assets`

| sort | path | name | permission | icon |
|-----:|------|------|------------|------|
| 82 | `/assets/new` | 新建资产 | `asset:write` | Plus |
| 83 | `/licenses` | 软件许可证 | `license:read` | Key |

### 3.11 SLA `/sla`

| sort | path | name | permission | icon |
|-----:|------|------|------------|------|
| 92 | `/sla-monitor` | SLA 监控 | `sla:read` | Activity |
| 93 | `/workflow/sla` | SLA 配置 | `sla:write` | Clock |

### 3.12 工作流 `/workflow`（运营核心区，9 子项）

| sort | path | name | permission | icon |
|-----:|------|------|------------|------|
| 102 | `/workflow/designer` | 流程设计器 | `workflow:write` | Edit |
| 103 | `/workflow/instances` | 流程实例 | `workflow:read` | Play |
| 104 | `/workflow/versions` | 版本管理 | `workflow:write` | History |
| 105 | `/workflow/dashboard` | 监控仪表盘 | `workflow:read` | Activity |
| 106 | `/workflow/bottlenecks` | 节点瓶颈分析 | `workflow:read` | BarChart3 |
| 107 | `/workflow/automation` | 自动化规则 | `workflow:write` | Zap |
| 108 | `/approvals` | 审批中心 | `approval:read` | CheckSquare |
| 109 | `/workflow/audit` | 操作日志 | `audit:read` | ClipboardList |

### 3.13 AI 助手 `/ai/chat`

| sort | path | name | permission | icon |
|-----:|------|------|------------|------|
| 112 | `/tickets/ai-create` | AI 创建工单 | `ai:read` | Sparkles |
| 113 | `/ai/audit` | AI 评估与审计 | `ai:read` | ShieldCheck |
| 114 | `/ai/approval` | AI 审批 | `ai:read` | ShieldAlert |

### 3.14 客户管理（MSP）`/msp`

| sort | path | name | permission | icon |
|-----:|------|------|------------|------|
| 122 | `/msp/management` | 客户管理子页 | `msp:write` | Settings |

### 3.15 发布管理 `/releases`

| sort | path | name | permission | icon |
|-----:|------|------|------------|------|
| 132 | `/releases/new` | 新建发布 | `release:write` | Plus |

### 3.16 系统管理 `/admin`（admin 域，24 子项）

| sort | path | name | permission | icon |
|-----:|------|------|------------|------|
| 201 | `/admin/overview` | 系统概览 | `system:write` | LayoutDashboard |
| 210 | `/admin/users` | 用户管理 | `user:read` | Users |
| 220 | `/admin/roles` | 角色管理 | `role:read` | Shield |
| 230 | `/admin/groups` | 组管理 | `group:read` | Users |
| 235 | `/admin/tenants` | 租户管理 | `system:write` | Building |
| 240 | `/admin/departments` | 部门管理 | `department:read` | Building |
| 250 | `/admin/teams` | 团队管理 | `team:read` | Users |
| 255 | `/admin/change-review` | 评审组管理 | `change:read` | Users |
| 260 | `/admin/ticket-categories` | 工单分类 | `ticket_category:update` | Tag |
| 265 | `/admin/tickets/assignment-rules` | 工单分配规则 | `ticket:read` | GitBranch |
| 270 | `/admin/tickets/automation-rules` | 自动化规则 | `ticket:read` | Zap |
| 275 | `/admin/approval-chains` | 审批链 | `approval:write` | Link |
| 280 | `/admin/permissions` | 权限管理 | `role:write` | Lock |
| 285 | `/admin/connectors` | 连接器/插件市场 | `connector:write` | Plug |
| 290 | `/admin/vector-store` | 向量存储配置 | `system:read` | Database |
| 295 | `/admin/system-config` | 系统配置 | `system:read` | Settings |
| 315 | `/admin/cmdb-types` | CMDB 类型 | `cmdb:write` | Database |
| 320 | `/admin/escalation-rules` | 升级规则 | `sla:write` | AlertTriangle |
| 325 | `/admin/escalation-matrices` | 升级矩阵 | `sla:read` | TrendingUp |
| 330 | `/admin/sla-templates` | SLA 模板 | `sla:write` | Layers |
| 335 | `/admin/service-catalogs` | 服务目录管理 | `service_catalog:read` | Boxes |
| 340 | `/admin/sla-definitions` | SLA 定义 | `sla:write` | Clock |
| 345 | `/admin/menus` | 菜单管理 | `system:write` | Menu |
| 350 | `/admin/workflows` | 工作流配置 | `workflow:write` | GitBranch |

---

## §4 能力治理（capability matrix）

前端通过 `capabilityPathRules` 把菜单路径前缀映射到 capability key，由 `useCapabilities` 提供真实状态，按 fail-closed 规则过滤：

```ts
// itsm-frontend/src/components/layout/sidebar/menu-config.ts + Sidebar.tsx（重复声明两处，需收敛）
export const capabilityPathRules: Array<[string, string]> = [
  ['/service-requests', 'serviceRequest'],
  ['/incidents', 'incident'],
  ['/problems', 'problem'],
  ['/changes', 'change'],
  ['/knowledge', 'knowledge'],
  ['/cmdb', 'cmdb'],
  ['/sla', 'sla'],
  ['/workflow', 'workflow'],
  ['/ai', 'ai'],
  ['/marketplace', 'marketplace'],
  ['/installations', 'marketplace'],
  ['/admin/connectors', 'marketplace'],
];
```

### 4.1 过滤规则（Sidebar.tsx:182-195）

```ts
filterByCapability(menus)：
  - 若路径命中 capability key：
    - capability 不存在 → 隐藏
    - maturity === 'disabled' → 隐藏
    - buildAvailable / deploymentReady / tenantReady 任一为 false → 隐藏
    - maturity === 'pilot' → 显示但打 Pilot 徽标
  - 命中规则但 capability 缺失时仍 fail-closed（避免未注册的菜单闪现）
```

### 4.2 当前生效的 capability 映射

| path 前缀 | capability key | 涉及菜单 | 备注 |
|-----------|---------------|---------|------|
| `/service-requests` | serviceRequest | 服务请求顶级 | 与 /tickets 同源 |
| `/incidents` | incident | 事件管理 + 子 |  |
| `/problems` | problem | 问题管理 + 子 |  |
| `/changes` | change | 变更管理 + 子 |  |
| `/knowledge` | knowledge | 知识库 + 子 |  |
| `/cmdb` | cmdb | CMDB + 子 |  |
| `/sla` | sla | SLA + 子（含 sla-monitor） |  |
| `/workflow` | workflow | 工作流 + 9 子 | 包含 `/approvals` 审批中心 |
| `/ai` | ai | AI 助手 + 3 子 |  |
| `/marketplace` | marketplace | ⚠️ **menuDefinitions 未注册，被规则保护** |
| `/installations` | marketplace | ⚠️ **menuDefinitions 未注册** |
| `/admin/connectors` | marketplace | admin/connectors 已注册，归属正确 |

### 4.3 已知重复声明

`capabilityPathRules` 在 `menu-config.ts` 与 `Sidebar.tsx` 各声明一份。**建议收敛**：从 `Sidebar.tsx` 改为 `import { capabilityPathRules, capabilityForPath } from './menu-config';`，避免后续漂移。

---

## §5 路径规范化（Sidebar.tsx:76-97）

```ts
const MENU_PATH_NORMALIZATIONS: Record<string, string> = {
  '/service-requests/list': '/service-requests',
  '/incidents/list': '/incidents',
  '/problems/list': '/problems',
  '/changes/list': '/changes',
  '/knowledge/list': '/knowledge',
  '/service-catalog/list': '/service-catalog',
  '/assets/list': '/assets',
  '/workflow/list': '/workflow',
  '/ai/chat/list': '/ai/chat',
  '/msp/list': '/msp',
  '/releases/list': '/releases',
  '/admin/index': '/admin',
  '/knowledge/articles/create': '/knowledge/articles/new',
  '/sla/overview': '/sla',
  '/email-intake/conversations': '/email-intake',
  '/knowledge/articles': '/knowledge',
  '/workflow/monitoring': '/workflow/dashboard',
};
```

兜底规则：`/xxx/list` 自动剥离 `/list` 后缀。

### 5.1 路径规范化反向问题

| 历史命名 | 当前等价 | 现状 |
|---------|---------|------|
| `/admin/index` | `/admin` | 已规范化 |
| `/sla/overview` | `/sla` | 已规范化 |
| `/email-intake/conversations` | `/email-intake` | 已规范化 |
| `/knowledge/articles` | `/knowledge` | 已规范化 |
| `/knowledge/articles/create` | `/knowledge/articles/new` | 已规范化 |
| `/workflow/monitoring` | `/workflow/dashboard` | 已规范化 |
| `/list` 后缀 | 全部剥离 | 已规范化 |

---

## §6 后端路由 / Handler / Service 链路

### 6.1 菜单运行时入口

```text
GET /api/v1/auth/menus
  ├─ middleware.AuthMiddleware(JWTSecret)
  ├─ handlers/rbac/handler.go:493 GetUserMenus
  │    ├─ 从 ctx 取 user_id, tenant_id
  │    └─ menuService.GetUserMenus(ctx, userID, tenantID)
  │         ├─ 查 user + roles
  │         ├─ getUserPermissions（超级管理员全部权限）
  │         ├─ menu 表查 is_enabled && is_visible (Order by sort_order)
  │         ├─ filterMenusByPermission
  │         ├─ buildMenuTree → main / admin
  │         └─ filterMSPMenus (若 !IsMSPEnabled)
  └─ common.Success(c, { main, admin })   // camelCase JSON
```

### 6.2 菜单管理端 CRUD

```text
GET    /api/v1/menus        RequirePermission(system_config, read)   ListMenus
POST   /api/v1/menus        RequirePermission(system_config, write)  CreateMenu
GET    /api/v1/menus/:id    RequirePermission(system_config, read)   GetMenu
PUT    /api/v1/menus/:id    RequirePermission(system_config, write)  UpdateMenu
DELETE /api/v1/menus/:id    RequirePermission(system_config, write)  DeleteMenu
POST   /api/v1/menus/init   RequirePermission(system_config, write)  InitDefaultMenus
```

> ⚠️ 注意权限是 `system_config:read/write`，不是 `menu:*`。管理 UI `/admin/menus` 前端调用 `MenuAdminAPI`，前端代码已就位 [menu-api.ts:67-102](file:///Users/heidsoft/Downloads/research/itsm/itsm-frontend/src/lib/api/menu-api.ts#L67-L102)。

### 6.3 关键路由注册点

| 路由 | 文件 | 说明 |
|------|------|------|
| `GET /api/v1/auth/menus` | [router/common_system_routes.go:31](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/router/common_system_routes.go#L31-L31) | 登录态菜单 |
| `/api/v1/menus/*` | [router/common_system_routes.go:158-167](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/router/common_system_routes.go#L158-L167) | 菜单管理端 |
| `/api/v1/marketplace/*` | [handlers/marketplace/handler.go:44-62](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/marketplace/handler.go#L44-L62) | marketplace 路由在 `router.go:612-613` 注册到 tenant 分组 |
| `/api/v1/org/{departments,teams}/*` | [router/common_system_routes.go:76-88](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/router/common_system_routes.go#L76-L88) | 组织架构（org 路径） |
| `/api/v1/projects/*` | [router/common_system_routes.go:92-100](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/router/common_system_routes.go#L92-L100) | 项目管理 |
| `/api/v1/applications/*` | [router/common_system_routes.go:104-111](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/router/common_system_routes.go#L104-L111) | 应用/微服务 |
| `/api/v1/system/tags`、`/api/v1/system/audit-logs` | [router/common_system_routes.go:115-117](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/router/common_system_routes.go#L115-L117) | 标签 + 审计 |

### 6.4 旧路径别名（向后兼容）

```go
// router/common_system_routes.go:121-132
tenant.GET("/departments", ...)              // alias /org/departments
tenant.GET("/departments/tree", ...)         // alias /org/departments/tree
tenant.GET("/teams", ...)                    // alias /org/teams
tenant.GET("/tags", ...)                     // alias /system/tags
tenant.Group("/admin").GET("/tenants", ...)  // alias /tenants
```

> 说明：这些 alias 仅 GET，避免破坏旧前端调用方。

---

## §7 前端 API client / hook 覆盖

### 7.1 API client 清单（按域）

| 域 | API 文件 | 测试文件 | 覆盖情况 |
|----|---------|---------|---------|
| Auth | `auth-api.ts` | `auth-api.test.ts` | ✅ |
| 用户 | `user-api.ts` | `user-api.test.ts` | ✅ |
| 角色/权限 | `role-api.ts` | `role-api.test.ts` | ✅ |
| 部门/组 | `group-api.ts` | `group-api.test.ts` | ✅ |
| 租户 | `tenant-api.ts` | `tenant-api.test.ts` | ✅ |
| **菜单** | `menu-api.ts` | `menu-api.test.ts` | ✅ |
| Ticket | `ticket-api.ts` + 11 子文件 | 11 测试 | ✅ |
| Incident | `incident-api.ts` | `incident-api.test.ts` | ✅ |
| Problem | `problem-api.ts` | `problem-api.test.ts` | ✅ |
| Change | `change-api.ts` | `change-api.test.ts` | ✅ |
| Change 分类 | `change-classification-api.ts` | `change-classification-api.test.ts` | ✅ |
| Change 评审 | `change-review-api.ts` | （无测试） | ⚠️ |
| Release | `release-api.ts` | `release-api.test.ts` | ✅ |
| 标准变更 | `standard-change-api.ts` | `standard-change-api.test.ts` | ✅ |
| Service Catalog | `service-catalog-api.ts` | `service-catalog-api.test.ts` | ✅ |
| Service Request | `service-request-api.ts` | `service-request-api.test.ts` | ✅ |
| Knowledge | `knowledge-base-api.ts` | `knowledge-base-api.test.ts` | ✅ |
| KEDB | `kedb-api.ts` | `kedb-api.test.ts` | ✅ |
| Problem Investigation | `problem-investigation.ts` | `problem-investigation.test.ts` | ✅ |
| CMDB | `cmdb-api.ts`、`cmdb-relationship.ts`、`cmdb-advanced-api.ts` | 3 测试 | ✅ |
| Cloud | `cloud-api.ts` | `cloud-api.test.ts` | ✅ |
| Asset | `asset-api.ts` | `asset-api.test.ts` | ✅ |
| SLA | `sla-api.ts`、`sla-template-api.ts` | 2 测试 | ✅ |
| 升级矩阵 | `escalation-matrix-api.ts` | `escalation-matrix-api.test.ts` | ✅ |
| 优先级矩阵 | `priority-matrix-api.ts` | `priority-matrix-api.test.ts` | ✅ |
| Workflow | `workflow-api.ts` + 8 子文件 | 9 测试 | ✅ |
| AI | `ai-api.ts`、`a2ui-api.ts`、`bpmn-ai-api.ts` | 3 测试 | ✅ |
| Template | `template-api.ts` | `template-api.test.ts` | ✅ |
| Notification | `notification-preference-api.ts` | `notification-preference-api.test.ts` | ✅ |
| MSP | `msp-api.ts` | `msp-api.test.ts` | ✅ |
| Reports | `reports-api.ts` | `reports-api.test.ts` | ✅ |
| Audit Log | `auditlog-api.ts` | `auditlog-api.test.ts` | ✅ |
| Vector Store | `vector-store-api.ts` | （无测试） | ⚠️ |
| Domain Config | `domain-config-api.ts` | `domain-config-api.test.ts` | ✅ |
| Common / 系统配置 | `common-api.ts`、`system-config-api.ts`、`capability-api.ts` | 3 测试 | ✅ |
| Batch Operations | `batch-operations-api.ts` | `batch-operations-api.test.ts` | ✅ |
| Collaboration | `collaboration-api.ts` | `collaboration-api.test.ts` | ✅ |
| Global Search | `global-search-api.ts` | `global-search-api.test.ts` | ✅ |

### 7.2 业务 hooks 覆盖

| Hook | 域 | 测试 |
|------|----|------|
| `useUserMenusQuery` | 菜单缓存 | `useUserMenusQuery` 间接测试通过 menu-api.test.ts |
| `useUserListQuery` | 用户 | （间接） |
| `useCapabilities` | 能力治理 | （间接） |
| `useTickets`、`useTicketsQuery` | Ticket | ✅ |
| `useTicketFilters`、`useTicketRelations` | Ticket | ✅ |
| `useIncidentsQuery`、`useIncidentFilters`、`useIncidentStats`、`useIncidentBatchOps` | Incident | ✅ |
| `useChangeClassification` | Change | ✅ |
| `usePriorityMatrix` | 优先级 | ✅ |
| `useDashboardData` | 看板 | ✅ |
| `useReports` | 报表 | ✅ |
| `useServiceCatalog` | 服务目录 | ✅ |
| `useKnowledgeBase` | 知识库 | ✅ |
| `useCMDB` | CMDB | ✅ |
| `useSLARealTime` | SLA | ✅ |
| `useWorkflow` | 工作流 | ✅ |
| `useTemplateQuery` | 模板 | ✅ |
| `useGlobalSearch` | 全局搜索 | ✅ |
| `useCollaboration` | 协作 | ✅ |
| `useBatchOperations` | 批量 | ✅ |
| `useFeedback` | 反馈 | ✅ |
| `useErrorHandler` | 错误处理 | ✅ |
| `useAccessibility` | 无障碍 | ✅ |
| `usePerformance` | 性能 | ✅ |
| `useResponsive` | 响应式 | ✅ |
| `useRowSelection` | 表格选择 | ✅ |
| `useTableKeyboardNav` | 表格键盘 | ✅ |
| `useCache` | 缓存 | ✅ |
| `useVersionControl` | 版本控制 | ✅ |
| `use-permissions` | 权限 | ✅ |
| `usePermissions`（来自 store） | 权限 | 间接覆盖 |

---

## §8 缺失菜单清单（page.tsx 存在但 menuDefinitions 未注册）

下表是 **页面与菜单脱钩** 的事实清单。每条都是 Bug #3 同类的"入口静默缺失"风险，必须收口。

### 8.1 一级缺登（顶级菜单缺失）

| path | name | 前端 page | 后端路由 | 建议 |
|------|------|----------|---------|------|
| `/marketplace` | 插件市场 | ✅ `marketplace/page.tsx` | ✅ `/api/v1/marketplace/items` (注册) | **新增顶级菜单** `marketplace`，permission `marketplace:read`；与 `/admin/connectors` 形成"用户浏览"+"管理员配置"双入口 |
| `/installations` | 已安装组件 | ✅ `installations/page.tsx` | ✅ `/api/v1/marketplace/installations` | 作为 `/marketplace` 的子菜单或顶级独立菜单 |
| `/tags` | 标签管理 | ✅ `tags/page.tsx` | ✅ `/api/v1/system/tags` | **新增顶级菜单**，permission `ticket_tag:read`，归属运营维度 |
| `/teams` | 团队管理 | ✅ `teams/page.tsx` + ✅ `enterprise/teams/page.tsx` | ✅ `/api/v1/org/teams` | 建议作为 `/admin` 子菜单（已存在 `/admin/teams`，重复） |
| `/projects` | 项目管理 | ✅ `projects/page.tsx` | ✅ `/api/v1/projects` | **新增顶级菜单**，permission `project:read` |
| `/applications` | 应用管理 | ✅ `applications/page.tsx` | ✅ `/api/v1/applications` | **新增顶级菜单**，permission `application:read` |
| `/improvements` | 改进措施 | ✅ `improvements/page.tsx` + `[id]` + `new` | （未发现独立 routes；可能是 change 子模块） | 暂挂起；先确认是否由 `/changes/pir` 承载 |
| `/templates` | 模板 | ✅ `templates/page.tsx` | `/api/v1/tickets/templates`、`/api/v1/dashboard/templates` | 建议作为 `/admin/templates` 或顶级 `/templates`，聚合工单/审批/邮件等多模板 |
| `/workflows` | 流程列表 | ✅ `workflows/page.tsx` | （与 `/workflow` 同根，是否 `/workflow/instances` 别名？） | 与 `/workflow/instances` 共存，建议在子菜单中保留 `workflow/instances` 与 `workflows` 跳转对齐 |
| `/sla-dashboard` | SLA 看板 | ✅ `sla-dashboard/page.tsx` | （`/api/v1/sla/...` 存在） | 作为 `/sla` 子菜单 `sla/dashboard` 或别名 `/sla/dashboard` |

### 8.2 二级缺登（顶级已有但子菜单缺失）

| 顶级 | 缺登子菜单 | 前端 page | 备注 |
|------|----------|----------|------|
| `/tickets` | `/tickets/create` | ✅ `tickets/create/page.tsx` | 当前 /tickets 顶级有页面，建议补"新建工单"子菜单 |
| `/tickets` | `/tickets/dashboard` | ✅ `tickets/dashboard/page.tsx` | 当前 /tickets/analytics 已存在，建议补 dashboard 或别名 |
| `/tickets` | `/tickets/cc` | ✅ `tickets/cc/page.tsx` | 抄送列表；建议作为工单管理下的运营维度 |
| `/tickets` | `/tickets/templates/*` | ✅ `tickets/templates/[id]/page.tsx` + `templates/page.tsx` | 工单模板管理；建议作为 `/admin/ticket-templates` 或 `/admin` 子菜单 |
| `/incidents` | `/incidents/[id]/edit` | ✅ `incidents/[id]/edit/page.tsx` | 编辑入口一般由详情页承担 |
| `/changes` | `/changes/pirs`、`/changes/[id]/edit`、`/changes/[id]/pir` | ✅ 存在 | PIR 是改进措施，建议子菜单归到 `/changes` 下 |
| `/cmdb` | `/cmdb/ci`、`/cmdb/ci-types`、`/cmdb/cis/[id]/edit`、`/cmdb/cis/[id]`、`/cmdb/cloud-*`、`/cmdb/reconciliation`、`/cmdb/registry` | ✅ 多个 | CI 详情/编辑、云账户/资源/服务、对账、注册表；当前 `/cmdb` 仅有 4 个子菜单 |
| `/assets` | `/assets/[id]`、`/assets/[id]/edit` | ✅ 存在 | 详情/编辑入口 |
| `/licenses` | `/licenses/[id]`、`/licenses/new` | ✅ 存在 | 许可证详情/新建 |
| `/knowledge` | `/knowledge/[id]`、`/knowledge/articles/[id]`、`/knowledge/articles/[id]/edit`、`/knowledge/reviews` | ✅ 多个 | 知识详情、文章详情/编辑、评审；当前仅 `articles/new` |
| `/service-catalog` | `/service-catalog/detail/[id]`、`/service-catalog/edit/[id]`、`/service-catalog/request/[id]` | ✅ 多个 | 详情/编辑/请求详情 |
| `/admin` | 缺 `/admin/workflows` 已存在 ✅，但 `/admin/teams` 与独立 `/teams` 重复 | ✅ | 去重 |
| `/system/organization`、`/system/users` | 无 page | — | 误识，删除 |

### 8.3 Capability 治理漏洞

- `capabilityPathRules` 已保护 `/marketplace`、`/installations`、`/admin/connectors`，但**缺** `/projects`、`/applications`、`/tags`、`/templates`、`/sla-dashboard`、`/workflows`、`/improvements` 的 capability 规则。
- 修复建议：补充 capability key（如 `project`、`application`、`tag`、`template`）并在 `useCapabilities` 中登记真实状态。

### 8.4 路由别名/直连风险

- `/api/v1/teams`、`/api/v1/departments`、`/api/v1/tags`、`/api/v1/admin/tenants` 顶层 alias 仅 GET；前端若误用这些路径 POST/PUT，会落到 alias 404 而不是真实 `/org/*`、`/system/*`、`/tenants/*`。
- `/api/v1/marketplace/items` POST `/install` 与 GET `/installations` 不在 `menu` 表注册（详见 §8.1），导致 `/marketplace` 与 `/installations` 页面对普通用户不可见。

---

## §9 改进建议（按优先级）

### P0：菜单收口（影响可见性）

1. **新增顶级菜单 `/marketplace`**（permission `marketplace:read`，icon `Store`）：解决 Bug #3 场景复现，让用户在前台看到插件列表。
2. **新增顶级菜单 `/tags`**（permission `ticket_tag:read`）：工单标签运营入口。
3. **新增顶级菜单 `/projects`**（permission `project:read`）：项目/工单关联运营入口。
4. **新增顶级菜单 `/applications`**（permission `application:read`）：应用/微服务运营入口。
5. **补全子菜单**：把 `/tickets/dashboard`、`/tickets/cc`、`/cmdb/ci-types`、`/cmdb/registry`、`/cmdb/cloud-resources`、`/cmdb/cloud-services`、`/cmdb/reconciliation`、`/knowledge/articles/[id]`、`/knowledge/reviews`、`/workflow/sla`（已存在）等补齐。

### P1：能力治理收敛

6. **去重 `capabilityPathRules`**：从 `Sidebar.tsx` 改为 `import { capabilityPathRules, capabilityForPath } from './menu-config';`。
7. **补 capability 规则**：在两处都补 `/projects` → `project`、`/applications` → `application`、`/tags` → `tag`、`/templates` → `template`。
8. **校验 capability 加载失败时 fail-closed**：当前 `filterByCapability` 已 fail-closed（capability 不存在就隐藏），需要 `useCapabilities` 在所有 capability 加载完成前不渲染菜单（避免闪现）。

### P2：菜单管理端能力

9. **`MenuAdminAPI.list()` 仅返回当前租户菜单**：建议支持跨租户查看（仅超管），便于运维。
10. **菜单导入导出**：复用 `pkg/seeder/seeder.go` 的 `menuDefinitions()` 作为导出基线，避免手动维护偏差。
11. **菜单 diff**：在 `MenuAdminAPI.initDefaults()` 后展示新增/未变更/冲突，便于审计。

### P3：可观测性

12. **菜单加载失败埋点**：当前 `menuError` 仅用 `message.error`，无遥测；建议接 `useErrorHandler` 上报。
13. **菜单权限拒绝埋点**：用户能看到菜单但点击触发 403 时，应记录 capability 与 user_permission 实际集合。

### P4：测试

14. **菜单契约测试**：新增 `menu-contract.test.ts` 校验 `/api/v1/auth/menus` 响应字段（`main/admin` 结构、camelCase、children 嵌套、permissionCode 类型）。
15. **菜单端到端测试**：登录 → 取菜单 → 点击顶级菜单 → 验证路径命中 `page.tsx`。

---

## §10 参考文件索引

- 后端种子： [itsm-backend/pkg/seeder/seeder.go:1805-2028](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/pkg/seeder/seeder.go#L1805-L2028)
- 后端运行时： [itsm-backend/service/menu_service.go:200-292](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/service/menu_service.go#L200-L292)
- 后端路由： [itsm-backend/router/common_system_routes.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/router/common_system_routes.go)
- 后端 Handler： [itsm-backend/handlers/rbac/handler.go:493-514](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/rbac/handler.go#L493-L514)
- 后端 Marketplace Handler： [itsm-backend/handlers/marketplace/handler.go](file:///Users/heidsoft/Downloads/research/itsm/itsm-backend/handlers/marketplace/handler.go)
- 前端 Sidebar： [itsm-frontend/src/components/layout/sidebar/Sidebar.tsx](file:///Users/heidsoft/Downloads/research/itsm/itsm-frontend/src/components/layout/sidebar/Sidebar.tsx)
- 前端类型/capability： [itsm-frontend/src/components/layout/sidebar/menu-config.ts](file:///Users/heidsoft/Downloads/research/itsm/itsm-frontend/src/components/layout/sidebar/menu-config.ts)
- 前端渲染： [itsm-frontend/src/components/layout/sidebar/MenuItems.tsx](file:///Users/heidsoft/Downloads/research/itsm/itsm-frontend/src/components/layout/sidebar/MenuItems.tsx)
- 前端 DTO + admin API： [itsm-frontend/src/lib/api/menu-api.ts](file:///Users/heidsoft/Downloads/research/itsm/itsm-frontend/src/lib/api/menu-api.ts)
- 前端 React Query： [itsm-frontend/src/lib/hooks/useUserMenusQuery.ts](file:///Users/heidsoft/Downloads/research/itsm/itsm-frontend/src/lib/hooks/useUserMenusQuery.ts)
- 关联文档： [docs/product/ai-automation-status.md](file:///Users/heidsoft/Downloads/research/itsm/docs/product/ai-automation-status.md)

---

> **后续动作**：把 §8.1 / §8.2 的缺登清单转化为 v1.6.x 的菜单收口任务，每条新增菜单需在 `menuDefinitions()` 中新增 `menuSpec`、在 `pkg/seeder/manifest_definitions.go` 或 `manifest_digest.go` 注册（生产初始化）、在 capability 矩阵登记，并在 `/api/v1/auth/menus` 通过契约测试。