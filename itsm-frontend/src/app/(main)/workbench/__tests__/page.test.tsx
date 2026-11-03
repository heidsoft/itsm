import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import '@testing-library/jest-dom';

/**
 * 工作台（/workbench）分页文案回归（D9）：
 * 中文界面下分页总数必须走 i18n「共 N 条记录」，不得出现「N items」。
 */

jest.mock('@/lib/user-preferences', () => ({
  userPreferences: {
    get: () => ({ language: 'zh-CN' }),
    subscribe: () => () => {},
  },
}));

const mockQueryWorkbench = jest.fn();

jest.mock('@/lib/api/workbench-api', () => ({
  queryWorkbench: (...args: unknown[]) => mockQueryWorkbench(...args),
}));

jest.mock('@/lib/store/auth-store', () => ({
  useAuthStore: (selector: (s: { user: { id: number } }) => unknown) =>
    selector({ user: { id: 1 } }),
}));

jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: jest.fn() }),
}));

import WorkbenchPage from '../page';

const ticketItem = {
  id: 1,
  recordType: 'ticket',
  title: '测试工单',
  priority: 'high',
  status: 'open',
  phase: 'active',
  tenantId: 1,
  createdAt: '2026-10-31T01:00:00Z',
  updatedAt: '2026-10-31T01:00:00Z',
};

describe('工作台分页文案（D9）', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockQueryWorkbench.mockResolvedValue({
      items: [ticketItem],
      total: 1,
      page: 1,
      pageSize: 20,
      totalPages: 1,
    });
  });

  it('分页总数显示「共 N 条记录」，不出现英文 items', async () => {
    render(
      <App>
        <WorkbenchPage />
      </App>
    );
    await waitFor(() => expect(mockQueryWorkbench).toHaveBeenCalled());

    expect(await screen.findByText('共 1 条记录')).toBeInTheDocument();
    expect(document.body.textContent || '').not.toMatch(/\bitems\b/);
  });
});

/**
 * 工作台默认阶段筛选（D18）：
 * 旧实现默认勾选 draft/submitted/active（白名单），会把 phase=unknown 与未来新增阶段
 * 的单据静默隐藏。现改为「未关闭」语义：默认不传 phase、改传 excludePhase=closed,resolved，
 * 终态被排除、其余（含 unknown）一律保留。
 */
describe('工作台默认阶段筛选为「未关闭」（D18）', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockQueryWorkbench.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      pageSize: 20,
      totalPages: 0,
    });
  });

  it('默认请求携带 excludePhase=closed,resolved，且不携带 phase', async () => {
    render(
      <App>
        <WorkbenchPage />
      </App>
    );
    await waitFor(() => expect(mockQueryWorkbench).toHaveBeenCalled());

    const params = mockQueryWorkbench.mock.calls[0][0] as {
      phase?: string[];
      excludePhase?: string[];
    };
    expect(params.phase).toBeUndefined();
    expect(params.excludePhase).toEqual(['closed', 'resolved']);
  });
});

