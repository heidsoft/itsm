import React from 'react';
import { render, screen, waitFor, fireEvent, act } from '@testing-library/react';
import { App } from 'antd';
import '@testing-library/jest-dom';

/**
 * AI 工具审批队列：审计可回放字段的前端契约（2026-11-21）
 *
 * 守的是三件在交互层容易被悄悄破坏的事：
 *   ① 放行依据三态（历史记录 / 未取到 / 指纹）**不得**在展示层被归一化成同一个「-」——
 *      一旦归一化，「有概念但没取到」这种需要运维介入的故障就永久隐身；
 *   ② 展开某行时拉取的是**该行 runId** 的整轮证据链，而不是「最后返回的那个响应」——
 *      快速连续展开两行时串台，会让审批人把 A 行的上下文当成 B 行的；
 *   ③ 轮次加载失败必须与「该轮确实没有动作」分流——
 *      都渲染成空列表会让人误判「前面什么都没做」，从而低估本次调用风险。
 *
 * 反向验证：把 policyVersionView 改成「非空即返回原值、空值统一 '-'」，
 * 或去掉 handleExpand 里的 seq 守卫，或把 catch 分支改回 setItems([])，
 * 对应断言应转红。
 */

jest.mock('@/lib/store/auth-store', () => ({
  useAuthStoreHydration: () => {},
}));

jest.mock('@/lib/hooks/use-permissions', () => ({
  usePermissions: () => ({
    hasPermission: (code: string) => code === 'ai:write',
  }),
}));

jest.mock('@/lib/api/ai-api', () => ({
  aiGetToolApprovals: jest.fn(),
  aiApproveTool: jest.fn(),
  aiGetRunInvocations: jest.fn(),
  // 复核必须出现在 mock 里：漏了它 aiReviewToolInvocation 是 undefined，
  // 调用处的 try/catch 会把它吞成「复核失败」，测试仍全绿却掩盖了真问题。
  // 这不属于本文件断言范围，但 mock 与真实导出集合不一致本身就是缺陷源。
  aiReviewToolInvocation: jest.fn(),
  // 控制面聚合读面同理：页面 mount 即调它。缺了它 aiGetControlPlane 是
  // undefined，调用抛 TypeError 被 catch 成「控制面读取失败」——
  // 本文件的断言不会红，但页面会永远停在错误态，属 mock 漂移导致的假绿。
  aiGetControlPlane: jest.fn(),
}));

import AIApprovalQueue from '../page';
import {
  aiGetToolApprovals,
  aiGetRunInvocations,
  aiReviewToolInvocation,
} from '@/lib/api/ai-api';

const mockGetApprovals = aiGetToolApprovals as jest.Mock;
const mockGetRun = aiGetRunInvocations as jest.Mock;
const mockReview = aiReviewToolInvocation as jest.Mock;

/** 造一条审批记录；policyVersion/runId 可显式给（含空串与 unknown 两种特殊值）。 */
const makeApproval = (over: Record<string, unknown> = {}) => ({
  id: 1,
  toolName: 'create_ticket',
  arguments: '{"title":"磁盘告警"}',
  status: 'pending',
  needsApproval: true,
  approvalState: 'pending_agent',
  permissionCheck: 'passed',
  permissionReason: 'principal=agent risk=high minApprovals=2',
  createdAt: '2026-11-21T08:00:00Z',
  conversationId: 0,
  userId: 1,
  riskLevel: 'high',
  minApprovals: 2,
  policyVersion: 'aabbccdd11223344aabbccdd11223344',
  runId: 'run_00112233445566778899aabbccddeeff',
  ...over,
});

/** 一条「判定通过、需 2 人审批」的预演结论，作为多数用例的默认返回值。 */
const basePreflight = (over: Record<string, unknown> = {}) => ({
  dryRun: true,
  toolName: 'create_ticket',
  allowed: true,
  needsApproval: true,
  minApprovals: 2,
  approvalState: 'pending_agent_two',
  riskLevel: 'high',
  policyVersion: 'aabbccdd11223344aabbccdd11223344',
  reason: 'principal=agent risk=high minApprovals=2 policyVersion=aabbccdd',
  auditId: 1,
  ...over,
});

