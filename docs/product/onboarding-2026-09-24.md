# ITSM 30 分钟入门 — 新人版

> **受众**：加入团队 1-2 周的新人 / 跨团队协作同事
> **阅读时间**：30 分钟（午休/通勤可读完）
> **读完你能**：① 知道 ITSM 是什么 ② 知道这个项目是什么 ③ 知道业务怎么组织 ④ 知道关键术语 ⑤ 知道怎么上手
> **不读你能**：❌ 不能改业务代码（还需要读具体 domain handler）❌ 不能拍产品决策（需要看齐活林的拍板件）
> **关联**：本文是 [业务快照](./business-snapshot-2026-09-24.md) 的"新人友好版"，不替代；细节去快照查

---

## 1. 这是一份什么文档（1 分钟）

你在仓库 `/Users/heidsoft/Downloads/research/itsm/`。这是 ITSM 全栈代码库——Go/Gin 后端 + Next.js 前端 + Python AI 服务，目标**对标 ServiceNow 的开源企业级服务管理平台**。

**30 分钟读完本文**，你会知道：
- 这个项目的定位和边界（不是 ServiceNow 全集复制）
- 12 个业务域、4 条主链路
- 5 个核心概念（BPMN / CMDB / RBAC / SLA / RAG）
- 代码结构（哪里写新代码、哪里不要碰）
- 治理闭环（为什么有这么多机器守卫）

**30 分钟后**，你可以：
- 跟同事对齐"我在做哪个域"
- 在 Slack / 会议上听懂术语
- 找到对应文档入口
- 跑起来本地 dev 环境

---

## 2. ITSM 是什么（3 分钟）

**ITSM** = IT Service Management，企业 IT 部门**管理服务全生命周期**的方法论与系统。

**企业 IT 部门每天做的事**：
- 用户报障："邮箱打不开了" → 工单
- 运维处理：派单给一线 → 二线 → 三线
- 例行变更："周三凌晨数据库升级" → 审批 → 实施 → 复盘
- 知识沉淀："这类问题以前怎么解决的" → 知识库
- SLA 监控："P1 工单 30 分钟必须响应" → 告警

**ITIL** 是 ITSM 的**方法论框架**（IT Infrastructure Library），定义"事件 / 问题 / 变更 / 发布 / 服务请求"等流程。本项目对齐 ITIL v3/v4 核心流程。

**ServiceNow** 是 ITSM 领域的**商业巨头**（年营收 $10B+），几乎成为行业事实标准。本项目定位：**比 ServiceNow 更轻、更私有部署、更适合中国市场**——不是替代 ServiceNow，而是给"买不起 / 不愿被锁定"的企业另一选择。

---

## 3. 这个项目是什么（3 分钟）

### 3.1 定位

| 项 | 内容 |
|---|---|
| **全称** | itsm（对标 ServiceNow 的开源企业级 IT 服务管理平台） |
| **目标市场** | 中国市场（私有化部署为主，SaaS + MSP 为辅） |
| **核心差异化** | ① 更轻的私有部署（单机可交付） ② 更强的本地企业集成（飞书 / 钉钉 / 企微原生） ③ AI-Native（把 AI 做进服务管理生命周期） |
| **当前阶段** | v1.6.x 收官期，核心 ITIL 流程可用；v1.7 启动 AI 评估器 + 飞书生产连接器 |
| **技术栈** | 后端 Go/Gin + Ent/Zap/Redis；前端 Next.js/TS + Antd v6；AI 服务 Python（RAG）；Docker Compose |

### 3.2 数字快照（代码现状）

- **后端**：handlers/ 67 域 · service/ 337 文件 · ent schema 137 个 · 装配文件 1776 行
- **前端**：168 页面 · 257 组件 · 78 菜单
- **测试**：后端 338 单测 / 22+5 BPMN E2E；前端 209 单测 / 64 Playwright
- **成熟度**：12 域里 3 个 GA 候选 + 9 个 Pilot + 0 个 Disabled

### 3.3 我们不做什么（NON-GOALS）

- 不复刻 ServiceNow 的全部 CSDM / Discovery / ITOM / HRSD
- 不把每个现有页面都升级为商业承诺
- 不用 AI 替代人类审批、权限、状态机
- 不同时生产化所有 IM 渠道（飞书优先，其他 Experimental）
- 不用演示数据或前端本地状态证明功能可用

