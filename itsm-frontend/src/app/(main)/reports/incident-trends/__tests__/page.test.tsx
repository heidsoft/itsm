import { fireEvent, render, screen, waitFor } from '@/lib/test-utils';
import { IncidentAPI } from '@/lib/api/incident-api';
import type { IncidentReport } from '@/lib/api/incident-api';
import IncidentTrendsPage from '../page';

// jest.setup.js 里全局 mock 了 dayjs：所有实例方法返回固定值（format 恒为 '2024-01-01'，
// hour/minute 返回数字后又被 rc-picker 链式调用），既让 antd RangePicker 在 jsdom 下
// 直接抛 TypeError，也让本页的窗口日期无法断言。这里按文件粒度换回真实 dayjs：
// 报表的日期算术正是要验证的行为，而改全局 mock 会影响仓库里所有用例。
jest.mock('dayjs', () => jest.requireActual('dayjs'));

jest.mock('@/lib/api/incident-api', () => ({
  IncidentAPI: {
    getIncidentReport: jest.fn(),
  },
}));

const mockGetIncidentReport = IncidentAPI.getIncidentReport as jest.Mock;

// recharts 在 jsdom 下没有容器尺寸，这里把喂给图表的数据摊出来断言：
// 本页的用户可见承诺是「读数与曲线都来自事件报表端点的窗口口径」，不是渲染尺寸。
// 趋势挂在 <AreaChart data>，两份分布挂在 <BarChart data>，按渲染顺序取三份 dump。
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
    AreaChart: withData,
    Area: passthrough,
    BarChart: withData,
    Bar: passthrough,
    Cell: () => null,
    XAxis: () => null,
    YAxis: () => null,
    CartesianGrid: () => null,
    Tooltip: () => null,
    Legend: () => null,
  };
});

interface TrendPointView {
  date: string;
  created: number;
  resolved: number;
}

interface SliceView {
  key: string;
  name: string;
  value: number;
  color: string;
}

function reportFixture(overrides: Partial<IncidentReport> = {}): IncidentReport {
  // 夹具遵守后端契约不变量：byStatus 与 byPriority 之和都等于 createdInWindow，
  // 否则页面断言就会对着一个后端不可能返回的形状成立。
  return {
    window: { dateFrom: '2026-09-05', dateTo: '2026-10-04', days: 30 },
    createdInWindow: 6,
    resolvedInWindow: 3,
    avgResolutionMinutes: 90,
    byStatus: [
      { value: 'new', count: 2 },
      { value: 'in_progress', count: 1 },
      { value: 'resolved', count: 2 },
      // 词表外的历史取值：原样显示、灰色分片，不并进已知状态。
      { value: 'awaiting_vendor', count: 1 },
    ],
    byPriority: [
      { value: 'low', count: 1 },
      { value: 'medium', count: 2 },
      { value: 'high', count: 1 },
      // critical 才是事件词表；修复前常量表写的是工单的 urgent，最高档一直取不到色。
      { value: 'critical', count: 2 },
    ],
    dailyTrend: [
      { date: '2026-10-02', created: 2, resolved: 0 },
      { date: '2026-10-03', created: 1, resolved: 2 },
      // 无事件的日子后端也返回 0，前端不得再把缺日当成读数缺失。
      { date: '2026-10-04', created: 0, resolved: 0 },
    ],
    ...overrides,
  };
}

function chartDumps(): unknown[] {
  return screen.getAllByTestId('chart-data').map(node => JSON.parse(node.textContent ?? '[]'));
}

function statisticText(title: string): string {
  const statistic = screen.getAllByText(title)[0]?.closest('.ant-statistic');
  if (!statistic) throw new Error(`${title} statistic 未渲染`);
  return statistic.textContent ?? '';
}

