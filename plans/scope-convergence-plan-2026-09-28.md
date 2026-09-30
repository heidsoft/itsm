# ITSM 收敛研发计划（2026-09-28）

> 目标：**不写新功能，只做三件事** —— 把 CI 从红变绿、把 9 个"预览"做成真的能用、把数据模型的硬伤补上。
> （原目标中的"把菜单里的空壳清掉"已撤销 —— 复核后真实空壳为 0，见 §1.5。）
> 输入：[output/product-scope-convergence-2026-09-28.md](./../output/product-scope-convergence-2026-09-28.md)（盘点与 STOP/FIX/KEEP）
> 执行纪律：分批次推进，**每批跑测试 + `git diff` 核对后再提交**；失败先用 `git stash` 判定是否自引入。

---

## 0. 仓库现状快照（2026-09-28 08:57 实测）

> ✅ **B0 已于 2026-09-28 执行完成并提交 `a4bc30bf`**（工作区干净，未提交 0 项）。
> 以下快照保留为执行前状态，用于回溯；当前 C.6 / C.7 均已 0 FAIL。

| 项 | 状态（执行前 → 执行后） |
|:---|:---|
| 分支 | `main`，与远端 **0/0**（无落后、无未推送） |
| 最近提交 | `662294e0` → **`a4bc30bf`**（B0：C.7 守卫 + C.6.4 基线上调 + 删 `.disabled`） |
| 未提交 | 7 项 → **0 项** |
| Gate C.7 收敛守卫 | ✅ **0 FAIL / 0 WAIVED**（真实空壳 0，非豁免；详见 §1.5） |
| Gate C.6 口径漂移 | 🔴 5 FAIL（全 C.6.4）→ ✅ **0 FAIL**（基线显式上调至实际值） |
| 后端构建 | ✅ `go build ./...` 通过（B0 执行时验证） |

---

## 1. ⚠️ 关键修正：C.6.4 归因 —— 推翻上一轮 D1 建议

上一轮盘点时我不知道"涨在哪"，凭原则建议 **D1 = 不上调基线、先缩表面**。
本轮用 `git log --diff-filter=A --since=2026-09-21` 归因后，**结论要改**：

| 指标 | 增长 | 归因（全部指向契约主链路） |
|:---|:---|:---|
| `handlers_dirs` 64→65 | +1 | `handlers/workbench/`（f0a87cea）—— **统一工作台，原 P0 根因"无跨域待办"的正解** |
| `frontend_pages` 168→169 | +1 | `(main)/workbench/page.tsx`（3f38f93a）—— 同上，前端 MVP |
| `service_go_files` 329→333 | +4 | `lifecycle/phase.go`、`approval_chain_resolver_adapter.go`、`approval_chain_shadow.go`、`bpmn/approval_chain_handler.go` —— **审批向 BPMN 收敛 + 生命周期映射层** |
| `ent_schema_files` 132→133 | +1 | `ent/schema/sla_state.go`（62867abb）—— **SLA 统一引擎 / Phase 3 读切换的产物** |
| `bootstrap_app_lines` 1799→1811 | +12 | 租户开通 outbox 接线 + 上述装配（414ba5fd） |

**五项里有四项是上一轮 PM 评审明确要求的收敛动作本身**（工作台、审批收口、SLA 统一）。
把它们算作"扩散"并逼着砍，是错误的激励 —— 会让人不敢做正确的收敛工作。

### 修正后的决策

- **D1 改为：显式上调基线 + 逐条说明理由**（B0 执行）。这些增长都能指向契约主链路，属于"合法的基线变更"。
- **表面收缩不再靠"砍空壳"**（空壳结论作废，见 §1.5），改由 B2–B4 收敛过程中的重复代码消除来贡献。
- 🔴 **保留一条机制要求**：棘轮只校验 `current > baseline`，所以**任何下降后都要同步收紧基线到实际值**，
  否则会留下免费空间、之后可无声明涨回。

> 教训：盘点评级前先做归因。这次靠 `git log --diff-filter=A` 才避免了一个错误决策。

---

## 1.5 ⚠️ 第二处修正：空壳结论作废，B1 整批撤销

