# ITSM 前端开发负面清单

> **版本**: 1.0
> **最后更新**: 2026-06-14
> **目的**: 列出前端开发中不应使用的反模式和坏习惯，供 AI 开发助手遵循

---

## 1. 配色与对比度反模式

### 1.1 禁止灰色透明背景配白色文字

```tsx
// ❌ 错误模式：灰色透明背景 + 白色文字 = 对比度不足，不可读
<div className="bg-gray-100/50 text-white">文字</div>
<div className="bg-gray-200/30 text-white">文字</div>
<div className="bg-gray-50/50 text-white">文字</div>

// ✅ 正确模式 1：深色背景 + 白色文字
<div className="bg-gray-900 text-white">文字</div>

// ✅ 正确模式 2：浅色背景 + 深色文字
<div className="bg-gray-100 text-gray-900">文字</div>
```

### 1.2 禁止灰色透明标签配白色文字

```tsx
// ❌ 错误模式
<span className="bg-gray-200/50 text-white">标签</span>
<span className="bg-gray-100/30 text-white">标签</span>

// ✅ 正确模式：使用明确的颜色组合
<span className="bg-blue-100 text-blue-800">标签</span>
<Tag color="blue">标签</Tag>
```

### 1.3 禁止使用透明背景配对比色文字

```tsx
// ❌ 错误模式：任何透明背景配高对比度文字
<div className="bg-white/50 text-gray-900">文字</div>
<div className="bg-black/10 text-black">文字</div>

// ✅ 正确模式：使用实色背景
<div className="bg-white text-gray-900">文字</div>
```

---

## 2. 页面状态文字反模式

### 2.1 禁止在生产环境显示开发状态提示

```tsx
// ❌ 错误模式：显示开发进度、技术细节
<ManagementNotice
  message="复杂域页面开始统一收口"
  description="总览、云资源、核对和配置项详情将逐步复用同一套页面头部..."
/>

<ManagementNotice
  message="设计态与运行态已分离处理"
  description="工作流定义、实例管理和设计器入口保持分层..."
/>

<ManagementNotice
  message="报表页已切换为稳态渲染"
  description="图表数据失败时会显示可恢复空态..."
/>

// ✅ 正确模式：移除这些提示，或替换为功能性描述
<ManagementPageHeader
  title="配置管理数据库 (CMDB)"
  description="管理配置项、云资源同步、关系拓扑和核对结果。"
/>
```

### 2.2 禁止使用 "开发中"、"Under Construction" 等文字

```tsx
// ❌ 错误模式
<div className="text-yellow-600">此功能开发中...</div>
<Alert type="warning" message="功能开发中，敬请期待" />

// ✅ 正确模式：使用空状态组件或禁用功能
<LoadingEmptyError
  state="empty"
  empty={{
    title: '暂无数据',
    description: '功能开发中，请稍后再试。',
  }}
/>

// 或使用 disabled 状态
<Button disabled>新增配置项</Button>
```

### 2.3 禁止暴露内部实现细节

```tsx
// ❌ 错误模式：暴露实现细节
<Notice type="info" message="正在使用新的渲染引擎" />
<Notice type="info" message="已切换为稳态渲染" />
<Notice type="info" message="将逐步复用组件基线" />

// ✅ 正确模式：面向用户的功能描述
<Notice type="info" message="配置项管理支持云资源同步" />
```

---

## 3. 布局一致性反模式

### 3.1 禁止在同一行卡片中堆叠多个 Statistic

```tsx
// ❌ 错误模式：一个 Card 中包含多个 Statistic，导致高度不一致
<Card>
  <Statistic title="合规率" value={95} suffix="%" />
  <Statistic title="总违规" value={5} />
</Card>

// ✅ 正确模式：每个 Statistic 独立一个 Card
<Card>
  <Statistic title="合规率" value={95} suffix="%" />
</Card>
<Card>
  <Statistic title="总违规" value={5} />
</Card>
```

