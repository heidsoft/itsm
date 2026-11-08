import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { App } from 'antd';
import CapabilitiesAdminPage from '../page';
import { getCapabilities } from '@/lib/api/capability-api';
import type { ProductCapabilityState } from '@/lib/api/capability-api';

/**
 * 能力矩阵控制台的回归。
 *
 * 这个页面存在的理由是「预览态必须可核对」，所以断言围绕三件事：
 *   1. 每一条 Pilot 都必须把「为什么还不是 GA」显示出来（不是只显示一个黄色标签）；
 *   2. 此刻不可用（部署未就绪 / 探测失败）必须与「本来就还没承诺」区分开——
 *      前者是排障信号，后者是发布决策，混在一起会让运维去查一个不存在的故障；
 *   3. 筛选必须真的过滤，否则「未就绪」计数与列表会自相矛盾。
 */

jest.mock('@/lib/api/capability-api', () => ({
  __esModule: true,
  getCapabilities: jest.fn(),
}));

jest.mock('@/app/components/PageContainer', () => ({
  PageContainer: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

const mockedGetCapabilities = getCapabilities as jest.MockedFunction<typeof getCapabilities>;

const items: ProductCapabilityState[] = [
  {
    key: 'ticket',
    maturity: 'ga',
    buildAvailable: true,
    deploymentReady: true,
    tenantReady: true,
    allowedActions: ['read', 'manage'],
    dependencies: [],
    health: { ok: true, checkedAt: '2026-10-22T10:00:00Z' },
    acceptanceVersion: '2026.10.31',
  },
  {
    key: 'problem',
    maturity: 'ga',
    buildAvailable: true,
    deploymentReady: true,
    tenantReady: true,
    allowedActions: ['read', 'manage'],
    dependencies: ['sla', 'workflow'],
    acceptanceVersion: '2026.10.31',
  },
  {
    key: 'cmdbDiscovery',
    maturity: 'pilot',
    buildAvailable: true,
    deploymentReady: false,
    tenantReady: true,
    allowedActions: ['read'],
    dependencies: ['cmdb'],
    degradedReason: 'deployment dependency is not ready',
    pilotReason:
      '发现任务没有执行者：ClaimDiscoveryJob 仅被单测调用，队列无消费者。晋升条件：worker 消费 + 真实账号发现验收。',
    acceptanceVersion: '2026.10.31',
  },
  {
    key: 'connector.feishu',
    maturity: 'pilot',
    buildAvailable: true,
    deploymentReady: false,
    tenantReady: true,
    allowedActions: ['read'],
    dependencies: ['notification'],
    degradedReason: 'callback token mismatch',
    health: { ok: false, reason: 'callback token mismatch', checkedAt: '2026-10-22T10:00:00Z' },
    pilotReason: '回调验签只有单测，未与真实飞书租户做生产验收。晋升条件：真实租户收发 + 验签验收。',
    acceptanceVersion: '2026.10.31',
  },
];

describe('能力矩阵控制台', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedGetCapabilities.mockResolvedValue({
      items,
      acceptanceVersion: '2026.10.31',
    });
  });

  it('Pilot 条目必须显示「为什么还不是 GA」，不可用条目显示具体原因', async () => {
    render(
      <App>
        <CapabilitiesAdminPage />
      </App>,
    );

    expect(await screen.findByText('cmdbDiscovery')).toBeInTheDocument();
    // 契约验收基线必须可见：售前/运维要能核对这份矩阵是哪一天对账的。
    expect(screen.getByText('2026.10.31')).toBeInTheDocument();
    // Pilot 理由：读者要能据此推进晋升，而不是只看到一个黄色「预览」标签。
    expect(screen.getByText(/发现任务没有执行者/)).toBeInTheDocument();
    // 不可用原因与 pilot 理由同时可见：修好 READY 变量后仍要知道还差什么。
    expect(screen.getByText('callback token mismatch')).toBeInTheDocument();
    expect(screen.getByText(/回调验签只有单测/)).toBeInTheDocument();
    // 部署未就绪与探测失败都计入「此刻不可用」，并给出汇总告警。
    expect(screen.getByText('2 项能力此刻不可用')).toBeInTheDocument();
  });

  it('GA 条目不得带 pilot 理由（否则成熟度与说明自相矛盾）', async () => {
    render(
      <App>
        <CapabilitiesAdminPage />
      </App>,
    );

    expect(await screen.findByText('problem')).toBeInTheDocument();
    const gaRow = screen.getByText('problem').closest('tr');
    expect(gaRow).not.toBeNull();
    expect(gaRow?.textContent).toContain('GA');
    expect(gaRow?.textContent).not.toContain('晋升条件');
  });

  it('「未就绪」筛选只保留此刻不可用的能力', async () => {
    render(
      <App>
        <CapabilitiesAdminPage />
      </App>,
    );

    expect(await screen.findByText('ticket')).toBeInTheDocument();

    const [filterLabel] = screen.getAllByText('未就绪');
    fireEvent.click(filterLabel);

    await waitFor(() => expect(screen.queryByText('ticket')).not.toBeInTheDocument());
    expect(screen.queryByText('problem')).not.toBeInTheDocument();
    expect(screen.getByText('cmdbDiscovery')).toBeInTheDocument();
    expect(screen.getByText('connector.feishu')).toBeInTheDocument();
  });

  it('后端不可达时给出空态而不是假装没有能力', async () => {
    mockedGetCapabilities.mockRejectedValueOnce(new Error('500 内部错误'));

    render(
      <App>
        <CapabilitiesAdminPage />
      </App>,
    );

    expect(await screen.findByText(/能力矩阵为空/)).toBeInTheDocument();
  });

  it('预览条目分区显示剩余阻断与晋升条件，并给出已声明条件数', async () => {
    render(
      <App>
        <CapabilitiesAdminPage />
      </App>,
    );

    expect(await screen.findByText('cmdbDiscovery')).toBeInTheDocument();
    expect(screen.getAllByText('剩余阻断')).toHaveLength(2);
    expect(screen.getAllByText('晋升条件')).toHaveLength(2);
    // 分母是预览总数，缺口不能藏起来。
    expect(screen.getByText(/晋升条件 2\/2/)).toBeInTheDocument();
    const gaRow = screen.getByText('problem').closest('tr');
    expect(gaRow?.textContent).not.toContain('晋升条件');
    expect(gaRow?.textContent).not.toContain('剩余阻断');
  });

  it('可以按预览理由内容搜索，而不只按能力键', async () => {
    render(
      <App>
        <CapabilitiesAdminPage />
      </App>,
    );

    expect(await screen.findByText('cmdbDiscovery')).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText(/按能力键或预览理由搜索/), {
      target: { value: '真实租户收发' },
    });

    await waitFor(() => expect(screen.queryByText('cmdbDiscovery')).not.toBeInTheDocument());
    expect(screen.getByText('connector.feishu')).toBeInTheDocument();
    expect(screen.queryByText('ticket')).not.toBeInTheDocument();
  });

  it('查看完整理由打开详情面板，三段与接口原文都可核对', async () => {
    render(
      <App>
        <CapabilitiesAdminPage />
      </App>,
    );

    expect(await screen.findByText('cmdbDiscovery')).toBeInTheDocument();
    fireEvent.click(screen.getAllByText('查看完整理由')[0]);

    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByText('已闭环进展')).toBeInTheDocument();
    expect(within(dialog).getByText('剩余阻断')).toBeInTheDocument();
    expect(within(dialog).getByText('晋升条件')).toBeInTheDocument();
    expect(within(dialog).getByText('接口原文')).toBeInTheDocument();
    // 切分只服务于展示，接口原文必须一字不少地可回看。
    const reason = items.find(item => item.key === 'cmdbDiscovery')?.pilotReason;
    expect(within(dialog).getByText(reason ?? '')).toBeInTheDocument();
  });

  it('理由缺晋升条件时如实标记契约不完整，不伪装成正常预览态', async () => {
    mockedGetCapabilities.mockResolvedValueOnce({
      items: [{ ...items[2], pilotReason: '只剩一段描述，没有给出晋升条件标记。' }],
      acceptanceVersion: '2026.10.31',
    });

    render(
      <App>
        <CapabilitiesAdminPage />
      </App>,
    );

    expect(await screen.findByText('未声明晋升条件（契约不完整）')).toBeInTheDocument();
    expect(screen.getByText(/晋升条件 0\/1/)).toBeInTheDocument();
    expect(screen.queryByText('查看完整理由')).not.toBeInTheDocument();
  });
});
