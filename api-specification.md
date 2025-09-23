# ITSM 系统 API 接口规范

## 1. API 设计原则

### 1.1 RESTful 设计
- 使用标准HTTP方法：GET、POST、PUT、DELETE、PATCH
- 资源导向的URL设计
- 统一的响应格式
- 合理的HTTP状态码使用

### 1.2 版本控制
- API版本通过URL路径控制：`/api/v1/`
- 向后兼容原则
- 废弃API的渐进式迁移

### 1.3 安全规范
- JWT Token认证
- RBAC权限控制
- 请求频率限制
- 数据加密传输

## 2. 通用数据结构

### 2.1 统一响应格式

```typescript
interface ApiResponse<T> {
  success: boolean
  data?: T
  message?: string
  error?: {
    code: string
    message: string
    details?: any
  }
  timestamp: string
  requestId: string
}

interface PaginatedResponse<T> {
  success: boolean
  data: T[]
  pagination: {
    page: number
    limit: number
    total: number
    totalPages: number
    hasNext: boolean
    hasPrev: boolean
  }
  message?: string
  timestamp: string
  requestId: string
}
```

### 2.2 错误码定义

```typescript
enum ErrorCode {
  // 通用错误
  INVALID_REQUEST = 'INVALID_REQUEST',
  UNAUTHORIZED = 'UNAUTHORIZED',
  FORBIDDEN = 'FORBIDDEN',
  NOT_FOUND = 'NOT_FOUND',
  INTERNAL_ERROR = 'INTERNAL_ERROR',
  
  // 业务错误
  TICKET_NOT_FOUND = 'TICKET_NOT_FOUND',
  INVALID_TICKET_STATUS = 'INVALID_TICKET_STATUS',
  INSUFFICIENT_PERMISSIONS = 'INSUFFICIENT_PERMISSIONS',
  DUPLICATE_RESOURCE = 'DUPLICATE_RESOURCE',
  
  // 验证错误
  VALIDATION_ERROR = 'VALIDATION_ERROR',
  MISSING_REQUIRED_FIELD = 'MISSING_REQUIRED_FIELD',
  INVALID_FORMAT = 'INVALID_FORMAT',
}
```

## 3. 认证授权 API

### 3.1 用户认证

```typescript
// POST /api/v1/auth/login
interface LoginRequest {
  username: string
  password: string
  remember?: boolean
}

interface LoginResponse {
  user: User
  token: string
  refreshToken: string
  expiresIn: number
}

// POST /api/v1/auth/refresh
interface RefreshTokenRequest {
  refreshToken: string
}

interface RefreshTokenResponse {
  token: string
  expiresIn: number
}

// POST /api/v1/auth/logout
interface LogoutRequest {
  token: string
}
```

### 3.2 用户管理

```typescript
// GET /api/v1/users
interface GetUsersQuery {
  page?: number
  limit?: number
  search?: string
  role?: string
  department?: string
  status?: 'active' | 'inactive'
}

// GET /api/v1/users/:id
// POST /api/v1/users
interface CreateUserRequest {
  username: string
  email: string
  password: string
  firstName: string
  lastName: string
  phone?: string
  department: string
  role: string
  permissions?: string[]
}

// PUT /api/v1/users/:id
interface UpdateUserRequest {
  email?: string
  firstName?: string
  lastName?: string
  phone?: string
  department?: string
  role?: string
  permissions?: string[]
  status?: 'active' | 'inactive'
}

interface User {
  id: string
  username: string
  email: string
  firstName: string
  lastName: string
  fullName: string
  phone?: string
  avatar?: string
  department: string
  role: string
  permissions: string[]
  status: 'active' | 'inactive'
  lastLoginAt?: string
  createdAt: string
  updatedAt: string
}
```

## 4. 工单管理 API

### 4.1 工单CRUD操作

