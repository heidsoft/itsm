/**
 * AI-Native ITSM: ticket investigation orchestration (triage → KB → CMDB → synthesis).
 *
 * Production chain:
 * POST /api/v1/ai/tickets/:id/investigate -> router/ai_routes.go (ai:read + ticket:read)
 * -> handlers/ai.Handler.InvestigateTicket -> handlers/ai.Service.InvestigateTicket
 * -> ExecuteTool (Gate 2 RBAC per step) -> ai_audit + ai_analysis_results.
 *
 * 决策支持契约：suggestedActions 只允许白名单动作且 requiresApproval 恒为 true；
 * 每次全量重跑落 ai_analysis_results(investigation) 历史与决策底账。
 */
import type { APIResponse, Page } from '@playwright/test';
import { test, expect } from '../fixtures/auth';

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8090';

type ApiEnvelope<T> = {
  code: number;
  message: string;
  data: T;
};

type Ticket = {
  id: number;
  title: string;
};

type InvestigationStep = {
  step: string;
  tool?: string;
  degraded?: boolean;
  degradedReason?: string;
  hitCount?: number;
};

type InvestigationResponse = {
  ticketId: number;
  ticketNumber?: string;
  status: string;
  degraded: boolean;
  degradedReason?: string;
  promptVersion: string;
  model?: string;
  confidence: number;
  steps: InvestigationStep[];
  findings?: string[];
  suspectedRootCause?: string;
  impactedServices?: string[];
  suggestedActions?: Array<{ action: string; description: string; requiresApproval: boolean }>;
  auditId?: number;
};

async function postAs(page: Page, token: string, path: string, data?: unknown): Promise<APIResponse> {
  return page.request.post(`${API_URL}${path}`, {
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    data,
  });
}

async function getAs(page: Page, token: string, path: string): Promise<APIResponse> {
  return page.request.get(`${API_URL}${path}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
}

async function readJson(response: APIResponse): Promise<Record<string, unknown>> {
  const contentType = response.headers()['content-type'] || '';
  if (!contentType.includes('application/json')) {
    return { message: await response.text() };
  }
  return response.json() as Promise<Record<string, unknown>>;
}

/** AI 依赖在当前环境不可用（503 未装配 / 5003 能力边界）时环境自适应跳过 */
function skipWhenAIUnavailable(response: APIResponse, body: Record<string, unknown>): void {
  const code = typeof body.code === 'number' ? body.code : undefined;
  const message = String(body.message ?? '');
  const unavailable =
    [502, 503, 504].includes(response.status()) ||
    code === 5003 ||
    /调查服务尚未就绪|service unavailable/i.test(message);

  test.skip(unavailable, `AI investigation dependency unavailable: HTTP ${response.status()} ${message}`);
}

async function createTicket(page: Page, token: string, title: string): Promise<Ticket> {
  const response = await postAs(page, token, '/api/v1/tickets', {
    title,
    description: '主库磁盘使用率 96%，告警持续 15 分钟，用于 AI 调查编排 E2E 验证',
    priority: 'high',
    category: 'hardware',
  });
  const body = (await readJson(response)) as ApiEnvelope<Ticket>;

  expect(response.status(), `创建调查前置工单失败: ${JSON.stringify(body)}`).toBe(200);
  expect(body.code).toBe(0);
  expect(body.data.id).toBeGreaterThan(0);
  return body.data;
}

test.describe('AI-Native ITSM: 工单调查编排', () => {
  test('调查编排返回结构化结论，白名单动作全部要求人工执行', async ({ page, loginAs }) => {
    const token = await loginAs('admin');

    // 权限探测：无 ai:read 的环境自适应跳过
    const probe = await getAs(page, token, '/api/v1/ai/metrics');
    if (probe.status() === 403) {
      test.skip(true, '当前账号无 ai:read 权限');
    }

    const ticket = await createTicket(page, token, `AI 调查编排 E2E-${Date.now()}`);

    const response = await postAs(page, token, `/api/v1/ai/tickets/${ticket.id}/investigate`, {});
    const body = (await readJson(response)) as ApiEnvelope<InvestigationResponse>;
    skipWhenAIUnavailable(response, body);

    expect(response.status(), JSON.stringify(body)).toBe(200);
    expect(body.code, JSON.stringify(body)).toBe(0);

    const data = body.data;
    // camelCase 契约键（不含 snake_case 泄漏）
    expect(data).toHaveProperty('ticketId');
    expect(data).toHaveProperty('status');
    expect(data).toHaveProperty('degraded');
    expect(data).toHaveProperty('promptVersion', 'investigation-v1');
    expect(Array.isArray(data.steps)).toBe(true);
    // 调查链至少含分诊步；completed 时综合步为 LLM
    expect(data.steps.length).toBeGreaterThanOrEqual(1);
    expect(data.steps[0]?.step).toBe('triage');
    expect(['completed', 'degraded']).toContain(data.status);
    expect(typeof data.degraded).toBe('boolean');

    if (!data.degraded) {
      expect(data.status).toBe('completed');
      expect(typeof data.suspectedRootCause === 'string').toBe(true);
      expect(Array.isArray(data.findings)).toBe(true);
    } else {
      // 降级路径不伪造根因
      expect(data.suspectedRootCause).toBeUndefined();
    }

    // 决策支持边界：建议只来自白名单且全部 requiresApproval=true
    if (data.suggestedActions) {
      const whitelist = new Set([
        'create_ticket',
        'update_ticket',
        'link_ticket_ci',
        'create_ci_relationship',
        'delete_ci_relationship',
      ]);
      for (const action of data.suggestedActions) {
        expect(whitelist.has(action.action), `越权建议 ${action.action}`).toBe(true);
        expect(action.requiresApproval).toBe(true);
      }
    }

    // 决策底账：每次调查落一条 ai_audit，响应携带行 ID
    expect(typeof data.auditId === 'number' && data.auditId > 0).toBe(true);
  });

  test('目标工单不存在返回 404/4004', async ({ page, loginAs }) => {
    const token = await loginAs('admin');
    const response = await postAs(page, token, '/api/v1/ai/tickets/99999999/investigate', {});
    const body = (await readJson(response)) as ApiEnvelope<unknown>;

    expect(response.status()).toBe(404);
    expect(body.code).toBe(4004);
  });
});

test.describe('AI-Native ITSM: 调查编排 UI 入口', () => {
  test('工单详情页展示 AI 调查卡片（ai:read 权限下）', async ({ page, loginAs }) => {
    const token = await loginAs('admin');

    const probe = await getAs(page, token, '/api/v1/ai/metrics');
    if (probe.status() === 403) {
      test.skip(true, '当前账号无 ai:read 权限');
    }

    const ticket = await createTicket(page, token, `AI 调查 UI E2E-${Date.now()}`);
    await page.goto(`/tickets/${ticket.id}`);
    await page.waitForLoadState('networkidle');

    // 卡片与说明文案存在（不点击运行：避免 E2E 依赖真实 LLM 时延）
    await expect(page.getByText('AI 调查', { exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: '运行 AI 调查' })).toBeVisible();
    await expect(page.getByText(/全程只读/)).toBeVisible();
  });
});
