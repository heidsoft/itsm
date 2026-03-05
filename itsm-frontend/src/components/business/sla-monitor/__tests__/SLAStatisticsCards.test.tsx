/**
 * SLAStatisticsCards - SLA 统计卡片组件测试
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import { SLAStatisticsCards } from '../components/SLAStatisticsCards';
import type { SLAStats } from '../types';

jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

describe('SLAStatisticsCards', () => {
  const mockStats: SLAStats = {
    total: 25,
    open: 8,
    resolved: 17,
    critical: 5,
  };

  it('应该渲染组件', () => {
    render(<SLAStatisticsCards stats={mockStats} />);
    expect(screen.getByText('总违规数')).toBeInTheDocument();
    expect(screen.getByText('待处理')).toBeInTheDocument();
    expect(screen.getByText('已解决')).toBeInTheDocument();
  });

  it('应该显示统计数据', () => {
    render(<SLAStatisticsCards stats={mockStats} />);

    expect(screen.getByText('25')).toBeInTheDocument();
    expect(screen.getByText('8')).toBeInTheDocument();
    expect(screen.getByText('17')).toBeInTheDocument();
  });

  it('应该显示严重统计', () => {
    render(<SLAStatisticsCards stats={mockStats} />);

    expect(screen.getByText('严重违规')).toBeInTheDocument();
    expect(screen.getByText('5')).toBeInTheDocument();
  });

  it('应该显示加载状态', () => {
    render(<SLAStatisticsCards stats={mockStats} loading={true} />);

    const spinElements = document.querySelectorAll('.ant-spin');
    expect(spinElements.length).toBeGreaterThan(0);
  });
});
