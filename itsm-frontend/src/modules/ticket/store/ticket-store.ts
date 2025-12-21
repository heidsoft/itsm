/**
 * ITSM前端架构 - 模块化状态管理示例
 * 
 * 工单管理模块状态管理
 * 展示如何使用统一的状态管理模式
 */

import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import { subscribeWithSelector } from 'zustand/middleware';
import { StateConfig, stateManagerFactory } from '@/lib/architecture/state';

// ==================== 类型定义 ====================

export interface Ticket {
  id: number;
  title: string;
  description: string;
  status: 'open' | 'in_progress' | 'resolved' | 'closed';
  priority: 'low' | 'medium' | 'high' | 'urgent';
  category: string;
  assignee_id?: number;
  assignee_name?: string;
  reporter_id: number;
  reporter_name: string;
  created_at: string;
  updated_at: string;
  due_date?: string;
  tags: string[];
  attachments: Attachment[];
  comments: Comment[];
}

export interface Attachment {
  id: number;
  name: string;
  url: string;
  size: number;
  type: string;
  uploaded_at: string;
}

export interface Comment {
  id: number;
  content: string;
  author_id: number;
  author_name: string;
  created_at: string;
  is_internal: boolean;
}

export interface TicketFilters {
  status?: string[];
  priority?: string[];
  category?: string[];
  assignee_id?: number[];
  reporter_id?: number[];
  tags?: string[];
  date_range?: {
    start: string;
    end: string;
  };
  search?: string;
}

