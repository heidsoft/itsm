/**
 * 会话真相客户端测试。
 *
 * 这里锁死三条不变量，任何一条被破坏都会让用户在瞬时故障时被踢出，或在过期后停留假登录态：
 * 1. 未登录/服务故障/已登录三态互不折算，故障不能当成未登录；
 * 2. 续签必须单飞——后端 refresh token 单次可用，并行续签会吊销有效会话；
 * 3. 登出必须 keepalive，否则调用方整页跳转会取消请求、服务端不吊销凭证。
 */
import type { SessionOutcome, RefreshOutcome, LogoutOutcome } from '@/lib/api/session-api';

const mockFetch = jest.fn();
global.fetch = mockFetch;

type SessionApi = typeof import('@/lib/api/session-api');

/** refreshInFlight 是模块级状态，每个用例需要拿到干净的模块实例。 */
async function freshSessionApi(): Promise<SessionApi> {
  jest.resetModules();
  return (await import('@/lib/api/session-api')) as SessionApi;
}

function jsonResponse(init: { status: number; ok?: boolean; body: unknown }) {
  return {
    ok: init.ok ?? (init.status >= 200 && init.status < 300),
    status: init.status,
    json: async () => init.body,
  };
}

describe('session-api loadSession', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.spyOn(console, 'error').mockImplementation(() => {});
  });

  it('返回后端的会话结论与相对剩余时间', async () => {
    const api = await freshSessionApi();
    mockFetch.mockResolvedValue(
      jsonResponse({
        status: 200,
        body: {
          code: 0,
          message: 'success',
          data: {
            user: { id: 1, username: 'admin' },
            tenants: [{ id: 1, name: '默认租户', code: 'default' }],
            expiresIn: 900,
          },
        },
      }),
    );

    const outcome = await api.loadSession();

    expect(outcome).toEqual({
      state: 'authenticated',
      user: { id: 1, username: 'admin' },
      tenants: [{ id: 1, name: '默认租户', code: 'default' }],
      expiresIn: 900,
    });
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/auth/session'),
      expect.objectContaining({ method: 'GET', credentials: 'include' }),
    );
  });

  it('只依赖 httpOnly cookie 会话，不自行拼 Authorization', async () => {
    const api = await freshSessionApi();
    mockFetch.mockResolvedValue(
      jsonResponse({ status: 200, body: { code: 0, data: { user: { id: 1 }, expiresIn: 60 } } }),
    );

    await api.loadSession();

    const options = mockFetch.mock.calls[0][1] as Record<string, unknown>;
    const headers = options.headers as Record<string, string>;
    expect(headers).not.toHaveProperty('Authorization');
  });

  it.each([401, 403])('把 %i 当作后端权威的未登录结论', async status => {
    const api = await freshSessionApi();
    mockFetch.mockResolvedValue(jsonResponse({ status, body: { code: 2001, message: 'unauthorized' } }));

    const outcome = await api.loadSession();
    expect(outcome).toEqual({ state: 'unauthenticated' });
  });

  it('把 5xx 当作 unavailable，而不是把用户踢出', async () => {
    const api = await freshSessionApi();
    mockFetch.mockResolvedValue(jsonResponse({ status: 503, body: { code: 5001, message: 'db down' } }));

    const outcome: SessionOutcome = await api.loadSession();
    expect(outcome.state).toBe('unavailable');
  });

  it('把业务 code 非 0 当作 unavailable 并保留原因', async () => {
    const api = await freshSessionApi();
    mockFetch.mockResolvedValue(jsonResponse({ status: 200, body: { code: 5001, message: '内部错误' } }));

    const outcome = await api.loadSession();
    expect(outcome).toEqual({ state: 'unavailable', reason: '内部错误' });
  });

  it('把缺少 user 的成功响应当作 unavailable，不伪造登录态', async () => {
    const api = await freshSessionApi();
    mockFetch.mockResolvedValue(jsonResponse({ status: 200, body: { code: 0, data: { expiresIn: 900 } } }));

    const outcome: SessionOutcome = await api.loadSession();
    expect(outcome.state).toBe('unavailable');
  });

  it('把网络异常当作 unavailable', async () => {
    const api = await freshSessionApi();
    mockFetch.mockRejectedValue(new Error('Network down'));

    const outcome = await api.loadSession();
    expect(outcome).toEqual({ state: 'unavailable', reason: 'Network down' });
  });
});

