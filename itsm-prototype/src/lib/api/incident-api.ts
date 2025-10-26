import { httpClient } from './http-client';
import { PaginationRequest, PaginationResponse } from './api-config';

// 事件管理API接口
export interface Incident {
  id: number;
  title: string;
  description: string;
  status: string;
  priority: string;
  source: string;
  type: string;
  incident_number: string;
  is_major_incident: boolean;
  reporter?: {
    id: number;
    name: string;
  };
  assignee?: {
    id: number;
    name: string;
  };
  // 配置项关联
  configuration_item_id?: number;
  configuration_item?: {
    id: number;
    name: string;
    type: string;
    status: string;
    description?: string;
  };
  // 阿里云相关字段
  alibaba_cloud_instance_id?: string;
  alibaba_cloud_region?: string;
  alibaba_cloud_service?: string;
  alibaba_cloud_alert_data?: unknown;
  alibaba_cloud_metrics?: unknown;
  // 安全事件相关字段
  security_event_type?: string;
  security_event_source_ip?: string;
  security_event_target?: string;
  security_event_details?: unknown;
  // 时间字段
  detected_at?: string;
  confirmed_at?: string;
  resolved_at?: string;
  closed_at?: string;
  created_at: string;
  updated_at: string;
  // 新增字段
  category?: string;
  subcategory?: string;
  resolution?: string;
}

export interface CreateIncidentRequest {
  title: string;
  description?: string;
  priority: string;
  source: string;
  type: string;
  is_major_incident?: boolean;
  assignee_id?: number;
  configuration_item_id?: number;
  category?: string;
  subcategory?: string;
  // 阿里云相关字段
  alibaba_cloud_instance_id?: string;
  alibaba_cloud_region?: string;
  alibaba_cloud_service?: string;
  alibaba_cloud_alert_data?: unknown;
  alibaba_cloud_metrics?: unknown;
  // 安全事件相关字段
  security_event_type?: string;
  security_event_source_ip?: string;
  security_event_target?: string;
  security_event_details?: unknown;
  // 关联的配置项
  affected_configuration_item_ids?: number[];
  form_fields?: Record<string, string | number | boolean>;
}

// 根因分析接口
export interface RootCauseAnalysis {
  id?: number;
  incident_id: number;
  analysis_method: string; // "5-whys" | "fishbone" | "timeline" | "fault-tree"
  root_cause: string;
  contributing_factors: string[];
  evidence: string[];
  preventive_actions: string[];
  status: "draft" | "in-progress" | "completed" | "approved";
  analyst_id?: number;
  analyst_name?: string;
  created_at?: string;
  updated_at?: string;
}

// 影响评估接口
export interface ImpactAssessment {
  id?: number;
  incident_id: number;
  business_impact: "low" | "medium" | "high" | "critical";
  technical_impact: "low" | "medium" | "high" | "critical";
  affected_services: string[];
  affected_users_count: number;
  financial_impact: number;
  reputation_impact: "low" | "medium" | "high" | "critical";
  compliance_impact: boolean;
  assessment_notes: string;
  assessor_id?: number;
  assessor_name?: string;
  created_at?: string;
  updated_at?: string;
}

// 事件分类接口
export interface IncidentClassification {
  id?: number;
  incident_id: number;
  category: string;
  subcategory: string;
  service_type: string;
  failure_type: string;
  urgency: "low" | "medium" | "high" | "critical";
  impact: "low" | "medium" | "high" | "critical";
  classification_confidence: number; // 0-100
  auto_classified: boolean;
  classifier_id?: number;
  classifier_name?: string;
  created_at?: string;
  updated_at?: string;
}

export interface CreateRootCauseAnalysisRequest {
  incident_id: number;
  analysis_method: string;
  root_cause: string;
  contributing_factors: string[];
  evidence: string[];
  preventive_actions: string[];
  status: string;
}

