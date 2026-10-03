import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import { httpClient } from '@/lib/api/http-client';
import { ServiceRequestStatus } from '@/types/service-catalog';

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

describe('ServiceCatalogApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('getServices', () => {
    const envelope = (items: unknown[], total: number, page = 1, pageSize = 10) => ({
      items,
      total,
      page,
      pageSize,
      totalPages: Math.ceil(total / pageSize),
    });

    it('reads only the items key and sends pageSize without the size alias', async () => {
      mockGet.mockResolvedValue(
        envelope([{ id: 1, name: 'Email', status: 'enabled', category: 'it_service' }], 1),
      );
      const result = await ServiceCatalogApi.getServices({ page: 1, pageSize: 10 });
      expect(mockGet).toHaveBeenCalledWith(
        '/api/v1/service-catalogs',
        expect.objectContaining({ page: 1, pageSize: 10 }),
      );
      const sentParams = mockGet.mock.calls[0][1];
      expect(sentParams).not.toHaveProperty('size');
      expect(result.services).toHaveLength(1);
      expect(result.total).toBe(1);
    });

    it('默认页长为 10 且不再降级读取 catalogs/services 幻影键', async () => {
      // 后端信封只有 items；若响应缺 items 必须直接抛错，而不是静默返回空列表。
      mockGet.mockResolvedValue({ total: 0, page: 1, pageSize: 10, totalPages: 0 });
      await expect(ServiceCatalogApi.getServices({})).rejects.toThrow(TypeError);
    });

    it('search 只过滤当前页，total 仍取后端全集', async () => {
      mockGet.mockResolvedValue(
        envelope(
          [
            { id: 1, name: 'Email Relay', status: 'enabled', category: 'it_service' },
            { id: 2, name: 'VPN Access', status: 'enabled', category: 'it_service' },
          ],
          42,
        ),
      );
      const result = await ServiceCatalogApi.getServices({ search: 'email' });
      expect(result.services.map(s => s.id)).toEqual(['1']);
      expect(result.total).toBe(42);
    });
  });

  describe('getAllServices', () => {
    const page = (items: unknown[], total: number, pageNo: number, pageSize = 100) => ({
      items,
      total,
      page: pageNo,
      pageSize,
      totalPages: Math.ceil(total / pageSize),
    });
    const row = (id: number) => ({ id, name: `Svc ${id}`, status: 'enabled', category: 'compute' });

    it('按 100 条一页翻到底，而不是把 pageSize 放大', async () => {
      mockGet
        .mockResolvedValueOnce(page(Array.from({ length: 100 }, (_, i) => row(i + 1)), 150, 1))
        .mockResolvedValueOnce(page(Array.from({ length: 50 }, (_, i) => row(i + 101)), 150, 2));

      const result = await ServiceCatalogApi.getAllServices({ category: 'compute' as never }, 4000);

      expect(result.services).toHaveLength(150);
      expect(result.total).toBe(150);
      expect(result.complete).toBe(true);
      expect(mockGet).toHaveBeenCalledTimes(2);
      expect(mockGet).toHaveBeenNthCalledWith(
        1,
        '/api/v1/service-catalogs',
        expect.objectContaining({ page: 1, pageSize: 100, category: 'compute' }),
      );
    });

    it('达到 maxRecords 时 complete=false，调用方不得当成全集', async () => {
      mockGet.mockResolvedValue(
        page(Array.from({ length: 100 }, (_, i) => row(i + 1)), 500, 1),
      );

      const result = await ServiceCatalogApi.getAllServices(undefined, 100);

      expect(result.services).toHaveLength(100);
      expect(result.total).toBe(500);
      expect(result.complete).toBe(false);
      expect(mockGet).toHaveBeenCalledTimes(1);
    });
  });

  describe('getService', () => {
    it('should get a single service', async () => {
      mockGet.mockResolvedValue({ id: 1, name: 'Email', status: 'enabled' });
      const result = await ServiceCatalogApi.getService('1');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/service-catalogs/1');
      expect(result.name).toBe('Email');
    });
  });

  describe('createService', () => {
    it('should create a service', async () => {
      mockPost.mockResolvedValue({ id: 2, name: 'VPN', status: 'enabled' });
      const result = await ServiceCatalogApi.createService({ name: 'VPN', category: 'it_service' as any } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/service-catalogs', expect.objectContaining({ name: 'VPN' }));
    });
  });

  describe('updateService', () => {
    it('should update a service', async () => {
      mockPut.mockResolvedValue({ id: 1, name: 'Updated', status: 'enabled' });
      const result = await ServiceCatalogApi.updateService('1', { name: 'Updated' } as any);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/service-catalogs/1', expect.objectContaining({ name: 'Updated' }));
    });
  });

  describe('deleteService', () => {
    it('should delete a service', async () => {
      mockDelete.mockResolvedValue(undefined);
      await ServiceCatalogApi.deleteService('1');
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/service-catalogs/1');
    });
  });

  describe('publishService', () => {
    it('should publish a service', async () => {
      mockPut.mockResolvedValue({ id: 1, status: 'enabled' });
      await ServiceCatalogApi.publishService('1');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/service-catalogs/1', { status: 'enabled' });
    });
  });

  describe('retireService', () => {
    it('should retire a service', async () => {
      mockPut.mockResolvedValue({ id: 1, status: 'disabled' });
      await ServiceCatalogApi.retireService('1');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/service-catalogs/1', { status: 'disabled' });
    });
  });

  describe('cancelServiceRequest', () => {
    it('should cancel a service request', async () => {
      mockPut.mockResolvedValue(undefined);
      await ServiceCatalogApi.cancelServiceRequest(1, 'No longer needed');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/service-requests/1/status', { status: 'cancelled', comment: 'No longer needed' });
    });
  });

  describe('approveServiceRequest', () => {
    it('should approve a service request', async () => {
      mockPost.mockResolvedValue(undefined);
      await ServiceCatalogApi.approveServiceRequest(1, 'Looks good');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/service-requests/1/approval', { action: 'approve', comment: 'Looks good' });
    });
  });

  describe('rejectServiceRequest', () => {
    it('should reject a service request', async () => {
      mockPost.mockResolvedValue(undefined);
      await ServiceCatalogApi.rejectServiceRequest(1, 'Budget issue');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/service-requests/1/approval', { action: 'reject', comment: 'Budget issue' });
    });
  });

  describe('getCatalogStats', () => {
    it('should pass through only the fields the backend returns', async () => {
      mockGet.mockResolvedValue({ totalServices: 20, publishedServices: 15, categories: { it_service: 7 } });
      const result = await ServiceCatalogApi.getCatalogStats();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/service-catalogs/stats');
      expect(result).toEqual({ totalServices: 20, publishedServices: 15, categories: { it_service: 7 } });
      expect(result).not.toHaveProperty('totalRequests');
      expect(result).not.toHaveProperty('topServices');
    });
  });

  describe('getFavorites', () => {
    it('should fail explicitly (no backend endpoint)', async () => {
      await expect(ServiceCatalogApi.getFavorites()).rejects.toThrow(/服务收藏/);
    });
  });

  describe('getPortalConfig', () => {
    it('should return default config', async () => {
      const result = await ServiceCatalogApi.getPortalConfig();
      expect(result.name).toBe('默认门户');
    });
  });

  describe('getServiceRequests', () => {
    it('发送 pageSize 并按五键信封解析 items', async () => {
      mockGet.mockResolvedValue({
        items: [{ id: 1, requestNumber: 'SR-202609-000001' }],
        total: 1,
        page: 1,
        pageSize: 10,
        totalPages: 1,
      });

      const result = await ServiceCatalogApi.getServiceRequests({ page: 1, pageSize: 10 });

      expect(mockGet).toHaveBeenCalledWith('/api/v1/service-requests/me', { page: 1, pageSize: 10 });
      expect(result.items).toHaveLength(1);
      expect(result.total).toBe(1);
      expect(result.pageSize).toBe(10);
      expect(result.totalPages).toBe(1);
    });

    it('待审批状态走收件箱路径且不携带 status 过滤', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 10, totalPages: 0 });

      await ServiceCatalogApi.getServiceRequests({ status: ServiceRequestStatus.PENDING_APPROVAL });

      expect(mockGet).toHaveBeenCalledWith('/api/v1/service-requests/approvals/pending', {
        page: 1,
        pageSize: 10,
      });
    });
  });

  describe('getServiceRequest', () => {
    it('should get single request', async () => {
      mockGet.mockResolvedValue({ id: 1, status: 'open' });
      await ServiceCatalogApi.getServiceRequest(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/service-requests/1');
    });
  });

  describe('createServiceRequest', () => {
    it('should create service request', async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await ServiceCatalogApi.createServiceRequest({ serviceId: '5', formData: { reason: 'Need access' } } as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/service-requests', expect.objectContaining({ catalogId: 5 }));
    });
  });

  describe('cancelServiceRequest', () => {
    it('should cancel request', async () => {
      mockPut.mockResolvedValue(undefined);
      await ServiceCatalogApi.cancelServiceRequest(1, 'No longer needed');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/service-requests/1/status', { status: 'cancelled', comment: 'No longer needed' });
    });
  });

  describe('approveServiceRequest', () => {
    it('should approve request', async () => {
      mockPost.mockResolvedValue(undefined);
      await ServiceCatalogApi.approveServiceRequest(1, 'LGTM');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/service-requests/1/approval', { action: 'approve', comment: 'LGTM' });
    });
  });

  describe('rejectServiceRequest', () => {
    it('should reject request', async () => {
      mockPost.mockResolvedValue(undefined);
      await ServiceCatalogApi.rejectServiceRequest(1, 'Budget issue');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/service-requests/1/approval', { action: 'reject', comment: 'Budget issue' });
    });
  });

  describe('completeServiceRequest', () => {
    it('should complete request', async () => {
      mockPut.mockResolvedValue(undefined);
      await ServiceCatalogApi.completeServiceRequest(1, 'Done');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/service-requests/1/status', { status: 'completed', comment: 'Done' });
    });
  });

  describe('getPendingApprovalCount', () => {
    it('should get pending count', async () => {
      mockGet.mockResolvedValue({ total: 5 });
      const result = await ServiceCatalogApi.getPendingApprovalCount();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/service-requests/approvals/pending');
      expect(result).toBe(5);
    });
  });

  describe('getCatalogStats', () => {
    it('should get catalog stats', async () => {
      mockGet.mockResolvedValue({ totalServices: 10, publishedServices: 8, categories: {} });
      const result = await ServiceCatalogApi.getCatalogStats();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/service-catalogs/stats');
      expect(result.totalServices).toBe(10);
    });
  });

  describe('getFavorites', () => {
    it('should fail explicitly instead of an empty list', async () => {
      await expect(ServiceCatalogApi.getFavorites()).rejects.toThrow(/服务收藏/);
    });
  });

  describe('getServiceRatings', () => {
    it('should fail explicitly instead of zero-filled ratings', async () => {
      await expect(ServiceCatalogApi.getServiceRatings('1')).rejects.toThrow(/服务评分查询/);
      expect(mockGet).not.toHaveBeenCalled();
    });
  });

  describe('recordServiceView', () => {
    it('should fail explicitly instead of silently dropping the write', async () => {
      await expect(ServiceCatalogApi.recordServiceView('1')).rejects.toThrow(/服务浏览记录/);
    });
  });

  describe('exportCatalog', () => {
    const row = (id: number) => ({
      id: String(id),
      name: `Svc ${id}`,
      category: 'IT',
      status: 'enabled',
      description: 'desc',
      deliveryTime: 1,
      createdAt: '2024-01-01',
      updatedAt: '2024-01-02',
    });
    const page = (items: unknown[], total: number, pageNo: number) => ({
      items,
      total,
      page: pageNo,
      pageSize: 100,
      totalPages: Math.ceil(total / 100),
    });

    it('翻页取全量而不是只导第一页', async () => {
      mockGet
        .mockResolvedValueOnce(page(Array.from({ length: 100 }, (_, i) => row(i + 1)), 150, 1))
        .mockResolvedValueOnce(page(Array.from({ length: 50 }, (_, i) => row(i + 101)), 150, 2));

      const result = await ServiceCatalogApi.exportCatalog('excel');

      expect(result).toBeInstanceOf(Blob);
      expect(mockGet).toHaveBeenCalledTimes(2);
      expect(mockGet).toHaveBeenCalledWith(
        '/api/v1/service-catalogs',
        expect.objectContaining({ page: 2, pageSize: 100 }),
      );
    });

    it('超过导出上限必须显式报错，不得静默导出半截 CSV', async () => {
      mockGet.mockImplementation(async (_path: string, params: { page: number }) =>
        page(Array.from({ length: 100 }, (_, i) => row((params.page - 1) * 100 + i + 1)), 5001, params.page),
      );

      await expect(ServiceCatalogApi.exportCatalog('excel')).rejects.toThrow(
        '超过单次导出上限',
      );
    });
  });

  describe('retireService', () => {
    it('should retire service', async () => {
      mockPut.mockResolvedValue({ id: '1', status: 'disabled' });
      await ServiceCatalogApi.retireService('1');
      expect(mockPut).toHaveBeenCalledWith('/api/v1/service-catalogs/1', { status: 'disabled' });
    });
  });

  describe('cloneService', () => {
    it('should clone service', async () => {
      mockGet.mockResolvedValue({ id: '1', name: 'Original', category: 'IT' });
      mockPost.mockResolvedValue({ id: '2', name: 'Clone' });
      await ServiceCatalogApi.cloneService('1', 'Clone');
      expect(mockGet).toHaveBeenCalledWith('/api/v1/service-catalogs/1');
      expect(mockPost).toHaveBeenCalled();
    });
  });
});
