# ITSM 产品完善方案

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 完善ITSM系统，修复API一致性、租户隔离、授权机制等核心问题，并补充缺失的业务模块

**Architecture:** 基于当前Go/Gin+Ent后端和Next.js/TypeScript前端的架构，进行API规范化改造、权限模型重构、知识库增强、SLA可视化完善，并新增供应商管理、合同管理模块

**Tech Stack:** Go/Gin, Ent ORM, Next.js/TypeScript, React, Ant Design, Redis, PostgreSQL

---

## 一、现状分析

### 1.1 已实现模块

| 模块 | 状态 | 代码位置 |
|------|------|----------|
| 工单(Ticket) | ✅ 完整 | `controller/ticket_controller.go`, `service/ticket_service.go` |
| 事件(Incident) | ✅ 完整 | `controller/incident_controller.go`, `service/incident_service.go` |
| 问题(Problem) | ✅ 完整 | `controller/problem_investigation_controller.go` |
| 变更(Change) | ✅ 完整 | `handlers/change/`, `dto/change_dto.go` |
| 服务目录 | ✅ 完整 | `controller/service_catalog_handler.go` |
| 知识库 | ✅ 完整 | `controller/knowledge_controller.go`, `service/rag_service.go` |
| BPMN工作流 | ✅ 完整 | `service/bpmn_process_engine.go` |
| SLA管理 | ✅ 完整 | `service/sla_monitor_service.go`, `service/sla_policy_service.go` |
| CMDB | ✅ 完整 | `controller/cmdb_controller.go` |
| 资产 | ✅ 完整 | `controller/asset_controller.go` |
| 通知 | ✅ 完整 | `service/notification_service.go` |
| 审批流 | ✅ 完整 | `controller/approval_controller.go` |
| AI功能 | ✅ 完整 | `service/llm_gateway.go`, `service/triage_service.go` |
| 多租户 | ✅ 完整 | `middleware/tenant.go` |
| RBAC | ✅ 完整 | `middleware/rbac.go` |

### 1.2 核心问题

#### 问题1: API命名不一致
- **现象**: 后端Ent使用snake_case，前端期望camelCase
- **影响**: 需要http-client转换，增加复杂度和潜在bug
- **根因**: DTO未统一规范，Mapper转换不完整

#### 问题2: 权限模型硬编码
- **现象**: 权限定义在`middleware/rbac.go`中硬编码
- **影响**: 无法动态管理权限，无法审计权限变更
- **根因**: 设计阶段未考虑DB驱动权限模型

#### 问题3: 租户隔离不完整
- **现象**: 部分服务缺少tenant_id过滤
- **影响**: 数据泄露风险
- **根因**: 部分查询遗漏租户过滤条件

#### 问题4: 错误处理不规范
- **现象**: 部分controller直接返回gin.H而非common.Fail()
- **影响**: API响应格式不一致
- **根因**: 缺乏统一的错误处理规范

#### 问题5: 缺失业务模块
- **供应商管理**: 无
- **合同/SLM管理**: 无
- **客户满意度调研**: 仅工单评分，无完整调研
- **移动端/API**: 无API-first设计

---

## 二、改进计划

### 阶段一: API一致性修复 (P0)

#### Task 1: 审计所有DTO的JSON标签

**Files:**
- Modify: `dto/ticket_dto.go`
- Modify: `dto/incident_dto.go`
- Modify: `dto/problem_dto.go`
- Modify: `dto/change_dto.go`
- Modify: `dto/cmdb_dto.go`
- Modify: `dto/sla_dto.go`
- Modify: `dto/knowledge_dto.go`

- [ ] **Step 1: 审计现有DTO**

```bash
# 查找所有使用snake_case的DTO
grep -r "json:\"" dto/*.go | grep -E "[a-z]+_[a-z]+" | head -50
```

- [ ] **Step 2: 创建DTO审计报告**

列出所有不一致的字段，制定转换计划

- [ ] **Step 3: 修复TicketDTO**

