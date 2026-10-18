/**
 * TicketService 契约测试：URL、method、请求体与返回值透传。
 *
 * 后端路由契约以 itsm-backend/router 为准，路径与 method 逐条断言完整字面量，
 * 不用 stringContaining 掩盖漂移。
 */
import { ticketService } from '../ticket-service';
import TicketService, {
  TicketPriority as ReExportedPriority,
  TicketStatus as ReExportedStatus,
} from '../ticket-service';
import { httpClient } from '@/lib/api/http-client';
import type {
  CreateTicketRequest,
  GetTicketsParams,
  Ticket,
  TicketExportRequest,
  TicketListResponse,
  TicketStatsResponse,
  UpdateTicketRequest,
} from '@/lib/api/ticket-api';
import { TicketPriority, TicketStatus } from '@/constants/taxonomy';

jest.mock('@/lib/api/http-client', () => ({
  httpClient: {
    get: jest.fn(),
    post: jest.fn(),
    put: jest.fn(),
    delete: jest.fn(),
    patch: jest.fn(),
    request: jest.fn(),
  },
}));

const mockGet = httpClient.get as jest.Mock;
const mockPost = httpClient.post as jest.Mock;
const mockPut = httpClient.put as jest.Mock;
const mockDelete = httpClient.delete as jest.Mock;
const mockRequest = httpClient.request as jest.Mock;

const ticket: Ticket = {
  id: 15,
  ticketNumber: 'TKT-000015',
  title: 'VPN 无法连接',
  description: '',
  status: TicketStatus.NEW,
  priority: TicketPriority.HIGH,
  type: 'ticket',
  requesterId: 3,
  tenantId: 1,
  version: 1,
  createdAt: '2026-10-18T01:00:00Z',
  updatedAt: '2026-10-18T01:00:00Z',
};

const listResponse: TicketListResponse = {
  items: [ticket],
  total: 1,
  page: 1,
  pageSize: 20,
  totalPages: 1,
};

