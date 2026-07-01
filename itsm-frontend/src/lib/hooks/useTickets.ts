'use client';

import { useState, useCallback, useEffect, useRef } from 'react';
import { message } from 'antd';
import {
  ticketService,
  Ticket,
  TicketStatus,
  TicketPriority,
  TicketType,
} from '../../lib/services/ticket-service';

export interface TicketQueryFilters {
  status?: TicketStatus;
  priority?: TicketPriority;
  type?: TicketType;
  category?: string;
  assignee_id?: number;
  keyword?: string;
  dateRange?: [string, string];
  tags?: string[];
  source?: string;
  impact?: string;
  urgency?: string;
}

export interface TicketStats {
  total: number;
  open: number;
  resolved: number;
  highPriority: number;
}

export interface UseTicketsReturn {
  // Data
  tickets: Ticket[];
  stats: TicketStats;
  loading: boolean;
  error: string | null;

  // Pagination
  pagination: {
    current: number;
    pageSize: number;
    total: number;
  };

  // Filters
  filters: Partial<TicketQueryFilters>;

  // Actions
  fetchTickets: (customFilters?: Partial<TicketQueryFilters>) => Promise<void>;
  fetchStats: () => Promise<void>;
  refreshData: () => Promise<void>;
  updateFilters: (newFilters: Partial<TicketQueryFilters>) => void;
  updatePagination: (page: number, pageSize: number) => void;

  // Ticket operations
  createTicket: (ticketData: unknown) => Promise<void>;
  updateTicket: (id: number, ticketData: unknown) => Promise<void>;
  deleteTicket: (id: number) => Promise<void>;
  batchDeleteTickets: (ids: number[]) => Promise<void>;
}

export const useTickets = (): UseTicketsReturn => {
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [stats, setStats] = useState<TicketStats>({
    total: 0,
    open: 0,
    resolved: 0,
    highPriority: 0,
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 20,
    total: 0,
  });
  const [filters, setFilters] = useState<Partial<TicketQueryFilters>>({});

  // 用 ref 持有最新值，避免 fetchTickets 因依赖变化无限重建
  const filtersRef = useRef(filters);
  const paginationRef = useRef(pagination);
  // 用于触发 fetchTickets 的计数器
  const [fetchTrigger, setFetchTrigger] = useState(0);

  useEffect(() => {
    filtersRef.current = filters;
  }, [filters]);

  useEffect(() => {
    paginationRef.current = pagination;
  }, [pagination]);

  // fetchTickets 不再依赖 filters/pagination state，通过 ref 读取最新值
  const fetchTickets = useCallback(async (customFilters?: Partial<TicketQueryFilters>) => {
    setLoading(true);
    setError(null);

    try {
      const currentFilters = customFilters ?? filtersRef.current;
      const { current: page, pageSize } = paginationRef.current;
      const dateRange = currentFilters.dateRange;

      const response = await ticketService.listTickets({
        page,
        pageSize,
        status: currentFilters.status,
        priority: currentFilters.priority,
        type: currentFilters.type,
        category: currentFilters.category,
        assignee_id: currentFilters.assignee_id,
        keyword: currentFilters.keyword,
        date_from: dateRange?.[0],
        date_to: dateRange?.[1],
        tags: currentFilters.tags,
      });

      setTickets(response.tickets ?? []);
      setPagination(prev => ({ ...prev, total: response.total ?? 0 }));
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to load tickets';
      setError(errorMessage);
      message.error(errorMessage);
    } finally {
      setLoading(false);
    }
  }, []); // 空依赖，永不重建

  // Fetch statistics
  const fetchStats = useCallback(async () => {
    try {
      const response = await ticketService.getTicketStats();
      setStats({
        total: response.total,
        open: response.open,
        resolved: response.resolved,
        highPriority: response.highPriority,
      });
    } catch (err) {
      console.error('Failed to load statistics:', err);
    }
  }, []);

  const refreshData = useCallback(async () => {
    await Promise.all([fetchTickets(), fetchStats()]);
  }, [fetchTickets, fetchStats]);

  // updateFilters：更新 filters state 并触发重新拉取
  const updateFilters = useCallback((newFilters: Partial<TicketQueryFilters>) => {
    setFilters(prev => {
      const next = { ...prev, ...newFilters };
      filtersRef.current = next;
      return next;
    });
    setPagination(prev => {
      const next = { ...prev, current: 1 };
      paginationRef.current = next;
      return next;
    });
    // 触发 fetchTickets
    setFetchTrigger(n => n + 1);
  }, []);

  // updatePagination：更新分页并触发重新拉取
  const updatePagination = useCallback((page: number, pageSize: number) => {
    setPagination(prev => {
      const next = { ...prev, current: page, pageSize };
      paginationRef.current = next;
      return next;
    });
    setFetchTrigger(n => n + 1);
  }, []);

  // Create ticket
  const createTicket = useCallback(
    async (ticketData: any) => {
      try {
        await ticketService.createTicket(ticketData);
        message.success('Ticket created successfully');
        await refreshData();
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Failed to create ticket';
        message.error(errorMessage);
        throw err;
      }
    },
    [refreshData]
  );

  // Update ticket
  const updateTicket = useCallback(
    async (id: number, ticketData: any) => {
      try {
        await ticketService.updateTicket(id, ticketData);
        message.success('Ticket updated successfully');
        await refreshData();
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Failed to update ticket';
        message.error(errorMessage);
        throw err;
      }
    },
    [refreshData]
  );

  // Delete ticket
  const deleteTicket = useCallback(
    async (id: number) => {
      try {
        await ticketService.deleteTicket(id);
        message.success('Ticket deleted successfully');
        await refreshData();
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Failed to delete ticket';
        message.error(errorMessage);
        throw err;
      }
    },
    [refreshData]
  );

  // Batch delete tickets
  const batchDeleteTickets = useCallback(
    async (ids: number[]) => {
      try {
        await Promise.all(ids.map(id => ticketService.deleteTicket(id)));
        message.success(`${ids.length} tickets deleted successfully`);
        await refreshData();
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Failed to delete tickets';
        message.error(errorMessage);
        throw err;
      }
    },
    [refreshData]
  );

  // 初始加载 + fetchTrigger 变化时重新拉取
  useEffect(() => {
    fetchTickets();
  }, [fetchTrigger]); // eslint-disable-line react-hooks/exhaustive-deps

  // 初始加载统计
  useEffect(() => {
    fetchStats();
  }, [fetchStats]);

  return {
    tickets,
    stats,
    loading,
    error,
    pagination,
    filters,
    fetchTickets,
    fetchStats,
    refreshData,
    updateFilters,
    updatePagination,
    createTicket,
    updateTicket,
    deleteTicket,
    batchDeleteTickets,
  };
};
