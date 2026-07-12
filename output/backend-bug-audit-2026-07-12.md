# ITSM 后端 Bug 审计报告（2026-07-12）

> 审计视角：资深后端测试 / 多租户安全审计
> 审计范围：`itsm-backend`（Go/Gin + Ent ORM + 原生 SQL），跳过 `ent/` 生成代码、`vendor/`、`*_test.go`、`router.go.backup`。
> 方法：3 路并行静态扫描（租户隔离泄漏 / 鉴权+业务逻辑 / 错误处理与空指针）+ 本人对 Critical/High 项逐条 `Read` 源码核实。
> 核实标记：**✅ = 已读源码确认**；**🔍 = agent 静态扫描标注，建议复核后再排期**。

---

## 1. 核心结论（根因）

**系统缺少全局租户查询拦截器（Ent `QueryInterceptor`）。** 已确认 `database/database.go:185` 仅 `ent.NewClient(ent.Driver(drv))`，无任何 `Intercept(...)` 注入 `tenant_id`；全仓搜到的 `Intercept` 均为 Ent 代码生成器样板，无业务调用。

→ 租户隔离**完全依赖每条查询人工加 `TenantIDEQ`**，一旦遗漏即为跨租户越权。这是绝大多数 Critical 项的根因。

## 2. 严重度总览

| 等级 | 数量 | 代表项 |
|---|---|---|
| **Critical** | 4 | C1 跨租户删审批链 · C2 ToolInvocation IDOR · C3 角色越权提权 · C4 vendor 误删/panic |
| **High** | 5 | H1 审批顺序不强制 · H2 canPerformAction 全放行 · H3 SLA 通知风暴 · H4 Refresh 当 Access · H5 Fail 无 Abort |
| **Medium** | ~12 | B4 SLA 违规永不解决 · B6 状态机未知态放行 · B7 currentLevel 脆弱 · 租户隔离遗漏若干 · 吞错静默失败等（见 §5） |

---

## 3. Critical 详述（✅ 已核实）

### C1 — 跨租户删除变更审批链（原生 SQL 无 tenant）✅
- **位置**：`service/change_approval_service.go:160, 173`
- **代码**：
  ```go
  _, err := s.client.Change.Get(ctx, req.ChangeID)                       // 160 无 tenant
  tx.ExecContext(ctx, "DELETE FROM change_approval_chains WHERE change_id = $1", req.ChangeID) // 173 无 tenant
  ```
- **根因**：`tenantID` 参数在该函数内未被使用；原生 `DELETE` 仅按 `change_id` 过滤，无租户谓词。
- **影响**：任一租户调用者传入其他租户的 `change_id`，即可**销毁该租户的变更审批链**（破坏性越权写），且无审计。
- **修复**：`DELETE` 加 `AND tenant_id = $2`；`Change.Get` 改为 `Change.Query().Where(change.ID(...), change.TenantIDEQ(tenantID)).Only(ctx)`；建议改用 Ent 写法 `client.ChangeApprovalChain.Delete().Where(...TenantIDEQ, ...ChangeIDEQ)`。

### C2 — ToolInvocation 跨租户 IDOR + 角色硬比较 Bug ✅
- **位置**：`controller/ai_controller.go:336-337, 343, 377`
- **代码**：
  ```go
  id, _ := strconv.Atoi(idStr)                                       // 错误被吞 → 非数字变 0
  inv, err := a.client.ToolInvocation.Get(c.Request.Context(), id)      // 无 tenant 过滤
  if role := c.GetString("role"); role != "admin" {                   // 硬比较
      common.Fail(c, common.ForbiddenCode, "仅管理员可审批")
  }
  ```
