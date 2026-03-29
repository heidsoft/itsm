# ITSM 业务流程测试计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to execute this plan task-by-task.

**Goal:** 测试 ITSM 核心模块的增删改查（CRUD）功能，验证用户体验

**Architecture:** 通过 API 测试和前端手动测试结合，验证系统的完整业务流程

**Tech Stack:** curl/HTTP API, 前端界面测试

---

## 测试前提

**后端服务:** http://localhost:8090
**前端服务:** http://localhost:3000
**测试账号:** admin / admin123 / tenant: default
**认证 Token:**
```
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJ1c2VybmFtZSI6ImFkbWluIiwicm9sZSI6InN1cGVyX2FkbWluIiwidGVuYW50X2lkIjoxLCJ0b2tlbl90eXBlIjoiYWNjZXNzIiwiZXhwIjoxNzc0NzAyNjU4LCJpYXQiOjE3NzQ3MDE3NTh9.fRiP2ZL2_sFxk08DcK__DOqE74o5MHTLtXFVvnQK1M4
```

---

## Task 1: 用户认证流程测试

**Files:**
- 后端 API: `/api/v1/auth/login`, `/api/v1/auth/refresh`
- 前端: `itsm-frontend/src/app/(auth)/login/`

- [ ] **Step 1: 测试登录 API**

```bash
curl -s -X POST http://localhost:8090/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123","tenant_code":"default"}'
```
预期: `{"code":0,"message":"success","data":{...access_token...}}`

- [ ] **Step 2: 测试 Token 刷新**

```bash
# 从登录响应获取 refresh_token
curl -s -X POST http://localhost:8090/api/v1/refresh-token \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"<refresh_token>"}'
```
预期: 返回新的 access_token

- [ ] **Step 3: 测试错误登录**

```bash
curl -s -X POST http://localhost:8090/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"wrongpassword","tenant_code":"default"}'
```
预期: `{"code":2001,"message":"认证失败"}`

- [ ] **Step 4: 前端登录界面测试**
- 打开 http://localhost:3000/login
- 输入正确凭据，验证跳转到 dashboard
- 输入错误凭据，验证错误提示

---

## Task 2: 工单（Ticket）增删改查测试

**Files:**
- 后端 API: `/api/v1/tickets`
- 前端: `itsm-frontend/src/app/(main)/tickets/`

- [ ] **Step 1: 查询工单列表**

```bash
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
curl -s "http://localhost:8090/api/v1/tickets?page=1&page_size=10" \
  -H "Authorization: Bearer $TOKEN"
```
预期: 返回工单列表，包含分页信息

- [ ] **Step 2: 创建工单（Create）**

```bash
curl -s -X POST http://localhost:8090/api/v1/tickets \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $(curl -s http://localhost:8090/api/v1/csrf-token | grep -o 'csrf_token":"[^"]*' | cut -d'"' -f4)" \
  -d '{
    "title": "API测试工单 - 20260328",
    "description": "这是一个通过API创建的测试工单",
    "priority": "medium",
    "category_id": 1,
    "requester_id": 1
  }'
```
预期: `{"code":0,"message":"success","data":{"id":...}}`

- [ ] **Step 3: 查询单个工单（Read）**

```bash
# 将 <ticket_id> 替换为上一步创建的工单ID
curl -s "http://localhost:8090/api/v1/tickets/<ticket_id>" \
  -H "Authorization: Bearer $TOKEN"
```
预期: 返回工单详情

- [ ] **Step 4: 更新工单（Update）**

```bash
curl -s -X PUT "http://localhost:8090/api/v1/tickets/<ticket_id>" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: <csrf_token>" \
  -d '{
    "title": "API测试工单 - 已更新",
    "status": "in_progress"
  }'
```
预期: `{"code":0,"message":"success"}`

- [ ] **Step 5: 删除工单（Delete）**

```bash
curl -s -X DELETE "http://localhost:8090/api/v1/tickets/<ticket_id>" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-CSRF-Token: <csrf_token>"
```
预期: `{"code":0,"message":"success"}`

- [ ] **Step 6: 前端工单页面测试**
- 打开 http://localhost:3000/tickets
- 验证工单列表显示
- 点击创建工单，填写表单，提交
- 点击工单查看详情
- 编辑工单
- 删除工单

---

## Task 3: 用户管理增删改查测试

**Files:**
- 后端 API: `/api/v1/users`
- 前端: `itsm-frontend/src/app/(main)/admin/users/`

- [ ] **Step 1: 查询用户列表**

```bash
curl -s "http://localhost:8090/api/v1/users?page=1&page_size=10" \
  -H "Authorization: Bearer $TOKEN"
```
预期: 返回用户列表

- [ ] **Step 2: 创建用户（Create）**