describe('session-api refreshSession', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('走 canonical 续签端点，不使用 legacy /api/v1/refresh-token 别名', async () => {
    const api = await freshSessionApi();
    mockFetch.mockResolvedValue(jsonResponse({ status: 200, body: { code: 0, data: { expiresIn: 900 } } }));

    const outcome: RefreshOutcome = await api.refreshSession();

    expect(outcome).toEqual({ state: 'refreshed', expiresIn: 900 });
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/auth/refresh'),
      expect.objectContaining({ method: 'POST', credentials: 'include' }),
    );
    const url = mockFetch.mock.calls[0][0] as string;
    expect(url).not.toContain('/api/v1/refresh-token');
  });

  it('把 401 当作会话确实过期', async () => {
    const api = await freshSessionApi();
    mockFetch.mockResolvedValue(jsonResponse({ status: 401, body: { code: 2001, message: 'expired' } }));

    expect(await api.refreshSession()).toEqual({ state: 'expired' });
  });

  it('把后端故障当作 unavailable，不折算成过期', async () => {
    const api = await freshSessionApi();
    mockFetch.mockResolvedValue(jsonResponse({ status: 500, body: { code: 5001, message: 'boom' } }));

    expect(await api.refreshSession()).toEqual({ state: 'unavailable', reason: 'boom' });
  });

  it('并行续签只发一次请求：单次可用凭证不能被重复提交', async () => {
    const api = await freshSessionApi();
    const pending: Array<(value: unknown) => void> = [];
    mockFetch.mockImplementation(
      () =>
        new Promise(resolve => {
          pending.push(resolve);
        }),
    );

    const calls = [api.refreshSession(), api.refreshSession(), api.refreshSession()];
    expect(mockFetch).toHaveBeenCalledTimes(1);

    pending.forEach(resolve => resolve(jsonResponse({ status: 200, body: { code: 0, data: { expiresIn: 900 } } })));
    const results = await Promise.all(calls);

    expect(results).toEqual([
      { state: 'refreshed', expiresIn: 900 },
      { state: 'refreshed', expiresIn: 900 },
      { state: 'refreshed', expiresIn: 900 },
    ]);
    expect(mockFetch).toHaveBeenCalledTimes(1);
  });

  it('续签结束后释放单飞锁，下一次过期仍能续签', async () => {
    const api = await freshSessionApi();
    mockFetch.mockResolvedValue(jsonResponse({ status: 200, body: { code: 0, data: { expiresIn: 900 } } }));

    await api.refreshSession();
    await api.refreshSession();

    expect(mockFetch).toHaveBeenCalledTimes(2);
  });
});

describe('session-api logoutSession', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('用 keepalive 发出吊销请求，保证整页跳转前送达', async () => {
    const api = await freshSessionApi();
    mockFetch.mockResolvedValue(jsonResponse({ status: 200, body: { code: 0, message: 'success' } }));

    const outcome: LogoutOutcome = await api.logoutSession();

    expect(outcome).toEqual({ revoked: true });
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/auth/logout'),
      expect.objectContaining({ method: 'POST', credentials: 'include', keepalive: true }),
    );
  });

  it('吊销失败时返回可观察的 revoked=false 与原因', async () => {
    const api = await freshSessionApi();
    mockFetch.mockResolvedValue(
      jsonResponse({ status: 500, body: { code: 5001, message: 'revocation store down' } }),
    );

    expect(await api.logoutSession()).toEqual({ revoked: false, reason: 'revocation store down' });
  });

  it('网络异常时同样返回 revoked=false 而不是抛出', async () => {
    const api = await freshSessionApi();
    mockFetch.mockRejectedValue(new Error('timeout'));

    expect(await api.logoutSession()).toEqual({ revoked: false, reason: 'timeout' });
  });
});