```go
// dto/ticket_dto.go
type CreateTicketRequest struct {
    Title               string   `json:"title"`
    Content             string   `json:"content"`
    CategoryID          int      `json:"categoryId"`
    Priority            string   `json:"priority"`
    RequesterID         int      `json:"requesterId"`
    AssigneeID          *int     `json:"assigneeId"`
    TemplateID          *int     `json:"templateId"`
    ParentTicketID      *int     `json:"parentTicketId"`
    TagIDs              []int    `json:"tagIds"`
    // ... 全部改为camelCase
}
```

- [ ] **Step 4: 修复IncidentDTO**

```go
// dto/incident_dto.go
type CreateIncidentRequest struct {
    Title               string   `json:"title"`
    IncidentNumber      string   `json:"incidentNumber"`  // 新增，缺失字段
    ReporterID          int      `json:"reporterId"`
    AssigneeID          *int     `json:"assigneeId"`
    ConfigurationItemID *int     `json:"configurationItemId"`
    Subcategory         string   `json:"subcategory"`
    ImpactAnalysis      string   `json:"impactAnalysis"`
    RootCause           string   `json:"rootCause"`
    ResolutionSteps     string   `json:"resolutionSteps"`
    DetectedAt          *time.Time `json:"detectedAt"`
    // ... 全部改为camelCase
}
```

- [ ] **Step 5: 修复ProblemDTO**

```go
// dto/problem_dto.go
type CreateProblemRequest struct {
    Title               string   `json:"title"`
    ProblemNumber       string   `json:"problemNumber"`  // 新增，缺失字段
    Description         string   `json:"description"`
    Category            string   `json:"category"`
    Priority            string   `json:"priority"`
    RequesterID         int      `json:"requesterId"`
    AssigneeID          *int     `json:"assigneeId"`
    RelatedIncidentIDs   []int    `json:"relatedIncidentIds"`
    RootCauseAnalysis   string   `json:"rootCauseAnalysis"`
    Workaround          string   `json:"workaround"`
    // ... 全部改为camelCase
}
```

- [ ] **Step 6: 修复ChangeDTO**

```go
// dto/change_dto.go - 确认以下结构
type CreateChangeRequest struct {
    Title               string   `json:"title"`
    ChangeNumber        string   `json:"changeNumber"`  // 新增，缺失字段
    Type                string   `json:"type"`
    Priority            string   `json:"priority"`
    ImpactScope         string   `json:"impactScope"`
    RiskLevel           string   `json:"riskLevel"`
    AssigneeID          *int     `json:"assigneeId"`
    PlannedStartDate    *time.Time `json:"plannedStartDate"`
    PlannedEndDate      *time.Time `json:"plannedEndDate"`
    ImplementationPlan  string   `json:"implementationPlan"`
    RollbackPlan        string   `json:"rollbackPlan"`
    AffectedCIs         []string `json:"affectedCIs"`
    RelatedTickets      []int    `json:"relatedTickets"`
    // ... 全部改为camelCase
}
```

- [ ] **Step 7: 修复CMDB/SLA/Knowledge DTO**

类似处理，确保所有JSON标签使用camelCase

- [ ] **Step 8: 运行测试验证**

```bash
cd /Users/heidsoft/Downloads/research/itsm/itsm-backend
go test ./dto/... -v
```

---

#### Task 2: 完善Mapper转换函数

**Files:**
- Modify: `dto/mappers.go`

- [ ] **Step 1: 审计mappers.go**

```bash
# 查看当前Mapper覆盖情况
grep -c "func To.*Response" dto/mappers.go
```

- [ ] **Step 2: 补充缺失的Mapper**

