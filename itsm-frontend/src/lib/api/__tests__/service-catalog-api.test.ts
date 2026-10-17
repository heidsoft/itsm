import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import { httpClient } from '@/lib/api/http-client';
import { ServiceRequestStatus, ServiceStatus } from '@/types/service-catalog';

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

    it('关键词、分类和关联筛选交给服务端，前端不再过滤当前页', async () => {
      mockGet.mockResolvedValue(
        envelope(
          [
            { id: 1, name: 'Email Relay', status: 'enabled', category: 'it_service' },
            { id: 2, name: 'VPN Access', status: 'enabled', category: 'it_service' },
          ],
          42,
        ),
      );

      const result = await ServiceCatalogApi.getServices({
        search: '  email  ',
        ciTypeId: 7,
        cloudServiceId: 3,
      });

      expect(mockGet).toHaveBeenCalledWith('/api/v1/service-catalogs', {
        page: 1,
        pageSize: 10,
        q: 'email',
        ciTypeId: 7,
        cloudServiceId: 3,
      });
      // 命中与否由后端判断；客户端把已加载的一页再筛一遍会伪装成全集结果。
      expect(result.services.map(s => s.id)).toEqual(['1', '2']);
      expect(result.total).toBe(42);
    });

    it('0 与未传都不进入查询参数，避免把合法零值当成过滤条件', async () => {
      mockGet.mockResolvedValue(envelope([], 0));

      await ServiceCatalogApi.getServices({ ciTypeId: 0, cloudServiceId: 0, search: '   ' });

      const sentParams = mockGet.mock.calls[0][1];
      expect(sentParams).not.toHaveProperty('ciTypeId');
      expect(sentParams).not.toHaveProperty('cloudServiceId');
      expect(sentParams).not.toHaveProperty('q');
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

      const result = await ServiceCatalogApi.getAllServices({ category: 'compute' }, 4000);

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

    it('过滤条件随每一页一起发送，不能只在第一页生效', async () => {
      mockGet
        .mockResolvedValueOnce(page(Array.from({ length: 100 }, (_, i) => row(i + 1)), 150, 1))
        .mockResolvedValueOnce(page(Array.from({ length: 50 }, (_, i) => row(i + 101)), 150, 2));

      const result = await ServiceCatalogApi.getAllServices(
        { search: 'compute', status: ServiceStatus.PUBLISHED, ciTypeId: 7 },
        5000,
      );

      expect(result.complete).toBe(true);
      expect(mockGet).toHaveBeenNthCalledWith(2, '/api/v1/service-catalogs', {
        page: 2,
        pageSize: 100,
        q: 'compute',
        status: 'enabled',
        ciTypeId: 7,
      });
    });

    it('末页判断用响应页长，后端把过大页长回落成 20 时也不会漏页', async () => {
      // 响应 pageSize=20 表示后端实际按 20 条返回，只有 18 条即末页。
      mockGet.mockResolvedValue({ items: Array.from({ length: 18 }, (_, i) => row(i + 1)), total: 18, page: 1, pageSize: 20, totalPages: 1 });

      const result = await ServiceCatalogApi.getAllServices({}, 5000);

      expect(result.services).toHaveLength(18);
      expect(result.complete).toBe(true);
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

    it('关联字段返回 0 表示未关联，布尔假值不得被兜底覆盖', async () => {
      mockGet.mockResolvedValue({
        id: 3,
        name: 'Email',
        category: 'it_service',
        status: 'disabled',
        description: '',
        deliveryTime: '0',
        requiresApproval: false,
        isActive: false,
        ciTypeId: 0,
        cloudServiceId: 0,
        price: 0,
        approvalLevel: 0,
      });
      const result = await ServiceCatalogApi.getService('3');

      // 后端恒定返回关联 id，0 就是「未关联」；组件的空值是 undefined。
      expect(result.ciTypeId).toBeUndefined();
      expect(result.cloudServiceId).toBeUndefined();
      expect(result.requiresApproval).toBe(false);
      expect(result.approvalLevel).toBe(0);
      expect(result.price).toBe(0);
      expect(result.status).toBe(ServiceStatus.RETIRED);
    });
  });

  describe('createService', () => {
    it('把 deliveryDays 映射成后端的天数字符串并始终携带 requiresApproval', async () => {
      mockPost.mockResolvedValue({ id: 2, name: 'VPN', status: 'enabled' });
      await ServiceCatalogApi.createService({
        name: 'VPN',
        category: 'it_service',
        shortDescription: '远程接入',
        deliveryDays: 3,
        requiresApproval: false,
      });

      const payload = mockPost.mock.calls[0][1];
      expect(payload).toEqual(
        expect.objectContaining({
          name: 'VPN',
          category: 'it_service',
          deliveryTime: '3',
          requiresApproval: false,
        }),
      );
    });
  });

  describe('updateService', () => {
    it('PATCH 只发送显式传入的键，false 与 0 必须真的写下去', async () => {
      mockPut.mockResolvedValue({ id: 1, name: 'Updated', status: 'enabled' });
      await ServiceCatalogApi.updateService('1', { name: 'Updated', price: 0, requiresApproval: false });

      const payload = mockPut.mock.calls[0][1];
      expect(payload).toEqual({ name: 'Updated', price: 0, requiresApproval: false });
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
    it('serviceId 转成数字 catalogId，reason 从 formData 取', async () => {
      mockPost.mockResolvedValue({ id: 1 });
      await ServiceCatalogApi.createServiceRequest({
        serviceId: '5',
        formData: { reason: 'Need access' },
      });
      expect(mockPost).toHaveBeenCalledWith(
        '/api/v1/service-requests',
        expect.objectContaining({ catalogId: 5, reason: 'Need access' }),
      );
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
