/**
 * BPMN Dashboard API 服务
 * 用于流程监控、审计日志、SLA管理等
 */

import { httpClient } from './http-client';

export interface DashboardMetrics {
  totalProcesses: number;
  activeInstances: number;
  completedToday: number;
  openTasks: number;
  slaComplianceRate: number;
  /** 合规样本数（后端 weighted compliant），用于区分“暂无样本”与真实 0% */
  slaCompliantSamples: number;
  /** 参与 SLA 合规统计的已完成实例总数，0 表示暂无数据 */
  slaTotalSamples: number;
  avgCompletionTimeMinutes: number;
  processHealth: ProcessHealth;
  topProcesses: ProcessStat[];
  taskDistribution: TaskStat[];
  trendData: TrendPoint[];
}

export interface ProcessHealth {
  healthy: number;
  warning: number;
  critical: number;
  healthScore: number;
}

export interface ProcessStat {
  processDefinitionKey: string;
  totalInstances: number;
  runningInstances: number;
  completedInstances: number;
  avgDurationMinutes: number;
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
  totalInstances: number;
  runningInstances: number;
  completedInstances: number;
  terminatedInstances: number;
  completionRate: number;
  slaComplianceRate: number;
  avgCompletionTimeMinutes: number;
}

export interface ProcessAuditLog {
  id: number;
  processInstanceId: number;
  processInstanceKey: string;
  processDefinitionKey: string;
  processDefinitionId: number;
  activityId: string;
  activityName: string;
  activityType: string;
  action: string;
  userId: number;
  userName: string;
  assigneeId: number;
  assigneeName: string;
  variablesBefore: Record<string, any>;
  variablesAfter: Record<string, any>;
  comment: string;
  ipAddress: string;
  userAgent: string;
  tenantId: number;
  timestamp: string;
  durationMs: number;
  metadata: Record<string, any>;
}

export interface QueryAuditLogsRequest {
  tenantId?: number;
  processInstanceId?: number;
  processDefinitionKey?: string;
  action?: string;
  userId?: number;
  activityType?: string;
  startTime?: string;
  endTime?: string;
  page?: number;
  pageSize?: number;
}

/**
 * SLA 违规实体，字段与后端 handlers/sla/entity.go::SLAViolation 逐一对齐。
 * 旧版 resourceType/startTime/deadline/slaStatus/elapsedMinutes 后端根本不存在，
 * 导致监控表渲染 "Invalid Date" 与空单元格。
 */
export interface SLAViolation {
  id: number;
  createdBy: number;
  ticketId: number;
  ticketNumber?: string;
  ticketTitle?: string;
  ticketPriority?: string;
  slaName?: string;
  slaDefinitionId: number;
  violationType: string;
  violationTime: string;
  description?: string;
  severity: string;
  isResolved: boolean;
  resolvedAt?: string | null;
  resolutionNotes?: string;
  tenantId: number;
}

export interface TenantBPMNStats {
  totalDefinitions: number;
  totalInstances: number;
  runningInstances: number;
  completedInstances: number;
  totalTasks: number;
  openTasks: number;
}

export interface BottleneckInfo {
  taskName: string;
  totalCount: number;
  avgDurationMinutes: number;
  maxDurationMinutes: number;
  minDurationMinutes: number;
}

export class BPMNDashboardApi {
  private static readonly baseUrl = '/api/v1/bpmn/dashboard';
  /**
   * 获取仪表盘指标
   */
  static async getDashboardMetrics(tenantId: number, startTime?: string, endTime?: string): Promise<DashboardMetrics> {
    const params = new URLSearchParams({ tenantId: tenantId.toString() });
    if (startTime) params.append('startTime', startTime);
    if (endTime) params.append('endTime', endTime);

    return httpClient.get<DashboardMetrics>(`${this.baseUrl}/metrics?${params}`);
  }

