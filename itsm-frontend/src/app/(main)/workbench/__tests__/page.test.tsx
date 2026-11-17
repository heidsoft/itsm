import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
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


/**
 * 工作台搜索框接通回归：
 * 此前页面未传 onSearch，模板搜索框输入只进本地 state，列表永远不过滤——
 * 用户感知「搜索框坏了」。修复后：输入 → 防抖(300ms) → queryWorkbench 带 keyword。
 */
describe('工作台搜索框接通 keyword 链路', () => {
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

  it('输入关键字后（防抖）请求携带 keyword', async () => {
    render(
      <App>
        <WorkbenchPage />
      </App>
    );
    await waitFor(() => expect(mockQueryWorkbench).toHaveBeenCalled());

    const input = screen.getByPlaceholderText('搜索标题…');
    fireEvent.change(input, { target: { value: '数据库故障' } });

    await waitFor(
      () =>
        expect(mockQueryWorkbench).toHaveBeenCalledWith(
          expect.objectContaining({ keyword: '数据库故障' })
        ),
      { timeout: 2000 }
    );
  });

  it('清空关键字后请求不再携带 keyword', async () => {
    render(
      <App>
        <WorkbenchPage />
      </App>
    );
    await waitFor(() => expect(mockQueryWorkbench).toHaveBeenCalled());

    const input = screen.getByPlaceholderText('搜索标题…');
    fireEvent.change(input, { target: { value: '数据库故障' } });
    await waitFor(
      () =>
        expect(mockQueryWorkbench).toHaveBeenCalledWith(
          expect.objectContaining({ keyword: '数据库故障' })
        ),
      { timeout: 2000 }
    );

    fireEvent.change(input, { target: { value: '' } });
    await waitFor(
      () =>
        expect(mockQueryWorkbench).toHaveBeenLastCalledWith(
          expect.not.objectContaining({ keyword: '数据库故障' })
        ),
      { timeout: 2000 }
    );
  });
});

/**
 * 工作台「筛选」按钮回归：
 * 此前 onToggle 为空实现、visible 写死 true，按钮点击无反应。修复后可折叠筛选面板。
 */
describe('工作台筛选按钮可折叠面板', () => {
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

  it('点击「筛选」隐藏面板，再点恢复', async () => {
    render(
      <App>
        <WorkbenchPage />
      </App>
    );
    await waitFor(() => expect(mockQueryWorkbench).toHaveBeenCalled());

    // 面板初始可见（含阶段筛选文案）
    expect(screen.getByText('生命周期阶段')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: '筛选' }));
    expect(screen.queryByText('生命周期阶段')).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: '筛选' }));
    expect(screen.getByText('生命周期阶段')).toBeInTheDocument();
  });
});
