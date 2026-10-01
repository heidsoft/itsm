/**
 * 会话真相 API —— 前端判断「我是否已登录」的唯一来源。
 *
 * 后端 GET /api/v1/auth/session 一次给出身份、可访问租户与 access token 的剩余秒数
 * （剩余值由服务端时钟计算）。前端不再用以下任何信号推断登录态：
 * document.cookie 里是否存在凭证、JWT 字符串形状、localStorage/Zustand 持久化状态、
 * 浏览器时钟推算的「15 分钟到了」。
 *
 * 三态是刻意的：unavailable（后端 5xx/网络故障）不得被当成未登录，否则一次瞬时故障
 * 就会把已登录用户踢出；它也不能被当成已登录，否则界面会停在假登录态。
 */
import type { Tenant, User } from '@/lib/api/api-config';
import { API_BASE_URL } from '@/lib/api/api-config';

const SESSION_PATH = '/api/v1/auth/session';
const REFRESH_PATH = '/api/v1/auth/refresh';
const LOGOUT_PATH = '/api/v1/auth/logout';

export type SessionOutcome =
  | { state: 'authenticated'; user: User; tenants: Tenant[]; expiresIn: number }
  | { state: 'unauthenticated' }
  | { state: 'unavailable'; reason: string };

export type RefreshOutcome =
  | { state: 'refreshed'; expiresIn: number }
  | { state: 'expired' }
  | { state: 'unavailable'; reason: string };

export type LogoutOutcome = { revoked: boolean; reason?: string };

interface SessionEnvelope {
  code?: number;
  message?: string;
  data?: {
    user?: User;
    tenants?: Tenant[];
    expiresIn?: number;
  } | null;
}

/**
 * 续签必须单飞：后端的 refresh token 是单次使用（原子认领），同一枚凭证被重复提交时，
 * 只有第一次能换到新会话。若每个 401 响应各自发起续签，一次令牌过期就会把有效会话踢出。
 */
let refreshInFlight: Promise<RefreshOutcome> | null = null;

async function parseEnvelope(response: Response): Promise<SessionEnvelope> {
  try {
    return (await response.json()) as SessionEnvelope;
  } catch {
    return {};
  }
}

export async function loadSession(signal?: AbortSignal): Promise<SessionOutcome> {
  let response: Response;
  try {
    response = await fetch(`${API_BASE_URL}${SESSION_PATH}`, {
      method: 'GET',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      signal,
    });
  } catch (error) {
    return { state: 'unavailable', reason: error instanceof Error ? error.message : 'network error' };
  }

  // 401/403 是后端对会话的权威结论；其余非 2xx 是服务故障，不能折算成未登录。
  if (response.status === 401 || response.status === 403) {
    return { state: 'unauthenticated' };
  }

  const envelope = await parseEnvelope(response);
  if (!response.ok || envelope.code !== 0 || !envelope.data?.user) {
    return {
      state: 'unavailable',
      reason: envelope.message || `session endpoint returned status ${response.status}`,
    };
  }

  return {
    state: 'authenticated',
    user: envelope.data.user,
    tenants: Array.isArray(envelope.data.tenants) ? envelope.data.tenants : [],
    expiresIn: Number(envelope.data.expiresIn ?? 0),
  };
}

function refreshOnce(): Promise<RefreshOutcome> {
  return (async () => {
    let response: Response;
    try {
      response = await fetch(`${API_BASE_URL}${REFRESH_PATH}`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: '{}',
      });
    } catch (error) {
      return {
        state: 'unavailable',
        reason: error instanceof Error ? error.message : 'network error',
      };
    }

    if (response.status === 401 || response.status === 403) {
      return { state: 'expired' };
    }

    const envelope = await parseEnvelope(response);
    if (!response.ok || envelope.code !== 0) {
      return {
        state: 'unavailable',
        reason: envelope.message || `refresh endpoint returned status ${response.status}`,
      };
    }

    return { state: 'refreshed', expiresIn: Number(envelope.data?.expiresIn ?? 0) };
  })();
}

export function refreshSession(): Promise<RefreshOutcome> {
  if (!refreshInFlight) {
    refreshInFlight = refreshOnce().finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
}

/**
 * 登出用 keepalive 发出：调用方随后立即做整页跳转，普通 fetch 会在页面卸载时被取消，
 * 服务端因此收不到吊销请求——cookie 清了，7 天的 refresh 凭证却还有效。
 */
export async function logoutSession(): Promise<LogoutOutcome> {
  try {
    const response = await fetch(`${API_BASE_URL}${LOGOUT_PATH}`, {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: '{}',
      keepalive: true,
    });
    const envelope = await parseEnvelope(response);
    if (response.ok && envelope.code === 0) {
      return { revoked: true };
    }
    return { revoked: false, reason: envelope.message || `logout returned status ${response.status}` };
  } catch (error) {
    return {
      revoked: false,
      reason: error instanceof Error ? error.message : 'network error',
    };
  }
}