盘点报告 `output/product-scope-convergence-2026-09-28.md` 曾给出「10 个前端空壳模块 / 15 页」，
并据此排了 B1。逐文件复核后确认**全部是误判，真实空壳 = 0**：

| 误判类型 | 涉及 | 实际 |
|:---|:---|:---|
| 组件路径未补 `.tsx` | `licenses`(4页) | `LicenseList.tsx` 真调 `AssetApi` |
| 漏 `lib/services/*-service.ts` 封装层 | `applications`、`tags` | 真调 `/api/v1/applications`、`/api/v1/tags` |
| 未排除兼容重定向页 | enterprise/settings/sla-dashboard/system/teams/templates/workflows | 页面只有 `redirect('/admin/...')`，是刻意保留的旧链接兼容路由 |

**B1（空壳收敛）撤销**。C.7.1 规则已修（补扩展名、扩 API 信号、排除重定向页），
当前 **0 空壳 / 0 豁免**；白名单清空，三类教训写进文件头。

> 差点基于脚本误报下线 15 个页面，其中大部分是对的、被需要的实现。
> **静态检测的下游结论，在下判断前必须逐文件复核。**

替代动作（可选，低优先级，不占本季度主线）：把 7 个兼容重定向模块（9 页）在文档里登记为
"兼容路由"，评估是否可收敛到统一的重定向表 —— **收益很小，可不做**。

---

---

## 1.6 B2 执行中发现的四件事（均已处理，留档以免重踩）

**① `edge.To` 单边定义会被 ent 当 O2M，不是 M2M。** 首轮生成出的是 `O2M` + 外键列
`configuration_items.change_affected_cis` —— 等于"一个 CI 只能属于一个变更"，语义错误且会污染 CMDB
主表。**必须**在 `ConfigurationItem` 上加反向边 `edge.From("changes", Change.Type).Ref("affected_cis")`，
才是 M2M 关联表 `change_affected_cis`（先例：CI↔incidents 就是这样成对定义的）。
代价：**多跑一轮 entc（约 19 分钟）**。

**② 新建关联表必须登记 tenant_guard，否则生产起不来。** M2M 关联表无 `tenant_id`，
`ApplyGuard` 在 `policy=fatal` 下遇到未豁免缺列的表会**拒绝启动**。已在
`internal/schema/tenant_guard.go` 登记 `change_affected_cis`（Scope=derived, Owner=change）。

**③ 生成方法名是 `AddAffectedCiIDs`（单数 Ci），不是 `AddAffectedCisIDs`。** 取实体是
`AddAffectedCis(v ...*ConfigurationItem)`。命名不一致，编译期才发现。

**④ 同一列存在两套互斥语义（重要，影响后续所有域的收敛判断）**：

| 运行时 | affected_cis 语义 | 校验方式 | 生产是否接线 |
|:---|:---|:---|:---|
| `handlers/change`（现行） | **数字 CI ID** 字符串 | `strconv.Atoi` → `ConfigurationItem.IDIn` | ✅ 是 |
| `service/change_service.go`（legacy） | **CI 名称** | `configurationitem.NameIn` | ❌ 否，仅测试引用 |

legacy 已被移植为「名称→ID 解析」，对外行为不变。**`related_tickets` 同理**：现行侧是工单号字符串、
从不解析为实体，转成 edge 需要产品决策（ID vs 编号），且没有任何代码按实体查询它 —— 因此 B2 不动它，
另立决策项（见 §8）。

> 推论：这类"同一份数据两套写法"很可能不止变更域一处，后续做域收敛时**先查是否存在第二运行时**。

## 1.7 🔴 新发现：`service` 包测试在 HEAD 上编译不过（自引入判定之外）

> ✅ **F8 已于 2026-09-30 修复**（提交 `09cabc3c`）。以下为原始记录，修复后的实测结论见 §1.8。

验证 B2 时顺带发现（**与本次改动无关，HEAD 既有**）：

| 文件 | 问题 |
|:---|:---|
| `service/change_service_test.go` | 用了 `change.Status(...)` 但**未导入** `itsm-backend/ent/change`（应是 `662294e0` 枚举化时漏改） |
| `service/sla_monitor_service_test.go` | `createViolation` 调用少一个参数（缺 `int`） |
| `service/sla_violation_alert_tx_test.go` | 同上，5 处 |

