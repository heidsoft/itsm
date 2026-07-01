import { httpClient } from './http-client';
import {
  API_URLS,
  ListQueryParams,
  normalizePaginationParams,
  normalizeDateRangeParams,
  PaginationResponse,
} from './types';

// 事件管理API接口
export interface Incident {
  id: number;
  title: string;
  description: string;
  status: string;
  priority: string;
  severity: string;
  source: string;
  type: string;
  // 支持 camelCase 和 snake_case 两种格式
  incidentNumber?: string;
  incident_number?: string;
  isMajorIncident?: boolean;
  reporter?: {
    id: number;
    name: string;
  };
  reporter_id?: number;
  assignee?: {
    id: number;
    name: string;
  };
  assignee_id?: number;
  // 配置项关联
  configurationItemId?: number;
  configurationItem?: {
    id: number;
    name: string;
    type: string;
    status: string;
    description?: string;
  };
  // 阿里云相关字段
  alibabaCloudInstanceId?: string;
  alibabaCloudRegion?: string;
  alibabaCloudService?: string;
  alibabaCloudAlertData?: unknown;
  alibabaCloudMetrics?: unknown;
  // 安全事件相关字段
  securityEventType?: string;
  securityEventSourceIp?: string;
  securityEventTarget?: string;
  securityEventDetails?: unknown;
  // 时间字段 - 同时支持两种格式
  detectedAt?: string;
  detected_at?: string;
  confirmedAt?: string;
  confirmed_at?: string;
  resolvedAt?: string;
  resolved_at?: string;
  closedAt?: string;
  closed_at?: string;
  escalatedAt?: string;
  escalated_at?: string;
  createdAt?: string;
  created_at?: string;
  updatedAt?: string;
  updated_at?: string;
  // 新增字段
  category?: string;
  subcategory?: string;
  resolution?: string;
  resolutionCode?: string;
  resolution_code?: string;
  problemId?: number; // 关联的问题ID
  problem_id?: number;
  escalationLevel?: number;
  escalation_level?: number;
  impactAnalysis?: {
    businessImpact?: {
      affectedUsers?: number;
      revenueImpact?: number;
      serviceAvailability?: number;
    };
    technicalImpact?: string;
    affectedUsers?: number;
    affectedServices?: string[];
    estimatedResolutionTime?: number;
    isOverdue?: boolean;
    hoursSinceCreation?: number;
    timeImpact?: {
      isOverdue?: boolean;
      hoursSinceCreation?: number;
      responseDeadline?: string;
      resolutionDeadline?: string;
    };
    metrics?: {
      totalCount?: number;
      criticalCount?: number;
      resolvedCount?: number;
      averageValue?: number;
      maxValue?: number;
      minValue?: number;
    };
  };
}

export interface CreateIncidentRequest {
  title: string;
  description?: string;
  priority: string;
  source: string;
  type: string;
  isMajorIncident?: boolean;
  // 同时支持 camelCase 和 snake_case
  assigneeId?: number;
  assignee_id?: number;
  assignedTo?: number; // Added for compatibility with UI forms
  assigned_to?: number; // Backend expects this
  configurationItemId?: number;
  configuration_item_id?: number;
  category?: string;
  subcategory?: string;
  impact?: string; // Added for UI
  urgency?: string; // Added for UI
  // 阿里云相关字段
  alibabaCloudInstanceId?: string;
  alibabaCloudRegion?: string;
  alibabaCloudService?: string;
  alibabaCloudAlertData?: unknown;
  alibabaCloudMetrics?: unknown;
  // 安全事件相关字段
  securityEventType?: string;
  securityEventSourceIp?: string;
  securityEventTarget?: string;
  securityEventDetails?: unknown;
  // 关联的配置项
  affectedConfigurationItemIds?: number[];
  formFields?: Record<string, string | number | boolean>;
}

// 根因分析接口
export interface RootCauseAnalysis {
  id?: number;
  incidentId: number;
  analysisMethod: string; // "5-whys" | "fishbone" | "timeline" | "fault-tree"
  rootCause: string;
  contributingFactors: string[];
  evidence: string[];
  preventiveActions: string[];
  status: 'draft' | 'in-progress' | 'completed' | 'approved';
  analystId?: number;
  analystName?: string;
  createdAt?: string;
  updatedAt?: string;
}

