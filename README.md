# ITSM - Enterprise IT Service Management Platform

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25-blue" alt="Go">
  <img src="https://img.shields.io/badge/Next.js-15.5-000000" alt="Next.js">
  <img src="https://img.shields.io/badge/TypeScript-5-3178C6" alt="TypeScript">
  <img src="https://img.shields.io/badge/License-MIT-yellow" alt="License">
  <br>
  <img src="https://github.com/heidsoft/itsm/actions/workflows/frontend-ci.yml/badge.svg?branch=main" alt="Frontend CI">
  <img src="https://github.com/heidsoft/itsm/actions/workflows/backend-ci.yml/badge.svg?branch=main" alt="Backend CI">
  <img src="https://github.com/heidsoft/itsm/actions/workflows/automated-tests.yml/badge.svg?branch=main" alt="Automated Tests">
</p>

<p align="center">
  <strong>🚀 企业级 IT 服务管理平台 | 基于 ITIL 最佳实践 | AI 智能赋能</strong>
</p>

---

## 📖 项目简介

ITSM 是一个现代化的企业级 IT 服务管理(ITSMB)平台，基于 Go/Gin 后端和 Next.js/React 前端构建。遵循 ITIL 最佳实践，集成 AI 智能功能，帮助企业高效管理 IT 服务。

### 什么是 ITSM?

ITSM(IT Service Management)是一种通过服务生命周期管理 IT 服务的方法，包括设计、交付、管理和改进 IT 服务。本项目实现了:
- **服务台**: 用户请求的统一入口
- **事件管理**: 快速恢复服务，减少业务中断
- **问题管理**: 找到根本原因，防止事件重复发生
- **变更管理**: 安全、可控地进行系统变更
- **知识库**: 积累和共享组织知识

### 目标用户

- IT 运维团队
- 服务台工程师
- IT 经理
- 开发团队(自建工单系统需求)

### ✨ 核心特性

- 🎫 **工单管理** - 完整的工单生命周期管理，SLA 支持，自动化工作流
- 🔥 **事件管理** - 实时监控，智能分类，自动升级
- 🐛 **问题管理** - 根因分析，已知错误库
- 🔄 **变更管理** - 风险评估，多级审批
- 📋 **服务目录** - 自助服务门户，服务请求
- 🤖 **AI 赋能** - 智能分类，RAG 知识库，自动摘要
- 📊 **BPMN 工作流** - 可视化流程设计，自定义审批流
- ⏱️ **SLA 监控** - 服务级别协议管理，实时告警

---

## 🚀 快速开始

### 方式一：Docker Compose（推荐，5分钟内启动）

```bash
# 1. 克隆项目
git clone https://github.com/heidsoft/itsm.git
cd itsm

# 2. 启动所有服务（包含 PostgreSQL、Redis、后端和前端）
docker-compose up -d

# 3. 等待服务启动（约1-2分钟），然后访问：
# 前端：http://localhost:3000
# 后端：http://localhost:8080
# API 文档：http://localhost:8080/swagger
```

> **首次登录**: 用户名 `admin`，密码 `admin123`

### 方式二：本地开发

#### 1. 环境要求

