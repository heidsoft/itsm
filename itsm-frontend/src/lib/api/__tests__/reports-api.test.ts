import { ReportsApi } from '../reports-api';
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
const mockPatch = httpClient.patch as jest.Mock;
const mockRequest = httpClient.request as jest.Mock;

describe('ReportsApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getReports', () => {
    it('should get reports', async () => {
      mockGet.mockResolvedValue({ reports: [], total: 0 });
      await ReportsApi.getReports({ page: 1 } as any);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/reports', { page: 1 });
    });
  });

  describe('getReport', () => {
    it('should get report', async () => {
      mockGet.mockResolvedValue({ id: '1', name: 'R1' });
      await ReportsApi.getReport('1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/reports/1');
    });
  });

  describe('createReport', () => {
    it('should create report', async () => {
      mockPost.mockResolvedValue({ id: '1' });
      await ReportsApi.createReport({ name: 'New' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/reports', { name: 'New' });
    });
  });

  describe('updateReport', () => {
    it('should update report', async () => {
      mockPut.mockResolvedValue({ id: '1' });
      await ReportsApi.updateReport('1', { name: 'Updated' } as any);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/reports/1', { name: 'Updated' });
    });
  });

  describe('deleteReport', () => {
    it('should delete report', async () => {
      mockDelete.mockResolvedValue(undefined);
      await ReportsApi.deleteReport('1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/reports/1');
    });
  });

  describe('cloneReport', () => {
    it('should clone report', async () => {
      mockPost.mockResolvedValue({ id: '2' });
      await ReportsApi.cloneReport('1', 'Copy');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/reports/1/clone', { name: 'Copy' });
    });
  });

  describe('updateReportStatus', () => {
    it('should update status', async () => {
      mockPatch.mockResolvedValue({ id: '1' });
      await ReportsApi.updateReportStatus('1', 'archived');
      expect(mockPatch).toHaveBeenCalledWith('/api/v1/reports/1/status', { status: 'archived' });
    });
  });

  describe('executeReport', () => {
    it('should execute report', async () => {
      mockPost.mockResolvedValue({ executionId: 'e1' });
      await ReportsApi.executeReport({ reportId: '1' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/reports/execute', { reportId: '1' });
    });
  });

  describe('getExecutionHistory', () => {
    it('should get execution history', async () => {
      mockGet.mockResolvedValue({ executions: [], total: 0 });
      await ReportsApi.getExecutionHistory('1', { page: 1 });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/reports/1/executions', { page: 1 });
    });
  });

  describe('getExecutionResult', () => {
    it('should get result', async () => {
      mockGet.mockResolvedValue({ id: 'e1' });
      await ReportsApi.getExecutionResult('e1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/reports/executions/e1');
    });
  });

  describe('cancelExecution', () => {
    it('should cancel', async () => {
      mockPost.mockResolvedValue(undefined);
      await ReportsApi.cancelExecution('e1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/reports/executions/e1/cancel');
    });
  });

  describe('exportReport', () => {
    it('should export', async () => {
      const blob = new Blob(['data']);
      mockRequest.mockResolvedValue(blob);
      const result = await ReportsApi.exportReport({ reportId: '1', format: 'pdf' });
      expect(mockRequest).toHaveBeenCalledWith(expect.objectContaining({ method: 'POST', url: '/api/v1/reports/1/export', responseType: 'blob' }));
      expect(result).toBe(blob);
    });
  });

  describe('emailReport', () => {
    it('should email report', async () => {
      mockPost.mockResolvedValue(undefined);
      await ReportsApi.emailReport({ reportId: '1', recipients: ['a@b.com'], format: 'pdf' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/reports/1/email', expect.objectContaining({ recipients: ['a@b.com'] }));
    });
  });

  describe('createSchedule', () => {
    it('should create schedule', async () => {
      mockPost.mockResolvedValue({ id: '1' });
      await ReportsApi.createSchedule('1', { enabled: true, frequency: 'daily', recipients: [], outputFormat: 'pdf' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/reports/1/schedule', expect.any(Object));
    });
  });

  describe('updateSchedule', () => {
    it('should update schedule', async () => {
      mockPut.mockResolvedValue({ id: '1' });
      await ReportsApi.updateSchedule('1', { enabled: true, frequency: 'weekly', recipients: [], outputFormat: 'csv' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/reports/1/schedule', expect.any(Object));
    });
  });

  describe('deleteSchedule', () => {
    it('should delete schedule', async () => {
      mockDelete.mockResolvedValue(undefined);
      await ReportsApi.deleteSchedule('1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/reports/1/schedule');
    });
  });

  describe('runScheduleNow', () => {
    it('should run schedule now', async () => {
      mockPost.mockResolvedValue({ executionId: 'e1' });
      await ReportsApi.runScheduleNow('1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/reports/1/schedule/run');
    });
  });

  describe('getTemplates', () => {
    it('should get templates', async () => {
      mockGet.mockResolvedValue({ templates: [], total: 0 });
      await ReportsApi.getTemplates({ category: 'it' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/reports/templates', { category: 'it' });
    });
  });

  describe('getTemplate', () => {
    it('should get template', async () => {
      mockGet.mockResolvedValue({ id: '1' });
      await ReportsApi.getTemplate('1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/reports/templates/1');
    });
  });

  describe('createFromTemplate', () => {
    it('should create from template', async () => {
      mockPost.mockResolvedValue({ id: '2' });
      await ReportsApi.createFromTemplate('t1', 'New Report');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/reports/templates/t1/create', { name: 'New Report' });
    });
  });

  describe('saveAsTemplate', () => {
    it('should save as template', async () => {
      mockPost.mockResolvedValue({ id: 't1' });
      await ReportsApi.saveAsTemplate('1', { name: 'Template', category: 'IT' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/reports/1/save-as-template', { name: 'Template', category: 'IT' });
    });
  });

  describe('getDatasets', () => {
    it('should get datasets', async () => {
      mockGet.mockResolvedValue([]);
      await ReportsApi.getDatasets();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/reports/datasets');
    });
  });

  describe('previewData', () => {
    it('should preview data', async () => {
      mockPost.mockResolvedValue({ columns: [], rows: [], total: 0 });
      await ReportsApi.previewData({ dataSource: { type: 'query' }, limit: 10 });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/reports/preview', expect.any(Object));
    });
  });

  describe('validateQuery', () => {
    it('should validate query', async () => {
      mockPost.mockResolvedValue({ valid: true });
      await ReportsApi.validateQuery('SELECT * FROM tickets');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/reports/validate-query', { query: 'SELECT * FROM tickets' });
    });
  });

  describe('getStats', () => {
    it('should get stats', async () => {
      mockGet.mockResolvedValue({ totalReports: 10 });
      await ReportsApi.getStats();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/reports/stats');
    });
  });

  describe('getPerformance', () => {
    it('should get performance', async () => {
      mockGet.mockResolvedValue({ avgExecutionTime: 5 });
      await ReportsApi.getPerformance('1', { startDate: '2024-01-01' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/reports/1/performance', { startDate: '2024-01-01' });
    });
  });

  describe('favoriteReport', () => {
    it('should favorite', async () => {
      mockPost.mockResolvedValue(undefined);
      await ReportsApi.favoriteReport('1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/reports/1/favorite');
    });
  });

  describe('unfavoriteReport', () => {
    it('should unfavorite', async () => {
      mockDelete.mockResolvedValue(undefined);
      await ReportsApi.unfavoriteReport('1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/reports/1/favorite');
    });
  });

  describe('shareReport', () => {
    it('should share report', async () => {
      mockPost.mockResolvedValue(undefined);
      await ReportsApi.shareReport('1', { userIds: [1, 2], permission: 'view' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/reports/1/share', { userIds: [1, 2], permission: 'view' });
    });
  });

  describe('unshareReport', () => {
    it('should unshare', async () => {
      mockDelete.mockResolvedValue(undefined);
      await ReportsApi.unshareReport('1', 5);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/reports/1/share/5');
    });
  });

  describe('getSharedUsers', () => {
    it('should get shared users', async () => {
      mockGet.mockResolvedValue([]);
      await ReportsApi.getSharedUsers('1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/reports/1/shares');
    });
  });

  describe('error propagation', () => {
    it('should propagate errors', async () => {
      mockGet.mockRejectedValue(new Error('Not found'));
      await expect(ReportsApi.getReport('999')).rejects.toThrow('Not found');
    });
  });
});
