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

ITSM 是一个现代化的企业级 IT 服务管理平台，采用 Go/Gin 后端和 Next.js/React 前端构建。支持 ITIL 最佳实践，集成 AI 智能功能，帮助企业高效管理 IT 服务。

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

### 方式一：Docker Compose（推荐）

```bash
# 克隆项目
git clone https://github.com/heidsoft/itsm.git
cd itsm

# 启动所有服务
docker-compose up -d

# 访问应用
# 前端：http://localhost:3000
# 后端：http://localhost:8080
# API 文档：http://localhost:8080/swagger/index.html
```

### 方式二：本地开发

#### 1. 环境准备

```bash
# 安装 Go 1.25+
# https://go.dev/dl/

# 安装 Node.js 22+
# https://nodejs.org/

# 安装 PostgreSQL 14+
# https://www.postgresql.org/download/
```

#### 2. 后端启动

```bash
cd itsm-backend

# 安装依赖
go mod download

# 配置数据库（编辑 config.yaml）
vim config.yaml

# 运行数据库迁移
go run -tags migrate main.go

# 启动后端服务
go run main.go

# 后端运行在 http://localhost:8080
```

#### 3. 前端启动

```bash
cd itsm-frontend

# 安装依赖
npm install

# 配置环境变量
cp .env.example .env.local
# 编辑 .env.local，设置 API 地址

# 启动开发服务器
npm run dev

# 前端运行在 http://localhost:3000
```

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

### 开发流程

1. **Fork 项目**
   ```bash
   # 在 GitHub 上 Fork 项目
   ```

2. **克隆项目**
   ```bash
   git clone https://github.com/your-username/itsm.git
   cd itsm
   ```

3. **创建分支**
   ```bash
   git checkout -b feature/your-feature
   ```

4. **开发并提交**
   ```bash
   # 开发完成后
   git add -A
   git commit -m "feat: add your feature"
   ```

5. **推送并创建 PR**
   ```bash
   git push origin feature/your-feature
   # 在 GitHub 上创建 Pull Request
   ```

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

### 运行测试

```bash
# 后端测试
cd itsm-backend
go test ./... -v

# 前端测试
cd itsm-frontend
npm test
```

### 测试覆盖率

```bash
# 后端覆盖率
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# 前端覆盖率
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