// 影响评估接口
export interface ImpactAssessment {
  id?: number;
  incidentId: number;
  businessImpact: 'low' | 'medium' | 'high' | 'critical';
  technicalImpact: 'low' | 'medium' | 'high' | 'critical';
  affectedServices: string[];
  affectedUsersCount: number;
  financialImpact: number;
  reputationImpact: 'low' | 'medium' | 'high' | 'critical';
  complianceImpact: boolean;
  assessmentNotes: string;
  assessorId?: number;
  assessorName?: string;
  createdAt?: string;
  updatedAt?: string;
}

// 事件分类接口
export interface IncidentClassification {
  id?: number;
  incidentId: number;
  category: string;
  subcategory: string;
  serviceType: string;
  failureType: string;
  urgency: 'low' | 'medium' | 'high' | 'critical';
  impact: 'low' | 'medium' | 'high' | 'critical';
  classificationConfidence: number; // 0-100
  autoClassified: boolean;
  classifierId?: number;
  classifierName?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface CreateRootCauseAnalysisRequest {
  incidentId: number;
  analysisMethod: string;
  rootCause: string;
  contributingFactors: string[];
  evidence: string[];
  preventiveActions: string[];
  status: string;
}

export interface CreateImpactAssessmentRequest {
  incidentId: number;
  businessImpact: string;
  technicalImpact: string;
  affectedServices: string[];
  affectedUsersCount: number;
  financialImpact: number;
  reputationImpact: string;
  complianceImpact: boolean;
  assessmentNotes: string;
}

export interface CreateIncidentClassificationRequest {
  incidentId: number;
  category: string;
  subcategory: string;
  serviceType: string;
  failureType: string;
  urgency: string;
  impact: string;
  classificationConfidence?: number;
  autoClassified?: boolean;
}

export interface UpdateIncidentRequest {
  title?: string;
  description?: string;
  priority?: string;
  type?: string;
  status?: string;
  assigneeId?: number;
  isMajorIncident?: boolean;
  category?: string;
  subcategory?: string;
  resolution?: string;
  resolutionNotes?: string;
  suspendReason?: string;
  resolvedAt?: string;
  closedAt?: string;
  formFields?: Record<string, string | number | boolean>;
  /** 版本号（用于乐观锁冲突检测） */
  version?: number;
}

export interface UpdateIncidentStatusRequest {
  status: string;
  resolutionNote?: string;
  suspendReason?: string;
}

export interface ListIncidentsRequest extends ListQueryParams {
  status?: string;
  priority?: string;
  source?: string;
  type?: string;
  category?: string;
  assigneeId?: number;
  isMajorIncident?: boolean;
  keyword?: string;
  dateFrom?: string;
  dateTo?: string;
  [key: string]: unknown;
}

export interface ListIncidentsResponse extends PaginationResponse<Incident> {
  incidents: Incident[]; // 保持向后兼容
}

export interface IncidentMetrics {
  // camelCase
  totalIncidents?: number;
  openIncidents?: number;
  criticalIncidents?: number;
  majorIncidents?: number;
  avgResolutionTime?: number;
  mtta?: number;
  mttr?: number;
  // snake_case
  total_incidents?: number;
  open_incidents?: number;
  critical_incidents?: number;
  major_incidents?: number;
  avg_resolution_time?: number;
}

// 阿里云告警事件
export interface AlibabaCloudAlertRequest {
  alertId: string;
  alertName: string;
  alertDescription: string;
  alertLevel: string;
  instanceId: string;
  region: string;
  service: string;
  metrics: unknown;
  alertData: unknown;
  detectedAt: string;
}

// 安全事件
export interface SecurityEventRequest {
  eventId: string;
  eventType: string;
  eventName: string;
  eventDescription: string;
  sourceIp: string;
  target: string;
  severity: string;
  eventDetails: unknown;
  detectedAt: string;
}

// 云产品事件
export interface CloudProductEventRequest {
  eventId: string;
  eventType: string;
  eventName: string;
  eventDescription: string;
  product: string;
  instanceId: string;
  region: string;
  eventData: unknown;
  detectedAt: string;
}

// 事件管理API类
export class IncidentAPI {
  // 获取事件列表
  static async listIncidents(params: ListIncidentsRequest = {}): Promise<ListIncidentsResponse> {
    try {
      // 标准化参数
      const normalizedParams = {
        ...normalizePaginationParams(params),
        ...normalizeDateRangeParams(params),
      };

      // 过滤掉undefined值
      const cleanParams = Object.fromEntries(
        Object.entries(normalizedParams).filter(([_, value]) => value !== undefined)
      );

      const response = await httpClient.get<ListIncidentsResponse>(
        API_URLS.INCIDENTS(),
        cleanParams
      );
      return response;
    } catch (error) {
      console.error('IncidentAPI.listIncidents error:', error);
      throw error;
    }
  }

