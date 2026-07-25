import { PriorityMatrixApi } from '../priority-matrix-api';
import { httpClient } from '../http-client';

jest.mock('../http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
    patch: jest.fn(),
    request: jest.fn(),
  },
}));

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;
const mockPut = httpClient.put as jest.Mock;
const mockDelete = httpClient.delete as jest.Mock;
const mockRequest = httpClient.request as jest.Mock;

describe('PriorityMatrixApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('calculatePriority', () => {
    it('should calculate priority', async () => {
      mockPost.mockResolvedValue({ priority: 'high', score: 85 });
      await PriorityMatrixApi.calculatePriority({ urgency: 'high', impact: 'high' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/priority/calculate', { urgency: 'high', impact: 'high' });
    });
  });

  describe('batchCalculatePriority', () => {
    it('should batch calculate', async () => {
      mockPost.mockResolvedValue([{ priority: 'high' }]);
      await PriorityMatrixApi.batchCalculatePriority([{ urgency: 'high', impact: 'low' }] as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/priority/batch-calculate', { requests: [{ urgency: 'high', impact: 'low' }] });
    });
  });

  describe('getPrioritySuggestion', () => {
    it('should get suggestion', async () => {
      mockGet.mockResolvedValue({ suggestedPriority: 'medium' });
      await PriorityMatrixApi.getPrioritySuggestion(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/1/priority-suggestion');
    });
  });

  describe('getBatchPrioritySuggestions', () => {
    it('should get batch suggestions', async () => {
      mockPost.mockResolvedValue({ suggestions: [] });
      await PriorityMatrixApi.getBatchPrioritySuggestions([1, 2]);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/priority/batch-suggestions', { ticketIds: [1, 2] });
    });
  });

  describe('getMatrixConfigs', () => {
    it('should get configs', async () => {
      mockGet.mockResolvedValue([]);
      await PriorityMatrixApi.getMatrixConfigs();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/priority/matrix-configs');
    });
  });

  describe('getActiveMatrixConfig', () => {
    it('should get active config', async () => {
      mockGet.mockResolvedValue({ id: '1', isActive: true });
      await PriorityMatrixApi.getActiveMatrixConfig();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/priority/matrix-configs/active');
    });
  });

  describe('getMatrixData', () => {
    it('should get matrix data', async () => {
      mockGet.mockResolvedValue({ matrix: [] });
      await PriorityMatrixApi.getMatrixData('c1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/priority/matrix-data', { configId: 'c1' });
    });
  });

  describe('createMatrixConfig', () => {
    it('should create config', async () => {
      mockPost.mockResolvedValue({ id: '1' });
      await PriorityMatrixApi.createMatrixConfig({ name: 'New' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/priority/matrix-configs', { name: 'New' });
    });
  });

  describe('updateMatrixConfig', () => {
    it('should update config', async () => {
      mockPut.mockResolvedValue({ id: '1' });
      await PriorityMatrixApi.updateMatrixConfig('1', { name: 'Updated' } as any);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/priority/matrix-configs/1', { name: 'Updated' });
    });
  });

  describe('deleteMatrixConfig', () => {
    it('should delete config', async () => {
      mockDelete.mockResolvedValue(undefined);
      await PriorityMatrixApi.deleteMatrixConfig('1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/priority/matrix-configs/1');
    });
  });

  describe('activateMatrixConfig', () => {
    it('should activate config', async () => {
      mockPost.mockResolvedValue(undefined);
      await PriorityMatrixApi.activateMatrixConfig('1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/priority/matrix-configs/1/activate');
    });
  });

  describe('getPriorityRules', () => {
    it('should get rules', async () => {
      mockGet.mockResolvedValue({ rules: [], total: 0 });
      await PriorityMatrixApi.getPriorityRules({ page: 1 } as any);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/priority/rules', { page: 1 });
    });
  });

  describe('getPriorityRule', () => {
    it('should get rule', async () => {
      mockGet.mockResolvedValue({ id: 'r1' });
      await PriorityMatrixApi.getPriorityRule('r1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/priority/rules/r1');
    });
  });

  describe('createPriorityRule', () => {
    it('should create rule', async () => {
      mockPost.mockResolvedValue({ id: 'r1' });
      await PriorityMatrixApi.createPriorityRule({ name: 'Rule1' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/priority/rules', { name: 'Rule1' });
    });
  });

  describe('updatePriorityRule', () => {
    it('should update rule', async () => {
      mockPut.mockResolvedValue({ id: 'r1' });
      await PriorityMatrixApi.updatePriorityRule('r1', { name: 'Updated' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/priority/rules/r1', { name: 'Updated' });
    });
  });

  describe('deletePriorityRule', () => {
    it('should delete rule', async () => {
      mockDelete.mockResolvedValue(undefined);
      await PriorityMatrixApi.deletePriorityRule('r1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/priority/rules/r1');
    });
  });

  describe('togglePriorityRule', () => {
    it('should toggle rule', async () => {
      mockPost.mockResolvedValue(undefined);
      await PriorityMatrixApi.togglePriorityRule('r1', true);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/priority/rules/r1/toggle', { enabled: true });
    });
  });

  describe('testPriorityRule', () => {
    it('should test rule', async () => {
      mockPost.mockResolvedValue({ matched: true, actions: ['set_priority'] });
      await PriorityMatrixApi.testPriorityRule('r1', 1);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/priority/rules/r1/test', { ticketId: 1 });
    });
  });

  describe('getPriorityHistory', () => {
    it('should get history', async () => {
      mockGet.mockResolvedValue({ history: [], total: 0 });
      await PriorityMatrixApi.getPriorityHistory({ ticketId: 1 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/priority/history', { ticketId: 1 });
    });
  });

  describe('getPriorityDistribution', () => {
    it('should get distribution', async () => {
      mockGet.mockResolvedValue({ distribution: {} });
      await PriorityMatrixApi.getPriorityDistribution({ startDate: '2024-01-01', endDate: '2024-01-31' } as any);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/priority/analysis/distribution', expect.any(Object));
    });
  });

  describe('getPriorityAccuracy', () => {
    it('should get accuracy', async () => {
      mockGet.mockResolvedValue({ accuracy: 0.9 });
      await PriorityMatrixApi.getPriorityAccuracy({ startDate: '2024-01-01', endDate: '2024-01-31' } as any);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/priority/analysis/accuracy', expect.any(Object));
    });
  });

  describe('exportPriorityReport', () => {
    it('should export report', async () => {
      const blob = new Blob(['data']);
      mockRequest.mockResolvedValue(blob);
      const result = await PriorityMatrixApi.exportPriorityReport({ format: 'pdf', startDate: '2024-01-01', endDate: '2024-01-31' });
      expect(mockRequest).toHaveBeenCalledWith(expect.objectContaining({ method: 'POST', url: '/api/v1/priority/export-report', responseType: 'blob' }));
      expect(result).toBe(blob);
    });
  });

  describe('error propagation', () => {
    it('should propagate errors', async () => {
      mockPost.mockRejectedValue(new Error('Server error'));
      await expect(PriorityMatrixApi.calculatePriority({} as any)).rejects.toThrow('Server error');
    });
  });
});
