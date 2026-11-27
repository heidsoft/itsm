# 飞书入站生产验收准备包（待人工执行 / 待审批）

> Status: draft · 2026-11-26 · 依赖：[PoC 演示脚本](../product/poc-demo-script-feishu-inbound.md)、[能力合同](../product/itsm-commercial-capability-contract.md)
>
> **本轮边界**：只产出文档。未配置任何真实凭证、未访问生产环境、未部署、未打 tag、未修改权限体系、未代签。
> 所有涉及 Secret、生产配置、故障注入与签字的动作必须由授权人执行。
>
> **晋升条件**（来自 `handlers/capability/handler.go:180` 的 `pilotReason`）：
> 真实租户收发 + 验签 + **一次真实故障的死信重放演练**。三项全部由人签字后才可在后续批次晋升成熟度。

---

## 1. 审批边界（谁可以做什么）

| 动作 | 执行人 | 是否需人工审批 | 本轮是否已执行 |
|:---|:---|:---|:---|
| 编写/审阅本文与证据模板 | AI / 工程 | 否 | **已执行** |
| 真实租户回调 URL 配置、密钥轮换 | 授权运维 | **是（Secret/凭证变更）** | 否 |
| 生产后端部署 / 配置下发 | 授权运维 | **是（生产部署）** | 否 |
| 受控故障注入与死信重放 | 授权运维 + 值班负责人 | **是（生产变更/破坏性操作）** | 否 |
| 跨租户/权限数据查询 | 审计或运维（只读） | 否（只读） | 否 |
| 验收结论与签字 | 真人验收人 | **是** | 否（栏位留空） |

---

## 2. 前置条件

1. 目标环境后端版本与 `/version` 一致，版本口径以 tag / CHANGELOG / `/version` 单一来源为准（当前 `v1.6.60`）。
2. 持 `connector:write` 的授权账号用于连接器配置与轮换；持 `system:write` 用于命令重放；持 `ai:write` 用于建议确认。
3. `ITSM_FEISHU_READY` 由运维按部署策略显式申报（未申报时能力面 `deploymentReady=false`，但**不影响**已装配的 webhook 实际可用性——不要把报告字段当执行闸）。
4. 飞书侧已创建应用、拿到 `app_id` / `app_secret` / `verification_token` / `encrypt_key`，并确定回调 URL
   `POST https://<host>/api/v1/feishu/webhook/<callbackInstanceId>`。
5. 目标租户已存在可用的活跃用户；发送者 `open_id` 需映射该用户，或配置
   `feishu_inbound.automation_reporter_user_id` 指向同租户活跃用户。
6. `feishu_inbound.mode` 默认 `observe_only`；如需建单须显式设为 `auto_create`（这是能力开关，不是权限绕过）。

> **禁止**：把任何真实密钥、`app_secret`、token、完整消息正文写入仓库、`output/` 证据文件或日志。
> 证据只留键名、ID、状态码、哈希与时间戳。

---

## 3. Step A — 能力面基线（只读）

```bash
curl -sS -H "Authorization: Bearer <TOKEN>" \
  "$ITSM_BASE_URL/api/v1/capabilities" \
  | jq '.data.items[] | select(.key=="connector.feishu")'
```

记录字段（用于签字栏）：

| 字段 | 期望 | 说明 |
|:---|:---|:---|
| `key` | `connector.feishu` | 能力唯一键 |
| `maturity` | `pilot` | 签字前保持 Pilot |
| `buildAvailable` | `true` | 代码已装配 |
| `deploymentReady` | 由 `ITSM_FEISHU_READY` 决定 | 报告字段，非执行闸 |
| `tenantReady` | 视目标租户健康探测 | 同上 |
| `dependencies` | `["notification"]` | 死信演练涉及通知投递 |
| `pilotReason` | 非空 | 写明真实租户验收与重放演练未做 |
| `acceptanceVersion` | `2026.11.25` | 能力矩阵口径版本 |

---

## 4. Step B — 连接器配置与密钥轮换（授权运维执行）

### 4.1 配置实例

`POST /api/v1/connectors/configs`（权限 `connector:write`）：

```json
{
  "name": "feishu",
  "provider": "feishu",
  "enabled": true,
  "credentials": {
    "app_id": "<由运维从密钥库注入>",
    "app_secret": "<由运维从密钥库注入>",
    "verification_token": "<由运维从密钥库注入>",
    "encrypt_key": "<由运维从密钥库注入>"
  },
  "settings": { "callbackInstanceId": "<高熵随机串>" }
}
```