export interface CreateImpactAssessmentRequest {
  incident_id: number;
  business_impact: string;
  technical_impact: string;
  affected_services: string[];
  affected_users_count: number;
  financial_impact: number;
  reputation_impact: string;
  compliance_impact: boolean;
  assessment_notes: string;
}

export interface CreateIncidentClassificationRequest {
  incident_id: number;
  category: string;
  subcategory: string;
  service_type: string;
  failure_type: string;
  urgency: string;
  impact: string;
  classification_confidence?: number;
  auto_classified?: boolean;
}

export interface UpdateIncidentRequest {
  title?: string;
  description?: string;
  priority?: string;
  type?: string;
  status?: string;
  assignee_id?: number;
  is_major_incident?: boolean;
  category?: string;
  subcategory?: string;
  resolution?: string;
  resolution_notes?: string;
  suspend_reason?: string;
  resolved_at?: string;
  closed_at?: string;
  form_fields?: Record<string, string | number | boolean>;
}

export interface UpdateIncidentStatusRequest {
  status: string;
  resolution_note?: string;
  suspend_reason?: string;
}

export interface ListIncidentsRequest extends PaginationRequest {
  status?: string;
  priority?: string;
  source?: string;
  type?: string;
  category?: string;
  assignee_id?: number;
  is_major_incident?: boolean;
  keyword?: string;
  date_from?: string;
  date_to?: string;
  [key: string]: unknown;
}

export interface ListIncidentsResponse extends PaginationResponse<Incident> {
  incidents: Incident[]; // 保持向后兼容
}

export interface IncidentMetrics {
  total_incidents: number;
  open_incidents: number;
  critical_incidents: number;
  major_incidents: number;
  avg_resolution_time: number;
  mtta: number;
  mttr: number;
}

// 阿里云告警事件
export interface AlibabaCloudAlertRequest {
  alert_id: string;
  alert_name: string;
  alert_description: string;
  alert_level: string;
  instance_id: string;
  region: string;
  service: string;
  metrics: unknown;
  alert_data: unknown;
  detected_at: string;
}

// 安全事件
export interface SecurityEventRequest {
  event_id: string;
  event_type: string;
  event_name: string;
  event_description: string;
  source_ip: string;
  target: string;
  severity: string;
  event_details: unknown;
  detected_at: string;
}

// 云产品事件
export interface CloudProductEventRequest {
  event_id: string;
  event_type: string;
  event_name: string;
  event_description: string;
  product: string;
  instance_id: string;
  region: string;
  event_data: unknown;
  detected_at: string;
}

// 事件管理API类
export class IncidentAPI {
  // 获取事件列表
  static async listIncidents(params: ListIncidentsRequest = {}): Promise<ListIncidentsResponse> {
    console.log('IncidentAPI.listIncidents called with params:', params);
    try {
      const response = await httpClient.get<ListIncidentsResponse>('/api/v1/incidents', params);
      console.log('IncidentAPI.listIncidents response:', response);
      return response;
    } catch (error) {
      console.error('IncidentAPI.listIncidents error:', error);
      throw error;
    }
  }

  // 获取事件详情
  static async getIncident(id: number): Promise<Incident> {
    const response = await httpClient.get<Incident>(`/api/v1/incidents/${id}`);
    return response;
  }

  // 创建事件
  static async createIncident(data: CreateIncidentRequest): Promise<Incident> {
    const response = await httpClient.post<Incident>('/api/v1/incidents', data);
    return response;
  }

  // 更新事件
  static async updateIncident(id: number, data: UpdateIncidentRequest): Promise<Incident> {
    const response = await httpClient.patch<Incident>(`/api/v1/incidents/${id}`, data);
    return response;
  }

  // 更新事件状态
  static async updateIncidentStatus(id: number, data: UpdateIncidentStatusRequest): Promise<Incident> {
    const response = await httpClient.put<Incident>(`/api/v1/incidents/${id}/status`, data);
    return response;
  }

