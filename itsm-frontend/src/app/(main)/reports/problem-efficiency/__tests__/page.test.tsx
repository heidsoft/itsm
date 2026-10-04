import { render, screen, waitFor } from '@/lib/test-utils';
import { ProblemApi } from '@/lib/api/problem-api';
import type { ProblemStatsResponse } from '@/lib/api/problem-api';
import ProblemEfficiencyPage from '../page';

jest.mock('@/lib/api/problem-api', () => ({
  ProblemApi: {
    getProblemStats: jest.fn(),
    getProblems: jest.fn(),
  },
}));

const mockGetProblemStats = ProblemApi.getProblemStats as jest.Mock;
const mockGetProblems = ProblemApi.getProblems as jest.Mock;

// recharts 在 jsdom 下需要容器尺寸，这里直接把传给图表的数据摊出来断言：
// 报表页的用户可见承诺就是「图上是租户全量的真实计数」，尺寸渲染不是本次回归目标。
// 注意饼图的数据挂在 <Pie data>，柱状图的数据挂在 <BarChart data>，两者都要 dump。
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

function statsFixture(overrides: Partial<ProblemStatsResponse> = {}): ProblemStatsResponse {
  // 折叠桶与分布必须自洽：inProgress=investigating+in_progress，
  // highPriority=high+critical，identified 不进任何单值桶，因此单值之和小于 total。
  return {
    total: 150,
    open: 60,
    inProgress: 50,
    resolved: 30,
    closed: 7,
    highPriority: 22,
    byStatus: [
      { status: 'open', count: 60 },
      { status: 'investigating', count: 35 },
      { status: 'identified', count: 2 },
      { status: 'in_progress', count: 15 },
      { status: 'resolved', count: 30 },
      { status: 'closed', count: 7 },
      // 词表外的历史脏值：原样显示、灰色分片，不并进已知状态。
      { status: 'awaiting_vendor', count: 1 },
    ],
    byPriority: [
      { priority: 'low', count: 40 },
      { priority: 'medium', count: 88 },
      { priority: 'high', count: 18 },
      { priority: 'critical', count: 4 },
    ],
    ...overrides,
  };
}

// 只用于「最新问题列表」：页长固定为 10，读数不来自这里。
function recentFixture(count = 2) {
  return {
    items: Array.from({ length: count }, (_, index) => ({
      id: index + 1,
      title: `问题 ${index + 1}`,
      description: '',
      status: index === 0 ? 'identified' : 'open',
      priority: index === 0 ? 'critical' : 'low',
      category: '',
      rootCause: '',
      workaround: '',
      resolution: '',
      impact: '',
      createdBy: 7,
      tenantId: 1,
      createdAt: '2026-03-0${index + 1}T00:00:00Z',
      updatedAt: '2026-03-01T00:00:00Z',
      assigneeName: index === 0 ? '张三' : undefined,
    })),
    total: count,
    page: 1,
    pageSize: 10,
    totalPages: 1,
  };
}

interface SliceView {
  key: string;
  name: string;
  value: number;
  color: string;
}

// 断言的是页面喂给图表的真实数据切片，按渲染顺序取回两份 dump。
function chartData(): { status: SliceView[]; priority: SliceView[] } {
  const dumps = screen.getAllByTestId('chart-data').map(node => JSON.parse(node.textContent ?? '[]'));
  return { status: dumps[0] as SliceView[], priority: dumps[1] as SliceView[] };
}

function statisticText(title: string): string {
  const statistic = screen.getAllByText(title)[0]?.closest('.ant-statistic');
  if (!statistic) throw new Error(`${title} statistic 未渲染`);
  return statistic.textContent ?? '';
}

