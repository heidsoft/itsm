/**
 * QueryProvider 重试策略回归。
 *
 * 写请求重试是「静默重复生效」类缺陷：服务端已受理但响应超时/断连时，客户端
 * 看到的是失败，服务端却已经落库了审批决策或新建的单据，自动重放会再写一次。
 * 因此 mutation 必须 retry: 0。
 *
 * 只读请求可以重试，但 4xx 是服务端明确拒绝（无权限/参数错/不存在），
 * 重试只是让用户多等几秒并在监控里刷出一堆同样的错误。
 */

import { createQueryClient } from '../QueryProvider';
import { ApiRequestError } from '@/lib/api/http-client';

describe('QueryProvider 重试策略', () => {
  const client = createQueryClient();
  const defaults = client.getDefaultOptions();

  it('写操作一律不重试：无法区分「没发出去」与「已受理但响应丢失」', () => {
    expect(defaults.mutations?.retry).toBe(0);
  });

  it('读操作不对 4xx 重试', () => {
    const retry = defaults.queries?.retry;
    expect(typeof retry).toBe('function');

    const shouldRetry = retry as (count: number, error: unknown) => boolean;
    expect(shouldRetry(0, new ApiRequestError('无权限', 403))).toBe(false);
    expect(shouldRetry(0, new ApiRequestError('冲突', 409))).toBe(false);
    expect(shouldRetry(0, new ApiRequestError('不存在', 404))).toBe(false);
    expect(shouldRetry(0, new ApiRequestError('参数错误', 422))).toBe(false);
  });

  it('读操作对可能自愈的失败重试（网络错误 / 5xx / 429）', () => {
    const retry = defaults.queries?.retry as (count: number, error: unknown) => boolean;
    expect(retry(0, new ApiRequestError('网关错误', 502))).toBe(true);
    expect(retry(0, new ApiRequestError('限流', 429))).toBe(true);
    // 网络层失败不会被包装成 ApiRequestError
    expect(retry(0, new Error('Failed to fetch'))).toBe(true);
  });

  it('读操作重试次数有上限', () => {
    const retry = defaults.queries?.retry as (count: number, error: unknown) => boolean;
    expect(retry(3, new ApiRequestError('网关错误', 502))).toBe(false);
  });
});
