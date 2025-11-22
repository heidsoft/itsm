# 🚀 ITSM 平台快速启动指南

## ✅ 重构已完成

目录结构已按照 `ITSM_OPTIMIZED_STRUCTURE.md` 优化完成：
- ✅ 创建了 `(auth)` 和 `(main)` 路由组
- ✅ 移动了所有页面到对应路由组
- ✅ 创建了专用布局文件
- ✅ 保持了所有 URL 路径不变

---

## 🎯 立即开始测试

### 步骤 1: 启动开发服务器

```bash
cd /Users/heidsoft/Downloads/research/itsm/itsm-prototype
npm run dev
```

### 步骤 2: 打开浏览器

访问: **http://localhost:3000**

---

## 🔍 快速验证点

### 1️⃣ 登录页面测试

**URL**: http://localhost:3000/login

**预期效果**:
- ✅ 全屏布局
- ✅ 无 Header（顶部导航）
- ✅ 无 Sidebar（侧边菜单）
- ✅ 干净简洁的登录界面

### 2️⃣ 主应用测试（需要先登录）

**默认测试账号** (如果后端已配置):
- 用户名: `admin` 或 `test`
- 密码: 根据后端配置

**测试 URL**: http://localhost:3000/dashboard

**预期效果**:
- ✅ 显示 Header（顶部导航栏）
- ✅ 显示 Sidebar（左侧菜单）
- ✅ 显示 Footer（底部信息）
- ✅ 内容区域有圆角和阴影
- ✅ 当前页面在 Sidebar 中高亮

### 3️⃣ 快速测试其他模块

打开以下任一 URL，应该都能正常显示：

```
http://localhost:3000/tickets        - 工单管理
http://localhost:3000/incidents      - 事件管理
http://localhost:3000/problems       - 问题管理
http://localhost:3000/changes        - 变更管理
http://localhost:3000/cmdb           - 配置管理
http://localhost:3000/admin          - 系统管理
```

---

## 📋 关键验证清单（5分钟）

快速验证以下项目：

- [ ] ✅ 服务器启动成功（无错误）
- [ ] ✅ `/login` 显示简洁布局
- [ ] ✅ `/dashboard` 显示完整布局（Header + Sidebar）
- [ ] ✅ Sidebar 菜单显示完整
- [ ] ✅ 点击 Sidebar 菜单项可以跳转
- [ ] ✅ 浏览器控制台无红色错误
- [ ] ✅ URL 路径正确（无 `(main)` 或 `(auth)` 出现）

---

## 🎨 布局效果预览

### 认证页面 (`/login`)
```
┌─────────────────────────────────────┐
│                                     │
│                                     │
│          [ITSM Logo]                │
│                                     │
│      ┌───────────────────┐          │
│      │  用户名: [____]   │          │
│      │  密码:   [____]   │          │
│      │  [   登录   ]     │          │
│      └───────────────────┘          │
│                                     │
│                                     │
└─────────────────────────────────────┘
     简洁全屏布局（无 Header/Sidebar）
```

### 主应用页面 (`/dashboard`, `/tickets` 等)
```
┌─────────────────────────────────────────────────┐
│ Header: [≡] 仪表盘 > ...   [搜索] [🔔] [👤]      │
├────────┬────────────────────────────────────────┤
│        │                                        │
│ Side   │  Content Area                         │
│ bar    │  ┌──────────────────────────────────┐ │
│        │  │                                  │ │
│ [仪表盘] │  │  页面内容显示在这里              │ │
│ [工单]  │  │                                  │ │
│ [事件]  │  │                                  │ │
│ [问题]  │  │                                  │ │
│ [变更]  │  └──────────────────────────────────┘ │
│ [CMDB] │                                        │
│ ...    │  Footer: © 2025 ITSM Platform         │
└────────┴────────────────────────────────────────┘
    完整布局（Header + Sidebar + Content + Footer）
```

---

## 🐛 遇到问题？

### 问题 1: 启动失败

```bash
# 清理并重新安装依赖
rm -rf node_modules package-lock.json
npm install
npm run dev
```

### 问题 2: 页面 404

**检查文件是否存在**:
```bash
ls -la "src/app/(main)/dashboard/page.tsx"
ls -la "src/app/(auth)/login/page.tsx"
```

如果文件存在但仍然 404，尝试：
```bash
# 清理 Next.js 缓存
rm -rf .next
npm run dev
```

### 问题 3: Sidebar 或 Header 不显示

**检查布局文件**:
```bash
cat "src/app/(main)/layout.tsx"
```

确保文件存在且没有语法错误。

### 问题 4: 认证重定向循环

打开浏览器控制台（F12），检查：
1. Network 标签 - 查看 API 请求状态
2. Console 标签 - 查看错误信息
3. Application 标签 - 检查 localStorage 中是否有 `auth_token`

---

## 📚 详细文档

如需更详细的信息，请查看：

1. **VERIFY_ROUTES.md** - 完整的路由验证指南（您当前查看的文档）
2. **ITSM_RESTRUCTURE_SUMMARY.md** - 重构总结和 URL 映射表
3. **BEFORE_AFTER_COMPARISON.md** - 前后对比分析
4. **ITSM_OPTIMIZED_STRUCTURE.md** - 原始优化设计文档

---

## 💡 重要提示

### URL 路径说明

路由组 `(auth)` 和 `(main)` **不会出现在 URL 中**：

```
✅ 正确:
  文件: src/app/(auth)/login/page.tsx  → URL: /login
  文件: src/app/(main)/tickets/page.tsx → URL: /tickets

❌ 错误（这不会发生）:
  URL: /(auth)/login
  URL: /(main)/tickets
```

### 后端服务

确保后端服务已启动：

```bash
# 在另一个终端窗口
cd /Users/heidsoft/Downloads/research/itsm/itsm-backend
go run main.go
```

默认后端地址: **http://localhost:8090**

---

## 🎉 成功标志

当您看到以下情况时，说明重构成功：

1. ✅ 开发服务器启动无错误
2. ✅ `/login` 页面显示且布局简洁
3. ✅ `/dashboard` 页面显示且有 Header + Sidebar
4. ✅ Sidebar 菜单完整且可点击
5. ✅ 页面跳转流畅无卡顿
6. ✅ 浏览器控制台无红色错误
7. ✅ 所有 URL 路径正确（无路由组名称）

---

## 🚀 下一步建议

### 立即行动
1. 启动服务器并验证基本功能
2. 测试登录流程
3. 浏览各个模块页面

### 后续优化
1. 移除各模块中重复的 `layout.tsx`（如果不提供额外功能）
2. 优化移动端响应式体验
3. 添加页面切换动画
4. 完善权限控制逻辑

---

## 📞 需要帮助？

如果遇到任何问题：

1. 查看浏览器控制台的错误信息
2. 检查终端的错误输出
3. 参考详细文档中的问题排查章节
4. 确保后端服务正常运行

---

**准备好了吗？现在就启动服务器试试看！** 🎉

```bash
npm run dev
```

然后访问: **http://localhost:3000** 🚀

