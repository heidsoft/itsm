# RBAC 权限系统与多租户/MSP 审查报告

## 📊 执行摘要

本报告对 ITSM 系统的 RBAC 权限系统、多租户架构和 MSP（托管服务提供商）功能进行了全面审查。

### 审查结论

**整体评级：⭐⭐⭐⭐ (4/5)**

系统已实现完整的 RBAC 权限控制、多租户隔离和 MSP 跨租户访问功能，架构设计合理，但存在一些需要改进的地方。

---

## 🎯 核心发现

### 优势

1. ✅ **完整的 RBAC 实现**
   - 基于角色的权限控制
   - 支持数据库动态权限配置
   - 权限缓存机制
   - 资源级别权限检查

2. ✅ **多租户架构完善**
   - 租户隔离中间件
   - 租户状态和过期检查
   - 支持子域名和头部识别

3. ✅ **MSP 功能完整**
   - MSP 员工分配管理
   - 跨租户访问控制
   - 客户级别权限验证
   - MSP 角色映射

### 待改进项

1. ⚠️ **权限配置混乱**
   - 硬编码权限和数据库权限混用
   - 缺少统一的权限管理界面
   - 权限缓存失效机制不完善

2. ⚠️ **数据隔离不彻底**
   - Service 层缺少自动租户过滤
   - 部分查询未强制租户隔离
   - 跨租户数据泄露风险

3. ⚠️ **MSP 权限模型复杂**
   - MSP 角色和 RBAC 角色映射不清晰
   - 客户级别权限检查分散
   - 缺少 MSP 审计日志

4. ⚠️ **缺少细粒度权限**
   - 字段级别权限控制缺失
   - 数据行级别权限不完整
   - 操作级别权限粒度粗

---

## 📋 详细审查

### 1. RBAC 权限系统

#### 1.1 权限模型

**当前实现**：
```go
type Permission struct {
    Resource string // 资源名称：ticket, user, dashboard
    Action   string // 操作类型：read, write, delete, admin
}
```

**角色定义**：
- super_admin：超级管理员（所有权限）
- admin：管理员
- manager：经理
- agent：客服
- technician：技术员
- security：安全审批
- end_user：最终用户
- MSP 角色：msp_admin, msp_manager, msp_tech, msp_specialist, msp_viewer

**问题**：
1. 权限粒度过粗，只有资源+操作两级
2. 缺少字段级别权限（如只能查看部分字段）
3. 缺少条件权限（如只能查看自己部门的数据）

**建议**：
```go
type Permission struct {
    Resource   string   // 资源名称
    Action     string   // 操作类型
    Scope      string   // 权限范围：own, department, tenant, all
    Conditions []string // 条件表达式
    Fields     []string // 允许访问的字段
}
```

#### 1.2 权限配置方式

**当前实现**：
- 硬编码权限映射（`RolePermissions` map）
- 数据库动态权限（`Role` 和 `Permission` 表）
- 混合使用：优先数据库，回退到硬编码

**问题**：
1. 两套权限系统并存，容易混淆
2. 权限更新需要重启服务（硬编码部分）
3. 缺少权限版本管理

**建议**：
- 统一使用数据库权限配置
- 保留硬编码作为默认权限模板
- 实现权限热更新机制
- 添加权限变更审计日志

#### 1.3 权限检查流程

**当前实现**：
```go
// 1. RBACMiddleware：全局权限检查
// 2. RequirePermission：特定资源权限检查
// 3. RequireRole：角色检查
// 4. 资源级别权限检查（如工单所有权）
```

**优势**：
- 多层次权限检查
- 支持资源级别权限
- 性能优化（缓存）

**问题**：
1. 权限检查逻辑分散在多个地方
2. 缺少统一的权限决策点
3. 难以追踪权限拒绝原因

**建议**：
```go
// 统一权限决策服务
type PermissionDecisionService struct {
    client *ent.Client
    cache  *PermissionCache
    logger *zap.SugaredLogger
}

type PermissionRequest struct {
    UserID    int
    TenantID  int
    Resource  string
    Action    string
    ResourceID *int
    Context   map[string]interface{}
}

type PermissionDecision struct {
    Allowed bool
    Reason  string
    Scope   string
}

func (s *PermissionDecisionService) Decide(ctx context.Context, req *PermissionRequest) (*PermissionDecision, error) {
    // 1. 检查用户状态
    // 2. 检查角色权限
    // 3. 检查资源级别权限
    // 4. 检查条件权限
    // 5. 记录决策日志
    // 6. 返回决策结果
}
```

---

### 2. 多租户架构

#### 2.1 租户隔离机制

