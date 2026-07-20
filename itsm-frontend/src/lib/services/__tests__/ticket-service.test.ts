/**
 * Real tests for `ticket-service-v2.ts`.
 *
 * Replaces the previous 67-line `ticket-service.test.ts`, which contained
 * only `typeof X === 'function'` assertions — never once calling the
 * service and never once verifying it sends the right HTTP request.
 *
 * What we exercise (TicketService extends BaseService, which delegates to
 * `httpClient.{get,post,put,patch,delete}`):
 *   - getTickets / getTicket  → correct path + query params
 *   - createTicket / updateTicket / deleteTicket → CRUD verbs
 *   - updateStatus / assign / resolve / close / reopen / escalate → workflow
 *   - approve / reject / accept / withdraw / forward → workflow actions
 *   - getComments / addComment / updateComment / deleteComment
 *   - searchTickets / getStats
 *   - error propagation when backend returns non-zero code
 */

import { ticketService } from '../ticket-service-v2';

jest.mock('@/lib/security', () => ({
  security: {
    csrf: { getToken: jest.fn().mockResolvedValue(undefined), clearToken: jest.fn() },
    network: {
      getSecureHeaders: jest.fn().mockReturnValue({ 'Content-Type': 'application/json' }),
    },
  },
}));

const fetchMock = jest.fn() as jest.Mock;
(global as unknown as { fetch: jest.Mock }).fetch = fetchMock;

const consoleSpy = {
  error: jest.spyOn(console, 'error').mockImplementation(() => {}),
  warn: jest.spyOn(console, 'warn').mockImplementation(() => {}),
  debug: jest.spyOn(console, 'debug').mockImplementation(() => {}),
};

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    statusText: status === 200 ? 'OK' : 'Error',
    json: async () => body,
    headers: new Headers(),
  } as unknown as Response;
}

function mockSuccess<T>(data: T): void {
  fetchMock.mockResolvedValueOnce(jsonResponse({ code: 0, message: 'ok', data }));
}

