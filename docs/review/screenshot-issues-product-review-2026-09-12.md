# 截图问题产品复盘与改进方案（2026-09-12）

## 范围与结论

本方案基于 2026-09-12 一轮浏览器截图反馈（10 处）与会话内已提交修复（7 处）复盘。结论：这些不是孤立 bug，而是 5 类系统性缺陷的表象。项目 `AGENTS.md` 已写明正确规则（契约单一事实来源、fail-closed 守卫、能力成熟度语义、去 AI 文案），但缺少把这些规则自动化落地的门禁，导致同类问题在不同页面反复出现。

本次不连接生产做完整 ITIL 验收，只针对截图暴露的契约、守卫、文案与呈现问题定级并给出可落地改进项与验收标准。

## 一、根因归类（而非按页面）

| 截图问题 | 根因类别 | 证据 |
| --- | --- | --- |
| SLA 监控页 Invalid Date | 契约漂移 | 前端 `resourceType/startTime/deadline` ← 后端实返 `ticketNumber/violationTime/slaName`（`handlers/sla/entity.go`） |
| 审批负责人列显示「1」 | 契约漂移 | `approvals/page.tsx:299` 渲染 `record.assignee`（用户 ID），后端 `dto.BPMNTaskResponse` 无 `assigneeName` |
| 我的请求状态英文 | 契约漂移 | 前端 camelCase key vs 后端 snake_case（已修 `bcaf8ba4`） |
| 新建组成员列表空 | 契约漂移 + 缺数据 | `pageSize` 越界 + 候选用户加载 |
| 新建租户保存失败 | 契约/UX 不一致 | `CreateTenantRequest.Code` 校验 `alphanum` 拒绝下划线，UI 用 `finops_111` |
| 暂停租户致全站锁死 | 缺领域守卫 | `service/tenant_service.go:87 UpdateTenantStatus` 无任何守卫 |
| CMDB「GA」 | 能力呈现失真 | 后端 `ga` vs README「GA候选」；disabled/unknown 塌缩成 Pilot（已修 `074a6825`） |
| 5 处文案 AI 味 | 产品语言未定稿 | 9 个页面含「使用提示/怎么用/使用顺序」长句块 |
| 通知列表密集 | 信息密度/排版 | 纯 UI |
| 值班排班名称是下拉 | 表单控件类型错 | 已修 `d6e57e1f` |

## 二、系统性诊断

1. **契约测试只测 URL，不测响应字段（最高危）**。`src/lib/__tests__/api-contract.test.ts` 仅比对路径/method（`byFile/parserGap/staleEntries`），字段名、大小写、分页结构未覆盖，故「字段对不上」类 bug 必然逃逸。附带 `incident-api.ts:376` 仍留被禁止的 `response.incidents ?? response.items ?? response.data` 三重兜底。
2. **破坏性/锁定操作缺 fail-closed 守卫**。可暂停默认/系统租户，一暂停全站中间件 `2003`，且页面「恢复」按钮本身也 403，只能进数据库救。
3. **能力成熟度呈现无统一组件**。仅 `CSDMHub`、`Sidebar` 各写各的 maturity 判断，无共享 `CapabilityBadge`；CSDMHub 把 disabled/unknown 塌缩成 Pilot 的模式会被下一个能力页重犯。
4. **假成功/静默降级**。SLA 页把字段缺失渲染成 `Invalid Date`、破折号，违反「禁止把契约错误伪装成空结果」。
5. **产品文案无 voice 规范**。运维界面长句、分号排比、括号举例，机器感强。

## 三、改进项与验收标准

### A. 立即修复（点状）

| # | 项 | 验收标准 |
| --- | --- | --- |
| P0-1 | 默认/系统租户暂停守卫 | `UpdateTenantStatus`/`CreateTenant` 对 `code='default'` 或系统租户置 `suspended/expired/deleted` 返回 409；前端禁用入口；含跨守卫回归测试 |
| P0-2 | SLA 监控页契约 | 改路径 `/api/v1/sla/violations`、`SLAViolation` 类型与表格列用后端真实字段；无 `Invalid Date`；窄测覆盖字段映射 |
| P0-3 | 审批负责人列 | 后端 `BPMNTaskResponse` 增 `assigneeName`，service 批量解析；前端渲染姓名，无值显示「未领取/未识别」而非 ID |
| P1-4 | 租户编码校验 | `alphanum` 放开下划线（或正则 `^[a-z0-9_]+$`），或前端强制拦截并提示；新建 `finops_111` 成功 |
| P1-5 | 新建组成员空 | 候选用户正确加载、`pageSize` 合规；能选人并保存 |
| P2-6 | 4 处 AI 文案 | 权限说明/工单分配规则/升级矩阵/工作流管理按菜单口语化标准重写 |

### B. 系统防复发（结构性）

| # | 项 | 验收标准 |
| --- | --- | --- |
| S-7 | 契约测试升级为字段级 | 对每个列表接口断言 `{items,total,page,pageSize,totalPages}` 与关键 camelCase 字段；CI 红即拦 |
| S-8 | 抽 `CapabilityBadge` 共享组件 | 统一 ga/pilot/disabled/unready/unknown 五态，替换 CSDMHub/Sidebar 各自实现 |
| S-9 | 消灭 `?? response.x ?? response.y` 兜底 | 全仓扫描清零，按唯一 DTO 读取（先清 `incident-api.ts`） |
| S-10 | 错误态组件约定 | 字段缺失/能力未就绪显式 error/empty 态，禁止 `Invalid Date` |

### C. 流程/门禁

| # | 项 | 验收标准 |
| --- | --- | --- |
| G-11 | 文案 lint | 增量检查命中 Alert/Collapse 长句（含「；」「（如」「怎么用」）告警，纳入 review |
| G-12 | 破坏性操作清单 | 暂停/删除/禁用类端点必须带守卫测试 |

## 四、优先级

- **P0（阻断/事故）**：P0-1 租户守卫、P0-2 SLA 页、P0-3 负责人列
- **P1（功能不可用）**：P1-4 租户编码、P1-5 组成员、S-7 契约字段级测试、S-9 兜底清理
- **P2（体验/一致性）**：P2-6 文案、S-8 CapabilityBadge、S-10 错误态、G-11/G-12 门禁

## 五、本会话状态

- 已提交：CSRF 缓存失效 + 服务请求审批 i18n（`8bb41d78`）、工单详情返回按钮 + Groups pageSize（`c92c392e`）、我的请求状态/审批步骤 i18n（`bcaf8ba4`）、值班排班表单（`d6e57e1f`）、服务目录信息条（`ce45ed6a`）、CMDB badge（`074a6825`）、菜单文案（`93456724`）。
- 运行时处置：误暂停默认租户已用 SQL 恢复 `active`（非代码 bug）。
- 待办：P0-1/P0-2/P0-3 本轮落地；P1/P2/S/G 后续批次。