describe('/reports/problem-efficiency', () => {
  beforeEach(() => {
    mockGetProblemStats.mockReset();
    mockGetProblems.mockReset();
  });

  it('plots the backend tenant-wide distribution instead of counting the current page', async () => {
    mockGetProblemStats.mockResolvedValue(statsFixture());
    mockGetProblems.mockResolvedValue(recentFixture(2));

    render(<ProblemEfficiencyPage />);

    await waitFor(() => expect(screen.getAllByTestId('chart-data')).toHaveLength(2));

    expect(mockGetProblems).toHaveBeenCalledWith({ page: 1, pageSize: 10 });

    const { status, priority } = chartData();
    // 修复前这里数的是 listProblems({pageSize:100}) 的当前页：第 100 条之后的问题被
    // 静默截断，页长之外的问题根本不进图；现在分片必须等于 stats.byStatus 的计数。
    expect(status).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ key: 'open', name: '待处理', value: 60 }),
        expect.objectContaining({ key: 'identified', name: '已识别', value: 2 }),
      ]),
    );
    expect(status).toHaveLength(7);
    expect(priority.map(slice => slice.value)).toEqual([40, 88, 18, 4]);
    // identified 不属于任何单值桶，只有分布能让两份切片都与 total 对账。
    expect(status.reduce((sum, slice) => sum + slice.value, 0)).toBe(150);
    expect(priority.reduce((sum, slice) => sum + slice.value, 0)).toBe(150);
    expect(statisticText('处理中')).toContain('50');
  });

  it('keeps out-of-vocabulary values visible and unstyled rather than merging them', async () => {
    mockGetProblemStats.mockResolvedValue(statsFixture());
    mockGetProblems.mockResolvedValue(recentFixture(1));

    render(<ProblemEfficiencyPage />);

    await waitFor(() => expect(screen.getAllByTestId('chart-data')).toHaveLength(2));

    const { status } = chartData();
    const dirty = status.find(slice => slice.key === 'awaiting_vendor');
    expect(dirty).toEqual({
      key: 'awaiting_vendor',
      name: 'awaiting_vendor',
      value: 1,
      color: '#d9d9d9',
    });
  });

  it('reads the metric rates from the stats buckets', async () => {
    mockGetProblemStats.mockResolvedValue(
      statsFixture({ total: 8, resolved: 2, inProgress: 3, highPriority: 1 }),
    );
    mockGetProblems.mockResolvedValue(recentFixture(1));

    render(<ProblemEfficiencyPage />);

    await waitFor(() => expect(screen.getByText('解决率')).toBeInTheDocument());

    expect(statisticText('问题总数')).toContain('8');
    expect(screen.getByText('25.0%')).toBeInTheDocument();
    expect(screen.getByText('37.5%')).toBeInTheDocument();
    expect(screen.getByText('12.5%')).toBeInTheDocument();
    expect(screen.getByText('已解决 2 / 总数 8')).toBeInTheDocument();
  });

  it('shows an explicit empty state instead of NaN rates when the tenant has no problems', async () => {
    mockGetProblemStats.mockResolvedValue({
      total: 0,
      open: 0,
      inProgress: 0,
      resolved: 0,
      closed: 0,
      highPriority: 0,
      byStatus: [],
      byPriority: [],
    });
    mockGetProblems.mockResolvedValue(recentFixture(0));

    render(<ProblemEfficiencyPage />);

    await waitFor(() => expect(screen.getByText('当前租户还没有问题记录')).toBeInTheDocument());
    // 修复前这里是 (0/0)*100，页面上会出现 "NaN%" 这样的读数。
    expect(screen.queryByText('解决率')).not.toBeInTheDocument();
    expect(screen.queryByText(/NaN/)).not.toBeInTheDocument();
    expect(screen.queryByTestId('chart-data')).not.toBeInTheDocument();
  });

  it('surfaces the load failure instead of disguising it as an empty tenant', async () => {
    mockGetProblemStats.mockRejectedValue(new Error('权限不足'));
    mockGetProblems.mockResolvedValue(recentFixture(1));

    render(<ProblemEfficiencyPage />);

    await waitFor(() => expect(screen.getByText('问题统计加载失败')).toBeInTheDocument());
    expect(screen.getByRole('button', { name: '重试' })).toBeInTheDocument();
    // 修复前失败只 console.warn('获取问题数据失败，使用空数据') + toast，
    // 然后照样渲染全 0 的「问题总数」卡片，把故障伪装成空态。
    expect(screen.queryByText('问题总数')).not.toBeInTheDocument();
    expect(screen.queryByTestId('chart-data')).not.toBeInTheDocument();
  });

  it('surfaces the recent list failure too, without falling back to silent zeros', async () => {
    mockGetProblemStats.mockResolvedValue(statsFixture());
    mockGetProblems.mockRejectedValue(new Error('服务不可用'));

    render(<ProblemEfficiencyPage />);

    await waitFor(() => expect(screen.getByText('问题统计加载失败')).toBeInTheDocument());
    expect(screen.getByText(/服务不可用/)).toBeInTheDocument();
  });

  it('labels recent problems from the shared taxonomy and keeps unknown values raw', async () => {
    mockGetProblemStats.mockResolvedValue(statsFixture());
    mockGetProblems.mockResolvedValue(recentFixture(2));

    render(<ProblemEfficiencyPage />);

    await waitFor(() => expect(screen.getByText('最新问题列表')).toBeInTheDocument());
    expect(screen.getByText('已识别')).toBeInTheDocument();
    expect(screen.getByText('极高')).toBeInTheDocument();
    expect(screen.getByText('处理人: 张三')).toBeInTheDocument();
    expect(screen.getByText('处理人: 未分配')).toBeInTheDocument();
  });
});
