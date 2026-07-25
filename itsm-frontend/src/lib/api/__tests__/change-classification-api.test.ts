import { ChangeClassificationApi } from '../change-classification-api';
import { httpClient } from '../http-client';

jest.mock('../http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
    patch: jest.fn(),
  },
}));

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;
const mockPut = httpClient.put as jest.Mock;
const mockDelete = httpClient.delete as jest.Mock;

describe('ChangeClassificationApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getClassifications', () => {
    it('should get classifications', async () => {
      mockGet.mockResolvedValue({ classifications: [], total: 0 });
      await ChangeClassificationApi.getClassifications({ page: 1 } as any);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/change-classifications', { page: 1 });
    });
  });

  describe('getClassification', () => {
    it('should get by id', async () => {
      mockGet.mockResolvedValue({ id: '1', name: 'Standard' });
      await ChangeClassificationApi.getClassification('1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/change-classifications/1');
    });
  });

  describe('createClassification', () => {
    it('should create', async () => {
      mockPost.mockResolvedValue({ id: '1' });
      await ChangeClassificationApi.createClassification({ name: 'New' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/change-classifications', { name: 'New' });
    });
  });

  describe('updateClassification', () => {
    it('should update', async () => {
      mockPut.mockResolvedValue({ id: '1' });
      await ChangeClassificationApi.updateClassification('1', { name: 'Updated' } as any);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/change-classifications/1', { name: 'Updated' });
    });
  });

  describe('deleteClassification', () => {
    it('should delete', async () => {
      mockDelete.mockResolvedValue(undefined);
      await ChangeClassificationApi.deleteClassification('1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/change-classifications/1');
    });
  });

  describe('assessRisk', () => {
    it('should assess risk', async () => {
      mockPost.mockResolvedValue({ riskLevel: 'medium' });
      await ChangeClassificationApi.assessRisk({ changeId: 1 } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/changes/assess-risk', { changeId: 1 });
    });
  });

  describe('getRiskAssessmentHistory', () => {
    it('should get history', async () => {
      mockGet.mockResolvedValue([]);
      await ChangeClassificationApi.getRiskAssessmentHistory(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/changes/1/risk-assessments');
    });
  });

  describe('analyzeImpact', () => {
    it('should analyze impact', async () => {
      mockPost.mockResolvedValue({ impactLevel: 'high' });
      await ChangeClassificationApi.analyzeImpact({ changeId: 1 } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/changes/analyze-impact', { changeId: 1 });
    });
  });

  describe('getImpactAnalysisHistory', () => {
    it('should get impact history', async () => {
      mockGet.mockResolvedValue([]);
      await ChangeClassificationApi.getImpactAnalysisHistory(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/changes/1/impact-analyses');
    });
  });

  describe('getClassificationSuggestion', () => {
    it('should get suggestion', async () => {
      mockGet.mockResolvedValue({ suggestedClassification: 'standard' });
      await ChangeClassificationApi.getClassificationSuggestion(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/changes/1/classification-suggestion');
    });
  });

  describe('applyClassificationSuggestion', () => {
    it('should apply suggestion', async () => {
      mockPost.mockResolvedValue(undefined);
      await ChangeClassificationApi.applyClassificationSuggestion(1, 'c1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/changes/1/apply-classification', { classificationId: 'c1' });
    });
  });

  describe('getClassificationRules', () => {
    it('should get rules', async () => {
      mockGet.mockResolvedValue([]);
      await ChangeClassificationApi.getClassificationRules();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/classification-rules');
    });
  });

  describe('createClassificationRule', () => {
    it('should create rule', async () => {
      mockPost.mockResolvedValue({ id: '1' });
      await ChangeClassificationApi.createClassificationRule({ name: 'Rule1' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/classification-rules', { name: 'Rule1' });
    });
  });

  describe('updateClassificationRule', () => {
    it('should update rule', async () => {
      mockPut.mockResolvedValue({ id: '1' });
      await ChangeClassificationApi.updateClassificationRule('1', { name: 'Updated' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/classification-rules/1', { name: 'Updated' });
    });
  });

  describe('deleteClassificationRule', () => {
    it('should delete rule', async () => {
      mockDelete.mockResolvedValue(undefined);
      await ChangeClassificationApi.deleteClassificationRule('1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/classification-rules/1');
    });
  });

  describe('getChangeTemplates', () => {
    it('should get templates', async () => {
      mockGet.mockResolvedValue([]);
      await ChangeClassificationApi.getChangeTemplates({ classificationId: 'c1' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/change-templates', { classificationId: 'c1' });
    });
  });

  describe('getChangeTemplate', () => {
    it('should get template', async () => {
      mockGet.mockResolvedValue({ id: '1' });
      await ChangeClassificationApi.getChangeTemplate('1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/change-templates/1');
    });
  });

  describe('createChangeTemplate', () => {
    it('should create template', async () => {
      mockPost.mockResolvedValue({ id: '1' });
      await ChangeClassificationApi.createChangeTemplate({ name: 'T1' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/change-templates', { name: 'T1' });
    });
  });

  describe('getApprovalMatrix', () => {
    it('should get approval matrix', async () => {
      mockGet.mockResolvedValue({ levels: [] });
      await ChangeClassificationApi.getApprovalMatrix('c1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/change-classifications/c1/approval-matrix');
    });
  });

  describe('updateApprovalMatrix', () => {
    it('should update approval matrix', async () => {
      mockPut.mockResolvedValue({ levels: [] });
      await ChangeClassificationApi.updateApprovalMatrix('c1', { levels: [] } as any);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/change-classifications/c1/approval-matrix', { levels: [] });
    });
  });

  describe('getClassificationStats', () => {
    it('should get stats', async () => {
      mockGet.mockResolvedValue([]);
      await ChangeClassificationApi.getClassificationStats({ startDate: '2024-01-01', endDate: '2024-01-31' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/change-classifications/stats', { startDate: '2024-01-01', endDate: '2024-01-31' });
    });
  });

  describe('getClassificationAnalysis', () => {
    it('should get analysis', async () => {
      mockGet.mockResolvedValue({ totalChanges: 50 });
      await ChangeClassificationApi.getClassificationAnalysis({ startDate: '2024-01-01', endDate: '2024-01-31', groupBy: 'week' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/change-classifications/analysis', expect.any(Object));
    });
  });

  describe('getClassificationHistory', () => {
    it('should get history', async () => {
      mockGet.mockResolvedValue([]);
      await ChangeClassificationApi.getClassificationHistory(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/changes/1/classification-history');
    });
  });

  describe('error propagation', () => {
    it('should propagate errors', async () => {
      mockGet.mockRejectedValue(new Error('Not found'));
      await expect(ChangeClassificationApi.getClassification('999')).rejects.toThrow('Not found');
    });
  });
});
