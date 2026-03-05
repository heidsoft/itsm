/**
 * SLAViolationMonitor - SLA 违规监控组件测试
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SLAViolationMonitor } from '../components/SLAViolationMonitor';
import type { SLAViolation, SLAPolicy } from '../types';

jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

describe('SLAViolationMonitor', () => {
  const mockViolations: SLAViolation[] = [
    {
      id: 'violation-1',
      ticketId: 'ticket-1',
      ticket: { title: 'Critical Incident', priority: 'critical' },
      policyId: 'policy-1',
      policyName: 'Response Time',
      breachedAt: '2024-01-20T10:00:00Z',
      currentValue: 120,
      targetValue: 60,
      slaMetric: 'response_time',
      status: 'open',
      severity: 'critical',
      assignedTo: 'user1',
    },
    {
      id: 'violation-2',
      ticketId: 'ticket-2',
      ticket: { title: 'High Priority Issue', priority: 'high' },
      policyId: 'policy-2',
      policyName: 'Resolution Time',
      breachedAt: '2024-01-20T11:00:00Z',
      currentValue: 240,
      targetValue: 120,
      slaMetric: 'resolution_time',
      status: 'acknowledged',
      severity: 'high',
      assignedTo: 'user2',
    },
  ];

  const mockPolicies: SLAPolicy[] = [
    {
      id: 'policy-1',
      name: 'Response Time',
      metric: 'response_time',
      target: 60,
      unit: 'minutes',
    },
    {
      id: 'policy-2',
      name: 'Resolution Time',
      metric: 'resolution_time',
      target: 120,
      unit: 'minutes',
    },
  ];

  const defaultProps = {
    violations: mockViolations,
    policies: mockPolicies,
    loading: false,
    onAcknowledge: jest.fn(),
    onResolve: jest.fn(),
    onFilterChange: jest.fn(),
    refreshInterval: 60000,
  };

  it('应该渲染监控面板', () => {
    render(<SLAViolationMonitor {...defaultProps} />);

    expect(screen.getByText(/sla violation monitor/i)).toBeInTheDocument();
  });

  it('应该显示违规总数', () => {
    render(<SLAViolationMonitor {...defaultProps} />);

    expect(screen.getByText('2')).toBeInTheDocument();
  });

  it('应该显示统计卡片', () => {
    render(<SLAViolationMonitor {...defaultProps} />);

    expect(screen.getByText(/open/i)).toBeInTheDocument();
    expect(screen.getByText(/acknowledged/i)).toBeInTheDocument();
    expect(screen.getByText(/critical/i)).toBeInTheDocument();
  });

  it('应该显示过滤器面板', () => {
    render(<SLAViolationMonitor {...defaultProps} />);

    expect(screen.getByLabelText(/filter/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/status/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/severity/i)).toBeInTheDocument();
  });

  it('应该显示违规表格', () => {
    render(<SLAViolationMonitor {...defaultProps} />);

    expect(screen.getByText('Critical Incident')).toBeInTheDocument();
    expect(screen.getByText('High Priority Issue')).toBeInTheDocument();
  });

  it('应该显示图表面板', () => {
    render(<SLAViolationMonitor {...defaultProps} />);

    expect(screen.getByTestId('sla-chart')).toBeInTheDocument();
  });

  it('应该显示刷新按钮', () => {
    render(<SLAViolationMonitor {...defaultProps} />);

    expect(screen.getByRole('button', { name: /refresh/i })).toBeInTheDocument();
  });

  it('点击刷新应调用 onFilterChange', async () => {
    const user = userEvent.setup();
    const mockOnFilter = jest.fn();

    render(<SLAViolationMonitor {...defaultProps} onFilterChange={mockOnFilter} />);

    const refreshButton = screen.getByRole('button', { name: /refresh/i });
    await user.click(refreshButton);

    expect(mockOnFilter).toHaveBeenCalled();
  });

  it('应该显示每个违规的确认按钮', () => {
    render(<SLAViolationMonitor {...defaultProps} />);

    const acknowledgeButtons = screen.getAllByRole('button', { name: /acknowledge/i });
    expect(acknowledgeButtons.length).toBeGreaterThan(0);
  });

  it('点击确认应调用 onAcknowledge', async () => {
    const user = userEvent.setup();
    const mockOnAcknowledge =jest.fn();

    render(<SLAViolationMonitor {...defaultProps} onAcknowledge={mockOnAcknowledge} />);

    const acknowledgeButtons = screen.getAllByRole('button', { name: /acknowledge/i });
    await user.click(acknowledgeButtons[0]);

    expect(mockOnAcknowledge).toHaveBeenCalledWith('violation-1');
  });

  it('应该显示每个违规的解决按钮', () => {
    render(<SLAViolationMonitor {...defaultProps} />);

    const resolveButtons = screen.getAllByRole('button', { name: /resolve/i });
    expect(resolveButtons.length).toBeGreaterThan(0);
  });

  it('点击解决应调用 onResolve', async () => {
    const user = userEvent.setup();
    const mockOnResolve = jest.fn();

    render(<SLAViolationMonitor {...defaultProps} onResolve={mockOnResolve} />);

    const resolveButtons = screen.getAllByRole('button', { name: /resolve/i });
    await user.click(resolveButtons[0]);

    expect(mockOnResolve).toHaveBeenCalledWith('violation-1');
  });

  it('加载状态应显示加载指示器', () => {
    render(<SLAViolationMonitor {...defaultProps} loading={true} />);

    expect(screen.getByText(/loading/i)).toBeInTheDocument();
  });

  it('没有违规应显示空状态', () => {
    render(<SLAViolationMonitor {...defaultProps} violations={[]} />);

    expect(screen.getByText(/no violations/i)).toBeInTheDocument();
  });

  it('应该显示政策名称', () => {
    render(<SLAViolationMonitor {...defaultProps} />);

    expect(screen.getByText('Response Time')).toBeInTheDocument();
    expect(screen.getByText('Resolution Time')).toBeInTheDocument();
  });

  it('应该显示违规超时值', () => {
    render(<SLAViolationMonitor {...defaultProps} />);

    expect(screen.getByText('120')).toBeInTheDocument();
    expect(screen.getByText('240')).toBeInTheDocument();
  });

  it('应该显示目标值', () => {
    render(<SLAViolationMonitor {...defaultProps} />);

    expect(screen.getByText('/ 60')).toBeInTheDocument();
    expect(screen.getByText('/ 120')).toBeInTheDocument();
  });

  it('应该显示工单优先级', () => {
    render(<SLAViolationMonitor {...defaultProps} />);

    expect(screen.getByText('critical')).toBeInTheDocument();
    expect(screen.getByText('high')).toBeInTheDocument();
  });

  it('应该显示违规时间', () => {
    render(<SLAViolationMonitor {...defaultProps} />);

    expect(screen.getByText(/2024|Jan|20/)).toBeInTheDocument();
  });

  it('应该显示分配给人', () => {
    render(<SLAViolationMonitor {...defaultProps} />);

    expect(screen.getByText('user1')).toBeInTheDocument();
    expect(screen.getByText('user2')).toBeInTheDocument();
  });

  it('应该显示严重程度徽章', () => {
    render(<SLAViolationMonitor {...defaultProps} />);

    const criticalBadges = screen.getAllByText('critical');
    expect(criticalBadges.length).toBeGreaterThan(0);
  });

  it('应该支持状态过滤', async () => {
    const user = userEvent.setup();
    const { rerender } = render(<SLAViolationMonitor {...defaultProps} />);

    // 过滤 open 状态
    rerender(
      <SLAViolationMonitor {...defaultProps} filterStatus="open" />
    );

    expect(screen.getByText('Critical Incident')).toBeInTheDocument();
    expect(screen.queryByText('High Priority Issue')).not.toBeInTheDocument();
  });

  it('应该支持严重程度过滤', async () => {
    const user = userEvent.setup();
    const { rerender } = render(<SLAViolationMonitor {...defaultProps} />);

    // 过滤 critical 严重程度
    rerender(
      <SLAViolationMonitor {...defaultProps} filterSeverity="critical" />
    );

    expect(screen.getByText('Critical Incident')).toBeInTheDocument();
    expect(screen.queryByText('High Priority Issue')).not.toBeInTheDocument();
  });

  it('应该显示导出按钮', () => {
    render(<SLAViolationMonitor {...defaultProps} />);

    expect(screen.getByRole('button', { name: /export/i })).toBeInTheDocument();
  });

  it('应该显示时间范围选择器', () => {
    render(<SLAViolationMonitor {...defaultProps} />);

    expect(screen.getByLabelText(/time range/i)).toBeInTheDocument();
  });

  it('应该显示分页控件', () => {
    render(<SLAViolationMonitor {...defaultProps} />);

    expect(screen.getByRole('button', { name: /next page/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /previous page/i })).toBeInTheDocument();
  });

  it('应该显示自动刷新指示器', () => {
    render(<SLAViolationMonitor {...defaultProps} refreshInterval={60000} />);

    expect(screen.getByText(/auto-refresh/i)).toBeInTheDocument();
  });

  it('应该可以暂停自动刷新', async () => {
    const user = userEvent.setup();
    const { rerender } = render(<SLAViolationMonitor {...defaultProps} />);

    const pauseButton = screen.getByRole('button', { name: /pause refresh/i });
    await user.click(pauseButton);

    rerender(
      <SLAViolationMonitor {...defaultProps} refreshEnabled={false} />
    );

    expect(screen.getByText(/paused/i)).toBeInTheDocument();
  });

  it('应该显示违规详情模态框', async () => {
    const user = userEvent.setup();

    render(<SLAViolationMonitor {...defaultProps} />);

    const detailButtons = screen.getAllByRole('button', { name: /view details/i });
    await user.click(detailButtons[0]);

    expect(screen.getByRole('dialog')).toBeInTheDocument();
  });

  it('应该支持批量操作', () => {
    render(<SLAViolationMonitor {...defaultProps} />);

    expect(screen.getByRole('button', { name: /bulk acknowledge/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /bulk resolve/i })).toBeInTheDocument();
  });

  it('应该显示排序列头', () => {
    render(<SLAViolationMonitor {...defaultProps} />);

    expect(screen.getByText(/severity/i)).toBeInTheDocument();
    expect(screen.getByText(/breached at/i)).toBeInTheDocument();
    expect(screen.getByText(/ticket/i)).toBeInTheDocument();
  });

  it('应该支持列显示配置', () => {
    render(<SLAViolationMonitor {...defaultProps} />);

    const columnSelector = screen.getByLabelText(/columns/i);
    expect(columnSelector).toBeInTheDocument();
  });

  it('应该显示总览摘要', () => {
    render(<SLAViolationMonitor {...defaultProps} />);

    expect(screen.getByText(/summary/i)).toBeInTheDocument();
    expect(screen.getByText('2')).toBeInTheDocument(); // Total violations
  });

  it('应该显示 SLA 策略信息', () => {
    render(<SLAViolationMonitor {...defaultProps} />);

    expect(screen.getByText('60 min')).toBeInTheDocument();
    expect(screen.getByText('120 min')).toBeInTheDocument();
  });
});
