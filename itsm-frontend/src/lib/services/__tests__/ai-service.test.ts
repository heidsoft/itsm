import { AIService } from '../ai-service';
import { httpClient } from '@/lib/api/http-client';

jest.mock('@/lib/api/http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
    patch: jest.fn(),
  },
}));

jest.mock('@/lib/env', () => ({
  logger: {
    debug: jest.fn(),
    error: jest.fn(),
    info: jest.fn(),
    warn: jest.fn(),
  },
}));

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;

describe('AIService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('classifyTicket', () => {
    it('should classify ticket and return structured result', async () => {
      const apiResponse = {
        title: 'Cannot login',
        description: 'User unable to login',
        suggestions: {
          category: 'authentication',
          priority: 'high',
          confidence: 0.92,
          reasoning: 'Authentication issue detected',
          urgency: 'high',
        },
      };
      mockPost.mockResolvedValue(apiResponse);

      const result = await AIService.classifyTicket({
        title: 'Cannot login',
        description: 'User unable to login',
      });

      expect(mockPost).toHaveBeenCalledWith('/api/v1/ai/triage', {
        title: 'Cannot login',
        description: 'User unable to login',
      });
      expect(result.category).toBe('authentication');
      expect(result.priority).toBe('high');
      expect(result.urgency).toBe('high');
      expect(result.confidence).toBe(0.92);
      expect(result.reasoning).toBe('Authentication issue detected');
    });

    it('should use defaults when suggestions are empty', async () => {
      mockPost.mockResolvedValue({ title: 'Test', description: 'Desc' });

      const result = await AIService.classifyTicket({
        title: 'Test',
        description: 'Desc',
      });

      expect(result.category).toBe('general');
      expect(result.priority).toBe('medium');
      expect(result.confidence).toBe(0);
      expect(result.reasoning).toBe('');
    });

    it('should throw error on API failure', async () => {
      mockPost.mockRejectedValue(new Error('Network error'));

      await expect(
        AIService.classifyTicket({ title: 'Test', description: 'Desc' })
      ).rejects.toThrow('Network error');
    });

    it('should wrap non-Error exceptions', async () => {
      mockPost.mockRejectedValue('string error');

      await expect(
        AIService.classifyTicket({ title: 'Test', description: 'Desc' })
      ).rejects.toThrow('AI分析失败，请稍后重试');
    });
  });

  describe('suggestSolutions', () => {
    it('should return mapped solution suggestions', async () => {
      const answers = [
        { id: 'kb-1', title: 'Reset Password Guide', snippet: 'Step 1: Go to settings', score: 0.85, source: 'KB-001' },
        { id: 'kb-2', title: 'Account Recovery', content: 'Use recovery email', score: 0.7 },
      ];
      mockPost.mockResolvedValue(answers);

      const result = await AIService.suggestSolutions({ query: 'password reset', limit: 5 });

      expect(mockPost).toHaveBeenCalledWith('/api/v1/ai/knowledge/search', {
        query: 'password reset',
        limit: 5,
        type: 'kb',
      });
      expect(result).toHaveLength(2);
      expect(result[0].title).toBe('Reset Password Guide');
      expect(result[0].description).toBe('Step 1: Go to settings');
      expect(result[0].confidence).toBe(0.85);
      expect(result[0].relatedKnowledge).toContain('KB-001');
      expect(result[1].title).toBe('Account Recovery');
      expect(result[1].description).toBe('Use recovery email');
    });

    it('should handle non-array response', async () => {
      mockPost.mockResolvedValue(null);

      const result = await AIService.suggestSolutions({ query: 'test' });

      expect(result).toEqual([]);
    });

    it('should default limit to 5 when not provided or invalid', async () => {
      mockPost.mockResolvedValue([]);

      await AIService.suggestSolutions({ query: 'test', limit: 0 });

      expect(mockPost).toHaveBeenCalledWith('/api/v1/ai/knowledge/search', {
        query: 'test',
        limit: 5,
        type: 'kb',
      });
    });

    it('should throw error on API failure', async () => {
      mockPost.mockRejectedValue(new Error('Search failed'));

      await expect(AIService.suggestSolutions({ query: 'test' })).rejects.toThrow('Search failed');
    });
  });

  describe('intelligentSearch', () => {
    it('should categorize search results by type', async () => {
      const apiResponse = {
        results: [
          { id: 1, type: 'ticket', title: 'Login Issue', description: 'Cannot login', status: 'open', ticketNumber: 'TKT-001' },
          { id: 2, type: 'incident', title: 'Server Down', description: 'Production outage' },
          { id: 3, type: 'knowledge', title: 'FAQ', description: 'Common questions' },
        ],
        total: 3,
      };
      mockGet.mockResolvedValue(apiResponse);

      const result = await AIService.intelligentSearch('login issue');

      expect(mockGet).toHaveBeenCalledWith('/api/v1/global-search?keyword=login%20issue');
      expect(result.tickets).toHaveLength(1);
      expect(result.tickets[0].number).toBe('TKT-001');
      expect(result.incidents).toHaveLength(1);
      expect(result.knowledge).toHaveLength(1);
      expect(result.suggestions).toEqual([]);
    });

    it('should handle empty results', async () => {
      mockGet.mockResolvedValue({ results: [], total: 0 });

      const result = await AIService.intelligentSearch('nothing');

      expect(result.tickets).toEqual([]);
      expect(result.incidents).toEqual([]);
      expect(result.knowledge).toEqual([]);
    });

    it('should handle null results gracefully', async () => {
      mockGet.mockResolvedValue({ results: null, total: 0 });

      const result = await AIService.intelligentSearch('test');

      expect(result.tickets).toEqual([]);
      expect(result.incidents).toEqual([]);
      expect(result.knowledge).toEqual([]);
    });

    it('should throw error on API failure', async () => {
      mockGet.mockRejectedValue(new Error('Search API error'));

      await expect(AIService.intelligentSearch('test')).rejects.toThrow('Search API error');
    });

    it('should wrap non-Error exceptions', async () => {
      mockGet.mockRejectedValue('unexpected');

      await expect(AIService.intelligentSearch('test')).rejects.toThrow('搜索失败，请稍后重试');
    });
  });
});
