/**
 * TicketFormStep2 - 附加信息步骤测试
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TicketFormStep2 } from '../TicketFormStep2';
import type { TicketFormStep2Props } from '../types';

jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

describe('TicketFormStep2', () => {
  const defaultProps: TicketFormStep2Props = {
    form: {} as any,
    onNext: jest.fn(),
    onBack: jest.fn(),
  };

  it('应该渲染附件和标签字段', () => {
    render(<TicketFormStep2 {...defaultProps} />);

    // 检查步骤标题
    expect(screen.getByText(/step 2/i)).toBeInTheDocument();

    // 检查上传区域
    expect(screen.getByText(/attachment/i)).toBeInTheDocument();
  });

  it('应该显示标签管理界面', () => {
    render(<TicketFormStep2 {...defaultProps} />);

    expect(screen.getByText(/tag/i)).toBeInTheDocument();
    expect(screen.getByText(/add tag/i)).toBeInTheDocument();
  });

  it('应该显示优先级选择', () => {
    render(<TicketFormStep2 {...defaultProps} />);

    expect(screen.getByLabelText(/priority/i)).toBeInTheDocument();
  });

  it('点击下一步应调用 onNext', async () => {
    const user = userEvent.setup();
    const mockOnNext = jest.fn();

    render(<TicketFormStep2 {...defaultProps} onNext={mockOnNext} />);

    const nextButton = screen.getByRole('button', { name: /next/i });
    await user.click(nextButton);

    expect(mockOnNext).toHaveBeenCalled();
  });

  it('点击返回应调用 onBack', async () => {
    const user = userEvent.setup();
    const mockOnBack = jest.fn();

    render(<TicketFormStep2 {...defaultProps} onBack={mockOnBack} />);

    const backButton = screen.getByRole('button', { name: /back/i });
    await user.click(backButton);

    expect(mockOnBack).toHaveBeenCalled();
  });

  it('应该显示文件上传区域', () => {
    render(<TicketFormStep2 {...defaultProps} />);

    const fileInput = screen.getByLabelText(/upload/i) ||
                     screen.getByText(/click to upload/i);
    expect(fileInput).toBeInTheDocument();
  });

  it('应该显示已上传文件列表', () => {
    const mockFiles = [
      { name: 'document.pdf', size: 1024, type: 'application/pdf' },
    ];

    render(<TicketFormStep2 {...defaultProps} />);

    // 如果有预定义的文件，应该显示
    mockFiles.forEach(file => {
      expect(screen.getByText(file.name)).toBeInTheDocument();
    });
  });

  it('应该允许选择分配人', () => {
    render(<TicketFormStep2 {...defaultProps} />);

    expect(screen.getByLabelText(/assignee/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/due date/i)).toBeInTheDocument();
  });

  it('应该显示截止日期选择器', () => {
    render(<TicketFormStep2 {...defaultProps} />);

    expect(screen.getByLabelText(/due date/i)).toBeInTheDocument();
  });

  it('应该支持拖放上传', () => {
    render(<TicketFormStep2 {...defaultProps} />);

    const uploadArea = screen.getByText(/drag and drop/i) ||
                      screen.getByLabelText(/upload/i);

    expect(uploadArea).toBeInTheDocument();
  });
});