```typescript
// GET /api/v1/tickets
interface GetTicketsQuery {
  page?: number
  limit?: number
  search?: string
  status?: TicketStatus
  priority?: TicketPriority
  category?: string
  assigneeId?: string
  reporterId?: string
  createdFrom?: string
  createdTo?: string
  sortBy?: 'createdAt' | 'updatedAt' | 'priority' | 'dueDate'
  sortOrder?: 'asc' | 'desc'
}

// GET /api/v1/tickets/:id
// POST /api/v1/tickets
interface CreateTicketRequest {
  title: string
  description: string
  category: string
  priority: TicketPriority
  urgency: TicketUrgency
  impact: TicketImpact
  assigneeId?: string
  dueDate?: string
  tags?: string[]
  customFields?: Record<string, any>
  attachments?: string[]
}

// PUT /api/v1/tickets/:id
interface UpdateTicketRequest {
  title?: string
  description?: string
  category?: string
  priority?: TicketPriority
  urgency?: TicketUrgency
  impact?: TicketImpact
  assigneeId?: string
  dueDate?: string
  tags?: string[]
  customFields?: Record<string, any>
}

// PATCH /api/v1/tickets/:id/status
interface UpdateTicketStatusRequest {
  status: TicketStatus
  comment?: string
}

// PATCH /api/v1/tickets/:id/assign
interface AssignTicketRequest {
  assigneeId: string
  comment?: string
}

enum TicketStatus {
  OPEN = 'open',
  IN_PROGRESS = 'in_progress',
  PENDING = 'pending',
  RESOLVED = 'resolved',
  CLOSED = 'closed',
  CANCELLED = 'cancelled'
}

enum TicketPriority {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  CRITICAL = 'critical'
}

enum TicketUrgency {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high'
}

enum TicketImpact {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high'
}

interface Ticket {
  id: string
  ticketNumber: string
  title: string
  description: string
  category: string
  status: TicketStatus
  priority: TicketPriority
  urgency: TicketUrgency
  impact: TicketImpact
  reporter: User
  assignee?: User
  dueDate?: string
  resolvedAt?: string
  closedAt?: string
  tags: string[]
  customFields: Record<string, any>
  attachments: Attachment[]
  slaStatus: 'within_sla' | 'approaching_breach' | 'breached'
  timeToResolve?: number
  createdAt: string
  updatedAt: string
}
```

### 4.2 工单评论

```typescript
// GET /api/v1/tickets/:id/comments
interface GetCommentsQuery {
  page?: number
  limit?: number
}

// POST /api/v1/tickets/:id/comments
interface CreateCommentRequest {
  content: string
  isInternal?: boolean
  attachments?: string[]
}

// PUT /api/v1/comments/:id
interface UpdateCommentRequest {
  content: string
}

interface Comment {
  id: string
  ticketId: string
  author: User
  content: string
  isInternal: boolean
  attachments: Attachment[]
  createdAt: string
  updatedAt: string
}
```

### 4.3 工单历史记录

```typescript
// GET /api/v1/tickets/:id/history
interface TicketHistory {
  id: string
  ticketId: string
  action: 'created' | 'updated' | 'status_changed' | 'assigned' | 'commented'
  field?: string
  oldValue?: any
  newValue?: any
  user: User
  comment?: string
  createdAt: string
}
```

## 5. 事件管理 API

### 5.1 事件CRUD操作

```typescript
// GET /api/v1/incidents
interface GetIncidentsQuery {
  page?: number
  limit?: number
  search?: string
  status?: IncidentStatus
  priority?: IncidentPriority
  category?: string
  assigneeId?: string
  affectedServices?: string[]
  createdFrom?: string
  createdTo?: string
}

// POST /api/v1/incidents
interface CreateIncidentRequest {
  title: string
  description: string
  category: string
  priority: IncidentPriority
  urgency: IncidentUrgency
  impact: IncidentImpact
  affectedServices: string[]
  assigneeId?: string
  estimatedResolutionTime?: string
}

enum IncidentStatus {
  NEW = 'new',
  INVESTIGATING = 'investigating',
  IDENTIFIED = 'identified',
  MONITORING = 'monitoring',
  RESOLVED = 'resolved',
  CLOSED = 'closed'
}

enum IncidentPriority {
  P1 = 'p1', // Critical
  P2 = 'p2', // High
  P3 = 'p3', // Medium
  P4 = 'p4'  // Low
}

interface Incident {
  id: string
  incidentNumber: string
  title: string
  description: string
  category: string
  status: IncidentStatus
  priority: IncidentPriority
  urgency: IncidentUrgency
  impact: IncidentImpact
  reporter: User
  assignee?: User
  affectedServices: Service[]
  estimatedResolutionTime?: string
  actualResolutionTime?: string
  rootCause?: string
  resolution?: string
  postMortemRequired: boolean
  postMortemUrl?: string
  relatedTickets: string[]
  timeline: IncidentTimelineEntry[]
  createdAt: string
  updatedAt: string
  resolvedAt?: string
  closedAt?: string
}

interface IncidentTimelineEntry {
  id: string
  timestamp: string
  user: User
  action: string
  description: string
  isPublic: boolean
}
```

