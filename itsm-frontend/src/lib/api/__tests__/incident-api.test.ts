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
    it('should delete an incident comment through the registered route', async () => {
      mockDelete.mockResolvedValue(undefined);
      await IncidentAPI.deleteIncidentComment(1, 1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/incidents/1/comments/1');
    });
  });

  describe('escalateIncident', () => {
    it('should escalate an incident', async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await IncidentAPI.escalateIncident(1, { escalationLevel: 2, reason: 'Urgent' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/incidents/1/escalate', {
        escalationLevel: 2,
        reason: 'Urgent',
        incidentId: 1,
      });
    });
  });

  describe('addComment', () => {
    it('should add comment to incident', async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await IncidentAPI.addComment(1, { content: 'Investigating' });
      expect(mockPost).toHaveBeenCalledWith('/api/v1/incidents/1/comments', { content: 'Investigating' });
    });
  });

  describe('reopenIncident', () => {
    it('should reopen an incident', async () => {
      mockPost.mockResolvedValue({ id: 1, status: 'in_progress' });
      await IncidentAPI.reopenIncident(1);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/incidents/1/reopen', {});
    });
  });

  describe('getIncidentComments', () => {
    it('should get incident comments', async () => {
      mockGet.mockResolvedValue([{ id: 1, content: 'test' }]);
      const result = await IncidentAPI.getIncidentComments(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/incidents/1/comments');
      expect(result).toHaveLength(1);
    });

    it('should propagate errors', async () => {
      mockGet.mockRejectedValue(new Error('fail'));
      await expect(IncidentAPI.getIncidentComments(1)).rejects.toThrow('fail');
    });
  });

  describe('getIncidentAlerts', () => {
    it('should get alerts without params', async () => {
      mockGet.mockResolvedValue({ alerts: [], total: 0 });
      await IncidentAPI.getIncidentAlerts(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/incidents/1/alerts', {});
    });

    it('should get alerts with params', async () => {
      mockGet.mockResolvedValue({ alerts: [], total: 0 });
      await IncidentAPI.getIncidentAlerts(1, { status: 'active', alertLevel: 'critical' });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/incidents/1/alerts', { status: 'active', alertLevel: 'critical' });
    });
  });

  describe('acknowledgeAlert', () => {
    it('should acknowledge an alert', async () => {
      mockPost.mockResolvedValue({ message: 'acknowledged' });
      const result = await IncidentAPI.acknowledgeAlert(5);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/incidents/alerts/5/acknowledge', {});
      expect(result.message).toBe('acknowledged');
    });
  });

  describe('getRootCauseAnalysis', () => {
    it('should get root cause analysis', async () => {
      mockGet.mockResolvedValue({ id: 1, rootCause: 'DNS' });
      const result = await IncidentAPI.getRootCauseAnalysis(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/incidents/1/root-cause');
    });

    it('should propagate errors', async () => {
      mockGet.mockRejectedValue(new Error('not found'));
      await expect(IncidentAPI.getRootCauseAnalysis(1)).rejects.toThrow('not found');
    });
  });

  describe('createRootCauseAnalysis', () => {
    it('should create root cause analysis', async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await IncidentAPI.createRootCauseAnalysis({ incidentId: 1, rootCause: 'DNS', category: 'network' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/incidents/root-cause', expect.any(Object));
    });
  });

  describe('updateRootCauseAnalysis', () => {
    it('should update root cause analysis', async () => {
      mockPut.mockResolvedValue({ id: 1 });
      await IncidentAPI.updateRootCauseAnalysis(1, { rootCause: 'Updated' } as any);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/incidents/root-cause/1', { rootCause: 'Updated' });
    });
  });

  describe('getImpactAssessment', () => {
    it('should get impact assessment', async () => {
      mockGet.mockResolvedValue({ id: 1, impactLevel: 'high' });
      await IncidentAPI.getImpactAssessment(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/incidents/1/impact-assessment');
    });
  });

  describe('createImpactAssessment', () => {
    it('should create impact assessment', async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await IncidentAPI.createImpactAssessment({ incidentId: 1, impactLevel: 'high' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/incidents/impact-assessment', expect.any(Object));
    });
  });

  describe('updateImpactAssessment', () => {
    it('should update impact assessment', async () => {
      mockPut.mockResolvedValue({ id: 1 });
      await IncidentAPI.updateImpactAssessment(1, { impactLevel: 'medium' } as any);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/incidents/impact-assessment/1', { impactLevel: 'medium' });
    });
  });

  describe('getIncidentClassification', () => {
    it('should get classification', async () => {
      mockGet.mockResolvedValue({ id: 1, category: 'network' });
      await IncidentAPI.getIncidentClassification(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/incidents/1/classification');
    });
  });

  describe('createIncidentClassification', () => {
    it('should create classification', async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await IncidentAPI.createIncidentClassification({ incidentId: 1, category: 'network' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/incidents/classification', expect.any(Object));
    });
  });

  describe('updateIncidentClassification', () => {
    it('should update classification', async () => {
      mockPut.mockResolvedValue({ id: 1 });
      await IncidentAPI.updateIncidentClassification(1, { category: 'app' } as any);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/incidents/classification/1', { category: 'app' });
    });
  });

  describe('getConfigurationItems', () => {
    it('should get CIs without params', async () => {
      mockGet.mockResolvedValue([{ id: 1, name: 'Server1' }]);
      const result = await IncidentAPI.getConfigurationItems();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/incidents/configuration-items', {});
    });

    it('should get CIs with params', async () => {
      mockGet.mockResolvedValue([]);
      await IncidentAPI.getConfigurationItems('web', 'server', 'active');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/incidents/configuration-items', { search: 'web', type: 'server', status: 'active' });
    });
  });

  describe('createIncidentFromAlibabaCloudAlert', () => {
    it('should create incident from alert', async () => {
      mockPost.mockResolvedValue({ id: 1, source: 'alibaba-cloud' });
      await IncidentAPI.createIncidentFromAlibabaCloudAlert({ alertId: 'a1', alertName: 'CPU High' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/incidents/alibaba-cloud-alert', expect.any(Object));
    });
  });

  describe('createIncidentFromSecurityEvent', () => {
    it('should create incident from security event', async () => {
      mockPost.mockResolvedValue({ id: 1, source: 'security' });
      await IncidentAPI.createIncidentFromSecurityEvent({ eventId: 's1', eventType: 'SSH' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/incidents/security-event', expect.any(Object));
    });
  });

  describe('createIncidentFromCloudProductEvent', () => {
    it('should create incident from cloud event', async () => {
      mockPost.mockResolvedValue({ id: 1, source: 'cloud' });
      await IncidentAPI.createIncidentFromCloudProductEvent({ eventId: 'c1', product: 'rds' } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/incidents/cloud-product-event', expect.any(Object));
    });
  });

  describe('simulateAlibabaCloudAlert', () => {
    it('should simulate alert and create incident', async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await IncidentAPI.simulateAlibabaCloudAlert();
      expect(mockPost).toHaveBeenCalledWith('/api/v1/incidents/alibaba-cloud-alert', expect.objectContaining({ alertName: 'CPU使用率过高告警' }));
    });
  });

  describe('simulateSecurityEvent', () => {
    it('should simulate security event', async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await IncidentAPI.simulateSecurityEvent();
      expect(mockPost).toHaveBeenCalledWith('/api/v1/incidents/security-event', expect.objectContaining({ eventType: 'SSH暴力破解' }));
    });
  });

  describe('simulateCloudProductEvent', () => {
    it('should simulate cloud product event', async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await IncidentAPI.simulateCloudProductEvent();
      expect(mockPost).toHaveBeenCalledWith('/api/v1/incidents/cloud-product-event', expect.objectContaining({ product: 'rds' }));
    });
  });

  describe('getIncidents (deprecated alias)', () => {
    it('should call listIncidents', async () => {
      mockGet.mockResolvedValue({ incidents: [], total: 0 });
      await IncidentAPI.getIncidents({ status: 'open' } as any);
      expect(mockGet).toHaveBeenCalled();
    });
  });

  describe('incidents accessor', () => {
    it('should have list method', async () => {
      mockGet.mockResolvedValue({ incidents: [{ id: 1 }], total: 1 });
      const result = await IncidentAPI.incidents.list();
      expect(result.incidents).toHaveLength(1);
    });

    it('should have items method that extracts items', async () => {
      mockGet.mockResolvedValue({ incidents: [{ id: 1 }], total: 1 });
      const result = await IncidentAPI.incidents.items();
      expect(result).toEqual([{ id: 1 }]);
    });
  });

  describe('getIncidentEvents', () => {
    it('should get events', async () => {
      mockGet.mockResolvedValue([{ id: 1, type: 'status_change' }]);
      const result = await IncidentAPI.getIncidentEvents(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/incidents/1/events');
      expect(result).toHaveLength(1);
    });
  });

  describe('getIncidentMetricsData', () => {
    it('should get metrics data', async () => {
      mockGet.mockResolvedValue({ responseTime: 120 });
      const result = await IncidentAPI.getIncidentMetricsData(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/incidents/1/metrics');
    });
  });
});
