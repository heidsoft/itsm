/**
 * SLAStatisticsCards - SLA 统计卡片组件测试
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import { SLAStatisticsCards } from '../components/SLAStatisticsCards';
import type { SLAStatistics } from '../types';

jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

describe('SLAStatisticsCards', () => {
  const mockStats: SLAStatistics = {
    totalViolations: 25,
    openViolations: 8,
    criticalViolations: 5,
    complianceRate: 87.5,
    avgResolutionTime: 120,
    trend: 'improving',
    totalPolicies: 5,
    resolvedLast24h: 3,
  };

  it('应该显示总违规数', () => {
    render(<SLAStatisticsCards statistics={mockStats} />);

    expect(screen.getByText('25')).toBeInTheDocument();
    expect(screen.getByText(/total violation/i)).toBeInTheDocument();
  });

  it('应该显示开放违规数', () => {
    render(<SLAStatisticsCards statistics={mockStats} />);

    expect(screen.getByText('8')).toBeInTheDocument();
    expect(screen.getByText(/open/i)).toBeInTheDocument();
  });

  it('应该显示严重违规数', () => {
    render(<SLAStatisticsCards statistics={mockStats} />);

    expect(screen.getByText('5')).toBeInTheDocument();
    expect(screen.getByText(/critical/i)).toBeInTheDocument();
  });

  it('应该显示合规率', () => {
    render(<SLAStatisticsCards statistics={mockStats} />);

    expect(screen.getByText('87.5%')).toBeInTheDocument();
    expect(screen.getByText(/compliance/i)).toBeInTheDocument();
  });

  it('应该显示平均解决时间', () => {
    render(<SLAStatisticsCards statistics={mockStats} />);

    expect(screen.getByText('120')).toBeInTheDocument();
    expect(screen.getByText(/resolution time/i)).toBeInTheDocument();
  });

  it('应该显示趋势指示器', () => {
    render(<SLAStatisticsCards statistics={mockStats} />);

    expect(screen.getByText('improving')).toBeInTheDocument();
  });

  it('应该显示总策略数', () => {
    render(<SLAStatisticsCards statistics={mockStats} />);

    expect(screen.getByText('5')).toBeInTheDocument();
    expect(screen.getByText(/policy/i)).toBeInTheDocument();
  });

  it('应该显示24小时解决数', () => {
    render(<SLAStatisticsCards statistics={mockStats} />);

    expect(screen.getByText('3')).toBeInTheDocument();
    expect(screen.getByText(/24h|24 hours?/i)).toBeInTheDocument();
  });

  it('高严重程度应有红色警示', () => {
    const criticalStats = {
      ...mockStats,
      criticalViolations: 15, // 高数量
    };

    render(<SLAStatisticsCards statistics={criticalStats} />);

    const criticalCard = screen.getByText('15').closest('div');
    expect(criticalCard).toHaveClass('critical-high');
  });

  it('低合规率应显示警告', () => {
    const poorStats = {
      ...mockStats,
      complianceRate: 60,
    };

    render(<SLAStatisticsCards statistics={poorStats} />);

    expect(screen.getByText('60%')).toBeInTheDocument();
    const complianceCard = screen.getByText('60%').closest('div');
    expect(complianceCard).toHaveClass('low-compliance');
  });

  it('应该显示趋势图标', () => {
    render(<SLAStatisticsCards statistics={mockStats} />);

    const trendIcon = document.querySelector('[data-trend]');
    expect(trendIcon).toBeInTheDocument();
    expect(trendIcon).toHaveAttribute('data-trend', 'improving');
  });

  it('应该支持自定义标题', () => {
    render(
      <SLAStatisticsCards
        statistics={mockStats}
        customTitles={{
          totalViolations: 'All Violations',
          openViolations: 'Pending Actions',
        }}
      />
    );

    expect(screen.getByText('All Violations')).toBeInTheDocument();
    expect(screen.getByText('Pending Actions')).toBeInTheDocument();
  });

  it('应该处理空数据', () => {
    const emptyStats: SLAStatistics = {
      totalViolations: 0,
      openViolations: 0,
      criticalViolations: 0,
      complianceRate: 0,
      avgResolutionTime: 0,
      trend: 'stable',
      totalPolicies: 0,
      resolvedLast24h: 0,
    };

    render(<SLAStatisticsCards statistics={emptyStats} />);

    expect(screen.getByText('0')).toBeInTheDocument();
  });

  it('应该显示趋势方向箭头', () => {
    render(<SLAStatisticsCards statistics={mockStats} />);

    const improvingArrow = document.querySelector('[data-trend="improving"]');
    expect(improvingArrow).toBeInTheDocument();
  });

  it('应该格式化时间显示', () => {
    const statsWithTime = {
      ...mockStats,
      avgResolutionTime: 125,
    };

    render(<SLAStatisticsCards statistics={statsWithTime} />);

    // 应格式化 125 分钟为 2h 5m 或类似
    expect(screen.getByText(/2.*h|125 min/i)).toBeInTheDocument();
  });

  it('应该显示卡片阴影和边框', () => {
    render(<SLAStatisticsCards statistics={mockStats} />);

    const cards = document.querySelectorAll('.stat-card');
    expect(cards.length).toBeGreaterThan(0);
    cards.forEach(card => {
      expect(card).toHaveClass('shadow');
    });
  });

  it('应该支持点击事件', async () => {
    const user = userEvent.setup();
    const mockOnClick = jest.fn();

    render(
      <SLAStatisticsCards
        statistics={mockStats}
        onCardClick={mockOnClick}
      />
    );

    const totalCard = screen.getByText('25').closest('div');
    await user.click(totalCard!);

    expect(mockOnClick).toHaveBeenCalledWith('totalViolations');
  });

  it('应该显示最后更新时间', () => {
    const statsWithTime = {
      ...mockStats,
      updatedAt: '2024-01-20T10:30:00Z',
    };

    render(<SLAStatisticsCards statistics={statsWithTime} />);

    expect(screen.getByText(/2024|Jan|20/)).toBeInTheDocument();
  });

  it('应该支持紧凑布局', () => {
    render(
      <SLAStatisticsCards statistics={mockStats} compact={true} />
    );

    const container = screen.getByText('25').closest('div')?.parentElement;
    expect(container).toHaveClass('compact');
  });

  it('应该显示加载状态', () => {
    render(<SLAStatisticsCards statistics={mockStats} loading={true} />);

    const spinners = document.querySelectorAll('.spinner');
    expect(spinners.length).toBeGreaterThan(0);
  });

  it('应该正确计算百分比', () => {
    render(<SLAStatisticsCards statistics={mockStats} />);

    // 87.5% 应该显示
    expect(screen.getByText('87.5%')).toBeInTheDocument();
  });

  it('应该显示工具提示', () => {
    render(<SLAStatisticsCards statistics={mockStats} showTooltips={true} />);

    const helpIcons = document.querySelectorAll('[data-tooltip]');
    expect(helpIcons.length).toBeGreaterThan(0);
  });

  it('应该支持颜色方案配置', () => {
    render(
      <SLAStatisticsCards
        statistics={mockStats}
        colorScheme="dark"
      />
    );

    const container = document.querySelector('.statistics-container');
    expect(container).toHaveClass('dark-scheme');
  });

  it('应该显示比较数据（如果有）', () => {
    const withComparison = {
      ...mockStats,
      previousPeriod: {
        totalViolations: 20,
        complianceRate: 90,
      },
    };

    render(<SLAStatisticsCards statistics={withComparison} />);

    expect(screen.getByText(/\+5|25%/i)).toBeInTheDocument();
  });
});
