import { httpClient } from './http-client';
import { security } from '@/lib/security';
import { getTenantId, getTenantCode } from '@/lib/auth/tenant-context';

export interface A2UIMessageResponse {
  code: number;
  message: string;
  messages: string[];
  success?: boolean;
}

/**
 * A2UI 响应为根级 { code, message, messages }（无 data 字段），
 * 与 httpClient.requestInternal 的 toCamelCase(responseData.data) 解析
 * 不兼容，因此保留裸 fetch；但必须对齐 httpClient 的请求头：
 * CSRF token（mutating 请求必需）+ 租户头 + X-Requested-With。
 */
async function postA2UI(path: string, payload: Record<string, unknown>): Promise<A2UIMessageResponse> {
  const headers: Record<string, string> = security.network.getSecureHeaders();
  const csrfToken = await httpClient.getCSRFTokenForExternal();
  if (csrfToken) {
    headers['X-CSRF-Token'] = csrfToken;
  }
  const tenantId = getTenantId();
  const tenantCode = getTenantCode();
  if (tenantId) {
    headers['X-Tenant-ID'] = tenantId.toString();
  }
  if (tenantCode) {
    headers['X-Tenant-Code'] = tenantCode;
  }

  const response = await fetch(`${httpClient.getBaseURL()}${path}`, {
    method: 'POST',
    credentials: 'include',
    headers,
    body: JSON.stringify(payload),
  });

  const data = (await response.json()) as Partial<A2UIMessageResponse>;
  if (!response.ok || data.code !== 0) {
    throw new Error(data.message || `HTTP error! status: ${response.status}`);
  }

  // Backend rotates the CSRF cookie after every successful mutation; invalidate
  // the cache so the next mutation forces a fresh token.
  httpClient.invalidateCSRFToken();

  return {
    code: data.code ?? 0,
    message: data.message || 'success',
    messages: data.messages || [],
    success: data.success,
  };
}

export class A2UIApi {
  static generateTicketForm(intent: string, surfaceId: string | null): Promise<A2UIMessageResponse> {
    return postA2UI('/api/v1/a2ui/ticket/form', { intent, surfaceId });
  }

  static handleTicketAction(
    action: string,
    surfaceId: string,
    context: Record<string, unknown>
  ): Promise<A2UIMessageResponse> {
    return postA2UI('/api/v1/a2ui/ticket/action', { action, surfaceId, context });
  }
}
