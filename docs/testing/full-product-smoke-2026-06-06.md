# ITSM 全产品烟测记录 - 2026-06-06

## 环境

- 前端: `http://localhost:3000`, `itsm-frontend-prod`
- 后端: `http://localhost:8090`, `itsm-backend-prod`
- 浏览器: Chrome 插件会话
- 账号: `admin`

## 验证命令

- `cd itsm-frontend && npm run type-check`
- `cd itsm-frontend && npm run build`
- `docker compose --env-file .env.prod -f docker-compose.prod.yml build itsm-backend itsm-frontend`
- `docker compose --env-file .env.prod -f docker-compose.prod.yml up -d --no-deps --build itsm-frontend`
- `docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"`

## Chrome 覆盖范围

| 模块 | 路由 | 结果 |
| --- | --- | --- |
| 系统管理 | `/admin` | 通过 |
| 用户管理 | `/admin/users` | 通过 |
| 角色管理 | `/admin/roles` | 通过 |
| 权限管理 | `/admin/permissions` | 通过 |
| 组管理 | `/admin/groups` | 通过 |
| 系统配置 | `/admin/system-config` | 通过 |
| 工单管理 | `/tickets` | 通过 |
| 创建工单 | `/tickets/create` | 通过 |
| 事件管理 | `/incidents` | 通过 |
| 创建事件 | `/incidents/create` | 通过 |
| 问题管理 | `/problems` | 通过 |
| 新建问题 | `/problems/new` | 通过 |
| 变更管理 | `/changes` | 通过 |
| 新建变更 | `/changes/new` | 通过 |
| 知识库 | `/knowledge` | 通过 |
| 知识审核 | `/knowledge/reviews` | 通过 |
| 发布管理 | `/releases` | 通过 |
| 新建发布 | `/releases/new` | 通过 |
| SLA 监控 | `/sla-monitor` | 通过 |
| SLA 配置 | `/admin/sla-definitions` | 通过 |
| SLA 总览 | `/sla` | 通过 |
| CMDB | `/cmdb` | 通过 |
| CI 类型 | `/admin/cmdb-types` | 通过 |
| 云资源 | `/cmdb/cloud-resources` | 通过 |
| 拓扑 | `/cmdb/topology` | 通过 |
| 工作流 | `/workflow` | 通过 |
| 流程设计器 | `/workflow/designer` | 通过 |
| 流程实例 | `/workflow/instances` | 通过 |
| 待审批 | `/approvals/pending` | 通过 |
| 服务目录 | `/service-catalog` | 通过 |
| 服务请求 | `/service-requests` | 通过 |
| 资产管理 | `/assets` | 通过 |
| 许可证 | `/licenses` | 通过 |
| 报表 | `/reports` | 通过 |

## 接口巡检

以下只读接口已通过登录态 + 租户头验证，均返回 `HTTP 200` 且 `code = 0`：

| 模块 | 接口 | 结果摘要 |
| --- | --- | --- |
| 当前用户 | `/api/v1/auth/me` | 返回超级管理员用户信息 |
| 角色 | `/api/v1/roles` | 20 条角色 |
| 权限 | `/api/v1/permissions` | 58 条权限 |
| 部门 | `/api/v1/org/departments/tree` | 14 个部门节点 |
| 团队 | `/api/v1/org/teams` | 18 个团队 |
| 事件 | `/api/v1/incidents` | 8 条事件 |
| 问题 | `/api/v1/problems` | 返回问题分页数据 |
| 问题统计 | `/api/v1/problems/stats` | 返回 open / resolved / closed 等统计 |
| 变更 | `/api/v1/changes` | 返回变更分页数据 |
| 变更统计 | `/api/v1/changes/stats` | 返回 draft / pending / completed 等统计 |
| 知识库 | `/api/v1/knowledge/articles` | 6 篇文章 |
| 知识统计 | `/api/v1/knowledge/stats` | 返回分类、发布数、浏览量等统计 |
| 发布 | `/api/v1/releases` | 5 条发布 |
| 发布统计 | `/api/v1/releases/stats` | 返回 draft / scheduled / completed 等统计 |
| SLA 定义 | `/api/v1/sla/definitions` | 6 条定义 |
| SLA 统计 | `/api/v1/sla/stats` | 返回定义数、违规数、达标率 |
| SLA 指标 | `/api/v1/sla/metrics` | 返回 metrics 列表 |
| CMDB | `/api/v1/configuration-items` | 当前 0 条 CI，列表接口正常 |
| CMDB 统计 | `/api/v1/configuration-items/stats` | 返回 active / inactive / total 等统计 |
| CI 类型 | `/api/v1/configuration-items/types` | 8 条类型 |
| 云资源 | `/api/v1/configuration-items/cloud-resources` | 当前 0 条云资源，接口正常 |

