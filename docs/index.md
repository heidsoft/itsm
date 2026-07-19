# ITSM · AI-Native IT 服务管理

> **一句话**：一个面向 2026+ 工程师的开源 ITSM — 票务/事件/问题/变更、CMDB、知识库、BPMN 工作流、SLA、AI Triage，全部内建。

[项目 README](https://github.com/heidsoft/itsm#readme) · [中文 README](https://github.com/heidsoft/itsm/blob/main/README.zh-CN.md) · [GitHub 仓库](https://github.com/heidsoft/itsm) · [变更日志](https://github.com/heidsoft/itsm/blob/main/CHANGELOG.md)

---

## 🎯 它解决什么问题

传统 ITSM（ServiceNow / BMC / Jira Service Management）存在三类痛点：

| 痛点 | 传统 ITSM | ITSM |
|:---|:---|:---|
| 💸 许可费用 | 每席位 $50-200/月 | Apache 2.0 开源，零许可费 |
| 🤖 AI 集成 | 需要额外购买插件 | 原生集成 LLM Triage / RAG / Agent |
| 🔒 数据主权 | SaaS，数据出境 | 完全自托管，可离线部署 |
| 🔧 扩展性 | 黑盒插件框架 | 模块化 Go service，可直接 fork |

---

## ⚡ 核心能力

### 1. ITSM 标准模块

- **工单 / 事件 / 问题 / 变更**：完整 ITSM 生命周期（创建 → 分派 → 处理 → 关闭 → 复盘）
- **CMDB**：配置项管理 + 影响分析 + 关系图谱
- **服务目录**：服务请求、审批流、Provisioning 自动化
- **SLA**：定义 → 监控 → 告警 → 升级 → 合规报告
- **知识库**：文章管理 + 全文搜索 + RAG 语义检索
- **BPMN 工作流**：可视化流程设计器 + 引擎（基于 `nitram509/lib-bpmn-engine`）
- **审批链**：多级审批 + 委托 + 超时升级
- **MSP 多租户**：跨租户资源调度（Managed Service Provider 场景）
- **资产/许可**：硬件资产、软件 License 管理

### 2. AI 能力

- **AI Triage**：LLM 自动分类 + 紧急度评估 + 智能分派
- **RAG 知识问答**：基于向量库的语义搜索 + 来源追溯
- **A2UI**：AI 主动生成表单（替代传统下拉菜单）
- **Agent 工具调用**：工单 Agent 可调用 connector + 外部 API
- **MCP 技能市场**：itsm-skill 插件 + Connector 集成（飞书、Slack、钉钉、Teams）

### 3. 平台能力

- **多租户隔离**：基于 `tenant_id` 的行级隔离
- **RBAC + ABAC**：角色 + 表达式（`expr-lang/expr`）混合权限模型
- **审计日志**：所有写操作留痕
- **WebSocket 实时通知**：工单状态变更、审批提醒
- **i18n**：中英双语
- **Prometheus metrics** + **结构化日志**（`zap`）

---

## 🏗️ 技术栈

```
┌────────────────────────────────────────────────────────────┐
│ Frontend:  Next.js 15 + React 19 + TypeScript 5.7 + Antd 6 │
│            + Zustand + Tailwind + Playwright                │
├────────────────────────────────────────────────────────────┤
│ Backend:   Go 1.25 + Gin + Ent ORM + PostgreSQL + Redis     │
│            + BPMN Engine + LLM Gateway + RAG (pgvector)    │
├────────────────────────────────────────────────────────────┤
│ Tooling:   itsm-cli (Node) + itsm-agent (Go) + itsm-skill   │
│            + itsm-ai-service (Python) + itsm-rag (Python)   │
├────────────────────────────────────────────────────────────┤
│ Ops:       Docker Compose + GitHub Actions + GHCR + Pages   │
│            + Prometheus + Grafana + nginx                   │
└────────────────────────────────────────────────────────────┘
```

---

## 🚀 快速开始

```bash
# 1. 克隆
git clone https://github.com/heidsoft/itsm.git
cd itsm

# 2. 配置环境变量
cp .env.dev.example .env.dev
# 编辑 .env.dev，仅用于本地开发

# 3. 一键启动（dev 模式）
docker compose -f docker-compose.dev.yml --env-file .env.dev --profile dev up -d --build

# 4. 访问
#   前端  http://localhost:3000
#   后端  http://localhost:8090/api/v1/health
#   默认账号 admin / admin123（生产环境务必修改！）
```

完整步骤见 [快速开始 - 安装](getting-started/install.md)。

---

## 📊 当前状态（v1.1）

| 维度 | 现状 | 目标 | 进度 |
|:---|:---|:---|:---:|
| 核心代码 CI | 后端、前端分层验证 | 持续加固 | ✅ |
| API 契约 | 路径静态校验 + 前端契约测试 | 扩大运行时覆盖 | 🟡 |
| 组装验证 | Compose GA Gate + API 烟测 | 补充角色 E2E | 🟡 |
| 安全扫描 | gosec、Trivy、npm audit、TruffleHog | 持续治理 | ✅ |
| 文档站点 | MkDocs 构建 + GitHub Pages | 持续维护 | ✅ |
| 依赖治理 | Dependabot 周更新，人工审核合并 | 保持 | ✅ |

详见 [Roadmap](roadmap.md) 和 [覆盖审计](testing/coverage-audit.md)。

---

## 🤝 贡献

欢迎 PR！流程：

1. Fork → 创建分支
2. 阅读 [CONTRIBUTING.md](https://github.com/heidsoft/itsm/blob/main/CONTRIBUTING.md)
3. 提交 PR（CI 会按变更路径执行构建、lint、测试、契约与集成检查）
4. 等待 review（首次贡献者会被自动欢迎 🎉）

详细规则见 [贡献指南](contributing.md)。

---

## 📜 许可证

Apache License 2.0 — 见 [LICENSE](https://github.com/heidsoft/itsm/blob/main/LICENSE)。

## 🙏 致谢

- 灵感来自 ServiceNow / Jira Service Management / BMC Helix
- BPMN 引擎：[nitram509/lib-bpmn-engine](https://github.com/nitram509/lib-bpmn-engine)
- ORM：[Ent](https://entgo.io)
- AI/LLM：[DeepSeek](https://deepseek.com) · [OpenAI](https://openai.com) · [Anthropic](https://anthropic.com)