- **根因**：① `Get` 无租户谓词 → 任意已登录用户可枚举 `:id` 读取**其他租户**的工具执行结果/错误；admin 还可 `reject`/`approve` 他人记录并触发执行队列。② `role != "admin"` 把 `super_admin`/`sysadmin`（拥有 `*` 的超级角色）**错误拒绝**（应放行却拒绝）。③ 非数字 id→0，若系统存在 ID=0 记录则越权。
- **影响**：跨租户数据泄露 + 最高权限者审批功能失效；端点已在 Swagger 标注 `/ai/tools/{id}`、`/ai/tools/{id}/approve`，属"写好随时接上即 IDOR"。
- **修复**：加载加 `Where(toolinvocation.TenantIDEQ(c.GetInt("tenant_id")))`；角色判断改用 `IsAtLeastRole(c,"admin")` 或权限 `tool:approve`；id 解析失败返回 400。

### C3 — 创建用户接受任意角色 → 权限提权（admin→super_admin）✅
- **位置**：`service/user_service.go:76-82`
- **代码**：
  ```go
  if strings.TrimSpace(req.Role) != "" {
      role := strings.ToLower(strings.TrimSpace(req.Role))
      if role == "user" { role = "end_user" }
      uc = uc.SetRole(user.Role(role))   // 无"调用者角色 ≥ 目标角色"校验
  }
  ```
- **根因**：直接使用请求体 `req.Role` 设角色；DTO `oneof` 白名单含 `super_admin`；无层级校验。`controller/user_controller.go:53-56` 还允许 `super_admin` 用 `req.TenantID` 跨租户。
- **影响**：`POST /api/v1/users` 的调用者（只要持有 `user:write`）可把本租户用户设为 `super_admin`（获 `*` 全权限，含 `role:*`、`system_config:write`、`user:delete`）；`super_admin` 进一步可借 `req.TenantID` 向**其他租户**注入 `super_admin` → 跨租户管理员提权。
- **修复**：`CreateUser`/`UpdateUser` 增加"调用者角色 ≥ 目标角色"校验；`super_admin` 目标仅允许由 `super_admin` 自身且显式受控地设置；目标角色值不信任客户端，由后端按策略分配。

### C4 — vendor 删除/查询：id 解析吞错→0 + tenant_id 断言 panic ✅
- **位置**：`controller/vendor_controller.go:48-49, 59-61`
- **代码**：
  ```go
  id, _ := strconv.Atoi(ctx.Param("id"))   // 非数字 → 0，错误被吞
  tenantID, _ := ctx.Get("tenant_id")         // 忽略 exists 第二个返回值
  c.svc.DeleteVendor(ctx, id, tenantID.(int)) // 缺失时 panic: interface is nil, not int
  ```
- **根因**：① `strconv.Atoi` 错误丢弃，`DELETE` 落到 `id=0`；② `ctx.Get` 的 `exists` 被忽略，`tenantID.(int)` 在租户未注入时直接 panic。
- **影响**：`DELETE /vendors/abc` 可能误删 ID=0 记录并回 200 `"deleted"`（掩盖参数错误）；路由若未过注入租户的中间件，则请求级 panic（500）。
- **修复**：`if id, err := strconv.Atoi(...); err != nil { common.Fail(c, 400, ...); return }`；用 `ctx.GetInt("tenant_id")` 或 `if t, ok := ctx.Get("tenant_id"); !ok {...}`。

---

## 4. High 详述（✅ 已核实）

### H1 — 审批链顺序不强制（越级 / 提前完成）✅
- **位置**：`service/approval_service.go:305, 380-403, 648-718`
- **现象**：`createApprovalRecords` 一次性把所有级别都建成 `pending`；`SubmitApproval`(305) 仅校验 `approvalRecord.ApproverID != userID`（身份），**不校验该记录是否为"当前活动级别"**；`handleApprovalApproved`(395) 以"剩余 pending == 0"判完成。
- **影响**：3 级串行（经理→总监→CTO），CTO 可先批自己的记录；只要最后一条 pending 被清掉（无论 1、2 级是否批），工单即 `approved`。**顺序审批形同虚设**。
- **修复**：审批推进改为"仅当前 `CurrentLevel` 的 pending 记录可处理"；完成一级后 `SetCurrentLevel(level+1)` 激活下一级，再判完成。

