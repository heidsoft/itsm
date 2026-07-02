import { httpClient } from './http-client';

export interface A2UIMessageResponse {
  code: number;
  message: string;
  messages: string[];
  success?: boolean;
}

async function postA2UI(path: string, payload: Record<string, unknown>): Promise<A2UIMessageResponse> {
  const response = await fetch(`${httpClient.getBaseURL()}${path}`, {
    method: 'POST',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(payload),
  });

  const data = (await response.json()) as Partial<A2UIMessageResponse>;
  if (!response.ok || data.code !== 0) {
    throw new Error(data.message || `HTTP error! status: ${response.status}`);
  }

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
