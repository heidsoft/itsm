# BPMN API 403错误排查技能

## 问题描述

前端调用工作流API (`/api/v1/bpmn/process-definitions`) 返回403 Forbidden错误。

## 排查步骤

### 1. 确认错误类型
- HTTP 403 = 权限不足（不是401认证失败）
- 错误发生在RBAC中间件层

### 2. 检查RBAC配置
文件: `middleware/rbac.go`

关键配置点：
- `ResourceActionMap`: HTTP方法与路径到资源的映射
- `RolePermissions`: 各角色对资源的权限定义

### 3. 问题根因
BPMN路由 `/api/v1/bpmn/*` 未在以下位置注册：
- `ResourceActionMap` 的 GET/POST/PUT/DELETE 映射
- 各角色的 `RolePermissions` 权限列表

## 解决方案

### 步骤1: 添加资源动作映射

在 `ResourceActionMap` 中添加:

```go
var ResourceActionMap = map[string]map[string]Permission{
    "GET": {
        // ...其他路由
        "/api/v1/bpmn/*": {Resource: "bpmn", Action: "read"},
    },
    "POST": {
        // ...其他路由
        "/api/v1/bpmn/*": {Resource: "bpmn", Action: "write"},
    },
    "PUT": {
        // ...其他路由
        "/api/v1/bpmn/*": {Resource: "bpmn", Action: "write"},
    },
    "DELETE": {
        // ...其他路由
        "/api/v1/bpmn/*": {Resource: "bpmn", Action: "delete"},
    },
}
```

### 步骤2: 添加角色权限

在 `RolePermissions` 中为各角色添加:

```go
var RolePermissions = map[string][]Permission{
    "admin": {
        // ...其他权限
        {Resource: "bpmn", Action: "read"},
        {Resource: "bpmn", Action: "write"},
        {Resource: "bpmn", Action: "delete"},
    },
    "manager": {
        // ...其他权限
        {Resource: "bpmn", Action: "read"},
        {Resource: "bpmn", Action: "write"},
    },
    "agent": {
        // ...其他权限
        {Resource: "bpmn", Action: "read"},
        {Resource: "bpmn", Action: "write"},
    },
    "technician": {
        // ...其他权限
        {Resource: "bpmn", Action: "read"},
        {Resource: "bpmn", Action: "write"},
    },
    "security": {
        // ...其他权限
        {Resource: "bpmn", Action: "read"},
        {Resource: "bpmn", Action: "write"},
    },
    "end_user": {
        // ...其他权限
        {Resource: "bpmn", Action: "read"}, // 只读
    },
}
```

## 排查公式

当遇到类似问题时，使用以下排查公式：

```
1. 确认请求路径和HTTP方法
2. 在 ResourceActionMap 中查找映射
3. 如果 nil → 需要添加路径映射
4. 获取请求资源的权限标识 (resource, action)
5. 检查 RolePermissions 中角色是否有该权限
6. 如果没有 → 添加权限配置
```

## 快速验证

```bash
# 重启后端服务
cd itsm-backend && go run main.go

# 测试API
curl -H "Authorization: Bearer <token>" \
     http://localhost:8080/api/v1/bpmn/process-definitions
```

## 相关文件

- `middleware/rbac.go` - RBAC权限配置
- `router/router.go` - 路由注册
- `controller/bpmn_workflow_controller.go` - BPMN控制器
