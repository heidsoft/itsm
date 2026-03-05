/**
 * SLAFilterPanel - SLA 过滤器面板组件测试
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SLAFilterPanel } from '../components/SLAFilterPanel';
import type { SLAFilterOptions } from '../types';

jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

describe('SLAFilterPanel', () => {
  const mockOnFilterChange = jest.fn();
  const mockOnReset = jest.fn();

  const defaultProps = {
    filters: {
      status: 'all',
      severity: 'all',
      policy: 'all',
      dateRange: {
        start: null,
        end: null,
      },
      assignedTo: null,
    },
    onFilterChange: mockOnFilterChange,
    onReset: mockOnReset,
    policies: [
      { id: 'policy-1', name: 'Response Time' },
      { id: 'policy-2', name: 'Resolution Time' },
    ],
    loading: false,
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('应该渲染过滤器标题', () => {
    render(<SLAFilterPanel {...defaultProps} />);

    expect(screen.getByText(/filter/i)).toBeInTheDocument();
  });

  it('应该显示状态选择器', () => {
    render(<SLAFilterPanel {...defaultProps} />);

    expect(screen.getByLabelText(/status/i)).toBeInTheDocument();
  });

  it('应该显示严重程度选择器', () => {
    render(<SLAFilterPanel {...defaultProps} />);

    expect(screen.getByLabelText(/severity/i)).toBeInTheDocument();
  });

  it('应该显示政策选择器', () => {
    render(<SLAFilterPanel {...defaultProps} />);

    expect(screen.getByLabelText(/policy/i)).toBeInTheDocument();
  });

  it('应该显示日期范围选择器', () => {
    render(<SLAFilterPanel {...defaultProps} />);

    expect(screen.getByLabelText(/date range|start date/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/end date/i)).toBeInTheDocument();
  });

  it('应该显示重置按钮', () => {
    render(<SLAFilterPanel {...defaultProps} />);

    expect(screen.getByRole('button', { name: /reset/i })).toBeInTheDocument();
  });

  it('点击重置应调用 onReset', async () => {
    const user = userEvent.setup();
    render(<SLAFilterPanel {...defaultProps} />);

    const resetButton = screen.getByRole('button', { name: /reset/i });
    await user.click(resetButton);

    expect(mockOnReset).toHaveBeenCalled();
  });

  it('应该显示分配给人选择器', () => {
    render(<SLAFilterPanel {...defaultProps} />);

    expect(screen.getByLabelText(/assigned to/i)).toBeInTheDocument();
  });

  it('应该支持搜索', () => {
    render(<SLAFilterPanel {...defaultProps} />);

    expect(screen.getByPlaceholderText(/search/i)).toBeInTheDocument();
  });

  it('应该显示已选过滤器标签', () => {
    const activeFilters = {
      ...defaultProps.filters,
      status: 'open',
      severity: 'critical',
    };

    render(<SLAFilterPanel {...defaultProps} filters={activeFilters} />);

    expect(screen.getByText('open')).toBeInTheDocument();
    expect(screen.getByText('critical')).toBeInTheDocument();
  });

  it('应该允许移除过滤器标签', async () => {
    const user = userEvent.setup();
    const activeFilters = {
      ...defaultProps.filters,
      status: 'open',
    };

    render(<SLAFilterPanel {...defaultProps} filters={activeFilters} />);

    const removeButtons = screen.getAllByRole('button', { name: /remove|close/i });
    await user.click(removeButtons[0]);

    expect(mockOnFilterChange).toHaveBeenCalledWith(
      expect.objectContaining({ status: 'all' })
    );
  });

  it('应该应用筛选时调用 onFilterChange', async () => {
    const user = userEvent.setup();

    render(<SLAFilterPanel {...defaultProps} />);

    const statusSelect = screen.getByLabelText(/status/i);
    await user.click(statusSelect);
    await user.keyboard('{arrowdown}{arrowdown}{enter}');

    expect(mockOnFilterChange).toHaveBeenCalled();
  });

  it('加载状态应显示加载指示器', () => {
    render(<SLAFilterPanel {...defaultProps} loading={true} />);

    expect(screen.getByText(/loading/i)).toBeInTheDocument();
  });

  it('应该显示快速筛选按钮', () => {
    render(<SLAFilterPanel {...defaultProps} />);

    expect(screen.getByRole('button', { name: /critical only/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /unassigned/i })).toBeInTheDocument();
  });

  it('点击快速筛选应设置过滤器', async () => {
    const user = userEvent.setup();

    render(<SLAFilterPanel {...defaultProps} />);

    const criticalOnlyButton = screen.getByRole('button', { name: /critical only/i });
    await user.click(criticalOnlyButton);

    expect(mockOnFilterChange).toHaveBeenCalledWith(
      expect.objectContaining({ severity: 'critical' })
    );
  });

  it('应该显示过滤器计数', () => {
    const activeFilters = {
      ...defaultProps.filters,
      status: 'open',
      severity: 'critical',
    };

    render(<SLAFilterPanel {...defaultProps} filters={activeFilters} />);

    expect(screen.getByText(/2 active filters/i)).toBeInTheDocument();
  });

  it('应该支持清除所有过滤器', async () => {
    const user = userEvent.setup();
    const activeFilters = {
      ...defaultProps.filters,
      status: 'open',
      severity: 'critical',
    };

    render(<SLAFilterPanel {...defaultProps} filters={activeFilters} />);

    const clearAllButton = screen.getByRole('button', { name: /clear all/i });
    await user.click(clearAllButton);

    expect(mockOnReset).toHaveBeenCalled();
  });

  it('应该显示扩展/收起按钮', () => {
    render(<SLAFilterPanel {...defaultProps} />);

    expect(screen.getByRole('button', { name: /expand|more filters/i })).toBeInTheDocument();
  });

  it('应该包含政策选项', () => {
    render(<SLAFilterPanel {...defaultProps} />);

    const policySelect = screen.getByLabelText(/policy/i);
    expect(policySelect).toBeInTheDocument();

    // 检查政策选项存在
    expect(screen.getByText('Response Time')).toBeInTheDocument();
    expect(screen.getByText('Resolution Time')).toBeInTheDocument();
  });

  it('应该包含状态选项', () => {
    render(<SLAFilterPanel {...defaultProps} />);

    const statusSelect = screen.getByLabelText(/status/i);
    expect(statusSelect).toBeInTheDocument();

    // 检查常见状态选项存在
    expect(screen.getByText('open')).toBeInTheDocument();
    expect(screen.getByText('acknowledged')).toBeInTheDocument();
    expect(screen.getByText('resolved')).toBeInTheDocument();
  });

  it('应该包含严重程度选项', () => {
    render(<SLAFilterPanel {...defaultProps} />);

    const severitySelect = screen.getByLabelText(/severity/i);
    expect(severitySelect).toBeInTheDocument();

    expect(screen.getByText('critical')).toBeInTheDocument();
    expect(screen.getByText('high')).toBeInTheDocument();
    expect(screen.getByText('medium')).toBeInTheDocument();
    expect(screen.getByText('low')).toBeInTheDocument();
  });

  it('应该支持保存筛选器预设', () => {
    render(<SLAFilterPanel {...defaultProps} />);

    const savePresetButton = screen.getByRole('button', { name: /save preset/i });
    expect(savePresetButton).toBeInTheDocument();
  });

  it('应该显示当前筛选器摘要', () => {
    const activeFilters = {
      ...defaultProps.filters,
      status: 'open',
    };

    render(<SLAFilterPanel {...defaultProps} filters={activeFilters} />);

    expect(screen.getByText(/status: open/i)).toBeInTheDocument();
  });

  it('应该处理日期范围选择', async () => {
    const user = userEvent.setup();

    render(<SLAFilterPanel {...defaultProps} />);

    const startDateInput = screen.getByLabelText(/start date/i);
    await user.type(startDateInput, '2024-01-01');

    expect(mockOnFilterChange).toHaveBeenCalledWith(
      expect.objectContaining({
        dateRange: expect.objectContaining({
          start: '2024-01-01',
        }),
      })
    );
  });

  it('应该分配给人搜索', () => {
    render(<SLAFilterPanel {...defaultProps} />);

    const assigneeSelect = screen.getByLabelText(/assigned to/i);
    expect(assigneeSelect).toBeInTheDocument();
  });

  it('应该显示过滤器操作按钮组', () => {
    render(<SLAFilterPanel {...defaultProps} />);

    expect(screen.getByRole('button', { name: /apply/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /reset/i })).toBeInTheDocument();
  });

  it('应该显示应用按钮', () => {
    render(<SLAFilterPanel {...defaultProps} />);

    expect(screen.getByRole('button', { name: /apply filters/i })).toBeInTheDocument();
  });

  it('应该支持复制过滤器链接', () => {
    render(<SLAFilterPanel {...defaultProps} />);

    const copyLinkButton = screen.getByRole('button', { name: /copy link/i });
    expect(copyLinkButton).toBeInTheDocument();
  });

  it('应该显示过滤器帮助文本', () => {
    render(<SLAFilterPanel {...defaultProps} />);

    expect(screen.getByText(/filter.*violation/i)).toBeInTheDocument();
  });
});
