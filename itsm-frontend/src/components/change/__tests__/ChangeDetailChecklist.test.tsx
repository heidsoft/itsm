/**
 * ChangeDetail 回滚清单（失败可回退的可验证形态）
 *
 * 锁定的行为契约（2026-10-22，链路 C 上线门禁）：
 *  1. 清单还有未确认步骤时，「回滚」按钮禁用，并说明还差几步——
 *     后端同样会 422，前端先把话说在前面，避免点完才收到错误；
 *  2. 清单全部确认后按钮恢复可用；
 *  3. 逐项确认按状态分别调用 done / skipped / failed，跳过与失败必须先写依据
 *     （没有依据的"跳过"在审计里与"漏做"无法区分）；
 *  4. 编排清单时从既有回滚计划预填步骤，保存时按行拆成数组发出。
 */

import React from 'react';
import { render, screen, waitFor, within } from '@/lib/test-utils';
import userEvent from '@testing-library/user-event';

jest.mock('next/navigation', () => ({
  useParams: () => ({ id: '9' }),
  useRouter: () => ({ push: jest.fn() }),
  usePathname: () => '/changes/9',
  useSearchParams: () => new URLSearchParams(),
}));

const resolveRollbackStep = jest.fn().mockResolvedValue(undefined);
const replaceRollbackSteps = jest.fn().mockResolvedValue([]);

let mockChange: Record<string, unknown> = {};

jest.mock('@/lib/api/', () => ({
  ChangeApi: {
    getChange: jest.fn(() => Promise.resolve(mockChange)),
    getApprovalSummary: jest.fn().mockResolvedValue({ chain: [] }),
    getChangeApprovals: jest.fn().mockResolvedValue([]),
    getRiskAssessment: jest.fn().mockResolvedValue(null),
    getImpactAnalysis: jest.fn().mockResolvedValue(null),
    resolveRollbackStep: (...args: unknown[]) => resolveRollbackStep(...args),
    replaceRollbackSteps: (...args: unknown[]) => replaceRollbackSteps(...args),
    failChange: jest.fn(),
    rollbackChange: jest.fn(),
    completeImplementation: jest.fn(),
    startImplementation: jest.fn(),
    scheduleChange: jest.fn(),
    closeChange: jest.fn(),
    cancelChange: jest.fn(),
    updateChange: jest.fn(),
    approveChange: jest.fn(),
    rejectChange: jest.fn(),
    submitForApproval: jest.fn(),
  },
}));

import ChangeDetail from '../ChangeDetail';

const baseChange = {
  id: 9,
  changeNumber: 'CHG-20261022-0001',
  title: '数据库索引变更',
  description: '为订单表补索引',
  justification: '查询超时',
  type: 'normal',
  priority: 'medium',
  impactScope: 'low',
  riskLevel: 'high',
  createdBy: 1,
  createdByName: '张三',
  tenantId: 1,
  implementationPlan: '在线建索引',
  rollbackPlan: 'DROP INDEX idx_orders_created',
  affectedCis: [],
  relatedTickets: [],
  createdAt: '2026-10-22T09:00:00Z',
  updatedAt: '2026-10-22T09:00:00Z',
};

const pendingSteps = [
  {
    id: 101,
    stepNo: 1,
    description: '摘除 SLB 流量',
    status: 'done' as const,
    resolved: true,
    resolvedByName: '李四',
    resolvedAt: '2026-10-22T10:00:00Z',
  },
  {
    id: 102,
    stepNo: 2,
    description: '回滚应用镜像到上一版本',
    status: 'pending' as const,
    resolved: false,
  },
];

// Tab 内容默认懒渲染：先切到「回滚清单」才拿得到步骤。
async function openChecklistTab() {
  const tab = await screen.findByRole('tab', { name: '回滚清单' });
  await userEvent.click(tab);
}

