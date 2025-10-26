import { create } from 'zustand';
import { TicketAPI, Ticket, ListTicketsRequest, CreateTicketRequest, UpdateTicketRequest } from '../api/ticket-api';

/**
 * 工单列表状态
 */
interface TicketListState {
  // 数据状态
  tickets: Ticket[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
  
  // UI状态
  loading: boolean;
  error: string | null;
  selectedTickets: Ticket[];
  
  // 过滤和排序
  filters: Partial<ListTicketsRequest>;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
  
  // 操作方法
  fetchTickets: (params?: Partial<ListTicketsRequest>) => Promise<void>;
  createTicket: (data: CreateTicketRequest) => Promise<void>;
  updateTicket: (id: number, data: UpdateTicketRequest) => Promise<void>;
  deleteTicket: (id: number) => Promise<void>;
  batchDeleteTickets: (ids: number[]) => Promise<void>;
  
  // 选择操作
  selectTicket: (ticket: Ticket) => void;
  deselectTicket: (ticket: Ticket) => void;
  selectAll: () => void;
  deselectAll: () => void;
  
  // 过滤和排序
  setFilters: (filters: Partial<ListTicketsRequest>) => void;
  updateFilter: (key: keyof ListTicketsRequest, value: unknown) => void;
  clearFilters: () => void;
  setSorting: (sortBy: string, sortOrder: 'asc' | 'desc') => void;
  
  // 分页
  setPage: (page: number) => void;
  setPageSize: (pageSize: number) => void;
  nextPage: () => void;
  prevPage: () => void;
  
  // 工具方法
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  clearError: () => void;
  reset: () => void;
}

/**
 * 工单详情状态
 */
interface TicketDetailState {
  // 数据状态
  ticket: Ticket | null;
  
  // UI状态
  loading: boolean;
  error: string | null;
  
  // 操作方法
  fetchTicket: (id: number) => Promise<void>;
  updateTicket: (data: UpdateTicketRequest) => Promise<void>;
  assignTicket: (assigneeId: number, comment?: string) => Promise<void>;
  escalateTicket: (level: number, reason: string, assigneeId?: number) => Promise<void>;
  resolveTicket: (resolution: string, category?: string) => Promise<void>;
  closeTicket: (reason?: string, rating?: number, comment?: string) => Promise<void>;
  reopenTicket: (reason?: string) => Promise<void>;
  