- `credentials` 的键名是飞书外部协议口径（snake_case），仅在 connector 适配边界使用；
  核心 DTO 与前端仍为 camelCase。
- `settings.callbackInstanceId` 是入站租户解析的**唯一依据**；重复实例 ID 会 fail-closed 不解析租户，
  必须使用高熵值，不得用租户号或可枚举字符串。
- 响应与 `GET /api/v1/connectors/configs` 的 `credentials` **全部脱敏为 `******`**，只返回键名。

### 4.2 密钥轮换

`POST /api/v1/connectors/configs/feishu/rotate-secret`（权限 `connector:write`）：

```json
{ "newCredentials": { "app_secret": "<新值，由运维注入>" } }
```

- 无灰度窗口：旋转前确认飞书侧回调配置已同步，否则会短暂拒签。
- 服务端写 `audit_log`：`resource=connector`、`action=rotate_secret`，**只含键名不含值**。
- 响应字段：`name` / `provider` / `rotated` / `rotatedAt` / `note`，不回显密钥。

### 4.3 健康与脱敏核对

```bash
curl -sS -H "Authorization: Bearer <TOKEN>" "$ITSM_BASE_URL/api/v1/connectors/health" \
  | jq '.data.items[] | select(.name=="feishu") | {name,provider,ok,message,checkedAt}'
curl -sS -H "Authorization: Bearer <TOKEN>" "$ITSM_BASE_URL/api/v1/connectors/configs" \
  | jq '.data.items[] | select(.name=="feishu") | {name,credentials}'
```

判定：`ok=true`；第二条的 `credentials` 每个值必须都是 `******`，否则记录为敏感信息泄露缺陷并中止验收。

---

## 5. Step C — 真实租户收发（授权运维 + 验收人）

1. 在飞书侧发送一条约定的测试消息到已绑定机器人会话；内容使用无个人信息的固定串
   （例如 `ACCEPTANCE-FEISHU-20261126-01`）。
2. 后端链路（应逐环留证）：
   `POST /api/v1/feishu/webhook/:instance_id` → 验签（`sha256(ts+nonce+key+body)`）→ AES 解密 →
   nonce 去重 → 消息级幂等 → 租户由 instance 解析 → 建单/审计/命令同事务 → 分诊建议写底账。
3. 预期响应（HTTP 200）：

```json
{"code":0,"message":"success","data":{"eventType":"im.message.receive_v1","action":"ticket_created","ticketId":<id>,"ticketNumber":"TKT-FS-...","triageSuggestion":{"category":"...","priority":"...","confidence":0.9},"triageAuditId":<id>}}
```

4. 权威产物核对：

| 核对项 | 命令/位置 | 判定 |
|:---|:---|:---|
| 工单租户归属 | `GET /api/v1/tickets/:id` | `tenantId` = 目标租户；title 来自测试消息 |
| 审计 | `GET /api/v1/audit-logs?resource=feishu_inbound` | `action=ticket.create`，仅含不可变标识 |
| 投递命令 | `GET /api/v1/admin/operations/commands?aggregateType=ticket&aggregateId=<id>` | 幂等键含 tenant/messageId/ticketId，状态最终 `succeeded` |
| AI 决策底账 | `GET /api/v1/ai/audit-logs?scenario=feishu_inbound_triage` | `accepted` 为未决策；含 model/promptVersion/confidence |
| 重放 | 同 nonce 重投 → `403/2003`；新 nonce 同 message_id → `409/4090` | 不产生第二张工单 |

---

## 6. Step D — 真实故障 + 死信重放演练（需人工审批）

> **此步骤会改变生产状态，必须由授权运维与值班负责人共同批准并在窗口期执行。**
> 未经审批不得注入故障；不得通过修改业务数据制造假故障。

### 6.1 受控故障（二选一，按审批结果）

- **方案 A（推荐，影响面最小）**：在演练窗口临时停用通知投递 worker / 使目标 channel 不可达，
  使 `notification.deliver` 命令重试耗尽并进入 `dead_letter`；随后恢复依赖。
- **方案 B**：若环境已有自然产生的死信，不注入新故障，直接取一条真实死信进行重放（须记录原始失败原因）。

### 6.2 观测与取证

```bash
./scripts/prod-dead-letter-check.sh --json        # 存量基线，阈值 DEAD_LETTER_THRESHOLD=50
curl -sS -H "Authorization: Bearer <TOKEN>" \
  "$ITSM_BASE_URL/api/v1/admin/operations/commands?status=dead_letter&commandType=notification.deliver&page=1&pageSize=20"
```

