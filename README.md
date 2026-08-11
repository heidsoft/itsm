<div align="center">

# AI-Native ITSM

面向国内企业的开源 IT 服务管理平台

ITIL 流程 · BPMN 编排 · CMDB · SLA · 知识库/RAG · 多租户 · 企业连接器

[![Go](https://img.shields.io/badge/Go-1.25.12-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Next.js](https://img.shields.io/badge/Next.js-15.5-black?logo=next.js)](https://nextjs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-6.0-3178C6?logo=typescript&logoColor=white)](https://www.typescriptlang.org/)
[![License](https://img.shields.io/badge/License-Apache--2.0-green)](./LICENSE)
[![Backend CI](https://github.com/heidsoft/itsm/actions/workflows/backend-ci.yml/badge.svg)](https://github.com/heidsoft/itsm/actions/workflows/backend-ci.yml)
[![Frontend CI](https://github.com/heidsoft/itsm/actions/workflows/frontend-ci.yml/badge.svg)](https://github.com/heidsoft/itsm/actions/workflows/frontend-ci.yml)
[![Stars](https://img.shields.io/github/stars/heidsoft/itsm?style=flat)](https://github.com/heidsoft/itsm/stargazers)

[简体中文](./README.md) · [English](./README.en.md) · [日本語](./README.ja.md)

[快速开始](#快速开始) · [能力边界](#能力与成熟度) · [生产部署](#生产部署) · [文档中心](./docs/README.md) · [参与贡献](#参与贡献)

</div>

![ITSM 仪表盘](./docs/images/01-仪表盘.png)

## 项目定位

ITSM 用一套可审计、可扩展的后端规则连接服务台、事件、问题、变更、服务请求、SLA、CMDB、知识和企业协作系统。项目目标不是堆出更多菜单，而是让真实企业流程能够连续运行：失败可恢复、权限可验证、操作可追踪、结果可验收。

项目坚持四个原则：

- **流程是主线**：BPMN 是统一编排层，业务状态机仍由各领域服务负责。
- **CMDB 是上下文**：配置项、关系和影响范围进入事件、问题、变更与服务请求，而不是停留在资产列表。
- **AI 是决策支持**：分诊、摘要、检索和建议可降级、可审计，不绕过权限和人工责任。
- **异步动作必须可靠**：工作流启动和关键通知通过事务 command/outbox、租约、fencing、重试和死信执行，不依赖请求内 goroutine。

> 当前处于商业化收敛阶段。核心 ITIL 能力已经具备可运行基础，但不同领域成熟度不同。代码或页面存在不等于已达到生产承诺；请以[商业能力契约](./docs/product/itsm-commercial-capability-contract.md)和对应验收结果为准。

运行时能力以认证接口 `GET /api/v1/capabilities` 为唯一事实来源。菜单和工作台必须同时满足构建可用、部署就绪、租户就绪及用户操作权限；仓库内的成熟度表用于发布说明，不替代运行时判断。

## 适用场景

- 企业 IT 服务台统一受理、分派和跟踪员工请求。
- 运维团队将事件、CI、SLA、问题和变更串成治理闭环。
- 数字化平台团队通过 BPMN 配置审批和跨系统流程。
- 私有化、SaaS 或 SaaS + MSP 模式下的多组织服务管理。
- 基于 Go、Next.js 和开放接口进行二次开发。

## 能力与成熟度

成熟度定义：

- **GA 候选**：核心模型、规则和主要接口已存在，可以进入企业生产验收。
- **Pilot**：存在真实实现，但跨模块闭环、运维或测试仍需补齐。
- **Disabled/规划中**：骨架或入口不构成可交付能力，不应作为生产承诺。

| 能力域 | 当前状态 | 已有基础 | 进入生产前重点 |
|:---|:---:|:---|:---|
| 工单与事件 | GA 候选 | 状态流转、分派、CI、SLA、BPMN、租户隔离 | 固化事件恢复旅程和容量验收 |
| 变更管理 | GA 候选 | 风险、受影响 CI、审批、回滚方案、PIR 基础 | 影响分析门禁、窗口冲突与回滚演练 |
| 问题与 Known Error | Pilot | 根因、临时方案、关联事件、知识沉淀基础 | 强化 CI 引用和知识发布闭环 |
| 服务目录与请求 | Pilot | 目录、请求、审批、服务任务基础 | 目录版本、交付补偿和 CI 变更闭环 |
| CMDB 核心 | GA 候选 | CI 类型、配置项、关系、历史、拓扑、影响分析 | 数据质量、规模和恢复验收 |
| CMDB 云发现 | Pilot | 阿里云适配与连通基础 | Job/Worker/Diff/对账、密钥服务和退役治理 |
| BPMN 与审批 | Pilot | 定义、绑定、实例、任务、变量、历史 | 继续迁移剩余非可靠触发路径 |
| SLA | GA 候选 | 策略、截止时间、预警、违规、指标 | 工作日历、暂停恢复和跨领域统一 |
| 知识与 RAG | Pilot | 文章、关键词/向量检索、问答降级 | 发布版本、可见性和索引删除一致性 |
| AI | Pilot | LLM Gateway、分诊、摘要、RAG、审计框架 | 统一 evaluator、反馈和高风险动作治理 |
| 通知与连接器 | Pilot | 可靠通知 outbox、投递审计、连接器框架 | 真实渠道健康检查、回调验签和重放运维 |
| RBAC/多租户 | GA 候选 | 角色权限、Endpoint ACL、租户过滤、审计 | 按领域持续补权限矩阵和跨租户回归 |

CMDB 的正式与试点边界见 [CMDB 商业 MVP](./docs/product/cmdb-commercial-mvp.md)。

## 快速开始

### 环境要求

- Docker Desktop 或兼容的 Docker Engine/Compose
- Git
- 建议至少 4 核 CPU、8 GB 内存

### 使用 Docker 启动开发环境

```bash
git clone https://github.com/heidsoft/itsm.git
cd itsm

cp .env.dev.example .env
make dev-start-docker
```

启动后访问：

| 服务 | 地址 |
|:---|:---|
| Web | <http://localhost:3000> |
| 后端 API | <http://localhost:8090> |
| Swagger | <http://localhost:8090/swagger/index.html> |
| MinIO Console | <http://localhost:9001> |

开发环境默认登录：

```text
用户名：admin
密码：admin123
```

该账号只用于本地开发。任何可被其他人访问的部署都必须先修改管理员密码、`JWT_SECRET`、数据库、Redis 和对象存储凭据。

### 验证启动结果

```bash
make dev-status
make dev-health

curl http://localhost:8090/api/v1/health
curl http://localhost:3000/api/health
```

查看日志和停止环境：

```bash
make dev-logs
make dev-stop-docker
```

清理数据卷会删除本地数据库和对象存储数据：

```bash
make dev-clean
```

### 本机热更新开发

Docker 提供 PostgreSQL、Redis 和 MinIO，本机运行 Go 与 Next.js：

```bash
make dev-start-local
make dev-status
```

本机模式、PostgreSQL 17/pgvector 和代理排查见[开发指南](./docs/development.md)与[开发命令参考](./docs/dev-commands-reference.md)。

### 可选 AI 与监控组件

基础开发栈不会强制启动 Ollama 和监控组件：

```bash
# Ollama
docker compose --env-file .env -f docker-compose.dev.yml \
  --profile dev --profile ai up -d

# Prometheus + Grafana
docker compose --env-file .env -f docker-compose.dev.yml \
  --profile dev --profile monitoring up -d
```

没有可用模型时，ITIL 主流程应保持运行；AI 能力必须按配置降级。

## 关键业务闭环

```mermaid
flowchart LR
    A[告警或人工报障] --> B[事件]
    B --> C[关联 CI 与影响范围]
    C --> D[SLA 与 BPMN]
    D --> E[处理与恢复]
    E --> F[问题 / Known Error / 知识]
    C --> G[受控变更]
    G --> H[风险与审批]
    H --> I[实施 / 验证 / 回滚 / PIR]
```

商业 MVP 聚焦四条可验收旅程：

1. 事件 → CI → SLA → 流程 → 恢复 → 审计。
2. 重复事件 → 问题 → Known Error → 知识发布 → RAG。
3. 变更 → 影响分析 → 风险 → 审批 → 实施/回滚 → PIR。
4. 服务目录 → 请求 → 审批 → 交付 → CI 创建或变更。

## 可靠执行架构

生产部署使用同一后端镜像的三个进程角色：`itsm-init` 只执行迁移和初始化，`ITSM_PROCESS_MODE=api` 只提供 HTTP/WebSocket，`ITSM_PROCESS_MODE=worker` 执行 command、SLA、升级和索引任务。`all` 仅用于开发环境，生产启动会拒绝该模式。

```mermaid
flowchart TB
    UI[Next.js Web / Open API] --> API[Go / Gin API]
    API --> DOMAIN[ITIL 领域服务]
    DOMAIN --> TX[(业务数据 + Operational Command)]
    TX --> WORKER[Lease + Heartbeat + Fencing Worker]
    WORKER --> BPMN[BPMN]
    WORKER --> NOTICE[站内通知 / 企业连接器]
    WORKER --> FUTURE[AI / CMDB 同步 / 索引]
    DOMAIN --> AUDIT[(审计与历史)]
    API --> REDIS[(Redis)]
    API --> OBJECT[(MinIO / S3)]
```

当前可靠执行基座已接管：

- 事件、变更的关键 BPMN 启动命令。
- 工单创建、SLA 违规和变更审批通知生产者。
- 站内通知及企业消息投递的幂等、重试、死信和投递审计基础。

设计与运维约束见 [Operational Command / Outbox](./docs/architecture/operational-command-outbox.md)。

## 产品界面

| 事件与问题 | 变更与 CMDB |
|:---:|:---:|
| ![事件管理](./docs/images/03-事件管理.png) | ![变更管理](./docs/images/06-变更管理.png) |
| ![问题管理](./docs/images/04-问题管理.png) | ![CMDB](./docs/images/08-cmdb.png) |

| 服务目录与知识 | 工作流与权限 |
|:---:|:---:|
| ![服务目录](./docs/images/09-服务目录.png) | ![工作流](./docs/images/11-工作流.png) |
| ![知识库](./docs/images/10-知识库.png) | ![角色管理](./docs/images/12-角色管理.png) |

## 技术栈与仓库结构

| 层 | 技术 |
|:---|:---|
| 后端 | Go 1.25.12、Gin、Ent、PostgreSQL、Redis |
| 前端 | Next.js 15.5、React 19、TypeScript 6、Ant Design 6、Tailwind CSS |
| 工作流 | BPMN 2.0、流程定义/实例/任务/变量/历史 |
| AI/RAG | LLM Gateway、pgvector、OpenAI/兼容接口、Ollama 可选 |
| 交付 | Docker Compose、GHCR、GitHub Actions、Prometheus/Grafana 可选 |

```text
itsm/
├── itsm-backend/     # Go API、领域服务、Ent Schema、Worker
├── itsm-frontend/    # Next.js 管理端、服务台与用户门户
├── itsm-ai-service/  # AI/RAG 辅助服务
├── itsm-agent/       # Agent 扩展
├── itsm-skill/       # Skill 扩展
├── itsm-cli/         # CLI 入口
├── docs/             # 产品、架构、开发、部署、测试文档
├── scripts/          # 开发、生产、发布与诊断脚本
└── monitoring/       # Prometheus/Grafana 配置
```

后端是业务规则、权限、租户隔离、工作流执行和审计的事实来源；前端不复制生命周期规则。

## 开发与测试

```bash
# 后端
cd itsm-backend
GOTOOLCHAIN=auto go test ./...
GOTOOLCHAIN=auto go vet ./...

# 前端
cd ../itsm-frontend
npm install
npm run type-check
npm test

# 根目录工程契约
cd ..
make check-contracts
```

详细分层、测试策略和 E2E 用法见[开发指南](./docs/development.md)和[测试指南](./docs/testing/README.md)。

## 生产部署

项目支持三种部署模式：

- `private`：私有化部署。
- `saas`：平台托管多个企业租户。
- `saas_msp`：平台与 MSP 协同服务多个客户组织。

生成生产配置后，先修改和核对所有凭据与域名，再部署：

```bash
make prod-init

# 编辑 .env.prod，配置真实密码、JWT、域名、TLS 和外部依赖
make prod-deploy
make prod-health
```

手工使用 Compose 时必须显式传入同一份环境文件：

```bash
docker compose --env-file .env.prod -f docker-compose.prod.yml up -d
```

上线前至少完成：

- TLS、强密码、SSO/组织同步方案和最小权限配置。
- 显式数据库迁移、备份恢复和版本回滚演练。
- 租户隔离、RBAC、审计、Webhook/回调验签验证。
- 容量、故障恢复、队列积压和死信重放测试。
- 对启用的 CMDB、AI、RAG、连接器逐项完成 readiness 验收。

不要把开发默认配置用于生产。完整操作见[部署指南](./docs/deployment.md)、[生产就绪计划](./docs/delivery/production-readiness-program.md)和[运维手册](./docs/operations.md)。

## 文档导航

| 文档 | 用途 |
|:---|:---|
| [文档中心](./docs/README.md) | 按角色和主题查找资料 |
| [商业能力契约](./docs/product/itsm-commercial-capability-contract.md) | 能力成熟度、商业 MVP 和非目标 |
| [商业化架构](./docs/architecture/commercial-ready-architecture.md) | 生产级总体架构 |
| [CMDB 商业 MVP](./docs/product/cmdb-commercial-mvp.md) | CMDB GA/Pilot 边界和验收门槛 |
| [Outbox 架构](./docs/architecture/operational-command-outbox.md) | 可靠异步执行规范 |
| [API 参考](./docs/api/API_REFERENCE.md) | HTTP 接口文档 |
| [配置参考](./docs/configuration.md) | 环境变量和配置项 |
| [测试指南](./docs/testing/README.md) | 单元、集成、契约和 E2E 测试 |
| [Roadmap](./docs/roadmap.md) | 迭代方向 |

## 参与贡献

欢迎提交 Issue、文档和代码。开始前请阅读 [CONTRIBUTING.md](./CONTRIBUTING.md)。

```bash
git checkout -b feature/your-feature
# 修改并运行相关测试
git commit -m "feat: describe your change"
```

- [提交 Bug](https://github.com/heidsoft/itsm/issues/new)
- [参与讨论](https://github.com/heidsoft/itsm/discussions)
- [查看贡献者](https://github.com/heidsoft/itsm/graphs/contributors)

## Star 趋势

[![GitHub Star 增长趋势](./docs/assets/star-history.svg)](https://github.com/heidsoft/itsm/stargazers)

## License

本项目采用 [Apache License 2.0](./LICENSE)。允许商业使用、修改和分发；使用时请遵守许可证和 [NOTICE](./NOTICE) 要求。

<div align="center">

[GitHub](https://github.com/heidsoft/itsm) · [Issues](https://github.com/heidsoft/itsm/issues) · [Discussions](https://github.com/heidsoft/itsm/discussions)

如果这个项目对你有帮助，欢迎 Star、试用并反馈真实场景。

</div>
