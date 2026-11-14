import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { App } from 'antd';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

import { AIInvestigationPanel } from '../AIInvestigationPanel';
import { ApiRequestError } from '@/lib/api/http-client';

// AI API 客户端：只 mock 面板消费的两个方法（ai-api.ts 其余导出与面板无关）。
const mockInvestigate = jest.fn();
const mockUpdateAudit = jest.fn();
jest.mock('@/lib/api/ai-api', () => ({
  aiInvestigateTicket: (...args: unknown[]) => mockInvestigate(...args),
  aiUpdateAuditAccepted: (...args: unknown[]) => mockUpdateAudit(...args),
}));

// http-client 保留真实实现：ApiRequestError / isPermissionDeniedError 是纯类与纯函数。
jest.mock('@/lib/api/http-client', () => jest.requireActual('@/lib/api/http-client'));

const mockPush = jest.fn();
jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: mockPush, replace: jest.fn(), back: jest.fn() }),
}));

// 后端 handlers/ai.TicketInvestigationResponse 的 camelCase 契约样例
const completedResponse = {
  ticketId: 42,
  ticketNumber: 'TK-202611-0001',
  status: 'completed',
  degraded: false,
  promptVersion: 'investigation-v1',
  model: 'test-model-1',
  confidence: 0.87,
  latencyMs: 4200,
  steps: [
    { step: 'triage' },
    { step: 'knowledge', tool: 'list_kb', hitCount: 2 },
    { step: 'cmdb', tool: 'get_ci_impact', hitCount: 1 },
    { step: 'synthesis', tool: 'llm' },
  ],
  findings: ['主库磁盘空间不足', '知识库存在同类处理记录'],
  suspectedRootCause: '日志写入量增长导致磁盘耗尽',
  impactedServices: ['订单库'],
  suggestedActions: [
    { action: 'create_ticket', description: '为日志清理创建变更工单', requiresApproval: true },
  ],
  auditId: 9,
};

const degradedResponse = {
  ticketId: 42,
  status: 'degraded',
  degraded: true,
  degradedReason: 'llm synthesis unavailable, deterministic summary only',
  promptVersion: 'investigation-v1',
  confidence: 0.82,
  steps: [{ step: 'triage' }, { step: 'knowledge', degraded: true, degradedReason: 'tool permission denied' }],
  findings: ['Triage: 关键词命中存储'],
};

const renderPanel = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <App>
        <AIInvestigationPanel ticketId={42} ticketTitle="数据库磁盘告警" />
      </App>
    </QueryClientProvider>,
  );
};

beforeEach(() => {
  mockInvestigate.mockReset();
  mockUpdateAudit.mockReset();
  mockPush.mockReset();
});

