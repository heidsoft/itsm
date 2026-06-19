# 设计令牌使用指南

## 目的
统一使用设计令牌，替换内联样式，提升代码可维护性和设计一致性。

## 设计令牌映射表

### 1. 颜色 (Colors)

| 内联样式 | 设计令牌 | 用途 |
|---------|---------|------|
| `color: '#3b82f6'` | `colors.accent` | 强调色、链接色 |
| `color: '#0f172a'` | `colors.primary` | 主色调 |
| `color: '#10b981'` | `colors.success` | 成功状态 |
| `color: '#f59e0b'` | `colors.warning` | 警告状态 |
| `color: '#ef4444'` | `colors.danger` | 危险、错误状态 |
| `color: '#1e293b'` | `colors.text` | 主文本色 |
| `color: '#64748b'` | `colors.textMuted` | 次要文本色 |
| `background: '#ffffff'` | `colors.surface` | 卡片背景 |
| `background: '#f8fafc'` | `colors.bgSubtle` | 页面背景 |
| `borderColor: '#e2e8f0'` | `colors.border` | 边框颜色 |

### 2. 间距 (Spacing)

| 内联样式 | 设计令牌 | 用途 |
|---------|---------|------|
| `padding: '4px'` | `spacing.xs` | 极小间距 |
| `padding: '8px'` | `spacing.sm` | 小间距 |
| `padding: '16px'` | `spacing.md` | 中等间距 |
| `padding: '24px'` | `spacing.lg` | 大间距 |
| `padding: '32px'` | `spacing.xl` | 超大间距 |
| `margin: '16px'` | `spacing.md` | 中等外边距 |
| `margin: '24px'` | `spacing.lg` | 大外边距 |

### 3. 圆角 (Border Radius)

| 内联样式 | 设计令牌 | 用途 |
|---------|---------|------|
| `borderRadius: '8px'` | `radius.sm` | 小圆角 |
| `borderRadius: '12px'` | `radius.md` | 中等圆角 |
| `borderRadius: '16px'` | `radius.lg` | 大圆角 |
| `borderRadius: '9999px'` | `radius.full` | 完整圆角（按钮、头像） |

### 4. 阴影 (Shadows)

| 内联样式 | 设计令牌 | 用途 |
|---------|---------|------|
| `boxShadow: '0 1px 3px 0 rgba(0, 0, 0, 0.1)'` | `shadows.card` | 卡片阴影 |
| `boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)'` | `shadows.cardHover` | 卡片悬停阴影 |
| `boxShadow: '0 10px 40px -10px rgba(0,0,0,0.15)'` | `shadows.dropdown` | 下拉菜单阴影 |

### 5. 字体 (Typography)

| 内联样式 | 设计令牌 | 用途 |
|---------|---------|------|
| `fontSize: '0.75rem'` | `typography.fontSize.xs` | 极小字体 |
| `fontSize: '0.875rem'` | `typography.fontSize.sm` | 小字体 |
| `fontSize: '1rem'` | `typography.fontSize.base` | 基础字体 |
| `fontSize: '1.125rem'` | `typography.fontSize.lg` | 大字体 |
| `fontSize: '1.25rem'` | `typography.fontSize.xl` | 超大字体 |
| `fontFamily: 'Inter, ...'` | `typography.fontFamily.sans` | 无衬线字体 |
| `fontFamily: 'JetBrains Mono, ...'` | `typography.fontFamily.mono` | 等宽字体 |

### 6. 过渡 (Transitions)

| 内联样式 | 设计令牌 | 用途 |
|---------|---------|------|
| `transition: '150ms ease'` | `transitions.fast` | 快速过渡 |
| `transition: '300ms ease'` | `transitions.normal` | 正常过渡 |
| `transition: '500ms ease'` | `transitions.slow` | 慢速过渡 |

## 使用示例

### ❌ 错误示例（内联样式）
```tsx
<Card style={{
  padding: '24px',
  borderRadius: '12px',
  background: '#ffffff',
  boxShadow: '0 1px 3px 0 rgba(0, 0, 0, 0.1)'
}}>
  <Text style={{ color: '#1e293b', fontSize: '1.125rem' }}>标题</Text>
</Card>
```

### ✅ 正确示例（设计令牌）
```tsx
import { DESIGN } from '@/design-system/tokens';

<Card style={{
  padding: DESIGN.spacing.lg,
  borderRadius: DESIGN.radius.md,
  background: DESIGN.colors.surface,
  boxShadow: DESIGN.shadows.card
}}>
  <Text style={{ 
    color: DESIGN.colors.text, 
    fontSize: DESIGN.typography.fontSize.lg 
  }}>标题</Text>
</Card>
```

### 🎯 最佳实践（CSS Modules）
```tsx
// Component.module.css
.card {
  padding: var(--spacing-lg);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  box-shadow: var(--shadow-card);
}

.title {
  color: var(--color-text);
  font-size: var(--font-size-lg);
}

// Component.tsx
<Card className={styles.card}>
  <Text className={styles.title}>标题</Text>
</Card>
```

## 替换策略

### 阶段1: 新代码使用设计令牌
- 所有新代码必须使用设计令牌
- Code Review时检查内联样式使用

### 阶段2: 逐步替换现有内联样式
- 优先替换高频使用的组件
- 按模块逐步替换

### 阶段3: 建立CSS Module系统
- 将复用样式提取到CSS Module
- 建立全局样式变量

## 检查工具

### 自动化检查
```bash
# 检查内联样式数量
grep -rn 'style={{' src --include='*.tsx' | wc -l

# 查找特定颜色值
grep -rn "#3b82f6\|#0f172a\|#ef4444" src --include='*.tsx'

# 查找硬编码间距
grep -rn "padding: '[0-9]px'\|margin: '[0-9]px'" src --include='*.tsx'
```

### ESLint规则
```json
{
  "rules": {
    "react/style-prop-object": "warn",
    "no-inline-styles/no-inline-styles": "warn"
  }
}
```

## 注意事项

1. **性能考虑**: 设计令牌不会影响运行时性能
2. **主题切换**: 使用设计令牌可以轻松实现主题切换
3. **维护性**: 集中管理样式，修改更方便
4. **一致性**: 确保整个应用视觉一致

## 下一步行动

1. 安装ESLint插件检查内联样式
2. 创建CSS Module模板
3. 逐步替换高频组件的内联样式
4. 建立样式Code Review规范

---

**更新日期**: 2026-04-26  
**维护者**: ITSM前端团队