```bash
curl -s -X POST http://localhost:8090/api/v1/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: <csrf_token>" \
  -d '{
    "username": "testuser_$(date +%s)",
    "password": "Test123456",
    "email": "test@example.com",
    "name": "测试用户",
    "role": "agent",
    "department": "IT部门"
  }'
```
预期: `{"code":0,"message":"success","data":{"id":...}}`

- [ ] **Step 3: 查询单个用户**

```bash
curl -s "http://localhost:8090/api/v1/users/<user_id>" \
  -H "Authorization: Bearer $TOKEN"
```
预期: 返回用户详情

- [ ] **Step 4: 更新用户**

```bash
curl -s -X PUT "http://localhost:8090/api/v1/users/<user_id>" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: <csrf_token>" \
  -d '{"name": "测试用户 - 已更新"}'
```
预期: `{"code":0,"message":"success"}`

- [ ] **Step 5: 删除用户**

```bash
curl -s -X DELETE "http://localhost:8090/api/v1/users/<user_id>" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-CSRF-Token: <csrf_token>"
```
预期: `{"code":0,"message":"success"}`

- [ ] **Step 6: 前端用户管理页面测试**
- 打开 http://localhost:3000/admin/users
- 验证用户列表
- 创建新用户
- 编辑用户
- 删除用户

---

## Task 4: 配置项（CI）增删改查测试

**Files:**
- 后端 API: `/api/v1/configuration-items`
- 前端: `itsm-frontend/src/app/(main)/cmdb/cis/`

- [ ] **Step 1: 查询配置项列表**

```bash
curl -s "http://localhost:8090/api/v1/configuration-items?page=1&page_size=10" \
  -H "Authorization: Bearer $TOKEN"
```
预期: 返回 CI 列表

- [ ] **Step 2: 创建配置项**

```bash
curl -s -X POST http://localhost:8090/api/v1/configuration-items \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: <csrf_token>" \
  -d '{
    "name": "测试服务器 - API",
    "type": "server",
    "status": "active"
  }'
```
预期: `{"code":0,"message":"success","data":{"id":...}}`

- [ ] **Step 3: 查询单个配置项**

```bash
curl -s "http://localhost:8090/api/v1/configuration-items/<ci_id>" \
  -H "Authorization: Bearer $TOKEN"
```
预期: 返回 CI 详情

- [ ] **Step 4: 更新配置项**

```bash
curl -s -X PUT "http://localhost:8090/api/v1/configuration-items/<ci_id>" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: <csrf_token>" \
  -d '{"name": "测试服务器 - 已更新"}'
```
预期: `{"code":0,"message":"success"}`

- [ ] **Step 5: 删除配置项**

```bash
curl -s -X DELETE "http://localhost:8090/api/v1/configuration-items/<ci_id>" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-CSRF-Token: <csrf_token>"
```
预期: `{"code":0,"message":"success"}`

- [ ] **Step 6: 前端 CMDB 页面测试**
- 打开 http://localhost:3000/cmdb/cis
- 验证 CI 列表
- 创建新 CI
- 查看 CI 详情和拓扑图
- 编辑 CI
- 删除 CI

---

## Task 5: 知识库（Knowledge）增删改查测试

**Files:**
- 后端 API: `/api/v1/knowledge-articles`
- 前端: `itsm-frontend/src/app/(main)/knowledge/`

- [ ] **Step 1: 查询知识库文章列表**

```bash
curl -s "http://localhost:8090/api/v1/knowledge-articles?page=1&page_size=10" \
  -H "Authorization: Bearer $TOKEN"
```
预期: 返回文章列表

- [ ] **Step 2: 创建知识库文章**

```bash
curl -s -X POST http://localhost:8090/api/v1/knowledge-articles \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: <csrf_token>" \
  -d '{
    "title": "API测试文章 - 20260328",
    "content": "# 测试文章\n\n这是一篇通过API创建的测试文章。",
    "category_id": 1,
    "status": "published"
  }'
```
预期: `{"code":0,"message":"success","data":{"id":...}}`

- [ ] **Step 3: 查询单个文章**

```bash
curl -s "http://localhost:8090/api/v1/knowledge-articles/<article_id>" \
  -H "Authorization: Bearer $TOKEN"
```
预期: 返回文章详情

- [ ] **Step 4: 更新文章**

```bash
curl -s -X PUT "http://localhost:8090/api/v1/knowledge-articles/<article_id>" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: <csrf_token>" \
  -d '{"title": "API测试文章 - 已更新"}'
```
预期: `{"code":0,"message":"success"}`

- [ ] **Step 5: 删除文章**

```bash
curl -s -X DELETE "http://localhost:8090/api/v1/knowledge-articles/<article_id>" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-CSRF-Token: <csrf_token>"
```
预期: `{"code":0,"message":"success"}`

- [ ] **Step 6: 前端知识库页面测试**
- 打开 http://localhost:3000/knowledge
- 验证文章列表和搜索
- 创建新文章
- 查看文章详情
- 编辑文章
- 删除文章

