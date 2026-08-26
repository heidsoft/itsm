# ITSM P0 问题修复计划

> **目标：** 修复审查发现的 5 个 P0 问题（4 个代码缺陷 + 1 个架构决策）
> **时间：** 2026-Q3
> **前提：** 每个 Task 独立子代理执行，两阶段 review（spec合规 → 代码质量）

---

## P0-1: Incident + SLA 关联

**问题：** `ent/schema/incident.go` 无 `sla_definition_id` / `sla_response_deadline` / `sla_resolution_deadline` 字段，`SLAMonitorService.CheckSLAViolations` 只查 Ticket，所有 P1/P2 事件不受 SLA 约束。

**影响：** Incident 工单完全绕过了 SLA 保护，不合规。

**修复方案：** 参考 `ent/schema/ticket.go` 的 SLA 字段模式，为 Incident schema 增加相同字段，并在 `IncidentService.Create` 时执行 SLA 策略匹配逻辑。

---

### Task 1: Ent Schema 添加 SLA 字段

**Objective:** 为 Incident schema 增加 SLA 相关字段

**Files:**
- Modify: `itsm-backend/ent/schema/incident.go`

**Step 1: 读取当前 schema**

```bash
cat itsm-backend/ent/schema/incident.go
```

**Step 2: 在 Fields() 中增加 SLA 字段（参考 ticket.go 第64-71行模式）**

在 `field.String("urgency")...Default("medium")` 之后、`field.Time("detected_at")...` 之前插入：

```go
field.Int("sla_definition_id").
    Comment("SLA定义ID").Optional(),
field.Time("sla_response_deadline").
    Comment("SLA响应截止时间").Optional(),
field.Time("sla_resolution_deadline").
    Comment("SLA解决截止时间").Optional(),
field.Time("sla_first_response_at").
    Comment("首次响应时间").Optional(),
field.Time("sla_resolved_at").
    Comment("SLA解决时间").Optional(),
```

**Step 3: 增加 Edges（参考 ticket.go 第145-146行）**

在 Edges() 中增加：

```go
edge.To("sla_violations", SLAViolation.Type),
edge.To("sla_alert_history", SLAAlertHistory.Type),
```

**Step 4: 验证 Ent 代码生成**

```bash
cd itsm-backend && go generate ./ent/schema/incident.go
# 预期：生成 ent/generated/incident.go 新增 SLA 相关方法
```

**Step 5: 编译验证**

```bash
cd itsm-backend && go build ./...
# 预期：0 errors
```

**Step 6: 提交**

```bash
git add itsm-backend/ent/schema/incident.go
git commit -m "feat(sla): add SLA fields to Incident schema (P0-1)"
```

---

### Task 2: IncidentService.Create 增加 SLA 策略匹配

**Objective:** 创建 Incident 时执行 SLA 策略匹配，写入 SLA deadline

**Files:**
- Modify: `itsm-backend/service/incident_service.go`（或 `handlers/incident/service.go`）

**Step 1: 查找 SLA 策略匹配实现**

```bash
grep -n "MatchSLAPolicy\|SLAPolicyService" itsm-backend/service/incident_service.go itsm-backend/service/sla_policy_service.go 2>/dev/null | head -20
```

**Step 2: 找到 Create 方法，在 incident 创建后调用 SLA 匹配**

参考 `ticket_sla_service.go` 中 `BindSLAToTicket` 模式：

```go
// 在 CreateIncident 成功后将以下逻辑加入事务
if incident.Priority != "" && incident.Severity != "" {
    policy, err := s.slaPolicyService.MatchSLAPolicy(ctx, tenantID, "incident", incident.Priority, incident.CategoryID)
    if err == nil && policy != nil {
        // 计算 deadline 并更新 incident
        responseDeadline := calculateBusinessDeadline(policy.ResponseTimeMinutes)
        resolutionDeadline := calculateBusinessDeadline(policy.ResolutionTimeMinutes)
        // Update incident with sla_definition_id, sla_response_deadline, sla_resolution_deadline
    }
}
```

**Step 3: 编译验证**

```bash
cd itsm-backend && go build ./...
```

**Step 4: 提交**

```bash
git add itsm-backend/service/incident_service.go
git commit -m "feat(sla): bind SLA policy on Incident creation (P0-1)"
```

---

### Task 3: SLAMonitorService 增加 Incident SLA 监控

**Objective:** `CheckSLAViolations` 同时查询 Ticket 和 Incident

**Files:**
- Modify: `itsm-backend/service/sla_monitor_service.go`

**Step 1: 读取当前 CheckSLAViolations 实现**

```bash
grep -n "CheckSLAViolations\|func.*Check" itsm-backend/service/sla_monitor_service.go | head -10
```

