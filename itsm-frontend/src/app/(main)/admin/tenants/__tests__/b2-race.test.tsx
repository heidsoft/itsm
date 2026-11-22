import { act, fireEvent, render, screen } from '@testing-library/react';
import { App } from 'antd';
import Page from '../page';
import { TenantAPI } from '@/lib/api/tenant-api';
jest.mock('@/lib/api/tenant-api', () => ({ TenantAPI: { getTenants: jest.fn() } }));
function deferred() {
  let resolve!: (value: unknown) => void;
  let reject!: (reason: Error) => void;
  const promise = new Promise((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
}
beforeEach(() => { jest.useFakeTimers(); });
afterEach(() => { jest.useRealTimers(); });
it('搜索等待 300ms，只请求最终输入', async () => {
  jest.mocked(TenantAPI.getTenants).mockResolvedValue({ items: [] } as never);
  render(<App><Page /></App>);
  await act(async () => {});
  fireEvent.change(screen.getByRole('textbox'), { target: { value: 'a' } });
  fireEvent.change(screen.getByRole('textbox'), { target: { value: 'ab' } });
  expect(TenantAPI.getTenants).toHaveBeenCalledTimes(1);
  await act(async () => { jest.advanceTimersByTime(299); });
  expect(TenantAPI.getTenants).toHaveBeenCalledTimes(1);
  await act(async () => { jest.advanceTimersByTime(1); });
  expect(TenantAPI.getTenants).toHaveBeenLastCalledWith(expect.objectContaining({ search: 'ab' }));
  expect(TenantAPI.getTenants).toHaveBeenCalledTimes(2);
});
it.each([false, true])('旧请求完成不关闭新请求 loading 或显示旧错误：%s', async fail => {
  const old = deferred(); const current = deferred();
  jest.mocked(TenantAPI.getTenants).mockReturnValueOnce(old.promise as never).mockReturnValueOnce(current.promise as never);
  render(<App><Page /></App>);
  fireEvent.change(screen.getByRole('textbox'), { target: { value: 'new' } });
  await act(async () => { jest.advanceTimersByTime(300); });
  await act(async () => { if (fail) old.reject(new Error('旧失败')); else old.resolve({ items: [] }); });
  expect(document.querySelector('.ant-spin-spinning')).not.toBeNull();
  expect(screen.queryByText('加载租户失败')).not.toBeInTheDocument();
  await act(async () => { current.resolve({ items: [] }); });
  expect(document.querySelector('.ant-spin-spinning')).toBeNull();
});

it('旧列表晚到不覆盖最新租户和统计', async () => {
  const old = deferred();
  jest.mocked(TenantAPI.getTenants).mockReturnValueOnce(old.promise as never).mockResolvedValue({ items: [] } as never);
  render(<App><Page /></App>);
  fireEvent.change(screen.getByRole('textbox'), { target: { value: 'new' } });
  await act(async () => { jest.advanceTimersByTime(300); });
  await act(async () => { old.resolve({ items: [{ id: 99, name: '旧请求租户99', status: 'active' }] }); });
  expect(screen.queryByText('旧请求租户99')).not.toBeInTheDocument();
  expect(document.querySelector('.ant-spin-spinning')).toBeNull();
});
