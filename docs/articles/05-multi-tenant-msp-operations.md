# 多租户架构：MSP服务商的高效运营之道

## 一套系统服务百家客户，技术架构与运营实践全解析

---

MSP（Managed Service Provider，托管服务提供商）是IT运维领域的重要角色。

他们同时为多家企业提供IT服务，用一套系统、一套团队服务多个客户。每个客户有自己的IT环境、自己的工单、自己的数据，但共享底层基础设施和运维资源。

这种商业模式对系统提出了特殊要求：**多租户隔离**。

如果租户A的数据泄露给了租户B，这是灾难性的。如果租户B的工单被租户C的处理人看到，这也是不可接受的。多租户架构是MSP业务的技术基石。

本文将详细介绍ITSM开源项目的多租户架构设计，以及MSP场景下的运营实践经验。

---

## 第一部分：MSP的商业逻辑与技术挑战

### 1.1 MSP的商业模式

MSP的本质是**资源复用**。

一家MSP可能同时服务50家客户。每个客户有自己的IT团队、有自己的工单处理流程、有自己的服务目录。如果为每个客户都部署一套独立系统，MSP需要维护50套环境，成本高昂。

MSP的解法是用**一套系统服务多个客户**。所有客户共享底层的服务器、数据库、缓存资源，但数据完全隔离、流程各自独立。

这种模式的优势：

- **成本分摊**：基础设施成本由多个客户分担
- **效率提升**：运维团队可以同时服务多个客户
- **标准化**：可以沉淀标准化的服务流程和最佳实践
- **规模效应**：客户越多，边际成本越低

但这也带来了技术挑战：如何在共享资源的同时保证数据隔离？如何满足不同客户的个性化需求？

### 1.2 多租户的技术挑战

多租户架构有三大挑战：

**数据隔离**。每个租户的数据必须完全隔离，租户A不能看到租户B的数据。这不只是功能问题，更是合规要求。很多MSP服务的客户是金融、医疗、政府等敏感行业，数据泄露可能引发严重后果。

**性能保障**。当一个租户负载激增时，不应该影响其他租户。这需要资源隔离和流量控制机制。

**灵活定制**。租户A可能用惯了JIRA的工单格式，租户B可能要求特殊的审批流程。系统需要支持一定程度的个性化定制，但不能因此破坏共享架构。

### 1.3 租户隔离策略

多租户隔离有三种主要策略：

**独立数据库**。每个租户一个数据库实例，完全隔离但成本高。

**共享数据库、独立Schema**。所有租户共用一个数据库实例，但每个租户有独立的Schema。隔离性中等，成本较低。

**共享数据库、共享Schema**。所有租户共用同一套表结构，通过租户ID字段区分。隔离性最低，但架构最简单、扩展性最好。

ITSM采用**第三种策略——共享Schema + 租户ID隔离**，同时提供Schema级别隔离的**企业版**作为可选项。

