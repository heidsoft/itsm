# 修复 Ant Design Card 组件 `bordered` 属性废弃警告

## 问题分析
用户报告在点击"问题管理"页面时出现 Ant Design 的警告：
`Warning: [antd: Card] 'bordered' is deprecated. Please use 'variant' instead.`

经过检查 `src/modules/problem/components/ProblemList.tsx` 文件，发现在第 178 行使用了 `<Card bordered={false}>`。Ant Design v5 版本中，`bordered` 属性已被弃用，建议使用 `variant` 属性来控制边框样式。

## 修复方案
将 `<Card bordered={false}>` 替换为 `<Card variant="borderless">`。

- `bordered={false}` 对应 `variant="borderless"`
- `bordered={true}` (默认) 对应 `variant="outlined"`

## 修改计划
1. 编辑 `src/modules/problem/components/ProblemList.tsx` 文件。
2. 找到第 178 行的 `Card` 组件。
3. 将 `bordered={false}` 替换为 `variant="borderless"`。

这将消除控制台警告并符合 Ant Design 的最新最佳实践。