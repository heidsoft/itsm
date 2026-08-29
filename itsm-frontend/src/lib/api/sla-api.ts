import { httpClient } from './http-client';

// SLA定义接口 (camelCase - httpClient 自动转换)
export interface SLADefinition {
  id: number;
  name: string;
  description: string;
  serviceType?: string;
  priority: string;
  responseTime: number;
  resolutionTime: number;
  availabilityTarget?: number;
  availability?: number;
  complianceRate?: number;
  isActive?: boolean;
  tenantId?: number;
  createdAt?: string;
  updatedAt?: string;
}

// SLA违规接口 (camelCase - httpClient 自动转换)
// ticketNumber/ticketTitle/ticketPriority/slaName 由后端 ListViolations 通过
// ticket edge 联查返回，前端不得再用占位文本退化。
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
  description: string;
  severity: string;
  isResolved: boolean;
  resolvedAt?: string;
  resolutionNotes: string;
  tenantId: number;
}

// SLA合规报告接口（字段为 camelCase；注意后端 compliance-report 端点接收的 query 参数为 snake_case start_date/end_date）
export interface SLAComplianceReport {
  totalTickets: number;
  metSla: number;
  violatedSla: number;
  complianceRate: number;
  avgResponseTime: number;
  avgResolutionTime: number;
  reportPeriod: {
    startDate: string;
    endDate: string;
  };
}

// ==================== 监控大屏契约（逐字段对应后端 handlers/sla/entity.go） ====================
//
// 口径约定，前端不得重新计算或推断：
//   - 所有 *Rate / *Compliance 字段是 0-100 的百分数，后端已保留一位小数；
//   - 每个比率都带样本数量，样本为 0 表示「暂无数据」且比率固定为 0，
//     界面必须渲染空态，禁止把无样本显示成 0% 或 100% 合规；
//   - 时长统一使用分钟（*Minutes），与 SLADefinition.responseTime/resolutionTime 同单位。

/** 剩余时间：绑定工单的解决截止时间；无截止时间时后端返回 null */
export interface SLATimeRemaining {
  /** 可为负数，表示已超时 */
  hours: number;
  /** 截止时间，RFC3339 */
  deadline: string;
}

/** 告警列表行，数据源是真实触发的 SLA 告警历史 */
export interface SLAAlertItem {
  id: number;
  ticketId: number;
  ticketNumber: string;
  ticketTitle: string;
  priority: string;
  alertLevel: string;
  alertRuleName: string;
  thresholdPercentage: number;
  actualPercentage: number;
  createdAt: string;
  resolvedAt?: string;
  timeRemaining?: SLATimeRemaining;
}

/** POST /api/v1/sla/monitoring 响应 */
export interface SLAMonitoringData {
  /** 实际生效的统计窗口，RFC3339 */
  startTime: string;
  endTime: string;
  /** true 表示窗口内工单数超过扫描上限，分母已截断 */
  truncated: boolean;

  // 工单解决率：生命周期状态为 resolved/closed 的工单占比
  totalTickets: number;
  resolvedTickets: number;
  resolutionRate: number;

  // SLA 合规：窗口内工单中当前仍有未解决违约的数量
  violatedTickets: number;
  metSlaTickets: number;
  complianceRate: number;
  violationRate: number;
  atRiskTickets: number;

  // 响应达成率：有响应截止时间且已首次响应的工单中，响应不晚于截止时间的比例
  responseTimeSamples: number;
  responseTimeMet: number;
  responseTimeCompliance: number;
  // 解决达成率：有解决截止时间且已解决的工单中，解决不晚于截止时间的比例
  resolutionTimeSamples: number;
  resolutionTimeMet: number;
  resolutionTimeCompliance: number;

  averageResponseMinutes: number;
  averageResolutionMinutes: number;

  // 违约记录维度（记录数，非工单数）
  totalViolations: number;
  resolvedViolations: number;
  activeViolations: number;

  // 活跃告警：截至窗口结束仍未解决的告警数量；alerts 是最近若干条明细
  activeAlerts: number;
  alerts: SLAAlertItem[];

  activeSlas: number;
  activeAlertRules: number;
}

