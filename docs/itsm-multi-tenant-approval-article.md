# 从零构建企业级ITSM平台：多租户架构与审批工作流实战

最近花了不少时间在一个企业ITSM（IT服务管理）平台的建设上，从多租户体系到审批工作流引擎，整个系统逐渐有了雏形。趁着还有印象，把核心设计和实现细节整理一下，给有类似需求的朋友做个参考。

---

## 为什么需要多租户

做企业级SaaS，多租户是绕不开的基础设施。最直接的驱动力是：**同一个系统要服务多个客户，但数据必须严格隔离**。

我们有三种典型的客户场景：

- **MSP（Managed Service Provider）**：云服务提供商，需要管理多个下游客户
- **大型企业**：自己有IT团队，需要对部门、项目做精细化隔离
- **中小团队**：简单的工单管理，几个人共用

这三种场景对"租户"的理解不一样，所以设计时不能简单套用一个"tenant_id"字段了事。

## 多租户的数据模型设计

```go
// 租户实体核心字段
type Tenant struct {
    ID      int64
    Code    string  // 租户编码，用于URL或API路径
    Name    string  // 租户名称
    Type    string  // msp | enterprise | smb
    Status  string  // active | paused | trial
}
```

关键是**层级关系**：

```
MSP租户
  └── 客户租户A
  └── 客户租户B
      └── 部门（Department）
          └── 团队（Team）
              └── 用户（User）
```

数据隔离我们用了两种策略并行的方案：

1. **Schema隔离**（适合大客户）：每个租户独立的PostgreSQL Schema
2. **Row-level隔离**（适合普通客户）：所有数据通过`tenant_id`字段过滤

具体用哪种，由租户类型决定，在中间件层统一处理。

## 权限体系：RBAC的工程落地

单纯用"角色-权限"二维表不够用。企业场景下，权限判断往往还需要考虑：

- 谁在哪个租户下操作？
- 谁属于哪个部门/团队？
- 资源的所有者是谁？

```go
// 权限检查中间件伪代码
func RequirePermission(resource, action string) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 获取当前用户的完整权限上下文
        permCtx := BuildPermissionContext(c)
        
        // 2. 基础权限检查
        if !permCtx.HasPermission(resource, action) {
            c.JSON(403, Response{Code: 403, Message: "无权限"})
            c.Abort()
            return
        }
        
        // 3. 租户上下文检查（防止跨租户访问）
        if !permCtx.ValidateTenantScope(resource, c) {
            c.JSON(403, Response{Code: 403, Message: "无访问该租户权限"})
            c.Abort()
            return
        }
        
        c.Next()
    }
}
```

这个设计解决了一个常见坑：**用户A在租户X有权限，不代表在租户Y也有**。很多系统的权限漏洞就出在这里——只检查了角色，没检查租户归属。

## 审批工作流：从需求到实现

审批流程是ITSM的核心功能。我们设计的审批引擎支持：

### 多级审批链

```
工单提交 → 直属领导审批 → IT运维审批 → 技术总监审批 → 完成
           ↓ (拒绝)       ↓ (拒绝)       ↓ (拒绝)
         返回申请人    返回申请人     返回申请人
```

### 配置化的审批规则

审批工作流不是硬编码的，而是通过数据库配置：

```go
type ApprovalWorkflow struct {
    ID          int64
    Name        string
    TicketType  string  // service_request | incident | problem | change
    Priority    string  // low | medium | high | urgent
    Nodes       []WorkflowNode  // 审批节点配置
    IsActive    bool
}

type WorkflowNode struct {
    Level         int
    Name          string
    ApproverType  string  // user | role | department_head | dynamic
    ApproverIDs   []int64
    ApprovalMode  string  // one_of | all_of | cascade
}
```

前端可以拖拽配置审批链，后端实时解析执行。

### 审批记录追溯

每个审批动作都记录完整审计链：

```go
type ApprovalRecord struct {
    ID           int64
    WorkflowID   int64
    TicketID     int64
    ApproverID   int64
    Action       string  // approve | reject | delegate
    Comment      string
    DelegatedTo  *int64  // 委托审批时记录
    CreatedAt    time.Time
}
```

## 实际遇到的问题

### 1. 日期显示"Invalid Date"

工作流实例的截止时间字段从数据库返回时是时间戳，但前端直接传给`new Date()`导致解析失败。解决：

```tsx
// 前端统一处理时间显示
const formatDate = (date: string | number) => {
  if (!date) return '-';
  const d = new Date(date);
  return isNaN(d.getTime()) ? '-' : d.toLocaleString('zh-CN');
};
```

### 2. 路由与权限不匹配

菜单配置了`/admin/overview`，但对应的页面组件不存在，导致404。一开始菜单项和实际页面是脱节的，后来加了个简单的路由映射校验。

### 3. 硬编码的仪表盘数据

管理后台的"系统健康状态"、"许可证到期时间"都是写死的固定值，看起来像演示数据。现在改成了动态获取：

```tsx
const getCurrentDate = () => {
  return new Date().toLocaleString('zh-CN', { /* ... */ });
};
```

这类细节问题在开发期容易被忽略，测试时才发现。

## 技术栈选择

| 层级 | 技术 |
|------|------|
| 前端 | Next.js + TypeScript + Ant Design |
| 后端 | Go + Gin + Ent ORM |
| 数据库 | PostgreSQL |
| 缓存 | Redis |
| 工作流 | BPMN (lib-bpmn-engine) |

选Ent而不是GORM，主要看中它的**类型安全和代码生成**能力——Schema定义清楚后，CRUD代码自动生成，减少手误。

## 接下来的计划

- 集成AI自动分类工单（基于RAG的知识库匹配）
- SLA监控与告警
- 移动端适配

---

整体做下来，企业级ITSM没有太多花哨的东西，核心就是把**多租户隔离、权限控制、工作流编排**这几件事做扎实。数据模型设计好了，后面扩展起来才不会到处填坑。
