import { render, screen, waitFor } from '@/lib/test-utils';
import { TicketApi } from '@/lib/api/ticket-api';
import type { TicketStatsResponse } from '@/lib/api/ticket-api';
import { ticketService } from '@/lib/services/ticket-service';
import { TicketStatus, TicketStatusConfig } from '@/constants/taxonomy';
import TicketsReportPage from '../page';

jest.mock('@/lib/api/ticket-api', () => ({
  TicketApi: {
    getTicketStats: jest.fn(),
  },
}));

jest.mock('@/lib/services/ticket-service', () => ({
  ticketService: {
    listTickets: jest.fn(),
  },
}));

const mockGetTicketStats = TicketApi.getTicketStats as jest.Mock;
const mockListTickets = ticketService.listTickets as jest.Mock;

// recharts 在 jsdom 下没有容器尺寸，这里把喂给图表的数据摊出来断言：
// 本页的用户可见承诺是「图上就是后端分组计数的真实分布」，渲染尺寸不是回归目标。
// 饼图数据挂在 <Pie data>，柱状图挂在 <BarChart data>，两份都要 dump。
jest.mock('recharts', () => {
  const React = require('react');
  const dump = (data: unknown) =>
    React.createElement('div', { 'data-testid': 'chart-data' }, JSON.stringify(data ?? []));
  const withData = ({ data, children }: { data?: unknown; children?: React.ReactNode }) =>
    React.createElement(React.Fragment, null, dump(data), children);
  const passthrough = ({ children }: { children?: React.ReactNode }) =>
    React.createElement(React.Fragment, null, children);
  return {
    __esModule: true,
    ResponsiveContainer: passthrough,
    PieChart: passthrough,
    BarChart: withData,
    Pie: withData,
    Bar: passthrough,
    Cell: () => null,
    XAxis: () => null,
    YAxis: () => null,
    CartesianGrid: () => null,
    Tooltip: () => null,
    Legend: () => null,
  };
});

// 201 份工单：刻意越过修复前 listTickets({pageSize:200}) 的页长，
// 只要页面还在浏览器里自己数，总数就会显示成 200。
function legacyTicketList(count: number) {
  return {
    items: Array.from({ length: count }, (_, index) => ({
      id: index + 1,
      ticketNumber: `TKT-${index}`,
      title: `ticket ${index}`,
      status: index % 2 === 0 ? 'open' : 'in_progress',
      priority: index % 3 === 0 ? 'high' : 'medium',
    })),
    total: count,
    page: 1,
    pageSize: 200,
    totalPages: 1,
  };
}

function statsFixture(overrides: Partial<TicketStatsResponse> = {}): TicketStatsResponse {
  return {
    total: 201,
    open: 41,
    inProgress: 40,
    resolved: 50,
    closed: 60,
    pending: 5,
    highPriority: 12,
    overdue: 7,
    byStatus: [
      { status: 'new', count: 11 },
      { status: 'open', count: 30 },
      { status: 'in_progress', count: 40 },
      { status: 'pending', count: 5 },
      { status: 'resolved', count: 50 },
      { status: 'closed', count: 60 },
      // 词表外的历史取值：分布必须原样保留，否则明细之和与 total 对不上账。
      { status: 'awaiting_vendor', count: 5 },
    ],
    byPriority: [
      { priority: 'low', count: 31 },
      { priority: 'medium', count: 100 },
      { priority: 'high', count: 40 },
      { priority: 'urgent', count: 20 },
      { priority: 'critical', count: 10 },
    ],
    ...overrides,
  };
}

function chartSlices(): { status: unknown[]; priority: unknown[] } {
  const dumps = screen.getAllByTestId('chart-data').map(node => {
    const parsed: unknown = JSON.parse(node.textContent ?? '[]');
    return Array.isArray(parsed) ? parsed : [];
  });
  return { status: dumps[0] ?? [], priority: dumps[1] ?? [] };
}

