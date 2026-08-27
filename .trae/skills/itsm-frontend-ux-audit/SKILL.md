---
name: "itsm-frontend-ux-audit"
description: "End-to-end UX audit for the ITSM frontend running locally: verifies every menu route, flags 404/mock-data/i18n/drift issues, and outputs a graded report. Invoke when user asks to review/check/audit the frontend UX or do a functionality retrospective."
---

# ITSM Frontend UX Audit (ITSM 前端体验巡检)

本 Skill 固化一套对本仓库 Next.js/Gin ITSM 系统的**端到端前端体验巡检流程**。执行结果是一份分级报告，涵盖"路由可达性、数据真实性、i18n 完整性、视觉一致性、菜单/路由契约一致性"五大维度。

## 1. 触发条件

用户使用以下关键词时立即触发本 Skill：
- 「前端体验/巡检/复盘」
- 「功能开发情况」
- 「UX audit / check frontend / frontend QA」
- 「菜单可达性检查」或「页面404排查」

本 Skill **只读**，不修改任何代码；如需修复问题，巡检完成后由用户确认再走修复分支。

## 2. 前置环境假设

- Docker Compose 生产或开发环境已启动，前端在 `http://localhost:3000` 可访问，后端在 `http://localhost:8090` 健康（可用 `GET /api/v1/health` 验证）。
- 已有可用管理员账号（从 `.env.prod` 或初始化脚本读取，**禁止硬编码账号密码**到 Skill/日志/响应中）。
- 浏览器工具（TRAE-browseruse 内置浏览器或 TRAE-browseruse-external）可用。

## 3. 巡检入口与登录

### 3.1 基线验证
```
登录前必须先做：
① GET /api/v1/health → 断言 code=0
② GET / → 页面包含登录表单
```
失败则直接中止巡检并输出「环境不可用」分段诊断。

### 3.2 登录流程
- 优先复用已有 session/cookie；若页面仍未登录再走账号密码流程。
- **禁止**把登录凭据打印到日志、响应或 Skill 内容中；凭据来源：用户现场输入 / 已授权的 .env 读取 / 或沿用浏览器已有 session。
- 登录成功判定：跳转到 `/dashboard` 或包含「总览 / Overview」字样的页面标题。

## 4. 巡检 URL 清单（强制全量覆盖）

以下清单必须**逐页访问**。菜单入口必须**先从菜单管理页 `/admin/menus` 读取真实记录**，然后与下表做并集比较；任何菜单声明路径不在下表、或下表路径在菜单里不存在，都记为「菜单/路由契约漂移」。

### 4.1 核心业务域（5 个）
| 页面 | 主 URL | 备选 / 正确路径（如有差异需要记录） |
|------|--------|-----------------------------------|
| 仪表盘 | `/dashboard` | |
| 工单列表 | `/tickets` | |
| 事件管理 | `/incidents` | |
| 问题管理 | `/problems` | |
| 变更管理 | `/changes` | |

### 4.2 支撑域（6 个）
| 页面 | 主 URL | 备注 |
|------|--------|------|
| CMDB 总览 | `/cmdb` | 检查 4 个能力卡是否带 GA/Pilot 标签 |
| 服务目录 | `/service-catalog` | 检查评分控件默认态 |
| 知识库 | `/knowledge` | 检查草稿/发布状态区分 |
| SLA 实时监控 | `/sla-monitor` | 注意：菜单管理常写成 `/sla-dashboard`，漂移必须记录 |
| 报表中心 | `/reports` | 报表入口数量核对 |
| 工单报表 | `/reports/ticket-report` | |

### 4.3 工作流与审批（4 个）
| 页面 | 主 URL | 备注 |
|------|--------|------|
| BPMN 设计器 | `/workflow/designer` | 进入后检查是否弹出模板对话框 |
| 工作流实例 | `/workflow/instances` | 关注 endTime 零值格式化 |
| 审批中心 | `/workflow/approvals` | |
| BPMN 节点分析 | `/tools/bpmn-node-analyzer` | 工具类页面 |

### 4.4 扩展域（5 个）
| 页面 | URL |
|------|-----|
| 发布管理 | `/releases` |
| 资产管理 | `/assets` |
| 许可证管理 | `/licenses` |
| MSP 管理 | `/msp` |
| 审计日志 | `/admin/audit-logs` |