**Step 2: 增加 Incident SLA 检查分支**

在 `CheckSLAViolations` 或新增 `CheckIncidentSLAViolations` 方法中：

```go
// 检查 Incident SLA 违规
incidents, err := s.client.Incident.Query().
    Where(
        incident.TenantID(tenantID),
        incident.SLADefinitionIDNE(0), // 有SLA关联
        incident.SLAStatusEQ("active"), // 未解决
    ).All(ctx)

// 对每个 incident 检查 deadline
for _, inc := range incidents {
    if inc.SLAResponseDeadline != nil && time.Now().After(*inc.SLAResponseDeadline) {
        s.createViolationIfNotExist(ctx, inc.SLADefinitionID, "incident", inc.ID, "response")
    }
    if inc.SLAResolutionDeadline != nil && time.Now().After(*inc.SLAResolutionDeadline) {
        s.createViolationIfNotExist(ctx, inc.SLADefinitionID, "incident", inc.ID, "resolution")
    }
}
```

**Step 3: 编译验证**

```bash
cd itsm-backend && go build ./...
```

**Step 4: 提交**

```bash
git add itsm-backend/service/sla_monitor_service.go
git commit -m "feat(sla): check Incident SLA violations in monitor (P0-1)"
```

---

## P0-2: SLA 暂停功能实现

**问题：** `common/constants.go:214` 定义了 `SLAStatusPaused = "paused"`，但全工程无 `Pause/Resume/Suspend` 实现，暂停后计时不暂停。

**影响：** 不符合企业 SLA 合规要求（客户节假日/等待外部响应时应暂停计时）。

**修复方案：** 在 `SLAMonitorService` 增加 `PauseSLA` / `ResumeSLA` 方法，在 Ticket/Incident 中增加 `PauseSLA` / `ResumeSLA` API 端点。

---

### Task 4: SLAMonitorService 增加 Pause/Resume 方法

**Objective:** 实现 SLA 暂停和恢复逻辑

**Files:**
- Modify: `itsm-backend/service/sla_monitor_service.go`
- Modify: `itsm-backend/ent/schema/ticket.go`（如有需要增加 paused_at 字段）

**Step 1: 读取 SLAMonitorService 结构**

```bash
sed -n '1,60p' itsm-backend/service/sla_monitor_service.go
```

**Step 2: 增加 PauseSLA 和 ResumeSLA 方法**

在 `SLAMonitorService` 结构体中增加方法：

```go
// PauseSLA 暂停工单/事件的SLA计时
func (s *SLAMonitorService) PauseSLA(ctx context.Context, tenantID int, entityType string, entityID int, reason string) error {
    // 1. 记录暂停事件到 sla_pause_history 表（或 sla_alert_history 的 reason 字段）
    // 2. 记录暂停时间点
    // 3. 更新 Ticket/Incident 的 sla_status = "paused"
    return nil
}

// ResumeSLA 恢复工单/事件的SLA计时
func (s *SLAMonitorService) ResumeSLA(ctx context.Context, tenantID int, entityType string, entityID int) error {
    // 1. 计算已暂停时长
    // 2. 将暂停时长追加到 deadline（向后延长）
    // 3. 更新 sla_status = "active"
    return nil
}
```

**Step 3: 增加 Ticket.Incident SLA 暂停状态字段（如果尚无）**

```bash
grep -n "sla_status\|SLAStatus" itsm-backend/ent/schema/ticket.go itsm-backend/ent/schema/incident.go
```

如果无此字段，在 ent/schema/ticket.go 中增加：
```go
field.String("sla_status").Comment("SLA状态：active/paused").Default("active"),
```

在 ent/schema/incident.go 中增加：
```go
field.String("sla_status").Comment("SLA状态：active/paused").Default("active"),
```

重新生成 Ent 代码：
```bash
cd itsm-backend && go generate ./ent/schema/ticket.go && go generate ./ent/schema/incident.go
```

**Step 4: 编译验证**

```bash
cd itsm-backend && go build ./...
```

**Step 5: 提交**

```bash
git add itsm-backend/service/sla_monitor_service.go itsm-backend/ent/schema/ticket.go itsm-backend/ent/schema/incident.go
git commit -m "feat(sla): implement SLA pause/resume (P0-2)"
```

---

### Task 5: Ticket/Incident API 增加 Pause/Resume 端点

**Objective:** 暴露 REST API 供前端调用

**Files:**
- Modify: `itsm-backend/controller/ticket_controller.go`
- Modify: `itsm-backend/controller/incident_controller.go`
- Modify: `itsm-backend/router/router.go`

**Step 1: 在 TicketController 增加端点**