```go
// dto/mappers.go

// ToIncidentResponse converts Incident to IncidentResponse
func ToIncidentResponse(i *ent.Incident) *IncidentResponse {
    if i == nil {
        return nil
    }
    return &IncidentResponse{
        ID:                   i.ID,
        Title:                i.Title,
        IncidentNumber:       i.IncidentNumber,
        Description:          i.Description,
        Status:               i.Status,
        Priority:             i.Priority,
        ReporterID:           i.ReporterID,
        AssigneeID:           i.AssigneeID,
        ConfigurationItemID:  i.ConfigurationItemID,
        Category:            i.Category,
        Subcategory:          i.Subcategory,
        ImpactAnalysis:       i.ImpactAnalysis,
        RootCause:            i.RootCause,
        ResolutionSteps:      i.ResolutionSteps,
        DetectedAt:           i.DetectedAt,
        ResolvedAt:           i.ResolvedAt,
        ClosedAt:             i.ClosedAt,
        EscalatedAt:          i.EscalatedAt,
        EscalationLevel:      i.EscalationLevel,
        IsAutomated:          i.IsAutomated,
        IsMajorIncident:      i.IsMajorIncident,
        TenantID:             i.TenantID,
        CreatedAt:            i.CreatedAt,
        UpdatedAt:            i.UpdatedAt,
    }
}

// ToProblemResponse converts Problem to ProblemResponse
func ToProblemResponse(p *ent.Problem) *ProblemResponse {
    // ... similar implementation
}
```

- [ ] **Step 3: 添加批量转换函数**

```go
// ToIncidentResponseList converts multiple Incidents
func ToIncidentResponseList(list []*ent.Incident) []*IncidentResponse {
    result := make([]*IncidentResponse, len(list))
    for i, item := range list {
        result[i] = ToIncidentResponse(item)
    }
    return result
}
```

- [ ] **Step 4: 验证Mapper编译**

```bash
go build ./dto/...
```

---

### 阶段二: 租户隔离加固 (P0)

#### Task 3: 审计服务层租户过滤

**Files:**
- Modify: `service/ticket_service.go`
- Modify: `service/incident_service.go`
- Modify: `service/problem_service.go`
- Modify: `service/change_service.go`
- Modify: `service/cmdb_service.go`

- [ ] **Step 1: 检查ticket_service.go租户过滤**

```bash
grep -n "tenant" service/ticket_service.go | head -20
```

- [ ] **Step 2: 验证List方法过滤**

```go
// service/ticket_service.go
func (s *TicketService) ListTickets(ctx context.Context, tenantID int, page, size int, status, search string) ([]*ent.Ticket, int, error) {
    query := s.client.Ticket.Query().
        Where(ticket.TenantID(tenantID))  // 确保有租户过滤
    // ...
}
```

- [ ] **Step 3: 检查incident_service.go**

```go
// service/incident_service.go
func (s *IncidentService) ListIncidents(ctx context.Context, tenantID int, page, size int, status, search string) ([]*ent.Incident, int, error) {
    query := s.client.Incident.Query().
        Where(incident.TenantID(tenantID))  // 确保有租户过滤
    // ...
}
```

- [ ] **Step 4: 修复缺失的租户过滤**

如果发现遗漏，添加`Where(tenant.TenantID(tenantID))`

- [ ] **Step 5: 验证修复**

```bash
go test ./service/... -run Tenant -v
```

---

#### Task 4: BPMN审计日志租户隔离

**Files:**
- Modify: `service/bpmn_audit_service.go`

- [ ] **Step 1: 确认GetProcessTimeline有租户过滤**

```go
// service/bpmn_audit_service.go - 确认已有以下实现
func (s *BPMNAuditService) GetProcessTimeline(ctx context.Context, processInstanceKey string, tenantID int) ([]*ent.ProcessAuditLog, error) {
    query := s.client.ProcessAuditLog.Query().
        Where(processauditlog.ProcessInstanceKey(processInstanceKey))
    if tenantID > 0 {
        query = query.Where(processauditlog.TenantID(tenantID))
    }
    // ...
}
```

- [ ] **Step 2: 审计其他BPMN查询方法**

```bash
grep -n "TenantID" service/bpmn_audit_service.go
```

- [ ] **Step 3: 修复缺失的租户过滤**

对所有查询添加tenantID > 0条件判断

---

### 阶段三: 权限模型重构 (P1)

#### Task 5: 创建数据库权限模型

**Files:**
- Create: `ent/schema/permission_definition.go`
- Create: `ent/schema/role_permission.go`
- Modify: `middleware/rbac.go`

- [ ] **Step 1: 创建权限定义Schema**