**当前实现**：
```go
// TenantMiddleware：租户识别和验证
// - 从头部 X-Tenant-Code 获取
// - 从子域名提取
// - 从 JWT token 获取
// - 从路径参数获取
```

**优势**：
- 多种租户识别方式
- 租户状态和过期检查
- 租户上下文传递

**问题**：
1. Service 层未强制租户过滤
2. 部分查询可能绕过租户隔离
3. 缺少租户级别的资源配额限制

**建议**：
```go
// 1. 在 Ent Client 层面强制租户过滤
type TenantAwareClient struct {
    *ent.Client
    tenantID int
}

func (c *TenantAwareClient) Ticket() *ent.TicketClient {
    return c.Client.Ticket.Query().Where(ticket.TenantIDEQ(c.tenantID))
}

// 2. 添加租户资源配额检查
type TenantQuotaService struct {
    client *ent.Client
}

func (s *TenantQuotaService) CheckQuota(ctx context.Context, tenantID int, resource string) error {
    // 检查租户是否超出资源配额
}

// 3. 租户级别审计日志
type TenantAuditService struct {
    client *ent.Client
}

func (s *TenantAuditService) LogAccess(ctx context.Context, tenantID int, action string) {
    // 记录租户访问日志
}
```

#### 2.2 数据隔离问题

**发现的问题**：

1. **Service 层缺少自动租户过滤**
   ```go
   // 当前实现：需要手动添加租户过滤
   tickets, err := s.client.Ticket.Query().
       Where(ticket.TenantIDEQ(tenantID)). // 容易遗漏
       All(ctx)
   ```

2. **关联查询可能泄露数据**
   ```go
   // 危险：可能查询到其他租户的数据
   ticket, err := s.client.Ticket.Query().
       Where(ticket.IDEQ(ticketID)).
       WithAssignee(). // 未过滤租户
       Only(ctx)
   ```

3. **批量操作缺少租户验证**
   ```go
   // 危险：可能操作其他租户的数据
   _, err := s.client.Ticket.Update().
       Where(ticket.IDIn(ticketIDs...)).
       SetStatus("closed").
       Save(ctx)
   ```

**建议修复**：
```go
// 1. 创建租户感知的查询构建器
type TenantAwareQuery struct {
    tenantID int
}

func (q *TenantAwareQuery) Ticket() *ent.TicketQuery {
    return client.Ticket.Query().Where(ticket.TenantIDEQ(q.tenantID))
}

// 2. 在 Service 层强制租户检查
func (s *TicketService) GetTicket(ctx context.Context, ticketID int, tenantID int) (*ent.Ticket, error) {
    ticket, err := s.client.Ticket.Query().
        Where(
            ticket.IDEQ(ticketID),
            ticket.TenantIDEQ(tenantID), // 强制租户过滤
        ).
        Only(ctx)
    if err != nil {
        return nil, err
    }
    return ticket, nil
}

// 3. 批量操作前验证所有资源属于当前租户
func (s *TicketService) BatchUpdate(ctx context.Context, ticketIDs []int, tenantID int, updates map[string]interface{}) error {
    // 先验证所有工单属于当前租户
    count, err := s.client.Ticket.Query().
        Where(
            ticket.IDIn(ticketIDs...),
            ticket.TenantIDEQ(tenantID),
        ).
        Count(ctx)
    if err != nil {
        return err
    }
    if count != len(ticketIDs) {
        return fmt.Errorf("部分工单不属于当前租户")
    }
    
    // 执行批量更新
    // ...
}
```

---

### 3. MSP 功能

#### 3.1 MSP 架构设计

**当前实现**：
- MSP 租户类型：`type = "msp"`
- 客户租户类型：`type = "customer"`
- MSP 分配表：`msp_allocations`
- MSP 中间件：`MSPMiddleware`

**优势**：
- 清晰的 MSP 和客户租户区分
- 灵活的分配管理
- 支持多个 MSP 员工服务同一客户

**问题**：
1. MSP 角色和 RBAC 角色映射复杂
2. 客户级别权限检查分散
3. 缺少 MSP 操作审计

#### 3.2 MSP 权限模型

**当前实现**：
```go
// MSP 角色
type MSPRole string
const (
    ProviderAdmin = "provider_admin"  // MSP 管理员
    ProviderAgent = "provider_agent"  // MSP 客服
    CustomerUser  = "customer_user"   // 客户用户
)

// MSP 角色到 RBAC 角色映射
var mspRoleToRBACRoleMap = map[string]string{
    "provider_admin":  "msp_manager",
    "provider_agent": "msp_tech",
    "customer_user":  "end_user",
}
```

**问题**：
1. 映射关系不够清晰
2. MSP 角色权限定义分散
3. 缺少 MSP 特定权限（如跨租户查询）