export interface TicketListState {
  // 数据状态
  tickets: Ticket[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
  
  // UI状态
  loading: boolean;
  error: string | null;
  selectedTickets: number[];
  
  // 过滤和排序
  filters: TicketFilters;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
  
  // 操作方法
  fetchTickets: (params?: any) => Promise<void>;
  createTicket: (data: Partial<Ticket>) => Promise<void>;
  updateTicket: (id: number, data: Partial<Ticket>) => Promise<void>;
  deleteTicket: (id: number) => Promise<void>;
  batchDeleteTickets: (ids: number[]) => Promise<void>;
  
  // 选择操作
  selectTicket: (id: number) => void;
  deselectTicket: (id: number) => void;
  selectAll: () => void;
  deselectAll: () => void;
  
  // 过滤和排序
  setFilters: (filters: Partial<TicketFilters>) => void;
  updateFilter: (key: keyof TicketFilters, value: any) => void;
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

export interface TicketDetailState {
  // 数据状态
  ticket: Ticket | null;
  
  // UI状态
  loading: boolean;
  error: string | null;
  editing: boolean;
  
  // 操作方法
  fetchTicket: (id: number) => Promise<void>;
  updateTicket: (data: Partial<Ticket>) => Promise<void>;
  assignTicket: (assigneeId: number, comment?: string) => Promise<void>;
  escalateTicket: (level: number, reason: string, assigneeId?: number) => Promise<void>;
  resolveTicket: (resolution: string, category?: string) => Promise<void>;
  closeTicket: (reason?: string, rating?: number, comment?: string) => Promise<void>;
  reopenTicket: (reason?: string) => Promise<void>;
  
  // 评论操作
  addComment: (content: string, isInternal?: boolean) => Promise<void>;
  updateComment: (commentId: number, content: string) => Promise<void>;
  deleteComment: (commentId: number) => Promise<void>;
  
  // 附件操作
  uploadAttachment: (file: File) => Promise<void>;
  deleteAttachment: (attachmentId: number) => Promise<void>;
  
  // 工具方法
  setTicket: (ticket: Ticket | null) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  setEditing: (editing: boolean) => void;
  clearError: () => void;
  reset: () => void;
}

// ==================== 初始状态 ====================

const initialTicketListState: TicketListState = {
  tickets: [],
  total: 0,
  page: 1,
  pageSize: 20,
  totalPages: 0,
  loading: false,
  error: null,
  selectedTickets: [],
  filters: {},
  sortBy: 'created_at',
  sortOrder: 'desc',
  
  // 操作方法将在store中实现
  fetchTickets: async () => {},
  createTicket: async () => {},
  updateTicket: async () => {},
  deleteTicket: async () => {},
  batchDeleteTickets: async () => {},
  
  selectTicket: () => {},
  deselectTicket: () => {},
  selectAll: () => {},
  deselectAll: () => {},
  
  setFilters: () => {},
  updateFilter: () => {},
  clearFilters: () => {},
  setSorting: () => {},
  
  setPage: () => {},
  setPageSize: () => {},
  nextPage: () => {},
  prevPage: () => {},
  
  setLoading: () => {},
  setError: () => {},
  clearError: () => {},
  reset: () => {},
};

const initialTicketDetailState: TicketDetailState = {
  ticket: null,
  loading: false,
  error: null,
  editing: false,
  
  // 操作方法将在store中实现
  fetchTicket: async () => {},
  updateTicket: async () => {},
  assignTicket: async () => {},
  escalateTicket: async () => {},
  resolveTicket: async () => {},
  closeTicket: async () => {},
  reopenTicket: async () => {},
  
  addComment: async () => {},
  updateComment: async () => {},
  deleteComment: async () => {},
  
  uploadAttachment: async () => {},
  deleteAttachment: async () => {},
  
  setTicket: () => {},
  setLoading: () => {},
  setError: () => {},
  setEditing: () => {},
  clearError: () => {},
  reset: () => {},
};

// ==================== 状态配置 ====================

const ticketListConfig: StateConfig = {
  name: 'ticket-list-store',
  type: 'persistent',
  persist: true,
  storage: 'localStorage',
  sync: true,
  version: '1.0.0',
};

const ticketDetailConfig: StateConfig = {
  name: 'ticket-detail-store',
  type: 'client',
  persist: false,
  sync: false,
  version: '1.0.0',
};

// ==================== Zustand Store ====================

/**
 * 工单列表状态管理
 */
export const useTicketListStore = create<TicketListState>()(
  subscribeWithSelector(
    persist(
      (set, get) => ({
        ...initialTicketListState,
        
        // 获取工单列表
        fetchTickets: async (params?: any) => {
          set({ loading: true, error: null });
          try {
            // 模拟API调用
            await new Promise(resolve => setTimeout(resolve, 1000));
            
            const mockTickets: Ticket[] = Array.from({ length: 50 }, (_, i) => ({
              id: i + 1,
              title: `工单 ${i + 1}`,
              description: `这是工单 ${i + 1} 的描述`,
              status: ['open', 'in_progress', 'resolved', 'closed'][i % 4] as Ticket['status'],
              priority: ['low', 'medium', 'high', 'urgent'][i % 4] as Ticket['priority'],
              category: '技术支持',
              assignee_id: i % 3 === 0 ? 1 : undefined,
              assignee_name: i % 3 === 0 ? '张三' : undefined,
              reporter_id: 2,
              reporter_name: '李四',
              created_at: new Date(Date.now() - i * 86400000).toISOString(),
              updated_at: new Date(Date.now() - i * 43200000).toISOString(),
              due_date: new Date(Date.now() + (i + 1) * 86400000).toISOString(),
              tags: ['标签1', '标签2'],
              attachments: [],
              comments: [],
            }));
            
            const { page, pageSize, filters } = get();
            let filteredTickets = mockTickets;
            
            // 应用过滤器
            if (filters.status && filters.status.length > 0) {
              filteredTickets = filteredTickets.filter(ticket => 
                filters.status!.includes(ticket.status)
              );
            }
            
            if (filters.priority && filters.priority.length > 0) {
              filteredTickets = filteredTickets.filter(ticket => 
                filters.priority!.includes(ticket.priority)
              );
            }
            
            if (filters.search) {
              filteredTickets = filteredTickets.filter(ticket => 
                ticket.title.toLowerCase().includes(filters.search!.toLowerCase()) ||
                ticket.description.toLowerCase().includes(filters.search!.toLowerCase())
              );
            }
            
            const startIndex = (page - 1) * pageSize;
            const endIndex = startIndex + pageSize;
            const paginatedTickets = filteredTickets.slice(startIndex, endIndex);
            
            set({
              tickets: paginatedTickets,
              total: filteredTickets.length,
              totalPages: Math.ceil(filteredTickets.length / pageSize),
              loading: false,
            });
          } catch (error) {
            set({
              error: error instanceof Error ? error.message : '获取工单列表失败',
              loading: false,
            });
          }
        },
        
        // 创建工单
        createTicket: async (data: Partial<Ticket>) => {
          set({ loading: true, error: null });
          try {
            // 模拟API调用
            await new Promise(resolve => setTimeout(resolve, 500));
            
            const newTicket: Ticket = {
              id: Date.now(),
              title: data.title || '新工单',
              description: data.description || '',
              status: 'open',
              priority: 'medium',
              category: '技术支持',
              reporter_id: 1,
              reporter_name: '当前用户',
              created_at: new Date().toISOString(),
              updated_at: new Date().toISOString(),
              tags: [],
              attachments: [],
              comments: [],
              ...data,
            };
            
            set(state => ({
              tickets: [newTicket, ...state.tickets],
              total: state.total + 1,
              loading: false,
            }));
          } catch (error) {
            set({
              error: error instanceof Error ? error.message : '创建工单失败',
              loading: false,
            });
          }
        },
        
        // 更新工单
        updateTicket: async (id: number, data: Partial<Ticket>) => {
          set({ loading: true, error: null });
          try {
            // 模拟API调用
            await new Promise(resolve => setTimeout(resolve, 500));
            
            set(state => ({
              tickets: state.tickets.map(ticket =>
                ticket.id === id
                  ? { ...ticket, ...data, updated_at: new Date().toISOString() }
                  : ticket
              ),
              loading: false,
            }));
          } catch (error) {
            set({
              error: error instanceof Error ? error.message : '更新工单失败',
              loading: false,
            });
          }
        },
        
        // 删除工单
        deleteTicket: async (id: number) => {
          set({ loading: true, error: null });
          try {
            // 模拟API调用
            await new Promise(resolve => setTimeout(resolve, 300));
            
            set(state => ({
              tickets: state.tickets.filter(ticket => ticket.id !== id),
              total: state.total - 1,
              selectedTickets: state.selectedTickets.filter(ticketId => ticketId !== id),
              loading: false,
            }));
          } catch (error) {
            set({
              error: error instanceof Error ? error.message : '删除工单失败',
              loading: false,
            });
          }
        },
        
        // 批量删除工单
        batchDeleteTickets: async (ids: number[]) => {
          set({ loading: true, error: null });
          try {
            // 模拟API调用
            await new Promise(resolve => setTimeout(resolve, 1000));
            
            set(state => ({
              tickets: state.tickets.filter(ticket => !ids.includes(ticket.id)),
              total: state.total - ids.length,
              selectedTickets: [],
              loading: false,
            }));
          } catch (error) {
            set({
              error: error instanceof Error ? error.message : '批量删除工单失败',
              loading: false,
            });
          }
        },
        
        // 选择操作
        selectTicket: (id: number) => {
          set(state => ({
            selectedTickets: [...state.selectedTickets, id],
          }));
        },
        
        deselectTicket: (id: number) => {
          set(state => ({
            selectedTickets: state.selectedTickets.filter(ticketId => ticketId !== id),
          }));
        },
        
        selectAll: () => {
          set(state => ({
            selectedTickets: state.tickets.map(ticket => ticket.id),
          }));
        },
        
        deselectAll: () => {
          set({ selectedTickets: [] });
        },
        
        // 过滤和排序
        setFilters: (filters: Partial<TicketFilters>) => {
          set(state => ({
            filters: { ...state.filters, ...filters },
            page: 1, // 重置到第一页
          }));
        },
        
        updateFilter: (key: keyof TicketFilters, value: any) => {
          set(state => ({
            filters: { ...state.filters, [key]: value },
            page: 1,
          }));
        },
        
        clearFilters: () => {
          set({ filters: {}, page: 1 });
        },
        
        setSorting: (sortBy: string, sortOrder: 'asc' | 'desc') => {
          set({ sortBy, sortOrder });
        },
        
        // 分页
        setPage: (page: number) => {
          set({ page });
        },
        
        setPageSize: (pageSize: number) => {
          set({ pageSize, page: 1 });
        },
        
        nextPage: () => {
          set(state => {
            if (state.page < state.totalPages) {
              return { page: state.page + 1 };
            }
            return state;
          });
        },
        
        prevPage: () => {
          set(state => {
            if (state.page > 1) {
              return { page: state.page - 1 };
            }
            return state;
          });
        },
        
        // 工具方法
        setLoading: (loading: boolean) => {
          set({ loading });
        },
        
        setError: (error: string | null) => {
          set({ error });
        },
        
        clearError: () => {
          set({ error: null });
        },
        
        reset: () => {
          set(initialTicketListState);
        },
      }),
      {
        name: ticketListConfig.name,
        storage: createJSONStorage(() => localStorage),
        partialize: (state) => {
          // 只持久化必要的状态
          const { fetchTickets, createTicket, updateTicket, deleteTicket, batchDeleteTickets, ...persistedState } = state;
          return persistedState;
        },
      }
    )
  )
);

/**
 * 工单详情状态管理
 */
export const useTicketDetailStore = create<TicketDetailState>()(
  subscribeWithSelector(
    (set, get) => ({
      ...initialTicketDetailState,
      
      // 获取工单详情
      fetchTicket: async (id: number) => {
        set({ loading: true, error: null });
        try {
          // 模拟API调用
          await new Promise(resolve => setTimeout(resolve, 800));
          
          const mockTicket: Ticket = {
            id,
            title: `工单 ${id}`,
            description: `这是工单 ${id} 的详细描述`,
            status: 'in_progress',
            priority: 'high',
            category: '技术支持',
            assignee_id: 1,
            assignee_name: '张三',
            reporter_id: 2,
            reporter_name: '李四',
            created_at: new Date(Date.now() - 86400000).toISOString(),
            updated_at: new Date().toISOString(),
            due_date: new Date(Date.now() + 86400000).toISOString(),
            tags: ['标签1', '标签2'],
            attachments: [
              {
                id: 1,
                name: '附件1.pdf',
                url: '/attachments/1.pdf',
                size: 1024000,
                type: 'application/pdf',
                uploaded_at: new Date().toISOString(),
              },
            ],
            comments: [
              {
                id: 1,
                content: '这是第一条评论',
                author_id: 1,
                author_name: '张三',
                created_at: new Date(Date.now() - 3600000).toISOString(),
                is_internal: false,
              },
            ],
          };
          
          set({ ticket: mockTicket, loading: false });
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : '获取工单详情失败',
            loading: false,
          });
        }
      },
      
      // 更新工单
      updateTicket: async (data: Partial<Ticket>) => {
        set({ loading: true, error: null });
        try {
          // 模拟API调用
          await new Promise(resolve => setTimeout(resolve, 500));
          
          set(state => ({
            ticket: state.ticket ? { ...state.ticket, ...data, updated_at: new Date().toISOString() } : null,
            loading: false,
          }));
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : '更新工单失败',
            loading: false,
          });
        }
      },
      
      // 分配工单
      assignTicket: async (assigneeId: number, comment?: string) => {
        set({ loading: true, error: null });
        try {
          // 模拟API调用
          await new Promise(resolve => setTimeout(resolve, 500));
          
          set(state => ({
            ticket: state.ticket ? {
              ...state.ticket,
              assignee_id: assigneeId,
              assignee_name: '新处理人',
              updated_at: new Date().toISOString(),
            } : null,
            loading: false,
          }));
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : '分配工单失败',
            loading: false,
          });
        }
      },
      
      // 解决工单
      resolveTicket: async (resolution: string, category?: string) => {
        set({ loading: true, error: null });
        try {
          // 模拟API调用
          await new Promise(resolve => setTimeout(resolve, 500));
          
          set(state => ({
            ticket: state.ticket ? {
              ...state.ticket,
              status: 'resolved',
              updated_at: new Date().toISOString(),
            } : null,
            loading: false,
          }));
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : '解决工单失败',
            loading: false,
          });
        }
      },
      
      // 关闭工单
      closeTicket: async (reason?: string, rating?: number, comment?: string) => {
        set({ loading: true, error: null });
        try {
          // 模拟API调用
          await new Promise(resolve => setTimeout(resolve, 500));
          
          set(state => ({
            ticket: state.ticket ? {
              ...state.ticket,
              status: 'closed',
              updated_at: new Date().toISOString(),
            } : null,
            loading: false,
          }));
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : '关闭工单失败',
            loading: false,
          });
        }
      },
      
      // 重新打开工单
      reopenTicket: async (reason?: string) => {
        set({ loading: true, error: null });
        try {
          // 模拟API调用
          await new Promise(resolve => setTimeout(resolve, 500));
          
          set(state => ({
            ticket: state.ticket ? {
              ...state.ticket,
              status: 'open',
              updated_at: new Date().toISOString(),
            } : null,
            loading: false,
          }));
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : '重新打开工单失败',
            loading: false,
          });
        }
      },
      
      // 添加评论
      addComment: async (content: string, isInternal = false) => {
        set({ loading: true, error: null });
        try {
          // 模拟API调用
          await new Promise(resolve => setTimeout(resolve, 300));
          
          const newComment: Comment = {
            id: Date.now(),
            content,
            author_id: 1,
            author_name: '当前用户',
            created_at: new Date().toISOString(),
            is_internal: isInternal,
          };
          
          set(state => ({
            ticket: state.ticket ? {
              ...state.ticket,
              comments: [...state.ticket.comments, newComment],
              updated_at: new Date().toISOString(),
            } : null,
            loading: false,
          }));
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : '添加评论失败',
            loading: false,
          });
        }
      },
      
      // 上传附件
      uploadAttachment: async (file: File) => {
        set({ loading: true, error: null });
        try {
          // 模拟API调用
          await new Promise(resolve => setTimeout(resolve, 1000));
          
          const newAttachment: Attachment = {
            id: Date.now(),
            name: file.name,
            url: URL.createObjectURL(file),
            size: file.size,
            type: file.type,
            uploaded_at: new Date().toISOString(),
          };
          
          set(state => ({
            ticket: state.ticket ? {
              ...state.ticket,
              attachments: [...state.ticket.attachments, newAttachment],
              updated_at: new Date().toISOString(),
            } : null,
            loading: false,
          }));
        } catch (error) {
          set({
            error: error instanceof Error ? error.message : '上传附件失败',
            loading: false,
          });
        }
      },
      
      // 工具方法
      setTicket: (ticket: Ticket | null) => {
        set({ ticket });
      },
      
      setLoading: (loading: boolean) => {
        set({ loading });
      },
      
      setError: (error: string | null) => {
        set({ error });
      },
      
      setEditing: (editing: boolean) => {
        set({ editing });
      },
      
      clearError: () => {
        set({ error: null });
      },
      
      reset: () => {
        set(initialTicketDetailState);
      },
    })
  )
);

// ==================== 状态管理器注册 ====================

// 注册工单列表状态管理器
stateManagerFactory.createClientState('ticket-list-store', ticketListConfig, initialTicketListState);

// 注册工单详情状态管理器
stateManagerFactory.createClientState('ticket-detail-store', ticketDetailConfig, initialTicketDetailState);

// ==================== 导出 ====================

export default {
  useTicketListStore,
  useTicketDetailStore,
  ticketListConfig,
  ticketDetailConfig,
};
