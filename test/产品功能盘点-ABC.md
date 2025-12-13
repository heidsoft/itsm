# 产品功能盘点（A/B/C）

> 盘点日期：2025-12-12  
> 盘点范围：`itsm-backend`（Go/Gin/Ent）+ `itsm-prototype`（Next.js/TS）  
> 目标：按 **A 模块**、**B V0闭环**、**C 契约一致性** 三条线给出“已具备/部分具备/不可用与风险”的结论与证据。

---

## 0. 关键结论（先说人话）

1. **前端页面覆盖很全**（能看到几乎所有ITSM菜单与页面），但**后端“真实已挂载的路由”只覆盖了部分模块**。不少模块属于“代码存在（Controller/Service/Ent/Swagger）但未装配到 `main.go` / 未注册到 `router.SetupRoutes`”，因此运行时接口不可用。
2. **契约存在明显漂移**：后端 Swagger 同时存在 `/api/*` 与 `/api/v1/*` 两套前缀；前端 API 也混用 `/api/v1/*`、`/api/*`、甚至无前缀（如 `/changes`）。这会直接造成“某些页面必然打不通后端”或“不同页面调用的是不同版本接口”。
3. **SaaS多租户在部分模块存在结构性风险**：例如 `ServiceRequest` 的 Ent Schema **没有 `tenant_id` 字段**，但 Controller/Service 已按租户思路在写（从 `context` 取 `tenant_id` 并传参），会导致后续做SaaS化时必须重构该域的数据模型与查询过滤。

---

## A) 按模块盘点（页面/接口/数据/可用性）

### A.1 后端“运行时已注册路由”清单（依据 `itsm-backend/router/router.go` + `itsm-backend/main.go`）

**已注册并可用（/api/v1 前缀 + JWT/RBAC/Tenant/Audit）**
- 工单：`/api/v1/tickets/*`、分类/标签/模板、评论/附件/通知/评分/视图/智能分配等（`router/router.go`）
- 事件：`/api/v1/incidents/*`（关闭接口 TODO 注释掉）
- SLA：`/api/v1/sla/*`
- 用户：`/api/v1/users/*`
- 通知：`/api/v1/notifications/*`
- AI：`/api/v1/ai/*`
- Dashboard：`/api/v1/dashboard/*`
- 组织与资源：`/api/v1/departments/*`、`/api/v1/teams/*`、`/api/v1/projects/*`、`/api/v1/applications/*`（含 microservices） 、`/api/v1/tags/*`
- 审计：`/api/v1/audit-logs`
- BPMN：通过 `BPMNWorkflowController.RegisterRoutes()` 注册到租户组下