→ `go test ./service/` 目前**整包编译失败**，等于这个包的所有测试（含 SLA、变更、知识）**都在空转**。
这比"某条用例失败"严重得多：**守卫存在但没在跑**。已列为 **F8，建议提到 B3 之前**。

## 1.8 F8 修复后的实测（2026-09-30）

修复 F8 的同时发现 `backend-ci` 的 Lint job 本身长期红：`Check formatting`（gofumpt）28 个文件不合规，
而 Build/Test 均 `needs: lint`，**因此远端 main 自 `662294e0`（09-27）起从未跑过后端测试**。
提交 `dba76f92`（纯格式化）+ `09cabc3c`（语义修复）后 Lint 三项归零：`gofumpt -l`=0、staticcheck=0、`go build ./...` 通过。

**过程中的一次误判值得留档**：staticcheck 在编译失败状态下报出 10 处 U1000「未使用代码」，
其中 9 处（`parseWorkflowNodes`/`resolveApprover`/`getEscalationNotifyUsers`/`addTagsToTicket`/`withNowFunc` 等）
实际只被测试引用——**编译失败导致测试文件未被加载，引用没被计入**。若按原清单删除，会连带删掉
审批链、SLA 时区、升级通知、标签的测试覆盖，与收敛目标相反。F8 修好后真实未使用仅剩 `toTicketResponse` 1 处。
> 推论：**门禁的报错清单本身也依赖守卫在跑**；先恢复执行能力，再解读失败。

`go test ./service/` 首次真实执行暴露的存量问题（**未修，属 B3/B4 范畴**）：

| 现象 | 位置 | 判断 |
|:---|:---|:---|
| panic：`invalid enum value for status field: "submitted"` | `service/dashboard_domain_metrics_test.go:39` | 枚举化（`662294e0`）收紧值域后，测试夹具仍用非枚举值建 Change；panic 会**中断整包后续用例**，优先级最高 |
| 5 个 `TestChangeService_*` 失败 + `TestDashboardDomainMetricsAreReal` 失败 | `service/change_service_test.go` | legacy 变更 service 的存量断言/错误，与 §1.6「同一列两套互斥语义」和 §8「第二运行时去留」直接相关，应在 B4 或 §8 决策时一并处理 |

`handlers/standard_change` 的断言已在 `09cabc3c` 内改到 B2 后的真实语义（名称型条目不构成 CI 关系；覆盖用例改为创建真实 CI 并断言边解析）。


## 2. 收敛三原则（PR 评审照此执行）

1. **预览不转可用，就不开新能力面。** 新增能力域前，先把一个"预览"转成"可用"。
2. **菜单上有的，必须能跑通。** 有页面无后端 = 空壳；接后端或从菜单下线，没有第三选项。
   - 例外（C.7.1 已内置，勿再误判）：`redirect()` 兼容路由、`lib/services/*` 封装层调用、`@/components/*` 穿透调用，都算"已接线"。
3. **棘轮只允许下降，且下降后必须收紧基线。** 要升必须显式改基线 + 写明契约主链路 + 回答"谁来维护"。

---

## 3. 研发批次

### B0 · 收口当前工作区（0.5 天，建议立即做）

**为什么先做**：当前 `make product-drift` 是红的，任何 CI 都会挂；带着红灯做后面的批次会分不清新旧失败。

| 任务 | 说明 |
|:---|:---|
| B0-1 | 提交 Gate C.7 三件套：`check-scope-creep.sh`、`product-scope-baseline.txt`、`scope-shell-whitelist.txt` |
| B0-2 | 提交接线改动：`run-all.sh`、`Makefile`（`scope-creep` 目标 + PHONY）、`CHANGELOG.md` |
| B0-3 | 提交遗留删除 `itsm-backend/dto/workflow_dto.go.disabled`（上轮复审项，已删未提交） |
| B0-4 | **上调 C.6.4 基线 5 项**至实际值，每项行尾注明主链路与批准理由（见下表） |

基线变更明细（写入 `scripts/docs-gate/product-surface-baseline.txt`）：