### 3.2 禁止同一行统计卡片使用不同的 Col 配置

```tsx
// ❌ 错误模式：同一行使用不同的 Col 配置
<Col xs={24} sm={12} md={6}>
  <Card>...</Card>
</Col>
<Col xs={24} sm={24} md={12}>  {/* 尺寸不一致 */}
  <Card>...</Card>
</Col>

// ✅ 正确模式：同一行使用相同的 Col 配置
<Row gutter={[16, 16]}>
  <Col xs={24} sm={12} md={6}><Card>...</Card></Col>
  <Col xs={24} sm={12} md={6}><Card>...</Card></Col>
  <Col xs={24} sm={12} md={6}><Card>...</Card></Col>
  <Col xs={24} sm={12} md={6}><Card>...</Card></Col>
</Row>
```

### 3.3 禁止卡片高度不统一

```tsx
// ❌ 错误模式：使用固定高度导致内容溢出
<Card style={{ height: 200 }}>
  <Statistic title="数值" value={123} />
  {/* 标题过长导致换行，卡片高度不一致 */}
</Card>

// ✅ 正确模式：使用 flex 布局确保高度一致
<Row gutter={[16, 16]} align="stretch">
  <Col xs={24} sm={12} md={6} className="flex">
    <Card className="h-full w-full">
      <Statistic title="数值" value={123} />
    </Card>
  </Col>
</Row>
```

---

## 4. 表单样式反模式

### 4.1 禁止 Select mode="tags" 使用默认标签样式

```tsx
// ❌ 错误模式：使用默认标签样式，可能在深色背景上不可见
<Select mode="tags" placeholder="输入标签后回车" />

// ✅ 正确模式：自定义 tagRender 确保可读性
<Select
  mode="tags"
  placeholder="输入标签后回车"
  tagRender={({ label, closable, onClose }) => (
    <Tag
      closable={closable}
      onClose={onClose}
      className="bg-blue-100 text-blue-800 border-blue-300 mr-1 mb-1"
    >
      {label}
    </Tag>
  )}
/>
```

### 4.2 禁止输入框 focus 状态干扰输入

```tsx
// ❌ 错误模式：focus 样式过于醒目
<input className="focus:ring-4 focus:ring-blue-500 focus:border-blue-500" />

// ✅ 正确模式：使用适度的 focus 样式
<input className="focus:ring-2 focus:ring-blue-500 focus:border-transparent" />
```

---

## 5. 按钮状态反模式

### 5.1 禁止通知中心入口按钮无状态变化

```tsx
// ❌ 错误模式：按钮状态无明显区分
<Button icon={<Bell />} onClick={toggleNotifications}>
  通知
</Button>

// ✅ 正确模式：明确显示激活状态
<Badge count={unreadCount} size="small">
  <Button
    type={notificationsOpen ? 'primary' : 'text'}
    icon={<Bell className={notificationsOpen ? 'text-primary' : ''} />}
    onClick={() => setNotificationsOpen(!notificationsOpen)}
  />
</Badge>
```

### 5.2 禁止悬停状态无反馈

```tsx
// ❌ 错误模式：悬停状态无变化
<div className="p-4">内容</div>

// ✅ 正确模式：悬停有视觉反馈
<div className="p-4 hover:bg-gray-50 cursor-pointer transition-colors">内容</div>
```

---

## 6. 错误处理反模式

### 6.1 禁止链接/按钮无错误处理

```tsx
// ❌ 错误模式：可能导致页面报错
<Button onClick={() => router.push('/cmdb/topology')}>
  资产服务拓扑
</Button>

// ✅ 正确模式：添加错误处理
<Button
  onClick={async () => {
    try {
      await router.push('/cmdb/topology');
    } catch (error) {
      message.error('页面加载失败，请稍后重试');
    }
  }}
>
  资产服务拓扑
</Button>
```

### 6.2 禁止异步操作无 loading 状态

