// Ticket API service with proper typing and error handling
import { api } from './client';
import type { PaginatedResponse } from './client';

// Ticket Types
export interface Ticket {
  id: string;
  ticket_number: string;
  title: string;
  description: string;
  status: TicketStatus;
  priority: TicketPriority;
  category: string;
  created_by: string;
  assignment?: TicketAssignment;
  comments: TicketComment[];
  attachments: TicketAttachment[];
  created_at: string;
  updated_at: string;
}

export interface TicketSummary {
  id: string;
  ticket_number: string;
  title: string;
  status: TicketStatus;
  priority: TicketPriority;
  category: string;
  assigned_to?: string;
  created_at: string;
  updated_at: string;
}

export interface TicketAssignment {
  assigned_to: string;
  assigned_by: string;
  assigned_at: string;
  team_id?: string;
  instructions: string;
}

export interface TicketComment {
  id: string;
  author_id: string;
  content: string;
  created_at: string;
  is_private: boolean;
}

export interface TicketAttachment {
  id: string;
  filename: string;
  file_size: number;
  mime_type: string;
  url: string;
  uploaded_by: string;
  uploaded_at: string;
}

// Enums
export type TicketStatus = 
  | 'new' 
  | 'open' 
  | 'in_progress' 
  | 'pending' 
  | 'resolved' 
  | 'closed' 
  | 'cancelled';

export type TicketPriority = 
  | 'low' 
  | 'normal' 
  | 'high' 
  | 'urgent' 
  | 'critical';

// Request Types
export interface CreateTicketRequest {
  title: string;
  description: string;
  priority: TicketPriority;
  category: string;
}

export interface CreateTicketResponse {
  id: string;
  ticket_number: string;
  status: TicketStatus;
  created_at: string;
}

export interface AssignTicketRequest {
  assigned_to: string;
  team_id?: string;
  instructions?: string;
}

export interface UpdateStatusRequest {
  status: TicketStatus;
  reason?: string;
}

export interface AddCommentRequest {
  content: string;
  is_private?: boolean;
}

export interface SearchTicketsParams {
  status?: TicketStatus[];
  priority?: TicketPriority[];
  assigned_to?: string;
  created_by?: string;
  category?: string;
  keywords?: string;
  date_from?: string;
  date_to?: string;
  page?: number;
  page_size?: number;
  sort_by?: 'created_at' | 'updated_at' | 'priority' | 'status';
  sort_order?: 'asc' | 'desc';
}

export interface SearchTicketsResponse {
  tickets: TicketSummary[];
  total_count: number;
  page: number;
  page_size: number;
}

// Ticket API Service Class
export class TicketApiService {
  private baseUrl = '/tickets';

  /**
   * Create a new ticket
   */
  async createTicket(request: CreateTicketRequest): Promise<CreateTicketResponse> {
    return api.post<CreateTicketResponse, CreateTicketRequest>(
      this.baseUrl,
      request
    );
  }

  /**
   * Get a ticket by ID
   */
  async getTicket(id: string): Promise<Ticket> {
    return api.get<Ticket>(`${this.baseUrl}/${id}`);
  }