```
┌─────────────────────────────────────────────────────────┐
│                     应用层                               │
│   租户A请求 │ 租户B请求 │ 租户C请求                      │
└─────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────┐
│                   租户中间件层                           │
│   租户上下文 │ 租户识别 │ 权限校验                      │
└─────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────┐
│                   共享数据库层                           │
│   ┌─────────────────────────────────────────────────┐  │
│   │  tenants: 租户表                                 │  │
│   │  users: 用户表 (tenant_id外键)                   │  │
│   │  tickets: 工单表 (tenant_id外键)                 │  │
│   │  ...                                             │  │
│   └─────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

---

## 第二部分：租户数据模型

### 2.1 租户实体

租户是系统中的顶级实体。

```go
// 租户模型
type Tenant struct {
    ID          string    // 租户唯一标识
    Name        string    // 租户名称
    Code        string    // 租户编码（用于URL等场景）
    Status      string    // active | suspended | deleted
    Plan        string    // free | standard | enterprise
    Quota       Quota     // 资源配额
    Settings    JSON      // 租户个性化设置
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// 资源配额
type Quota struct {
    MaxUsers      int // 最大用户数
    MaxTickets    int // 最大工单数
    MaxStorageGB  int // 最大存储GB
    APIRateLimit  int // API调用频率限制
}
```

每个租户有自己的：

- 用户和角色
- 工单和服务目录
- 工作流和审批链
- 知识库内容
- SLA策略
- 通知配置

### 2.2 租户感知的数据模型

所有租户相关的数据表都有`tenant_id`字段，这是隔离的基础。

```sql
-- 用户表
CREATE TABLE users (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    username VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL,
    -- ... 其他字段
    UNIQUE(tenant_id, username)
);

-- 工单表
CREATE TABLE tickets (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    title VARCHAR(500) NOT NULL,
    status VARCHAR(50) NOT NULL,
    priority VARCHAR(20) NOT NULL,
    assignee_id UUID REFERENCES users(id),
    -- ... 其他字段
);

-- 服务目录表
CREATE TABLE service_catalogs (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(200) NOT NULL,
    description TEXT,
    -- ... 其他字段
);
```

在数据操作层，所有查询都会自动带上`tenant_id`过滤条件，确保数据不会跨租户泄露。

### 2.3 跨级联删除

当租户被删除时，需要级联删除所有相关数据。

```go
// 租户删除服务
func (s *TenantService) DeleteTenant(ctx context.Context, tenantID string) error {
    return s.db.WithTx(ctx, func(tx *ent.Tx) error {
        // 1. 删除租户用户
        if err := tx.User.Delete().
            Where(user.TenantID(tenantID)).
            Exec(ctx); err != nil {
            return err
        }

        // 2. 删除工单
        if err := tx.Ticket.Delete().
            Where(ticket.TenantID(tenantID)).
            Exec(ctx); err != nil {
            return err
        }

        // 3. 删除服务目录
        if err := tx.ServiceCatalog.Delete().
            Where(svc.TenantID(tenantID)).
            Exec(ctx); err != nil {
            return err
        }

        // 4. 删除知识库
        if err := tx.KnowledgeArticle.Delete().
            Where(ka.TenantID(tenantID)).
            Exec(ctx); err != nil {
            return err
        }

        // 5. 最后删除租户本身
        return tx.Tenant.DeleteOneID(tenantID).Exec(ctx)
    })
}
```

---

## 第三部分：租户上下文与识别

### 3.1 租户上下文传递

每个请求都需要知道它属于哪个租户。ITSM通过**租户上下文**（Tenant Context）来传递这个信息。

租户上下文可以来自多个途径，按优先级排序：

**1. Header中的租户标识**

```
X-Tenant-ID: tenant_abc123
```

这是API调用时的标准方式。

**2. Subdomain中的租户编码**

```
https://acme.itsm-cloud.com/
```

SaaS化部署时常用，不同租户对应不同子域名。

**3. JWT Token中的租户信息**

如果使用JWT认证，租户ID可以直接编码在Token中。

**4. Cookie或Session**

Web场景下，登录时记录租户ID在Session中，后续请求自动带上。

```go
// 租户上下文中间件
func TenantMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        tenantID := extractTenantID(c)

        if tenantID == "" {
            c.AbortWithStatusJSON(400, gin.H{
                "code":    1001,
                "message": "租户ID不能为空",
            })
            return
        }

        // 验证租户存在且有效
        tenant, err := tenantService.GetByID(c.Request.Context(), tenantID)
        if err != nil {
            c.AbortWithStatusJSON(404, gin.H{
                "code":    1002,
                "message": "租户不存在",
            })
            return
        }

        // 设置到上下文
        ctx := context.WithValue(c.Request.Context(), "tenant", tenant)
        c.Request = c.Request.WithContext(ctx)

        c.Next()
    }
}

