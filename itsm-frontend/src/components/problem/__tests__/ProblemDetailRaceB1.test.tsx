import React from 'react';
import { act, render, screen } from '@testing-library/react';
import { ProblemApi } from '@/lib/api/';
import ProblemDetail from '../ProblemDetail';

let mockId = '1';
const mockRouter = { push: jest.fn() };
jest.mock('next/navigation', () => ({
  useParams: () => ({ id: mockId }),
  useRouter: () => mockRouter,
}));
jest.mock('@/lib/api/', () => ({ ProblemApi: { getProblem: jest.fn(),  } }));
jest.mock('@/lib/api/kedb-api', () => ({ KEDBApi: {} }));
jest.mock('../ProblemInvestigationTab', () => ({ __esModule: true, default: () => null }));
jest.mock('../BasicInfoCard', () => ({ __esModule: true, default: () => null }));

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>(done => { resolve = done; });
  return { promise, resolve };
}
const record = (id: number, title: string): Awaited<ReturnType<typeof ProblemApi.getProblem>> => ({
  id, title, description: '详情', status: 'closed', priority: 'medium',
  category: '', rootCause: '', workaround: '', resolution: '', impact: '', createdBy: 1, tenantId: 1,
  createdAt: '2026-11-22T00:00:00Z', updatedAt: '2026-11-22T00:00:00Z',
});
beforeEach(() => {
  mockId = '1';
});
it('旧 id 慢响应后到不得覆盖新 id 数据', async () => {
  const old = deferred<ReturnType<typeof record>>();
  jest.mocked(ProblemApi.getProblem).mockImplementation(id =>
    (id === 1 ? old.promise : Promise.resolve(record(2, '新对象乙'))) as ReturnType<typeof ProblemApi.getProblem>);
  const { rerender } = render(<ProblemDetail id={mockId} />);
  expect(ProblemApi.getProblem).toHaveBeenCalledWith(1);
  mockId = '2';
  rerender(<ProblemDetail id={mockId} />);
  expect(await screen.findByText('新对象乙')).toBeInTheDocument();
  await act(async () => { old.resolve(record(1, '旧对象甲')); });
  expect(screen.queryByText('旧对象甲')).not.toBeInTheDocument();
  expect(screen.getByText('新对象乙')).toBeInTheDocument();
});

it('旧请求 finally 不得提前关闭新 id 的 loading', async () => {
  const old = deferred<ReturnType<typeof record>>();
  const next = deferred<ReturnType<typeof record>>();
  jest.mocked(ProblemApi.getProblem).mockImplementation(id =>
    (id === 1 ? old.promise : next.promise) as ReturnType<typeof ProblemApi.getProblem>);
  const { rerender, container } = render(<ProblemDetail id={mockId} />);
  mockId = '2';
  rerender(<ProblemDetail id={mockId} />);
  await act(async () => { old.resolve(record(1, '旧对象甲')); });
  expect(container.querySelector('.ant-skeleton')).toBeInTheDocument();
  expect(screen.queryByText('旧对象甲')).not.toBeInTheDocument();
  await act(async () => { next.resolve(record(2, '新对象乙')); });
  expect(screen.getByText('新对象乙')).toBeInTheDocument();
});
