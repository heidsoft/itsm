"use client";

import { create } from "zustand";
import { devtools } from "zustand/middleware";

export interface Ticket {
  id: number;
  title: string;
  description: string;
  status: string;
  priority: string;
  type?: string;
  ticket_number: string;
  requester_id: number;
  assignee_id?: number;
  tenant_id: number;
  created_at: string;
  updated_at: string;
  requester?: any;
  assignee?: any;
  tenant?: any;
  category?: string;
  subcategory?: string;
  impact?: string;
  urgency?: string;
  resolution?: string;
  work_notes?: string;
  attachments?: any[];
  workflow_steps?: any[];
  comments?: any[];
  sla_info?: any;
  tags?: string[];
  due_date?: string;
  escalation_level?: number;
}

export interface TicketStats {
  total: number;
  open: number;
  resolved: number;
  highPriority: number;
  overdue: number;
  dueToday: number;
  thisWeek: number;
  thisMonth: number;
}

interface TicketDataState {
  // Data
  tickets: Ticket[];
  stats: TicketStats;
  currentTicket: Ticket | null;
  
  // Loading states
  loading: boolean;
  refreshing: boolean;
  error: string | null;
  
  // Cache management
  lastFetchTime: number;
  cacheValidMs: number; // Cache validity in milliseconds
}

interface TicketDataActions {
  // Data actions
  setTickets: (tickets: Ticket[]) => void;
  setStats: (stats: TicketStats) => void;
  setCurrentTicket: (ticket: Ticket | null) => void;
  
  // CRUD operations
  addTicket: (ticket: Ticket) => void;
  updateTicket: (id: number, updates: Partial<Ticket>) => void;
  removeTicket: (id: number) => void;
  removeTickets: (ids: number[]) => void;
  addComment: (ticketId: number, comment: any) => void;
  addAttachment: (ticketId: number, attachment: any) => void;
  
  // Loading actions
  setLoading: (loading: boolean) => void;
  setRefreshing: (refreshing: boolean) => void;
  setError: (error: string | null) => void;
  
  // Cache actions
  updateCacheTime: () => void;
  isCacheValid: () => boolean;
  clearCache: () => void;
  
  // Reset actions
  resetData: () => void;
  clearError: () => void;
}

type TicketDataStore = TicketDataState & TicketDataActions;

const initialDataState: TicketDataState = {
  tickets: [],
  stats: {
    total: 0,
    open: 0,
    resolved: 0,
    highPriority: 0,
    overdue: 0,
    dueToday: 0,
    thisWeek: 0,
    thisMonth: 0,
  },
  currentTicket: null,
  loading: false,
  refreshing: false,
  error: null,
  lastFetchTime: 0,
  cacheValidMs: 5 * 60 * 1000, // 5 minutes
};

