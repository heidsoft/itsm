import React from 'react';
import { act, fireEvent, render, screen } from '@testing-library/react';
import dayjs from 'dayjs';
import { ChangeApi } from '@/lib/api/change-api';
import ChangesPage from '../page';

const mockRouter = { push: jest.fn(), replace: jest.fn() };
jest.mock('next/navigation', () => ({
  useRouter: () => mockRouter,
  useSearchParams: () => new URLSearchParams(),
}));
jest.mock('@/lib/hooks/use-permissions', () => ({
  useOperationPermissions: () => ({ change: { canCreate: () => true } }),
}));
jest.mock('@/lib/api/change-api', () => ({
  ChangeApi: { getChanges: jest.fn(), getChangeStats: jest.fn(), getCalendar: jest.fn() },
}));
jest.mock('@/components/layout/BusinessPageTemplate', () => ({
  BusinessPageTemplate: ({ children, stats, statsLoading, extraActions }: {
    children: React.ReactNode;
    stats: { value: number }[];
    statsLoading: boolean;
    extraActions: { key: string; onClick: () => void }[];
  }) => <div>
    <div data-testid="stats">{stats[0].value}</div>
    <div data-testid="stats-loading">{String(statsLoading)}</div>
    <button onClick={extraActions.find(action => action.key === 'refresh')?.onClick}>刷新数据</button>
    {children}
  </div>,
}));
jest.mock('@/components/change/ChangeList', () => ({
  __esModule: true,
  default: ({ items, loading }: { items: { title: string }[]; loading: boolean }) =>
    <div data-testid="rows">{String(loading)}:{items.map(item => item.title).join(',')}</div>,
}));
jest.mock('@/components/business/UnifiedKanbanBoard', () => ({
  UnifiedKanbanBoard: ({ items, loading }: { items: { title: string }[]; loading: boolean }) =>
    <div data-testid="rows">{String(loading)}:{items.map(item => item.title).join(',')}</div>,
}));
jest.mock('antd', () => {
  const actual = jest.requireActual('antd');
  return {
    ...actual,
    Calendar: () => null,
  };
});

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>(done => { resolve = done; });
  return { promise, resolve };
}
const rows = (title: string) => ({
  items: [{ id: 1, title, status: 'draft' }], total: 1,
});
const stats = (total: number) => ({ total, pending: 0, inProgress: 0, completed: 0 });
const calendar = (title: string) => ({
  items: [{ id: 1, title, status: 'draft', plannedStart: dayjs().format('YYYY-MM-DD') }],
});

beforeEach(() => {
  jest.mocked(ChangeApi.getChanges).mockResolvedValue(rows('默认') as Awaited<ReturnType<typeof ChangeApi.getChanges>>);
  jest.mocked(ChangeApi.getChangeStats).mockResolvedValue(stats(0) as Awaited<ReturnType<typeof ChangeApi.getChangeStats>>);
  jest.mocked(ChangeApi.getCalendar).mockResolvedValue(calendar('默认日历') as Awaited<ReturnType<typeof ChangeApi.getCalendar>>);
});

it('列表慢响应后到不得覆盖看板数据', async () => {
  const old = deferred<ReturnType<typeof rows>>();
  jest.mocked(ChangeApi.getChanges)
    .mockReturnValueOnce(old.promise as ReturnType<typeof ChangeApi.getChanges>)
    .mockResolvedValueOnce(rows('新看板') as Awaited<ReturnType<typeof ChangeApi.getChanges>>);
  render(<ChangesPage />);
  expect(ChangeApi.getChanges).toHaveBeenCalledTimes(1);
  fireEvent.click(screen.getByRole('button', { name: /看板/ }));
  expect(await screen.findByText('false:新看板')).toBeInTheDocument();
  await act(async () => { old.resolve(rows('旧列表')); });
  expect(screen.getByTestId('rows')).toHaveTextContent('false:新看板');
});

it('统计慢响应后到不得覆盖新统计', async () => {
  const old = deferred<ReturnType<typeof stats>>();
  jest.mocked(ChangeApi.getChangeStats)
    .mockReturnValueOnce(old.promise as ReturnType<typeof ChangeApi.getChangeStats>)
    .mockResolvedValueOnce(stats(22) as Awaited<ReturnType<typeof ChangeApi.getChangeStats>>);
  render(<ChangesPage />);
  fireEvent.click(screen.getByRole('button', { name: '刷新数据' }));
  await act(async () => {});
  expect(screen.getByTestId('stats')).toHaveTextContent('22');
  await act(async () => { old.resolve(stats(11)); });
  expect(screen.getByTestId('stats')).toHaveTextContent('22');
});

it('日历慢响应后到不得覆盖新日历', async () => {
  const old = deferred<ReturnType<typeof calendar>>();
  jest.mocked(ChangeApi.getCalendar)
    .mockReturnValueOnce(old.promise as ReturnType<typeof ChangeApi.getCalendar>)
    .mockResolvedValueOnce(calendar('新日历乙') as Awaited<ReturnType<typeof ChangeApi.getCalendar>>);
  render(<ChangesPage />);
  fireEvent.click(screen.getByRole('button', { name: /日历/ }));
  expect(ChangeApi.getCalendar).toHaveBeenCalledTimes(1);
  fireEvent.click(screen.getByRole('button', { name: '刷新数据' }));
  expect(await screen.findByText('新日历乙')).toBeInTheDocument();
  await act(async () => { old.resolve(calendar('旧日历甲')); });
  expect(screen.queryByText('旧日历甲')).not.toBeInTheDocument();
  expect(screen.getByText('新日历乙')).toBeInTheDocument();
});

it('旧列表 finally 不得关闭正在加载的看板', async () => {
  const old = deferred<ReturnType<typeof rows>>();
  const next = deferred<ReturnType<typeof rows>>();
  jest.mocked(ChangeApi.getChanges)
    .mockReturnValueOnce(old.promise as ReturnType<typeof ChangeApi.getChanges>)
    .mockReturnValueOnce(next.promise as ReturnType<typeof ChangeApi.getChanges>);
  render(<ChangesPage />);
  fireEvent.click(screen.getByRole('button', { name: /看板/ }));
  await act(async () => { old.resolve(rows('旧列表')); });
  expect(screen.getByTestId('rows')).toHaveTextContent('true:');
  expect(screen.getByTestId('rows')).not.toHaveTextContent('旧列表');
  await act(async () => { next.resolve(rows('新看板')); });
  expect(screen.getByTestId('rows')).toHaveTextContent('false:新看板');
});