describe('ChangeDetail 回滚清单', () => {
  beforeEach(() => {
    resolveRollbackStep.mockClear();
    replaceRollbackSteps.mockClear();
  });

  it('清单未走完时禁用「回滚」并说明还差几步', async () => {
    mockChange = { ...baseChange, status: 'in_progress', rollbackSteps: pendingSteps };
    render(<ChangeDetail />);

    const rollbackButtons = await screen.findAllByRole('button', { name: /回\s*滚/ });
    expect(rollbackButtons[0]).toBeDisabled();

    await openChecklistTab();
    expect(await screen.findByText(/还有 1 步未确认/)).toBeInTheDocument();
  });

  it('清单全部确认后「回滚」恢复可用', async () => {
    mockChange = {
      ...baseChange,
      status: 'in_progress',
      rollbackSteps: pendingSteps.map(step => ({ ...step, status: 'skipped', resolved: true })),
    };
    render(<ChangeDetail />);

    const rollbackButtons = await screen.findAllByRole('button', { name: /回\s*滚/ });
    expect(rollbackButtons[0]).toBeEnabled();
  });

  it('点「已完成」直接记录 done，不弹备注框', async () => {
    mockChange = { ...baseChange, status: 'in_progress', rollbackSteps: pendingSteps };
    render(<ChangeDetail />);
    await openChecklistTab();

    const doneButtons = await screen.findAllByRole('button', { name: '已完成' });
    await userEvent.click(doneButtons[0]);

    await waitFor(() => expect(resolveRollbackStep).toHaveBeenCalledWith(9, 102, 'done', undefined));
  });

  it('标记「不适用」必须先写依据', async () => {
    mockChange = { ...baseChange, status: 'in_progress', rollbackSteps: pendingSteps };
    render(<ChangeDetail />);
    await openChecklistTab();

    const skippedButtons = await screen.findAllByRole('button', { name: /不\s*适\s*用/ });
    await userEvent.click(skippedButtons[0]);

    const dialog = await screen.findByRole('dialog');
    const confirm = within(dialog).getByRole('button', { name: /确\s*定/ });
    expect(confirm).toBeDisabled();

    await userEvent.type(within(dialog).getByRole('textbox'), '本次未新增临时配置，无需清理');
    await userEvent.click(confirm);

    await waitFor(() =>
      expect(resolveRollbackStep).toHaveBeenCalledWith(9, 102, 'skipped', '本次未新增临时配置，无需清理')
    );
  });

  it('编排清单时从回滚计划预填步骤，保存按行拆成数组', async () => {
    mockChange = {
      ...baseChange,
      status: 'scheduled',
      rollbackPlan: '1. 摘除 SLB 流量\n2. 回滚镜像',
      rollbackSteps: [],
    };
    render(<ChangeDetail />);
    await openChecklistTab();

    await userEvent.click(await screen.findByRole('button', { name: '编排回滚清单' }));

    const dialog = await screen.findByRole('dialog');
    const textarea = within(dialog).getByRole('textbox') as HTMLTextAreaElement;
    // 预填时去掉行首序号，清单里不该带 "1." 这种噪音
    expect(textarea.value).toBe('摘除 SLB 流量\n回滚镜像');

    await userEvent.click(within(dialog).getByRole('button', { name: /保\s*存/ }));

    await waitFor(() =>
      expect(replaceRollbackSteps).toHaveBeenCalledWith(9, ['摘除 SLB 流量', '回滚镜像'])
    );
  });

  it('已回滚的变更不再提供步骤确认按钮（清单是审计事实）', async () => {
    mockChange = {
      ...baseChange,
      status: 'rolled_back',
      rollbackSteps: [{ ...pendingSteps[1], status: 'failed', resolved: true, note: '回滚卡住' }],
    };
    render(<ChangeDetail />);
    await openChecklistTab();

    expect(await screen.findByText('回滚卡住')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '已完成' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '撤销确认' })).not.toBeInTheDocument();
  });
});
