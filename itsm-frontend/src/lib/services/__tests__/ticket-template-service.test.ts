/**
 * TicketTemplateService unit tests
 */
import { ticketTemplateService } from '../ticket-template-service';
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

// Suppress console.error/warn from service error handling
jest.spyOn(console, 'error').mockImplementation(() => {});
jest.spyOn(console, 'warn').mockImplementation(() => {});

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;
const mockPut = httpClient.put as jest.Mock;
const mockDelete = httpClient.delete as jest.Mock;
const mockPatch = httpClient.patch as jest.Mock;

describe('TicketTemplateService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('getTemplates', () => {
    it('should call GET /api/v1/tickets/templates with query params', async () => {
      const mockData = { templates: [], total: 0, page: 1, pageSize: 20 };
      mockGet.mockResolvedValueOnce(mockData);

      const result = await ticketTemplateService.getTemplates({ page: 2, pageSize: 10, category: 'IT' });

      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/templates', { page: 2, pageSize: 10, category: 'IT' });
      expect(result).toEqual(mockData);
    });

    it('should handle isActive param', async () => {
      mockGet.mockResolvedValueOnce({ templates: [], total: 0, page: 1, pageSize: 20 });
      await ticketTemplateService.getTemplates({ isActive: true });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/templates', { isActive: true });
    });

    it('should propagate errors', async () => {
      mockGet.mockRejectedValueOnce(new Error('Network'));
      await expect(ticketTemplateService.getTemplates()).rejects.toThrow('Network');
    });
  });

  describe('getTemplate', () => {
    it('should call GET /api/v1/tickets/templates/:id', async () => {
      mockGet.mockResolvedValueOnce({ id: 3, name: 'Standard' });
      const result = await ticketTemplateService.getTemplate(3);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/templates/3');
      expect(result.name).toBe('Standard');
    });

    it('should propagate errors', async () => {
      mockGet.mockRejectedValueOnce(new Error('Not found'));
      await expect(ticketTemplateService.getTemplate(999)).rejects.toThrow('Not found');
    });
  });

  describe('createTemplate', () => {
    it('should call POST /api/v1/tickets/templates', async () => {
      const data = { name: 'New', description: 'Desc', category: 'IT', fields: [] };
      mockPost.mockResolvedValueOnce({ id: 10, ...data });
      const result = await ticketTemplateService.createTemplate(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/templates', data);
      expect(result.id).toBe(10);
    });
  });

  describe('updateTemplate', () => {
    it('should call PUT /api/v1/tickets/templates/:id', async () => {
      mockPut.mockResolvedValueOnce({ id: 3, name: 'Updated' });
      const result = await ticketTemplateService.updateTemplate(3, { name: 'Updated' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/tickets/templates/3', { name: 'Updated' });
      expect(result.name).toBe('Updated');
    });
  });

  describe('deleteTemplate', () => {
    it('should call DELETE /api/v1/tickets/templates/:id', async () => {
      mockDelete.mockResolvedValueOnce(undefined);
      await ticketTemplateService.deleteTemplate(5);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/tickets/templates/5');
    });

    it('should propagate errors', async () => {
      mockDelete.mockRejectedValueOnce(new Error('forbidden'));
      await expect(ticketTemplateService.deleteTemplate(5)).rejects.toThrow('forbidden');
    });
  });

  describe('toggleTemplateStatus', () => {
    it('should call PATCH /api/v1/tickets/templates/:id/status', async () => {
      mockPatch.mockResolvedValueOnce({ id: 3, isActive: false });
      const result = await ticketTemplateService.toggleTemplateStatus(3, false);
      expect(mockPatch).toHaveBeenCalledWith('/api/v1/tickets/templates/3/status', { isActive: false });
      expect(result.isActive).toBe(false);
    });
  });

  describe('copyTemplate', () => {
    it('should call POST /api/v1/tickets/templates/:id/copy', async () => {
      mockPost.mockResolvedValueOnce({ id: 11, name: 'Copy of Standard' });
      const result = await ticketTemplateService.copyTemplate(3, 'Copy of Standard');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/templates/3/copy', { name: 'Copy of Standard' });
      expect(result.name).toBe('Copy of Standard');
    });
  });

  describe('getTemplateCategories', () => {
    it('should call GET /api/v1/tickets/templates/categories', async () => {
      mockGet.mockResolvedValueOnce(['IT', 'HR', 'Finance']);
      const result = await ticketTemplateService.getTemplateCategories();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/templates/categories');
      expect(result).toEqual(['IT', 'HR', 'Finance']);
    });
  });

  describe('healthCheck', () => {
    it('should return true on success', async () => {
      mockGet.mockResolvedValueOnce({ status: 'ok' });
      const result = await ticketTemplateService.healthCheck();
      expect(result).toBe(true);
    });

    it('should return false on failure', async () => {
      mockGet.mockRejectedValueOnce(new Error('down'));
      const result = await ticketTemplateService.healthCheck();
      expect(result).toBe(false);
    });
  });
});
