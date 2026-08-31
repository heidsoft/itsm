/**
 * AI-Native ITSM: intelligent triage and ticket root-cause analysis.
 *
 * Production chain:
 * /api/v1/ai/* -> handlers/ai.Handler -> handlers/ai.Service ->
 * TriageService / RootCauseService / AI analysis repository.
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

type RootCauseAnalysis = {
  ticketId: number;
  ticketNumber: string;
  ticketTitle: string;
  analysisDate: string;
  rootCauses: Array<{
    id: string;
    title: string;
    description: string;
    confidence: number;
    category: string;
    status: string;
  }>;
  analysisSummary: string;
  confidenceScore: number;
  analysisMethod: string;
  generatedAt: string;
};

type AnalysisHistoryItem = {
  id: number;
  analysisType: string;
  requestPrompt: string;
  resultJson: string;
  degraded: boolean;
  createdAt: string;
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

function skipWhenAIUnavailable(response: APIResponse, body: Record<string, unknown>): void {
  const code = typeof body.code === 'number' ? body.code : undefined;
  const message = String(body.message ?? '');
  const unavailable =
    [502, 503, 504].includes(response.status()) ||
    code === 5003 ||
    /AI.*(?:不可用|未就绪)|service unavailable/i.test(message);

  test.skip(unavailable, `AI dependency is unavailable: HTTP ${response.status()} ${message}`);
}

async function createTicket(page: Page, token: string, title: string): Promise<Ticket> {
  const response = await postAs(page, token, '/api/v1/tickets', {
    title,
    description: '系统启动失败并持续蓝屏，用于 AI 根因分析 E2E 验证',
    priority: 'high',
    category: 'hardware',
  });
  const body = (await readJson(response)) as ApiEnvelope<Ticket>;

  expect(response.status(), `创建 RCA 前置工单失败: ${JSON.stringify(body)}`).toBe(200);
  expect(body.code).toBe(0);
  expect(body.data.id).toBeGreaterThan(0);
  expect(body.data.title).toBe(title);
  return body.data;
}

test.describe('AI-Native ITSM: 智能分诊与 RCA', () => {
  test('AI 智能分诊返回结构化建议，并支持 CreateTicketByAI 降级建议', async ({
    page,
    loginAs,
  }) => {
    const token = await loginAs('user1');
    const description = '我的电脑蓝屏了，无法启动，屏幕上出现 STOP ERROR 0x0000007B';

    const triageResponse = await postAs(page, token, '/api/v1/ai/triage', {
      title: '电脑蓝屏且无法启动',
      description,
    });
    const triageBody = await readJson(triageResponse);
    skipWhenAIUnavailable(triageResponse, triageBody);

    expect(triageResponse.status(), JSON.stringify(triageBody)).toBe(200);
    expect(triageBody.code).toBe(0);

    const triageData = (triageBody as ApiEnvelope<{
      title: string;
      description: string;
      suggestions: {
        category: string;
        priority: string;
        confidence: number;
        reasoning: string;
        urgency: string;
      };
    }>).data;
    expect(triageData.title).toBe('电脑蓝屏且无法启动');
    expect(triageData.description).toBe(description);
    expect(triageData.suggestions.category).toMatch(/^(hardware|general|incident)$/);
    expect(triageData.suggestions.priority).toMatch(/^(medium|high|urgent|critical)$/);
    expect(triageData.suggestions.confidence).toBeGreaterThanOrEqual(0);
    expect(triageData.suggestions.confidence).toBeLessThanOrEqual(1);
    expect(triageData.suggestions.reasoning.length).toBeGreaterThan(0);

    // CreateTicketByAI is a suggestion endpoint: it must not silently create a ticket.
    const createByAIResponse = await postAs(page, token, '/api/v1/ai/ticket/create', {
      description,
    });
    const createByAIBody = await readJson(createByAIResponse);
    skipWhenAIUnavailable(createByAIResponse, createByAIBody);

    expect(createByAIResponse.status(), JSON.stringify(createByAIBody)).toBe(200);

    // CreateTicketByAI 返回扁平结构 {code, message, data: {suggested_category, suggested_priority, confidence, ...}}
    // common.Success 包装后: {code:0, data: {suggested_category, suggested_priority, confidence, status:'draft', message}}
    const createData = (createByAIBody as ApiEnvelope<Record<string, unknown>>).data as Record<string, unknown>;
    expect(createData.suggested_category).toMatch(/^(hardware|general|incident)$/);
    expect(createData.suggested_priority).toMatch(/^(medium|high|urgent|critical)$/);
    expect(Number(createData.confidence)).toBeGreaterThanOrEqual(0);
    expect(Number(createData.confidence)).toBeLessThanOrEqual(1);
    expect(String(createData.reasoning ?? createData.message)).not.toHaveLength(0);
    expect(createData.status).toBe('draft');
  });

  test('AnalyzeTicket 对独立工单执行 RCA，并返回可操作的分析结构', async ({
    page,
    loginAs,
  }) => {
    const token = await loginAs('admin');
    const title = `AI-RCA-E2E-${Date.now()}`;
    const ticket = await createTicket(page, token, title);

    const response = await postAs(page, token, `/api/v1/ai/tickets/${ticket.id}/analyze`);
    const body = await readJson(response);
    skipWhenAIUnavailable(response, body);

    expect(response.status(), JSON.stringify(body)).toBe(200);
    expect(body.code).toBe(0);
    const analysis = (body as ApiEnvelope<RootCauseAnalysis>).data;
    expect(analysis.ticketId).toBe(ticket.id);
    expect(analysis.ticketTitle).toBe(title);
    expect(analysis.ticketNumber.length).toBeGreaterThan(0);
    expect(analysis.rootCauses.length).toBeGreaterThan(0);
    expect(analysis.rootCauses[0].title.length).toBeGreaterThan(0);
    expect(analysis.rootCauses[0].confidence).toBeGreaterThanOrEqual(0);
    expect(analysis.rootCauses[0].confidence).toBeLessThanOrEqual(1);
    expect(analysis.analysisSummary).toContain('根本原因');
    expect(analysis.analysisMethod).toBe('automatic');
    expect(Number.isNaN(Date.parse(analysis.generatedAt))).toBe(false);
  });

  test('AnalyzeTicketWithAudit 将 RCA 结果持久化到租户分析历史', async ({
    page,
    loginAs,
  }) => {
    const token = await loginAs('admin');
    const ticket = await createTicket(page, token, `AI-RCA-AUDIT-E2E-${Date.now()}`);

    const analyzeResponse = await postAs(
      page,
      token,
      `/api/v1/ai/tickets/${ticket.id}/analyze`,
    );
    const analyzeBody = await readJson(analyzeResponse);
    skipWhenAIUnavailable(analyzeResponse, analyzeBody);
    expect(analyzeResponse.status(), JSON.stringify(analyzeBody)).toBe(200);
    expect(analyzeBody.code).toBe(0);

    const historyResponse = await getAs(page, token, '/api/v1/ai/analysis-results?type=rca&limit=50');
    const historyBody = await readJson(historyResponse);
    expect(historyResponse.status(), JSON.stringify(historyBody)).toBe(200);
    expect(historyBody.code).toBe(0);

    const history = (historyBody as ApiEnvelope<{ results: AnalysisHistoryItem[] }>).data.results;
    const persisted = history.find(
      item => item.analysisType === 'rca' && item.requestPrompt === `AnalyzeTicket ticketID=${ticket.id}`,
    );
    expect(persisted, `未找到 ticketId=${ticket.id} 的 RCA 分析审计记录`).toBeDefined();
    expect(persisted?.id).toBeGreaterThan(0);
    expect(persisted?.degraded).toBe(false);
    expect(Number.isNaN(Date.parse(persisted?.createdAt || ''))).toBe(false);

    const persistedResult = JSON.parse(persisted?.resultJson || '{}') as RootCauseAnalysis;
    expect(persistedResult.ticketId).toBe(ticket.id);
    expect(persistedResult.ticketTitle).toBe(ticket.title);
    expect(persistedResult.analysisSummary.length).toBeGreaterThan(0);
  });
});