```go
// ent/schema/permission_definition.go
package ent

import (
    "entgo.io/ent"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/index"
)

type PermissionDefinition struct {
    ent.Schema
}

func (PermissionDefinition) Fields() []ent.Field {
    return []ent.Field{
        field.String("resource").NotEmpty(),
        field.String("action").NotEmpty(),
        field.String("description"),
        field.String("display_name"),
        field.Int("category"),
    }
}

func (PermissionDefinition) Edges() []ent.Edge {
    return []ent.Edge{
        edge.To("roles", RolePermission.Type),
    }
}

func (PermissionDefinition) Index() []ent.Index {
    return []ent.Index{
        index.Fields("resource", "action").Unique(),
    }
}
```

- [ ] **Step 2: 创建角色权限关联Schema**

```go
// ent/schema/role_permission.go
package ent

import (
    "entgo.io/ent"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/edge"
)

type RolePermission struct {
    ent.Schema
}

func (RolePermission) Fields() []ent.Field {
    return []ent.Field{
        field.Int("role_id"),
        field.Int("permission_id"),
    }
}

func (RolePermission) Edges() []ent.Edge {
    return []ent.Edge{
        edge.From("role", Role.Type).Field(role_id).Ref("permissions").Unique(),
        edge.From("permission", PermissionDefinition.Type).Field(permission_id).Ref("roles").Unique(),
    }
}
```

- [ ] **Step 3: 生成Ent代码**

```bash
cd ent
go generate ./...
```

- [ ] **Step 4: 重构RBAC中间件**

```go
// middleware/rbac.go
func (rbac *RBAC) loadPermissionsFromDB() error {
    // 从数据库加载权限
    perms, err := rbac.client.PermissionDefinition.Query().All(ctx)
    if err != nil {
        return err
    }
    // 构建内存缓存
    rbac.permissionCache = make(map[string][]string)
    for _, p := range perms {
        key := fmt.Sprintf("%s:%s", p.Resource, p.Action)
        rbac.permissionCache[key] = append(rbac.permissionCache[key], p.Role)
    }
    return nil
}
```

- [ ] **Step 5: 添加权限管理API**

```go
// controller/permission_controller.go
func (c *PermissionController) ListPermissions(ctx *gin.Context) {
    perms, _ := c.client.PermissionDefinition.Query().All(ctx)
    common.Success(ctx, perms)
}

func (c *PermissionController) AssignPermission(ctx *gin.Context) {
    var req struct {
        RoleID       int `json:"roleId"`
        PermissionID int `json:"permissionId"`
    }
    // 创建角色权限关联
}
```

- [ ] **Step 6: 运行测试**

```bash
go test ./middleware/... -v
```

---

### 阶段四: 错误处理规范化 (P1)

#### Task 6: 审计Controller错误响应

**Files:**
- Modify: `controller/ticket_controller.go`
- Modify: `controller/incident_controller.go`
- Modify: `controller/problem_controller.go`

- [ ] **Step 1: 查找不一致的错误响应**

```bash
grep -rn "gin.H{" controller/*.go | grep -v "common.Success\|common.Fail"
```

- [ ] **Step 2: 修复ticket_controller.go**

```go
// 错误示例 (需修复)
c.JSON(200, gin.H{"code": 0, "data": result})

// 正确示例
common.Success(c, result)
common.Fail(c, 5001, "error message")
```

- [ ] **Step 3: 修复incident_controller.go**

同上，确保所有响应使用common.Success/Fail

- [ ] **Step 4: 修复problem_controller.go**

同上

- [ ] **Step 5: 添加统一错误日志**

```go
// common/response.go
func Fail(c *gin.Context, code int, message string) {
    c.JSON(http.StatusOK, gin.H{
        "code": code,
        "message": message,
    })
    // 添加审计日志
    zap.L().Warn("APIError",
        zap.Int("code", code),
        zap.String("path", c.Request.URL.Path),
        zap.String("message", message),
    )
}
```

---

### 阶段五: 新增业务模块 (P2)

#### Task 7: 供应商管理模块

**Files:**
- Create: `ent/schema/vendor.go`
- Create: `controller/vendor_controller.go`
- Create: `service/vendor_service.go`
- Create: `dto/vendor_dto.go`

- [ ] **Step 1: 创建Vendor Schema**

