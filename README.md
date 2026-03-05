# ITSM 企业级IT服务管理平台

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/Next.js-15.5-000000?style=flat&logo=nextdotjs" alt="Next.js">
  <img src="https://img.shields.io/badge/TypeScript-5-3178C6?style=flat&logo=typescript" alt="TypeScript">
  <img src="https://img.shields.io/badge/PostgreSQL-17-336791?style=flat&logo=postgresql" alt="PostgreSQL">
  <img src="https://img.shields.io/badge/Redis-7-DC382D?style=flat&logo=redis" alt="Redis">
  <img src="https://img.shields.io/badge/License-MIT-yellowgreen?style=flat" alt="License">
</p>

<p align="center">
  <strong>🚀 企业级 IT 服务管理平台 | 基于 ITIL 最佳实践 | AI 智能赋能</strong>
</p>

---

## 📋 目录

- [项目简介](#项目简介)
- [快速开始](#快速开始)
- [系统要求](#系统要求)
- [技术架构](#技术架构)
- [功能特性](#功能特性)
- [文档导航](#文档导航)
- [参与贡献](#参与贡献)
- [许可证](#许可证)

---

## 🎯 项目简介

ITSM 是一个现代化的企业级 IT 服务管理(ITSM)平台，基于 **Go/Gin** 后端和 **Next.js/React** 前端构建。遵循 ITIL 最佳实践，集成 AI 智能功能，帮助企业高效管理 IT 服务。

### 核心价值

- **标准化**: 遵循 ITIL 最佳实践流程
- **自动化**: BPMN 工作流引擎，自动化审批流程
- **智能化**: AI 辅助工单分类、智能摘要、RAG 知识库
- **可扩展**: 微服务架构，支持高并发、大规模部署

---

## 🚀 快速开始

### 方式一：Docker Compose（推荐，5分钟启动）

```bash
# 1. 克隆项目
git clone https://github.com/heidsoft/itsm.git
cd itsm

# 2. 启动所有服务
make dev-up

# 3. 访问应用
# 前端: http://localhost:3000
# 后端: http://localhost:8090
# API 文档: http://localhost:8090/swagger
```

> **首次登录**: 用户名 `admin`，密码 `admin123`

### 方式二：本地开发

```bash
# 前置条件：Go 1.25+、Node.js 22+、PostgreSQL 14+、Redis 7+

# 1. 克隆项目
git clone https://github.com/heidsoft/itsm.git
cd itsm

# 2. 启动数据库
docker compose up -d postgres redis

# 3. 启动后端
cd itsm-backend
cp .env.example .env
go run main.go

# 4. 启动前端（新终端）
cd itsm-frontend
cp .env.example .env.local
npm run dev
```

---

## 💻 系统要求

### 最低配置

| 组件 | 最低版本 | 推荐版本 | 说明 |
|------|---------|---------|------|
| Go | 1.21 | 1.25+ | 后端运行时 |
| Node.js | 18 | 22+ | 前端运行时 |
| PostgreSQL | 14 | 17 | 主数据库 |
| Redis | 6 | 7+ | 缓存/会话 |
| Docker | 20 | 24+ | 容器化部署 |
| Make | 3.8+ | - | 构建工具 |

### 开发环境

- **操作系统**: macOS / Linux (Ubuntu 22.04+) / Windows (WSL2)
- **内存**: 8GB+ RAM
- **磁盘**: 20GB+ 可用空间
- **浏览器**: Chrome 100+ / Edge 100+ / Firefox 100+

### 生产环境

| 规模 | 用户数 | CPU | 内存 | 磁盘 |
|------|--------|-----|------|------|
| 小型 | <100 | 2核 | 4GB | 50GB |
| 中型 | 100-500 | 4核 | 8GB | 100GB |
| 大型 | 500+ | 8核+ | 16GB+ | 200GB+ |

---

## 🏗 技术架构

### 技术栈

#### 后端技术

| 技术 | 版本 | 用途 |
|------|------|------|
| Go | 1.25+ | 编程语言 |
| Gin | v1.9+ | Web 框架 |
| Ent | v0.13+ | ORM 框架 |
| PostgreSQL | 14+ | 关系数据库 |
| Redis | 7+ | 缓存/会话 |
| BPMN Engine | - | 工作流引擎 |
| Swagger | latest | API 文档 |

#### 前端技术

| 技术 | 版本 | 用途 |
|------|------|------|
| Next.js | 15.x | React 框架 |
| React | 19.x | UI 库 |
| TypeScript | 5.x | 类型系统 |
| Ant Design | 6.x | UI 组件库 |
| Tailwind CSS | 4.x | CSS 框架 |
| TanStack Query | 5.x | 数据获取 |
| Zustand | 5.x | 状态管理 |

### 系统架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                         客户端层                                  │
│   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐        │
│   │   Web 浏览器  │    │   移动端     │    │   API 调用   │        │
│   └─────────────┘    └─────────────┘    └─────────────┘        │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                       接入层 (Nginx)                             │
│              负载均衡 / SSL 终止 / 静态资源缓存                    │
└─────────────────────────────────────────────────────────────────┘
                              │
              ┌───────────────┴───────────────┐
              ▼                               ▼
┌─────────────────────────┐       ┌─────────────────────────┐
│      Next.js 前端        │       │      Go 后端 API        │
│      端口: 3000          │       │      端口: 8090         │
└─────────────────────────┘       └─────────────────────────┘
              │                               │
              │                               ├──────────────┐
              │                               ▼              ▼
              │                    ┌─────────────┐  ┌─────────┐
              │                    │  PostgreSQL │  │  Redis  │
              │                    │   端口:5432 │  │ 6379    │
              │                    └─────────────┘  └─────────┘
              │                               │
              │                               ▼
              │                    ┌─────────────────────┐
              │                    │    AI 服务层        │
              │                    │  (RAG/分类/摘要)     │
              │                    └─────────────────────┘
              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      存储层                                       │
│     ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│     │   文件存储   │  │  向量存储   │  │  对象存储   │          │
│     │  (MinIO/S3) │  │ (Chroma)   │  │             │          │
│     └─────────────┘  └─────────────┘  └─────────────┘          │
└─────────────────────────────────────────────────────────────────┘
```

---

## ✨ 功能特性

### 核心模块

| 模块 | 功能描述 |
|------|----------|
| 🎫 **工单管理** | 创建、分配、跟踪、关闭工单；SLA 管理；优先级处理 |
| 🔥 **事件管理** | 事件发现、记录、分类、升级；实时监控告警 |
| 🐛 **问题管理** | 根因分析、已知错误库、问题解决跟踪 |
| 🔄 **变更管理** | 变更请求、风险评估、多级审批流程 |
| 📋 **服务目录** | 服务请求模板、自助服务门户、服务水平协议 |
| 📚 **知识库** | RAG 智能搜索、知识分类、FAQ 管理 |
| 🔀 **工作流** | BPMN 可视化设计器、审批流程自动化 |
| 👥 **用户管理** | 多租户、角色权限、LDAP 集成 |
| 📊 **报表** | 自定义报表、BI 仪表盘、数据导出 |

### AI 智能功能

| 功能 | 说明 |
|------|------|
| 🤖 **智能分类** | 自动识别工单类型、优先级、影响范围 |
| 📝 **自动摘要** | AI 生成工单/事件摘要 |
| 🔍 **RAG 知识库** | 基于向量搜索的智能知识推荐 |
| 💡 **智能建议** | 推荐解决方案、相似工单 |

---

## 📚 文档导航

### 快速链接

| 文档 | 说明 |
|------|------|
| [📖 开发指南](./docs/DEVELOPMENT.md) | 开发环境搭建、本地运行、联调配置 |
| [🚀 部署指南](./docs/DEPLOYMENT.md) | 生产环境部署、Docker、K8s |
| [⚙️ 配置参考](./docs/CONFIGURATION.md) | 环境变量详解、配置文件说明 |
| [🗄️ 数据库](./docs/DATABASE.md) | 数据库配置、迁移、备份恢复 |
| [🔧 运维手册](./docs/OPERATIONS.md) | 日志管理、监控告警、故障处理 |
| [🔐 安全指南](./docs/SECURITY.md) | 安全加固、权限管理、审计日志 |
| [📡 API 文档](./docs/API.md) | 接口文档、认证方式 |
| [🏗️ 架构设计](./docs/ARCHITECTURE.md) | 系统架构、技术选型说明 |

### 常用命令

```bash
# Docker 开发环境
make dev-up          # 启动
make dev-down        # 停止
make dev-logs        # 查看日志
make db-shell        # 进入数据库

# 本地开发
cd itsm-backend && go run main.go   # 后端
cd itsm-frontend && npm run dev     # 前端

# 构建
make build-backend    # 构建后端
make build-frontend   # 构建前端

# 测试
make test            # 运行所有测试
make test-coverage   # 生成覆盖率报告

# 代码质量
make lint            # 代码检查
make fmt             # 代码格式化
```

---

## 🤝 参与贡献

欢迎贡献代码！请阅读 [CONTRIBUTING.md](./CONTRIBUTING.md) 了解详情。

### 开发流程

```bash
# 1. Fork 项目
# 2. 克隆并创建分支
git clone https://github.com/heidsoft/itsm.git
cd itsm
git checkout -b feature/your-feature

# 3. 开发并提交
git commit -m "feat: add new feature"

# 4. 推送并创建 PR
git push origin feature/your-feature
```

### 代码规范

- **Go**: 遵循标准 Go 项目结构，使用 `gofumpt` 格式化
- **TypeScript**: 严格模式，ESLint + Prettier
- **提交信息**: 使用 [Conventional Commits](https://www.conventionalcommits.org/) 格式
- **测试**: 新增功能需配套单元测试

---

## 📄 许可证

[MIT](./LICENSE) - 请自由使用和修改。

---

## 📞 联系方式

- **项目地址**: https://github.com/heidsoft/itsm
- **问题反馈**: https://github.com/heidsoft/itsm/issues
- **讨论社区**: https://github.com/heidsoft/itsm/discussions
- **Email**: heidsoft@qq.com

---

<p align="center">
  <strong>⭐ 如果这个项目对你有帮助，请给一个 Star 支持！</strong>
</p>

<p align="center">
  Made with ❤️ by ITSM Team
</p>
