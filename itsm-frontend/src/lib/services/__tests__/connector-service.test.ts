import { connectorService } from '../connector-service';
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
const mockDelete = httpClient.delete as jest.Mock;

describe('ConnectorService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('list', () => {
    it('should return connector manifests', async () => {
      const items = [
        { name: 'slack', version: '1.0.0', title: 'Slack', provider: 'slack', type: 'im', capabilities: ['send'], local: true, installed: true, category: 'communication' },
        { name: 'teams', version: '1.0.0', title: 'Teams', provider: 'microsoft', type: 'im', capabilities: ['send'], local: false, installed: false, category: 'communication' },
      ];
      mockGet.mockResolvedValue({ items, total: 2 });

      const result = await connectorService.list();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/connectors');
      expect(result).toHaveLength(2);
      expect(result[0].name).toBe('slack');
    });

    it('should return empty array when items is undefined', async () => {
      mockGet.mockResolvedValue({ total: 0 });

      const result = await connectorService.list();

      expect(result).toEqual([]);
    });
  });

  describe('configs', () => {
    it('should return configured connector instances', async () => {
      const items = [
        { name: 'slack', provider: 'slack', type: 'im', enabled: true, healthy: true },
      ];
      mockGet.mockResolvedValue({ items, total: 1 });

      const result = await connectorService.configs();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/connectors/configs');
      expect(result).toHaveLength(1);
      expect(result[0].enabled).toBe(true);
    });

    it('should return empty array when items is undefined', async () => {
      mockGet.mockResolvedValue({ total: 0 });

      const result = await connectorService.configs();

      expect(result).toEqual([]);
    });
  });

  describe('provision', () => {
    it('should provision a connector', async () => {
      const payload = { name: 'slack', enabled: true, credentials: { token: 'abc' } };
      const response = { name: 'slack', provider: 'slack', type: 'im', enabled: true };
      mockPost.mockResolvedValue(response);

      const result = await connectorService.provision(payload);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/connectors/configs', payload);
      expect(result.name).toBe('slack');
      expect(result.enabled).toBe(true);
    });
  });

  describe('revoke', () => {
    it('should revoke a connector by name', async () => {
      mockDelete.mockResolvedValue(undefined);

      await connectorService.revoke('slack');

      expect(mockDelete).toHaveBeenCalledWith('/api/v1/connectors/configs/slack');
    });

    it('should encode special characters in name', async () => {
      mockDelete.mockResolvedValue(undefined);

      await connectorService.revoke('my connector');

      expect(mockDelete).toHaveBeenCalledWith('/api/v1/connectors/configs/my%20connector');
    });
  });

  describe('test', () => {
    it('should send a test message', async () => {
      mockPost.mockResolvedValue({ sent: true, channel: '#debug' });

      const result = await connectorService.test('slack');

      expect(mockPost).toHaveBeenCalledWith('/api/v1/connectors/slack/test');
      expect(result.sent).toBe(true);
      expect(result.channel).toBe('#debug');
    });
  });

  describe('send', () => {
    it('should send a message through a connector', async () => {
      const payload = { channel: '#general', content: 'Hello world', type: 'text' as const };
      mockPost.mockResolvedValue({ sent: true, channel: '#general' });

      const result = await connectorService.send('slack', payload);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/connectors/slack/send', payload);
      expect(result.sent).toBe(true);
      expect(result.channel).toBe('#general');
    });

    it('should handle markdown message type', async () => {
      const payload = { channel: '#updates', content: '**Bold**', type: 'markdown' as const };
      mockPost.mockResolvedValue({ sent: true, channel: '#updates' });

      const result = await connectorService.send('teams', payload);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/connectors/teams/send', payload);
      expect(result.sent).toBe(true);
    });
  });

  describe('health', () => {
    it('should return health status for all connectors', async () => {
      const healthData = {
        slack: { ok: true, latencyMs: 50, checkedAt: '2024-01-01T00:00:00Z' },
        teams: { ok: false, message: 'Connection timeout', checkedAt: '2024-01-01T00:00:00Z' },
      };
      mockGet.mockResolvedValue(healthData);

      const result = await connectorService.health();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/connectors/health');
      expect(result.slack.ok).toBe(true);
      expect(result.teams.ok).toBe(false);
      expect(result.teams.message).toBe('Connection timeout');
    });
  });
});
