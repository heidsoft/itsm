import { TicketApi } from '@/lib/api/ticket-api';

// Mock fetch globally
global.fetch = jest.fn();

// Mock console methods to avoid noise in tests
const consoleSpy = {
  error: jest.spyOn(console, 'error').mockImplementation(() => {}),
  warn: jest.spyOn(console, 'warn').mockImplementation(() => {}),
  log: jest.spyOn(console, 'log').mockImplementation(() => {}),
};

describe('TicketApi', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (fetch as jest.Mock).mockClear();
  });

  afterAll(() => {
    Object.values(consoleSpy).forEach(spy => spy.mockRestore());
  });

  describe('getTickets', () => {
    it('should fetch tickets successfully', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          tickets: [
            {
              id: 1,
              title: 'Test Ticket',
              description: 'Test description',
              status: 'open',
              priority: 'high',
              category: 'incident',
            },
          ],
          total: 1,
          page: 1,
          page_size: 20,
          total_pages: 1,
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      const result = await TicketApi.getTickets();

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/tickets'),
        expect.objectContaining({
          method: 'GET',
        })
      );

      expect(result.tickets).toHaveLength(1);
      expect(result.tickets[0].title).toBe('Test Ticket');
    });

    it('should handle empty ticket list', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          tickets: [],
          total: 0,
          page: 1,
          page_size: 20,
          total_pages: 0,
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      const result = await TicketApi.getTickets();

      expect(result.tickets).toHaveLength(0);
      expect(result.total).toBe(0);
    });

    it('should pass query parameters correctly', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          items: [],
          total: 0,
          page: 1,
          page_size: 10,
          total_pages: 0,
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      await TicketApi.getTickets({ page: 2, pageSize: 10, status: 'open' });

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('page=2'),
        expect.any(Object)
      );
    });

    it('should handle API error response', async () => {
      const mockResponse = {
        code: 5001,
        message: 'Internal server error',
        data: null,
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      // The API client throws an error for non-zero codes
      await expect(TicketApi.getTickets({})).rejects.toThrow('Internal server error');
    });
  });

  describe('getTicket', () => {
    it('should fetch single ticket successfully', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          id: 1,
          title: 'Test Ticket',
          description: 'Test description',
          status: 'open',
          priority: 'high',
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      const result = await TicketApi.getTicket(1);

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/tickets/1'),
        expect.objectContaining({
          method: 'GET',
        })
      );

      expect(result.id).toBe(1);
      expect(result.title).toBe('Test Ticket');
    });

    it('should handle ticket not found', async () => {
      const mockResponse = {
        code: 4004,
        message: 'Ticket not found',
        data: null,
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      await expect(TicketApi.getTicket(999)).rejects.toBeDefined();
    });
  });

  describe('createTicket', () => {
    it('should create ticket successfully', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          id: 1,
          title: 'New Ticket',
          description: 'New description',
          status: 'open',
          priority: 'medium',
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 201,
        json: async () => mockResponse,
      });

      const result = await TicketApi.createTicket({
        title: 'New Ticket',
        description: 'New description',
        priority: 'medium',
        category: 'incident',
      });

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/tickets'),
        expect.objectContaining({
          method: 'POST',
        })
      );

      expect(result.title).toBe('New Ticket');
    });

    it('should validate required fields', async () => {
      const mockResponse = {
        code: 1001,
        message: 'Title is required',
        data: null,
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      // Missing required fields should be handled by API
      await expect(
        TicketApi.createTicket({
          title: '',
          description: 'test',
          priority: 'medium',
        } as any)
      ).rejects.toBeDefined();
    });
  });

  describe('updateTicket', () => {
    it('should update ticket successfully', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          id: 1,
          title: 'Updated Ticket',
          status: 'in_progress',
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      const result = await TicketApi.updateTicket(1, {
        title: 'Updated Ticket',
        status: 'in_progress',
      });

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/tickets/1'),
        expect.objectContaining({
          method: 'PUT',
        })
      );

      expect(result.title).toBe('Updated Ticket');
    });
  });

  describe('deleteTicket', () => {
    it('should delete ticket successfully', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: null,
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      await TicketApi.deleteTicket(1);

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/tickets/1'),
        expect.objectContaining({
          method: 'DELETE',
        })
      );
    });
  });

  describe('updateTicketStatus', () => {
    it('should update ticket status successfully', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          id: 1,
          status: 'closed',
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      const result = await TicketApi.updateTicketStatus(1, 'closed');

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/tickets/1/status'),
        expect.objectContaining({
          method: 'PUT',
        })
      );

      expect(result.status).toBe('closed');
    });
  });

  describe('assignTicket', () => {
    it('should assign ticket to user', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          id: 1,
          assignee_id: 42,
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      const result = await TicketApi.assignTicket(1, 42);

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/tickets/1/assign'),
        expect.objectContaining({
          method: 'POST',
        })
      );

      expect(result.assigneeId).toBe(42);
    });
  });

  describe('addComment', () => {
    it('should add comment to ticket', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          id: 1,
          ticket_id: 1,
          content: 'Test comment',
          created_by: 1,
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 201,
        json: async () => mockResponse,
      });

      const result = await TicketApi.addComment(1, 'Test comment');

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/tickets/1/comment'),
        expect.objectContaining({
          method: 'POST',
        })
      );

      expect(result.content).toBe('Test comment');
    });
  });
});