  /**
   * 获取流程指标
   */
  static async getProcessMetrics(key: string, tenantId: number, startTime?: string, endTime?: string): Promise<ProcessMetrics> {
    const params = new URLSearchParams({
      key,
      tenantId: tenantId.toString()
    });
    if (startTime) params.append('startTime', startTime);
    if (endTime) params.append('endTime', endTime);

    return httpClient.get<ProcessMetrics>(`${this.baseUrl}/process/${key}/metrics?${params}`);
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

    return httpClient.get<{ list: ProcessAuditLog[]; total: number; page: number }>(`${this.baseUrl}/audit-logs?${params}`);
  }

  /**
   * 获取流程时间线
   *
   * 优先调用 BPMNMonitoringController 的新端点（任务 3），它会返回
   * 完整的 process_instance_id + entries 结构，包含活动、网关、变量
   * 等事件。旧 dashboard 端点已不再维护。
   */
  static async getProcessTimeline(processInstanceKey: string): Promise<ProcessAuditLog[]> {
    const res = await httpClient.get<
      | { data?: { entries: ProcessAuditLog[] } }
      | { entries: ProcessAuditLog[] }
      | ProcessAuditLog[]
    >(`/api/v1/bpmn/monitoring/instances/${encodeURIComponent(processInstanceKey)}/timeline`);
    if (Array.isArray(res)) return res;
    const entries = (res as { data?: { entries: ProcessAuditLog[] } }).data?.entries ??
      (res as { entries?: ProcessAuditLog[] }).entries;
    return entries ?? [];
  }

  /**
   * 获取用户活动
   */
  static async getUserActivity(userId: number, tenantId: number, startTime?: string, endTime?: string): Promise<ProcessAuditLog[]> {
    const params = new URLSearchParams({
      tenantId: tenantId.toString()
    });
    if (startTime) params.append('startTime', startTime);
    if (endTime) params.append('endTime', endTime);

    return httpClient.get<ProcessAuditLog[]>(`${this.baseUrl}/audit-logs/user/${userId}?${params}`);
  }

  /**
   * 获取 SLA 违规列表。真实路由为 /api/v1/sla/violations（非 bpmn dashboard 前缀），
   * 响应为分页信封 {items,total,page,pageSize}，这里解包返回 items。
   */
  static async getSLAViolations(
    tenantId: number,
    page = 1,
    pageSize = 100
  ): Promise<SLAViolation[]> {
    const params = new URLSearchParams({
      tenantId: tenantId.toString(),
      page: page.toString(),
      pageSize: pageSize.toString(),
    });
    const data = await httpClient.get<{
      items: SLAViolation[];
      total: number;
      page: number;
      pageSize: number;
    }>(`/api/v1/sla/violations?${params}`);
    return data?.items ?? [];
  }

  /**
   * 获取SLA合规率
   */
  static async getSLACompliance(key: string, tenantId: number, startTime?: string, endTime?: string): Promise<{
    complianceRate: number;
    compliant: number;
    total: number;
  }> {
    const params = new URLSearchParams({
      key,
      tenantId: tenantId.toString()
    });
    if (startTime) params.append('startTime', startTime);
    if (endTime) params.append('endTime', endTime);

    return httpClient.get<{ complianceRate: number; compliant: number; total: number }>(`${this.baseUrl}/sla/compliance?${params}`);
  }

  /**
   * 获取租户统计
   */
  static async getTenantStats(tenantId: number): Promise<TenantBPMNStats> {
    return httpClient.get<TenantBPMNStats>(`${this.baseUrl}/tenant/stats?tenantId=${tenantId}`);
  }

  /**
   * 获取瓶颈分析
   */
  static async getBottleneckAnalysis(key: string, tenantId: number): Promise<BottleneckInfo[]> {
    return httpClient.get<BottleneckInfo[]>(`${this.baseUrl}/bottlenecks?key=${key}&tenantId=${tenantId}`);
  }
}

export default BPMNDashboardApi;
