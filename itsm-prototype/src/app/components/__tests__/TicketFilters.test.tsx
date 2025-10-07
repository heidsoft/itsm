import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import TicketFilters, { TicketFilterState } from '@/components/TicketFilters';

describe('TicketFilters 组件', () => {
  const setup = (initial?: Partial<TicketFilterState>) => {
    const onChange = jest.fn();
    render(
      <TicketFilters
        value={{
          status: 'all',
          priority: 'all',
          keyword: '',
          dateStart: '',
          dateEnd: '',
          sortBy: 'createdAt_desc',
          ...initial,
        }}
        onChange={onChange}
      />
    );
    return { onChange };
  };

  it('应渲染所有过滤控件并展示默认值', () => {
    setup();
    expect(screen.getByTestId('filter-status-select')).toBeInTheDocument();
    expect(screen.getByTestId('filter-priority-select')).toBeInTheDocument();
    expect(screen.getByTestId('filter-keyword-input')).toBeInTheDocument();
    expect(screen.getByTestId('filter-date-start')).toBeInTheDocument();
    expect(screen.getByTestId('filter-date-end')).toBeInTheDocument();
    expect(screen.getByTestId('filter-sort-select')).toBeInTheDocument();
    expect(screen.getByTestId('filter-apply-btn')).toBeInTheDocument();
    expect(screen.getByTestId('filter-reset-btn')).toBeInTheDocument();
  });

  it('状态过滤变化时应调用 onChange', async () => {
    const user = userEvent.setup();
    const { onChange } = setup();

    await user.selectOptions(screen.getByTestId('filter-status-select'), 'open');
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ status: 'open' })
    );
  });

  it('优先级过滤变化时应调用 onChange', async () => {
    const user = userEvent.setup();
    const { onChange } = setup();

    await user.selectOptions(screen.getByTestId('filter-priority-select'), 'p1');
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ priority: 'p1' })
    );
  });

  it('关键字输入后点击应用应调用 onChange，包含 keyword', async () => {
    const user = userEvent.setup();
    const { onChange } = setup();

    const input = screen.getByTestId('filter-keyword-input');
    await user.type(input, '数据库');
    await user.click(screen.getByTestId('filter-apply-btn'));

    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ keyword: '数据库' })
    );
  });

  it('设置日期范围后点击应用应包含 dateStart 与 dateEnd', async () => {
    const user = userEvent.setup();
    const { onChange } = setup();

    await user.type(screen.getByTestId('filter-date-start'), '2025-01-01');
    await user.type(screen.getByTestId('filter-date-end'), '2025-01-31');
    await user.click(screen.getByTestId('filter-apply-btn'));

    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ dateStart: '2025-01-01', dateEnd: '2025-01-31' })
    );
  });

  it('排序变化时应调用 onChange', async () => {
    const user = userEvent.setup();
    const { onChange } = setup();

    await user.selectOptions(screen.getByTestId('filter-sort-select'), 'priority_desc');
    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ sortBy: 'priority_desc' })
    );
  });

  it('重置按钮应将过滤条件恢复默认并调用 onChange', async () => {
    const user = userEvent.setup();
    const { onChange } = setup({ status: 'open', priority: 'p2', keyword: 'x' });

    await user.click(screen.getByTestId('filter-reset-btn'));
    expect(onChange).toHaveBeenLastCalledWith({
      status: 'all',
      priority: 'all',
      keyword: '',
      dateStart: '',
      dateEnd: '',
      sortBy: 'createdAt_desc',
    });
  });
});