```go
// ent/schema/vendor.go
package ent

import (
    "entgo.io/ent"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/edge"
    "time"
)

type Vendor struct {
    ent.Schema
}

func (Vendor) Fields() []ent.Field {
    return []ent.Field{
        field.String("name").NotEmpty(),
        field.String("code").NotEmpty().Unique(),
        field.String("type"), // IT服务, 硬件供应商, 软件供应商
        field.String("contact_name"),
        field.String("contact_email"),
        field.String("contact_phone"),
        field.String("address"),
        field.String("website"),
        field.Float("rating"),
        field.Int("tenantID"),
        field.Time("createdAt").Default(time.Now),
        field.Time("updatedAt").Default(time.Now),
    }
}

func (Vendor) Edges() []ent.Edge {
    return []ent.Edge{
        edge.To("contracts", Contract.Type),
        edge.To("assets", Asset.Type),
    }
}
```

- [ ] **Step 2: 创建Contract Schema**

```go
// ent/schema/contract.go
package ent

import (
    "entgo.io/ent"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/edge"
    "time"
)

type Contract struct {
    ent.Schema
}

func (Contract) Fields() []ent.Field {
    return []ent.Field{
        field.String("contractNumber").NotEmpty().Unique(),
        field.Int("vendorID"),
        field.String("title"),
        field.String("type"), // 服务协议, 维护合同, 采购合同
        field.Float("value"),
        field.Time("startDate"),
        field.Time("endDate"),
        field.String("status"), // 生效, 到期, 终止
        field.String("description"),
        field.Int("tenantID"),
        field.Time("createdAt").Default(time.Now),
        field.Time("updatedAt").Default(time.Now),
    }
}

func (Contract) Edges() []ent.Edge {
    return []ent.Edge{
        edge.From("vendor", Vendor.Type).Field(vendorID).Ref("contracts").Unique(),
    }
}
```

- [ ] **Step 3: 生成Ent代码**

```bash
cd ent && go generate ./...
```

- [ ] **Step 4: 创建Vendor Service**

```go
// service/vendor_service.go
package service

type VendorService struct {
    client *ent.Client
}

func (s *VendorService) CreateVendor(ctx context.Context, vendor *Vendor) (*Vendor, error) {
    return s.client.Vendor.Create().
        SetName(vendor.Name).
        SetCode(vendor.Code).
        // ...
        Save(ctx)
}
```

- [ ] **Step 5: 创建Vendor Controller**

```go
// controller/vendor_controller.go
func (c *VendorController) CreateVendor(ctx *gin.Context) {
    var req dto.CreateVendorRequest
    if err := ctx.ShouldBindJSON(&req); err != nil {
        common.Fail(ctx, 1001, "参数错误")
        return
    }
    // ...
}
```

- [ ] **Step 6: 注册路由**

```go
// router/router.go
vendor := v1.Group("/vendors")
vendor.GET("", middleware.RequirePermission("vendor", "read"), vendorCtrl.ListVendors)
vendor.POST("", middleware.RequirePermission("vendor", "write"), vendorCtrl.CreateVendor)
```

---

#### Task 8: 客户满意度调研模块

**Files:**
- Create: `ent/schema/survey.go`
- Create: `ent/schema/survey_response.go`
- Create: `controller/survey_controller.go`
- Create: `service/survey_service.go`

- [ ] **Step 1: 创建Survey Schema**

```go
// ent/schema/survey.go
package ent

import (
    "entgo.io/ent"
    "entgo.io/ent/schema/field"
    "time"
)

type Survey struct {
    ent.Schema
}

func (Survey) Fields() []ent.Field {
    return []ent.Field{
        field.String("title").NotEmpty(),
        field.String("description"),
        field.String("type"), // NPS, CSAT, CES
        field.Bool("isActive"),
        field.Time("startDate"),
        field.Time("endDate"),
        field.JSON("questions", []SurveyQuestion{}),
        field.Int("tenantID"),
        field.Time("createdAt").Default(time.Now),
        field.Time("updatedAt").Default(time.Now),
    }
}

type SurveyQuestion struct {
    Question string `json:"question"`
    Type     string `json:"type"` // rating, text, choice
    Options  []string `json:"options,omitempty"`
    Required bool `json:"required"`
}
```

