# 清理剩余 Mock 数据并执行全面功能检查

## 目标

响应用户 "回顾检查产品缺陷，并测试产品功能，确保产品达到正式产品程度" 的要求，并执行 "禁止mock需要与后端api对接" 的策略。
搜索结果显示，除了已经处理的核心模块（Change, Problem, Dashboard Hook）外，系统中仍存在以下 Mock 数据：

1. **Ticket Analytics (`src/app/(main)/tickets/analytics/page.tsx`)**: 包含 `getMockData`。
2. **Role API (`src/lib/api/role-api.ts`)**: 包含 `MOCK_ROLES` 和回退逻辑。
3. **Ticket Association (`src/components/business/TicketAssociation.tsx`)**: 包含 `loadMockData`。
4. **Mock Data File (`src/app/lib/mock-data.ts`)**: 整个文件都是 Mock 数据，虽然可能未被核心流程使用，但应确认并清理。
5. **Test Utils (`src/lib/test-utils.tsx`)**: 测试工具中的 Mock 数据可以保留，因为它们用于单元测试。

## 执行计划

### 第一阶段：清理剩余 Mock 数据

**工单分析页面**: 修改 `src/app/(main)/tickets/analytics/page.tsx`，移除 `getMockData`，改为初始空状态，依赖 API。

* **角色管理 API**: 修改 `src/lib/api/role-api.ts`，移除 `MOCK_ROLES` 和回退逻辑。
* **工单关联组件**: 修改 `src/components/business/TicketAssociation.tsx`，移除 `loadMockData`，确保依赖 API。
* **Mock 数据文件**: 检查 `src/app/lib/mock-data.ts` 的引用情况，如果不再被页面直接使用，则保留作为参考或删除（建议保留但标记为仅限开发参考，或者如果用户要求严格则删除）。鉴于用户要求“正式产品程度”，建议**删除**该文件以防止意外引用。

### 第二阶段：功能验证（根据 Todo 列表）

在清理完 Mock 数据后，按照 Todo 列表逐一检查各模块功能。由于我无法直接点击浏览器，我将通过检查代码逻辑和可能的 API 调用点来“模拟”验证过程。

## 待办事项更新

我将添加清理剩余 Mock 数据的任务。
