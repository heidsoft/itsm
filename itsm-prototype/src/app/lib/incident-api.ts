import { httpClient } from './http-client';

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
  // 阿里云相关字段
  alibaba_cloud_instance_id?: string;
  alibaba_cloud_region?: string;
  alibaba_cloud_service?: string;
  alibaba_cloud_alert_data?: any;
  alibaba_cloud_metrics?: any;
  // 安全事件相关字段
  security_event_type?: string;
  security_event_source_ip?: string;
  security_event_target?: string;
  security_event_details?: any;
  // 时间字段
  detected_at?: string;
  confirmed_at?: string;
  resolved_at?: string;
  closed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateIncidentRequest {
  title: string;
  description?: string;
  priority: string;
  source: string;
  type: string;
  is_major_incident?: boolean;
  assignee_id?: number;
  // 阿里云相关字段
  alibaba_cloud_instance_id?: string;
  alibaba_cloud_region?: string;
  alibaba_cloud_service?: string;
  alibaba_cloud_alert_data?: any;
  alibaba_cloud_metrics?: any;
  // 安全事件相关字段
  security_event_type?: string;
  security_event_source_ip?: string;
  security_event_target?: string;
  security_event_details?: any;
  // 关联的配置项
  affected_configuration_item_ids?: number[];
}

export interface UpdateIncidentRequest {
  title?: string;
  description?: string;
  priority?: string;
  type?: string;
  assignee_id?: number;
  is_major_incident?: boolean;
}

export interface UpdateIncidentStatusRequest {
  status: string;
  resolution_note?: string;
  suspend_reason?: string;
}

export interface ListIncidentsRequest {
  page?: number;
  page_size?: number;
  status?: string;
  priority?: string;
  source?: string;
  type?: string;
  assignee_id?: number;
  is_major_incident?: boolean;
  keyword?: string;
}

export interface ListIncidentsResponse {
  incidents: Incident[];
  total: number;
  page: number;
  page_size: number;
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
  metrics: any;
  alert_data: any;
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
  event_details: any;
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
  event_data: any;
  detected_at: string;
}

// 事件管理API类
export class IncidentAPI {
  // 获取事件列表
  static async listIncidents(params: ListIncidentsRequest = {}): Promise<ListIncidentsResponse> {
    const response = await httpClient.get('/api/incidents', { params });
    return response.data.data;
  }

  // 获取事件详情
  static async getIncident(id: number): Promise<Incident> {
    const response = await httpClient.get(`/api/incidents/${id}`);
    return response.data.data;
  }

  // 创建事件
  static async createIncident(data: CreateIncidentRequest): Promise<Incident> {
    const response = await httpClient.post('/api/incidents', data);
    return response.data.data;
  }

  // 更新事件
  static async updateIncident(id: number, data: UpdateIncidentRequest): Promise<Incident> {
    const response = await httpClient.patch(`/api/incidents/${id}`, data);
    return response.data.data;
  }

  // 更新事件状态
  static async updateIncidentStatus(id: number, data: UpdateIncidentStatusRequest): Promise<Incident> {
    const response = await httpClient.put(`/api/incidents/${id}/status`, data);
    return response.data.data;
  }

  // 获取事件指标
  static async getIncidentMetrics(): Promise<IncidentMetrics> {
    const response = await httpClient.get('/api/incidents/metrics');
    return response.data.data;
  }

  // 从阿里云告警创建事件
  static async createIncidentFromAlibabaCloudAlert(data: AlibabaCloudAlertRequest): Promise<Incident> {
    const response = await httpClient.post('/api/incidents/alibaba-cloud-alert', data);
    return response.data.data;
  }

  // 从安全事件创建事件
  static async createIncidentFromSecurityEvent(data: SecurityEventRequest): Promise<Incident> {
    const response = await httpClient.post('/api/incidents/security-event', data);
    return response.data.data;
  }

  // 从云产品事件创建事件
  static async createIncidentFromCloudProductEvent(data: CloudProductEventRequest): Promise<Incident> {
    const response = await httpClient.post('/api/incidents/cloud-product-event', data);
    return response.data.data;
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