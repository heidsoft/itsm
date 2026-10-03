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

- **问卷 survey**：`handlers/survey/handler.go:34` 的 `RegisterRoutes` 是死代码（真实注册在 `router/survey_routes.go:12`）**〔已删，E3-3〕**；`service/survey_service.go:31` 提交响应时 `SetRespondentID(0)`，注释写着"Should come from auth context"——**答卷人身份永久丢失**，违反「身份必须来自认证上下文」；无 seed、无菜单、无消费方，14 条路由里 7 条是重复计数。
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

### 四.0 实测校订（E3-2，2026-10-02）

上表的判定方式（按目录名与调用数）在收口时改成了两条可验证的证明，因此结论有出入：

- **判定方法**：以 `src/app/**` 的 `page|layout|template|error|global-error|loading|not-found|route` 为入口，对 import 图（静态 `from`、动态 `import()`、`require()`，`@/` → `src/`）做 BFS，闭包外的生产文件即不可达；再用 `tsc --noEmit` 反证——只要树里还有任何文件（页面或非页面）引用被删模块，编译期必然报 TS2307。删除后全量 jest 与 type-check 均绿，证明没有隐藏的动态引用面。
- **本表判错的一条**：`ticket-relations-api.ts`（33 调用 / 12 无路由）**不是死实现**，真实链路是 `components/ticket/TicketDetail.tsx:69` → `RelationPanel` → `useTicketRelations`，已保留；该行的「链条终点 = `TicketDependencyManager.tsx` 无页面挂载」只对后半段成立，被删的是 `TicketDependencyManager.tsx` 本身（748 行，全仓零引用）。
- **本表漏记的死实现**（同法实测，均已删）：`components/business/ticket-modal/**`（15 文件 1 630 行，全仓零引用）、`src/lib/templates/**`（9 文件 1 543 行，连自身测试都不引用它）、`src/lib/reports/report-engine.ts`（2 文件 744 行）、`src/lib/hooks/useServiceCatalog.ts`（18 个 hook 外部引用数全为 0，页面另有 route-local `service-catalog/hooks/useServiceCatalogData` 承担同一职责）。
- **实际清账数**：78 个文件 / 22 503 行（不是「约 10k LOC」）。
- **不能照抄的后续候选清单**：同一次 BFS 在删除后仍列出 208 个闭包外文件（`src/components/business` 51 个 / 15 954 行、`common` 18 个 / 2 736 行、`auth` 8 个 / 1 715 行、`src/lib/theme/index.ts`、`src/lib/utils/validation.ts`、`src/lib/api/cmdb-advanced-api.ts` 等）。抽查两类：`src/lib/theme/index.ts` 在 `src/lib/theme/` 之外**零 importers**（应用实际用 `src/lib/design-system/theme`），是同类真阳性；`src/components/auth/*` 只被自己的 `auth/index.ts` barrel 引用，barrel 本身不可达。但这张表只能当**候选**：BFS 只认静态 import，且按「生产可达」判定会把「只有测试引用」的文件一并列入（那些测试仍会在删除时红），任何一批删除都必须重新用「tsc 反证 + 全量 jest」逐文件证明，不得按这张表批量删。


### 四.1 文档把这些死实现申报成已交付（同一批证据）

- `docs/product/frontend-menu-and-features.md:420,434,445,446` 给 `change-classification-api`、`priority-matrix-api`、`batch-operations-api`、`collaboration-api` 全部打 ✅；`:459-471` 又把 `useChangeClassification`、`usePriorityMatrix`、`useReports`、`useTemplateQuery`、`useCollaboration`、`useBatchOperations` 列为 ✅。上述 hook **零页面消费者**。
- `docs/product/itsm-commercial-capability-contract.md:20` 与 `docs/product/business-snapshot-2026-09-24.md:58` 把「优先级矩阵」列为**工单/事件 GA 候选**的业务证据；实测后端**没有任何含 priority 的路由**（`/api/v1/priority*` 不存在），前端只有死链。
- 报表域存在两套实现：权威活的是 `src/app/(main)/reports/page.tsx:3` → `components/business/AdvancedReporting` + `reports/hooks/useReportData.ts`（打到真实存在的 `GET /api/v1/reports`，`router/dashboard_routes.go:281`）；`src/lib/api/reports-api.ts` 是第二套死实现。模板域同理，活的是 `/tickets/templates` + `TicketApi`（`router/ticket_routes.go:39-46`），死的是 `template-api.ts`。

按 AGENTS「Doc drift is a defect」，这些 ✅ 与 GA 证据必须在同一批次里收回，不能只删代码。

## 五、契约与工程卫生的可计数欠债（门禁是绿的，因为它们是 advisory）

实测数字，全部由脚本产出：

- **67 条**列表信封违规（`*ListResponse` 缺 `items/total/page/pageSize/totalPages`），集中在 `dto/{ticket_workflow,standard_change,role,release,project,menu,asset_license,notification,change_pir,survey,tenant,service,msp,knowledge,cmdb*,cloud,change,known_error,asset}_dto.go`。
  `scripts/static-gates/check-pagination-shape.sh` 自己打印了这些违规却以 ADV 退出 0，注释写着"需独立 PR 修复"——即批次 0 第 2 项（信封收敛）只在被扫描的范围内收了口，**DTO 侧从未收敛**。
  → **E4-1/E4-2/E4-3（2026-10-03）已收口该条的判定与门禁部分**：重复扫描器删除、5.5 委托棘轮升硬、8 个死 DTO 清掉；债务本身按棘轮基线如实登记为 **40 条 / 33 个结构体**（E4-6a 按文档改判分页键规则后为 **37 条 / 30 个结构体**），逐模块收敛见「E4 收口进度」。原「67」是字段级计数口径（每缺一键算一条），与棘轮的结构体级条目不可直接比较。
