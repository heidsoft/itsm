"use client";

import { create } from "zustand";
import { devtools, persist } from "zustand/middleware";
import { Ticket, TicketStatus, TicketPriority, TicketType } from "@/lib/services/ticket-service";

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

export interface TicketStore {
  // Data state
  tickets: Ticket[];
  stats: TicketStats;
  loading: boolean;
  error: string | null;
  
  // UI state
  selectedRowKeys: Array<string | number>;
  modalVisible: boolean;
  editingTicket: Ticket | null;
  templateModalVisible: boolean;
  
  // Filters and pagination
  filters: Partial<TicketQueryFilters>;
  pagination: PaginationState;
  
  // Actions - Data management
  setTickets: (tickets: Ticket[]) => void;
  setStats: (stats: TicketStats) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  
  // Actions - UI state
  setSelectedRowKeys: (keys: Array<string | number>) => void;
  setModalVisible: (visible: boolean) => void;
  setEditingTicket: (ticket: Ticket | null) => void;
  setTemplateModalVisible: (visible: boolean) => void;
  
  // Actions - Filters and pagination
  setFilters: (filters: Partial<TicketQueryFilters>) => void;
  updateFilters: (filters: Partial<TicketQueryFilters>) => void;
  setPagination: (pagination: Partial<PaginationState>) => void;
  updatePagination: (page: number, pageSize: number) => void;
  
  // Actions - Ticket operations
  addTicket: (ticket: Ticket) => void;
  updateTicket: (id: number, updates: Partial<Ticket>) => void;
  removeTicket: (id: number) => void;
  removeTickets: (ids: number[]) => void;
  
  // Actions - Reset
  resetStore: () => void;
  resetFilters: () => void;
  resetUI: () => void;
}

const initialState = {
  // Data state
  tickets: [],
  stats: {
    total: 0,
    open: 0,
    resolved: 0,
    highPriority: 0,
  },
  loading: false,
  error: null,
  
  // UI state
  selectedRowKeys: [],
  modalVisible: false,
  editingTicket: null,
  templateModalVisible: false,
  
  // Filters and pagination
  filters: {},
  pagination: {
    current: 1,
    pageSize: 20,
    total: 0,
  },
};

export const useTicketStore = create<TicketStore>()(
  devtools(
    persist(
      (set, get) => ({
        ...initialState,
        
        // Data management actions
        setTickets: (tickets) => set({ tickets }, false, "setTickets"),
        setStats: (stats) => set({ stats }, false, "setStats"),
        setLoading: (loading) => set({ loading }, false, "setLoading"),
        setError: (error) => set({ error }, false, "setError"),
        
        // UI state actions
        setSelectedRowKeys: (selectedRowKeys) => 
          set({ selectedRowKeys }, false, "setSelectedRowKeys"),
        setModalVisible: (modalVisible) => 
          set({ modalVisible }, false, "setModalVisible"),
        setEditingTicket: (editingTicket) => 
          set({ editingTicket }, false, "setEditingTicket"),
        setTemplateModalVisible: (templateModalVisible) => 
          set({ templateModalVisible }, false, "setTemplateModalVisible"),
        
        // Filters and pagination actions
        setFilters: (filters) => set({ filters }, false, "setFilters"),
        updateFilters: (newFilters) => 
          set((state) => ({ 
            filters: { ...state.filters, ...newFilters },
            pagination: { ...state.pagination, current: 1 }
          }), false, "updateFilters"),
        setPagination: (pagination) => 
          set((state) => ({ 
            pagination: { ...state.pagination, ...pagination }
          }), false, "setPagination"),
        updatePagination: (current, pageSize) => 
          set((state) => ({ 
            pagination: { ...state.pagination, current, pageSize }
          }), false, "updatePagination"),
        
        // Ticket operations
        addTicket: (ticket) => 
          set((state) => ({ 
            tickets: [ticket, ...state.tickets],
            stats: {
              ...state.stats,
              total: state.stats.total + 1,
            }
          }), false, "addTicket"),
        updateTicket: (id, updates) => 
          set((state) => ({
            tickets: state.tickets.map(ticket => 
              ticket.id === id ? { ...ticket, ...updates } : ticket
            )
          }), false, "updateTicket"),
        removeTicket: (id) => 
          set((state) => ({
            tickets: state.tickets.filter(ticket => ticket.id !== id),
            stats: {
              ...state.stats,
              total: Math.max(0, state.stats.total - 1),
            }
          }), false, "removeTicket"),
        removeTickets: (ids) => 
          set((state) => ({
            tickets: state.tickets.filter(ticket => !ids.includes(ticket.id)),
            stats: {
              ...state.stats,
              total: Math.max(0, state.stats.total - ids.length),
            }
          }), false, "removeTickets"),
        
        // Reset actions
        resetStore: () => set(initialState, false, "resetStore"),
        resetFilters: () => set({ filters: {} }, false, "resetFilters"),
        resetUI: () => set({
          selectedRowKeys: [],
          modalVisible: false,
          editingTicket: null,
          templateModalVisible: false,
        }, false, "resetUI"),
      }),
      {
        name: "ticket-store",
        partialize: (state) => ({
          // Only persist filters and pagination, not data
          filters: state.filters,
          pagination: state.pagination,
        }),
      }
    ),
    {
      name: "ticket-store",
    }
  )
);

// Selectors for better performance
export const useTicketsData = () => useTicketStore((state) => ({
  tickets: state.tickets,
  stats: state.stats,
  loading: state.loading,
  error: state.error,
}));

export const useTicketsUI = () => useTicketStore((state) => ({
  selectedRowKeys: state.selectedRowKeys,
  modalVisible: state.modalVisible,
  editingTicket: state.editingTicket,
  templateModalVisible: state.templateModalVisible,
}));

export const useTicketsFilters = () => useTicketStore((state) => ({
  filters: state.filters,
  pagination: state.pagination,
}));

export const useTicketsActions = () => useTicketStore((state) => ({
  setTickets: state.setTickets,
  setStats: state.setStats,
  setLoading: state.setLoading,
  setError: state.setError,
  setSelectedRowKeys: state.setSelectedRowKeys,
  setModalVisible: state.setModalVisible,
  setEditingTicket: state.setEditingTicket,
  setTemplateModalVisible: state.setTemplateModalVisible,
  setFilters: state.setFilters,
  updateFilters: state.updateFilters,
  setPagination: state.setPagination,
  updatePagination: state.updatePagination,
  addTicket: state.addTicket,
  updateTicket: state.updateTicket,
  removeTicket: state.removeTicket,
  removeTickets: state.removeTickets,
  resetStore: state.resetStore,
  resetFilters: state.resetFilters,
  resetUI: state.resetUI,
}));
