# ITSM v1.0 GA 收口验收指南

本文档用于开源 v1.0 GA 前的产品默认能力、部署和安全验收。

## 核心入口

```bash
curl http://localhost:8090/api/v1/readiness/ga
```

该接口返回：

- `modules`: 租户、权限菜单、角色、服务目录模板、SLA 模板、审批流、流程绑定、CMDB 类型、标准变更、连接器、AI、Audit 的 GA 状态。
- `checks`: API 响应、租户隔离、连接器生命周期、AI 可追踪、产品默认初始化的收口状态。
- `summary`: 已就绪模块数量。

连接器生命周期：

```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:8090/api/v1/connectors/lifecycle
```

AI 审计记录：

```bash
curl -X POST http://localhost:8090/api/v1/ai/audit \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "scenario": "ticket_triage",
    "input_ref": "ticket:TCK-0001",
    "prompt_version": "triage-v1",
    "model": "configured-provider",
    "confidence": 0.86,
    "suggestion": {"priority": "high", "category": "identity"},
    "accepted": true
  }'
```

## 必过验证

后端：

```bash
cd itsm-backend
go test ./...
```

前端：

```bash
cd itsm-frontend
npm run type-check
npm run build
```

端到端主流程：

- 登录与菜单加载。
- 工单创建、分派、处理、解决、关闭、评价。
- 事件创建、确认、评论、根因、影响评估、关闭。
- 问题根因、Known Error、Workaround。
- 变更风险评估、CAB/审批、发布窗口、PIR。
- 服务目录申请、审批、交付任务。
- 知识库搜索、推荐、评审。
- CMDB CI 关系、拓扑、影响分析。
- SLA 告警、违规、合规报表。
- 连接器安装、启用、健康检查、测试发送。
- AI 分诊、知识推荐、处理总结、审计记录。
- RBAC 权限隔离与跨租户拒绝。

## 部署模式

每种模式都需要独立初始化验证：

```bash
DEPLOYMENT_MODE=private docker compose up -d --build
DEPLOYMENT_MODE=saas docker compose up -d --build
DEPLOYMENT_MODE=saas_msp docker compose up -d --build
```

每次启动后检查：

- `/api/v1/health` 返回可用。
- `/api/v1/readiness/ga` 返回 `ga_candidate`。
- `admin / admin123` 可登录开发环境。
- 菜单、角色、权限、服务目录、SLA、流程绑定、CI 类型、连接器 lifecycle 可用。
- 默认初始化不生成虚构 Incident、Problem、Change 或真实资产实例。

## 默认初始化配置

默认初始化文件：

```bash
itsm-backend/config/seed/default.json
```

建议随系统初始化的产品配置：

- 组织结构、团队、角色、权限、菜单。
- 服务目录模板、SLA 模板、审批流模板、流程绑定。
- CMDB 类型、标准变更模板、Known Error 字段模板、标签。
- 连接器市场清单、AI 审计契约和后续 Prompt 模板。

不建议默认初始化的业务数据：

- 真实工单、真实事件、真实问题、真实变更。
- 真实资产实例、真实客户数据、真实用户业务数据。
- 只为演示而构造的故障故事线。

企业自定义初始化：

```bash
export ITSM_SEED_CONFIG=/path/to/company-seed.json
docker compose run --rm itsm-init
```

配置合并规则：

- 省略字段：沿用系统内置默认。
- 提供非空数组：覆盖对应默认模板。
- 提供空数组：明确关闭该类初始化，例如 `"incidents": []`。

## GA 默认边界

- v1.0 优先保证私有化部署、产品默认模板、核心 ITIL 闭环、AI 可追踪和连接器市场雏形。
- AI 建议默认不自动执行高风险动作，必须保留人工采纳/拒绝记录。
- MSP 白标、计费、套餐和配额归入 v1.1/v1.2，不阻塞 v1.0 GA。
