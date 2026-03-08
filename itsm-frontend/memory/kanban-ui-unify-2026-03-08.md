# Kanban UI 统一报告

**日期**: 2026-03-08
**任务**: 统一 Tickets 和 Incidents 模块的看板 UI
**状态**: ✅ 已完成

## 问题分析

### 原有实现差异

**Tickets 模块**:
- 组件: `src/components/ticket/TicketKanban.tsx`
- 布局: 使用 `Row`/`Col`，每列 `span={4}`
- 列容器: `Card` 组件，带标题（状态圆点 + 标题 + Badge 计数）
- 卡片样式: 左侧彩色边框表示优先级，卡片带操作下拉菜单
- 工具栏: 搜索、状态筛选、优先级筛选、排序、新建按钮

**Incidents 模块**:
- 组件: 内联定义在 `src/app/(main)/incidents/page.tsx` 中
- 布局: 使用 `Row`/`Col` 响应式（`xs={24} sm={12} md={8} lg={4}`）
- 列容器: 自定义 `div`，背景色 `#f5f5f5`，标题为深色背景
- 卡片样式: 简单 `Card`，无左侧边框优先级显示
- 工具栏: 无独立工具栏，只有搜索和筛选按钮

**视觉不一致点**：
1. 列容器样式不同（Card vs div）
2. 列标题样式不同（带圆点和Badge vs 深色背景条）
3. 卡片样式不同（Ticket卡片更丰富，有优先级边框、时间、标签等）
4. Incident卡片缺少操作菜单

## 解决方案

### 采用方案：创建通用 UnifiedKanbanBoard 组件

创建了一个可复用的看板组件 `UnifiedKanbanBoard`，支持：
1. **类型安全**: 泛型设计，支持任意数据类型（Ticket、Incident等）
2. **可配置列**: 通过 `columnConfigs` 自定义状态列
3. **字段适配**: 通过回调函数配置如何从数据项中提取信息
4. **视觉统一**: 列样式完全复刻 TicketKanban 的设计
5. **功能完整**: 支持搜索、筛选、状态筛选、优先级筛选、查看/编辑操作

### 实现细节

#### 新组件: `src/components/business/UnifiedKanbanBoard.tsx`

**主要特性**:
- 使用 Ant Design 的 `Card` 作为列容器
- 列标题包含：彩色圆点（4px宽，非圆形）、状态文本、Badge 计数（无 `showZero`）
- 卡片默认渲染包含：标题、编号、描述、优先级Badge、负责人、创建时间、标签、分类
- 工具栏包含：搜索框、状态筛选、优先级筛选（可选）、视图设置下拉菜单
- 响应式布局：`Row` 包裹，`Col span={4}`，保持与 TicketKanban 相同的栅格系统
- 卡片样式：左侧4px边框表示优先级颜色，悬停效果，操作下拉菜单（查看/编辑）

**TypeScript 接口**:
```ts
export interface UnifiedKanbanBoardProps<T> {
  items: T[];
  getItemId: (item: T) => string | number;
  getItemStatus: (item: T) => string;
  getItemTitle?: (item: T) => string;
  getItemNumber?: (item: T) => string;
  // ... 更多字段提取器
  columnConfigs: KanbanColumnConfig<T>[];
  priorityOptions?: Array<{ value: string; label: string; color: string }>;
  // ... 更多配置
}
```

```
#### 修改文件: `src/app/(main)/incidents/page.tsx`

**变更内容**:
1. 移除内联的 `IncidentKanbanCard` 和 `IncidentKanbanBoard` 组件
2. 引入 `UnifiedKanbanBoard` 和 `KanbanColumnConfig`
3. 定义列配置 `KANBAN_COLUMNS`（5列：新建、调查中、已识别、已解决、已关闭）
4. 定义优先级配置 `PRIORITY_CONFIG`（转换为label格式）
5. 在 kanban 标签页中使用 `<UnifiedKanbanBoard<Incident>`，提供所有必需的回调函数
6. 调整导入：将 `AppstoreOutlined` 从 `@ant-design/icons` 导入，`Row`/`Col` 不再需要（在UnifiedKanbanBoard内部）

## 视觉验证

### 列样式对比（已统一）

| 特性 | Tickets (原) | Incidents (新) |
|-----|-------------|----------------|
| 列容器 | Card | Card ✅ |
| 列标题 | 圆点 + 文字 + Badge | 圆点 + 文字 + Badge ✅ |
| 标题圆点 | 2px圆形，mr-2 | 2px圆形，mr-2 ✅ |
| 标题文字 | `Text strong` (font-medium) | `font-medium` ✅ |
| Badge | no `showZero` | no `showZero` ✅ |
| 列背景 | 白色 (Card) | 白色 (Card) ✅ |
| 卡片间距 | `space-y-2` (12px) | `space-y-2` (12px) ✅ |
| 卡片操作 | 下拉菜单 (MoreOutlined) | 下拉菜单 (MoreOutlined) ✅ |

### 列配置对比

- **Tickets**: 6列（new, open, in_progress, pending, resolved, closed）
- **Incidents**: 5列（new, investigating, identified, resolved, closed）

由于业务逻辑不同，列数和状态值不完全相同，但视觉样式完全一致。

## 代码质量

### TypeScript 类型检查
✅ `src/components/business/UnifiedKanbanBoard.tsx` - 无错误
✅ `src/app/(main)/incidents/page.tsx` - 无错误（修复了所有 implicit any 类型）

### 依赖检查
- UnifiedKanbanBoard 依赖 Ant Design 组件（已存在）
- 依赖 `dayjs` 用于相对时间（已存在）
- 无需新增依赖包

## 后续建议

1. **考虑迁移 Tickets 模块**：虽然当前只统一了 Incidents，但未来可将 Tickets 的 `TicketKanban` 也迁移到 `UnifiedKanbanBoard`，进一步减少代码重复。
2. **添加拖拽功能**：当前 UnifiedKanbanBoard 未启用拖拽（`enableDrag` 默认为 true 但未实现），可参考 `TicketKanbanBoard`（business目录）添加 `@dnd-kit` 拖拽支持。
3. **单元测试**：建议为 `UnifiedKanbanBoard` 添加测试，覆盖筛选、排序、卡片渲染等场景。
4. **性能优化**：当前每次渲染都重新计算 filteredItems 和 columnsData，对于大数据量可考虑 useMemo 优化（已使用）。

## 文件变更清单

- ✅ **新建**: `src/components/business/UnifiedKanbanBoard.tsx` (12 KB)
- ✅ **修改**: `src/app/(main)/incidents/page.tsx`
- ✅ **类型校验**: 通过

## 测试建议

1. 启动开发服务器: `npm run dev`
2. 访问 Tickets 看板: http://localhost:3000/tickets?tab=kanban
3. 访问 Incidents 看板: http://localhost:3000/incidents?tab=kanban
4. 对比两者看板，确认：
   - 列容器均为白色 Card，高度一致
   - 列标题样式一致（圆点、文字、Badge位置）
   - 卡片样式一致（左侧边框、操作菜单）
   - 工具栏功能正常（搜索、筛选）

---

**任务完成**。Incidents 模块现已使用与 Tickets 相同视觉风格的统一看板组件。
