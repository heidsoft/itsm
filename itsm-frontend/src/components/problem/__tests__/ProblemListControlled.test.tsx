/**
 * 回归锁：ProblemList 受控模式下不得自己发请求（消除双数据源竞态）。
 *
 * 缺陷：嵌入 /problems 列表视图时，父级拉一份用于分页/空态，ProblemList 内部
 * 又拉一份用于表格——两次请求竞态，出现「分页总数已更新、表格仍是上一页」。
 * 受控模式（传 items）下组件应退化为纯展示，渲染父级数据、不发 getProblems。
 */
import React from 'react';
import { render, screen } from '@testing-library/react';
import ProblemList from '../ProblemList';

const mockGetProblems = jest.fn();

jest.mock('@/lib/api/', () => ({
  ProblemApi: {
    getProblems: (...args: unknown[]) => mockGetProblems(...args),
    deleteProblem: jest.fn(),
  },
}));

jest.mock('next/navigation', () => ({ useRouter: () => ({ push: jest.fn() }) }));

const problems = [
  { id: 1, title: '受控问题A', status: 'open', priority: 'high', category: '网络',
    createdAt: '2026-01-01T00:00:00Z', updatedAt: '2026-01-01T00:00:00Z' },
  { id: 2, title: '受控问题B', status: 'resolved', priority: 'low', category: '数据库',
    createdAt: '2026-01-02T00:00:00Z', updatedAt: '2026-01-02T00:00:00Z' },
] as any;

describe('ProblemList 受控模式', () => {
  beforeEach(() => jest.clearAllMocks());

  it('传入 items 时渲染父级数据且完全不调用 getProblems', async () => {
    render(
      <ProblemList
        showHeader={false}
        items={problems}
        total={42}
        loading={false}
        page={1}
        pageSize={10}
      />
    );

    // 渲染父级数据
    expect(await screen.findByText('受控问题A')).toBeInTheDocument();
    expect(screen.getByText('受控问题B')).toBeInTheDocument();

    // 关键：受控模式下不得有内部第二份请求
    expect(mockGetProblems).not.toHaveBeenCalled();
  });

  it('父级 items 更新后表格随之更新（单一数据源）', async () => {
    const { rerender } = render(
      <ProblemList showHeader={false} items={problems} total={42} loading={false} />
    );
    expect(await screen.findByText('受控问题A')).toBeInTheDocument();

    rerender(
      <ProblemList
        showHeader={false}
        items={[{ ...problems[0], id: 3, title: '父级更新后' }]}
        total={1}
        loading={false}
      />
    );
    expect(await screen.findByText('父级更新后')).toBeInTheDocument();
    expect(screen.queryByText('受控问题A')).toBeNull();
    expect(mockGetProblems).not.toHaveBeenCalled();
  });
});
