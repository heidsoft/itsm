export interface ApiResponse<T> {
    code: number;
    message: string;
    data: T;
}
export interface LoginRequest {
    username: string;
    password: string;
}
export interface LoginResponse {
    token: string;
    refresh_token: string;
    user: User;
    tenant_id: number;
    tenant_name: string;
}
export interface User {
    id: number;
    username: string;
    name: string;
    email: string;
    role: string;
    tenant_id: number;
}
export interface Ticket {
    id: number;
    ticket_number: string;
    title: string;
    description: string;
    status: string;
    priority: string;
    category: string;
    assignee_id?: number;
    assignee_name?: string;
    requester_name?: string;
    created_at: string;
    updated_at: string;
    sla_status?: string;
    due_date?: string;
}
export interface TicketListResponse {
    items: Ticket[];
    total: number;
    page: number;
    pageSize: number;
    totalPages: number;
}
export interface CreateTicketRequest {
    title: string;
    description: string;
    priority: string;
    category?: string;
    category_id?: number;
}
export interface PaginationParams {
    page?: number;
    pageSize?: number;
    status?: string;
    priority?: string;
}
export interface Credentials {
    token: string;
    refreshToken: string;
    user: User;
    tenantId: number;
    tenantName: string;
    expiresAt?: string;
}
