/**
 * BPMN Dashboard API 服务
 * 用于流程监控、审计日志、SLA管理等
 */

import { httpClient } from './http-client';

export interface DashboardMetrics {
  total_processes: number;
  active_instances: number;
  completed_today: number;
  open_tasks: number;
  sla_compliance_rate: number;
  avg_completion_time_minutes: number;
  process_health: ProcessHealth;
  top_processes: ProcessStat[];
  task_distribution: TaskStat[];
  trend_data: TrendPoint[];
}

export interface ProcessHealth {
  healthy: number;
  warning: number;
  critical: number;
  health_score: number;
}

export interface ProcessStat {
  process_definition_key: string;
  total_instances: number;
  running_instances: number;
  completed_instances: number;
  avg_duration_minutes: number;
}

export interface TaskStat {
  status: string;
  count: number;
  percent: number;
}

export interface TrendPoint {
  date: string;
  count: number;
}

export interface ProcessMetrics {
  total_instances: number;
  running_instances: number;
  completed_instances: number;
  terminated_instances: number;
  completion_rate: number;
  sla_compliance_rate: number;
  avg_completion_time_minutes: number;
}

export interface ProcessAuditLog {
  id: number;
  process_instance_id: number;
  process_instance_key: string;
  process_definition_key: string;
  process_definition_id: number;
  activity_id: string;
  activity_name: string;
  activity_type: string;
  action: string;
  user_id: number;
  user_name: string;
  assignee_id: number;
  assignee_name: string;
  variables_before: Record<string, any>;
  variables_after: Record<string, any>;
  comment: string;
  ip_address: string;
  user_agent: string;
  tenant_id: number;
  timestamp: string;
  duration_ms: number;
  metadata: Record<string, any>;
}

export interface QueryAuditLogsRequest {
  tenant_id: number;
  process_instance_id?: number;
  process_definition_key?: string;
  action?: string;
  user_id?: number;
  activity_type?: string;
  start_time?: string;
  end_time?: string;
  page?: number;
  page_size?: number;
}

export interface SLAViolation {
  resource_type: string;
  resource_id: number;
  resource_key: string;
  sla_status: string;
  start_time: string;
  deadline: string;
  elapsed_minutes: number;
  tenant_id: number;
}

export interface TenantBPMNStats {
  total_definitions: number;
  total_instances: number;
  running_instances: number;
  completed_instances: number;
  total_tasks: number;
  open_tasks: number;
}

export interface BottleneckInfo {
  task_name: string;
  total_count: number;
  avg_duration_minutes: number;
  max_duration_minutes: number;
  min_duration_minutes: number;
}

export class BPMNDashboardApi {
  /**
   * 获取仪表盘指标
   */
  static async getDashboardMetrics(tenantId: number, startTime?: string, endTime?: string): Promise<DashboardMetrics> {
    const params = new URLSearchParams({ tenant_id: tenantId.toString() });
    if (startTime) params.append('start_time', startTime);
    if (endTime) params.append('end_time', endTime);

    const response = await httpClient.get(`/bpmn/dashboard/metrics?${params}`);
    return response.data;
  }

  /**
   * 获取流程指标
   */
  static async getProcessMetrics(key: string, tenantId: number, startTime?: string, endTime?: string): Promise<ProcessMetrics> {
    const params = new URLSearchParams({
      key,
      tenant_id: tenantId.toString()
    });
    if (startTime) params.append('start_time', startTime);
    if (endTime) params.append('end_time', endTime);

    const response = await httpClient.get(`/bpmn/dashboard/process/${key}/metrics?${params}`);
    return response.data;
  }

  /**
   * 查询审计日志
   */
  static async queryAuditLogs(request: QueryAuditLogsRequest): Promise<{ list: ProcessAuditLog[]; total: number; page: number }> {
    const params = new URLSearchParams();
    Object.entries(request).forEach(([key, value]) => {
      if (value !== undefined && value !== null) {
        params.append(key, value.toString());
      }
    });

    const response = await httpClient.get(`/bpmn/dashboard/audit-logs?${params}`);
    return response.data;
  }

  /**
   * 获取流程时间线
   */
  static async getProcessTimeline(processInstanceKey: string): Promise<ProcessAuditLog[]> {
    const response = await httpClient.get(`/bpmn/dashboard/audit-logs/timeline?process_instance_key=${processInstanceKey}`);
    return response.data;
  }

  /**
   * 获取用户活动
   */
  static async getUserActivity(userId: number, tenantId: number, startTime?: string, endTime?: string): Promise<ProcessAuditLog[]> {
    const params = new URLSearchParams({
      tenant_id: tenantId.toString()
    });
    if (startTime) params.append('start_time', startTime);
    if (endTime) params.append('end_time', endTime);

    const response = await httpClient.get(`/bpmn/dashboard/audit-logs/user/${userId}?${params}`);
    return response.data;
  }

  /**
   * 获取SLA违规
   */
  static async getSLAViolations(tenantId: number): Promise<SLAViolation[]> {
    const response = await httpClient.get(`/bpmn/dashboard/sla/violations?tenant_id=${tenantId}`);
    return response.data;
  }

  /**
   * 获取SLA合规率
   */
  static async getSLACompliance(key: string, tenantId: number, startTime?: string, endTime?: string): Promise<{
    compliance_rate: number;
    compliant: number;
    total: number;
  }> {
    const params = new URLSearchParams({
      key,
      tenant_id: tenantId.toString()
    });
    if (startTime) params.append('start_time', startTime);
    if (endTime) params.append('end_time', endTime);

    const response = await httpClient.get(`/bpmn/dashboard/sla/compliance?${params}`);
    return response.data;
  }

  /**
   * 获取租户统计
   */
  static async getTenantStats(tenantId: number): Promise<TenantBPMNStats> {
    const response = await httpClient.get(`/bpmn/dashboard/tenant/stats?tenant_id=${tenantId}`);
    return response.data;
  }

  /**
   * 获取瓶颈分析
   */
  static async getBottleneckAnalysis(key: string, tenantId: number): Promise<BottleneckInfo[]> {
    const response = await httpClient.get(`/bpmn/dashboard/bottlenecks?key=${key}&tenant_id=${tenantId}`);
    return response.data;
  }
}

export default BPMNDashboardApi;
