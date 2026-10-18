import { skillApi } from '../skill-api';
import skillApiDefault, { type SkillEntry, type SkillUpsertRequest } from '../skill-api';
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

// 与后端 handlers/skill/handler.go 的 service.SkillEntry 逐键一致（camelCase）。
const skill = (patch: Partial<SkillEntry> = {}): SkillEntry => ({
  code: 'ticket-summarize',
  name: '工单摘要',
  version: '1.0.0',
  title: 'Ticket Summarize',
  provider: 'builtin',
  category: 'ga',
  tags: ['ai'],
  capabilities: ['ticket:read'],
  requiredPermissions: ['ai:read'],
  isOfficial: true,
  checksum: 'sha256:abc',
  isBuiltin: true,
  status: 'active',
  manifest: {},
  metrics: { totalCalls: 12, successRate: 1, avgLatencyMs: 40, errorCount: 0 },
  ...patch,
});

const upsert = (patch: Partial<SkillUpsertRequest> = {}): SkillUpsertRequest => ({
  code: 'ticket-summarize',
  version: '1.1.0',
  title: 'Ticket Summarize',
  category: 'pilot',
  tags: ['ai'],
  capabilities: ['ticket:read'],
  requiredPermissions: ['ai:read'],
  provider: 'custom',
  ...patch,
});

describe('skillApi', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('list', () => {
    it('无筛选条件时请求裸路径，不带问号', async () => {
      const body = { items: [skill()], total: 1, page: 1, pageSize: 20, totalPages: 1 };
      mockGet.mockResolvedValue(body);

      const result = await skillApi.list();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/admin/skills');
      expect(mockGet).toHaveBeenCalledTimes(1);
      expect(result).toBe(body);
    });

    it('默认参数与不传参数等价', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 20, totalPages: 0 });

      await skillApi.list({});

      expect(mockGet).toHaveBeenCalledWith('/api/v1/admin/skills');
    });

    it('全部筛选条件按 tag/category/status/q/page/pageSize 顺序拼进 query', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0, page: 2, pageSize: 50, totalPages: 0 });

      await skillApi.list({
        tag: 'ai',
        category: 'pilot',
        status: 'disabled',
        q: '摘要',
        page: 2,
        pageSize: 50,
      });

      expect(mockGet).toHaveBeenCalledWith(
        '/api/v1/admin/skills?tag=ai&category=pilot&status=disabled&q=%E6%91%98%E8%A6%81&page=2&pageSize=50',
      );
    });

    it('分页参数用 !== undefined 判断，page/pageSize 为 0 仍会发送', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 20, totalPages: 0 });

      await skillApi.list({ page: 0, pageSize: 0 });

      // 后端 List 把 ≤0 回退为默认值，但那是服务端契约；前端不得静默丢参数。
      expect(mockGet).toHaveBeenCalledWith('/api/v1/admin/skills?page=0&pageSize=0');
    });

    it('空字符串过滤项不进入 query', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 20, totalPages: 0 });

      await skillApi.list({ tag: '', category: 'ga', q: '' });

      expect(mockGet).toHaveBeenCalledWith('/api/v1/admin/skills?category=ga');
    });

    it('保留字符按 application/x-www-form-urlencoded 转义', async () => {
      mockGet.mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 20, totalPages: 0 });

      await skillApi.list({ tag: 'ai &ml', q: 'a=b&c' });

      expect(mockGet).toHaveBeenCalledWith(
        '/api/v1/admin/skills?tag=ai+%26ml&q=a%3Db%26c',
      );
    });
  });

  describe('get', () => {
    it('按 code 请求详情', async () => {
      const entry = skill();
      mockGet.mockResolvedValue(entry);

      const result = await skillApi.get('ticket-summarize');

      expect(mockGet).toHaveBeenCalledWith('/api/v1/admin/skills/ticket-summarize');
      expect(result).toBe(entry);
    });

    it('对 code 做路径转义，避免斜杠被当成新路径段', async () => {
      mockGet.mockResolvedValue(skill());

      await skillApi.get('ns/skill é');

      expect(mockGet).toHaveBeenCalledWith('/api/v1/admin/skills/ns%2Fskill%20%C3%A9');
    });
  });

  describe('create', () => {
    it('POST 列表端点并原样透传 manifest 请求体', async () => {
      const body = upsert();
      const created = skill({ status: 'active' });
      mockPost.mockResolvedValue(created);

      const result = await skillApi.create(body);

      expect(mockPost).toHaveBeenCalledTimes(1);
      expect(mockPost.mock.calls[0][0]).toBe('/api/v1/admin/skills');
      expect(mockPost.mock.calls[0][1]).toEqual(body);
      expect(result).toBe(created);
    });

    it('不自报 tenantId 等身份字段，归属由后端上下文决定', async () => {
      mockPost.mockResolvedValue(skill());

      await skillApi.create(upsert());

      const payload = mockPost.mock.calls[0][1] as Record<string, unknown>;
      expect(payload).not.toHaveProperty('tenantId');
      expect(payload).not.toHaveProperty('tenant_id');
      expect(payload).not.toHaveProperty('operatorId');
    });
  });

  describe('update', () => {
    it('PUT code 路径并返回后端确认的 updated/manifest', async () => {
      const resp = { updated: true, manifest: { iconUrl: 'https://cdn/icon.png' } } as const;
      mockPut.mockResolvedValue(resp);

      const body = upsert({ inputSchema: { type: 'object' } });
      const result = await skillApi.update('my-skill', body);

      expect(mockPut).toHaveBeenCalledWith('/api/v1/admin/skills/my-skill', body);
      expect(result).toEqual({
        updated: true,
        manifest: { iconUrl: 'https://cdn/icon.png' },
      });
    });
  });

  describe('promote', () => {
    it('POST promote 动作段，携带空对象 body 而非 undefined', async () => {
      mockPost.mockResolvedValue({ category: 'ga' });

      const result = await skillApi.promote('my-skill');

      expect(mockPost).toHaveBeenCalledWith('/api/v1/admin/skills/my-skill/promote', {});
      expect(result).toEqual({ category: 'ga' });
    });

    it('转义 code', async () => {
      mockPost.mockResolvedValue({ category: 'ga' });

      await skillApi.promote('a b');

      expect(mockPost).toHaveBeenCalledWith('/api/v1/admin/skills/a%20b/promote', {});
    });
  });

  describe('disable', () => {
    it('DELETE code 路径且不发送 body', async () => {
      mockDelete.mockResolvedValue({ disabled: true });

      const result = await skillApi.disable('my-skill');

      expect(mockDelete).toHaveBeenCalledTimes(1);
      expect(mockDelete.mock.calls[0]).toEqual(['/api/v1/admin/skills/my-skill']);
      expect(result).toEqual({ disabled: true });
    });
  });

  it('后端错误原样 reject，不吞成空结果', async () => {
    mockGet.mockRejectedValue(new Error('skill not found'));

    await expect(skillApi.get('missing')).rejects.toThrow('skill not found');
  });

  it('默认导出与 skillApi 是同一对象', () => {
    expect(skillApiDefault).toBe(skillApi);
  });
});