```go
func (c *TicketController) PauseSLA(ctx *gin.Context) {
    // POST /api/v1/tickets/:id/sla/pause
    // Body: { "reason": "等待客户响应" }
    // 调用 s.monitorService.PauseSLA(ctx, tenantID, "ticket", id, reason)
}

func (c *TicketController) ResumeSLA(ctx *gin.Context) {
    // POST /api/v1/tickets/:id/sla/resume
    // 调用 s.monitorService.ResumeSLA(ctx, tenantID, "ticket", id)
}
```

**Step 2: 在 IncidentController 增加端点**

同上模式。

**Step 3: 在 router.go 注册路由**

```go
tickets.PUT("/:id/sla/pause", ...)
tickets.PUT("/:id/sla/resume", ...)
inc.PUT("/:id/sla/pause", ...)
inc.PUT("/:id/sla/resume", ...)
```

**Step 4: 编译验证 + 提交**

```bash
cd itsm-backend && go build ./... && git add -A && git commit -m "feat(sla): add SLA pause/resume API endpoints (P0-2)"
```

---

## P0-3: SLA 违规率计算 Bug 修复

**问题：** `sla_monitor_service.go:488-490` 使用 `slaviolation.ResolvedAtIsNil()` 查询未解决违规，但意图是"未解决的 Ticket 对应的违规"，应改为联合 `ticket.ResolvedAtIsNil()`。

**影响：** 违规数永远偏低，SLA 合规率虚高。

**修复方案：** 修改 `GetSLAComplianceByDefinition` 中的违规统计查询。

---

### Task 6: 修复 SLA 违规统计查询

**Objective:** 修正违规率计算逻辑

**Files:**
- Modify: `itsm-backend/service/sla_monitor_service.go`

**Step 1: 读取当前错误代码**

```bash
sed -n '480,510p' itsm-backend/service/sla_monitor_service.go
```

**Step 2: 修复查询逻辑**

将：
```go
violated, _ := s.client.SLAViolation.Query().
    Where(
        slaviolation.SLADefinitionID(sla.ID),
        slaviolation.ResolvedAtIsNil(),
    ).
    Count(ctx)
```

改为（通过 JOIN ticket 表过滤未解决的 Ticket 对应的违规）：
```go
// 正确逻辑：查询有未解决 SLA 截止时间的 Ticket
violated, _ := s.client.Ticket.Query().
    Where(
        ticket.TenantIDEQ(tenantID),
        ticket.SLADefinitionID(sla.ID),
        ticket.DeletedAtIsNil(),
        ticket.ResolvedAtIsNil(), // Ticket 未解决
        // 以下条件任一满足则违规：
        ticket.SLAResponseDeadlineLT(time.Now()),  // 响应截止时间已过
        ticket.SLAResolutionDeadlineLT(time.Now()), // 解决截止时间已过
    ).
    Count(ctx)
```

**Step 3: 同样修复 metrics 计算中的违规逻辑（第420行附近）**

```bash
sed -n '415,430p' itsm-backend/service/sla_monitor_service.go
```

**Step 4: 编译验证**

```bash
cd itsm-backend && go build ./...
```

**Step 5: 提交**

```bash
git add itsm-backend/service/sla_monitor_service.go
git commit -m "fix(sla): correct violation count query (P0-3)"
```

---

## P0-4: BPMN CallbackRegistry 租户过滤

**问题：** `bpmn_callback_registry.go:63` 的 `HandleCallback` 中 `r.client.ProcessTask.Query().Where(processtask.ID(req.ProcessInstanceID))` **未加 TenantID 条件**，存在跨租户回调风险。

**影响：** 恶意或错误请求可操作其他租户的 BPMN 任务。

**修复方案：** 在 HandleCallback 中增加租户校验，或通过 context 注入 tenantID 并校验。

---

### Task 7: HandleCallback 增加租户校验

**Objective:** 修复跨租户回调风险

**Files:**
- Modify: `itsm-backend/service/bpmn/bpmn_callback_registry.go`

**Step 1: 读取当前 HandleCallback 实现**

```bash
sed -n '55,100p' itsm-backend/service/bpmn/bpmn_callback_registry.go
```

**Step 2: 修复 Query（从 ProcessInstance 获取 TenantID）**

```go
func (r *CallbackRegistry) HandleCallback(ctx context.Context, req *dto.CallbackRequest) error {
    // 1. 获取任务 + 实例
    task, err := r.client.ProcessTask.Query().
        Where(processtask.ID(req.ProcessInstanceID)).
        WithProcessInstance().
        Only(ctx)
    if err != nil {
        return errors.Wrap(err, "查询任务失败")
    }

    // 2. 租户校验（fail-closed）
    tenantID, ok := ctx.Value("tenant_id").(int)
    if !ok || tenantID == 0 {
        return errors.New("缺少有效租户上下文")
    }
    if task.Edges.ProcessInstance == nil || task.Edges.ProcessInstance.TenantID != tenantID {
        return errors.New("无权操作此任务")
    }

    // 3. 其余逻辑不变...
}
```

