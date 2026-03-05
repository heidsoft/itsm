/**
 * CIDetail 组件测试
 * 配置项详情页的主容器组件
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import { CIDetail } from '../CIDetail';
import type { CIDetailProps } from '../types';

jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

describe('CIDetail', () => {
  const mockCI = {
    id: 'ci-001',
    name: 'Web Server 1',
    type: 'server',
    status: 'active',
    description: 'Production web server',
    ipAddress: '192.168.1.100',
    location: 'Data Center A',
    team: 'DevOps',
    environment: 'production',
  };

  const defaultProps: CIDetailProps = {
    ci: mockCI,
    loading: false,
    error: null,
    onEdit: jest.fn(),
    onDelete: jest.fn(),
    onRefresh: jest.fn(),
  };

  it('应该正确渲染 CI 名称', () => {
    render(<CIDetail {...defaultProps} />);

    expect(screen.getByText(mockCI.name)).toBeInTheDocument();
  });

  it('应该显示 CI 状态', () => {
    render(<CIDetail {...defaultProps} />);

    expect(screen.getByText(mockCI.status)).toBeInTheDocument();
  });

  it('应该显示所有基本信息字段', () => {
    render(<CIDetail {...defaultProps} />);

    expect(screen.getByText(/ip address/i)).toBeInTheDocument();
    expect(screen.getByText(mockCI.ipAddress)).toBeInTheDocument();
    expect(screen.getByText(/location/i)).toBeInTheDocument();
    expect(screen.getByText(mockCI.location)).toBeInTheDocument();
  });

  it('应该显示编辑和删除按钮', () => {
    render(<CIDetail {...defaultProps} />);

    expect(screen.getByRole('button', { name: /edit/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /delete/i })).toBeInTheDocument();
  });

  it('点击编辑应调用 onEdit', async () => {
    const user = userEvent.setup();
    render(<CIDetail {...defaultProps} />);

    const editButton = screen.getByRole('button', { name: /edit/i });
    await user.click(editButton);

    expect(defaultProps.onEdit).toHaveBeenCalledWith(mockCI.id);
  });

  it('点击删除应调用 onDelete', async () => {
    const user = userEvent.setup();
    render(<CIDetail {...defaultProps} />);

    const deleteButton = screen.getByRole('button', { name: /delete/i });
    await user.click(deleteButton);

    expect(defaultProps.onDelete).toHaveBeenCalledWith(mockCI.id);
  });

  it('应该显示刷新按钮', () => {
    render(<CIDetail {...defaultProps} />);

    expect(screen.getByRole('button', { name: /refresh/i })).toBeInTheDocument();
  });

  it('点击刷新应调用 onRefresh', async () => {
    const user = userEvent.setup();
    const mockOnRefresh = jest.fn();

    render(<CIDetail {...defaultProps} onRefresh={mockOnRefresh} />);

    const refreshButton = screen.getByRole('button', { name: /refresh/i });
    await user.click(refreshButton);

    expect(mockOnRefresh).toHaveBeenCalled();
  });

  it('加载状态应显示加载指示器', () => {
    render(<CIDetail {...defaultProps} loading={true} />);

    expect(screen.getByRole('progressbar') || screen.getByText(/loading/i)).toBeInTheDocument();
  });

  it('错误状态应显示错误信息', () => {
    const mockError = { message: 'Failed to load CI' };
    render(<CIDetail {...defaultProps} error={mockError} />);

    expect(screen.getByText(mockError.message)).toBeInTheDocument();
  });

  it('应该显示 CI 类型', () => {
    render(<CIDetail {...defaultProps} />);

    expect(screen.getByText(mockCI.type)).toBeInTheDocument();
  });

  it('应该显示所属团队', () => {
    render(<CIDetail {...defaultProps} />);

    expect(screen.getByText(mockCI.team)).toBeInTheDocument();
  });

  it('应该显示环境信息', () => {
    render(<CIDetail {...defaultProps} />);

    expect(screen.getByText(mockCI.environment)).toBeInTheDocument();
  });

  it('应该包含标签页导航', () => {
    render(<CIDetail {...defaultProps} />);

    expect(screen.getByText(/basic info/i)).toBeInTheDocument();
    expect(screen.getByText(/topology/i)).toBeInTheDocument();
    expect(screen.getByText(/relationships/i)).toBeInTheDocument();
    expect(screen.getByText(/impact analysis/i)).toBeInTheDocument();
    expect(screen.getByText(/change history/i)).toBeInTheDocument();
  });

  it('没有 CI 数据应显示空状态', () => {
    render(<CIDetail {...defaultProps} ci={null} />);

    expect(screen.getByText(/no data/i)).toBeInTheDocument();
  });

  it('应该显示描述信息', () => {
    render(<CIDetail {...defaultProps} />);

    expect(screen.getByText(mockCI.description)).toBeInTheDocument();
  });
});