---

## 4. 业务架构（5 分钟）

**6 层架构**(完整图见 [业务快照 §2](./business-snapshot-2026-09-24.md#2-业务架构图))：

```
L0 入口        邮件接入 / IM-Webhook（收消息）
L1 编排        BPMN 引擎（★ 唯一流程编排层）
L2 业务域      工单/事件 · 变更 · 问题/KE · 服务目录/请求（4 主链路）
L3 上下文      CMDB（全业务域引用同一 CI 身份）
L4 横切        AI / SLA / 知识-RAG / 连接器 / RBAC
L5 运营        报表
```

### 4.1 4 条商业 MVP 主链路

| 链路 | 名称 | 起点 → 终点 |
|---|---|---|
| **A** | 事件到恢复 | 告警/人工报障 → 事件 → CI → SLA → 派单 → 处理 → 恢复 → 关闭 |
| **B** | 重复事件到知识 | 重复事件 → 问题 → 根因 CI → workaround → Known Error → 解决 → 知识草稿 → 审核发布 → RAG 可检索 |
| **C** | 受控变更 | 变更创建 → 受影响 CI → 风险策略 → CAB/BPMN → 窗口 → 实施 → 验证/回滚 → PIR → 关闭 |
| **D** | 服务请求到交付 | 目录项 → 表单 → 审批 → 交付任务 → CI 创建 → 验证 → 完成 |

**商业目标不是让每个菜单都有页面，而是让这 4 条主链路可以由真实客户数据连续运行、失败可恢复、过程可审计、结果可验收**。

---

## 5. 5 个核心概念（8 分钟）

新人最容易卡在术语上。下面 5 个核心概念必须懂。

### 5.1 BPMN（业务流程建模标记）

**类比**：BPMN 像**流程图编辑器**（Visio / draw.io），但能执行。

- **定义**：BPMN 2.0 标准，用图形符号（开始 / 结束 / 任务 / 网关 / 事件）描述业务流程
- **能力**：可视化设计 → 引擎执行 → 任务分配 → 状态追踪
- **本项目约束**：**BPMN 是唯一流程编排层**（CONSTRAINTS#3）。任何业务域**不允许**有第二套审批或状态流引擎
- **代码位置**：`itsm-backend/handlers/bpmn/`（引擎）+ `router/bpmn_*.go`（路由）
- **何时碰**：变更审批、新工作流设计、流程优化

⚠️ **当前陷阱**：项目里有**三套审批运行时并存**（BPMN + ApprovalService + ApprovalChainService），违反 CONSTRAINTS#3，正在 D-1 决策收敛中（详见 [决策对话 §D-1](./decision-dialogue-2026-09-24.md#d-1--审批三运行时去留最硬骨头)）。

### 5.2 CMDB（配置管理数据库）

**类比**：CMDB 像**企业 IT 的"花名册"**——记录所有 IT 资产（服务器、数据库、应用、网络设备）及其关系。

- **核心实体**：
  - **CI**（Configuration Item）：一个具体的资产（如"订单数据库 MySQL-5"）
  - **CI Type**：资产类型（如"MySQL Server、Container、Kubernetes Service"）
  - **CI Relationship**：CI 间关系（如"应用 A 依赖 数据库 B"）
- **能力**：拓扑图、影响分析、对账、质量报表
- **关键约束**（CONSTRAINTS#2）：**CMDB 是流程上下文，不是独立资产表**——事件 / 问题 / 变更 / 请求必须引用同一 CI 身份
- **代码位置**：`itsm-backend/handlers/cmdb/`（主体 21 端点）+ `ent/schema/configuration_item*.go`
- **何时碰**：CI 创建、关系建模、影响分析、对账
- **ADR-003**：`docs/architecture/adr-003-cmdb-tenant-wide-default.md`——**CMDB 显式 tenant-wide 默认权限**（IT infra 跨部门可见是工作常态）

### 5.3 RBAC（基于角色的访问控制）

**类比**：RBAC 像**权限开关矩阵**——每个角色有"能做什么"的清单。

- **核心实体**：
  - **Permission**（权限码）：如 `ticket:create`、`cmdb:read`
  - **Role**（角色）：如 `admin`、`agent`、`end_user`
  - **User-Role 边**：用户 ↔ 角色（可多对多）
  - **ResourceActionMap**：资源 + 动作 → 权限码的预检映射
- **能力**：接口级 ACL + 行级 DataScope（6 域已落地）+ 审计
- **代码位置**：`internal/authz/`（单一真源）+ `cmd/authz-gen`（代码生成器）
- **何时碰**：加新接口 / 加新角色 / 加新权限
- **铁律**：
  - 改词表流程：`internal/authz` → `go run ./cmd/authz-gen` → 跑 5 道守卫
  - **新路由必须在 ResourceActionMap 对应方法段加条目**——放错方法段=没放（POST 落进 GET 段会误判）
  - 守卫必须做"注入回归→FAIL→恢复"验证，否则可能空转
  - 排查 403：先分轨道（业务域走 RequirePermission，SmartCheckPermission 只挂 `/auth+` / `/msp`）

### 5.4 SLA（服务水平协议）

**类比**：SLA 像**计时器 + 闹钟**——给每个工单 / 事件 / 请求设"必须 X 分钟内响应 / Y 分钟内解决"，违规就告警。

- **核心实体**：
  - **SLA Policy**（策略）：截止时间、违规处理、升级动作
  - **SLA Instance**（实例）：挂到工单上的 SLA（开始计时 / 暂停 / 恢复）
- **能力**：定义、截止时间、违规、预警、通知、指标、逐租户 watcher
- **代码位置**：`itsm-backend/handlers/sla/` + `ent/schema/sla*.go`
- **何时碰**：配置 SLA 模板 / 改 SLA 策略 / 处理违规告警
- **历史踩坑**：9-15 R4-b 修复了"模板键名与解析器错配 + 时区问题"（`ce1bc7ef`）——已闭环

### 5.5 RAG（检索增强生成）

**类比**：RAG 像**带搜索的 AI 助手**——LLM 不是"凭记忆"回答，而是先在知识库里检索相关文章，再基于检索结果回答。

- **核心实体**：
  - **Knowledge Article**（知识文章）：Markdown / 富文本
  - **Vector Index**（向量索引）：文章嵌入向量，用于相似度检索
- **能力**：关键词检索 → 向量检索降级 → LLM 问答 → 租户过滤
- **代码位置**：`itsm-backend/handlers/knowledge/`（知识）+ `itsm-rag/`（独立服务，Python）
- **何时碰**：知识发布 / 索引更新 / AI 问答调优

---

## 6. 4 条主链路（5 分钟）

新人最容易犯的错：**埋头改某个域，不看主链路**。下面 4 条主链路是商业承诺，**改任何代码都要先问自己"这影响哪条主链路？"**

### 链路 A：事件到恢复

- **业务**：`告警/人工报障 → 事件 → CI → SLA → 派单 → 处理 → 恢复 → 关闭`
- **关键**：**CI 关联**（必须）+ **SLA 计时**（自动）+ **BPMN 触发**（可执行）
- **典型故障**：CI 关联失败 = 不知道影响哪些资产；SLA 算错 = 误告警
- **owner**：后端 incident domain

### 链路 B：重复事件到知识

- **业务**：`重复事件 → 问题 → 根因 CI → workaround → Known Error → 解决 → 知识草稿 → 审核发布 → RAG 可检索`
- **关键**：**CI 强引用**（不能是字符串）+ **知识版本管理** + **向量索引更新**
- **典型故障**：CI 用字符串引用 = 退役后查不到；知识发布后 RAG 检索不到
- **owner**：后端 problem / known_error / knowledge domain

### 链路 C：受控变更

- **业务**：`变更创建 → 受影响 CI → CMDB 影响摘要 → 风险策略 → CAB/BPMN → 窗口 → 实施 → 验证/回滚 → PIR → 关闭`
- **关键**：**影响分析门禁化**（不能跳过）+ **审批与状态原子** + **回滚计划**
- **典型故障**：漏掉影响 CI = 变更引发新事件；审批通过但状态未更新
- **owner**：后端 change / cab / release domain
- **状态**：⚠️ 变更口径 README "可用" vs 契约 "Pilot"——D-W1 决策中

### 链路 D：服务请求到交付

- **业务**：`目录项 → 表单 → SLA/流程绑定 → 审批 → 交付任务 → CI 创建/变更 → 验证 → 完成/补偿`
- **关键**：**目录版本固定** + **审批与交付分态** + **Provisioning 幂等**
- **典型故障**：审批通过但 CI 没创建；创建 CI 失败但状态显示成功
- **owner**：后端 service_catalog / service_request domain

---

## 7. 代码库怎么走（3 分钟）

### 7.1 仓库结构（全栈）

```
itsm/
├── itsm-backend/        # Go 后端（Gin + Ent + Redis）
├── itsm-frontend/       # Next.js 前端（TS + Antd v6）
├── itsm-ai-service/     # Python AI 服务（LLM Gateway + RAG 协调）
├── itsm-rag/            # Python RAG 服务（向量检索）
├── itsm-cli/            # Node CLI 工具
├── itsm-agent/          # Go agent（运行在客户内网，采集数据）
├── itsm-skill/          # 可声明技能
├── docs/                # 文档（architecture / product / articles / ...）
├── output/              # 审计/治理报告（产物勿入 Git）
├── scripts/docs-gate/   # 文档门禁脚本
├── prd/                 # 产品需求文档（部分）
├── plans/               # 实施计划（部分）
├── specs/               # spec-kit 产物
├── .workbuddy/memory/   # AI 工作记忆
└── Makefile             # dev / build / docs-gate 入口
```

### 7.2 后端结构

```
itsm-backend/
├── main.go              # 入口
├── handlers/<domain>/   # 目标架构（垂直切片，只在这里加新代码）
├── service/             # 共享业务逻辑（冻结扩张，逐步迁入 handlers/）
├── controller/          # 冻结（已清空，不要恢复）
├── ent/schema/          # 数据库 schema（Ent ORM）
├── middleware/          # auth / tenant / audit
├── dto/                 # 请求/响应 DTO
├── router/              # 路由注册（20 个 routes 文件）
├── internal/
│   ├── bootstrap/       # 装配（目前 1776 行，反向恶化）
│   ├── authz/           # 权限码单一真源
│   └── container/       # ⚠️ 0 引用的 DI 容器，D-W2/O1 待删
├── cmd/authz-gen/       # 权限码生成器
└── integration/         # 集成测试
```

### 7.3 前端结构

```
itsm-frontend/src/
├── app/                 # Next.js App Router（168 个 page.tsx）
├── components/          # 257 组件（注意巨型组件待拆）
├── lib/                 # 工具库（API client / hooks）
├── types/               # TypeScript 类型
└── ...
```

### 7.4 关键纪律

1. **新代码只进 `handlers/<domain>/`**；`controller/` 已冻结
2. **改权限词表**：`internal/authz` → `go run ./cmd/authz-gen` → 跑 5 道守卫
3. **改 schema**：`go build -o /tmp/entc entgo.io/ent/cmd/ent && cd ent && /tmp/entc generate ./schema`
4. **改 PR 前必跑**：`make docs-gate`（产品口径守卫）

---

## 8. 治理怎么运转（2 分钟）

项目有**两套机器守卫**（防止文档 / 代码漂移）：

### 8.1 授权平面 5 道守卫（零漂移）

| 守卫 | 形式 |
|---|---|
| 写路由必挂门 | `middleware/route_scan.go` |
| 预检全 (resource, action) 对齐 | `middleware/precheck_codegen.go` |
| 路由码 ⊆ 权威源 | `permission_code_catalog_guard_test.go` |
| 生成物新鲜度 | `TestPrecheckMapIsFresh` |
| 跨包 parity | `TestRoutePrecheckAlignment` |

**结论**：授权平面零漂移，因为有机器守门。

### 8.2 产品口径 5 道守卫（刚上线）

| 规则 | 检查 | 状态 |
|---|---|---|
| C.6.1 | 成熟度口径双向闭合（README vs 契约） | 4 waiver 待销（2026-10-31 反向 FAIL） |
| C.6.2 | 领域清单走目录（删硬编码） | ✅ 已落地 |
| C.6.3 | 新建包必被 router 引用（零引用 FAIL） | 3 waiver 待销（同上） |
| C.6.4 | 表面棘轮（只降不升） | ✅ 已落地 |
| C.6.5 | 覆盖率口径披露 | 1 waiver 待销 |

**关键洞察**：**有机器守卫的平面不漂移，只靠文档约定的平面必然漂移**。当前 5 项漂移全部是"只写在文档里、没机器校验"的平面。

### 8.3 治理节奏

- **每周一**：P0 看板更新（`output/global-p0-*.md`）
- **每个 PR**：`make docs-gate` 必须通过
- **每周节点**：docs-gate 反向 FAIL 触发立即修复

---

## 附 A：5 分钟 Quickstart

```bash
# 1. 启动 dev 环境
docker compose -f docker-compose.dev.yml up -d
# 等待 2-3 分钟（itsm-init 跑迁移 + 种子）

# 2. 访问
# 前端：http://localhost:3000（默认账号 admin / admin123）
# 后端：http://localhost:8090（API 文档 /swagger/index.html）

# 3. 跑后端测试
cd itsm-backend
go test ./... -count=1

# 4. 跑前端测试
cd itsm-frontend
npm run test:unit

# 5. 跑治理守卫
make docs-gate
```

---

## 附 B：必读文档清单

按顺序读完本文 30 分钟 + 下面 6 个文档，新人即可上手：

| 序 | 文档 | 时间 | 读完你能 |
|---|---|---|---|
| 0 | 本文（onboarding-2026-09-24） | 30 分钟 | 全景认知 |
| 1 | [业务快照](./business-snapshot-2026-09-24.md) | 15 分钟 | 12 域 + 22 项 P0-P2 |
| 2 | [AGENTS.md](../../AGENTS.md) | 20 分钟 | 后端分层、文档同步纪律 |
| 3 | [商业能力契约](./itsm-commercial-capability-contract.md) | 30 分钟 | 12 域详细 + CONSTRAINTS |
| 4 | [领域所有权](../architecture/domain-ownership.md) | 10 分钟 | 各域迁移状态、不变量 |
| 5 | [决策对话](./decision-dialogue-2026-09-24.md) | 15 分钟 | 4 项决策详细论证（改代码前必看） |

**总计**：30 + 15 + 20 + 30 + 10 + 15 = **2 小时**（2 个下午读完）

---

## 附 C：常用命令速查

```bash
# 代码生成
go build -o /tmp/entc entgo.io/ent/cmd/ent
cd itsm-backend/ent && /tmp/entc generate ./schema

# 权限码生成
cd itsm-backend && go run ./cmd/authz-gen

# 守卫
make docs-gate          # 产品口径 + 文档门禁
make product-drift      # 仅 C.6 产品口径

# 测试
go test ./...                                # 后端全量
npm run test:unit                            # 前端单测
npm run test:e2e                             # 前端 E2E
go test ./integration -v                     # 后端集成

# Docker
docker compose -f docker-compose.dev.yml up -d    # dev 环境
docker compose -f docker-compose.prod.yml up -d   # prod 环境

# Git
git log --oneline -10                        # 最近 10 提交
git grep -l "TODO" src/                      # 找 TODO
```

---

## 附 D：找同事问什么

| 我想... | 问谁 | 渠道 |
|---|---|---|
| 改某个业务域代码 | 看 [owner-boards](./owner-boards-2026-09-24.md) @backend-itsm / @fe-itsm | Slack #itsm-dev |
| 加新权限 / 新角色 | backend-itsm | Slack #itsm-dev |
| 加新 IM 渠道 | im-itsm | Slack #itsm-im |
| 改业务契约 / 主链路 | product（齐活林） | 邮件 |
| 改 CI / 文档门禁 | devops-itsm | Slack #itsm-devops |
| 提需求 / 反馈 | product（齐活林） | 邮件 |
| 找历史决策 | 看 [decision-dialogue](./decision-dialogue-2026-09-24.md) + [`output/`](../../output/) | 自助 |

---

*新人入门 · 日期：2026-09-24 · 维护者：docs-itsm · 下次更新：2026-10-24（月度）*