**建议**：
```go
// 统一 MSP 权限模型
type MSPPermission struct {
    Role              string   // MSP 角色
    AllowedResources  []string // 允许访问的资源
    AllowedActions    []string // 允许的操作
    CrossTenantAccess bool     // 是否允许跨租户访问
    CustomerScope     string   // 客户范围：assigned, all
}

var MSPPermissions = map[string]MSPPermission{
    "msp_admin": {
        Role:              "msp_admin",
        AllowedResources:  []string{"*"},
        AllowedActions:    []string{"*"},
        CrossTenantAccess: true,
        CustomerScope:     "all",
    },
    "msp_manager": {
        Role:              "msp_manager",
        AllowedResources:  []string{"ticket", "customer", "allocation", "report"},
        AllowedActions:    []string{"read", "write"},
        CrossTenantAccess: true,
        CustomerScope:     "all",
    },
    "msp_tech": {
        Role:              "msp_tech",
        AllowedResources:  []string{"ticket", "customer"},
        AllowedActions:    []string{"read", "write"},
        CrossTenantAccess: true,
        CustomerScope:     "assigned",
    },
}
```

#### 3.3 MSP 数据访问控制

**当前实现**：
```go
// MSPMiddleware：验证 MSP 员工是否有权访问客户
// - 检查 msp_allocations 表
// - 验证 X-Customer-Tenant-ID 头
// - 设置 MSP 上下文
```

**问题**：
1. 每次请求都查询数据库（性能问题）
2. 缺少 MSP 访问日志
3. 客户级别权限检查不够细粒度

**建议**：
```go
// 1. MSP 分配缓存
type MSPAllocationCache struct {
    cache *redis.Client
}

func (c *MSPAllocationCache) GetAllocations(mspUserID int) ([]int, error) {
    // 从 Redis 获取 MSP 员工的客户列表
    // 缓存 5 分钟
}

// 2. MSP 访问审计
type MSPAuditService struct {
    client *ent.Client
}

func (s *MSPAuditService) LogAccess(ctx context.Context, req *MSPAccessRequest) {
    // 记录 MSP 员工访问客户数据的日志
    s.client.MSPAccessLog.Create().
        SetMSPUserID(req.MSPUserID).
        SetCustomerTenantID(req.CustomerTenantID).
        SetResource(req.Resource).
        SetAction(req.Action).
        SetAccessTime(time.Now()).
        Save(ctx)
}

// 3. 细粒度客户权限
type MSPCustomerPermission struct {
    MSPUserID        int
    CustomerTenantID int
    AllowedResources []string
    AllowedActions   []string
    ExpiresAt        time.Time
}
```

---

## 🔥 高优先级改进建议

### P0：数据隔离加强

**问题**：Service 层缺少自动租户过滤，存在数据泄露风险

**解决方案**：
1. 在 Ent Client 层面实现租户感知查询
2. Service 层强制租户参数传递
3. 添加租户隔离测试用例

**实施步骤**：
```go
// Step 1: 创建租户感知的 Client 包装器
type TenantClient struct {
    *ent.Client
    tenantID int
}

func NewTenantClient(client *ent.Client, tenantID int) *TenantClient {
    return &TenantClient{
        Client:   client,
        tenantID: tenantID,
    }
}

// Step 2: 重写查询方法，自动添加租户过滤
func (c *TenantClient) Ticket() *TenantTicketClient {
    return &TenantTicketClient{
        TicketClient: c.Client.Ticket,
        tenantID:     c.tenantID,
    }
}

type TenantTicketClient struct {
    *ent.TicketClient
    tenantID int
}

func (c *TenantTicketClient) Query() *ent.TicketQuery {
    return c.TicketClient.Query().Where(ticket.TenantIDEQ(c.tenantID))
}

// Step 3: Service 层使用租户感知 Client
type TicketService struct {
    getTenantClient func(tenantID int) *TenantClient
}

func (s *TicketService) GetTicket(ctx context.Context, ticketID int, tenantID int) (*ent.Ticket, error) {
    client := s.getTenantClient(tenantID)
    return client.Ticket().Query().
        Where(ticket.IDEQ(ticketID)).
        Only(ctx)
}
```

### P0：统一权限管理

**问题**：硬编码权限和数据库权限混用，管理混乱

**解决方案**：
1. 统一使用数据库权限配置
2. 提供权限管理界面
3. 实现权限热更新

