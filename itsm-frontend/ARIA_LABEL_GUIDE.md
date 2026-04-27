# ARIA标签覆盖指南

## 目的
增强可访问性，确保所有交互元素都有适当的ARIA标签，支持屏幕阅读器和键盘导航。

## ARIA标签最佳实践

### 1. 表单元素

#### ✅ 正确示例
```tsx
<Input
  placeholder="请输入用户名"
  aria-label="用户名输入框"
  aria-required="true"
  aria-invalid={hasError ? 'true' : 'false'}
  aria-describedby="username-help"
/>
<Form.Item
  label="用户名"
  name="username"
  rules={[{ required: true, message: '请输入用户名' }]}
>
  <Input 
    placeholder="请输入用户名" 
    aria-label="用户名输入框"
  aria-required="true"
  aria-describedby="username-help-text"
  aria-invalid={errors.username ? 'true' : 'false'}
  aria-errormessage="username-error"
  />
  <div id="username-help-text" className="text-sm text-gray-500">
    用户名只能包含字母和数字
  </div>
  {errors.username && (
    <div id="username-error" role="alert" className="text-red-500">
      {errors.username.message}
    </div>
  )}
</Form.Item>
```

#### ❌ 错误示例
```tsx
<Input placeholder="用户名" />
<Input /> {/* 缺少label和aria-label */}
```

### 2. 按钮和链接

#### ✅ 正确示例
```tsx
<Button 
  type="primary" 
  icon={<PlusOutlined />}
  aria-label="新建工单"
>
  新建
</Button>

<Button
  icon={<EditOutlined />}
  onClick={handleEdit}
  aria-label="编辑工单"
/>

<a 
  href="/tickets" 
  aria-label="查看所有工单"
>
  工单列表
</a>
```

#### ❌ 错误示例
```tsx
<Button icon={<PlusOutlined />} /> {/* 缺少aria-label */}
<a href="/tickets">更多</a> {/* 链接文本不明确 */}
```

### 3. 表格和列表

#### ✅ 正确示例
```tsx
<Table
  dataSource={data}
  columns={columns}
  aria-label="工单列表表格"
  aria-describedby="table-description"
/>
<div id="table-description" className="sr-only">
  包含所有工单的详细信息，包括标题、状态、优先级等
</div>

<List
  dataSource={items}
  renderItem={(item) => (
    <List.Item role="listitem" aria-label={`工单 ${item.title}`}>
      {item.title}
    </List.Item>
  )}
  role="list"
  aria-label="工单列表"
/>
```

### 4. 对话框和模态框

#### ✅ 正确示例
```tsx
<Modal
  title="新建工单"
  open={visible}
  onCancel={handleClose}
  aria-labelledby="modal-title"
  aria-describedby="modal-description"
  role="dialog"
  aria-modal="true"
>
  <h2 id="modal-title">新建工单</h2>
  <p id="modal-description">请填写工单详细信息</p>
  <Form>
    {/* 表单内容 */}
  </Form>
</Modal>
```

### 5. 图标和视觉元素

#### ✅ 正确示例
```tsx
<AlertCircle 
  className="w-5 h-5" 
  aria-label="错误提示图标"
  role="img"
/>

<Badge 
  count={5} 
  aria-label="5个未读通知"
  role="status"
/>

<Progress 
  percent={75} 
  aria-label="进度75%"
  role="progressbar"
  aria-valuenow={75}
  aria-valuemin={0}
  aria-valuemax={100}
/>
```

### 6. 动态内容

#### ✅ 正确示例
```tsx
<div 
  role="status" 
  aria-live="polite"
  aria-label="加载状态"
>
  {loading ? '加载中...' : '加载完成'}
</div>

<div
  role="alert"
  aria-live="assertive"
  aria-label="错误提示"
>
  {error && `错误：${error}`}
</div>
```

## 常用ARIA属性

