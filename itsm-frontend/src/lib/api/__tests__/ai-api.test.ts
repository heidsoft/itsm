import {
  aiTriage,
  aiSearchKB,
  aiSimilarIncidents,
  aiSummarize,
  aiSaveFeedback,
  aiGetMetrics,
  aiGetEvaluation,
  aiGetAuditLogs,
  aiClassifyTicket,
  aiSuggestSolutions,
  aiIntelligentSearch,
  aiDryRunTool,
  aiReviewToolInvocation,
  AIApi,
} from '../ai-api';
import { httpClient } from '../http-client';

jest.mock('../http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
    getBaseURL: jest.fn(() => 'http://localhost:3000'),
    getAuthToken: jest.fn(() => 'token123'),
    getTenantId: jest.fn(() => 1),
  },
}));

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;

describe('AI API', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('aiTriage', () => {
    it('should triage and transform response', async () => {
      mockPost.mockResolvedValue({ suggestions: { category: 'network', priority: 'high', confidence: 0.9, reasoning: 'network issue', urgency: 'urgent' } });
      const result = await aiTriage('Network down', 'Cannot connect');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/ai/triage', { title: 'Network down', description: 'Cannot connect' });
      expect(result.category).toBe('network');
      expect(result.priority).toBe('high');
      expect(result.confidence).toBe(0.9);
      expect(result.explanation).toBe('network issue');
    });

    it('should handle missing suggestions', async () => {
      mockPost.mockResolvedValue({});
      const result = await aiTriage('Test', 'Desc');
      expect(result.category).toBe('general');
      expect(result.priority).toBe('medium');
      expect(result.confidence).toBe(0);
    });
  });

  describe('aiSearchKB', () => {
    it('should search knowledge base', async () => {
      mockPost.mockResolvedValue({ items: [{ id: 1, objectType: 'article', snippet: 'found' }] });
      const result = await aiSearchKB('password reset', 3);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/ai/rag/search', { query: 'password reset', limit: 3, type: 'kb' });
      expect(result.answers).toHaveLength(1);
    });

    it('should handle missing results', async () => {
      mockPost.mockResolvedValue({});
      const result = await aiSearchKB('test');
      expect(result.answers).toEqual([]);
    });
  });

  describe('aiSimilarIncidents', () => {
    it('should find similar incidents', async () => {
      mockPost.mockResolvedValue({ items: [{ id: 1, objectType: 'incident', snippet: 'similar' }] });
      const result = await aiSimilarIncidents('server crash', 5);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/ai/rag/search', { query: 'server crash', limit: 5, type: 'incident' });
      expect(result.incidents).toHaveLength(1);
    });
  });

  describe('incident analysis contract', () => {
    it('uses the incident aggregate endpoint', async () => {
      mockPost.mockResolvedValue({ incidentId: 42, degraded: false });
      await AIApi.analyzeIncident(42);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/ai/incidents/42/analyze', {});
    });
  });

  describe('aiSummarize', () => {
    it('should summarize text', async () => {
      mockPost.mockResolvedValue({ answers: ['This is a summary'] });
      const result = await aiSummarize('Long text here', 100);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/ai/chat', expect.objectContaining({ limit: 1 }));
      expect(result.summary).toBe('This is a summary');
    });

    it('should handle empty answers', async () => {
      mockPost.mockResolvedValue({ answers: [] });
      const result = await aiSummarize('text');
      expect(result.summary).toBe('');
    });
  });

  describe('aiSaveFeedback', () => {
    it('should save feedback', async () => {
      mockPost.mockResolvedValue({ message: 'saved' });
      const feedback = { kind: 'triage', useful: true };
      const result = await aiSaveFeedback(feedback);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/ai/feedback', feedback);
      expect(result.message).toBe('saved');
    });
  });

  describe('aiGetMetrics', () => {
    it('should get metrics', async () => {
      mockGet.mockResolvedValue({ totalRequests: 100, usefulRate: 0.8 });
      const result = await aiGetMetrics(14);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/ai/metrics?days=14');
      expect(result.totalRequests).toBe(100);
    });

    it('should default to 7 days', async () => {
      mockGet.mockResolvedValue({});
      await aiGetMetrics();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/ai/metrics?days=7');
    });
  });

  describe('aiGetEvaluation', () => {
    it('should fetch evaluation report with days param', async () => {
      mockGet.mockResolvedValue({ healthScore: 57.5, hasData: true, byScenario: [{ kind: 'triage' }] });
      const result = await aiGetEvaluation(30);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/ai/evaluation?days=30');
      expect(result.healthScore).toBe(57.5);
      expect(result.hasData).toBe(true);
    });

    it('should default to 30 days', async () => {
      mockGet.mockResolvedValue({});
      await aiGetEvaluation();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/ai/evaluation?days=30');
    });
  });

  describe('aiGetAuditLogs', () => {
    it('should fetch audit logs with all params', async () => {
      mockGet.mockResolvedValue({
        items: [{ id: 1, scenario: 'analyze' }],
        total: 3,
        page: 2,
        pageSize: 10,
        totalPages: 1,
      });
      const result = await aiGetAuditLogs({ page: 2, pageSize: 10, kind: 'analyze', days: 90 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/ai/audit-logs?page=2&pageSize=10&kind=analyze&days=90');
      expect(result.items).toHaveLength(1);
      expect(result.total).toBe(3);
    });

    it('should omit empty params', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 20, totalPages: 0 });
      await aiGetAuditLogs();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/ai/audit-logs');
    });

    // 后端只有 common.SuccessWithPagination 一套信封（items/total/page/pageSize/totalPages）。
    // 客户端不得再猜测缺省或重算 totalPages，否则缺省漂移时界面静默少页/多页。
    it('should pass the five-key envelope through unchanged', async () => {
      mockGet.mockResolvedValue({
        items: [{ id: 1 }, { id: 2 }],
        total: 42,
        page: 3,
        pageSize: 20,
        totalPages: 3,
      });
      const result = await aiGetAuditLogs({ page: 3 });
      expect(result).toEqual({
        items: [{ id: 1 }, { id: 2 }],
        total: 42,
        page: 3,
        pageSize: 20,
        totalPages: 3,
      });
    });
  });

  /**
   * 预演 vs 复核：两条不同的端点，必须分别校验 URL。
   *
   * 为什么这组断言非写不可（2026-11-21 实测留下的教训）：
   *   审批页的 `audit-replay.test.tsx` mock 的是 `aiReviewToolInvocation`
   *   **函数符号**，因此它只能锁住「页面把 record.id 传给了复核函数」。
   *   我做过一次反向验证——把本文件的复核实现偷偷改回调
   *   `/api/v1/agent/tools/dry-run`（即退回「按审批人自己的身份判定」，
   *   正是这个功能要消灭的那个缺陷），21 个前端测试**全绿**。
   *
   *   而 `api-contract.test.ts` 也拦不住：`dry-run` 本身是一条真实注册的
   *   后端路由，逐段匹配成功。于是「语义退化为另一个函数」在两层防护之间
   *   形成缝隙——路径合法、函数调用形式也合法，只有**语义**错了。
   *
   *   唯一能看见它的位置就是这里：在 httpClient 这一层断言被调用的 URL
   *   是 invocations/:id/review 而不是 dry-run，且两者不可互换。
   */
  describe('工具调用判定端点（预演 / 复核）', () => {
    it('预演打到 dry-run，且不包含任何记录 id 段', async () => {
      mockPost.mockResolvedValue({ dryRun: true, allowed: true });
      await aiDryRunTool('create_ticket', { title: 't' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/agent/tools/dry-run', {
        name: 'create_ticket',
        args: { title: 't' },
      });
    });

    it('复核打到 invocations/:id/review，并携带记录 id', async () => {
      mockPost.mockResolvedValue({ dryRun: true, allowed: true });
      await aiReviewToolInvocation(42);
      expect(mockPost).toHaveBeenCalledWith(
        '/api/v1/agent/tools/invocations/42/review',
        {}
      );
    });

    // 反向断言：复核绝不能命中 dry-run。写成独立用例而不是只断言上面的
    // 正向相等，是为了让「误退回预演」这条注入失败在一条**具名**断言上，
    // 而不是混在正向期望里让人读不出是哪一种退化。
    it('复核不得退化回 dry-run（退回即按审批人身份判定）', async () => {
      mockPost.mockResolvedValue({ dryRun: true, allowed: true });
      await aiReviewToolInvocation(7);
      const calledPaths = mockPost.mock.calls.map(call => call[0]);
      expect(calledPaths).not.toContain('/api/v1/agent/tools/dry-run');
      expect(calledPaths).toContain('/api/v1/agent/tools/invocations/7/review');
    });

    // 复核不带工具名/参数：判据必须来自被复核记录的 arguments 与 role_snapshot，
    // 由前端重传参数会让「复核」变成披着复核外衣的新预演（前端可以说谎）。
    it('复核不传 name/args，避免前端伪造判定输入', async () => {
      mockPost.mockResolvedValue({ dryRun: true, allowed: true });
      await aiReviewToolInvocation(9);
      expect(mockPost.mock.calls[0][1]).toEqual({});
    });

    it('id 经 encodeURIComponent 编码，不会拼出额外路径段', async () => {
      mockPost.mockResolvedValue({ dryRun: true, allowed: true });
      // 正常数字 id 不应被破坏
      await aiReviewToolInvocation(12345);
      expect(mockPost.mock.calls[0][0]).toBe(
        '/api/v1/agent/tools/invocations/12345/review'
      );
    });
  });

  describe('AIApi class wrapper', () => {
    it('triage should delegate', async () => {
      mockPost.mockResolvedValue({ suggestions: { category: 'test' } });
      const result = await AIApi.triage('title', 'desc');
      expect(result.category).toBe('test');
    });

    it('chat should call post', async () => {
      mockPost.mockResolvedValue({ answers: [] });
      await AIApi.chat({ query: 'hello', limit: 5 });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/ai/chat', { query: 'hello', limit: 5, conversationId: undefined });
    });

    it('searchKB should delegate', async () => {
      mockPost.mockResolvedValue({ items: [] });
      const result = await AIApi.searchKB('test');
      expect(result.answers).toEqual([]);
    });

    it('similarIncidents should delegate', async () => {
      mockPost.mockResolvedValue({ items: [] });
      const result = await AIApi.similarIncidents('test');
      expect(result.incidents).toEqual([]);
    });

    it('summarize should delegate', async () => {
      mockPost.mockResolvedValue({ answers: ['sum'] });
      const result = await AIApi.summarize('text');
      expect(result.summary).toBe('sum');
    });

    it('saveFeedback should delegate', async () => {
      mockPost.mockResolvedValue({ message: 'ok' });
      await AIApi.saveFeedback({ kind: 'test', useful: true });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/ai/feedback', { kind: 'test', useful: true });
    });

    it('getMetrics should delegate', async () => {
      mockGet.mockResolvedValue({ totalRequests: 50 });
      const result = await AIApi.getMetrics(30);
      expect(result.totalRequests).toBe(50);
    });
  });

  describe('aiClassifyTicket (merged from legacy AIService)', () => {
    it('should classify with suggestions mapping', async () => {
      mockPost.mockResolvedValue({
        suggestions: { category: 'network', priority: 'high', confidence: 0.88, reasoning: 'network down', urgency: 'urgent' },
      });
      const result = await aiClassifyTicket({ title: 'Test', description: 'Desc' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/ai/triage', { title: 'Test', description: 'Desc' });
      expect(result.category).toBe('network');
      expect(result.priority).toBe('high');
      expect(result.urgency).toBe('urgent');
      expect(result.confidence).toBe(0.88);
      expect(result.reasoning).toBe('network down');
    });

    it('should default urgency to priority and handle missing suggestions', async () => {
      mockPost.mockResolvedValue({ suggestions: { category: 'software', priority: 'medium' } });
      const result = await aiClassifyTicket({ title: 'Test', description: 'Desc' });
      expect(result.category).toBe('software');
      expect(result.urgency).toBe('medium');
    });

    it('should fall back to general/medium on empty response', async () => {
      mockPost.mockResolvedValue({});
      const result = await aiClassifyTicket({ title: 'Test', description: 'Desc' });
      expect(result.category).toBe('general');
      expect(result.priority).toBe('medium');
      expect(result.confidence).toBe(0);
    });
  });

  describe('aiSuggestSolutions (merged from legacy AIService)', () => {
    const answers = [
      { id: 1, title: '重置密码指南', snippet: '通过SSO重置', score: 0.9, source: 'kb/1' },
      { id: 2, title: 'VPN连接排查', snippet: '检查客户端版本', score: 0.7 },
    ];

    it('should map rag results to solution suggestions', async () => {
      mockPost.mockResolvedValue({ items: answers });
      const result = await aiSuggestSolutions({ query: 'password reset', limit: 5 });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/ai/rag/search', { query: 'password reset', limit: 5, type: 'kb' });
      expect(result).toHaveLength(2);
      expect(result[0].solutionId).toBe('1');
      expect(result[0].title).toBe('重置密码指南');
      expect(result[0].description).toBe('通过SSO重置');
      expect(result[0].relatedKnowledge).toEqual(['kb/1']);
    });

    it('should default limit to 5 when not provided or invalid', async () => {
      mockPost.mockResolvedValue({ items: [] });
      await aiSuggestSolutions({ query: 'test', limit: 0 });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/ai/rag/search', { query: 'test', limit: 5, type: 'kb' });
    });

    it('should handle non-array response', async () => {
      mockPost.mockResolvedValue(null);
      const result = await aiSuggestSolutions({ query: 'test' });
      expect(result).toEqual([]);
    });
  });

  describe('aiIntelligentSearch (merged from legacy AIService)', () => {
    it('should split results by type', async () => {
      mockGet.mockResolvedValue({
        results: [
          { id: 1, type: 'ticket', title: 'T1', ticketNumber: 'TCK-1' },
          { id: 2, type: 'incident', title: 'I1' },
          { id: 3, type: 'knowledge', title: 'K1' },
          { id: 4, type: 'unknown', title: 'X1' },
        ],
      });
      const result = await aiIntelligentSearch('login issue');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/global-search?q=login%20issue');
      expect(result.tickets).toHaveLength(1);
      expect(result.tickets[0].number).toBe('TCK-1');
      expect(result.incidents).toHaveLength(1);
      expect(result.knowledge).toHaveLength(1);
    });

    it('should handle empty results', async () => {
      mockGet.mockResolvedValue({ results: [] });
      const result = await aiIntelligentSearch('nothing');
      expect(result.tickets).toEqual([]);
      expect(result.suggestions).toEqual([]);
    });
  });

  describe('error propagation', () => {
    it('should propagate errors', async () => {
      mockPost.mockRejectedValue(new Error('AI service unavailable'));
      await expect(aiTriage('t', 'd')).rejects.toThrow('AI service unavailable');
    });
  });

  describe('aiChatStream', () => {
    let originalFetch: typeof global.fetch;
    const encode = (str: string) => Buffer.from(str);

    beforeEach(() => {
      originalFetch = global.fetch;
      // Polyfill TextDecoder for jsdom
      if (typeof TextDecoder === 'undefined') {
        (global as any).TextDecoder = class { decode(buf: any) { return Buffer.from(buf).toString('utf-8'); } };
      }
    });
    afterEach(() => {
      global.fetch = originalFetch;
    });

    it('should stream SSE events and invoke callbacks', async () => {
      const { aiChatStream } = require('../ai-api');
      const chunks = [
        encode('event:sources\ndata:[{"objectType":"article","id":1,"snippet":"hi"}]\n\n'),
        encode('event:delta\ndata:{"content":"Hello"}\n\n'),
        encode('event:done\ndata:{"conversationId":42}\n\n'),
      ];
      let chunkIndex = 0;
      const mockReader = {
        read: jest.fn().mockImplementation(() => {
          if (chunkIndex < chunks.length) {
            return Promise.resolve({ value: chunks[chunkIndex++], done: false });
          }
          return Promise.resolve({ value: undefined, done: true });
        }),
      };
      const mockBody = { getReader: () => mockReader };
      global.fetch = jest.fn().mockResolvedValue({ ok: true, status: 200, body: mockBody });

      const onSources = jest.fn();
      const onDelta = jest.fn();
      const onDone = jest.fn();
      const result = await aiChatStream({ query: 'test', limit: 5 }, { onSources, onDelta, onDone });

      expect(onSources).toHaveBeenCalledWith([expect.objectContaining({ id: 1 })]);
      expect(onDelta).toHaveBeenCalledWith('Hello');
      expect(onDone).toHaveBeenCalledWith(42);
      expect(result).toBe(42);
    });

    it('should handle HTTP error', async () => {
      const { aiChatStream } = require('../ai-api');
      global.fetch = jest.fn().mockResolvedValue({ ok: false, status: 500, body: null });
      const onError = jest.fn();
      await expect(aiChatStream({ query: 'test' }, { onError })).rejects.toThrow();
      expect(onError).toHaveBeenCalled();
    });

    it('should handle error event in stream', async () => {
      const { aiChatStream } = require('../ai-api');
      const chunks = [
        encode('event:error\ndata:{"message":"rate limited"}\n\n'),
      ];
      let chunkIndex = 0;
      const mockReader = {
        read: jest.fn().mockImplementation(() => {
          if (chunkIndex < chunks.length) {
            return Promise.resolve({ value: chunks[chunkIndex++], done: false });
          }
          return Promise.resolve({ value: undefined, done: true });
        }),
      };
      global.fetch = jest.fn().mockResolvedValue({ ok: true, status: 200, body: { getReader: () => mockReader } });
      const onError = jest.fn();
      await aiChatStream({ query: 'test' }, { onError });
      expect(onError).toHaveBeenCalledWith('rate limited');
    });

    it('should handle invalid JSON in SSE data gracefully', async () => {
      const { aiChatStream } = require('../ai-api');
      const chunks = [
        encode('event:delta\ndata:not-json\n\n'),
        encode('event:done\ndata:{"conversationId":7}\n\n'),
      ];
      let chunkIndex = 0;
      const mockReader = {
        read: jest.fn().mockImplementation(() => {
          if (chunkIndex < chunks.length) {
            return Promise.resolve({ value: chunks[chunkIndex++], done: false });
          }
          return Promise.resolve({ value: undefined, done: true });
        }),
      };
      global.fetch = jest.fn().mockResolvedValue({ ok: true, status: 200, body: { getReader: () => mockReader } });
      const onDelta = jest.fn();
      const result = await aiChatStream({ query: 'test' }, { onDelta });
      expect(onDelta).not.toHaveBeenCalled();
      expect(result).toBe(7);
    });
  });

  describe('AIApi.chatStream', () => {
    it('should delegate to aiChatStream', async () => {
      const encode = (str: string) => Buffer.from(str);
      const chunks = [encode('event:done\ndata:{"conversationId":99}\n\n')];
      let chunkIndex = 0;
      const mockReader = {
        read: jest.fn().mockImplementation(() => {
          if (chunkIndex < chunks.length) {
            return Promise.resolve({ value: chunks[chunkIndex++], done: false });
          }
          return Promise.resolve({ value: undefined, done: true });
        }),
      };
      global.fetch = jest.fn().mockResolvedValue({ ok: true, status: 200, body: { getReader: () => mockReader } });
      const result = await AIApi.chatStream({ query: 'hello' });
      expect(result).toBe(99);
    });
  });
});
