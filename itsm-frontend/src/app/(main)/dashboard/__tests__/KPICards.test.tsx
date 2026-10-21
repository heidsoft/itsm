import { render, screen } from '@testing-library/react';
import React from 'react';
import { KPICards } from '../components/KPICards';
import type { KPIMetric } from '../types/dashboard.types';

const baseMetric: KPIMetric = {
  id: 'total-tickets',
  title: '总工单数',
  value: 5,
  unit: '个',
  color: '#3b82f6',
  trend: 'stable',
  change: 0,
  changeType: 'stable',
  description: '范围内全部未删除工单',
};

describe('KPICards 环比展示', () => {
  it('comparable=false 时不渲染"相比上期"，避免显示不存在的趋势', () => {
    render(<KPICards metrics={[{ ...baseMetric, comparable: false }]} />);
    expect(screen.queryByText(/相比上期/)).not.toBeInTheDocument();
    expect(screen.getByText('5')).toBeInTheDocument();
  });

  it('comparable 缺失按不可比处理（未知契约不得当成 0% 平稳）', () => {
    render(<KPICards metrics={[baseMetric]} />);
    expect(screen.queryByText(/相比上期/)).not.toBeInTheDocument();
  });

  it('comparable=true 时按后端数值渲染环比', () => {
    render(
      <KPICards
        metrics={[{ ...baseMetric, comparable: true, change: 12, trend: 'up', changeType: 'increase' }]}
      />
    );
    expect(screen.getByText(/相比上期/)).toBeInTheDocument();
    expect(screen.getAllByText(/12\.0/).length).toBeGreaterThan(0);
  });
});