  // 工具方法
  setTicket: (ticket: Ticket | null) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  clearError: () => void;
  reset: () => void;
}

/**
 * 创建工单列表store
 */
export const useTicketListStore = create<TicketListState>((set, get) => ({
  // 初始状态
  tickets: [],
  total: 0,
  page: 1,
  pageSize: 20,
  totalPages: 0,
  loading: false,
  error: null,
  selectedTickets: [],
  filters: {},
  sortBy: undefined,
  sortOrder: undefined,

  // 获取工单列表
  fetchTickets: async (params = {}) => {
    const state = get();
    const requestParams = {
      page: state.page,
      page_size: state.pageSize,
      sort_by: state.sortBy,
      sort_order: state.sortOrder,
      ...state.filters,
      ...params,
    };

    try {
      set({ loading: true, error: null });
      const response = await TicketAPI.listTickets(requestParams);
      
      set({
        tickets: response.tickets,
        total: response.total,
        page: response.page,
        pageSize: response.page_size,
        totalPages: response.total_pages,
        loading: false,
      });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : '获取工单列表失败';
      set({ error: errorMessage, loading: false });
      console.error('Failed to fetch tickets:', error);
    }
  },

  // 创建工单
  createTicket: async (data) => {
    try {
      set({ loading: true, error: null });
      await TicketAPI.createTicket(data);
      
      // 重新获取列表
      const state = get();
      await state.fetchTickets();
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : '创建工单失败';
      set({ error: errorMessage, loading: false });
      console.error('Failed to create ticket:', error);
      throw error;
    }
  },

  // 更新工单
  updateTicket: async (id, data) => {
    try {
      set({ loading: true, error: null });
      const response = await TicketAPI.updateTicket(id, data);
      
      // 更新本地状态
      const state = get();
      const updatedTickets = state.tickets.map(ticket =>
        ticket.id === id ? response.ticket : ticket
      );
      
      set({ tickets: updatedTickets, loading: false });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : '更新工单失败';
      set({ error: errorMessage, loading: false });
      console.error('Failed to update ticket:', error);
      throw error;
    }
  },

  // 删除工单
  deleteTicket: async (id) => {
    try {
      set({ loading: true, error: null });
      await TicketAPI.deleteTicket(id);
      
      // 从本地状态中移除
      const state = get();
      const filteredTickets = state.tickets.filter(ticket => ticket.id !== id);
      const filteredSelected = state.selectedTickets.filter(ticket => ticket.id !== id);
      
      set({
        tickets: filteredTickets,
        selectedTickets: filteredSelected,
        total: state.total - 1,
        loading: false,
      });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : '删除工单失败';
      set({ error: errorMessage, loading: false });
      console.error('Failed to delete ticket:', error);
      throw error;
    }
  },

  // 批量删除工单
  batchDeleteTickets: async (ids) => {
    try {
      set({ loading: true, error: null });
      await TicketAPI.batchDeleteTickets(ids);
      
      // 从本地状态中移除
      const state = get();
      const filteredTickets = state.tickets.filter(ticket => !ids.includes(ticket.id));
      
      set({
        tickets: filteredTickets,
        selectedTickets: [],
        total: state.total - ids.length,
        loading: false,
      });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : '批量删除工单失败';
      set({ error: errorMessage, loading: false });
      console.error('Failed to batch delete tickets:', error);
      throw error;
    }
  },

  // 选择工单
  selectTicket: (ticket) => {
    const state = get();
    const isSelected = state.selectedTickets.some(selected => selected.id === ticket.id);
    if (!isSelected) {
      set({ selectedTickets: [...state.selectedTickets, ticket] });
    }
  },

  // 取消选择工单
  deselectTicket: (ticket) => {
    const state = get();
    const filteredSelected = state.selectedTickets.filter(selected => selected.id !== ticket.id);
    set({ selectedTickets: filteredSelected });
  },

  // 全选
  selectAll: () => {
    const state = get();
    set({ selectedTickets: [...state.tickets] });
  },

  // 取消全选
  deselectAll: () => {
    set({ selectedTickets: [] });
  },

  // 设置过滤条件
  setFilters: (filters) => {
    set({ filters, page: 1 });
  },

  // 更新单个过滤条件
  updateFilter: (key, value) => {
    const state = get();
    set({
      filters: { ...state.filters, [key]: value },
      page: 1,
    });
  },

  // 清除过滤条件
  clearFilters: () => {
    set({ filters: {}, page: 1 });
  },

  // 设置排序
  setSorting: (sortBy, sortOrder) => {
    set({ sortBy, sortOrder });
  },

  // 设置页码
  setPage: (page) => {
    set({ page });
  },

  // 设置页大小
  setPageSize: (pageSize) => {
    set({ pageSize, page: 1 });
  },

  // 下一页
  nextPage: () => {
    const state = get();
    if (state.page < state.totalPages) {
      set({ page: state.page + 1 });
    }
  },

  // 上一页
  prevPage: () => {
    const state = get();
    if (state.page > 1) {
      set({ page: state.page - 1 });
    }
  },

  // 设置加载状态
  setLoading: (loading) => {
    set({ loading });
  },

  // 设置错误信息
  setError: (error) => {
    set({ error });
  },

  // 清除错误信息
  clearError: () => {
    set({ error: null });
  },

  // 重置状态
  reset: () => {
    set({
      tickets: [],
      total: 0,
      page: 1,
      pageSize: 20,
      totalPages: 0,
      loading: false,
      error: null,
      selectedTickets: [],
      filters: {},
      sortBy: undefined,
      sortOrder: undefined,
    });
  },
}));

/**
 * 创建工单详情store
 */
export const useTicketDetailStore = create<TicketDetailState>((set, get) => ({
  // 初始状态
  ticket: null,
  loading: false,
  error: null,

  // 获取工单详情
  fetchTicket: async (id) => {
    try {
      set({ loading: true, error: null });
      const response = await TicketAPI.getTicket(id);
      set({ ticket: response.ticket, loading: false });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : '获取工单详情失败';
      set({ error: errorMessage, loading: false });
      console.error('Failed to fetch ticket:', error);
    }
  },

  // 更新工单
  updateTicket: async (data) => {
    const state = get();
    if (!state.ticket) return;

    try {
      set({ loading: true, error: null });
      const response = await TicketAPI.updateTicket(state.ticket.id, data);
      set({ ticket: response.ticket, loading: false });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : '更新工单失败';
      set({ error: errorMessage, loading: false });
      console.error('Failed to update ticket:', error);
      throw error;
    }
  },

  // 分配工单
  assignTicket: async (assigneeId, comment) => {
    const state = get();
    if (!state.ticket) return;

    try {
      set({ loading: true, error: null });
      const response = await TicketAPI.assignTicket(state.ticket.id, { assignee_id: assigneeId, comment });
      set({ ticket: response.ticket, loading: false });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : '分配工单失败';
      set({ error: errorMessage, loading: false });
      console.error('Failed to assign ticket:', error);
      throw error;
    }
  },

  // 升级工单
  escalateTicket: async (level, reason, assigneeId) => {
    const state = get();
    if (!state.ticket) return;

    try {
      set({ loading: true, error: null });
      const response = await TicketAPI.escalateTicket(state.ticket.id, {
        escalation_level: level,
        reason,
        assignee_id: assigneeId,
      });
      set({ ticket: response.ticket, loading: false });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : '升级工单失败';
      set({ error: errorMessage, loading: false });
      console.error('Failed to escalate ticket:', error);
      throw error;
    }
  },

  // 解决工单
  resolveTicket: async (resolution, category) => {
    const state = get();
    if (!state.ticket) return;

    try {
      set({ loading: true, error: null });
      const response = await TicketAPI.resolveTicket(state.ticket.id, {
        resolution,
        resolution_category: category,
      });
      set({ ticket: response.ticket, loading: false });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : '解决工单失败';
      set({ error: errorMessage, loading: false });
      console.error('Failed to resolve ticket:', error);
      throw error;
    }
  },

  // 关闭工单
  closeTicket: async (reason, rating, comment) => {
    const state = get();
    if (!state.ticket) return;

    try {
      set({ loading: true, error: null });
      const response = await TicketAPI.closeTicket(state.ticket.id, {
        close_reason: reason,
        satisfaction_rating: rating,
        satisfaction_comment: comment,
      });
      set({ ticket: response.ticket, loading: false });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : '关闭工单失败';
      set({ error: errorMessage, loading: false });
      console.error('Failed to close ticket:', error);
      throw error;
    }
  },

  // 重新打开工单
  reopenTicket: async (reason) => {
    const state = get();
    if (!state.ticket) return;

    try {
      set({ loading: true, error: null });
      const response = await TicketAPI.reopenTicket(state.ticket.id, reason);
      set({ ticket: response.ticket, loading: false });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : '重新打开工单失败';
      set({ error: errorMessage, loading: false });
      console.error('Failed to reopen ticket:', error);
      throw error;
    }
  },

  // 设置工单
  setTicket: (ticket) => {
    set({ ticket });
  },

  // 设置加载状态
  setLoading: (loading) => {
    set({ loading });
  },

  // 设置错误信息
  setError: (error) => {
    set({ error });
  },

  // 清除错误信息
  clearError: () => {
    set({ error: null });
  },

  // 重置状态
  reset: () => {
    set({
      ticket: null,
      loading: false,
      error: null,
    });
  },
}));

/**
 * 主工单 Store - 组合工单列表和详情
 * 为了向后兼容，提供一个统一的 useTicketStore
 */
export const useTicketStore = () => {
  const listStore = useTicketListStore();
  const detailStore = useTicketDetailStore();
  
  return {
    // 列表相关
    ...listStore,
    // 详情相关
    ticket: detailStore.ticket,
    fetchTicket: detailStore.fetchTicket,
    updateTicketDetail: detailStore.updateTicket,
    assignTicket: detailStore.assignTicket,
    escalateTicket: detailStore.escalateTicket,
    resolveTicket: detailStore.resolveTicket,
    closeTicket: detailStore.closeTicket,
    reopenTicket: detailStore.reopenTicket,
    setTicket: detailStore.setTicket,
  };
};