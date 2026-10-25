import { render, screen, waitFor } from '@/lib/test-utils';
import userEvent from '@testing-library/user-event';
import { ApiRequestError } from '@/lib/api/http-client';
import { ticketAnalyticsService } from '@/lib/services/analytics-service';
import { ticketService } from '@/lib/services/ticket-service';
import TicketAnalyticsPage from '../page';

// jest.setup.js 全局 mock 的 dayjs 实例方法会被 rc-picker 链式调用，jsdom 下 RangePicker
// 直接抛 TypeError。这里按文件粒度换回真实 dayjs，本页断言的是失败语义而不是日期控件。
jest.mock('dayjs', () => jest.requireActual('dayjs'));

jest.mock('@/lib/services/analytics-service', () => ({
  ticketAnalyticsService: { getAnalytics: jest.fn() },
}));

jest.mock('@/lib/services/ticket-service', () => ({
  ticketService: { getTicketStats: jest.fn() },
}));

// 深度分析 tab 自己会发请求，本页回归目标不覆盖它。
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

const mockGetAnalytics = ticketAnalyticsService.getAnalytics as jest.Mock;
const mockGetTicketStats = ticketService.getTicketStats as jest.Mock;

// 两种口径刻意不同：stats 是租户全量（22），analytics 是所选窗口（20）。
const ANALYTICS = {
  totalTickets: 20,
  openTickets: 5,
  resolvedTickets: 10,
  closedTickets: 4,
  overdueTickets: 1,
  dailyTrend: [{ date: '2026-10-01', created: 3, resolved: 1, open: 2 }],
  statusDistribution: [{ name: 'open', value: 5, color: '#1890ff' }],
  priorityDistribution: [{ name: 'high', value: 4, color: '#ff4d4f' }],
  typeDistribution: [],
  processingTimeStats: { avgProcessingTime: 2, avgResolutionTime: 8, slaComplianceRate: 90 },
  teamPerformance: [],
  hotCategories: [],
};

const STATS = {
  total: 22,
  open: 6,
  inProgress: 2,
  resolved: 11,
  closed: 3,
  pending: 1,
  highPriority: 4,
  overdue: 2,
  byStatus: [{ status: 'open', count: 6 }],
  byPriority: [{ priority: 'high', count: 4 }],
};

describe('工单分析页加载语义', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('两个数据源都成功时不显示降级提示', async () => {
    mockGetAnalytics.mockResolvedValue(ANALYTICS);
    mockGetTicketStats.mockResolvedValue(STATS);
    render(<TicketAnalyticsPage />);

    expect(await screen.findByText('22')).toBeInTheDocument();
    expect(screen.queryByText(/加载失败|没有访问权限/)).not.toBeInTheDocument();
  });

  // 菜单权限码修复前，坐席（有 ticket:read、无 report:read）点进来就是这个场景：
  // 旧实现用 Promise.all + getInitialData()，把 403 渲染成一屏 0 加一句「获取分析数据失败」。
  it('分析接口 403 时保留工单总量真实数字，并说明缺 report:read', async () => {
    mockGetAnalytics.mockRejectedValue(new ApiRequestError('权限不足', 403, 2003));
    mockGetTicketStats.mockResolvedValue(STATS);
    render(<TicketAnalyticsPage />);

    expect(await screen.findByText('日期窗口内的趋势与分布没有访问权限')).toBeInTheDocument();
    expect(screen.getByText(/需要 report:read/)).toBeInTheDocument();
    expect(screen.getByText(/缺少的数据块不等于租户里没有工单/)).toBeInTheDocument();
    expect(screen.getByText('22')).toBeInTheDocument();
    expect(screen.queryByText('获取分析数据失败')).not.toBeInTheDocument();
  });

  it('总量接口失败但分析接口可用时，仍渲染窗口数字并标注失败源', async () => {
    mockGetAnalytics.mockResolvedValue(ANALYTICS);
    mockGetTicketStats.mockRejectedValue(new ApiRequestError('权限不足', 403, 2003));
    render(<TicketAnalyticsPage />);

    expect(await screen.findByText('工单总量没有访问权限')).toBeInTheDocument();
    expect(screen.getByText(/需要 ticket:read/)).toBeInTheDocument();
    expect(screen.getByText('20')).toBeInTheDocument();
  });

  it('两个数据源都失败时不渲染任何统计卡', async () => {
    mockGetAnalytics.mockRejectedValue(new ApiRequestError('权限不足', 403, 2003));
    mockGetTicketStats.mockRejectedValue(new ApiRequestError('权限不足', 403, 2003));
    render(<TicketAnalyticsPage />);

    expect(await screen.findByText('工单分析数据没有访问权限')).toBeInTheDocument();
    expect(screen.getByText(/权限不足/)).toBeInTheDocument();
    expect(screen.queryByText('工单总数')).not.toBeInTheDocument();
    expect(screen.queryByText('待处理')).not.toBeInTheDocument();
  });

  it('整页失败重试成功后恢复数据', async () => {
    mockGetAnalytics.mockRejectedValue(new ApiRequestError('后端未就绪', 503, 5001));
    mockGetTicketStats.mockRejectedValue(new ApiRequestError('后端未就绪', 503, 5001));
    render(<TicketAnalyticsPage />);

    expect(await screen.findByText('工单分析数据加载失败')).toBeInTheDocument();
    expect(screen.queryByText('工单总数')).not.toBeInTheDocument();

    mockGetAnalytics.mockResolvedValue(ANALYTICS);
    mockGetTicketStats.mockResolvedValue(STATS);
    await userEvent.click(screen.getByRole('button', { name: '重试' }));

    await waitFor(() => expect(mockGetAnalytics).toHaveBeenCalledTimes(2));
    expect(await screen.findByText('22')).toBeInTheDocument();
    expect(screen.queryByText('工单分析数据加载失败')).not.toBeInTheDocument();
  });
});
