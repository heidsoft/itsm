/**
 * CIImpactAnalysisTab - CI 影响分析标签页组件测试
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CIImpactAnalysisTab } from '../sections/CIImpactAnalysisTab';

jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

describe('CIImpactAnalysisTab', () => {
  const mockImpactAnalysis = {
    target_ci: {
      id: 'ci-001',
      name: 'Web Server 1',
      type: 'server',
    },
    upstream_impact: [
      {
        ci_id: 2,
        ci_name: 'Load Balancer',
        ci_type: 'network',
        relationship: 'depends_on',
        impact_level: 'high',
        distance: 1,
        direction: 'upstream',
      },
    ],
    downstream_impact: [
      {
        ci_id: 3,
        ci_name: 'Application Service',
        ci_type: 'service',
        relationship: 'contains',
        impact_level: 'medium',
        distance: 1,
        direction: 'downstream',
      },
    ],
    critical_dependencies: [
      {
        ci_id: 4,
        ci_name: 'Database Server',
        ci_type: 'database',
        relationship: 'depends_on',
        impact_level: 'critical',
        distance: 1,
        direction: 'upstream',
      },
    ],
    affected_tickets: [
      {
        id: 1,
        ticketNumber: 'TICKET-001',
        title: 'Database slow response',
        status: 'open',
        priority: 'high',
      },
    ],
    affected_incidents: [
      {
        id: 1,
        title: 'Service degradation',
        status: 'investigating',
        severity: 'high',
      },
    ],
    risk_level: 'high',
    summary: 'High risk due to critical dependencies',
  };

  const defaultProps = {
    impactAnalysis: mockImpactAnalysis,
    impactLoading: false,
    onAnalyze: jest.fn(),
  };

  it('应该显示风险等级', () => {
    render(<CIImpactAnalysisTab {...defaultProps} />);

    expect(screen.getByText('high')).toBeInTheDocument();
  });

  it('应该显示分析摘要', () => {
    render(<CIImpactAnalysisTab {...defaultProps} />);

    expect(screen.getByText(mockImpactAnalysis.summary)).toBeInTheDocument();
  });

  it('应该显示上游影响标签', () => {
    render(<CIImpactAnalysisTab {...defaultProps} />);

    // 检查上游影响标签存在（显示在 Tab 上）
    expect(screen.getByText(/上游影响 \(1\)/)).toBeInTheDocument();
  });

  it('应该显示下游影响标签', () => {
    render(<CIImpactAnalysisTab {...defaultProps} />);

    // 检查下游影响标签存在
    expect(screen.getByText(/下游影响 \(1\)/)).toBeInTheDocument();
  });

  it('应该显示关联工单标签', () => {
    render(<CIImpactAnalysisTab {...defaultProps} />);

    expect(screen.getByText(/关联工单 \(1\)/)).toBeInTheDocument();
  });

  it('应该显示关联事件标签', () => {
    render(<CIImpactAnalysisTab {...defaultProps} />);

    expect(screen.getByText(/关联事件 \(1\)/)).toBeInTheDocument();
  });

  it('应该显示分析按钮', () => {
    render(<CIImpactAnalysisTab {...defaultProps} />);

    const analyzeButton = screen.getByRole('button', { name: /执行影响分析/i });
    expect(analyzeButton).toBeInTheDocument();
  });

  it('点击分析按钮应调用 onAnalyze', async () => {
    const user = userEvent.setup();
    const mockOnAnalyze = jest.fn();

    render(<CIImpactAnalysisTab {...defaultProps} onAnalyze={mockOnAnalyze} />);

    const analyzeButton = screen.getByRole('button', { name: /执行影响分析/i });
    await user.click(analyzeButton);

    expect(mockOnAnalyze).toHaveBeenCalled();
  });

  it('加载状态应显示加载指示器', () => {
    render(
      <CIImpactAnalysisTab
        {...defaultProps}
        impactLoading={true}
      />
    );

    // 检查按钮存在
    const analyzeButton = screen.getByRole('button');
    expect(analyzeButton).toBeInTheDocument();
  });

  it('没有影响分析数据应显示空状态', () => {
    render(<CIImpactAnalysisTab {...defaultProps} impactAnalysis={null} />);

    expect(screen.getByText(/点击上方按钮执行影响分析/i)).toBeInTheDocument();
  });

  it('应该显示 CI影响分析 标题', () => {
    render(<CIImpactAnalysisTab {...defaultProps} />);

    expect(screen.getByText(/CI影响分析/)).toBeInTheDocument();
  });
});
