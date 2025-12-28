import { create } from 'zustand';
import { devtools, persist } from 'zustand/middleware';
import { immer } from 'zustand/middleware/immer';
import { ticketApi } from '../api/ticket';
import type { 
  Ticket, 
  TicketSummary, 
  SearchTicketsParams, 
  CreateTicketRequest, 
  AssignTicketRequest,
  UpdateStatusRequest,
  AddCommentRequest,
  TicketStatus,
  TicketPriority 
} from '../api/ticket';

interface TicketFilter {
  status?: TicketStatus[];
  priority?: TicketPriority[];
  assignedTo?: string;
  category?: string;
  keywords?: string;
  dateRange?: {
    from: string;
    to: string;
  };
}

interface PaginationState {
  page: number;
  pageSize: number;
  totalCount: number;
  hasNext: boolean;
  hasPrevious: boolean;
}

interface LoadingState {
  tickets: boolean;
  ticketDetails: Record<string, boolean>;
  creating: boolean;
  updating: Record<string, boolean>;
  deleting: Record<string, boolean>;
  bulkOperations: boolean;
}

interface ErrorState {
  tickets: string | null;
  ticketDetails: Record<string, string>;
  creating: string | null;
  updating: Record<string, string>;
  deleting: Record<string, string>;
  bulkOperations: string | null;
}

interface OptimisticUpdate {
  id: string;
  type: 'create' | 'update' | 'delete';
  data: any;
  timestamp: number;
}

interface ModernTicketState {
  // Data
  tickets: TicketSummary[];
  ticketDetails: Record<string, Ticket>;
  
  // UI State
  filters: TicketFilter;
  sorting: {
    sortBy: 'created_at' | 'updated_at' | 'priority' | 'status';
    sortOrder: 'asc' | 'desc';
  };
  pagination: PaginationState;
  
  // Loading & Error States
  loading: LoadingState;
  errors: ErrorState;
  
  // Optimistic Updates
  optimisticUpdates: OptimisticUpdate[];
  
  // Selected Items
  selectedTickets: string[];
  
  // Actions
  fetchTickets: () => Promise<void>;
  fetchTicketDetails: (id: string) => Promise<void>;
  createTicket: (data: CreateTicketRequest) => Promise<string | null>;
  updateTicket: (id: string, data: Partial<CreateTicketRequest>) => Promise<void>;
  assignTicket: (id: string, data: AssignTicketRequest) => Promise<void>;
  updateTicketStatus: (id: string, data: UpdateStatusRequest) => Promise<void>;
  addComment: (id: string, data: AddCommentRequest) => Promise<void>;
  deleteTicket: (id: string) => Promise<void>;
  
  // Bulk Operations
  bulkAssign: (ticketIds: string[], assignedTo: string) => Promise<void>;
  bulkUpdateStatus: (ticketIds: string[], status: TicketStatus) => Promise<void>;
  bulkDelete: (ticketIds: string[]) => Promise<void>;
  
  // Filters & Search
  setFilters: (filters: Partial<TicketFilter>) => void;
  clearFilters: () => void;
  setSorting: (sortBy: string, sortOrder?: 'asc' | 'desc') => void;
  setPage: (page: number) => void;
  setPageSize: (pageSize: number) => void;
  
  // Selection
  selectTicket: (id: string) => void;
  deselectTicket: (id: string) => void;
  selectAllTickets: () => void;
  clearSelection: () => void;
  
  // Optimistic Updates
  addOptimisticUpdate: (update: Omit<OptimisticUpdate, 'timestamp'>) => void;
  removeOptimisticUpdate: (id: string) => void;
  clearOptimisticUpdates: () => void;
  
  // Reset
  reset: () => void;
}

const initialState = {
  tickets: [],
  ticketDetails: {},
  filters: {},
  sorting: {
    sortBy: 'created_at' as const,
    sortOrder: 'desc' as const,
  },
  pagination: {
    page: 1,
    pageSize: 20,
    totalCount: 0,
    hasNext: false,
    hasPrevious: false,
  },
  loading: {
    tickets: false,
    ticketDetails: {},
    creating: false,
    updating: {},
    deleting: {},
    bulkOperations: false,
  },
  errors: {
    tickets: null,
    ticketDetails: {},
    creating: null,
    updating: {},
    deleting: {},
    bulkOperations: null,
  },
  optimisticUpdates: [],
  selectedTickets: [],
};

