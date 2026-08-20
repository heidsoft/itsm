# ITSM 文档中心

> Status: current。首次进入请先阅读[文档状态与事实源](./documentation-governance.md)，确认规范、计划、历史报告和归档材料的优先级。

这个目录包含产品说明、部署运维、开发协作、测试报告和阶段性评审文档。为了避免新用户在大量历史文档中迷路，建议先从本页按角色阅读。

## 快速入口

| 角色 | 建议阅读 |
|:---|:---|
| 试用者 | [README 快速开始](../README.md#快速开始)、[安装指南](./getting-started/install.md) |
| 部署人员 | [部署优化报告](./DEPLOYMENT_OPTIMIZATION.md)、[运维运行手册](./runbooks/production-initialization.md) |
| 后端开发 | [本地开发命令](./dev-commands-reference.md)、[PostgreSQL 升级运行手册](./pg-upgrade-runbook.md)、[后端 CI](../.github/workflows/backend-ci.yml) |
| 前端开发 | [本地开发命令](./dev-commands-reference.md)、[前端 CI](../.github/workflows/frontend-ci.yml) |
| 产品/方案 | [开源发布能力说明](./product/open-source-release-capability.md)、ServiceNow 对标评审（参见 [archive](./archive/reviews/servicenow-benchmark-2026-06-18.md)） |
| 测试/QA | [角色视角测试方案](./testing/role-based-product-test-plan.md)、[深度业务测试报告](./review/deep-business-test-report-2026-06-18.md) |
| 发布维护 | [Release workflow](../.github/workflows/release.yml)、[Production Readiness Program](./delivery/production-readiness-program.md) |
| 文档维护 | [文档状态与事实源](./documentation-governance.md)、[文档命名与维护规范](./documentation-style-guide.md) |

## 核心文档

- [部署优化报告](./DEPLOYMENT_OPTIMIZATION.md): Docker Compose、生产部署、反向代理和发布部署建议。
- [本地开发命令](./dev-commands-reference.md): 本地开发、前后端命令、调试和常见问题。
- [PostgreSQL 升级运行手册](./pg-upgrade-runbook.md): 数据库迁移、备份和模型说明。
- [运维运行手册](./runbooks/production-initialization.md): 日志、健康检查、备份、恢复和故障排查。
- [文档状态与事实源](./documentation-governance.md): 当前规范、计划、测试报告和归档材料的权威层级。
- [文档命名与维护规范](./documentation-style-guide.md): 目录分层、命名规则和归档标准。

## 产品与架构

- [AI-Native ITSM 架构解析](./articles/07-ai-native-architecture-guidance-harness-skill.md)
- [开源发布能力说明](./product/open-source-release-capability.md)
- [商业就绪架构评审](./architecture/commercial-ready-architecture.md)
- [企业级 v1 就绪度评估](./archive/reviews/enterprise-v1-readiness-2026-06-07.md)
- [工作流控制台诊断与设计](./product/workflow-console-diagnosis-and-design.md)

## 测试与评审

- [模块功能复盘与改善迭代计划](./review/module-function-retrospective-2026-07-10.md)
- [角色视角测试方案](./testing/role-based-product-test-plan.md)
- [浏览器 E2E 测试报告](./review/browser-e2e-test-report-2026-06-18.md)
- [深度业务测试报告](./review/deep-business-test-report-2026-06-18.md)
- [前端 UX Review](./review/frontend-ux-review-2026-06-19.md)
- [商用就绪验收报告](./review/commercial-readiness-acceptance-report-2026-06-18.md)

## CI/CD 与发布

当前保留的 GitHub Actions:

| Workflow | 作用 | 触发 |
|:---|:---|:---|
| [backend-ci](../.github/workflows/backend-ci.yml) | 后端格式、静态分析、构建、测试、Go module 校验 | 后端代码或 workflow 变化 |
| [frontend-ci](../.github/workflows/frontend-ci.yml) | 前端 lint、类型检查、单测、Next.js standalone 构建 | 前端代码或 workflow 变化 |
| [api-contract-check](../.github/workflows/api-contract-check.yml) | 前后端 API 路径与字段命名静态校验 | API client、router 或 workflow 变化 |
| [test-coverage-guard](../.github/workflows/test-coverage-guard.yml) | 校验受管源码变更有对应测试 | 受管前后端源码变化 |
| [GA Gate](../.github/workflows/ga-gate.yml) | 启动核心 Compose 栈并执行健康检查与 API 烟测 | 核心应用或编排变化 |
| [Security Scan](../.github/workflows/security.yml) | gosec、Trivy、npm audit、TruffleHog | main/develop、PR、每周定时、手动 |
| [Build & Release](../.github/workflows/release.yml) | 多平台后端二进制、前端产物、GitHub Release、GHCR 镜像 | `v*` tag |

CI 按后端、前端、契约、集成、安全和发布分层。`ga-gate` 只验证组装后的核心栈，不重复执行前后端单元测试。

## 文档维护原则

- README 只放项目定位、核心能力、最快启动和主入口。
- `docs/README.md` 作为文档导航，不承载大量业务细节。
- 阶段性评审、测试报告和历史复盘必须标记为 historical，不能作为当前实现或发布结论。
- 新增长期有效文档时，优先补到本页索引；临时报告使用日期命名，避免和正式指南混淆。
- 历史 bug 报告、过期计划和阶段性复盘统一放入 [archive](./archive/README.md)，避免干扰当前用户路径。
