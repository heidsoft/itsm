'use client';

import { useState, useCallback, useEffect, useRef } from 'react';
import { message } from 'antd';
import type {
  Ticket,
  TicketStatus,
  TicketPriority,
  TicketType,
} from '../../lib/services/ticket-service';
import { ticketService } from '../../lib/services/ticket-service';

export interface TicketQueryFilters {
  status?: TicketStatus;
  priority?: TicketPriority;
  type?: TicketType;
  category?: string;
  assigneeId?: number;
  keyword?: string;
  dateRange?: [string, string];
  tags?: string[];
  source?: string;
  impact?: string;
  urgency?: string;
}

export interface BatchDeleteResult {
  readonly successCount: number;
  readonly failedIds: number[];
  readonly errors: ReadonlyArray<{ readonly id: number; readonly message: string }>;
}

export interface UseTicketsReturn {
  // Data
  tickets: Ticket[];
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
  refreshData: () => Promise<void>;
  updateFilters: (newFilters: Partial<TicketQueryFilters>) => void;
  updatePagination: (page: number, pageSize: number) => void;

  // Ticket operations
  createTicket: (ticketData: unknown) => Promise<void>;
  updateTicket: (id: number, ticketData: unknown) => Promise<void>;
  deleteTicket: (id: number) => Promise<void>;
  batchDeleteTickets: (ids: number[]) => Promise<BatchDeleteResult>;
}

export const useTickets = (): UseTicketsReturn => {
  const [tickets, setTickets] = useState<Ticket[]>([]);
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
        assigneeId: currentFilters.assigneeId,
        keyword: currentFilters.keyword,
        dateFrom: dateRange?.[0],
        dateTo: dateRange?.[1],
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

  const refreshData = useCallback(async () => {
    // 仅刷新工单列表。统计由上层页面负责，避免重复调用 stats 接口。
    await fetchTickets();
  }, [fetchTickets]);

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
  // H4: switched from Promise.all to Promise.allSettled so a single failure
  // does not collapse the whole operation. Returns per-id detail so the UI
  // can show actionable error info (CLAUDE.md: high-risk actions must be
  // auditable; we surface the failed ids in the caller).
  const batchDeleteTickets = useCallback(
    async (ids: number[]): Promise<BatchDeleteResult> => {
      const errors: { id: number; message: string }[] = [];
      let successCount = 0;
      const results = await Promise.allSettled(
        ids.map(id => ticketService.deleteTicket(id).then(() => id))
      );
      const failedIds: number[] = [];
      results.forEach((r, idx) => {
        if (r.status === 'fulfilled') {
          successCount++;
        } else {
          const id = ids[idx]!;
          failedIds.push(id);
          errors.push({
            id,
            message: r.reason instanceof Error ? r.reason.message : String(r.reason),
          });
        }
      });
      if (failedIds.length === 0) {
        message.success(`成功删除 ${successCount} 个工单`);
      } else if (successCount === 0) {
        message.error(`批量删除失败：${failedIds.length} 个工单均未删除`);
      } else {
        message.warning(`部分成功：${successCount} 个删除成功，${failedIds.length} 个失败`);
      }
      await refreshData();
      return { successCount, failedIds, errors };
    },
    [refreshData]
  );

  // 初始加载 + fetchTrigger 变化时重新拉取
  useEffect(() => {
    fetchTickets();
  }, [fetchTrigger]);

  // 注：统计拉取由上层页面（tickets/page.tsx）统一负责，避免同一会话内
  // /api/v1/tickets/stats 被多次重复调用。TicketList 组件不需要 stats 状态。

  return {
    tickets,
    loading,
    error,
    pagination,
    filters,
    fetchTickets,
    refreshData,
    updateFilters,
    updatePagination,
    createTicket,
    updateTicket,
    deleteTicket,
    batchDeleteTickets,
  };
};
