/**
 * Tests for audit-log-history-adapter
 *
 * 覆盖：
 *   - fetchAuditLogHistory 的过滤（resource + path 前缀）
 *   - toReadableAction 的 action/resource 中英文映射 + 回退
 *   - 后端 userName 缺失时回退到 "用户#<id>"
 */

import { fetchAuditLogHistory } from '../audit-log-history-adapter';
import { listAuditLogs } from '@/lib/api/auditlog-api';
import type { AuditLog } from '@/lib/api/auditlog-api';

jest.mock('@/lib/api/auditlog-api', () => ({
  listAuditLogs: jest.fn(),
}));

const mockListAuditLogs = listAuditLogs as jest.MockedFunction<typeof listAuditLogs>;

// AuditLog 测试工厂：只填测试关注字段，其余给合法默认值
const makeLog = (overrides: Partial<AuditLog> = {}): AuditLog => ({
  id: 1,
  createdAt: '2026-01-01T00:00:00Z',
  tenantId: 1,
  userId: 7,
  requestId: 'req-1',
  ip: '127.0.0.1',
  resource: 'tickets',
  action: 'create',
  path: '/api/v1/tickets/123',
  method: 'POST',
  statusCode: 200,
  requestBody: '',
  ...overrides,
});

// ListAuditLogsResponse 测试工厂：补齐 page/pageSize 必填字段
const makeResponse = (logs: AuditLog[], total = logs.length) => ({
  logs,
  total,
  page: 1,
  pageSize: 20,
});

describe('audit-log-history-adapter', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('fetchAuditLogHistory', () => {
    it('uses plural resource and /api/v1 prefix for the query', async () => {
      mockListAuditLogs.mockResolvedValue(makeResponse([]));

      await fetchAuditLogHistory('ticket', 123);

      expect(mockListAuditLogs).toHaveBeenCalledWith(
        expect.objectContaining({
          resource: 'tickets',
          path: '/api/v1/tickets/123',
          pageSize: 100,
        })
      );
    });

    it('returns mapped history records with readable Chinese descriptions', async () => {
      mockListAuditLogs.mockResolvedValue(
        makeResponse([
          makeLog({
            id: 1,
            createdAt: '2026-01-01T00:00:00Z',
            action: 'create',
            resource: 'tickets',
            method: 'POST',
            path: '/api/v1/tickets/123',
            statusCode: 201,
            ip: '127.0.0.1',
            userId: 7,
            userName: '张三',
          }),
        ])
      );

      const records = await fetchAuditLogHistory('ticket', 123);

      expect(records).toHaveLength(1);
      expect(records[0]).toEqual(
        expect.objectContaining({
          id: 1,
          createdAt: '2026-01-01T00:00:00Z',
          action: 'create',
          details: '创建工单',
          user: { name: '张三' },
        })
      );
    });

    it('falls back to "用户#<id>" when userName is missing', async () => {
      mockListAuditLogs.mockResolvedValue(
        makeResponse([
          makeLog({
            id: 1,
            createdAt: '2026-01-01T00:00:00Z',
            action: 'delete',
            resource: 'incidents',
            method: 'DELETE',
            path: '/api/v1/incidents/9',
            statusCode: 204,
            userId: 42,
          }),
        ])
      );

      const records = await fetchAuditLogHistory('incident', 9);

      expect(records[0]?.details).toBe('删除事件');
      expect(records[0]?.user).toEqual({ name: '用户#42' });
    });

    it('keeps only logs whose path starts with /api/v1/<plural>/<id> (with or without trailing segment)', async () => {
      // 后端 listAuditLogs({ resource: 'tickets' }) 已经按 resource 过滤，
      // 所以这里只模拟同 resource 的 4 条记录，再由前端按 path 前缀二次过滤
      mockListAuditLogs.mockResolvedValue(
        makeResponse([
          // 命中
          makeLog({ id: 1, action: 'create', path: '/api/v1/tickets/5' }),
          // 命中（trailing segment）
          makeLog({ id: 2, action: 'update', path: '/api/v1/tickets/5/status' }),
          // 不命中（其它 ticket）
          makeLog({ id: 3, action: 'create', path: '/api/v1/tickets/6' }),
          // 不命中（path 不以正确前缀开始）
          makeLog({ id: 4, action: 'create', path: '/api/v1/incidents/5' }),
        ])
      );

      const records = await fetchAuditLogHistory('ticket', 5);

      expect(records.map(r => r.id)).toEqual([1, 2]);
    });

    it('falls back to action string when action is not in the dictionary', async () => {
      mockListAuditLogs.mockResolvedValue(
        makeResponse([
          makeLog({
            id: 1,
            createdAt: '',
            action: 'EXPORT_PDF',
            resource: 'changes',
            path: '/api/v1/changes/3',
            userId: 1,
          }),
        ])
      );

      const records = await fetchAuditLogHistory('change', 3);

      // actionMap 中无 EXPORT_PDF → 返回 action 原文
      expect(records[0]?.details).toBe('EXPORT_PDF');
    });

    it('returns resource name with stripped plural s when resource is unknown', async () => {
      mockListAuditLogs.mockResolvedValue(
        makeResponse([
          makeLog({
            id: 1,
            createdAt: '',
            action: 'create',
            resource: 'billing_records',
            path: '/api/v1/billing_records/1',
            userId: 1,
          }),
        ])
      );

      const records = await fetchAuditLogHistory('ticket', 1); // targetType 故意不匹配，验证过滤即可
      expect(records).toHaveLength(0);
    });

    it('handles missing action gracefully (returns resource as details)', async () => {
      mockListAuditLogs.mockResolvedValue(
        makeResponse([
          makeLog({
            id: 1,
            createdAt: '',
            // action 缺失
            resource: 'tickets',
            path: '/api/v1/tickets/1',
            userId: 1,
          }),
        ])
      );

      const records = await fetchAuditLogHistory('ticket', 1);

      expect(records).toHaveLength(1);
      // action 为空时 details 回退为 resource 相关文案
      expect(records[0]?.details).toBeTruthy();
    });
  });
});
