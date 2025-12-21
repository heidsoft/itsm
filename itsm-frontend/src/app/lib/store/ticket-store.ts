"use client";

// Re-export from specialized stores for backward compatibility and convenience
export * from "./ticket-data-store";
export * from "./ticket-ui-store";
export * from "./ticket-filter-store";

// Convenience hooks that combine multiple stores for common use cases
import { useTicketsData, useTicketStats, useTicketCRUD } from "./ticket-data-store";
import { useTicketSelection, useTicketModals, useTicketView, useTicketLoading } from "./ticket-ui-store";
import { useTicketFilters, useTicketPagination, useTicketAdvancedFilters } from "./ticket-filter-store";

// Combined hooks for common patterns
export const useTicketManagement = () => {
  const data = useTicketsData();
  const stats = useTicketStats();
  const crud = useTicketCRUD();
  const selection = useTicketSelection();
  const modals = useTicketModals();
  const view = useTicketView();
  const loading = useTicketLoading();
  const filters = useTicketFilters();
  const pagination = useTicketPagination();
  const advancedFilters = useTicketAdvancedFilters();

  return {
    // Data
    tickets: data.tickets,
    stats: stats.stats,
    loading: data.loading || loading.actionLoading,
    refreshing: data.refreshing,
    error: data.error,
    
    // UI state
    selectedRowKeys: selection.selectedRowKeys,
    modalVisible: modals.modalVisible,
    editingTicket: modals.editingTicket,
    viewMode: view.viewMode,
    sidebarVisible: view.sidebarVisible,
    
    // Filters and pagination
    filters: filters.filters,
    searchQuery: filters.searchQuery,
    sortBy: filters.sortBy,
    sortOrder: filters.sortOrder,
    pagination: pagination.pagination,
    advancedFiltersVisible: advancedFilters.advancedFiltersVisible,
    savedFilters: advancedFilters.savedFilters,
    
    // Actions
    setTickets: data.setTickets,
    setStats: stats.setStats,
    setLoading: data.setLoading,
    setError: data.setError,
    
    // CRUD
    addTicket: crud.addTicket,
    updateTicket: crud.updateTicket,
    removeTicket: crud.removeTicket,
    removeTickets: crud.removeTickets,
    addComment: crud.addComment,
    addAttachment: crud.addAttachment,
    
    // Selection
    setSelectedRowKeys: selection.setSelectedRowKeys,
    clearSelection: selection.clearSelection,
    
    // Modals
    setModalVisible: modals.setModalVisible,
    setEditingTicket: modals.setEditingTicket,
    
    // View
    setViewMode: view.setViewMode,
    setSidebarVisible: view.setSidebarVisible,
    
    // Filters
    setFilters: filters.setFilters,
    updateFilters: filters.updateFilters,
    addFilter: filters.addFilter,
    removeFilter: filters.removeFilter,
    clearFilters: filters.clearFilters,
    setSearchQuery: filters.setSearchQuery,
    setSort: filters.setSort,
    toggleSort: filters.toggleSort,
    
    // Pagination
    setPagination: pagination.setPagination,
    updatePagination: pagination.updatePagination,
    setTotal: pagination.setTotal,
    
    // Advanced filters
    toggleAdvancedFilters: advancedFilters.toggleAdvancedFilters,
    saveFilter: advancedFilters.saveFilter,
    applySavedFilter: advancedFilters.applySavedFilter,
    deleteSavedFilter: advancedFilters.deleteSavedFilter,
    
    // Utility
    clearError: data.clearError,
    resetData: crud.resetData,
  };
};

// Legacy exports for backward compatibility
export const useTicketStore = useTicketManagement;