```
handlers_dirs=65           # 2026-09-28 +1 workbench（统一工作台，P0 根因正解 f0a87cea）
service_go_files=333       # 2026-09-28 +4：lifecycle/phase.go + approval_chain 收敛三件套
bootstrap_app_lines=1811   # 2026-09-28 +12：租户开通 outbox 接线 + 上述装配
ent_schema_files=133       # 2026-09-28 +1 sla_state.go（SLA 统一引擎 62867abb）
frontend_pages=169         # 2026-09-28 +1 workbench 页面（无空壳可清，见 §1.5；未来若有下降须同步收紧）
```

**验收**
```bash
make verify-scripts && make scope-creep && make product-drift   # 三条全绿
cd itsm-backend && go build ./...                               # 后端构建通过
```

**提交纪律**（用户既定规范，不得跳过）
```bash
git add scripts/docs-gate/check-scope-creep.sh scripts/docs-gate/product-scope-baseline.txt \
        scripts/docs-gate/scope-shell-whitelist.txt scripts/docs-gate/run-all.sh \
        Makefile CHANGELOG.md itsm-backend/dto/workflow_dto.go.disabled   # 显式挑选，禁 git add -A
git diff --cached | grep -nE "AdminProd|RlsAdmin|password|secret|token"    # 敏感扫描，必须为空
git commit -F - <<'EOF'   # 多行用 -F，commitlint 不得 --no-verify
chore(gate): 新增 Gate C.7 收敛守卫并显式上调 C.6.4 基线

C.7 把"继续扩散"变成构建失败：空壳前端模块零容忍 / 预览域棘轮 / 规划能力面冻结。
C.6.4 五项增长经 git 归因全部指向契约主链路（workbench、审批收敛、SLA 统一引擎），
按基线文件规定的流程显式上调并逐条说明理由。
EOF
```

---

### ~~B1 · 空壳收敛~~ —— 撤销

> 复核后真实空壳 = 0，本批次撤销。详见 §1.5。
> 可选替代：整理 7 个兼容重定向模块（低优先级，收益小，可不做）。

### B2 · 变更域数据模型：JSON → Ent edge ✅ **已完成 2026-09-28（提交 `7e3344b0`）**

- `ent/schema/change.go:87/90` 的 `affected_cis` 改为多对多 edge（关联表 `change_affected_cis`）；
  `related_tickets` **未动**（见 §1.6 决策）。
- 配套：存量回填 SQL、tenant_guard 豁免登记、legacy 运行时「名称→ID」解析移植。
- **验收**：`go build ./...` 通过；C.6/C.7 均 0 FAIL；
  `handlers/change` 与 `service.ChangeService` 用例失败清单与 HEAD 基线**完全一致**（零新增失败，用干净 worktree 对比确认）。

执行中发现的四个坑（都已处理，详见 §1.6）：O2M 陷阱、tenant_guard 硬耦合、方法名 `AddAffectedCiIDs`、语义分叉。

**验收**：`grep -n 'field.JSON' ent/schema/change.go` 无关联字段；影响分析可 JOIN；`go test ./...` 绿。
**风险**：entc 重新生成需 ~6GB 内存（`go build -o /tmp/entc entgo.io/ent/cmd/ent && cd ent && /tmp/entc generate ./schema`）；残片用 `git checkout -- itsm-backend/ent` 回退；**dev 改 schema 必须 `docker compose build itsm-init`**。

---

### B3 · SLA 收口：删内联字段，结束双写（1–2 天）

- `incident_service.go:334/337` 停止写 `sla_response_deadline` / `sla_resolution_deadline`，读写唯一入口为 `sla_states`。
- 先做一致性校验（对比内联字段与 `sla_states` 存量差异），再下线字段。

**验收**：写路径唯一；一致性校验脚本输出零差异；`go test ./service/... ` 绿。
**风险**：存量数据可能有差异窗口，必须先校验后下线，不可反序。

---

### B4 · 状态枚举对齐 + 审批 100% 收口（3–4 天）