## 多角色权限验证

使用 `admin`、`user1(end_user)`、`security1(agent-like)` 三个账号对关键接口进行了权限矩阵验证。

### 验证通过

| 角色 | 验证点 | 结果 |
| --- | --- | --- |
| `admin` | 角色、权限、知识库、发布、SLA、CMDB、事件、问题、变更、用户管理接口 | 全部可访问 |
| `end_user` | 角色、权限、发布、SLA、CMDB、事件、问题、变更接口 | 正确返回 `403` / `权限不足` |
| `agent/security1` | 角色、权限、知识库、发布、SLA、CMDB、事件、问题、变更接口 | 正确返回 `403` / `权限不足` |

### 发现的权限风险

| 编号 | 现象 | 风险 |
| --- | --- | --- |
| RBAC-01 | `end_user` 和 `agent` 可直接访问 `/api/v1/users`，并返回包含 `admin/user1/security1` 的完整用户列表 | 用户信息越权暴露 |
| RBAC-02 | `end_user` 的 `/api/v1/auth/menus` 返回了 `用户管理`、`部门管理`、`团队管理`、`SLA配置`、`工单分类`、`工作流` 等管理菜单 | 前端暴露不应可见的管理入口 |
| RBAC-03 | `agent/security1` 的 `/api/v1/auth/menus` 返回了 `用户管理`、`工作流` 等管理菜单 | 菜单权限与接口权限不一致，容易形成误导或后续越权入口 |

## 写入闭环验证

以下模块已执行“创建 -> 验证 -> 删除”临时数据闭环，验证后已清理，无残留测试数据：

| 模块 | 操作 | 结果 |
| --- | --- | --- |
| 角色 | 创建临时角色并删除 | 通过 |
| 知识库 | 创建临时文章并删除 | 通过 |
| 发布 | 创建临时发布并删除 | 通过 |
| CMDB | 创建临时 CI 并删除 | 通过 |
| 问题 | 创建临时问题并删除 | 通过 |
| 事件 | 创建临时事件并删除 | 通过 |
| 变更 | 创建临时变更并删除 | 通过 |
| 部门 | 创建、更新、删除临时部门 | 通过 |
| 团队 | 创建、更新、删除临时团队 | 通过 |

## 生命周期动作验证

为覆盖 ITIL v3 生命周期里的关键状态推进，本轮额外验证了以下动作：

| 模块 | 动作 | 结果 |
| --- | --- | --- |
| 事件 | `/api/v1/incidents/:id/escalate` | 通过，返回升级记录 |
| 变更 | `/api/v1/changes/:id/submit` | 通过，状态从 `draft` 进入 `pending` |

## 系统配置现状

| 项目 | 结果 |
| --- | --- |
| `/api/v1/system-configs` 列表 | 返回成功，但当前租户配置数为 `0` |
| `/api/v1/system-configs/status` | 可正常返回 CPU / memory / goroutines 等运行状态 |
| 配置详情 / 更新验证 | 因当前无初始化配置数据，未能执行真实配置项更新回滚 |

## 重建后回归

本轮在确认“今天前后端源码已有较多改动”后，执行了完整的重建后验证：

| 项目 | 结果 |
| --- | --- |
| 前端生产镜像 | 重新 build 成功 |
| 后端生产镜像 | 重新 build 成功 |
| 生产服务 | `itsm-backend-prod`、`itsm-frontend-prod`、`itsm-nginx-prod`、`postgres`、`redis`、`minio` 全部 `healthy` |
| 前端连通性 | `http://127.0.0.1:3000/login` 返回 `HTTP 200` |
| 后端连通性 | `http://127.0.0.1:8090/api/v1/health` 返回 `HTTP 200` |
| 管理员登录 | 通过 `/api/v1/auth/login` 成功获取 token 并可访问 `/api/v1/auth/me` |
| 关键统计接口 | `/knowledge/stats`、`/releases/stats`、`/sla/stats`、`/configuration-items/stats` 均返回 `code = 0` |