const renderPage = () =>
  render(
    <App>
      <AIApprovalQueue />
    </App>
  );

/** 展开第一行（Table expandable 的展开按钮是 aria-label 为 Expand row 的图标按钮）。 */
const expandFirstRow = async () => {
  const row = await screen.findByText('create_ticket');
  const tr = row.closest('tr');
  expect(tr).not.toBeNull();
  const buttons = tr!.querySelectorAll('button');
  // antd v6 的展开按钮是行内第一个 button（无文本），点它等同 onExpand(true)。
  //
  // 点击必须包在 act 里：onExpand 会同步发起两条请求（轮次链 + 复核），
  // 它们的 state 落地发生在点击之后。不包 act 时这些更新跑到测试的
  // act 作用域之外，React 报「not wrapped in act」——被掩盖的真相是
  // 「断言可能在请求落地前就执行了」。
  await act(async () => {
    fireEvent.click(buttons[0]);
  });
};

describe('AI 审批队列：策略版本三态展示', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockGetRun.mockResolvedValue({ items: [], runId: 'run_x', count: 0 });
    // 本组不断言预演，但展开行会并发发起预演请求：不给默认返回值会让它
    // reject，进而在控制台刷「预演失败」——与本组结论无关的噪音会掩盖真问题。
    mockReview.mockResolvedValue(basePreflight());
  });

  it('三态不可归一化：历史记录 / 未取到 / 指纹显示为不同标签', async () => {
    mockGetApprovals.mockResolvedValue({
      items: [
        makeApproval({ id: 1, policyVersion: '' }),
        makeApproval({ id: 2, policyVersion: 'unknown' }),
        makeApproval({ id: 3, policyVersion: 'aabbccdd11223344aabbccdd11223344' }),
      ],
      state: 'pending',
    });

    renderPage();
    await waitFor(() => expect(mockGetApprovals).toHaveBeenCalled());

    // 空串 = 历史行（字段上线前），unknown = 本次未取到，指纹 = 前 8 位
    expect(await screen.findAllByText('历史记录')).toHaveLength(1);
    expect(await screen.findAllByText('未取到')).toHaveLength(1);
    expect(await screen.findAllByText('aabbccdd')).toHaveLength(1);
  });

  it('未取到必须给出「需检查策略目录」而非普通空值的提示', async () => {
    mockGetApprovals.mockResolvedValue({
      items: [makeApproval({ id: 1, policyVersion: 'unknown' })],
      state: 'pending',
    });

    renderPage();
    await waitFor(() => expect(mockGetApprovals).toHaveBeenCalled());

    const tag = await screen.findByText('未取到');
    // Tag 自身就是 <span>，提示挂在外层 TooltipWrapper 上，故取 parentElement。
    const wrapper = tag.closest('.ant-tag')?.parentElement ?? tag.parentElement;
    expect(wrapper?.getAttribute('title') || '').toContain('检查权限目录');
  });
});