// 提取租户ID的多种方式
func extractTenantID(c *gin.Context) string {
    // 1. Header
    if id := c.GetHeader("X-Tenant-ID"); id != "" {
        return id
    }

    // 2. Subdomain
    if parts := strings.Split(c.Request.Host, "."); len(parts) > 0 {
        return parts[0]
    }

    // 3. JWT Token
    if claims, ok := c.Get("jwt_claims"); ok {
        return claims.(jwt.Claims).TenantID
    }

    return ""
}
```

### 3.2 数据层自动注入

业务代码不应该每次都手动添加`tenant_id`过滤条件。ITSM在数据层实现了**自动注入**。

```go
// 基于Ent的数据层扩展
func (s *TenantClient) Ticket() *TenantTicketQuery {
    q := s.client.Ticket.Query()

    // 自动添加租户过滤
    tenantID := s.getTenantID()
    if tenantID != "" {
        q = q.Where(ticket.TenantID(tenantID))
    }

    return &TenantTicketQuery{Query: q}
}

// 业务代码中使用
func ListTickets(ctx context.Context) ([]*Ticket, error) {
    // 不需要手动加 tenant_id 过滤
    return client.Ticket().List(ctx)
}
```

### 3.3 跨租户操作的权限控制

某些场景下需要跨租户操作，比如MSP管理员需要查看所有租户的数据。

这需要额外的权限控制。

```go
// 权限检查
func CheckPermission(ctx context.Context, action string, resourceID string) error {
    user := ctx.Value("user").(*User)

    // MSP管理员可以操作所有租户
    if user.Role == "msp_admin" {
        return nil
    }

    // 普通用户只能操作自己租户的资源
    tenant := ctx.Value("tenant").(*Tenant)
    if user.TenantID != tenant.ID {
        return ErrForbidden
    }

    return nil
}
```

---

## 第四部分：资源配额与计费

### 4.1 租户配额管理

MSP通常按套餐（Plan）向客户收费，不同套餐对应不同的资源配额。

```go
// 套餐定义
var Plans = map[string]*Plan{
    "free": {
        Name:       "免费版",
        MaxUsers:   5,
        MaxTickets: 100,
        MaxStorage: 1, // GB
        Features:   []string{"basic_tickets", "basic_kb"},
    },
    "standard": {
        Name:       "标准版",
        MaxUsers:   50,
        MaxTickets: 10000,
        MaxStorage: 50,
        Features:   []string{"full_tickets", "full_kb", "workflows", "sla"},
    },
    "enterprise": {
        Name:       "企业版",
        MaxUsers:   -1, // 无限制
        MaxTickets: -1,
        MaxStorage: -1,
        Features:   []string{"*"}, // 所有功能
    },
}

// 配额检查
func (s *QuotaService) CheckQuota(ctx context.Context, resource string, count int) error {
    tenant := ctx.Value("tenant").(*Tenant)
    plan := Plans[tenant.Plan]

    switch resource {
    case "users":
        if plan.MaxUsers > 0 {
            current := s.countUsers(ctx)
            if current+count > plan.MaxUsers {
                return ErrUserQuotaExceeded
            }
        }
    case "tickets":
        if plan.MaxTickets > 0 {
            current := s.countTickets(ctx)
            if current+count > plan.MaxTickets {
                return ErrTicketQuotaExceeded
            }
        }
    }

    return nil
}
```

### 4.2 资源使用统计

MSP需要了解每个租户的资源使用情况，用于计费和容量规划。

```go
// 租户资源使用统计
type TenantUsage struct {
    TenantID      string
    Plan         string
    UserCount    int
    TicketCount  int
    StorageUsedGB float64
    APICallCount  int
    Period        string // 月份
}

func (s *UsageService) GetTenantUsage(ctx context.Context, tenantID, period string) (*TenantUsage, error) {
    tenant, err := s.GetTenantByID(ctx, tenantID)
    if err != nil {
        return nil, err
    }

    return &TenantUsage{
        TenantID:      tenantID,
        Plan:          tenant.Plan,
        UserCount:     s.countUsersWithPeriod(ctx, tenantID, period),
        TicketCount:   s.countTicketsWithPeriod(ctx, tenantID, period),
        StorageUsedGB: s.calculateStorageUsed(ctx, tenantID),
        APICallCount:  s.countAPICalls(ctx, tenantID, period),
        Period:        period,
    }, nil
}
```

### 4.3 超配额处理

当租户接近或超过配额时，系统需要做出响应。

```go
// 配额警告级别
const (
    QuotaWarning  = 0.8  // 80%时警告
    QuotaCritical = 0.95 // 95%时限制
    QuotaExceeded = 1.0  // 100%时拒绝
)

