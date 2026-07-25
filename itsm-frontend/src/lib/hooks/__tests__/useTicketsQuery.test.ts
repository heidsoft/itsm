import { renderHook, waitFor } from '@testing-library/react';
import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  useTicketsQuery,
  useTicketStatsQuery,
  useTicketDetailQuery,
  useCreateTicketMutation,
  useUpdateTicketMutation,
  useDeleteTicketMutation,
  useBatchDeleteTicketsMutation,
  usePrefetchTicketDetail,
  useRefreshTickets,
  ticketKeys,
} from '../useTicketsQuery';

// Mock antd
jest.mock('antd', () => ({
  message: {
    success: jest.fn(),
    error: jest.fn(),
  },
}));

// Mock ticket service
jest.mock('@/lib/services/ticket-service', () => ({
  ticketService: {
    listTickets: jest.fn(),
    getTicketStats: jest.fn(),
    getTicket: jest.fn(),
    createTicket: jest.fn(),
    updateTicket: jest.fn(),
    deleteTicket: jest.fn(),
  },
}));

import { ticketService } from '@/lib/services/ticket-service';
const mockService = ticketService as jest.Mocked<typeof ticketService>;

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
};

describe('useTicketsQuery hooks', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    (console.error as jest.Mock).mockRestore();
  });

  describe('ticketKeys', () => {
    it('should generate correct query keys', () => {
      expect(ticketKeys.all).toEqual(['tickets']);
      expect(ticketKeys.lists()).toEqual(['tickets', 'list']);
      expect(ticketKeys.details()).toEqual(['tickets', 'detail']);
      expect(ticketKeys.detail(1)).toEqual(['tickets', 'detail', 1]);
      expect(ticketKeys.stats()).toEqual(['tickets', 'stats']);
    });

    it('should generate list key with filters and pagination', () => {
      const filters = { status: 'open' as any };
      const pagination = { current: 1, pageSize: 20, total: 0 };
      expect(ticketKeys.list(filters, pagination)).toEqual([
        'tickets', 'list', filters, pagination,
      ]);
    });
  });

  describe('useTicketsQuery', () => {
    it('should fetch tickets with default params', async () => {
      mockService.listTickets.mockResolvedValue({
        tickets: [{ id: 1, title: 'Test' }],
        total: 1,
        page: 1,
        pageSize: 20,
      } as any);

      const { result } = renderHook(() => useTicketsQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data?.tickets).toEqual([{ id: 1, title: 'Test' }]);
      expect(result.current.data?.total).toBe(1);
    });

    it('should pass filters to service', async () => {
      mockService.listTickets.mockResolvedValue({
        tickets: [],
        total: 0,
        page: 1,
        pageSize: 20,
      } as any);

      const filters = { status: 'open' as any };
      const pagination = { current: 2, pageSize: 10, total: 50 };

      const { result } = renderHook(() => useTicketsQuery(filters, pagination), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockService.listTickets).toHaveBeenCalledWith({
        page: 2,
        pageSize: 10,
        status: 'open',
      });
    });

    it('should handle empty response gracefully', async () => {
      mockService.listTickets.mockResolvedValue({
        tickets: null,
        total: 0,
      } as any);

      const { result } = renderHook(() => useTicketsQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data?.tickets).toEqual([]);
      expect(result.current.data?.total).toBe(0);
    });

    it('should handle fetch error', async () => {
      mockService.listTickets.mockRejectedValue(new Error('Network error'));

      const { result } = renderHook(() => useTicketsQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isError).toBe(true));
    });

    it('should calculate totalPages correctly', async () => {
      mockService.listTickets.mockResolvedValue({
        tickets: [],
        total: 45,
        page: 1,
        pageSize: 10,
        size: 10,
      } as any);

      const { result } = renderHook(
        () => useTicketsQuery({}, { current: 1, pageSize: 10, total: 0 }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data?.totalPages).toBe(5);
    });
  });

  describe('useTicketStatsQuery', () => {
    it('should fetch ticket stats', async () => {
      mockService.getTicketStats.mockResolvedValue({
        total: 100,
        open: 30,
        resolved: 60,
        highPriority: 10,
      } as any);

      const { result } = renderHook(() => useTicketStatsQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data).toEqual({
        total: 100,
        open: 30,
        resolved: 60,
        highPriority: 10,
      });
    });

    it('should return defaults when stats call fails', async () => {
      mockService.getTicketStats.mockRejectedValue(new Error('Stats failed'));

      const { result } = renderHook(() => useTicketStatsQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data).toEqual({
        total: 0,
        open: 0,
        resolved: 0,
        highPriority: 0,
      });
    });

    it('should handle null response fields', async () => {
      mockService.getTicketStats.mockResolvedValue({
        total: null,
        open: null,
        resolved: null,
        highPriority: null,
      } as any);

      const { result } = renderHook(() => useTicketStatsQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(result.current.data).toEqual({
        total: 0,
        open: 0,
        resolved: 0,
        highPriority: 0,
      });
    });
  });

  describe('useTicketDetailQuery', () => {
    it('should fetch ticket detail', async () => {
      mockService.getTicket.mockResolvedValue({ id: 1, title: 'Test' } as any);

      const { result } = renderHook(() => useTicketDetailQuery(1), {
        wrapper: createWrapper(),
      });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockService.getTicket).toHaveBeenCalledWith(1);
    });

    it('should not fetch when id is 0', () => {
      renderHook(() => useTicketDetailQuery(0), { wrapper: createWrapper() });
      expect(mockService.getTicket).not.toHaveBeenCalled();
    });
  });

  describe('useCreateTicketMutation', () => {
    it('should create a ticket', async () => {
      mockService.createTicket.mockResolvedValue({ id: 1 } as any);

      const { result } = renderHook(() => useCreateTicketMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ title: 'New Ticket' } as any);

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockService.createTicket).toHaveBeenCalledWith({ title: 'New Ticket' });
    });

    it('should handle create error', async () => {
      mockService.createTicket.mockRejectedValue(new Error('Create failed'));

      const { result } = renderHook(() => useCreateTicketMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ title: 'Bad' } as any);

      await waitFor(() => expect(result.current.isError).toBe(true));
    });
  });

  describe('useUpdateTicketMutation', () => {
    it('should update a ticket', async () => {
      mockService.updateTicket.mockResolvedValue({ id: 1, title: 'Updated' } as any);

      const { result } = renderHook(() => useUpdateTicketMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ id: 1, data: { title: 'Updated' } as any });

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockService.updateTicket).toHaveBeenCalledWith(1, { title: 'Updated' });
    });

    it('should handle update error', async () => {
      mockService.updateTicket.mockRejectedValue(new Error('Update failed'));

      const { result } = renderHook(() => useUpdateTicketMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ id: 1, data: {} as any });

      await waitFor(() => expect(result.current.isError).toBe(true));
    });
  });

  describe('useDeleteTicketMutation', () => {
    it('should delete a ticket', async () => {
      mockService.deleteTicket.mockResolvedValue(undefined as any);

      const { result } = renderHook(() => useDeleteTicketMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate(1);

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockService.deleteTicket).toHaveBeenCalledWith(1);
    });

    it('should handle delete error', async () => {
      mockService.deleteTicket.mockRejectedValue(new Error('Delete failed'));

      const { result } = renderHook(() => useDeleteTicketMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate(1);

      await waitFor(() => expect(result.current.isError).toBe(true));
    });
  });

  describe('useBatchDeleteTicketsMutation', () => {
    it('should batch delete tickets', async () => {
      mockService.deleteTicket.mockResolvedValue(undefined as any);

      const { result } = renderHook(() => useBatchDeleteTicketsMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate([1, 2, 3]);

      await waitFor(() => expect(result.current.isSuccess).toBe(true));
      expect(mockService.deleteTicket).toHaveBeenCalledTimes(3);
    });

    it('should handle batch delete error', async () => {
      mockService.deleteTicket.mockRejectedValue(new Error('Delete failed'));

      const { result } = renderHook(() => useBatchDeleteTicketsMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate([1, 2]);

      await waitFor(() => expect(result.current.isError).toBe(true));
    });
  });

  describe('usePrefetchTicketDetail', () => {
    it('should return a prefetch function', () => {
      const { result } = renderHook(() => usePrefetchTicketDetail(), {
        wrapper: createWrapper(),
      });

      expect(typeof result.current).toBe('function');
    });

    it('should call prefetchQuery when invoked', () => {
      mockService.getTicket.mockResolvedValue({ id: 1 } as any);

      const { result } = renderHook(() => usePrefetchTicketDetail(), {
        wrapper: createWrapper(),
      });

      result.current(1);
      // prefetchQuery is fire-and-forget, just verify no throw
    });
  });

  describe('useRefreshTickets', () => {
    it('should return a refresh function', () => {
      const { result } = renderHook(() => useRefreshTickets(), {
        wrapper: createWrapper(),
      });

      expect(typeof result.current).toBe('function');
    });

    it('should invalidate queries when called', () => {
      const { result } = renderHook(() => useRefreshTickets(), {
        wrapper: createWrapper(),
      });

      // Should not throw
      result.current();
    });
  });
});
