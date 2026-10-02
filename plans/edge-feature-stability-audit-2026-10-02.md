# 边缘功能稳定性盘点（2026-10-02）

核心流程（工单/事件/问题/变更/审批）已在批次 0 与 N1/N2 收敛。本轮只回答一个问题：
**核心之外的"小功能"到底哪些是真能用、哪些是半成品、哪些是假的。**

方法：不信文档、不信门禁，从真实 Router 注册表与前端调用双向比对。

- 后端路由注册 892 条，去重后 **876 个 method+path**（`router/*_routes.go` 与 `handlers/*/handler.go` 有 11 条重复注册）。
- 前端 `src/lib/api|services` 共 881 个 HTTP 调用；Next.js 页面 168 个；种子菜单 97 项；能力矩阵 25 项。
- 每条结论都给出 `文件:行`，可在仓库内直接复核。

## 一、用户点了就坏（P0，最高优先）

这 5 项都有活的 UI 入口，点下去必然失败，且现有任何测试都没抓住。

| # | 功能 | 生产入口 | 实测行为 | 根因位置 |
|---|---|---|---|---|
| 1 | 工单导出 | `POST /api/v1/tickets/export`（`router/ticket_routes.go:32`，权限 `ticket:export`）+ `TicketList.tsx:186` 导出按钮（`/tickets` 页真实挂载） | 永远 HTTP 500 / code 5001 / 「导出失败」 | `handlers/ticket/service.go:309` 是占位 `return nil, fmt.Errorf("export not implemented in handlers layer")`；**可用实现已在** `service/ticket_service.go:2022`（带 `TenantID` 谓词），且 `handlers/ticket/service.go:26` 已持有 `productionSvc`，只差一行委派 |
| 2 | 附件下载/预览 | `TicketDetail.tsx:1268` 附件页签 → `ticket-attachment-api.ts:71,78` | `GET /tickets/:id/attachments/:aid[/preview]` **未注册**（只有 list/POST/DELETE，`router/ticket_routes.go:172-174`）→ 404；上传时写入的 `fileURL`（`service/ticket_attachment_service.go:165`）本身也指向不存在的路由 | `handlers/ticket_attachment/handler.go:132,166` 两个 handler 全仓无任何注册点 |
| 3 | 全局标签管理 | 侧边栏「全局标签」→ `/tags`（`src/app/(main)/tags/page.tsx`） | 列表可读；**新建/编辑/删除/绑定全部 404** | `tag-service.ts:21` baseUrl `/api/v1/tags`，而 router 只注册了 `GET /tags`、`GET /system/tags`（`router/common_system_routes.go:107,117`）；真 CRUD 在 `/api/v1/ticket-tags`（`handlers/ticket-tag-service.ts:26`，无 UI 调用），且后端另有第二张 `tag` 表 → **两套标签模型并存** |
| 4 | 侧边栏「全局标签」链接 | 菜单基线 `pkg/menubaseline/baseline.go:147` → `/admin/tags` | 97 条菜单里唯一一条**指向不存在页面**的死链（`src/app/(main)/admin/` 下无 `tags`） | 前端 capability 过滤对无映射路径直接放行（`Sidebar.tsx:163-167`），所以死链会渲染出来 |
| 5 | 工单依赖影响分析 | `GET /api/v1/tickets/:id/dependencies`（`router/ticket_routes.go:283`） | **每次 GET 必 400**「请求参数错误」 | `handlers/ticket_dependency/handler.go:39-46` 在 GET 上用 `ShouldBindJSON` + `binding:"required,oneof=close delete change_status"`；无 body 即失败 |

同类待确认（无 UI，暂不排 P0）：`POST /api/v1/tickets/import` handler 存在但**没有注册路由**（`handlers/ticket/handler.go:686`）。

## 二、后端有接口、前端够不着（功能对操作者不存在）

实测零前端引用的路由根：`/surveys`(7)、`/process-trigger`(5)、`/admin/skills`+`/skills`(9)、`/vendors`(4)、`/timers`(3)、`/my-approvals`、`/approval-records`、`/service-catalog-services`(2，与 `/service-catalogs` 同 handler 的死别名)。

其中真正是"半成品"而非"未做 UI"的：

