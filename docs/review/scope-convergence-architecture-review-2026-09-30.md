# 架构审查：收敛状态实测报告（2026-09-30）

> 审查人：架构师视角（Qoder agent）
> 输入基线：[plans/scope-convergence-plan-2026-09-28.md](../../plans/scope-convergence-plan-2026-09-28.md)
> 审查方式：**全部结论基于当日代码/命令实测**，不复述文档结论（遵循计划 §7 末尾"验收必须以代码实测为准"的要求）。
> 审查前提：不新增功能，只评估已有功能是否稳定、收敛批次是否按序落地。

---

## 1. 总体判断

收敛方向正确、门禁骨架（Gate C.6 / C.7）已生效且实测全绿，**但存在一个比"功能未稳定"更严重的结构性问题：守卫在空转**。

- `service` 包测试整包编译失败（F8 未修复），意味着 SLA、变更、知识三个域的回归保护实际上是假的。
- 在 F8 修复之前执行 B3（SLA 删双写）/ B4（状态枚举化），等于无仪表飞行——这两批改动恰恰最需要该包的测试保护。

**当前不该讨论任何新功能。第一个动作是恢复测试可信度，而不是继续推进批次。**

---

## 2. 实测结果对照收敛计划

| 项 | 计划状态 | 2026-09-30 实测结论 | 证据 |
|:---|:---|:---|:---|
| Gate C.6 / C.7 | 0 FAIL | ✅ 实测均 0 FAIL / 0 WAIVED；`go build ./...` 通过；C.6.4 棘轮 5 项全部等于基线（未涨） | `make product-drift`、`make scope-creep` |
| **F8** service 包测试编译 | 提到 B3 之前 | 🔴 **未修复，仍是最高优先级** | `go vet ./service/` → `sla_monitor_service_test.go:258` 等多处 `createViolation` 缺 `int` 参数（`662294e0` 枚举化漏改）；`change_service_test.go` 缺 `itsm-backend/ent/change` 导入问题同源 |
| B3 SLA 删双写 | F8 后 | ⏳ 未动。内联字段仍在写 | `service/incident_service.go:334` 仍调用 `SetSLAResponseDeadline`，随后双写 `sla_states`（Phase 3 Step 3.5 阶段 1） |
| B4 状态枚举 6/6 | B3 后 | 🔶 仍 **2/6** | `change.go`/`release.go` 已 `field.Enum("status")`；`incident.go`/`problem.go`/`ticket.go`/`servicerequest.go` 仍是 `field.String("status")` |
| B5 预览域转可用 | 穿插 | ⏳ 未动。README 实测 **可用 5 / 预览 9** | README 成熟度表：工单与事件、工单类型、CMDB、SLA、RBAC/多租户 = 可用；距完成判据（≥9 可用 / ≤5 预览）还差 4 个域 |
| B2 change JSON→edge | 已完成 | ✅ `affected_cis` 已是 M2M edge；`related_tickets` 按计划保留 JSON | `ent/schema/change.go:87` 仅剩 `related_tickets` 一处 `field.JSON` |
| §8 待拍板 3 项 | 需产品决策 | ⏳ 全部悬而未决 | `related_tickets` 语义（A/B/C）、legacy `service/change_service.go` 第二运行时去留、swagger `docs/docs.go` 重新生成 |

---

## 3. 架构师提出的三个问题

### 3.1 F8 是信任链断裂，不是普通 bug

计划 §1.6 已发现"同一列两套互斥语义"风险（change 域 JSON 列在现行 handler 存数字 CI ID、legacy service 存 CI 名称），而覆盖这类问题的 `service` 包测试整包编译不过。这意味着：

- B2/B4 这类枚举化、语义收敛改动的回归保护是**假的**；
- 门禁显示"测试绿"的表象下，该包所有用例（含 SLA、变更、知识）从未执行。

**必须最先修复**，且修复后要重建该包的失败基线（哪些用例本来就红、哪些是新暴露的）。