- F6：`incident.go:27` / `problem.go:29` / `ticket.go:26` / `servicerequest.go:20` 从 `field.String` 升级 `field.Enum`（对齐已完成的 change/release）。
- F5：给 `ticket_workflow_service.go:378/796` 的 `ApproveTicket`/`ResolveTicket` 定 sunset 日期（建议与 legacy sunset **2026-11-01** 对齐），shadow 双轨同步明确删除时间。

**验收**：6/6 域编译期状态保护；审批单一编排层（BPMN）；`make product-drift` 仍绿（schema 数不增）。
**风险**：枚举化会收紧合法值域，存量脏数据需先清洗或显式映射；审批切换涉及业务中断，需在低峰期做并保留只读归档。

---

### B5 · AI 单一源 + 预览域转可用（持续，穿插进行）

- F7：固化 Go `LLMGateway` ↔ Python `itsm-ai-service` 边界（规则引擎唯一来源，Go 侧只调用不内置副本）。
- 按排队顺序把 9 个预览域转可用（**先易后难**）：

| 序 | 预览域 | 前置 |
|:--|:---|:---|
| 1 | 通知与连接器 | 飞书/钉钉/企微真实渠道验收（在途） |
| 2 | 报表与运营 | 统一管理者口径补齐（假数据已清） |
| 3 | BPMN 与审批 | B4 完成 |
| 4 | 知识与 RAG | F7 边界固化 |
| 5 | AI 辅助 | AI Evaluator v1 + F7 |
| 6 | 服务目录与请求 | 现有能力验收 |
| 7 | 问题与 Known Error | B4 枚举化 |
| 8 | 变更管理 | B2 + B4 |
| 9 | CMDB 云发现 | 依赖云厂商账号，优先级最低 |

**验收**：每转一个域，同步更新 README 成熟度表 + `docs/product/` + CHANGELOG（C.6.1 守卫会校验双向闭合）；README「预览」数下降，**同步收紧 `preview_domains` 基线**。

---

### B6 · 门禁强化（0.5 天，可并行）

- CI 传 `--strict`：`.github/workflows/docs-gate.yml` 当前未传，C.1–C.5 仍是 advisory（存量清零后再切）。
- 覆盖率口径：扩大 `jest.config.js` 的 `collectCoverageFrom`（当前仅 `src/lib/**`，168 页面/257 组件不在分母）。

**验收**：CI 全 hard；覆盖率口径披露与实际一致（C.6.5 守卫校验）。

---

## 4. 不做清单（STOP —— 本季度不启动）

| 来源 | 项 | 处置 |
|:---|:---|:---|
| v2.0 | Service decomposition / Event-driven / Multi-region / MSP billing / AI auto-triage(full) / Impact analysis skill / Plugin marketplace v2 / SOC 2 / BYOK | **Park**（文档保留，C.7.3 冻结增长） |
| v3.0 | Self-hostable LLM / Agent marketplace / Mobile PWA | **删除** |
| v3.0 | Multilingual UI (ja/ko) | **Park**（保留 en-US） |
| v1.7 | Skill registry v1 / RAG v2 / SLA forecast skill / Auto-triage | **降级**为"预览域转可用"附属项，不为它们单开能力面 |
| 通用 | 任何新菜单项 / 新域包 / 新页面 | "预览"域降到 ≤5 前**一律不批** |

---

## 5. 全批次通用纪律

1. **判定失败是否自引入**：`git stash push -u <显式列表>` → 干净基线重跑 → `git stash pop`。
2. **敏感扫描**：`git diff --cached | grep -nE "AdminProd|RlsAdmin|password|secret|token"`，必须在 commit 前跑，禁 `git add -A`。
3. **commitlint 不得跳过**（禁 `--no-verify`）；多行 message 用 `git commit -F`。
4. **改 Go 必 build + `up -d`**；改权限必重启 backend（权限缓存 TTL 5min，无失效端点）。
5. **每批结束必须跑**：`make verify-scripts && make scope-creep && make product-drift`。

---

## 6. 完成判据（5 条同时成立才算"不再扩散"）

