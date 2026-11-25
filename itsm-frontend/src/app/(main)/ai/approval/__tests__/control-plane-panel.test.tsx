import React from 'react';
import { render, screen, waitFor, act } from '@testing-library/react';
import { App } from 'antd';
import '@testing-library/jest-dom';

/**
 * AI 控制面概览面板的前端契约（2026-11-21）
 *
 * 这个面板是把四个真源（工具可见面 / 待批队列 / 策略覆盖 / 最近拒绝）
 * 聚合成一次请求的展示层。它自己不判定任何东西，所以测试重点不是
 * 「数字对不对」（那是后端用例的职责），而是**这四类信息在展示时
 * 会不会被悄悄合并、被失败伪装成空、或被过期响应覆盖**。
 *
 * 守四件事：
 *   ① 读失败必须显式告知，**不得**渲染成「待批 0 条」——
 *      空态与读失败在界面上都是空的，但前者是「队列干净」，
 *      后者是「不知道」。把它渲染成空态，运维会以为清空了而停止处理。
 *   ② 四种审批态**分开显示**：pendingAgentTwo 与 awaitingSecond 尤其不能合并，
 *      后者表示「已有一票、首票不放行」，合并后这批会静默卡死。
 *   ③ 策略三类**分开显示**，`配置损坏` 单独可见：
 *      它看起来配了策略但鉴权时 fail-closed 全拒，并进任一类都会让
 *      最危险的一类彻底隐身。
 *   ④ 并发刷新时只有**最后一次**响应能落地（generation 守卫）——
 *      先发后到会把新快照顶掉，界面上表现为「点了刷新数字反而回退」，
 *      且不报任何错误。
 *
 * 反向验证：把 error 分支改成渲染空态、把四态合并成单值、
 * 或去掉 seq 守卫，对应断言应转红。
 */

jest.mock('@/lib/store/auth-store', () => ({
  useAuthStoreHydration: () => {},
}));

jest.mock('@/lib/hooks/use-permissions', () => ({
  usePermissions: () => ({ hasPermission: (code: string) => code === 'ai:write' }),
}));

jest.mock('@/lib/api/ai-api', () => ({
  aiGetToolApprovals: jest.fn(),
  aiApproveTool: jest.fn(),
  aiGetRunInvocations: jest.fn(),
  aiReviewToolInvocation: jest.fn(),
  aiGetControlPlane: jest.fn(),
}));

import AIApprovalQueue from '../page';
import { aiGetToolApprovals, aiGetControlPlane } from '@/lib/api/ai-api';
import type { AgentControlPlaneOverview } from '@/lib/api/ai-api';

const mockGetApprovals = aiGetToolApprovals as jest.Mock;
const mockGetControlPlane = aiGetControlPlane as jest.Mock;

/** 构造一份完整快照；每个字段都可覆盖，便于逐字段断言。 */
const overview = (patch: Partial<AgentControlPlaneOverview> = {}): AgentControlPlaneOverview => ({
  tenantId: 1,
  role: 'admin',
  toolSurface: [],
  visibleTools: 6,
  totalTools: 14,
  queue: { pending: 2, pendingAgent: 1, pendingAgentTwo: 3, awaitingSecond: 4, total: 10 },
  policies: { total: 20, withPolicy: 15, policyFree: 4, unreadable: 1 },
  recentDenial: [],
  gatheredAt: '2026-11-21T12:00:00+08:00',
  ...patch,
});

const renderPage = async () => {
  await act(async () => {
    render(
      <App>
        <AIApprovalQueue />
      </App>
    );
  });
};