/** 绩效分组维度：服务类型来自 SLA 定义的 serviceType，优先级取工单自身字段 */
export type SLAPerformanceDimension = 'serviceType' | 'priority';

/** GET /api/v1/sla/performance 单行；`unassigned` 表示工单未绑定 SLA 定义 */
export interface SLAPerformanceRow {
  key: string;
  totalTickets: number;
  resolvedTickets: number;
  resolutionRate: number;
  violatedTickets: number;
  metSlaTickets: number;
  complianceRate: number;
  responseSamples: number;
  responseAchievementRate: number;
  resolutionSamples: number;
  resolutionAchievementRate: number;
  averageResponseMinutes: number;
  averageResolutionMinutes: number;
}

export interface SLAPerformanceResult {
  items: SLAPerformanceRow[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
  dimension: SLAPerformanceDimension;
  truncated: boolean;
}

/** GET /api/v1/sla/metrics 的单条指标记录（SLA 指标表，非时间序列） */
export interface SLAMetricRecord {
  id: number;
  slaDefinitionId: number;
  metricType: string;
  metricName: string;
  metricValue: number;
  unit: string;
  measurementTime: string;
  tenantId: number;
}

// 创建SLA定义请求 (camelCase)
export interface CreateSLADefinitionRequest {
  name: string;
  description: string;
  serviceType: string;
  priority: string;
  responseTimeMinutes: number;
  resolutionTimeMinutes: number;
  availabilityTarget: number;
}

export class SLAApi {
  // 获取SLA定义列表
  static async getSLADefinitions(params?: {
    page?: number;
    pageSize?: number;
    isActive?: boolean;
    name?: string;
  }): Promise<{
    items: SLADefinition[];
    total: number;
    page: number;
    pageSize: number;
  }> {
    const queryParams: Record<string, string> = {};
    if (params) {
      if (params.page) queryParams.page = String(params.page);
      if (params.pageSize) queryParams.pageSize = String(params.pageSize);
      if (params.isActive !== undefined) queryParams.isActive = String(params.isActive);
      if (params.name) queryParams.name = params.name;
    }
    return httpClient.get('/api/v1/sla/definitions', queryParams);
  }

  // 获取SLA定义详情
  static async getSLADefinition(id: number): Promise<SLADefinition> {
    return httpClient.get(`/api/v1/sla/definitions/${id}`);
  }

  // 创建SLA定义
  static async createSLADefinition(data: Partial<SLADefinition>): Promise<SLADefinition> {
    return httpClient.post('/api/v1/sla/definitions', data);
  }

  // 更新SLA定义
  static async updateSLADefinition(
    id: number,
    data: Partial<SLADefinition>
  ): Promise<SLADefinition> {
    return httpClient.put(`/api/v1/sla/definitions/${id}`, data);
  }

  // 删除SLA定义
  static async deleteSLADefinition(id: number): Promise<void> {
    return httpClient.delete(`/api/v1/sla/definitions/${id}`);
  }

  // 检查工单SLA合规性
  static async checkTicketCompliance(ticketId: number): Promise<{
    isCompliant: boolean;
    violations: SLAViolation[];
  }> {
    return httpClient.post(`/api/v1/sla/check-compliance/${ticketId}`);
  }

  // 获取SLA违规列表
  static async getSLAViolations(params?: {
    page?: number;
    pageSize?: number;
    isResolved?: boolean;
    severity?: string;
    violationType?: string;
    slaDefinitionId?: number;
  }): Promise<{
    items: SLAViolation[];
    total: number;
    page: number;
    pageSize: number;
  }> {
    const queryParams: Record<string, string> = {};
    if (params) {
      if (params.page) queryParams.page = String(params.page);
      if (params.pageSize) queryParams.pageSize = String(params.pageSize);
      if (params.isResolved !== undefined) queryParams.isResolved = String(params.isResolved);
      if (params.severity) queryParams.severity = params.severity;
      if (params.violationType) queryParams.violationType = params.violationType;
      if (params.slaDefinitionId) queryParams.slaDefinitionId = String(params.slaDefinitionId);
    }
    return httpClient.get('/api/v1/sla/violations', queryParams);
  }

