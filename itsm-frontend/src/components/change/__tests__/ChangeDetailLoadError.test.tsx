/**
 * 回归锁：变更详情加载失败（5xx/超时）不能伪装成 404「未找到」，必须给重试入口。
 */
import React from 'react';
import { render, screen, waitFor } from '@/lib/test-utils';

const mockGetChange = jest.fn();
let mockShouldFail = false;

jest.mock('next/navigation', () => ({
  useParams: () => ({ id: '9' }),
  useRouter: () => ({ push: jest.fn() }),
}));

jest.mock('@/lib/api/', () => ({
  ChangeApi: {
    getChange: (...args: unknown[]) => {
      if (mockShouldFail) return Promise.reject(new Error('500'));
      return mockGetChange(...args);
    },
    getApprovalSummary: jest.fn().mockResolvedValue({ chain: [] }),
    getChangeApprovals: jest.fn().mockResolvedValue([]),
    getRiskAssessment: jest.fn().mockResolvedValue(null),
    getChangeChecklist: jest.fn().mockResolvedValue([]),
    getRollbackPlan: jest.fn().mockResolvedValue(null),
    failChange: jest.fn(), rollbackChange: jest.fn(),
    completeImplementation: jest.fn(), startImplementation: jest.fn(),
    scheduleChange: jest.fn(), closeChange: jest.fn(), cancelChange: jest.fn(),
    updateChange: jest.fn(), approveChange: jest.fn(), rejectChange: jest.fn(),
    submitForApproval: jest.fn(),
  },
}));

import ChangeDetail from '../ChangeDetail';

const okChange = {
  id: 9, changeNumber: 'CHG-9', title: '索引变更', description: '补索引',
  type: 'normal', priority: 'medium', status: 'in_progress', riskLevel: 'low',
  impactScope: 'low', createdBy: 1, createdByName: '张三', tenantId: 1,
  affectedCis: [], relatedTickets: [], createdAt: '2026-10-22T09:00:00Z', updatedAt: '2026-10-22T09:00:00Z',
};

describe('ChangeDetail 加载错误态', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockShouldFail = false;
    mockGetChange.mockResolvedValue(okChange);
  });

  it('加载失败显示错误态与「重试」，不得显示 404；重试后恢复', async () => {
    mockShouldFail = true;
    render(<ChangeDetail />);

    await waitFor(() => expect(screen.getByText('加载变更详情失败')).toBeInTheDocument());
    expect(screen.queryByText('未找到指定的变更')).toBeNull();

    mockShouldFail = false;
    const retry = screen.getByRole('button', { name: /重\s*试/ });
    retry.click();

    // 重试后渲染真实详情（mockChange 已恢复）；错误态的 Result 按钮消失
    expect(await screen.findByText('索引变更')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /重\s*试/ })).toBeNull();
  });
});