  // 获取事件详情
  static async getIncident(id: number): Promise<Incident> {
    const response = await httpClient.get<Incident>(API_URLS.INCIDENT(id));
    return response;
  }

  // 创建事件
  static async createIncident(data: CreateIncidentRequest): Promise<Incident> {
    const response = await httpClient.post<Incident>(API_URLS.INCIDENTS(), data);
    return response;
  }

  // 更新事件
  static async updateIncident(id: number, data: UpdateIncidentRequest): Promise<Incident> {
    const response = await httpClient.put<Incident>(`/api/v1/incidents/${id}`, data);
    return response;
  }

  // 更新事件状态
  static async updateIncidentStatus(
    id: number,
    data: UpdateIncidentStatusRequest
  ): Promise<Incident> {
    const response = await httpClient.put<Incident>(`/api/v1/incidents/${id}/status`, data);
    return response;
  }

  /**
   * 解决事件（ITIL 合规）
   * 使用专门的 resolve 端点，而非直接更新状态
   * 要求填写解决方案
   */
  static async resolveIncident(
    id: number,
    data: {
      resolution: string;
      resolution_code?: string;
      root_cause?: string;
      problem_id?: number;
    }
  ): Promise<Incident> {
    const response = await httpClient.post<Incident>(`/api/v1/incidents/${id}/resolve`, data);
    return response;
  }

  // 分配事件
  static async assignIncident(id: number, assigneeId: number): Promise<Incident> {
    const response = await httpClient.post<Incident>(`/api/v1/incidents/${id}/assign`, { assigneeId });
    return response;
  }

  // 添加评论（注意：后端期望 content 字段）
  static async addComment(id: number, data: { content: string }): Promise<Incident> {
    const response = await httpClient.post<Incident>(`/api/v1/incidents/${id}/comments`, data);
    return response;
  }

  // 获取事件指标
  static async getIncidentMetrics(): Promise<IncidentMetrics> {
    const response = await httpClient.get<IncidentMetrics>('/api/v1/incidents/stats');
    return response;
  }

  // 根因分析相关API
  static async getRootCauseAnalysis(incidentId: number): Promise<RootCauseAnalysis> {
    try {
      const response = await httpClient.get<RootCauseAnalysis>(
        `/api/v1/incidents/${incidentId}/root-cause`
      );
      return response;
    } catch (error) {
      console.error('IncidentAPI.getRootCauseAnalysis error:', error);
      throw error;
    }
  }

  static async createRootCauseAnalysis(
    request: CreateRootCauseAnalysisRequest
  ): Promise<RootCauseAnalysis> {
    try {
      const response = await httpClient.post<RootCauseAnalysis>(
        '/api/v1/incidents/root-cause',
        request
      );
      return response;
    } catch (error) {
      console.error('IncidentAPI.createRootCauseAnalysis error:', error);
      throw error;
    }
  }

  static async updateRootCauseAnalysis(
    id: number,
    request: Partial<CreateRootCauseAnalysisRequest>
  ): Promise<RootCauseAnalysis> {
    try {
      const response = await httpClient.put<RootCauseAnalysis>(
        `/api/v1/incidents/root-cause/${id}`,
        request
      );
      return response;
    } catch (error) {
      console.error('IncidentAPI.updateRootCauseAnalysis error:', error);
      throw error;
    }
  }