记录：`id` / `tenantId` / `commandType` / `idempotencyKey` / `attempt` / `maxAttempts` / `lastError`（**净化后**：只保留稳定错误类，
不得含 token、密钥、正文）。

### 6.3 重放

```bash
curl -X POST -H "Authorization: Bearer <TOKEN>" \
  "$ITSM_BASE_URL/api/v1/admin/operations/commands/<id>/replay"
```

服务端保证（已由代码与测试锁定）：从 `dead_letter/cancelled` 才允许重放；按 `tenant_id + fencing_token` CAS；
递增 fencing token、清 lease、重置 `pending`；**保留原 idempotency key**；写审计含 `previousStatus`。

判定：

- 重放后命令最终 `succeeded`；通知投递审计出现一条与命令绑定的 `NotificationDelivery`，无重复投递。
- 若通知记录仍 failed：`POST /api/v1/notification-deliveries/:id/retry`（权限 `notification:write`）单独重试，
  记录失败分类，不把两类重试混为一谈。

### 6.4 中止条件

出现任一情况立即中止并回滚演练：死信数超过阈值且无法恢复；跨租户命令被观察到；`lastError` 泄漏敏感信息；
重复投递产生两条已送达记录。

---

## 7. Step E — 分诊建议人工确认（HITL）

```bash
curl -X PATCH -H "Authorization: Bearer <TOKEN>" -H 'Content-Type: application/json' \
  -d '{"accepted":true}' "$ITSM_BASE_URL/api/v1/ai/audit-logs/<auditId>"
```

- 权限 `ai:write`；仅翻转 `accepted`，**不会**自动改工单。
- 采纳 category/priority 在工单详情执行，带 `version` 乐观锁；记录采纳人、时间与 version。
- 拒绝时提交 `accepted:false` 并记录原因（原因写工单评论或审计备注，不写消息正文）。

---

## 8. 证据清单（只留脱敏字段）

| 编号 | 证据 | 来源 | 必含字段 |
|:---|:---|:---|:---|
| E1 | 能力面快照 | Step A | key/maturity/deploymentReady/acceptanceVersion/pilotReason |
| E2 | 连接器健康与脱敏快照 | Step B 4.3 | ok=true；credentials 全为 `******` |
| E3 | 密钥轮换审计 | Step B 4.2 | audit action/key names/rotatedAt，无值 |
| E4 | 真实收发响应 | Step C | code/ticketId/ticketNumber/triageAuditId |
| E5 | 工单 + 审计 + 命令 | Step C | tenantId/action/idempotencyKey |
| E6 | 死信存量与重放前后 | Step D | id/commandType/idempotencyKey/previousStatus/fencingToken/lastError(净化) |
| E7 | 建议确认 | Step E | auditId/accepted/采纳人/version |
| E8 | 自动化回归 | `prod-post-deploy-verify.sh`、`product-function-smoke.py`、`prod-dead-letter-check.sh` | 命令与输出路径 |

证据文件落本地 `output/acceptance/`（该目录按 `.gitignore` 为本地产物，不入库）。

---

## 9. 签字表（空栏待真人填写）

| 项目 | 填写 |
| --- | --- |
| 验收人 |  |
| 角色 / 部门 |  |
| 验收日期 |  |
| 目标环境 / 后端版本 |  |
| 证据文件（JSON / MD 路径） |  |
| 自动化回归命令与结果 |  |
| 真实租户收发（通过 / 不通过） |  |
| 验签与重放拒绝（通过 / 不通过） |  |
| 死信重放演练（通过 / 不通过） |  |
| 结论（通过 / 不通过 / 有条件通过） |  |
| 遗留问题与跟进批次 |  |
| 签字 |  |

> 签字栏为空即视为**未验收**；脚本绿灯、AI 结论或文档描述都不能代替签字。
> 三项晋升条件全部签字后，才能在**后续独立批次**修改 capability 成熟度，且需重跑文档门禁。

---

## 10. 回滚与中止

- 连接器配置异常：`DELETE /api/v1/connectors/configs/feishu`（`connector:write`）后按审批重新 Provision；
  注意这会破坏入站回调解析，必须在窗口期执行。
- 密钥轮换失败：用上次已审批的密钥再次 `rotate-secret`（不得在本文件中保存旧值）。
- 重放导致重复投递：立即 `POST /api/v1/admin/operations/commands/:id/cancel`，保留审计，人工对账。
- 任一敏感字段（密钥/token/消息正文）进入日志或证据文件：立即停止、按安全流程清理并上报。
