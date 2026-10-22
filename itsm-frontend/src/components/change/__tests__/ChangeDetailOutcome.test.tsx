/**
 * ChangeDetail 实施结果留痕（标记失败 / 回滚）
 *
 * 锁定的行为契约（2026-10-22）：
 *  1. `in_progress` 变更必须能看到「标记失败」与「回滚」——此前后端状态机允许
 *     failed 却没有任何入口，失败只能被报成"完成"或直接回滚；
 *  2. 两者都必须先填写原因：原因为空时确认按钮禁用（后端也会 400/1002 拒收，
 *     但让按钮先关掉能省掉一次注定失败的请求）；
 *  3. 原因随请求发出（`failChange(id, reason)`），不是空调用。
 *  4. 「实施结果说明」存在时在详情里展示。
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

const failChange = jest.fn().mockResolvedValue(undefined);
const rollbackChange = jest.fn().mockResolvedValue(undefined);

let mockChange: Record<string, unknown> = {};

jest.mock('@/lib/api/', () => ({
  ChangeApi: {
    getChange: jest.fn(() => Promise.resolve(mockChange)),
    getApprovalSummary: jest.fn().mockResolvedValue({ chain: [] }),
    getChangeApprovals: jest.fn().mockResolvedValue([]),
    getRiskAssessment: jest.fn().mockResolvedValue(null),
    getImpactAnalysis: jest.fn().mockResolvedValue(null),
    failChange: (...args: unknown[]) => failChange(...args),
    rollbackChange: (...args: unknown[]) => rollbackChange(...args),
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
  riskLevel: 'low',
  createdBy: 1,
  createdByName: '张三',
  tenantId: 1,
  implementationPlan: '在线建索引',
  rollbackPlan: 'DROP INDEX',
  affectedCis: [],
  relatedTickets: [],
  createdAt: '2026-10-22T09:00:00Z',
  updatedAt: '2026-10-22T09:00:00Z',
};

describe('ChangeDetail 实施结果留痕', () => {
  beforeEach(() => {
    failChange.mockClear();
    rollbackChange.mockClear();
  });

  it('实施中的变更提供「标记失败」，原因必填后才可提交', async () => {
    mockChange = { ...baseChange, status: 'in_progress' };
    render(<ChangeDetail />);

    const failButton = await screen.findByRole('button', { name: '标记失败' });
    await userEvent.click(failButton);

    const dialog = await screen.findByRole('dialog');
    // 原因为空：确认按钮禁用（同名的「标记失败」在页面上出现两处，先限定在弹窗内）
    const confirm = within(dialog).getByRole('button', { name: '标记失败' });
    expect(confirm).toBeDisabled();

    const textarea = within(dialog).getByRole('textbox');
    await userEvent.type(textarea, '第二步报错导致连接池耗尽');
    expect(confirm).toBeEnabled();

    await userEvent.click(confirm);

    await waitFor(() => expect(failChange).toHaveBeenCalledWith(9, '第二步报错导致连接池耗尽'));
  });

  it('回滚同样要求填写原因，且不把空原因发出去', async () => {
    mockChange = { ...baseChange, status: 'in_progress' };
    render(<ChangeDetail />);

    const rollbackButtons = await screen.findAllByRole('button', { name: /回\s*滚/ });
    await userEvent.click(rollbackButtons[0]);

    const dialog = await screen.findByRole('dialog');
    const confirm = within(dialog).getByRole('button', { name: /回\s*滚/ });
    expect(confirm).toBeDisabled();

    await userEvent.type(within(dialog).getByRole('textbox'), '窗口内锁等待超时');
    await userEvent.click(confirm);

    await waitFor(() => expect(rollbackChange).toHaveBeenCalledWith(9, '窗口内锁等待超时'));
  });

  it('已失败的变更不是死胡同：提供重新排期与回滚入口', async () => {
    mockChange = { ...baseChange, status: 'failed', outcomeNote: '连接池耗尽，已恢复' };
    render(<ChangeDetail />);

    expect(await screen.findByRole('button', { name: '重新排期' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /回\s*滚/ })).toBeInTheDocument();
    // 实施结果说明要能看到，否则失败原因只存在库里
    expect(await screen.findByText('连接池耗尽，已恢复')).toBeInTheDocument();
  });
});
