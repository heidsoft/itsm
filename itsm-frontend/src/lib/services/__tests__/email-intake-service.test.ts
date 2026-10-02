/**
 * EmailIntakeService 契约测试
 *
 * 路径/动词/请求体逐项对齐后端已注册路由与请求 DTO
 * （itsm-backend/handlers/email_intake/handler.go RegisterRoutes 与
 * conversationVersionRequest / correctionRequest / overrideRequest）。
 * 邮件 intake 的生产页面（src/app/(main)/email-intake/**）直接调用本 service，
 * 因此这里是该能力唯一的前端契约锁。
 */
import { emailIntakeService } from '../emailIntakeService';
import { httpClient } from '@/lib/api/http-client';

jest.mock('@/lib/api/http-client', () => ({
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

const BASE = '/api/v1/email-intake';

const conversation = {
  id: 7,
  conversationToken: 'tok-7',
  status: 'MANUAL_REVIEW',
  confidence: 0.42,
  missingFields: ['branchId'],
  version: 3,
  lastMessageAt: '2026-10-02T00:00:00Z',
  createdAt: '2026-10-01T00:00:00Z',
};

describe('EmailIntakeService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('conversations', () => {
    it('GET /conversations 必须显式发送 page/pageSize，否则后端默认只给 20 条', async () => {
      const payload = {
        items: [conversation],
        total: 25,
        page: 2,
        pageSize: 10,
        totalPages: 3,
      };
      mockGet.mockResolvedValueOnce(payload);

      const result = await emailIntakeService.conversations({
        page: 2,
        pageSize: 10,
        status: 'MANUAL_REVIEW',
      });

      expect(mockGet).toHaveBeenCalledWith(`${BASE}/conversations`, {
        page: 2,
        pageSize: 10,
        status: 'MANUAL_REVIEW',
      });
      expect(result).toEqual(payload);
    });

    it('缺 status 时仍发送分页参数，status 留空由 httpClient 过滤', async () => {
      mockGet.mockResolvedValueOnce({ items: [], total: 0, page: 1, pageSize: 20, totalPages: 0 });

      await emailIntakeService.conversations({ page: 1, pageSize: 20 });

      const [url, query] = mockGet.mock.calls[0];
      expect(url).toBe(`${BASE}/conversations`);
      expect(query).toEqual(
        expect.objectContaining({ page: 1, pageSize: 20 }) as Record<string, unknown>
      );
      expect(query.status).toBeUndefined();
    });
  });

  describe('conversation', () => {
    it('GET /conversations/:id 返回含消息与分析的详情', async () => {
      const detail = { ...conversation, messages: [], analyses: [], outboundMessages: [] };
      mockGet.mockResolvedValueOnce(detail);

      await expect(emailIntakeService.conversation(7)).resolves.toEqual(detail);
      expect(mockGet).toHaveBeenCalledWith(`${BASE}/conversations/7`);
    });
  });

  describe('状态迁移动作（必须携带乐观锁 version）', () => {
    it.each([
      ['revalidate', 'revalidate'],
      ['confirm', 'confirm'],
      ['reject', 'reject'],
      ['retry', 'retry'],
    ] as const)('POST /conversations/7/%s 只发送 { version }', async (method, segment) => {
      mockPost.mockResolvedValueOnce(conversation);

      const result = await emailIntakeService[method](7, 3);

      expect(mockPost).toHaveBeenCalledWith(`${BASE}/conversations/7/${segment}`, { version: 3 });
      expect(result).toEqual(conversation);
    });

    it('POST /conversations/:id/corrections 同时发送 version 与 fields', async () => {
      mockPost.mockResolvedValueOnce(conversation);

      await emailIntakeService.correct(7, 3, { customerId: 11, branchId: 22 });

      expect(mockPost).toHaveBeenCalledWith(`${BASE}/conversations/7/corrections`, {
        version: 3,
        fields: { customerId: 11, branchId: 22 },
      });
    });

    it('POST /conversations/:id/override 必须带 reason 且 confirmed 为 true', async () => {
      // 后端 overrideRequest.confirmed 为 required 且 false 时直接参数错误，
      // 因此前端不得省略或发送 false。
      mockPost.mockResolvedValueOnce(conversation);

      await emailIntakeService.override(7, 3, '合同已线下确认，人工放行');

      expect(mockPost).toHaveBeenCalledWith(`${BASE}/conversations/7/override`, {
        version: 3,
        reason: '合同已线下确认，人工放行',
        confirmed: true,
      });
    });
  });

  describe('客户与分支主数据', () => {
    const customer = {
      id: 11,
      name: '示例客户',
      shortName: '示例',
      aliases: ['Example'],
      historicalNames: [],
      status: 'active',
      createdAt: '2026-10-01T00:00:00Z',
      updatedAt: '2026-10-02T00:00:00Z',
    };

    it('GET /customers 读取 items 信封', async () => {
      mockGet.mockResolvedValueOnce({ items: [customer], total: 1 });

      await expect(emailIntakeService.customers()).resolves.toEqual({ items: [customer], total: 1 });
      expect(mockGet).toHaveBeenCalledWith(`${BASE}/customers`);
    });

    it('POST /customers 原样透传 payload', async () => {
      const payload = { ...customer };
      delete (payload as Partial<typeof customer>).id;
      mockPost.mockResolvedValueOnce(customer);

      await emailIntakeService.createCustomer(payload);

      expect(mockPost).toHaveBeenCalledWith(`${BASE}/customers`, payload);
    });

    it('PUT /customers/:id 走更新而不是重建', async () => {
      mockPut.mockResolvedValueOnce({ ...customer, name: '新名字' });

      await emailIntakeService.updateCustomer(11, { name: '新名字' } as never);

      expect(mockPut).toHaveBeenCalledWith(`${BASE}/customers/11`, { name: '新名字' });
    });

    it('DELETE /customers/:id 用于停用（后端软删除，无响应体）', async () => {
      mockDelete.mockResolvedValueOnce(undefined);

      await expect(emailIntakeService.disableCustomer(11)).resolves.toBeUndefined();
      expect(mockDelete).toHaveBeenCalledWith(`${BASE}/customers/11`);
    });

    it('GET /branches 带与不带 customerId 的两种形态', async () => {
      mockGet.mockResolvedValueOnce({ items: [], total: 0 });

      await emailIntakeService.branches(11);
      expect(mockGet).toHaveBeenLastCalledWith(`${BASE}/branches`, { customerId: 11 });

      await emailIntakeService.branches();
      expect(mockGet).toHaveBeenLastCalledWith(`${BASE}/branches`, undefined);
    });

    it('POST /branches 与 PUT /branches/:id、DELETE /branches/:id', async () => {
      mockPost.mockResolvedValueOnce({ id: 22, customerId: 11, name: '华东', aliases: [], status: 'active' });
      await emailIntakeService.createBranch({ customerId: 11, name: '华东', aliases: [], status: 'active' });
      expect(mockPost).toHaveBeenCalledWith(`${BASE}/branches`, {
        customerId: 11,
        name: '华东',
        aliases: [],
        status: 'active',
      });

      mockPut.mockResolvedValueOnce({ id: 22, customerId: 11, name: '华中华东', aliases: [], status: 'active' });
      await emailIntakeService.updateBranch(22, { name: '华中华东', customerId: 11 });
      expect(mockPut).toHaveBeenCalledWith(`${BASE}/branches/22`, { name: '华中华东', customerId: 11 });

      mockDelete.mockResolvedValueOnce(undefined);
      await emailIntakeService.disableBranch(22);
      expect(mockDelete).toHaveBeenCalledWith(`${BASE}/branches/22`);
    });
  });

  describe('支持合同与外部合同号', () => {
    it('合同 CRUD 与终止使用 /support-contracts', async () => {
      mockGet.mockResolvedValueOnce({ items: [], total: 0 });
      await emailIntakeService.contracts();
      expect(mockGet).toHaveBeenCalledWith(`${BASE}/support-contracts`);

      const payload = { customerId: 11, contractNumber: 'C-1', status: 'active' as const };
      mockPost.mockResolvedValueOnce({ id: 31, ...payload });
      await emailIntakeService.createContract(payload);
      expect(mockPost).toHaveBeenCalledWith(`${BASE}/support-contracts`, payload);

      mockPut.mockResolvedValueOnce({ id: 31, ...payload, status: 'expired' });
      await emailIntakeService.updateContract(31, { ...payload, status: 'expired' });
      expect(mockPut).toHaveBeenCalledWith(`${BASE}/support-contracts/31`, {
        ...payload,
        status: 'expired',
      });

      mockDelete.mockResolvedValueOnce(undefined);
      await expect(emailIntakeService.terminateContract(31)).resolves.toBeUndefined();
      expect(mockDelete).toHaveBeenCalledWith(`${BASE}/support-contracts/31`);
    });

    it('来源组织 CRUD 使用 /source-organizations', async () => {
      mockGet.mockResolvedValueOnce({ items: [], total: 0 });
      await emailIntakeService.sourceOrganizations();
      expect(mockGet).toHaveBeenCalledWith(`${BASE}/source-organizations`);

      const payload = { name: '示例发件方', emailAddresses: [], emailDomains: ['a.com'], status: 'active' };
      mockPost.mockResolvedValueOnce({ id: 41, ...payload });
      await emailIntakeService.createSourceOrganization(payload);
      expect(mockPost).toHaveBeenCalledWith(`${BASE}/source-organizations`, payload);

      mockPut.mockResolvedValueOnce({ id: 41, ...payload });
      await emailIntakeService.updateSourceOrganization(41, { name: '改名' });
      expect(mockPut).toHaveBeenCalledWith(`${BASE}/source-organizations/41`, { name: '改名' });

      mockDelete.mockResolvedValueOnce(undefined);
      await emailIntakeService.disableSourceOrganization(41);
      expect(mockDelete).toHaveBeenCalledWith(`${BASE}/source-organizations/41`);
    });

    it('外部合同号引用使用 /external-contract-references', async () => {
      mockGet.mockResolvedValueOnce({ items: [], total: 0 });
      await emailIntakeService.externalContractReferences();
      expect(mockGet).toHaveBeenCalledWith(`${BASE}/external-contract-references`);

      const payload = {
        sourceOrganizationId: 41,
        supportContractId: 31,
        externalContractNumber: 'EXT-1',
      };
      mockPost.mockResolvedValueOnce({ id: 51, customerId: 11, createdAt: '', ...payload });
      await emailIntakeService.createExternalContractReference(payload);
      expect(mockPost).toHaveBeenCalledWith(`${BASE}/external-contract-references`, payload);

      mockDelete.mockResolvedValueOnce(undefined);
      await emailIntakeService.deleteExternalContractReference(51);
      expect(mockDelete).toHaveBeenCalledWith(`${BASE}/external-contract-references/51`);
    });
  });

  describe('值班排班', () => {
    it('排班使用 /on-call/schedules 与 /on-call/shifts', async () => {
      mockGet.mockResolvedValueOnce({ items: [], total: 0 });
      await emailIntakeService.schedules();
      expect(mockGet).toHaveBeenCalledWith(`${BASE}/on-call/schedules`);

      const schedule = { id: 61, groupId: 5, name: '一线', timezone: 'Asia/Shanghai', status: 'active' };
      mockPost.mockResolvedValueOnce(schedule);
      await emailIntakeService.createSchedule(schedule);
      expect(mockPost).toHaveBeenCalledWith(`${BASE}/on-call/schedules`, schedule);

      const shiftPayload = { scheduleId: 61, userId: 9, startAt: 'a', endAt: 'b' };
      mockPost.mockResolvedValueOnce(undefined);
      await emailIntakeService.createShift(shiftPayload);
      expect(mockPost).toHaveBeenCalledWith(`${BASE}/on-call/shifts`, shiftPayload);

      mockGet.mockResolvedValueOnce({ items: [], total: 0 });
      await emailIntakeService.shifts(61);
      expect(mockGet).toHaveBeenLastCalledWith(`${BASE}/on-call/shifts`, { scheduleId: 61 });

      await emailIntakeService.shifts();
      expect(mockGet).toHaveBeenLastCalledWith(`${BASE}/on-call/shifts`, undefined);

      const shift = { id: 71, ...shiftPayload };
      mockPut.mockResolvedValueOnce(shift);
      await emailIntakeService.updateShift(71, shiftPayload);
      expect(mockPut).toHaveBeenCalledWith(`${BASE}/on-call/shifts/71`, shiftPayload);

      mockDelete.mockResolvedValueOnce(undefined);
      await emailIntakeService.deleteShift(71);
      expect(mockDelete).toHaveBeenCalledWith(`${BASE}/on-call/shifts/71`);
    });

    it('当前值班人查询按 groupId', async () => {
      mockGet.mockResolvedValueOnce(null);

      await expect(emailIntakeService.currentOnCall(5)).resolves.toBeNull();
      expect(mockGet).toHaveBeenCalledWith(`${BASE}/on-call/current`, { groupId: 5 });
    });
  });

  describe('失败传播', () => {
    it('后端冲突（version 不匹配）必须原样抛出，不得伪装成功', async () => {
      mockPost.mockRejectedValueOnce(new Error('版本冲突，请刷新后重试'));

      await expect(emailIntakeService.confirm(7, 2)).rejects.toThrow('版本冲突，请刷新后重试');
      expect(mockGet).not.toHaveBeenCalled();
    });
  });
});
