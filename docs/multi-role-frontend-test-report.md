# ITSM 多角色前端功能测试报告 - 详细版

**测试日期**: 2026-05-05  
**测试环境**: <http://localhost:3000>  
**测试用户**: admin, l1_engineer, employee  

---

## 一、测试结果总览

| 角色 | 登录 | 仪表盘 | 事件管理 | 问题管理 | 变更管理 | CMDB | 工作流管理 | 用户管理 |
|------|------|--------|----------|----------|----------|------|------------|----------|
| **admin** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **l1_engineer** | ✅ | ❌ 空白 | ❌ 403 | ❌ 403 | ❌ 403 | ❌ 403 | ❌ 重定向 | ❌ 重定向 |
| **employee** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ 重定向 | ⚠️ 加载失败 |

---

## 二、详细测试结果

### 2.1 admin 用户 (超级管理员)

**登录凭证**: `admin / admin123`

**菜单结构**:

- 主菜单: 仪表盘、工单管理、事件管理、问题管理、变更管理、CMDB、服务目录、知识库、SLA监控、报表、发布管理、资产管理、许可证管理、MSP管理
- 管理菜单: 工作流、用户管理、角色管理、组管理、部门管理、团队管理、审批管理、SLA配置、工单分类、系统配置、CI类型管理

**功能测试**:

| 页面 | URL | 结果 | 说明 |
|------|-----|------|------|
| 仪表盘 | /dashboard | ✅ | 正常显示统计卡片、图表 |
| 事件列表 | /incidents | ✅ | 显示32条事件，分页正常 |
| 工作流管理 | /admin/workflows | ✅ | 可访问，显示创建按钮 |
| 用户管理 | /admin/users | ✅ | 显示9个用户 |
| SLA配置 | /admin/sla | ❌ 404 | 页面不存在 |

---

### 2.2 l1_engineer 用户 (一线工程师)

**登录凭证**: `l1_engineer / admin123`  
**角色代码**: `l1_support`  
**问题**: 登录后无法正常使用系统

**症状**:

- 登录后访问仪表盘显示空白页面
- 所有API请求返回 403 权限不足
- 被重定向到仪表盘，无法访问任何功能

**测试的API响应**:

```
/api/v1/incidents    → 403 权限不足
/api/v1/problems     → 403 权限不足
/api/v1/changes      → 403 权限不足
/api/v1/users/me     → 403 权限不足
/api/v1/permissions  → 403 权限不足
```

**根因分析**:
数据库查询发现 `l1_support` 角色在 `role_permissions` 表中的权限数为 **0**！

```sql
SELECT r.name, r.code, COUNT(rp.id) as permission_count
FROM roles r
LEFT JOIN role_permissions rp ON r.id = rp.role_id
WHERE r.tenant_id = 1 AND r.code = 'l1_support'
GROUP BY r.id, r.name, r.code;

-- 结果: l1_support 角色 permission_count = 0
```

---

### 2.3 employee 用户 (普通员工)

**登录凭证**: `employee / admin123`  
**角色代码**: `end_user`  
**权限列表**: 包含 ticket、notification、knowledge、dashboard、ai、service_catalog、service_request 等读/写权限

**菜单结构** (受限):

- 主菜单: 仪表盘、工单管理、事件管理、问题管理、变更管理、CMDB、服务目录、知识库、SLA监控、发布管理、资产管理、许可证管理
- 管理菜单: ❌ 无管理菜单

**功能测试**:

| 页面 | URL | 结果 | 说明 |
|------|-----|------|------|
| 仪表盘 | /dashboard | ✅ | 正常显示统计卡片 |
| 事件列表 | /incidents | ✅ | 可访问，显示列表 |
| 事件创建 | /incidents/create | ✅ | 表单可填写 |
| 用户管理 | /admin/users | ⚠️ | 页面显示但加载失败 |
| 工作流管理 | /admin/workflows | ❌ | 重定向到仪表盘 |

---

## 三、发现的问题

### 🔴 严重问题

#### 1. l1_support 角色权限缺失

**问题**: `l1_support` 角色在 `role_permissions` 表中没有权限记录

**影响**: 所有分配了 `l1_support` 角色的用户无法正常使用系统

**解决方案**: 执行以下SQL脚本