  /**
   * Search tickets with filters and pagination
   */
  async searchTickets(params: SearchTicketsParams = {}): Promise<SearchTicketsResponse> {
    const searchParams = new URLSearchParams();
    
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null) {
        if (Array.isArray(value)) {
          value.forEach(v => searchParams.append(key, v));
        } else {
          searchParams.append(key, String(value));
        }
      }
    });

    const queryString = searchParams.toString();
    const url = queryString ? `${this.baseUrl}?${queryString}` : this.baseUrl;
    
    return api.get<SearchTicketsResponse>(url);
  }

  /**
   * Assign a ticket to a user or team
   */
  async assignTicket(id: string, request: AssignTicketRequest): Promise<void> {
    return api.put<void, AssignTicketRequest>(
      `${this.baseUrl}/${id}/assign`,
      request
    );
  }

  /**
   * Update ticket status
   */
  async updateStatus(id: string, request: UpdateStatusRequest): Promise<void> {
    return api.put<void, UpdateStatusRequest>(
      `${this.baseUrl}/${id}/status`,
      request
    );
  }

  /**
   * Add a comment to a ticket
   */
  async addComment(id: string, request: AddCommentRequest): Promise<void> {
    return api.post<void, AddCommentRequest>(
      `${this.baseUrl}/${id}/comments`,
      request
    );
  }

  /**
   * Upload an attachment to a ticket
   */
  async uploadAttachment(
    id: string, 
    file: File, 
    onProgress?: (progress: number) => void
  ): Promise<TicketAttachment> {
    return api.upload<TicketAttachment>(
      `${this.baseUrl}/${id}/attachments`,
      file,
      onProgress
    );
  }

  /**
   * Download an attachment
   */
  async downloadAttachment(
    ticketId: string, 
    attachmentId: string, 
    filename?: string,
    onProgress?: (progress: number) => void
  ): Promise<void> {
    return api.download(
      `${this.baseUrl}/${ticketId}/attachments/${attachmentId}/download`,
      filename,
      onProgress
    );
  }

  /**
   * Delete a ticket (soft delete)
   */
  async deleteTicket(id: string): Promise<void> {
    return api.delete<void>(`${this.baseUrl}/${id}`);
  }

  /**
   * Bulk operations
   */
  async bulkAssign(ticketIds: string[], assignedTo: string): Promise<void> {
    return api.post<void>(`${this.baseUrl}/bulk/assign`, {
      ticket_ids: ticketIds,
      assigned_to: assignedTo,
    });
  }

  async bulkUpdateStatus(ticketIds: string[], status: TicketStatus): Promise<void> {
    return api.post<void>(`${this.baseUrl}/bulk/status`, {
      ticket_ids: ticketIds,
      status,
    });
  }

  async bulkDelete(ticketIds: string[]): Promise<void> {
    return api.post<void>(`${this.baseUrl}/bulk/delete`, {
      ticket_ids: ticketIds,
    });
  }

  /**
   * Get ticket statistics
   */
  async getStatistics(filters?: Partial<SearchTicketsParams>): Promise<{
    total: number;
    by_status: Record<TicketStatus, number>;
    by_priority: Record<TicketPriority, number>;
    by_category: Record<string, number>;
    assigned_count: number;
    unassigned_count: number;
  }> {
    const searchParams = new URLSearchParams();
    
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          if (Array.isArray(value)) {
            value.forEach(v => searchParams.append(key, v));
          } else {
            searchParams.append(key, String(value));
          }
        }
      });
    }

    const queryString = searchParams.toString();
    const url = queryString ? `${this.baseUrl}/statistics?${queryString}` : `${this.baseUrl}/statistics`;
    
    return api.get(url);
  }
}

// Create singleton instance
export const ticketApi = new TicketApiService();

// Export convenience functions
export const ticketService = {
  // CRUD operations
  create: (request: CreateTicketRequest) => ticketApi.createTicket(request),
  get: (id: string) => ticketApi.getTicket(id),
  search: (params?: SearchTicketsParams) => ticketApi.searchTickets(params),
  delete: (id: string) => ticketApi.deleteTicket(id),
  
  // State changes
  assign: (id: string, request: AssignTicketRequest) => ticketApi.assignTicket(id, request),
  updateStatus: (id: string, request: UpdateStatusRequest) => ticketApi.updateStatus(id, request),
  
  // Content
  addComment: (id: string, request: AddCommentRequest) => ticketApi.addComment(id, request),
  uploadAttachment: (id: string, file: File, onProgress?: (progress: number) => void) => 
    ticketApi.uploadAttachment(id, file, onProgress),
  downloadAttachment: (ticketId: string, attachmentId: string, filename?: string) => 
    ticketApi.downloadAttachment(ticketId, attachmentId, filename),
  
  // Bulk operations
  bulkAssign: (ticketIds: string[], assignedTo: string) => ticketApi.bulkAssign(ticketIds, assignedTo),
  bulkUpdateStatus: (ticketIds: string[], status: TicketStatus) => ticketApi.bulkUpdateStatus(ticketIds, status),
  bulkDelete: (ticketIds: string[]) => ticketApi.bulkDelete(ticketIds),
  
  // Analytics
  getStatistics: (filters?: Partial<SearchTicketsParams>) => ticketApi.getStatistics(filters),
};