**Step 3: 编译验证**

```bash
cd itsm-backend && go build ./...
```

**Step 4: 提交**

```bash
git add itsm-backend/service/bpmn/bpmn_callback_registry.go
git commit -m "fix(bpmn): add tenant check to HandleCallback (P0-4)"
```

---

## P0-5: Ticket DDD 层架构决策

**问题：** `handlers/ticket/` 只有 `aggregate.go` 骨架，缺少实际 `handler.go/service.go/repository.go`，真正的业务逻辑在 `controller/ticket_controller.go` + `service/ticket_service.go`，造成**两套架构并存**的混乱。

**影响：** 长期维护困惑，新开发者不知该在哪层加代码。

**修复方案（二选一）：**

### Option A（推荐）：废弃 DDD 层，聚焦现有三层

删除 `handlers/ticket/` 目录，明确现有三层架构为唯一事实。

### Option B：完成 DDD 层，将 controller 层迁移

将 `controller/ticket_controller.go` 的逻辑迁移到 `handlers/ticket/` 的 handler.go/service.go/repository.go，接入现有的 `aggregate.go`。

---

### Task 8: 架构决策与执行

**Files:**
- Delete 或 Modify: `itsm-backend/handlers/ticket/`（根据决策）

**Step 1: 确认决策（与用户确认）**

```
建议选择 Option A（废弃 DDD 层），理由：
- controller + service + repository 三层已完整运行
- aggregate.go 无实际调用方
- 迁移成本高，风险大
- v1.6.x 应聚焦可靠性而非架构重构
```

**Step 2: 执行 Option A（如选择）**

```bash
# 1. 确认无任何代码引用 handlers/ticket/
grep -r "handlers/ticket" itsm-backend/ --include="*.go" | grep -v "^handlers/ticket/"

# 2. 删除目录
rm -rf itsm-backend/handlers/ticket/

# 3. 编译验证
cd itsm-backend && go build ./...

# 4. 提交
git add -A
git commit -m "refactor(ticket): remove incomplete DDD layer, standardize on controller+service+repository (P0-5)"
```

**Step 3: 执行 Option B（如选择）**

迁移路线图（较大工程，建议另开专项计划）。

---

## P0-6: Ticket 两套状态机统一

**问题：** `handlers/ticket/aggregate.go` 内的 `isValidStatusTransition` 与 `common.IsValidTicketStatusTransition` 不完全对齐，且 aggregate 方法未被调用。

**影响：** 两套状态转换规则，维护困惑。

**依赖：** Task 8（P0-5）先完成才能安全删除 aggregate.go。

---

### Task 9: 删除重复状态机，统一到 common 常量

**Files:**
- Modify: `itsm-backend/handlers/ticket/aggregate.go`（删除 `isValidStatusTransition` 相关代码）
- Verify: `common/constants.go:129` 确认单点权威

**Step 1: 确认 aggregate.go 中的 isValidStatusTransition 无调用方**

```bash
grep -rn "isValidStatusTransition" itsm-backend/
```

**Step 2: 删除 aggregate.go 中的重复状态转换 map**

在 `aggregate.go` 中找到并删除 `isValidStatusTransition` map 定义，同时移除引用它的 `TransitionAllowed` 方法（如果有）。

**Step 3: 编译验证**

```bash
cd itsm-backend && go build ./...
```

**Step 4: 提交**

```bash
git add itsm-backend/handlers/ticket/aggregate.go
git commit -m "refactor(ticket): remove duplicate status transition map, use common constants (P0-6)"
```

---

## 验证与测试

所有 Task 完成后执行以下验证：

```bash
# 1. 编译检查
cd itsm-backend && go build ./...

# 2. Ent 代码生成检查
go generate ./ent/...

# 3. 测试（如存在）
go test ./service/sla_monitor_service_test.go ./service/bpmn/... -v -count=1

# 4. 前端类型检查
cd itsm-frontend && npx tsc --noEmit
```

---

## 任务依赖关系

```
P0-5 (DDD层决策) → P0-6 (删除重复状态机)
P0-1 (Incident+SLA) → Task 2 (IncidentService.Create) → Task 3 (Monitor检查Incident)
P0-2 (SLA暂停) → Task 4 (Monitor Pause/Resume) → Task 5 (API端点)
P0-3 (违规Bug) → Task 6 (修复查询)
P0-4 (BPMN租户) → Task 7 (HandleCallback修复)
```

**建议执行顺序：** P0-4 → P0-3 → P0-1 → P0-2 → P0-5 → P0-6