describe('/reports/incident-trends', () => {
  beforeEach(() => {
    mockGetIncidentReport.mockReset();
    jest.useFakeTimers({ advanceTimers: true }).setSystemTime(new Date(2026, 9, 4, 12, 0, 0));
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('reads the incident report endpoint with a paired date window', async () => {
    mockGetIncidentReport.mockResolvedValue(reportFixture());

    render(<IncidentTrendsPage />);

    await waitFor(() => expect(screen.getAllByTestId('chart-data')).toHaveLength(3));
    // 修复前这里调的是 ticketAnalyticsService.getAnalytics + ticketService.getTicketStats：
    // 「事件趋势报表」画的是工单数据，且 selectedPeriod/日期控件从未进入请求。
    expect(mockGetIncidentReport).toHaveBeenCalledWith({
      dateFrom: '2026-09-05',
      dateTo: '2026-10-04',
    });
    // 日期控件与请求同源：修复前页面有 7/30/90 预设和 RangePicker，
    // 但两者都只是摆设，selectedPeriod 与选中的日期从未进入任何请求。
    const rangeValues = screen
      .getAllByRole('textbox')
      .map(input => (input as HTMLInputElement).value);
    expect(rangeValues).toEqual(['2026-09-05', '2026-10-04']);
  });

  it('plots the backend window trend without client-side re-aggregation', async () => {
    mockGetIncidentReport.mockResolvedValue(reportFixture());

    render(<IncidentTrendsPage />);

    await waitFor(() => expect(screen.getAllByTestId('chart-data')).toHaveLength(3));
    const [trend, priority, status] = chartDumps() as [TrendPointView[], SliceView[], SliceView[]];

    // 曲线逐点等于后端 dailyTrend：不再把分页结果累加成「全量」，也没有第 4 条虚构序列。
    expect(trend).toEqual([
      { date: '2026-10-02', created: 2, resolved: 0 },
      { date: '2026-10-03', created: 1, resolved: 2 },
      { date: '2026-10-04', created: 0, resolved: 0 },
    ]);
    expect(Object.keys(trend[0]).sort()).toEqual(['created', 'date', 'resolved']);

    expect(priority.map(slice => [slice.key, slice.name, slice.value])).toEqual([
      ['low', '低', 1],
      ['medium', '中', 2],
      ['high', '高', 1],
      ['critical', '紧急', 2],
    ]);
    expect(status).toHaveLength(4);
    // 分片颜色取自词表映射：修复前 toSlices 把原始取值字符串当色值塞给 <Cell fill>，
    // 于是已知状态也拿不到配色（'new'、'low' 不是合法 CSS 颜色）。
    expect(priority.map(slice => slice.color)).toEqual([
      '#52c41a',
      '#faad14',
      '#ff4d4f',
      '#722ed1',
    ]);
    const knownStatuses = status.filter(slice => slice.key !== 'awaiting_vendor');
    expect(knownStatuses.map(slice => slice.color)).toEqual(['#1890ff', '#faad14', '#52c41a']);
    // 后端保证两份分布之和都等于窗口内新建数，前端原样透传。
    expect(status.reduce((sum, slice) => sum + slice.value, 0)).toBe(6);
    expect(priority.reduce((sum, slice) => sum + slice.value, 0)).toBe(6);
    expect(statisticText('窗口内新建')).toContain('6');
    expect(statisticText('窗口内解决')).toContain('3');
  });

  it('keeps out-of-vocabulary values visible instead of merging them into known buckets', async () => {
    mockGetIncidentReport.mockResolvedValue(reportFixture());

    render(<IncidentTrendsPage />);

    await waitFor(() => expect(screen.getAllByTestId('chart-data')).toHaveLength(3));
    const [, , status] = chartDumps() as [TrendPointView[], SliceView[], SliceView[]];
    expect(status.find(slice => slice.key === 'awaiting_vendor')).toEqual({
      key: 'awaiting_vendor',
      name: 'awaiting_vendor',
      value: 1,
      color: '#d9d9d9',
    });
  });

  it('labels the average resolution duration with the unit the number actually means', async () => {
    // 修复前 avgResolutionTime.toFixed(1) + suffix="小时" 直出后端的分钟值，读数差 60 倍。
    const cases: Array<{ minutes: number; want: string }> = [
      { minutes: 45, want: '45 分钟' },
      { minutes: 90, want: '1.5 小时' },
      { minutes: 60 * 24 * 3 + 60, want: '3.0 天' },
    ];
    for (const testCase of cases) {
      mockGetIncidentReport.mockResolvedValue(
        reportFixture({ avgResolutionMinutes: testCase.minutes, resolvedInWindow: 3 }),
      );
      const { unmount } = render(<IncidentTrendsPage />);
      await waitFor(() => expect(screen.getByText(testCase.want)).toBeInTheDocument());
      // 旧实现直接把分钟值挂上「小时」后缀。
      expect(screen.queryByText(`${testCase.minutes.toFixed(1)} 小时`)).not.toBeInTheDocument();
      unmount();
    }
  });

  it('does not turn the empty-set average into a fake zero duration', async () => {
    mockGetIncidentReport.mockResolvedValue(
      reportFixture({
        resolvedInWindow: 0,
        avgResolutionMinutes: 0,
        createdInWindow: 2,
        byStatus: [{ value: 'new', count: 2 }],
        byPriority: [{ value: 'medium', count: 2 }],
        dailyTrend: [{ date: '2026-10-04', created: 2, resolved: 0 }],
      }),
    );

    render(<IncidentTrendsPage />);

    await waitFor(() => expect(screen.getByText('窗口内无解决事件')).toBeInTheDocument());
    // 后端对空集合 COALESCE 出 0，那是「无样本」而不是「0 分钟」。
    expect(statisticText('平均解决时长')).not.toContain('0 分钟');
  });

  it('surfaces the load failure instead of rendering zeroed cards', async () => {
    mockGetIncidentReport.mockRejectedValue(new Error('网络异常'));

    render(<IncidentTrendsPage />);

    await waitFor(() => expect(screen.getByText('事件趋势数据加载失败')).toBeInTheDocument());
    expect(screen.getByText(/网络异常/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '重试' })).toBeInTheDocument();
    // 修复前失败只 console.error + toast，四张卡片继续显示 0，把故障伪装成空态。
    expect(screen.queryByText('窗口内新建')).not.toBeInTheDocument();
    expect(screen.queryByTestId('chart-data')).not.toBeInTheDocument();
  });

  it('reloads the same window when retry is pressed', async () => {
    mockGetIncidentReport.mockRejectedValueOnce(new Error('超时'));
    render(<IncidentTrendsPage />);
    await waitFor(() => expect(screen.getByText('事件趋势数据加载失败')).toBeInTheDocument());

    mockGetIncidentReport.mockResolvedValue(reportFixture());
    fireEvent.click(screen.getByRole('button', { name: '重试' }));

    await waitFor(() => expect(screen.getByText('窗口内新建')).toBeInTheDocument());
    expect(mockGetIncidentReport).toHaveBeenCalledTimes(2);
    expect(mockGetIncidentReport).toHaveBeenLastCalledWith({
      dateFrom: '2026-09-05',
      dateTo: '2026-10-04',
    });
  });

  it('shows an explicit empty window state instead of charts full of fabricated zeros', async () => {
    mockGetIncidentReport.mockResolvedValue(
      reportFixture({
        createdInWindow: 0,
        resolvedInWindow: 0,
        avgResolutionMinutes: 0,
        byStatus: [],
        byPriority: [],
        dailyTrend: [
          { date: '2026-10-03', created: 0, resolved: 0 },
          { date: '2026-10-04', created: 0, resolved: 0 },
        ],
      }),
    );

    render(<IncidentTrendsPage />);

    await waitFor(() => expect(screen.getByText('所选窗口内没有事件记录')).toBeInTheDocument());
    expect(screen.queryByTestId('chart-data')).not.toBeInTheDocument();
    expect(screen.queryByText(/NaN/)).not.toBeInTheDocument();
  });

  it('echoes the server-resolved window instead of assuming the requested one', async () => {
    // 后端按服务器时区整日边界解释窗口，页面显示的是响应回显值。
    mockGetIncidentReport.mockResolvedValue(
      reportFixture({ window: { dateFrom: '2026-09-06', dateTo: '2026-10-04', days: 29 } }),
    );

    render(<IncidentTrendsPage />);

    await waitFor(() =>
      expect(screen.getByText(/2026-09-06 至 2026-10-04（29 天/)).toBeInTheDocument(),
    );
    expect(statisticText('统计天数')).toContain('29');
  });
});
