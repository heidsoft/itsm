/**
 * SLATable - SLA 违规表格组件测试
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SLATable } from '../components/SLATable';
import type { SLAViolation } from '../types';

jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

describe('SLATable', () => {
  const mockViolations: SLAViolation[] = [
    {
      id: 'violation-1',
      ticketId: 'ticket-1',
      ticket: {
        id: 'ticket-1',
        title: 'Critical Incident',
        priority: 'critical',
        status: 'open',
      },
      policyId: 'policy-1',
      policyName: 'Response Time',
      breachedAt: '2024-01-20T10:00:00Z',
      currentValue: 120,
      targetValue: 60,
      slaMetric: 'response_time',
      status: 'open',
      severity: 'critical',
      assignedTo: {
        id: 'user1',
        name: 'John Doe',
        email: 'john@example.com',
      },
    },
    {
      id: 'violation-2',
      ticketId: 'ticket-2',
      ticket: {
        id: 'ticket-2',
        title: 'High Priority Issue',
        priority: 'high',
        status: 'in_progress',
      },
      policyId: 'policy-2',
      policyName: 'Resolution Time',
      breachedAt: '2024-01-20T11:00:00Z',
      currentValue: 240,
      targetValue: 120,
      slaMetric: 'resolution_time',
      status: 'acknowledged',
      severity: 'high',
      assignedTo: {
        id: 'user2',
        name: 'Jane Smith',
        email: 'jane@example.com',
      },
    },
  ];

  const defaultProps = {
    violations: mockViolations,
    loading: false,
    onAcknowledge: jest.fn(),
    onResolve: jest.fn(),
    onViewDetails: jest.fn(),
    sortBy: 'breachedAt',
    sortOrder: 'desc',
    onSort: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('应该渲染表格标题', () => {
    render(<SLATable {...defaultProps} />);

    expect(screen.getByText(/ticket/i)).toBeInTheDocument();
    expect(screen.getByText(/policy/i)).toBeInTheDocument();
    expect(screen.getByText(/severity/i)).toBeInTheDocument();
    expect(screen.getByText(/breached/i)).toBeInTheDocument();
  });

  it('应该显示所有违规记录', () => {
    render(<SLATable {...defaultProps} />);

    expect(screen.getByText('Critical Incident')).toBeInTheDocument();
    expect(screen.getByText('High Priority Issue')).toBeInTheDocument();
  });

  it('应该显示工单优先级', () => {
    render(<SLATable {...defaultProps} />);

    expect(screen.getByText('critical')).toBeInTheDocument();
    expect(screen.getByText('high')).toBeInTheDocument();
  });

  it('应该显示政策名称', () => {
    render(<SLATable {...defaultProps} />);

    expect(screen.getByText('Response Time')).toBeInTheDocument();
    expect(screen.getByText('Resolution Time')).toBeInTheDocument();
  });

  it('应该显示当前值和目标值', () => {
    render(<SLATable {...defaultProps} />);

    expect(screen.getByText('120')).toBeInTheDocument();
    expect(screen.getByText('60')).toBeInTheDocument();
    expect(screen.getByText('240')).toBeInTheDocument();
    expect(screen.getByText('120')).toBeInTheDocument();
  });

  it('应该显示分配给人', () => {
    render(<SLATable {...defaultProps} />);

    expect(screen.getByText('John Doe')).toBeInTheDocument();
    expect(screen.getByText('Jane Smith')).toBeInTheDocument();
  });

  it('应该显示违规状态', () => {
    render(<SLATable {...defaultProps} />);

    expect(screen.getByText('open')).toBeInTheDocument();
    expect(screen.getByText('acknowledged')).toBeInTheDocument();
  });

  it('应该为列头提供排序功能', () => {
    render(<SLATable {...defaultProps} />);

    const ticketHeader = screen.getByRole('columnheader', { name: /ticket/i });
    expect(ticketHeader).toHaveAttribute('data-sortable', 'true');

    // 点击表头应该触发排序
    fireEvent.click(ticketHeader);
    expect(defaultProps.onSort).toHaveBeenCalled();
  });

  it('应该显示排序指示器', () => {
    render(<SLATable {...defaultProps} sortBy="breachedAt" sortOrder="desc" />);

    const sortIndicator = screen.getByText(/sorted by/i);
    expect(sortIndicator).toBeInTheDocument();
  });

  it('应该为每行显示操作按钮', () => {
    render(<SLATable {...defaultProps} />);

    const acknowledgeButtons = screen.getAllByRole('button', { name: /acknowledge/i });
    const resolveButtons = screen.getAllByRole('button', { name: /resolve/i });
    const detailButtons = screen.getAllByRole('button', { name: /details/i });

    expect(acknowledgeButtons.length).toBe(2);
    expect(resolveButtons.length).toBe(2);
    expect(detailButtons.length).toBe(2);
  });

  it('点击确认按钮调用 onAcknowledge', async () => {
    const user = userEvent.setup();
    render(<SLATable {...defaultProps} />);

    const acknowledgeButtons = screen.getAllByRole('button', { name: /acknowledge/i });
    await user.click(acknowledgeButtons[0]);

    expect(defaultProps.onAcknowledge).toHaveBeenCalledWith('violation-1');
  });

  it('点击解决按钮调用 onResolve', async () => {
    const user = userEvent.setup();
    render(<SLATable {...defaultProps} />);

    const resolveButtons = screen.getAllByRole('button', { name: /resolve/i });
    await user.click(resolveButtons[0]);

    expect(defaultProps.onResolve).toHaveBeenCalledWith('violation-1');
  });

  it('点击详情按钮调用 onViewDetails', async () => {
    const user = userEvent.setup();
    render(<SLATable {...defaultProps} />);

    const detailButtons = screen.getAllByRole('button', { name: /details/i });
    await user.click(detailButtons[0]);

    expect(defaultProps.onViewDetails).toHaveBeenCalledWith('violation-1');
  });

  it('加载状态应显示加载指示器', () => {
    render(<SLATable {...defaultProps} loading={true} />);

    expect(screen.getByRole('progressbar')).toBeInTheDocument();
  });

  it('无数据应显示空状态', () => {
    render(<SLATable {...defaultProps} violations={[]} />);

    expect(screen.getByText(/no violation/i)).toBeInTheDocument();
  });

  it('应该正确格式化日期', () => {
    render(<SLATable {...defaultProps} />);

    // 显示日期
    expect(screen.getByText(/2024|Jan|20/)).toBeInTheDocument();
  });

  it('应该显示 SLO 进度条', () => {
    render(<SLATable {...defaultProps} />);

    const progressBars = document.querySelectorAll('[data-slo-progress]');
    expect(progressBars.length).toBeGreaterThan(0);
  });

  it('应该根据严重程度应用颜色', () => {
    render(<SLATable {...defaultProps} />);

    const criticalCells = screen.getAllByText('critical');
    criticalCells.forEach(cell => {
      expect(cell.closest('td')).toHaveClass('severity-critical');
    });
  });

  it('应该支持行选择', () => {
    render(<SLATable {...defaultProps} selectable={true} />);

    const checkboxes = screen.getAllByRole('checkbox');
    expect(checkboxes.length).toBeGreaterThan(0);
  });

  it('选择行应触发选择事件', async () => {
    const user = userEvent.setup();
    const onSelectionChange = jest.fn();

    render(
      <SLATable
        {...defaultProps}
        selectable={true}
        onSelectionChange={onSelectionChange}
      />
    );

    const checkbox = screen.getByRole('checkbox');
    await user.click(checkbox);

    expect(onSelectionChange).toHaveBeenCalled();
  });

  it('应该支持批量操作', () => {
    render(
      <SLATable
        {...defaultProps}
        selectable={true}
        selectedRows={['violation-1']}
      />
    );

    const batchActions = screen.getByText(/batch action/i);
    expect(batchActions).toBeInTheDocument();
  });

  it('应该显示行数信息', () => {
    render(<SLATable {...defaultProps} />);

    expect(screen.getByText(/2 items/i)).toBeInTheDocument();
  });

  it('应该支持分页', () => {
    render(<SLATable {...defaultProps} pagination={true} total={50} />);

    expect(screen.getByText(/50 total/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /next/i })).toBeInTheDocument();
  });

  it('应该显示列可见性控件', () => {
    render(<SLATable {...defaultProps} />);

    const columnToggle = screen.getByLabelText(/column/i);
    expect(columnToggle).toBeInTheDocument();
  });

  it('应该允许导出表格数据', () => {
    render(<SLATable {...defaultProps} />);

    const exportButton = screen.getByRole('button', { name: /export/i });
    expect(exportButton).toBeInTheDocument();
  });

  it('应该支持列宽调整', () => {
    render(<SLATable {...defaultProps} />);

    const resizeHandle = document.querySelector('[data-resizable]');
    expect(resizeHandle).toBeInTheDocument();
  });

  it('应该显示行悬停效果', () => {
    render(<SLATable {...defaultProps} />);

    const rows = screen.getAllByRole('row');
    expect(rows.length).toBeGreaterThan(1); // 至少有一行数据 + 表头
  });

  it('应该固定列（如果配置）', () => {
    render(<SLATable {...defaultProps} fixedColumns={['ticket', 'status']} />);

    const fixedCells = document.querySelectorAll('[data-fixed]');
    expect(fixedCells.length).toBeGreaterThan(0);
  });

  it('应该显示行上下文菜单', async () => {
    const user = userEvent.setup();

    render(<SLATable {...defaultProps} />);

    const row = screen.getByText('Critical Incident').closest('tr');
    fireEvent.contextMenu(row!);

    const contextMenu = screen.getByRole('menu');
    expect(contextMenu).toBeInTheDocument();
  });

  it('应该支持列筛选', () => {
    render(<SLATable {...defaultProps} />);

    const columnFilters = screen.getByPlaceholderText(/filter/i);
    expect(columnFilters).toBeInTheDocument();
  });

  it('应该显示工具提示', () => {
    render(<SLATable {...defaultProps} showTooltips={true} />);

    const tooltips = document.querySelectorAll('[data-tooltip]');
    expect(tooltips.length).toBeGreaterThan(0);
  });
});
