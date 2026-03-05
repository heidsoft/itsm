# ITSM 项目文档索引

**最后更新**: 2026-03-04
**版本**: v2.0

---

## 文档导航

本索引帮助您快速找到所需的文档。请根据您的需求选择相应的文档。

---

## 新手入门

| 文档 | 说明 | 适合人群 |
|------|------|----------|
| [README.md](../README.md) | 项目首页，快速了解 | 所有人 |
| [DEVELOPMENT.md](./DEVELOPMENT.md) | 开发环境搭建指南 | 开发者 |
| [DEPLOYMENT.md](./DEPLOYMENT.md) | 生产环境部署指南 | 运维人员 |

---

## 核心文档

### 配置与设置

| 文档 | 说明 |
|------|------|
| [CONFIGURATION.md](./CONFIGURATION.md) | 环境变量完整配置参考 |
| [DATABASE.md](./DATABASE.md) | 数据库配置、迁移、备份 |
| [POSTGRESQL_GUIDE.md](./POSTGRESQL_GUIDE.md) | PostgreSQL 详细使用指南 |
| [LOGGING.md](./LOGGING.md) | 日志管理与分析 |

### 开发与构建

| 文档 | 说明 |
|------|------|
| [BUILD_STARTUP.md](./BUILD_STARTUP.md) | 程序编译与启动指南 |
| [DEVELOPMENT.md](./DEPLOYMENT.md) | 开发环境搭建 |

### 运维

| 文档 | 说明 |
|------|------|
| [OPERATIONS.md](./OPERATIONS.md) | 运维手册 |
| [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) | 故障排除 |
| [BACKUP_RECOVERY.md](./BACKUP_RECOVERY.md) | 备份与恢复 |

---

## API 与架构

| 文档 | 说明 |
|------|------|
| [API.md](./API.md) | API 接口文档 |
| [API-DOCUMENTATION.md](./API-DOCUMENTATION.md) | API 开发指南 |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | 系统架构设计 |

---

## 安全与合规

| 文档 | 说明 |
|------|------|
| [SECURITY.md](./SECURITY.md) | 安全指南 |
| [PERMISSION_MANAGEMENT.md](./PERMISSION_MANAGEMENT.md) | 权限管理 |

---

## 模块指南

| 文档 | 说明 |
|------|------|
| [TICKET_MANAGEMENT.md](./TICKET_MANAGEMENT.md) | 工单管理 |
| [INCIDENT_MANAGEMENT.md](./INCIDENT_MANAGEMENT.md) | 事件管理 |
| [PROBLEM_MANAGEMENT.md](./PROBLEM_MANAGEMENT.md) | 问题管理 |
| [CHANGE_MANAGEMENT.md](./CHANGE_MANAGEMENT.md) | 变更管理 |
| [SLA_MANAGEMENT.md](./SLA_MANAGEMENT.md) | SLA 管理 |
| [KNOWLEDGE_BASE.md](./KNOWLEDGE_BASE.md) | 知识库 |
| [WORKFLOW_GUIDE.md](./WORKFLOW_GUIDE.md) | 工作流 |

---

## 部署方式

| 文档 | 说明 |
|------|------|
| [DEPLOYMENT.md](./DEPLOYMENT.md) | Docker Compose 部署 |
| [ALIYUN_ECS_DEPLOYMENT.md](./ALIYUN_ECS_DEPLOYMENT.md) | 阿里云 ECS 部署 |
| [WEBSITE_DEPLOYMENT.md](./WEBSITE_DEPLOYMENT.md) | 网站部署 |

---

## 快速命令参考

### 开发环境

```bash
# 启动开发环境
make dev-up

# 查看日志
make logs

# 进入数据库
make db-shell

# 停止环境
make dev-down
```

### 构建与测试

```bash
# 构建后端
make build-backend

# 构建前端
make build-frontend

# 运行测试
make test

# 代码检查
make lint
```

### 部署

```bash
# 生产环境启动
make prod-up

# 生产环境停止
make prod-down

# 查看生产日志
make prod-logs
```

---

## 文档版本历史

| 版本 | 日期 | 更新内容 |
|------|------|----------|
| v1.0 | 2026-01-01 | 初始版本 |
| v2.0 | 2026-03-04 | 完善配置、编译启动、日志文档 |

---

## 贡献文档

欢迎为项目贡献文档！请阅读 [CONTRIBUTING.md](../CONTRIBUTING.md) 了解如何参与。

---

## 获取帮助

- 💬 [GitHub Discussions](https://github.com/heidsoft/itsm/discussions)
- 🐛 [Issue Tracker](https://github.com/heidsoft/itsm/issues)
- 📧 Email: heidsoft@qq.com

---

**文档维护**: ITSM 团队
**最后更新**: 2026-03-04