- **问卷 survey**：`handlers/survey/handler.go:34` 的 `RegisterRoutes` 是死代码（真实注册在 `router/survey_routes.go:12`）；`service/survey_service.go:31` 提交响应时 `SetRespondentID(0)`，注释写着"Should come from auth context"——**答卷人身份永久丢失**，违反「身份必须来自认证上下文」；无 seed、无菜单、无消费方，14 条路由里 7 条是重复计数。
- **技能 skill**：注册表是进程内 `map[string]Skill`（`service/skill_base.go:24`），**无持久化、无任何 tenant 谓词** → 跨租户可见、重启即丢；与 v1.7「Skill registry」目标直接冲突。
- **my-approvals**：`service/approval_service.go:242-256` 只按 tenant/ticket/workflow/status 过滤，从不按调用人过滤 → "我的审批"返回整个租户的记录。路由已标 Deprecated/Sunset。
- **vendor**：`service/vendor_service.go:49-50` `total, _ :=` 吞掉 Count/All 的错误后返回 200 空列表 → **DB 故障伪装成空成功**，违反能力状态规则。
- **timer**：`service/timer_store.go:293,330` 的租户谓词是 `if filter.TenantID > 0`，即 `TenantID<=0` 时**退化为全表扫描**，只靠 handler 兜住。

## 三、假成功与未实现被当成可用能力卖

| 位置 | 用户看到 | 性质 |
|---|---|---|
| `handlers/problem/handler.go:880-888` | 问题 SLA 恒返回 `slaStatus:"none"` 成功 | 伪装：schema 里本就没有问题 SLA，与"未配置"不可区分 |
| `src/lib/api/service-catalog-api.ts:547` | 服务分析返回全零 `ServiceAnalytics`；`:523` 补零 stats；`:480` 返回硬编码 portal 配置 | 伪装（当前无页面消费，属死代码） |
| `src/lib/services/smart-assignment-service.ts:239,261` | `getUserSkills`/`getAllUserWorkloads` 返回 `[]` 成功，同文件 `:249,:255,:292,:302` 却 throw | 语义不一致 |
| `service/cloud_discovery_service.go:282-340` | 各 discover* 返回空切片 + 记日志，`DiscoverAccount:183` 返回 nil | 伪装（未被路由使用，注册路径 `/cmdb/discovery/jobs` 已正确 fail-closed 503） |
| `handlers/capability/handler.go:70-71` | `tag`、`template` 申报 **GA** | 与第一/三节实测冲突：标签写操作 404、模板 GA 但真实实现只覆盖 `/tickets/templates`，另有一份 43 调用的死客户端 |

诚实的一批（无需改语义）：问题评论 503/5003（已在 swagger 标注）、知识评论 503/5003、`handlers/dingtalk/handler.go:51`/`wecom/handler.go:48` 的不可达防御分支。

## 四、前端平行死实现（不可从任何 URL 到达）

按调用数排序，均**已从 router 表逐条比对确认无对应后端路由**，且组件链没有任何 `src/app/**/page.tsx` 导入它们：

| 死模块 | 调用数 | 无路由 | 链条终点 |
|---|---|---|---|
| `src/lib/api/template-api.ts` | 43 | 43 | `useTemplateQuery` → `components/templates/{TemplateList,TemplateEditor,TemplateCard,FieldDesigner}` → **无页面** |
| `src/lib/api/batch-operations-api.ts` | 38 | 35 | `useBatchOperations` → `components/batch-operations/{BatchOperationModal,BatchProgressModal}` → **无页面** |
| `src/lib/api/reports-api.ts` | 33 | 28 | `useReports` → **无消费者**；活的报表页走 `src/app/(main)/reports/hooks/useReportData.ts` |
| `src/lib/api/collaboration-api.ts` | 40 | 17 | `useCollaboration` → 无消费者（含 `POST /tickets/:id/comments/export` 这种根本不存在的路径，`:430`） |
| `src/lib/services/ticket-service-v2.ts` | 34 | 15 | 仅被 `src/lib/services/index.ts` barrel 引住 |
| `src/lib/api/change-classification-api.ts` | 23 | 16 | `useChangeClassification` → 无消费者 |
| `src/lib/api/priority-matrix-api.ts` | 21 | 20 | `usePriorityMatrix` → 无消费者 |
| `src/lib/api/ticket-relations-api.ts` | 33 | 12 | `components/business/TicketDependencyManager.tsx` → **无页面挂载** |

约 **178 个前端调用打向不存在的后端**。补充实测：`src/lib/api/index.ts`、`src/lib/services/index.ts` 两个 barrel **没有任何 app 文件导入**（只有 `__tests__` 导入），所以这些链不进产物包，但仍在 `type-check` 面内长期存活。死/孤儿前端合计约 **10,099 LOC**（含 `components/templates` 3,759、`components/reports` 1,616、batch 链 1,753、collaboration 874、ticket-service-v2 513、change-classification 389、priority-matrix 386），另有 6 个零外部引用的组件目录：`components/dashboard`(457)、`forms`(295)、`charts`(243)、`system`(150)、`service-catalog`(14)、`knowledge-base`(16)。

