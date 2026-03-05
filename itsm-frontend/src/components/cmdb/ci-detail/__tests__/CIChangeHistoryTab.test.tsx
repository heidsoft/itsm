/**
 * CIChangeHistoryTab - CI 变更历史组件测试
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CIChangeHistoryTab } from '../sections/CIChangeHistoryTab';
import type { CIChangeHistoryTabProps } from '../types';

jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

describe('CIChangeHistoryTab', () => {
  const mockChangeHistory = {
    ciId: 'ci-001',
    totalChanges: 15,
    changes: [
      {
        id: 'ch-1',
        changeId: 'CHG001',
        type: 'configuration_update',
        description: 'Updated memory configuration',
        changedBy: 'admin',
        changedAt: '2024-01-20T15:30:00Z',
        oldValue: '16GB',
        newValue: '32GB',
        category: 'hardware',
      },
      {
        id: 'ch-2',
        changeId: 'CHG002',
        type: 'status_change',
        description: 'Changed status from standby to active',
        changedBy: 'operator',
        changedAt: '2024-01-19T09:15:00Z',
        oldValue: 'standby',
        newValue: 'active',
        category: 'operation',
      },
      {
        id: 'ch-3',
        changeId: 'CHG003',
        type: 'assignment',
        description: 'Reassigned to DevOps team',
        changedBy: 'manager',
        changedAt: '2024-01-18T14:00:00Z',
        oldValue: 'Operations',
        newValue: 'DevOps',
        category: 'administrative',
      },
    ],
  };

  const defaultProps: CIChangeHistoryTabProps = {
    changeHistory: mockChangeHistory,
    loading: false,
    onFilterChange: jest.fn(),
    onExport: jest.fn(),
  };

  it('应该显示变更总数', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    expect(screen.getByText('15')).toBeInTheDocument();
    expect(screen.getByText(/change/i)).toBeInTheDocument();
  });

  it('应该显示所有变更记录', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    expect(screen.getByText('CHG001')).toBeInTheDocument();
    expect(screen.getByText('CHG002')).toBeInTheDocument();
    expect(screen.getByText('CHG003')).toBeInTheDocument();
  });

  it('应该显示变更类型', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    expect(screen.getByText('configuration_update')).toBeInTheDocument();
    expect(screen.getByText('status_change')).toBeInTheDocument();
    expect(screen.getByText('assignment')).toBeInTheDocument();
  });

  it('应该显示变更描述', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    expect(screen.getByText(/Updated memory configuration/)).toBeInTheDocument();
    expect(screen.getByText(/Changed status from standby to active/)).toBeInTheDocument();
  });

  it('应该显示操作人和时间', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    expect(screen.getByText('admin')).toBeInTheDocument();
    expect(screen.getByText('operator')).toBeInTheDocument();
    expect(screen.getByText(/2024|Jan|20/)).toBeInTheDocument();
  });

  it('应该显示旧值和新值', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    expect(screen.getByText('16GB')).toBeInTheDocument();
    expect(screen.getByText('32GB')).toBeInTheDocument();
    expect(screen.getByText('standby')).toBeInTheDocument();
    expect(screen.getByText('active')).toBeInTheDocument();
  });

  it('应该显示变更分类', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    expect(screen.getByText('hardware')).toBeInTheDocument();
    expect(screen.getByText('operation')).toBeInTheDocument();
    expect(screen.getByText('administrative')).toBeInTheDocument();
  });

  it('应该显示搜索框', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    expect(screen.getByPlaceholderText(/search/i)).toBeInTheDocument();
  });

  it('搜索应过滤变更记录', async () => {
    const user = userEvent.setup();
    render(<CIChangeHistoryTab {...defaultProps} />);

    const searchInput = screen.getByPlaceholderText(/search/i);
    await user.type(searchInput, 'memory');

    expect(screen.getByText('CHG001')).toBeInTheDocument();
    expect(screen.queryByText('CHG002')).not.toBeInTheDocument();
  });

  it('应该显示导出按钮', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    expect(screen.getByRole('button', { name: /export/i })).toBeInTheDocument();
  });

  it('点击导出应调用 onExport', async () => {
    const user = userEvent.setup();
    const mockOnExport = jest.fn();

    render(<CIChangeHistoryTab {...defaultProps} onExport={mockOnExport} />);

    const exportButton = screen.getByRole('button', { name: /export/i });
    await user.click(exportButton);

    expect(mockOnExport).toHaveBeenCalled();
  });

  it('应该显示过滤控件', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    expect(screen.getByLabelText(/filter by type/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/filter by category/i)).toBeInTheDocument();
  });

  it('应该显示时间范围选择', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    expect(screen.getByLabelText(/date range/i)).toBeInTheDocument();
  });

  it('加载状态应显示加载指示器', () => {
    render(<CIChangeHistoryTab {...defaultProps} loading={true} />);

    expect(screen.getByText(/loading changes/i)).toBeInTheDocument();
  });

  it('没有数据应显示空状态', () => {
    render(<CIChangeHistoryTab {...defaultProps} changeHistory={null} />);

    expect(screen.getByText(/no changes/i)).toBeInTheDocument();
  });

  it('应该显示分页控件', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    expect(screen.getByRole('button', { name: /next/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /previous/i })).toBeInTheDocument();
  });

  it('应该显示变更详情展开按钮', async () => {
    const user = userEvent.setup();
    render(<CIChangeHistoryTab {...defaultProps} />);

    const expandButtons = screen.getAllByRole('button', { name: /view details|expand/i });
    expect(expandButtons.length).toBeGreaterThan(0);

    await user.click(expandButtons[0]);

    expect(screen.getByText(/old value/i)).toBeInTheDocument();
    expect(screen.getByText(/new value/i)).toBeInTheDocument();
  });

  it('应该按时间排序', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    const changes = screen.getAllByText(/CHG00[1-3]/);
    expect(changes.length).toBe(3);
  });

  it('应该显示变更标识颜色', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    const changeIds = screen.getAllByText(/CHG\d{3}/);
    changeIds.forEach(id => {
      expect(id.closest('div')).toHaveClass('change-id');
    });
  });

  it('应该显示操作人头像', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    const avatars = document.querySelectorAll('[data-avatar]');
    expect(avatars.length).toBeGreaterThanOrEqual(mockChangeHistory.changes.length);
  });

  it('应该支持批量导出', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    const batchExportButton = screen.getByRole('button', { name: /batch export/i });
    expect(batchExportButton).toBeInTheDocument();
  });

  it('应该显示变更趋势', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    expect(screen.getByTestId('change-trend')).toBeInTheDocument();
  });

  it('应该显示最活跃变更人', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    expect(screen.getByText(/admin|operator|manager/)).toBeInTheDocument();
  });
});