- **13 处** handler 直接用非 `items` 键返回列表：`handlers/vendor/handler.go`(`list`)、`handlers/ticket_notification`(`notifications`)、`handlers/msp`(`reports`,`tickets`)、`handlers/problem_investigation/routes.go` 与 `handlers/knowledge/handler.go`(`tags`)、`handlers/common/handler.go`(`tenants`)、`handlers/bpmn/monitoring.go`(`instances`,`logs`)、`handlers/bpmn/dashboard.go`(`list`)、`handlers/ai/handler.go`(`results`,`data`)、`handlers/a2ui/handler.go`(`data`)。
- **11 条**重复路由注册（survey 7 + feishu 4）在 `handlers/*/handler.go` 的 `RegisterRoutes` 里是死代码；若哪天两处同时生效，gin 会直接 panic。**已修（E3-3，2026-10-02）**：两份副本删除，飞书四个安全分支与分组归属改由真实注册函数 `router.SetupFeishuRoutes` 的测试锁死，见「E3 收口进度」行 5b。
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
- 前端平行死实现：约 178 个调用、8 条不可达组件链。→ **E3-2 已清账 78 文件 / 22 503 行**；其中「8 条」里的 `ticket-relations-api` 经实测是在用的，未删（见 §四.0）。

## 七、建议批次（等拍板）