### 3.2 完成判据 5 存在口径歧义

计划 §6 写"`service/` 生产文件 **≤300**（当前 333）"，但守卫脚本 `scripts/docs-gate/check-product-drift.sh:346` 的统计口径是 `ls service/*.go`（**含测试文件**）= 333；实测**生产文件（排除 `_test.go`）只有 207，早已达标**（递归统计为 207）。

建议：把判据措辞与守卫口径对齐（明确"含/不含测试"），避免后续误判为未达标，或据此做出错误的删文件动作。

### 3.3 本地 main 领先 origin/main 4 个提交未推送

收敛计划要求"每批跑测试 + CI 核对后再提交"，但 B0（`a4bc30bf`）、B2（`7e3344b0`）等关键批次**尚未经过远端 CI 验证**。建议尽早推送（推送前按既定规范确认提交内容归属、无夹带敏感文件）。

---

## 4. 建议执行顺序（维持"不新增功能"）

| 序 | 动作 | 说明 |
|:--|:---|:---|
| 1 | **修 F8**（约 0.5 天） | 修复 `service` 包 3 个测试文件的编译错误，让 `go test ./service/` 真实跑起来；以实际结果重建该包失败基线并记录在计划 §1.7 |
| 2 | **推送 main** | 让远端 CI 验证已提交的 B0/B2/F8 批次 |
| 3 | **B3：SLA 删双写** | 先出一致性校验脚本（内联字段 vs `sla_states` 存量差异），**零差异确认后**再停写 `SetSLAResponseDeadline`/`SetSLAResolutionDeadline`；顺序不可反 |
| 4 | **B4：状态枚举 6/6** | incident/problem/ticket/servicerequest 升级 `field.Enum`；先做存量脏数据清洗或显式映射；审批 sunset 日期（建议 2026-11-01）落档 `ticket_workflow_service.go:378/796` |
| 5 | **B5 穿插推进** | 每完成一个域收敛，按排队顺序把对应预览域转"可用"，同步更新 README 成熟度表并**收紧 `preview_domains` 基线**——这是唯一被允许的"进展"形式 |
| 6 | **§8 三项决策定截止点** | 悬而不决会让 legacy `change_service.go` 变成 permanent dead code；`related_tickets` 建议先查用量再定（无业务依赖则倾向删除，但注意其出现在 `governanceFieldsAlwaysEditable`，删除属 API 契约变更） |

---

## 5. 完成判据现状（计划 §6 五条）

| 判据 | 当前实测 | 达标 |
|:---|:---|:--|
| 1. `make product-drift` C.6.4 0 FAIL | 5 项全部等于基线 | ✅ |
| 2. `make scope-creep` 0 FAIL 且 0 WAIVED | 0/0/0 | ✅ |
| 3. README 可用 ≥9（预览 ≤5） | 可用 5 / 预览 9 | ❌ 差 4 个域 |
| 4. ROADMAP v2.0/v3.0 未完条目 ≤14 且不增 | 14 | ✅（临界） |
| 5. `service/` 文件 ≤300 | 门禁口径 333（含测试）；生产文件 207 | ⚠️ 口径待澄清（见 §3.2） |

另有一项隐含判据：**F8 = 守卫真实在跑**。当前不成立，应显式纳入"不再扩散"的判定条件。

---

## 6. 一句话结论

> 门禁让"不再扩散"变成了构建失败，这是对的；但 **`service` 包测试空转使当前所有"绿"都带着一个未定价的风险**。先修 F8、推 CI，再按 B3→B4 收口硬骨头，预览域转可用只能作为收敛批次完成的副产品，不单独开工。

---

*实测命令记录：`go vet ./service/`、`go build ./...`、`make product-drift`、`make scope-creep`、`grep field.Enum/String ent/schema/{incident,problem,ticket,servicerequest,change,release}.go`、`sed -n '325,345p' service/incident_service.go`、README 成熟度表人工核对。*
