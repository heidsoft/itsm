import { aiTriage, aiSearchKB, aiSimilarIncidents, aiSummarize, aiSaveFeedback, aiGetMetrics, AIApi } from '../ai-api';
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
      mockPost.mockResolvedValue([{ id: 1, objectType: 'article', snippet: 'found' }]);
      const result = await aiSearchKB('password reset', 3);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/ai/knowledge/search', { query: 'password reset', limit: 3, type: 'kb' });
      expect(result.answers).toHaveLength(1);
    });

    it('should handle non-array response', async () => {
      mockPost.mockResolvedValue(null);
      const result = await aiSearchKB('test');
      expect(result.answers).toEqual([]);
    });
  });

  describe('aiSimilarIncidents', () => {
    it('should find similar incidents', async () => {
      mockPost.mockResolvedValue([{ id: 1, objectType: 'incident', snippet: 'similar' }]);
      const result = await aiSimilarIncidents('server crash', 5);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/ai/knowledge/search', { query: 'server crash', limit: 5, type: 'incident' });
      expect(result.incidents).toHaveLength(1);
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
      mockPost.mockResolvedValue([]);
      const result = await AIApi.searchKB('test');
      expect(result.answers).toEqual([]);
    });

    it('similarIncidents should delegate', async () => {
      mockPost.mockResolvedValue([]);
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
