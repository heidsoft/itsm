import { projectService } from '../project-service';
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

describe('ProjectService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('listProjects', () => {
    it('should fetch all projects', async () => {
      const projects = [
        { id: 1, name: 'Project Alpha', code: 'PA', status: 'active' },
        { id: 2, name: 'Project Beta', code: 'PB', status: 'completed' },
      ];
      mockGet.mockResolvedValue(projects);

      const result = await projectService.listProjects();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/projects');
      expect(result).toHaveLength(2);
      expect(result[0].name).toBe('Project Alpha');
    });

    it('should return empty array when no projects', async () => {
      mockGet.mockResolvedValue([]);

      const result = await projectService.listProjects();

      expect(result).toEqual([]);
    });
  });

  describe('createProject', () => {
    it('should create a project', async () => {
      const request = { name: 'New Project', code: 'NP', departmentId: 1, managerId: 5 };
      const response = { id: 3, ...request };
      mockPost.mockResolvedValue(response);

      const result = await projectService.createProject(request);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/projects', request);
      expect(result).toEqual(response);
    });

    it('should create a project with minimal fields', async () => {
      const request = { name: 'Minimal', code: 'MIN' };
      const response = { id: 4, ...request };
      mockPost.mockResolvedValue(response);

      const result = await projectService.createProject(request);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/projects', request);
      expect(result.id).toBe(4);
    });
  });

  describe('updateProject', () => {
    it('should update a project by id', async () => {
      const updateData = { name: 'Updated Project' };
      const response = { id: 1, name: 'Updated Project', code: 'PA' };
      mockPut.mockResolvedValue(response);

      const result = await projectService.updateProject(1, updateData);

      expect(mockPut).toHaveBeenCalledWith('/api/v1/projects/1', updateData);
      expect(result.name).toBe('Updated Project');
    });
  });

  describe('deleteProject', () => {
    it('should delete a project by id', async () => {
      mockDelete.mockResolvedValue(undefined);

      await projectService.deleteProject(3);

      expect(mockDelete).toHaveBeenCalledWith('/api/v1/projects/3');
    });
  });
});