describe('ticketService', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('再导出的词表与 @/constants/taxonomy 同源，本模块不另立枚举', () => {
    expect(ReExportedStatus.IN_PROGRESS).toBe(TicketStatus.IN_PROGRESS);
    expect(ReExportedPriority.URGENT).toBe(TicketPriority.URGENT);
    expect(ticketService).toBeInstanceOf(TicketService);
  });

  describe('listTickets', () => {
    const params: GetTicketsParams = { page: 2, pageSize: 50, status: 'open', isOverdue: true };

    it('GET /api/v1/tickets 并原样透传筛选参数', async () => {
      mockGet.mockResolvedValueOnce(listResponse);

      await expect(ticketService.listTickets(params)).resolves.toBe(listResponse);

      expect(mockGet).toHaveBeenCalledTimes(1);
      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets', params);
    });

    it('缺省参数时发送空对象，不伪造默认分页', async () => {
      mockGet.mockResolvedValueOnce(listResponse);

      await ticketService.listTickets();

      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets', {});
    });
  });

  it('getTicket GET /api/v1/tickets/15', async () => {
    mockGet.mockResolvedValueOnce(ticket);

    await expect(ticketService.getTicket(15)).resolves.toBe(ticket);

    expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/15');
  });

  it('createTicket POST /api/v1/tickets 带请求体', async () => {
    const data: CreateTicketRequest = {
      title: '需要新增 GitLab 账号',
      priority: TicketPriority.MEDIUM,
      ticketTypeId: 7,
      formFields: { accountType: 'new' },
    };
    mockPost.mockResolvedValueOnce(ticket);

    await expect(ticketService.createTicket(data)).resolves.toBe(ticket);

    expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets', data);
  });

  it('updateTicket PUT /api/v1/tickets/15 带乐观锁版本', async () => {
    const data: UpdateTicketRequest = {
      status: TicketStatus.IN_PROGRESS,
      version: 1,
    };
    mockPut.mockResolvedValueOnce({ message: 'updated', ticketId: 15 });

    await expect(ticketService.updateTicket(15, data)).resolves.toEqual({
      message: 'updated',
      ticketId: 15,
    });

    expect(mockPut).toHaveBeenCalledWith('/api/v1/tickets/15', {
      status: TicketStatus.IN_PROGRESS,
      version: 1,
    });
  });

  it('deleteTicket DELETE /api/v1/tickets/15', async () => {
    mockDelete.mockResolvedValueOnce({ message: 'deleted', ticketId: 15 });

    await ticketService.deleteTicket(15);

    expect(mockDelete).toHaveBeenCalledWith('/api/v1/tickets/15');
  });

  it('getTicketStats GET /api/v1/tickets/stats', async () => {
    const stats: TicketStatsResponse = {
      total: 2,
      open: 1,
      inProgress: 1,
      resolved: 0,
      closed: 0,
      pending: 0,
      highPriority: 1,
      overdue: 0,
      byStatus: [{ status: 'open', count: 1 }],
      byPriority: [{ priority: 'high', count: 1 }],
    };
    mockGet.mockResolvedValueOnce(stats);

    await expect(ticketService.getTicketStats()).resolves.toBe(stats);

    expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/stats');
  });

  it('assignTicket POST /api/v1/tickets/15/assign', async () => {
    mockPost.mockResolvedValueOnce({ message: 'assigned', ticketId: 15 });

    await ticketService.assignTicket(15, { assigneeId: 9 });

    expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/15/assign', { assigneeId: 9 });
  });

  describe('评论', () => {
    it('getTicketComments GET /api/v1/tickets/15/comments', async () => {
      mockGet.mockResolvedValueOnce([]);

      await ticketService.getTicketComments(15);

      expect(mockGet).toHaveBeenCalledWith('/api/v1/tickets/15/comments');
    });

    it('addTicketComment 默认公开（isInternal=false）', async () => {
      mockPost.mockResolvedValueOnce({ message: 'ok', commentId: 4 });

      await ticketService.addTicketComment(15, '已联系用户');

      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/15/comments', {
        content: '已联系用户',
        isInternal: false,
      });
    });

    it('addTicketComment 显式内部备注', async () => {
      mockPost.mockResolvedValueOnce({ message: 'ok', commentId: 5 });

      await ticketService.addTicketComment(15, '内部排查记录', true);

      expect(mockPost).toHaveBeenCalledWith('/api/v1/tickets/15/comments', {
        content: '内部排查记录',
        isInternal: true,
      });
    });
  });

  describe('附件', () => {
    it('uploadTicketAttachment 用 multipart 提交单个 file 字段', async () => {
      const file = new File(['abc'], 'log.txt', { type: 'text/plain' });
      mockPost.mockResolvedValueOnce({ message: 'ok', attachmentId: 6 });

      await ticketService.uploadTicketAttachment(15, file);

      expect(mockPost).toHaveBeenCalledTimes(1);
      const [url, body] = mockPost.mock.calls[0];
      expect(url).toBe('/api/v1/tickets/15/attachments');
      expect(body).toBeInstanceOf(FormData);
      expect((body as FormData).get('file')).toBe(file);
    });

    it('deleteTicketAttachment DELETE /api/v1/tickets/15/attachments/6', async () => {
      mockDelete.mockResolvedValueOnce({ message: 'ok', attachmentId: 6 });

      await ticketService.deleteTicketAttachment(15, 6);

      expect(mockDelete).toHaveBeenCalledWith('/api/v1/tickets/15/attachments/6');
    });
  });

  describe('exportTickets', () => {
    const data: TicketExportRequest = { format: 'csv', filters: { status: 'open' } };

    it('走 request 的 blob 通道，而不是会被按 JSON 解析的 post', async () => {
      const blob = new Blob(['csv'], { type: 'text/csv' });
      mockRequest.mockResolvedValueOnce(blob);

      await expect(ticketService.exportTickets(data)).resolves.toBe(blob);

      expect(mockRequest).toHaveBeenCalledTimes(1);
      expect(mockRequest).toHaveBeenCalledWith({
        method: 'POST',
        url: '/api/v1/tickets/export',
        data,
        responseType: 'blob',
      });
      expect(mockPost).not.toHaveBeenCalled();
    });
  });

  it('服务端错误原样抛出，不在 service 层吞掉', async () => {
    const failure = new Error('conflict');
    mockPut.mockRejectedValueOnce(failure);

    await expect(ticketService.updateTicket(15, { title: 'x' })).rejects.toThrow('conflict');
  });
});
