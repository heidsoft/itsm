/**
 * IncidentService unit tests
 */
import { incidentService, IncidentStatus, IncidentPriority } from '../incident-service';
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

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;
const mockPut = httpClient.put as jest.Mock;
const mockDelete = httpClient.delete as jest.Mock;

describe('IncidentService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('listIncidents', () => {
    it('should call GET /api/v1/incidents with params', async () => {
      const mockData = { incidents: [{ id: 1 }], items: [{ id: 1 }], total: 1, page: 1, pageSize: 20 };
      mockGet.mockResolvedValueOnce(mockData);

      const result = await incidentService.listIncidents({ page: 1, status: IncidentStatus.NEW });

      expect(mockGet).toHaveBeenCalledWith('/api/v1/incidents', { page: 1, status: 'new' });
      expect(result).toEqual(mockData);
    });

    it('should populate items from incidents when items is missing', async () => {
      const incidents = [{ id: 1, title: 'Test' }];
      mockGet.mockResolvedValueOnce({ incidents, total: 1, page: 1, pageSize: 20 });

      const result = await incidentService.listIncidents();

      expect(result.items).toEqual(incidents);
    });

    it('should call with empty params by default', async () => {
      mockGet.mockResolvedValueOnce({ incidents: [], items: [], total: 0, page: 1, pageSize: 20 });

      await incidentService.listIncidents();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/incidents', {});
    });
  });

  describe('getIncidents (alias)', () => {
    it('should delegate to listIncidents', async () => {
      mockGet.mockResolvedValueOnce({ incidents: [], items: [], total: 0, page: 1, pageSize: 20 });

      await incidentService.getIncidents({ priority: IncidentPriority.HIGH });

      expect(mockGet).toHaveBeenCalledWith('/api/v1/incidents', { priority: 'high' });
    });
  });

  describe('getIncident', () => {
    it('should call GET /api/v1/incidents/:id', async () => {
      mockGet.mockResolvedValueOnce({ id: 42, title: 'Server down' });

      const result = await incidentService.getIncident(42);

      expect(mockGet).toHaveBeenCalledWith('/api/v1/incidents/42');
      expect(result.id).toBe(42);
    });
  });

  describe('createIncident', () => {
    it('should call POST /api/v1/incidents with data', async () => {
      const data = { title: 'New incident', description: 'Desc', priority: IncidentPriority.CRITICAL };
      mockPost.mockResolvedValueOnce({ message: 'created', incidentId: 5 });

      const result = await incidentService.createIncident(data);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/incidents', data);
      expect(result.incidentId).toBe(5);
    });
  });

  describe('updateIncident', () => {
    it('should call PUT /api/v1/incidents/:id with data', async () => {
      mockPut.mockResolvedValueOnce({ message: 'updated', incidentId: 3 });

      const result = await incidentService.updateIncident(3, { status: IncidentStatus.RESOLVED });

      expect(mockPut).toHaveBeenCalledWith('/api/v1/incidents/3', { status: 'resolved' });
      expect(result.incidentId).toBe(3);
    });
  });

  describe('deleteIncident', () => {
    it('should call DELETE /api/v1/incidents/:id', async () => {
      mockDelete.mockResolvedValueOnce({ message: 'deleted', incidentId: 7 });

      await incidentService.deleteIncident(7);

      expect(mockDelete).toHaveBeenCalledWith('/api/v1/incidents/7');
    });
  });

  describe('getIncidentStats', () => {
    it('should call GET /api/v1/incidents/stats', async () => {
      const stats = { total: 50, open: 10, inProgress: 15, resolved: 20, closed: 5, highPriority: 8, critical: 3, mttr: 120 };
      mockGet.mockResolvedValueOnce(stats);

      const result = await incidentService.getIncidentStats();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/incidents/stats');
      expect(result.mttr).toBe(120);
    });
  });

  describe('getStatusColor', () => {
    it('should return correct colors for each status', () => {
      expect(incidentService.getStatusColor(IncidentStatus.NEW)).toBe('blue');
      expect(incidentService.getStatusColor(IncidentStatus.IN_PROGRESS)).toBe('processing');
      expect(incidentService.getStatusColor(IncidentStatus.RESOLVED)).toBe('success');
      expect(incidentService.getStatusColor(IncidentStatus.CLOSED)).toBe('default');
      expect(incidentService.getStatusColor(IncidentStatus.CANCELLED)).toBe('error');
      expect(incidentService.getStatusColor('unknown' as IncidentStatus)).toBe('default');
    });
  });

  describe('getPriorityColor', () => {
    it('should return correct colors for each priority', () => {
      expect(incidentService.getPriorityColor(IncidentPriority.LOW)).toBe('green');
      expect(incidentService.getPriorityColor(IncidentPriority.MEDIUM)).toBe('orange');
      expect(incidentService.getPriorityColor(IncidentPriority.HIGH)).toBe('red');
      expect(incidentService.getPriorityColor(IncidentPriority.CRITICAL)).toBe('purple');
      expect(incidentService.getPriorityColor('unknown' as IncidentPriority)).toBe('default');
    });
  });

  describe('getStatusLabel', () => {
    it('should return Chinese labels', () => {
      expect(incidentService.getStatusLabel(IncidentStatus.NEW)).toBe('新建');
      expect(incidentService.getStatusLabel(IncidentStatus.IN_PROGRESS)).toBe('处理中');
      expect(incidentService.getStatusLabel(IncidentStatus.RESOLVED)).toBe('已解决');
      expect(incidentService.getStatusLabel(IncidentStatus.CLOSED)).toBe('已关闭');
      expect(incidentService.getStatusLabel(IncidentStatus.CANCELLED)).toBe('已取消');
      expect(incidentService.getStatusLabel('other' as IncidentStatus)).toBe('other');
    });
  });

  describe('getPriorityLabel', () => {
    it('should return Chinese labels', () => {
      expect(incidentService.getPriorityLabel(IncidentPriority.LOW)).toBe('低');
      expect(incidentService.getPriorityLabel(IncidentPriority.MEDIUM)).toBe('中');
      expect(incidentService.getPriorityLabel(IncidentPriority.HIGH)).toBe('高');
      expect(incidentService.getPriorityLabel(IncidentPriority.CRITICAL)).toBe('严重');
      expect(incidentService.getPriorityLabel('other' as IncidentPriority)).toBe('other');
    });
  });

  describe('getStatusText', () => {
    it('should return English text', () => {
      expect(incidentService.getStatusText(IncidentStatus.NEW)).toBe('New');
      expect(incidentService.getStatusText(IncidentStatus.IN_PROGRESS)).toBe('In Progress');
      expect(incidentService.getStatusText(IncidentStatus.RESOLVED)).toBe('Resolved');
      expect(incidentService.getStatusText(IncidentStatus.CLOSED)).toBe('Closed');
      expect(incidentService.getStatusText(IncidentStatus.CANCELLED)).toBe('Cancelled');
      expect(incidentService.getStatusText('x' as IncidentStatus)).toBe('Unknown');
    });
  });

  describe('getPriorityText', () => {
    it('should return English text', () => {
      expect(incidentService.getPriorityText(IncidentPriority.LOW)).toBe('Low');
      expect(incidentService.getPriorityText(IncidentPriority.MEDIUM)).toBe('Medium');
      expect(incidentService.getPriorityText(IncidentPriority.HIGH)).toBe('High');
      expect(incidentService.getPriorityText(IncidentPriority.CRITICAL)).toBe('Critical');
      expect(incidentService.getPriorityText('x' as IncidentPriority)).toBe('Unknown');
    });
  });
});
