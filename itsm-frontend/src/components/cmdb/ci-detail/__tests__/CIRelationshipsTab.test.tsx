/**
 * CIRelationshipsTab - CI 关系标签页测试
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CIRelationshipsTab } from '../sections/CIRelationshipsTab';
import type { CIRelationshipsTabProps } from '../types';

jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

describe('CIRelationshipsTab', () => {
  const mockRelationships = [
    {
      id: 'rel-1',
      type: 'depends_on',
      targetCi: {
        id: 'ci-002',
        name: 'Database Server',
        type: 'database',
        status: 'active',
      },
      description: 'Web server depends on database',
    },
    {
      id: 'rel-2',
      type: 'contains',
      targetCi: {
        id: 'ci-003',
        name: 'Application Service',
        type: 'service',
        status: 'running',
      },
      description: 'Server hosts application service',
    },
  ];

  const defaultProps: CIRelationshipsTabProps = {
    relationships: mockRelationships,
    loading: false,
    onAddRelationship: jest.fn(),
    onRemoveRelationship: jest.fn(),
  };

  it('应该显示关系总数', () => {
    render(<CIRelationshipsTab {...defaultProps} />);

    expect(screen.getByText(/2/)).toBeInTheDocument();
    expect(screen.getByText(/relationship/i)).toBeInTheDocument();
  });

  it('应该显示所有关系类型', () => {
    render(<CIRelationshipsTab {...defaultProps} />);

    expect(screen.getByText('depends_on')).toBeInTheDocument();
    expect(screen.getByText('contains')).toBeInTheDocument();
  });

  it('应该显示目标 CI 名称', () => {
    render(<CIRelationshipsTab {...defaultProps} />);

    expect(screen.getByText('Database Server')).toBeInTheDocument();
    expect(screen.getByText('Application Service')).toBeInTheDocument();
  });

  it('应该显示目标 CI 类型', () => {
    render(<CIRelationshipsTab {...defaultProps} />);

    expect(screen.getByText('database')).toBeInTheDocument();
    expect(screen.getByText('service')).toBeInTheDocument();
  });

  it('应该显示关系描述', () => {
    render(<CIRelationshipsTab {...defaultProps} />);

    expect(screen.getByText(/Web server depends on database/)).toBeInTheDocument();
  });

  it('应该显示添加关系按钮', () => {
    render(<CIRelationshipsTab {...defaultProps} />);

    expect(screen.getByRole('button', { name: /add relationship/i })).toBeInTheDocument();
  });

  it('点击添加按钮应调用 onAddRelationship', async () => {
    const user = userEvent.setup();
    const mockOnAdd = jest.fn();

    render(<CIRelationshipsTab {...defaultProps} onAddRelationship={mockOnAdd} />);

    const addButton = screen.getByRole('button', { name: /add relationship/i });
    await user.click(addButton);

    expect(mockOnAdd).toHaveBeenCalled();
  });

  it('应该显示每个关系的移除按钮', () => {
    render(<CIRelationshipsTab {...defaultProps} />);

    const removeButtons = screen.getAllByRole('button', { name: /remove|delete/i });
    expect(removeButtons.length).toBe(mockRelationships.length);
  });

  it('点击移除应调用 onRemoveRelationship', async () => {
    const user = userEvent.setup();
    const mockOnRemove = jest.fn();

    render(<CIRelationshipsTab {...defaultProps} onRemoveRelationship={mockOnRemove} />);

    const removeButtons = screen.getAllByRole('button', { name: /remove|delete/i });
    await user.click(removeButtons[0]);

    expect(mockOnRemove).toHaveBeenCalledWith('rel-1');
  });

  it('加载状态应显示加载指示器', () => {
    render(<CIRelationshipsTab {...defaultProps} loading={true} />);

    expect(screen.getByText(/loading/i)).toBeInTheDocument();
  });

  it('没有关系应显示空状态', () => {
    render(<CIRelationshipsTab {...defaultProps} relationships={[]} />);

    expect(screen.getByText(/no relationships/i)).toBeInTheDocument();
  });

  it('应该显示关系类型图标', () => {
    render(<CIRelationshipsTab {...defaultProps} />);

    // 检查是否存在图标元素
    const icons = document.querySelectorAll('[data-relationship-type]');
    expect(icons.length).toBeGreaterThan(0);
  });

  it('应该根据类型显示不同颜色', () => {
    render(<CIRelationshipsTab {...defaultProps} />);

    const dependsOn = screen.getByText('depends_on');
    expect(dependsOn.closest('div')).toHaveClass('type-depends_on');

    const contains = screen.getByText('contains');
    expect(contains.closest('div')).toHaveClass('type-contains');
  });

  it('应该显示目标 CI 状态', () => {
    render(<CIRelationshipsTab {...defaultProps} />);

    expect(screen.getByText('active')).toBeInTheDocument();
    expect(screen.getByText('running')).toBeInTheDocument();
  });

  it('应该显示关系方向', () => {
    render(<CIRelationshipsTab {...defaultProps} />);

    // 方向指示器（箭头图标等）
    const arrows = document.querySelectorAll('[data-direction]');
    expect(arrows.length).toBeGreaterThan(0);
  });

  it('应该允许搜索过滤', () => {
    render(<CIRelationshipsTab {...defaultProps} />);

    const searchInput = screen.getByPlaceholderText(/search/i);
    expect(searchInput).toBeInTheDocument();
  });

  it('搜索应过滤关系列表', async () => {
    const user = userEvent.setup();
    render(<CIRelationshipsTab {...defaultProps} />);

    const searchInput = screen.getByPlaceholderText(/search/i);
    await user.type(searchInput, 'Database');

    // 应该只显示包含 "Database" 的关系
    expect(screen.getByText('Database Server')).toBeInTheDocument();
    expect(screen.queryByText('Application Service')).not.toBeInTheDocument();
  });

  it('应该支持排序', () => {
    render(<CIRelationshipsTab {...defaultProps} />);

    const sortSelect = screen.getByLabelText(/sort/i);
    expect(sortSelect).toBeInTheDocument();
  });

  it('应该显示关系计数', () => {
    render(<CIRelationshipsTab {...defaultProps} />);

    expect(screen.getByText(/2\s+relationships/i)).toBeInTheDocument();
  });
});
