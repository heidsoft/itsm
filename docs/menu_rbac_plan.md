# 菜单权限管理系统设计方案

## 背景

当前系统存在以下问题：
1. 菜单硬编码在前端 `Sidebar.tsx`
2. 菜单权限字段定义了但未实际使用
3. 权限系统与菜单系统分离，无法统一管理
4. 无法通过后台配置菜单和权限

## 设计目标

1. 菜单数据存储到数据库，支持多租户
2. 菜单与权限绑定，根据用户角色显示菜单
3. 提供菜单管理后台（CRUD）
4. 提供用户菜单查询 API

---

## 数据模型设计

### 1. Menu 实体

```
┌─────────────────┐       ┌─────────────────┐
│      Menu       │       │   Permission   │
├─────────────────┤       ├─────────────────┤
│ id              │       │ id             │
│ name            │       │ code           │
│ path            │       │ resource       │
│ icon            │       │ action         │
│ parent_id (自关联)│ ←──→│ ...            │
│ permission_code │       └─────────────────┘
│ sort_order      │
│ tenant_id       │
│ is_visible      │
│ is_enabled      │
└─────────────────┘
```

### 2. 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int | 主键 |
| name | string | 菜单名称（显示用） |
| path | string | 路由路径 |
| icon | string | 图标名称 |
| parent_id | int? | 父菜单ID（支持多级菜单） |
| permission_code | string? | 绑定权限代码（如 `ticket:read`） |
| sort_order | int | 排序（越小越靠前） |
| tenant_id | int | 租户ID |
| is_visible | bool | 是否可见 |
| is_enabled | bool | 是否启用 |

---

## API 设计

### 1. 菜单管理 API

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/menus | 获取菜单列表 | menu:read |
| POST | /api/v1/menus | 创建菜单 | menu:write |
| PUT | /api/v1/menus/:id | 更新菜单 | menu:write |
| DELETE | /api/v1/menus/:id | 删除菜单 | menu:write |

### 2. 用户菜单 API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/auth/menus | 获取当前用户可见菜单 |

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "main": [
      {
        "id": 1,
        "name": "工单管理",
        "path": "/tickets",
        "icon": "FileText",
        "children": [
          {"id": 2, "name": "工单列表", "path": "/tickets"}
        ]
      }
    ],
    "admin": [
      {"id": 10, "name": "用户管理", "path": "/admin/users"}
    ]
  }
}
```

---

## 实施计划

### 阶段一：后端数据库和API

| 序号 | 任务 | 文件 |
|------|------|------|
| 1.1 | 创建 Menu 实体 schema | ent/schema/menu.go |
| 1.2 | 生成 Menu CRUD 代码 | ent 生成 |
| 1.3 | 创建 Menu DTO | dto/menu_dto.go |
| 1.4 | 创建 Menu Service | service/menu_service.go |
| 1.5 | 创建 Menu Controller | controller/menu_controller.go |
| 1.6 | 注册菜单路由 | router/router.go |
| 1.7 | 添加菜单初始化种子数据 | pkg/seeder/seeder.go |

### 阶段二：用户菜单API

| 序号 | 任务 | 文件 |
|------|------|------|
| 2.1 | 修改登录返回用户权限 | service/auth_service.go |
| 2.2 | 添加获取用户菜单API | service/menu_service.go |

### 阶段三：前端集成

| 序号 | 任务 | 文件 |
|------|------|------|
| 3.1 | 添加菜单API客户端 | src/lib/api/menu-api.ts |
| 3.2 | 修改 AuthStore 缓存菜单 | src/lib/store/auth-store.ts |
| 3.3 | 修改 Sidebar 使用动态菜单 | src/components/layout/Sidebar.tsx |

### 阶段四：种子数据

| 序号 | 任务 | 说明 |
|------|------|------|
| 4.1 | 初始化默认菜单 | 将现有硬编码菜单导入数据库 |

---

## 菜单权限映射表

| 菜单路径 | 权限代码 | 说明 |
|----------|----------|------|
| /dashboard | dashboard:view | 仪表盘 |
| /tickets | ticket:read | 工单列表 |
| /incidents | incident:read | 事件管理 |
| /problems | problem:read | 问题管理 |
| /changes | change:read | 变更管理 |
| /cmdb | cmdb:read | CMDB |
| /knowledge | knowledge:read | 知识库 |
| /service-catalog | service:read | 服务目录 |
| /sla-dashboard | sla:read | SLA监控 |
| /reports | report:read | 报表 |
| /releases | release:read | 发布管理 |
| /msp | msp:read | MSP管理 |
| /workflow | workflow:view | 工作流 |
| /admin/users | user:read | 用户管理 |
| /admin/roles | role:read | 角色管理 |
| /admin/groups | group:read | 组管理 |

---

## 向后兼容

1. **旧版登录用户**：首次登录时调用菜单API获取菜单
2. **无菜单数据**：fallback 到前端硬编码菜单
3. **超级管理员**：默认拥有所有菜单权限
