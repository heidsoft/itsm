/**
 * CIImpactAnalysisTab - CI 影响分析组件测试
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import { CIImpactAnalysisTab } from '../sections/CIImpactAnalysisTab';
import type { CIImpactAnalysisTabProps } from '../types';

jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

describe('CIImpactAnalysisTab', () => {
  const mockImpactAnalysis = {
    ciId: 'ci-001',
    criticality: 'high',
    businessServices: [
      {
        id: 'svc-1',
        name: 'Customer Portal',
        impact: 'critical',
        status: 'affected',
      },
      {
        id: 'svc-2',
        name: 'Internal Dashboard',
        impact: 'medium',
        status: 'monitoring',
      },
    ],
    downstreamCis: [
      {
        id: 'ci-002',
        name: 'Database Server',
        type: 'database',
        impact: 'high',
      },
      {
        id: 'ci-003',
        name: 'Cache Server',
        type: 'cache',
        impact: 'low',
      },
    ],
    upstreamCis: [
      {
        id: 'ci-004',
        name: 'Load Balancer',
        type: 'network',
        impact: 'medium',
      },
    ],
    riskScore: 85,
    recommendations: [
      'Implement monitoring alerts',
      'Create backup procedures',
      'Document recovery steps',
    ],
  };

  const defaultProps: CIImpactAnalysisTabProps = {
    impactAnalysis: mockImpactAnalysis,
    impactLoading: false,
    onAnalyze: jest.fn(),
  };

  it('应该显示 CI 关键性', () => {
    render(<CIImpactAnalysisTab {...defaultProps} />);

    expect(screen.getByText('high')).toBeInTheDocument();
  });

  it('应该显示风险评分', () => {
    render(<CIImpactAnalysisTab {...defaultProps} />);

    expect(screen.getByText('85')).toBeInTheDocument();
  });

  it('应该显示受影响的服务', () => {
    render(<CIImpactAnalysisTab {...defaultProps} />);

    expect(screen.getByText('Customer Portal')).toBeInTheDocument();
    expect(screen.getByText('Internal Dashboard')).toBeInTheDocument();
  });

  it('应该显示下游 CI', () => {
    render(<CIImpactAnalysisTab {...defaultProps} />);

    expect(screen.getByText('Database Server')).toBeInTheDocument();
    expect(screen.getByText('Cache Server')).toBeInTheDocument();
  });

  it('应该显示上游 CI', () => {
    render(<CIImpactAnalysisTab {...defaultProps} />);

    expect(screen.getByText('Load Balancer')).toBeInTheDocument();
  });

  it('应该显示建议项', () => {
    render(<CIImpactAnalysisTab {...defaultProps} />);

    expect(screen.getByText('Implement monitoring alerts')).toBeInTheDocument();
    expect(screen.getByText('Create backup procedures')).toBeInTheDocument();
  });

  it('应该显示影响级别', () => {
    render(<CIImpactAnalysisTab {...defaultProps} />);

    expect(screen.getByText('critical')).toBeInTheDocument();
    expect(screen.getByText('medium')).toBeInTheDocument();
    expect(screen.getByText('low')).toBeInTheDocument();
  });

  it('加载状态应显示加载指示器', () => {
    render(<CIImpactAnalysisTab {...defaultProps} loading={true} />);

    expect(screen.getByText(/analyzing/i)).toBeInTheDocument();
  });

  it('没有数据应显示空状态', () => {
    render(<CIImpactAnalysisTab {...defaultProps} impactAnalysis={null} />);

    expect(screen.getByText(/no impact analysis/i)).toBeInTheDocument();
  });

  it('应该显示分析时间戳', () => {
    const withTimestamp = {
      ...mockImpactAnalysis,
      analyzedAt: '2024-01-20T10:00:00Z',
    };

    render(<CIImpactAnalysisTab impactAnalysis={withTimestamp} loading={false} />);

    expect(screen.getByText(/2024|Jan|20/)).toBeInTheDocument();
  });

  it('应该显示影响拓扑图', () => {
    render(<CIImpactAnalysisTab {...defaultProps} />);

    expect(screen.getByTestId('impact-topology')).toBeInTheDocument();
  });

  it('应该显示 CI 类型', () => {
    render(<CIImpactAnalysisTab {...defaultProps} />);

    expect(screen.getByText('database')).toBeInTheDocument();
    expect(screen.getByText('cache')).toBeInTheDocument();
    expect(screen.getByText('network')).toBeInTheDocument();
  });

  it('应该显示服务状态', () => {
    render(<CIImpactAnalysisTab {...defaultProps} />);

    expect(screen.getByText('affected')).toBeInTheDocument();
    expect(screen.getByText('monitoring')).toBeInTheDocument();
  });

  it('应该显示风险等级指示器', () => {
    render(<CIImpactAnalysisTab {...defaultProps} />);

    const riskIndicator = screen.getByText('85').closest('div');
    expect(riskIndicator).toHaveClass('risk-score');
  });

  it('高关键性应显示警告', () => {
    render(<CIImpactAnalysisTab {...defaultProps} />);

    expect(screen.getByText(/critical/i)).toBeInTheDocument();
    const warning = document.querySelector('.warning-badge');
    expect(warning).toBeInTheDocument();
  });

  it('应该显示下游影响数量', () => {
    render(<CIImpactAnalysisTab {...defaultProps} />);

    expect(screen.getByText(/2\s+downstream/)).toBeInTheDocument();
  });

  it('应该显示上游影响数量', () => {
    render(<CIImpactAnalysisTab {...defaultProps} />);

    expect(screen.getByText(/1\s+upstream/)).toBeInTheDocument();
  });

  it('应该显示服务影响数量', () => {
    render(<CIImpactAnalysisTab {...defaultProps} />);

    expect(screen.getByText(/2\s+business services/)).toBeInTheDocument();
  });

  it('应该支持展开详情', async () => {
    const user = userEvent.setup();
    render(<CIImpactAnalysisTab {...defaultProps} />);

    const expandButton = screen.getByRole('button', { name: /expand|details/i });
    await user.click(expandButton);

    expect(screen.getByText(/recommendations/i)).toBeInTheDocument();
  });

  it('应该生成影响报告', () => {
    render(<CIImpactAnalysisTab {...defaultProps} />);

    const reportButton = screen.getByRole('button', { name: /执行影响分析/i });
    expect(reportButton).toBeInTheDocument();
  });

  it('应该重新计算影响', () => {
    render(<CIImpactAnalysisTab {...defaultProps} />);

    const recalcButton = screen.getByRole('button', { name: /执行影响分析/i });
    expect(recalcButton).toBeInTheDocument();
  });
});