- **批次 E1｜点了就坏（P0，约 1 天）**：export 一行委派到已有实现；注册 attachment download/preview 两条 GET 并让 `fileURL` 与路由一致；标签写操作改指向 `/api/v1/ticket-tags` 并把菜单 `/admin/tags` 修到真实页面 + 二表收敛为一个所有者；依赖分析 GET 改查询参数绑定。每项配"修复前必红"的路由级回归（含跨租户拒绝）。
- **批次 E2｜假成功清零 + 申报口径收回（约 0.5 天）**：problem SLA / 服务分析 / 云发现 / vendor 一律改显式 unavailable 或删除死面；把 `handlers/*` 13 处非 `items` 键并入信封棘轮；同时收回 `frontend-menu-and-features.md` 的 6 条 ✅ 与两份产品文档里的「优先级矩阵 = GA 证据」。
- **批次 E3｜死实现大扫除（约 0.5 天）**：删掉第四节 8 条不可达链（约 10k LOC，先从两个 barrel 摘除引用再删文件，保留仍有真实调用者的 `ticket-service.ts`/`advanced-reporting`）与 11 条重复 `RegisterRoutes`；让"未实现"只剩一处真相。**进度（2026-10-02）**：前端不可达链已按 §四.0 的实测方法清账（E3-2，实删 78 文件 / 22 503 行，且实删集与本文「8 条」不同）；11 条重复 `RegisterRoutes` 已删并把飞书安全契约迁到真实注册函数上（E3-3，见「E3 收口进度」行 5b）；邮件接入队列的默认分页截断已按后端信封 + 前端服务端分页闭环（E3-4，见行 5）。**E3 批次四项全部收口。**
- **批次 E4｜契约与错误语义（1-2 天，可切片）**：67 条 DTO 违规按模块分批收敛，并把 `check-pagination-shape` 从 ADV 升为硬门禁；survey 等域的 `5001` 改按语义映射 404/409；274 处 `err.Error()` 直出逐步净化。**进度（2026-10-03）**：判定与门禁已收口——5.5 按实测改判为「委托棘轮 + exit 1」而不是给互相矛盾的旧规则加硬（E4-2），8 个零引用死 DTO 与其 10 条基线同批删除（E4-3），存量债务如实降为 **40 条 / 33 个结构体**并逐模块收敛（E4-6；E4-6a 把分页键判定改对齐文档后为 **37 条 / 30 个结构体**，E4-6b 已收口变更 PIR 列表的真分页键与参数夹紧）；实测新登记一条真实契约漂移 E4-4（服务请求列表三套形状 + 前端多字段 fallback，因消费页正被并发会话修改而延后）。`5001` 语义映射与 `err.Error()` 净化需先拍板错误分类法归一（E4-5），见「E4 收口进度」。
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
| 2c | 服务目录四个无消费方的死 hook 与 `ServiceAnalytics` 类型 | 已修（E3-2，2026-10-02） | `useServiceAnalyticsQuery`/`useFavoritesQuery`/`usePortalConfigQuery`/`useCatalogStatsQuery` 只被自己的测试引用：实测它们所在的 `src/lib/hooks/useServiceCatalog.ts` **18 个 hook 的外部引用数全为 0**（页面另有 route-local `src/app/(main)/service-catalog/hooks/useServiceCatalogData.ts` 承担同一职责），因此整文件与其测试一并删除，而不是只删四个。`ServiceAnalytics` 接口与 `getServiceAnalytics` 的 `unsupportedFeature` 桩、对应测试块同批删除（前端已无任何消费方，保留桩等于把「未实现」留在两处）。详见本文件「E3 收口进度」行 2c/§四.0 |
| 3 | `handlers/*` 其余非 `items` 键并入信封棘轮 | 已修（handler 层；DTO 结构体层别名并入 E4） | handler 层 10 个 `gin.H` 别名/伪造分页点收敛（另有 `known_error` stats 的伪分页删除）：BPMN dashboard+monitoring×2、MSP 两个报表与客户工单、`standard_change`、`ticket.GetTicketTemplates`、`known_error` stats+categories、`ticket_notification`。新增硬门禁 **5.10** `scripts/static-gates/check-list-envelope.sh`（分页字段必须与 `items` 同现、领域列表键黑名单），已接 `.github/workflows/backend-ci.yml` lint job；实测干净树 PASS、临时探针文件（`gin.H{"tickets":…, "page":1}`）FAIL rc=1、删除后回到 rc=0。**引入当轮即命中两处真实缺陷**：`GET /known-errors/stats` 伪造 `page/totalPages`，追查发现 `handlers/known_error/service.go` 用同一个 Ent 查询连做 8 个计数，而 Ent 的 `Where` 就地修改接收者（`ent/knownerror_query.go:36` `_q.predicates = append(...)`），后 7 个计数被 AND 串联 ⇒ `resolved`/`deprecated`/四个 severity 恒为 0，改 `Clone()` 后修复（修复前实测 `expected 1 / actual 0` 六条断言红）；`GET /known-errors/categories` 返回 `{items}` 而前端按 `categories` 取值再 `|| []` 兜底 ⇒ 分类筛选下拉恒为空。回归 `handlers/known_error/handler_test.go`（真实 `RegisterRoutes`，断言键集合精确、total 为全量计数、pageSize=2 时 total 仍为 3）、`handlers/bpmn/list_envelope_test.go`、`handlers/msp/list_envelope_test.go`。前端契约同步删除虚构字段与多键猜测：`types/msp.ts`（`customerName`/`slaComplianceRate` 等后端从不产出）、`msp-service.ts`（`Array.isArray(res.tickets)`、`res.total || 0`）、MSP 页面按虚构列渲染的表格、`known-errors` 页的 `response.categories \|\| []`、workflow audit 的 `result.list`、模板消费方 `templateResponse.templates`。实测：`go build ./...`、`go test ./handlers/... ./service/... ./dto/...` 全绿、staticcheck 无命中、`npm run type-check`/`lint:antd` 干净、前端全量 Jest 198 套件 3377 通过。契约见 `docs/api-reference.md`「不分页的列表」 |
| 3b | MSP 报表把 `user_id` 当 `tenant_id` 用（跨租户读数） | 已修 | `handlers/msp/handler.go` 的 `GetCustomerReports`/`GetPerformanceReports` 从认证上下文取 `user_id` 后传给按 `entTicket.TenantID(...)` 过滤的服务方法 ⇒ 调用者读到的是「ID 恰好等于自己用户 ID 的那个租户」的工单数；`?mspUserId=<任意用户ID>` 还能显式指定这个错位维度。现在聚合租户只取 `tenant_id`（缺失 401 fail closed），未实现的 `mspUserId` 员工维度改为 400 并说明原因（不静默忽略）。回归 `handlers/msp/handler_test.go`：构造 `caller.ID == foreignTenant.ID` 的 fixture（本租户 1 张 / 对方 2 张），断言两个报表都只计 1 张 —— 探针实测修复前两条断言都拿到 2。同批 `GetCustomerTicketsForMSP` 返回真实全量 `total`（此前只能 `len(page)` 冒充） |
| 2d | MSP 报表未按被服务的客户租户分组 | **待拍板**（不伪造补齐） | `GetMSPCustomerReports` 按单个 tenant 聚合，一次只产出一行汇总，而页面把它当「每客户一行」的报表展示。补齐需要产品决策：按 `mspCtx.AllowedCustomers` 逐客户出行（并定义 SLA 达成率/客户名的取数口径），还是明确改成「本租户区间汇总」。本批只统一字段契约（camelCase DTO + `items` 信封）并删除前端虚构列，口径不动 |
| 3c | 「分配历史」页签打的是不存在的 route | **已修（E3-1，2026-10-02）** | 实测确认 `msp_allocations` 有 `assigned_at`/`deassigned_at`/`role`/`msp_user_id`/`customer_tenant_id`，真实历史读不需要迁移，因此选择**接线而不是删除**：`handlers/msp/handler.go` 新增 `GetAllocationHistory`，`router/msp_routes.go` 以 `RequireMSPPermission("msp_allocation","read")` 注册 `GET /api/v1/msp/allocations/history`，`service.ListHistory` 的租户边界来自认证上下文（该表没有 tenant_id 列，只能通过 `HasMspUserWith(user.TenantIDEQ(...))` 收敛，Count/List 用 `Clone()` 复用条件）。顺带在同批修掉两个会立刻污染这份历史的真实缺陷：① `Create` 第 4 步把同一「员工×客户」下所有 `DeassignedAtNotNil` 的归档行 `SetDeassignedAt(now)`，即每次重新分配都改写历史结束时间（回归 `service/msp_allocation_service_test.go` 断言归档行保持 2026-02-01，修复前实测拿到当前时间即红）；② `Deactivate` 忽略影响行数，匹配不到活跃行照样返回成功，现改为 404/4004。`dto.MSPAllocationHistory`（声明 `deallocationReason`/`createdBy`/`createdByName`/`customerName`，全仓零生产者）删除，响应统一用 `MSPAllocationDTO` + 标准 `items` 信封；前端 `types/msp.ts` 换成 `MSPAllocationHistoryResponse`、页面删除「解除原因」列并把 `customerName` 改为 `customerTenantName`（`MSPAllocationDTO` 实际产出的键，此前分配列表/管理页那一列恒为空），`msp-service.getAllocationHistory` 去掉 `res || []` 的假成功兜底；`PRODUCT_CAPABILITIES.mspAllocationHistory` 打开并从 `DISABLED_API_CONTRACTS` 删除该豁免。回归：`router/msp_route_registration_test.go` 断言真实注册表里有该 GET（去掉注册即红，实测已证）；`handlers/msp/allocation_history_test.go` 覆盖跨租户排除、含已解除行、endDate 覆盖整天、缺上下文 401、畸形参数 400、deallocate 404；`npm run type-check` 与 msp-api/msp-service/api-contract 3 suites 44 tests 绿。**遗留**：解除原因/操作人要真正可查，需要给 `msp_allocations` 加落库列并写入审计，属新的契约面，不在本批伪造 |
| 4 | 收回 `frontend-menu-and-features.md` 的 ✅ 与两份产品文档的「优先级矩阵 = GA 证据」 | 已修 | §7 口径改为「✅ 只代表存在同名测试文件」，并把判定依据指向两份代码内清单（`api-contract.test.ts` 的 `KNOWN_UNMATCHED_FRONTEND_PATHS` 实测为空，未注册路径集中在 `product-capabilities.ts` 的 `DISABLED_API_CONTRACTS`）。实测改标 ⚠️ 共 11 行：7.1 五个 client（`change-classification-api`/`priority-matrix-api`/`batch-operations-api`/`collaboration-api`/`template-api`，router 里这四类路径零注册，通用 `/api/v1/templates` 也不存在，只有 `/api/v1/tickets/batch-delete` 而非 `batch/execute` 等）+ 7.2 六个 hook（外部引用实测：`src/components/{change-classification,priority-matrix,batch-operations,collaboration,templates,reports}` 六目录对目录外引用均为 0；`useCollaboration.ts`/`useReports.ts` 除自身测试外无人 import）。`itsm-commercial-capability-contract.md:21` 与 `business-snapshot-2026-09-24.md:58` 已把「优先级矩阵」从工单/事件 GA 候选证据列移除并写入缺口；同格「重大事件」经实测保留（`POST /api/v1/incidents/:id/major-incident` 已注册且有测试）。同文档顺带按实测纠正 capability 治理条目：§4.3/§9-6 的「`capabilityPathRules` 两处重复声明」已不成立（`Sidebar.tsx:15` 只 import `capabilityForPath`），§9-7 的四条规则已在 `menu-config.ts:44-47`；§4.2 补四行映射，§8.3 改为仍未保护的 `/sla-dashboard`、`/workflows`、`/improvements` |
| 4b | capability 控制平面把 `tag`/`template` 申报为 GA | 已修 | `handlers/capability/handler.go` 两条降为 `MaturityPilot`（标签双表未收敛、通用模板路由未注册），侧边栏按 `Sidebar.tsx:173` 显示 Pilot 徽标，`read`/`manage` 动作与鉴权不变。回归 `TestUnprovenCapabilitiesAreNotDeclaredGA` 打真实 `Handler`，临时改回 GA 实测即红 |
| 4c | 优先级矩阵后端是未接线的孤儿 service | **待拍板**（交 E5） | `service/priority_matrix_service.go`（含 `DefaultPriorityMatrix`、`GetMatrix`/`SetMatrix`/`CalculatePriority`/`ValidatePriority`）确实存在，但注入方法 `SetPriorityMatrixService` 全仓**零调用点**（实测只有自身定义），`priorityMatrixService` 字段在生产装配里恒为 nil ⇒ `service/incident_service.go:242,627` 的 impact×urgency→priority 派生分支永不执行；也没有任何 `/api/v1/priority*` 路由与矩阵持久化表。E5 需裁决：接线（补注入 + 明确矩阵是租户可配置还是固定 ITIL 默认，并决定要不要落库）还是退役（删 service 与两处死分支，优先级由调用方显式给定）。两者都不影响本批的文档收口 |

