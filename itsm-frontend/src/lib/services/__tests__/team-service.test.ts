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

const TEAM = {
  id: 1,
  name: 'Platform Team',
  code: 'PLT',
  description: 'Platform engineering',
  status: 'active',
  managerId: 5,
  managerName: 'Boss',
  tenantId: 7,
  members: [{ id: 10, username: 'alice', name: 'Alice', email: 'alice@example.com' }],
  createdAt: '2024-01-01T00:00:00Z',
  updatedAt: '2024-01-02T00:00:00Z',
};

describe('TeamService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('listTeams', () => {
    it('requests page/pageSize/search as query params and unwraps the list envelope', async () => {
      mockGet.mockResolvedValue({ items: [TEAM], total: 12, page: 2, pageSize: 10, totalPages: 2 });

      const result = await teamService.listTeams({ page: 2, pageSize: 10, search: 'dev' });

      expect(mockGet).toHaveBeenCalledWith('/api/v1/org/teams', {
        page: 2,
        pageSize: 10,
        search: 'dev',
      });
      expect(result.total).toBe(12);
      expect(result.page).toBe(2);
      expect(result.totalPages).toBe(2);
      expect(result.items).toHaveLength(1);
      expect(result.items[0].managerName).toBe('Boss');
      expect(result.items[0].members[0]).toEqual({
        id: 10,
        username: 'alice',
        name: 'Alice',
        email: 'alice@example.com',
      });
    });

    it('keeps members as an array so the UI never has to guess its presence', async () => {
      mockGet.mockResolvedValue({
        items: [{ id: 2, name: 'Mobile Team', code: 'MOB' }],
        total: 1,
        page: 1,
        pageSize: 10,
        totalPages: 1,
      });

      const result = await teamService.listTeams();

      expect(result.items[0].members).toEqual([]);
      expect(result.items[0].description).toBe('');
      expect(result.items[0].managerId).toBe(0);
      expect(result.items[0].managerName).toBe('');
    });

    it('handles an empty page', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 10, totalPages: 0 });

      const result = await teamService.listTeams();

      expect(result.items).toEqual([]);
      expect(result.total).toBe(0);
    });
  });

  describe('getTeam', () => {
    it('fetches a single team by id', async () => {
      mockGet.mockResolvedValue(TEAM);

      const result = await teamService.getTeam(1);

      expect(mockGet).toHaveBeenCalledWith('/api/v1/org/teams/1');
      expect(result.name).toBe('Platform Team');
      expect(result.members).toHaveLength(1);
    });
  });

  describe('createTeam', () => {
    it('posts to the collection path and returns the server projection', async () => {
      const request = { name: 'QA Team', code: 'QA', description: 'Quality Assurance' };
      mockPost.mockResolvedValue({ ...TEAM, id: 3, name: 'QA Team', code: 'QA', members: [] });

      const result = await teamService.createTeam(request);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/org/teams', request);
      expect(result.id).toBe(3);
      expect(result.members).toEqual([]);
    });
  });

  describe('members', () => {
    it('lists members through the paginated member endpoint', async () => {
      mockGet.mockResolvedValue({
        items: TEAM.members,
        total: 1,
        page: 1,
        pageSize: 20,
        totalPages: 1,
      });

      const result = await teamService.listMembers(1, { page: 1, pageSize: 20 });

      expect(mockGet).toHaveBeenCalledWith('/api/v1/org/teams/1/members', {
        page: 1,
        pageSize: 20,
      });
      expect(result.items[0].username).toBe('alice');
    });

    it('adds a member with the camelCase userId body and returns the server member list', async () => {
      mockPost.mockResolvedValue(TEAM.members);

      const result = await teamService.addMember(1, 10);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/org/teams/1/members', { userId: 10 });
      expect(result).toHaveLength(1);
    });

    it('removes a member by userId and returns the remaining member list', async () => {
      mockDelete.mockResolvedValue([]);

      const result = await teamService.removeMember(1, 10);

      expect(mockDelete).toHaveBeenCalledWith('/api/v1/org/teams/1/members', { userId: 10 });
      expect(result).toEqual([]);
    });
  });

  describe('updateTeam', () => {
    it('should update a team by id', async () => {
      const updateData = { name: 'Platform Team v2' };
      mockPut.mockResolvedValue({ ...TEAM, name: 'Platform Team v2' });

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