| 属性 | 用途 | 示例 |
|------|------|------|
| `aria-label` | 元素的可访问名称 | `aria-label="提交按钮"` |
| `aria-labelledby` | 引用标签元素 | `aria-labelledby="title-id"` |
| `aria-describedby` | 引用描述元素 | `aria-describedby="help-text"` |
| `aria-required` | 标记必填项 | `aria-required="true"` |
| `aria-invalid` | 标记无效输入 | `aria-invalid="true"` |
| `aria-disabled` | 标记禁用状态 | `aria-disabled="true"` |
| `aria-hidden` | 隐藏装饰性元素 | `aria-hidden="true"` |
| `aria-live` | 动态内容更新 | `aria-live="polite"` |
| `role` | 定义元素角色 | `role="button"` |

## 屏幕阅读器测试

### NVDA (Windows)
- 下载: https://www.nvaccess.org/
- 快捷键: `Insert + Space` 开始/停止朗读
- 导航: `H` 键跳过标题，`Tab` 键导航

### VoiceOver (macOS/iOS)
- 开启: `Cmd + F5`
- 导航: `Ctrl + Option + 箭头键`
- 朗读: `Ctrl + Option + A`

### JAWS (Windows)
- 下载: https://www.freedomscientific.com/
- 导航: `Tab` 键，`H` 键跳过标题

## 检查工具

### 自动化工具
```bash
# 检查缺失的aria-label
grep -rn '<Button' src --include='*.tsx' | grep -v 'aria-label'
grep -rn '<Input' src --include='*.tsx' | grep -v 'aria-label'
grep -rn '<Select' src --include='*.tsx' | grep -v 'aria-label'

# 检查图标元素
grep -rn '<LucideIcon' src --include='*.tsx' | grep -v 'aria-label'
```

### 浏览器扩展
- **axe DevTools**: 自动检测可访问性问题
- **WAVE**: Web可访问性评估工具
- **Lighthouse**: Chrome内置可访问性审计

### 代码检查
```json
// .eslintrc.json
{
  "plugins": ["jsx-a11y"],
  "rules": {
    "jsx-a11y/alt-text": "error",
    "jsx-a11y/aria-props": "error",
    "jsx-a11y/aria-proptypes": "error",
    "jsx-a11y/aria-unsupported-elements": "error",
    "jsx-a11y/role-has-required-aria-props": "error",
    "jsx-a11y/role-supports-aria-props": "error"
  }
}
```

## 键盘导航

### 基本原则
1. 所有交互元素可通过Tab键访问
2. Enter键激活链接和按钮
3. Escape键关闭对话框和菜单
4. 箭头键用于列表和菜单导航

### 示例
```tsx
<div
  role="button"
  tabIndex={0}
  onKeyDown={(e) => {
    if (e.key === 'Enter' || e.key === ' ') {
      handleClick();
    }
  }}
  aria-label="自定义按钮"
>
  点击我
</div>
```

## 实施计划

### 阶段1: 关键交互元素
- [ ] 所有按钮添加aria-label
- [ ] 所有表单输入添加aria-label
- [ ] 所有链接添加明确的文本或aria-label

### 阶段2: 数据展示元素
- [ ] 表格添加aria-label和role
- [ ] 列表添加aria-label
- [ ] 图标添加aria-label或aria-hidden

### 阶段3: 动态内容
- [ ] 加载状态添加aria-live
- [ ] 错误提示添加role="alert"
- [ ] 通知添加role="status"

## 验证清单

- [ ] 所有表单字段有标签或aria-label
- [ ] 所有按钮有可识别的文本或aria-label
- [ ] 所有图片有alt文本或aria-hidden
- [ ] 颜色对比度符合WCAG AA标准 (4.5:1)
- [ ] 键盘导航完全可用
- [ ] 屏幕阅读器测试通过
- [ ] 动态内容有适当的aria-live属性

---

**更新日期**: 2026-04-26  
**维护者**: ITSM前端团队