describe('AI 控制面概览面板', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockGetApprovals.mockResolvedValue({ items: [], state: 'pending' });
  });

  // ① 读失败不得伪装成空态。
  //
  // 反向验证过的形态：把 error 分支改成 `if (error) return null`，
  // 或让 catch 里 setControlPlane({ loading: false, data: 零值快照 })。
  it('读失败时显式告知，不得渲染成「0 条待批」', async () => {
    mockGetControlPlane.mockRejectedValue(new Error('后端 500'));
    await renderPage();

    await waitFor(() => {
      // 文案随产品语言治理调整：标题从引擎词「控制面状态」改为业务语义
      // 「队列概况」。断言与新文案同源，并反向锁住旧词不得复活。
      expect(screen.getByText('队列概况读取失败')).toBeInTheDocument();
    });
    // 旧引擎词不得回到用户可见文案里。
    expect(screen.queryByText('控制面状态读取失败')).not.toBeInTheDocument();
    // 关键反向断言：错误态下绝不能出现「合计 0 条」这类会被读成「队列干净」的文案。
    expect(screen.queryByText(/合计 0 条/)).not.toBeInTheDocument();
    // 并且必须说明「这不代表队列为空」，否则用户只会看到失败二字。
    expect(screen.getByText(/这不代表队列为空/)).toBeInTheDocument();
  });

  // ② 队列四态分开显示，awaitingSecond 单独可见。
  //
  // 为什么 awaitingSecond 不能并进 pendingAgentTwo：后者尚无人投第一票，
  // 前者已有一票、首票不放行。合并成「AI 待批 7」会让运维看不出有一批
  // 已经批了一半却停住了——那是最容易静默卡死的地方。
  it('四种审批态分别展示，awaiting_second 不与双人待批合并', async () => {
    mockGetControlPlane.mockResolvedValue(overview());
    await renderPage();

    await waitFor(() => {
      expect(screen.getByText(/待审批 2/)).toBeInTheDocument();
    });
    expect(screen.getByText(/AI 待批 1/)).toBeInTheDocument();
    expect(screen.getByText(/AI 待双人 3/)).toBeInTheDocument();
    expect(screen.getByText(/已一票待二审 4/)).toBeInTheDocument();
    // 存在 awaitingSecond 时必须提示要尽快安排第二人复审。
    expect(screen.getByText(/需尽快安排第二人复审/)).toBeInTheDocument();
  });

  // ③ 策略三类分开显示，配置损坏必须独立可见。
  //
  // 反向验证过的形态：把 unreadable 并进 withPolicy（显示「已配置 16」），
  // 或并进 policyFree（显示「未配置 5」）——两种都会让本用例红在
  // 「配置损坏 1」这条断言上。
  it('策略三类分别展示，配置损坏不得并入已配置或未配置', async () => {
    mockGetControlPlane.mockResolvedValue(overview());
    await renderPage();

    await waitFor(() => {
      expect(screen.getByText(/已配置 15/)).toBeInTheDocument();
    });
    expect(screen.getByText(/未配置 4/)).toBeInTheDocument();
    expect(screen.getByText(/配置损坏 1/)).toBeInTheDocument();
    // 合并后的数字一个都不该出现。
    expect(screen.queryByText(/已配置 16/)).not.toBeInTheDocument();
    expect(screen.queryByText(/未配置 5/)).not.toBeInTheDocument();
    // 配置损坏存在时必须解释后果（当前一律拒绝）。
    expect(screen.getByText(/这类权限点当前一律拒绝/)).toBeInTheDocument();
  });

  // 工具可见性必须回显当前角色。
  //
  // 同一租户同一时刻，admin 看到 14 个、普通技术员看到 6 个。
  // 不说明「按谁算的」，界面上的「可见工具数」就没有确定含义。
  it('工具可见性回显调用者角色，结尾不写成绝对结论', async () => {
    mockGetControlPlane.mockResolvedValue(overview({ role: 'technician', visibleTools: 3 }));
    await renderPage();

    await waitFor(() => {
      expect(screen.getByText(/按角色「technician」计算/)).toBeInTheDocument();
    });
    expect(screen.getByText('3 / 14')).toBeInTheDocument();
  });

  // ④ 并发刷新：只有最后一次响应能落地。
  //
  // 构造：第一次刷新返回 A（旧快照），第二次返回 B（新快照），但 A 后到。
  // 若无 seq 守卫，A 会覆盖 B —— 界面上表现为「刷新后数字回退」，
  // 且不产生任何错误，属最隐蔽的一类假数据。
  it('并发刷新时，先发后到的旧响应不得覆盖新快照', async () => {
    let resolveFirst: (v: AgentControlPlaneOverview) => void = () => {};
    let resolveSecond: (v: AgentControlPlaneOverview) => void = () => {};
    mockGetControlPlane
      .mockReturnValueOnce(new Promise(r => { resolveFirst = r; }))
      .mockReturnValueOnce(new Promise(r => { resolveSecond = r; }));

    await renderPage();

    const { container } = { container: document.body };
    // 触发第二次请求（刷新按钮在面板 extra 上）。
    const refreshLink = await waitFor(() => {
      const links = Array.from(container.querySelectorAll('a')).filter(a =>
        /刷新/.test(a.textContent ?? '')
      );
      // 面板内的刷新与页头刷新都叫「刷新」，取最后一个（面板 extra）。
      expect(links.length).toBeGreaterThan(0);
      return links[links.length - 1] as HTMLElement;
    });
    await act(async () => { refreshLink.click(); });

    // 先落地第二次（新），再落地第一次（旧）。
    await act(async () => {
      resolveSecond(overview({ queue: { pending: 0, pendingAgent: 0, pendingAgentTwo: 0, awaitingSecond: 0, total: 0 } }));
    });
    await waitFor(() => expect(screen.getByText(/合计 0 条/)).toBeInTheDocument());

    await act(async () => {
      resolveFirst(overview({ queue: { pending: 5, pendingAgent: 0, pendingAgentTwo: 0, awaitingSecond: 0, total: 5 } }));
    });

    // 迟到响应到达后，新内容必须**仍在**：这是区分「竞态已修」与
    // 「断言不出现就通过」的唯一办法（后者是假绿）。
    await waitFor(() => {
      expect(screen.getByText(/合计 0 条/)).toBeInTheDocument();
      expect(screen.queryByText(/合计 5 条/)).not.toBeInTheDocument();
    });
  });
});
