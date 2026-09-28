# ITSM 收敛研发计划（2026-09-28）

> 目标：**不写新功能，只做三件事** —— 把 CI 从红变绿、把 9 个"预览"做成真的能用、把数据模型的硬伤补上。
> （原目标中的"把菜单里的空壳清掉"已撤销 —— 复核后真实空壳为 0，见 §1.5。）
> 输入：[output/product-scope-convergence-2026-09-28.md](./../output/product-scope-convergence-2026-09-28.md)（盘点与 STOP/FIX/KEEP）
> 执行纪律：分批次推进，**每批跑测试 + `git diff` 核对后再提交**；失败先用 `git stash` 判定是否自引入。

---

## 0. 仓库现状快照（2026-09-28 08:57 实测）

| 项 | 状态 |
|:---|:---|
| 分支 | `main`，与远端 **0/0**（无落后、无未推送） |
| 最近提交 | `662294e0 feat(change/release): 状态枚举类型安全化 + Release→Change 关联 edge` |
| 未提交 | 7 项（详见 §1 B0） |
| Gate C.7 收敛守卫 | ✅ **0 FAIL / 0 WAIVED**（真实空壳 0，非豁免；详见 §1.5） |
| Gate C.6 口径漂移 | 🔴 **5 FAIL**（全部是 C.6.4 表面棘轮） |
| 后端构建 | 未在本轮验证（B0 需跑一次确认基线干净） |

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

### B2 · 变更域数据模型：JSON → Ent edge（2–3 天，**撤销 B1 后升为最高优先级重构**）

- `ent/schema/change.go:87/90` 的 `affected_cis` / `related_tickets` 改为多对多 edge。
- 配套：迁移脚本（存量 JSON 转关系行）、影响分析改为基于 edge 查询。
- **这是影响分析在变更维度唯一能闭环的前提**，也是 v2.0 "Impact analysis skill" 被 Park 的原因（它依赖此项）。

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
| **B0** | 收口工作区 + 上调基线 5 项 + 删 `.disabled` | **立即**（CI 当前红，唯一能转绿的动作） |
| ~~B1~~ | ~~空壳收敛（−15 页）~~ | **撤销** —— 真实空壳 0，见 §1.5 |
| **B2** | `change` 关联字段 JSON → Ent edge | **B0 后立即**，见下方"为什么 B2 顶上" |
| B3 | SLA 删内联双写 | B2 后（先校验后下线，不可反序） |
| B4 | 状态枚举对齐 6/6（当前 **2/6**）+ 审批收口 BPMN | B3 后，择低峰 |
| B5 | AI 单一源 + 9 个预览域按序转可用 | 与 B2–B4 穿插 |
| B6 | 门禁强化（CI 传 `--strict`） | 可并行，存量清零后切 |

**为什么 B2 顶上 B1 的位置**（2026-09-28 代码复核确认，非文档推断）：

| 待办 | 复核结论 | 证据 |
|:---|:---|:---|
| B2 change JSON→edge | **未做，仍需排期** | `ent/schema/change.go:87/90` 仍是 `field.JSON("affected_cis"/"related_tickets")`；最近提交 `662294e0` 只加了 `edge.To("releases")` 与状态枚举，**没有动这两个字段** |
| B4 状态枚举 | **只完成 2/6** | `change.go:35`、`release.go:31` 已是 `field.Enum("status")`；`incident.go:27`、`problem.go:29`、`ticket.go:26`、`servicerequest.go:20` 仍是 `field.String` |

B2 顶上的理由：它是唯一同时满足「修缺陷」+「让一个预览域（变更）具备转可用条件」的事，且改动面收敛在单个实体，风险可控——这正是 B1 原本的生态位（高性价比、零架构风险）。

> 一句话：**B0 是这个季度唯一必须现在做的**（让 CI 从红变绿、让守卫重新可信），
> B2–B4 是硬骨头但都在既有能力内收口，不做完它们，9 个预览域永远转不了"可用"。
> ⚠️ 本轮两次修正（C.6.4 归因、空壳结论）都源于"用提交信息代替代码验证"，
> 后续每批次的验收必须以代码实测为准，不得复用文档结论。