1. `make product-drift` → C.6.4 **0 FAIL**（表面只降不升，且下降后已收紧基线）
2. `make scope-creep` → **0 FAIL 且 0 WAIVED** —— **已达成**（真实空壳 0，非豁免）
3. README「可用」能力域 **≥ 9**（当前 5；「预览」≤ 5）
4. ROADMAP v2.0/v3.0 未完条目 **≤ 14 且不增**（当前 14）
5. `service/` 生产文件 **≤ 300**（当前 333）

---

## 7. 建议排期

| 批次 | 内容 | 顺序建议 |
|:---|:---|:---|
| **B0** | 收口工作区 + 上调基线 5 项 + 删 `.disabled` | ✅ 已完成（`a4bc30bf`） |
| ~~B1~~ | ~~空壳收敛（−15 页）~~ | **撤销** —— 真实空壳 0，见 §1.5 |
| **B2** | `change.affected_cis` JSON → Ent edge | ✅ 已完成（`7e3344b0`），见 §1.6 四个坑 |
| **F8** | 修复 `service` 包测试编译失败，让守卫真的跑起来 | ✅ **已完成 2026-09-30**（`dba76f92` + `09cabc3c`），修复后暴露的存量失败见 §1.8 |
| B3 | SLA 删内联双写 | F8 后（先校验后下线，不可反序）—— **F8 已清，B3 可启动** |
| B4 | 状态枚举对齐 6/6（当前 **2/6**）+ 审批收口 BPMN | B3 后，择低峰 |
| B5 | AI 单一源 + 9 个预览域按序转可用 | 与 B2–B4 穿插 |
| B6 | 门禁强化（CI 传 `--strict`） | 可并行，存量清零后切 |

**为什么 B2 顶上 B1 的位置**（2026-09-28 代码复核确认，非文档推断）：

| 待办 | 复核结论 | 证据 |
|:---|:---|:---|
| B2 change JSON→edge | **未做，仍需排期** | `ent/schema/change.go:87/90` 仍是 `field.JSON("affected_cis"/"related_tickets")`；最近提交 `662294e0` 只加了 `edge.To("releases")` 与状态枚举，**没有动这两个字段** |
| B4 状态枚举 | **只完成 2/6** | `change.go:35`、`release.go:31` 已是 `field.Enum("status")`；`incident.go:27`、`problem.go:29`、`ticket.go:26`、`servicerequest.go:20` 仍是 `field.String` |

B2 顶上的理由：它是唯一同时满足「修缺陷」+「让一个预览域（变更）具备转可用条件」的事，且改动面收敛在单个实体，风险可控——这正是 B1 原本的生态位（高性价比、零架构风险）。

> 一句话：**B0 已让 CI 从红变绿，B2 已让变更域的影响分析拿到真实关系**；
> 接下来 **F8 优先级最高**（`service` 包测试整包编译不过 = 守卫空转），
> B3–B4 是硬骨头但都在既有能力内收口，不做完它们，9 个预览域永远转不了"可用"。
> ⚠️ 本轮两次修正（C.6.4 归因、空壳结论）都源于"用提交信息代替代码验证"，
> 后续每批次的验收必须以代码实测为准，不得复用文档结论。

---

## 8. 待拍板（B2 未覆盖，需产品决策）

| 项 | 现状 | 选项 | 建议 |
|:---|:---|:---|:---|
| `change.related_tickets` | JSON 字符串数组，存**工单号**；全仓**没有任何代码**把它解析成 Ticket 实体（只在治理字段白名单里允许提审后修改） | A 转 edge（需先定 ID vs 编号语义）/ B 保持 JSON / C 删除 | **先查用量再定**：若确认无业务依赖，C 最符合收敛；但它出现在 `governanceFieldsAlwaysEditable`，删除会改变提审后可编辑字段集合，属 API 契约变更，需单独评估 |
| 变更域第二运行时 `service/change_service.go` | 生产零引用，仅被自身测试与 2 个场景测试引用；语义与现行侧互斥 | A 删除（含测试迁移）/ B 保留 | **B（暂保留）** —— 场景测试 `scenario3/scenario6` 覆盖审批链与租户隔离，删除会掉覆盖；待有等价 `handlers/change` 场景用例后再删 |
| swagger `docs/docs.go` | 仍描述已移除的 `affected_cis` 字段 | 重新生成 | 低优先级，下次跑 swag 时一并更新 |