### 四.1 文档把这些死实现申报成已交付（同一批证据）

- `docs/product/frontend-menu-and-features.md:420,434,445,446` 给 `change-classification-api`、`priority-matrix-api`、`batch-operations-api`、`collaboration-api` 全部打 ✅；`:459-471` 又把 `useChangeClassification`、`usePriorityMatrix`、`useReports`、`useTemplateQuery`、`useCollaboration`、`useBatchOperations` 列为 ✅。上述 hook **零页面消费者**。
- `docs/product/itsm-commercial-capability-contract.md:20` 与 `docs/product/business-snapshot-2026-09-24.md:58` 把「优先级矩阵」列为**工单/事件 GA 候选**的业务证据；实测后端**没有任何含 priority 的路由**（`/api/v1/priority*` 不存在），前端只有死链。
- 报表域存在两套实现：权威活的是 `src/app/(main)/reports/page.tsx:3` → `components/business/AdvancedReporting` + `reports/hooks/useReportData.ts`（打到真实存在的 `GET /api/v1/reports`，`router/dashboard_routes.go:281`）；`src/lib/api/reports-api.ts` 是第二套死实现。模板域同理，活的是 `/tickets/templates` + `TicketApi`（`router/ticket_routes.go:39-46`），死的是 `template-api.ts`。

按 AGENTS「Doc drift is a defect」，这些 ✅ 与 GA 证据必须在同一批次里收回，不能只删代码。

## 五、契约与工程卫生的可计数欠债（门禁是绿的，因为它们是 advisory）

实测数字，全部由脚本产出：

- **67 条**列表信封违规（`*ListResponse` 缺 `items/total/page/pageSize/totalPages`），集中在 `dto/{ticket_workflow,standard_change,role,release,project,menu,asset_license,notification,change_pir,survey,tenant,service,msp,knowledge,cmdb*,cloud,change,known_error,asset}_dto.go`。
  `scripts/static-gates/check-pagination-shape.sh` 自己打印了这些违规却以 ADV 退出 0，注释写着"需独立 PR 修复"——即批次 0 第 2 项（信封收敛）只在被扫描的范围内收了口，**DTO 侧从未收敛**。
- **13 处** handler 直接用非 `items` 键返回列表：`handlers/vendor/handler.go`(`list`)、`handlers/ticket_notification`(`notifications`)、`handlers/msp`(`reports`,`tickets`)、`handlers/problem_investigation/routes.go` 与 `handlers/knowledge/handler.go`(`tags`)、`handlers/common/handler.go`(`tenants`)、`handlers/bpmn/monitoring.go`(`instances`,`logs`)、`handlers/bpmn/dashboard.go`(`list`)、`handlers/ai/handler.go`(`results`,`data`)、`handlers/a2ui/handler.go`(`data`)。
- **11 条**重复路由注册（survey 7 + feishu 4）在 `handlers/*/handler.go` 的 `RegisterRoutes` 里是死代码；若哪天两处同时生效，gin 会直接 panic。
- **274 处** `common.Fail(..., err.Error())` 直接把底层错误文本回给客户端，边缘域最集中：`handlers/cmdb/production_service.go`(73)、`handlers/bpmn/workflow.go`(31)、`handlers/notification`(27)、`handlers/email_intake`(27)、`handlers/bpmn/process_trigger.go`(22)。典型后果：`handlers/survey/handler.go` 所有分支写死 `5001`，即**资源不存在返回 500 而不是 404**。
- advisory 门禁存量：`check-bare-json`（dingtalk/wecom/approval 裸 `c.JSON`）、`check-context-bg`（`service/incident_service.go:398,413` 等在 goroutine 里用 `context.Background()`）、`check-raw-fetch`（前端 4 处裸 `fetch`：`session-api.ts:137`、`ai-api.ts:584`、`a2ui-api.ts:33`、`base-api.ts:309`）。
- 零测试资源根合计 **77 个端点**：releases(10)、assets(9)、known-errors(9)、marketplace(7)、surveys(7)、admin/skills(7)、standard-changes(7)、change-review(4)、vendors(4)、timers(3)、a2ui(3)、escalation-matrices(3)、skills(2)、workbench(1)、global-search(1)。
- 工单 7 个边缘子域（view/tag/rating/attachment/comment/dependency/category）**在真实 router 入口上没有任何跨租户拒绝测试**，只 category 有 handler 级测试（`handlers/ticket_category/handler_test.go:86-146`）。
- 前端 i18n：193 个文件在**没有 i18n hook** 的情况下把中文写进 `title/label/placeholder` 等可见属性（`src/app/(main)/admin/escalation-rules` 37 处、`admin/tenants` 33 处、`admin/sla-definitions` 29 处、`admin/ticket-categories` 21 处）；`console.*` 436 处/171 文件；50 个文件仍静态 `import { message } from 'antd'`（AGENTS 明令禁止）。
- `Sidebar.tsx:175` 给 pilot 能力打英文 `Pilot` 徽章，与全中文化界面不一致。

