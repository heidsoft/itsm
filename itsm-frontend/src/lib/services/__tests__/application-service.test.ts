import { applicationService } from '../application-service';
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

describe('ApplicationService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('listApplications', () => {
    it('should fetch all applications', async () => {
      const apps = [
        { id: 1, name: 'Web Portal', code: 'WP', type: 'web', status: 'active' },
        { id: 2, name: 'Mobile App', code: 'MA', type: 'mobile', status: 'active' },
      ];
      mockGet.mockResolvedValue(apps);

      const result = await applicationService.listApplications();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/applications');
      expect(result).toHaveLength(2);
      expect(result[0].name).toBe('Web Portal');
    });

    it('should return empty array when no applications', async () => {
      mockGet.mockResolvedValue([]);

      const result = await applicationService.listApplications();

      expect(result).toEqual([]);
    });
  });

  describe('createApplication', () => {
    it('should create an application', async () => {
      const request = { name: 'New App', code: 'NA', type: 'api', projectId: 1 };
      const response = { id: 3, ...request };
      mockPost.mockResolvedValue(response);

      const result = await applicationService.createApplication(request);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/applications', request);
      expect(result.id).toBe(3);
    });

    it('should create an application with minimal fields', async () => {
      const request = { name: 'Minimal App', code: 'MIN' };
      const response = { id: 4, ...request };
      mockPost.mockResolvedValue(response);

      const result = await applicationService.createApplication(request);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/applications', request);
      expect(result.code).toBe('MIN');
    });
  });

  describe('createMicroservice', () => {
    it('should create a microservice', async () => {
      const request = { name: 'Auth Service', code: 'auth-svc', language: 'go', framework: 'gin', applicationId: 1 };
      const response = { id: 10, ...request };
      mockPost.mockResolvedValue(response);

      const result = await applicationService.createMicroservice(request);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/applications/microservices', request);
      expect(result.name).toBe('Auth Service');
      expect(result.language).toBe('go');
    });
  });

  describe('updateApplication', () => {
    it('should update an application by id', async () => {
      const updateData = { name: 'Updated App' };
      const response = { id: 1, name: 'Updated App', code: 'WP' };
      mockPut.mockResolvedValue(response);

      const result = await applicationService.updateApplication(1, updateData);

      expect(mockPut).toHaveBeenCalledWith('/api/v1/applications/1', updateData);
      expect(result.name).toBe('Updated App');
    });
  });

  describe('deleteApplication', () => {
    it('should delete an application by id', async () => {
      mockDelete.mockResolvedValue(undefined);

      await applicationService.deleteApplication(1);

      expect(mockDelete).toHaveBeenCalledWith('/api/v1/applications/1');
    });
  });

  describe('listMicroservices', () => {
    it('should fetch all microservices', async () => {
      const services = [
        { id: 1, name: 'Auth Service', code: 'auth', language: 'go' },
        { id: 2, name: 'Payment Service', code: 'pay', language: 'java' },
      ];
      mockGet.mockResolvedValue(services);

      const result = await applicationService.listMicroservices();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/applications/microservices');
      expect(result).toHaveLength(2);
    });
  });

  describe('updateMicroservice', () => {
    it('should update a microservice by id', async () => {
      const updateData = { name: 'Updated Service', framework: 'echo' };
      const response = { id: 1, name: 'Updated Service', code: 'auth', framework: 'echo' };
      mockPut.mockResolvedValue(response);

      const result = await applicationService.updateMicroservice(1, updateData);

      expect(mockPut).toHaveBeenCalledWith('/api/v1/applications/microservices/1', updateData);
      expect(result.framework).toBe('echo');
    });
  });

  describe('deleteMicroservice', () => {
    it('should delete a microservice by id', async () => {
      mockDelete.mockResolvedValue(undefined);

      await applicationService.deleteMicroservice(5);

      expect(mockDelete).toHaveBeenCalledWith('/api/v1/applications/microservices/5');
    });
  });
});
