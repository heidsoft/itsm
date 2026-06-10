import type { LoginRequest, LoginResponse, User, Ticket, TicketListResponse, CreateTicketRequest, PaginationParams, Incident, IncidentListResponse, Change, ChangeListResponse, CI, CIListResponse, KnowledgeArticle, KnowledgeListResponse, NotificationListResponse, ConnectorManifest, ProcessInstance, ProcessInstanceListResponse, ApprovalTaskListResponse } from './types.js';
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
    getOverdueTickets(): Promise<Ticket[]>;
    globalSearch(q: string, page?: number): Promise<Record<string, unknown>>;
    listIncidents(params?: PaginationParams & {
        status?: string;
        priority?: string;
    }): Promise<IncidentListResponse>;
    getIncident(id: number): Promise<Incident>;
    aiTriage(payload: {
        title: string;
        description: string;
    }): Promise<Record<string, unknown>>;
    listChanges(params?: PaginationParams & {
        status?: string;
    }): Promise<ChangeListResponse>;
    getChange(id: number): Promise<Change>;
    listCIs(params?: PaginationParams & {
        type?: string;
        status?: string;
    }): Promise<CIListResponse>;
    getCI(id: number): Promise<CI>;
    listKnowledge(params?: PaginationParams & {
        category?: string;
    }): Promise<KnowledgeListResponse>;
    searchKnowledge(q: string): Promise<KnowledgeListResponse>;
    getKnowledgeArticle(id: number): Promise<KnowledgeArticle>;
    getSLAStats(): Promise<Record<string, unknown>>;
    getSLAOverdue(): Promise<Ticket[]>;
    listProcessInstances(params?: PaginationParams): Promise<ProcessInstanceListResponse>;
    getProcessInstance(id: string): Promise<ProcessInstance>;
    listUserTasks(): Promise<ApprovalTaskListResponse>;
    completeTask(id: string, outcome: string, comment?: string): Promise<Record<string, unknown>>;
    listApprovals(status?: string): Promise<ApprovalTaskListResponse>;
    approveTask(id: string, comment?: string): Promise<Record<string, unknown>>;
    rejectTask(id: string, comment?: string): Promise<Record<string, unknown>>;
    listNotifications(params?: PaginationParams & {
        unread?: boolean;
    }): Promise<NotificationListResponse>;
    markNotificationRead(id: number): Promise<Record<string, unknown>>;
    listConnectors(): Promise<{
        items: ConnectorManifest[];
        total: number;
    }>;
    listConnectorConfigs(): Promise<{
        items: unknown[];
        total: number;
    }>;
    provisionConnector(cfg: {
        name: string;
        provider: string;
        enabled: boolean;
        credentials: Record<string, string>;
        settings: Record<string, unknown>;
    }): Promise<Record<string, unknown>>;
    testConnector(name: string): Promise<Record<string, unknown>>;
    sendViaConnector(name: string, payload: {
        channel: string;
        type: string;
        title?: string;
        content: string;
        card?: unknown;
    }): Promise<Record<string, unknown>>;
    connectorHealth(): Promise<Record<string, unknown>>;
    getDashboardOverview(): Promise<Record<string, unknown>>;
}
export declare const apiClient: ApiClient;
