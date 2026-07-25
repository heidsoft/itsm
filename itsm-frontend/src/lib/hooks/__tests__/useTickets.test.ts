import { renderHook, act, waitFor } from '@testing-library/react';
import { useTickets } from '../useTickets';

// Mock antd message
jest.mock('antd', () => ({
  message: {
    success: jest.fn(),
    error: jest.fn(),
    warning: jest.fn(),
    info: jest.fn(),
  },
}));

// Mock ticket service
jest.mock('../../../lib/services/ticket-service', () => ({
  ticketService: {
    listTickets: jest.fn(),
    getTicketStats: jest.fn(),
    createTicket: jest.fn(),
    updateTicket: jest.fn(),
    deleteTicket: jest.fn(),
  },
  TicketStatus: { OPEN: 'open', IN_PROGRESS: 'in_progress', RESOLVED: 'resolved', CLOSED: 'closed' },
  TicketPriority: { URGENT: 'urgent', HIGH: 'high', MEDIUM: 'medium', LOW: 'low' },
}));

import { ticketService } from '../../../lib/services/ticket-service';

const mockTicketService = ticketService as jest.Mocked<typeof ticketService>;

describe('useTickets', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockTicketService.listTickets.mockResolvedValue({ tickets: [], total: 0 } as any);
    mockTicketService.getTicketStats.mockResolvedValue({
      total: 10,
      open: 5,
      resolved: 3,
      highPriority: 2,
    } as any);
  });

  it('should return initial state with loading', async () => {
    const { result } = renderHook(() => useTickets());

    // Initially loading should be true due to useEffect triggering fetchTickets
    expect(result.current.tickets).toEqual([]);
    expect(result.current.error).toBeNull();
    expect(result.current.pagination).toEqual({ current: 1, pageSize: 20, total: 0 });
    expect(result.current.filters).toEqual({});
  });

  it('should fetch tickets on mount', async () => {
    mockTicketService.listTickets.mockResolvedValue({
      tickets: [{ id: 1, title: 'Test Ticket' }],
      total: 1,
    } as any);

    const { result } = renderHook(() => useTickets());

    await waitFor(() => {
      expect(result.current.tickets).toEqual([{ id: 1, title: 'Test Ticket' }]);
    });

    expect(mockTicketService.listTickets).toHaveBeenCalled();
  });

  it('should fetch stats on mount', async () => {
    const { result } = renderHook(() => useTickets());

    await waitFor(() => {
      expect(result.current.stats.total).toBe(10);
    });

    expect(result.current.stats).toEqual({
      total: 10,
      open: 5,
      resolved: 3,
      highPriority: 2,
    });
  });

  it('should handle fetchTickets error', async () => {
    mockTicketService.listTickets.mockRejectedValue(new Error('Network error'));

    const { result } = renderHook(() => useTickets());

    await waitFor(() => {
      expect(result.current.error).toBe('Network error');
    });

    expect(result.current.loading).toBe(false);
  });

  it('should update filters and reset pagination', async () => {
    const { result } = renderHook(() => useTickets());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    act(() => {
      result.current.updateFilters({ status: 'open' as any });
    });

    expect(result.current.filters).toEqual({ status: 'open' });
    expect(result.current.pagination.current).toBe(1);
  });

  it('should update pagination', async () => {
    const { result } = renderHook(() => useTickets());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    act(() => {
      result.current.updatePagination(2, 10);
    });

    expect(result.current.pagination.current).toBe(2);
    expect(result.current.pagination.pageSize).toBe(10);
  });

  it('should create ticket and refresh data', async () => {
    mockTicketService.createTicket.mockResolvedValue({ id: 1 } as any);

    const { result } = renderHook(() => useTickets());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    await act(async () => {
      await result.current.createTicket({ title: 'New ticket' });
    });

    expect(mockTicketService.createTicket).toHaveBeenCalledWith({ title: 'New ticket' });
  });

  it('should delete ticket and refresh data', async () => {
    mockTicketService.deleteTicket.mockResolvedValue(undefined as any);

    const { result } = renderHook(() => useTickets());

    await waitFor(() => {
      expect(result.current.loading).toBe(false);
    });

    await act(async () => {
      await result.current.deleteTicket(1);
    });

    expect(mockTicketService.deleteTicket).toHaveBeenCalledWith(1);
  });
});
