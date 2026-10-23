import { CliError } from './errors.js';
import type { AgentError } from './errors.js';
import type { LoginRequest, LoginResponseAuth, SwitchTenantResponse, User, Ticket, TicketListResponse, CreateTicketRequest, PaginationParams, Incident, IncidentListResponse, Change, ChangeListResponse, CI, CIListResponse, KnowledgeArticle, KnowledgeListResponse, NotificationListResponse, ConnectorManifest, ProcessInstance, ProcessInstanceListResponse } from './types.js';
export interface TransportDescription {
    baseURL: string;
    protocol: string;
    host: string;
    secure: boolean;
    loopback: boolean;
    /** Whether credentials may be sent to this base URL. */
    credentialSafe: boolean;
    reason?: string;
}
export declare function describeTransport(baseURL: string): TransportDescription;
/** Fail closed before a secret is attached to a request. */
export declare function assertTransportSafety(baseURL: string, opts: {
    sendingSecret: boolean;
}): void;
export declare function configureBaseURL(url?: string): void;
/**
 * Transport policy. Agent runs are bounded: every request gets a deadline and
 * idempotent reads get transparent retries, so a flaky backend surfaces as a
 * retryable `network` error instead of an indefinite hang that burns the
 * agent's turn budget.
 */
interface TransportConfig {
    timeoutMs: number;
    retries: number;
}
export declare function configureTransport(patch: Partial<TransportConfig>): void;
export declare function getTransportConfig(): TransportConfig;
export declare function setExternalTokenProvider(fn: () => string | undefined): void;
export interface TransportOperationError extends CliError {
}
export declare function transportError(status: number, message: string, endpoint: string): AgentError;
export declare class ApiClient {
    private explicitBaseURL?;
    constructor(explicitBaseURL?: string | undefined);
    /** Resolved per request so ITSM_BASE_URL / --base-url apply to every domain, not just infra. */
    private get baseURL();
    withBaseURL(baseURL: string): ApiClient;
    /** Generic raw request used by the agent escape hatch (`infra.api`). Full path required. */
    /**
     * Raw escape hatch (`itsm infra api`). Delegates to request() so the caller gets
     * the exact same auth chain (ITSM_TOKEN → stored credentials → silent refresh)
     * and error classification as first-class commands.
     *
     * It previously built its own fetch without an Authorization header, so every
     * authenticated endpoint answered 401 — which made the one surface meant for
     * debugging unreachable ones useless for exactly the cases it exists for.
     */
    requestRaw(method: string, fullPath: string, body?: unknown): Promise<unknown>;
    login(loginData: LoginRequest): Promise<LoginResponseAuth>;
    /**
     * Revoke the server-side session before clearing the local store.
     *
     * The backend reads the refresh token from the `refresh_token` cookie on
     * /auth/logout. A CLI has no cookie jar, so without this header `logout`
     * only deletes the local file and leaves a valid 7-day refresh token behind
     * — exactly the credential an attacker would keep after a file leak.
     */
    logout(refreshToken?: string): Promise<void>;
    /** Explicit rotation used by `auth refresh`; also used by the silent 401 path. */
    refreshStoredSession(): Promise<{
        accessToken: string;
        refreshToken: string;
        expiresAt: string;
        expiresIn: number;
    }>;
    getUserInfo(): Promise<User>;
    getSession(): Promise<unknown>;
    getUserTenants(): Promise<unknown>;
    getUserMenus(): Promise<unknown>;
    register(data: Record<string, unknown>): Promise<unknown>;
    forgotPassword(data: Record<string, unknown>): Promise<unknown>;
    resetPassword(data: Record<string, unknown>): Promise<unknown>;
    validateResetToken(data: Record<string, unknown>): Promise<unknown>;
    listSSOProviders(): Promise<unknown>;
    initiateSSO(data: Record<string, unknown>): Promise<unknown>;
    listSSOIdentities(): Promise<unknown>;
    bindSSO(data: Record<string, unknown>): Promise<unknown>;
    unbindSSO(data: Record<string, unknown>): Promise<unknown>;
    switchTenant(tenantId: string): Promise<SwitchTenantResponse & {
        accessToken?: string;
        refreshToken?: string;
    }>;
    getHealth(): Promise<unknown>;
    getReady(): Promise<unknown>;
    getVersion(): Promise<unknown>;
    getCapabilities(): Promise<unknown>;
    globalSearch(q: string, page?: number): Promise<unknown>;
    listTickets(params?: PaginationParams): Promise<TicketListResponse>;
    getTicket(id: number | string): Promise<Ticket>;
    createTicket(data: CreateTicketRequest): Promise<Ticket>;
    updateTicket(id: number | string, data: unknown): Promise<unknown>;
    deleteTicket(id: number | string): Promise<void>;
    searchTickets(query: string, params?: PaginationParams): Promise<TicketListResponse>;
    getOverdueTickets(): Promise<Ticket[]>;
    getTicketStats(): Promise<unknown>;
    getMyCC(): Promise<unknown>;
    getApprovalRecords(params?: Record<string, string | number | boolean | undefined>): Promise<unknown>;
    assignTicket(id: number | string, assigneeId: number | string): Promise<unknown>;
    resolveTicket(id: number | string, data?: Record<string, unknown>): Promise<unknown>;
    closeTicket(id: number | string, data?: Record<string, unknown>): Promise<unknown>;
    escalateTicket(id: number | string, data?: Record<string, unknown>): Promise<unknown>;
    getTicketHistory(id: number | string): Promise<unknown>;
    getTicketWorkflowState(id: number | string): Promise<unknown>;
    ticketWorkflowAction(action: 'accept' | 'reject' | 'withdraw' | 'forward' | 'approve' | 'resolve' | 'close' | 'reopen' | 'cc', data: unknown): Promise<unknown>;
    getTicketSLA(id: number | string): Promise<unknown>;
    aiSummarizeTicket(id: number | string): Promise<unknown>;
    aiAnalyzeTicket(id: number | string): Promise<unknown>;
    listTicketComments(id: number | string): Promise<unknown>;
    createTicketComment(id: number | string, data: unknown): Promise<unknown>;
    updateTicketStatus(id: number | string, status: string): Promise<unknown>;
    pauseTicketSLA(id: number | string): Promise<unknown>;
    resumeTicketSLA(id: number | string): Promise<unknown>;
    listSubtasks(id: number | string): Promise<unknown>;
    createSubtask(id: number | string, data: Record<string, unknown>): Promise<unknown>;
    updateSubtask(id: number | string, subtaskId: number | string, data: Record<string, unknown>): Promise<unknown>;
    deleteSubtask(id: number | string, subtaskId: number | string): Promise<void>;
    getTicketRelations(id: number | string): Promise<unknown>;
    updateTicketRelations(id: number | string, data: Record<string, unknown>): Promise<unknown>;
    getTicketRelationStats(id: number | string): Promise<unknown>;
    getTicketCC(id: number | string): Promise<unknown>;
    addTicketTags(id: number | string, data: Record<string, unknown>): Promise<unknown>;
    removeTicketTags(id: number | string, data: Record<string, unknown>): Promise<unknown>;
    getTicketRating(id: number | string): Promise<unknown>;
    submitTicketRating(id: number | string, data: Record<string, unknown>): Promise<unknown>;
    getRatingStats(params?: Record<string, string | number | boolean | undefined>): Promise<unknown>;
    getTicketConfigurationItems(id: number | string): Promise<unknown>;
    getTicketDependencies(id: number | string): Promise<unknown>;
    listTicketNotifications(id: number | string): Promise<unknown>;
    sendTicketNotification(id: number | string, data: Record<string, unknown>): Promise<unknown>;
    getTicketWorkflowHistory(id: number | string): Promise<unknown>;
    autoAssignTicket(id: number | string, data?: Record<string, unknown>): Promise<unknown>;
    listIncidents(params?: PaginationParams & {
        status?: string;
        priority?: string;
    }): Promise<IncidentListResponse>;
    getIncident(id: number | string): Promise<Incident>;
    createIncident(data: unknown): Promise<unknown>;
    deleteIncident(id: number | string): Promise<void>;
    updateIncident(id: number | string, data: unknown): Promise<unknown>;
    incidentLifecycle(id: number | string, action: 'acknowledge' | 'resolve' | 'close' | 'reopen' | 'assign' | 'escalate' | 'convert-to-problem', data?: unknown): Promise<unknown>;
    updateIncidentRootCause(id: number | string, data: unknown): Promise<unknown>;
    listIncidentComments(id: number | string): Promise<unknown>;
    createIncidentComment(id: number | string, data: unknown): Promise<unknown>;
    updateIncidentStatus(id: number | string, data: Record<string, unknown>): Promise<unknown>;
    getIncidentRootCause(id: number | string): Promise<unknown>;
    getIncidentClassification(id: number | string): Promise<unknown>;
    updateIncidentClassification(id: number | string, data: Record<string, unknown>): Promise<unknown>;
    getIncidentImpactAssessment(id: number | string): Promise<unknown>;
    updateIncidentImpactAssessment(id: number | string, data: Record<string, unknown>): Promise<unknown>;
    escalateMajorIncident(id: number | string, data: Record<string, unknown>): Promise<unknown>;
    getIncidentEvents(id: number | string): Promise<unknown>;
    getIncidentAlerts(id: number | string): Promise<unknown>;
    getIncidentMetrics(id: number | string): Promise<unknown>;
    pauseIncidentSLA(id: number | string): Promise<unknown>;
    resumeIncidentSLA(id: number | string): Promise<unknown>;
    getIncidentStats(): Promise<unknown>;
    incidentReport(params: {
        dateFrom?: string;
        dateTo?: string;
    }): Promise<unknown>;
    aiTriage(payload: {
        title: string;
        description: string;
    }): Promise<unknown>;
    listProblems(params?: Record<string, string | number | boolean | undefined>): Promise<unknown>;
    getProblem(id: number | string): Promise<unknown>;
    createProblem(data: unknown): Promise<unknown>;
    updateProblem(id: number | string, data: unknown): Promise<unknown>;
    deleteProblem(id: number | string): Promise<void>;
    problemLifecycle(id: number | string, action: 'close' | 'investigate', data?: unknown): Promise<unknown>;
    updateProblemRootCause(id: number | string, data: unknown): Promise<unknown>;
    updateProblemSolution(id: number | string, data: unknown): Promise<unknown>;
    listProblemComments(id: number | string): Promise<unknown>;
    getProblemStats(): Promise<unknown>;
    getProblemTrends(params?: Record<string, string | number | boolean | undefined>): Promise<unknown>;
    getProblemHotspots(params?: Record<string, string | number | boolean | undefined>): Promise<unknown>;
    addProblemComment(id: number | string, data: unknown): Promise<unknown>;
    listKnownErrors(params?: Record<string, string | number | boolean | undefined>): Promise<unknown>;
    getKnownError(id: number | string): Promise<unknown>;
    searchKnownErrors(q: string): Promise<unknown>;
    listChanges(params?: PaginationParams & {
        status?: string;
    }): Promise<ChangeListResponse>;
    getChange(id: number | string): Promise<Change>;
    createChange(data: unknown): Promise<unknown>;
    updateChange(id: number | string, data: unknown): Promise<unknown>;
    deleteChange(id: number | string): Promise<void>;
    changeLifecycle(id: number | string, action: 'submit' | 'assign' | 'approve' | 'reject' | 'schedule' | 'start' | 'complete' | 'fail' | 'close' | 'rollback' | 'cancel', data?: unknown): Promise<unknown>;
    getChangeApprovals(id: number | string): Promise<unknown>;
    getChangeRisk(id: number | string): Promise<unknown>;
    updateChangeRisk(id: number | string, data: unknown): Promise<unknown>;
    getChangePIR(id: number | string): Promise<unknown>;
    createChangePIR(id: number | string, data: Record<string, unknown>): Promise<unknown>;
    updateChangePIR(pirId: number | string, data: Record<string, unknown>): Promise<unknown>;
    getChangeApprovalSummary(id: number | string): Promise<unknown>;
    createChangeApproval(id: number | string, data: Record<string, unknown>): Promise<unknown>;
    getChangeRiskAssessment(id: number | string): Promise<unknown>;
    getChangeCMDBImpact(id: number | string): Promise<unknown>;
    getChangeRollbackSteps(id: number | string): Promise<unknown>;
    replaceChangeRollbackSteps(id: number | string, steps: string[]): Promise<unknown>;
    resolveChangeRollbackStep(id: number | string, stepId: number | string, data: Record<string, unknown>): Promise<unknown>;
    listChangeCalendar(params?: Record<string, string | number | boolean | undefined>): Promise<unknown>;
    listChangePIRs(params?: Record<string, string | number | boolean | undefined>): Promise<unknown>;
    getChangeStats(): Promise<unknown>;
    listReleases(params?: Record<string, string | number | boolean | undefined>): Promise<unknown>;
    getRelease(id: number | string): Promise<unknown>;
    createRelease(data: unknown): Promise<unknown>;
    updateRelease(id: number | string, data: unknown): Promise<unknown>;
    updateReleaseStatus(id: number | string, status: string, comment?: string): Promise<unknown>;
    releaseLifecycle(id: number | string, action: 'approve' | 'reject' | 'rollback', data?: unknown): Promise<unknown>;
    deleteRelease(id: number | string): Promise<void>;
    getReleaseStats(): Promise<unknown>;
    listServiceCatalog(params?: Record<string, string | number | boolean | undefined>): Promise<unknown>;
    getServiceCatalogItem(id: number | string): Promise<unknown>;
    searchServiceCatalog(q: string): Promise<unknown>;
    createServiceCatalogItem(data: unknown): Promise<unknown>;
    updateServiceCatalogItem(id: number | string, data: unknown): Promise<unknown>;
    deleteServiceCatalogItem(id: number | string): Promise<void>;
    listServiceRequests(params?: Record<string, string | number | boolean | undefined>): Promise<unknown>;
    getServiceRequest(id: number | string): Promise<unknown>;
    createServiceRequest(data: unknown): Promise<unknown>;
    updateServiceRequest(id: number | string, data: unknown): Promise<unknown>;
    updateServiceRequestStatus(id: number | string, status: string): Promise<unknown>;
    serviceRequestApproval(id: number | string, data: unknown): Promise<unknown>;
    startServiceRequestProvisioning(id: number | string, data?: unknown): Promise<unknown>;
    listProvisioningTasks(id: number | string): Promise<unknown>;
    getServiceRequestSLA(id: number | string): Promise<unknown>;
    listServiceRequestApprovals(id: number | string): Promise<unknown>;
    deleteServiceRequest(id: number | string): Promise<void>;
    /** Failed provisioning task dispositions: execute (retry now), retry (requeue), resolve (manual closure). */
    provisioningTaskAction(id: number | string, action: 'execute' | 'retry' | 'resolve', data?: unknown): Promise<unknown>;
    listCIs(params?: PaginationParams & {
        type?: string;
        status?: string;
    }): Promise<CIListResponse>;
    getCI(id: number | string): Promise<CI>;
    createCI(data: unknown): Promise<unknown>;
    updateCI(id: number | string, data: unknown): Promise<unknown>;
    deleteCI(id: number | string): Promise<void>;
    /**
     * Keyword CI search. GET /cmdb/cis is the live endpoint (`search` is its
     * keyword param); POST /cmdb/cis/search is deprecated *and* only accepts a
     * structured envelope — a bare `{q}` there trips the `page: min=1` binding.
     */
    searchCIsByKeyword(q: string, params?: PaginationParams): Promise<CIListResponse>;
    /**
     * Structured CI search (filters/sort). `page` is defaulted because the
     * backend binds `min=1` while an omitted page arrives as 0 → 400, so a
     * filters-only body would be rejected for a field the caller never set.
     */
    searchCIs(filters: Record<string, unknown>): Promise<unknown>;
    getCIStats(): Promise<unknown>;
    listCITypes(params?: Record<string, string | number | boolean | undefined>): Promise<unknown>;
    getCITopology(id: number | string): Promise<unknown>;
    getCIImpactAnalysis(id: number | string): Promise<unknown>;
    getCIHistory(id: number | string): Promise<unknown>;
    updateCILifecycle(id: number | string, data: unknown): Promise<unknown>;
    listCIRelationships(params?: Record<string, string | number | boolean | undefined>): Promise<unknown>;
    createCIRelationship(data: unknown): Promise<unknown>;
    deleteCIRelationship(id: number | string): Promise<void>;
    getCIRelationship(id: number | string): Promise<unknown>;
    updateCIRelationship(id: number | string, data: unknown): Promise<unknown>;
    listCIRelationshipsByCI(id: number | string): Promise<unknown>;
    addCITags(id: number | string, tagIds: number[]): Promise<unknown>;
    removeCITags(id: number | string, tagIds: number[]): Promise<unknown>;
    getCILifecycleHistory(id: number | string): Promise<unknown>;
    revertCIVersion(id: number | string, version: number, remark?: string): Promise<unknown>;
    createCIType(data: unknown): Promise<unknown>;
    updateCIType(id: number | string, data: unknown): Promise<unknown>;
    deleteCIType(id: number | string): Promise<void>;
    getCIType(id: number | string): Promise<unknown>;
    listCITypeAttributes(id: number | string): Promise<unknown>;
    listRelationshipTypes(): Promise<unknown>;
    getCMDBOntology(): Promise<unknown>;
    getCMDBCapabilities(): Promise<unknown>;
    getCMDBReconciliation(params?: Record<string, string | number | boolean | undefined>): Promise<unknown>;
    batchUpdateCIs(ids: number[], data: unknown): Promise<unknown>;
    batchUpdateCILifecycle(ids: number[], data: unknown): Promise<unknown>;
    listKnowledge(params?: PaginationParams & {
        category?: string;
    }): Promise<KnowledgeListResponse>;
    /**
     * The backend binds `query` with `binding:"required"` and reads paging from
     * the JSON body (not the query string), so `{q}` — or `pageSize` passed as a
     * query param — is a 400.
     */
    searchKnowledge(query: string, opts?: {
        category?: string;
        pageSize?: number;
    }): Promise<KnowledgeListResponse>;
    getKnowledgeArticle(id: number | string): Promise<KnowledgeArticle>;
    createKnowledgeArticle(data: unknown): Promise<unknown>;
    updateKnowledgeArticle(id: number | string, data: unknown): Promise<unknown>;
    deleteKnowledgeArticle(id: number | string): Promise<void>;
    knowledgeLifecycle(id: number | string, action: 'publish' | 'unpublish'): Promise<unknown>;
    /** Editorial review workflow: submit → approve/reject, with withdraw/republish for revisions. */
    knowledgeReviewAction(id: number | string, action: 'submit' | 'approve' | 'reject' | 'withdraw' | 'republish', data?: Record<string, unknown>): Promise<unknown>;
    listKnowledgeReviews(id: number | string): Promise<unknown>;
    listKnowledgeCategories(): Promise<unknown>;
    listKnowledgeVersions(id: number | string): Promise<unknown>;
    getKnowledgeStats(): Promise<unknown>;
    getSLAStats(): Promise<unknown>;
    listSLADefinitions(params?: Record<string, string | number | boolean | undefined>): Promise<unknown>;
    getSLADefinition(id: number | string): Promise<unknown>;
    getSLAMetrics(): Promise<unknown>;
    getSLAViolations(params?: Record<string, string | number | boolean | undefined>): Promise<unknown>;
    getSLAMonitoring(): Promise<unknown>;
    getSLAPerformance(): Promise<unknown>;
    getSLAAlertHistory(): Promise<unknown>;
    getSLAComplianceReport(params?: Record<string, string | number | boolean | undefined>): Promise<unknown>;
    checkSLACompliance(ticketId: number | string): Promise<unknown>;
    listSLATemplates(): Promise<unknown>;
    installSLATemplate(key: string): Promise<unknown>;
    listProcessInstances(params?: PaginationParams): Promise<ProcessInstanceListResponse>;
    getProcessInstance(id: string): Promise<ProcessInstance>;
    startProcess(data: unknown): Promise<unknown>;
    processInstanceAction(id: string, action: 'suspend' | 'resume' | 'terminate', data?: unknown): Promise<unknown>;
    listBPMNDefinitions(params?: Record<string, string | number | boolean | undefined>): Promise<unknown>;
    getBPMNDefinition(key: string): Promise<unknown>;
    /**
     * `GET /bpmn/tasks` is "my tasks" (assignee/candidate identity comes from the
     * token, never from the query). `status` is optional server-side and — when
     * omitted — returns completed tasks too, so a caller that means "待办" has to
     * say so.
     */
    listUserTasks(params?: PaginationParams & {
        status?: string;
        processDefinitionKey?: string;
    }): Promise<unknown>;
    getTask(id: string): Promise<unknown>;
    assignTask(id: string, assigneeId: number): Promise<unknown>;
    decideTask(id: string, data: Record<string, unknown>): Promise<unknown>;
    cancelTask(id: string, reason: string): Promise<unknown>;
    setProcessInstanceVariables(id: string, variables: Record<string, unknown>): Promise<unknown>;
    completeTask(id: string, data?: unknown): Promise<unknown>;
    claimTask(id: string, data?: unknown): Promise<unknown>;
    listApprovalWorkflows(params?: Record<string, string | number | boolean | undefined>): Promise<unknown>;
    getApprovalWorkflow(id: string): Promise<unknown>;
    listApprovalRecords(params?: Record<string, string | number | boolean | undefined>): Promise<unknown>;
    submitApproval(data: unknown): Promise<unknown>;
    listNotifications(params?: PaginationParams & {
        unread?: boolean;
    }): Promise<NotificationListResponse>;
    getUnreadNotificationCount(): Promise<unknown>;
    markNotificationRead(id: number | string): Promise<unknown>;
    markAllNotificationsRead(): Promise<unknown>;
    deleteNotification(id: number | string): Promise<void>;
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
    }): Promise<unknown>;
    revokeConnector(name: string, provider?: string): Promise<void>;
    testConnector(name: string): Promise<unknown>;
    sendViaConnector(name: string, payload: {
        channel: string;
        type: string;
        title?: string;
        content: string;
        card?: unknown;
    }): Promise<unknown>;
    connectorHealth(): Promise<unknown>;
    listNotificationDeliveries(params?: Record<string, string | number | boolean | undefined>): Promise<unknown>;
    getDeliveryStats(): Promise<unknown>;
    retryDelivery(id: string): Promise<unknown>;
    getDashboardOverview(): Promise<unknown>;
    listUsers(params?: Record<string, string | number | boolean | undefined>): Promise<unknown>;
    getUser(id: number | string): Promise<unknown>;
    listTeams(): Promise<unknown>;
    listDepartments(): Promise<unknown>;
    listRoles(params?: Record<string, string | number | boolean | undefined>): Promise<unknown>;
    listPermissions(): Promise<unknown>;
    listTenants(params?: Record<string, string | number | boolean | undefined>): Promise<unknown>;
    getTenant(id: number | string): Promise<unknown>;
    listAuditLogs(params?: Record<string, string | number | boolean | undefined>): Promise<unknown>;
    listOperationalCommands(params?: Record<string, string | number | boolean | undefined>): Promise<unknown>;
    getOperationalCommand(id: number | string): Promise<unknown>;
    replayOperationalCommand(id: number | string): Promise<unknown>;
    cancelOperationalCommand(id: number | string): Promise<unknown>;
    bulkReplayOperationalCommands(data: Record<string, unknown>): Promise<unknown>;
    bulkCancelOperationalCommands(data: Record<string, unknown>): Promise<unknown>;
    getMSPStatus(): Promise<unknown>;
    getMSPContext(): Promise<unknown>;
    listMSPCustomers(): Promise<unknown>;
    listMSPAllocations(): Promise<unknown>;
}
export declare const apiClient: ApiClient;
export {};
