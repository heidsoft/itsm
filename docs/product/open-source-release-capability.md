# Open Source Release Capability

## CAPABILITY

开源用户可以从干净检出开始，通过文档化的环境变量、Docker Compose 和健康检查路径，把 ITSM 平台部署成一个可登录、可验证、可运维的实例。产品承诺聚焦核心 ITSM 闭环：认证、工单、事件、问题、变更、知识库、服务目录、服务请求审批、CMDB、SLA 与工作流。

## CONSTRAINTS

- 默认部署路径必须以 `docker compose` 和 `.env.example` / `.env.prod.example` 为入口，不依赖本地私有配置。
- 前端默认后端地址统一为 `http://localhost:8090`；Docker 生产构建使用 `http://itsm-backend:8090`。
- API 健康检查以 `/api/v1/health` 为准。
- 所有用户可见的写入操作必须真实持久化，或显式提示未接入后端；禁止用本地模拟数据伪装成功。
- 默认账号只能作为开发/首次安装引导出现，生产部署文档必须要求更改密码和密钥。
- 多租户、RBAC、审批和服务请求状态流转属于发布门禁，不能作为演示功能处理。

## IMPLEMENTATION CONTRACT

- Actors: 系统管理员、服务台工程师、审批人、普通员工、开源部署者。
- Surfaces: `README.md`、`.env.example`、`.env.prod.example`、`docker-compose.yml`、`docker-compose.prod.yml`、前端自助门户、后端 `/api/v1/*`。
- States and transitions: 服务目录项只能在后端支持的 `enabled` / `disabled` 与前端展示的 `published` / `retired` 间转换；服务请求至少支持创建、查看、审批、拒绝、完成。
- Interfaces: 前端 API 适配层负责 camelCase/snake_case 转换，不能泄露后端 Ent 模型；服务目录能力必须以 `/api/v1/service-catalogs` 和 `/api/v1/service-requests` 为真实来源。
- Data implications: 收藏、评分、门户配置、服务分析、目录导出在后端模型和接口补齐前，不应显示为可持久化能力。
- Operator requirements: 发布前需要通过前端 `type-check`、前端生产构建、后端 `go test ./...`、Docker Compose 健康检查和登录烟测。

## NON-GOALS

- 不在本能力中新增商业化计费、多组织 SaaS 订阅或云市场分发。
- 不承诺所有规划中的 AI 能力在无 LLM Key 时完整可用；必须有清晰降级或关闭入口。
- 不用前端假数据替代未完成的后端服务。

## OPEN QUESTIONS

- 是否把服务目录收藏、评分、门户配置作为 v1.0 必须功能，还是明确归入 v1.1？
- 生产安装是否需要内置初始化向导，用于修改管理员密码、JWT secret 和对象存储配置？
- 开源版与商业版是否需要能力矩阵，避免用户误解哪些功能已经可用？

## HANDOFF

当前能力可以进入直接实现和验证阶段。下一步优先补齐发布门禁：统一环境变量、删除用户可见假成功路径、完善首次安装文档，并用 smoke test 固化登录与核心 ITSM 流程。