```tsx
// ❌ 错误模式：操作期间无反馈
<Button onClick={handleSubmit}>提交</Button>

// ✅ 正确模式：显示 loading 状态
<Button type="primary" loading={loading} onClick={handleSubmit}>
  提交
</Button>
```

---

## 7. 命名反模式

### 7.1 禁止使用模糊的 className 名称

```tsx
// ❌ 错误模式：无法从名称理解用途
<div className="box-1">...</div>
<div className="item">...</div>
<div className="temp">...</div>

// ✅ 正确模式：使用语义化的名称
<div className="stats-card">...</div>
<div className="user-permission-tag">...</div>
<div className="notification-badge">...</div>
```

### 7.2 禁止组件命名模糊

```tsx
// ❌ 错误模式
const Component1 = () => { ... }
const Box = () => { ... }
const Temp = () => { ... }

// ✅ 正确模式
const StatsOverview = () => { ... }
const PermissionTag = () => { ... }
const NotificationBadge = () => { ... }
```

---

## 8. 代码结构反模式

### 8.1 禁止大文件（超过 800 行）

```tsx
// ❌ 错误模式：单个文件超过 800 行
// ❌ 错误模式：单个组件超过 500 行

// ✅ 正确模式：拆分组件和工具函数
// - 页面组件：< 500 行
// - 业务组件：< 300 行
// - 工具函数：< 100 行
```

### 8.2 禁止硬编码颜色值（内联）

```tsx
// ❌ 错误模式：内联颜色值难以维护
<div style={{ color: '#1890ff' }}>文字</div>
<div style={{ backgroundColor: 'rgba(0,0,0,0.5)' }}>背景</div>

// ✅ 正确模式：使用设计系统 token 或 CSS 变量
<div className="text-primary">文字</div>
<div className="bg-black/50">背景</div>

// 或在 theme 中定义
const theme = {
  colors: {
    primary: '#1890ff',
  },
};
```

### 8.3 禁止魔法数字

```tsx
// ❌ 错误模式：使用未命名的数字
setTimeout(() => doSomething(), 3000);
const limit = data.length > 100 ? data.slice(0, 100) : data;

// ✅ 正确模式：使用命名常量
const REFRESH_INTERVAL_MS = 3000;
const MAX_DISPLAY_ITEMS = 100;

setTimeout(() => doSomething(), REFRESH_INTERVAL_MS);
const limit = data.length > MAX_DISPLAY_ITEMS ? data.slice(0, MAX_DISPLAY_ITEMS) : data;
```

---

## 9. Git 提交检查命令

```bash
# 检查是否包含开发状态文字
grep -rn "开发中\|Under construction\|开发状态\|逐步复用\|稳态渲染\|设计态与运行态" src --include="*.tsx"

# 检查是否有灰色透明背景
grep -rn "bg-gray-[0-9]\+/[0-4][0-9] text-white" src --include="*.tsx"

# 检查标签样式
grep -rn "bg-gray-[0-9]*/[0-4][0-9] text-white" src --include="*.tsx"

# 检查大文件（超过 800 行）
find src -name "*.tsx" -exec wc -l {} \; | awk '$1 > 800 {print $2, $1}'
```

---

## 10. 快速修复对照表

| 反模式 | 错误示例 | 正确示例 |
|--------|----------|----------|
| 透明背景+白字 | `bg-gray-100/50 text-white` | `bg-gray-900 text-white` |
| 开发状态提示 | `<ManagementNotice message="..." />` | 移除或功能描述 |
| 堆叠 Statistic | `<Card><Statistic/><Statistic/></Card>` | 拆分独立 Card |
| 默认 tags 样式 | `<Select mode="tags" />` | 添加 `tagRender` |
| 无错误处理 | `<Button onClick={navigate} />` | 添加 try-catch |
| 硬编码颜色 | `style={{ color: '#1890ff' }}` | 使用 CSS 类或变量 |
| 魔法数字 | `setTimeout(fn, 3000)` | `const INTERVAL = 3000` |