# PoC 演示脚本：飞书消息 → 工单（受控闭环）

> Status: draft · 2026-11-25 · 证据等级：Observed Problem（代码与测试可复核）+ Hypothesis（客户价值未验证）
>
> 用途：为决策记录 Q3 / Q6 提供**可现场演示的真实业务闭环**，不声称客户已确认，不构成报价依据。

---

## 1. 演示边界（先看不能演示什么）

当前代码实际具备的链路是：

```
飞书事件（验签 → AES 解密 → nonce 去重）
  → 消息级幂等去重（provider message_id）
  → 租户由 connector instance_id 解析（不信任消息体）
  → 发送者 open_id 映射本租户用户（或回落自动化账号）
  → 建单 + 审计 + 投递命令（同一事务）
  → 事务外生成分诊建议 + 写 ai_feedbacks 决策底账（accepted=NULL）
  → 响应 {code:0, data:{ticketId, ticketNumber, triageSuggestion, triageAuditId}}
  → 人工 PATCH /ai/audit-logs/:id 确认/拒绝 → 工单详情采纳 category/priority（带 version）
```

**尚未接线、不得在演示中声称具备**：

- 飞书侧回执消息：当前回执是 HTTP 200 信封，不是 IM 卡片/私聊回复。
- 自动应用建议：建议**不会**自动改写工单 category/priority/assignee/status，必须人工采纳。
- 生产租户真实收发、一次真实故障的死信重放演练：**未执行**，签字栏留空。

---

## 2. 前置条件（本地/演示环境）

1. 一个已启用飞书连接器实例，配置 `callbackInstanceId`（高熵值），`encrypt_key` 与 `verification_token` 已设置。
2. 租户 `SystemConfig`：
   - `feishu_inbound.mode = auto_create`（默认 `observe_only`，不建单）
   - 可选 `feishu_inbound.automation_reporter_user_id = <本租户活跃用户 ID>`
3. 发送者 `open_id` 映射到本租户活跃用户；否则需上面第 2 条的自动化账号。
4. 事件签名：`sha256(timestamp + nonce + encrypt_key + body)`，timestamp 在 ±5 分钟内。

---

## 3. 演示步骤

### 步骤 1：确认能力默认安全态

```bash
curl -s http://localhost:8090/api/v1/capabilities | jq '.data.items[] | select(.key=="connector.feishu")'
```

预期：`maturity=pilot`，`deploymentReady` 取决于 `ITSM_FEISHU_READY`。**不要**把它说成 GA。

### 步骤 2：发送一条合法消息事件

`POST /api/v1/feishu/webhook/:instance_id`（public 分组，无本系统鉴权），body 为飞书加密信封，
带 `X-Lark-Request-Timestamp` / `X-Lark-Request-Nonce` / `X-Lark-Signature`。

预期响应（HTTP 200）：

```json
{"code":0,"message":"success","data":{"eventType":"im.message.receive_v1","action":"ticket_created","ticketId":123,"ticketNumber":"TKT-FS-3-2147483647","triageSuggestion":{"category":"network","priority":"high","confidence":0.91,"model":"keyword","promptVersion":"triage-v1"},"triageAuditId":88}}
```

若 `ITSM_AI_READY` 未配置或 Guidance/LLM 不可用，建议走 keyword 降级链并带
`degraded:true`；不会伪造模型名，也不会让建单失败。

自动化验证已锁定该完整链路：`router/feishu_inbound_ticket_route_test.go`
（真实 `SetupFeishuRoutes` + 加密签名 + 租户/用户/配置种子）。

### 步骤 3：展示权威产物

```bash
# 工单（tenant 归属、title/description、requester）
GET /api/v1/tickets/123

# 审计（resource=feishu_inbound, action=ticket.create, 只含不可变标识）
GET /api/v1/audit-logs?resource=feishu_inbound

# 投递命令（op=notification.deliver，幂等键含 tenant/messageId/ticketId）
GET /api/v1/operational-commands?aggregateType=ticket&aggregateId=123

# 分诊建议底账（accepted 初始为 null/false，等待人工确认）
GET /api/v1/ai/audit-logs?scenario=feishu_inbound_triage
```

### 步骤 3b：人工确认（Human-in-the-Loop）

```bash
# 由持 ai:write 的操作者确认或拒绝；不是 AI 自行执行
PATCH /api/v1/ai/audit-logs/88  {"accepted": true}
# 采纳建议后在工单详情执行（带 version 乐观锁）
PATCH /api/v1/tickets/123       {"category":"network","priority":"high","version":1}
```

### 步骤 4：重复与重放

- 同一 `timestamp:nonce` 重投 → **403**（nonce 重放拒绝）。
- 新 nonce + 同一 `message_id` → **409**（消息级去重，不产生第二张工单）。
- 未知 `instance_id` → **400**（不得解析出租户）。

---

## 4. 演示证据清单（现场留存）

| 证据 | 获取方式 | 用途 |
|:---|:---|:---|
| HTTP 状态 + 业务 code + camelCase data | 步骤 2 响应 | 契约断言 |
| 工单 ID / 编号 / tenant / requester | 步骤 3 | 证明在正确租户建单 |
| AuditLog 行 | 步骤 3 | 证明来源可追溯且正文未入审计 |
| OperationalCommand 幂等键 | 步骤 3 | 证明 durable side effect |
| AI 建议 + 决策底账（`accepted=NULL`） | 步骤 3 / 3b | 证明建议是决策支持而非自动执行 |
| 403 / 409 / 400 三类拒绝 | 步骤 4 | 证明 fail-closed 与幂等 |

---

## 5. 版本归属建议（未批准）

| 环节 | 建议归属 | 理由 | 代码现状 |
|:---|:---|:---|:---|
| 验签 / 解密 / nonce 去重 | CE | 安全底线不应锁付费版 | 已 built，Pilot |
| 消息→工单建单闭环 | CE（基础能力） | 减少人工录入是核心产品价值 | 已 built，默认 observe_only |
| 生产飞书租户验收 + 死信重放演练 | EE 候选 | 企业交付背书成本 | 未执行，人工签字留空 |
| AI 分诊建议 → 人工确认 | EE 候选 | 治理/编排/承诺范畴 | 已接线：建议写底账、人工 PATCH 确认后采纳；不自动执行 |
| 多 IM 渠道（企微/钉钉）生产化 | EE 候选 | 需逐个真实租户验收 | Pilot，无入站派发 |

> 以上均为建议。当前代码**没有 edition gate**，不得对外宣称“企业版已限制某能力”。
> 需先决策 Q6/Q7，再由工程实现服务端门控与测试。

---

## 6. 人工签字（必须留空到真人签署）

- [ ] 真实飞书租户收发验收：签署人 ______ 日期 ______
- [ ] 一次真实故障的死信重放演练：签署人 ______ 日期 ______
- [ ] 客户价值确认（是否愿意为此付费）：签署人 ______ 日期 ______

AI 与脚本不得代签。未签字前该能力维持 Preview。

真实生产执行步骤、审批边界、故障注入与死信重放演练方案见
[飞书入站生产验收准备包](../acceptance/feishu-inbound-production-acceptance.md)。