// 检查并处理配额
func (s *QuotaService) EnforceQuota(ctx context.Context, tenantID, resource string) error {
    usage, err := s.GetTenantUsage(ctx, tenantID, currentPeriod())
    if err != nil {
        return err
    }

    plan := Plans[usage.Plan]
    var limit, used int

    switch resource {
    case "users":
        limit, used = plan.MaxUsers, usage.UserCount
    case "tickets":
        limit, used = plan.MaxTickets, usage.TicketCount
    case "storage":
        limit, used = plan.MaxStorage, int(usage.StorageUsedGB)
    }

    if limit <= 0 {
        return nil // 无限制
    }

    ratio := float64(used) / float64(limit)

    if ratio >= QuotaExceeded {
        return ErrQuotaExceeded
    }

    if ratio >= QuotaCritical {
        // 发送紧急通知
        s.notifyTenant(ctx, tenantID, "quota_critical", resource)
    } else if ratio >= QuotaWarning {
        // 发送警告通知
        s.notifyTenant(ctx, tenantID, "quota_warning", resource)
    }

    return nil
}
```

---

## 第五部分：个性化定制

### 5.1 租户设置

每个租户可以有自己的个性化设置。

```go
// 租户设置
type TenantSettings struct {
    // 界面定制
    Logo       string // Logo URL
    Theme      string // light | dark
    PrimaryColor string // 主色调

    // 工单设置
    TicketPrefix  string // 工单编号前缀
    DefaultPriority string
    RequireApproval bool

    // 通知设置
    EmailEnabled bool
    SMSEnabled   bool
    WebhookURL   string

    // 高级设置
    CustomFields []CustomField // 自定义字段
    WorkflowTemplate string // 工作流模板ID
}

// 自定义字段
type CustomField struct {
    Name     string      // 字段名
    Type     string      // string | number | date | select
    Required bool
    Options  []string    // 如果是select类型
}
```

### 5.2 自定义字段

不同租户可能需要不同的工单字段。

```go
// 工单模型扩展
type Ticket struct {
    // 标准字段
    ID          string
    Title       string
    Description string
    Status      string

    // 租户自定义字段 (JSON存储)
    CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
}

// 租户A的自定义字段示例
// {
//   "business_unit": "电商事业部",
//   "cost_center": "CC-2024-001",
//   "sla_tier": "P1"
// }

// 租户B的自定义字段示例
// {
//   "patient_id": "P12345",
//   "department": "内科",
//   "insurance_type": "职工医保"
// }
```

### 5.3 服务目录隔离

每个租户有自己的服务目录。

```go
// 租户服务目录
type ServiceCatalog struct {
    TenantID    string
    Items       []ServiceItem
    EnabledFeatures []string
    WorkflowTemplates []WorkflowTemplate
}
```

MSP可以为不同行业的客户配置不同的服务目录模板。比如医疗行业有"挂号系统故障"、"药房系统故障"等服务项，制造业有"MES系统问题"、"PLC设备故障"等服务项。

---

## 第六部分：运营实践

### 6.1 MSP管理后台

MSP运营商需要一套管理后台来管理所有租户。

```go
// MSP管理后台路由
func (s *MSPAdminRouter) Register(r *gin.RouterGroup) {
    // 租户管理
    r.GET("/tenants", s.ListTenants)
    r.POST("/tenants", s.CreateTenant)
    r.GET("/tenants/:id", s.GetTenant)
    r.PUT("/tenants/:id", s.UpdateTenant)
    r.DELETE("/tenants/:id", s.DeleteTenant)

    // 租户使用统计
    r.GET("/tenants/:id/usage", s.GetTenantUsage)
    r.GET("/tenants/:id/billing", s.GetTenantBilling)

    // 资源监控
    r.GET("/system/overview", s.GetSystemOverview)
    r.GET("/system/health", s.GetSystemHealth)
}
```

管理后台展示所有租户的概览：

- 租户总数、活跃租户数
- 总用户数、总工单数
- 本月新增租户趋势
- 资源使用TOP租户
- 系统健康状态

### 6.2 租户监控

MSP需要监控每个租户的使用情况和健康状态。

```go
// 租户健康指标
type TenantHealth struct {
    TenantID      string
    HealthScore   float64 // 0-100
    ActiveUsers   int     // 近7天活跃用户数
    TicketLoad    float64 // 工单负载
    ResponseTime  int     // 平均响应时间(ms)
    ErrorRate     float64 // 错误率
    LastActiveAt  time.Time
}