describe('AI 审批队列：轮次证据链', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockReview.mockResolvedValue(basePreflight());
  });

  it('展开某行时按该行 runId 拉取整轮记录', async () => {
    mockGetApprovals.mockResolvedValue({
      items: [makeApproval({ id: 1, runId: 'run_aaa' })],
      state: 'pending',
    });
    mockGetRun.mockResolvedValue({
      items: [
        makeApproval({ id: 1, runId: 'run_aaa', toolName: 'create_ticket' }),
        makeApproval({ id: 9, runId: 'run_aaa', toolName: 'search_knowledge', approvalState: 'auto' }),
      ],
      runId: 'run_aaa',
      count: 2,
    });

    renderPage();
    await waitFor(() => expect(mockGetApprovals).toHaveBeenCalled());
    await expandFirstRow();

    await waitFor(() => expect(mockGetRun).toHaveBeenCalledWith('run_aaa'));
    // 同轮的前序动作必须出现在展开区域
    expect(await screen.findByText('search_knowledge')).toBeInTheDocument();
    expect(await screen.findByText('← 当前条目')).toBeInTheDocument();
  });

  it('无 runId 的历史行不发起请求，并明确告知无法追溯', async () => {
    mockGetApprovals.mockResolvedValue({
      items: [makeApproval({ id: 1, runId: '' })],
      state: 'pending',
    });

    renderPage();
    await waitFor(() => expect(mockGetApprovals).toHaveBeenCalled());
    await expandFirstRow();

    expect(mockGetRun).not.toHaveBeenCalled();
    expect(await screen.findByText(/未归属任何运行轮次/)).toBeInTheDocument();
  });

  it('加载失败与「该轮无动作」分流，失败须提示不等于没有动作', async () => {
    mockGetApprovals.mockResolvedValue({
      items: [makeApproval({ id: 1, runId: 'run_bbb' })],
      state: 'pending',
    });
    mockGetRun.mockRejectedValue(new Error('网络错误'));

    renderPage();
    await waitFor(() => expect(mockGetApprovals).toHaveBeenCalled());
    await expandFirstRow();

    expect(await screen.findByText(/加载轮次记录失败/)).toBeInTheDocument();
    expect(await screen.findByText(/这不等于该轮没有动作/)).toBeInTheDocument();
    // 不得渲染成「该轮次没有可读的工具调用记录」这种伪空态
    expect(screen.queryByText(/没有可读的工具调用记录/)).not.toBeInTheDocument();
  });

  it('轮次确实为空时给出「无动作」而非错误', async () => {
    mockGetApprovals.mockResolvedValue({
      items: [makeApproval({ id: 1, runId: 'run_ccc' })],
      state: 'pending',
    });
    mockGetRun.mockResolvedValue({ items: [], runId: 'run_ccc', count: 0 });

    renderPage();
    await waitFor(() => expect(mockGetApprovals).toHaveBeenCalled());
    await expandFirstRow();

    expect(await screen.findByText(/没有可读的工具调用记录/)).toBeInTheDocument();
    expect(screen.queryByText(/加载轮次记录失败/)).not.toBeInTheDocument();
  });

  it('同一轮重复展开：过期响应不得覆盖最新一次', async () => {
    mockGetApprovals.mockResolvedValue({
      items: [makeApproval({ id: 1, runId: 'run_same' })],
      state: 'pending',
    });

    // 第一次展开返回「旧内容」，第二次展开返回「新内容」。若守卫失效，
    // 迟到的旧响应会把新内容顶掉——这正是「同一轮在两次展开之间又发生了动作」的真实场景。
    let resolveOld: (v: unknown) => void = () => {};
    const oldPromise = new Promise(resolve => {
      resolveOld = resolve;
    });
    let call = 0;
    mockGetRun.mockImplementation(() => {
      call += 1;
      if (call === 1) return oldPromise;
      return Promise.resolve({
        items: [
          makeApproval({ id: 1, runId: 'run_same', toolName: 'create_ticket' }),
          makeApproval({ id: 2, runId: 'run_same', toolName: 'new_step_later' }),
        ],
        runId: 'run_same',
        count: 2,
      });
    });

    renderPage();
    await waitFor(() => expect(mockGetApprovals).toHaveBeenCalled());

    const row = (await screen.findAllByRole('row'))[1].closest('tr') as HTMLElement;
    const expandBtn = () => fireEvent.click(row.querySelectorAll('button')[0]);

    expandBtn(); // 第一次展开（挂起）
    expandBtn(); // 收起
    expandBtn(); // 第三次展开 → 第二次请求，立刻返回新内容
    await waitFor(() => expect(screen.getByText('new_step_later')).toBeInTheDocument());

    // 迟到返回的旧响应若落地，新内容会被顶掉 → 断言必须守住它仍在
    resolveOld({
      items: [makeApproval({ id: 1, runId: 'run_same', toolName: 'stale_step' })],
      runId: 'run_same',
      count: 1,
    });
    // waitFor 会给 microtask 充分时间落地：若守卫失效，此处 stale_step 出现且 new_step_later 消失
    await waitFor(() => expect(screen.queryByText('stale_step')).not.toBeInTheDocument());
    expect(screen.getByText('new_step_later')).toBeInTheDocument();
  });

  it('多行同时展开：各行的证据链互不顶掉', async () => {
    mockGetApprovals.mockResolvedValue({
      items: [
        makeApproval({ id: 1, runId: 'run_a', toolName: 'create_ticket' }),
        makeApproval({ id: 2, runId: 'run_b', toolName: 'update_ticket' }),
      ],
      state: 'pending',
    });
    mockGetRun.mockImplementation((rid: string) =>
      Promise.resolve({
        items: [
          makeApproval({
            id: rid === 'run_a' ? 1 : 2,
            runId: rid,
            toolName: rid === 'run_a' ? 'create_ticket' : 'update_ticket',
          }),
          makeApproval({
            id: rid === 'run_a' ? 11 : 22,
            runId: rid,
            toolName: rid === 'run_a' ? 'step_of_a' : 'step_of_b',
          }),
        ],
        runId: rid,
        count: 2,
      })
    );

    renderPage();
    await waitFor(() => expect(mockGetApprovals).toHaveBeenCalled());

    const rows = await screen.findAllByRole('row');
    // 两次展开必须各自包一层 act：同一个 act 内的连续点击会被 antd 的
    // 受控 expandedRowKeys 合并成一次更新，第二行的 onExpand 不触发，
    // 于是「两行同时展开」这个被测场景根本没发生（会假绿）。
    await act(async () => {
      fireEvent.click((rows[1].closest('tr') as HTMLElement).querySelectorAll('button')[0]);
    });
    await act(async () => {
      fireEvent.click((rows[2].closest('tr') as HTMLElement).querySelectorAll('button')[0]);
    });
    expect(await screen.findByText('step_of_a')).toBeInTheDocument();
    expect(await screen.findByText('step_of_b')).toBeInTheDocument();
  });
});

