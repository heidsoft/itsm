# ITSM 前后端API对接测试文档

## 测试概述

本文档用于逐个模块测试前端到后端的API对接情况。

## 测试模块列表

### 1. Dashboard模块
- **前端API**: `itsm-prototype/src/lib/api/dashboard-api.ts`
- **后端路由**: `/api/v1/dashboard/*`
- **测试端点**:
  - `GET /api/v1/dashboard/overview` - 获取概览数据
  - `GET /api/v1/dashboard/kpi-metrics` - 获取KPI指标
  - `GET /api/v1/dashboard/ticket-trend` - 获取工单趋势
  - `GET /api/v1/dashboard/incident-distribution` - 获取事件分布

### 2. Tickets模块
- **前端API**: `itsm-prototype/src/lib/api/ticket-api.ts`
- **后端路由**: `/api/v1/tickets/*`
- **测试端点**:
  - `GET /api/v1/tickets` - 获取工单列表（支持分页和筛选）
  - `GET /api/v1/tickets/:id` - 获取工单详情
  - `POST /api/v1/tickets` - 创建工单
  - `PUT /api/v1/tickets/:id` - 更新工单
  - `POST /api/v1/tickets/:id/assign` - 分配工单
  - `POST /api/v1/tickets/:id/resolve` - 解决工单
  - `POST /api/v1/tickets/:id/close` - 关闭工单
  - `GET /api/v1/tickets/stats` - 获取工单统计

### 3. Incidents模块
- **前端API**: `itsm-prototype/src/lib/api/incident-api.ts`
- **后端路由**: `/api/v1/incidents/*`
- **测试端点**:
  - `GET /api/v1/incidents` - 获取事件列表（支持分页和筛选）
  - `GET /api/v1/incidents/:id` - 获取事件详情
  - `POST /api/v1/incidents` - 创建事件
  - `PUT /api/v1/incidents/:id` - 更新事件
  - `GET /api/v1/incidents/stats` - 获取事件统计

### 4. Workflow模块
- **前端API**: `itsm-prototype/src/lib/api/workflow-api.ts`
- **后端路由**: `/api/v1/bpmn/*`
- **测试端点**:
  - `GET /api/v1/bpmn/process-definitions` - 获取流程定义列表
  - `GET /api/v1/bpmn/process-definitions/:key` - 获取流程定义详情
  - `POST /api/v1/bpmn/process-definitions` - 创建流程定义
  - `PUT /api/v1/bpmn/process-definitions/:key` - 更新流程定义
  - `DELETE /api/v1/bpmn/process-definitions/:key` - 删除流程定义
  - `GET /api/v1/bpmn/process-instances` - 获取流程实例列表
  - `POST /api/v1/bpmn/process-instances` - 启动流程实例

### 5. Enterprise Management模块

#### 5.1 Departments (部门管理)
- **前端页面**: `itsm-prototype/src/app/(main)/enterprise/departments/page.tsx`
- **后端路由**: `/api/v1/departments/*`
- **测试端点**:
  - `GET /api/v1/departments/tree` - 获取部门树
  - `POST /api/v1/departments` - 创建部门
  - `PUT /api/v1/departments/:id` - 更新部门
  - `DELETE /api/v1/departments/:id` - 删除部门

#### 5.2 Projects (项目管理)
- **前端页面**: `itsm-prototype/src/app/(main)/enterprise/projects/page.tsx`
- **后端路由**: `/api/v1/projects/*`
- **测试端点**:
  - `GET /api/v1/projects` - 获取项目列表
  - `POST /api/v1/projects` - 创建项目
  - `PUT /api/v1/projects/:id` - 更新项目
  - `DELETE /api/v1/projects/:id` - 删除项目

#### 5.3 Applications (应用管理)
- **前端页面**: `itsm-prototype/src/app/(main)/enterprise/applications/page.tsx`
- **后端路由**: `/api/v1/applications/*`
- **测试端点**:
  - `GET /api/v1/applications` - 获取应用列表
  - `POST /api/v1/applications` - 创建应用
  - `PUT /api/v1/applications/:id` - 更新应用
  - `DELETE /api/v1/applications/:id` - 删除应用
  - `GET /api/v1/applications/microservices` - 获取微服务列表
  - `POST /api/v1/applications/microservices` - 创建微服务
  - `PUT /api/v1/applications/microservices/:id` - 更新微服务
  - `DELETE /api/v1/applications/microservices/:id` - 删除微服务

#### 5.4 Teams (团队管理)
- **前端页面**: `itsm-prototype/src/app/(main)/enterprise/teams/page.tsx`
- **后端路由**: `/api/v1/teams/*`
- **测试端点**:
  - `GET /api/v1/teams` - 获取团队列表
  - `POST /api/v1/teams` - 创建团队
  - `PUT /api/v1/teams/:id` - 更新团队
  - `DELETE /api/v1/teams/:id` - 删除团队
  - `POST /api/v1/teams/members` - 添加团队成员

#### 5.5 Tags (标签管理)
- **前端页面**: `itsm-prototype/src/app/(main)/enterprise/tags/page.tsx`
- **后端路由**: `/api/v1/tags/*`
- **测试端点**:
  - `GET /api/v1/tags` - 获取标签列表
  - `POST /api/v1/tags` - 创建标签
  - `PUT /api/v1/tags/:id` - 更新标签
  - `DELETE /api/v1/tags/:id` - 删除标签
  - `POST /api/v1/tags/bind` - 绑定标签到实体

## 运行测试

### 前置条件
1. 确保后端服务运行在 `http://localhost:8090`
2. 确保数据库已初始化
3. 确保有测试用户（默认: admin/password）

### 运行测试脚本
```bash
# 设置环境变量（可选）
export API_BASE_URL=http://localhost:8090
export AUTH_TOKEN=your_token_here

# 运行测试
./test-api-integration.sh
```

### 手动测试

#### 1. 登录获取Token
```bash
curl -X POST http://localhost:8090/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}'
```

#### 2. 测试Dashboard API
```bash
TOKEN="your_token_here"

curl -X GET http://localhost:8090/api/v1/dashboard/overview \
  -H "Authorization: Bearer $TOKEN"
```

#### 3. 测试企业管理API
```bash
# 获取部门树
curl -X GET http://localhost:8090/api/v1/departments/tree \
  -H "Authorization: Bearer $TOKEN"

# 获取项目列表
curl -X GET http://localhost:8090/api/v1/projects \
  -H "Authorization: Bearer $TOKEN"
```

## 常见问题

### 1. 401 Unauthorized
- 检查Token是否有效
- 检查Token是否在请求头中正确设置

### 2. 404 Not Found
- 检查路由路径是否正确
- 检查后端路由是否已注册

### 3. 500 Internal Server Error
- 检查后端日志
- 检查数据库连接
- 检查数据格式是否正确

## 测试检查清单

- [ ] Dashboard API 响应格式正确
- [ ] Tickets API 支持分页和筛选
- [ ] Incidents API 支持分页和筛选
- [ ] Workflow API 与前端workflow-api.ts对接
- [ ] Departments API CRUD操作正常
- [ ] Projects API CRUD操作正常
- [ ] Applications API CRUD操作正常
- [ ] Teams API CRUD操作正常
- [ ] Tags API CRUD操作正常
- [ ] 所有API返回格式统一（code, message, data）
- [ ] 错误处理正确
- [ ] 认证和授权正常

