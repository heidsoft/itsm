import { act, renderHook } from '@testing-library/react';
import { useTickets } from '../useTickets';
import { ticketService } from '@/lib/services/ticket-service';
jest.mock('antd', () => ({ message: { error: jest.fn() } }));
jest.mock('@/lib/services/ticket-service', () => ({ ticketService: { listTickets: jest.fn() } }));
function deferred() {
  let resolve!: (value: unknown) => void;
  let reject!: (reason: Error) => void;
  const promise = new Promise((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
}
it('慢旧响应后到不覆盖新数据', async () => {
  const old = deferred();
  jest.mocked(ticketService.listTickets).mockReturnValueOnce(old.promise as never).mockResolvedValue({ items: [{ id: 2 }], total: 1 } as never);
  const { result } = renderHook(() => useTickets());
  await act(async () => { await result.current.fetchTickets(); });
  await act(async () => { old.resolve({ items: [{ id: 1 }], total: 9 }); });
  expect(result.current.tickets).toEqual([{ id: 2 }]);
  expect(result.current.pagination.total).toBe(1);
});
it.each([false, true])('过期请求 finally 不关新 loading，过期错误不写回：%s', async fail => {
  const old = deferred(); const current = deferred();
  jest.mocked(ticketService.listTickets).mockReturnValueOnce(old.promise as never).mockReturnValueOnce(current.promise as never);
  const { result } = renderHook(() => useTickets());
  let pending!: Promise<void>;
  act(() => { pending = result.current.fetchTickets(); });
  await act(async () => { if (fail) old.reject(new Error('旧失败')); else old.resolve({ items: [], total: 0 }); });
  expect(result.current.loading).toBe(true);
  expect(result.current.error).toBeNull();
  await act(async () => { current.resolve({ items: [], total: 0 }); await pending; });
  expect(result.current.loading).toBe(false);
});

it('旧请求失败晚到不覆盖新请求成功', async () => {
  const old = deferred();
  jest.mocked(ticketService.listTickets).mockReturnValueOnce(old.promise as never).mockResolvedValue({ items: [{ id: 2 }], total: 1 } as never);
  const { result } = renderHook(() => useTickets());
  await act(async () => { await result.current.fetchTickets(); });
  await act(async () => { old.reject(new Error('旧失败')); });
  expect(result.current.error).toBeNull();
  expect(result.current.tickets).toEqual([{ id: 2 }]);
});
