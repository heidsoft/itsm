<div align="center">

# 🤖 AI-Driven ITSM

## 企业级IT服务管理平台

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Next.js](https://img.shields.io/badge/Next.js-15.5-000000?style=flat&logo=nextdotjs)](https://nextjs.org)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.7-3178C6?style=flat&logo=typescript)](https://typescriptlang.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-17-336791?style=flat&logo=postgresql)](https://postgresql.org)
[![License](https://img.shields.io/badge/License-Apache_2.0-yellowgreen?style=flat)](LICENSE)
[![AI Powered](https://img.shields.io/badge/AI-Powered-FF6B6B?style=flat&logo=openai)](https://openai.com)

**🚀 基于 ITIL 最佳实践 | AI 智能驱动 | 开源免费**

**[🌐 官网](https://cloudmesh.top/)**

[English](./README_EN.md) · [快速开始](#快速开始) · [功能特性](#核心功能) · [AI 智能](#ai-智能功能) · [贡献代码](#参与贡献)

</div>

---

## ⭐ 项目简介

> 企业级 IT 服务管理平台的全新定义 - 让 AI 成为您的智能 IT 助手

ITSM 是一个现代化的 **AI 驱动**企业级 IT 服务管理平台，采用 Go/Gin 后端 + Next.js/React 前端架构，深度集成 AI 能力，助力企业实现 IT 服务的智能化转型。

### 核心优势

<div align="center">

| 🤖 AI 智能 | ⚡ 自动化 | 🌍 多租户 | 📈 企业级 |
|:---:|:---:|:---:|:---:|
| 智能分类 · RAG 知识库 · 自动摘要 | BPMN 工作流 · 智能分配 · 自动告警 | MSP 模式 · 租户隔离 · 资源配额 | 高可用 · 可扩展 · 安全合规 |

</div>

---

## 🚀 快速开始

### Docker 一键启动（推荐）

```bash
# 克隆项目
git clone https://github.com/heidsoft/itsm.git
cd itsm

# 启动所有服务
make dev-up

# 访问应用
# 🌐 前端:    http://localhost:3000
# 🔧 后端:    http://localhost:8080
# 📚 API文档: http://localhost:8080/swagger
```

> **👤 首次登录**: 用户名 `admin`，密码 `admin123`

### 本地开发

```bash
# 前置要求: Go 1.25+ | Node.js 22+ | PostgreSQL 14+ | Redis 7+

# 1. 启动数据库
docker compose up -d postgres redis

# 2. 启动后端
cd itsm-backend
cp .env.example .env
go run main.go

# 3. 启动前端 (新终端)
cd itsm-frontend
cp .env.example .env.local
npm run dev
```

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

## ✨ 核心功能

### 🎫 服务管理

<div align="center">

| 工单管理 | 事件管理 | 问题管理 | 变更管理 |
|:---:|:---:|:---:|:---:|
| 智能分配<br>SLA 保障<br>自动化流转 | 实时监控<br>智能告警<br>升级策略 | 根因分析<br>RFC 关联<br>知识沉淀 | 风险评估<br>多级审批<br>回滚方案 |

| 发布管理 | 服务请求 | 服务目录 | 知识库 |
|:---:|:---:|:---:|:---:|
| 发布计划<br>阶段控制<br>回滚支持 | 自助门户<br>审批流程<br>进度追踪 | 服务Offering<br>SLA 定义<br>自助申请 | RAG 检索<br>智能问答<br>知识推荐 |

</div>

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

### 🤖 AI 智能核心

| 功能 | 说明 | 效果 |
|:---|:---|:---|
| 🎯 **智能分类** | ML 自动识别工单类型、优先级 | 分类准确率 95%+ |
| 📝 **自动摘要** | LLM 生成工单/事件摘要 | 节省 70% 阅读时间 |
| 🔍 **RAG 知识库** | 向量检索 + 大模型问答 | 知识查找秒级响应 |
| 💡 **智能推荐** | 推荐解决方案、相似工单 | 提升解决效率 50%+ |
| 👷 **智能分配** | 基于技能/负载的自动派单 | 派单准确率 90%+ |
| 📊 **趋势预测** | 时序预测事件趋势 | 提前预警容量风险 |

### 🌍 MSP 多租户

<div align="center">

```
┌─────────────────────────────────────────────────────────┐
│                    🏢 MSP 服务商                         │
├─────────────┬─────────────┬─────────────┬──────────────┤
│  🏢 租户 A  │  🏢 租户 B  │  🏢 租户 C  │  ...         │
├─────────────┴─────────────┴─────────────┴──────────────┤
│  📊 资源配额    │  💰 计费管理    │  🔍 监控告警     │
└─────────────────────────────────────────────────────────┘
```

</div>

- 服务商 (MSP) 视角的全局管理
- 租户资源分配与配额控制
- 跨租户服务目录
- 统一监控与报表

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

</div>

### 系统架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                         🖥️ 客户端层                              │
│     Web (Next.js)      │      移动端 PWA      │    API        │
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
              │                    │  RAG / 分类 / 摘要     │
              │                    └─────────────────────────┘
              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        💾 存储层                                 │
│    文件存储 (MinIO/S3)   │   向量存储 (Chroma)   │   对象存储   │
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

## 📚 文档导航

| 📖 [开发指南](./docs/DEVELOPMENT.md) | 🚀 [部署指南](./DEPLOYMENT.md) | ⚙️ [配置参考](./docs/CONFIGURATION.md) |
|:---:|:---:|:---:|
| 开发环境搭建 | Docker/K8s 部署 | 环境变量详解 |

| 🗄️ [数据库](./docs/DATABASE.md) | 🔧 [运维手册](./docs/OPERATIONS.md) | 🔐 [安全指南](./docs/SECURITY.md) |
|:---:|:---:|:---:|
| 迁移与备份 | 日志与监控 | 权限与审计 |

| 🛠️ [自动化测试](./itsm-frontend/tests/e2e) |
|:---:|
| E2E 测试 |

---

## 🛠️ 常用命令

```bash
# Docker 开发环境
make dev-up          # 启动所有服务 (前端:3000 | 后端:8080)
make dev-down        # 停止服务
make dev-logs        # 查看日志

# 测试
cd itsm-frontend && npm run test:e2e   # E2E 测试
cd itsm-backend && go test ./...        # 后端测试
```

---

## 🤝 参与贡献

欢迎提交 Pull Request！请阅读 [CONTRIBUTING.md](./CONTRIBUTING.md) 了解详情。

```bash
# 1. Fork 项目
# 2. 创建分支
git checkout -b feature/amazing-feature

# 3. 提交更改
git commit -m "feat: add amazing feature"

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

Apache License 2.0 - 开源免费，企业级商用首选

> **商业化授权声明**: 如需将本项目用于商业产品，请访问 [官网](https://cloudmesh.top/) 获取商业授权。未经授权的商业使用将视为侵权行为。

---

## 📞 联系我们

<div align="center">

🐙 **GitHub**: [heidsoft/itsm](https://github.com/heidsoft/itsm)

💬 **讨论**: [Discussions](https://github.com/heidsoft/itsm/discussions)

🐛 **问题**: [Issues](https://github.com/heidsoft/itsm/issues)

📧 **Email**: heidsoft@qq.com

</div>

---

<div align="center">

**⭐ 如果这个项目对您有帮助，请 Star 支持！**

Made with ❤️ by ITSM Team

</div>