## 6. 问题管理 API

### 6.1 问题CRUD操作

```typescript
// GET /api/v1/problems
interface GetProblemsQuery {
  page?: number
  limit?: number
  search?: string
  status?: ProblemStatus
  priority?: ProblemPriority
  category?: string
  assigneeId?: string
}

// POST /api/v1/problems
interface CreateProblemRequest {
  title: string
  description: string
  category: string
  priority: ProblemPriority
  rootCause?: string
  workaround?: string
  relatedIncidents: string[]
  assigneeId?: string
}

enum ProblemStatus {
  NEW = 'new',
  INVESTIGATING = 'investigating',
  ROOT_CAUSE_IDENTIFIED = 'root_cause_identified',
  WORKAROUND_AVAILABLE = 'workaround_available',
  RESOLVED = 'resolved',
  CLOSED = 'closed'
}

enum ProblemPriority {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  CRITICAL = 'critical'
}

interface Problem {
  id: string
  problemNumber: string
  title: string
  description: string
  category: string
  status: ProblemStatus
  priority: ProblemPriority
  reporter: User
  assignee?: User
  rootCause?: string
  workaround?: string
  permanentSolution?: string
  relatedIncidents: string[]
  relatedChanges: string[]
  knownErrors: KnownError[]
  createdAt: string
  updatedAt: string
  resolvedAt?: string
  closedAt?: string
}

interface KnownError {
  id: string
  title: string
  description: string
  workaround: string
  status: 'active' | 'resolved'
  createdAt: string
}
```

## 7. 变更管理 API

### 7.1 变更请求CRUD操作

```typescript
// GET /api/v1/changes
interface GetChangesQuery {
  page?: number
  limit?: number
  search?: string
  status?: ChangeStatus
  priority?: ChangePriority
  type?: ChangeType
  requesterId?: string
  assigneeId?: string
  scheduledFrom?: string
  scheduledTo?: string
}

// POST /api/v1/changes
interface CreateChangeRequest {
  title: string
  description: string
  type: ChangeType
  priority: ChangePriority
  category: string
  reason: string
  implementationPlan: string
  backoutPlan: string
  testPlan: string
  affectedServices: string[]
  scheduledStartTime: string
  scheduledEndTime: string
  assigneeId?: string
  approvers: string[]
  riskAssessment: RiskAssessment
}

enum ChangeStatus {
  DRAFT = 'draft',
  SUBMITTED = 'submitted',
  UNDER_REVIEW = 'under_review',
  APPROVED = 'approved',
  REJECTED = 'rejected',
  SCHEDULED = 'scheduled',
  IN_PROGRESS = 'in_progress',
  COMPLETED = 'completed',
  FAILED = 'failed',
  CANCELLED = 'cancelled'
}

enum ChangeType {
  STANDARD = 'standard',
  NORMAL = 'normal',
  EMERGENCY = 'emergency'
}

enum ChangePriority {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  CRITICAL = 'critical'
}

interface RiskAssessment {
  level: 'low' | 'medium' | 'high'
  impact: string
  likelihood: string
  mitigation: string
}

interface Change {
  id: string
  changeNumber: string
  title: string
  description: string
  type: ChangeType
  status: ChangeStatus
  priority: ChangePriority
  category: string
  reason: string
  requester: User
  assignee?: User
  implementationPlan: string
  backoutPlan: string
  testPlan: string
  affectedServices: Service[]
  scheduledStartTime: string
  scheduledEndTime: string
  actualStartTime?: string
  actualEndTime?: string
  approvals: ChangeApproval[]
  riskAssessment: RiskAssessment
  postImplementationReview?: string
  createdAt: string
  updatedAt: string
}

interface ChangeApproval {
  id: string
  approver: User
  status: 'pending' | 'approved' | 'rejected'
  comment?: string
  approvedAt?: string
}
```

## 8. 配置管理数据库 (CMDB) API

### 8.1 配置项管理