E2-1 附带发现（不在本批动，交 E5）：`ent/schema/vendor.go` 的 `code` 是**全局**唯一索引而不是 `(tenant_id, code)` 组合唯一，跨租户占用同编码会互相冲突；同一 schema 里 `vendor_type`/`contact_*`/`address`/`website` 全部是 Ent 必填字段（无 `Optional`），而 `CreateVendorRequest` 只把 `name`/`code` 声明为必填——这属于「后端可达性与归属裁决」条目，若 vendors 保留就必须连表结构与 DTO 一起重做。

### E3 收口进度（2026-10-02 起，逐项提交）

| # | 项 | 状态 | 证据 |
|---|---|---|---|
| 3c | MSP「分配历史」页签打不存在的 route | 已修 | 见 E2 表行 3c（同一批已改判为接线），提交 `06bc2332` |
| §四 | 前端平行死实现清账 | 已修 | 判定方式换成「入口 BFS + `tsc` 反证」，实删 **78 文件 / 22 503 行**：六个无路由 client（template/batch-operations/reports/collaboration/change-classification/priority-matrix）连同 hook、组件目录、类型与测试；`ticket-service-v2.ts`；`src/lib/templates/**`（9/1 543）；`src/lib/reports/report-engine.ts`（2/744）；`components/business/ticket-modal/**`（15/1 630）；`TicketDependencyManager.tsx`（748）；`lib/hooks/useServiceCatalog.ts`（18 hook 外部引用全为 0）。两个 barrel（`lib/api/index.ts`、`lib/services/index.ts`）摘掉死实现导出面；`DISABLED_API_CONTRACTS` 11→5 行（删除已不存在的实现所对应的豁免，`changeClassification` 因 `change-api.ts` 仍发 `/changes/templates/*` 保留）；行 2c 的 `ServiceAnalytics` 与 `getServiceAnalytics` 桩同批删。**审计判错的一条已纠正**：`ticket-relations-api` 真实在用（`TicketDetail.tsx:69` → `RelationPanel` → `useTicketRelations`），保留。契约测试现报 0 条路径不匹配（此前约 178 条靠豁免掩盖）。覆盖率门槛**未下调**：jest 只统计 `src/lib/**`，HEAD 基线 80.45% → 首删 78.91% → 删 `lib/templates` 80.17% → 删 `useServiceCatalog.ts` 79.93%（差 6 条语句），用新增 `src/lib/services/__tests__/email-intake-service.test.ts`（21 用例，逐项对照 `handlers/email_intake/handler.go` 已注册路由与三个请求 DTO：六个动作必带乐观锁 `version`、`override` 必带 `confirmed: true`、可选查询参数缺失时发 `undefined`）补到 80.39%。实测：`npm run type-check` 干净、`lint:check` 0 error（12 条存量 warning 均在本批未触达文件）、全量 jest **181 suites / 2 871 passed + 13 skipped**、`make docs-gate` 7 total / 0 failed |
| 5 | 邮件接入队列被后端默认分页截断，运维看到「第一页」当成整个待处理队列 | **已修（E3-4，2026-10-03）** | 实测把成因收敛到前端而不是后端：`ListConversations` 一直按 `page/pageSize` 出 `Offset/Limit` 并返回全量 `Count`，缺的是前端从不发送分页参数——`emailIntakeService.conversations(status)` 只发 `{status}`，于是恒命中后端默认 `pageSize=20`；页面又只用 `response.items`，把 `total` 丢掉，让 Antd `Table` 对已到手的 20 行做客户端翻页。**修法按契约闭环，不用 `pageSize=1000` 绕过上限**：service 签名改为 `conversations({page,pageSize,status?})` 并返回精确的 `ConversationListResponse{items,total,page,pageSize,totalPages}`；页面用 `current/pageSize/total` 受控分页、`showTotal` 显式给出总量、状态过滤归零页、旧响应由 `listRequest` 序号丢弃（不再让慢的上一页覆盖新过滤条件）。后端两处同批改：① 手拼 `gin.H{items,total,page,pageSize}` 换回 `common.SuccessWithList`，**实测修复前响应键为 `["items","page","pageSize","total"]`、缺 `totalPages`**，前端无从核对页数；② 排序补 `ent.Asc(ID)` 并列键——`last_message_at` 只有 `(tenant_id,status,last_message_at)` 复合索引、不是唯一列，同秒到达的会话在页边界归属不确定。**回归** `handlers/email_intake/conversations_list_test.go`（6 例，20+5 条同时间戳的租户 A 数据 + 3 条租户 B）：三页切分不重不漏且 `total=25` 恒为全量、status 过滤把 `total` 收敛到 5、租户 B 看不到任何租户 A 的 ID、7 页×4 条等于完整升序 ID 序列（诚实记录：sqlite 在无并列键时大概率也通过，这条是表面锁不是修复证明）、HTTP 层打**生产 `h.RegisterRoutes`** 断言信封键集合精确 + `page=2&pageSize=10` 的页长与 `totalPages=2`、缺租户上下文 401 且业务码非 0。负证明：还原 handler 那行旧 `common.Success` 后仅信封用例转红（实测 `expected [...,"totalPages"] actual [...4 keys]`），其余 5 例修复前即绿，说明它们锁的是既有正确行为。**同域另外 7 个 `{items,total}` 列表已逐个读服务层确认无静默截断**（`ListCustomers`/`ListBranches`/`ListSourceOrganizations`/`ListSupportContracts`/`ListExternalContractReferences`/`ListOnCallSchedules`/`ListShifts` 均为无 `Limit` 的 `All(ctx)`，`total = len(items)` 诚实），其缺 `page/pageSize/totalPages` 的信封形状归 E4 统一收敛，本批不改（改会动 7 个调用方契约）。实测：`go build ./...` 通过、`gofumpt -l handlers/email_intake` 空输出、`staticcheck ./handlers/email_intake/` 无命中、`go test ./handlers/email_intake/` 绿、`npm run type-check` 干净、全量 jest **181 suites / 2 871 passed + 13 skipped**、statements **80.39%**（门槛未下调，与 E3-2 收口时同值） |
| 5b | 11 条重复 `RegisterRoutes`（survey 7 + feishu 4） | **已修（E3-3，2026-10-02）** | 两份副本都是**从未被调用**的死注册（真实所有者分别是 `router/survey_routes.go:12` 与 `router/feishu_routes.go:12`，由 `router/router.go:643,658` 接线），删除后在原地留注释指明唯一所有者；survey 那条 2026-09-17 P0「越权写收口」的授权口径注释跟着所有者迁到 `router/survey_routes.go`，`handlers/feishu/handler.go` 因此不再引用 `middleware`，导入一并删除。**副本不只是重复**：feishu 那份的分组参数由调用方决定，旧测试正是以 `RegisterRoutes(api, api)`（同一分组既当 auth 又当 public）调用的，按该形态接线会把两条外部回调挪进鉴权组。因此原 `handlers/feishu/handler_test.go` 的 `TestWebhookSecurityContract` 测的其实是生产永不走的那条路径，属于「只覆盖孤立 helper、生产路由未接线」的违规：四个安全分支（合法签名放行、nonce 重放 403、时间戳过期 403、未知 instance 400）已迁到 `router/feishu_route_test.go` 并改打真实 `SetupFeishuRoutes`，另加注册表面精确等于 4 条、以及分组归属断言（`oauth/auth-url` 与 `sync/ticket/:ticket_id` 在鉴权哨兵之后，`oauth/callback` 与 `webhook` 必须在 public 分组；哨兵用 418 而非 401，才能把「分组归属」与 handler 自身缺租户上下文的 fail-closed 401 分开）。`router/survey_route_test.go` 锁住 `SetupSurveyRoutes` 的 7 条路由表面（此前 survey 域零测试）。**顺带命中一处真实漂移**：删掉副本后 `route_permission_guard_test.go` 的 `writeRouteExemptions` 中 `../handlers/feishu/handler.go POST /feishu/webhook/:instance_id` 变成失效条目，该守卫的白名单防腐化检查实测 FAIL 并打印「未命中任何已注册路由」，证明守卫真在跑，同批删除该条。**不新增「重复注册」静态门禁**（诚实记录取舍）：两处同时接线时 gin 直接 panic，属于大声失败而非静默腐化；而静态识别「不可达 route owner」需要调用图——`handlers/bpmn/` 的 7 个嵌套 `RegisterRoutes` 由同包 `handler.go:48-67` 转发、本身合法，文本启发式会假阳性，故本类问题由行为测试 + §四.0 死代码台账覆盖。实测：`go build ./...` 通过、`gofumpt -l handlers/feishu handlers/survey router` 空输出、`staticcheck` 三包无命中、`go test ./router/ ./handlers/feishu/ ./handlers/survey/` 全绿；**负证明**：把两个 `Setup*Routes` 的分组前缀改成 `/__disabled__` 后 shape 与分组归属测试转红，把 webhook 路径改名后安全契约测试首条即 404 红，恢复源文件后回到全绿且 `git diff` 无残留 |
| 6（新） | `middleware/rbac_precheck_gen.go` 与路由声明不同步（E3-1 漏生成） | **已修（2026-10-02）** | 全量套件实测 `--- FAIL: TestPrecheckMapIsFresh`。先在 HEAD 的独立 worktree 复现，确认与本批 E3-3 无关：E3-1（`06bc2332`）接线 `GET /api/v1/msp/allocations/history` 时只跑了 `./handlers/msp/` 与 `./router/` 窄测，没跑 `./middleware/`，生成物少这一条。按守卫指令 `go run ./cmd/authz-gen` 重新生成，`git diff` 实测只新增 `"/api/v1/msp/allocations/history": {msp_allocation, read}` 一行（与 `router/msp_routes.go` 行内权限一致），且**证明 E3-3 的两份副本删除对预检映射零影响**——副本与 router 那份的键完全重合，删除不改变生成物；`go test ./middleware/` 绿。教训：改路由的窄测必须含 `./middleware/`（预检生成物守卫在那里），不能只测注册所在包 |
| §四遗留候选 | 剩余 208 个闭包外文件 | **不做批量删** | 见 §四.0：清单只作候选，含「仅测试引用」这类需要产品判断的条目（如 `src/lib/pwa.ts` 只被自身测试引用、`src/components/auth/*` 只被自家 barrel 引用、`src/lib/utils/error-message-handler.ts` 不可达），逐文件裁定并各自配 tsc+jest 证明，不并入本批 |

