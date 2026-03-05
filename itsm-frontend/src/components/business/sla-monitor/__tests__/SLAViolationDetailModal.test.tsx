/**
 * SLAViolationDetailModal - SLA 违规详情模态框测试
 */

import React from 'react';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SLAViolationDetailModal } from '../components/SLAViolationDetailModal';
import type { SLAViolation } from '../types';

jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

describe('SLAViolationDetailModal', () => {
  const mockViolation: SLAViolation = {
    id: 'violation-1',
    ticketId: 'ticket-1',
    ticket: {
      id: 'ticket-1',
      title: 'Critical Incident',
      description: 'System outage affecting all users',
      priority: 'critical',
      status: 'open',
      createdBy: {
        id: 'user1',
        name: 'Admin User',
      },
      assignedTo: {
        id: 'user2',
        name: 'John Doe',
      },
      attachments: [],
    },
    policyId: 'policy-1',
    policyName: 'Response Time',
    policy: {
      id: 'policy-1',
      name: 'Response Time',
      metric: 'response_time',
      target: 60,
      unit: 'minutes',
      description: 'Initial response within 60 minutes',
    },
    breachedAt: '2024-01-20T10:00:00Z',
    currentValue: 120,
    targetValue: 60,
    slaMetric: 'response_time',
    status: 'open',
    severity: 'critical',
    assignedTo: {
      id: 'user2',
      name: 'John Doe',
      email: 'john@example.com',
      avatar: null,
    },
    acknowledgedBy: null,
    acknowledgedAt: null,
    resolvedAt: null,
    resolution: null,
    notes: [
      {
        id: 'note-1',
        content: 'Initial assessment completed',
        createdBy: 'user1',
        createdAt: '2024-01-20T10:30:00Z',
      },
    ],
  };

  const defaultProps = {
    visible: true,
    violation: mockViolation,
    onClose: jest.fn(),
    onAcknowledge: jest.fn(),
    onResolve: jest.fn(),
    onAddNote: jest.fn(),
    loading: false,
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('应该渲染模态框', () => {
    render(<SLAViolationDetailModal {...defaultProps} />);

    expect(screen.getByRole('dialog')).toBeInTheDocument();
  });

  it('应该显示违规 ID', () => {
    render(<SLAViolationDetailModal {...defaultProps} />);

    expect(screen.getByText('violation-1')).toBeInTheDocument();
  });

  it('应该显示工单详情', () => {
    render(<SLAViolationDetailModal {...defaultProps} />);

    expect(screen.getByText('Critical Incident')).toBeInTheDocument();
    expect(screen.getByText(/system outage/i)).toBeInTheDocument();
  });

  it('应该显示政策信息', () => {
    render(<SLAViolationDetailModal {...defaultProps} />);

    expect(screen.getByText('Response Time')).toBeInTheDocument();
    expect(screen.getByText(/initial response within 60 minutes/i)).toBeInTheDocument();
  });

  it('应该显示违规时间', () => {
    render(<SLAViolationDetailModal {...defaultProps} />);

    expect(screen.getByText(/2024|Jan|20/)).toBeInTheDocument();
  });

  it('应该显示当前值和目标值', () => {
    render(<SLAViolationDetailModal {...defaultProps} />);

    expect(screen.getByText('120')).toBeInTheDocument();
    expect(screen.getByText('60')).toBeInTheDocument();
  });

  it('应该显示状态', () => {
    render(<SLAViolationDetailModal {...defaultProps} />);

    expect(screen.getByText('open')).toBeInTheDocument();
  });

  it('应该显示严重程度', () => {
    render(<SLAViolationDetailModal {...defaultProps} />);

    expect(screen.getByText('critical')).toBeInTheDocument();
  });

  it('应该显示分配给谁', () => {
    render(<SLAViolationDetailModal {...defaultProps} />);

    expect(screen.getByText('John Doe')).toBeInTheDocument();
    expect(screen.getByText('john@example.com')).toBeInTheDocument();
  });

  it('应该显示备注', () => {
    render(<SLAViolationDetailModal {...defaultProps} />);

    expect(screen.getByText('Initial assessment completed')).toBeInTheDocument();
    expect(screen.getByText('user1')).toBeInTheDocument();
  });

  it('应该显示关闭按钮', () => {
    render(<SLAViolationDetailModal {...defaultProps} />);

    const closeButton = screen.getByRole('button', { name: /close/i });
    expect(closeButton).toBeInTheDocument();
  });

  it('点击关闭应调用 onClose', async () => {
    const user = userEvent.setup();
    render(<SLAViolationDetailModal {...defaultProps} />);

    const closeButton = screen.getByRole('button', { name: /close/i });
    await user.click(closeButton);

    expect(defaultProps.onClose).toHaveBeenCalled();
  });

  it('应该显示确认按钮', () => {
    render(<SLAViolationDetailModal {...defaultProps} />);

    const acknowledgeButton = screen.getByRole('button', { name: /acknowledge/i });
    expect(acknowledgeButton).toBeInTheDocument();
  });

  it('点击确认应调用 onAcknowledge', async () => {
    const user = userEvent.setup();
    render(<SLAViolationDetailModal {...defaultProps} />);

    const acknowledgeButton = screen.getByRole('button', { name: /acknowledge/i });
    await user.click(acknowledgeButton);

    expect(defaultProps.onAcknowledge).toHaveBeenCalledWith('violation-1');
  });

  it('应该显示解决按钮', () => {
    render(<SLAViolationDetailModal {...defaultProps} />);

    const resolveButton = screen.getByRole('button', { name: /resolve/i });
    expect(resolveButton).toBeInTheDocument();
  });

  it('点击解决应调用 onResolve', async () => {
    const user = userEvent.setup();
    render(<SLAViolationDetailModal {...defaultProps} />);

    const resolveButton = screen.getByRole('button', { name: /resolve/i });
    await user.click(resolveButton);

    expect(defaultProps.onResolve).toHaveBeenCalledWith('violation-1', expect.anything());
  });

  it('应该显示添加备注输入框', () => {
    render<SLAViolationDetailModal>(<SLAViolationDetailModal {...defaultProps} />);

    expect(screen.getByPlaceholderText(/add note/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /add/i })).toBeInTheDocument();
  });

  it('添加备注应调用 onAddNote', async () => {
    const user = userEvent.setup();
    render(<SLAViolationDetailModal {...defaultProps} />);

    const noteInput = screen.getByPlaceholderText(/add note/i);
    await user.type(noteInput, 'New note content');

    const addButton = screen.getByRole('button', { name: /add/i });
    await user.click(addButton);

    expect(defaultProps.onAddNote).toHaveBeenCalledWith('violation-1', 'New note content');
  });

  it('加载状态应禁用按钮', () => {
    render(<SLAViolationDetailModal {...defaultProps} loading={true} />);

    const acknowledgeButton = screen.getByRole('button', { name: /acknowledge/i });
    const resolveButton = screen.getByRole('button', { name: /resolve/i });

    expect(acknowledgeButton).toBeDisabled();
    expect(resolveButton).toBeDisabled();
  });

  it('应该显示时间线', () => {
    render(<SLAViolationDetailModal {...defaultProps} />);

    expect(screen.getByText(/timeline/i)).toBeInTheDocument();
    expect(screen.getByText('Initial assessment completed')).toBeInTheDocument();
  });

  it('应该显示创建者信息', () => {
    render(<SLAViolationDetailModal {...defaultProps} />);

    expect(screen.getByText('Admin User')).toBeInTheDocument();
  });

  it('应该显示进度指示器', () => {
    render(<SLAViolationDetailModal {...defaultProps} />);

    const progressBar = screen.getByRole('progressbar');
    expect(progressBar).toBeInTheDocument();
  });

  it('应该显示 SLO 违规详情', () => {
    render(<SLAViolationDetailModal {...defaultProps} />);

    expect(screen.getByText(/breached by 60 minutes/i)).toBeInTheDocument();
  });

  it('应该显示相关工单链接', () => {
    render(<SLAViolationDetailModal {...defaultProps} />);

    const ticketLink = screen.getByRole('link', { name: /ticket-1/i });
    expect(ticketLink).toBeInTheDocument();
  });

  it('关闭后不应渲染内容', () => {
    const { rerender } = render(
      <SLAViolationDetailModal {...defaultProps} visible={true} />
    );

    rerender(
      <SLAViolationDetailModal {...defaultProps} visible={false} />
    );

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('应该显示可能的违规类型', () => {
    render(<SLAViolationDetailModal {...defaultProps} />);

    expect(screen.getByText('response_time')).toBeInTheDocument();
  });

  it('应该显示 SLO 历史', () => {
    render<SLAViolationDetailModal>(<SLAViolationDetailModal {...defaultProps} />);

    expect(screen.getByText(/slo history/i)).toBeInTheDocument();
  });

  it('应该显示相似违规', () => {
    render(<SLAViolationDetailModal {...defaultProps} />);

    const similarSection = screen.getByText(/similar violations/i);
    expect(similarSection).toBeInTheDocument();
  });

  it('应该显示附件列表（如果有）', () => {
    const withAttachments = {
      ...mockViolation,
      ticket: {
        ...mockViolation.ticket,
        attachments: [
          { id: 'att-1', name: 'log.txt', size: 1024 },
        ],
      },
    };

    render(
      <SLAViolationDetailModal
        {...defaultProps}
        violation={withAttachments}
      />
    );

    expect(screen.getByText('log.txt')).toBeInTheDocument();
  });

  it('应该显示工单状态时间线', () => {
    render(<SLAViolationDetailModal {...defaultProps} />);

    expect(screen.getByText(/status timeline/i)).toBeInTheDocument();
  });

  it('应该显示确认时间（如果已确认）', () => {
    const acknowledgedViolation = {
      ...mockViolation,
      status: 'acknowledged',
      acknowledgedBy: {
        id: 'user3',
        name: 'Manager',
      },
      acknowledgedAt: '2024-01-20T10:15:00Z',
    };

    render(
      <SLAViolationDetailModal
        {...defaultProps}
        violation={acknowledgedViolation}
      />
    );

    expect(screen.getByText('Manager')).toBeInTheDocument();
    expect(screen.getByText(/10:15/i)).toBeInTheDocument();
  });

  it('应该支持全屏模式', () => {
    render(
      <SLAViolationDetailModal
        {...defaultProps}
        fullScreen={true}
      />
    );

    const dialog = screen.getByRole('dialog');
    expect(dialog).toHaveClass('fullscreen');
  });

  it('应该显示相关事件', () => {
    render(<SLAViolationDetailModal {...defaultProps} />);

    expect(screen.getByText(/related events/i)).toBeInTheDocument();
  });

  it('应该显示扩展操作菜单', () => {
    render(<SLAViolationDetailModal {...defaultProps} />);

    const menuButton = screen.getByRole('button', { name: /more actions/i });
    expect(menuButton).toBeInTheDocument();
  });

  it('应该显示 SLO 配置详情', () => {
    render(<SLAViolationDetailModal {...defaultProps} />);

    expect(screen.getByText(/slo configuration/i)).toBeInTheDocument();
  });
});