**实施步骤**：
```go
// Step 1: 权限同步服务
type PermissionSyncService struct {
    client *ent.Client
    logger *zap.SugaredLogger
}

func (s *PermissionSyncService) SyncDefaultPermissions(ctx context.Context) error {
    // 将硬编码的 RolePermissions 同步到数据库
    for roleName, perms := range RolePermissions {
        role, err := s.client.Role.Query().
            Where(role.Code(roleName)).
            First(ctx)
        if err != nil {
            // 创建角色
            role, err = s.client.Role.Create().
                SetCode(roleName).
                SetName(roleName).
                Save(ctx)
        }
        
        // 同步权限
        for _, perm := range perms {
            s.client.Permission.Create().
                SetResource(perm.Resource).
                SetAction(perm.Action).
                AddRoles(role).
                Save(ctx)
        }
    }
    return nil
}

// Step 2: 权限热更新服务
type PermissionHotReloadService struct {
    client *ent.Client
    cache  *PermissionCache
}

func (s *PermissionHotReloadService) ReloadPermissions(ctx context.Context) error {
    // 清空缓存
    InvalidateAllPermissionCaches()
    
    // 重新加载权限
    // ...
    
    return nil
}

// Step 3: 权限管理 API
type PermissionController struct {
    permissionService *PermissionService
}

func (c *PermissionController) UpdateRolePermissions(ctx *gin.Context) {
    // 更新角色权限
    // 触发热更新
    // 记录审计日志
}
```

### P1：MSP 审计日志

**问题**：缺少 MSP 操作审计，无法追踪跨租户访问

**解决方案**：
```go
// MSP 审计日志表
type MSPAccessLog struct {
    ID               int
    MSPUserID        int
    MSPUsername      string
    CustomerTenantID int
    CustomerName     string
    Resource         string
    Action           string
    ResourceID       *int
    IPAddress        string
    UserAgent        string
    AccessTime       time.Time
    Success          bool
    ErrorMessage     string
}

// MSP 审计中间件
func MSPAuditMiddleware(client *ent.Client) gin.HandlerFunc {
    return func(c *gin.Context) {
        mspCtx, exists := GetMSPContext(c)
        if !exists || !mspCtx.IsMSP {
            c.Next()
            return
        }
        
        // 记录访问日志
        log := &MSPAccessLog{
            MSPUserID:        mspCtx.MSPUserID,
            CustomerTenantID: *mspCtx.CustomerTenantID,
            Resource:         c.Request.URL.Path,
            Action:           c.Request.Method,
            IPAddress:        c.ClientIP(),
            UserAgent:        c.Request.UserAgent(),
            AccessTime:       time.Now(),
        }
        
        c.Next()
        
        log.Success = c.Writer.Status() < 400
        // 保存日志
        saveAccessLog(client, log)
    }
}
```

### P1：细粒度权限控制

**问题**：权限粒度过粗，缺少字段级别和条件权限

**解决方案**：
```go
// 扩展权限模型
type Permission struct {
    Resource   string
    Action     string
    Scope      string   // own, department, tenant, all
    Conditions []string // 条件表达式
    Fields     []string // 允许访问的字段
}

// 权限决策引擎
type PermissionEngine struct {
    client *ent.Client
}

func (e *PermissionEngine) CheckFieldPermission(userID int, resource string, field string) bool {
    // 检查用户是否有权访问特定字段
}

func (e *PermissionEngine) FilterFields(userID int, resource string, data map[string]interface{}) map[string]interface{} {
    // 根据权限过滤字段
}

// 使用示例
func (s *TicketService) GetTicket(ctx context.Context, ticketID int, userID int) (map[string]interface{}, error) {
    ticket, err := s.client.Ticket.Get(ctx, ticketID)
    if err != nil {
        return nil, err
    }
    
    // 转换为 map
    data := ticketToMap(ticket)
    
    // 根据权限过滤字段
    filtered := s.permissionEngine.FilterFields(userID, "ticket", data)
    
    return filtered, nil
}
```

---

## 📊 改进优先级矩阵

| 改进项 | 优先级 | 影响范围 | 实施难度 | 预计工期 |
|--------|--------|---------|---------|---------|
| 数据隔离加强 | P0 | 高 | 中 | 2周 |
| 统一权限管理 | P0 | 高 | 中 | 2周 |
| MSP 审计日志 | P1 | 中 | 低 | 1周 |
| 细粒度权限控制 | P1 | 中 | 高 | 3周 |
| 权限热更新 | P1 | 中 | 中 | 1周 |
| 租户资源配额 | P2 | 低 | 中 | 2周 |
| MSP 分配缓存 | P2 | 低 | 低 | 1周 |

---

## ✅ 总结

当前 ITSM 系统的 RBAC 权限系统、多租户架构和 MSP 功能已经实现了基本功能，但在数据隔离、权限管理和审计方面还有改进空间。

建议按照优先级逐步实施改进，预计 2-3 个月可以达到企业级安全标准。
