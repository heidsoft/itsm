/**
 * TicketFormStep3 - 确认和提交步骤测试
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TicketFormStep3 } from '../TicketFormStep3';
import type { TicketFormStep3Props } from '../types';

jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

describe('TicketFormStep3', () => {
  const mockFormData = {
    title: 'Test Ticket',
    type: 'incident',
    category: 'hardware',
    priority: 'high',
    description: 'Test description',
    attachments: [],
    tags: ['urgent'],
    assigneeId: 'user1',
    dueDate: '2024-12-31',
  };

  const defaultProps: TicketFormStep3Props = {
    formData: mockFormData,
    onBack: jest.fn(),
    onSubmit: jest.fn(),
    loading: false,
    isEditing: false,
  };

  it('应该显示表单数据摘要', () => {
    render(<TicketFormStep3 {...defaultProps} />);

    expect(screen.getByText(/summary/i)).toBeInTheDocument();
    expect(screen.getByText(mockFormData.title)).toBeInTheDocument();
    expect(screen.getByText(mockFormData.description)).toBeInTheDocument();
  });

  it('应该显示所有字段值', () => {
    render(<TicketFormStep3 {...defaultProps} />);

    expect(screen.getByText('incident')).toBeInTheDocument();
    expect(screen.getByText('hardware')).toBeInTheDocument();
    expect(screen.getByText('high')).toBeInTheDocument();
  });

  it('应该显示标签', () => {
    render(<TicketFormStep3 {...defaultProps} />);

    mockFormData.tags.forEach(tag => {
      expect(screen.getByText(tag)).toBeInTheDocument();
    });
  });

  it('点击提交应调用 onSubmit', async () => {
    const user = userEvent.setup();
    const mockOnSubmit = jest.fn();

    render(<TicketFormStep3 {...defaultProps} onSubmit={mockOnSubmit} />);

    const submitButton = screen.getByRole('button', { name: /submit/i });
    await user.click(submitButton);

    expect(mockOnSubmit).toHaveBeenCalledWith(mockFormData);
  });

  it('点击返回应调用 onBack', async () => {
    const user = userEvent.setup();
    const mockOnBack = jest.fn();

    render(<TicketFormStep3 {...defaultProps} onBack={mockOnBack} />);

    const backButton = screen.getByRole('button', { name: /back/i });
    await user.click(backButton);

    expect(mockOnBack).toHaveBeenCalled();
  });

  it('加载状态应禁用提交按钮', () => {
    render(<TicketFormStep3 {...defaultProps} loading={true} />);

    const submitButton = screen.getByRole('button', { name: /submit/i });
    expect(submitButton).toBeDisabled();
  });

  it('编辑模式应显示不同文本', () => {
    render(<TicketFormStep3 {...defaultProps} isEditing={true} />);

    expect(screen.getByText(/edit ticket/i)).toBeInTheDocument();
  });

  it('应该显示确认提示', () => {
    render(<TicketFormStep3 {...defaultProps} />);

    expect(screen.getByText(/confirm/i)).toBeInTheDocument();
  });

  it('应该显示截止日期', () => {
    render(<TicketFormStep3 {...defaultProps} />);

    expect(screen.getByText(mockFormData.dueDate)).toBeInTheDocument();
  });

  it('应该显示附件数量', () => {
    const withAttachments = {
      ...mockFormData,
      attachments: [
        { name: 'file1.pdf' },
        { name: 'file2.pdf' },
      ],
    };

    render(<TicketFormStep3 {...defaultProps} formData={withAttachments} />);

    expect(screen.getByText(/2 files/i)).toBeInTheDocument();
  });

  it('无附件时应显示无附件文本', () => {
    const noAttachments = {
      ...mockFormData,
      attachments: [],
    };

    render(<TicketFormStep3 {...defaultProps} formData={noAttachments} />);

    expect(screen.getByText(/no attachments/i)).toBeInTheDocument();
  });

  it('应该显示表单验证提示', () => {
    render(<TicketFormStep3 {...defaultProps} />);

    expect(screen.getByText(/review/i)).toBeInTheDocument();
    expect(screen.getByText(/submit/i)).toBeInTheDocument();
  });
});
