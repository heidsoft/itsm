import { teamService } from '../team-service';
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

describe('TeamService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('listTeams', () => {
    it('should fetch and normalize teams', async () => {
      const rawTeams = [
        {
          id: 1,
          name: 'Platform Team',
          code: 'PLT',
          description: 'Platform engineering',
          managerId: 5,
          createdAt: '2024-01-01',
          updatedAt: '2024-01-02',
          edges: { users: [{ id: 10, name: 'Alice' }] },
        },
        {
          id: 2,
          name: 'Mobile Team',
          code: 'MOB',
        },
      ];
      mockGet.mockResolvedValue(rawTeams);

      const result = await teamService.listTeams();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/org/teams');
      expect(result).toHaveLength(2);
      expect(result[0].id).toBe(1);
      expect(result[0].name).toBe('Platform Team');
      expect(result[0].edges?.users).toHaveLength(1);
      expect(result[1].id).toBe(2);
      expect(result[1].code).toBe('MOB');
    });

    it('should handle empty list', async () => {
      mockGet.mockResolvedValue([]);

      const result = await teamService.listTeams();

      expect(result).toEqual([]);
    });

    it('should normalize missing fields with defaults', async () => {
      mockGet.mockResolvedValue([{}]);

      const result = await teamService.listTeams();

      expect(result[0].id).toBe(0);
      expect(result[0].name).toBe('');
      expect(result[0].code).toBe('');
    });
  });

  describe('createTeam', () => {
    it('should create a team', async () => {
      const request = { name: 'QA Team', code: 'QA', description: 'Quality Assurance' };
      const response = { id: 3, ...request };
      mockPost.mockResolvedValue(response);

      const result = await teamService.createTeam(request);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/org/teams', request);
      expect(result).toEqual(response);
    });
  });

  describe('addMember', () => {
    it('should add a member to a team', async () => {
      const request = { teamId: 1, userId: 10 };
      mockPost.mockResolvedValue(undefined);

      await teamService.addMember(request);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/org/teams/members', request);
    });
  });

  describe('updateTeam', () => {
    it('should update a team by id', async () => {
      const updateData = { name: 'Platform Team v2' };
      const response = { id: 1, name: 'Platform Team v2', code: 'PLT' };
      mockPut.mockResolvedValue(response);

      const result = await teamService.updateTeam(1, updateData);

      expect(mockPut).toHaveBeenCalledWith('/api/v1/org/teams/1', updateData);
      expect(result.name).toBe('Platform Team v2');
    });
  });

  describe('deleteTeam', () => {
    it('should delete a team by id', async () => {
      mockDelete.mockResolvedValue(undefined);

      await teamService.deleteTeam(2);

      expect(mockDelete).toHaveBeenCalledWith('/api/v1/org/teams/2');
    });
  });
});