describe('AI 审批队列：决策复核（按原发起主体 + 当前策略）', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockGetRun.mockResolvedValue({ items: [], runId: 'run_x', count: 0 });
  });

  it('展开行时按**记录 ID** 发起复核，而不是拿工具名重放一次预演', async () => {
    mockGetApprovals.mockResolvedValue({
      items: [makeApproval({ id: 42, toolName: 'create_ticket', arguments: '{"title":"磁盘告警"}' })],
      state: 'pending',
    });
    mockReview.mockResolvedValue(basePreflight());

    renderPage();
    await waitFor(() => expect(mockGetApprovals).toHaveBeenCalled());
    await expandFirstRow();

    // 必须按记录 id 复核：这是「按原发起主体判定」的唯一入口。
    // 若退回 aiDryRunTool(工具名, 参数)，判定主体就变成当前登录的审批人
    // （通常是管理员，永远 allowed=true），复核沦为仪式。
    await waitFor(() => expect(mockReview).toHaveBeenCalledWith(42));
  });

  it('无 runId 的历史行同样复核（复核只依赖记录本身，不依赖轮次）', async () => {
    mockGetApprovals.mockResolvedValue({
      items: [makeApproval({ id: 1, runId: '' })],
      state: 'pending',
    });
    mockReview.mockResolvedValue(basePreflight());

    renderPage();
    await waitFor(() => expect(mockGetApprovals).toHaveBeenCalled());
    await expandFirstRow();

    // 轮次链会因无 runId 而跳过，但复核不能跟着跳过：
    // 历史行恰恰最缺上下文，最需要「批了会发生什么」这个答案。
    await waitFor(() => expect(mockReview).toHaveBeenCalled());
    expect(mockGetRun).not.toHaveBeenCalled();
  });

  it('复核结论要给出审批人数，让审批人知道还差几个人', async () => {
    mockGetApprovals.mockResolvedValue({ items: [makeApproval({ id: 1 })], state: 'pending' });
    mockReview.mockResolvedValue(basePreflight({ minApprovals: 2 }));

    renderPage();
    await waitFor(() => expect(mockGetApprovals).toHaveBeenCalled());
    await expandFirstRow();

    expect(await screen.findByText('判定通过')).toBeInTheDocument();
    expect(await screen.findByText('需 2 人审批')).toBeInTheDocument();
    expect(await screen.findByText(/风险 high/)).toBeInTheDocument();
  });

  it('必须显示判定主体：AI 发起的调用要标「按 AI 主体判定」', async () => {
    mockGetApprovals.mockResolvedValue({ items: [makeApproval({ id: 1 })], state: 'pending' });
    mockReview.mockResolvedValue(basePreflight({ evaluatedPrincipal: 'agent' }));

    renderPage();
    await waitFor(() => expect(mockGetApprovals).toHaveBeenCalled());
    await expandFirstRow();

    // 不显示主体会让「判定拒绝」被读成「这个工具不能用」，
    // 而实际含义是「AI 主体没被授权」——处置方式完全不同。
    expect(await screen.findByText('按 AI 主体判定')).toBeInTheDocument();
    expect(screen.queryByText(/按 human/)).not.toBeInTheDocument();
  });

  it('判定为拒绝时必须显式呈现拒绝，不得渲染成「通过」', async () => {
    mockGetApprovals.mockResolvedValue({ items: [makeApproval({ id: 1 })], state: 'pending' });
    mockReview.mockResolvedValue(
      basePreflight({
        allowed: false,
        needsApproval: false,
        minApprovals: 0,
        approvalState: '',
        reason: 'AI 主体 agent 未获授权 ticket:write',
      })
    );

    renderPage();
    await waitFor(() => expect(mockGetApprovals).toHaveBeenCalled());
    await expandFirstRow();

    expect(await screen.findByText('判定拒绝')).toBeInTheDocument();
    expect(screen.queryByText('判定通过')).not.toBeInTheDocument();
    expect(await screen.findByText(/未获授权/)).toBeInTheDocument();
  });

  it('无需审批时必须说「批准即刻执行」，不能只写「无需审批」', async () => {
    // 只写「无需审批」会让审批人以为「那我还批什么」，而实际上点通过就会立刻执行。
    mockGetApprovals.mockResolvedValue({ items: [makeApproval({ id: 1 })], state: 'pending' });
    mockReview.mockResolvedValue(
      basePreflight({ needsApproval: false, minApprovals: 0, approvalState: '' })
    );

    renderPage();
    await waitFor(() => expect(mockGetApprovals).toHaveBeenCalled());
    await expandFirstRow();

    expect(await screen.findByText(/批准即刻执行/)).toBeInTheDocument();
  });

  it('复核失败与「判定拒绝」分流：失败不得表现为拒绝结论', async () => {
    mockGetApprovals.mockResolvedValue({ items: [makeApproval({ id: 1 })], state: 'pending' });
    mockReview.mockRejectedValue(new Error('网络错误'));

    renderPage();
    await waitFor(() => expect(mockGetApprovals).toHaveBeenCalled());
    await expandFirstRow();

    expect(await screen.findByText(/复核失败/)).toBeInTheDocument();
    expect(await screen.findByText(/这不等于该操作会被拒绝/)).toBeInTheDocument();
    // 关键：不能因为请求失败就渲染「判定拒绝」——两者对审批决策的含义完全相反。
    expect(screen.queryByText('判定拒绝')).not.toBeInTheDocument();
  });

  it('多行同时展开：各行的复核结论互不顶掉', async () => {
    mockGetApprovals.mockResolvedValue({
      items: [
        makeApproval({ id: 1, toolName: 'create_ticket' }),
        makeApproval({ id: 2, toolName: 'update_ticket', arguments: '{"ticketId":1}' }),
      ],
      state: 'pending',
    });
    // 两条记录给出不同结论：若 state 是单值，后落地的会把先落地的顶掉，
    // 于是某一行显示的是另一行的审批人数——审批人据此判断直接出错。
    mockReview.mockImplementation((id: number) =>
      Promise.resolve(basePreflight({ minApprovals: id === 1 ? 2 : 1 }))
    );

    renderPage();
    await waitFor(() => expect(mockGetApprovals).toHaveBeenCalled());

    const rows = await screen.findAllByRole('row');
    // 同上：两次展开各包一层 act，否则第二次点击被合并、第二行结论根本不产生。
    await act(async () => {
      fireEvent.click((rows[1].closest('tr') as HTMLElement).querySelectorAll('button')[0]);
    });
    await act(async () => {
      fireEvent.click((rows[2].closest('tr') as HTMLElement).querySelectorAll('button')[0]);
    });
    expect(await screen.findByText('需 2 人审批')).toBeInTheDocument();
    expect(await screen.findByText('需 1 人审批')).toBeInTheDocument();
  });
});

