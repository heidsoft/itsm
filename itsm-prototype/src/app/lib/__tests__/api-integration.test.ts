import { TicketApi } from '@/lib/api/ticket-api';
import { DashboardAPI } from '@/lib/api/dashboard-api';

// Mock fetch globally
global.fetch = jest.fn();

// Mock console methods to avoid noise in tests
const consoleSpy = {
  error: jest.spyOn(console, 'error').mockImplementation(() => {}),
  warn: jest.spyOn(console, 'warn').mockImplementation(() => {}),
  log: jest.spyOn(console, 'log').mockImplementation(() => {}),
};

describe('API Integration Tests', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    // Reset fetch mock
    (fetch as jest.Mock).mockClear();
  });

  afterAll(() => {
    // Restore console methods
    Object.values(consoleSpy).forEach(spy => spy.mockRestore());
  });

  describe('TicketApi', () => {
    describe('getTickets', () => {
      it('should fetch tickets successfully', async () => {
        const mockResponse = {
          code: 0,
          message: 'success',
          data: {
            tickets: [
              {
                id: 1,
                title: '系统登录问题',
                description: '用户无法正常登录',
                status: 'open',
                priority: 'high',
                created_at: '2024-01-01T10:00:00Z',
              },
            ],
            total: 1,
            page: 1,
            page_size: 20,
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
            headers: expect.objectContaining({
              'Content-Type': 'application/json',
            }),
          })
        );

        expect(result).toEqual(mockResponse.data);
      });

      it('should handle API error responses', async () => {
        const mockErrorResponse = {
          code: 5001,
          message: '服务器内部错误',
          data: null,
        };

        (fetch as jest.Mock).mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: async () => mockErrorResponse,
        });

        await expect(TicketApi.getTickets()).rejects.toThrow('服务器内部错误');
      });

      it('should handle network errors', async () => {
        (fetch as jest.Mock).mockRejectedValueOnce(new Error('Network error'));

        await expect(TicketApi.getTickets()).rejects.toThrow('Network error');
      });

      it('should include query parameters correctly', async () => {
        const mockResponse = {
          code: 0,
          message: 'success',
          data: { tickets: [], total: 0, page: 1, page_size: 10 },
        };

        (fetch as jest.Mock).mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: async () => mockResponse,
        });

        const params = {
          page: 2,
          page_size: 10,
          status: 'open',
          priority: 'high',
        };

        await TicketApi.getTickets(params);

        expect(fetch).toHaveBeenCalledWith(
          expect.stringContaining('/api/v1/tickets'),
          expect.objectContaining({
            method: 'GET',
          })
        );
      });
    });

    describe('createTicket', () => {
      it('should create ticket successfully', async () => {
        const mockTicketData = {
          title: '新工单',
          description: '工单描述',
          priority: 'medium',
          type: 'incident',
        };

        const mockResponse = {
          code: 0,
          message: 'success',
          data: {
            id: 1,
            ...mockTicketData,
            status: 'open',
            created_at: '2024-01-01T10:00:00Z',
          },
        };

        (fetch as jest.Mock).mockResolvedValueOnce({
          ok: true,
          status: 201,
          json: async () => mockResponse,
        });

        const result = await TicketApi.createTicket(mockTicketData);

        expect(fetch).toHaveBeenCalledWith(
          expect.stringContaining('/api/v1/tickets'),
          expect.objectContaining({
            method: 'POST',
            headers: expect.objectContaining({
              'Content-Type': 'application/json',
            }),
            body: JSON.stringify(mockTicketData),
          })
        );

        expect(result).toEqual(mockResponse.data);
      });

      it('should handle validation errors', async () => {
        const mockErrorResponse = {
          code: 1001,
          message: '请求参数错误: title is required',
          data: null,
        };

        (fetch as jest.Mock).mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: async () => mockErrorResponse,
        });

        const invalidData = { 
          description: '缺少标题',
          priority: 'medium' as const,
          type: 'incident' as const
        };

        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        await expect(TicketApi.createTicket(invalidData as any)).rejects.toThrow(
          '请求参数错误: title is required'
        );
      });
    });
  });

  describe('DashboardAPI', () => {
    describe('getDashboardConfig', () => {
      it('should fetch dashboard config successfully', async () => {
        const mockResponse = {
          code: 0,
          message: 'success',
          data: {
            id: 1,
            name: '默认仪表盘',
            layout: 'grid',
            widgets: [
              {
                id: 'widget-1',
                type: 'stats',
                title: '工单统计',
                position: { x: 0, y: 0, w: 6, h: 4 },
              },
            ],
          },
        };

        (fetch as jest.Mock).mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: async () => mockResponse,
        });

        const result = await DashboardAPI.getDashboardConfig();

        expect(fetch).toHaveBeenCalledWith(
          expect.stringContaining('/api/v1/dashboard/config'),
          expect.objectContaining({
            method: 'GET',
            headers: expect.objectContaining({
              'Content-Type': 'application/json',
            }),
          })
        );

        expect(result).toEqual(mockResponse.data);
      });
    });

    describe('getTicketStats', () => {
      it('should fetch ticket statistics successfully', async () => {
        const mockResponse = {
          code: 0,
          message: 'success',
          data: {
            total: 150,
            open: 45,
            in_progress: 10,
            resolved: 95,
            by_priority: {
              low: 50,
              medium: 70,
              high: 25,
              critical: 5,
            },
          },
        };

        (fetch as jest.Mock).mockResolvedValueOnce({
          ok: true,
          status: 200,
          json: async () => mockResponse,
        });

        const result = await DashboardAPI.getTicketStats();

        expect(fetch).toHaveBeenCalledWith(
          expect.stringContaining('/api/v1/dashboard/stats/tickets'),
          expect.objectContaining({
            method: 'GET',
          })
        );

        expect(result).toEqual(mockResponse.data);
      });
    });
  });

  describe('Error Handling', () => {
    it('should handle HTTP error status codes', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
        json: async () => ({ message: 'Server error' }),
      });

      await expect(TicketApi.getTickets()).rejects.toThrow();
    });

    it('should handle malformed JSON responses', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => {
          throw new Error('Invalid JSON');
        },
      });

      await expect(TicketApi.getTickets()).rejects.toThrow('Invalid JSON');
    });

    it('should handle timeout errors', async () => {
      (fetch as jest.Mock).mockImplementationOnce(
        () => new Promise((_, reject) => 
          setTimeout(() => reject(new Error('Request timeout')), 100)
        )
      );

      await expect(TicketApi.getTickets()).rejects.toThrow('Request timeout');
    });

    it('should handle CORS errors', async () => {
      (fetch as jest.Mock).mockRejectedValueOnce(
        new TypeError('Failed to fetch')
      );

      await expect(TicketApi.getTickets()).rejects.toThrow('无法连接到服务器');
    });
  });

  describe('Authentication Integration', () => {
    it('should include authorization header when token exists', async () => {
      // Mock localStorage
      const mockToken = 'mock-jwt-token';
      Object.defineProperty(window, 'localStorage', {
        value: {
          getItem: jest.fn(() => mockToken),
          setItem: jest.fn(),
          removeItem: jest.fn(),
        },
        writable: true,
      });

      const mockResponse = {
        code: 0,
        message: 'success',
        data: { tickets: [] },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      await TicketApi.getTickets();

      expect(fetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          headers: expect.objectContaining({
            'Authorization': `Bearer ${mockToken}`,
          }),
        })
      );
    });

    it('should handle token expiration', async () => {
      const mockErrorResponse = {
        code: 2001,
        message: 'Token已过期',
        data: null,
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockErrorResponse,
      });

      await expect(TicketApi.getTickets()).rejects.toThrow('Token已过期');
    });

    it('should handle unauthorized access', async () => {
      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: false,
        status: 401,
        statusText: 'Unauthorized',
        json: async () => ({ message: 'Unauthorized' }),
      });

      await expect(TicketApi.getTickets()).rejects.toThrow();
    });
  });

  describe('Performance and Caching', () => {
    it('should handle concurrent requests', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: { tickets: [] },
      };

      (fetch as jest.Mock).mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      // Make multiple concurrent requests
      const promises = [
        TicketApi.getTickets(),
        TicketApi.getTickets(),
        TicketApi.getTickets(),
      ];

      const results = await Promise.all(promises);

      expect(results).toHaveLength(3);
      expect(fetch).toHaveBeenCalledTimes(3);
    });

    it('should handle large response payloads', async () => {
      // Create a large mock response
      const largeTicketList = Array.from({ length: 1000 }, (_, i) => ({
        id: i + 1,
        title: `工单 ${i + 1}`,
        status: 'open',
        priority: 'medium',
      }));

      const mockResponse = {
        code: 0,
        message: 'success',
        data: {
          tickets: largeTicketList,
          total: 1000,
          page: 1,
          page_size: 1000,
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => mockResponse,
      });

      const result = await TicketApi.getTickets({ page_size: 1000 });

      expect(result.tickets).toHaveLength(1000);
      expect(result.total).toBe(1000);
    });
  });

  describe('Data Validation', () => {
    it('should handle missing response fields', async () => {
      const partialResponse = {
        code: 0,
        message: 'success',
        data: {
          tickets: [
            {
              id: 1,
              title: '工单标题',
              // Missing other required fields
            },
          ],
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => partialResponse,
      });

      const result = await TicketApi.getTickets();
      
      expect(result.tickets).toHaveLength(1);
      expect(result.tickets[0].id).toBe(1);
    });

    it('should validate response data structure', async () => {
      const invalidResponse = {
        code: 0,
        message: 'success',
        data: {
          // Missing required fields
          tickets: null,
        },
      };

      (fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: async () => invalidResponse,
      });

      // The API should handle invalid response structures gracefully
      const result = await TicketApi.getTickets();
      
      // Should provide fallback values or throw descriptive error
      expect(result).toBeDefined();
    });
  });
});
