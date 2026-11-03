import { render, screen, waitFor } from '@/lib/test-utils';
import userEvent from '@testing-library/user-event';
import dayjs from 'dayjs';
import { ApiRequestError } from '@/lib/api/http-client';
import { TicketApi } from '@/lib/api/ticket-api';
import type { TicketStatsResponse } from '@/lib/api/ticket-api';
import TicketAnalyticsPage from '../page';

// jest.setup.js 全局 mock 的 dayjs 实例方法会被 rc-picker 链式调用，jsdom 下 RangePicker
// 直接抛 TypeError。这里按文件粒度换回真实 dayjs，本页断言的是加载语义而不是日期控件。
jest.mock('dayjs', () => jest.requireActual('dayjs'));

jest.mock('@/lib/api/ticket-api', () => ({
  __esModule: true,
  TicketApi: { getTicketStats: jest.fn() },
}));

// 深度分析 tab 自己会发请求（report:read），本页回归目标不覆盖它。
jest.mock('@/components/business/TicketDeepAnalytics', () => ({
  __esModule: true,
  default: () => null,
}));

jest.mock('recharts', () => {
  const React = require('react');
  const passthrough = ({ children }: { children?: React.ReactNode }) =>
    React.createElement(React.Fragment, null, children);
  return {
    __esModule: true,
    ResponsiveContainer: passthrough,
    LineChart: passthrough,
    BarChart: passthrough,
    PieChart: passthrough,
    Pie: () => null,
    Line: () => null,
    Bar: () => null,
    Cell: () => null,
    XAxis: () => null,
    YAxis: () => null,
    CartesianGrid: () => null,
    Tooltip: () => null,
    Legend: () => null,
  };
});

const mockGetTicketStats = TicketApi.getTicketStats as jest.Mock;

const WINDOW = { dateFrom: '2026-10-03', dateTo: '2026-11-01', days: 30 };

function statsFixture(
  overrides: Partial<TicketStatsResponse> = {}
): TicketStatsResponse {
  return {
    total: 22,
    open: 6,
    inProgress: 2,
    resolved: 11,
    closed: 3,
    pending: 1,
    highPriority: 4,
    overdue: 2,
    byStatus: [
      { status: 'open', count: 6 },
      { status: 'resolved', count: 11 },
      { status: 'closed', count: 5 },
    ],
    byPriority: [{ priority: 'high', count: 4 }],
    scope: 'all',
    window: WINDOW,
    dailyTrend: [{ date: '2026-11-01', created: 3, resolved: 1 }],
    sla: {
      complianceRate: 91.4,
      sampledTickets: 7,
      violatedTickets: 1,
      avgResponseMinutes: 42,
      avgResolutionHours: 8.5,
      windowDays: 30,
      scope: 'tenant',
    },
    ...overrides,
  };
}