  // 添加评论
  static async addComment(id: number, data: { text: string }): Promise<Incident> {
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
      const response = await httpClient.get<RootCauseAnalysis>(`/api/incidents/${incidentId}/root-cause`);
      return response;
    } catch (error) {
      console.error('IncidentAPI.getRootCauseAnalysis error:', error);
      throw error;
    }
  }

  static async createRootCauseAnalysis(request: CreateRootCauseAnalysisRequest): Promise<RootCauseAnalysis> {
    try {
      const response = await httpClient.post<RootCauseAnalysis>('/api/incidents/root-cause', request);
      return response;
    } catch (error) {
      console.error('IncidentAPI.createRootCauseAnalysis error:', error);
      throw error;
    }
  }

  static async updateRootCauseAnalysis(id: number, request: Partial<CreateRootCauseAnalysisRequest>): Promise<RootCauseAnalysis> {
    try {
      const response = await httpClient.put<RootCauseAnalysis>(`/api/incidents/root-cause/${id}`, request);
      return response;
    } catch (error) {
      console.error('IncidentAPI.updateRootCauseAnalysis error:', error);
      throw error;
    }
  }

  // 影响评估相关API
  static async getImpactAssessment(incidentId: number): Promise<ImpactAssessment> {
    try {
      const response = await httpClient.get<ImpactAssessment>(`/api/incidents/${incidentId}/impact-assessment`);
      return response;
    } catch (error) {
      console.error('IncidentAPI.getImpactAssessment error:', error);
      throw error;
    }
  }

  static async createImpactAssessment(request: CreateImpactAssessmentRequest): Promise<ImpactAssessment> {
    try {
      const response = await httpClient.post<ImpactAssessment>('/api/incidents/impact-assessment', request);
      return response;
    } catch (error) {
      console.error('IncidentAPI.createImpactAssessment error:', error);
      throw error;
    }
  }

  static async updateImpactAssessment(id: number, request: Partial<CreateImpactAssessmentRequest>): Promise<ImpactAssessment> {
    try {
      const response = await httpClient.put<ImpactAssessment>(`/api/incidents/impact-assessment/${id}`, request);
      return response;
    } catch (error) {
      console.error('IncidentAPI.updateImpactAssessment error:', error);
      throw error;
    }
  }

  // 事件分类相关API
  static async getIncidentClassification(incidentId: number): Promise<IncidentClassification> {
    try {
      const response = await httpClient.get<IncidentClassification>(`/api/incidents/${incidentId}/classification`);
      return response;
    } catch (error) {
      console.error('IncidentAPI.getIncidentClassification error:', error);
      throw error;
    }
  }

  static async createIncidentClassification(request: CreateIncidentClassificationRequest): Promise<IncidentClassification> {
    try {
      const response = await httpClient.post<IncidentClassification>('/api/incidents/classification', request);
      return response;
    } catch (error) {
      console.error('IncidentAPI.createIncidentClassification error:', error);
      throw error;
    }
  }

  static async updateIncidentClassification(id: number, request: Partial<CreateIncidentClassificationRequest>): Promise<IncidentClassification> {
    try {
      const response = await httpClient.put<IncidentClassification>(`/api/incidents/classification/${id}`, request);
      return response;
    } catch (error) {
      console.error('IncidentAPI.updateIncidentClassification error:', error);
      throw error;
    }
  }

  // AI辅助分析
  static async analyzeIncidentWithAI(incidentId: number): Promise<{
    suggested_classification: Partial<IncidentClassification>;
    suggested_root_causes: string[];
    similar_incidents: Incident[];
  }> {
    try {
      const response = await httpClient.post<{
        suggested_classification: Partial<IncidentClassification>;
        suggested_root_causes: string[];
        similar_incidents: Incident[];
      }>(`/api/incidents/${incidentId}/ai-analysis`);
      return response;
    } catch (error) {
      console.error('IncidentAPI.analyzeIncidentWithAI error:', error);
      throw error;
    }
  }

  // 获取可关联的配置项列表
  static async getConfigurationItems(search?: string, type?: string, status?: string): Promise<Array<{
    id: number;
    name: string;
    type: string;
    status: string;
    description?: string;
  }>> {
    const params: Record<string, string> = {};
    if (search) params.search = search;
    if (type) params.type = type;
    if (status) params.status = status;
    
    const response = await httpClient.get<Array<{
      id: number;
      name: string;
      type: string;
      status: string;
      description?: string;
    }>>('/api/v1/incidents/configuration-items', params);
    return response;
  }

  // 从阿里云告警创建事件
  static async createIncidentFromAlibabaCloudAlert(data: AlibabaCloudAlertRequest): Promise<Incident> {
    const response = await httpClient.post<Incident>('/api/v1/incidents/alibaba-cloud-alert', data);
    return response;
  }

  // 从安全事件创建事件
  static async createIncidentFromSecurityEvent(data: SecurityEventRequest): Promise<Incident> {
    const response = await httpClient.post<Incident>('/api/v1/incidents/security-event', data);
    return response;
  }

  // 从云产品事件创建事件
  static async createIncidentFromCloudProductEvent(data: CloudProductEventRequest): Promise<Incident> {
    const response = await httpClient.post<Incident>('/api/v1/incidents/cloud-product-event', data);
    return response;
  }

  // 模拟阿里云告警事件
  static async simulateAlibabaCloudAlert(): Promise<Incident> {
    const mockAlert: AlibabaCloudAlertRequest = {
      alert_id: `alert_${Date.now()}`,
      alert_name: 'CPU使用率过高告警',
      alert_description: '实例 i-bp1abcdefg 的CPU使用率在过去15分钟内持续高于95%',
      alert_level: 'critical',
      instance_id: 'i-bp1abcdefg',
      region: 'cn-hangzhou',
      service: 'ecs',
      metrics: {
        cpu_usage: 95.5,
        memory_usage: 78.2,
        disk_usage: 45.1
      },
      alert_data: {
        alert_rule: 'CPU使用率 > 90%',
        threshold: 90,
        current_value: 95.5,
        duration: '15分钟'
      },
      detected_at: new Date().toISOString()
    };

    return this.createIncidentFromAlibabaCloudAlert(mockAlert);
  }

  // 模拟安全事件
  static async simulateSecurityEvent(): Promise<Incident> {
    const mockSecurityEvent: SecurityEventRequest = {
      event_id: `security_${Date.now()}`,
      event_type: 'SSH暴力破解',
      event_name: '检测到可疑的SSH登录尝试',
      event_description: '安全系统检测到来自未知IP地址 (47.98.x.x) 对生产环境服务器的多次SSH登录失败尝试',
      source_ip: '47.98.x.x',
      target: 'PROD-BASTION-HOST',
      severity: 'high',
      event_details: {
        failed_attempts: 15,
        time_window: '10分钟',
        target_port: 22,
        attack_type: 'brute_force'
      },
      detected_at: new Date().toISOString()
    };

    return this.createIncidentFromSecurityEvent(mockSecurityEvent);
  }

  // 模拟云产品事件
  static async simulateCloudProductEvent(): Promise<Incident> {
    const mockCloudEvent: CloudProductEventRequest = {
      event_id: `cloud_${Date.now()}`,
      event_type: 'RDS主备同步延迟',
      event_name: '数据库主备同步延迟告警',
      event_description: '监控系统告警，生产数据库（RDS）主备同步延迟超过阈值（30秒）',
      product: 'rds',
      instance_id: 'rm-bp1abcdefg',
      region: 'cn-hangzhou',
      event_data: {
        sync_delay: 45,
        threshold: 30,
        master_status: 'running',
        slave_status: 'running'
      },
      detected_at: new Date().toISOString()
    };

    return this.createIncidentFromCloudProductEvent(mockCloudEvent);
  }
}