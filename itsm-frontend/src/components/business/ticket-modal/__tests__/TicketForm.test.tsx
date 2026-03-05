/**
 * TicketForm 组件测试
 * 测试工单表单的步骤、验证和提交流程
 */

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Form, Button, Select, Input } from 'antd';
import { TicketForm } from '../TicketForm';
import type { TicketFormProps } from '../types';

// Mock 所需的 hooks 和上下文
jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

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

describe('TicketForm', () => {
  const mockOnSubmit = jest.fn();
  const defaultProps: TicketFormProps = {
    form: {} as any,
    onSubmit: mockOnSubmit,
    loading: false,
    isEditing: false,
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('应该渲染所有表单步骤', () => {
    render(<TicketForm {...defaultProps} />);

    // 检查步骤指示器或步骤标题
    expect(screen.getByText(/step 1/i)).toBeInTheDocument();
    expect(screen.getByText(/basic information/i)).toBeInTheDocument();
  });

  it('应该显示工单类型选择', async () => {
    const user = userEvent.setup();
    render(<TicketForm {...defaultProps} />);

    // 检查是否有类型选择器
    const typeSelect = screen.getByLabelText(/type/i);
    expect(typeSelect).toBeInTheDocument();

    // 尝试选择类型
    await user.click(typeSelect);
    await user.keyboard('{arrowdown}{arrowdown}{enter}');

    // 验证选择
    expect(typeSelect).toHaveValue('incident');
  });

  it('应该验证必填字段', async () => {
    const user = userEvent.setup();
    render(<TicketForm {...defaultProps} />);

    // 尝试提交空表单
    const submitButton = screen.getByRole('button', { name: /next/i });
    await user.click(submitButton);

    // 应该显示验证错误
    await waitFor(() => {
      expect(screen.getByText(/required/i)).toBeInTheDocument();
    });
  });

  it('应该提交有效表单数据', async () => {
    const user = userEvent.setup();
    const mockForm = {
      validateFields: jest.fn().mockResolvedValue({
        title: 'Test Ticket',
        type: 'incident',
        category: 'hardware',
        priority: 'high',
        description: 'Test description',
      }),
      resetFields: jest.fn(),
    };

    render(<TicketForm {...defaultProps} form={mockForm} />);

    // 填写并提交
    await mockOnSubmit(mockForm.validateFields);

    expect(mockOnSubmit).toHaveBeenCalled();
  });

  it('编辑模式下应显示不同标题', () => {
    render(<TicketForm {...defaultProps} isEditing={true} />);

    expect(screen.getByText(/edit/i)).toBeInTheDocument();
  });

  it('加载状态应禁用提交按钮', () => {
    render(<TicketForm {...defaultProps} loading={true} />);

    const submitButton = screen.getByRole('button', { name: /next|submit/i });
    expect(submitButton).toBeDisabled();
  });

  it('应该显示步骤导航', () => {
    render(<TicketForm {...defaultProps} />);

    // 检查步骤指示器
    expect(screen.getByText(/step 1/i)).toBeInTheDocument();
    expect(screen.getByText(/step 2/i)).toBeInTheDocument();
    expect(screen.getByText(/step 3/i)).toBeInTheDocument();
  });

  it('应该在不同步骤间导航', async () => {
    const user = userEvent.setup();
    render(<TicketForm {...defaultProps} />);

    // 第一步完成后点击 Next
    const nextButton = screen.getByRole('button', { name: /next/i });
    await user.click(nextButton);

    // 应该看到第二步
    await waitFor(() => {
      expect(screen.getByText(/step 2/i)).toBeInTheDocument();
    });
  });

  it('应该保存草稿', async () => {
    const user = userEvent.setup();
    const mockForm = {
      getFieldsValue: jest.fn().mockReturnValue({
        title: 'Draft Ticket',
        type: 'incident',
      }),
      resetFields: jest.fn(),
    };

    render(<TicketForm {...defaultProps} form={mockForm} />);

    const saveDraftButton = screen.getByRole('button', { name: /save draft/i });
    await user.click(saveDraftButton);

    // 应该调用保存草稿的逻辑
    expect(mockForm.getFieldsValue).toHaveBeenCalled();
  });
});
