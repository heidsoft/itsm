import { render, screen, waitFor } from '@testing-library/react';
import { DashboardAPI } from '@/lib/api/dashboard-api';
import type { SystemStats } from '@/types/dashboard';
import { SystemHealth } from '../SystemHealth';

jest.mock('@/lib/api/dashboard-api', () => ({
  DashboardAPI: {
    getSystemStats: jest.fn(),
  },
}));

const mockedApi = DashboardAPI as jest.Mocked<typeof DashboardAPI>;

function statsFixture(overrides: Partial<SystemStats> = {}): SystemStats {
  return {
    uptime: 3600,
    cpuUsage: 12,
    memoryUsage: 30,
    diskUsage: 40,
    avgResponseTime: 25,
    requestsPerSecond: 4.2,
    errorRate: 0.5,
    metricsWindowSeconds: 30,
    dbConnections: 8,
    dbSize: 500 * 1024 * 1024,
    cacheHitRate: 95,
    cores: 8,
    goroutines: 42,
    sampledAt: '2026-10-14T02:00:00.000Z',
    availability: {
      cpu: 'measured',
      memory: 'measured',
      disk: 'measured',
      http: 'measured',
      dbSize: 'measured',
      cache: 'measured',
    },
    ...overrides,
  };
}

describe('SystemHealth', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('按后端可用性标注展示实测值，不再显示假"实时"标签', async () => {
    mockedApi.getSystemStats.mockResolvedValue(statsFixture());

    render(<SystemHealth />);

    await waitFor(() => expect(mockedApi.getSystemStats).toHaveBeenCalledTimes(1));

    expect(screen.getByText('95.0%')).toBeInTheDocument();
    expect(screen.getByText('4.2')).toBeInTheDocument();
    expect(screen.queryByText('实时')).not.toBeInTheDocument();
  });

  it('未接入的指标显示"未接入"，且不参与整体健康评级', async () => {
    // cacheHitRate 默认值是 0：若按数值评级会误报「命中率 0% ⇒ 严重」。
    mockedApi.getSystemStats.mockResolvedValue(
      statsFixture({
        cacheHitRate: 0,
        availability: {
          cpu: 'measured',
          memory: 'measured',
          disk: 'measured',
          http: 'measured',
          dbSize: 'measured',
          cache: 'unconfigured',
        },
      }),
    );

    render(<SystemHealth />);

    await waitFor(() => expect(screen.getAllByText('未接入').length).toBeGreaterThan(0));
    expect(screen.queryByText('0.0%')).not.toBeInTheDocument();
    expect(screen.queryByText('严重')).not.toBeInTheDocument();
    expect(screen.getAllByText('优秀').length).toBeGreaterThan(0);
  });

  it('HTTP 指标未采集到时不出错误率与吞吐读数', async () => {
    mockedApi.getSystemStats.mockResolvedValue(
      statsFixture({
        errorRate: 0,
        requestsPerSecond: 0,
        availability: {
          cpu: 'measured',
          memory: 'measured',
          disk: 'measured',
          http: 'unavailable',
          dbSize: 'measured',
          cache: 'measured',
        },
      }),
    );

    render(<SystemHealth />);

    await waitFor(() => expect(screen.getAllByText('未接入').length).toBeGreaterThan(1));
    // 未采集时不能把默认 0 渲染成「0.00% 错误率 / 0.0 吞吐」。
    expect(screen.queryByText('0.00%')).not.toBeInTheDocument();
    expect(screen.queryByText('0.0')).not.toBeInTheDocument();
  });

  it('亚秒级采样区间用毫秒显示，不显示成 0s', async () => {
    mockedApi.getSystemStats.mockResolvedValue(statsFixture({ metricsWindowSeconds: 0.4 }));

    render(<SystemHealth />);

    await waitFor(() => expect(screen.getByText('采样区间 400ms')).toBeInTheDocument());
    expect(screen.queryByText('采样区间 0s')).not.toBeInTheDocument();
  });

  it('最后更新时间取服务端采样时间而不是浏览器当前时间', async () => {
    mockedApi.getSystemStats.mockResolvedValue(statsFixture());

    render(<SystemHealth />);

    const expected = new Date('2026-10-14T02:00:00.000Z').toLocaleString('zh-CN');
    await waitFor(() =>
      expect(screen.getByText(`最后更新: ${expected}`)).toBeInTheDocument(),
    );
  });
});