## 六、判定汇总

- 能用且被测：`ticket_category`（最佳样本）、`service_catalogs`、`sla`、`cmdb` 主干、工单评论（可用但无跨租测试）。
- 能用但零测试：releases、assets、known-errors、marketplace、standard-changes、change-review、escalation-matrices、timers。
- 点了就坏：**导出、附件下载/预览、标签写操作、侧边栏标签死链、依赖分析**。
- 后端做完但没人能用到：surveys、process-trigger、skills 管理面、vendors、my-approvals。
- 假成功：problem SLA `none`、服务分析全零、云发现空切片、vendor 吞错。
- 前端平行死实现：约 178 个调用、8 条不可达组件链。

## 七、建议批次（等拍板）

- **批次 E1｜点了就坏（P0，约 1 天）**：export 一行委派到已有实现；注册 attachment download/preview 两条 GET 并让 `fileURL` 与路由一致；标签写操作改指向 `/api/v1/ticket-tags` 并把菜单 `/admin/tags` 修到真实页面 + 二表收敛为一个所有者；依赖分析 GET 改查询参数绑定。每项配"修复前必红"的路由级回归（含跨租户拒绝）。
- **批次 E2｜假成功清零 + 申报口径收回（约 0.5 天）**：problem SLA / 服务分析 / 云发现 / vendor 一律改显式 unavailable 或删除死面；把 `handlers/*` 13 处非 `items` 键并入信封棘轮；同时收回 `frontend-menu-and-features.md` 的 6 条 ✅ 与两份产品文档里的「优先级矩阵 = GA 证据」。
- **批次 E3｜死实现大扫除（约 0.5 天）**：删掉第四节 8 条不可达链（约 10k LOC，先从两个 barrel 摘除引用再删文件，保留仍有真实调用者的 `ticket-service.ts`/`advanced-reporting`）与 11 条重复 `RegisterRoutes`；让"未实现"只剩一处真相。
- **批次 E4｜契约与错误语义（1-2 天，可切片）**：67 条 DTO 违规按模块分批收敛，并把 `check-pagination-shape` 从 ADV 升为硬门禁；survey 等域的 `5001` 改按语义映射 404/409；274 处 `err.Error()` 直出逐步净化。
- **批次 E5｜可达性与归属裁决**：surveys / timers / vendors / skills（进程内无租户注册表）/ my-approvals 决定「补 UI 并加租户与身份维度」还是「明确下线」，不要继续留无人可及或越权可见的后端。
- **批次 E6｜前端体验债**：193 个文件的硬编码中文可见文案、50 个静态 `message` 导入、436 处 `console.*`、`Sidebar` 英文 `Pilot` 徽章；建议随所在域改动顺带收，不单独立项。

既有待办不变：#25 权限回归夹具收敛、#26 生产污染报告、fresh-install-gate 转必需。

### E1 收口进度（2026-10-02 起，逐项提交）

| # | 项 | 状态 | 证据 |
|---|---|---|---|
| 1 | 工单导出 | 已修 | `11c2d8fa`；`router/ticket_export_route_test.go` 打真实 `SetupRoutes`，修复前 8 条断言红 |
| 2 | 附件下载/预览 | 已修 | `bdcae568`；`router/ticket_attachment_route_test.go` 含跨租户 404/403 拒绝，修复前 7 子用例 13 断言红 |
| 3 | 标签写操作 | 已修 | `79d253ef`；写链统一到 `/api/v1/ticket-tags`，删除 `tag-service.ts` 死客户端；`router/ticket_tag_routes_test.go` 修复前 8/10 子用例红 |
| 4 | 侧边栏「全局标签」死链 | 已修 | 页面 `git mv` 到 `src/app/(main)/admin/tags/page.tsx`（种子菜单 path 不变 ⇒ 存量租户零数据变更）；`capabilityPathRules` 同步 `/admin/tags`；新增门禁 C.6.6「基线 path ↔ 前端页面」，撤掉页面移动即实测 1 FAIL / exit 1，恢复后 97 条 path 全绿 |
| 5 | 工单依赖影响分析 GET | 已修 | `handlers/ticket_dependency/handler.go` 改查询参数绑定 + 服务层 `ErrDependencyTicketNotFound`；`router/ticket_dependency_route_test.go` 打真实 `SetupRoutes`（10 子用例，修复前 7 个为红，含跨租户与自报 `tenantId` 均 404/4004）；契约见 `docs/api-reference.md`「工单依赖影响分析接口」 |

