'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { message } from 'antd';
import { ticketService, Ticket, TicketStatus, TicketPriority, TicketType } from '@/lib/services/ticket-service';

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

export interface PaginationState {
  current: number;
  pageSize: number;
  total: number;
}

// Query Keys
export const ticketKeys = {
  all: ['tickets'] as const,
  lists: () => [...ticketKeys.all, 'list'] as const,
  list: (filters: Partial<TicketQueryFilters>, pagination: PaginationState) =>
    [...ticketKeys.lists(), filters, pagination] as const,
  details: () => [...ticketKeys.all, 'detail'] as const,
  detail: (id: string) => [...ticketKeys.details(), id] as const,
  stats: () => [...ticketKeys.all, 'stats'] as const,
};

// 获取工单列表
export const useTicketsQuery = (
  filters: Partial<TicketQueryFilters> = {},
  pagination: PaginationState = { current: 1, pageSize: 20, total: 0 }
) => {
  return useQuery({
    queryKey: ticketKeys.list(filters, pagination),
    queryFn: async () => {
      const response = await ticketService.listTickets({
        page: pagination.current,
        page_size: pagination.pageSize,
        ...filters,
      });
      return response;
    },
    enabled: true,
    staleTime: 2 * 60 * 1000, // 2分钟缓存
    gcTime: 5 * 60 * 1000, // 5分钟垃圾回收
  });
};

// 获取工单统计
export const useTicketStatsQuery = () => {
  return useQuery({
    queryKey: ticketKeys.stats(),
    queryFn: async () => {
      const response = await ticketService.getTicketStats();
      return {
        total: response.total,
        open: response.open,
        resolved: response.resolved,
        highPriority: response.high_priority,
      };
    },
    staleTime: 1 * 60 * 1000, // 1分钟缓存
    gcTime: 3 * 60 * 1000, // 3分钟垃圾回收
  });
};

// 获取单个工单详情
export const useTicketDetailQuery = (id: string) => {
  return useQuery({
    queryKey: ticketKeys.detail(id),
    queryFn: async () => {
      const response = await ticketService.getTicket(id);
      return response;
    },
    enabled: !!id,
    staleTime: 5 * 60 * 1000, // 5分钟缓存
    gcTime: 10 * 60 * 1000, // 10分钟垃圾回收
  });
};

// 创建工单
export const useCreateTicketMutation = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (ticketData: any) => {
      const response = await ticketService.createTicket(ticketData);
      return response;
    },
    onSuccess: (data) => {
      message.success('Ticket created successfully');
      
      // 使相关查询失效，触发重新获取
      queryClient.invalidateQueries({ queryKey: ticketKeys.lists() });
      queryClient.invalidateQueries({ queryKey: ticketKeys.stats() });
      
      // 乐观更新统计
      queryClient.setQueryData(ticketKeys.stats(), (old: TicketStats | undefined) => {
        if (!old) return old;
        return {
          ...old,
          total: old.total + 1,
        };
      });
    },
    onError: (error: any) => {
      const errorMessage = error?.message || 'Failed to create ticket';
      message.error(errorMessage);
    },
  });
};

// 更新工单
export const useUpdateTicketMutation = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async ({ id, data }: { id: string; data: any }) => {
      const response = await ticketService.updateTicket(id, data);
      return response;
    },
    onSuccess: (data, variables) => {
      message.success('Ticket updated successfully');
      
      // 更新缓存中的工单详情
      queryClient.setQueryData(ticketKeys.detail(variables.id), data);
      
      // 使列表查询失效
      queryClient.invalidateQueries({ queryKey: ticketKeys.lists() });
      queryClient.invalidateQueries({ queryKey: ticketKeys.stats() });
    },
    onError: (error: any) => {
      const errorMessage = error?.message || 'Failed to update ticket';
      message.error(errorMessage);
    },
  });
};

// 删除工单
export const useDeleteTicketMutation = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) => {
      await ticketService.deleteTicket(Number(id));
      return id;
    },
    onSuccess: (id) => {
      message.success('Ticket deleted successfully');
      
      // 从缓存中移除工单详情
      queryClient.removeQueries({ queryKey: ticketKeys.detail(id) });
      
      // 使列表查询失效
      queryClient.invalidateQueries({ queryKey: ticketKeys.lists() });
      queryClient.invalidateQueries({ queryKey: ticketKeys.stats() });
      
      // 乐观更新统计
      queryClient.setQueryData(ticketKeys.stats(), (old: TicketStats | undefined) => {
        if (!old) return old;
        return {
          ...old,
          total: Math.max(0, old.total - 1),
        };
      });
    },
    onError: (error: any) => {
      const errorMessage = error?.message || 'Failed to delete ticket';
      message.error(errorMessage);
    },
  });
};

// 批量删除工单
export const useBatchDeleteTicketsMutation = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (ids: string[]) => {
      await Promise.all(ids.map(id => ticketService.deleteTicket(Number(id))));
      return ids;
    },
    onSuccess: (ids) => {
      message.success(`${ids.length} tickets deleted successfully`);
      
      // 从缓存中移除工单详情
      ids.forEach(id => {
        queryClient.removeQueries({ queryKey: ticketKeys.detail(id) });
      });
      
      // 使列表查询失效
      queryClient.invalidateQueries({ queryKey: ticketKeys.lists() });
      queryClient.invalidateQueries({ queryKey: ticketKeys.stats() });
      
      // 乐观更新统计
      queryClient.setQueryData(ticketKeys.stats(), (old: TicketStats | undefined) => {
        if (!old) return old;
        return {
          ...old,
          total: Math.max(0, old.total - ids.length),
        };
      });
    },
    onError: (error: any) => {
      const errorMessage = error?.message || 'Failed to delete tickets';
      message.error(errorMessage);
    },
  });
};

// 预加载工单详情
export const usePrefetchTicketDetail = () => {
  const queryClient = useQueryClient();

  return (id: string) => {
    queryClient.prefetchQuery({
      queryKey: ticketKeys.detail(id),
      queryFn: async () => {
        const response = await ticketService.getTicket(id);
        return response;
      },
      staleTime: 5 * 60 * 1000,
    });
  };
};

// 手动刷新数据
export const useRefreshTickets = () => {
  const queryClient = useQueryClient();

  return () => {
    queryClient.invalidateQueries({ queryKey: ticketKeys.all });
  };
};