```typescript
// GET /api/v1/cmdb/items
interface GetCIQuery {
  page?: number
  limit?: number
  search?: string
  type?: string
  status?: CIStatus
  environment?: string
  owner?: string
}

// POST /api/v1/cmdb/items
interface CreateCIRequest {
  name: string
  type: string
  description?: string
  status: CIStatus
  environment: string
  owner: string
  attributes: Record<string, any>
  relationships?: CIRelationship[]
}

enum CIStatus {
  ACTIVE = 'active',
  INACTIVE = 'inactive',
  RETIRED = 'retired',
  UNDER_MAINTENANCE = 'under_maintenance'
}

interface ConfigurationItem {
  id: string
  name: string
  type: string
  description?: string
  status: CIStatus
  environment: string
  owner: User
  attributes: Record<string, any>
  relationships: CIRelationship[]
  changeHistory: CIChangeHistory[]
  createdAt: string
  updatedAt: string
}

interface CIRelationship {
  id: string
  type: 'depends_on' | 'part_of' | 'connects_to' | 'hosted_on'
  targetCI: string
  description?: string
}

interface CIChangeHistory {
  id: string
  changeId?: string
  field: string
  oldValue: any
  newValue: any
  changedBy: User
  changedAt: string
}
```

### 8.2 服务管理

```typescript
// GET /api/v1/services
interface GetServicesQuery {
  page?: number
  limit?: number
  search?: string
  status?: ServiceStatus
  criticality?: ServiceCriticality
  owner?: string
}

// POST /api/v1/services
interface CreateServiceRequest {
  name: string
  description: string
  status: ServiceStatus
  criticality: ServiceCriticality
  owner: string
  dependencies: string[]
  slaTargets: SLATarget[]
}

enum ServiceStatus {
  OPERATIONAL = 'operational',
  DEGRADED = 'degraded',
  PARTIAL_OUTAGE = 'partial_outage',
  MAJOR_OUTAGE = 'major_outage',
  MAINTENANCE = 'maintenance'
}

enum ServiceCriticality {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  CRITICAL = 'critical'
}

interface Service {
  id: string
  name: string
  description: string
  status: ServiceStatus
  criticality: ServiceCriticality
  owner: User
  dependencies: Service[]
  configurationItems: ConfigurationItem[]
  slaTargets: SLATarget[]
  currentSLA: SLAMetrics
  createdAt: string
  updatedAt: string
}

interface SLATarget {
  metric: 'availability' | 'response_time' | 'resolution_time'
  target: number
  unit: string
}

interface SLAMetrics {
  availability: number
  averageResponseTime: number
  averageResolutionTime: number
  period: string
}
```

## 9. 工作流引擎 API

### 9.1 工作流定义

```typescript
// GET /api/v1/workflows
interface GetWorkflowsQuery {
  page?: number
  limit?: number
  search?: string
  category?: string
  status?: 'active' | 'inactive'
}

// POST /api/v1/workflows
interface CreateWorkflowRequest {
  name: string
  description: string
  category: string
  definition: WorkflowDefinition
  isActive: boolean
}

interface WorkflowDefinition {
  nodes: WorkflowNode[]
  edges: WorkflowEdge[]
  startNode: string
  endNodes: string[]
}

interface WorkflowNode {
  id: string
  type: 'start' | 'end' | 'task' | 'decision' | 'parallel' | 'merge'
  name: string
  config: Record<string, any>
  assignee?: string
  dueDate?: string
}

interface WorkflowEdge {
  id: string
  source: string
  target: string
  condition?: string
  label?: string
}

interface Workflow {
  id: string
  name: string
  description: string
  category: string
  version: number
  definition: WorkflowDefinition
  isActive: boolean
  createdBy: User
  createdAt: string
  updatedAt: string
}
```

### 9.2 工作流实例

```typescript
// GET /api/v1/workflow-instances
interface GetWorkflowInstancesQuery {
  page?: number
  limit?: number
  workflowId?: string
  status?: WorkflowInstanceStatus
  assignee?: string
}

// POST /api/v1/workflow-instances
interface StartWorkflowRequest {
  workflowId: string
  context: Record<string, any>
  priority?: 'low' | 'medium' | 'high'
}

enum WorkflowInstanceStatus {
  RUNNING = 'running',
  COMPLETED = 'completed',
  FAILED = 'failed',
  CANCELLED = 'cancelled',
  SUSPENDED = 'suspended'
}

interface WorkflowInstance {
  id: string
  workflowId: string
  workflow: Workflow
  status: WorkflowInstanceStatus
  context: Record<string, any>
  currentNodes: string[]
  completedNodes: string[]
  tasks: WorkflowTask[]
  startedAt: string
  completedAt?: string
  startedBy: User
}

interface WorkflowTask {
  id: string
  instanceId: string
  nodeId: string
  name: string
  status: 'pending' | 'in_progress' | 'completed' | 'failed'
  assignee?: User
  dueDate?: string
  completedAt?: string
  result?: Record<string, any>
}
```

