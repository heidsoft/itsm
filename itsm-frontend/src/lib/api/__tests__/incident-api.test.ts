import { IncidentAPI } from '@/lib/api/incident-api';
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

jest.mock('@/lib/api/types', () => ({
  API_URLS: {
    INCIDENTS: () => '/api/v1/incidents',
    INCIDENT: (id: number) => `/api/v1/incidents/${id}`,
  },
  normalizePaginationParams: (p: any) => p,
  normalizeDateRangeParams: (p: any) => ({}),
}));

jest.spyOn(console, 'error').mockImplementation(() => {});

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;
const mockPut = httpClient.put as jest.Mock;
const mockDelete = httpClient.delete as jest.Mock;

describe('IncidentAPI', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('listIncidents', () => {
    it('should list incidents', async () => {
      mockGet.mockResolvedValue({ incidents: [{ id: 1, title: 'Server down' }], total: 1 });
      const result = await IncidentAPI.listIncidents({});
      expect(mockGet).toHaveBeenCalledWith('/api/v1/incidents', expect.any(Object));
      expect(result.incidents).toHaveLength(1);
    });
  });

  describe('getIncident', () => {
    it('should get incident by id', async () => {
      mockGet.mockResolvedValue({ id: 1, title: 'Server down' });
      const result = await IncidentAPI.getIncident(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/incidents/1');
      expect(result.id).toBe(1);
    });
  });

  describe('createIncident', () => {
    it('should create an incident', async () => {
      const data = { title: 'New Incident', priority: 'high', source: 'monitoring', type: 'alert' };
      mockPost.mockResolvedValue({ id: 2, ...data });
      const result = await IncidentAPI.createIncident(data);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/incidents', data);
      expect(result.title).toBe('New Incident');
    });
  });

  describe('updateIncident', () => {
    it('should update an incident', async () => {
      mockPut.mockResolvedValue({ id: 1, title: 'Updated' });
      const result = await IncidentAPI.updateIncident(1, { title: 'Updated' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/incidents/1', { title: 'Updated' });
      expect(result.title).toBe('Updated');
    });
  });

  describe('updateIncidentStatus', () => {
    it('should update status', async () => {
      mockPut.mockResolvedValue({ id: 1, status: 'resolved' });
      const result = await IncidentAPI.updateIncidentStatus(1, { status: 'resolved' });
      expect(mockPut).toHaveBeenCalledWith('/api/v1/incidents/1/status', { status: 'resolved' });
    });
  });

  describe('resolveIncident', () => {
    it('should resolve an incident', async () => {
      mockPost.mockResolvedValue({ id: 1, status: 'resolved' });
      await IncidentAPI.resolveIncident(1, { resolution: 'Fixed the server' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/incidents/1/resolve', { resolution: 'Fixed the server' });
    });
  });

  describe('assignIncident', () => {
    it('should assign an incident', async () => {
      mockPost.mockResolvedValue({ id: 1, assigneeId: 5 });
      await IncidentAPI.assignIncident(1, 5);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/incidents/1/assign', { assigneeId: 5 });
    });
  });

  describe('acknowledgeIncident', () => {
    it('should acknowledge incident', async () => {
      mockPost.mockResolvedValue({ message: 'acknowledged' });
      const result = await IncidentAPI.acknowledgeIncident(1);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/incidents/1/acknowledge', {});
      expect(result.message).toBe('acknowledged');
    });
  });

  describe('closeIncident', () => {
    it('should close an incident', async () => {
      mockPost.mockResolvedValue({ message: 'closed' });
      await IncidentAPI.closeIncident(1, { closeNotes: 'Done' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/incidents/1/close', { closeNotes: 'Done' });
    });
  });

  describe('deleteIncident', () => {
    it('should delete an incident', async () => {
      mockDelete.mockResolvedValue(undefined);
      await IncidentAPI.deleteIncident(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/incidents/1');
    });
  });

  describe('getIncidentMetrics', () => {
    it('should get metrics', async () => {
      mockGet.mockResolvedValue({ totalIncidents: 100 });
      const result = await IncidentAPI.getIncidentMetrics();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/incidents/stats');
      expect(result.totalIncidents).toBe(100);
    });
  });

  describe('deleteIncidentComment', () => {
    it('should throw not implemented', async () => {
      await expect(IncidentAPI.deleteIncidentComment(1, 1)).rejects.toThrow();
    });
  });

  describe('escalateIncident', () => {
    it('should escalate an incident', async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await IncidentAPI.escalateIncident(1, { escalationLevel: 2, reason: 'Urgent' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/incidents/1/escalate', { escalationLevel: 2, reason: 'Urgent' });
    });
  });
});