### E4 收口进度（2026-10-03 起，逐项提交）

| # | 项 | 状态 | 证据 |
|---|---|---|---|
| E4-1 | 「67 条 DTO 违规」的存活与分型实测（先建台账再动手） | **已完成（2026-10-03）** | 按棘轮同款判定（AST 扫 `dto/*.go`，名称含 `List` 且带 `json:"total"`）实测：名称含 List 的结构体 **57** 个，其中 **18** 个是 `List*Request`/过滤器（无 `total`，本就不是信封，旧扫描器把它们也算成待收敛是数字虚高的一部分），**39** 个是列表信封；这 39 个里 **28** 个以 `ListResponse` 结尾、正是旧文本正则 `type [A-Za-z]*ListResponse struct` 的命中集，且 28 个**全部**含 `total` ⇒ 棘轮是旧扫描器的**严格超集**，额外覆盖 11 个旧扫描器完全失明的信封（`ListTicketsResponse`、`ListCIsResponse`、`ListProblemsResponse`、`ListAuditLogsResponse`、`ListTicket*Response`×5、`ListParticipantsResponse`）。违规分型（E4-3 删死码后实测）：领域名集合键 **25** + 分页键不完整 **8** + 分页别名 `size` **7** = **40 条 / 33 个结构体**（7 个结构体同时命中多类）。旧文本规则在本批删除前报 **67** 处字段级命中、删除后仍报 **47** 处（口径是「每缺一个字段算一条」，与棘轮「每结构体一条 + 缺失键并集」不可直接比较，`67 → 40` 不是同一把尺）。 |
| E4-2 | `check-pagination-shape.sh`（5.5）从 advisory 升为硬门禁 | **已修（2026-10-03，按实测改判：删重复扫描器而不是给旧规则加硬）** | **没有照 §七 的字面执行**，因为旧规则要求「所有 `*ListResponse` 必须含五元组」与 `docs/api-reference.md` 里已评审承认的「不分页的列表 `{items,total}`」直接冲突：照字面升硬等于锁死一条永远无法满足的规则，后续只会逼人放宽阈值。改判后的 5.5 只做委托：① 跑 `common` 分页序列化单测（`totalPages` 必须由 `NewListResponse` 真实算出），② 跑 `tests/contract/list_envelope_ratchet_test.go` 三条棘轮，任一失败 **exit 1**，③ 打印基线条目数让债务趋势可观察。自带的那套 `grep` + 60 行文本窗口扫描器整段删除——同一件债务写在两处正是 E4 要消除的第二套真相。文档同步：§5.5 改写为「两种合法形状 + 委托理由 + 超集证据」，并新增一张表把「谁真的阻断 CI」写清（结构体层=棘轮随 backend-ci `go test` 硬失败；handler 层=5.10；本地 `run-all.sh`=没有任何 workflow 调用，不阻断），避免「脚本变硬 = CI 变硬」的误读。**实测**：脚本 PASS 且打印「存量基线条目 40」；**负证明**——临时放入 `dto/zz_gate_probe.go`（`Foos []Foo json:"foos"` + `Size int json:"size"`）后脚本 exit 1，且失败原因同时命中 `TestListEnvelopeRatchet` 与 `TestListEnvelopePagingAliases` 两条（证明委托真在判定，不是恒 0），删除探针后回 exit 0。`totalPage`（少一个 s）旧拼写从脚本 grep 迁进棘轮的 `forbiddenPagingKeys`，判定面不缩小。 |
| E4-3 | 零生产引用的死 `*ListResponse` DTO | **已删 8 个（2026-10-03）** | 先摆引用计数再删：`git grep -w <Type> HEAD -- ':!itsm-backend/dto'` 逐个实测，8 个结构体在 itsm-backend 内的引用**只有定义自身与棘轮基线字符串**，无任何 handler/service/mapper/test 构造点（`PermissionListResponse`、`KnowledgeArticleVersionListResponse`、`KEDBListResponse`、`ServiceRequestListResponse`、`StandardChangeListResponse`、`ProjectListResponse`、`ConfigurationItemListResponse`、`SurveyListResponse`）。前端 `kedb-api.ts`/`service-request-api.ts`/`standard-change-api.ts` 里的同名标识是**独立声明的 TS 接口**，不引用 Go 类型，删除不改变任何线上响应。**这批删除顺带证明它们是「假契约」**：例如被删的 `dto.ServiceRequestListResponse` 声明 `{items,total,page,size}`，而生产 `handlers/service_request/handler.go:283,369` 实际返回 `{items,total,page,pageSize,totalPages}`——死结构体既不被使用、又不与线上形状一致，留着只会误导下一个改契约的人。同批从三条基线删掉对应 10 条目（50 → **40**）。实测：`go build ./...` 通过、`go test ./tests/contract/ ./dto/... -count=1` 全绿（双向棘轮不会容忍「删了结构体却忘记收基线」）、`gofumpt -l dto/ tests/contract/` 空输出、staticcheck 无命中、全量 `go test ./...` **RC=0 / 0 条 FAIL**（87 个含测试的包）。 |
| E4-4（新） | 服务请求列表同时存在三套形状，前端用被禁止的多字段 fallback 猜 | **待修（本批只登记）** | 实测：生产返回 `{items,total,page,pageSize,totalPages}`（`handlers/service_request/handler.go:283-289,369-375`，两处还是手拼 `map[string]interface{}` 而非 `common.SuccessWithList`）；被删的死 Go 结构体是第二套（`size`、缺 `totalPages`）；前端 `src/lib/api/service-request-api.ts:54` 的 `ServiceRequestListResponse` 是第三套（`{requests,total,page,size}`），靠 `:194` 的 `normalizeList` 用 `raw.requests \|\| (raw as any).items \|\| []`、`raw.size \|\| (raw as any).pageSize \|\| requests.length` 兜住——正是 AGENTS「请求/响应多字段兼容零新增」与「类型安全零新增」两条明令禁止的写法，且 `total` 缺失时静默变成「当前页长度」。**不在本批动**：修复要同时改 `src/app/(main)/service-requests/page.tsx`，该文件此刻正被另一个会话修改（共享工作区），并改会覆盖在途工作。修法已定：后端两处手拼换 `common.SuccessWithList`，前端类型改单键 `{items,total,page,pageSize,totalPages}`、删 `normalizeList`，调用点与 jest 夹具同批。 |
| E4-5 | survey 等域 `5001` 一律当内部错误 + `err.Error()` 直出 | **待办（下一片）** | §五 的 **274** 处 `common.Fail(..., err.Error())` 实测复现（`handlers/**` 非测试行）。并行错误分类法实测：`common.AppError` 构造点在 `handlers/{change,cmdb,incident,problem,service_catalog,service_request,standard_change,ticket}` 与 `service/` 共 **207** 处，其余 handler 各自 `switch ent.IsNotFound/IsConstraint`——两套真相；而 `common.FailWithErr`（294 处引用）只 sanitize 文案、HTTP status 恒为 5001，不做 404/409 语义映射，所以「把 5001 改按语义映射」必须先决定统一走哪一套，不能逐 handler 手改切片。 |
| E4-6 | 剩余基线逐模块收敛 | **进行中（2026-10-03 起切片）** | 每收敛一条=改 json tag 为 `items`（真分页端点补 `page/pageSize/totalPages`，确认不分页的端点保留诚实 `{items,total}`）+ 同步 Mapper、真实路由契约测试、前端 `src/lib/api` 类型与调用点、文档，最后从对应基线删除该条。E4-6a/6b 已把 `envelopeKeyBaseline` 从 8 条收到 **5 条**（余下全是 CMDB：`page` + `size` 顶替 `pageSize`、无 `totalPages`，服务层实测真分页，见 `service/ci_tag_service.go:102` 一类），这 5 条与 7 条 `size` 别名重叠、应同批做；之后再按模块啃 25 条领域名集合键。 |
| E4-6a | 分页键棘轮与「不分页的列表」契约互相矛盾 | **已修（2026-10-03）** | `TestListEnvelopeKeys` 原本对每个信封都要求五元组，把 `docs/api-reference.md` 明确承认、且**禁止伪造分页键**的 `{items,total}` 也算成债务——与被 E4-2 删掉的旧扫描器是同一类判定错误（只是这次在守卫侧）。改判为「出现过 `page`/`pageSize`/`totalPages` 任一键才要求凑满」，并逐条读服务层分型而不是批量放行：`service/ticket_view_service.go:28`、`service/ticket_comment_service.go:104` 均为无 `Limit` 的 `All(ctx)`、handler 里 `total=len(items)` 诚实 → 移出基线；`dto/change_pir_dto.go\|ChangePIRListResponse` 实测真做 `Count` + `Offset/Limit` → 走 E4-6b 补键。**不是放宽**：注入「只有 `page`」的探针实测 `--- FAIL: TestListEnvelopeKeys`（`[zz_probe_partial.go\|ListProbePartialResponse\|pageSize,totalPages]`，rc 1），注入纯 `{items,total}` 的探针 rc 0；两个探针随即删除。基线 8 → 5，合计 40 条 / 33 结构体 → **37 条 / 30 结构体**（脚本打印同步），`docs/testing/static-analysis-gates.md` §5.5 记下两种形状边界。 |
| E4-6b | 变更 PIR 列表真分页却只回 `{items,total}`，且 `page/pageSize` 完全不夹紧 | **已修（2026-10-03）** | 实测两处独立缺陷叠加：① `dto.ChangePIRListResponse` 只有两个键，调用方无法核对页数；② `handlers/change/handler.go:955` 用裸 `strconv.Atoi` 把参数原样交给 `service/pir_service.go`，而 Ent 的 sqlgraph 是 `if q.Limit != 0 { selector.Limit(q.Limit) }` —— **`pageSize=0` 等于不加 `LIMIT`，实测 25 条数据一次返回 25 条**；`page<=0` 会算出负 `OFFSET`。修法按契约闭环：DTO 补 `page/pageSize/totalPages`，值由 `common.NewPaginationResponse` 统一算（除法只有一份，避免历史 `totalPages` 除零溢出类别再犯）；handler 改走 `common.GetPaginationFromQuery`（1/20、上限 100），service 先 `common.ValidatePagination` 归一再落 `Offset/Limit`；排序补 `ent.Asc(changepir.FieldID)` 并列键（`review_date` 非唯一列）。swagger 的「默认 10」改成「默认 20，上限 100」并 `make swagger-gen`，实测三件套各只变 1 行。前端 `change-api.ts` 的 `PIRListResponse` 补三键、`changes/pirs/page.tsx` 去掉 `\|\| []` / `\|\| 0` 兜底并加请求序号防旧响应覆盖新过滤条件。回归 `router/change_pir_route_test.go`（7 例）打在**生产注册** `SetupRoutes → SetupChangeRoutes → RequirePermission`：五键精确集合、三页（10/10/5）拼接等于完整升序 ID、`pageSize=0` 夹紧、`page=-1` 回落第 1 页、`result=failed` 把 `total` 收敛到 5、空结果序列化 `[]`、租户 B 只见自己 3 条且不含租户 A 的 ID、未认证 401。**负证明**：把 `dto/change_pir_dto.go`+`service/pir_service.go`+`handlers/change/handler.go` 三个文件临时还原到 HEAD（用 `git show HEAD:<path>` 覆盖，修复本身未提交所以即修复前形态），四条转红——信封断言实测 `expected [items,page,pageSize,total,totalPages] actual [items,total]`、`pageSize=0` 实测 `should have 20 item(s), but has 25`；另三条（result 过滤、空 `[]`、租户隔离）修复前即绿，锁的是既有正确行为。测试解码目标声明为线上形状的 `pirListWire` 而非 `dto.*`，因此修复前也能编译、证明的是运行时行为。**登记一条不在本批修的债**：`/api/v1/changes/pirs` 从未出现在 `docs/api-reference.md`（全文 grep `pirs` 0 命中），文档补写归 E4 文档同步项。实测：`gofumpt -l dto service handlers/change router tests/contract` 空、staticcheck 五包无命中、`go test ./tests/contract/ ./router/ ./handlers/change/ ./service/` 全绿、5.5 门禁 rc 0 打印 37、`npm run type-check` 干净。 |
