/**
 * TicketFormStep1 - 基础信息步骤测试
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TicketFormStep1 } from '../TicketFormStep1';
import type { TicketFormStep1Props } from '../types';

jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

describe('TicketFormStep1', () => {
  const defaultProps: TicketFormStep1Props = {
    form: {} as any,
    onNext: jest.fn(),
    isEditing: false,
  };

  it('应该渲染所有基础信息字段', () => {
    render(<TicketFormStep1 {...defaultProps} />);

    expect(screen.getByLabelText(/title/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/description/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/type/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/category/i)).toBeInTheDocument();
  });

  it('应该验证标题字段', async () => {
    const user = userEvent.setup();
    render(<TicketFormStep1 {...defaultProps} />);

    // 不填写标题直接点击下一步
    const nextButton = screen.getByRole('button', { name: /next/i });
    await user.click(nextButton);

    // 应该显示验证错误
    expect(await screen.findByText(/required/i)).toBeInTheDocument();
  });

  it('应该允许输入标题', async () => {
    const user = userEvent.setup();
    render(<TicketFormStep1 {...defaultProps} />);

    const titleInput = screen.getByLabelText(/title/i);
    await user.type(titleInput, 'Test Ticket Title');

    expect(titleInput).toHaveValue('Test Ticket Title');
  });

  it('应该显示标题字符计数', () => {
    render(<TicketFormStep1 {...defaultProps} />);

    const titleInput = screen.getByLabelText(/title/i);
    expect(titleInput).toHaveAttribute('maxLength', '200');
  });

  it('应该显示描述字段为必填', () => {
    render(<TicketFormStep1 {...defaultProps} />);

    const descInput = screen.getByLabelText(/description/i);
    expect(descInput).toHaveAttribute('required', 'true');
  });

  it('应该限制描述长度', () => {
    render(<TicketFormStep1 {...defaultProps} />);

    const descInput = screen.getByLabelText(/description/i);
    expect(descInput).toHaveAttribute('maxLength', '5000');
  });

  it('应该预填充编辑数据', () => {
    const mockForm = {
      getFieldValue: jest.fn().mockReturnValue('Incident'),
      setFieldsValue: jest.fn(),
    };

    render(<TicketFormStep1 form={mockForm} {...defaultProps} isEditing={true} />);

    expect(mockForm.getFieldValue).toHaveBeenCalledWith('title');
  });
});
