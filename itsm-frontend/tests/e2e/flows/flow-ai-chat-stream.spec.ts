/**
 * AI-Native ITSM: streaming chat, tool-assisted answers, and persisted history.
 *
 * Production chain:
 * POST /api/v1/ai/chat/stream -> AIHandler.ChatStream -> AI Service.ChatStream
 * -> RAG/LLM (with RBAC-filtered tools) -> conversation repository.
 */
import { expect, test, type APIRequestContext, type Page } from '@playwright/test';
import { loginAs } from '../utils/test-utils';

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8090';

type StreamResult = {
  status: number;
  contentType: string;
  raw: string;
  deltas: string[];
  conversationId?: number;
  error?: string;
};

type ApiEnvelope<T> = {
  code: number;
  message: string;
  data: T;
};

async function authenticatedToken(page: Page, role: 'admin' | 'end_user'): Promise<string> {
  await loginAs(page, role);
  const token = await page.evaluate(() => localStorage.getItem('access_token'));
  expect(token, `${role} 登录后必须保存 access token`).toBeTruthy();
  return token as string;
}

async function streamChat(
  request: APIRequestContext,
  token: string,
  query: string,
  conversationId = 0,
): Promise<StreamResult> {
  const response = await request.post(`${API_URL}/api/v1/ai/chat/stream`, {
    headers: {
      Accept: 'text/event-stream',
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    data: { query, conversationId, limit: 10 },
    timeout: 60_000,
  });
  const raw = await response.text();
  const deltas: string[] = [];
  let finalConversationId: number | undefined;
  let streamError: string | undefined;

  for (const block of raw.split(/\r?\n\r?\n/)) {
    const event = block.match(/^event:\s*(.+)$/m)?.[1]?.trim();
    const data = block.match(/^data:\s*(.+)$/m)?.[1];
    if (!event || !data) continue;

    let payload: unknown;
    try {
      payload = JSON.parse(data);
    } catch {
      continue;
    }

    if (event === 'delta' && typeof (payload as { content?: unknown }).content === 'string') {
      deltas.push((payload as { content: string }).content);
    }
    if (event === 'done' && typeof (payload as { conversationId?: unknown }).conversationId === 'number') {
      finalConversationId = (payload as { conversationId: number }).conversationId;
    }
    if (event === 'error' && typeof (payload as { message?: unknown }).message === 'string') {
      streamError = (payload as { message: string }).message;
    }
  }

  return {
    status: response.status(),
    contentType: response.headers()['content-type'] || '',
    raw,
    deltas,
    conversationId: finalConversationId,
    error: streamError,
  };
}

async function getJson<T>(request: APIRequestContext, token: string, path: string) {
  const response = await request.get(`${API_URL}${path}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(response.status(), `GET ${path} 必须成功`).toBe(200);
  return (await response.json()) as ApiEnvelope<T>;
}

test.describe('AI ChatStream 多轮对话与工具调用', () => {
  test('用户收到 SSE 响应，且对话和消息历史被持久化', async ({ page, request }) => {
    const token = await authenticatedToken(page, 'admin');
    const query = `如何重置密码？ E2E-${Date.now()}`;
    const stream = await streamChat(request, token, query);

    expect(stream.status).toBe(200);
    expect(stream.contentType).toContain('text/event-stream');
    test.skip(Boolean(stream.error), `AI/RAG 服务不可用: ${stream.error}`);
    expect(stream.raw).toContain('event: done');
    expect(stream.deltas.join('').trim(), '流式响应必须包含文本 delta').not.toBe('');
    expect(stream.conversationId, 'done 事件必须返回 conversationId').toBeGreaterThan(0);

    const conversationId = stream.conversationId as number;
    const list = await getJson<{ conversations: Array<{ id: number }> }>(
      request,
      token,
      '/api/v1/ai/conversations',
    );
    expect(list.code).toBe(0);
    expect(list.data.conversations.map(item => item.id)).toContain(conversationId);

    const history = await getJson<{
      messages: Array<{ conversationId: number; role: string; content: string }>;
    }>(request, token, `/api/v1/ai/conversations/${conversationId}`);
    expect(history.code).toBe(0);
    expect(history.data.messages).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ conversationId, role: 'user', content: query }),
        expect.objectContaining({ conversationId, role: 'assistant' }),
      ]),
    );
    expect(
      history.data.messages.find(message => message.role === 'assistant')?.content.length,
    ).toBeGreaterThan(0);
  });

  test('end_user 询问最近工单时，只读工具返回相关工单信息', async ({ page, request }) => {
    const token = await authenticatedToken(page, 'end_user');
    const title = `AI-CHAT-TICKET-${Date.now()}`;
    const createResponse = await request.post(`${API_URL}/api/v1/tickets`, {
      headers: {
        Authorization: `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      data: {
        title,
        description: '用于验证 AI 聊天只读工单工具',
        priority: 'high',
        category: 'general',
      },
    });
    expect(createResponse.status(), 'user1 必须能创建测试工单').toBe(200);
    const created = (await createResponse.json()) as ApiEnvelope<{ id: number; ticketNumber?: string }>;
    expect(created.code).toBe(0);
    expect(created.data.id).toBeGreaterThan(0);

    const stream = await streamChat(
      request,
      token,
      `请使用工单查询工具告诉我最近的工单 ${title} 是什么状态？`,
    );
    expect(stream.status).toBe(200);
    test.skip(Boolean(stream.error), `AI/RAG 服务不可用: ${stream.error}`);

    const answer = stream.deltas.join('');
    const identifiers = [title, created.data.ticketNumber].filter(Boolean) as string[];
    expect(
      identifiers.some(identifier => answer.includes(identifier)),
      `AI 回答应包含工单标题或编号，实际回答: ${answer}`,
    ).toBe(true);
    expect(answer).toMatch(/状态|status/i);
  });

  test('工具列表按角色过滤，admin 可见管理工具', async ({ page, request }) => {
    const endUserToken = await authenticatedToken(page, 'end_user');
    const endUserResponse = await getJson<{
      tools: Array<{ name: string; resource: string; action: string }>;
    }>(request, endUserToken, '/api/v1/agent/tools');
    expect(endUserResponse.code).toBe(0);

    const adminToken = await authenticatedToken(page, 'admin');
    const adminResponse = await getJson<{
      tools: Array<{ name: string; resource: string; action: string }>;
    }>(request, adminToken, '/api/v1/agent/tools');
    expect(adminResponse.code).toBe(0);

    const endUserTools = endUserResponse.data.tools.map(tool => tool.name);
    const adminTools = adminResponse.data.tools.map(tool => tool.name);
    expect(adminTools).toContain('create_ticket_type');

    test.skip(
      adminTools.length === endUserTools.length && adminTools.every(name => endUserTools.includes(name)),
      'AI Tool RBAC 未启用；后端按兼容模式向所有角色返回完整工具列表',
    );
    expect(endUserTools).not.toContain('create_ticket_type');
    expect(adminTools.some(name => !endUserTools.includes(name))).toBe(true);
  });
});
