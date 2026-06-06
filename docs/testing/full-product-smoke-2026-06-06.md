# ITSM 全产品烟测记录 - 2026-06-06

## 环境

- 前端: `http://localhost:3000`, `itsm-frontend-prod`
- 后端: `http://localhost:8090`, `itsm-backend-prod`
- 浏览器: Chrome 插件会话
- 账号: `admin`

## 验证命令

- `cd itsm-frontend && npm run type-check`
- `cd itsm-frontend && npm run build`
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

## 限制与说明

- Playwright CLI 在当前 macOS 沙箱中无法启动 Chromium，报 `MachPortRendezvousServer Permission denied (1100)`，因此浏览器验证改用 Chrome 插件完成。
- 生产前端重建期间出现多次外部 TLS 连接重试，最终构建成功并重启 `itsm-frontend-prod`，容器状态为 healthy。