| 工具 | 版本 | 安装链接 |
|------|------|----------|
| Go | 1.25+ | [go.dev/dl](https://go.dev/dl/) |
| Node.js | 22+ | [nodejs.org](https://nodejs.org/) |
| PostgreSQL | 14+ | [postgresql.org](https://www.postgresql.org/download/) |
| Redis | 7+ | [redis.io](https://redis.io/download/) |

#### 2. 后端启动

```bash
cd itsm-backend

# 安装依赖
go mod download

# 配置环境变量（开发环境使用 trust 认证，无需密码）
cp .env.example .env

# 启动后端服务（自动创建数据库表和初始数据）
go run main.go
# 后端运行在 http://localhost:8090
```

> **注意**: 后端启动时会自动：
> - 创建数据库表结构
> - 创建默认租户（tenant）
> - 创建默认管理员用户 admin/admin123
> - 创建示例部门和团队数据
> - 创建其他初始测试用户（user1/user123, security1/security123）

#### 2.1 PostgreSQL 配置（首次部署）

如果首次部署，需要先创建数据库：

```bash
# macOS (Homebrew 安装)
brew services start postgresql@14

# 创建数据库
psql -U postgres -c "CREATE DATABASE itsm;"
# 或使用 trust 认证（开发环境）
psql -U postgres -c "CREATE DATABASE itsm OWNER dev;"
```

> **生产环境提示**: 生产部署时需要配置强密码，并修改 `config.yaml` 或设置环境变量 `DB_PASSWORD`

#### 3. 前端启动

```bash
cd itsm-frontend

# 安装依赖
npm install

# 启动开发服务器（无需额外配置）
npm run dev
# 前端运行在 http://localhost:3000
```

> **注意**: 前端默认连接 `http://localhost:8090` 作为后端 API 地址。如需修改，编辑 `.env.local` 文件。

### 快速验证

服务启动后，验证各组件是否正常运行:

```bash
# 后端健康检查
curl http://localhost:8090/health

# 测试登录
curl -X POST http://localhost:8090/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'

# 访问 API 文档
# 浏览器打开: http://localhost:8090/swagger

# 前端页面
# 浏览器打开: http://localhost:3000
```

---

## 📡 API 接口规范

### 响应格式

所有 API 返回统一的 JSON 格式:

```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

| 状态码 | 说明 |
|--------|------|
| `code: 0` | 成功 |
| `code: 1001+` | 参数错误 |
| `code: 2001` | 认证失败 |
| `code: 5001` | 服务器内部错误 |

### 常用端点

| 模块 | 端点前缀 | 说明 |
|------|----------|------|
| 认证 | `/api/v1/auth` | 登录、注册、令牌刷新 |
| 工单 | `/api/v1/tickets` | 工单 CRUD 操作 |
| 事件 | `/api/v1/incidents` | 事件管理 |
| 问题 | `/api/v1/problems` | 问题管理 |
| 变更 | `/api/v1/changes` | 变更管理 |
| 知识库 | `/api/v1/knowledge` | 知识文章 |
| 工作流 | `/api/v1/workflows` | BPMN 流程 |

### 认证方式

```bash
# 使用 Bearer Token
curl -H "Authorization: Bearer <token>" http://localhost:8080/api/v1/tickets
```

启动后访问 `http://localhost:8080/swagger` 查看完整 API 文档。

---

## 📁 项目结构

```
itsm/
├── itsm-backend/              # Go 后端
│   ├── cmd/                   # 命令行工具
│   ├── config/                # 配置文件
│   ├── controller/            # 控制器层
│   ├── service/               # 服务层
│   ├── repository/            # 数据访问层
│   ├── model/                 # 数据模型
│   ├── dto/                   # 数据传输对象
│   ├── middleware/            # 中间件
│   └── docs/                  # API 文档
│
├── itsm-frontend/             # Next.js 前端
│   ├── src/
│   │   ├── app/               # 页面路由
│   │   ├── components/        # 组件
│   │   ├── lib/               # 工具库
│   │   ├── hooks/             # React Hooks
│   │   └── types/             # TypeScript 类型
│   ├── public/                # 静态资源
│   └── tests/                 # 测试文件
│
├── docs/                      # 项目文档
│   ├── OPTIMIZATION_SUMMARY.md    # 优化总结
│   ├── DEVELOPMENT_STATUS.md      # 开发状态
│   └── ...                      # 更多文档
│
├── .github/workflows/         # CI/CD 配置
│   ├── frontend-ci.yml        # 前端 CI
│   ├── backend-ci.yml         # 后端 CI
│   └── automated-tests.yml    # 自动化测试
│
└── docker-compose.yml         # Docker 配置
```

---

## 🛠️ 技术栈

### 后端技术

| 技术 | 版本 | 用途 |
|------|------|------|
| Go | 1.25 | 编程语言 |
| Gin | v1.9+ | Web 框架 |
| Ent ORM | v0.13+ | ORM 框架 |
| PostgreSQL | 14+ | 数据库 |
| Redis | 7+ | 缓存 |
| Swagger | latest | API 文档 |

### 前端技术

| 技术 | 版本 | 用途 |
|------|------|------|
| Next.js | 15.5 | React 框架 |
| React | 19 | UI 库 |
| TypeScript | 5 | 类型系统 |
| Ant Design | 6 | UI 组件库 |
| Tailwind CSS | 4 | CSS 框架 |
| TanStack Query | 5 | 数据获取 |
| Zustand | 5 | 状态管理 |

### DevOps 工具

| 工具 | 用途 |
|------|------|
| GitHub Actions | CI/CD |
| Docker | 容器化 |
| Docker Compose | 本地开发环境 |

---

## 📊 项目状态

### ✅ 已完成

- [x] 核心工单管理
- [x] 事件管理
- [x] 问题管理
- [x] 变更管理
- [x] 服务目录
- [x] BPMN 工作流引擎
- [x] SLA 管理
- [x] 知识库
- [x] AI 智能功能
- [x] CI/CD 流程
- [x] 文档完善

### 🔄 进行中

- [ ] 测试覆盖率提升（目标：60%）
- [ ] 性能优化
- [ ] 移动端适配

### ⏳ 计划中

- [ ] E2E 测试
- [ ] 性能监控
- [ ] 安全加固
- [ ] 多语言支持

---

## ❓ 常见问题

### Q1: 启动时数据库连接失败

**问题**: `dial tcp localhost:5432: connect: connection refused`

**解决**:
- 确认 PostgreSQL 已启动: `pg_ctl -D /usr/local/var/postgres start`
- 或使用 Docker: `docker run -d -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgres:14`

### Q2: 前端无法连接后端 API

**问题**: `Network Error` 或 `CORS` 错误

**解决**:
- 确认后端已启动在 http://localhost:8080
- 检查浏览器控制台是否有 CORS 错误
- 确认 `.env.local` 中 `NEXT_PUBLIC_API_URL=http://localhost:8080`

### Q3: Redis 连接失败

**问题**: `dial tcp localhost:6379: connect: connection refused`

**解决**:
- 启动 Redis: `redis-server`
- 或使用 Docker: `docker run -d -p 6379:6379 redis:7`

### Q4: 数据库迁移失败

**问题**: `pq: password authentication failed`

**解决**:
- 编辑 `itsm-backend/config.yaml`，修改数据库密码
- 或修改 PostgreSQL 配置允许密码登录

### Q5: 如何运行单个模块的测试?

```bash
# 后端 - 只测试 tickets 模块
cd itsm-backend
go test ./service/ticket_service_test.go ./service/ticket_service.go -v

# 前端 - 只测试某个组件
cd itsm-frontend
npm test -- --testPathPattern=TicketList
```

### Q6: 贡献代码需要了解什么?

- 后端遵循 Go 标准项目结构，参考 [CLAUDE.md](./CLAUDE.md)
- 前端使用 Next.js App Router + TypeScript
- 提交前请运行: `go test ./...` (后端) 和 `npm run type-check` (前端)

---

## 📚 文档导航

### 开发文档

- [📝 优化总结](./docs/OPTIMIZATION_SUMMARY.md) - 完整优化报告
- [📈 开发状态](./docs/DEVELOPMENT_STATUS_2026_02_28.md) - 最新开发进度
- [🔧 CI 优化指南](./docs/CI_OPTIMIZATION_SUMMARY.md) - CI/CD 优化详情
- [📋 代码优化计划](./docs/CODE_OPTIMIZATION_PLAN.md) - 4 阶段优化策略

### 部署文档

- [🚀 部署指南](./docs/DEPLOYMENT.md) - 生产环境部署
- [🐳 Docker 部署](./docker-compose.yml) - Docker 配置说明

### API 文档

- [📖 Swagger UI](http://localhost:8080/swagger/index.html) - 本地 API 文档
- [📄 API 规范](./itsm-backend/docs/) - API 设计文档

---

## 🤝 贡献指南

我们欢迎所有形式的贡献！无论是 bug 修复、新功能开发还是文档完善。

### 新手友好任务

如果你初次贡献，可以从以下任务开始:

- [ ] 改进文档（README、代码注释）
- [ ] 添加单元测试覆盖
- [ ] 修复简单 bug（拼写错误、样式问题）
- [ ] 翻译界面文本
- [ ] 添加新的 UI 组件

### 开发流程

1. **Fork 项目**
   点击 GitHub 页面右上角的 Fork 按钮

2. **克隆项目**
   ```bash
   git clone https://github.com/your-username/itsm.git
   cd itsm
   ```

3. **添加上游仓库**
   ```bash
   git remote add upstream https://github.com/heidsoft/itsm.git
   ```

4. **创建功能分支**
   ```bash
   git checkout -b feature/your-feature
   # 或修复 bug
   git checkout -b fix/issue-description
   ```

5. **开发并提交**
   ```bash
   # 开发完成后
   git add .
   git commit -m "feat: add your feature"
   ```

6. **保持分支更新**
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

7. **推送并创建 PR**
   ```bash
   git push origin feature/your-feature
   # 在 GitHub 上创建 Pull Request
   ```

8. **等待 Code Review**
   维护者会尽快审核您的 PR，请耐心等待

### 代码规范

#### Go 代码

```bash
# 格式化代码
gofumpt -w .

# 运行测试
go test ./... -v

# 运行静态检查
staticcheck ./...
```

#### TypeScript 代码

```bash
# 类型检查
npm run type-check

# Lint 检查
npm run lint

# 格式化代码
npm run format
```

### 提交信息规范

```
feat: 新功能
fix: 修复 bug
docs: 文档更新
style: 代码格式
refactor: 重构
test: 测试
chore: 构建/工具
```

---

## 🧪 测试

### 快速测试

```bash
# 后端 - 运行所有测试
cd itsm-backend
go test ./... -v

# 前端 - 运行所有测试
cd itsm-frontend
npm test
```

### 单元测试 vs 集成测试

```bash
# 后端单元测试
cd itsm-backend
go test ./service/... -v

# 前端单元测试
cd itsm-frontend
npm run test:unit

# 前端集成测试
npm run test:integration

# E2E 测试（需要后端运行）
npm run test:e2e
```

### 测试覆盖率

```bash
# 后端覆盖率报告
cd itsm-backend
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# 前端覆盖率
cd itsm-frontend
npm run test:coverage
```

---

## 📈 CI/CD 状态

| 工作流 | 状态 | 链接 |
|--------|------|------|
| Frontend CI | ✅ 100% | [查看](https://github.com/heidsoft/itsm/actions/workflows/frontend-ci.yml) |
| Backend CI | ✅ 100% | [查看](https://github.com/heidsoft/itsm/actions/workflows/backend-ci.yml) |
| Automated Tests | ✅ 100% | [查看](https://github.com/heidsoft/itsm/actions/workflows/automated-tests.yml) |
| Release | ✅ Ready | [查看](https://github.com/heidsoft/itsm/actions/workflows/release.yml) |

---

## 🗺️ 路线图

### Q1 2026

- [x] CI/CD 流程优化
- [x] 代码质量提升
- [ ] 测试覆盖率达到 60%
- [ ] 性能优化

### Q2 2026

- [ ] E2E 测试框架
- [ ] 性能监控
- [ ] 安全加固
- [ ] 移动端适配

### Q3 2026

- [ ] 多语言支持
- [ ] 插件系统
- [ ] 高级分析报表
- [ ] 自动化运维

---

## 👥 团队

### 维护者

- [@刘彬](https://github.com/heidsoft) - Project Lead

### 贡献者

感谢所有贡献者！🙏

[![Contributors](https://contrib.rocks/image?repo=heidsoft/itsm)](https://github.com/heidsoft/itsm/graphs/contributors)

---

## 📄 许可证

本项目采用 [MIT 许可证](./LICENSE)

---

## 📞 联系方式

- **项目地址**: https://github.com/heidsoft/itsm
- **Issues**: https://github.com/heidsoft/itsm/issues
- **Discussions**: https://github.com/heidsoft/itsm/discussions
- **Releases**: https://github.com/heidsoft/itsm/releases
- **Email**: heidsoft@qq.com

---

## 🙏 致谢

感谢以下开源项目：

- [Ant Design](https://ant.design/) - UI 组件库
- [Next.js](https://nextjs.org/) - React 框架
- [Gin](https://gin-gonic.com/) - Go Web 框架
- [BPMN.js](https://bpmn.io/) - BPMN 工作流设计器
- 所有贡献者和支持者！

---

<p align="center">
  <strong>⭐ 如果这个项目对你有帮助，请给一个 Star 支持！</strong>
</p>

<p align="center">
  Made with ❤️ by ITSM Team
</p>
