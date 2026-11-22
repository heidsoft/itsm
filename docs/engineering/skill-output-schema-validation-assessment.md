# Skill OutputSchema 执行后校验：评估与暂缓建议

> 结论先行：**本阶段不实施执行后 OutputSchema 校验**。理由不是技术上做不到，
> 而是它缺少「校验结果由谁消费」的答案——在没有消费方时新增校验器，只会再造一个
> `input_schema.go` 修复前那种「字段即声明、声明无消费」的假闭环。
>
> 本文是判断依据与「将来要做的三个前置条件」，对应
> `docs/engineering/skill-input-schema-validation-design.md` §边界与遗留的未做项。
> 日期 2026-11-22，基线 `HEAD=bceb5ac70`。

---

## 1. 现状事实（可指认）

- **OutputSchema 只存不验**：`handlers/skill/handler.go:341/445/855`、
  `persistence.go:55/75/122/143/177/250/314`、`custom_skill.go:107` 覆盖
  注册、更新、入库、读回、发现面，但执行链 `Invoke → InvokeWithMetrics` 从不读它。
- **声明本身准确，无漂移**：以 `TriageSkill` 为例，声明
  `{type:object, properties:{suggestions:{...}}}`（`handlers/ai/skills.go:151-164`）；
  `TriageTicket` 的正常路径、降级路径、缓存还原路径**全部**返回带 `suggestions`
  的 map（`handlers/ai/service.go:1721-1824`，三个 `suggestions` 构造点），形状一致。
  所以这里不存在「声明写得不对」的缺陷，问题纯粹是「声明没有消费方」。
- **声明质量参差**：`ChatSkill` 声明 `"sources": {"type":"array"}`（无 `items`），
  属松散声明；`TriageSkill` 声明则较严格。同一套校验器对两者的严格度需要产品口径。
- **无告警消费方**：全仓搜索 `outputSchema/output_schema/schemaMismatch`，
  除存储/发现面与 MCP 协议层（那是工具 `outputSchema`，与本议题不同）外，
  没有任何模块会「因为输出不符合声明」而采取行动。

---

## 2. 为什么不能直接照搬 InputSchema 的做法

`input_schema.go` 的成功前提是**校验发生在副作用之前**：拒绝即 `400`、
`calls=0`，声明落地且无副作用风险。OutputSchema 完全不同：

| 维度 | InputSchema（已落地） | OutputSchema（本议题） |
|:--|:--|:--|
| 校验时机 | 执行前 | 执行后 |
| 副作用 | 拒绝时尚未发生 | 失败时**已经发生** |
| 失败语义 | fail-closed 返回 400，不执行技能 | 不能返回 400——那会谎报「未执行」 |
| 回执一致性 | 简单 | 必须保持「已执行」的真实回执，另附契约告警 |

若执行后因 schema 不符返回 `4xx`，就会重蹈已修复的回执缺陷覆辙：
调用方/审计收到「失败」，但工单/知识库副作用已经发生——这正是
`output/agent-control-plane-gap-2026-11-21.md` §4 反复强调要根除的
「回执与实际状态不一致」。因此**输出校验绝不能复用 fail-closed 拒绝语义**。

---

## 3. 三个前置条件（缺一不可）

将来若要做，必须同时满足，否则不要开工：

1. **明确的告警消费方**：至少一处会「消费」契约漂移信号——
   例如运维看板统计「输出契约漂移率」、或技能市场下架连续漂移的技能、
   或把漂移计入 `ai_audit` 供回放。没有消费方的校验器 = 新的死信号。
2. **告警与执行解耦**：校验失败只产生可观测信号（结构化日志 +
   `errorClass=output_schema_mismatch` 审计），响应仍是真实 output + 成功回执，
   不改写、不拒绝、不回滚。与 MCP pending 回执「不用 isError」同一纪律。
3. **声明严格度收敛**：先统一内置技能的 OutputSchema 粒度（是否要求 `items`、
   是否允许额外字段 `additionalProperties`），再启用校验；否则松散声明要么
   永远不触发（等于没做），要么误报不断（噪声淹没信号）。

---

## 4. 本阶段建议

- **不做**执行后校验，也不新增「契约漂移」字段/列——避免制造无消费方的死信号。
- **保留** InputSchema 的前置校验（已落地、有 400 + calls=0 的闭环证据），
  它是真正阻止非法入参产生副作用的防线。
- 将本判断同步进 InputSchema 设计文档的「边界与遗留」，避免后续会话把
  「OutputSchema 未做」误读为「待补的快速缺口」。

## 5. 口径提醒

- 「声明存在」≠「声明正确」≠「声明被消费」。本次核实确认声明正确，
  但正确却无消费方依然不构成能力——这与 `dry_run` 死字段是同一类问题的不同阶段。
- 不要在「能写一个校验函数」与「形成了业务闭环」之间画等号。本仓已多次证明：
  没有消费方的校验/字段/声明，最终都会被当成装饰。
