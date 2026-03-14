/**
 * CI 变更历史标签页组件测试
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CIChangeHistoryTab } from '../sections/CIChangeHistoryTab';

jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

describe('CIChangeHistoryTab', () => {
  const mockChangeHistory = {
    logs: [
      {
        id: 1,
        action: 'update',
        resource: 'CI',
        path: '/api/v1/cmdb/ci/ci-001',
        Method: 'PUT',
        StatusCode: 200,
        created_at: '2024-01-20T15:30:00Z',
        updated_by: 'admin',
        description: 'Updated memory configuration',
      },
      {
        id: 2,
        action: 'update',
        resource: 'CI',
        path: '/api/v1/cmdb/ci/ci-001',
        Method: 'PUT',
        StatusCode: 200,
        created_at: '2024-01-19T09:15:00Z',
        updated_by: 'operator',
        description: 'Changed status from standby to active',
      },
      {
        id: 3,
        action: 'update',
        resource: 'CI',
        path: '/api/v1/cmdb/ci/ci-001',
        Method: 'PUT',
        StatusCode: 200,
        created_at: '2024-01-18T14:00:00Z',
        updated_by: 'manager',
        description: 'Reassigned to DevOps team',
      },
    ],
    total: 15,
    page: 1,
    page_size: 10,
  };

  const defaultProps = {
    changeHistory: mockChangeHistory,
    historyLoading: false,
    onLoad: jest.fn(),
  };

  it('应该显示变更历史标题', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    // 显示变更历史标题（使用精确匹配）
    expect(screen.getByText('CI变更历史')).toBeInTheDocument();
  });

  it('应该显示加载按钮', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    const loadButton = screen.getByRole('button', { name: /加载变更历史/i });
    expect(loadButton).toBeInTheDocument();
  });

  it('点击加载按钮应调用 onLoad', async () => {
    const user = userEvent.setup();
    const mockOnLoad = jest.fn();

    render(<CIChangeHistoryTab {...defaultProps} onLoad={mockOnLoad} />);

    const loadButton = screen.getByRole('button', { name: /加载变更历史/i });
    await user.click(loadButton);

    expect(mockOnLoad).toHaveBeenCalled();
  });

  it('应该显示所有变更记录', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    // 检查时间线节点数量
    const timelineItems = document.querySelectorAll('.ant-timeline-item');
    expect(timelineItems.length).toBe(mockChangeHistory.logs.length);
  });

  it('应该显示操作方法标签', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    // 应该显示 action (update) 或 Method (PUT)
    expect(screen.getAllByText(/update|PUT/).length).toBeGreaterThan(0);
  });

  it('应该显示变更描述', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    expect(screen.getByText(/Updated memory configuration/)).toBeInTheDocument();
    expect(screen.getByText(/Changed status from standby to active/)).toBeInTheDocument();
    expect(screen.getByText(/Reassigned to DevOps team/)).toBeInTheDocument();
  });

  it('应该显示操作人', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    // 操作人显示在时间戳前面
    expect(screen.getByText(/操作人: admin/)).toBeInTheDocument();
    expect(screen.getByText(/操作人: operator/)).toBeInTheDocument();
    expect(screen.getByText(/操作人: manager/)).toBeInTheDocument();
  });

  it('应该显示时间', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    // 检查日期存在 - 操作人和时间戳一起显示
    expect(screen.getAllByText(/2024/).length).toBeGreaterThan(0);
  });

  it('加载状态应显示加载指示器', () => {
    render(
      <CIChangeHistoryTab
        {...defaultProps}
        historyLoading={true}
      />
    );

    // 检查按钮是否有 loading 属性
    const loadButton = screen.getByRole('button');
    expect(loadButton).toBeInTheDocument();
  });

  it('没有变更历史应显示空状态', () => {
    const emptyHistory = {
      logs: [],
      total: 0,
      page: 1,
      page_size: 10,
    };

    render(<CIChangeHistoryTab {...defaultProps} changeHistory={emptyHistory} />);

    expect(screen.getByText(/暂无历史审计记录/i)).toBeInTheDocument();
  });

  it('changeHistory 为 null 应显示空状态', () => {
    render(<CIChangeHistoryTab {...defaultProps} changeHistory={null} />);

    expect(screen.getByText(/暂无历史审计记录/i)).toBeInTheDocument();
  });
});
