import { render, screen, waitFor } from '@/lib/test-utils';
import userEvent from '@testing-library/user-event';
import ServiceQualityOverview from '../ServiceQualityOverview';
import { DashboardAPI } from '@/lib/api/dashboard-api';
import type { ServiceQualityReport } from '@/app/(main)/dashboard/types/dashboard.types';

jest.mock('@/lib/api/dashboard-api', () => ({
  DashboardAPI: {
    getServiceQuality: jest.fn(),
  },
}));

const mockGetServiceQuality = DashboardAPI.getServiceQuality as jest.Mock;

function buildReport(
  overrides: Partial<ServiceQualityReport> = {}
): ServiceQualityReport {
  return {
    metricVersion: '1.1',
    metricVersions: { kpi: '1.0', problemStats: '1.0', changeStats: '1.0' },
    generatedAt: '2026-11-07T08:00:00Z',
    window: { days: 30, from: '2026-10-08T08:00:00Z', to: '2026-11-07T08:00:00Z' },
    scope: 'tenant',
    kpi: [
      { id: 'total-tickets', title: '未关闭工单', value: 120, unit: '张', color: '', trend: 'up', change: 8 },
    ],
    sla: {
      complianceRate: 95.5,
      responseTimeCompliance: 97,
      resolutionTimeCompliance: 94,
      atRiskTickets: 2,
    } as ServiceQualityReport['sla'],
    incidentDistribution: [{ category: 'network', count: 3, color: '' }],
    satisfaction: [{ month: '2026-10', rating: 4.5, responses: 12 }],
    ticketTrend: [{ date: '2026-11-01', open: 2, inProgress: 0, resolved: 1, closed: 0 }],
    ticketStats: {
      total: 121,
      statusGroups: [{ status: 'open', count: 12 }],
      priorityGroups: [{ priority: 'high', count: 5 }],
      trend30d: [{ date: '2026-11-01', count: 3 }],
      generatedAt: '2026-11-07T08:00:00Z',
    },
    problemStats: {
      total: 8,
      open: 3,
      inProgress: 1,
      resolved: 2,
      closed: 2,
      highPriority: 1,
      byStatus: [{ status: 'open', count: 3 }],
      byPriority: [],
    },
    changeStats: {
      total: 5,
      draft: 1,
      pending: 1,
      approved: 0,
      scheduled: 0,
      inProgress: 1,
      completed: 2,
      failed: 0,
      rolledBack: 0,
      rejected: 0,
      cancelled: 0,
      closed: 0,
      byType: [{ type: 'normal', count: 5 }],
    },
    ...overrides,
  };
}

describe('ServiceQualityOverview', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('首次拉取未完成时显示加载骨架，不渲染空数字', async () => {
    mockGetServiceQuality.mockReturnValue(new Promise(() => {}));
    render(<ServiceQualityOverview />);

    await waitFor(() => {
      expect(screen.getByText('Service Quality Overview')).toBeInTheDocument();
    });
    expect(screen.queryByText('Total Tickets')).not.toBeInTheDocument();
    expect(mockGetServiceQuality).toHaveBeenCalledWith(30);
  });

  it('渲染 KPI、口径标签与问题/变更统计块', async () => {
    mockGetServiceQuality.mockResolvedValue(buildReport());
    render(<ServiceQualityOverview />);

    expect(await screen.findByText('未关闭工单')).toBeInTheDocument();
    expect(screen.getByText('120')).toBeInTheDocument();
    expect(screen.getByText('All tenant records')).toBeInTheDocument();
    expect(screen.getByText('Problem Statistics')).toBeInTheDocument();
    expect(screen.getByText('Change Statistics')).toBeInTheDocument();
  });

  it('块缺省时如实标注未接入，不用全零冒充统计结果', async () => {
    mockGetServiceQuality.mockResolvedValue(
      buildReport({ problemStats: undefined, changeStats: undefined, ticketStats: undefined })
    );
    render(<ServiceQualityOverview />);

    await waitFor(() => {
      expect(screen.getAllByText('This statistics block is not connected')).toHaveLength(3);
    });
    // 缺省块不渲染任何统计数值
    expect(screen.queryByText('Total Problems')).not.toBeInTheDocument();
    expect(screen.queryByText('Total Changes')).not.toBeInTheDocument();
  });

  it('加载失败时显示错误与重试入口', async () => {
    mockGetServiceQuality.mockRejectedValue(new Error('service unavailable'));
    render(<ServiceQualityOverview />);

    expect(await screen.findByText('Failed to load service quality overview')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Retry/ })).toBeInTheDocument();
  });

  it('切换窗口后按新天数重新请求', async () => {
    mockGetServiceQuality.mockResolvedValue(buildReport());
    render(<ServiceQualityOverview />);

    await screen.findByText('未关闭工单');
    await userEvent.click(screen.getByText('Last 60 Days'));
    await waitFor(() => {
      expect(mockGetServiceQuality).toHaveBeenLastCalledWith(60);
    });
  });
});
