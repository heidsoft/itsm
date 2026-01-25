# 全面移除 Mock 数据，强制对接后端 API

## 目标
根据用户“禁止mock需要与后端api对接”的要求，我将移除系统中所有已知的 Mock 数据回退机制，确保前端页面完全依赖后端 API。如果 API 调用失败，页面应显示错误或空状态，而不是虚假的演示数据。

## 修改计划

### 1. 变更管理 (Change Management)
- **文件**: `src/modules/change/components/ChangeList.tsx`
- **操作**:
    - 删除 `mockChanges` 常量定义。
    - 修改 `loadData` 函数：在 `catch` 块中移除 `setData(mockChanges)` 等回退逻辑，仅保留错误日志和提示。

### 2. 问题管理 (Problem Management)
- **文件**: `src/modules/problem/components/ProblemList.tsx`
- **操作**:
    - 删除 `mockProblems` 常量定义。
    - 修改 `loadData` 函数：在 `catch` 块中移除 `setData(mockProblems)` 等回退逻辑。

### 3. 仪表盘 (Dashboard)
- **文件**: `src/app/(main)/dashboard/hooks/useDashboardData.ts`
- **操作**:
    - 将 `getMockData` 函数重构为 `getInitialData`，使其仅返回结构完整的空数据（如数值为 0，数组为空），而不是预填充的模拟数据。
    - 修改数据获取逻辑：仅使用 API 返回的数据更新状态，API 失败时不进行 Mock 数据填充。
    - 保留 `quickActions` 等属于 UI 配置的静态数据。

### 4. 事件管理 (Incident Management)
- **检查**: 经核查 `src/modules/incident/components/IncidentList.tsx`，该文件当前未包含 Mock 数据，无需修改。

## 预期结果
- 所有列表页（工单、变更、问题、事件）在后端 API 不可用时将显示为空列表或报错，不再显示演示数据。
- 仪表盘在无数据时将显示为 0 或空图表。
- 系统将真实反映后端服务的状态。