  // 影响评估相关API
  static async getImpactAssessment(incidentId: number): Promise<ImpactAssessment> {
    try {
      const response = await httpClient.get<ImpactAssessment>(
        `/api/v1/incidents/${incidentId}/impact-assessment`
      );
      return response;
    } catch (error) {
      console.error('IncidentAPI.getImpactAssessment error:', error);
      throw error;
    }
  }

  static async createImpactAssessment(
    request: CreateImpactAssessmentRequest
  ): Promise<ImpactAssessment> {
    try {
      const response = await httpClient.post<ImpactAssessment>(
        '/api/v1/incidents/impact-assessment',
        request
      );
      return response;
    } catch (error) {
      console.error('IncidentAPI.createImpactAssessment error:', error);
      throw error;
    }
  }

  static async updateImpactAssessment(
    id: number,
    request: Partial<CreateImpactAssessmentRequest>
  ): Promise<ImpactAssessment> {
    try {
      const response = await httpClient.put<ImpactAssessment>(
        `/api/v1/incidents/impact-assessment/${id}`,
        request
      );
      return response;
    } catch (error) {
      console.error('IncidentAPI.updateImpactAssessment error:', error);
      throw error;
    }
  }

  // 事件分类相关API
  static async getIncidentClassification(incidentId: number): Promise<IncidentClassification> {
    try {
      const response = await httpClient.get<IncidentClassification>(
        `/api/v1/incidents/${incidentId}/classification`
      );
      return response;
    } catch (error) {
      console.error('IncidentAPI.getIncidentClassification error:', error);
      throw error;
    }
  }

  static async createIncidentClassification(
    request: CreateIncidentClassificationRequest
  ): Promise<IncidentClassification> {
    try {
      const response = await httpClient.post<IncidentClassification>(
        '/api/v1/incidents/classification',
        request
      );
      return response;
    } catch (error) {
      console.error('IncidentAPI.createIncidentClassification error:', error);
      throw error;
    }
  }

  static async updateIncidentClassification(
    id: number,
    request: Partial<CreateIncidentClassificationRequest>
  ): Promise<IncidentClassification> {
    try {
      const response = await httpClient.put<IncidentClassification>(
        `/api/v1/incidents/classification/${id}`,
        request
      );
      return response;
    } catch (error) {
      console.error('IncidentAPI.updateIncidentClassification error:', error);
      throw error;
    }
  }

  // 获取可关联的配置项列表
  static async getConfigurationItems(
    search?: string,
    type?: string,
    status?: string
  ): Promise<
    Array<{
      id: number;
      name: string;
      type: string;
      status: string;
      description?: string;
    }>
  > {
    const params: Record<string, string> = {};
    if (search) params.search = search;
    if (type) params.type = type;
    if (status) params.status = status;

    const response = await httpClient.get<
      Array<{
        id: number;
        name: string;
        type: string;
        status: string;
        description?: string;
      }>
    >('/api/v1/incidents/configuration-items', params);
    return response;
  }

  // 从阿里云告警创建事件
  static async createIncidentFromAlibabaCloudAlert(
    data: AlibabaCloudAlertRequest
  ): Promise<Incident> {
    const response = await httpClient.post<Incident>('/api/v1/incidents/alibaba-cloud-alert', data);
    return response;
  }

  // 从安全事件创建事件
  static async createIncidentFromSecurityEvent(data: SecurityEventRequest): Promise<Incident> {
    const response = await httpClient.post<Incident>('/api/v1/incidents/security-event', data);
    return response;
  }

  // 从云产品事件创建事件
  static async createIncidentFromCloudProductEvent(
    data: CloudProductEventRequest
  ): Promise<Incident> {
    const response = await httpClient.post<Incident>('/api/v1/incidents/cloud-product-event', data);
    return response;
  }

