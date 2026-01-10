"use client";

import { useState, useCallback, useEffect } from "react";
import { message } from "antd";
import { ticketService, Ticket, TicketStatus, TicketPriority, TicketType } from "../../lib/services/ticket-service";

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
  fetchTickets: () => Promise<void>;
  fetchStats: () => Promise<void>;
  refreshData: () => Promise<void>;
  updateFilters: (newFilters: Partial<TicketQueryFilters>) => void;
  updatePagination: (page: number, pageSize: number) => void;
  
  // Ticket operations
  createTicket: (ticketData: any) => Promise<void>;
  updateTicket: (id: number, ticketData: any) => Promise<void>;
  deleteTicket: (id: number) => Promise<void>;
  batchDeleteTickets: (ids: number[]) => Promise<void>;
}

export const useTickets = (): UseTicketsReturn => {
  // State
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

  // Fetch tickets
  const fetchTickets = useCallback(async () => {
    setLoading(true);
    setError(null);
    
    try {
      const response = await ticketService.listTickets({
        page: pagination.current,
        page_size: pagination.pageSize,
        status: filters.status,
        priority: filters.priority,
        type: filters.type,
        category: filters.category,
        assignee_id: filters.assignee_id,
        keyword: filters.keyword,
      });
      
      setTickets(response.tickets);
      setPagination(prev => ({ ...prev, total: response.total }));
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : "Failed to load tickets";
      setError(errorMessage);
      message.error(errorMessage);
      console.error("Failed to load tickets:", err);
    } finally {
      setLoading(false);
    }
  }, [pagination.current, pagination.pageSize, filters]);

  // Fetch statistics
  const fetchStats = useCallback(async () => {
    try {
      const response = await ticketService.getTicketStats();
      setStats({
        total: response.total,
        open: response.open,
        resolved: response.resolved,
        highPriority: response.high_priority,
      });
    } catch (err) {
      console.error("Failed to load statistics:", err);
    }
  }, []);

  // Refresh all data
  const refreshData = useCallback(async () => {
    await Promise.all([fetchTickets(), fetchStats()]);
  }, [fetchTickets, fetchStats]);

  // Update filters
  const updateFilters = useCallback((newFilters: Partial<TicketQueryFilters>) => {
    setFilters(prev => ({ ...prev, ...newFilters }));
    setPagination(prev => ({ ...prev, current: 1 }));
  }, []);

  // Update pagination
  const updatePagination = useCallback((page: number, pageSize: number) => {
    setPagination(prev => ({ ...prev, current: page, pageSize }));
  }, []);

  // Create ticket
  const createTicket = useCallback(async (ticketData: any) => {
    try {
      await ticketService.createTicket(ticketData);
      message.success("Ticket created successfully");
      await refreshData();
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : "Failed to create ticket";
      message.error(errorMessage);
      throw err;
    }
  }, [refreshData]);

  // Update ticket
  const updateTicket = useCallback(async (id: number, ticketData: any) => {
    try {
      await ticketService.updateTicket(id, ticketData);
      message.success("Ticket updated successfully");
      await refreshData();
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : "Failed to update ticket";
      message.error(errorMessage);
      throw err;
    }
  }, [refreshData]);

  // Delete ticket
  const deleteTicket = useCallback(async (id: number) => {
    try {
      await ticketService.deleteTicket(id);
      message.success("Ticket deleted successfully");
      await refreshData();
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : "Failed to delete ticket";
      message.error(errorMessage);
      throw err;
    }
  }, [refreshData]);

  // Batch delete tickets
  const batchDeleteTickets = useCallback(async (ids: number[]) => {
    try {
      await Promise.all(ids.map(id => ticketService.deleteTicket(id)));
      message.success(`${ids.length} tickets deleted successfully`);
      await refreshData();
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : "Failed to delete tickets";
      message.error(errorMessage);
      throw err;
    }
  }, [refreshData]);

  // Initial data fetch
  useEffect(() => {
    refreshData();
  }, [refreshData]);

  // Refetch when filters or pagination change
  useEffect(() => {
    fetchTickets();
  }, [fetchTickets]);

  return {
    // Data
    tickets,
    stats,
    loading,
    error,
    
    // Pagination
    pagination,
    
    // Filters
    filters,
    
    // Actions
    fetchTickets,
    fetchStats,
    refreshData,
    updateFilters,
    updatePagination,
    
    // Ticket operations
    createTicket,
    updateTicket,
    deleteTicket,
    batchDeleteTickets,
  };
};
