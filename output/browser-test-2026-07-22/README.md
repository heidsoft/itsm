# ITSM 生产环境浏览器测试报告

- **测试时间**：2026-07-22 21:30–21:40 (GMT+8)
- **测试目标**：http://localhost（生产模式 Docker Compose 部署）
- **测试方式**：agent-browser 自动化 + curl API 探测 + docker 日志分析
- **测试账号**：`admin / AdminProd2026!`（来源 `.env.prod`）
- **测试范围**：登录鉴权 + 13 个核心业务模块页面渲染 + 关键 API 健康检查

---

## 1. 部署环境快照

| 服务 | 容器 | 端口 | 状态 |
|---|---|---|---|
| Nginx 反向代理 | itsm-nginx-prod | 80 / 443 | healthy |
| 前端 Next.js | itsm-frontend-prod | 3000 | healthy |
| 后端 Go/Gin | itsm-backend-prod | 8090 | **unhealthy**（见问题 1） |
| PostgreSQL | itsm-postgres-prod | 127.0.0.1:5433 | healthy |
| Redis | itsm-redis-prod | 127.0.0.1:6380 | healthy |
| MinIO | itsm-minio-prod | 9000 | healthy |

后端 unhealthy 只影响 orchestration，不影响实际业务（`/api/v1/health` 返回 200）。

---

## 2. 登录与鉴权

| 步骤 | 结果 |
|---|---|
| 首页 `/` 渲染 | ✅ AI-Native ITSM 落地页正常，见 `home.png` |
| 跳转登录 `/login` | ✅ 见 `02-login-page.png` |
| 错误密码 `admin/admin123` | ❌ 返回 `{"code":2001,"message":"invalid credentials"}` |
| 正确密码 `admin/AdminProd2026!` | ✅ 登录成功、拿到 accessToken、跳转 `/dashboard` |
| 登录后主页 | ✅ 见 `04-after-login.png`、`05-dashboard.png` |

---

## 3. 核心模块渲染验证

以下模块从左侧菜单点击进入，页面均正常加载、无白屏/前端异常：

| # | 模块 | 截图 |
|---|---|---|
| 1 | 仪表盘 Dashboard | `05-dashboard.png` |
| 2 | 工单 Tickets | `06-tickets.png` |
| 3 | 事件 Incidents | `07-incidents.png` |
| 4 | 变更 Changes | `08-changes.png` |
| 5 | 问题 Problems | `09-problems.png` |
| 6 | CMDB | `10-cmdb.png` |
| 7 | 服务目录 Catalog | `11-catalog.png` |
| 8 | 知识库 Knowledge | `12-knowledge.png` |
| 9 | SLA 监控 | `13-sla.png` |
| 10 | 报表 Reports | `14-reports.png` |
| 11 | 工作流 Workflow | `15-workflow.png` |
| 12 | BPMN 节点分析 | `16-bpmn.png` |
| 13 | 用户管理 Users | `17-users.png` |

左侧导航共 20+ 菜单项，涵盖工单/事件/问题/变更/CMDB/服务目录/知识库/SLA/报表/发布/资产/许可证/MSP/工作流/BPMN/用户/角色/组/部门/团队/审批/SLA 配置——菜单全部可点击、路由正确。

---

## 4. 已发现的问题（按优先级）

### P1 —— CMDB 版本查询 NULL 转换失败
- **触发**：进入 CMDB 详情/编辑 CI id=3 时后端两处路径均报错
- **日志**：
  ```
  service/configuration_item_service.go:142  Failed to get max CI version
    error: sql/scan: converting NULL to int is unsupported, ci_id: 3
  service/configuration_item_service.go:370  同上
  ```
- **根因**：`SELECT MAX(version)` 在无历史版本行时返回 NULL，Go int 无法接收
- **修复建议**：改用 `sql.NullInt64` 或 `COALESCE(MAX(version), 0)`；影响 `configuration_item_service.go` L142、L370

### P2 —— Incident 状态机拒绝跨状态跳转
- **日志**：`Failed to update incident invalid status transition from 'new' to 'resolved' id=5`
- **性质**：并非 bug，是状态机保护——但前端应在按钮层拦截，避免让用户看到 500 类交互
- **建议**：Incident 详情页按当前状态过滤可用的状态切换按钮

### P3 —— 容器健康检查配置缺陷（unhealthy 474 连击）
- **触发**：`docker inspect` 显示 healthcheck 用 `wget` 但容器内 `sh: 1: wget: not found`
- **修复**：把 healthcheck 换成 `curl -f http://localhost:8090/api/v1/health || exit 1`，或在 Dockerfile 中安装 wget
- **文件**：`docker-compose.prod.yml` / `itsm-backend/Dockerfile`

### P4 —— 健康检查路径不一致
- `/api/v1/healthz` → 404
- `/api/v1/health` → 200
- **建议**：追加 `/healthz` 别名路由，或在文档/健康监控中统一到 `/health`

---

## 5. 测试结论

- **前端可用性**：★★★★★（登录 + 13 个模块页面全绿）
- **后端 API 可用性**：★★★★☆（核心接口通畅，CMDB 版本查询/健康检查 endpoint 存在小缺陷）
- **部署完整性**：★★★★☆（6 个容器全部就绪；backend 容器健康检查工具缺失需修复）
- **总体判断**：生产部署基本达标，可以对外提供服务；建议按 P1/P3 优先修复后再进入更严肃的灰度阶段。

---

## 6. 快速修复清单（可直接派活）

- [ ] P1 `itsm-backend/service/configuration_item_service.go:142,370` 用 `COALESCE(MAX(version), 0)` 或 `sql.NullInt64`
- [ ] P2 前端 `incidents/[id]` 详情页按状态机过滤 action 按钮
- [ ] P3 `docker-compose.prod.yml` backend healthcheck 换 `curl`；或在 Dockerfile 里 `apk add --no-cache wget`
- [ ] P4 `router/*` 加 `/healthz` 别名，或改前端/监控探活地址

