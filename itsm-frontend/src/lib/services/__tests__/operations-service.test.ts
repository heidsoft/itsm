import operationsService from '../operations-service';
import { httpClient } from '@/lib/api/http-client';

jest.mock('@/lib/api/http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
  },
}));

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;

describe('OperationsService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('查询死信命令时把筛选条件原样带给后端', async () => {
    mockGet.mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 20, summary: {} });

    await operationsService.listCommands({
      status: 'dead_letter',
      commandType: 'notification.deliver',
      page: 2,
      pageSize: 50,
    });

    expect(mockGet).toHaveBeenCalledWith('/api/v1/admin/operations/commands', {
      status: 'dead_letter',
      commandType: 'notification.deliver',
      page: 2,
      pageSize: 50,
    });
  });

  it('单条重放走 replay 端点且不带 body（保留原幂等键是后端语义）', async () => {
    mockPost.mockResolvedValue({ id: 7, status: 'pending' });

    await operationsService.replayCommand(7);

    expect(mockPost).toHaveBeenCalledWith('/api/v1/admin/operations/commands/7/replay');
  });

  it('批量重放把筛选条件作为 body，避免服务端退化成「全部重放」', async () => {
    mockPost.mockResolvedValue({ matchedIds: [1, 2], updated: 2, skipped: 0, pageSize: 200 });

    const result = await operationsService.bulkReplayCommands({
      status: 'dead_letter',
      commandType: 'notification.deliver',
    });

    expect(mockPost).toHaveBeenCalledWith('/api/v1/admin/operations/commands/bulk-replay', {
      status: 'dead_letter',
      commandType: 'notification.deliver',
    });
    expect(result.updated).toBe(2);
  });

  it('失败投递列表固定按 failed 查询，渠道为空时不传 channel', async () => {
    mockGet.mockResolvedValue({ items: [], pagination: { page: 1, pageSize: 20, total: 0 } });

    await operationsService.listDeliveries({ status: 'failed', page: 1, pageSize: 20 });

    expect(mockGet).toHaveBeenCalledWith('/api/v1/notification-deliveries', {
      status: 'failed',
      channel: undefined,
      page: 1,
      pageSize: 20,
    });
  });

  it('投递重试的错误原样抛出，由页面透出服务端 409 文案', async () => {
    const conflict = new Error('该投递已有在途重试（命令 #91，状态 pending）');
    mockPost.mockRejectedValue(conflict);

    await expect(operationsService.retryDelivery(5)).rejects.toThrow('已有在途重试');
    expect(mockPost).toHaveBeenCalledWith('/api/v1/notification-deliveries/5/retry');
  });
});
