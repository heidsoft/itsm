/**
 * TemplateCard - 工单模板卡片测试
 */

import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TemplateCard } from '../TemplateCard';
import type { TicketTemplate } from '../types';

jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

describe('TemplateCard', () => {
  const mockTemplate: TicketTemplate = {
    id: 'template-1',
    name: 'Network Issue',
    description: 'Template for network related issues',
    icon: 'network',
    category: 'network',
    priority: 'high',
    defaultValues: {
      title: 'Network Issue Report',
      type: 'incident',
      description: '',
    },
  };

  const mockOnSelect = jest.fn();

  const defaultProps = {
    template: mockTemplate,
    onSelect: mockOnSelect,
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('应该渲染模板名称', () => {
    render(<TemplateCard {...defaultProps} />);

    expect(screen.getByText(mockTemplate.name)).toBeInTheDocument();
  });

  it('应该渲染模板描述', () => {
    render(<TemplateCard {...defaultProps} />);

    expect(screen.getByText(mockTemplate.description)).toBeInTheDocument();
  });

  it('应该显示优先级指示器', () => {
    render(<TemplateCard {...defaultProps} />);

    expect(screen.getByText(mockTemplate.priority)).toBeInTheDocument();
  });

  it('应该显示分类标签', () => {
    render(<TemplateCard {...defaultProps} />);

    expect(screen.getByText(mockTemplate.category)).toBeInTheDocument();
  });

  it('点击卡片应调用 onSelect', async () => {
    const user = userEvent.setup();
    render(<TemplateCard {...defaultProps} />);

    const card = screen.getByText(mockTemplate.name).closest('div') ||
                 screen.getByRole('button');
    await user.click(card!);

    expect(mockOnSelect).toHaveBeenCalledWith(mockTemplate);
  });

  it('应该显示图标', () => {
    render(<TemplateCard {...defaultProps} />);

    // 检查图标是否存在（通过类名或文本）
    const iconElement = screen.getByText(mockTemplate.icon) ||
                       document.querySelector(`[data-icon="${mockTemplate.icon}"]`);
    expect(iconElement || document.querySelector('.icon')).toBeInTheDocument();
  });

  it('应该显示预填字段预览', () => {
    render(<TemplateCard {...defaultProps} />);

    expect(screen.getByText(mockTemplate.defaultValues.title)).toBeInTheDocument();
    expect(screen.getByText(mockTemplate.defaultValues.type)).toBeInTheDocument();
  });

  it('应该正确显示默认值', () => {
    const withDefaults = {
      ...mockTemplate,
      defaultValues: {
        title: 'Default Title',
        type: 'problem',
        description: 'Default description',
        priority: 'medium',
      },
    };

    render(<TemplateCard template={withDefaults} onSelect={mockOnSelect} />);

    expect(screen.getByText('Default Title')).toBeInTheDocument();
  });

  it('应该支持键盘访问', () => {
    render(<TemplateCard {...defaultProps} />);

    const card = screen.getByText(mockTemplate.name).closest('div');
    expect(card).toHaveAttribute('tabIndex', '0');
  });

  it('应该显示使用统计（如果有）', () => {
    const withStats = {
      ...mockTemplate,
      usageCount: 42,
    };

    render(<TemplateCard template={withStats} onSelect={mockOnSelect} />);

    expect(screen.getByText(/42/)).toBeInTheDocument();
  });

  it('鼠标悬停应有视觉反馈', () => {
    render(<TemplateCard {...defaultProps} />);

    const card = screen.getByText(mockTemplate.name).closest('div');
    expect(card).toHaveClass('hover');
  });

  it('应该显示创建时间', () => {
    const withDate = {
      ...mockTemplate,
      createdAt: '2024-01-15T10:00:00Z',
    };

    render(<TemplateCard template={withDate} onSelect={mockOnSelect} />);

    expect(screen.getByText(/2024|Jan|15/)).toBeInTheDocument();
  });

  it('应该正确处理空默认值', () => {
    const noDefaults = {
      ...mockTemplate,
      defaultValues: {},
    };

    render(<TemplateCard template={noDefaults} onSelect={mockOnSelect} />);

    // 不应崩溃
    expect(screen.getByText(mockTemplate.name)).toBeInTheDocument();
  });

  it('应该显示图标背景', () => {
    render(<TemplateCard {...defaultProps} />);

    const iconContainer = document.querySelector('.icon-background') ||
                         screen.getByText(mockTemplate.icon).closest('div');
    expect(iconContainer).toBeInTheDocument();
  });

  it('应该支持多种图标类型', () => {
    const differentIcons = [
      'network', 'server', 'database', 'cloud', 'security'
    ];

    differentIcons.forEach(icon => {
      const template = { ...mockTemplate, icon };
      const { unmount } = render(<TemplateCard template={template} onSelect={mockOnSelect} />);

      expect(screen.getByText(icon) || document.querySelector(`[data-icon="${icon}"]`)).toBeInTheDocument();
      unmount();
    });
  });
});
