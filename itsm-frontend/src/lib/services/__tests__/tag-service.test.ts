import { tagService } from '../tag-service';
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

describe('TagService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('listTags', () => {
    it('should fetch all tags', async () => {
      const tags = [
        { id: 1, name: 'Bug', code: 'bug', color: '#ff0000' },
        { id: 2, name: 'Feature', code: 'feature', color: '#00ff00' },
      ];
      mockGet.mockResolvedValue(tags);

      const result = await tagService.listTags();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/tags');
      expect(result).toHaveLength(2);
      expect(result[0].name).toBe('Bug');
      expect(result[1].color).toBe('#00ff00');
    });

    it('should return empty array when no tags', async () => {
      mockGet.mockResolvedValue([]);

      const result = await tagService.listTags();

      expect(result).toEqual([]);
    });
  });

  describe('createTag', () => {
    it('should create a tag', async () => {
      const request = { name: 'Urgent', code: 'urgent', color: '#ff9900' };
      const response = { id: 3, ...request };
      mockPost.mockResolvedValue(response);

      const result = await tagService.createTag(request);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/tags', request);
      expect(result).toEqual(response);
    });

    it('should create a tag without optional fields', async () => {
      const request = { name: 'Simple', code: 'simple' };
      const response = { id: 4, ...request };
      mockPost.mockResolvedValue(response);

      const result = await tagService.createTag(request);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/tags', request);
      expect(result.id).toBe(4);
    });
  });

  describe('updateTag', () => {
    it('should update a tag by id', async () => {
      const updateData = { name: 'Critical Bug', color: '#cc0000' };
      const response = { id: 1, name: 'Critical Bug', code: 'bug', color: '#cc0000' };
      mockPut.mockResolvedValue(response);

      const result = await tagService.updateTag(1, updateData);

      expect(mockPut).toHaveBeenCalledWith('/api/v1/tags/1', updateData);
      expect(result.name).toBe('Critical Bug');
    });
  });

  describe('deleteTag', () => {
    it('should delete a tag by id', async () => {
      mockDelete.mockResolvedValue(undefined);

      await tagService.deleteTag(2);

      expect(mockDelete).toHaveBeenCalledWith('/api/v1/tags/2');
    });
  });

  describe('bindTag', () => {
    it('should bind a tag to an entity', async () => {
      mockPost.mockResolvedValue(undefined);

      await tagService.bindTag(1, 'ticket', 100);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/tags/bind', {
        tagId: 1,
        entityType: 'ticket',
        entityId: 100,
      });
    });

    it('should bind a tag to a different entity type', async () => {
      mockPost.mockResolvedValue(undefined);

      await tagService.bindTag(2, 'incident', 50);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/tags/bind', {
        tagId: 2,
        entityType: 'incident',
        entityId: 50,
      });
    });
  });
});
