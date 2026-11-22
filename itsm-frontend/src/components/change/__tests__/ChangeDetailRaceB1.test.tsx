import React from 'react';
import { act, fireEvent, render, screen } from '@testing-library/react';
import { ChangeApi } from '@/lib/api/';
import ChangeDetail from '../ChangeDetail';

let mockId = '1';
const mockRouter = { push: jest.fn() };
jest.mock('next/navigation', () => ({
  useParams: () => ({ id: mockId }),
  useRouter: () => mockRouter,
}));
jest.mock('@/lib/api/', () => ({ ChangeApi: { getChange: jest.fn(), getApprovalSummary: jest.fn().mockResolvedValue({ chain: [] }), getRiskAssessment: jest.fn().mockResolvedValue(null), } }));
jest.mock('../ChangeRiskAssessment', () => ({ __esModule: true, default: ({ initialData }: { initialData?: { riskDescription?: string } }) => <div>{initialData?.riskDescription}</div> }));
jest.mock('../ChangeCMDBImpactPanel', () => ({ __esModule: true, default: () => null }));
jest.mock('../ChangeRollbackPlan', () => ({ __esModule: true, default: () => null }));
jest.mock('@/components/common/UserSelect', () => ({ UserSelect: () => null }));

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>(done => { resolve = done; });
  return { promise, resolve };
}
const record = (id: number, title: string): Awaited<ReturnType<typeof ChangeApi.getChange>> => ({
  id, title, description: '详情', status: 'closed', priority: 'medium',
  type: 'normal', riskLevel: 'low', impactScope: 'low', justification: '', createdBy: 1, createdByName: '管理员', tenantId: 1, implementationPlan: '', rollbackPlan: '', affectedCis: [], relatedTickets: [],
  createdAt: '2026-11-22T00:00:00Z', updatedAt: '2026-11-22T00:00:00Z',
});
beforeEach(() => {
  mockId = '1';
});
it('旧 id 慢响应后到不得覆盖新 id 数据', async () => {
  const old = deferred<ReturnType<typeof record>>();
  jest.mocked(ChangeApi.getChange).mockImplementation(id =>
    (id === 1 ? old.promise : Promise.resolve(record(2, '新对象乙'))) as ReturnType<typeof ChangeApi.getChange>);
  const { rerender } = render(<ChangeDetail />);
  expect(ChangeApi.getChange).toHaveBeenCalledWith(1);
  mockId = '2';
  rerender(<ChangeDetail />);
  expect(await screen.findByText('新对象乙')).toBeInTheDocument();
  await act(async () => { old.resolve(record(1, '旧对象甲')); });
  expect(screen.queryByText('旧对象甲')).not.toBeInTheDocument();
  expect(screen.getByText('新对象乙')).toBeInTheDocument();
});

it('旧请求 finally 不得提前关闭新 id 的 loading', async () => {
  const old = deferred<ReturnType<typeof record>>();
  const next = deferred<ReturnType<typeof record>>();
  jest.mocked(ChangeApi.getChange).mockImplementation(id =>
    (id === 1 ? old.promise : next.promise) as ReturnType<typeof ChangeApi.getChange>);
  const { rerender, container } = render(<ChangeDetail />);
  mockId = '2';
  rerender(<ChangeDetail />);
  await act(async () => { old.resolve(record(1, '旧对象甲')); });
  expect(container.querySelector('.ant-skeleton')).toBeInTheDocument();
  expect(screen.queryByText('旧对象甲')).not.toBeInTheDocument();
  await act(async () => { next.resolve(record(2, '新对象乙')); });
  expect(screen.getByText('新对象乙')).toBeInTheDocument();
});

it('旧 id 的风险评估慢响应不得覆盖新 id 的风险评估', async () => {
  const risk = (riskDescription: string): Awaited<ReturnType<typeof ChangeApi.getRiskAssessment>> => ({
    riskLevel: 'low', riskDescription, impactAnalysis: '', mitigationMeasures: '',
    contingencyPlan: '', riskOwner: '',
  });
  const old = deferred<ReturnType<typeof risk>>();
  jest.mocked(ChangeApi.getChange).mockImplementation(id => Promise.resolve(record(id, id === 1 ? '旧对象甲' : '新对象乙')));
  jest.mocked(ChangeApi.getRiskAssessment).mockImplementation(id => id === 1 ? old.promise : Promise.resolve(risk('新风险乙')));
  const { rerender } = render(<ChangeDetail />);
  await screen.findByText('旧对象甲');
  fireEvent.click(screen.getByRole('tab', { name: '风险评估' }));
  expect(ChangeApi.getRiskAssessment).toHaveBeenCalledWith(1);
  mockId = '2';
  rerender(<ChangeDetail />);
  await screen.findByText('新对象乙');
  fireEvent.click(screen.getByRole('tab', { name: '风险评估' }));
  expect(await screen.findByText('新风险乙')).toBeInTheDocument();
  await act(async () => { old.resolve(risk('旧风险甲')); });
  expect(screen.queryByText('旧风险甲')).not.toBeInTheDocument();
  expect(screen.getByText('新风险乙')).toBeInTheDocument();
});
