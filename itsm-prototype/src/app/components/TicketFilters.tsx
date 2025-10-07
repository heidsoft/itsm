import React from 'react';

export type TicketFilterState = {
  status: 'all' | 'open' | 'in_progress' | 'resolved' | 'closed';
  priority: 'all' | 'p1' | 'p2' | 'p3' | 'p4';
  keyword: string;
  dateStart: string; // YYYY-MM-DD
  dateEnd: string;   // YYYY-MM-DD
  sortBy: 'createdAt_desc' | 'createdAt_asc' | 'priority_desc' | 'priority_asc';
};

interface Props {
  value: TicketFilterState;
  onChange: (next: TicketFilterState) => void;
}

const DEFAULT_VALUE: TicketFilterState = {
  status: 'all',
  priority: 'all',
  keyword: '',
  dateStart: '',
  dateEnd: '',
  sortBy: 'createdAt_desc',
};

// 组件策略：
// - 下拉选择（状态、优先级、排序）即时触发 onChange
// - 文本与日期输入在“应用”按钮点击时提交
// - “重置”按钮恢复默认并触发 onChange
export default function TicketFilters({ value, onChange }: Props) {
  const [local, setLocal] = React.useState<TicketFilterState>(value);

  React.useEffect(() => {
    setLocal(value);
  }, [value.status, value.priority, value.keyword, value.dateStart, value.dateEnd, value.sortBy]);

  const patch = (partial: Partial<TicketFilterState>) => ({ ...local, ...partial }) as TicketFilterState;

  const handleImmediateChange = (partial: Partial<TicketFilterState>) => {
    const next = patch(partial);
    setLocal(next);
    onChange(next);
  };

  const handleApply = () => {
    onChange(local);
  };

  const handleReset = () => {
    setLocal(DEFAULT_VALUE);
    onChange(DEFAULT_VALUE);
  };

  return (
    <div data-testid="ticket-filters" aria-label="工单过滤与排序">
      {/* 状态 */}
      <label>
        状态
        <select
          data-testid="filter-status-select"
          value={local.status}
          onChange={(e) => handleImmediateChange({ status: e.target.value as TicketFilterState['status'] })}
        >
          <option value="all">全部</option>
          <option value="open">待处理</option>
          <option value="in_progress">处理中</option>
          <option value="resolved">已解决</option>
          <option value="closed">已关闭</option>
        </select>
      </label>

      {/* 优先级 */}
      <label>
        优先级
        <select
          data-testid="filter-priority-select"
          value={local.priority}
          onChange={(e) => handleImmediateChange({ priority: e.target.value as TicketFilterState['priority'] })}
        >
          <option value="all">全部</option>
          <option value="p1">P1</option>
          <option value="p2">P2</option>
          <option value="p3">P3</option>
          <option value="p4">P4</option>
        </select>
      </label>

      {/* 关键字 */}
      <label>
        关键字
        <input
          data-testid="filter-keyword-input"
          type="text"
          value={local.keyword}
          onChange={(e) => setLocal(patch({ keyword: e.target.value }))}
          placeholder="搜索标题或描述"
        />
      </label>

      {/* 日期范围 */}
      <label>
        开始日期
        <input
          data-testid="filter-date-start"
          type="date"
          value={local.dateStart}
          onChange={(e) => setLocal(patch({ dateStart: e.target.value }))}
        />
      </label>
      <label>
        结束日期
        <input
          data-testid="filter-date-end"
          type="date"
          value={local.dateEnd}
          onChange={(e) => setLocal(patch({ dateEnd: e.target.value }))}
        />
      </label>

      {/* 排序 */}
      <label>
        排序
        <select
          data-testid="filter-sort-select"
          value={local.sortBy}
          onChange={(e) => handleImmediateChange({ sortBy: e.target.value as TicketFilterState['sortBy'] })}
        >
          <option value="createdAt_desc">最近创建</option>
          <option value="createdAt_asc">最早创建</option>
          <option value="priority_desc">优先级高到低</option>
          <option value="priority_asc">优先级低到高</option>
        </select>
      </label>

      {/* 操作 */}
      <div style={{ display: 'inline-flex', gap: 8, marginLeft: 8 }}>
        <button data-testid="filter-apply-btn" onClick={handleApply}>应用</button>
        <button data-testid="filter-reset-btn" onClick={handleReset}>重置</button>
      </div>
    </div>
  );
}