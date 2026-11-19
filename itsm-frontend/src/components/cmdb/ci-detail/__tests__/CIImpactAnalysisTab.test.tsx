/**
 * CIImpactAnalysisTab - CI 影响分析标签页组件测试
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CIImpactAnalysisTab } from '../sections/CIImpactAnalysisTab';
import type { ImpactAnalysisData } from '../types';

import { translations } from '@/lib/i18n/translations';

// 用真实 zh-CN 词表 mock t：`key => key` 会让枚举反查拿到 i18n key，
// 测试就锁不住「风险等级/优先级直出 high/medium」这个历史缺陷。
const zhCN = translations['zh-CN'] as unknown as Record<string, unknown>;
const resolve = (key: string): unknown => {
  let value: unknown = zhCN;
  for (const part of key.split('.')) {
    if (value && typeof value === 'object' && part in value) {
      value = (value as Record<string, unknown>)[part];
    } else {
      return undefined;
    }
  }
  return value;
};

jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => {
      const value = resolve(key);
      return typeof value === 'string' ? value : key;
    },
  }),
}));

describe('CIImpactAnalysisTab', () => {
  const mockImpactAnalysis: ImpactAnalysisData = {
    targetCi: {
      id: 'ci-001',
      name: 'Web Server 1',
      type: 'server',
    },
    upstreamImpact: [
      {
        ciId: 2,
        ciName: 'Load Balancer',
        ciType: 'network',
        relationship: 'depends_on',
        impactLevel: 'high',
        distance: 1,
        direction: 'upstream',
      },
    ],
    downstreamImpact: [
      {
        ciId: 3,
        ciName: 'Application Service',
        ciType: 'service',
        relationship: 'contains',
        impactLevel: 'medium',
        distance: 1,
        direction: 'downstream',
      },
    ],
    criticalDependencies: [
      {
        ciId: 4,
        ciName: 'Database Server',
        ciType: 'database',
        relationship: 'depends_on',
        impactLevel: 'critical',
        distance: 1,
        direction: 'upstream',
      },
    ],
    affectedTickets: [
      {
        id: 1,
        ticketNumber: 'TICKET-001',
        title: 'Database slow response',
        status: 'open',
        priority: 'high',
      },
    ],
    affectedIncidents: [
      {
        id: 1,
        title: 'Service degradation',
        status: 'investigating',
        severity: 'high',
      },
    ],
    riskLevel: 'high',
    summary: 'High risk due to critical dependencies',
  };

  const defaultProps = {
    impactAnalysis: mockImpactAnalysis,
    impactLoading: false,
    onAnalyze: jest.fn(),
  };

  it('应该把风险等级与优先级翻译成中文，而不是直出后端原始值', () => {
    render(<CIImpactAnalysisTab {...defaultProps} />);

    // 历史缺陷：头部与列表项都直出 high/medium/critical。
    // 风险等级走 RISK_LEVEL_LABELS（high→高），优先级走 criticality 词表。
    // 头部风险等级 + 列表项 impactLevel 同为 high，故出现多处「高」。
    expect(screen.getAllByText('高').length).toBeGreaterThan(0);
    expect(screen.queryByText('high')).not.toBeInTheDocument();
    expect(screen.queryByText('medium')).not.toBeInTheDocument();
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
    render(<CIImpactAnalysisTab {...defaultProps} impactLoading={true} />);

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
