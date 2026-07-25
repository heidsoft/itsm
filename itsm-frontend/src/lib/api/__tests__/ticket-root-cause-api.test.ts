import { TicketRootCauseApi } from '../ticket-root-cause-api';
import { httpClient } from '../http-client';

jest.mock('../http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
  },
}));

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;

describe('TicketRootCauseApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('analyzeTicket', () => {
    it('should analyze ticket root cause', async () => {
      const report = { ticketId: 1, rootCauses: [{ id: 'rc1', title: 'DNS failure' }] };
      mockPost.mockResolvedValue(report);
      const result = await TicketRootCauseApi.analyzeTicket(1);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/incidents/1/root-cause', {});
      expect(result).toEqual(report);
    });
  });

  describe('getAnalysisReport', () => {
    it('should get analysis report', async () => {
      const report = { ticketId: 1, rootCauses: [], analysisSummary: 'Summary' };
      mockGet.mockResolvedValue(report);
      const result = await TicketRootCauseApi.getAnalysisReport(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/incidents/1/root-cause');
      expect(result).toEqual(report);
    });

    it('should return null when no report', async () => {
      mockGet.mockResolvedValue(null);
      const result = await TicketRootCauseApi.getAnalysisReport(1);
      expect(result).toBeNull();
    });
  });

  describe('confirmRootCause', () => {
    it('should confirm root cause', async () => {
      mockPost.mockResolvedValue(undefined);
      await TicketRootCauseApi.confirmRootCause(1, 'rc1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/incidents/1/root-cause/rc1/confirm', {});
    });
  });

  describe('resolveRootCause', () => {
    it('should resolve root cause', async () => {
      mockPost.mockResolvedValue(undefined);
      await TicketRootCauseApi.resolveRootCause(1, 'rc1');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/incidents/1/root-cause/rc1/resolve', {});
    });
  });

  describe('error propagation', () => {
    it('should propagate errors', async () => {
      mockPost.mockRejectedValue(new Error('Analysis failed'));
      await expect(TicketRootCauseApi.analyzeTicket(999)).rejects.toThrow('Analysis failed');
    });
  });
});
