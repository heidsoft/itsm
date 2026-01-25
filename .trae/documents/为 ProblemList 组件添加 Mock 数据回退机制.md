# 为问题管理页面添加 Mock 数据回退

## 问题分析
用户遇到 `Error: Failed to load problems: {}`，表明后端 `/api/v1/problems` 接口调用失败（可能是 404/500 或服务未启动）。为了保证页面功能可用，需要像“变更管理”页面一样，添加前端 Mock 数据回退机制。

## 修复计划
1.  **定义 Mock 数据**: 在 `src/modules/problem/components/ProblemList.tsx` 中定义 `mockProblems` 常量，包含几条符合 `Problem` 接口定义的示例数据。
2.  **修改数据加载逻辑**: 在 `loadData` 函数的 `catch` 块中，不再只是打印错误，而是：
    -   调用 `console.warn` 提示 API 失败。
    -   设置 `data` 为 `mockProblems`。
    -   设置 `total` 为 `mockProblems.length`。
    -   显示警告消息给用户。

## 具体修改
修改 `src/modules/problem/components/ProblemList.tsx`：
- 添加 `mockProblems` 数组。
- 更新 `loadData` 函数中的错误处理部分。