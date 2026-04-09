import type { LoginRequest, LoginResponse, User, Ticket, TicketListResponse, CreateTicketRequest, PaginationParams } from './types.js';
export declare class ApiClient {
    private baseURL;
    constructor(baseURL?: string);
    login(loginData: LoginRequest): Promise<LoginResponse>;
    logout(): Promise<void>;
    getUserInfo(): Promise<User>;
    listTickets(params?: PaginationParams): Promise<TicketListResponse>;
    getTicket(id: number): Promise<Ticket>;
    createTicket(data: CreateTicketRequest): Promise<Ticket>;
    searchTickets(query: string, params?: PaginationParams): Promise<TicketListResponse>;
    getSLAStats(): Promise<unknown>;
    getOverdueTickets(): Promise<Ticket[]>;
}
export declare const apiClient: ApiClient;