  // 更新SLA违规状态
  static async updateSLAViolationStatus(
    id: number,
    isResolved: boolean,
    notes?: string
  ): Promise<void> {
    return httpClient.put(`/api/v1/sla/violations/${id}`, { isResolved: isResolved, notes });
  }

  // 获取SLA合规报告
  // 使用后端 /sla/compliance-report 端点
  // 查询参数统一使用 camelCase，与请求体/响应体保持一致。
  static async getSLAComplianceReport(params: {
    startDate: string;
    endDate: string;
  }): Promise<SLAComplianceReport> {
    const report = await httpClient.get<SLAComplianceReport>('/api/v1/sla/compliance-report', {
      startDate: params.startDate,
      endDate: params.endDate,
    });

    return {
      totalTickets: report.totalTickets || 0,
      metSla: report.metSla || 0,
      violatedSla: report.violatedSla || 0,
      complianceRate: report.complianceRate || 0,
      avgResponseTime: report.avgResponseTime || 0,
      avgResolutionTime: report.avgResolutionTime || 0,
      reportPeriod: {
        startDate: report.reportPeriod?.startDate || params.startDate,
        endDate: report.reportPeriod?.endDate || params.endDate,
      },
    };
  }

  // 检查工单SLA违规
  static async checkTicketSLAViolation(ticketId: number): Promise<void> {
    return httpClient.post(`/api/v1/sla/check-compliance/${ticketId}`);
  }

  // 获取SLA统计信息
  static async getSLAStats(): Promise<{
    totalDefinitions: number;
    activeDefinitions: number;
    totalViolations: number;
    openViolations: number;
    overallComplianceRate: number;
  }> {
    return httpClient.get('/api/v1/sla/stats');
  }

  // 获取SLA实时监控数据
  // 契约：POST /api/v1/sla/monitoring，请求体 { startTime?, endTime? } 使用 RFC3339；
  // 不传则由后端套用默认 30 天窗口。租户来自认证上下文，前端不传 tenantId。
  static async getSLAMonitoring(params?: {
    startTime?: string;
    endTime?: string;
  }): Promise<SLAMonitoringData> {
    const requestBody: { startTime?: string; endTime?: string } = {};
    if (params?.startTime) requestBody.startTime = params.startTime;
    if (params?.endTime) requestBody.endTime = params.endTime;

    const response = await httpClient.post<SLAMonitoringData>(
      '/api/v1/sla/monitoring',
      requestBody
    );
    if (!response) {
      // 请求被取消或响应为空：交给调用方渲染错误态，禁止用全 0 数据伪装成功。
      throw new Error('SLA 监控数据为空');
    }

    return {
      ...response,
      alerts: response.alerts ?? [],
    };
  }

  // 获取SLA绩效：按服务类型或优先级聚合，供监控大屏绩效表格使用。
  // 契约：GET /api/v1/sla/performance?dimension&startDate&endDate&serviceType&priority&page&pageSize
  // serviceType / priority 是统计种群预过滤，不是对已分页结果集的二次筛选；
  // 传入未知 serviceType 时后端返回空集，前端不得回退成全量数据。
  static async getSLAPerformance(params: {
    dimension: SLAPerformanceDimension;
    startDate?: string;
    endDate?: string;
    serviceType?: string;
    priority?: string;
    page?: number;
    pageSize?: number;
  }): Promise<SLAPerformanceResult> {
    const query: Record<string, string> = { dimension: params.dimension };
    if (params.startDate) query.startDate = params.startDate;
    if (params.endDate) query.endDate = params.endDate;
    if (params.serviceType) query.serviceType = params.serviceType;
    if (params.priority) query.priority = params.priority;
    if (params.page) query.page = String(params.page);
    if (params.pageSize) query.pageSize = String(params.pageSize);

    const response = await httpClient.get<SLAPerformanceResult>('/api/v1/sla/performance', query);
    return {
      items: response?.items ?? [],
      total: response?.total ?? 0,
      page: response?.page ?? params.page ?? 1,
      pageSize: response?.pageSize ?? params.pageSize ?? 20,
      totalPages: response?.totalPages ?? 0,
      dimension: params.dimension,
      truncated: response?.truncated ?? false,
    };
  }

