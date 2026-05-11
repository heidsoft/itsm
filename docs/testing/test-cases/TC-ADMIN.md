# 系统管理模块测试用例

- **模块**: 系统管理 (System Administration)
- **版本**: v1.0
- **最后更新**: 2026-05-10
- **总计**: 76 个测试用例

---

## 目录

1. [用户管理](#1-用户管理)
2. [角色管理](#2-角色管理)
3. [权限配置](#3-权限配置)
4. [部门管理](#4-部门管理)
5. [团队管理](#5-团队管理)
6. [租户管理](#6-租户管理)
7. [系统配置](#7-系统配置)
8. [操作日志](#8-操作日志)
9. [审批链管理](#9-审批链管理)
10. [工单分类](#10-工单分类)
11. [导入模板管理](#11-导入模板管理)

---

## 1. 用户管理

| ID | 测试用例 | 优先级 | 测试类型 | 操作步骤 | 预期结果 |
|----|----------|--------|----------|----------|----------|
| UM-001 | 用户列表-正常分页 | P1 | 功能 | GET /api/v1/users?page=1&page_size=10 | 返回用户列表，分页信息正确 |
| UM-002 | 用户列表-指定每页数量 | P1 | 功能 | GET /api/v1/users?page=1&page_size=20 | 返回20条用户数据 |
| UM-003 | 用户列表-按状态筛选 | P2 | 功能 | GET /api/v1/users?status=active | 仅返回激活状态的用户 |
| UM-004 | 用户列表-按部门筛选 | P2 | 功能 | GET /api/v1/users?department=技术部 | 仅返回技术部用户 |
| UM-005 | 用户列表-关键词搜索 | P2 | 功能 | GET /api/v1/users?search=张三 | 返回包含"张三"的用户 |
| UM-006 | 用户列表-组合筛选 | P2 | 功能 | GET /api/v1/users?status=active&department=技术部 | 返回技术部激活用户 |
| UM-007 | 创建用户-必填字段完整 | P1 | 功能 | POST /api/v1/users {username, email, name, password, tenant_id} | 返回201和创建的用户信息 |
| UM-008 | 创建用户-缺少用户名 | P2 | 边界 | POST /api/v1/users {email, name, password, tenant_id} | 返回400，提示username为必填 |
| UM-009 | 创建用户-邮箱格式错误 | P2 | 边界 | POST /api/v1/users {username: "testuser", email: "invalid", name: "Test", password: "123456", tenant_id: 1} | 返回400，提示邮箱格式错误 |
| UM-010 | 创建用户-密码过短 | P2 | 边界 | POST /api/v1/users {username: "testuser", email: "test@example.com", name: "Test", password: "123", tenant_id: 1} | 返回400，提示密码至少6位 |
| UM-011 | 创建用户-用户名重复 | P2 | 边界 | POST /api/v1/users两次相同username | 返回400，提示用户已存在 |
| UM-012 | 创建用户-邮箱重复 | P2 | 边界 | POST /api/v1/users两次相同email | 返回400，提示邮箱已存在 |
| UM-013 | 创建用户-指定角色 | P1 | 功能 | POST /api/v1/users {username, email, name, password, tenant_id, role: "admin"} | 用户创建时分配admin角色 |
| UM-014 | 获取用户详情-正常 | P1 | 功能 | GET /api/v1/users/{id} | 返回该用户完整信息 |
| UM-015 | 获取用户详情-用户不存在 | P2 | 边界 | GET /api/v1/users/99999 | 返回404，提示用户不存在 |
| UM-016 | 编辑用户-更新基本信息 | P1 | 功能 | PUT /api/v1/users/{id} {name: "新姓名"} | 返回更新后的用户信息 |
| UM-017 | 编辑用户-更新邮箱 | P1 | 功能 | PUT /api/v1/users/{id} {email: "new@example.com"} | 邮箱更新成功 |
| UM-018 | 编辑用户-更新角色 | P1 | 功能 | PUT /api/v1/users/{id} {role: "manager"} | 角色更新成功 |
| UM-019 | 编辑用户-无权限更新他租户用户 | P2 | 安全 | PUT /api/v1/users/{id} (跨租户) | 返回403或404 |
| UM-020 | 删除用户-正常删除 | P1 | 功能 | DELETE /api/v1/users/{id} | 返回200，软删除成功 |
| UM-021 | 删除用户-删除已删除用户 | P2 | 边界 | DELETE /api/v1/users/{id} (已删除) | 返回404，提示用户不存在 |
| UM-022 | 禁用用户-正常禁用 | P1 | 功能 | PUT /api/v1/users/{id}/status {active: false} | 用户被禁用，active=false |
| UM-023 | 禁用用户-激活已禁用用户 | P1 | 功能 | PUT /api/v1/users/{id}/status {active: true} | 用户被激活，active=true |
| UM-024 | 禁用用户-无法禁用自己 | P2 | 业务规则 | PUT /api/v1/users/{current_user_id}/status {active: false} | 返回400，不能禁用自己 |
| UM-025 | 重置密码-正常重置 | P1 | 功能 | PUT /api/v1/users/{id}/reset-password {new_password: "newpass123"} | 密码重置成功 |
| UM-026 | 重置密码-新密码过短 | P2 | 边界 | PUT /api/v1/users/{id}/reset-password {new_password: "123"} | 返回400，提示密码至少6位 |
| UM-027 | 批量启用用户 | P1 | 功能 | PUT /api/v1/users/batch {user_ids: [1,2,3], action: "activate"} | 多个用户被激活 |
| UM-028 | 批量禁用用户 | P1 | 功能 | PUT /api/v1/users/batch {user_ids: [1,2,3], action: "deactivate"} | 多个用户被禁用 |
| UM-029 | 批量更新部门 | P1 | 功能 | PUT /api/v1/users/batch {user_ids: [1,2], action: "department", department: "运营部"} | 用户转移到运营部 |
| UM-030 | 搜索用户-正常搜索 | P1 | 功能 | GET /api/v1/users/search?keyword=张三&limit=10 | 返回匹配的用户列表 |
| UM-031 | 搜索用户-空关键词 | P2 | 边界 | GET /api/v1/users/search?keyword= | 返回400，关键词不能为空 |
| UM-032 | 用户统计-获取统计信息 | P2 | 功能 | GET /api/v1/users/stats | 返回用户统计数据 |
| UM-033 | 导出用户-正常导出 | P2 | 功能 | GET /api/v1/users/export | 返回用户数据文件 |
| UM-034 | 导入用户-正常导入 | P1 | 功能 | POST /api/v1/users/import {users: [...]} | 返回导入结果统计 |
| UM-035 | 导入用户-部分成功 | P2 | 边界 | POST /api/v1/users/import {users: [valid, invalid, valid]} | 返回成功和失败明细 |
| UM-036 | 导入用户-数据格式错误 | P2 | 边界 | POST /api/v1/users/import {users: [incomplete_data]} | 返回400，提示数据格式错误 |

---

## 2. 角色管理

| ID | 测试用例 | 优先级 | 测试类型 | 操作步骤 | 预期结果 |
|----|----------|--------|----------|----------|----------|
| RM-001 | 角色列表-正常分页 | P1 | 功能 | GET /api/v1/roles?page=1&page_size=20 | 返回角色列表，分页正确 |
| RM-002 | 角色列表-按状态筛选 | P2 | 功能 | GET /api/v1/roles?status=active | 仅返回激活状态的角色 |
| RM-003 | 创建角色-必填字段完整 | P1 | 功能 | POST /api/v1/roles {name: "运维管理员", description: "负责运维"} | 返回创建的角色信息 |
| RM-004 | 创建角色-缺少名称 | P2 | 边界 | POST /api/v1/roles {description: "test"} | 返回400，name为必填 |
| RM-005 | 创建角色-名称重复 | P2 | 边界 | POST /api/v1/roles两次相同name | 返回400，提示角色名已存在 |
| RM-006 | 创建角色-自动生成编码 | P2 | 功能 | POST /api/v1/roles {name: "新角色"} 不提供code | 自动生成唯一code |
| RM-007 | 获取角色详情-正常 | P1 | 功能 | GET /api/v1/roles/{id} | 返回角色完整信息，包含权限列表 |
| RM-008 | 获取角色详情-角色不存在 | P2 | 边界 | GET /api/v1/roles/99999 | 返回404 |
| RM-009 | 编辑角色-更新基本信息 | P1 | 功能 | PUT /api/v1/roles/{id} {name: "新名称"} | 返回更新后的角色信息 |
| RM-010 | 编辑角色-更新描述 | P1 | 功能 | PUT /api/v1/roles/{id} {description: "新描述"} | 描述更新成功 |
| RM-011 | 编辑系统内置角色 | P2 | 安全 | PUT /api/v1/roles/{id} (is_system=true) | 返回400，系统角色不可编辑 |
| RM-012 | 删除角色-正常删除 | P1 | 功能 | DELETE /api/v1/roles/{id} | 返回200，删除成功 |
| RM-013 | 删除角色-角色被用户使用 | P2 | 业务规则 | DELETE /api/v1/roles/{id} (有用户分配) | 返回400，提示角色正在使用 |
| RM-014 | 删除系统内置角色 | P2 | 安全 | DELETE /api/v1/roles/{id} (is_system=true) | 返回400，系统角色不可删除 |
| RM-015 | 分配权限-正常分配 | P1 | 功能 | PUT /api/v1/roles/{id}/permissions {permission_ids: [1,2,3]} | 权限分配成功 |
| RM-016 | 分配权限-分配不存在的权限 | P2 | 边界 | PUT /api/v1/roles/{id}/permissions {permission_ids: [999]} | 返回400，权限不存在 |
| RM-017 | 分配权限-空权限列表 | P2 | 边界 | PUT /api/v1/roles/{id}/permissions {permission_ids: []} | 清空角色所有权限 |
| RM-018 | 角色复制-正常复制 | P1 | 功能 | POST /api/v1/roles/{id}/clone {name: "新角色名称"} | 创建拥有相同权限的新角色 |
| RM-019 | 角色复制-不提供名称 | P2 | 边界 | POST /api/v1/roles/{id}/clone {} | 返回400，name为必填 |

---

## 3. 权限配置

| ID | 测试用例 | 优先级 | 测试类型 | 操作步骤 | 预期结果 |
|----|----------|--------|----------|----------|----------|
| PM-001 | 权限列表-正常查询 | P1 | 功能 | GET /api/v1/permissions | 返回权限列表 |
| PM-002 | 权限列表-按资源筛选 | P2 | 功能 | GET /api/v1/permissions?resource=ticket | 返回资源为ticket的权限 |
| PM-003 | 创建权限-必填字段完整 | P1 | 功能 | POST /api/v1/permissions {code: "ticket:create", name: "创建工单", resource: "ticket", action: "create"} | 返回创建的权限信息 |
| PM-004 | 创建权限-缺少权限码 | P2 | 边界 | POST /api/v1/permissions {name: "测试", resource: "ticket", action: "create"} | 返回400，code为必填 |
| PM-005 | 创建权限-重复权限码 | P2 | 边界 | POST /api/v1/permissions两次相同code | 返回400，权限码已存在 |
| PM-006 | 初始化默认权限 | P1 | 功能 | POST /api/v1/permissions/init | 创建系统默认权限集 |
| PM-007 | 权限验证-用户有所需权限 | P2 | 功能 | 验证用户拥有ticket:create权限 | 返回true |
| PM-008 | 权限验证-用户无所需权限 | P2 | 功能 | 验证用户没有admin:delete权限 | 返回false |
| PM-009 | 获取权限组列表 | P2 | 功能 | GET /api/v1/permission-groups | 返回按组分类的权限列表 |
| PM-010 | 创建权限组 | P2 | 功能 | POST /api/v1/permission-groups {name: "工单权限组", code: "ticket_group"} | 返回创建的权限组 |

---

## 4. 部门管理

| ID | 测试用例 | 优先级 | 测试类型 | 操作步骤 | 预期结果 |
|----|----------|--------|----------|----------|----------|
| DM-001 | 部门树-正常查询 | P1 | 功能 | GET /api/v1/departments/tree | 返回树形结构部门列表 |
| DM-002 | 部门树-空数据 | P2 | 边界 | GET /api/v1/departments/tree (无部门) | 返回空数组 |
| DM-003 | 创建部门-必填字段完整 | P1 | 功能 | POST /api/v1/departments {name: "技术部", code: "TECH"} | 返回创建的部门信息 |
| DM-004 | 创建部门-缺少名称 | P2 | 边界 | POST /api/v1/departments {code: "TECH"} | 返回400，name为必填 |
| DM-005 | 创建部门-缺少编码 | P2 | 边界 | POST /api/v1/departments {name: "技术部"} | 返回400，code为必填 |
| DM-006 | 创建子部门 | P1 | 功能 | POST /api/v1/departments {name: "前端组", code: "FE", parent_id: 1} | 在父部门下创建子部门 |
| DM-007 | 获取部门详情-正常 | P1 | 功能 | GET /api/v1/departments/{id} | 返回部门完整信息 |
| DM-008 | 编辑部门-更新基本信息 | P1 | 功能 | PUT /api/v1/departments/{id} {name: "新名称"} | 返回更新后的部门信息 |
| DM-009 | 编辑部门-调整父部门 | P1 | 功能 | PUT /api/v1/departments/{id} {parent_id: 2} | 部门移动到新父部门下 |
| DM-010 | 编辑部门-设置为顶级部门 | P2 | 功能 | PUT /api/v1/departments/{id} {parent_id: null} | 部门变为顶级部门 |
| DM-011 | 删除部门-正常删除 | P1 | 功能 | DELETE /api/v1/departments/{id} | 返回200，删除成功 |
| DM-012 | 删除部门-有子部门 | P2 | 业务规则 | DELETE /api/v1/departments/{id} (有子部门) | 返回400，提示存在子部门 |
| DM-013 | 删除部门-有用户 | P2 | 业务规则 | DELETE /api/v1/departments/{id} (有用户) | 返回400，提示部门下有用户 |
| DM-014 | 设置部门负责人 | P1 | 功能 | PUT /api/v1/departments/{id} {manager_id: 5} | 指定用户为部门负责人 |
| DM-015 | 获取部门成员 | P2 | 功能 | GET /api/v1/departments/{id}/members | 返回部门下的用户列表 |

---

## 5. 团队管理

| ID | 测试用例 | 优先级 | 测试类型 | 操作步骤 | 预期结果 |
|----|----------|--------|----------|----------|----------|
| TM-001 | 团队列表-正常查询 | P1 | 功能 | GET /api/v1/teams | 返回团队列表 |
| TM-002 | 创建团队-必填字段完整 | P1 | 功能 | POST /api/v1/teams {name: "故障处理组", code: "INCIDENT"} | 返回创建的团队信息 |
| TM-003 | 创建团队-缺少名称 | P2 | 边界 | POST /api/v1/teams {code: "INCIDENT"} | 返回400，name为必填 |
| TM-004 | 创建团队-缺少编码 | P2 | 边界 | POST /api/v1/teams {name: "故障处理组"} | 返回400，code为必填 |
| TM-005 | 创建团队-编码重复 | P2 | 边界 | POST /api/v1/teams两次相同code | 返回400，编码已存在 |
| TM-006 | 获取团队详情-正常 | P1 | 功能 | GET /api/v1/teams/{id} | 返回团队完整信息 |
| TM-007 | 编辑团队-更新基本信息 | P1 | 功能 | PUT /api/v1/teams/{id} {name: "新名称"} | 返回更新后的团队信息 |
| TM-008 | 编辑团队-设置负责人 | P1 | 功能 | PUT /api/v1/teams/{id} {manager_id: 3} | 团队负责人更新成功 |
| TM-009 | 删除团队-正常删除 | P1 | 功能 | DELETE /api/v1/teams/{id} | 返回200，删除成功 |
| TM-010 | 删除团队-有成员 | P2 | 业务规则 | DELETE /api/v1/teams/{id} (有成员) | 返回400，先移除成员 |
| TM-011 | 添加团队成员 | P1 | 功能 | POST /api/v1/teams/{id}/members {user_id: 5} | 成员添加成功 |
| TM-012 | 添加团队成员-重复添加 | P2 | 边界 | POST /api/v1/teams/{id}/members (已存在成员) | 返回400，已是成员 |
| TM-013 | 移除团队成员 | P1 | 功能 | DELETE /api/v1/teams/{id}/members/{user_id} | 成员移除成功 |
| TM-014 | 获取团队成员列表 | P1 | 功能 | GET /api/v1/teams/{id}/members | 返回团队成员列表 |
| TM-015 | 获取团队统计 | P2 | 功能 | GET /api/v1/teams/{id}/stats | 返回团队工作负载统计 |

---

## 6. 租户管理

| ID | 测试用例 | 优先级 | 测试类型 | 操作步骤 | 预期结果 |
|----|----------|--------|----------|----------|----------|
| TN-001 | 租户列表-正常分页 | P1 | 功能 | GET /api/admin/tenants?page=1&page_size=10 | 返回租户列表 |
| TN-002 | 租户列表-按状态筛选 | P2 | 功能 | GET /api/admin/tenants?status=active | 仅返回激活状态租户 |
| TN-003 | 租户列表-按类型筛选 | P2 | 功能 | GET /api/admin/tenants?type=enterprise | 仅返回企业版租户 |
| TN-004 | 租户列表-关键词搜索 | P2 | 功能 | GET /api/admin/tenants?search=公司名 | 返回匹配的租户 |
| TN-005 | 创建租户-必填字段完整 | P1 | 功能 | POST /api/admin/tenants {name: "测试公司", code: "TEST", type: "trial"} | 返回创建的租户信息 |
| TN-006 | 创建租户-缺少名称 | P2 | 边界 | POST /api/admin/tenants {code: "TEST", type: "trial"} | 返回400，name为必填 |
| TN-007 | 创建租户-缺少编码 | P2 | 边界 | POST /api/admin/tenants {name: "测试公司", type: "trial"} | 返回400，code为必填 |
| TN-008 | 创建租户-编码包含特殊字符 | P2 | 边界 | POST /api/admin/tenants {name: "测试公司", code: "TEST-01", type: "trial"} | 返回400，编码只能包含字母数字 |
| TN-009 | 创建租户-设置过期时间 | P1 | 功能 | POST /api/admin/tenants {... , expires_at: "2027-01-01T00:00:00Z"} | 租户创建时设置过期时间 |
| TN-010 | 创建租户-设置资源配额 | P2 | 功能 | POST /api/admin/tenants {... , quota: {users: 100, tickets: 1000}} | 租户创建时设置配额 |
| TN-011 | 获取租户详情-正常 | P1 | 功能 | GET /api/admin/tenants/{id} | 返回租户完整信息 |
| TN-012 | 获取租户详情-租户不存在 | P2 | 边界 | GET /api/admin/tenants/99999 | 返回404 |
| TN-013 | 编辑租户-更新基本信息 | P1 | 功能 | PUT /api/admin/tenants/{id} {name: "新公司名"} | 返回更新后的租户信息 |
| TN-014 | 编辑租户-升级类型 | P1 | 功能 | PUT /api/admin/tenants/{id} {type: "enterprise"} | 租户类型升级成功 |
| TN-015 | 更新租户状态-激活 | P1 | 功能 | PUT /api/admin/tenants/{id}/status {status: "active"} | 租户被激活 |
| TN-016 | 更新租户状态-禁用 | P1 | 功能 | PUT /api/admin/tenants/{id}/status {status: "suspended"} | 租户被禁用 |
| TN-017 | 更新租户状态-删除 | P2 | 功能 | PUT /api/admin/tenants/{id}/status {status: "deleted"} | 租户标记为已删除 |
| TN-018 | 删除租户-正常删除 | P1 | 功能 | DELETE /api/admin/tenants/{id} | 返回200，租户删除 |
| TN-019 | 删除租户-有活跃用户 | P2 | 业务规则 | DELETE /api/admin/tenants/{id} (有活跃用户) | 返回400，提示先处理用户 |
| TN-020 | 配置租户套餐 | P1 | 功能 | PUT /api/admin/tenants/{id} {type: "professional", quota: {...}} | 租户套餐配置成功 |

---

## 7. 系统配置

| ID | 测试用例 | 优先级 | 测试类型 | 操作步骤 | 预期结果 |
|----|----------|--------|----------|----------|----------|
| SC-001 | 配置列表-正常分页 | P1 | 功能 | GET /api/v1/system-configs?page=1&page_size=20 | 返回配置列表 |
| SC-002 | 配置列表-按分类筛选 | P1 | 功能 | GET /api/v1/system-configs?category=email | 仅返回邮件相关配置 |
| SC-003 | 获取单个配置-正常 | P1 | 功能 | GET /api/v1/system-configs/{id} | 返回配置详情 |
| SC-004 | 根据Key获取配置 | P1 | 功能 | GET /api/v1/system-configs/key/{key} | 通过key返回配置 |
| SC-005 | 根据Key获取不存在的配置 | P2 | 边界 | GET /api/v1/system-configs/key/invalid_key | 返回404 |
| SC-006 | 更新配置-正常更新 | P1 | 功能 | PUT /api/v1/system-configs/{id} {value: "newvalue"} | 配置值更新成功 |
| SC-007 | 更新配置-更新描述 | P2 | 功能 | PUT /api/v1/system-configs/{id} {description: "新描述"} | 配置描述更新成功 |
| SC-008 | 批量更新配置 | P1 | 功能 | PUT /api/v1/system-configs/batch [{id: 1, value: "v1"}, {id: 2, value: "v2"}] | 多个配置同时更新 |
| SC-009 | 初始化默认配置 | P1 | 功能 | POST /api/v1/system-configs/init | 创建系统默认配置 |
| SC-010 | 配置邮件服务器 | P1 | 功能 | PUT /api/v1/system-configs/{id} {key: "smtp_host", value: "smtp.example.com"} | 邮件服务器配置成功 |
| SC-011 | 配置SMS服务 | P2 | 功能 | PUT /api/v1/system-configs/{id} {key: "sms_provider", value: "aliyun"} | SMS服务商配置成功 |
| SC-012 | 配置通知设置 | P2 | 功能 | PUT /api/v1/system-configs/{id} {key: "notify_email", value: "true"} | 通知配置更新成功 |
| SC-013 | 配置安全设置 | P2 | 功能 | PUT /api/v1/system-configs/{id} {key: "password_min_length", value: "8"} | 安全策略配置更新 |
| SC-014 | 配置缓存设置 | P2 | 功能 | PUT /api/v1/system-configs/{id} {key: "cache_ttl", value: "3600"} | 缓存配置更新成功 |

---

## 8. 操作日志

| ID | 测试用例 | 优先级 | 测试类型 | 操作步骤 | 预期结果 |
|----|----------|--------|----------|----------|----------|
| AL-001 | 日志列表-正常分页 | P1 | 功能 | GET /api/v1/audit-logs?page=1&page_size=20 | 返回审计日志列表 |
| AL-002 | 日志列表-按用户筛选 | P2 | 功能 | GET /api/v1/audit-logs?user_id=5 | 返回该用户的操作日志 |
| AL-003 | 日志列表-按资源类型筛选 | P2 | 功能 | GET /api/v1/audit-logs?resource=user | 返回用户相关操作日志 |
| AL-004 | 日志列表-按操作类型筛选 | P2 | 功能 | GET /api/v1/audit-logs?action=create | 返回创建操作日志 |
| AL-005 | 日志列表-按HTTP方法筛选 | P2 | 功能 | GET /api/v1/audit-logs?method=POST | 返回POST请求日志 |
| AL-006 | 日志列表-按状态码筛选 | P2 | 功能 | GET /api/v1/audit-logs?status_code=200 | 返回成功请求日志 |
| AL-007 | 日志列表-按时间范围筛选 | P2 | 功能 | GET /api/v1/audit-logs?from=2026-05-01T00:00:00Z&to=2026-05-10T23:59:59Z | 返回指定时间范围日志 |
| AL-008 | 日志列表-按路径筛选 | P2 | 功能 | GET /api/v1/audit-logs?path=/api/v1/users | 返回该路径操作日志 |
| AL-009 | 日志列表-按请求ID筛选 | P2 | 功能 | GET /api/v1/audit-logs?request_id=abc123 | 返回该请求ID的日志 |
| AL-010 | 导出日志 | P2 | 功能 | GET /api/v1/audit-logs/export | 导出会话内的操作日志 |
| AL-011 | 获取日志详情 | P2 | 功能 | GET /api/v1/audit-logs/{id} | 返回日志详细信息，包含请求/响应详情 |

---

## 9. 审批链管理

| ID | 测试用例 | 优先级 | 测试类型 | 操作步骤 | 预期结果 |
|----|----------|--------|----------|----------|----------|
| AC-001 | 审批链列表-正常分页 | P1 | 功能 | GET /api/v1/approval-chains?page=1&page_size=20 | 返回审批链列表 |
| AC-002 | 审批链列表-按实体类型筛选 | P2 | 功能 | GET /api/v1/approval-chains?entity_type=ticket | 仅返回工单审批链 |
| AC-003 | 审批链列表-按状态筛选 | P2 | 功能 | GET /api/v1/approval-chains?status=active | 仅返回激活的审批链 |
| AC-004 | 创建审批链-必填字段完整 | P1 | 功能 | POST /api/v1/approval-chains {name: "紧急工单审批", entity_type: "ticket", nodes: [...]} | 返回创建的审批链 |
| AC-005 | 创建审批链-缺少名称 | P2 | 边界 | POST /api/v1/approval-chains {entity_type: "ticket", nodes: [...]} | 返回400，name为必填 |
| AC-006 | 创建审批链-配置单个节点 | P1 | 功能 | POST /api/v1/approval-chains {...nodes: [{order: 1, type: "user", user_id: 5}]} | 审批链包含单个节点 |
| AC-007 | 创建审批链-配置多级节点 | P1 | 功能 | POST /api/v1/approval-chains {...nodes: [{order: 1, type: "role", role_id: 2}, {order: 2, type: "user", user_id: 5}]} | 多级审批链创建成功 |
| AC-008 | 获取审批链详情-正常 | P1 | 功能 | GET /api/v1/approval-chains/{id} | 返回审批链完整信息，包含节点列表 |
| AC-009 | 更新审批链-更新基本信息 | P1 | 功能 | PUT /api/v1/approval-chains/{id} {name: "新名称"} | 审批链名称更新成功 |
| AC-010 | 更新审批链-添加节点 | P2 | 功能 | PUT /api/v1/approval-chains/{id} {nodes: [..., new_node]} | 新增审批节点 |
| AC-011 | 删除审批链-正常删除 | P1 | 功能 | DELETE /api/v1/approval-chains/{id} | 返回200，删除成功 |
| AC-012 | 删除审批链-审批链正在使用 | P2 | 业务规则 | DELETE /api/v1/approval-chains/{id} (被工单使用) | 返回400，提示审批链正在使用 |
| AC-013 | 获取审批链统计 | P2 | 功能 | GET /api/v1/approval-chains/stats | 返回审批链使用统计 |
| AC-014 | 启用审批链 | P2 | 功能 | PUT /api/v1/approval-chains/{id} {status: "active"} | 审批链被激活 |
| AC-015 | 禁用审批链 | P2 | 功能 | PUT /api/v1/approval-chains/{id} {status: "inactive"} | 审批链被禁用 |

---

## 10. 工单分类

| ID | 测试用例 | 优先级 | 测试类型 | 操作步骤 | 预期结果 |
|----|----------|--------|----------|----------|----------|
| TC-001 | 分类列表-正常查询 | P1 | 功能 | GET /api/v1/ticket-categories | 返回分类列表 |
| TC-002 | 分类列表-按父级筛选 | P2 | 功能 | GET /api/v1/ticket-categories?parent_id=1 | 返回该父级下的子分类 |
| TC-003 | 分类列表-按层级筛选 | P2 | 功能 | GET /api/v1/ticket-categories?level=1 | 返回一级分类 |
| TC-004 | 分类列表-按状态筛选 | P2 | 功能 | GET /api/v1/ticket-categories?active=true | 仅返回激活分类 |
| TC-005 | 分类树形结构 | P1 | 功能 | GET /api/v1/ticket-categories/tree | 返回树形结构分类 |
| TC-006 | 创建分类-必填字段完整 | P1 | 功能 | POST /api/v1/ticket-categories {name: "网络故障", slug: "network"} | 返回创建的分类 |
| TC-007 | 创建分类-缺少名称 | P2 | 边界 | POST /api/v1/ticket-categories {slug: "network"} | 返回400，name为必填 |
| TC-008 | 创建分类-创建子分类 | P1 | 功能 | POST /api/v1/ticket-categories {name: "路由器故障", slug: "router", parent_id: 1} | 在父分类下创建子分类 |
| TC-009 | 获取分类详情-正常 | P1 | 功能 | GET /api/v1/ticket-categories/{id} | 返回分类完整信息 |
| TC-010 | 更新分类-更新基本信息 | P1 | 功能 | PUT /api/v1/ticket-categories/{id} {name: "新名称"} | 分类名称更新成功 |
| TC-011 | 更新分类-调整排序 | P2 | 功能 | PUT /api/v1/ticket-categories/{id} {sort_order: 5} | 分类排序更新 |
| TC-012 | 删除分类-正常删除 | P1 | 功能 | DELETE /api/v1/ticket-categories/{id} | 返回200，删除成功 |
| TC-013 | 删除分类-有子分类 | P2 | 业务规则 | DELETE /api/v1/ticket-categories/{id} (有子分类) | 返回400，提示存在子分类 |
| TC-014 | 删除分类-被工单使用 | P2 | 业务规则 | DELETE /api/v1/ticket-categories/{id} (有工单) | 返回400，提示分类正在使用 |
| TC-015 | 移动分类位置 | P2 | 功能 | PUT /api/v1/ticket-categories/{id}/move {new_parent_id: 2, new_sort_order: 3} | 分类位置调整 |
| TC-016 | 拖拽排序-更新排序 | P1 | 功能 | PUT /api/v1/ticket-categories/reorder {categories: [{id: 1, sort_order: 1}, {id: 2, sort_order: 2}]} | 批量更新分类排序 |

---

## 11. 导入模板管理

| ID | 测试用例 | 优先级 | 测试类型 | 操作步骤 | 预期结果 |
|----|----------|--------|----------|----------|
| IM-001 | 下载用户导入模板 | P1 | 功能 | GET /api/v1/templates/users/download | 下载Excel模板文件 |
| IM-002 | 下载资产导入模板 | P2 | 功能 | GET /api/v1/templates/assets/download | 下载资产导入模板 |
| IM-003 | 上传用户数据-正常上传 | P1 | 功能 | POST /api/v1/templates/users/upload (文件) | 返回验证结果 |
| IM-004 | 上传用户数据-部分成功 | P2 | 边界 | POST /api/v1/templates/users/upload (包含错误数据) | 返回部分成功结果 |
| IM-005 | 上传用户数据-格式错误 | P2 | 边界 | POST /api/v1/templates/users/upload (非Excel文件) | 返回400，文件格式错误 |
| IM-006 | 上传用户数据-缺少必填列 | P2 | 边界 | POST /api/v1/templates/users/upload (缺少username列) | 返回400，提示缺少必填列 |
| IM-007 | 上传用户数据-数据验证失败 | P2 | 边界 | POST /api/v1/templates/users/upload (邮箱格式错误) | 返回400，列出验证失败行 |
| IM-008 | 模板验证-验证必填字段 | P1 | 功能 | POST /api/v1/templates/validate (users数据) | 返回每行数据验证结果 |
| IM-009 | 模板验证-验证数据格式 | P1 | 功能 | POST /api/v1/templates/validate (邮箱、密码长度) | 验证数据格式正确性 |
| IM-010 | 模板验证-验证重复数据 | P2 | 功能 | POST /api/v1/templates/validate (重复邮箱) | 提示重复数据 |

---

## 附录

### A. 通用API响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

### B. 错误码说明

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 1001+ | 参数错误 |
| 2001 | 认证失败 |
| 5001 | 内部错误 |

### C. 测试环境要求

- 租户ID: 1 (测试租户)
- 用户角色: super_admin
- 认证方式: Bearer Token

### D. 测试数据准备

1. 创建测试租户
2. 创建管理员用户
3. 创建测试部门和团队
4. 初始化权限数据
5. 准备测试审批链和分类
