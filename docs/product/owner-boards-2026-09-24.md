# 按 Owner 拆分看板 — 2026-09-24

> **目的**：把 [业务快照 §4](./business-snapshot-2026-09-24.md#4-p0p1p2-优先级看板精简版) 的 22 项 P0/P1/P2 按 owner 拆成可执行子看板——每人只看自己负责的项 + 具体 verification 步骤。
> **纪律**：每项 verification 必须可立刻执行（命令 / grep / 跑测试 / git rm），不写"修一下"这类模糊描述。
> **来源**：本看板的 owner 列与业务快照 §4.1/§4.2/§4.3 完全一致；任何字段冲突以业务快照为准。

---

## 总览：22 项 × 7 owner

| owner | P0(止血) | P1(季度) | P2(半年) | 主导合计 |
|---|---|---|---|---|
| **@backend-itsm** | 5 | 4 | 3 | **12** |
| **@docs-itsm** | 1 | 2 | 1 | 4 |
| **@devops-itsm** | 1 | 0 | 1 | 2 |
| **@im-itsm** | 1 | 1 | 0 | 2 |
| **@fe-itsm** | 0 | 0 | 4 | 4 |
| **@ai-itsm** | 1(协作) | 0 | 0 | 0(主导) |
| **@product** | 3(决策权) | 0 | 0 | 0(技术主导) |

**协作热点**：
- P0-1 审批三运行时 → backend-itsm 主导 + product 决策权
- P0-2 变更成熟度 → docs-itsm 主导 + product 决策权
- P0-5 NON-GOALS 多渠道 → im-itsm 主导 + product 决策权
- P0-8 AI 资产所有权 → backend-itsm 主导 + ai-itsm 协作

**整体节奏**：
- 本周：5 项 P0 waiver 销账（D-1 拍板 + D-W1/D-W2/D-2 销账 + P0-6 CI 验证）
- T+1 周：D-1 工程启动 + im/docs 协作项推进
- T+2 周：D-1 进入灰度
- 2026-10-31：所有 P0 waiver 销账 + Gate C.6 不再反向 FAIL

---

# @devops-itsm

**看板定位**：CI 流水线 + 部署编排 + 监控告警

## 我的 P0（止血期 · 1 项）

### P0-6 · CI 全绿验证（本批 4 commit）

**截止**：T+24h（推送后 1 天内）
**commit 范围**：`c5d786fd` / `f226c94a` / `ce1bc7ef` / `68992a9d`

#### Verification（可执行步骤）

```bash
# 1. 确认 commit 在 origin/main
git log --oneline -4 origin/main

# 2. 跑 4 条流水线（GitHub Actions 面板或本地等效）
gh run list --workflow=backend-ci    --limit=1  # 应 SUCCESS
gh run list --workflow=frontend-ci   --limit=1  # 应 SUCCESS
gh run list --workflow=docs-gate     --limit=1  # 应 SUCCESS
gh run list --workflow=security      --limit=1  # 应 SUCCESS

# 3. 截图归档（贴到本节末尾的"验收留痕"）
gh run view <run-id> --json status,conclusion > ci-evidence.json
```

#### 验收留痕
- [ ] backend-ci 全绿
- [ ] frontend-ci 全绿
- [ ] docs-gate 全绿（含 C.6.1-C.6.5 hard）
- [ ] security 全绿（trivy + govulncheck）

## 我的 P2（半年视角 · 1 项）

### P2-1 · 缩编 prod 默认编排（监控栈 → profile 可选）

**截止**：v1.7
**改动目标**：`docker-compose.prod.yml` 的 prometheus/alertmanager/grafana/3 exporter 改为 `--profile observability` 可选，默认只起 7 个核心服务

#### Verification
- [ ] `docker compose --profile observability up -d` 起完整监控栈
- [ ] `docker compose up -d`（无 profile）只起 7 个核心服务
- [ ] `docker ps | wc -l` = 7（无 profile 时）
- [ ] 文档更新 README"轻量化私有部署"卖点

## 协作项（我是配角 · 0 项）

无

## 本周时间盒（@devops-itsm）

| 时段 | 动作 |
|---|---|
| T+0h~T+24h | P0-6 CI 全绿验证（4 流水线） |
| T+24h~T+48h | 留 buffer 给 backend-itsm 的 CI 反馈 |
| 本周末 | 检视 P0 看板 22 项，更新状态 |

---

# @backend-itsm

**看板定位**：Go 后端核心 + ent schema + 业务编排 + RBAC + BPMN

## 我的 P0（止血期 · 5 项）

### P0-1 · 审批三运行时去留拍板（主导）

**截止**：本周内拍板 + T+1.5 周工程完成
**关联决策**：D-1（[决策对话](../../output/decision-memo-2026-09-23.md) 我的倾向 A 收敛到 BPMN）
**协作**：product 决策权

#### Verification（可执行步骤）

```bash
# 阶段 1:立项（拍板后）
git checkout -b refactor/approval-converge-to-bpmn
gh issue create --title "refactor(approval): 收敛到 BPMN (D-1)"

# 阶段 2:legacy 标记
grep -rn "NewApprovalService\|NewApprovalChainService" internal/bootstrap/app.go
# 应输出 2 行 → 改成 legacy=true 包裹

# 阶段 3:灰度开关(环境变量)
# 在 config_loader 增加 APPROVAL_USE_BPMN_ONLY (default=false)
# 在 router/approval_chain_routes.go 加 middleware 校验

# 阶段 4:删除
git rm ent/schema/approval_records.go ent/schema/approval_workflow.go \
       ent/schema/approvalchain.go ent/schema/process_approval_decision.go \
       ent/schema/servicerequestapproval.go
rm service/approval_service.go service/approval_chain_service.go

# 阶段 5:跑全量
go build ./... && go test ./... -count=1

# 阶段 6:CHANGELOG 撤回
# 在 CHANGELOG.md 1.6.10 加 "上版本描述不准确" 段
```

#### 验收（最终态）
- [ ] `grep -c "NewApprovalService\|NewApprovalChainService" internal/bootstrap/app.go` = 1（仅 BPMN 装配点）
- [ ] `ls ent/schema | grep -i approval` = 2（`bpmn_approval_tasks` + `bpmn_approval_history`）
- [ ] `go build ./...` 通过
- [ ] frontend 8 入口 → 2 入口（`admin/approvals` 走 BPMN + `admin/approval-chains` 只读历史）
- [ ] CHANGELOG 撤回段可见
- [ ] 探针矩阵 0 异常（跑一周生产）

### P0-3 · 3 个孤儿包删除（department / root_cause / dashboard）

**截止**：2026-10-31（waiver 销账）
**协作**：无（独立执行）

#### Verification（可执行步骤）

```bash
# 1. 前置 grep（必须在动手前确认无引用）
git grep -l "handlers/department" frontend/ docs/ tests/ 2>/dev/null
git grep -l "handlers/root_cause" frontend/ docs/ tests/ 2>/dev/null
git grep -l "handlers/dashboard"   frontend/ docs/ tests/ 2>/dev/null
# 三条命令应输出为空

# 2. prod 部署验证
docker compose -f docker-compose.prod.yml up -d backend
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8090/api/v1/incidents
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8090/api/v1/changes
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8090/api/v1/problems
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8090/api/v1/cmdb/cis
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8090/api/v1/cmdb/topology
# 应全部 200

# 3. 删除
git rm -r itsm-backend/handlers/department
git rm -r itsm-backend/handlers/root_cause
git rm -r itsm-backend/handlers/dashboard

# 4. 同步文档
# CHANGELOG.md [Unreleased] 加:清理:删除 3 个零路由领域包
# docs/architecture/domain-ownership.md 移除相关段落
# scripts/docs-gate/product-drift-waivers.txt 移除 D-W2

# 5. 守卫验证
make docs-gate  # 应 C.6.3 PASS
```

#### 验收
- [ ] 前置 grep 全部为空
- [ ] prod 5 域核心场景 200
- [ ] 3 个目录已 git rm
- [ ] `go build ./...` 通过
- [ ] CHANGELOG + domain-ownership 同步
- [ ] waiver 销账
- [ ] `make docs-gate` 通过

### P0-4 · DI 容器删除（internal/container 247 行 0 引用）

**截止**：2026-10-31（waiver 销账）
**协作**：无（独立执行）

#### Verification（可执行步骤）

```bash
# 1. 确认零引用
grep -rn "internal/container" itsm-backend --include='*.go'
# 应仅命中 1 个 .md（service/README-sequence-service.md）

# 2. 删除
git rm itsm-backend/internal/container/container.go

# 3. 同步 README
# service/README-sequence-service.md 移除对 internal/container 的引用

# 4. 跑全量
go build ./... && go test ./... -count=1

# 5. waiver 销账
# scripts/docs-gate/product-drift-waivers.txt 移除 O1
make docs-gate  # 应 PASS
```

#### 验收
- [ ] grep 命中为 0（`.md` 也移除引用）
- [ ] `git rm` 完成
- [ ] `go build ./...` 通过
- [ ] `make docs-gate` 通过

### P0-7 · CMDB 阿里云云发现闭环

**截止**：v1.7
**改动**：阿里云 ECS 的 Job/Worker/Diff/对账/审计 5 件套 + 质量指标

#### Verification（可执行步骤）

```bash
# 1. Job 落地
ls itsm-backend/handlers/cmdb/jobs/aliyun_ecs*.go  # 应存在
# 含 sync/diff/reconcile/audit 4 个 worker

# 2. Diff 逻辑
grep -rn "aliyun.*diff\|cloud.*reconcile" itsm-backend/service/cmdb/ | head

# 3. 审计完整
ls itsm-backend/ent/schema | grep -i cmdb_audit

# 4. 质量指标
grep -rn "cmdb.*quality\|stale.*ratio" itsm-backend/handlers/cmdb/

# 5. E2E
go test ./integration -run TestCMDBAliyunSync -v
```

#### 验收
- [ ] 4 个 worker + 审计 schema 存在
- [ ] 阿里云连通 E2E 通过（mock + 真实连通各 1 例）
- [ ] CMDB 完整率/同步覆盖指标上线

### P0-8 · AI 资产所有权 ADR

**截止**：v1.7 启动前
**协作**：ai-itsm
**目标**：明确 LLM 走 gateway / RAG 二选一（itsm-rag 或后端 rag_service）/ itsm-ai-service 与 itsm-agent 归档或编排

#### Verification

```bash
# 1. ADR 文件存在
ls docs/architecture/adr-ai-asset-ownership.md  # 应存在

# 2. 决策表
grep -E "llm_gateway|rag_service|itsm-rag|itsm-ai-service|itsm-agent" \
  docs/architecture/adr-ai-asset-ownership.md | wc -l
# 应至少 5 行提及

# 3. 编排落实
grep -rn "itsm-ai-service\|itsm-rag\|itsm-agent" docker-compose.prod.yml
# 应至少 2 个服务在主编排

# 4. 8 月评审 R8 销账
grep -rn "ai_asset_ownership" docs/review/architecture-review-2026-06-14.md
# 应有对应段落
```

#### 验收
- [ ] ADR 文件落地
- [ ] `itsm-ai-service` 已在 prod compose 默认编排
- [ ] 8 月评审 R8 标记完成

## 我的 P1（本季度 · 4 项）

### P1-1 · 问题 → KE → 知识 → RAG 闭环（主导）

**协作**：docs-itsm（知识发布）
**改动**：问题解决 → KE workaround → 知识草稿 → 审核发布 → 向量索引更新

#### Verification
- [ ] 问题详情页有"创建知识草稿"按钮
- [ ] 知识发布后 5 分钟内向量索引更新
- [ ] 向量删除生效（`DELETE /api/v1/knowledge/{id}/vectors`）
- [ ] E2E：问题 → KE → 知识 → RAG 检索 hit

### P1-2 · 服务目录 → 请求 → CI → 交付闭环

**改动**：Provisioning 生命周期 + 失败补偿 + CI 保留请求来源

#### Verification
- [ ] 服务请求 delivery 状态机：pending/running/succeeded/failed/manual_takeover
- [ ] 失败可重试 + 人工接管按钮
- [ ] 创建的 CI 保留 request_id + executor_id

### P1-3 · SLA 工作日历 / 暂停 / 升级

**改动**：日历配置 + 暂停/恢复事件 + 升级策略

#### Verification
- [ ] SLA 模板支持日历配置（`work_days` + `holidays`）
- [ ] 暂停事件审计可查
- [ ] 升级动作触发通知

### P1-5 · `service/` 分治（337 文件拆分）

**改动**：按 bpmn/approval/ticket/cmdb/rag/reporting 分组；冻结 service/ 新增文件需评审

#### Verification
```bash
ls itsm-backend/service/*.go | wc -l  # 应比 337 减少（首批 -20%）
ls itsm-backend/service/bpmn/ 2>/dev/null  # 应存在
ls itsm-backend/service/cmdb/ 2>/dev/null  # 应存在
```

## 我的 P2（半年视角 · 3 项）

### P2-2 · 统一指标 + 报表合并（与 fe-itsm 协作）

主导 backend 指标层；前端 13 报表页 → 1 套统一指标 + 下钻视图

### P2-6 · V1 fire-and-forget `go func` 删除

```bash
grep -n "go func" itsm-backend/service/ticket_service.go | head
# 应输出 0 行（修复 ticket_service.go:180,471,484,2494 等 4 处）
```

### P2-7 · DataScope department tier 死代码清理

```bash
grep -rn "ApplyTicketFilter\|DepartmentID" itsm-backend --include='*.go'
# ApplyTicketFilter 应 0 调用方；ctx.DepartmentID 0 注入点
```

## 协作项（我是配角 · 0 项）

主导 P0-1 的工程实施 + 主导 P0-8 的 ADR 起草；其他 owner 通过我会签

## 本周时间盒（@backend-itsm）

| 时段 | 动作 |
|---|---|
| T+0 | 等齐活林批 D-1（决策对话 §1） |
| T+1d~T+2d | D-1 立项 + legacy 标记 |
| T+2d~T+5d | D-W2 三孤儿包删除（半天）+ D-1 灰度开关 |
| T+5d~T+7d | D-W1 D-2 文档配合 + 守卫验证 |
| T+7d~T+14d | D-1 灰度生产 + 验证 |
| 本周末 | 检视 P1-1~P1-5 启动条件 |

---

# @fe-itsm

**看板定位**：Next.js 前端 + 组件库 + 菜单路由 + 测试

## 我的 P0（止血期 · 0 项）

**注**：本周期前端无 P0 必修项。但要**配合 backend-itsm** 完成 P0-1 审批改写（详见协作项）。

## 我的 P1（本季度 · 0 项）

业务快照 §4.2 中前端未分配主导 P1 项。前端 P1-18（`/approvals` 路由前后缀统一）已在 canonical §6.1 标记，但本看板按 owner 重新归并后归入 P0-1 协作。

## 我的 P2（半年视角 · 4 项）

### P2-3 · 前端覆盖率口径披露 + 收尾

**截止**：v1.7

#### Verification

```bash
# 当前状态
grep collectCoverageFrom itsm-frontend/jest.config.js
# 当前只含 src/lib/** → 80% 门槛 ≠ 产品覆盖率

# 修复方案 A(扩范围)
# jest.config.js collectCoverageFrom 扩到 src/app/** / src/components/**
# 同步 ROADMAP 披露口径

# 修复方案 B(披露现状)
# ROADMAP 明确声明"覆盖率仅统计 src/lib"
```

#### 验收
- [ ] 口径方案选定并写入 ROADMAP
- [ ] CI 守卫验证 (`make docs-gate`)

### P2-4 · 前端巨型组件拆解 + 测试

**截止**：v1.7
**目标**：`BPMNDesigner` 1497 / `TicketDetail` 1382 / `WorkflowNodeInspector` 1745 拆解后单测覆盖率 ≥ 60%

#### Verification

```bash
# 当前最大组件
wc -l itsm-frontend/src/components/{bpmn,BPMNDesigner}/**/*.tsx 2>/dev/null | sort -rn | head
wc -l itsm-frontend/src/components/ticket/TicketDetail.tsx 2>/dev/null

# 拆解后
# - BPMNDesigner.tsx 单文件 < 500 行
# - TicketDetail.tsx 单文件 < 500 行
# - 每个拆出子组件 *.test.tsx 覆盖率 ≥ 60%
```

### P2-5 · 依赖收尾

**截止**：v1.7
**目标**：删 `swr` 死依赖 + 图表/图标/日期库各选其一 + zustand vs react-query 边界写进前端规范

#### Verification

```bash
# swr 死依赖
grep -rn "from \"swr\"" itsm-frontend/src/  # 应 0 命中
grep -rn "\"swr\":" itsm-frontend/package.json  # 应已删除

# 图表库
grep -E "recharts|ant-design/charts" itsm-frontend/package.json  # 应只保留 1 个

# 图标库
grep -E "lucide-react|ant-design/icons" itsm-frontend/package.json  # 应只保留 1 个

# 日期库
grep -E "dayjs|date-fns" itsm-frontend/package.json  # 应只保留 1 个

# 状态库边界(写入前端规范)
ls itsm-frontend/docs/state-management.md  # 应存在 + 写明边界
```

### P2-2 · 统一指标 + 报表合并（与 backend-itsm 协作）

主导前端层：13 报表页 → 1 套统一指标页 + 下钻视图（详见 backend-itsm P2-2 章节）

## 协作项（我是配角 · 1 项）

### P0-1 协作 · 7 个审批入口改走 BPMN 任务视图

**截止**：随 D-1 工程（T+1.5 周）
**主导**：backend-itsm
**我的动作**：7 个入口组件重写 + 路由 301 重定向

#### Verification

```bash
# 现状
grep -rn "approvals\|approval-chains" itsm-frontend/src/app/ --include="page.tsx" | head

# 重写后
ls itsm-frontend/src/app/admin/approvals/page.tsx        # 走 BPMN
ls itsm-frontend/src/app/admin/approval-chains/page.tsx  # 只读历史
# 其他 6 个入口应被删除或 301 重定向
```

## 本周时间盒（@fe-itsm）

| 时段 | 动作 |
|---|---|
| 本周内 | 配合 backend-itsm 立项 P0-1，列出 7 入口清单 + 重写方案 |
| T+1 周 | 启动 P0-1 前端重写工作 |
| 本季度 | 推进 P2-3~P2-5 依赖收尾（背景工作） |
| 半年视角 | P2-4 巨型组件拆解 + P2-2 统一指标 |

---

# @im-itsm

**看板定位**：连接器 / 通知 / 即时通讯（飞书 / 钉钉 / 企微 / Webhook）

## 我的 P0（止血期 · 1 项）

### P0-5 · NON-GOALS 多渠道对齐

**截止**：T+1 周
**协作**：product（决策权）
**关联决策**：D-2（[决策对话](../../output/decision-memo-2026-09-23.md) 我的倾向 A 改契约允许多）

#### Verification（可执行步骤）

```bash
# 1. 等 product 批 D-2（5 分钟决策）
# 2. 改契约 NON-GOALS 段（product 操作）
#    docs/product/itsm-commercial-capability-contract.md:188
#    "允许多渠道并行；飞书优先生产化闭环，钉钉/企微/email_intake 可选 Experimental；
#     不在 v1.6.x 收官期承诺所有渠道 GA 候选。"

# 3. CHANGELOG 同步
# CHANGELOG.md [Unreleased] 加:口径收敛:多渠道并行,飞书优先

# 4. waiver 销账
# scripts/docs-gate/product-drift-waivers.txt 移除 D-2

# 5. 守卫验证
make docs-gate  # 应 PASS
```

#### 验收
- [ ] 契约 NON-GOALS 段更新
- [ ] CHANGELOG 同步
- [ ] waiver 销账
- [ ] 飞书渠道正式列入 P1 首位（钉钉/企微/email_intake 列入 Experimental）

## 我的 P1（本季度 · 1 项）

### P1-4 · Connector 三渠道真实联调验收

**截止**：Q4 OKR
**目标**：钉钉 / 企微 / 飞书各 1 条端到端 + 验签失败 / 重放攻击拒绝证据

#### Verification（可执行步骤）

```bash
# 1. 钉钉端到端
docker compose -f docker-compose.prod.yml up -d backend dingtalk-mock
curl -X POST http://localhost:8090/api/v1/connectors/dingtalk/inbound \
  -H "X-DingTalk-Signature: <valid>" \
  -d '{"msg": "test"}'
# 应 200 + 触发工单创建

# 2. 钉钉验签失败
curl -X POST http://localhost:8090/api/v1/connectors/dingtalk/inbound \
  -H "X-DingTalk-Signature: <invalid>" \
  -d '{"msg": "test"}'
# 应 401

# 3. 钉钉重放攻击
curl -X POST http://localhost:8090/api/v1/connectors/dingtalk/inbound \
  -H "X-DingTalk-Timestamp: <old>" \
  -d '{"msg": "test"}'
# 应 401

# 4. 同样跑企微、飞书
```

#### 验收
- [ ] 3 渠道端到端各 1 例
- [ ] 验签失败 3 例（每渠道 1 例）
- [ ] 重放攻击 3 例（每渠道 1 例）
- [ ] 失败原因可读（不是 500）

## 我的 P2（半年视角 · 0 项）

无

## 协作项（我是配角 · 0 项）

无

## 本周时间盒（@im-itsm）

| 时段 | 动作 |
|---|---|
| 本周内 | 等 product 批 D-2 → 跟进文档同步 + waiver 销账 |
| T+1 周 | 启动 P1-4 三渠道真实联调（Q4 OKR 第一阶段） |
| 季度内 | 完成 9 例测试用例（3 渠道 × 3 场景） |

---

# @docs-itsm

**看板定位**：文档治理 / CHANGELOG / ADR / README / 治理闭环

## 我的 P0（止血期 · 1 项）

### P0-2 · 变更成熟度口径对齐

**截止**：2026-10-31（waiver 销账）
**协作**：product（决策权）
**关联决策**：D-W1（[决策对话](../../output/decision-memo-2026-09-23.md) 我的倾向 A README → Pilot）

#### Verification（可执行步骤）

```bash
# 1. 等 product 批 D-W1
# 2. 改 README
sed -i 's/变更管理 = 可用/变更管理 = Pilot（发布前重评）/' README.md
# 或手动编辑 README.md:75

# 3. CHANGELOG 同步
# CHANGELOG.md 1.6.10 [Unreleased] 加:口径收敛:变更管理对齐契约 Pilot 口径

# 4. waiver 销账
# scripts/docs-gate/product-drift-waivers.txt 移除 D-W1

# 5. 守卫验证
make docs-gate  # 应 C.6.1 PASS
```

#### 验收
- [ ] `grep "变更管理 = " README.md` 输出"变更管理 = Pilot（发布前重评）"
- [ ] CHANGELOG 段落可见
- [ ] waiver 销账
- [ ] `make docs-gate` C.6.1 PASS

## 我的 P1（本季度 · 2 项）

### P1-1 协作 · 问题 → KE → 知识 → RAG 闭环

**主导**：backend-itsm
**我的动作**：知识发布 / RBAC 可见性 / 知识版本管理

#### Verification
- [ ] 知识草稿 → 审核 → 发布 workflow 完整
- [ ] RAG 检索权限与知识可见性对齐
- [ ] 向量索引与知识版本同步

### P1-6 · PRD §10 Timer Event 用户文档

**截止**：T+48h
**关联**：已在 canonical §6.1 标记

#### Verification

```bash
ls docs/prd/timer-event-design-outline.md  # 应存在
head -5 docs/prd/timer-event-design-outline.md | grep -i "supersede"
# 应有 supersede banner

grep -c "✅\|🟡\|🔴" docs/prd/timer-event-design-outline.md
# 状态列统一
```

#### 验收
- [ ] 文档完成
- [ ] supersede banner 可见
- [ ] 状态列统一（✅/🟡/🔴）
- [ ] 链接入 README

## 我的 P2（半年视角 · 1 项）

### P2-8 · 规划体系收敛到一条

**截止**：v1.7
**目标**：`prd/` / `plans/` / `specs/` / `.specify/` 四选一或明确职责边界并写入 `docs/documentation-governance.md`

#### Verification

```bash
# 现状统计
echo "docs/prd: $(ls docs/prd/*.md | wc -l)"
echo "prd:     $(ls prd/*.md | wc -l)"
echo "plans:   $(ls plans/*.md | wc -l)"
echo "specs:   $(ls specs/*.md 2>/dev/null | wc -l)"
echo ".specify: $(ls .specify/**/*.md 2>/dev/null | wc -l)"
```

#### 验收
- [ ] 4 套明确职责边界（或合并）
- [ ] `documentation-governance.md` 更新
- [ ] `make docs-gate` 仍 PASS

## 协作项（我是配角 · 0 项）

主导 P0-2 + P1-6 + P2-8；其他 owner 的 CHANGELOG/ADR 通过我会签

## 本周时间盒（@docs-itsm）

| 时段 | 动作 |
|---|---|
| 本周内 | 等 product 批 D-W1 → 改 README + CHANGELOG + waiver |
| T+48h | P1-6 PRD §10 文档收口 |
| 本季度 | 配合 P1-1 知识发布流程 |
| 半年视角 | P2-8 规划体系收敛 |

---

# @ai-itsm

**看板定位**：AI 资产 / LLM Gateway / RAG / itsm-ai-service / itsm-agent

## 我的 P0（止血期 · 1 项，全部为协作）

### P0-8 协作 · AI 资产所有权 ADR

**主导**：backend-itsm
**截止**：v1.7 启动前

#### Verification（我的部分）

```bash
# 1. 配合 backend-itsm 起草 ADR
# 贡献 3 段:
#   - itsm-ai-service 编排归属（ai-itsm 视角）
#   - itsm-rag vs 后端 rag_service 二选一理由
#   - itsm-agent 是否进入 v1.7 路线

# 2. 评审
# 评审 backend-itsm 提交的 ADR 草稿，确认 LLM 能力统一走 llm_gateway

# 3. 跟进编排
ls docker-compose.prod.yml | xargs grep -E "itsm-ai-service|itsm-rag"
# 确认编排归属
```

#### 验收
- [ ] ADR 文件落地（backend-itsm 主导）
- [ ] LLM 能力统一走 llm_gateway（评审确认）
- [ ] RAG 归属明确（itsm-rag 或后端 rag_service 二选一）

## 我的 P1（本季度 · 0 项）

无主导 P1（AI 资产收敛在 P0-8）

## 我的 P2（半年视角 · 0 项）

无

## 协作项（全部已合并到 P0-8）

## 本周时间盒（@ai-itsm）

| 时段 | 动作 |
|---|---|
| 本周内 | 配合 backend-itsm 启动 P0-8 ADR 起草（贡献 ai-itsm 视角 3 段） |
| T+2 周 | ADR v1 评审 |
| v1.7 启动前 | ADR 落地 |

---

# @product

**看板定位**：产品决策权 / 对外口径 / 商业契约

## 我的 P0（决策权 · 3 项，全部是 D-前缀决策）

**注**：product 在这 3 项中是**决策权 owner**，技术实施由其他 owner 主导（backend-itsm / docs-itsm / im-itsm）。

### D-1 决策权 · 审批三运行时去留

**截止**：本周内
**主导执行**：backend-itsm（[P0-1](./business-snapshot-2026-09-24.md#p0-1)）
**关联文档**：[decision-dialogue §D-1](./decision-dialogue-2026-09-24.md#d-1--审批三运行时去留最硬骨头)

#### 我的动作（product）

- [ ] 阅读 [decision-dialogue §D-1](./decision-dialogue-2026-09-24.md)（完整论证）
- [ ] 在文档对应段做 diff 标注（批准/拒绝/修订）
- [ ] 或开 PR：`docs(product/decision-dialogue): D-1 决策批准`
- [ ] 拍板后通知 backend-itsm 启动工程

### D-W1 决策权 · 变更成熟度口径

**截止**：2026-10-31
**主导执行**：docs-itsm（[P0-2](./business-snapshot-2026-09-24.md#p0-2)）
**关联文档**：[decision-dialogue §D-W1](./decision-dialogue-2026-09-24.md#d-w1--变更成熟度口径)

#### 我的动作（product）

- [ ] 阅读 decision-dialogue §D-W1
- [ ] 拍板 README → Pilot
- [ ] 通知 docs-itsm 改文档

### D-2 决策权 · NON-GOALS 多渠道

**截止**：T+1 周
**主导执行**：im-itsm（[P0-5](./business-snapshot-2026-09-24.md#p0-5)）
**关联文档**：[decision-dialogue §D-2](./decision-dialogue-2026-09-24.md#d-2--non-goals-多渠道对齐)

#### 我的动作（product）

- [ ] 阅读 decision-dialogue §D-2
- [ ] 拍板改契约允许多渠道、飞书优先
- [ ] 通知 im-itsm 跟进

## 我的 P1（本季度 · 0 项）

无

## 我的 P2（半年视角 · 0 项）

无

## 协作项

- **D-W2**：D-W2（三孤儿包删除）已在 decision-dialogue 中 product 默认批准（无新增决策点）。如需复议，单独处理。

## 本周时间盒（@product）

| 时段 | 动作 |
|---|---|
| T+0~T+24h | 阅读 decision-dialogue，对 4 项决策做 diff 标注 |
| T+24h | 通知 backend-itsm / docs-itsm / im-itsm 启动 |
| 本周内 | 决策落地 + 工程启动 |
| T+1 周 | 跟踪 4 项决策落地状态 |

---

## 跨 Owner 协作矩阵

| 项 | 主导 | 协作 | 关键接口 |
|---|---|---|---|
| **P0-1 审批三运行时** | backend-itsm | product(决策权) + fe-itsm(前端改写) | D-1 拍板 → backend-itsm 立项 → fe-itsm 重写 7 入口 |
| **P0-2 变更成熟度口径** | docs-itsm | product(决策权) | D-W1 拍板 → docs-itsm 改 README/CHANGELOG |
| **P0-5 NON-GOALS 多渠道** | im-itsm | product(决策权) | D-2 拍板 → im-itsm 跟进文档 |
| **P0-8 AI 资产所有权 ADR** | backend-itsm | ai-itsm | 共同起草 → backend-itsm 落地 |
| **P1-1 问题 → KE → 知识** | backend-itsm | docs-itsm(知识发布) | 状态机 → 文档流程 |
| **P2-2 统一指标 + 报表合并** | backend-itsm | fe-itsm(前端层) | 指标定义 → 前端展现 |

---

## 看板使用纪律

1. **owner 列不可空**——无人认领的项必须升级到 P0（默认 24h SLA）
2. **verification 列必须可量化**——本看板每项已给可执行命令/grep，跑一遍就是 PASS/FAIL
3. **commit 列必填**——已闭环的项立即挪到对应章节"已闭环"段
4. **看板每周一更新**——新一周开始时重排优先级
5. **跨 owner 同步**——任意 P0 销账前必须通知协作方，避免接口不对齐

---

## 关联文档

| 文档 | 路径 | 关系 |
|---|---|---|
| 业务快照 | [business-snapshot-2026-09-24.md](./business-snapshot-2026-09-24.md) | §4 优先级看板的"按 owner 拆分版" |
| 决策对话 | [decision-dialogue-2026-09-24.md](./decision-dialogue-2026-09-24.md) | 4 项决策的完整论证（@product 决策权） |
| 全局 P0 看板 | [global-p0-2026-09-16.md](../../output/global-p0-2026-09-16.md) | 35 条历史项的完整列表 |
| 决策备忘（简版） | [decision-memo-2026-09-23.md](../../output/decision-memo-2026-09-23.md) | "我的倾向 + 简略理由" |

---

*快照日期：2026-09-24 · 下次更新：2026-09-30（周一，按 P0 看板节奏）*
*使用方式：每个 owner 只读自己章节；协作方通过"跨 Owner 协作矩阵"对齐接口*