/**
 * TicketPreviewStep - 预览步骤测试
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TicketPreviewStep } from '../TicketPreviewStep';
import type { TicketPreviewStepProps } from '../types';

jest.mock('@/lib/i18n', () => {
  return {
    useI18n: () => ({
      t: (key: string) => key,
    }),
  };
});

describe('TicketPreviewStep', () => {
  const mockFormData = {
    title: 'Preview Ticket',
    type: 'incident',
    category: 'network',
    priority: 'critical',
    description: 'This is a preview of the ticket',
    attachments: [{ name: 'attachment.pdf', size: 1024 }],
    tags: ['network', 'urgent'],
    assigneeId: 'user123',
    dueDate: '2024-12-31T23:59:59Z',
  };

  const defaultProps: TicketPreviewStepProps = {
    formData: mockFormData,
    onSubmit: jest.fn(),
    onBack: jest.fn(),
    loading: false,
  };

  it('应该渲染预览卡片', () => {
    render(<TicketPreviewStep {...defaultProps} />);

    expect(screen.getByText(/preview/i)).toBeInTheDocument();
  });

  it('应该显示标题', () => {
    render(<TicketPreviewStep {...defaultProps} />);

    expect(screen.getByText(mockFormData.title)).toBeInTheDocument();
  });

  it('应该显示描述', () => {
    render(<TicketPreviewStep {...defaultProps} />);

    expect(screen.getByText(mockFormData.description)).toBeInTheDocument();
  });

  it('应该显示类型和分类', () => {
    render(<TicketPreviewStep {...defaultProps} />);

    expect(screen.getByText(mockFormData.type)).toBeInTheDocument();
    expect(screen.getByText(mockFormData.category)).toBeInTheDocument();
  });

  it('应该显示优先级', () => {
    render(<TicketPreviewStep {...defaultProps} />);

    expect(screen.getByText(mockFormData.priority)).toBeInTheDocument();
  });

  it('应该显示标签', () => {
    render(<TicketPreviewStep {...defaultProps} />);

    mockFormData.tags.forEach(tag => {
      expect(screen.getByText(tag)).toBeInTheDocument();
    });
  });

  it('应该显示附件信息', () => {
    render(<TicketPreviewStep {...defaultProps} />);

    expect(screen.getByText(mockFormData.attachments[0].name)).toBeInTheDocument();
    expect(screen.getByText(/1 file/i)).toBeInTheDocument();
  });

  it('应该显示截止日期', () => {
    render(<TicketPreviewStep {...defaultProps} />);

    // 格式化后的日期应该显示
    expect(screen.getByText(/2024|Dec|31/)).toBeInTheDocument();
  });

  it('点击提交应调用 onSubmit', async () => {
    const user = userEvent.setup();
    const mockOnSubmit = jest.fn();

    render(<TicketPreviewStep {...defaultProps} onSubmit={mockOnSubmit} />);

    const submitButton = screen.getByRole('button', { name: /submit|confirm/i });
    await user.click(submitButton);

    expect(mockOnSubmit).toHaveBeenCalledWith(mockFormData);
  });

  it('点击返回应调用 onBack', async () => {
    const user = userEvent.setup();
    const mockOnBack = jest.fn();

    render(<TicketPreviewStep {...defaultProps} onBack={mockOnBack} />);

    const backButton = screen.getByRole('button', { name: /back/i });
    await user.click(backButton);

    expect(mockOnBack).toHaveBeenCalled();
  });

  it('加载状态应显示加载指示器', () => {
    render(<TicketPreviewStep {...defaultProps} loading={true} />);

    const submitButton = screen.getByRole('button', { name: /submit/i });
    expect(submitButton).toBeDisabled();
  });

  it('没有附件时应显示无附件', () => {
    const noAttachments = {
      ...mockFormData,
      attachments: [],
    };

    render(<TicketPreviewStep {...defaultProps} formData={noAttachments} />);

    expect(screen.getByText(/no attachments/i)).toBeInTheDocument();
  });

  it('应该显示表单部分标题', () => {
    render(<TicketPreviewStep {...defaultProps} />);

    expect(screen.getByText(/basic information/i)).toBeInTheDocument();
    expect(screen.getByText(/attachments/i)).toBeInTheDocument();
    expect(screen.getByText(/tags/i)).toBeInTheDocument();
  });

  it('应该格式化文件大小', () => {
    const withLargeFile = {
      ...mockFormData,
      attachments: [{ name: 'large.pdf', size: 1048576 }], // 1MB
    };

    render(<TicketPreviewStep {...defaultProps} formData={withLargeFile} />);

    expect(screen.getByText(/1 MB|1.0 MB/i)).toBeInTheDocument();
  });
});
