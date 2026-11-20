/**
 * 回归锁：变更详情深链 /changes/1 → /changes/2 时，旧 id 的审批链慢响应
 * 不得覆盖新 id 的审批链（页面级 approvals 无 generation 守卫导致的串页）。
 */
import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';

const mockGetChangeApprovals = jest.fn();
let mockRouteId: string = '1';

jest.mock('next/navigation', () => ({ useParams: () => ({ id: mockRouteId }) }));

jest.mock('@/lib/api/change-api', () => ({
  ChangeApi: {
    getChangeApprovals: (...args: unknown[]) => mockGetChangeApprovals(...args),
  },
}));

// ChangeDetail 本体与本测试无关，替身避免连带拉取
jest.mock('@/components/change/ChangeDetail', () => ({
  __esModule: true,
  default: () => <div>change-detail</div>,
}));

// 把审批链渲染成可断言的纯文本标记，隔离页面级 fetch/race 逻辑
jest.mock('@/components/business/detail-tabs', () => ({
  ApprovalTimeline: ({ approvals }: { approvals: { id: number; approverName: string }[] }) => (
    <div data-testid="approvals">{approvals.map(a => a.approverName).join(',') || 'empty'}</div>
  ),
  HistoryTimeline: () => null,
  fetchAuditLogHistory: jest.fn().mockResolvedValue([]),
}));

jest.mock('@/lib/i18n/useI18n', () => ({ useI18n: () => ({ t: (k: string) => k }) }));

import ChangeDetailPage from '../page';

describe('变更详情审批链深链不串页', () => {
  beforeEach(() => jest.clearAllMocks());

  it('旧 id 审批请求后到，不得覆盖新 id 的审批链', async () => {
    let resolveId1: (v: unknown) => void = () => {};
    mockGetChangeApprovals.mockImplementation((id: number) => {
      if (id === 1) return new Promise(resolve => { resolveId1 = resolve; });
      return Promise.resolve([{ id: 20, approverName: '新审批人', status: 'approved' }]);
    });

    mockRouteId = '1';
    const { rerender } = render(<ChangeDetailPage />);

    // 切到 id=2，同一组件实例
    mockRouteId = '2';
    rerender(<ChangeDetailPage />);

    // 新 id 的审批链应渲染
    await waitFor(() => expect(screen.getByTestId('approvals')).toHaveTextContent('新审批人'));

    // 旧 id 慢响应到达，必须被 generation 守卫丢弃
    resolveId1([{ id: 10, approverName: '旧审批人', status: 'approved' }]);
    await waitFor(() => expect(mockGetChangeApprovals).toHaveBeenCalledWith(1));

    expect(screen.getByTestId('approvals')).not.toHaveTextContent('旧审批人');
    expect(screen.getByTestId('approvals')).toHaveTextContent('新审批人');
  });
});