### 4.5 管理后台（8 个 + 3 个候选校验）
| 页面 | 菜单记录常见值 | 真实可达路径（以浏览器真实结果为准） |
|------|--------------|----------------------------------|
| 用户管理 | `/admin/users` | `/admin/users` |
| 角色管理 | `/admin/roles` | `/admin/roles` |
| 组管理 | `/admin/groups` | `/admin/groups` |
| 部门管理 | `/admin/departments` | `/admin/departments` |
| 团队管理 | `/admin/teams` | `/admin/teams` |
| 租户管理 | `/admin/tenants` | `/admin/tenants` |
| 菜单管理 | `/admin/menus` | `/admin/menus`（用来取权威菜单列表） |
| CI 类型管理 | `/admin/ci-types` | `/admin/ci-types` |
| 系统配置 | `/admin/system-config` | `/admin/system-config` |
| 工单分类 | `/admin/ticket-types` ❌ → 常配错 | `/admin/ticket-categories` ✅ |
| SLA 配置 | `/admin/sla-config` ❌ → 常配错 | `/workflow/sla` ✅ |
| 升级矩阵 | `/admin/escalation-matrix` ❌ → 通常缺失 | 不存在时列入"页面需补建或菜单需删除" |

**规则**：任一路由在声明值处返回 404 但候选路径 200 → 归类为「菜单路径漂移 P0」。两处都 404 → 归类为「页面缺失 P0」。

## 5. 每页必查项（采集 checklist）

对每个成功加载的页面，至少采集并记录以下 10 项：
1. **URL 与 page heading**：页面是否有对应 `<h2>` 主标题；标题内容是否与菜单一致。
2. **面包屑**：是否与 Banner 中的面包屑重复（重复记 P3）。
3. **统计 KPI 合理性**：数字逻辑自洽性检查（例如"总 0 + 达成率 92.5%"自相矛盾 → P0 mock）。
4. **表格数据存在性**：非种子页应有 0 或真实多条；种子页（服务目录 22、CI 类型 8、菜单 30、角色 20、SLA 6 等）数量核对。
5. **空态处理**：无数据页是否显示空状态 + "去创建"按钮（没有则 P2）。
6. **零值时间戳**：`endTime` / `deletedAt` / `closedAt` 等可选时间列，未填充时应为 `-`，不得显示 `1/1/1 ...`。
7. **i18n**：任何位置出现 `xxx.xxx`（形如 `common.totalLabel`）的裸 key 都记 P2。
8. **分页控件**：每页大小是否合理；列表 total 是否与后端返回一致。
9. **可交互控件可用性**：搜索框、筛选下拉、Tab（列表/看板/分析）都要点击一次，确认无 JS 错误。
10. **默认选中异常**：如星级评分全 checked、Checkbox 默认 checked 等视觉误导，记 P2。

## 6. 问题分级标准

| 等级 | 命名 | 判定条件 |
|------|------|---------|
| P0 | 阻断 | 菜单点击 404、核心 KPI 硬编码与真实数据矛盾、升级矩阵等关键管理页缺失 |
| P1 | 高 | 接口重复调用导致异常审计日志、系统监控全 0（健康数据未接）、零值时间戳直接渲染、审计日志字段缺失 |
| P2 | 中 | i18n key 未翻译、统计卡片重复、主标题缺失、默认全选 UI Bug、表格列错位、空状态无引导 |
| P3 | 低 | 面包屑重复、菜单配置路径与别名双轨（但跳转可用）、文案小瑕疵、无影响功能的交互细节 |

## 7. 报告输出格式

巡检完成后必须输出 **6 段结构化中文报告**，禁止只罗列零散问题。

### 7.1 总体完成度
一个 4 列 Markdown 表格：能力域、入口数、GA / Pilot / 404、完成度百分比。

### 7.2 已验证可用清单
「✅ 模块名 + 亮点一句话」列表，证明不是"只找问题"。

### 7.3 P0 阻断级问题
逐条：问题标题 + 复现 URL + 现象描述 + 根因猜测（路由漂移 / 页面缺失 / Mock 残留）。

### 7.4 P1 高级问题
逐条：问题标题 + 复现 URL + 实锤证据 + 修复方向建议（不写代码）。

### 7.5 P2 / P3 中低级问题
合并分节列出，按页面聚合，每条不超过 3 行。

### 7.6 修复优先级清单
表格：优先级 / 任务简述 / 预估工作量（小时，精确到 0.1h）。最后给出合计工作量。

## 8. 效率建议：批量访问

允许使用 `integrated_code_mode/Exec` 在一个块内 for-of 批量执行 4~6 个页面的 `navigate → wait(1.5s) → snapshot`，输出中使用 `=== [页面名] (URL) ===` 分段标记，便于后处理。

严禁在循环中做交互点击（不稳定）；交互点击只对关键页面单独执行。

## 9. 边界与安全

- **绝不**在 Skill 或最终报告中泄露账号、密码、JWT、API Key、数据库地址。
- **绝不**包含具体用户姓名、邮箱、IP 的明细。审计日志举例时用「user_admin」等通用名。
- 发现菜单权限码与前端路由声明不一致时，不擅自修改菜单数据；只在报告中标注「需修复 menu_id=X 的 path 字段」。
- 所有代码改动请求必须由用户确认；本 Skill 默认只出报告不做修改。
