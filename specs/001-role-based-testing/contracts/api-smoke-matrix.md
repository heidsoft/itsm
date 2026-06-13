# Contract: API 冒烟矩阵（≥25 端点）

**Feature**: 001-role-based-testing
**Authority**: 与 spec FR-204 / SC-204 / 源文档 §六绑定。
**Token**: 默认 `admin / admin123`。
**Base**: `http://localhost:8090`。

## 矩阵

| # | 分组 | 方法 | 路径 | 期望 | 验证字段 | 触发角色 |
|---|------|------|------|------|----------|----------|
| 1 | health | GET | `/api/v1/health` | 200 | `status=ok` | * |
| 2 | readiness | GET | `/api/v1/readiness/ga` | 200 | `data.modules.length=12` | super_admin |
| 3 | auth | POST | `/api/v1/auth/login` | 200 | `data.access_token` | * |
| 4 | auth | GET | `/api/v1/auth/me` | 200 | `data.username` | * |
| 5 | menus | GET | `/api/v1/auth/menus` | 200 | `data.length>=20` | super_admin |
| 6 | tickets | GET | `/api/v1/tickets` | 200 | `code=0` | * |
| 7 | tickets | POST | `/api/v1/tickets` | 200 | `data.ticketNumber ~ /TKT-\d{6}-\d{6}/` | end_user |
| 8 | incidents | GET | `/api/v1/incidents` | 200 | `code=0` | engineer |
| 9 | incidents | POST | `/api/v1/incidents` | 200 | `code=0` | engineer |
| 10 | problems | GET | `/api/v1/problems` | 200 | `code=0` | engineer |
| 11 | changes | GET | `/api/v1/changes` | 200 | `code=0` | engineer / approver |
| 12 | configuration-items | GET | `/api/v1/configuration-items` | 200 | `code=0` | engineer |
| 13 | configuration-items | POST | `/api/v1/configuration-items` | 200 | `data.id` | engineer |
| 14 | ci-types | GET | `/api/v1/configuration-items/types` | 200 | `code=0` | tenant_admin |
| 15 | knowledge | GET | `/api/v1/knowledge-articles` | 200 | `code=0` | end_user |
| 16 | knowledge | POST | `/api/v1/knowledge-articles` | 200 | `data.id` | engineer |
| 17 | service-catalog | GET | `/api/v1/service-catalogs` | 200 | `code=0` | end_user |
| 18 | sla | GET | `/api/v1/sla/definitions` | 200 | `data.length>=6` | tenant_admin |
| 19 | sla-monitoring | POST | `/api/v1/sla/monitoring` | 200 | `code=0` | sd_manager |
| 20 | connectors | GET | `/api/v1/connectors/lifecycle` | 200 | `data.length>=5` | super_admin |
| 21 | bpmn-process | GET | `/api/v1/bpmn/process-instances` | 200 | `code=0` | engineer / approver |
| 22 | workflow | GET | `/api/v1/workflow/instances` | 200 | `code=0` | approver |
| 23 | approval-workflows | GET | `/api/v1/approval-workflows` | 200 | `code=0` | approver |
| 24 | process-bindings | GET | `/api/v1/process-bindings` | 200 | `code=0` | super_admin |
| 25 | ai-triage | POST | `/api/v1/ai/triage` | 200 | `code=0` | engineer |
| 26 | ai-audit | POST | `/api/v1/ai/audit` | 200 | `code=0` | engineer |
| 27 | dashboard | GET | `/api/v1/dashboard/overview` | 200 | `code=0` | sd_manager |
| 28 | analytics-tickets | GET | `/api/v1/analytics/tickets` | 200 | `code=0` | sd_manager |
| 29 | audit-logs | GET | `/api/v1/audit-logs` | 200 | `code=0` | security |
| 30 | users | GET | `/api/v1/users` | 200 | `code=0` | super_admin |
| 31 | tenants | GET | `/api/v1/tenants` | 200 | `code=0` | super_admin |
| 32 | roles | GET | `/api/v1/roles` | 200 | `code=0` | super_admin |

## 移出 GA 矩阵（NC-002 决议）

- `/api/v1/templates/tickets` — 后端无实际路由；前端 `/tickets/templates` 入口同步下架。
- `/api/v1/incident-rules` — 后端无实际路由；列入 post-GA backlog。
- `/api/v1/process-instances` — 路径错误，已用 #21/#22 真实路径替代。
- `/api/v1/reports/*` — 后端报表未实现，post-GA。

## 退出码约定

| Exit | 含义 |
|------|------|
| 0 | 全部通过 |
| 1 | 至少 1 项失败（具体在 stdout `[FAIL]` 行） |
| 2 | 前置（登录 / 取 token）失败 |

## 安全用例（A05/A07/A09/A10 走冒烟）

| 用例 | 方法 | 期望 |
|------|------|------|
| A05 SERVER_MODE 校验 | 启动期检查 | `release` 时 CORS 必须显式 |
| A07 暴力登录 5 次 | POST `/auth/login` × 5 错误密码 | 第 6 次返回 429 + audit `login_failed` |
| A09 关键操作审计 | 创建工单 → GET `/audit-logs?resource=ticket&action=create` | 至少 1 条 |
| A10 SSRF | POST 连接器 webhook URL = `http://127.0.0.1:22` | 返回 4xx 拒绝 |