export const useModernTicketStore = create<ModernTicketState>()(
  devtools(
    persist(
      immer((set, get) => ({
        ...initialState,

        fetchTickets: async () => {
          set((state) => {
            state.loading.tickets = true;
            state.errors.tickets = null;
          });

          try {
            const { filters, sorting, pagination } = get();
            
            const params: SearchTicketsParams = {
              ...filters,
              page: pagination.page,
              page_size: pagination.pageSize,
              sort_by: sorting.sortBy,
              sort_order: sorting.sortOrder,
            };

            const result = await ticketApi.searchTickets(params);
            
            set((state) => {
              state.tickets = result.tickets;
              state.pagination.totalCount = result.total_count;
              state.pagination.hasNext = result.page * result.page_size < result.total_count;
              state.pagination.hasPrevious = result.page > 1;
              state.loading.tickets = false;
            });
          } catch (error) {
            set((state) => {
              state.loading.tickets = false;
              state.errors.tickets = error instanceof Error ? error.message : 'Failed to fetch tickets';
            });
          }
        },

        fetchTicketDetails: async (id: string) => {
          set((state) => {
            state.loading.ticketDetails[id] = true;
            delete state.errors.ticketDetails[id];
          });

          try {
            const ticket = await ticketApi.getTicket(id);
            
            set((state) => {
              state.ticketDetails[id] = ticket;
              state.loading.ticketDetails[id] = false;
            });
          } catch (error) {
            set((state) => {
              state.loading.ticketDetails[id] = false;
              state.errors.ticketDetails[id] = error instanceof Error ? error.message : 'Failed to fetch ticket details';
            });
          }
        },

        createTicket: async (data: CreateTicketRequest) => {
          set((state) => {
            state.loading.creating = true;
            state.errors.creating = null;
          });

          // Optimistic update
          const optimisticTicket: TicketSummary = {
            id: `temp-${Date.now()}`,
            ticket_number: 'TEMP-000',
            title: data.title,
            status: 'new',
            priority: data.priority,
            category: data.category,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
          };

          set((state) => {
            state.tickets.unshift(optimisticTicket);
            state.optimisticUpdates.push({
              id: optimisticTicket.id,
              type: 'create',
              data: optimisticTicket,
              timestamp: Date.now(),
            });
          });

          try {
            const result = await ticketApi.createTicket(data);
            
            set((state) => {
              // Remove optimistic update
              state.tickets = state.tickets.filter(t => t.id !== optimisticTicket.id);
              state.optimisticUpdates = state.optimisticUpdates.filter(u => u.id !== optimisticTicket.id);
              
              // Add real ticket
              const newTicket: TicketSummary = {
                id: result.id,
                ticket_number: result.ticket_number,
                title: data.title,
                status: result.status as TicketStatus,
                priority: data.priority,
                category: data.category,
                created_at: result.created_at,
                updated_at: result.created_at,
              };
              state.tickets.unshift(newTicket);
              state.loading.creating = false;
            });

            return result.id;
          } catch (error) {
            set((state) => {
              // Remove optimistic update on error
              state.tickets = state.tickets.filter(t => t.id !== optimisticTicket.id);
              state.optimisticUpdates = state.optimisticUpdates.filter(u => u.id !== optimisticTicket.id);
              state.loading.creating = false;
              state.errors.creating = error instanceof Error ? error.message : 'Failed to create ticket';
            });
            return null;
          }
        },

        assignTicket: async (id: string, data: AssignTicketRequest) => {
          set((state) => {
            state.loading.updating[id] = true;
            delete state.errors.updating[id];
          });

          // Optimistic update
          set((state) => {
            const ticket = state.tickets.find(t => t.id === id);
            if (ticket) {
              ticket.assigned_to = data.assigned_to;
            }
            
            state.optimisticUpdates.push({
              id,
              type: 'update',
              data: { assigned_to: data.assigned_to },
              timestamp: Date.now(),
            });
          });

          try {
            await ticketApi.assignTicket(id, data);
            
            set((state) => {
              state.loading.updating[id] = false;
              state.optimisticUpdates = state.optimisticUpdates.filter(u => u.id !== id);
            });
          } catch (error) {
            set((state) => {
              // Revert optimistic update
              const ticket = state.tickets.find(t => t.id === id);
              if (ticket) {
                delete ticket.assigned_to;
              }
              
              state.optimisticUpdates = state.optimisticUpdates.filter(u => u.id !== id);
              state.loading.updating[id] = false;
              state.errors.updating[id] = error instanceof Error ? error.message : 'Failed to assign ticket';
            });
          }
        },

        updateTicketStatus: async (id: string, data: UpdateStatusRequest) => {
          set((state) => {
            state.loading.updating[id] = true;
            delete state.errors.updating[id];
          });

          // Optimistic update
          const oldStatus = get().tickets.find(t => t.id === id)?.status;
          set((state) => {
            const ticket = state.tickets.find(t => t.id === id);
            if (ticket) {
              ticket.status = data.status;
              ticket.updated_at = new Date().toISOString();
            }
            
            state.optimisticUpdates.push({
              id,
              type: 'update',
              data: { status: data.status, updated_at: ticket?.updated_at },
              timestamp: Date.now(),
            });
          });

          try {
            await ticketApi.updateStatus(id, data);
            
            set((state) => {
              state.loading.updating[id] = false;
              state.optimisticUpdates = state.optimisticUpdates.filter(u => u.id !== id);
            });
          } catch (error) {
            set((state) => {
              // Revert optimistic update
              const ticket = state.tickets.find(t => t.id === id);
              if (ticket && oldStatus) {
                ticket.status = oldStatus;
              }
              
              state.optimisticUpdates = state.optimisticUpdates.filter(u => u.id !== id);
              state.loading.updating[id] = false;
              state.errors.updating[id] = error instanceof Error ? error.message : 'Failed to update ticket status';
            });
          }
        },

        addComment: async (id: string, data: AddCommentRequest) => {
          set((state) => {
            state.loading.updating[id] = true;
            delete state.errors.updating[id];
          });

          try {
            await ticketApi.addComment(id, data);
            
            // Refresh ticket details if loaded
            if (get().ticketDetails[id]) {
              await get().fetchTicketDetails(id);
            }
            
            set((state) => {
              state.loading.updating[id] = false;
            });
          } catch (error) {
            set((state) => {
              state.loading.updating[id] = false;
              state.errors.updating[id] = error instanceof Error ? error.message : 'Failed to add comment';
            });
          }
        },

        deleteTicket: async (id: string) => {
          set((state) => {
            state.loading.deleting[id] = true;
            delete state.errors.deleting[id];
          });

          // Optimistic update
          const ticketIndex = get().tickets.findIndex(t => t.id === id);
          const ticket = get().tickets[ticketIndex];
          
          set((state) => {
            state.tickets = state.tickets.filter(t => t.id !== id);
            delete state.ticketDetails[id];
            
            state.optimisticUpdates.push({
              id,
              type: 'delete',
              data: ticket,
              timestamp: Date.now(),
            });
          });

          try {
            await ticketApi.deleteTicket(id);
            
            set((state) => {
              state.loading.deleting[id] = false;
              state.optimisticUpdates = state.optimisticUpdates.filter(u => u.id !== id);
            });
          } catch (error) {
            set((state) => {
              // Restore ticket on error
              if (ticket) {
                state.tickets.splice(ticketIndex, 0, ticket);
              }
              
              state.optimisticUpdates = state.optimisticUpdates.filter(u => u.id !== id);
              state.loading.deleting[id] = false;
              state.errors.deleting[id] = error instanceof Error ? error.message : 'Failed to delete ticket';
            });
          }
        },

        bulkAssign: async (ticketIds: string[], assignedTo: string) => {
          set((state) => {
            state.loading.bulkOperations = true;
            state.errors.bulkOperations = null;
          });

          try {
            await ticketApi.bulkAssign(ticketIds, assignedTo);
            
            set((state) => {
              ticketIds.forEach(id => {
                const ticket = state.tickets.find(t => t.id === id);
                if (ticket) {
                  ticket.assigned_to = assignedTo;
                }
              });
              state.loading.bulkOperations = false;
            });
          } catch (error) {
            set((state) => {
              state.loading.bulkOperations = false;
              state.errors.bulkOperations = error instanceof Error ? error.message : 'Failed to bulk assign tickets';
            });
          }
        },

        bulkUpdateStatus: async (ticketIds: string[], status: TicketStatus) => {
          set((state) => {
            state.loading.bulkOperations = true;
            state.errors.bulkOperations = null;
          });

          try {
            await ticketApi.bulkUpdateStatus(ticketIds, status);
            
            set((state) => {
              ticketIds.forEach(id => {
                const ticket = state.tickets.find(t => t.id === id);
                if (ticket) {
                  ticket.status = status;
                  ticket.updated_at = new Date().toISOString();
                }
              });
              state.loading.bulkOperations = false;
            });
          } catch (error) {
            set((state) => {
              state.loading.bulkOperations = false;
              state.errors.bulkOperations = error instanceof Error ? error.message : 'Failed to bulk update ticket status';
            });
          }
        },

        bulkDelete: async (ticketIds: string[]) => {
          set((state) => {
            state.loading.bulkOperations = true;
            state.errors.bulkOperations = null;
          });

          try {
            await ticketApi.bulkDelete(ticketIds);
            
            set((state) => {
              state.tickets = state.tickets.filter(t => !ticketIds.includes(t.id));
              ticketIds.forEach(id => {
                delete state.ticketDetails[id];
              });
              state.selectedTickets = state.selectedTickets.filter(id => !ticketIds.includes(id));
              state.loading.bulkOperations = false;
            });
          } catch (error) {
            set((state) => {
              state.loading.bulkOperations = false;
              state.errors.bulkOperations = error instanceof Error ? error.message : 'Failed to bulk delete tickets';
            });
          }
        },

        setFilters: (newFilters: Partial<TicketFilter>) => {
          set((state) => {
            Object.assign(state.filters, newFilters);
            state.pagination.page = 1; // Reset to first page
          });
          get().fetchTickets();
        },

        clearFilters: () => {
          set((state) => {
            state.filters = {};
            state.pagination.page = 1;
          });
          get().fetchTickets();
        },

        setSorting: (sortBy: string, sortOrder?: 'asc' | 'desc') => {
          set((state) => {
            state.sorting.sortBy = sortBy as any;
            state.sorting.sortOrder = sortOrder || (state.sorting.sortBy === sortBy && state.sorting.sortOrder === 'asc' ? 'desc' : 'asc');
            state.pagination.page = 1;
          });
          get().fetchTickets();
        },

        setPage: (page: number) => {
          set((state) => {
            state.pagination.page = page;
          });
          get().fetchTickets();
        },

        setPageSize: (pageSize: number) => {
          set((state) => {
            state.pagination.pageSize = pageSize;
            state.pagination.page = 1;
          });
          get().fetchTickets();
        },

        selectTicket: (id: string) => {
          set((state) => {
            if (!state.selectedTickets.includes(id)) {
              state.selectedTickets.push(id);
            }
          });
        },

        deselectTicket: (id: string) => {
          set((state) => {
            state.selectedTickets = state.selectedTickets.filter(ticketId => ticketId !== id);
          });
        },

        selectAllTickets: () => {
          set((state) => {
            state.selectedTickets = state.tickets.map(ticket => ticket.id);
          });
        },

        clearSelection: () => {
          set((state) => {
            state.selectedTickets = [];
          });
        },

        addOptimisticUpdate: (update: Omit<OptimisticUpdate, 'timestamp'>) => {
          set((state) => {
            state.optimisticUpdates.push({
              ...update,
              timestamp: Date.now(),
            });
          });
        },

        removeOptimisticUpdate: (id: string) => {
          set((state) => {
            state.optimisticUpdates = state.optimisticUpdates.filter(update => update.id !== id);
          });
        },

        clearOptimisticUpdates: () => {
          set((state) => {
            state.optimisticUpdates = [];
          });
        },

        updateTicket: async (id: string, data: Partial<CreateTicketRequest>) => {
          set((state) => {
            state.loading.updating[id] = true;
            delete state.errors.updating[id];
          });

          // Store original data for rollback
          const originalTicket = get().tickets.find(t => t.id === id);
          
          // Optimistic update
          set((state) => {
            const ticket = state.tickets.find(t => t.id === id);
            if (ticket) {
              if (data.title) ticket.title = data.title;
              if (data.priority) ticket.priority = data.priority;
              if (data.category) ticket.category = data.category;
              ticket.updated_at = new Date().toISOString();
            }
          });

          try {
            // Note: This would need a corresponding backend endpoint
            // For now, we'll simulate success
            await new Promise(resolve => setTimeout(resolve, 500));
            
            set((state) => {
              state.loading.updating[id] = false;
            });
          } catch (error) {
            // Rollback optimistic update
            set((state) => {
              if (originalTicket) {
                const ticketIndex = state.tickets.findIndex(t => t.id === id);
                if (ticketIndex !== -1) {
                  state.tickets[ticketIndex] = originalTicket;
                }
              }
              state.loading.updating[id] = false;
              state.errors.updating[id] = error instanceof Error ? error.message : 'Failed to update ticket';
            });
          }
        },

        reset: () => {
          set(initialState);
        },
      })),
      {
        name: 'modern-ticket-store',
        partialize: (state) => ({
          filters: state.filters,
          sorting: state.sorting,
          pagination: { ...state.pagination, page: 1 }, // Reset page on reload
        }),
      }
    ),
    {
      name: 'modern-ticket-store',
    }
  )
);

