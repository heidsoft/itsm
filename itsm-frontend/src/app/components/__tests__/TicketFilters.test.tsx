import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import TicketFilters, { TicketFilterState } from '@/components/business/TicketFilters';

// Mock Ant Design DatePicker to avoid complex interaction issues
jest.mock('antd', () => {
  const antd = jest.requireActual('antd');
  return {
    ...antd,
    DatePicker: {
      RangePicker: ({ onChange, value }: any) => (
        <div data-testid="filter-date-range">
          <input
            data-testid="filter-date-start"
            placeholder="开始日期"
            onChange={e => {
              // Simulate start date change (partial)
              // In real component, onChange returns Moment/Dayjs array
              // We need to mock this behavior if we want to test logic strictly
              // But for now let's just trigger onChange with dummy values to verify callback
            }}
          />
           <button 
             data-testid="mock-date-change"
             onClick={() => onChange && onChange([
               { format: () => '2025-01-01' }, 
               { format: () => '2025-01-31' }
             ])}
           >
             Set Date
           </button>
        </div>
      ),
    },
    Select: ({ onChange, options, value, 'data-testid': testId, placeholder }: any) => (
      <div data-testid={testId} onClick={() => {}}>
        <div className="select-trigger">{placeholder}</div>
        {options?.map((opt: any) => (
          <div
            key={opt.value}
            data-testid={`${testId}-option-${opt.value}`}
            onClick={() => onChange(opt.value)}
          >
            {opt.label}
          </div>
        ))}
        <div data-testid="select-value">{value}</div>
      </div>
    ),
  };
});

describe('TicketFilters 组件', () => {
  const setup = (initial?: Partial<TicketFilterState>) => {
    const onFilterChange = jest.fn();
    render(
      <TicketFilters
        filters={{
          status: 'all',
          priority: 'all',
          type: 'all',
          keyword: '',
          dateStart: '',
          dateEnd: '',
          sortBy: 'createdAt_desc',
          ...initial,
        }}
        onFilterChange={onFilterChange}
      />
    );
    return { onFilterChange };
  };

  it('应渲染所有过滤控件并展示默认值', () => {
    setup();
    expect(screen.getByTestId('filter-status-select')).toBeInTheDocument();
    expect(screen.getByTestId('filter-priority-select')).toBeInTheDocument();
    expect(screen.getByTestId('filter-keyword-input')).toBeInTheDocument();
    expect(screen.getByTestId('filter-date-range')).toBeInTheDocument();
    expect(screen.getByTestId('filter-sort-select')).toBeInTheDocument();
    expect(screen.getByTestId('filter-apply-btn')).toBeInTheDocument();
    expect(screen.getByTestId('filter-reset-btn')).toBeInTheDocument();
  });

  it('状态过滤变化时应调用 onFilterChange', async () => {
    const user = userEvent.setup();
    const { onFilterChange } = setup();

    // Click the option directly (since we mocked Select)
    await user.click(screen.getByTestId('filter-status-select-option-open'));
    
    expect(onFilterChange).toHaveBeenLastCalledWith(expect.objectContaining({ status: 'open' }));
  });

  it('优先级过滤变化时应调用 onFilterChange', async () => {
    const user = userEvent.setup();
    const { onFilterChange } = setup();

    await user.click(screen.getByTestId('filter-priority-select-option-p1'));
    
    expect(onFilterChange).toHaveBeenLastCalledWith(expect.objectContaining({ priority: 'p1' }));
  });

  it('关键字输入后点击应用应调用 onFilterChange，包含 keyword', async () => {
    const user = userEvent.setup();
    const { onFilterChange } = setup();

    const input = screen.getByTestId('filter-keyword-input');
    await user.type(input, '数据库');
    await user.click(screen.getByTestId('filter-apply-btn'));

    expect(onFilterChange).toHaveBeenLastCalledWith(expect.objectContaining({ keyword: '数据库' }));
  });

  it('设置日期范围后点击应用应包含 dateStart 与 dateEnd', async () => {
    const user = userEvent.setup();
    const { onFilterChange } = setup();

    // Use our mock button to trigger change
    await user.click(screen.getByTestId('mock-date-change'));

    expect(onFilterChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ dateStart: '2025-01-01', dateEnd: '2025-01-31' })
    );
  });

  it('排序变化时应调用 onFilterChange', async () => {
    const user = userEvent.setup();
    const { onFilterChange } = setup();

    await user.click(screen.getByTestId('filter-sort-select-option-priority_desc'));
    
    expect(onFilterChange).toHaveBeenLastCalledWith(expect.objectContaining({ sortBy: 'priority_desc' }));
  });

  it('重置按钮应将过滤条件恢复默认并调用 onFilterChange', async () => {
    const user = userEvent.setup();
    const { onFilterChange } = setup({ status: 'open', priority: 'p2', keyword: 'x' });

    await user.click(screen.getByTestId('filter-reset-btn'));
    
    // Note: The component calls onFilterChange with DEFAULT_VALUE
    expect(onFilterChange).toHaveBeenCalledWith(expect.objectContaining({
      status: 'all',
      priority: 'all',
      keyword: '',
      dateStart: '',
      dateEnd: '',
      sortBy: 'createdAt_desc',
    }));
  });
});
