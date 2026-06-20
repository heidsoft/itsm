# ITSM系统浏览器测试问题修复计划

## Context

在ITSM系统浏览器测试中发现了5个问题，涉及角色权限保存、事件创建、租户创建、工作流页面和Ant Design弃用API。这些问题影响了系统的核心业务功能，需要优先修复高严重度问题。

## 问题总览

| 编号 | 严重程度 | 模块 | 问题描述 |
|------|----------|------|----------|
| BUG-001 | 🔴 高 | 权限管理 | 角色权限保存失败 |
| BUG-002 | 🔴 高 | 事件管理 | 事件创建弃用API警告 |
| BUG-003 | 🟡 中 | 租户管理 | 租户创建下拉选择器验证失败 |
| BUG-004 | 🟡 中 | 工作流 | 工作流页面内容为空 |
| BUG-005 | 🟢 低 | 全局 | Ant Design组件弃用API警告 |

---

## BUG-001 (高): 角色权限保存失败

### 根因分析

**文件**: `itsm-frontend/src/app/(main)/admin/roles/page.tsx`

`handleSaveRole`函数（行245-288）将权限作为字符串数组（如`['ticket:view', 'ticket:create']`）与角色基本信息一起发送到`updateRole`接口。但后端API期望接收权限ID（数字），而非字符串编码。

**关键代码路径**:
- 行261-267: `roleData`包含`permissions: string[]`
- 行271: `await RoleAPI.updateRole(selectedRole.id, roleData)`
- `RoleAPI`有独立的`assignPermissions(roleId, permissionIds)`方法（行101-103）

### 修复方案

**修改文件**: `itsm-frontend/src/app/(main)/admin/roles/page.tsx`

**修改内容**: 重构`handleSaveRole`为两步操作:
1. 更新角色基本信息（不含权限）
2. 调用`RoleAPI.assignPermissions()`分配权限

```typescript
// 行245-288: 重写handleSaveRole
const handleSaveRole = async () => {
  setLoading(true);
  try {
    const values = await form.validateFields();

    // 构建权限编码列表
    const permissionCodes: string[] = [];
    Object.values(PERMISSION_MODULES).forEach(module => {
      Object.values(PERMISSION_ACTIONS).forEach(action => {
        const fieldName = `${module}_${action}`;
        if (values[fieldName]) {
          permissionCodes.push(`${module}:${action}`);
        }
      });
    });

    // 更新角色基本信息（不含权限）
    const roleData = {
      name: values.name,
      code: values.code,
      description: values.description,
      status: (values.status ? 'active' : 'inactive') as 'active' | 'inactive',
    };

    let roleId: number;
    if (selectedRole) {
      const updated = await RoleAPI.updateRole(selectedRole.id, roleData);
      roleId = updated.id;
      message.success('角色更新成功');
    } else {
      const created = await RoleAPI.createRole({ ...roleData, permissions: permissionCodes });
      roleId = created.id;
      message.success('角色创建成功');
    }

    // 分配权限（使用专用接口）
    if (permissionCodes.length > 0) {
      try {
        const catalog = await RoleAPI.getPermissionCatalog();
        const codeToId = new Map(catalog.map(p => [p.code, p.id]));
        const permissionIds = permissionCodes
          .map(code => codeToId.get(code))
          .filter((id): id is number => id !== undefined && id !== 0);

        if (permissionIds.length > 0) {
          await RoleAPI.assignPermissions(roleId, permissionIds);
        }
      } catch (permError) {
        console.error('Failed to assign permissions:', permError);
        message.warning('角色基本信息已保存，但权限分配失败，请重试');
      }
    }

    setShowModal(false);
    form.resetFields();
    setSelectedRole(null);
    loadRoles();
  } catch (error) {
    message.error('保存角色失败');
  } finally {
    setLoading(false);
  }
};
```

### 验证方法

1. 创建新角色，选择多个权限，验证权限保存成功
2. 编辑现有角色，修改权限，验证更改持久化
3. 编辑角色不修改权限，验证现有权限保持不变
4. 测试空权限场景（未选择任何复选框）

---

## BUG-002 (高): 事件创建弃用API警告

### 根因分析

**文件**: `itsm-frontend/src/app/(main)/incidents/create/page.tsx`

行422使用了Ant Design v6已弃用的API:
```tsx
<Space direction="vertical" className="w-full">
```

### 修复方案

**修改文件**: `itsm-frontend/src/app/(main)/incidents/create/page.tsx`

**修改内容**: 行422
```tsx
// 修改前
<Space direction="vertical" className="w-full">

// 修改后
<Space orientation="vertical" className="w-full">
```

### 验证方法

1. 访问事件创建页面
2. 验证浏览器控制台无弃用警告
3. 验证右侧信息面板垂直布局正常
4. 提交事件验证表单功能正常

