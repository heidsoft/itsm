<div align="center">

# 🤖 AI-Native ITSM

## 企业级IT服务管理平台 | AI First, Not AI After

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Next.js](https://img.shields.io/badge/Next.js-15.5-000000?style=flat&logo=nextdotjs)](https://nextjs.org)
[![TypeScript](https://img.shields.io/badge/TypeScript-6.0-3178C6?style=flat&logo=typescript)](https://typescriptlang.org)
[![License](https://img.shields.io/badge/License-Apache_2.0-yellowgreen?style=flat)](LICENSE)
[![Backend CI](https://github.com/heidsoft/itsm/actions/workflows/backend-ci.yml/badge.svg)](https://github.com/heidsoft/itsm/actions/workflows/backend-ci.yml)
[![Frontend CI](https://github.com/heidsoft/itsm/actions/workflows/frontend-ci.yml/badge.svg)](https://github.com/heidsoft/itsm/actions/workflows/frontend-ci.yml)
[![AI-Native](https://img.shields.io/badge/AI-Native-FF6B6B?style=flat&logo=openai)](https://openai.com)
[![Stars](https://img.shields.io/github/stars/heidsoft/itsm?style=flat)](https://github.com/heidsoft/itsm/stargazers)
[![Forks](https://img.shields.io/github/forks/heidsoft/itsm?style=flat)](https://github.com/heidsoft/itsm/network)
[![Issues](https://img.shields.io/github/issues/heidsoft/itsm?style=flat)](https://github.com/heidsoft/itsm/issues)
[![Contributors](https://img.shields.io/github/contributors/heidsoft/itsm?style=flat)](https://github.com/heidsoft/itsm/graphs/contributors)

**🚀 ITIL 流程治理 | BPMN 工作流 | CMDB | AI 决策支持 | Apache-2.0**

**[🌐 官网](https://cloudmesh.top/)** · **[📖 架构解析](./docs/articles/07-ai-native-architecture-guidance-harness-skill.md)**

</div>

---

## 项目说明

ITSM 是一个面向国内企业数字化流程治理的开源 IT 服务管理平台，目标是对标 ServiceNow 的核心 ITSM 能力，同时保持更轻量、更易私有化部署、更适合本土企业集成环境。

项目覆盖 ITIL v3 的核心流程：工单、事件、问题、变更、发布、服务请求、服务目录、知识库、SLA、CMDB 和流程编排。系统内置 BPMN 工作流引擎，支持按企业实际管理制度自定义流程，并预留飞书、企业微信、钉钉、Webhook、连接器市场、Skill 市场和插件市场扩展方向。

AI 能力不是外挂式聊天框，而是嵌入到工单分诊、摘要、知识检索、流程建议、审计追踪和自动化工具调用中。当前定位是“AI Native ITSM 基座”：先把流程、权限、数据模型、审计、连接器和可部署性打牢，再逐步让 AI 参与更多可控的企业服务管理动作。

> **当前阶段：v1.1 稳定性加固。** v1.0 已完成核心流程、BPMN、CMDB v1、SLA、RBAC、多租户和部署基础；当前重点是接口与测试覆盖、连接器市场 v1、权限加固和 AI 审计。生产落地前仍应按企业规模完成安全配置、备份恢复、容量测试、SSO/组织同步和灾备验收，详见[生产就绪计划](./docs/delivery/production-readiness-program.md)。

### 适合谁使用

- 企业 IT、运维、服务台团队：统一处理工单、事件、变更、问题和服务请求。
- 数字化平台团队：把企业内部流程、审批、CMDB 和系统集成统一到一个可扩展平台。
- MSP 服务商：通过多租户和 MSP 模式服务多个客户组织。
- 开源贡献者和二次开发团队：基于 Go + Next.js + BPMN + AI 能力建设企业级流程产品。

### 当前能力地图

| 领域 | 已覆盖能力 |
|:---|:---|
| ITIL 流程 | 工单、事件、问题、变更、发布、服务请求、服务目录、SLA |
| CMDB | CI 类型、配置项、关系、拓扑、影响分析、云资源发现基础能力 |
| 工作流 | BPMN 流程定义、流程实例、用户任务、流程绑定、流程触发 |
| AI | 智能分诊、摘要、RAG 检索、LLM 网关、AI 审计、工具调用框架 |
| 知识库 | 文章、分类、搜索、推荐、RAG 接入 |
| 企业集成 | 连接器生命周期与市场基础，内置飞书、企微、钉钉、Webhook、Console 连接器骨架 |
| 权限与租户 | RBAC、多租户、MSP 模式、菜单权限、组织结构 |
| 交付 | Docker Compose、GitHub Releases、GHCR 镜像、前后端 CI |

### 发布物怎么选

| 方式 | 适用场景 | 入口 |
|:---|:---|:---|
| 源码 Docker Compose | 本地体验、二次开发、查看完整配置 | `docker compose -f docker-compose.dev.yml --profile dev up -d --build` |
| GitHub Packages 镜像 | 只想快速部署运行，不想本地构建 | `ghcr.io/heidsoft/itsm-backend` / `ghcr.io/heidsoft/itsm-frontend` |
| Release zip | 离线分发、自定义部署脚本 | [GitHub Releases](https://github.com/heidsoft/itsm/releases) |

更多文档入口见 [文档中心](./docs/README.md)。

---

## ⭐ AI-Native 设计原则

这里的 AI-Native 不是“让系统依赖模型才能运行”，而是让 AI 参与完整服务管理生命周期，同时保持企业系统所需的可控性：

- **统一入口**：模型调用通过 LLM Gateway/AI 服务抽象，业务控制器不直接绑定厂商 SDK。
- **人在回路**：分诊、影响分析和流程建议默认是决策支持，高风险动作需权限校验和人工确认。
- **可审计**：保留模型、提供商、提示词版本、置信度、建议结果、采纳/拒绝状态和操作反馈。
- **安全降级**：模型关闭、超时或低置信度时使用规则、关键词或人工处理，ITIL 主流程仍可运行。
- **权限优先**：RAG 检索和工具调用必须经过租户、RBAC、知识可见性和审计边界。
- **可扩展**：通过 Prompt、Skill、Connector 与 Workflow 扩展点增加能力，避免在页面和控制器中散落大段提示词。

当前已具备智能分诊、摘要、知识检索、LLM 网关、AI 审计与工具调用基础；模型评测闭环、更多企业连接器和可声明式 Skill Registry 仍在持续建设。

---

## 🚀 快速开始

### 交付模式

同一套代码基线支持三种部署模式，通过 `DEPLOYMENT_MODE` 切换：

- `private`: 私有化部署，默认创建一个根租户和管理员
- `saas`: SaaS 托管模式，平台托管多个企业客户租户
- `saas_msp`: SaaS + MSP 模式，平台方可并行服务多个客户公司

容器编排内置了一次性 `itsm-init` 初始化任务，负责数据库迁移和幂等 seed。常驻后端服务默认不再隐式做初始化。

### Docker 开发环境（推荐）

```bash
# 克隆项目
git clone https://github.com/heidsoft/itsm.git
cd itsm

# 启动核心开发栈：PostgreSQL、Redis、MinIO、初始化器、后端、前端
docker compose -f docker-compose.dev.yml --profile dev up -d --build

# 等价的项目命令
make dev-start

# 查看服务状态
docker compose -f docker-compose.dev.yml --profile dev ps

# 访问应用
# 前端:    http://localhost:3000
# 后端:    http://localhost:8090
# API文档: http://localhost:8090/swagger/index.html
```

> **首次登录（开发/首次安装）**: 用户名 `admin`，密码 `admin123`。生产部署前必须通过环境变量或初始化流程修改管理员密码、`JWT_SECRET`、数据库密码和 Redis 密码。
>
> **前端访问链路**: 浏览器统一访问同源 `/api`，前端服务端代理再转发到 `ITSM_BACKEND_URL`。不要把容器内地址直接配置到浏览器侧 `NEXT_PUBLIC_API_URL`。

开发配置会幂等执行迁移与基础 seed。默认账号仅用于本地体验：`admin / admin123`。

### 可选组件

基础开发环境不会强制下载 AI 和监控镜像，需要时显式启用：

```bash
# 本地 Ollama（首次镜像较大）
docker compose -f docker-compose.dev.yml --profile dev --profile ai up -d

# Prometheus + Grafana
docker compose -f docker-compose.dev.yml --profile dev --profile monitoring up -d

# 同时启用全部可选组件
docker compose -f docker-compose.dev.yml \
  --profile dev --profile ai --profile monitoring up -d
```

可选组件入口：MinIO Console `http://localhost:9001`、Prometheus `http://localhost:9090`、Grafana `http://localhost:3001`、Ollama `http://localhost:11434`。

### 快速验证

```bash
# 检查服务健康状态
curl http://localhost:8090/api/v1/health

# 检查前端服务及后端代理链路
curl http://localhost:3000/api/health

# 检查 v1.0 GA 就绪度（默认功能模板、连接器、AI 审计契约）
curl http://localhost:8090/api/v1/readiness/ga

# 查看日志
docker compose -f docker-compose.dev.yml --profile dev logs -f

# 停止服务
docker compose -f docker-compose.dev.yml --profile dev down

# 完全清理（包括数据卷）
docker compose -f docker-compose.dev.yml --profile dev down -v
```

> `down -v` 会删除本地数据库和对象存储数据，请确认不再需要后执行。

### 本机开发模式

```bash
# 启动本机前后端开发进程及所需依赖
./scripts/start-dev.sh

# 方式2: 使用 Makefile
make dev-start

# 停止服务
./scripts/stop-dev.sh
# 或
make dev-stop

# 查看日志
make dev-logs

# 查看服务状态
make dev-status
```

要求：

- Docker Desktop 已启动。
- macOS 本机开发推荐使用 Homebrew `postgresql@17`，并启用 `pgvector`。不要同时启动 `postgresql@16`，否则 5432 端口可能连接到旧版本，RAG 向量能力会降级或迁移失败。

```bash
# 确认只有 PostgreSQL 17 在运行
brew services stop postgresql@16
brew services start postgresql@17
brew services list | grep postgresql

# 推荐使用 PostgreSQL 17 客户端，避免 PATH 中旧版 psql/pg_dump 被优先使用
/usr/local/opt/postgresql@17/bin/psql -h localhost -p 5432 -U heidsoft -d itsm -c "SELECT version();"

# 确认 pgvector 已启用
/usr/local/opt/postgresql@17/bin/psql -h localhost -p 5432 -U heidsoft -d itsm -c "CREATE EXTENSION IF NOT EXISTS vector; SELECT extname, extversion FROM pg_extension WHERE extname = 'vector';"
```

本地启动脚本默认连接：

```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=itsm_user
DB_PASSWORD=dev123
DB_NAME=itsm
```

访问地址：
- 前端: http://localhost:3000
- 后端API: http://localhost:8090
- Swagger文档: http://localhost:8090/swagger/index.html
- PostgreSQL: localhost:5432
- Redis: localhost:6379

首次登录：用户名 `admin`，密码 `admin123`。

**前端无法访问排查**:

```bash
# 清理旧进程并重新启动
./scripts/start-dev.sh restart

# 验证前端和后端是否真的可访问
./scripts/start-dev.sh status
curl -I http://127.0.0.1:3000
curl http://127.0.0.1:8090/api/v1/health

# 查看监听端口和日志
lsof -nP -iTCP:3000 -sTCP:LISTEN
lsof -nP -iTCP:8090 -sTCP:LISTEN
tail -f logs/frontend.log
```

在 local 模式下，如果使用本机 Homebrew PostgreSQL/Redis，`./scripts/start-dev.sh status` 会显示 `PostgreSQL: external` / `Redis: external`，表示脚本检测到外部服务正在监听端口，不会再启动 Docker 容器。

如果 `curl` 返回 200 但浏览器打不开，优先检查浏览器代理配置，确保 `localhost` / `127.0.0.1` 不走 HTTP 代理。

### 初始化与生产部署

```bash
# 手动执行一次性初始化（迁移 + seed）
docker compose -f docker-compose.dev.yml --profile dev run --rm itsm-init

# 生产环境必须显式传入环境文件
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
```

生产环境必须更换 `ADMIN_PASSWORD`、`JWT_SECRET`、数据库/Redis/MinIO 凭据，配置 TLS、备份、日志留存与告警；不要直接复用仓库中的开发默认值。完整步骤见[部署指南](./docs/deployment.md)和[运维手册](./docs/operations.md)。

默认初始化模板：

- `private`: 创建默认根租户和管理员，适合集团/事业部/子公司模式
- `saas`: 创建平台系统租户，不预置客户业务数据
- `saas_msp`: 创建 MSP 提供方租户、示例客户租户和基础分配关系

### v1.0 GA 初始化检查

默认 seed 会加载 `itsm-backend/config/seed/default.json`，用于 10 分钟内确认产品基础能力已可配置：

1. 使用 `admin / admin123` 登录。
2. 检查菜单、角色、权限和默认租户是否已初始化。
3. 检查服务目录模板：账号申请、软件安装、网络接入、云资源、数据库、安全扫描等。
4. 检查 SLA、审批流、流程绑定、CI 类型和标准变更模板是否可配置。
5. 进入连接器市场，使用 `/api/v1/connectors/lifecycle` 验证内置飞书、钉钉、企微、Webhook、Console 连接器生命周期。
6. 使用 `/api/v1/ai/audit` 验证 AI 建议可追踪，不自动执行高风险动作。

默认初始化不预置虚构事件、问题、变更或真实资产业务数据；企业可通过 `ITSM_SEED_CONFIG` 或 `config/seed/default.json` 定制自己的初始化模板。

更多验收项见 [v1.0 GA 收口验收指南](./docs/v1-ga-readiness.md)。

---

## 📸 产品截图

### 核心管理界面

| 仪表盘 | 工单管理 |
|:---:|:---:|
| ![仪表盘](docs/images/01-仪表盘.png) | ![工单管理](docs/images/02-工单管理.png) |

| 事件管理 | 问题管理 |
|:---:|:---:|
| ![事件管理](docs/images/03-事件管理.png) | ![问题管理](docs/images/04-问题管理.png) |

| 变更管理 | CMDB 配置管理 |
|:---:|:---:|
| ![变更管理](docs/images/06-变更管理.png) | ![CMDB](docs/images/08-cmdb.png) |

| 服务目录 | 知识库 |
|:---:|:---:|
| ![服务目录](docs/images/09-服务目录.png) | ![知识库](docs/images/10-知识库.png) |

| 工作流引擎 | 角色管理 |
|:---:|:---:|
| ![工作流](docs/images/11-工作流.png) | ![角色管理](docs/images/12-角色管理.png) |

### 登录界面

![登录](docs/images/login.png)

---

## ✨ AI-Native 核心能力

### 🤖 Guidance-Harness-Skill 架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                     Skill Orchestrator                               │
│    流水线编排 │ 输入输出转换 │ 错误处理 │ 降级策略                     │
└─────────────────────────────────────────────────────────────────────┘
              │                   │               │
              ▼                   ▼               ▼
    ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
    │ TriageSkill    │ │ SummarizeSkill  │ │ KBSkill        │
    │ (Guidance程序) │ │ (Guidance程序)  │ │ (Guidance程序) │
    └─────────────────┘ └─────────────────┘ └─────────────────┘
              │                   │               │
              └───────────────────┴───────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Harness Controller                              │
│    Prompt管理 │ 参数配置 │ 执行控制 │ 结果解析                        │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                 Evaluator (质量评估闭环)                             │
│    准确性评估 │ 性能监控 │ 回归测试 │ Bad Case 积累                  │
└─────────────────────────────────────────────────────────────────────┘
```

### 🎯 AI 智能功能

| 功能 | 说明 | 失败与治理边界 |
|:---|:---|:---|
| 🎯 **智能分诊** | LLM 与规则协同生成分类、优先级建议 | 低置信度降级，结果由操作员确认 |
| 📝 **自动摘要** | 生成工单、事件和处理过程摘要 | 保留来源上下文，不替代原始记录 |
| 🔍 **RAG 知识库** | 基于 pgvector 的知识检索与问答 | 受租户、RBAC、可见性和文章版本约束 |
| 💡 **流程建议** | 推荐流程、解决方案和相似记录 | 建议可采纳、拒绝并进入审计反馈 |
| 🛠️ **工具调用** | 在明确权限内调用受控系统能力 | 高风险动作必须校验权限并记录审计 |

### 🔧 Skill 扩展体系

| Skill | 功能 | 状态 |
|:---|:---|:---|
| TriageSkill | 工单智能分类 | ✅ 已实现 |
| SummarizeSkill | 工单/事件摘要 | ✅ 已实现 |
| KBSkill | RAG 知识库问答 | ✅ 已实现 |
| SecurityTriageSkill | 安全事件专项分类 | 🔜 待开发 |
| ImpactAnalysisSkill | 变更影响范围分析 | 🔜 待开发 |
| SLAForecastSkill | SLA 达成率预测 | 🔜 待开发 |

---

## 🔀 传统 ITSM 功能

### 🎫 服务管理

| 工单管理 | 事件管理 | 问题管理 | 变更管理 |
|:---:|:---:|:---:|:---:|
| 智能分配<br>SLA 保障<br>自动化流转 | 实时监控<br>智能告警<br>升级策略 | 根因分析<br>RFC 关联<br>知识沉淀 | 风险评估<br>多级审批<br>回滚方案 |

| 发布管理 | 服务请求 | 服务目录 | 知识库 |
|:---:|:---:|:---:|:---:|
| 发布计划<br>阶段控制<br>回滚支持 | 自助门户<br>审批流程<br>进度追踪 | 服务Offering<br>SLA 定义<br>自助申请 | RAG 检索<br>智能问答<br>知识推荐 |

### 🔀 BPMN 工作流引擎

```
┌──────────────────────────────────────────────────────────────────┐
│  🏗️ 可视化设计器    │  📊 流程监控    │  🔒 权限控制   │  📝 审计日志  │
├──────────────────────────────────────────────────────────────────┤
│  拖拽式流程设计    │  实时追踪      │  精细权限      │  全程记录     │
│  BPMN 2.0 标准    │  性能分析      │  角色绑定      │  合规追溯     │
│  版本管理         │  SLA 集成      │  数据隔离      │  报表导出     │
└──────────────────────────────────────────────────────────────────┘
```

### 🌍 MSP 多租户

```
┌─────────────────────────────────────────────────────────┐
│                    🏢 MSP 服务商                         │
├─────────────┬─────────────┬─────────────┬──────────────┤
│  🏢 租户 A  │  🏢 租户 B  │  🏢 租户 C  │  ...         │
├─────────────┴─────────────┴─────────────┴──────────────┤
│  📊 资源配额    │  💰 计费管理    │  🔍 监控告警     │
└─────────────────────────────────────────────────────────┘
```

### 📊 SLA 监控体系

- 多级别 SLA 策略配置
- 实时合规率监控面板
- 违约预警与自动升级
- 完整的 SLA 报表分析

---

## 🏗 技术架构

### 技术栈

<div align="center">

**后端** | Go 1.25+ | Gin | Ent ORM | PostgreSQL | Redis | BPMN Engine

**前端** | Next.js 15 | React 19 | TypeScript | Ant Design 6 | Tailwind CSS | Zustand

**AI** | OpenAI | Claude | Ollama (私有化) | Guidance

</div>

### 仓库结构

```text
itsm/
├── itsm-backend/     # Go/Gin API、领域服务、Ent Schema、迁移与后台任务
├── itsm-frontend/    # Next.js 管理端、服务台与用户门户
├── itsm-ai-service/  # AI/RAG 辅助服务
├── itsm-agent/       # Agent 扩展实验区
├── itsm-skill/       # Skill 定义与运行扩展
├── itsm-cli/         # CLI 扩展入口
├── docs/             # 架构、开发、部署、测试和产品文档
├── scripts/          # 开发、生产、发布和诊断脚本
└── monitoring/       # Prometheus/Grafana 配置
```

后端是领域规则、权限、租户隔离、工作流执行和审计的事实来源；前端通过 DTO/API 契约交互，不在 UI 中复制生命周期规则。

### 系统架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                         🖥️ 客户端层                              │
│     Web (Next.js)      │      企业门户        │    Open API   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      🌐 接入层 (Nginx)                          │
│              负载均衡 / SSL 终止 / 静态资源缓存                   │
└─────────────────────────────────────────────────────────────────┘
                              │
              ┌───────────────┴───────────────┐
              ▼                               ▼
┌─────────────────────────┐       ┌─────────────────────────┐
│    🌐 Next.js 前端      │       │     ⚙️ Go 后端 API      │
│       端口: 3000        │       │       端口: 8090         │
└─────────────────────────┘       └─────────────────────────┘
              │                               │
              │                               ├──────────────┐
              │                               ▼              ▼
              │                    ┌─────────────┐  ┌─────────┐
              │                    │ PostgreSQL  │  │  Redis  │
              │                    │   端口:5432  │  │  6379   │
              │                    └─────────────┘  └─────────┘
              │                               │
              │                               ▼
              │                    ┌─────────────────────────┐
              │                    │     🤖 AI 服务层        │
              │                    │  Guidance-Harness-Skill │
              │                    │     LLM Gateway         │
              │                    └─────────────────────────┘
              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        💾 存储层                                 │
│    文件存储 (MinIO/S3)   │ PostgreSQL + pgvector │  Redis 缓存  │
└─────────────────────────────────────────────────────────────────┘
```

### 数据模型 (100+ 实体)

```
核心模块          扩展模块           BPMN 工作流         MSP 多租户
─────────        ───────           ───────────         ─────────
├─ 工单           ├─ 服务目录        ├─ 流程定义          ├─ 租户
├─ 事件           ├─ 知识库          ├─ 流程实例          ├─ 部门
├─ 问题           ├─ SLA             ├─ 流程任务          ├─ 团队
├─ 变更           ├─ 审批链          ├─ 流程变量          ├─ 项目
├─ 发布           ├─ 通知             ├─ 审计日志          └─ 资源分配
├─ 资产           └─ 报表            └─ 权限控制
└─ 许可证
```

---

## 文档导航

| [文档中心](./docs/README.md) | [开发指南](./docs/development.md) | [部署指南](./docs/deployment.md) |
|:---:|:---:|:---:|
| 按角色查找文档 | 开发环境搭建 | Docker/K8s 部署 |

| [配置参考](./docs/configuration.md) | [数据库](./docs/database.md) | [运维手册](./docs/operations.md) |
|:---:|:---:|:---:|
| 环境变量详解 | 迁移与备份 | 日志与监控 |

| [v1.0 验收](./docs/v1-ga-readiness.md) | [AI架构解析](./docs/articles/07-ai-native-architecture-guidance-harness-skill.md) | [贡献指南](./CONTRIBUTING.md) |
|:---:|:---:|:---:|
| GA 能力检查 | Guidance-Harness-Skill 三层体系 | PR 流程 |

| [审批节点语义](./docs/architecture/approval-node-semantics.md) | [CMDB × ITIL4 集成](./docs/cmdb/cmdb-workflow-itil4-integration.md) | [中文 README](./README.zh-CN.md) |
|:---:|:---:|:---:|
| 节点级 / 流程级审批边界 | 事件/问题/变更与 CMDB 联动 | 中文快速上手 |

---

## 常用命令

```bash
# 开发环境
make dev-start
make dev-status
make dev-health
make dev-logs
make dev-stop

# 后端
cd itsm-backend
go run main.go
go test ./...

# 前端
cd itsm-frontend
npm install
npm run dev
npm run type-check
npm run lint:check
npm test -- --runInBand
npm run build

# 生产部署：校验、备份、构建、部署、健康检查
make prod-init
make prod-deploy
make prod-health
make prod-logs
```

测试目录、分层策略和 E2E 使用方式见[测试指南](./docs/testing/README.md)。

---

## 🤝 参与贡献

欢迎提交 Pull Request！请阅读 [CONTRIBUTING.md](./CONTRIBUTING.md) 了解详情。

```bash
# 1. Fork 项目
# 2. 创建分支
git checkout -b feature/amazing-feature

# 3. 提交更改
git commit -m "feat: add amazing-feature"

# 4. 推送分支
git push origin feature/amazing-feature
```

### 代码规范

- ✅ Go: 使用 `gofumpt` 格式化
- ✅ TypeScript: ESLint + Prettier
- ✅ 提交信息: [Conventional Commits](https://www.conventionalcommits.org/)
- ✅ 测试: 新增功能需配套测试

---

## 📄 许可证

Apache License 2.0 - 开源免费，允许自由使用于商业产品。

- ✅ 个人学习与使用
- ✅ 商业产品集成
- ✅ 闭源项目使用
- ✅ 二次开发与分发

> 详见 [LICENSE](./LICENSE) 和 [NOTICE](./NOTICE) 文件。

---

## 📞 联系我们

<div align="center">

🐙 **GitHub**: [heidsoft/itsm](https://github.com/heidsoft/itsm)

💬 **讨论**: [Discussions](https://github.com/heidsoft/itsm/discussions)

🐛 **问题**: [Issues](https://github.com/heidsoft/itsm/issues)

📧 **Email**: <heidsoft@qq.com>

</div>

---

<div align="center">

**⭐ 如果这个项目对您有帮助，请 Star 支持！**

**🤖 AI-Native ITSM: AI First, Not AI After**

**[贡献者](./CONTRIBUTORS.md)** | 感谢您的参与！

Made with ❤️ by ITSM Team

</div>