**明显“未注册/未装配”的模块（Controller/Swagger有，但 `main.go` 未初始化对应Controller，`SetupRoutes` 也未挂载）**
- 服务目录/服务请求（ServiceCatalog/ServiceRequest）
- CMDB（/api/cmdb/*）
- 知识库（/api/knowledge-articles/*）
- 问题管理（problems）
- 变更管理（changes）

> 证据：  
> - `main.go` 只装配了 Ticket/Incident/SLA/Approval(User)/AI/Dashboard/Dept/Team/Project/Application/Tag/Audit/BPMN 等 controller；未见 `ServiceController` / `CMDBController` / `KnowledgeController` / `ProblemController` / `ChangeController` 的初始化。  
> - `router/router.go` 仅显示 `/api/v1` 组内路由注册；未出现 `/service-*` `/cmdb` `/knowledge-articles` `/problems` `/changes` 等 group。

### A.2 模块矩阵（“页面存在≠功能可用”）

| 模块 | 前端页面（示例） | 后端实现（Controller/Service/Schema/Swagger） | 运行时可用性 | 结论 |
|---|---|---|---|---|
| 认证与租户 | `/login`、租户切换相关 | 有 AuthController + JWT/RBAC/Tenant Middleware | ✅ 可用（/api/v1/login） | **可用** |
| 工单 Ticket | `/tickets` `/tickets/[id]` | Controller/Service/多子模块路由齐 | ✅ 可用 | **可用（完成度最高）** |
| 事件 Incident | `/incidents` | 有基础 CRUD/Stats | ⚠️ 部分可用（CloseIncident未实现） | **可用但不闭环** |
| SLA | `/sla` `/sla-dashboard` | 定义/监控/预警规则 | ✅ 可用 | **可用** |
| Dashboard/报表 | `/dashboard` `/reports/*` | Dashboard handler + analytics/prediction | ✅ 可用 | **可用** |
| 用户/组织/资源 | `/admin/*`、`/enterprise/*`、`/projects`、`/applications` | 后端已注册 departments/teams/projects/applications/tags/users | ✅ 可用 | **可用（但需完善详情/关联等产品层）** |
| 变更 Change | `/changes` `/changes/[id]` | 有 `ChangeController` + `ChangeApprovalController`（Swagger标注部分 /api/v1/changes/*） | ❌ 未注册到 SetupRoutes | **页面存在，但后端接口当前不可用** |
| 问题 Problem | `/problems` | 有 `ProblemController`/Service | ❌ 未注册到 SetupRoutes | **页面存在，但后端接口当前不可用** |
| 服务目录/服务请求 | `/service-catalog` 等 | 有 `ServiceController` + `ServiceCatalog/ServiceRequest` 表 + Swagger `/api/service-*` | ❌ 未注册到 SetupRoutes（且数据模型缺 `tenant_id`） | **当前不可用/不满足SaaS** |
| CMDB | `/cmdb` | 有 `CMDBController` + Swagger `/api/cmdb/*` | ❌ 未注册到 SetupRoutes | **当前不可用（接口未挂载）** |
| 知识库 | `/knowledge-base` | 有 `KnowledgeController` + Swagger `/api/knowledge-articles/*` | ❌ 未注册到 SetupRoutes（且存在 tenantId 键名错误） | **当前不可用/存在Bug** |
| 工作流/BPMN | `/workflow/*` | 有 BPMN controller 注册接口 | ⚠️ 可用性依赖BPMN适配/编译与稳定性 | **可用但需专项稳定化** |
| AI | 多处嵌入/助手组件 | Swagger `/api/v1/ai/*`，工具执行/审批 | ✅ 可用（按路由） | **可用但需产品化与测试** |

---

## B) 按 V0闭环盘点（服务请求自助 + 三段审批 + 阿里云自动交付）

> V0目标参见：`prd/PRD-V0-服务请求自助-阿里云自动交付.md` 与 `prd/TECH-V0-多租户LandingZone-阿里云连接器-自动交付.md`

### B.1 当前已具备的“可复用能力”
- **多租户中间件、RBAC、审计、统一响应码**：后端骨架已具备（SaaS化关键底座）。
- **AI工具执行与审批骨架**：`/api/v1/ai/tools/*` 已存在，可作为“自动交付/危险操作需审批”的通用机制。
- **审批服务（当前主要在工单域）**：`tickets/approval/*` 已挂载，可复用“审批记录/流程框架”的设计经验。

### B.2 V0闭环的关键缺口（当前状态）

**1）服务目录/服务请求接口未挂载（阻断闭环）**
- 后端虽有 `ServiceController` 与 Swagger `/api/service-catalogs` `/api/service-requests`，但 **运行时未注册到 `SetupRoutes`** → 前端即使写了API也无法打通。

**2）ServiceRequest 数据模型不满足 SaaS**
- Ent Schema：`ent/schema/servicerequest.go` **缺少 `tenant_id`**，无法做租户隔离与按租户查询，是 SaaS 的结构性缺陷。
- 现有 ServiceRequestService 虽然函数签名带 `tenantID`，但无法落库/过滤（因为表里没有该字段）。

**3）字段与策略能力缺失（与V0 PRD要求不匹配）**
V0要求的关键字段/能力（到期回收、成本中心、数据分级、合规条款、公网/白名单/堡垒机策略等）在当前 ServiceRequest/ServiceCatalog 领域模型与接口中 **尚未体现**：
- 缺字段：`expire_at`、`cost_center`、`data_classification`、`compliance_ack`、`needs_public_ip`、`source_ip_whitelist`、`bastion_required`、`form_data/schema` 等
- 缺流程：三段审批（主管→IT→安全）对 ServiceRequest 的状态机与审批记录
- 缺履约：任务清单/执行日志/回写、失败回滚/接管
- 缺连接器：阿里云 Provider（ECS/RDS/OSS/VPC/RAM）与 STS最小权限、标签、回收作业

**4）前端实现存在两套不一致的“服务目录/请求”API**
- `src/lib/api/service-catalog-api.ts` 期望 `/api/v1/service-catalog/*`（大量端点）
- `src/lib/api/service-request-api.ts` 期望 `/api/service-requests/*`
- 后端 Swagger 则提供 `/api/service-catalogs` 与 `/api/service-requests`（命名风格不同，且运行时未挂载）

> 结论：以当前代码状态，**B 这条 V0闭环尚不可用**，但“底座能力”可复用，补齐路由装配 + 数据模型 + 连接器与门禁后可快速落地。

---

## C) 契约一致性盘点（Swagger/路由/前端API/类型）

### C.1 API 路径前缀混用（高风险）
- 后端运行时路由主要是：**`/api/v1/*`**（见 `router/router.go`）
- Swagger 同时存在：
  - `/api/v1/*`（如 AI、用户等部分）
  - `/api/*`（如 service-catalogs/service-requests、cmdb、knowledge-articles）
- 前端调用同时存在：
  - `/api/v1/...`（例如 `src/lib/api/service-catalog-api.ts`）
  - `/api/...`（例如 `src/lib/api/service-request-api.ts`）
  - 甚至无前缀（例如变更 `change-service.ts` 使用 `${API_BASE_URL}/changes`）

**影响**：同一产品内“不同模块指向不同API版本/不同后端路径”，必然造成部分页面不可用或数据不一致。

### C.2 租户上下文键名不一致（明确Bug）
- `KnowledgeController` 的 Create/Get/List 使用 `c.Get("tenant_id")`  
- 但 Delete/Categories 使用 `c.Get("tenantId")`（大小写/命名不同）  

**影响**：删除文章与获取分类接口即便挂载，也会在租户上下文读取时直接异常/失败。

### C.3 SaaS租户隔离字段缺失（结构性问题）
- `ServiceCatalog` 表有 `tenant_id`（`ent/schema/servicecatalog.go`），但 Swagger 的 `ServiceCatalogResponse` 并不包含 `tenant_id/is_active` 等字段 → 前端类型存在多套口径。
- `ServiceRequest` 表 **没有 `tenant_id`**（`ent/schema/servicerequest.go`），但 Controller/Service 的签名与实现按租户维度在写（从 `context` 读租户并传参） → 该模块在SaaS化时必须补字段并全量改造查询过滤。

### C.4 前端类型与API层存在“多套定义”（长期维护风险）
- `src/app/lib/api-config.ts` 内定义了 `ServiceCatalog/ServiceRequest/Ticket/...` 等类型（且带 `tenant_id/is_active/form_schema` 等字段）
- `src/types/*`（如 `types/service-catalog.ts`）与 `src/lib/api/*-api.ts` 又有另一套类型与端点约定

**影响**：同一实体在不同页面/模块出现字段不一致，导致 TypeScript 错误、运行时空字段、以及接口改动时需要多处同步。

### C.5 统一建议（C线修复策略）
- **确定唯一API前缀**：建议统一为 `/api/v1`（或统一为 `/api`，但必须全系统一致）
- **Swagger 与运行时对齐**：保证 swagger 中的路径一定在 `SetupRoutes` 中真实注册（否则文档误导）
- **前端API层收敛**：只保留一套 `httpClient + api modules + types`（建议以 `src/lib/api/*` + `src/types/*` 为唯一来源）
- **契约生成**：中期建议用 OpenAPI 生成 TS client/types，避免手写漂移

---

## 建议的下一步（按优先级）

### P0（阻塞可用性）
- 将 **CMDB/知识库/服务目录/服务请求/问题/变更** 等已实现Controller的模块，补齐 `main.go` 装配 + `SetupRoutes` 路由挂载，使“页面→接口”真正连通
- 统一 API 前缀（/api vs /api/v1）并同步更新前端调用
- 修复 `KnowledgeController` 的 `tenantId` 键名错误
- ServiceRequest 领域：补 `tenant_id`（以及V0需要的关键字段雏形），否则无法SaaS化

### P1（闭环落地）
- 服务请求自助（V0）落地：三段审批 + 策略门禁（公网仅443、源IP白名单、堡垒机）+ 执行审计 + 到期回收作业
- 阿里云 Provider（STS最小权限、标签、幂等/回滚/验证）作为爆点能力实现

### P2（质量与可维护性）
- 清理前端类型多套定义、API层混用；建立“唯一契约来源”
- 提升测试覆盖率（尤其 ServiceRequest/Provisioning/Policy Guardrails）