## 10. 报表分析 API

### 10.1 仪表板数据

```typescript
// GET /api/v1/dashboard/overview
interface DashboardOverview {
  tickets: {
    total: number
    open: number
    inProgress: number
    resolved: number
    overdue: number
  }
  incidents: {
    total: number
    active: number
    resolved: number
    p1Count: number
  }
  changes: {
    total: number
    pending: number
    approved: number
    completed: number
  }
  slaCompliance: {
    overall: number
    byPriority: Record<string, number>
  }
}

// GET /api/v1/dashboard/metrics
interface GetMetricsQuery {
  period: 'day' | 'week' | 'month' | 'quarter' | 'year'
  from?: string
  to?: string
  granularity?: 'hour' | 'day' | 'week' | 'month'
}

interface MetricsResponse {
  ticketVolume: TimeSeriesData[]
  resolutionTime: TimeSeriesData[]
  slaCompliance: TimeSeriesData[]
  customerSatisfaction: TimeSeriesData[]
}

interface TimeSeriesData {
  timestamp: string
  value: number
  label?: string
}
```

### 10.2 报表生成

```typescript
// GET /api/v1/reports
interface GetReportsQuery {
  page?: number
  limit?: number
  type?: ReportType
  status?: 'draft' | 'published'
}

// POST /api/v1/reports
interface CreateReportRequest {
  name: string
  type: ReportType
  description?: string
  parameters: ReportParameters
  schedule?: ReportSchedule
}

enum ReportType {
  TICKET_SUMMARY = 'ticket_summary',
  SLA_COMPLIANCE = 'sla_compliance',
  PERFORMANCE_METRICS = 'performance_metrics',
  CUSTOMER_SATISFACTION = 'customer_satisfaction',
  TREND_ANALYSIS = 'trend_analysis'
}

interface ReportParameters {
  dateRange: {
    from: string
    to: string
  }
  filters: Record<string, any>
  groupBy?: string[]
  metrics: string[]
}

interface ReportSchedule {
  frequency: 'daily' | 'weekly' | 'monthly'
  time: string
  recipients: string[]
  format: 'pdf' | 'excel' | 'csv'
}

interface Report {
  id: string
  name: string
  type: ReportType
  description?: string
  parameters: ReportParameters
  schedule?: ReportSchedule
  status: 'draft' | 'published'
  lastGenerated?: string
  createdBy: User
  createdAt: string
  updatedAt: string
}
```

## 11. AI 服务 API

### 11.1 智能分析

```typescript
// POST /api/v1/ai/analyze-ticket
interface AnalyzeTicketRequest {
  title: string
  description: string
}

interface AnalyzeTicketResponse {
  suggestedCategory: string
  suggestedPriority: TicketPriority
  suggestedAssignee?: string
  similarTickets: SimilarTicket[]
  sentiment: 'positive' | 'neutral' | 'negative'
  confidence: number
}

interface SimilarTicket {
  id: string
  title: string
  similarity: number
  resolution?: string
}

// POST /api/v1/ai/predict-resolution-time
interface PredictResolutionTimeRequest {
  ticketId: string
}

interface PredictResolutionTimeResponse {
  estimatedHours: number
  confidence: number
  factors: string[]
}

// POST /api/v1/ai/suggest-solution
interface SuggestSolutionRequest {
  ticketId: string
  description: string
}

interface SuggestSolutionResponse {
  suggestions: Solution[]
  knowledgeBaseArticles: KnowledgeBaseArticle[]
}

interface Solution {
  title: string
  description: string
  steps: string[]
  confidence: number
  source: string
}

interface KnowledgeBaseArticle {
  id: string
  title: string
  summary: string
  url: string
  relevance: number
}
```

### 11.2 预测性维护

```typescript
// GET /api/v1/ai/predictive-maintenance
interface GetPredictiveMaintenanceQuery {
  serviceId?: string
  ciId?: string
  riskLevel?: 'low' | 'medium' | 'high'
}

interface PredictiveMaintenanceResponse {
  predictions: MaintenancePrediction[]
  recommendations: MaintenanceRecommendation[]
}

interface MaintenancePrediction {
  id: string
  targetId: string
  targetType: 'service' | 'ci'
  targetName: string
  riskLevel: 'low' | 'medium' | 'high'
  probability: number
  predictedFailureDate: string
  confidence: number
  factors: string[]
}

interface MaintenanceRecommendation {
  id: string
  title: string
  description: string
  priority: 'low' | 'medium' | 'high'
  estimatedEffort: string
  expectedBenefit: string
}
```