describe('ticketService', () => {
  beforeEach(() => {
    fetchMock.mockReset();
    document.cookie = 'access_token=; path=/; max-age=-1;';
  });

  afterAll(() => {
    Object.values(consoleSpy).forEach((spy) => spy.mockRestore());
  });

  describe('CRUD', () => {
    it('getTickets calls GET /api/v1/tickets with query params', async () => {
      mockSuccess({ data: [], total: 0, page: 1, pageSize: 20, totalPages: 0 });
      await ticketService.getTickets({ page: 2, pageSize: 10, status: 'open' });

      const [url, init] = fetchMock.mock.calls[0];
      expect(url).toContain('/api/v1/tickets');
      expect(url).toContain('page=2');
      expect(url).toContain('pageSize=10');
      expect(url).toContain('status=open');
      expect(init.method).toBe('GET');
    });

    it('getTicket calls GET /api/v1/tickets/:id', async () => {
      mockSuccess({ id: 42 });
      await ticketService.getTicket(42);

      const [url, init] = fetchMock.mock.calls[0];
      expect(url).toContain('/api/v1/tickets/42');
      expect(init.method).toBe('GET');
    });

    it('createTicket calls POST /api/v1/tickets with the body', async () => {
      mockSuccess({ id: 1, ticketNumber: 'TKT-001' });
      await ticketService.createTicket({
        title: 'New',
        priority: 'medium' as never,
        requesterId: 1,
      });

      const [url, init] = fetchMock.mock.calls[0];
      expect(url).toContain('/api/v1/tickets');
      // basePath should not have a trailing slash, so URL must end with /tickets (no // )
      expect(url.endsWith('/api/v1/tickets')).toBe(true);
      expect(init.method).toBe('POST');
      const body = JSON.parse(init.body as string);
      expect(body.title).toBe('New');
      expect(body.priority).toBe('medium');
    });

    it('updateTicket calls PUT /api/v1/tickets/:id with the body', async () => {
      mockSuccess({ id: 1 });
      await ticketService.updateTicket(1, { title: 'Updated' });

      const [url, init] = fetchMock.mock.calls[0];
      expect(url).toContain('/api/v1/tickets/1');
      expect(init.method).toBe('PUT');
      expect(JSON.parse(init.body as string).title).toBe('Updated');
    });

    it('deleteTicket calls DELETE /api/v1/tickets/:id', async () => {
      mockSuccess(null);
      await ticketService.deleteTicket(7);

      const [url, init] = fetchMock.mock.calls[0];
      expect(url).toContain('/api/v1/tickets/7');
      expect(init.method).toBe('DELETE');
    });
  });

  describe('workflow actions', () => {
    it('updateStatus calls PUT /api/v1/tickets/:id/status with { status }', async () => {
      mockSuccess({ id: 1 });
      await ticketService.updateStatus(1, 'in_progress' as never);

      const [url, init] = fetchMock.mock.calls[0];
      expect(url).toContain('/api/v1/tickets/1/status');
      expect(init.method).toBe('PUT');
      expect(JSON.parse(init.body as string).status).toBe('in_progress');
    });

    it('assign calls POST /api/v1/tickets/:id/assign with assigneeId + optional comment', async () => {
      mockSuccess({ id: 1 });
      await ticketService.assign(1, 99, 'Urgent');

      const [url, init] = fetchMock.mock.calls[0];
      expect(url).toContain('/api/v1/tickets/1/assign');
      expect(init.method).toBe('POST');
      const body = JSON.parse(init.body as string);
      expect(body.assigneeId).toBe(99);
      expect(body.comment).toBe('Urgent');
    });

    it('assign omits comment when not provided', async () => {
      mockSuccess({ id: 1 });
      await ticketService.assign(1, 99);

      const [, init] = fetchMock.mock.calls[0];
      const body = JSON.parse(init.body as string);
      expect(body.assigneeId).toBe(99);
      expect(body.comment).toBeUndefined();
    });

    it('resolve calls POST /api/v1/tickets/:id/resolve with resolution + code', async () => {
      mockSuccess({ id: 1 });
      await ticketService.resolve(1, 'Fixed by rebooting', 'CODE-001');

      const [url, init] = fetchMock.mock.calls[0];
      expect(url).toContain('/api/v1/tickets/1/resolve');
      const body = JSON.parse(init.body as string);
      expect(body.resolution).toBe('Fixed by rebooting');
      expect(body.resolutionCode).toBe('CODE-001');
    });

    it('close calls POST /api/v1/tickets/:id/close', async () => {
      mockSuccess({ id: 1 });
      await ticketService.close(1, 'Looks good');
      const [url, init] = fetchMock.mock.calls[0];
      expect(url).toContain('/api/v1/tickets/1/close');
      expect(init.method).toBe('POST');
    });

    it('reopen calls POST /api/v1/tickets/workflow/reopen (not /:id/reopen)', async () => {
      // Implementation uses /workflow/reopen with ticketId in the body
      mockSuccess({ message: 'reopened' });
      await ticketService.reopen(7, 'still broken');

      const [url, init] = fetchMock.mock.calls[0];
      expect(url).toContain('/api/v1/tickets/workflow/reopen');
      expect(init.method).toBe('POST');
      const body = JSON.parse(init.body as string);
      expect(body.ticketId).toBe(7);
      expect(body.reason).toBe('still broken');
    });

    it('escalate calls POST /api/v1/tickets/:id/escalate with level/reason/assigneeId', async () => {
      mockSuccess({ id: 1 });
      await ticketService.escalate(1, { level: 'L2', reason: 'too complex', assigneeId: 9 });

      const [url, init] = fetchMock.mock.calls[0];
      expect(url).toContain('/api/v1/tickets/1/escalate');
      const body = JSON.parse(init.body as string);
      expect(body).toEqual({ level: 'L2', reason: 'too complex', assigneeId: 9 });
    });
  });

  describe('approval & flow actions', () => {
    it('approve calls POST /api/v1/tickets/workflow/approve with ticketId + action', async () => {
      mockSuccess({ success: true, message: 'approved' });
      await ticketService.approve(1, { action: 'approve', comment: 'lgtm' });

      const [url, init] = fetchMock.mock.calls[0];
      expect(url).toContain('/api/v1/tickets/workflow/approve');
      const body = JSON.parse(init.body as string);
      expect(body.ticketId).toBe(1);
      expect(body.action).toBe('approve');
      expect(body.comment).toBe('lgtm');
    });

    it('reject calls POST /api/v1/tickets/workflow/reject', async () => {
      mockSuccess({ message: 'rejected' });
      await ticketService.reject(1, 'invalid');

      const [url] = fetchMock.mock.calls[0];
      expect(url).toContain('/api/v1/tickets/workflow/reject');
    });

    it('accept calls POST /api/v1/tickets/workflow/accept', async () => {
      mockSuccess({ message: 'accepted' });
      await ticketService.accept(1);

      const [url] = fetchMock.mock.calls[0];
      expect(url).toContain('/api/v1/tickets/workflow/accept');
    });

    it('forward calls POST /api/v1/tickets/workflow/forward with toUserId + comment', async () => {
      mockSuccess({ message: 'forwarded' });
      await ticketService.forward(1, 99, 'please handle');

      const [url, init] = fetchMock.mock.calls[0];
      expect(url).toContain('/api/v1/tickets/workflow/forward');
      const body = JSON.parse(init.body as string);
      expect(body.toUserId).toBe(99);
      expect(body.comment).toBe('please handle');
    });
  });

  describe('comments', () => {
    it('getComments calls GET /api/v1/tickets/:id/comments', async () => {
      mockSuccess({ comments: [], total: 0 });
      await ticketService.getComments(1);

      const [url] = fetchMock.mock.calls[0];
      expect(url).toContain('/api/v1/tickets/1/comments');
    });

    it('addComment calls POST /api/v1/tickets/:id/comments with body', async () => {
      mockSuccess({ id: 1 });
      await ticketService.addComment(1, { content: 'Hi', isInternal: true, mentions: [2] });

      const [url, init] = fetchMock.mock.calls[0];
      expect(url).toContain('/api/v1/tickets/1/comments');
      expect(init.method).toBe('POST');
      const body = JSON.parse(init.body as string);
      expect(body.content).toBe('Hi');
      expect(body.isInternal).toBe(true);
      expect(body.mentions).toEqual([2]);
    });

    it('updateComment calls PUT /api/v1/tickets/:id/comments/:commentId', async () => {
      mockSuccess({ id: 1 });
      await ticketService.updateComment(1, 99, { content: 'Edited' });

      const [url, init] = fetchMock.mock.calls[0];
      expect(url).toContain('/api/v1/tickets/1/comments/99');
      expect(init.method).toBe('PUT');
    });

    it('deleteComment calls DELETE /api/v1/tickets/:id/comments/:commentId', async () => {
      mockSuccess(null);
      await ticketService.deleteComment(1, 99);

      const [url, init] = fetchMock.mock.calls[0];
      expect(url).toContain('/api/v1/tickets/1/comments/99');
      expect(init.method).toBe('DELETE');
    });
  });

  describe('queries & stats', () => {
    it('searchTickets calls GET /api/v1/tickets/search?q=...', async () => {
      mockSuccess([]);
      await ticketService.searchTickets('login');

      const [url] = fetchMock.mock.calls[0];
      expect(url).toContain('/api/v1/tickets/search');
      expect(url).toContain('q=login');
    });

    it('getStats calls GET /api/v1/tickets/stats', async () => {
      mockSuccess({
        total: 100,
        open: 30,
        inProgress: 20,
        resolved: 50,
        highPriority: 10,
        overdue: 5,
      });
      const stats = await ticketService.getStats();

      const [url] = fetchMock.mock.calls[0];
      expect(url).toContain('/api/v1/tickets/stats');
      expect(stats.total).toBe(100);
      expect(stats.overdue).toBe(5);
    });
  });

  describe('error propagation', () => {
    it('propagates backend error message on non-zero code', async () => {
      fetchMock.mockResolvedValueOnce(
        jsonResponse({ code: 1001, message: '标题不能为空', data: null })
      );

      await expect(
        ticketService.createTicket({
          title: '',
          priority: 'medium' as never,
          requesterId: 1,
        })
      ).rejects.toThrow('标题不能为空');
    });

    it('propagates HTTP error on 500', async () => {
      fetchMock.mockResolvedValueOnce(jsonResponse({}, 500));
      await expect(ticketService.getTicket(1)).rejects.toThrow(/status: 500/);
    });
  });
});