// antd Statistic 的标题和数值是两个节点，取卡片容器文本断言读数。
function statisticText(title: string): string {
  // 「已解决」既是一张卡片的标题，也是饼图切片的名字，取第一个匹配即可：
  // 两处都必须来自同一份分组计数，读数值相同。
  const node = screen.getAllByText(title)[0]?.closest('.ant-statistic');
  if (!node) throw new Error(`${title} statistic 未渲染`);
  return node.textContent ?? '';
}

describe('/reports/tickets', () => {
  beforeEach(() => {
    mockGetTicketStats.mockReset();
    mockListTickets.mockReset();
    mockListTickets.mockResolvedValue(legacyTicketList(201));
  });

  it('reads the authoritative stats endpoint instead of aggregating one list page', async () => {
    mockGetTicketStats.mockResolvedValue(statsFixture());

    render(<TicketsReportPage />);

    await waitFor(() => expect(screen.getAllByTestId('chart-data')).toHaveLength(2));

    expect(mockGetTicketStats).toHaveBeenCalledTimes(1);
    // 修复前这里调 listTickets({pageSize:200}) 再在浏览器里数：
    // 201 条数据只会数出 200，页面却把它当全量展示。
    expect(mockListTickets).not.toHaveBeenCalled();
    expect(statisticText('工单总数')).toContain('201');
  });

  it('reports the real SLA breach count instead of a never-matching overdue status', async () => {
    mockGetTicketStats.mockResolvedValue(statsFixture());

    render(<TicketsReportPage />);

    await waitFor(() => expect(screen.getByText('SLA 超时')).toBeInTheDocument());

    // 修复前那张卡片过滤 status === 'overdue'，而 overdue 不是工单状态机的取值，
    // 所以永远显示 0；权威口径是 sla_states 里已超时的工单数。
    expect(statisticText('SLA 超时')).toContain('7');
  });

  it('keeps resolved and closed separate and labels the distribution from the shared taxonomy', async () => {
    mockGetTicketStats.mockResolvedValue(statsFixture());

    render(<TicketsReportPage />);

    await waitFor(() => expect(screen.getAllByTestId('chart-data')).toHaveLength(2));

    const { status, priority } = chartSlices();
    expect(status).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          key: 'in_progress',
          name: TicketStatusConfig[TicketStatus.IN_PROGRESS].label,
          value: 40,
          color: expect.any(String),
        }),
        // 修复前「已完成」把 resolved 和 closed 并成一个数，两者再也分不开。
        expect.objectContaining({ key: 'resolved', value: 50 }),
        expect.objectContaining({ key: 'closed', value: 60 }),
        // 枚举外的历史值保留原始字符串展示，不静默丢弃。
        expect.objectContaining({ key: 'awaiting_vendor', name: 'awaiting_vendor', value: 5 }),
      ]),
    );
    expect(priority).toHaveLength(5);
    expect(statisticText('已解决')).toContain('50');
    expect(statisticText('已关闭')).toContain('60');
  });

  it('surfaces the load failure instead of rendering empty charts', async () => {
    mockGetTicketStats.mockRejectedValue(new Error('权限不足'));

    render(<TicketsReportPage />);

    await waitFor(() => expect(screen.getByText('工单统计加载失败')).toBeInTheDocument());
    expect(screen.getByRole('button', { name: '重试' })).toBeInTheDocument();
    // 修复前异常只弹一条 toast，页面照常渲染全 0 卡片和「暂无数据」图表，
    // 接口故障被伪装成「这个租户没有工单」。
    expect(screen.queryByTestId('chart-data')).not.toBeInTheDocument();
    expect(screen.queryByText('工单总数')).not.toBeInTheDocument();
  });

  it('shows the empty state for a tenant without tickets', async () => {
    mockGetTicketStats.mockResolvedValue(
      statsFixture({ total: 0, open: 0, inProgress: 0, resolved: 0, closed: 0, pending: 0, overdue: 0, byStatus: [], byPriority: [] }),
    );

    render(<TicketsReportPage />);

    await waitFor(() => expect(screen.getByText('当前租户还没有工单')).toBeInTheDocument());
    expect(screen.queryByText('工单状态分布')).not.toBeInTheDocument();
  });
});
