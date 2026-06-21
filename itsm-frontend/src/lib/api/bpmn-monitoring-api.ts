/**
 * BPMN 监控 API（v2）
 *
 * 对应后端 controller/bpmn_monitoring_controller.go
 * 路由：/api/v1/bpmn/monitoring
 *
 * 与 bpmn-dashboard-api.ts 的区别：
 * - monitoring 接口侧重于单个流程实例/单流程 key 的深度指标
 * - dashboard 接口侧重于租户级聚合
 * - 时间线 (timeline) 入口在此
 * - 瓶颈分析 (bottleneck) 的新指标（WaitTimeSeconds/P95）在此
 */

import { httpClient } from './http-client';

// ─── 流程指标 ───────────────────────────────────────────────
export interface ProcessMetrics {
  processDefinitionKey?: string;
  totalInstances: number;
  runningInstances: number;
  completedInstances: number;
  terminatedInstances: number;
  failedInstances?: number;
  completionRate: number;
  slaComplianceRate: number;
  avgCompletionTimeMinutes: number;
  successRate?: number;
  bottleneckAnalysis?: BottleneckAnalysis;
}

export interface PerformanceMetrics {
  throughput: number;
  avgLeadTime: number;
  avgCycleTime: number;
  efficiency: number;
}

export interface ProcessInstanceStatus {
  instanceId: string;
  processDefinitionKey: string;
  status: string;
  startTime: string;
  endTime?: string;
  currentActivities?: string[];
  progress: number;
  variables?: Record<string, unknown>;
}

export interface ProcessInstanceStatusListResponse {
  instances: ProcessInstanceStatus[];
  total: number;
  page: number;
  pageSize: number;
}

// ─── 流程实例完整时间线（任务 3） ──────────────────────────
export interface ProcessTimelineEntry {
  eventType: string; // start | task_assigned | task_completed | gateway | end ...
  activityId: string;
  activityName: string;
  activityType: string;
  assigneeId?: number;
  assigneeName?: string;
  startTime?: string;
  endTime?: string;
  durationSeconds?: number;
  comment?: string;
  variables?: Record<string, unknown>;
}

export interface ProcessTimelineResponse {
  processInstanceId: string;
  entries: ProcessTimelineEntry[];
  total: number;
}

// ─── 瓶颈任务（任务 5） ────────────────────────────────────
export interface BottleneckTask {
  taskId: string;
  taskName: string;
  bottleneckType: string; // processing | waiting | resource
  impactScore: number; // 0-100
  averageDuration: string; // duration string
  waitTime: string;
  queueLength: number;
  assignee: string;
  recommendation: string;
  // 新增的秒级指标（修复 avgWaitTime=0 bug 后）
  waitTimeSeconds: number;
  processingTimeSeconds: number;
  totalDurationSeconds: number;
  p95WaitTimeSeconds: number;
  p95ProcessingTimeSeconds: number;
  p95TotalDurationSeconds: number;
  sampleCount: number;
}

export interface SlowestPath {
  pathId: string;
  pathName: string;
  totalDuration: string;
  taskCount: number;
  bottleneckTasks: string[];
  optimization: string;
}

export interface ResourceConstraint {
  resourceType: string;
  resourceName: string;
  utilization: number;
  capacity: number;
  currentLoad: number;
  constraintType: string;
}

export interface BottleneckAnalysis {
  bottleneckTasks: BottleneckTask[];
  slowestPaths: SlowestPath[];
  resourceConstraints: ResourceConstraint[];
  recommendations: string[];
  severity: string; // low | medium | high | critical
}

// ─── 性能告警 ──────────────────────────────────────────────
export interface PerformanceAlert {
  alertType: string;
  severity: string;
  message: string;
  resourceId?: string;
  metricValue?: number;
  threshold?: number;
  timestamp: string;
}

// ─── 系统健康 ──────────────────────────────────────────────
export interface SystemHealth {
  status: string;
  components: Record<string, { status: string; message?: string; latencyMs?: number }>;
  uptimeSeconds: number;
  version: string;
}

// ─── API 类 ────────────────────────────────────────────────
export class BPMNMonitoringApi {
  private static readonly baseUrl = '/api/v1/bpmn/monitoring';

  /**
   * 获取流程指标（聚合）
   * @param timeRange 默认 24h
   */
  static async getProcessMetrics(params?: {
    timeRange?: string;
    startTime?: string;
    endTime?: string;
  }): Promise<ProcessMetrics> {
    const query: Record<string, string> = {};
    if (params?.timeRange) query.time_range = params.timeRange;
    if (params?.startTime) query.start_time = params.startTime;
    if (params?.endTime) query.end_time = params.endTime;
    const res = await httpClient.get<{ data?: ProcessMetrics } & ProcessMetrics>(
      `${this.baseUrl}/metrics`,
      query
    );
    return (res as { data?: ProcessMetrics }).data ?? (res as ProcessMetrics);
  }

