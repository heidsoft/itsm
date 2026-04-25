# 布局状态管理统一设计

## 问题背景

当前 ITSM 前端存在两套布局实现：
1. `src/app/(main)/layout.tsx` - 主布局
2. `src/components/layout/AppLayout.tsx` - 可复用布局组件

**核心问题：**
- `collapsed` 状态通过 props 层层传递：layout → Header → Sidebar
- 两处重复的响应式逻辑（isMobile 检测）
- 状态分散，难以维护

## 解决方案

采用 Zustand Store 统一管理布局状态，与现有 `auth-store` 风格一致。

## 设计细节

### 1. Store 结构

```typescript
// src/lib/store/layout-store.ts

import { create } from 'zustand';

interface LayoutState {
  // 核心状态
  collapsed: boolean;

  // 操作
  toggleCollapsed: () => void;
  setCollapsed: (collapsed: boolean) => void;
}

export const useLayoutStore = create<LayoutState>((set) => ({
  collapsed: false,

  toggleCollapsed: () => set((state) => ({
    collapsed: !state.collapsed
  })),

  setCollapsed: (collapsed) => set({ collapsed }),
}));
```

**设计决策：**
- 不使用 persist 中间件（布局状态无需跨会话持久化）
- `isMobile` 等响应式状态保留在 `useResponsive` hook
- Store 保持纯粹的状态管理

### 2. 组件改造

**改造前：**
```tsx
// layout.tsx
const [collapsed, setCollapsed] = useState(false);
<Header collapsed={collapsed} onCollapse={setCollapsed} />
<Sidebar collapsed={collapsed} onCollapse={setCollapsed} />
```

**改造后：**
```tsx
// layout.tsx
<Header />
<Sidebar />

// Header.tsx
const { collapsed, toggleCollapsed } = useLayoutStore();

// Sidebar.tsx
const { collapsed, setCollapsed } = useLayoutStore();
```

### 3. 响应式逻辑

移动端自动折叠逻辑保留在 `layout.tsx`：

```tsx
const { isMobile } = useResponsive();
const { setCollapsed } = useLayoutStore();

useEffect(() => {
  if (isMobile) {
    setCollapsed(true);
  }
}, [isMobile, setCollapsed]);
```

## 文件变更

| 文件 | 操作 | 说明 |
|------|------|------|
| `src/lib/store/layout-store.ts` | 新建 | 布局状态 Store |
| `src/lib/store/index.ts` | 修改 | 导出新 Store |
| `src/app/(main)/layout.tsx` | 修改 | 使用 Store，移除 useState |
| `src/components/layout/header/Header.tsx` | 修改 | 从 Store 获取状态 |
| `src/components/layout/sidebar/Sidebar.tsx` | 修改 | 从 Store 获取状态 |
| `src/components/layout/AppLayout.tsx` | 修改 | 统一使用 Store |

## 测试策略

### TDD 测试用例

1. **Store 初始状态**
   - 初始 collapsed 为 false

2. **toggleCollapsed 操作**
   - 切换 collapsed 从 false 到 true
   - 切换 collapsed 从 true 到 false

3. **setCollapsed 操作**
   - 设置 collapsed 为 true
   - 设置 collapsed 为 false

4. **组件集成**
   - Header 可直接访问 Store
   - Sidebar 可直接访问 Store
   - 状态变化同步到所有消费者

## 风险评估

- **低风险**：改动范围可控，主要是状态管理方式变化
- **向后兼容**：不改变用户可见行为
- **渐进式**：可分步实施，每步可验证

## 成功标准

- [ ] Store 测试覆盖率 >= 80%
- [ ] 无 props 传递 collapsed/onCollapse
- [ ] 折叠/展开功能正常
- [ ] 移动端自动折叠正常
- [ ] TypeScript 类型检查通过
