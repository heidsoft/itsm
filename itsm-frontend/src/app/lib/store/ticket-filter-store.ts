"use client";

import { create } from "zustand";
import { devtools, persist } from "zustand/middleware";
import type { TicketFilters } from "./ticket-ui-store";
import { PaginationState } from "./ticket-ui-store";

interface TicketFilterState {
  filters: TicketFilters;
  pagination: PaginationState;
  sortBy: string;
  sortOrder: "asc" | "desc";
  searchQuery: string;
  advancedFiltersVisible: boolean;
  savedFilters: Array<{ id: string; name: string; filters: TicketFilters }>;
}

interface TicketFilterActions {
  // Filter actions
  setFilters: (filters: TicketFilters) => void;
  updateFilters: (filters: Partial<TicketFilters>) => void;
  addFilter: (key: keyof TicketFilters, value: any) => void;
  removeFilter: (key: keyof TicketFilters) => void;
  clearFilters: () => void;
  
  // Pagination actions
  setPagination: (pagination: Partial<PaginationState>) => void;
  updatePagination: (current: number, pageSize?: number) => void;
  setTotal: (total: number) => void;
  
  // Sort actions
  setSort: (sortBy: string, sortOrder: "asc" | "desc") => void;
  toggleSort: (field: string) => void;
  
  // Search actions
  setSearchQuery: (query: string) => void;
  clearSearch: () => void;
  
  // UI actions
  toggleAdvancedFilters: () => void;
  setAdvancedFiltersVisible: (visible: boolean) => void;
  
  // Saved filters actions
  saveFilter: (name: string) => void;
  applySavedFilter: (id: string) => void;
  deleteSavedFilter: (id: string) => void;
  
  // Reset actions
  resetAll: () => void;
}

type TicketFilterStore = TicketFilterState & TicketFilterActions;

const initialFilterState: TicketFilterState = {
  filters: {},
  pagination: {
    current: 1,
    pageSize: 20,
    total: 0,
  },
  sortBy: "created_at",
  sortOrder: "desc",
  searchQuery: "",
  advancedFiltersVisible: false,
  savedFilters: [],
};

export const useTicketFilterStore = create<TicketFilterStore>()(
  devtools(
    persist(
      (set, get) => ({
        ...initialFilterState,
        
        // Filter actions
        setFilters: (filters) => 
          set({ filters, pagination: { ...get().pagination, current: 1 } }, false, "setFilters"),
        
        updateFilters: (newFilters) => 
          set((state) => ({
            filters: { ...state.filters, ...newFilters },
            pagination: { ...state.pagination, current: 1 },
          }), false, "updateFilters"),
        
        addFilter: (key, value) => 
          set((state) => ({
            filters: { ...state.filters, [key]: value },
            pagination: { ...state.pagination, current: 1 },
          }), false, "addFilter"),
        
        removeFilter: (key) => 
          set((state) => {
            const newFilters = { ...state.filters };
            delete newFilters[key];
            return {
              filters: newFilters,
              pagination: { ...state.pagination, current: 1 },
            };
          }, false, "removeFilter"),
        
        clearFilters: () => 
          set({ 
            filters: {}, 
            pagination: { ...get().pagination, current: 1 } 
          }, false, "clearFilters"),
        
        // Pagination actions
        setPagination: (pagination) => 
          set((state) => ({ 
            pagination: { ...state.pagination, ...pagination } 
          }), false, "setPagination"),
        
        updatePagination: (current, pageSize) => 
          set((state) => ({ 
            pagination: { 
              ...state.pagination, 
              current, 
              ...(pageSize && { pageSize }) 
            } 
          }), false, "updatePagination"),
        
        setTotal: (total) => 
          set((state) => ({ 
            pagination: { ...state.pagination, total } 
          }), false, "setTotal"),
        
        // Sort actions
        setSort: (sortBy, sortOrder) => 
          set({ sortBy, sortOrder }, false, "setSort"),
        
        toggleSort: (field) => {
          const { sortBy, sortOrder } = get();
          const newOrder = sortBy === field && sortOrder === "desc" ? "asc" : "desc";
          set({ sortBy: field, sortOrder: newOrder }, false, "toggleSort");
        },
        
        // Search actions
        setSearchQuery: (searchQuery) => 
          set({ searchQuery, pagination: { ...get().pagination, current: 1 } }, false, "setSearchQuery"),
        
        clearSearch: () => 
          set({ searchQuery: "", pagination: { ...get().pagination, current: 1 } }, false, "clearSearch"),
        
        // UI actions
        toggleAdvancedFilters: () => 
          set((state) => ({ 
            advancedFiltersVisible: !state.advancedFiltersVisible 
          }), false, "toggleAdvancedFilters"),
        
        setAdvancedFiltersVisible: (advancedFiltersVisible) => 
          set({ advancedFiltersVisible }, false, "setAdvancedFiltersVisible"),
        
        // Saved filters actions
        saveFilter: (name) => {
          const { filters } = get();
          const id = Date.now().toString();
          const newFilter = { id, name, filters: { ...filters } };
          
          set((state) => ({
            savedFilters: [...state.savedFilters, newFilter],
          }), false, "saveFilter");
        },
        
        applySavedFilter: (id) => {
          const { savedFilters } = get();
          const savedFilter = savedFilters.find(filter => filter.id === id);
          
          if (savedFilter) {
            set({
              filters: { ...savedFilter.filters },
              pagination: { ...get().pagination, current: 1 },
            }, false, "applySavedFilter");
          }
        },
        
        deleteSavedFilter: (id) => 
          set((state) => ({
            savedFilters: state.savedFilters.filter(filter => filter.id !== id),
          }), false, "deleteSavedFilter"),
        
        // Reset actions
        resetAll: () => set(initialFilterState, false, "resetAll"),
      }),
      {
        name: "ticket-filter-store",
        partialize: (state) => ({
          filters: state.filters,
          pagination: state.pagination,
          sortBy: state.sortBy,
          sortOrder: state.sortOrder,
          searchQuery: state.searchQuery,
          savedFilters: state.savedFilters,
        }),
      }
    ),
    {
      name: "ticket-filter-store",
    }
  )
);

// Selectors for optimized re-renders
export const useTicketFilters = () => useTicketFilterStore((state) => ({
  filters: state.filters,
  sortBy: state.sortBy,
  sortOrder: state.sortOrder,
  searchQuery: state.searchQuery,
  setFilters: state.setFilters,
  updateFilters: state.updateFilters,
  addFilter: state.addFilter,
  removeFilter: state.removeFilter,
  clearFilters: state.clearFilters,
  setSort: state.setSort,
  toggleSort: state.toggleSort,
  setSearchQuery: state.setSearchQuery,
  clearSearch: state.clearSearch,
}));

export const useTicketPagination = () => useTicketFilterStore((state) => ({
  pagination: state.pagination,
  setPagination: state.setPagination,
  updatePagination: state.updatePagination,
  setTotal: state.setTotal,
}));

export const useTicketAdvancedFilters = () => useTicketFilterStore((state) => ({
  advancedFiltersVisible: state.advancedFiltersVisible,
  savedFilters: state.savedFilters,
  toggleAdvancedFilters: state.toggleAdvancedFilters,
  setAdvancedFiltersVisible: state.setAdvancedFiltersVisible,
  saveFilter: state.saveFilter,
  applySavedFilter: state.applySavedFilter,
  deleteSavedFilter: state.deleteSavedFilter,
}));