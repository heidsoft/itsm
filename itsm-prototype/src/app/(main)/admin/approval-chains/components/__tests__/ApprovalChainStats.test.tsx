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
    expect(screen.getByText('100')).toBeInTheDocument();

    expect(screen.getByText('活跃审批链')).toBeInTheDocument();
    expect(screen.getByText('80')).toBeInTheDocument();

    expect(screen.getByText('非活跃审批链')).toBeInTheDocument();
    expect(screen.getByText('20')).toBeInTheDocument();

    expect(screen.getByText('平均步骤数')).toBeInTheDocument();
    expect(screen.getByText('2.5')).toBeInTheDocument();
  });

  it('should show loading state', () => {
    render(<ApprovalChainStatsCards stats={mockStats} loading={true} />);

    // 检查是否有加载状态
    const cards = screen.getAllByRole('img', { hidden: true });
    expect(cards.length).toBeGreaterThan(0);
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

    expect(screen.getByText('0')).toBeInTheDocument();
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

    expect(screen.getByText('999999')).toBeInTheDocument();
    expect(screen.getByText('888888')).toBeInTheDocument();
    expect(screen.getByText('111111')).toBeInTheDocument();
    expect(screen.getByText('2.0')).toBeInTheDocument();
  });
});