describe('AIInvestigationPanel', () => {
  it('运行调查：POST 精确路径 + 渲染结论 + create_ticket 建议带预填跳转', async () => {
    mockInvestigate.mockResolvedValue(completedResponse);
    const user = userEvent.setup();
    renderPanel();

    await user.click(screen.getByRole('button', { name: '运行 AI 调查' }));

    await waitFor(() => {
      expect(mockInvestigate).toHaveBeenCalledTimes(1);
    });
    expect(mockInvestigate).toHaveBeenCalledWith(42);

    // 结论渲染
    expect(await screen.findByText('疑似根因')).toBeInTheDocument();
    expect(screen.getByText('日志写入量增长导致磁盘耗尽')).toBeInTheDocument();
    expect(screen.getByText('主库磁盘空间不足')).toBeInTheDocument();
    expect(screen.getByText('订单库')).toBeInTheDocument();

    // 白名单动作：预填跳转（不自动提交，只导航）
    expect(screen.getByText('去建单（预填）')).toBeInTheDocument();
    fireEvent.click(screen.getByText('去建单（预填）'));
    expect(mockPush).toHaveBeenCalledTimes(1);
    const pushedUrl = mockPush.mock.calls[0][0] as string;
    expect(pushedUrl.startsWith('/tickets/create?')).toBe(true);
    expect(pushedUrl).toContain('prefillTitle=');
    expect(pushedUrl).toContain('prefillDescription=');

    // 非白名单交互形态不出现「自动执行」按钮语义
    expect(screen.getByText('需人工执行')).toBeInTheDocument();
  });

  it('degraded 结论：显示降级徽标与原因，不含 AI 独有字段', async () => {
    mockInvestigate.mockResolvedValue(degradedResponse);
    const user = userEvent.setup();
    renderPanel();

    await user.click(screen.getByRole('button', { name: '运行 AI 调查' }));

    expect(await screen.findByText('降级结果')).toBeInTheDocument();
    expect(screen.getByText('AI 综合暂不可用')).toBeInTheDocument();
    expect(
      screen.getByText(/llm synthesis unavailable, deterministic summary only/),
    ).toBeInTheDocument();
    // 降级路径无建议动作入口
    expect(screen.queryByText('去建单（预填）')).not.toBeInTheDocument();
  });

  it('权限拒绝：403 展示权限提示而非通用错误', async () => {
    mockInvestigate.mockRejectedValue(new ApiRequestError('forbidden', 403, 2003));
    const user = userEvent.setup();
    renderPanel();

    await user.click(screen.getByRole('button', { name: '运行 AI 调查' }));

    expect(await screen.findByText('无权执行 AI 调查')).toBeInTheDocument();
    expect(screen.getByText(/ai:read 与 ticket:read/)).toBeInTheDocument();
  });

  it('工单不存在：404 展示专属提示', async () => {
    mockInvestigate.mockRejectedValue(new ApiRequestError('not found', 404, 4004));
    const user = userEvent.setup();
    renderPanel();

    await user.click(screen.getByRole('button', { name: '运行 AI 调查' }));

    expect(await screen.findByText('工单不存在或已被删除')).toBeInTheDocument();
  });

  it('底账闭环：auditId 有值时反馈走 PATCH 翻转且锁定为单次', async () => {
    mockInvestigate.mockResolvedValue(completedResponse);
    mockUpdateAudit.mockResolvedValue({ id: 9, accepted: true });
    const user = userEvent.setup();
    renderPanel();

    await user.click(screen.getByRole('button', { name: '运行 AI 调查' }));
    // 先等结果完整渲染（auditId 区块是结论的最后一段，出现即代表前面各段已提交）
    await screen.findByText(/底账 #9/);

    await user.click(screen.getByRole('button', { name: /有\s*用/ }));
    await waitFor(() => {
      expect(mockUpdateAudit).toHaveBeenCalledTimes(1);
    });
    expect(mockUpdateAudit).toHaveBeenCalledWith(9, true);

    // 翻转后锁定：按钮消失，显示已反馈态
    expect(await screen.findByText('已标记有用')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /有\s*用/ })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /不\s*准/ })).not.toBeInTheDocument();
  }, 15000);

  it('步骤时间线：四步齐全且降级步骤有标记', async () => {
    mockInvestigate.mockResolvedValue(completedResponse);
    const user = userEvent.setup();
    renderPanel();

    await user.click(screen.getByRole('button', { name: '运行 AI 调查' }));

    expect(await screen.findByText('调查步骤')).toBeInTheDocument();
    expect(screen.getByText('智能分诊')).toBeInTheDocument();
    expect(screen.getByText('知识库检索')).toBeInTheDocument();
    expect(screen.getByText('CMDB 影响分析')).toBeInTheDocument();
    expect(screen.getByText('AI 综合')).toBeInTheDocument();
    // 成功路径步骤不带降级标记
    expect(screen.queryByText('降级')).not.toBeInTheDocument();
  });

  it('degraded 步骤渲染降级标记', async () => {
    mockInvestigate.mockResolvedValue(degradedResponse);
    const user = userEvent.setup();
    renderPanel();

    await user.click(screen.getByRole('button', { name: '运行 AI 调查' }));

    expect(await screen.findByText('降级')).toBeInTheDocument();
  });
});
