# ITSM 路由验证指南

## 🧪 快速验证步骤

### 1. 启动开发服务器

```bash
cd /Users/heidsoft/Downloads/research/itsm/itsm-prototype
npm run dev
```

### 2. 验证认证路由（公开访问）

打开浏览器并访问以下 URL：

#### ✅ 登录页面

- **URL**: <http://localhost:3000/login>
- **预期**: 显示登录页面（简洁布局，无 Header/Sidebar）
- **文件**: `src/app/(auth)/login/page.tsx`

### 3. 验证主应用路由（需要认证）

**注意**: 以下路由需要先登录才能访问

#### ✅ 首页重定向

- **URL**: <http://localhost:3000/>
- **预期**:
  - 未登录 → 重定向到 `/login`
  - 已登录 → 重定向到 `/dashboard`

#### ✅ 仪表盘

- **URL**: <http://localhost:3000/dashboard>
- **预期**: 显示仪表盘（带 Header + Sidebar）
- **文件**: `src/app/(main)/dashboard/page.tsx`
- **布局**: 使用 `(main)/layout.tsx`

#### ✅ 工单管理

- **列表**: <http://localhost:3000/tickets>
- **详情**: <http://localhost:3000/tickets/1>
- **创建**: <http://localhost:3000/tickets/create>
- **模板**: <http://localhost:3000/tickets/templates>

#### ✅ 事件管理

- **列表**: <http://localhost:3000/incidents>
- **详情**: <http://localhost:3000/incidents/1>
- **创建**: <http://localhost:3000/incidents/new>

#### ✅ 问题管理

- **列表**: <http://localhost:3000/problems>
- **详情**: <http://localhost:3000/problems/1>
- **创建**: <http://localhost:3000/problems/new>

#### ✅ 变更管理

- **列表**: <http://localhost:3000/changes>
- **详情**: <http://localhost:3000/changes/1>
- **创建**: <http://localhost:3000/changes/new>

#### ✅ 配置管理 (CMDB)

- **主页**: <http://localhost:3000/cmdb>
- **CI详情**: <http://localhost:3000/cmdb/1>

#### ✅ 知识库

- **主页**: <http://localhost:3000/knowledge-base>
- **文章**: <http://localhost:3000/knowledge-base/1>
- **创建**: <http://localhost:3000/knowledge-base/new>

#### ✅ 服务目录

- **主页**: <http://localhost:3000/service-catalog>

#### ✅ SLA 管理

- **主页**: <http://localhost:3000/sla>
- **仪表盘**: <http://localhost:3000/sla-dashboard>

#### ✅ 报告中心

- **主页**: <http://localhost:3000/reports>

#### ✅ 工作流管理

- **主页**: <http://localhost:3000/workflow>
- **设计器**: <http://localhost:3000/workflow/designer>
- **实例**: <http://localhost:3000/workflow/instances>

#### ✅ 系统管理

- **主页**: <http://localhost:3000/admin>
- **用户**: <http://localhost:3000/admin/users>
- **角色**: <http://localhost:3000/admin/roles>
- **租户**: <http://localhost:3000/admin/tenants>
- **审批链**: <http://localhost:3000/admin/approval-chains>

#### ✅ 个人中心

- **主页**: <http://localhost:3000/profile>

---

## 🔍 验证要点

### 1. 布局检查

#### 认证页面 (`(auth)` 路由组)

- ✅ **无** Header
- ✅ **无** Sidebar
- ✅ 全屏布局
- ✅ 简洁设计

#### 主应用页面 (`(main)` 路由组)

- ✅ **有** Header（顶部导航栏）
- ✅ **有** Sidebar（左侧菜单）
- ✅ **有** Footer（底部信息）
- ✅ 内容区域有圆角和阴影

### 2. Sidebar 菜单检查

打开任意主应用页面，检查 Sidebar：

- ✅ Logo 区域显示正确
- ✅ 菜单项包含：
  - 仪表盘
  - 工单管理
  - 事件管理
  - 问题管理
  - 变更管理
  - 配置管理
  - 服务目录
  - 知识库
  - SLA管理
  - 报表分析
  - 工作流管理（管理员）
  - 系统管理（管理员）
- ✅ 当前页面高亮
- ✅ 折叠/展开功能正常

### 3. Header 检查

- ✅ 折叠按钮工作正常
- ✅ 面包屑导航正确显示
- ✅ 搜索框功能
- ✅ 通知中心
- ✅ 用户菜单（头像 + 下拉）

### 4. 响应式检查

调整浏览器宽度：

- ✅ 宽度 < 768px 时，Sidebar 自动折叠
- ✅ 移动端点击内容区域，Sidebar 关闭
- ✅ 移动端出现遮罩层

### 5. 认证流程检查

1. ✅ 未登录访问主应用 → 重定向到 `/login`
2. ✅ 登录成功 → 重定向到 `/dashboard`
3. ✅ 退出登录 → 返回 `/login`
4. ✅ 直接访问 `/` → 根据登录状态重定向

---

## 🐛 常见问题排查

### 问题 1: 页面显示 404

**可能原因**:

- 文件路径不正确
- 文件名不是 `page.tsx`

**检查**:

```bash
# 检查文件是否存在
ls -la src/app/(main)/dashboard/page.tsx
```

### 问题 2: Sidebar 或 Header 不显示

**可能原因**:

- `(main)/layout.tsx` 未生效
- 组件导入路径错误

**检查**:

```bash
# 检查布局文件
cat src/app/(main)/layout.tsx
```

### 问题 3: 认证重定向不工作

**可能原因**:

- AuthService 未正确导入
- Cookie/localStorage 未设置

**检查**:

- 打开浏览器控制台
- 查看 Network 标签
- 检查 localStorage 是否有 `auth_token`

### 问题 4: 路由组出现在 URL 中

**错误示例**: `/（main)/dashboard`

**原因**: 这是 Next.js 的 bug 或配置问题

**解决**:

- 确保文件夹名称使用英文括号 `(main)` 而非中文括号
- 重启开发服务器

---

## 📊 验证清单

完成以下检查后，在 [ ] 中打勾：

### 基础验证

- [ ] 开发服务器启动成功
- [ ] `/login` 页面显示正常
- [ ] `/dashboard` 页面显示正常
- [ ] Header 显示正常
- [ ] Sidebar 显示正常

### 路由验证

- [ ] 所有主模块页面可访问
- [ ] 动态路由（如 `/tickets/1`）正常工作
- [ ] 子路由（如 `/admin/users`）正常工作

### 布局验证

- [ ] 认证页面使用简洁布局
- [ ] 主应用页面使用完整布局
- [ ] 布局切换正常

### 功能验证

- [ ] 登录流程正常
- [ ] 退出登录正常
- [ ] 页面跳转正常
- [ ] Sidebar 折叠/展开正常
- [ ] 响应式布局正常

### 性能验证

- [ ] 页面加载速度正常
- [ ] 无控制台错误
- [ ] 无 404 错误

---

## 📝 测试记录

**测试日期**: _____________  
**测试人员**: _____________  
**浏览器**: _____________  
**分辨率**: _____________

**发现的问题**:
1.
2.
3.

**备注**:

---

## 🎉 验证通过标准

当满足以下条件时，视为验证通过：

1. ✅ 所有认证路由可访问
2. ✅ 所有主应用路由可访问
3. ✅ Header + Sidebar 正常显示和工作
4. ✅ 认证流程正常
5. ✅ 响应式布局正常
6. ✅ 无控制台错误
7. ✅ URL 路径正确（无路由组名称）

---

**准备好了吗？开始验证吧！** 🚀