- [ ] **Step 2: 创建SurveyResponse Schema**

```go
// ent/schema/survey_response.go
package ent

import (
    "entgo.io/ent"
    "entgo.io/ent/schema/field"
    "time"
)

type SurveyResponse struct {
    ent.Schema
}

func (SurveyResponse) Fields() []ent.Field {
    return []ent.Field{
        field.Int("surveyID"),
        field.Int("ticketID"),
        field.Int("respondentID"),
        field.JSON("answers", []Answer{}),
        field.Int("score"),
        field.String("comment"),
        field.Int("tenantID"),
        field.Time("submittedAt").Default(time.Now),
    }
}

type Answer struct {
    QuestionIndex int `json:"questionIndex"`
    Value         interface{} `json:"value"`
}
```

- [ ] **Step 3: 生成Ent代码**

```bash
cd ent && go generate ./...
```

- [ ] **Step 4: 创建Survey Service和Controller**

```go
// service/survey_service.go
func (s *SurveyService) SubmitResponse(ctx context.Context, ticketID int, answers []Answer) error {
    // 计算NPS/CSAT分数
    score := calculateScore(answers)
    _, err := s.client.SurveyResponse.Create().
        SetTicketID(ticketID).
        SetAnswers(answers).
        SetScore(score).
        Save(ctx)
    return err
}
```

- [ ] **Step 5: 注册路由**

```go
// router/router.go
survey := v1.Group("/surveys")
survey.GET("", surveyCtrl.ListSurveys)
survey.POST("/responses", surveyCtrl.SubmitResponse)
survey.GET("/analytics", surveyCtrl.GetSurveyAnalytics)
```

---

### 阶段六: 前端增强 (P2)

#### Task 9: 前端类型与API对齐

**Files:**
- Modify: `src/types/biz/ticket.ts`
- Modify: `src/types/biz/incident.ts`
- Modify: `src/types/biz/problem.ts`
- Modify: `src/lib/api/ticket-api.ts`

- [ ] **Step 1: 统一Ticket类型**

```typescript
// src/types/biz/ticket.ts
export interface Ticket {
  id: number;
  ticketNumber: string;
  title: string;
  content: string;
  status: TicketStatus;
  priority: TicketPriority;
  categoryId: number;
  categoryName?: string;
  requesterId: number;
  requesterName?: string;
  assigneeId?: number;
  assigneeName?: string;
  slaResponseDeadline?: string;
  slaResolutionDeadline?: string;
  firstResponseAt?: string;
  resolvedAt?: string;
  ratedAt?: string;
  ratingComment?: string;
  isManagedByMsp: boolean;
  mspProviderId?: number;
  parentTicketId?: number;
  tagIds?: number[];
  // 一致使用camelCase
}
```

- [ ] **Step 2: 统一Incident类型**

```typescript
// src/types/biz/incident.ts
export interface Incident {
  id: number;
  incidentNumber: string;
  title: string;
  description: string;
  status: IncidentStatus;
  priority: IncidentPriority;
  reporterId: number;
  reporterName?: string;
  assigneeId?: number;
  assigneeName?: string;
  configurationItemId?: number;
  configurationItemName?: string;
  category: string;
  subcategory: string;
  impactAnalysis?: string;
  rootCause?: string;
  resolutionSteps?: string;
  detectedAt?: string;
  resolvedAt?: string;
  closedAt?: string;
  escalatedAt?: string;
  escalationLevel: number;
  isAutomated: boolean;
  isMajorIncident: boolean;
  tenantId: number;
  createdAt: string;
  updatedAt: string;
}
```

- [ ] **Step 3: 验证类型一致性**

```bash
cd /Users/heidsoft/Downloads/research/itsm/itsm-frontend
npm run type-check 2>&1 | head -50
```

- [ ] **Step 4: 修复类型错误**

根据类型检查输出修复剩余的不一致问题

---

### 阶段七: SLA可视化完善 (P1)

#### Task 10: SLA Dashboard增强

