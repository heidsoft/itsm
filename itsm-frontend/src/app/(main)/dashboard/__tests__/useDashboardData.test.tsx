import { renderHook, waitFor } from '@testing-library/react';
import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useDashboardData } from '../hooks/useDashboardData';
import { DashboardAPI } from '@/lib/api/dashboard-api';
import { ticketService } from '@/lib/services/ticket-service';

jest.mock('@/lib/api/dashboard-api', () => ({
  DashboardAPI: { getOverview: jest.fn() },
}));

jest.mock('@/lib/services/ticket-service', () => ({
  ticketService: { getTicketStats: jest.fn() },
}));

const mockOverview = DashboardAPI as jest.Mocked<typeof DashboardAPI>;
const mockStats = ticketService as jest.Mocked<typeof ticketService>;

function createWrapper() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const Wrapper = ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  return { Wrapper, queryClient };
}

const overviewPayload = {
  kpiMetrics: [
    {
      id: 'pending-tickets',
      title: '待处理工单',
      value: 7,
      unit: '个',
      color: '#f59e0b',
      trend: 'stable' as const,
      change: 0,
      changeType: 'stable' as const,
      comparable: false,
      scope: 'own' as const,
      description: '当前待处理存量',
    },
  ],
  ticketTrend: [],
  incidentDistribution: [],
  slaData: [],
  satisfactionData: [],
  quickActions: [],
  recentActivities: [],
};

beforeEach(() => {
  jest.clearAllMocks();
  mockOverview.getOverview.mockResolvedValue(overviewPayload as never);
  mockStats.getTicketStats.mockResolvedValue({ total: 999, open: 999 } as never);
});

describe('useDashboardData KPI 契约', () => {
  it('卡片数值只取后端 overview，不再被 /tickets/stats 覆盖', async () => {
    const { Wrapper } = createWrapper();
    const { result } = renderHook(() => useDashboardData(), { wrapper: Wrapper });

    await waitFor(() => expect(result.current.loading).toBe(false));

    const metric = result.current.data?.kpiMetrics[0];
    expect(metric?.value).toBe(7);
    expect(metric?.comparable).toBe(false);
    expect(metric?.scope).toBe('own');
    // 覆盖率口径由后端唯一持有：坐席看到的工作台数字必须与下钻列表同源
    expect(mockStats.getTicketStats).not.toHaveBeenCalled();
  });

  it('overview 失败必须进入错误态，不得伪装成 0 数据已连接', async () => {
    mockOverview.getOverview.mockRejectedValue(new Error('boom'));
    const { Wrapper } = createWrapper();
    const { result } = renderHook(() => useDashboardData(), { wrapper: Wrapper });

    // hook 内部固定 retry:1 / retryDelay:1000，等待窗口需覆盖一次重试
    await waitFor(() => expect(result.current.isConnected).toBe(false), { timeout: 3000 });
    expect(result.current.error).toContain('boom');
  });
});
