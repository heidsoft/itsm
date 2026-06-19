# ITSM 文档中心

这个目录包含产品说明、部署运维、开发协作、测试报告和阶段性评审文档。为了避免新用户在大量历史文档中迷路，建议先从本页按角色阅读。

## 快速入口

| 角色 | 建议阅读 |
|:---|:---|
| 试用者 | [README 快速开始](../README.md#-快速开始)、[v1.0 GA 收口验收指南](./V1_GA_READINESS.md) |
| 部署人员 | [部署指南](./DEPLOYMENT.md)、[配置参考](./CONFIGURATION.md)、[运维手册](./OPERATIONS.md) |
| 后端开发 | [开发指南](./DEVELOPMENT.md)、[数据库说明](./DATABASE.md)、[后端 CI](../.github/workflows/backend-ci.yml) |
| 前端开发 | [开发指南](./DEVELOPMENT.md)、[前端 CI](../.github/workflows/frontend-ci.yml) |
| 产品/方案 | [开源发布能力说明](./product/open-source-release-capability.md)、[ServiceNow 对标评审](./review/ITSM-ServiceNow-Benchmark-2026-06-18.md) |
| 测试/QA | [角色视角测试方案](./ITSM-基于角色视角的产品测试方案.md)、[深度业务测试报告](./review/ITSM-Deep-Business-Test-Report-2026-06-18.md) |
| 发布维护 | [Release workflow](../.github/workflows/release.yml)、[CI/CD 评审](./delivery/CICD-REVIEW.md) |

## 核心文档

- [部署指南](./DEPLOYMENT.md): Docker Compose、生产部署、反向代理和发布部署建议。
- [配置参考](./CONFIGURATION.md): 环境变量、端口、数据库、Redis、AI 服务配置。
- [开发指南](./DEVELOPMENT.md): 本地开发、前后端命令、调试和常见问题。
- [数据库说明](./DATABASE.md): 数据库迁移、备份和模型说明。
- [运维手册](./OPERATIONS.md): 日志、健康检查、备份、恢复和故障排查。
- [v1.0 GA 收口验收指南](./V1_GA_READINESS.md): 默认能力、连接器、AI 审计和部署模式检查。

## 产品与架构

- [AI-Native ITSM 架构解析](./articles/07-AI-Native-ITSM的架构进化论：Guidance-Harness-Skill三层体系设计.md)
- [开源发布能力说明](./product/open-source-release-capability.md)
- [商业就绪架构评审](./architecture/ITSM-Commercial-Ready-Architecture.md)
- [企业级 v1 就绪度评估](./enterprise-v1-readiness-2026-06-07.md)
- [工作流控制台诊断与设计](./workflow-console-diagnosis-and-design.md)

## 测试与评审

- [角色视角测试方案](./ITSM-基于角色视角的产品测试方案.md)
- [浏览器 E2E 测试报告](./review/ITSM-Browser-E2E-Test-Report-2026-06-18.md)
- [深度业务测试报告](./review/ITSM-Deep-Business-Test-Report-2026-06-18.md)
- [前端 UX Review](./review/ITSM-Frontend-UX-Review-2026-06-19.md)
- [商用就绪验收报告](./review/ITSM-Commercial-Readiness-Acceptance-Report-2026-06-18.md)

## CI/CD 与发布

当前保留的 GitHub Actions:

| Workflow | 作用 | 触发 |
|:---|:---|:---|
| [backend-ci](../.github/workflows/backend-ci.yml) | 后端格式、静态分析、构建、测试、Go module 校验 | 后端代码或 workflow 变化 |
| [frontend-ci](../.github/workflows/frontend-ci.yml) | 前端 lint、类型检查、单测、Next.js standalone 构建 | 前端代码或 workflow 变化 |
| [Security Scan](../.github/workflows/security.yml) | gosec、Trivy、npm audit、TruffleHog | main/develop、PR、每周定时、手动 |
| [Build & Release](../.github/workflows/release.yml) | 多平台后端二进制、前端产物、GitHub Release、GHCR 镜像 | `v*` tag |

已删除的 `ITSM GA Readiness` workflow 与前后端 CI 大量重复，且 E2E 步骤失败不阻断，不适合作为有效门禁。GA 检查保留在 [v1.0 GA 收口验收指南](./V1_GA_READINESS.md) 和 `docs/scripts/smoke-api.sh` 中，需要时手动或后续接入独立 smoke workflow。

## 文档维护原则

- README 只放项目定位、核心能力、最快启动和主入口。
- `docs/README.md` 作为文档导航，不承载大量业务细节。
- 阶段性评审、测试报告、历史复盘保留在 `docs/review/`、`docs/testing/`、`docs/delivery/` 等目录。
- 新增长期有效文档时，优先补到本页索引；临时报告使用日期命名，避免和正式指南混淆。