  /** 获取单个流程 key 的指标 */
  static async getProcessMetricsByKey(processKey: string, timeRange?: string): Promise<ProcessMetrics> {
    const query: Record<string, string> = {};
    if (timeRange) query.time_range = timeRange;
    const res = await httpClient.get<{ data?: ProcessMetrics } & ProcessMetrics>(
      `${this.baseUrl}/metrics/${encodeURIComponent(processKey)}`,
      query
    );
    return (res as { data?: ProcessMetrics }).data ?? (res as ProcessMetrics);
  }

  /** 获取流程实例完整时间线（任务 3） */
  static async getProcessTimeline(instanceId: string): Promise<ProcessTimelineResponse> {
    const res = await httpClient.get<
      { data?: ProcessTimelineResponse } & ProcessTimelineResponse
    >(`${this.baseUrl}/instances/${encodeURIComponent(instanceId)}/timeline`);
    return (res as { data?: ProcessTimelineResponse }).data ?? (res as ProcessTimelineResponse);
  }

  /** 获取单个流程实例状态 */
  static async getProcessInstanceStatus(instanceId: string): Promise<ProcessInstanceStatus> {
    const res = await httpClient.get<
      { data?: ProcessInstanceStatus } & ProcessInstanceStatus
    >(`${this.baseUrl}/instances/${encodeURIComponent(instanceId)}/status`);
    return (res as { data?: ProcessInstanceStatus }).data ?? (res as ProcessInstanceStatus);
  }

  /** 获取流程实例状态列表 */
  static async listProcessInstancesStatus(params?: {
    page?: number;
    pageSize?: number;
    processKey?: string;
    status?: string;
  }): Promise<ProcessInstanceStatusListResponse> {
    const query: Record<string, string> = {};
    if (params?.page) query.page = String(params.page);
    if (params?.pageSize) query.page_size = String(params.pageSize);
    if (params?.processKey) query.process_key = params.processKey;
    if (params?.status) query.status = params.status;
    const res = await httpClient.get<
      { data?: ProcessInstanceStatusListResponse } & ProcessInstanceStatusListResponse
    >(`${this.baseUrl}/instances/status`, query);
    return (
      (res as { data?: ProcessInstanceStatusListResponse }).data ??
      (res as ProcessInstanceStatusListResponse)
    );
  }

  /** 获取性能指标 */
  static async getPerformanceMetrics(timeRange?: string): Promise<PerformanceMetrics> {
    const query: Record<string, string> = {};
    if (timeRange) query.time_range = timeRange;
    const res = await httpClient.get<
      { data?: PerformanceMetrics } & PerformanceMetrics
    >(`${this.baseUrl}/performance`, query);
    return (res as { data?: PerformanceMetrics }).data ?? (res as PerformanceMetrics);
  }

  /** 获取性能告警 */
  static async getPerformanceAlerts(): Promise<PerformanceAlert[]> {
    const res = await httpClient.get<
      { data?: PerformanceAlert[] } | PerformanceAlert[]
    >(`${this.baseUrl}/performance/alerts`);
    if (Array.isArray(res)) return res;
    return (res as { data?: PerformanceAlert[] }).data ?? [];
  }

  /** 获取系统健康 */
  static async getSystemHealth(): Promise<SystemHealth> {
    const res = await httpClient.get<{ data?: SystemHealth } & SystemHealth>(
      `${this.baseUrl}/health`
    );
    return (res as { data?: SystemHealth }).data ?? (res as SystemHealth);
  }

  /**
   * 获取瓶颈分析（任务 5）
   *
   * 后端没有独立的 /bottlenecks 端点，瓶颈分析结果嵌在
   * /metrics/:processKey 响应里的 bottleneck_analysis 字段。
   *
   * @param processKey 流程 key
   * @param timeRange  默认 24h
   */
  static async getBottleneckAnalysis(
    processKey: string,
    timeRange?: string
  ): Promise<BottleneckAnalysis> {
    const query: Record<string, string> = {};
    if (timeRange) query.time_range = timeRange;
    const res = await httpClient.get<{ data?: ProcessMetrics } & ProcessMetrics>(
      `${this.baseUrl}/metrics/${encodeURIComponent(processKey)}`,
      query
    );
    const data = (res as { data?: ProcessMetrics }).data ?? (res as ProcessMetrics);
    // 兼容 bottleneck_analysis 为空时返回空骨架，避免上层 NPE
    return (
      data.bottleneckAnalysis ?? {
        bottleneckTasks: [],
        slowestPaths: [],
        resourceConstraints: [],
        recommendations: [],
        severity: 'low',
      }
    );
  }
}

export default BPMNMonitoringApi;