describe('工单统计页加载语义', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  // 唯一数据源是 GET /api/v1/tickets/stats：整页只发这一次请求，
  // 不允许再拼 /analytics/tickets 之类的第二数据源补齐卡片。
  it('只取一次工单统计端点，并带上默认 30 天窗口', async () => {
    mockGetTicketStats.mockResolvedValue(statsFixture());
    render(<TicketAnalyticsPage />);

    expect(await screen.findByText('22')).toBeInTheDocument();
    expect(mockGetTicketStats).toHaveBeenCalledTimes(1);
    expect(mockGetTicketStats).toHaveBeenCalledWith({
      dateFrom: dayjs().subtract(29, 'day').format('YYYY-MM-DD'),
      dateTo: dayjs().format('YYYY-MM-DD'),
    });
  });

  it('显示后端回显的数据范围与时间窗口', async () => {
    mockGetTicketStats.mockResolvedValue(statsFixture({ scope: 'owner' }));
    render(<TicketAnalyticsPage />);

    expect(await screen.findByText(/本人创建或受理的工单/)).toBeInTheDocument();
    expect(screen.getByText(/2026-10-03 至 2026-11-01（30 天，按创建时间）/)).toBeInTheDocument();
  });

  it('SLA 可用时显示达标率与样本明细，并标注租户口径', async () => {
    mockGetTicketStats.mockResolvedValue(statsFixture());
    render(<TicketAnalyticsPage />);

    expect(await screen.findByText(/样本 7 张，违约 1 张/)).toBeInTheDocument();
    expect(screen.getByText(/91%/)).toBeInTheDocument();
    expect(screen.getByText(/SLA 读数按租户全量计算/)).toBeInTheDocument();
  });

  // sla 缺省是「没取到」，渲染成 0% 会让使用者以为全部超时。
  it('SLA 缺省时显示暂不可用而不是 0%', async () => {
    mockGetTicketStats.mockResolvedValue(statsFixture({ sla: undefined }));
    render(<TicketAnalyticsPage />);

    expect(await screen.findByText('SLA 达标率暂不可用')).toBeInTheDocument();
    expect(screen.queryByText(/样本 7 张/)).not.toBeInTheDocument();
    expect(screen.queryByText('0%')).not.toBeInTheDocument();
  });

  // 后端把「未取趋势」（缺省）和「窗口内没有工单」（空数组）区分开，前端必须分别呈现。
  it('趋势序列缺省时说明后端未启用窗口趋势', async () => {
    mockGetTicketStats.mockResolvedValue(statsFixture({ dailyTrend: undefined }));
    render(<TicketAnalyticsPage />);

    expect(
      await screen.findByText('本次请求未返回趋势序列（后端未启用窗口趋势）')
    ).toBeInTheDocument();
  });

  it('趋势序列为空数组时说明窗口内没有工单', async () => {
    mockGetTicketStats.mockResolvedValue(statsFixture({ dailyTrend: [] }));
    render(<TicketAnalyticsPage />);

    expect(await screen.findByText('所选时间范围内没有新建或解决的工单')).toBeInTheDocument();
  });

  // 与修复前的差别：403 不再渲染成一屏 0，而是明确缺哪个权限码。
  it('403 时不渲染任何统计卡并说明需要 ticket:read', async () => {
    mockGetTicketStats.mockRejectedValue(new ApiRequestError('权限不足', 403, 2003));
    render(<TicketAnalyticsPage />);

    expect(await screen.findByText('工单统计没有访问权限')).toBeInTheDocument();
    expect(screen.getByText(/需要 ticket:read/)).toBeInTheDocument();
    expect(screen.getByText(/缺少数据不等于租户里没有工单/)).toBeInTheDocument();
    expect(screen.queryByText('工单总数')).not.toBeInTheDocument();
    expect(screen.queryByText('待处理')).not.toBeInTheDocument();
  });

  it('整页失败重试成功后恢复数据', async () => {
    mockGetTicketStats.mockRejectedValue(new ApiRequestError('后端未就绪', 503, 5001));
    render(<TicketAnalyticsPage />);

    expect(await screen.findByText('工单统计加载失败')).toBeInTheDocument();
    expect(screen.queryByText('工单总数')).not.toBeInTheDocument();

    mockGetTicketStats.mockResolvedValue(statsFixture({ total: 22 }));
    await userEvent.click(screen.getByRole('button', { name: '重试' }));

    await waitFor(() => expect(mockGetTicketStats).toHaveBeenCalledTimes(2));
    expect(await screen.findByText('22')).toBeInTheDocument();
    expect(screen.queryByText('工单统计加载失败')).not.toBeInTheDocument();
  });

  // 后端未提供这些读数（dto.AnalyticsTicketStatsResponse 只有 total/statusGroups/
  // priorityGroups/trend30d），页面不再保留假控件；导出由深度分析区块自己的按钮承担。
  it('不再保留无数据源的假控件与页面级导出', async () => {
    mockGetTicketStats.mockResolvedValue(statsFixture());
    render(<TicketAnalyticsPage />);

    await screen.findByText('22');
    expect(screen.queryByText('团队表现')).not.toBeInTheDocument();
    expect(screen.queryByText('热门类别')).not.toBeInTheDocument();
    expect(screen.queryByText('类型分布')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /导出报表/ })).not.toBeInTheDocument();
    expect(screen.getByRole('tab', { name: '深度分析' })).toBeInTheDocument();
  });
});
