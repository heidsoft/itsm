# ITSM 生产模式部署与回归记录 - 2026-09-24

> Status: current

## 环境

- 本地生产栈：`docker-compose.prod.yml`（`itsm-nginx-prod` / `itsm-frontend-prod` / `itsm-backend-prod` / `itsm-worker-prod` / `itsm-postgres-prod` / `itsm-redis-prod` / `itsm-ai-service-prod`）
- 入口：`http://localhost`（nginx）；后端就绪 `GET /api/v1/health`、`GET /api/v1/readyz`
- 数据库：栈内本地库 `itsm_prod`；未触碰任何真实业务库

## 镜像与提交溯源

| 目标提交 | `90208859` |
| --- | --- |
| `itsm-backend:90208859` | `2c3e21602f48`（运行于 `itsm-backend-prod` / `itsm-worker-prod`） |
| `itsm-frontend:90208859` | `db7c93d67ed7`（运行于 `itsm-frontend-prod`） |

- 两个镜像同时重打 `:latest`，以匹配 compose 的 `${VERSION}` 镜像命名约定
- 清理：回收悬空镜像 39MB，删除过期 tag `itsm-backend:3d91b61b`、`itsm-frontend:3d91b61b`；基础镜像与数据卷未动

## 执行命令

- `make build-backend build-frontend VERSION=90208859`
- `docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --force-recreate itsm-backend itsm-worker itsm-frontend`
- 迁移执行者：一次性容器 `itsm-init-prod`，退出码 0（启动期迁移失败会阻断启动）

## 迁移落地校验（栈内 `itsm_prod`）

`migrations/20260923_tenant_scope_baseline_unique_keys.sql` 的三个组合唯一索引均已建立：

- `ticketcategory_tenant_id_code`
- `tag_tenant_id_code`
- `processdeployment_tenant_id_deployment_id`

## 回归结果

| 项 | 结果 |
| --- | --- |
| 容器健康 | 7/7 healthy |
| 镜像溯源 | 运行镜像 ID 与新构建一致（排除陈旧镜像） |
| 入口可达 | `GET /api/v1/health` 200；前端 `/` 200 |
| fail-closed | 未认证 `GET /api/v1/tenants/:id/initialization` → 401 |
| 登录态 UI | 登录表单 → 落地 `/dashboard`；`/admin/tenants` 渲染正常、侧边菜单可见 |
| 租户初始化状态接口 | `GET /api/v1/tenants/1/initialization` → `ready=true`、6 组件 verified、`commandStatus=none`（该租户创建于 outbox 机制之前，如实报告）、`templateVersion=1.0.0` |
| 浏览器质量门 | 零 console 错误、零失败请求 |

登录凭据仅经 shell 变量与测试进程环境注入，未打印、未落盘、未进入命令文本。

## 已知问题（非本次部署引入）

1. `handlers/msp` 的 `TestGetCustomerTickets_WithMSPContext` 在 `90208859` 失败：`85a9859f` 的 MSP 客户租户隔离逻辑与其测试不一致（期望 200，实际 403「无权访问指定客户租户」）。
2. `tests/e2e/auth-utils.ts` 登录工具过期：内置默认口令已失效（401），且仍读取令牌 Cookie 化后登录响应不再返回的 `accessToken`，会影响仓库 e2e 套件。

## 边界声明

- 未放宽任何门禁，未使用个人 LLM 密钥；AI 能力不在本轮验收范围。
- 本记录只覆盖本地生产栈；真实生产环境的存量数据前滚与生产验收属于初始化改造第四阶段，尚未授权执行。