## 12. 通知服务 API

### 12.1 通知管理

```typescript
// GET /api/v1/notifications
interface GetNotificationsQuery {
  page?: number
  limit?: number
  read?: boolean
  type?: NotificationType
}

// POST /api/v1/notifications
interface CreateNotificationRequest {
  recipients: string[]
  type: NotificationType
  title: string
  message: string
  data?: Record<string, any>
  channels: NotificationChannel[]
  priority?: 'low' | 'medium' | 'high'
}

enum NotificationType {
  TICKET_ASSIGNED = 'ticket_assigned',
  TICKET_STATUS_CHANGED = 'ticket_status_changed',
  INCIDENT_CREATED = 'incident_created',
  SLA_BREACH_WARNING = 'sla_breach_warning',
  CHANGE_APPROVED = 'change_approved',
  SYSTEM_ALERT = 'system_alert'
}

enum NotificationChannel {
  EMAIL = 'email',
  SMS = 'sms',
  PUSH = 'push',
  SLACK = 'slack',
  TEAMS = 'teams'
}

interface Notification {
  id: string
  recipient: User
  type: NotificationType
  title: string
  message: string
  data?: Record<string, any>
  channels: NotificationChannel[]
  priority: 'low' | 'medium' | 'high'
  read: boolean
  readAt?: string
  createdAt: string
}
```

## 13. 系统管理 API

### 13.1 系统配置

```typescript
// GET /api/v1/admin/settings
interface SystemSettings {
  general: {
    siteName: string
    siteUrl: string
    timezone: string
    language: string
  }
  email: {
    smtpHost: string
    smtpPort: number
    smtpUser: string
    smtpSsl: boolean
    fromAddress: string
    fromName: string
  }
  sla: {
    businessHours: BusinessHours
    holidays: Holiday[]
    defaultTargets: Record<string, number>
  }
  security: {
    passwordPolicy: PasswordPolicy
    sessionTimeout: number
    maxLoginAttempts: number
    lockoutDuration: number
  }
}

interface BusinessHours {
  monday: TimeRange
  tuesday: TimeRange
  wednesday: TimeRange
  thursday: TimeRange
  friday: TimeRange
  saturday?: TimeRange
  sunday?: TimeRange
}

interface TimeRange {
  start: string // HH:mm
  end: string   // HH:mm
}

interface Holiday {
  date: string
  name: string
  recurring: boolean
}

interface PasswordPolicy {
  minLength: number
  requireUppercase: boolean
  requireLowercase: boolean
  requireNumbers: boolean
  requireSpecialChars: boolean
  maxAge: number
  historyCount: number
}
```

### 13.2 审计日志

```typescript
// GET /api/v1/admin/audit-logs
interface GetAuditLogsQuery {
  page?: number
  limit?: number
  userId?: string
  action?: string
  resource?: string
  from?: string
  to?: string
}

interface AuditLog {
  id: string
  user: User
  action: string
  resource: string
  resourceId?: string
  details: Record<string, any>
  ipAddress: string
  userAgent: string
  timestamp: string
}
```

## 14. 文件管理 API

### 14.1 文件上传下载

```typescript
// POST /api/v1/files/upload
interface UploadFileRequest {
  file: File
  category?: string
  description?: string
}

interface UploadFileResponse {
  id: string
  filename: string
  originalName: string
  size: number
  mimeType: string
  url: string
}

// GET /api/v1/files/:id
// DELETE /api/v1/files/:id

interface Attachment {
  id: string
  filename: string
  originalName: string
  size: number
  mimeType: string
  url: string
  uploadedBy: User
  uploadedAt: string
}
```

## 15. 搜索 API

### 15.1 全局搜索

```typescript
// GET /api/v1/search
interface GlobalSearchQuery {
  q: string
  type?: 'all' | 'tickets' | 'incidents' | 'problems' | 'changes' | 'users' | 'ci'
  limit?: number
  page?: number
}

interface SearchResult {
  type: string
  id: string
  title: string
  description?: string
  url: string
  relevance: number
  highlights: string[]
}

interface GlobalSearchResponse {
  results: SearchResult[]
  total: number
  facets: SearchFacet[]
}

interface SearchFacet {
  field: string
  values: Array<{
    value: string
    count: number
  }>
}
```

这个API规范文档提供了ITSM系统所有核心功能模块的详细接口定义，包括数据模型、请求响应格式、错误处理等，为前后端开发提供了完整的接口契约。