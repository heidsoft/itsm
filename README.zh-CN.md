<div align="center">

# 🤖 AI-Native ITSM · 智能服务管理

## 企业级 IT 服务管理平台 | AI First, Not AI After

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Next.js](https://img.shields.io/badge/Next.js-15.5-000000?style=flat&logo=nextdotjs)](https://nextjs.org)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.7-3178C6?style=flat&logo=typescript)](https://typescriptlang.org)
[![License](https://img.shields.io/badge/License-Apache_2.0-yellowgreen?style=flat)](LICENSE)
[![Backend CI](https://github.com/heidsoft/itsm/actions/workflows/backend-ci.yml/badge.svg)](https://github.com/heidsoft/itsm/actions/workflows/backend-ci.yml)
[![Frontend CI](https://github.com/heidsoft/itsm/actions/workflows/frontend-ci.yml/badge.svg)](https://github.com/heidsoft/itsm/actions/workflows/frontend-ci.yml)
[![GA Gate](https://github.com/heidsoft/itsm/actions/workflows/ga-gate.yml/badge.svg)](https://github.com/heidsoft/itsm/actions/workflows/ga-gate.yml)
[![AI-Native](https://img.shields.io/badge/AI-Native-FF6B6B?style=flat&logo=openai)](https://openai.com)
[![Stars](https://img.shields.io/github/stars/heidsoft/itsm?style=flat)](https://github.com/heidsoft/itsm/stargazers)

**🚀 LLM-first 智能分诊 | Guidance-Harness-Skill 工程体系 | 开源免费**

[English](./README.md) · [架构解析](./docs/articles/07-ai-native-architecture-guidance-harness-skill.md) · [GA 能力矩阵](./docs/v1-ga/capability-matrix.md)

</div>

---

## 项目简介

AI-Native ITSM（智能服务管理）是一个面向国内企业数字化流程治理的**开源** IT 服务管理平台，目标是**对标 ServiceNow 的核心 ITSM 能力**，同时保持：

- ✨ **更轻量**：单二进制部署，资源占用仅 ServiceNow 的 5%
- 🔒 **更易私有化**：支持完全离线部署，数据不出企业
- 🇨🇳 **更适合本土**：飞书 / 企微 / 钉钉原生集成，中文 UI + 文档优先
- 🤖 **AI First**：LLM-first 智能分诊，RAG 知识库，AI 审计可追溯

## 核心能力

### ITIL 核心流程

- **工单 / 事件 / 问题 / 变更 / 发布**：完整 ITIL v4 生命周期
- **服务目录 + 服务请求**：员工自助门户
- **SLA 监控 + 告警**：实时合规率、违约预警
- **CMDB + 影响分析**：配置项关系 + 拓扑 + 变更影响图
- **BPMN 流程引擎**：可视化设计器，多级审批 / 加签 / 委派
- **知识库 + RAG**：全文搜索 + 向量召回 + 评审流

### AI 能力（v1.0 GA 实验性）

- 🤖 **LLM 智能分诊**：自动分类 / 优先级 / 处理人推荐
- 📝 **自动摘要**：工单、事件、问题一键总结
- 🔍 **知识推荐**：基于上下文的 RAG 召回
- 📊 **AI 审计**：prompt_version / model / confidence 全留痕
- ⚠️ **声明**：当前 AI 功能依赖外部 LLM（OpenAI / Azure / 自部署），未配置时降级为规则引擎

### 多租户与部署

- **三种部署模式**：
  - `private`：单租户，完全私有化
  - `saas`：单租户 SaaS
  - `saas_msp`：MSP 多租户，含资源配额
- **RBAC + 跨租户隔离**：所有数据强校验 tenant_id

## 快速开始

### 方式一：Docker Compose（推荐）

```bash
# 1. 克隆
git clone https://github.com/heidsoft/itsm.git
cd itsm

# 2. 启动（默认 private 模式）
cp .env.dev.example .env.dev
docker compose -f docker-compose.dev.yml --env-file .env.dev --profile dev up -d --build

# 3. 访问
# 前端：http://localhost:3000
# 后端：http://localhost:8090
# 默认账号：admin / admin123（仅开发环境）
```

### 方式二：本地开发

```bash
# 后端
cd itsm-backend
go run main.go  # http://localhost:8090

# 前端
cd itsm-frontend
npm install
npm run dev  # http://localhost:3000
```

### 方式三：生产部署

```bash
# 1. 修改环境变量
cp .env.prod.example .env.prod
# 必须修改：DB_PASSWORD / REDIS_PASSWORD / JWT_SECRET / ADMIN_PASSWORD
# ⚠️ 生产环境会自动检测默认密码并拒绝启动

# 2. 部署
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d

# 3. 验证
curl http://localhost/health  # 通过 Nginx 验证整体链路，期望 200
curl http://localhost:8090/api/v1/readiness/ga  # 验证后端就绪度
```

## 文档索引

| 类别 | 文档 |
|------|------|
| **入门** | [快速开始](#快速开始) · [贡献指南](./CONTRIBUTING.md) |
| **架构** | [商业就绪架构](./docs/architecture/commercial-ready-architecture.md) · [AI Native 架构](./docs/articles/07-ai-native-architecture-guidance-harness-skill.md) |
| **v1.0 GA** | [GA 准入指南](./docs/v1-ga-readiness.md) · [能力矩阵](./docs/v1-ga/capability-matrix.md) · [回滚指南](./docs/migrations/rollback-guide.md) |
| **对比** | [ServiceNow 差距分析](./docs/cmdb/servicenow-gap-analysis.md) · [CMDB-ITIL4 集成](./docs/cmdb/cmdb-workflow-itil4-integration.md) |
| **审批** | [审批节点语义](./docs/architecture/approval-node-semantics.md) |
| **质量** | [RBAC 跨租户回归](./docs/rbac/regression-report.md) · [Raw SQL 治理清单](./docs/sqlx/inventory.md) |
| **运维** | [部署指南](./docs/deployment.md) · [运维手册](./docs/operations.md) · [配置说明](./docs/configuration.md) |

## 验证 GA 准入

v1.0 GA 必须通过 4 项发布门禁：

```bash
# G1: 后端测试
cd itsm-backend && go test ./...

# G2: 前端构建
cd itsm-frontend && npm run type-check && npm run build

# G3: Docker Compose 健康检查
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
curl -sf http://localhost/health

# G4: 端到端 API 烟测
./scripts/smoke-test.sh
```

详细准入清单见 [docs/v1-ga-readiness.md](./docs/v1-ga-readiness.md)。

## 参与贡献

我们欢迎所有形式的贡献：

- 🐛 报告 Bug：[Issue Tracker](https://github.com/heidsoft/itsm/issues)
- 💡 功能建议：[Discussions](https://github.com/heidsoft/itsm/discussions)
- 🔧 提交 PR：参考 [CONTRIBUTING.md](./CONTRIBUTING.md)
- 🌍 翻译：补充 [README.zh-CN.md](./README.zh-CN.md) 之外的语种

## 许可证

本项目基于 **Apache 2.0** 开源，详见 [LICENSE](./LICENSE)。

## 致谢

感谢所有贡献者（[Contributors](./CONTRIBUTORS.md)）和开源依赖。

---

**智能服务管理 · 让 IT 服务更高效** · [官网](https://cloudmesh.top/) · [GitHub](https://github.com/heidsoft/itsm)
