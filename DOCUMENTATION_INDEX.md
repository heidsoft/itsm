# ITSM 文档体系导航

**最后更新**: 2026-03-04

欢迎使用 ITSM 文档！本文档索引帮助您快速找到所需信息。

---

## 📚 按场景查找

### 我刚开始接触项目
**👉 从这里开始：**
1. [README.md](./README.md) - 项目概览、快速开始（5分钟上手）
2. [docs/DEVELOPMENT.md](./docs/DEVELOPMENT.md) - 开发环境搭建详细步骤

### 我要部署到生产环境
**📖 必读文档：**
1. [docs/DEPLOYMENT.md](./docs/DEPLOYMENT.md) - Docker Compose 部署、K8s 部署
2. [docs/OPERATIONS.md](./docs/OPERATIONS.md) - 监控、备份、故障排除
3. [docs/SECURITY.md](./docs/SECURITY.md) - 安全配置、防火墙、SSL

### 我是开发人员
**🔧 开发指南：**
- [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) - C4 架构图、技术选型、模块说明
- [docs/API.md](./docs/API.md) - 所有 API 端点详情、错误码、示例代码
- [CONTRIBUTING.md](./CONTRIBUTING.md) - 代码规范、PR 流程、测试要求

### 我遇到了问题
**🆘 排障帮助：**
1. [docs/TROUBLESHOOTING.md](./docs/TROUBLESHOOTING.md) - 常见问题、错误码速查
2. [docs/OPERATIONS.md](./docs/OPERATIONS.md#日志查看) - 日志查看、诊断步骤
3. [README.md#常见问题](./README.md#常见问题) - 快速 FAQ

### 我需要配置系统
**⚙️ 配置文件参考：**
- [Makefile](./Makefile) - 所有便捷命令（`make dev-up`、`make test` 等）
- [.env.example](./.env.example) - 环境变量完整说明
- [.golangci-lint.yml](./.golangci-lint.yml) - Go 代码检查配置
- [OPENAPI.yaml](./OPENAPI.yaml) - API 规范（可用于生成客户端）
- [docker-compose.dev.yml](./docker-compose.dev.yml) - 开发环境
- [docker-compose.prod.yml](./docker-compose.prod.yml) - 生产环境
- [Dockerfile.backend](./Dockerfile.backend) - 后端镜像构建
- [Dockerfile.frontend](./Dockerfile.frontend) - 前端镜像构建

---

## 📖 文档清单（完整列表）

| 文档 | 用途 | 适用对象 | 更新日期 |
|------|------|----------|----------|
| [README.md](./README.md) | 项目首页、快速开始 | 所有人 | 2026-03-04 |
| [CONTRIBUTING.md](./CONTRIBUTING.md) | 贡献指南 | 开发者 | 2026-03-04 |
| [DOCUMENTATION_INDEX.md](./DOCUMENTATION_INDEX.md) | 本文档 | 管理员 | 2026-03-04 |
| | | | |
| [docs/DEVELOPMENT.md](./docs/DEVELOPMENT.md) | 开发环境搭建 | 开发者、运维 | 2026-03-04 |
| [docs/DEPLOYMENT.md](./docs/DEPLOYMENT.md) | 生产部署 | 运维、DevOps | 2026-03-04 |
| [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) | 系统架构设计 | 架构师、开发者 | 2026-03-04 |
| [docs/API.md](./docs/API.md) | API 参考文档 | 前端、后端、集成者 | 2026-03-04 |
| [docs/OPERATIONS.md](./docs/OPERATIONS.md) | 运维手册 | 运维工程师 | 2026-03-04 |
| [docs/SECURITY.md](./docs/SECURITY.md) | 安全指南 | 安全团队、运维 | 2026-03-04 |
| [docs/TROUBLESHOOTING.md](./docs/TROUBLESHOOTING.md) | 故障排除 | 所有人 | 2026-03-04 |

---

## 🚀 快速命令参考

### 开发环境

```bash
# 首次启动（Docker Compose）
make dev-up              # 启动所有服务
make dev-down            # 停止服务
make dev-logs            # 查看日志
make dev-shell           # 进入后端容器

# 本地开发（非 Docker）
make run                 # 启动后端（需先有数据库）
make frontend-run        # 启动前端
```

### 测试

```bash
make test                # 运行所有测试
make test-backend        # 后端测试
make test-frontend       # 前端测试
make test-coverage       # 覆盖率报告
```

### 代码质量

```bash
make lint                # 代码检查
make lint-backend        # 后端检查
make lint-frontend       # 前端检查
make fmt                 # 代码格式化
```

### 数据库

```bash
make db-migrate          # 运行迁移
make db-seed             # 导入初始数据
make db-backup           # 备份
make db-shell            # 进入数据库命令行
```

### 生产环境

```bash
make prod-up             # 启动生产环境
make prod-down           # 停止
make prod-logs           # 查看日志
make health              # 健康检查
```

### 构建与发布

```bash
make build               # 构建镜像
make build-backend       # 构建后端镜像
make build-frontend      # 构建前端镜像
make docker-push         # 推送镜像到仓库
make ci                  # 运行 CI 检查（lint + test）
```

---

## 📊 技术栈速查

### 后端

| 技术 | 版本 | 用途 |
|------|------|------|
| Go | 1.25+ | 编程语言 |
| Gin | v1.9+ | Web 框架 |
| GORM | v1.25+ | ORM |
| PostgreSQL | 14+ | 主数据库 |
| Redis | 7+ | 缓存/会话 |

### 前端

| 技术 | 版本 | 用途 |
|------|------|------|
| Next.js | 15.5 | React 框架 |
| React | 19 | UI 库 |
| TypeScript | 5 | 类型系统 |
| Ant Design | 6 | UI 组件 |
| Tailwind CSS | 4 | CSS 框架 |

---

## 🔗 常用链接

| 服务 | 地址 | 说明 |
|------|------|------|
| 前端（开发） | http://localhost:3000 | Next.js Dev Server |
| 后端（开发） | http://localhost:8080 | Gin API Server |
| API 文档 | http://localhost:8080/swagger | Swagger UI |
| 健康检查 | http://localhost:8080/health | 健康状态 |
| 指标数据 | http://localhost:9090/metrics | Prometheus |
| Grafana（开发） | http://localhost:3001 | 监控面板（可选） |

**生产环境**:
- API: `https://api.yourdomain.com`
- 前端: `https://app.yourdomain.com`
- Grafana: `https://monitor.yourdomain.com`

---

## ❓ 如何选择文档？

**Q: 我想快速运行项目，该看哪个文档？**
A: 看 [README.md#快速开始](./README.md#快速开始)，5 分钟启动。

**Q: 我想了解系统架构，该看哪个文档？**
A: 看 [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md)，包含 C4 模型图和模块说明。

**Q: 我想调用 API 开发前端，该看哪个文档？**
A: 看 [docs/API.md](./docs/API.md)，或访问 `http://localhost:8080/swagger` 交互式文档。

**Q: 生产环境部署需要注意什么？**
A: 看 [docs/DEPLOYMENT.md](./docs/DEPLOYMENT.md) 和 [docs/SECURITY.md](./docs/SECURITY.md)。

**Q: 遇到错误怎么排查？**
A: 先查 [docs/TROUBLESHOOTING.md](./docs/TROUBLESHOOTING.md)，再查 [docs/OPERATIONS.md#日志查看](./docs/OPERATIONS.md#日志查看)。

**Q: 我想贡献代码，流程是什么？**
A: 阅读 [CONTRIBUTING.md](./CONTRIBUTING.md)，了解代码规范和 PR 流程。

---

## 🔄 文档版本

本文档体系对应 ITSM **v1.0** 版本。

| 版本 | 日期 | 更新说明 |
|------|------|----------|
| v1.0 | 2026-03-04 | 初始版本，完整文档体系 |

---

## 📝 维护指南

如需更新文档，请：

1. **Fork 项目**
2. **修改对应文档**（Markdown 格式）
3. **提交 PR**，描述变更内容
4. **等待审查**

**文档规范**:
- 使用 Markdown 语法
- 代码块标注语言（如 \`\`\`bash）
- 命令需经过验证（可在开发环境测试）
- 包含版本要求（如 Go 1.25+）
- 关键术语保留英文（如 API、CLI、Token）

---

## 🆘 需要帮助？

- 📮 提交 [Issue](https://github.com/heidsoft/itsm/issues)
- 💬 加入 [Discussions](https://github.com/heidsoft/itsm/discussions)
- 📧 邮件联系: heidsoft@qq.com

**祝您使用愉快！** 🎉
