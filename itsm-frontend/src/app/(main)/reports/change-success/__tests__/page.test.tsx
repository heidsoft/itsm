import { render, screen, waitFor } from '@/lib/test-utils';
import { ChangeApi } from '@/lib/api/change-api';
import type { ChangeStatsResponse } from '@/lib/api/change-api';
import ChangeSuccessReport from '../page';

jest.mock('@/lib/api/change-api', () => ({
  ChangeApi: {
    getChangeStats: jest.fn(),
  },
}));

const mockGetChangeStats = ChangeApi.getChangeStats as jest.Mock;

// recharts 在 jsdom 下需要容器尺寸，这里直接把传给图表的数据摊出来断言：
// 报表页的用户可见承诺就是「图上是真实计数」，尺寸渲染不是本次回归目标。
// 注意饼图的数据挂在 <Pie data>，柱状图的数据挂在 <BarChart data>，两者都要 dump。
jest.mock('recharts', () => {
  const React = require('react');
  const dump = (data: unknown) =>
    React.createElement('div', { 'data-testid': 'chart-data' }, JSON.stringify(data ?? []));
  const withData = ({
    data,
    children,
  }: {
    data?: unknown;
    children?: React.ReactNode;
  }) => React.createElement(React.Fragment, null, dump(data), children);
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

function statsFixture(overrides: Partial<ChangeStatsResponse> = {}): ChangeStatsResponse {
  return {
    total: 20,
    draft: 4,
    pending: 3,
    approved: 2,
    scheduled: 1,
    inProgress: 5,
    completed: 3,
    failed: 1,
    rolledBack: 1,
    rejected: 0,
    cancelled: 0,
    closed: 0,
    byType: [
      { type: 'standard', count: 8 },
      { type: 'normal', count: 9 },
      { type: 'emergency', count: 3 },
    ],
    ...overrides,
  };
}

interface StatusSliceView {
  status: string;
  name: string;
  value: number;
  color: string;
}

interface TypeSliceView {
  type: string;
  name: string;
  count: number;
  color: string;
}

// 断言的是页面喂给图表的真实数据切片，因此这里按页面渲染顺序取回两份 dump。
function chartData(): { status: StatusSliceView[]; type: TypeSliceView[] } {
  const dumps = screen.getAllByTestId('chart-data').map(node => {
    const parsed: unknown = JSON.parse(node.textContent ?? '[]');
    return parsed;
  });
  return {
    status: dumps[0] as StatusSliceView[],
    type: dumps[1] as TypeSliceView[],
  };
}

// antd Statistic 会把整数和小数拆成两个 span（-value-int / -value-decimal），
// 拼好的「60.0」并不是任何单个文本节点，所以从卡片容器取 textContent 断言。
function successRateText(): string {
  const statistic = screen.getByText('成功率').closest('.ant-statistic');
  if (!statistic) throw new Error('成功率 statistic 未渲染');
  return statistic.textContent ?? '';
}

describe('/reports/change-success', () => {
  beforeEach(() => {
    mockGetChangeStats.mockReset();
  });

  it('renders real status and type counts instead of a fabricated distribution', async () => {
    mockGetChangeStats.mockResolvedValue(statsFixture());

    render(<ChangeSuccessReport />);

    await waitFor(() => expect(screen.getAllByTestId('chart-data')).toHaveLength(2));

    const { status, type } = chartData();
    expect(status).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ name: '实施中', value: 5 }),
        expect.objectContaining({ name: '草稿', value: 4 }),
      ]),
    );
    // 修复前这里读 stats.implementing（后端字段是 inProgress），
    // 且 DTO 根本没有 draft，两个分片永远都是 0。
    expect(status).toEqual(
      expect.arrayContaining([expect.objectContaining({ name: '已完成', value: 3 })]),
    );

    // 修复前类型分布是 Math.floor(total*0.3/0.5/0.2) 造出来的假数：
    // total=20 时会显示 6/10/4，而不是真实的 8/9/3。
    expect(type).toEqual([
      { type: 'standard', name: '标准变更', count: 8, color: expect.any(String) },
      { type: 'normal', name: '普通变更', count: 9, color: expect.any(String) },
      { type: 'emergency', name: '紧急变更', count: 3, color: expect.any(String) },
    ]);
  });

  it('computes the success rate over implemented changes, not over all changes', async () => {
    mockGetChangeStats.mockResolvedValue(statsFixture());

    render(<ChangeSuccessReport />);

    await waitFor(() => expect(screen.getByText('成功率')).toBeInTheDocument());

    const rate = successRateText();
    // completed 3 / (completed 3 + failed 1 + rolledBack 1) = 60.0
    // 修复前分母用的是 total=20，读数会变成 15%，把草稿和待审批都算成不成功。
    expect(rate).toContain('60.0');
    expect(screen.getByText('已实施变更')).toBeInTheDocument();
    expect(screen.getByText('/ 完成+失败+回滚')).toBeInTheDocument();
  });

  it('shows an explicit unavailable rate when nothing has been implemented', async () => {
    mockGetChangeStats.mockResolvedValue(
      statsFixture({ completed: 0, failed: 0, rolledBack: 0 }),
    );

    render(<ChangeSuccessReport />);

    await waitFor(() => expect(screen.getByText('成功率')).toBeInTheDocument());
    expect(screen.getByText('暂无实施结果')).toBeInTheDocument();
    // 分母为 0 不能显示成 0.0%：那会把「还没有实施结果」伪装成一个真实的成功率读数。
    expect(successRateText()).not.toContain('%');
  });

  it('surfaces the load failure instead of silently rendering zeros', async () => {
    mockGetChangeStats.mockRejectedValue(new Error('权限不足'));

    render(<ChangeSuccessReport />);

    await waitFor(() => expect(screen.getByText('变更统计加载失败')).toBeInTheDocument());
    expect(screen.getByRole('button', { name: '重试' })).toBeInTheDocument();
    expect(screen.queryByTestId('chart-data')).not.toBeInTheDocument();
  });

  it('shows the empty state for a tenant without changes', async () => {
    mockGetChangeStats.mockResolvedValue({
      total: 0,
      draft: 0,
      pending: 0,
      approved: 0,
      scheduled: 0,
      inProgress: 0,
      completed: 0,
      failed: 0,
      rolledBack: 0,
      rejected: 0,
      cancelled: 0,
      closed: 0,
      byType: [],
    });

    render(<ChangeSuccessReport />);

    await waitFor(() => expect(screen.getByText('当前租户还没有变更记录')).toBeInTheDocument());
    expect(screen.queryByText('变更状态分布')).not.toBeInTheDocument();
  });
});