export const useTicketDataStore = create<TicketDataStore>()(
  devtools(
    (set, get) => ({
      ...initialDataState,
      
      // Data actions
      setTickets: (tickets) => 
        set({ 
          tickets, 
          lastFetchTime: Date.now(),
          error: null 
        }, false, "setTickets"),
      
      setStats: (stats) => 
        set({ stats }, false, "setStats"),
      
      setCurrentTicket: (currentTicket) => 
        set({ currentTicket }, false, "setCurrentTicket"),
      
      // CRUD operations
      addTicket: (ticket) => 
        set((state) => {
          const newTickets = [ticket, ...state.tickets];
          const updatedStats = calculateStats(newTickets);
          
          return {
            tickets: newTickets,
            stats: updatedStats,
            error: null,
          };
        }, false, "addTicket"),
      
      updateTicket: (id, updates) => 
        set((state) => {
          const newTickets = state.tickets.map(ticket => 
            ticket.id === id 
              ? { ...ticket, ...updates, updated_at: new Date().toISOString() }
              : ticket
          );
          const updatedStats = calculateStats(newTickets);
          
          // Update currentTicket if it's the one being updated
          const updatedCurrentTicket = state.currentTicket?.id === id 
            ? { ...state.currentTicket, ...updates, updated_at: new Date().toISOString() }
            : state.currentTicket;
          
          return {
            tickets: newTickets,
            stats: updatedStats,
            currentTicket: updatedCurrentTicket,
            error: null,
          };
        }, false, "updateTicket"),
      
      removeTicket: (id) => 
        set((state) => {
          const newTickets = state.tickets.filter(ticket => ticket.id !== id);
          const updatedStats = calculateStats(newTickets);
          
          // Clear currentTicket if it's the one being removed
          const updatedCurrentTicket = state.currentTicket?.id === id 
            ? null 
            : state.currentTicket;
          
          return {
            tickets: newTickets,
            stats: updatedStats,
            currentTicket: updatedCurrentTicket,
          };
        }, false, "removeTicket"),
      
      removeTickets: (ids) => 
        set((state) => {
          const newTickets = state.tickets.filter(ticket => !ids.includes(ticket.id));
          const updatedStats = calculateStats(newTickets);
          
          // Clear currentTicket if it's in the removed list
          const updatedCurrentTicket = state.currentTicket && ids.includes(state.currentTicket.id)
            ? null 
            : state.currentTicket;
          
          return {
            tickets: newTickets,
            stats: updatedStats,
            currentTicket: updatedCurrentTicket,
          };
        }, false, "removeTickets"),
      
      addComment: (ticketId, comment) => 
        set((state) => {
          const newTickets = state.tickets.map(ticket => 
            ticket.id === ticketId 
              ? { 
                  ...ticket, 
                  comments: [...(ticket.comments || []), comment],
                  updated_at: new Date().toISOString()
                }
              : ticket
          );
          
          // Update currentTicket if it's the one receiving the comment
          const updatedCurrentTicket = state.currentTicket?.id === ticketId 
            ? { 
                ...state.currentTicket, 
                comments: [...(state.currentTicket.comments || []), comment],
                updated_at: new Date().toISOString()
              }
            : state.currentTicket;
          
          return {
            tickets: newTickets,
            currentTicket: updatedCurrentTicket,
          };
        }, false, "addComment"),
      
      addAttachment: (ticketId, attachment) => 
        set((state) => {
          const newTickets = state.tickets.map(ticket => 
            ticket.id === ticketId 
              ? { 
                  ...ticket, 
                  attachments: [...(ticket.attachments || []), attachment],
                  updated_at: new Date().toISOString()
                }
              : ticket
          );
          
          const updatedCurrentTicket = state.currentTicket?.id === ticketId 
            ? { 
                ...state.currentTicket, 
                attachments: [...(state.currentTicket.attachments || []), attachment],
                updated_at: new Date().toISOString()
              }
            : state.currentTicket;
          
          return {
            tickets: newTickets,
            currentTicket: updatedCurrentTicket,
          };
        }, false, "addAttachment"),
      
      // Loading actions
      setLoading: (loading) => 
        set({ loading }, false, "setLoading"),
      
      setRefreshing: (refreshing) => 
        set({ refreshing }, false, "setRefreshing"),
      
      setError: (error) => 
        set({ error }, false, "setError"),
      
      // Cache actions
      updateCacheTime: () => 
        set({ lastFetchTime: Date.now() }, false, "updateCacheTime"),
      
      isCacheValid: () => {
        const { lastFetchTime, cacheValidMs } = get();
        return Date.now() - lastFetchTime < cacheValidMs;
      },
      
      clearCache: () => 
        set({ 
          tickets: [], 
          stats: initialDataState.stats, 
          lastFetchTime: 0 
        }, false, "clearCache"),
      
      // Reset actions
      resetData: () => 
        set({ ...initialDataState }, false, "resetData"),
      
      clearError: () => 
        set({ error: null }, false, "clearError"),
    }),
    {
      name: "ticket-data-store",
    }
  )
);

// Helper function to calculate stats
function calculateStats(tickets: Ticket[]): TicketStats {
  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const thisWeek = new Date(today.getTime() - today.getDay() * 24 * 60 * 60 * 1000);
  const thisMonth = new Date(now.getFullYear(), now.getMonth(), 1);

  return {
    total: tickets.length,
    open: tickets.filter(t => t.status !== 'resolved' && t.status !== 'closed').length,
    resolved: tickets.filter(t => t.status === 'resolved' || t.status === 'closed').length,
    highPriority: tickets.filter(t => t.priority === 'high' || t.priority === 'critical').length,
    overdue: tickets.filter(t => {
      if (t.due_date) return new Date(t.due_date) < now;
      return false;
    }).length,
    dueToday: tickets.filter(t => {
      if (t.due_date) {
        const dueDate = new Date(t.due_date);
        return dueDate >= today && dueDate < new Date(today.getTime() + 24 * 60 * 60 * 1000);
      }
      return false;
    }).length,
    thisWeek: tickets.filter(t => {
      const createdDate = new Date(t.created_at);
      return createdDate >= thisWeek;
    }).length,
    thisMonth: tickets.filter(t => {
      const createdDate = new Date(t.created_at);
      return createdDate >= thisMonth;
    }).length,
  };
}

// Selectors for optimized re-renders
export const useTicketsList = () => useTicketDataStore((state) => ({
  tickets: state.tickets,
  loading: state.loading,
  refreshing: state.refreshing,
  error: state.error,
  setTickets: state.setTickets,
  setLoading: state.setLoading,
  setRefreshing: state.setRefreshing,
  setError: state.setError,
  clearError: state.clearError,
}));

export const useTicketStats = () => useTicketDataStore((state) => ({
  stats: state.stats,
  setStats: state.setStats,
}));

export const useCurrentTicket = () => useTicketDataStore((state) => ({
  currentTicket: state.currentTicket,
  setCurrentTicket: state.setCurrentTicket,
}));

export const useTicketCRUD = () => useTicketDataStore((state) => ({
  addTicket: state.addTicket,
  updateTicket: state.updateTicket,
  removeTicket: state.removeTicket,
  removeTickets: state.removeTickets,
  addComment: state.addComment,
  addAttachment: state.addAttachment,
}));

export const useTicketCache = () => useTicketDataStore((state) => ({
  lastFetchTime: state.lastFetchTime,
  isCacheValid: state.isCacheValid,
  updateCacheTime: state.updateCacheTime,
  clearCache: state.clearCache,
  resetData: state.resetData,
}));