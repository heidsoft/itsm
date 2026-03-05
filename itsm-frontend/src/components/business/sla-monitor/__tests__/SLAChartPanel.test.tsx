/**
 * SLAChartPanel - SLA 违规图表面板组件测试
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import { SLAChartPanel } from '../components/SLAChartPanel';
import type { SLAViolation } from '../types';

describe('SLAChartPanel', () => {
  const mockViolations: SLAViolation[] = [
    {
      id: 'v1',
      severity: 'critical',
      status: 'open',
      slaMetric: 'response_time',
    },
    {
      id: 'v2',
      severity: 'high',
      status: 'acknowledged',
      slaMetric: 'resolution_time',
    },
    {
      id: 'v3',
      severity: 'critical',
      status: 'open',
      slaMetric: 'response_time',
    },
    {
      id: 'v4',
      severity: 'medium',
      status: 'resolved',
      slaMetric: 'response_time',
    },
  ];

  it('应该渲染图表面板', () => {
    render(<SLAChartPanel violations={mockViolations} />);

    expect(screen.getByText(/趋势分析|distribution/i)).toBeInTheDocument();
  });

  it('应该显示严重程度分布', () => {
    render(<SLAChartPanel violations={mockViolations} />);

    expect(screen.getByText('critical')).toBeInTheDocument();
    expect(screen.getByText('high')).toBeInTheDocument();
    expect(screen.getByText('medium')).toBeInTheDocument();
  });

  it('应该显示每个严重程度的数量', () => {
    render(<SLAChartPanel violations={mockViolations} />);

    // critical 有 2 个
    expect(screen.getByText('2')).toBeInTheDocument();
  });

  it('应该显示状态分布', () => {
    render(<SLAChartPanel violations={mockViolations} />);

    expect(screen.getByText('open')).toBeInTheDocument();
    expect(screen.getByText('acknowledged')).toBeInTheDocument();
    expect(screen.getByText('resolved')).toBeInTheDocument();
  });

  it('应该显示时间趋势图表', () => {
    render(<SLAChartPanel violations={mockViolations} />);

    expect(screen.getByTestId('time-trend-chart') ||
           screen.getByText(/time series|trend/i)).toBeInTheDocument();
  });

  it('空数据应显示 Empty 状态', () => {
    render(<SLAChartPanel violations={[]} />);

    expect(screen.getByText(/暂无数据|no data/i)).toBeInTheDocument();
  });

  it('应该渲染进度条', () => {
    render(<SLAChartPanel violations={mockViolations} />);

    const progressBars = document.querySelectorAll('[role="progressbar"]');
    expect(progressBars.length).toBeGreaterThan(0);
  });

  it('应该正确计算百分比', () => {
    render(<SLAChartPanel violations={mockViolations} />);

    // critical 占 50% (2/4)
    const criticalLabel = screen.getByText('critical');
    const progressBar = criticalLabel.closest('div')?.querySelector('.bg-blue-500');
    expect(progressBar).toHaveStyle({ width: '50%' });
  });

  it('应该显示图表标题', () => {
    render(<SLAChartPanel violations={mockViolations} />);

    expect(screen.getByText('严重程度分布')).toBeInTheDocument();
    expect(screen.getByText('状态分布')).toBeInTheDocument();
  });

  it('应该使用不同颜色区分严重程度', () => {
    render(<SLAChartPanel violations={mockViolations} />);

    const criticalBars = document.querySelectorAll('[data-severity="critical"]');
    criticalBars.forEach(bar => {
      expect(bar).toHaveClass('bg-red-500', 'bg-orange-500'); // 严重颜色
    });
  });

  it('应该显示 SLO 达标率', () => {
    render(<SLAChartPanel violations={mockViolations} />);

    expect(screen.getByText(/compliance|达标/i)).toBeInTheDocument();
  });

  it('应该支持时间粒度切换', () => {
    render(<SLAChartPanel violations={mockViolations} />);

    const timeSelector = screen.getByLabelText(/time grain|granularity/i);
    expect(timeSelector).toBeInTheDocument();
  });

  it('应该点击图例显示/隐藏系列', async () => {
    const user = userEvent.setup();

    render(<SLAChartPanel violations={mockViolations} />);

    const legendItems = screen.getAllByText(/critical|high|medium/i);
    await user.click(legendItems[0]);

    // 点击后对应系列应该隐藏
    expect(legendItems[0].closest('button')).toHaveClass('inactive');
  });

  it('应该显示工具提示', async () => {
    const user = userEvent.setup();

    render(<SLAChartPanel violations={mockViolations} />);

    const chartArea = screen.getByText('严重程度分布').closest('div')?.nextElementSibling;
    await user.hover(chartArea!);

    const tooltip = await screen.findByRole('tooltip');
    expect(tooltip).toBeInTheDocument();
  });

  it('应该显示总计数量', () => {
    render(<SLAChartPanel violations={mockViolations} />);

    expect(screen.getByText('4')).toBeInTheDocument(); // 总计
  });

  it('应该支持导出图表', () => {
    render(<SLAChartPanel violations={mockViolations} />);

    const exportButton = screen.getByRole('button', { name: /export/i });
    expect(exportButton).toBeInTheDocument();
  });

  it('应该刷新图表', () => {
    const { rerender } = render(<SLAChartPanel violations={mockViolations} />);

    // 用新数据重新渲染
    const newViolations = [...mockViolations, {
      id: 'v5',
      severity: 'low',
      status: 'open',
      slaMetric: 'response_time',
    }];

    rerender(<SLAChartPanel violations={newViolations} />);

    expect(screen.getByText('low')).toBeInTheDocument();
  });

  it('应该显示数据更新指示器', () => {
    render(<SLAChartPanel violations={mockViolations} />);

    const updateIndicator = document.querySelector('[data-last-updated]');
    expect(updateIndicator).toBeInTheDocument();
  });

  it('应该支持响应式布局', () => {
    render(<SLAChartPanel violations={mockViolations} />);

    const container = document.querySelector('.chart-container');
    expect(container).toHaveClass('responsive');
  });

  it('应该显示缺失数据提示', () => {
    const incompleteViolations = mockViolations.map(v => ({
      ...v,
      severity: undefined as any,
    }));

    render(<SLAChartPanel violations={incompleteViolations} />);

    expect(screen.getByText(/missing data/i)).toBeInTheDocument();
  });

  it('应该显示图表说明', () => {
    render(<SLAChartPanel violations={mockViolations} />);

    expect(screen.getByText(/chart description|description/i)).toBeInTheDocument();
  });

  it('应该支持图表类型切换', () => {
    render(<SLAChartPanel violations={mockViolations} />);

    const chartTypeSelector = screen.getByLabelText(/chart type/i);
    expect(chartTypeSelector).toBeInTheDocument();
  });

  it('应该处理大数据量', () => {
    const manyViolations = Array.from({ length: 1000 }, (_, i) => ({
      id: `v${i}`,
      severity: ['critical', 'high', 'medium', 'low'][i % 4],
      status: ['open', 'acknowledged', 'resolved'][i % 3],
      slaMetric: 'response_time',
    }));

    render(<SLAChartPanel violations={manyViolations} />);

    expect(screen.getByText('1000')).toBeInTheDocument();
  });

  it('should display loading state', () => {
    const { container } = render(<SLAChartPanel violations={[]} loading={true} />);

    const loadingSpinner = container.querySelector('.ant-spin');
    expect(loadingSpinner).toBeInTheDocument();
  });
});
