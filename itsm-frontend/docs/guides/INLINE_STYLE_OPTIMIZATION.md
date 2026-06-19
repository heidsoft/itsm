# 内联样式优化方案

## 现状分析

### 统计数据
- **app目录**: ~200处内联样式
- **components目录**: ~793处内联样式
- **总计**: 约1000处内联样式

### 内联样式类型分布

#### 1. 布局相关（约40%）
```tsx
style={{ width: '100%', padding: '24px', margin: '16px' }}
```

#### 2. 颜色相关（约30%）
```tsx
style={{ color: '#3b82f6', background: '#ffffff' }}
```

#### 3. 响应式相关（约20%）
```tsx
style={{ marginLeft: isMobile ? 0 : 280 }}
```

#### 4. 其他（约10%）
```tsx
style={{ borderRadius: '8px', boxShadow: '0 2px 8px' }}
```

## 优化策略

### 阶段1: 创建CSS Module基础设施

#### 创建全局样式变量
```css
/* src/styles/variables.module.css */

/* 颜色 */
:root {
  --color-primary: #0f172a;
  --color-accent: #3b82f6;
  --color-success: #10b981;
  --color-warning: #f59e0b;
  --color-danger: #ef4444;
  --color-surface: #ffffff;
  --color-border: #e2e8f0;
  --color-text: #1e293b;
  --color-text-muted: #64748b;
}

/* 间距 */
:root {
  --spacing-xs: 4px;
  --spacing-sm: 8px;
  --spacing-md: 16px;
  --spacing-lg: 24px;
  --spacing-xl: 32px;
}

/* 圆角 */
:root {
  --radius-sm: 4px;
  --radius-md: 6px;
  --radius-lg: 8px;
  --radius-xl: 12px;
}
```

#### 创建通用样式类
```css
/* src/styles/utilities.module.css */

/* 布局 */
.fullWidth {
  width: 100%;
}

.flex1 {
  flex: 1;
}

/* 间距 */
.paddingMd {
  padding: var(--spacing-md);
}

.marginBottomMd {
  margin-bottom: var(--spacing-md);
}

/* 文本 */
.textAccent {
  color: var(--color-accent);
}

.textMuted {
  color: var(--color-text-muted);
}
```

### 阶段2: 分批替换策略

#### 优先级1: 高频组件（预计减少200处）
- PageContainer
- Layout组件
- 表单组件
- 列表组件

#### 优先级2: 业务组件（预计减少300处）
- Ticket相关组件
- Dashboard组件
- 报表组件

#### 优先级3: 其他组件（预计减少300处）
- UI组件
- 工具组件

### 阶段3: 响应式样式处理

#### 使用Tailwind CSS类
```tsx
/* 替换前 */
<div style={{ marginLeft: isMobile ? 0 : 280 }}>

/* 替换后 */
<div className="ml-0 lg:ml-[280px]">
```

#### 使用CSS Module
```tsx
/* 替换前 */
<div style={{ padding: collapsed ? '16px' : '24px' }}>

/* 替换后 */
<div className={collapsed ? styles.compact : styles.normal}>
```

## 替换工具

### 自动化脚本
```bash
#!/bin/bash
# scripts/replace-inline-styles.sh

# 统计内联样式
echo "统计内联样式数量:"
grep -rn 'style={{' src --include='*.tsx' | wc -l

# 查找特定模式
echo "\n查找宽度样式:"
grep -rn "style={{ width:" src --include='*.tsx' | head -20

echo "\n查找颜色样式:"
grep -rn "style={{ color:" src --include='*.tsx' | head -20

# 生成替换报告
echo "\n生成替换报告:"
grep -rn 'style={{' src --include='*.tsx' > inline-styles-report.txt
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

## 替换示例

### 示例1: PageContainer

#### 替换前
```tsx
<div style={{ padding: '24px', background: '#fff', minHeight: '100%' }}>
  <Breadcrumb items={header.breadcrumb.items} style={{ marginBottom: '8px' }} />
  <Title level={4} style={{ margin: 0 }}>标题</Title>
  <Divider style={{ margin: '0 0 24px 0' }} />
</div>
```

#### 替换后
```tsx
import styles from './PageContainer.module.css';

<div className={styles.container}>
  <Breadcrumb items={header.breadcrumb.items} className={styles.breadcrumb} />
  <Title level={4} className={styles.title}>标题</Title>
  <Divider className={styles.divider} />
</div>
```

```css
/* PageContainer.module.css */
.container {
  padding: var(--spacing-lg);
  background: var(--color-surface);
  min-height: 100%;
}

.breadcrumb {
  margin-bottom: var(--spacing-sm);
}

.title {
  margin: 0;
}

.divider {
  margin: 0 0 var(--spacing-lg) 0;
}
```

### 示例2: 颜色样式

#### 替换前
```tsx
<FileText size={16} style={{ color: '#3b82f6' }} />
```

#### 替换后
```tsx
<FileText size={16} className="text-accent" />
```

```css
/* utilities.module.css */
.text-accent {
  color: var(--color-accent);
}
```

## 验证和测试

### 单元测试
```tsx
describe('样式优化验证', () => {
  it('应该减少内联样式数量', () => {
    const inlineStyles = grepInlineStyles();
    expect(inlineStyles.length).toBeLessThan(100);
  });

  it('应该使用CSS变量', () => {
    const cssModules = findCSSModules();
    expect(cssModules.length).toBeGreaterThan(0);
  });
});
```

### 视觉回归测试
```bash
# 运行视觉回归测试
npm run test:e2e:visual
```

## 进度跟踪

### 目标设定
- **当前**: ~1000处内联样式
- **阶段1目标**: < 500处 (减少50%)
- **最终目标**: < 100处 (减少90%)

### 检查命令
```bash
# 检查进度
echo "当前内联样式数量:"
grep -rn 'style={{' src --include='*.tsx' | wc -l

# 检查CSS Module数量
echo "\nCSS Module文件数量:"
find src -name '*.module.css' | wc -l

# 检查设计令牌使用
echo "\n设计令牌使用次数:"
grep -rn 'DESIGN\.' src --include='*.tsx' | wc -l
```

## 执行计划

### Week 1: 基础设施
- [x] 创建CSS变量文件
- [x] 创建通用样式类
- [x] 配置ESLint规则
- [x] 创建替换脚本

### Week 2: 高频组件
- [ ] 替换PageContainer
- [ ] 替换Layout组件
- [ ] 替换表单组件

### Week 3-4: 业务组件
- [ ] 替换Ticket组件
- [ ] 替换Dashboard组件
- [ ] 替换报表组件

### Week 5-6: 验证优化
- [ ] 视觉回归测试
- [ ] 性能测试
- [ ] 文档更新

---

**更新日期**: 2026-04-26  
**负责人**: 前端优化团队
