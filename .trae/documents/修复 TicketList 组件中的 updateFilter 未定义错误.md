# 修复 ReferenceError: updateFilter is not defined

## 问题原因
在 `src/components/ticket/TicketList.tsx` 文件中，`handleDateRangeChange` 函数试图调用 `updateFilter`（单数形式），但该函数并未定义。实际从 `useTickets` hook 中导出的是 `updateFilters`（复数形式），且支持批量更新过滤器。

此外，该函数中还尝试向 `fetchTickets` 传递参数，而当前的 `useTickets` 实现中 `fetchTickets` 不接受参数，而是依赖内部状态。

## 修复计划
修改 `src/components/ticket/TicketList.tsx` 文件：
1.  将 `handleDateRangeChange` 中的 `updateFilter` 调用替换为 `updateFilters`。
2.  移除无效的 `fetchTickets({...})` 调用，因为 `updateFilters` 会更新状态并自动触发数据刷新。
3.  修正 `useCallback` 的依赖数组。

## 预期代码变更
```typescript
  // 日期范围变更处理
  const handleDateRangeChange = useCallback(
    (dates: [dayjs.Dayjs | null, dayjs.Dayjs | null] | null, field: 'created' | 'due') => {
      const startKey = field === 'created' ? 'created_after' : 'due_after';
      const endKey = field === 'created' ? 'created_before' : 'due_before';

      if (dates && dates[0] && dates[1]) {
        // 使用 updateFilters 批量更新
        updateFilters({
          [startKey]: dates[0].format('YYYY-MM-DD'),
          [endKey]: dates[1].format('YYYY-MM-DD'),
        });
      } else {
        // 清除过滤器
        updateFilters({
          [startKey]: undefined,
          [endKey]: undefined,
        });
      }
    },
    [updateFilters] // 修正依赖
  );
```