  // 获取SLA度量记录
  // 契约：GET /api/v1/sla/metrics，返回 SLA 指标表记录；后端没有按日时间序列端点，
  // 因此本方法不提供 trendData，趋势图必须等真实时间序列接口后再做。
  static async getSLAMetrics(params?: {
    slaDefinitionId?: number;
    metricType?: string;
  }): Promise<{ metrics: SLAMetricRecord[]; count: number }> {
    const query: Record<string, string> = {};
    if (params?.slaDefinitionId) query.slaDefinitionId = String(params.slaDefinitionId);
    if (params?.metricType) query.metricType = params.metricType;

    const response = await httpClient.get<{
      metrics?: SLAMetricRecord[];
      count?: number;
    }>('/api/v1/sla/metrics', query);

    return {
      metrics: response?.metrics ?? [],
      count: response?.count ?? 0,
    };
  }

  // 获取SLA预警
  // 数据源与监控大屏完全一致（真实告警历史），不再从违规记录推导或伪造告警标题。
  static async getSLAAlerts(params?: {
    startTime?: string;
    endTime?: string;
  }): Promise<SLAAlertItem[]> {
    const monitoring = await SLAApi.getSLAMonitoring(params);
    return monitoring.alerts;
  }

  // 创建SLA预警规则
  static async createAlertRule(data: {
    name: string;
    slaDefinitionId: number;
    alertLevel: 'warning' | 'critical' | 'severe';
    thresholdPercentage: number;
    notificationChannels: string[];
    escalationEnabled?: boolean;
    escalationLevels?: Array<{
      level: number;
      threshold: number;
      notifyUsers: number[];
    }>;
    isActive: boolean;
  }): Promise<unknown> {
    return httpClient.post('/api/v1/sla/alert-rules', data);
  }

  // 更新SLA预警规则
  static async updateAlertRule(
    id: number,
    data: Partial<{
      name: string;
      alertLevel: 'warning' | 'critical' | 'severe';
      thresholdPercentage: number;
      notificationChannels: string[];
      isActive: boolean;
    }>
  ): Promise<any> {
    // 修正: 确保路径与后端一致，去掉 v2
    return httpClient.put(`/api/v1/sla/alert-rules/${id}`, data);
  }

  // 删除SLA预警规则
  static async deleteAlertRule(id: number): Promise<void> {
    // 修正: 确保路径与后端一致，去掉 v2
    return httpClient.delete(`/api/v1/sla/alert-rules/${id}`);
  }

  // 获取SLA预警规则列表
  static async getAlertRules(params?: {
    slaDefinitionId?: number;
    isActive?: boolean;
    alertLevel?: string;
  }): Promise<any[]> {
    return httpClient.get('/api/v1/sla/alert-rules', params);
  }

  // 获取SLA预警规则详情
  static async getAlertRule(id: number): Promise<any> {
    // 修正: 确保路径与后端一致，去掉 v2
    return httpClient.get(`/api/v1/sla/alert-rules/${id}`);
  }

  // 获取SLA预警历史
  static async getAlertHistory(params?: {
    slaDefinitionId?: number;
    alertRuleId?: number;
    ticketId?: number;
    alertLevel?: string;
    startTime?: string;
    endTime?: string;
    page?: number;
    pageSize?: number;
  }): Promise<{
    items: unknown[];
    total: number;
    page: number;
    pageSize: number;
  }> {
    // 修正: 确保路径与后端一致，去掉 v2
    return httpClient.get('/api/v1/sla/alert-history', params);
  }

  // ==================== 兼容别名（旧代码使用） ====================

  /** @deprecated 使用 getSLADefinitions */
  static getDefinitions(params?: { page?: number; pageSize?: number; isActive?: boolean; name?: string }) {
    return this.getSLADefinitions(params);
  }

  /** @deprecated 使用 getSLADefinition */
  static getDefinition(id: number) {
    return this.getSLADefinition(id);
  }

  /** @deprecated 使用 updateSLADefinition */
  static updateDefinition(id: number, data: Partial<SLADefinition>) {
    return this.updateSLADefinition(id, data);
  }

  /** @deprecated 使用 deleteSLADefinition */
  static deleteDefinition(id: number) {
    return this.deleteSLADefinition(id);
  }
}

export default SLAApi;