  // 模拟阿里云告警事件
  static async simulateAlibabaCloudAlert(): Promise<Incident> {
    const mockAlert: AlibabaCloudAlertRequest = {
      alertId: `alert_${Date.now()}`,
      alertName: 'CPU使用率过高告警',
      alertDescription: '实例 i-bp1abcdefg 的CPU使用率在过去15分钟内持续高于95%',
      alertLevel: 'critical',
      instanceId: 'i-bp1abcdefg',
      region: 'cn-hangzhou',
      service: 'ecs',
      metrics: {
        cpuUsage: 95.5,
        memoryUsage: 78.2,
        diskUsage: 45.1,
      },
      alertData: {
        alertRule: 'CPU使用率 > 90%',
        threshold: 90,
        currentValue: 95.5,
        duration: '15分钟',
      },
      detectedAt: new Date().toISOString(),
    };

    return this.createIncidentFromAlibabaCloudAlert(mockAlert);
  }

  // 模拟安全事件
  static async simulateSecurityEvent(): Promise<Incident> {
    const mockSecurityEvent: SecurityEventRequest = {
      eventId: `security_${Date.now()}`,
      eventType: 'SSH暴力破解',
      eventName: '检测到可疑的SSH登录尝试',
      eventDescription:
        '安全系统检测到来自未知IP地址 (47.98.x.x) 对生产环境服务器的多次SSH登录失败尝试',
      sourceIp: '47.98.x.x',
      target: 'PROD-BASTION-HOST',
      severity: 'high',
      eventDetails: {
        failedAttempts: 15,
        timeWindow: '10分钟',
        targetPort: 22,
        attackType: 'brute_force',
      },
      detectedAt: new Date().toISOString(),
    };

    return this.createIncidentFromSecurityEvent(mockSecurityEvent);
  }

  // 模拟云产品事件
  static async simulateCloudProductEvent(): Promise<Incident> {
    const mockCloudEvent: CloudProductEventRequest = {
      eventId: `cloud_${Date.now()}`,
      eventType: 'RDS主备同步延迟',
      eventName: '数据库主备同步延迟告警',
      eventDescription: '监控系统告警，生产数据库（RDS）主备同步延迟超过阈值（30秒）',
      product: 'rds',
      instanceId: 'rm-bp1abcdefg',
      region: 'cn-hangzhou',
      eventData: {
        syncDelay: 45,
        threshold: 30,
        masterStatus: 'running',
        slaveStatus: 'running',
      },
      detectedAt: new Date().toISOString(),
    };

    return this.createIncidentFromCloudProductEvent(mockCloudEvent);
  }

  // ==================== 兼容别名 ====================

  /** @deprecated 使用 listIncidents */
  static async getIncidents(params?: ListIncidentsRequest): Promise<ListIncidentsResponse> {
    return this.listIncidents(params);
  }

  /** @deprecated 使用 listIncidents */
  static get incidents() {
    return {
      list: (params?: ListIncidentsRequest) => this.listIncidents(params),
      items: (params?: ListIncidentsRequest) =>
        this.listIncidents(params).then((r: ListIncidentsResponse) => r.incidents || r.data || []),
    };
  }

  // 获取事件活动记录
  static async getIncidentEvents(incidentId: number): Promise<any[]> {
    const response = await httpClient.get<any[]>(`/api/v1/incidents/${incidentId}/events`);
    return response;
  }

  // 获取事件告警
  static async getIncidentAlerts(incidentId: number): Promise<any[]> {
    const response = await httpClient.get<any[]>(`/api/v1/incidents/${incidentId}/alerts`);
    return response;
  }

  // 获取事件指标
  static async getIncidentMetricsData(incidentId: number): Promise<any> {
    const response = await httpClient.get<any>(`/api/v1/incidents/${incidentId}/metrics`);
    return response;
  }

  // 事件升级（reason 为必填，后端 binding:"required"）
  static async escalateIncident(
    id: number,
    data: {
      escalationLevel: number;
      reason: string;
      notifyUsers?: number[];
      autoAssign?: boolean;
    }
  ): Promise<any> {
    const response = await httpClient.post<any>(`/api/v1/incidents/${id}/escalate`, data);
    return response;
  }

  // 删除事件
  static async deleteIncident(id: number): Promise<void> {
    await httpClient.delete(`/api/v1/incidents/${id}`);
  }
}
