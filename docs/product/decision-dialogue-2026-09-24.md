# 决策对话 — 4 项产品/架构待拍板（2026-09-24 给齐活林）

> **受众**：齐活林
> **目的**：把 [业务快照 §5](./business-snapshot-2026-09-24.md#5-决策清单等待对齐--齐活林) 的 4 项决策展开为**可拍板级**详细论证；每项给出**明确单一选择**（A 或 B 或 C）+ **详细根因分析** + **实施路线** + **验收**
> **紧急度**：D-1 拍板后立即启动 1-2 周工程；D-W1 / D-W2 在 2026-10-31 前必须销账（否则 Gate C.6 反向 FAIL）；D-2 在 1 周内拍板
> **关联**：本文是业务快照 §5 的"详写版"，与 [decision-memo-2026-09-23.md](../../output/decision-memo-2026-09-23.md) 互补——memo 是"我的简略倾向"，本文是"完整论证 + 可执行"

---

## 元信息

| 项 | 值 |
|---|---|
| 决策者 | 齐活林 |
| 文档作者 | 寇豆码 |
| 拍板期望时间 | D-1：本周内 / D-W1 + D-W2：2026-10-31 前 / D-2：T+1 周 |
| 回执方式 | （任选其一）①本文对应决策段做 diff 标注 ②回 decision-memo 修订 ③开 PR `docs(product/decision-dialogue): D-1/D-W1/D-W2/D-2 决策批准` |
| 守卫触发 | D-W1 / D-W2 / O1 已登记 waiver（[product-drift-waivers.txt](../../scripts/docs-gate/product-drift-waivers.txt)），逾期 Gate C.6 反向 FAIL |

---

## 决策全景（紧急度排序）

| # | 决策项 | 我的倾向 | 截止 | 触发条件 |
|---|---|---|---|---|
| **D-1** | 审批三运行时去留 | **A. 收敛到 BPMN** | 拍板后立即启动 | CONSTRAINTS#3 永久失效 |
| **D-W1** | 变更成熟度口径 | **A. README → Pilot** | 2026-10-31 | C.6.1 反向 FAIL |
| **D-W2** | 三个孤儿包去留 | **A. 三个都删** | 2026-10-31 | C.6.3 反向 FAIL |
| **D-2** | NON-GOALS 多渠道 | **A. 改契约允许多** | T+1 周 | 诚实优先 |

**最急迫**：D-1（决策一旦做，1-2 周工程排期开始；影响 8 个前端入口 + 6 张实体表）
**次急迫**：D-W1 + D-W2（10-31 waiver 截止，逾期反向 FAIL）
**最轻松**：D-2（1 段话修改）

---

# D-1 · 审批三运行时去留（最硬骨头）

## 1. 现状（代码事实）

**三套运行时并存，直接违反能力契约 `CONSTRAINTS#3`（BPMN 是唯一流程编排层）**：

| 运行时 | 装配位置 | 实体表 | 前端入口数 |
|---|---|---|---|
| ① BPMN 用户任务 | `router/router.go:58` 等 | `bpmn_approval_tasks` `bpmn_approval_history` | 1（`workflow/ticket-approval`） |
| ② `service.ApprovalService`（兼容遗留） | `internal/bootstrap/app.go:533, :562` | `approval_records` `approval_workflow` `servicerequestapproval` `ticket_approval` | 6（`approvals` `approvals/pending` `admin/approvals` `settings/approvals` `ai/approval` `service-catalog/approvals`） |
| ③ `service.ApprovalChainService`（链式） | `internal/bootstrap/app.go:864` | `approvalchain` `process_approval_decision` | 1（`admin/approval-chains`） |

**关键证据**：
- `grep -n "NewApprovalService\|NewApprovalChainService" itsm-backend/internal/bootstrap/app.go` 应输出 2 行
- `ls itsm-backend/ent/schema/ | grep -i approval` 应输出 6 张表
- **CHANGELOG.md:47**（1.6.10 / 2026-09-20，`### Changed`）**自打脸**："BREAKING: 审批架构迁移 — 统一消费 BPMN 任务，消除双审批实例，ApprovalRecord 标记废弃"
- **同一版本** `CHANGELOG.md:33`（`### Added`）："审批链运行时能力 — 新增加签 / 委派 / 拒绝策略配置 / 动态层级适配"

**即**：CHANGELOG 一边宣布"消除双审批"，一边给第二条审批链加功能。这是漂移审计 D1 的根因。

**影响**：
- 8 个前端入口（用户路径分裂）
- 6 张实体表（数据存储分裂，跨域报表 / 审计无法合并）
- 对"可审计"核心卖点是**实质性风险** —— 同一张变更单的审批历史可能分裂在 3 张表

## 2. 选项矩阵

| 选项 | 改动量 | 工作量 | 风险 | 长期收益 | 推荐 |
|---|---|---|---|---|---|
| **A 收敛到 BPMN** | 大（实体去 4，前端改 7） | **1-2 周** | 中（数据迁移 + CHANGELOG 撤回） | **★★★★★** | **✅** |
| **B 收敛到 ApprovalChain** | 中（实体去 3，前端改 6） | 1 周 | 中（BPMN 流程全重做） | ★★★ | — |
| **C 仅改 CHANGELOG** | 极小（改字 + 加注释） | 半天 | **高**（下次审计再抓） | ★ | — |

## 3. 选项 A 详细论证（我的倾向）

### 改动清单

1. 删除 `service/approval_service.go` 的非 BPMN 路径实现（保留审批业务逻辑迁移到 BPMN 任务）
2. 删除 `service/approval_chain_service.go` 全部实现
3. 删除 6 张实体表中的非 BPMN 4 张：`approval_records` `approval_workflow` `approvalchain` `process_approval_decision` `servicerequestapproval`（保留 `bpmn_approval_tasks` + `bpmn_approval_history`）
4. 前端 7 入口改走 BPMN 任务视图（保留 `admin/approval-chains` 作历史只读）
5. CHANGELOG 1.6.10 加撤回段："上版本描述不准确，详见 ADR / issue"

### 工作量拆解

| 部分 | 内容 | 周期 |
|---|---|---|
| Backend | 实体删除 + service 删除 + 路由引用迁移 | ≈ 1 周 |
| Frontend | 7 入口组件重写为 BPMN 任务视图 + 路由 301 重定向 | ≈ 1 周 |
| 文档 | CHANGELOG 撤回段 + ADR + README 同步 | ≈ 0.5 天 |

**总周期**：1.5-2 周

### 风险评估

| 风险 | 概率 | 影响 | 缓解 |
|---|---|---|---|
| 数据迁移：历史审批记录的表结构变化 | 高 | 中 | 兼容层读旧表（只读）+ 双写缓冲期 |
| 前端 UX 改变：7 入口改走 BPMN 视图 | 中 | 中 | 灰度开关（A/B）+ 旧路径 301 重定向保留 1 个季度 |
| CHANGELOG 撤回：市场已读"已消除"的对外表述要更正 | 低 | 低 | CHANGELOG + README 同步修正 + 公众号"更正声明" |

### 长期收益

- **根本解决** CONSTRAINTS#3 失效 → 能力契约可被机器校验
- 实体表 6 → 2（**-67%**）
- 前端入口 8 → 2（**-75%**）
- 长期维护成本 **-50%**
- **解锁 D-W1**：D-1 收尾后变更成熟度可同步升"可用"（因为变更审批闭环已统一）

## 4. 根因分析：为什么是 A 而不是 B / C

### 为什么不选 B（收敛到 ApprovalChain）

- **BPMN 表达力是链式的真超集**。BPMN 支持并行网关、包容网关、条件分支（本项目已有 `bpmn_process_executor.go:474,477` 并行 / 包容实现）。
- 如果保留 ApprovalChain 删 BPMN，等于**降级流程能力**——链式只能做"逐级审批"，BPMN 还能做并行、包容、条件分支。
- 已有 BPMN 引擎成熟度（22+5 例 E2E 已入库），再把 BPMN 流程全重做 = 1 周工程 + 流程表达力下降 = 净亏。

### 为什么不选 C（仅改 CHANGELOG）

- C **不是决策，是逃避决策**。CHANGELOG 已经写了"已消除"，代码还在跑——下一次审计还会抓出来。
- 这是"用文档谎言掩盖代码事实"。9-22 漂移审计已经因为这个打了 D1 红标。
- 工作量虽小，但**守卫会反复报**，团队会反复解释，长期成本远高于 A。

### 为什么不拖延不决

- 8 月评审（`architecture-understanding.md:45`）已指出 P0-B"非 BPMN 路径是权宜"，本应 9-30 前收敛——**已经延期到 10-31 waiver 才登记**。
- 延期不是节省工程量，是**把决策成本转移给未来的自己**——届时守卫反向 FAIL 是被动应对，不是主动拍板。
- 加上 D-W1 是顺序约束（契约升"可用"必须先过 D-1 审批收敛），不拍 D-1 就不能动 D-W1。

## 5. 实施路线（若选 A）

| 阶段 | 内容 | 周期 | 验证 |
|---|---|---|---|
| **1. 立项** | 提交 `refactor(approval): 收敛到 BPMN` issue，绑定 D-1 + D-W1 顺序约束 | T+1d | issue 链接到 CHANGELOG + ADR |
| **2. 标记 deprecated** | 以 `legacy=true` 标记非 BPMN 路径；frontend 7 入口标 deprecated | T+2d | `grep legacy=true internal/` 命中 |
| **3. 灰度** | 每个入口加 A/B 开关（env flag），跑一周生产 | T+7d | prod 探针矩阵 0 异常 |
| **4. 删除** | 删非 BPMN 路径 + 删 4 张实体 + 删 7 个 deprecated 入口 | T+3d | `go build ./...` + frontend build 通过 |
| **5. CHANGELOG 撤回** | 1.6.10 加撤回段，README 同步 | T+1d | CHANGELOG diff 可见 |
| **6. D-W1 同步** | D-1 收尾后，D-W1 升契约"可用" | T+1d | C.6.1 waiver 自然消亡 |

### 验收（最终态）

- 装配处（`app.go:533/562/864`）只剩 BPMN 一条
- `grep -c "NewApprovalService\|NewApprovalChainService"` = **1**（只命中 BPMN 装配点）
- `ls ent/schema | grep -i approval` = **2**（`bpmn_approval_tasks` + `bpmn_approval_history`）
- 前端 8 入口 → 2 入口（`admin/approvals` 走 BPMN + `admin/approval-chains` 只读历史）

---

# D-W1 · 变更成熟度口径（README "可用" vs 契约 "Pilot"）

## 1. 现状（代码事实）

- `README.md:75` 写"变更管理 = 可用（可进入生产验收）"
- `docs/product/itsm-commercial-capability-contract.md:23` 写"变更 = Pilot，发布前需重新评估 GA 候选"

**两者不可同真**：
- 若 README "可用"为真 → 契约"GA 候选"必须升 → 但缺真实生产变更样本证据
- 若契约 "Pilot" 为真 → README 公开承诺"可用"是过度承诺 → 市场 / 客户预期落差

## 2. 选项矩阵

| 选项 | 改动 | 工作量 | 风险 | 收益 | 推荐 |
|---|---|---|---|---|---|
| **A 改 README 为"Pilot"** | 改一行 | **5 分钟** | **低**（诚实优先） | **★★★** | **✅** |
| **B 改契约为"可用"** | 改一段 + 4 处引用 | 中 | **高**（缺证据，且依赖 D-1） | ★★ | — |
| **C 两边都改"GA 候选 + 已签发变更单 X 张证据"** | 改两边 + 补证据 | 大（需生产样本） | 中 | ★★★★★（但本周做不到） | 留给 v1.7 |

## 3. 选项 A 详细论证（我的倾向）

### 改动

- `README.md:75` 一行：`变更管理` 行的"可用"改为"Pilot（发布前重评）"
- 同步 CHANGELOG（若 1.6.10 提及）加一行"口径收敛说明"

### 工作量

5 分钟。

### 风险与缓解

| 风险 | 概率 | 缓解 |
|---|---|---|
| 公开承诺降级：市场 / 客户已读"可用"会有预期落差 | 低 | README 改的同时，在公众号 / 客户邮件加"口径同步声明"（短文案，1 段话） |

### 收益

- 诚实优先 —— 契约和 README 一致
- 解锁 C.6.1 waiver 销账 → 守卫不再反向 FAIL
- 为 v1.7 升"GA 候选"留出真实生产样本的时间窗

## 4. 根因分析

### 为什么不选 B（改契约升"可用"）

- 契约升"可用"必须有**真实生产变更样本**作为证据。当前没有。
- 升"可用"必须**先过 D-1 审批收敛**（顺序约束）—— 因为变更审批闭环依赖 BPMN 唯一编排，而 BPMN 唯一编排正是 D-1 要做的事。
- 即：**B 不可独立拍板，必须在 D-1 收尾后才能做**。

### 为什么不选 C（两边都改 GA 候选）

- C 是终极目标，但**需要真实生产样本**（已签发变更单 + 客户验收）。当前没有。
- C 是 v1.7 视角的事，不在 v1.6.x 收官期。
- 当前最务实的做法是 A，然后在 v1.7 有样本后再升 C。

## 5. 实施路线（若选 A）

**周期**：5 分钟

| 步骤 | 内容 | 验证 |
|---|---|---|
| 1. README 改 | `README.md:75` 改"Pilot（发布前重评）" | diff 可见 |
| 2. CHANGELOG 加 | 1.6.10 `[Unreleased]` 加"口径收敛：变更管理对齐契约 Pilot 口径" | CHANGELOG 段落可见 |
| 3. C.6.1 waiver 销账 | `product-drift-waivers.txt` 移除 D-W1 | waiver 文件 diff |
| 4. 守卫验证 | `make docs-gate` 通过 | CI 全绿 |

---

# D-W2 · 三个孤儿包去留（department / root_cause / dashboard）

## 1. 现状（代码事实）

经全仓 grep 确认（`grep -rln "handlers/department\|handlers/root_cause\|handlers/dashboard" itsm-backend --include='*.go' | head`）：

| 孤儿包 | 文件大小 | 路由引用 | 替代实现 |
|---|---|---|---|
| `handlers/department/handler.go` | 5.5 KB | **0** | `config.CommonHandler` 提供 `/departments` 端点 |
| `handlers/root_cause/handler.go` | 3.9 KB | **0** | 自 `controller/` 迁移后从未接线 |
| `handlers/dashboard/`（handler.go + service.go） | 多文件 | **0** | `handlers` 根包的 `dashboard_handler.go` 提供 `/dashboard` 路由 |

**额外背景**：8 月评审 R2 是"incident 域新层零路由（编译占位）"——今天 incident 已接线 ✅，但 **department / team / root_cause / dashboard 变成了同样的"编译占位"**。同一缺陷换了对象复发。

## 2. 选项矩阵

| 选项 | 改动 | 工作量 | 风险 | 收益 | 推荐 |
|---|---|---|---|---|---|
| **A 三个都删** | `git rm` + CHANGELOG | **半天** | **低**（已确认替代） | **★★★** | **✅** |
| **B 三个都接** | 加 route + service + ent 实体 | 1-2 周 | 高（未驱动业务） | ★（强加负担） | — |
| **C 选一接选二删** | 混合方案 | 1 周 | 中 | ★★ | — |

## 3. 选项 A 详细论证（我的倾向）

### 改动清单

1. **前置验证**：`git grep -l "handlers/department\|handlers/root_cause\|handlers/dashboard"` 在 frontend / docs / tests 找调用——确认 0 调用
2. 删除 3 个孤儿包：`git rm -r itsm-backend/handlers/{department,root_cause,dashboard}`
3. 同步 `docs/architecture/domain-ownership.md`（如有提及）
4. CHANGELOG `[Unreleased]` 加"清理：删除 3 个零路由领域包"

### 工作量

半天（含前置 grep + 部署验证 + 删 + 文档同步）

### 风险与缓解

| 风险 | 概率 | 缓解 |
|---|---|---|
| 若有 frontend 仍在调 → 404 | 低 | **前置 grep 验证**（动手前必做） |
| 删除影响已知用例 | 低 | prod 部署验证 5 域核心场景（incident/change/problem/CMDB） |
| 概念页面引用 | 低 | frontend E2E：登录 → 仪表盘 → 工单详情 → 改状态 |

### 收益

- 解除 C.6.3 waiver，守卫 PASS
- 减少编译时间 + 认知成本
- 防止 8 月评审 R2 的"incident 域零路由"缺陷**换对象复发**

## 4. 根因分析

### 为什么不选 B（三个都接）

- 接入需要**新业务需求驱动**。当前 v1.6.x 收官期**不应开新口子**。
- 每个孤儿都有"被另一实现替代"的事实（CommonHandler / handlers/ticket / handlers/dashboard_handler）—— 再接入是**重复建设**。
- 接入需要新增 ent 实体 + service，工程量 1-2 周，但**当前无业务诉求**。

### 为什么不选 C（混合）

- 决策复杂度增加，且没有"必须接"的理由。
- 三个都是孤儿，处置标准一致更清晰。

## 5. 实施路线（若选 A）

**周期**：半天

| 步骤 | 内容 | 验证 |
|---|---|---|
| 1. 前置 grep | `git grep -l "handlers/department\|handlers/root_cause\|handlers/dashboard"` frontend/docs/tests | 输出为空 |
| 2. prod 部署验证 | 用 `prod_*.yml` 部署环境跑核心 6 域 | 5 域全 200 |
| 3. frontend E2E | preview 跑一次：登录 → 仪表盘 → 工单详情 → 改状态 | E2E 通过 |
| 4. 删除 | `git rm -r` 3 个目录 | `go build ./...` 通过 |
| 5. CHANGELOG + 文档 | `[Unreleased]` 同步 + `domain-ownership.md` 移除 | diff 可见 |
| 6. C.6.3 waiver 销账 | `product-drift-waivers.txt` 移除 D-W2 | waiver 文件 diff |

**先例**：`handlers/team/` 同类已删（2026-09-22 commit），有先例可参考。

---

# D-2 · NON-GOALS 多渠道对齐（飞书 / 钉钉 / 企微 / email_intake）

## 1. 现状（代码事实）

`docs/product/itsm-commercial-capability-contract.md:186-193` NON-GOALS 段："**不在当前阶段同时生产化飞书、企微、钉钉和所有云厂商**"。

实际：
- 后端 `handlers/` 下 4 个渠道域并存：`feishu` / `dingtalk` / `wecom` / `email_intake`
- `ROADMAP.md:99` 标记"钉钉 / 企微入站回调、持久化去重、健康度与凭据轮换"为**已落地**
- `CHANGELOG.md:35`（1.6.10）"钉钉 / 企微入站回调 — 入站消息回调 API 及幂等去重"
- 契约 P1 排序是"飞书生产连接器，随后再扩企微和钉钉"——实际执行顺序是**钉钉+企微先行、飞书仍未生产闭环**

**冲突**：
- 契约：不同时生产化 + 飞书优先
- 实际：同时生产化 + 钉钉 / 企微优先

## 2. 选项矩阵

| 选项 | 改动 | 工作量 | 风险 | 收益 | 推荐 |
|---|---|---|---|---|---|
| **A 改契约允许多渠道（飞书优先）** | 改契约 + 加优先级表 | **1 段话** | **无** | **★★★** | **✅** |
| **B 下线 email_intake 回到契约口径** | 删 email_intake 域 | 高 | **高**（影响已用客户） | ★ | — |
| **C 维持现状不动** | 仅记录 | 0 | 高（守卫再报） | ★ | — |

## 3. 选项 A 详细论证（我的倾向）

### 改动

- `docs/product/itsm-commercial-capability-contract.md` 第 188 行 NON-GOALS 段，改为：

  ```
  - 允许多渠道并行；飞书优先生产化闭环，钉钉 / 企微 / email_intake 可选 Experimental；
    不在 v1.6.x 收官期承诺所有渠道 GA 候选。
  ```

### 工作量

5 分钟。

### 风险

**无**——诚实优先，与代码事实一致。

### 收益

- 契约与代码一致 → 守卫不再报 D3
- 解锁 D-3 漂移
- 公开承诺清晰：飞书是第一个生产闭环，其余可选 Experimental

## 4. 根因分析

### 为什么不选 B（下线 email_intake）

- email_intake 已有客户用，**砍渠道是产品倒退**。
- 改契约比改代码代价低 100 倍。

### 为什么不选 C（维持现状）

- 让守卫不再报 = 加 waiver。但 waiver 是债务登记，不是解决。10-31 同样会反向 FAIL。

## 5. 实施路线（若选 A）

**周期**：5 分钟

| 步骤 | 内容 | 验证 |
|---|---|---|
| 1. 改契约 NON-GOALS 段 | 1 段话修改 | diff 可见 |
| 2. CHANGELOG | `[Unreleased]` 加"口径收敛：多渠道并行，飞书优先" | 段落可见 |
| 3. waiver 销账 | 移除 D-2 waiver 条目 | waiver 文件 diff |

---

## 拍板汇总（给齐活林的"看一眼就批"清单）

| # | 决策项 | 我的倾向 | 改动量 | 截止 | 期望批 |
|---|---|---|---|---|---|
| **D-1** | 审批三运行时去留 | **A. 收敛到 BPMN** | 大（1-2 周） | 拍板后立即 | ✅ / ❌ |
| **D-W1** | 变更成熟度口径 | **A. README → Pilot** | 1 行 + CHANGELOG | 2026-10-31 | ✅ / ❌ |
| **D-W2** | 三个孤儿包 | **A. 三个都删** | git rm + CHANGELOG | 2026-10-31 | ✅ / ❌ |
| **D-2** | NON-GOALS 多渠道 | **A. 改契约允许多** | 1 段话 | T+1 周 | ✅ / ❌ |

**期望回应方式**（任选其一）：
- 在本文档对应决策段做 diff 标注（类似 code review）
- 直接回 [decision-memo-2026-09-23.md](../../output/decision-memo-2026-09-23.md) 修订
- 或开 GitHub PR：`docs(product/decision-dialogue): D-1/D-W1/D-W2/D-2 决策批准`，把批准的选项作为 PR 主体

---

## 决策流程建议（给齐活林的时间线）

| 时段 | 动作 |
|---|---|
| **本周内**（2026-09-24 至 2026-09-30） | 优先批 **D-1**（决定工程排期起点）；同步批 **D-W2**（半天工作，无依赖，可立即销账 waiver） |
| **T+1 周**（2026-09-30 至 2026-10-07） | 批 **D-2**（5 分钟，无依赖）；D-1 启动工程期 |
| **T+2 周**（2026-10-08 至 2026-10-15） | D-1 工程进入"灰度"阶段 |
| **2026-10-31 前** | D-W1 拍板（若 D-1 已收尾，可同步升契约"可用"）；所有 P0 waiver 销账；若仍有 waiver 未销账 → Gate C.6 反向 FAIL，触发构建失败 |

---

## 关联文档

| 文档 | 路径 | 关系 |
|---|---|---|
| 业务快照 | [business-snapshot-2026-09-24.md](./business-snapshot-2026-09-24.md) | §5 决策清单的"详写来源" |
| 决策备忘（简版） | [decision-memo-2026-09-23.md](../../output/decision-memo-2026-09-23.md) | "我的倾向 + 简略理由" |
| 漂移审计 | [product-drift-overdesign-audit-2026-09-22.md](../../output/product-drift-overdesign-audit-2026-09-22.md) | D-1 / D-W1 / D-W2 / D-2 的代码事实来源 |
| 商业契约 | [itsm-commercial-capability-contract.md](./itsm-commercial-capability-contract.md) | CONSTRAINTS#3 + NON-GOALS 修改对象 |
| 产品口径守卫 | [check-product-drift.sh](../../scripts/docs-gate/check-product-drift.sh) | 10-31 反向 FAIL 触发机制 |
| Waiver 登记 | [product-drift-waivers.txt](../../scripts/docs-gate/product-drift-waivers.txt) | 5 项 waiver 当前状态 |

---

*对话发起：寇豆码 · 日期：2026-09-24 · 期望批：齐活林 · 期望执行：backend-itsm + devops-itsm + docs-itsm*