### H2 — canPerformAction 对 approve 恒 true + 缺配置默认放行 ✅
- **位置**：`service/approval_service.go:447-469`
- **代码**：`case dto.ApprovalActionApprove: return true`；函数末尾 `return true // 未找到节点配置时默认允许`。
- **影响**：即便工作流配置了 `AllowReject=false` 等约束，approve 永远放行；节点配置缺失时 reject/delegate 也放行。与 H1 叠加，审批动作几乎不受工作流约束。
- **修复**：approve 也按节点配置校验；缺配置时**默认拒绝（fail-closed）**。

### H3 — SLA 升级通知风暴（去重失效）✅
- **位置**：`service/escalation_service.go:274-360, 362-433`
- **现象**：`processLongPendingTickets`/`processUnassignedTickets` 用 `SLAAlertHistory`（ResolvedAtIsNil）构建 `alertedTickets` 去重集合，但**发送通知后从不写入 `SLAAlertHistory` 记录** → 集合恒空 → 每轮全部重发。
- **影响**：`StartSLAWatcher` 每 5 分钟触发，所有 >24h 未解决、>2h 未分配工单的 requester/assignee/全体 admin 持续收到重复通知（通知通道 DoS / 运维骚扰）。
- **修复**：发送后 `Create` 一条 `SLAAlertHistory{AlertLevel, ResolvedAtNil, TicketID}`（与查询的 `"long_pending"`/`"unassigned"` 对齐）；或更稳妥用幂等键。
- **备注**：若 `SendNotification` 内部已落 `SLAAlertHistory`，需复核其 `AlertLevel` 取值是否与去重查询一致（目前查询用 `"long_pending"`/`"unassigned"`）。

### H4 — AuthMiddleware 不校验 TokenType（Refresh 可当 Access）✅
- **位置**：`middleware/auth.go:141-168`
- **代码**：仅校验签名算法（防 alg 混淆）+ `token.Valid`，**从不校验 `claims.TokenType == "access"`**。`GenerateRefreshToken` 生成的不含 Role/TenantID 的 refresh token 由 `ValidateRefreshToken` 单独校验类型，但 `AuthMiddleware` 从不调用它。
- **影响**：refresh token 直接放入 `Authorization: Bearer <refresh>` 即通过 `AuthMiddleware`（因 Role 为空，RBAC 多数会拒，但属校验完整性缺陷；同时缺 `aud`/`iss` 校验）。拿到 refresh token 者可长期试探路径。
- **修复**：`AuthMiddleware` 内 `if claims.TokenType != "access" { Fail(401); Abort }`；并补 `aud`/`iss` 校验。

### H5 — common.Fail 不调用 c.Abort() → 双重响应 panic（系统性）✅
- **位置**：`common/response.go:45-66`
- **代码**：`Fail` 仅 `c.JSON(...)`，**无 `c.Abort()`**。
- **影响**：任何 handler 在 `common.Fail(...)` 后忘记 `return` 并继续执行到 `c.JSON`/`Success`，触发 `http: superfluous WriteHeader call` panic。设计缺陷使"漏写 return"成为高风险隐性 bug 类。
- **修复**：在 `Fail`/`FailWithData` 末尾加 `c.Abort()`（注意：若调用方依赖 `Fail` 后继续逻辑需调整，建议全局替换检查）。

---

## 5. Medium / 待复核清单（🔍 agent 扫描，建议复核后再排期）

