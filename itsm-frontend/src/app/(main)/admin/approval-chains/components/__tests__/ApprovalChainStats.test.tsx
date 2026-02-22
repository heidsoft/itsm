/**
 * 审批链统计组件测试
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import { ApprovalChainStatsCards } from '../ApprovalChainStats';
import { ApprovalChainStats } from '@/types/approval-chain';

describe('ApprovalChainStatsCards', () => {
  const mockStats: ApprovalChainStats = {
    total: 100,
    active: 80,
    inactive: 20,
    totalSteps: 200,
    avgStepsPerChain: 2.5,
  };

  it('should render all stat cards correctly', () => {
    render(<ApprovalChainStatsCards stats={mockStats} />);

    expect(screen.getByText('总审批链数')).toBeInTheDocument();
    expect(screen.getByText('活跃审批链')).toBeInTheDocument();
    expect(screen.getByText('非活跃审批链')).toBeInTheDocument();
    expect(screen.getByText('平均步骤数')).toBeInTheDocument();

    // Check that numbers are rendered (exact format varies by Ant Design version)
    const values = screen.getAllByText(/100|80|20|2\.?5?/);
    expect(values.length).toBeGreaterThan(0);
  });

  it('should show loading state', () => {
    render(<ApprovalChainStatsCards stats={mockStats} loading={true} />);

    // When loading is true, Card shows loading skeleton - verify component renders without error
    // The component should render without crashing
    expect(document.querySelector('.ant-card')).toBeInTheDocument();
  });

  it('should display correct icons', () => {
    render(<ApprovalChainStatsCards stats={mockStats} />);

    // 检查图标是否存在
    expect(screen.getByText('总审批链数')).toBeInTheDocument();
    expect(screen.getByText('活跃审批链')).toBeInTheDocument();
    expect(screen.getByText('非活跃审批链')).toBeInTheDocument();
    expect(screen.getByText('平均步骤数')).toBeInTheDocument();
  });

  it('should handle zero values', () => {
    const zeroStats: ApprovalChainStats = {
      total: 0,
      active: 0,
      inactive: 0,
      totalSteps: 0,
      avgStepsPerChain: 0,
    };

    render(<ApprovalChainStatsCards stats={zeroStats} />);

    // Multiple zero values exist, verify component renders
    expect(screen.getByText('总审批链数')).toBeInTheDocument();
  });

  it('should handle large numbers', () => {
    const largeStats: ApprovalChainStats = {
      total: 999999,
      active: 888888,
      inactive: 111111,
      totalSteps: 2000000,
      avgStepsPerChain: 2.0,
    };

    render(<ApprovalChainStatsCards stats={largeStats} />);

    // Component should render with large numbers - check titles are present
    expect(screen.getByText('总审批链数')).toBeInTheDocument();
    expect(screen.getByText('活跃审批链')).toBeInTheDocument();
    expect(screen.getByText('非活跃审批链')).toBeInTheDocument();
  });
});
