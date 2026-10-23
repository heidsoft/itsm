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
    /** Optional: disambiguates when the account belongs to several tenants. */
    tenantCode?: string;
    tenantId?: number;
}
/**
 * Actual backend contract (handlers/common AuthResult): tokens travel in
 * HttpOnly cookies (`access_token` / `refresh_token`), so the JSON body has
 * no token fields. The CLI captures the cookie server-side and re-exposes it
 * as `accessToken` for credentials storage.
 */
export interface LoginUser {
    id: number;
    username: string;
    email: string;
    name: string;
    role: string;
    mspRole?: string;
    department?: string;
    departmentId?: number;
    phone?: string;
    active: boolean;
    tenantId: number;
    permissions?: string[];
}
export interface LoginResponseAuth {
    expiresIn: number;
    user: LoginUser;
    /** Extracted from the Set-Cookie header by the client, not present in the raw body. */
    accessToken?: string;
    refreshToken?: string;
}
export interface User extends LoginUser {
}
export interface SwitchTenantResponse {
    user: LoginUser;
    tenant: unknown;
}
export interface Ticket {
    id: number;
    ticketNumber: string;
    title: string;
    description?: string;
    status: string;
    priority: string;
    type?: string;
    category?: string;
    categoryId?: number;
    assigneeId?: number;
    assigneeName?: string;
    requesterName?: string;
    createdAt: string;
    updatedAt: string;
    slaStatus?: string;
    dueDate?: string;
    version?: number;
}
export interface TicketListResponse {
    items: Ticket[];
    total: number;
    page?: number;
    pageSize?: number;
    totalPages?: number;
}
export interface CreateTicketRequest {
    title: string;
    description: string;
    priority: string;
    category?: string;
}
export interface Incident {
    id: number;
    incidentNumber?: string;
    title: string;
    description?: string;
    status: string;
    priority: string;
    severity?: string;
    category?: string;
    assigneeId?: number;
    assigneeName?: string;
    reporterName?: string;
    relatedCiName?: string;
    createdAt: string;
    updatedAt: string;
    resolvedAt?: string;
    slaStatus?: string;
    dueDate?: string;
}
export interface IncidentListResponse {
    items: Incident[];
    total: number;
    page?: number;
    pageSize?: number;
    totalPages?: number;
}
export interface Change {
    id: number;
    changeNumber?: string;
    title: string;
    description?: string;
    type?: string;
    riskLevel?: string;
    status: string;
    priority: string;
    assigneeName?: string;
    implementerName?: string;
    plannedStartDate?: string;
    createdAt: string;
    updatedAt: string;
    rollbackPlan?: string;
}
export interface ChangeListResponse {
    items: Change[];
    total: number;
    page?: number;
    pageSize?: number;
    totalPages?: number;
}
export interface CI {
    id: number;
    name: string;
    typeName?: string;
    status: string;
    ownedBy?: string;
    location?: string;
    createdAt: string;
    updatedAt: string;
}
export interface CIListResponse {
    items: CI[];
    total: number;
    page?: number;
    pageSize?: number;
    totalPages?: number;
}
export interface KnowledgeArticle {
    id: number;
    title: string;
    summary?: string;
    content?: string;
    category?: string;
    tags?: string[];
    authorName?: string;
    viewCount?: number;
    status: string;
    createdAt: string;
    updatedAt: string;
}
export interface KnowledgeListResponse {
    items: KnowledgeArticle[];
    total: number;
    page?: number;
    pageSize?: number;
    totalPages?: number;
}
export interface ProcessInstance {
    id: string;
    processDefinitionKey?: string;
    businessKey?: string;
    state: string;
    startTime?: string;
    endTime?: string;
    initiator?: string;
    variables?: Record<string, unknown>;
}
export interface ProcessInstanceListResponse {
    items: ProcessInstance[];
    total: number;
    page?: number;
    pageSize?: number;
    totalPages?: number;
}
export interface ApprovalTask {
    id: string;
    name?: string;
    processInstanceId?: string;
    assigneeId?: number;
    assigneeName?: string;
    createdAt?: string;
    dueDate?: string;
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
    createdAt: string;
}
export interface NotificationListResponse {
    items: Notification[];
    total: number;
    unread?: number;
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
    /** Store schema version (2 = hardened atomic store). */
    version?: number;
    /** When the store was last written (ISO-8601). */
    savedAt?: string;
}