```bash
psql -h localhost -U postgres -d <dbname> -f scripts/fix_l1_support_permissions.sql
```

或手动执行:

```sql
-- 从 technician 角色复制权限到 l1_support
INSERT INTO role_permissions (role_id, permission_id, permission_definition_role_permissions)
SELECT 
    (SELECT id FROM roles WHERE code = 'l1_support' AND tenant_id = 1),
    pd.id,
    pd.id
FROM permission_definitions pd
WHERE pd.id IN (
    SELECT permission_definition_role_permissions 
    FROM role_permissions rp
    JOIN roles r ON rp.role_id = r.id
    WHERE r.code = 'technician' AND r.tenant_id = 1
)
ON CONFLICT DO NOTHING;

-- 从 agent 角色复制权限到 l1_support
INSERT INTO role_permissions (role_id, permission_id, permission_definition_role_permissions)
SELECT 
    (SELECT id FROM roles WHERE code = 'l1_support' AND tenant_id = 1),
    pd.id,
    pd.id
FROM permission_definitions pd
WHERE pd.id IN (
    SELECT permission_definition_role_permissions 
    FROM role_permissions rp
    JOIN roles r ON rp.role_id = r.id
    WHERE r.code = 'agent' AND r.tenant_id = 1
)
ON CONFLICT DO NOTHING;
```

---

#### 2. SLA配置页面不存在

**问题**: URL `/admin/sla` 返回 404

**解决方案**:

1. 创建 `/admin/sla` 页面，或
2. 修正前端路由配置

---

#### 3. 用户管理页面加载失败

**问题**: employee 用户访问 `/admin/users` 显示 "加载用户列表失败"

**可能原因**:

- API `/api/v1/users` 需要更高权限
- 前端权限检查逻辑问题

---

## 四、权限配置表

| 角色代码 | 角色名称 | 权限数 | 说明 |
|----------|----------|--------|------|
| super_admin | 超级管理员 | 58 | 完全权限 |
| sysadmin | 系统管理员 | 58 | 完全权限 |
| admin | 管理员 | 39 | 大部分权限 |
| manager | 经理 | 20 | 管理权限 |
| agent | 服务台坐席 | 15 | 客服权限 |
| end_user | 普通用户 | 15 | 基础权限 |
| l1_support | 一线工程师 | **0** | ❌ 缺失 |
| l2_support | 二线工程师 | 0 | 缺失 |
| l3_expert | 三线专家 | 0 | 缺失 |
| ops_manager | 运维经理 | 0 | 缺失 |
| sd_manager | 服务台主管 | 0 | 缺失 |

---

## 五、修复建议

### 5.1 立即修复 (高优先级)

1. **执行权限修复脚本**

   ```bash
   psql -h localhost -U postgres -d <dbname> -f scripts/fix_l1_support_permissions.sql
   ```

2. **修复 l2_support, l3_expert, ops_manager, sd_manager 角色权限**
   - 参考 `fix_l1_support_permissions.sql` 脚本
   - 从 `technician` 或 `agent` 角色复制权限

### 5.2 中期修复 (中优先级)

1. **修复 SLA 配置页面**
   - 创建页面或修正路由

2. **修复用户管理页面权限**
   - 检查前端权限判断逻辑

### 5.3 长期改进 (低优先级)

1. **完善权限配置初始化脚本**
   - 确保所有角色在初始化时都有正确权限

2. **添加权限配置验证**
   - 启动时检查权限配置完整性

---

## 六、测试截图

测试截图保存位置: `/Users/heidsoft/Downloads/research/itsm/test_screenshots/`

---

## 七、结论

| 问题 | 严重程度 | 状态 |
|------|---------|------|
| l1_support 角色权限缺失 | 🔴 严重 | ⏳ 需修复 |
| l2/l3/ops/sd 角色权限缺失 | 🔴 严重 | ⏳ 需修复 |
| SLA配置页面404 | ⚠️ 中等 | ⏳ 需修复 |
| 用户管理加载失败 | ⚠️ 中等 | ⏳ 需修复 |

**总体评估**: 系统核心功能正常，但部分角色权限配置不完整需要修复后才能正常使用。

---

**报告日期**: 2026-05-05  
**报告人员**: AI架构师
