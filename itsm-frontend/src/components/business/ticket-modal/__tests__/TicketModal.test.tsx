/**
 * TicketModal 组件测试
 * 测试工单弹窗的显示、提交、取消等功能
 */

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Modal, Form, Button } from 'antd';
import { TicketModal } from '../TicketModal';
import type { Ticket } from '@/lib/services/ticket-service';

// Mock useI18n
jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

// Mock App.useApp
jest.mock('antd', () => ({
  ...jest.requireActual('antd'),
  App: {
    useApp: () => ({
      message: {
        success: jest.fn(),
        error: jest.fn(),
      },
    }),
  },
}));

describe('TicketModal', () => {
  const mockOnCancel = jest.fn();
  const mockOnSubmit = jest.fn();
  const defaultProps = {
    visible: true,
    onCancel: mockOnCancel,
    onSubmit: mockOnSubmit,
    loading: false,
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('应该正确渲染创建工单弹窗', () => {
    render(<TicketModal {...defaultProps} />);

    // 检查标题
    expect(screen.getByText('tickets.create')).toBeInTheDocument();
    expect(screen.getByText('tickets.createDescription')).toBeInTheDocument();
  });

  it('应该正确渲染编辑工单弹窗', () => {
    const editingTicket: Ticket = {
      id: '1',
      title: 'Test Ticket',
      type: 'incident',
      category: 'hardware',
      priority: 'high',
      assigneeId: 'user1',
      description: 'Test description',
      status: 'open',
      createdAt: '2024-01-01T00:00:00Z',
      updatedAt: '2024-01-01T00:00:00Z',
    };

    render(<TicketModal {...defaultProps} editingTicket={editingTicket} />);

    // 检查标题
    expect(screen.getByText('tickets.edit')).toBeInTheDocument();
    expect(screen.getByText('tickets.editDescription')).toBeInTheDocument();
  });

  it('应该显示表单', () => {
    render(<TicketModal {...defaultProps} />);

    // 检查 TicketForm 存在（通过检查表单字段）
    expect(screen.getByLabelText(/title/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/description/i)).toBeInTheDocument();
  });

  it('点击取消按钮应调用 onCancel', async () => {
    const user = userEvent.setup();
    render(<TicketModal {...defaultProps} />);

    // 找到取消按钮（Modal 的 X 按钮或 footer 中的取消按钮）
    const closeButton = screen.getByRole('button', { name: /close/i }) ||
                       screen.getByLabelText(/close/i);

    if (closeButton) {
      await user.click(closeButton);
      expect(mockOnCancel).toHaveBeenCalledTimes(1);
    }
  });

  it('编辑模式下应预填充表单数据', async () => {
    const editingTicket: Ticket = {
      id: '1',
      title: 'Existing Ticket',
      type: 'problem',
      category: 'software',
      priority: 'medium',
      assigneeId: 'user2',
      description: 'Existing description',
      status: 'open',
      createdAt: '2024-01-01T00:00:00Z',
      updatedAt: '2024-01-01T00:00:00Z',
    };

    render(<TicketModal {...defaultProps} editingTicket={editingTicket} />);

    // 检查表单字段是否预填充
    const titleInput = screen.getByLabelText(/title/i);
    expect(titleInput).toHaveValue('Existing Ticket');

    const descInput = screen.getByLabelText(/description/i);
    expect(descInput).toHaveValue('Existing description');
  });

  it('切换 visible 时应重置表单', async () => {
    const { rerender } = render(<TicketModal {...defaultProps} />);

    // 关闭弹窗
    rerender(<TicketModal {...defaultProps} visible={false} />);

    // 重新打开
    rerender(<TicketModal {...defaultProps} visible={true} />);

    // 表单应该是空的（创建模式）
    const titleInput = screen.getByLabelText(/title/i);
    expect(titleInput).toHaveValue('');
  });

  it('提交表单时应调用 onSubmit', async () => {
    const user = userEvent.setup();
    render(<TicketModal {...defaultProps} />);

    // 填写表单
    const titleInput = screen.getByLabelText(/title/i);
    await user.type(titleInput, 'New Ticket');

    const descInput = screen.getByLabelText(/description/i);
    await user.type(descInput, 'New description');

    // 提交表单（假设有一个提交按钮）
    const submitButton = screen.getByRole('button', { name: /submit/i });
    await user.click(submitButton);

    expect(mockOnSubmit).toHaveBeenCalled();
  });

  it('加载状态应显示加载指示器', () => {
    render(<TicketModal {...defaultProps} loading={true} />);

    // 检查是否有加载状态（按钮禁用或显示 spinner）
    const submitButton = screen.getByRole('button', { name: /submit/i });
    expect(submitButton).toBeDisabled();
  });

  it('应该显示模板卡片部分', () => {
    render(<TicketModal {...defaultProps} />);

    // 检查 TemplateCard 是否存在（通过文本内容）
    expect(screen.getByText(/template/i)).toBeInTheDocument();
  });
});
