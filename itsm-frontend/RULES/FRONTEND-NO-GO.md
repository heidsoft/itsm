# ITSM 前端开发负面清单

> **版本**: 1.1
> **最后更新**: 2026-06-14
> **目的**: 列出前端开发中不应使用的反模式和坏习惯，供 AI 开发助手遵循。AI 开发任何新功能或修改现有代码前，必须先检查本清单。

---

## 0. 规则优先级

| 级别 | 含义 | AI 行为 |
|------|------|---------|
| P0 / CRITICAL | 会导致用户不可用或严重体验问题 | **必须修复**，禁止合入 |
| P1 / HIGH | 明显影响用户体验 | **必须修复**，尽量在同一次提交中解决 |
| P2 / MEDIUM | 可维护性或健壮性问题 | **应修复**，不阻塞但需跟进 |

---

## 0.5 看板卡片数字下方文字透明度

### 0.5.1 禁止数字下方次要文字使用透明背景配白色文字

**优先级**: P0

```tsx
// ❌ 错误模式：数字下方文字使用透明背景 + 白色文字
<Card>
  <Statistic value={123} />
  <Text className="text-white/60">处理中</Text>  {/* 对比度不足 */}
</Card>

// ❌ 错误模式：使用灰色透明背景
<Card className="bg-gray-100/30">
  <Text className="text-white">{count}</Text>
  <Text className="text-gray-300">个待处理</Text>
</Card>

// ✅ 正确模式：使用 Ant Design 文字类型，自动适配主题
<Statistic value={123} />
<Typography.Text type="secondary">处理中</Typography.Text>

// ✅ 正确模式：使用语义化颜色（非透明）
<Card>
  <Statistic value={123} />
  <span className="text-gray-500">处理中</span>  {/* 固定颜色，无透明度 */}
</Card>
```

**检测命令**:
```bash
grep -rn "text-white/[0-9] text-white\|text-gray-[0-9]*/[0-9] text-white\|bg-gray-[0-9]*/[0-4][0-9] text-white" src --include="*.tsx"
```

---

## 1. 配色与对比度反模式

### 1.1 [P0] 禁止灰色透明背景配白色文字

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

### 5.1 [P0] 禁止通知中心入口按钮无状态变化

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

### 5.2 [P0] 禁止 Select mode="tags" 使用默认标签样式

**问题**：输入文字确认后标签是灰色透明背景配白色文字，看不清楚。

```tsx
// ❌ 错误模式：使用默认标签样式，在深色背景上不可见
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

### 5.3 [P1] 禁止输入框 focus 状态出现蓝色边框干扰输入

**问题**：新建知识库文章标签输入框光标出现蓝色框。

```tsx
// ❌ 错误模式：focus 样式过于醒目或与 Ant Design 主题冲突
<Select className="focus:ring-4 focus:ring-blue-500" />

// ✅ 正确模式：使用适度的 focus 样式或依赖 Ant Design 默认样式
<Select className="focus:ring-2 focus:ring-primary focus:ring-offset-0" />

// ✅ 正确模式：禁用 focus ring，改用 border 颜色变化
<Select
  className="focus:border-primary border-gray-300 transition-colors"
  styles={{
    focus: { borderColor: 'var(--ant-primary-color)' },
  }}
/>
```

### 5.4 禁止悬停状态无反馈

```tsx
// ❌ 错误模式：悬停状态无变化
<div className="p-4">内容</div>

// ✅ 正确模式：悬停有视觉反馈
<div className="p-4 hover:bg-gray-50 cursor-pointer transition-colors">内容</div>
```

---

## 6. 错误处理反模式

### 6.1 [P0] 禁止链接/按钮无错误处理（导致页面报错）

**问题**：点击"资产服务拓扑"弹出报错。

```tsx
// ❌ 错误模式：可能导致页面报错，无任何错误处理
<Button onClick={() => router.push('/cmdb/topology')}>
  资产服务拓扑
</Button>

// ✅ 正确模式：添加 try-catch 和错误提示
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

// ✅ 正确模式：使用 disabled 状态防止错误操作（功能不可用时）
<Button
  disabled={!isTopologyAvailable}
  onClick={() => router.push('/cmdb/topology')}
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

## 9. Git 提交前自检命令

> AI 开发在每次提交前必须运行以下命令，确保无负面清单中的问题。

```bash
# === P0 检查（必须全部通过）===

# 检查是否包含开发状态文字
grep -rEn "开发中|Under construction|开发状态|逐步复用|稳态渲染|设计态与运行态|复杂域页面|统一收口" src --include="*.tsx"

# 检查是否有灰色透明背景 + 白色文字
grep -rEn "bg-gray-[0-9]+/[0-4][0-9] text-white|text-white/[0-9] text-white" src --include="*.tsx"

# 检查是否有通知中心按钮缺少状态区分（排除已有 Badge 的）
grep -rEn "<Button[^>]*icon=\{<Bell" src --include="*.tsx" | grep -v "Badge\|notificationsOpen\|type=.*primary"

# 检查 Select mode="tags" 是否缺少 tagRender
grep -rEn 'mode="tags"' src --include="*.tsx" | while read f; do
  file=$(echo "$f" | cut -d: -f1)
  if ! grep -q "tagRender" "$file"; then
    echo "MISSING tagRender: $file"
  fi
done

# === P1 检查 ===

# 检查 focus ring 样式是否过于醒目
grep -rEn "focus:ring-4|focus:ring-[5-9]" src --include="*.tsx"

# 检查看板卡片数字下方文字是否使用透明背景
grep -rEn "text-white/[0-9]|bg-gray-[0-9]+/[0-3][0-9]" src --include="*.tsx"

# === P2 检查 ===

# 检查大文件（超过 800 行）
find src -name "*.tsx" -exec wc -l {} \; | awk '$1 > 800 {print $2, $1" lines"}'

# 检查无错误处理的导航按钮
grep -rEn "onClick=\{.*router\.push" src --include="*.tsx" | grep -v "try\|catch\|error\|message\." | grep -v "disabled="
```

---

## 10. 快速修复对照表

| 优先级 | 反模式 | 错误示例 | 正确示例 |
|--------|--------|----------|----------|
| P0 | 透明背景+白字 | `bg-gray-100/50 text-white` | `bg-gray-900 text-white` |
| P0 | 开发状态提示 | `<ManagementNotice message="..." />` | 移除或功能描述 |
| P0 | 无 tagRender | `<Select mode="tags" />` | 添加 `tagRender` 自定义样式 |
| P0 | 无通知状态 | `<Button icon={<Bell />}>` | 加 `Badge` + `type` 区分 |
| P0 | 无错误处理导航 | `<Button onClick={push}>` | 加 `try-catch` |
| P1 | 看板卡片透明文字 | `text-white/60` | `type="secondary"` |
| P1 | focus ring 过强 | `focus:ring-4` | `focus:ring-2` |
| P2 | 堆叠 Statistic | `<Card><Statistic/><Statistic/></Card>` | 拆分独立 Card |
| P2 | 硬编码颜色 | `style={{ color: '#1890ff' }}` | 使用 CSS 类或变量 |