// 健康评分计算
func CalculateHealthScore(t *TenantHealth) float64 {
    // 权重配置
    weights := map[string]float64{
        "active_users": 0.2,
        "ticket_load":  0.2,
        "response_time": 0.3,
        "error_rate":    0.3,
    }

    // 评分计算
    score := 0.0
    score += (1 - t.ActiveUsers/100.0) * weights["active_users"] * 100
    score += (1 - t.TicketLoad/1000.0) * weights["ticket_load"] * 100
    score += (1 - t.ResponseTime/1000.0) * weights["response_time"] * 100
    score += (1 - t.ErrorRate) * weights["error_rate"] * 100

    return math.Max(0, math.Min(100, score))
}
```

### 6.3 典型案例

**案例背景**：

某MSP服务商"云运维公司"，为30家中型企业提供IT托管服务。这些客户分布在制造、零售、医疗三个行业，每个行业有自己的服务需求和合规要求。

**技术方案**：

- 部署一套ITSM系统服务所有租户
- 按行业分为三个"租户组"，每个组有独立的工单模板和服务目录
- 制造业租户：工单需要关联生产系统、合规要求严格
- 零售业租户：重视客服体验，需要微信通知集成
- 医疗行业租户：数据隔离要求最高，开启Schema级别隔离

**运营效果**：

- IT团队从维护30套系统减少到1套，节省60%运维成本
- 新增客户接入时间从2周缩短到2天
- 租户满意度从3.2分提升到4.5分（5分制）

---

## 第七部分：v1.1 多租户与 MSP 加固（2026-Q3）

前面的章节描述了多租户与 MSP 的设计意图，下面这部分说明 v1.1 实际落到代码里的加固点。开发者按文章对照仓库时，下表是真实可执行的对应关系。

### 7.1 信任模型与中间件分层

`middleware/tenant.go` 把租户识别收敛到三条来源（JWT claims.tenant_id / `X-Tenant-Code` / subdomain / path），每条来源都必须做 status / expires 校验；当 JWT 带 tenant_id 时，最终结果必须与之相等，否则拒绝。`X-Tenant-ID` 头被刻意忽略（历史修复确认），不要把它加回来。

`SwitchTenant` 在 `service/auth_service.go` 里拆成三条路径：

- **native**：用户从自己 home tenant 切到另一个自己有权访问的 tenant；
- **super_admin**：平台管理员的全局切换；
- **msp_cross**：MSP 员工凭 `msp_role` + 有效 `MSPAllocation` 切到客户租户，三路径互不干扰。

### 7.2 MSP 访问控制重构

`middleware/msp_middleware.go`（485 行）按职责拆成 4 个文件：

- `msp_gate.go`：`mspEnabled` 全局门 + `SetMSPEnabled` / `IsMSPEnabled` / `ApplyDeploymentMode`；
- `msp_context.go`：`MSPContext` 结构 + 上下文读写；
- `msp_rbac.go`：`RequireMSPAccess` / `RequireMSPManager` / `ValidateCustomerTenantHeader` / `RequireMSPPermission`；
- `msp_middleware.go`：保留 `MSPMiddleware(client)` 主流程 + `MSPFilterByCustomer` 查询过滤。

admin 角色不再自动获得 MSP 访问权；MSP 身份判定改为"租户类型为 msp_provider 且 msp_role 非空"。`GetMSPContextFromGin` 与 `GetMSPContext` 重复定义已去重。

### 7.3 部署模式 Gate

`middleware.ApplyDeploymentMode(mode)` 把 `DEPLOYMENT_MODE` env 映射到 `mspEnabled`：

- `private`：MSP 路由族（`/api/v1/msp/*`）整族返回 404，避免私有化部署意外暴露客户管理面；
- `saas` / `saas_msp` / 空 / 未知值：MSP 保持开启，typo 不会悄悄关掉 MSP。

`main.go` 启动时在 `NewApplication` 之前调用，避免 router 注册完后才生效。

### 7.4 MSP 客户校验

`service/MSPAccessValidator` 是 service 层的分配校验入口，提供三个方法：

- `ValidateCustomerAccess(mspUserID, customerTenantID)`：检查分配是否有效；
- `GetAllowedCustomerIDs(mspUserID)`：返回该 MSP 员工当前所有有效分配的客户租户；
- `FilterByMSPAllocation(mspUserID, tenantIDs)`：把候选租户 ID 收敛到分配列表内。

`middleware.ResolveRequestTenantID(c)` 把这一切打包成 controller 一行调用：返回有效租户 ID（普通用户 home tenant / MSP 用户锁定客户 / 未授权客户返回 `ErrMSPCustomerDenied`）。`incident_controller.go` / `cmdb_controller.go` / `knowledge_controller.go` 已经全部从 `middleware.GetTenantID` 切到本地 `resolveTenantID(ctx)` 包装，MSP 用户在普通工单/事件/CMDB/知识路由上的越权读被 fail-closed 拦住。

### 7.5 租户软删除与审计

`TenantService.DeleteTenant` 改为 `UpdateOneID status="deleted"`，租户记录保留但不再参与路由与分配校验。`tenant_controller.go` 五个写动作（创建 / 状态切换 / 更新 / 删除）现在打结构化审计日志，便于事后追溯。

### 7.6 跨租户枚举回归

`tests/rbac/cross_tenant_test.go` 现在覆盖四个域：

- `TestCrossTenantUserList`：admin 用户列表的 RBAC 边界（5 sub-tests）；
- `TestCrossTenantEnumeration`：user listing 的 status-code 同形（5 sub-tests）；
- `TestCrossTenantTicketEnumeration`：ticket 域（3 sub-tests）；
- `TestCrossTenantCIEnumeration`：CMDB 影响分析入口（3 sub-tests）；
- `TestCrossTenantKnowledgeEnumeration`：知识库 / RAG 高风险面（3 sub-tests）。

每域三个断言：同租户读得到 / 跨租户读不到 / 跨租户命中与不存在 ID 必须返回同一种 not-found，避免状态码 oracle。

### 7.7 待办（v1.5+）

- 配额与计费：当前 schema 有 plan / quota 字段但没有 enforcement 路径；
- 租户级 Redis keyspace：`tenant:{tid}:...` 命名空间约定尚需落地；
- 租户生命周期 event bus：创建 / 暂停 / 删除 / 配额变更目前没有事件流，CMDB discovery、connector 安装、SLA 模板这些下游无法自动反应；
- CMDB 影响分析 tenant 边界保护：递归遍历 CI 关系时如果 tenant filter 漏一处就是泄漏，需要专门的递归深度 + 租户边界测试。

## 总结

多租户架构是MSP服务商的核心技术能力。

ITSM开源项目的多租户架构实现了：

- **数据隔离**：通过租户ID自动过滤，保证数据安全
- **资源共享**：一套系统服务多个租户，降低成本
- **配额管理**：灵活的套餐和配额体系，支持多样化商业模式
- **个性化定制**：租户可以有自己Logo、字段、流程
- **统一运维**：MSP管理后台统一管理所有租户

**本系列其他文章**：

- 第1篇：《为什么你的IT团队每天像在救火？》
- 第2篇：《从混乱到有序：AI驱动的智能化工单分类实战》
- 第3篇：《变更管理不再是噩梦：BPMN工作流设计器详解》
- 第4篇：《让知识流动起来：RAG知识库搭建指南》
- 第6篇：《30分钟搭建一套完整的ITSM系统》

如果你是MSP服务商，正在寻找合适的多租户ITSM解决方案，欢迎评论区交流。
