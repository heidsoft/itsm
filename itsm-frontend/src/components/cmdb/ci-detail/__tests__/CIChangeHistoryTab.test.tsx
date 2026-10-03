/**
 * CI 变更历史标签页组件测试
 *
 * fixture 逐字段对齐后端 `dto.CIHistoryResponse` + 平台五键信封
 * （itsm-backend/dto/cmdb_dto.go:467、GET /api/v1/cmdb/cis/:id/history）。
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CIChangeHistoryTab } from '../sections/CIChangeHistoryTab';
import type { ChangeHistoryData } from '../types';

jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

describe('CIChangeHistoryTab', () => {
  // operation 取后端实测写入值：create / update / delete / revert / lifecycle_update
  const mockChangeHistory: ChangeHistoryData = {
    items: [
      {
        id: 3,
        ciId: 101,
        version: 3,
        operation: 'update',
        changedFields: ['memory', 'status'],
        operatorId: 7,
        operatorName: 'admin',
        remark: '扩容内存',
        createdAt: '2024-01-20T15:30:00Z',
      },
      {
        id: 2,
        ciId: 101,
        version: 2,
        operation: 'lifecycle_update',
        operatorId: 8,
        operatorName: 'operator',
        createdAt: '2024-01-19T09:15:00Z',
      },
      {
        id: 1,
        ciId: 101,
        version: 1,
        operation: 'create',
        operatorId: 0,
        createdAt: '2024-01-18T14:00:00Z',
      },
    ],
    total: 25,
    page: 1,
    pageSize: 20,
    totalPages: 2,
  };

  const defaultProps = {
    changeHistory: mockChangeHistory,
    historyLoading: false,
    historyError: false,
    onLoad: jest.fn(),
    onPageChange: jest.fn(),
  };

  it('应该显示变更历史标题', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    expect(screen.getByText('CI变更历史')).toBeInTheDocument();
  });

  it('点击刷新按钮应调用 onLoad', async () => {
    const user = userEvent.setup();
    const mockOnLoad = jest.fn();

    render(<CIChangeHistoryTab {...defaultProps} onLoad={mockOnLoad} />);

    await user.click(screen.getByRole('button', { name: '刷新变更历史' }));

    expect(mockOnLoad).toHaveBeenCalled();
  });

  it('应该按信封 items 渲染全部条目', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    expect(document.querySelectorAll('.ant-timeline-item')).toHaveLength(
      mockChangeHistory.items.length
    );
  });

  it('应该显示中文操作标签与版本号', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    expect(screen.getByText('更新')).toBeInTheDocument();
    expect(screen.getByText('生命周期变更')).toBeInTheDocument();
    expect(screen.getByText('创建')).toBeInTheDocument();
    expect(screen.getByText('v3')).toBeInTheDocument();
  });

  it('未知 operation 值原样显示，不吞成空白', () => {
    render(
      <CIChangeHistoryTab
        {...defaultProps}
        changeHistory={{
          ...mockChangeHistory,
          items: [{ ...mockChangeHistory.items[0], operation: 'auto_reconciled' }],
        }}
      />
    );

    expect(screen.getByText('auto_reconciled')).toBeInTheDocument();
  });

  it('应该显示变更字段与备注', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    expect(screen.getByText('变更字段: memory, status')).toBeInTheDocument();
    expect(screen.getByText('扩容内存')).toBeInTheDocument();
  });

  it('应该显示操作人，后端未给姓名时不伪造', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    expect(screen.getByText('操作人: admin')).toBeInTheDocument();
    expect(screen.getByText('操作人: operator')).toBeInTheDocument();
    // 三条记录里只有两条带 operatorName（系统操作者的那条为空）
    expect(screen.getAllByText(/操作人:/)).toHaveLength(2);
  });

  it('应该用响应元数据如实说明总量与页码', () => {
    render(<CIChangeHistoryTab {...defaultProps} />);

    expect(screen.getByText(/共 25 条，第 1\/2 页/)).toBeInTheDocument();
  });

  it('第一页时上一页禁用、下一页按 items 翻页', async () => {
    const user = userEvent.setup();
    const mockOnPageChange = jest.fn();

    render(<CIChangeHistoryTab {...defaultProps} onPageChange={mockOnPageChange} />);

    expect(screen.getByRole('button', { name: '变更历史上一页' })).toBeDisabled();

    await user.click(screen.getByRole('button', { name: '变更历史下一页' }));
    expect(mockOnPageChange).toHaveBeenCalledWith(2);
  });

  it('末页时下一页禁用', () => {
    render(
      <CIChangeHistoryTab
        {...defaultProps}
        changeHistory={{ ...mockChangeHistory, page: 2, totalPages: 2, items: [mockChangeHistory.items[0]] }}
      />
    );

    expect(screen.getByRole('button', { name: '变更历史下一页' })).toBeDisabled();
    expect(screen.getByRole('button', { name: '变更历史上一页' })).not.toBeDisabled();
  });

  it('加载失败必须显示错误而不是「暂无历史审计记录」', () => {
    render(<CIChangeHistoryTab {...defaultProps} changeHistory={null} historyError />);

    expect(screen.getByText('变更历史加载失败')).toBeInTheDocument();
    expect(screen.queryByText(/暂无历史审计记录/)).not.toBeInTheDocument();
  });

  it('空历史应显示空状态', () => {
    render(
      <CIChangeHistoryTab
        {...defaultProps}
        changeHistory={{ items: [], total: 0, page: 1, pageSize: 20, totalPages: 0 }}
      />
    );

    expect(screen.getByText(/暂无历史审计记录/i)).toBeInTheDocument();
  });

  it('changeHistory 为 null 应显示空状态', () => {
    render(<CIChangeHistoryTab {...defaultProps} changeHistory={null} />);

    expect(screen.getByText(/暂无历史审计记录/i)).toBeInTheDocument();
  });
});
