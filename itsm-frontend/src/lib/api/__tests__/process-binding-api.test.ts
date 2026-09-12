import { ProcessBindingApi } from '../process-binding-api';
import type { ProcessBindingPayload } from '../process-binding-api';
import { httpClient } from '../http-client';

jest.mock('../http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
  },
}));

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;
const mockPut = httpClient.put as jest.Mock;
const mockDelete = httpClient.delete as jest.Mock;

describe('ProcessBindingApi', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('list', () => {
    it('should list bindings without query', async () => {
      mockGet.mockResolvedValue([{ id: 1, businessType: 'ticket', processDefinitionKey: 'proc1' }]);
      const result = await ProcessBindingApi.list();
      expect(mockGet).toHaveBeenCalledWith('/api/v1/process-bindings', undefined);
      expect(result[0].id).toBe(1);
      expect(result[0].businessType).toBe('ticket');
    });

    it('should list bindings with query', async () => {
      mockGet.mockResolvedValue([]);
      await ProcessBindingApi.list({ businessType: 'change', isActive: true });
      expect(mockGet).toHaveBeenCalledWith('/api/v1/process-bindings', { businessType: 'change', isActive: true });
    });

    it('should normalize with defaults', async () => {
      mockGet.mockResolvedValue([{}]);
      const result = await ProcessBindingApi.list();
      expect(result[0].id).toBe(0);
      expect(result[0].businessType).toBe('');
      expect(result[0].processDefinitionKey).toBe('');
      expect(result[0].isDefault).toBe(false);
      expect(result[0].priority).toBe(0);
      expect(result[0].isActive).toBe(false);
    });

    it('ProcessBindingPayload excludes server-managed fields (compile-time gate)', () => {
      // Fix for #1: 旧版 Omit 用 snake_case（tenant_id/created_at/updated_at），
      // 而 ProcessBinding 是 camelCase → Omit 是 no-op，创建/更新负载会带上
      // tenantId/createdAt/updatedAt。下面 3 处 @ts-expect-error 一旦不再报红，
      // 说明 Omit 又坏回去了，能在 tsc 阶段挡住回归。
      // 必须用直接对象字面量（不能 spread），否则 TS excess property check 不触发。
      const valid: ProcessBindingPayload = {
        businessType: 'ticket',
        processDefinitionKey: 'proc1',
        priority: 0,
        isActive: true,
        isDefault: false,
      };
      expect(valid.businessType).toBe('ticket');
      // 编译时断言：以下三个赋值都应因 excess property check 报红
      // @ts-expect-error tenantId must be omitted (server-managed)
      const _t: ProcessBindingPayload = { ...valid, tenantId: 1 };
      // @ts-expect-error createdAt must be omitted (server-managed)
      const _c: ProcessBindingPayload = { ...valid, createdAt: 'x' };
      // @ts-expect-error updatedAt must be omitted (server-managed)
      const _u: ProcessBindingPayload = { ...valid, updatedAt: 'x' };
      expect(_t && _c && _u).toBeTruthy();
    });

    it('should handle null response', async () => {
      mockGet.mockResolvedValue(null);
      const result = await ProcessBindingApi.list();
      expect(result).toEqual([]);
    });
  });

  describe('get', () => {
    it('should get binding by id', async () => {
      mockGet.mockResolvedValue({ id: 1, businessType: 'ticket', processDefinitionKey: 'p1', priority: 10 });
      const result = await ProcessBindingApi.get(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/process-bindings/1');
      expect(result.businessType).toBe('ticket');
      expect(result.priority).toBe(10);
    });
  });

  describe('create', () => {
    it('should create binding and clean payload', async () => {
      mockPost.mockResolvedValue({ id: 1, businessType: 'ticket', processDefinitionKey: 'p1' });
      const payload = { businessType: 'ticket', processDefinitionKey: 'p1', isActive: true, priority: 1, isDefault: false };
      const result = await ProcessBindingApi.create(payload as any);
      expect(mockPost).toHaveBeenCalledWith('/api/v1/process-bindings', expect.objectContaining({ businessType: 'ticket' }));
      expect(result.id).toBe(1);
    });

    it('should strip undefined/null/empty values', async () => {
      mockPost.mockResolvedValue({ id: 1, businessType: 'ticket', processDefinitionKey: 'p1' });
      const payload = { businessType: 'ticket', processDefinitionKey: 'p1', isActive: true, priority: 1, isDefault: false, businessSubType: undefined, scenario: '' };
      await ProcessBindingApi.create(payload as any);
      const calledWith = mockPost.mock.calls[0][1];
      expect(calledWith.businessSubType).toBeUndefined();
      expect(calledWith.scenario).toBeUndefined();
    });
  });

  describe('update', () => {
    it('should update binding', async () => {
      mockPut.mockResolvedValue({ id: 1, businessType: 'ticket', processDefinitionKey: 'p2' });
      const result = await ProcessBindingApi.update(1, { processDefinitionKey: 'p2' } as any);
      expect(mockPut).toHaveBeenCalledWith('/api/v1/process-bindings/1', { processDefinitionKey: 'p2' });
      expect(result.processDefinitionKey).toBe('p2');
    });
  });

  describe('delete', () => {
    it('should delete binding', async () => {
      mockDelete.mockResolvedValue(undefined);
      await ProcessBindingApi.delete(1);
      expect(mockDelete).toHaveBeenCalledWith('/api/v1/process-bindings/1');
    });
  });

  describe('listDepartmentProcesses', () => {
    it('should list department processes', async () => {
      mockGet.mockResolvedValue([{ id: 1, businessType: 'ticket' }]);
      const result = await ProcessBindingApi.listDepartmentProcesses(5);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/departments/5/processes');
      expect(result).toHaveLength(1);
    });

    it('should handle null response', async () => {
      mockGet.mockResolvedValue(null);
      const result = await ProcessBindingApi.listDepartmentProcesses(5);
      expect(result).toEqual([]);
    });
  });

  describe('initDepartmentProcesses', () => {
    it('should init department processes', async () => {
      mockPost.mockResolvedValue(undefined);
      await ProcessBindingApi.initDepartmentProcesses(5, 'IT');
      expect(mockPost).toHaveBeenCalledWith('/api/v1/departments/5/init-processes', { departmentType: 'IT' });
    });
  });

  describe('error propagation', () => {
    it('should propagate errors', async () => {
      mockGet.mockRejectedValue(new Error('Binding not found'));
      await expect(ProcessBindingApi.get(999)).rejects.toThrow('Binding not found');
    });
  });
});
