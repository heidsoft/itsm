import React from 'react';
import { act, render, screen, waitFor } from '@testing-library/react';
import { IncidentAPI } from '@/lib/api/';
import IncidentDetail from '../IncidentDetail';

let mockId = '1';
const mockRouter = { push: jest.fn() };
jest.mock('next/navigation', () => ({
  useParams: () => ({ id: mockId }),
  useRouter: () => mockRouter,
}));
jest.mock('@/lib/api/', () => ({ IncidentAPI: { getIncident: jest.fn(), getRootCauseAnalysis: jest.fn().mockResolvedValue(null), getImpactAssessment: jest.fn().mockResolvedValue(null), getIncidentClassification: jest.fn().mockResolvedValue(null), } }));
jest.mock('@/lib/hooks/useUserListQuery', () => ({ useUserPickerQuery: () => ({ users: [], isLoading: false }) }));
jest.mock('@/lib/hooks/useErrorHandler', () => ({ useErrorHandler: () => ({ handleError: jest.fn() }) }));

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>(done => { resolve = done; });
  return { promise, resolve };
}
const record = (id: number, title: string): Awaited<ReturnType<typeof IncidentAPI.getIncident>> => ({
  id, title, description: '详情', status: 'closed', priority: 'medium',
  type: 'incident', severity: 'low', source: 'manual',
  createdAt: '2026-11-22T00:00:00Z', updatedAt: '2026-11-22T00:00:00Z',
});
beforeEach(() => {
  mockId = '1';
});
it('旧 id 慢响应后到不得覆盖新 id 数据', async () => {
  const old = deferred<ReturnType<typeof record>>();
  jest.mocked(IncidentAPI.getIncident).mockImplementation(id =>
    (id === 1 ? old.promise : Promise.resolve(record(2, '新对象乙'))) as ReturnType<typeof IncidentAPI.getIncident>);
  const { rerender } = render(<IncidentDetail id={mockId} />);
  expect(IncidentAPI.getIncident).toHaveBeenCalledWith(1);
  mockId = '2';
  rerender(<IncidentDetail id={mockId} />);
  expect(await screen.findByText('新对象乙')).toBeInTheDocument();
  await act(async () => { old.resolve(record(1, '旧对象甲')); });
  expect(screen.queryByText('旧对象甲')).not.toBeInTheDocument();
  expect(screen.getByText('新对象乙')).toBeInTheDocument();
});

it('旧请求 finally 不得提前关闭新 id 的 loading', async () => {
  const old = deferred<ReturnType<typeof record>>();
  const next = deferred<ReturnType<typeof record>>();
  jest.mocked(IncidentAPI.getIncident).mockImplementation(id =>
    (id === 1 ? old.promise : next.promise) as ReturnType<typeof IncidentAPI.getIncident>);
  const { rerender, container } = render(<IncidentDetail id={mockId} />);
  mockId = '2';
  rerender(<IncidentDetail id={mockId} />);
  await act(async () => { old.resolve(record(1, '旧对象甲')); });
  expect(container.querySelector('.ant-skeleton')).toBeInTheDocument();
  expect(screen.queryByText('旧对象甲')).not.toBeInTheDocument();
  await act(async () => { next.resolve(record(2, '新对象乙')); });
  expect(screen.getByText('新对象乙')).toBeInTheDocument();
});

it('旧 id 的分析响应不得覆盖新 id 的根因、影响与分类', async () => {
  const root = deferred<{ rootCause: string }>();
  const impact = deferred<{ assessmentNotes: string }>();
  const classification = deferred<{ category: string }>();
  jest.mocked(IncidentAPI.getIncident).mockImplementation(id =>
    Promise.resolve(record(id, id === 1 ? '旧对象甲' : '新对象乙')) as ReturnType<typeof IncidentAPI.getIncident>);
  jest.mocked(IncidentAPI.getRootCauseAnalysis).mockImplementation(id =>
    (id === 1 ? root.promise : Promise.resolve({ rootCause: '新根因' })) as ReturnType<typeof IncidentAPI.getRootCauseAnalysis>);
  jest.mocked(IncidentAPI.getImpactAssessment).mockImplementation(id =>
    (id === 1 ? impact.promise : Promise.resolve({ assessmentNotes: '新影响' })) as ReturnType<typeof IncidentAPI.getImpactAssessment>);
  jest.mocked(IncidentAPI.getIncidentClassification).mockImplementation(id =>
    (id === 1 ? classification.promise : Promise.resolve({ category: '新分类' })) as ReturnType<typeof IncidentAPI.getIncidentClassification>);
  const { rerender } = render(<IncidentDetail id={mockId} />);
  await waitFor(() => expect(IncidentAPI.getRootCauseAnalysis).toHaveBeenCalledWith(1));
  mockId = '2';
  rerender(<IncidentDetail id={mockId} />);
  expect(await screen.findByText('新根因')).toBeInTheDocument();
  expect(screen.getByText('新影响')).toBeInTheDocument();
  expect(screen.getByText('新分类')).toBeInTheDocument();
  await act(async () => {
    root.resolve({ rootCause: '旧根因' });
    impact.resolve({ assessmentNotes: '旧影响' });
    classification.resolve({ category: '旧分类' });
  });
  for (const text of ['旧根因', '旧影响', '旧分类']) {
    expect(screen.queryByText(text)).not.toBeInTheDocument();
  }
  for (const text of ['新根因', '新影响', '新分类']) {
    expect(screen.getByText(text)).toBeInTheDocument();
  }
});