**Files:**
- Modify: `src/app/(main)/sla-monitor/page.tsx`
- Modify: `service/sla_monitor_service.go`
- Create: `dto/sla_dashboard_dto.go`

- [ ] **Step 1: 完善SLAMonitoringDTO**

```go
// dto/sla_dashboard_dto.go
type SLAMonitoringDashboard struct {
    ComplianceRate      float64           `json:"complianceRate"`
    ViolationRate       float64           `json:"violationRate"`
    TotalTickets        int               `json:"totalTickets"`
    AtRiskTickets       int               `json:"atRiskTickets"`
    BreachedTickets     int               `json:"breachedTickets"`
    UpcomingDeadlines   []SLADeadline     `json:"upcomingDeadlines"`
    TopViolations       []SLAViolationItem `json:"topViolations"`
    SLAByPriority       map[string]float64 `json:"slaByPriority"`
    TrendData           []SLATrendPoint   `json:"trendData"`
}

type SLADeadline struct {
    TicketID    int       `json:"ticketId"`
    TicketTitle string    `json:"ticketTitle"`
    Deadline    time.Time `json:"deadline"`
    SLA policy  string    `json:"slaPolicy"`
    TimeLeft    string    `json:"timeLeft"`
}
```

- [ ] **Step 2: 增强SLAMonitorService**

```go
// service/sla_monitor_service.go
func (s *SLAMonitorService) GetDashboardMetrics(ctx context.Context, tenantID int) (*SLAMonitoringDashboard, error) {
    // 实现完整的SLA仪表盘数据
    dashboard := &SLAMonitoringDashboard{
        ComplianceRate: calculateComplianceRate(tenantID),
        ViolationRate: calculateViolationRate(tenantID),
        TotalTickets: countTotalTickets(tenantID),
        AtRiskTickets: countAtRiskTickets(tenantID),
        BreachedTickets: countBreachedTickets(tenantID),
        UpcomingDeadlines: getUpcomingDeadlines(tenantID),
        TopViolations: getTopViolations(tenantID),
        SLAByPriority: getSLAByPriority(tenantID),
        TrendData: getTrendData(tenantID),
    }
    return dashboard, nil
}
```

- [ ] **Step 3: 增强前端SLA页面**

```tsx
// src/app/(main)/sla-monitor/page.tsx
const SLAMonitorPage = () => {
  const { data, loading } = useSLAStats();

  return (
    <div className="sla-monitor">
      {/* 合规率概览 */}
      <ComplianceOverview
        rate={data.complianceRate}
        total={data.totalTickets}
        breached={data.breachedTickets}
      />

      {/* 即将到期 */}
      <DeadlineList deadlines={data.upcomingDeadlines} />

      {/* 趋势图 */}
      <TrendChart trend={data.trendData} />

      {/* 按优先级SLA */}
      <PrioritySLAGrid slaByPriority={data.slaByPriority} />
    </div>
  );
};
```

- [ ] **Step 4: 验证SLA API**

```bash
curl -s "http://localhost:8090/api/v1/sla/monitoring?tenant_id=1" | jq .
```

---

## 三、实施优先级

| 阶段 | 任务 | 优先级 | 估计工时 |
|------|------|--------|----------|
| 一 | API一致性修复 | P0 | 16h |
| 二 | 租户隔离加固 | P0 | 8h |
| 三 | 权限模型重构 | P1 | 24h |
| 四 | 错误处理规范化 | P1 | 8h |
| 五 | 供应商管理模块 | P2 | 16h |
| 六 | 客户满意度调研 | P2 | 16h |
| 七 | SLA可视化完善 | P1 | 8h |
| 六 | 前端类型对齐 | P2 | 8h |

---

## 四、验收标准

- [ ] 所有DTO使用统一的camelCase JSON标签
- [ ] 所有服务层方法正确过滤tenant_id
- [ ] 权限系统支持数据库配置
- [ ] 所有Controller使用common.Success/Fail响应
- [ ] 供应商管理模块CRUD完整
- [ ] 客户满意度调研模块完整
- [ ] SLA仪表盘显示完整的合规率数据
- [ ] 前端type-check通过
- [ ] 后端go build通过
- [ ] 单元测试覆盖核心业务逻辑