describe('AI 审批队列：策略比对四态（复核的核心增量信息）', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockGetRun.mockResolvedValue({ items: [], runId: 'run_x', count: 0 });
    mockGetApprovals.mockResolvedValue({ items: [makeApproval({ id: 1 })], state: 'pending' });
  });

  it('identical → 「策略未变」，可沿用原结论', async () => {
    mockReview.mockResolvedValue(
      basePreflight({
        policyComparison: 'identical',
        baselinePolicyVersion: 'aabbccdd11223344aabbccdd11223344',
      })
    );
    renderPage();
    await waitFor(() => expect(mockGetApprovals).toHaveBeenCalled());
    await expandFirstRow();
    expect(await screen.findByText('策略未变')).toBeInTheDocument();
  });

  it('changed → 「策略已变更」，不得沿用原结论', async () => {
    mockReview.mockResolvedValue(
      basePreflight({
        policyComparison: 'changed',
        baselinePolicyVersion: '00112233445566770011223344556677',
      })
    );
    renderPage();
    await waitFor(() => expect(mockGetApprovals).toHaveBeenCalled());
    await expandFirstRow();
    expect(await screen.findByText('策略已变更')).toBeInTheDocument();
    // 关键：不得同时出现「未变」，否则审批人只看到对他有利的那条。
    expect(screen.queryByText('策略未变')).not.toBeInTheDocument();
  });

  it('unknown → 必须提示「不可读」，文案不得含「未变」', async () => {
    mockReview.mockResolvedValue(
      basePreflight({ policyComparison: 'unknown', policyVersion: 'unknown' })
    );
    renderPage();
    await waitFor(() => expect(mockGetApprovals).toHaveBeenCalled());
    await expandFirstRow();

    expect(await screen.findByText('策略版本不可读')).toBeInTheDocument();
    // 这是本组最重要的一条：把「读不出」显示成「未变」是审计场景最危险的
    // 假阴性——审批人会以为策略没动过，而系统根本没读到策略。
    expect(screen.queryByText('策略未变')).not.toBeInTheDocument();
    expect(screen.queryByText(/未变/)).not.toBeInTheDocument();
  });

  it('not_applicable → 说明「不涉及比对」而不是「未变」', async () => {
    mockReview.mockResolvedValue(basePreflight({ policyComparison: 'not_applicable' }));
    renderPage();
    await waitFor(() => expect(mockGetApprovals).toHaveBeenCalled());
    await expandFirstRow();
    expect(await screen.findByText('不涉及策略比对')).toBeInTheDocument();
    expect(screen.queryByText(/未变/)).not.toBeInTheDocument();
  });

  it('未知比对取值不静默降级为「未变」，而是显式报无法识别', async () => {
    // 后端新增第五态时前端必须暴露，否则它悄悄落进某个默认分支。
    mockReview.mockResolvedValue(basePreflight({ policyComparison: 'future_state' }));
    renderPage();
    await waitFor(() => expect(mockGetApprovals).toHaveBeenCalled());
    await expandFirstRow();

    expect(await screen.findByText(/比对结果无法识别/)).toBeInTheDocument();
    expect(screen.queryByText('策略未变')).not.toBeInTheDocument();
  });
});