**租户隔离遗漏（原生 SQL / `User.Get` 无 tenant）：**
- 🔍 `service/problem_investigation_service.go:260,407,457` 用户 PII 按 `users WHERE id=$1` 无 tenant；`:387,497` 按 `problem_id` 查分析表无 tenant。
- 🔍 `service/cab_service.go:34,67,132,157` `User.Get`/`Change.Get` 无 tenant，可枚举跨租户用户 PII。
- 🔍 `service/escalation_service.go:151,437`、`service/ticket_automation_rule_service.go:72,100,129,172,250`、`service/simple_notification_service.go:137`、`service/bpmn_process_engine.go:403` 多处 `User.Get`/`Ticket.Get` 无租户。
- 🔍 `service/sla_policy_service.go:63,83,139` 非租户版 `Get/Update/Delete`，目前仅测试调用，属"一旦接上即越权"危险接口。
- 🔍 `handlers/common/service.go:240`、`handlers/common/repository_impl.go:139` `User.Get` 无 tenant。
- 🔍 `connector/builtin/feishu/connector.go:352` 飞书同步 `Ticket.Get` 无 tenant（风险低，来源已租户绑定）。

**鉴权 / RBAC：**
- 🔍 A3 `middleware/rbac.go:803-826`：授权判定仍用 JWT 里的 `role`，不与 DB 当前角色对齐 → 降权不即时生效。
- 🔍 A5 `middleware/rbac.go:362,963-973`：`DBOnly` 模式 DB 空权限时静默回退硬编码宽权限。
- 🔍 A6 `internal/bootstrap/default_credentials_guard.go:46-54`：默认 JWT secret 仅告警，非 production/saas 环境不检测 → 可被伪造 `super_admin`。

**业务逻辑：**
- 🔍 B4 `service/sla_monitor_service.go:240,411-424,548-576`：SLA 违规 `SetIsResolved(false)` 后全仓无 `SetIsResolved(true)` → 合规率/违约计数永久失真。
- 🔍 B5 `service/sla_monitor_service.go:272-291`：预警进度 `elapsed/totalDuration` 无 `totalDuration<=0` 保护 → 负时长 SLA 时预警被静默跳过或误触发。
- 🔍 B6 `service/incident_service.go:1048-1052`：状态机对未知 `status` 返回 `true`（允许），可被绕过（closed→in_progress）。
- 🔍 B7 `service/change_approval_service.go:271-277,338-339`：`currentLevel` 取"最后一行 level"、`nextApprover` 取切片首元素非最小 level → 顺序推导脆弱。

**错误处理 / 静默失败：**
- 🔍 `service/vendor_service.go:49-50`、`handlers/problem/repository_impl.go:260-264`：`Count`/`All` 错误被吞，DB 故障时返回 200 空数据/错误指标。
- 🔍 `service/bpmn_gateway_engine.go:374` `ProcessInstanceID` 非数字静默变 0 入库；`service/bpmn_version_service.go:174,217` 版本号解析失败变 0。
- 🔍 约 60+ 处分页 `strconv.Atoi(..., _)` 忽略错误（回落默认值，一般仅分页异常，Low）。

---

## 6. 最高优先级系统性修复

**在 `ent.Client` 挂载全局租户查询拦截器**，对带 `tenant_id` 字段的实体自动注入 `TenantIDEQ(currentTenant)`（从 context 取租户），从根上消除 C1/C2/§5 租户遗漏类缺陷。配套：
1. 写路径一律 `UpdateOneID(id).Where(TenantIDEQ(t))`（越权 ID 因 0 行影响被拒）。
2. 原生 SQL 必须带 `AND tenant_id = $n`。
3. 删除/加固 `sla_policy_service.go` 非租户版方法。
4. 路由级确认 `ai_controller` 的 `/ai/tools/{id}` 是否真的未注册（Swagger 已标注，属"随时接上即 IDOR"）。

## 7. 建议排期（P0→P1→P2）

- **P0（本周）**：C1、C2、C3、C4、H1、H2、H5（数据破坏 / 越权 / panic，阻塞生产）。
- **P1**：H3、H4、B4、B6、A3、A6（合规 / 安全 / 通知风暴）。
- **P2**：B5、B7、🔍 复核清单、租户隔离补全 + Ent 拦截器。
