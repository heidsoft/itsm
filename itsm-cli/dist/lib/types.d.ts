export interface ApiResponse<T> {
    code: number;
    message: string;
    data: T;
}
export interface PaginationParams {
    page?: number;
    pageSize?: number;
    status?: string;
    priority?: string;
    type?: string;
    category?: string;
    q?: string;
    search?: string;
    unread?: boolean;
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
    description?: string;
    status: string;
    priority: string;
    category?: string;
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
    page?: number;
    pageSize?: number;
}
export interface CreateTicketRequest {
    title: string;
    description: string;
    priority: string;
    category?: string;
}
export interface Incident {
    id: number;
    number: string;
    title: string;
    description?: string;
    status: string;
    priority: string;
    severity?: string;
    category?: string;
    assignee_id?: number;
    assignee_name?: string;
    reporter_name?: string;
    cmdb_ci_id?: number;
    cmdb_ci_name?: string;
    created_at: string;
    updated_at: string;
    resolved_at?: string;
    sla_status?: string;
    due_date?: string;
}
export interface IncidentListResponse {
    items: Incident[];
    total: number;
    page?: number;
    pageSize?: number;
}
export interface Change {
    id: number;
    number: string;
    title: string;
    description?: string;
    type?: string;
    risk_level?: string;
    status: string;
    priority: string;
    assignee_name?: string;
    implementer_name?: string;
    scheduled_at?: string;
    created_at: string;
    updated_at: string;
    rollback_plan?: string;
}
export interface ChangeListResponse {
    items: Change[];
    total: number;
}
export interface CI {
    id: number;
    name: string;
    type: string;
    status: string;
    owner?: string;
    location?: string;
    created_at: string;
    updated_at: string;
}
export interface CIListResponse {
    items: CI[];
    total: number;
}
export interface KnowledgeArticle {
    id: number;
    title: string;
    summary?: string;
    content?: string;
    category?: string;
    tags?: string[];
    author_name?: string;
    views: number;
    likes: number;
    status: string;
    created_at: string;
    updated_at: string;
}
export interface KnowledgeListResponse {
    items: KnowledgeArticle[];
    total: number;
}
export interface ProcessInstance {
    id: string;
    process_key: string;
    business_key?: string;
    state: string;
    start_time: string;
    end_time?: string;
    initiator?: string;
    variables?: Record<string, unknown>;
}
export interface ProcessInstanceListResponse {
    items: ProcessInstance[];
    total: number;
}
export interface ApprovalTask {
    id: string;
    name: string;
    process_instance_id?: string;
    assignee?: string;
    candidate_groups?: string[];
    created_at: string;
    due_date?: string;
    priority?: string;
    status?: string;
    variables?: Record<string, unknown>;
}
export interface ApprovalTaskListResponse {
    items: ApprovalTask[];
    total: number;
}
export interface Notification {
    id: number;
    title: string;
    content?: string;
    type: string;
    read: boolean;
    level?: string;
    created_at: string;
}
export interface NotificationListResponse {
    items: Notification[];
    total: number;
    unread: number;
}
export interface ConnectorManifest {
    name: string;
    version: string;
    title: string;
    provider: string;
    type: string;
    description?: string;
    capabilities: string[];
    tags?: string[];
    local: boolean;
    installed: boolean;
    category: string;
    homepage?: string;
    icon_url?: string;
}
export interface Credentials {
    token: string;
    refreshToken?: string;
    user?: User;
    tenantId?: number;
    tenantName?: string;
    expiresAt?: string;
}
