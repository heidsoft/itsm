# 继续去 Mock 化改造计划

根据之前的检查报告，我们将继续对 Workflow（工作流）和 Admin（系统管理）模块进行"去 Mock 化"改造，使其对接真实后端 API。

## 1. Workflow 模块改造
**目标**：移除前端页面硬编码的工作流数据，完全依赖后端 BPMN 引擎接口。

- **修改文件**：`src/app/(main)/workflow/page.tsx`
- **具体动作**：
    1.  删除 `mockWorkflows` 常量及其相关定义。
    2.  修改 `loadWorkflows` 函数，移除 `catch` 块中回退到 `mockWorkflows` 的逻辑，API 失败时应显示错误提示并保持列表为空。
    3.  确保页面加载状态（Loading）真实反映 API 请求过程。

## 2. Admin 模块改造
**目标**：让系统概览（System Overview）显示真实统计数据，而非写死的"1,234 用户"。

- **Step 1: 增强数据 Hook**
    -   **修改文件**：`src/app/(main)/admin/hooks/useAdminData.ts`
    -   **具体动作**：引入 `DashboardAPI` 和 `WorkflowAPI`，并发请求获取：
        -   活跃用户数 (调用 `DashboardAPI.getUserStats`)
        -   运行中工作流数 (调用 `WorkflowAPI.getInstances` 或通过列表计算)
        -   组合成 `adminStats` 对象返回。

- **Step 2: 组件参数化**
    -   **修改文件**：`src/app/(main)/admin/components/SystemOverview.tsx`
    -   **具体动作**：修改组件接口，使其接收 `stats` 属性，并用传入的真实数据替换内部硬编码的 `systemStats`。

- **Step 3: 页面集成**
    -   **修改文件**：`src/app/(main)/admin/page.tsx`
    -   **具体动作**：将 `useAdminData` 返回的真实数据传递给 `SystemOverview` 组件。

## 3. 预期结果
- **Workflow 页面**：将显示后端实际配置的工作流列表。如果后端为空，列表应为空。
- **Admin 仪表盘**：顶部的"系统概览"卡片将显示真实的活跃用户数和运行中工作流数量。