### 重建后浏览器烟测

使用 `itsm-frontend/screenshot.js` 基于 Playwright 对以下页面执行了登录后截图烟测：

`/dashboard`、`/tickets`、`/incidents`、`/problems`、`/changes`、`/knowledge`、`/workflow/designer`、`/workflow/dashboard`、`/sla-monitor`、`/cmdb`、`/assets`、`/msp/management`

截图已保存到 `docs/images/`。

### 重建后新增问题

| 编号 | 页面/模块 | 现象 | 判断 |
| --- | --- | --- | --- |
| WF-01 | `/workflow/designer` | 控制台报错 `unsupported configuration <keyboard.bindTo>`，随后 `Failed to create blank diagram: Error: no diagram to display` | BPMN 设计器初始化与当前 `bpmn-js` 配置不兼容 |
| WF-02 | `/workflow/dashboard` | 页面请求返回 `HTTP 404`，控制台记录 `Failed to fetch dashboard metrics` | 前端 `bpmn-dashboard-api` 与后端已注册路由不匹配，存在接口路径偏差 |

### 重建后修复与复测

已对上述两条工作流问题完成前端修复并重新 build 镜像、重启前端服务：

| 编号 | 修复 | 结果 |
| --- | --- | --- |
| WF-01 | `BPMNDesigner` 去掉已不兼容的 `keyboard.bindTo` 配置，空白流程改为 `modeler.createDiagram()` 初始化 | 复测通过，`/workflow/designer` 无控制台报错，截图成功 |
| WF-02 | `bpmn-dashboard-api` 统一补全 `/api/v1/bpmn/dashboard` 前缀 | 复测通过，`/workflow/dashboard` 正常打开并完成截图 |

## 已修复问题

1. `/admin/groups` 渲染角色权限对象时触发 React 错误边界。
   - 原因: 后端返回的权限可能是 `{ id, code, name }` 对象，页面直接当字符串渲染。
   - 修复: 权限字段统一格式化为字符串，兼容字符串和对象两种返回形态。

2. `/admin/system-config` 获取系统状态时读取 `startTime` 失败。
   - 原因: 系统状态接口可能返回空对象或不同命名字段。
   - 修复: 兼容 `startTime`、`start_time`、`startedAt`、`started_at`、`uptime`、`upTime`，缺失时显示 `未知`。

3. 后端生产 Dockerfile 与 distroless 镜像不匹配。
   - 原因: Dockerfile 构建阶段和 healthcheck 使用 `sh`/`wget`，distroless runtime 不提供 shell。
   - 修复: 改为 Alpine 静态二进制运行环境，创建非 root 用户并保留 HTTP healthcheck。

4. `/workflow/designer` 空白流程初始化失败。
   - 原因: 当前 `bpmn-js` 版本不再接受 `keyboard.bindTo` 显式绑定，且手写极简 BPMN XML 不能稳定生成可展示图。
   - 修复: 移除不兼容键盘绑定，改用 `modeler.createDiagram()` 初始化空白流程图。

5. `/workflow/dashboard` 请求路径不匹配导致 404。
   - 原因: 前端 API 使用 `/bpmn/dashboard/*`，后端实际注册在 `/api/v1/bpmn/dashboard/*`。
   - 修复: `bpmn-dashboard-api` 统一改为 `/api/v1/bpmn/dashboard` 基路径。

## 限制与说明

- Playwright CLI 在当前 macOS 沙箱中无法启动 Chromium，报 `MachPortRendezvousServer Permission denied (1100)`，因此浏览器验证改用 Chrome 插件完成。
- 本轮重建后浏览器烟测改用仓库内 `itsm-frontend/screenshot.js` 执行，登录和主路径截图成功，但暴露出两条工作流相关前端问题（见 `WF-01`、`WF-02`）。
- 生产前端重建期间出现多次外部 TLS 连接重试，最终构建成功并重启 `itsm-frontend-prod`，容器状态为 healthy。
- 后端重建最初被 `alpine:3.20` 拉取超时阻塞，重试后成功下载并完成镜像构建。
- `incident/escalate` 与 `change/submit` 不是空 body 动作接口，调用时必须提交 JSON 请求体；使用正确请求体后接口行为正常。
- 系统配置模块当前缺少初始化数据，导致配置列表为空；这更像环境准备缺口，不是接口崩溃。