---

## BUG-003 (中): 租户创建下拉选择器验证失败

### 根因分析

**文件**: `itsm-frontend/src/app/(main)/admin/tenants/page.tsx`

Form.Item绑定结构正确，但新建租户时未设置`type`和`status`的默认值，可能导致验证失败。

### 修复方案

**修改文件**: `itsm-frontend/src/app/(main)/admin/tenants/page.tsx`

**修改内容**: 行512，为Form添加`initialValues`
```tsx
// 修改前
<Form form={form} layout="vertical" className="mt-4" disabled={viewOnly}>

// 修改后
<Form
  form={form}
  layout="vertical"
  className="mt-4"
  disabled={viewOnly}
  initialValues={{ type: 'standard', status: 'active' }}
>
```

### 验证方法

1. 点击"新建租户"，选择类型和状态下拉框，验证无验证错误
2. 不选择下拉框直接提交，验证显示验证消息
3. 编辑现有租户，验证下拉框显示当前值
4. 以只读模式打开租户，验证下拉框禁用

---

## BUG-004 (中): 工作流页面内容为空

### 根因分析

**文件**: `itsm-frontend/src/app/(main)/workflow/page.tsx`

工作流页面从`WorkflowAPI.getWorkflows()`加载数据，调用`GET /api/v1/bpmn/process-definitions`。页面已有完善的错误处理和空状态显示。

**可能原因**:
1. BPMN后端服务未运行或未配置
2. API返回错误（认证、权限或服务不可用）
3. API返回数据格式与前端适配器不匹配

### 修复方案

**验证步骤**（无需代码修改）:
1. 检查后端BPMN服务是否可访问
2. 检查浏览器Network标签页的API响应
3. 检查认证token是否有效

**如果API正常但返回空数据**: 页面已正确显示空状态消息和"新建工作流"按钮，这是预期行为。

### 验证方法

1. 验证后端BPMN服务可访问
2. 加载工作流页面，检查浏览器控制台错误
3. 检查Network标签页的API响应
4. 如果API返回数据，验证表格正确渲染

---

## BUG-005 (低): Ant Design弃用API警告

### 影响范围

需要修改的文件（`<Space direction="vertical"` → `<Space orientation="vertical"`）:

| 文件 | 行号 |
|------|------|
| `src/components/asset/AssetDetail.tsx` | 141 |
| `src/app/(main)/workflow/page.tsx` | 1301 |
| `src/app/(main)/admin/groups/page.tsx` | 161 |
| `src/app/(main)/admin/connectors/page.tsx` | 155, 368 |
| `src/app/(main)/incidents/create/page.tsx` | 422 (BUG-002) |
| `src/app/(main)/problems/known-errors/page.tsx` | 320 |
| `src/components/problem/ProblemDetail.tsx` | 92 |
| `src/components/license/LicenseDetail.tsx` | 124 |

### 修复方案

在每个文件中，将`direction="vertical"`改为`orientation="vertical"`。

**注意**: 不要修改`Steps`组件或自定义组件的`direction`属性。

### 验证方法

1. 运行`npm run build`验证无编译错误
2. 访问每个受影响页面
3. 验证浏览器控制台无弃用警告
4. 验证视觉布局不变

---

## 实施顺序

1. **BUG-001** (30分钟) -- 最高优先级，核心功能损坏
2. **BUG-002** (5分钟) -- 快速修复，属于BUG-005范围
3. **BUG-003** (15分钟) -- 中等影响，表单可用性问题
4. **BUG-004** (30分钟) -- 可能是后端问题，需要调查
5. **BUG-005** (20分钟) -- 批量修复剩余弃用警告

## 风险和注意事项

1. **BUG-001**: `getPermissionCatalog()`端点可能返回`id: 0`的项目（权限字典未初始化）。修复包含`filter(id => id !== 0)`保护，但应先使用角色页面的"初始化权限字典"按钮确保后端权限目录已填充。

2. **BUG-003**: 如果添加`initialValues`不能解决问题，可能需要考虑Ant Design v6版本特性，尝试添加`preserve={false}`到Form组件。

3. **BUG-004**: 工作流页面空状态可能是后端/配置问题而非前端bug。前端代码结构正确，有完善的错误处理和空状态显示。

4. **BUG-005**: `orientation`属性替换已确认适用于Ant Design v6的`Space`组件。不要修改`Steps`组件或自定义组件的`direction`属性。

## 验证清单

- [ ] 角色权限保存功能正常
- [ ] 事件创建无弃用警告
- [ ] 租户创建下拉选择器正常工作
- [ ] 工作流页面正常加载或显示空状态
- [ ] 所有页面无Ant Design弃用警告
- [ ] `npm run build`编译成功
