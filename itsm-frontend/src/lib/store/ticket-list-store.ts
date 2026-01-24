/**
 * 工单列表Store - 兼容层
 *
 * @deprecated 请使用 @/lib/hooks/useTickets 替代
 *
 * 此文件提供向后兼容的store接口，用于现有组件。
 * 新代码应直接使用 useTickets hook。
 */

"use client";

import { useCallback } from "react";
import { useTickets, Ticket, TicketQueryFilters } from "@/lib/hooks/useTickets";

// 简化的Store接口（与原useTicketListStore兼容）
interface UseTicketListStore {
  tickets: Ticket[];
  loading: boolean;
  error: string | null;
  pagination: {
    current: number;
    pageSize: number;
    total: number;
  };
  filters: Partial<TicketQueryFilters>;
  selectedTickets: Set<string>;
  // Actions
  fetchTickets: (params?: Record<string, unknown>) => Promise<void>;
  updateTicket: (id: string, data: Partial<Ticket>) => Promise<void>;
  deleteTicket: (id: string) => Promise<void>;
  selectTicket: (id: string) => void;
  deselectTicket: (id: string) => void;
  deselectAll: () => void;
  batchDeleteTickets: (ids: string[]) => Promise<void>;
  setPage: (page: number) => void;
  setPageSize: (pageSize: number) => void;
  setFilters: (filters: Partial<TicketQueryFilters>) => void;
}

export function useTicketListStore(): UseTicketListStore {
  const {
    tickets,
    loading,
    error,
    pagination,
    filters,
    fetchTickets,
    updateFilters,
    updatePagination,
  } = useTickets();

  const [selectedTickets, setSelectedTickets] = require("react").useState<Set<string>>(new Set());

  const selectTicket = useCallback((id: string) => {
    setSelectedTickets((prev) => new Set(prev).add(id));
  }, []);

  const deselectTicket = useCallback((id: string) => {
    setSelectedTickets((prev) => {
      const next = new Set(prev);
      next.delete(id);
      return next;
    });
  }, []);

  const deselectAll = useCallback(() => {
    setSelectedTickets(new Set());
  }, []);

  const updateTicket = useCallback(async (id: string, data: Partial<Ticket>) => {
    // TODO: 实现更新逻辑
    console.warn("updateTicket not fully implemented - use ticket-service directly");
  }, []);

  const deleteTicket = useCallback(async (id: string) => {
    // TODO: 实现删除逻辑
    console.warn("deleteTicket not fully implemented - use ticket-service directly");
  }, []);

  const batchDeleteTickets = useCallback(async (ids: string[]) => {
    // TODO: 实现批量删除逻辑
    console.warn("batchDeleteTickets not fully implemented - use ticket-service directly");
  }, []);

  const setPage = useCallback((page: number) => {
    updatePagination({ page });
  }, [updatePagination]);

  const setPageSize = useCallback((pageSize: number) => {
    updatePagination({ pageSize });
  }, [updatePagination]);

  const setFilters = useCallback((newFilters: Partial<TicketQueryFilters>) => {
    updateFilters(newFilters);
  }, [updateFilters]);

  return {
    tickets,
    loading,
    error,
    pagination,
    filters,
    selectedTickets,
    fetchTickets,
    updateTicket,
    deleteTicket,
    selectTicket,
    deselectTicket,
    deselectAll,
    batchDeleteTickets,
    setPage,
    setPageSize,
    setFilters,
  };
}

export default useTicketListStore;
