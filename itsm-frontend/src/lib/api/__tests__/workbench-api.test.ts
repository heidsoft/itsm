import { queryWorkbench, type WorkbenchItem, type WorkbenchResponse } from '../workbench-api';
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

const item = (patch: Partial<WorkbenchItem> = {}): WorkbenchItem => ({
  id: 1,
  recordType: 'ticket',
  title: '打印机无法联机',
  priority: 'high',
  status: 'open',
  phase: 'in_progress',
  tenantId: 7,
  createdAt: '2026-10-18T01:00:00Z',
  updatedAt: '2026-10-18T02:00:00Z',
  ...patch,
});

const listResponse = (items: WorkbenchItem[]): WorkbenchResponse => ({
  items,
  total: items.length,
  page: 1,
  pageSize: 20,
  totalPages: 1,
});

describe('queryWorkbench', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('空条件请求裸路径，不带问号', async () => {
    const resp = listResponse([]);
    mockGet.mockResolvedValue(resp);

    const result = await queryWorkbench({});

    expect(mockGet).toHaveBeenCalledWith('/api/v1/workbench');
    expect(mockGet).toHaveBeenCalledTimes(1);
    expect(result).toBe(resp);
  });

  it('多值过滤按逗号连接为单个 query 值', async () => {
    mockGet.mockResolvedValue(listResponse([]));

    await queryWorkbench({
      assigneeId: 42,
      phase: ['in_progress', 'pending_approval'],
      priority: ['high', 'urgent'],
      recordType: ['incident', 'change'],
      page: 2,
      pageSize: 50,
    });

    // URLSearchParams 把逗号转义为 %2C；Gin 的 c.Query 解码回 "," 后按逗号 split。
    expect(mockGet).toHaveBeenCalledWith(
      '/api/v1/workbench?assigneeId=42' +
        '&phase=in_progress%2Cpending_approval' +
        '&priority=high%2Curgent' +
        '&recordType=incident%2Cchange' +
        '&page=2&pageSize=50',
    );
  });

  it('assigneeId 用 !== undefined 判断，0 仍会发送', async () => {
    mockGet.mockResolvedValue(listResponse([]));

    await queryWorkbench({ assigneeId: 0 });

    expect(mockGet).toHaveBeenCalledWith('/api/v1/workbench?assigneeId=0');
  });

  it('空数组与 undefined 一样被省略，不产生空值参数', async () => {
    mockGet.mockResolvedValue(listResponse([]));

    await queryWorkbench({ phase: [], priority: [], recordType: [] });

    expect(mockGet).toHaveBeenCalledWith('/api/v1/workbench');
  });

  it('page/pageSize 为 0 时被省略（truthy 判断，与 skill-api 的 !== undefined 不同）', async () => {
    mockGet.mockResolvedValue(listResponse([]));

    await queryWorkbench({ page: 0, pageSize: 0 });

    // 第 0 页不是合法请求，省略后由后端回退默认分页；此用例锁定现有口径，避免被误改成透传 0。
    expect(mockGet).toHaveBeenCalledWith('/api/v1/workbench');
  });

  it('单元素数组不带多余分隔符', async () => {
    mockGet.mockResolvedValue(listResponse([]));

    await queryWorkbench({ recordType: ['problem'] });

    expect(mockGet).toHaveBeenCalledWith('/api/v1/workbench?recordType=problem');
  });

  it('query 键保持 camelCase，不混入 snake_case', async () => {
    mockGet.mockResolvedValue(listResponse([]));

    await queryWorkbench({ assigneeId: 9, recordType: ['ticket'], pageSize: 10 });

    const url = mockGet.mock.calls[0][0] as string;
    expect(url).toContain('assigneeId=9');
    expect(url).toContain('recordType=ticket');
    expect(url).toContain('pageSize=10');
    expect(url).not.toMatch(/[?&][a-z0-9]+_[a-z0-9]+=/);
    expect(url).not.toContain('record_type');
  });

  it('透传后端列表信封（集合只在 items 下）', async () => {
    const resp = listResponse([item(), item({ id: 2, recordType: 'incident' })]);
    mockGet.mockResolvedValue(resp);

    const result = await queryWorkbench({ page: 1 });

    expect(result.items).toHaveLength(2);
    expect(result.total).toBe(2);
    expect(result).not.toHaveProperty('workbench');
    expect(result).not.toHaveProperty('data');
  });

  it('后端错误原样 reject，不降级成空列表', async () => {
    mockGet.mockRejectedValue(new Error('permission denied'));

    await expect(queryWorkbench({ assigneeId: 42 })).rejects.toThrow('permission denied');
    expect(mockGet).toHaveBeenCalledTimes(1);
  });
});