// Selectors for derived state
export const useModernTicketSelectors = () => {
  const store = useModernTicketStore();
  
  return {
    // Get filtered tickets with optimistic updates applied
    getVisibleTickets: () => {
      const { tickets, optimisticUpdates } = store;
      let visibleTickets = [...tickets];
      
      // Apply optimistic updates
      optimisticUpdates.forEach(update => {
        switch (update.type) {
          case 'create':
            if (!visibleTickets.find(t => t.id === update.id)) {
              visibleTickets.unshift(update.data);
            }
            break;
          case 'update':
            const ticketIndex = visibleTickets.findIndex(t => t.id === update.id);
            if (ticketIndex !== -1) {
              visibleTickets[ticketIndex] = { ...visibleTickets[ticketIndex], ...update.data };
            }
            break;
          case 'delete':
            visibleTickets = visibleTickets.filter(t => t.id !== update.id);
            break;
        }
      });
      
      return visibleTickets;
    },
    
    // Get loading state for specific ticket
    isTicketLoading: (id: string) => {
      const { loading } = store;
      return loading.updating[id] || loading.deleting[id] || false;
    },
    
    // Get error state for specific ticket
    getTicketError: (id: string) => {
      const { errors } = store;
      return errors.updating[id] || errors.deleting[id] || null;
    },
    
    // Check if ticket is selected
    isTicketSelected: (id: string) => {
      return store.selectedTickets.includes(id);
    },
    
    // Get selection stats
    getSelectionStats: () => {
      const { selectedTickets, tickets } = store;
      return {
        selectedCount: selectedTickets.length,
        totalCount: tickets.length,
        isAllSelected: selectedTickets.length === tickets.length && tickets.length > 0,
        hasSelection: selectedTickets.length > 0,
      };
    },
    
    // Get active filters count
    getActiveFiltersCount: () => {
      const { filters } = store;
      return Object.values(filters).filter(value => 
        value !== undefined && value !== null && value !== '' && 
        (!Array.isArray(value) || value.length > 0)
      ).length;
    },
    
    // Check if there are any optimistic updates
    hasOptimisticUpdates: () => {
      return store.optimisticUpdates.length > 0;
    },
  };
};

export default useModernTicketStore;