仍待拍板（不在 E1 范围内动）：`tag` 第二张表的退役边界（`code` 列 + `pkg/seeder/seeder.go:2364` 清单按 Code 建键 + 5 张关联表外键）、只读别名 `GET /tags`/`GET /system/tags` 的下线，以及菜单基线 `PermissionCode: "system:read"`（`baseline.go:147`）与页面真实所需 `ticket_tag:*` 不一致 —— 该字段会随 `upsertMenu` 刷新到存量行，属访问控制变更。

### E2 收口进度（2026-10-02 起，逐项提交）

| # | 项 | 状态 | 证据 |
|---|---|---|---|
| 1 | vendor 吞错与 `list` 信封 | 已修 | `service/vendor_service.go` 不再 `total, _ :=`，新增 `ErrVendorNotFound`/`ErrVendorCodeExists`；`handlers/vendor/handler.go` 按 409/404/400/500 分语义、信封改 `common.SuccessWithList`；`router/vendor_routes_test.go` 打真实 `SetupRoutes`（含用第二连接 drop `vendors` 表制造的「仅供应商查询失败」故障），修复前两个测试函数全红 |
| 2 | problem SLA `none` 空成功 | 已修 | `handlers/problem/handler.go` `GetProblemSLA` 改显式 503/5003（资源存在性先判：404/4004、400/1001、401 不变）；`router/problem_sla_route_test.go` 打真实 `SetupRoutes`，修复前 4 条断言红；删除不可达的 `ProblemSLACard.tsx`、`ProblemApi.getProblemSLA` 与 `Problem` 上后端从不返回的 `slaStatus`/`responseDeadline`/`resolutionDeadline` 字段；`docs/api-reference.md` 新增「问题 SLA 与评论（能力未就绪）」，`make swagger-gen` 重生成 |
| 2b | 服务分析全零 / 云发现空切片 / smart-assignment 语义不一致 | 已修 | 服务目录四个不发请求的方法（`getServiceAnalytics`/`getServiceRatings`/`getFavorites`/`recordServiceView`）改为本文件既有的 `unsupportedFeature` 显式失败；`getCatalogStats` 停止补零，`ServiceCatalogStats` 减为后端真实三字段。删除两套并行死实现：`service/cloud_discovery_service.go`（674 行，AWS/Azure 返回空切片、国内三家 `return nil`，`upsertCloudResource:422` 查重缺租户条件；真实链路是 command-bus `cloud.DiscoveryWorker` + 已 fail-closed 的 `/api/v1/cmdb/discovery/jobs`）与 `src/lib/services/smart-assignment-service.ts`（455 行，与 `ticket-assignment-api.ts` 双实现且用 POST 打后端 GET 路由）。`service-catalog-api.test.ts` 修复前实测 6 个用例红 / 修复后 31 绿；`go build ./...`、`go test ./service/ ./service/cloud/... ./handlers/cmdb/`、staticcheck、type-check、eslint 全绿 |
| 2c | 服务目录四个无消费方的死 hook 与 `ServiceAnalytics` 类型 | 待实施（并入 E3） | `useServiceAnalyticsQuery`/`useFavoritesQuery`/`usePortalConfigQuery`/`useCatalogStatsQuery` 只被自己的测试引用，删除应与 E3「不可达前端实现」清账一起做，避免本批扩大面 |
| 3 | `handlers/*` 其余 12 处非 `items` 键并入信封棘轮 | 待实施 | — |
| 4 | 收回 `frontend-menu-and-features.md` 6 条 ✅ 与两份产品文档的「优先级矩阵 = GA 证据」 | 待实施 | — |

E2-1 附带发现（不在本批动，交 E5）：`ent/schema/vendor.go` 的 `code` 是**全局**唯一索引而不是 `(tenant_id, code)` 组合唯一，跨租户占用同编码会互相冲突；同一 schema 里 `vendor_type`/`contact_*`/`address`/`website` 全部是 Ent 必填字段（无 `Optional`），而 `CreateVendorRequest` 只把 `name`/`code` 声明为必填——这属于「后端可达性与归属裁决」条目，若 vendors 保留就必须连表结构与 DTO 一起重做。