---

## Task 6: 服务请求（Service Request）增删改查测试

**Files:**
- 后端 API: `/api/v1/service-requests`
- 前端: `itsm-frontend/src/app/(main)/service-requests/`

- [ ] **Step 1: 查询服务请求列表**

```bash
curl -s "http://localhost:8090/api/v1/service-requests?page=1&page_size=10" \
  -H "Authorization: Bearer $TOKEN"
```
预期: 返回请求列表

- [ ] **Step 2: 创建服务请求**

```bash
curl -s -X POST http://localhost:8090/api/v1/service-requests \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: <csrf_token>" \
  -d '{
    "title": "API测试请求 - 20260328",
    "description": "这是一个通过API创建的测试请求",
    "service_catalog_id": 1,
    "priority": "medium"
  }'
```
预期: `{"code":0,"message":"success","data":{"id":...}}`

- [ ] **Step 3: 查询单个请求**

```bash
curl -s "http://localhost:8090/api/v1/service-requests/<request_id>" \
  -H "Authorization: Bearer $TOKEN"
```
预期: 返回请求详情

- [ ] **Step 4: 更新请求**

```bash
curl -s -X PUT "http://localhost:8090/api/v1/service-requests/<request_id>" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: <csrf_token>" \
  -d '{"status": "in_progress"}'
```
预期: `{"code":0,"message":"success"}`

- [ ] **Step 5: 删除请求**

```bash
curl -s -X DELETE "http://localhost:8090/api/v1/service-requests/<request_id>" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-CSRF-Token: <csrf_token>"
```
预期: `{"code":0,"message":"success"}`

- [ ] **Step 6: 前端服务请求页面测试**
- 打开 http://localhost:3000/service-requests
- 验证请求列表
- 创建新请求
- 查看请求详情
- 更新请求状态
- 删除请求

---

## Task 7: SLA 定义增删改查测试

**Files:**
- 后端 API: `/api/v1/sla-definitions`
- 前端: `itsm-frontend/src/app/(main)/admin/sla-definitions/`

- [ ] **Step 1: 查询 SLA 列表**

```bash
curl -s "http://localhost:8090/api/v1/sla-definitions?page=1&page_size=10" \
  -H "Authorization: Bearer $TOKEN"
```
预期: 返回 SLA 列表

- [ ] **Step 2: 创建 SLA 定义**

```bash
curl -s -X POST http://localhost:8090/api/v1/sla-definitions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: <csrf_token>" \
  -d '{
    "name": "API测试SLA",
    "description": "测试SLA",
    "service_type": "incident",
    "priority": "high",
    "response_time": 30,
    "resolution_time": 240
  }'
```
预期: `{"code":0,"message":"success","data":{"id":...}}`

- [ ] **Step 3: 查询单个 SLA**

```bash
curl -s "http://localhost:8090/api/v1/sla-definitions/<sla_id>" \
  -H "Authorization: Bearer $TOKEN"
```
预期: 返回 SLA 详情

- [ ] **Step 4: 更新 SLA**

```bash
curl -s -X PUT "http://localhost:8090/api/v1/sla-definitions/<sla_id>" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: <csrf_token>" \
  -d '{"name": "API测试SLA - 已更新"}'
```
预期: `{"code":0,"message":"success"}`

- [ ] **Step 5: 删除 SLA**

```bash
curl -s -X DELETE "http://localhost:8090/api/v1/sla-definitions/<sla_id>" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-CSRF-Token: <csrf_token>"
```
预期: `{"code":0,"message":"success"}`

- [ ] **Step 6: 前端 SLA 管理页面测试**
- 打开 http://localhost:3000/admin/sla-definitions
- 验证 SLA 列表
- 创建新 SLA
- 编辑 SLA
- 删除 SLA

---

## 测试结果记录模板

```markdown
## 测试结果

### 用户认证
- [ ] 登录: PASS/FAIL - 备注
- [ ] Token刷新: PASS/FAIL - 备注
- [ ] 错误登录: PASS/FAIL - 备注
- [ ] 前端登录: PASS/FAIL - 备注

### 工单管理
- [ ] 查询列表: PASS/FAIL - 备注
- [ ] 创建工单: PASS/FAIL - 备注
- [ ] 查询详情: PASS/FAIL - 备注
- [ ] 更新工单: PASS/FAIL - 备注
- [ ] 删除工单: PASS/FAIL - 备注
- [ ] 前端工单页面: PASS/FAIL - 备注

[... 其他模块 ...]
```

---

## 执行方式

**Plan complete and saved to `docs/superpowers/plans/2026-03-28-itsm-crud-testing.md`**

**Two execution options:**

**1. Subagent-Driven（推荐）** - 每个模块分配一个 subagent 测试，快速迭代

**2. Inline Execution** - 在当前 session 中按批次执行测试，有检查点

**选择哪个方式?**
