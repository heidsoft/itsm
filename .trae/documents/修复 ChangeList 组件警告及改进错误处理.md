# 修复变更管理页面错误

## 问题分析
用户在点击变更管理页面时遇到报错：
1. `Error: {} at loadData`：这是一个未捕获的错误，通常是因为 API 调用失败或数据处理异常，但没有打印具体的错误信息（`console.error(error)` 打印空对象时可能显示为 `{}`）。
2. `[antd: Card] 'bordered' is deprecated`：这是一个警告，提示 `Card` 组件的 `bordered` 属性已弃用。

## 修复计划
1.  **修复 API 错误处理**: 在 `src/modules/change/components/ChangeList.tsx` 的 `loadData` 函数中，`console.error(error)` 可能未正确显示错误详情。我们将改进错误处理，确保在 API 调用失败时提供更有用的反馈，并检查 `ChangeApi.getChanges` 的调用是否正确。
2.  **修复 Ant Design 警告**: 将 `<Card bordered={false}>` 替换为 `<Card variant="borderless">`，以符合 Ant Design v5 的规范。

## 具体修改
修改 `src/modules/change/components/ChangeList.tsx`:
- 将 `bordered={false}` 替换为 `variant="borderless"`。
- 在 `loadData` 的 `catch` 块中，改进错误日志输出，例如 `console.error('Failed to load changes:', error);`，以便在浏览器控制台看到更详细的错误信息。
- *自我修正*: 用户提供的报错信息主要是 Console Error，这通常意味着 API 请求失败。虽然我们不能直接修复后端，但我们可以优化前端代码以避免因数据格式问题导致的渲染崩溃，并修复可见的 Deprecation Warning。

**注意**: 看起来 `ChangeList.tsx` 和之前修复的 `ProblemList.tsx` 结构非常相似，可能存在相同的代码模式问题。