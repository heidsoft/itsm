import { globalSearch } from '../global-search-api';
import { httpClient } from '../http-client';

jest.mock('../http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
  },
}));

const mockGet = httpClient.get as jest.Mock;

describe('globalSearch', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  it('should search with keyword', async () => {
    mockGet.mockResolvedValue({ results: [{ id: 1, type: 'ticket', title: 'Test' }], total: 1 });
    const result = await globalSearch('test');
    expect(mockGet).toHaveBeenCalledWith('/api/v1/global-search', { keyword: 'test' });
    expect(result.results).toHaveLength(1);
    expect(result.total).toBe(1);
  });

  it('should handle empty results', async () => {
    mockGet.mockResolvedValue({ results: [], total: 0 });
    const result = await globalSearch('nonexistent');
    expect(result.results).toEqual([]);
    expect(result.total).toBe(0);
  });

  it('should propagate errors', async () => {
    mockGet.mockRejectedValue(new Error('Search failed'));
    await expect(globalSearch('test')).rejects.toThrow('Search failed');
  });
});