// Utility functions
export const ticketUtils = {
  /**
   * Format ticket number for display
   */
  formatTicketNumber: (ticket: Ticket | TicketSummary): string => {
    return `#${ticket.ticket_number}`;
  },

  /**
   * Get status color based on ticket status
   */
  getStatusColor: (status: TicketStatus): string => {
    const colors: Record<TicketStatus, string> = {
      new: 'blue',
      open: 'cyan',
      in_progress: 'orange',
      pending: 'yellow',
      resolved: 'green',
      closed: 'gray',
      cancelled: 'red',
    };
    return colors[status] || 'gray';
  },

  /**
   * Get priority color based on ticket priority
   */
  getPriorityColor: (priority: TicketPriority): string => {
    const colors: Record<TicketPriority, string> = {
      low: 'green',
      normal: 'blue',
      high: 'orange',
      urgent: 'red',
      critical: 'magenta',
    };
    return colors[priority] || 'blue';
  },

  /**
   * Check if status transition is valid
   */
  isValidStatusTransition: (from: TicketStatus, to: TicketStatus): boolean => {
    const validTransitions: Record<TicketStatus, TicketStatus[]> = {
      new: ['open', 'cancelled'],
      open: ['in_progress', 'pending', 'resolved', 'cancelled'],
      in_progress: ['pending', 'resolved', 'open', 'cancelled'],
      pending: ['in_progress', 'open', 'resolved', 'cancelled'],
      resolved: ['closed', 'open'],
      closed: [],
      cancelled: [],
    };
    
    return validTransitions[from]?.includes(to) ?? false;
  },

  /**
   * Get next available statuses for a ticket
   */
  getAvailableStatuses: (currentStatus: TicketStatus): TicketStatus[] => {
    const validTransitions: Record<TicketStatus, TicketStatus[]> = {
      new: ['open', 'cancelled'],
      open: ['in_progress', 'pending', 'resolved', 'cancelled'],
      in_progress: ['pending', 'resolved', 'open', 'cancelled'],
      pending: ['in_progress', 'open', 'resolved', 'cancelled'],
      resolved: ['closed', 'open'],
      closed: [],
      cancelled: [],
    };
    
    return validTransitions[currentStatus] || [];
  },

  /**
   * Calculate priority weight for sorting
   */
  getPriorityWeight: (priority: TicketPriority): number => {
    const weights: Record<TicketPriority, number> = {
      low: 1,
      normal: 2,
      high: 3,
      urgent: 4,
      critical: 5,
    };
    return weights[priority] || 2;
  },

  /**
   * Format relative time
   */
  formatRelativeTime: (date: string): string => {
    const now = new Date();
    const ticketDate = new Date(date);
    const diffInMinutes = Math.floor((now.getTime() - ticketDate.getTime()) / (1000 * 60));

    if (diffInMinutes < 1) return '刚刚';
    if (diffInMinutes < 60) return `${diffInMinutes}分钟前`;
    
    const diffInHours = Math.floor(diffInMinutes / 60);
    if (diffInHours < 24) return `${diffInHours}小时前`;
    
    const diffInDays = Math.floor(diffInHours / 24);
    if (diffInDays < 7) return `${diffInDays}天前`;
    
    const diffInWeeks = Math.floor(diffInDays / 7);
    if (diffInWeeks < 4) return `${diffInWeeks}周前`;
    
    const diffInMonths = Math.floor(diffInDays / 30);
    return `${diffInMonths}个月前`;
  },
};

export default ticketService;