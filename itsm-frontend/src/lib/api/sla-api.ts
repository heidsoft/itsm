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
  isActive?: boolean;
  tenantId?: number;
  createdAt?: string;
  updatedAt?: string;
}

// SLA违规接口 (camelCase - httpClient 自动转换)
export interface SLAViolation {
  id: number;
  ticketId: number;
  slaDefId: number;
  violationType: string;
  expectedTime: string;
  actualTime: string;
  delayMinutes: number;
  status: string;
  severity: string;
  description: string;
  tenantId: number;
  createdAt: string;
  updatedAt: string;
}

// SLA合规报告接口 (camelCase - 因为 httpClient 会自动转换)
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
    size?: number;
    isActive?: boolean;
    name?: string;
  }): Promise<{
    items: SLADefinition[];
    total: number;
    page: number;
    pageSize: number;
  }> {
    // 后端期望参数是 snake_case
    const queryParams: Record<string, string> = {};
    if (params) {
      if (params.page) queryParams.page = String(params.page);
      if (params.size) queryParams.size = String(params.size);
      if (params.isActive !== undefined) queryParams.is_active = String(params.isActive);
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
    is_compliant: boolean;
    violations: SLAViolation[];
  }> {
    return httpClient.post(`/api/v1/sla/check-compliance/${ticketId}`);
  }

  // 获取SLA违规列表
  static async getSLAViolations(params?: {
    page?: number;
    size?: number;
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
    // 后端期望参数是 snake_case
    const queryParams: Record<string, string> = {};
    if (params) {
      if (params.page) queryParams.page = String(params.page);
      if (params.size) queryParams.size = String(params.size);
      if (params.isResolved !== undefined) queryParams.is_resolved = String(params.isResolved);
      if (params.severity) queryParams.severity = params.severity;
      if (params.violationType) queryParams.violation_type = params.violationType;
      if (params.slaDefinitionId) queryParams.sla_definition_id = String(params.slaDefinitionId);
    }
    return httpClient.get('/api/v1/sla/violations', queryParams);
  }

  // 更新SLA违规状态
  static async updateSLAViolationStatus(
    id: number,
    isResolved: boolean,
    notes?: string
  ): Promise<void> {
    return httpClient.put(`/api/v1/sla/violations/${id}`, { is_resolved: isResolved, notes });
  }

  // 获取SLA合规报告
  // 使用后端 /sla/compliance-report 端点
  static async getSLAComplianceReport(params: {
    startDate: string;
    endDate: string;
  }): Promise<SLAComplianceReport> {
    // 调用合规报告端点 - 后端期望 snake_case 参数
    const report = await httpClient.get<SLAComplianceReport>('/api/v1/sla/compliance-report', {
      start_date: params.startDate,
      end_date: params.endDate,
    });

    // httpClient 已自动将 snake_case 转换为 camelCase
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
  // 注意：httpClient 会自动将 snake_case 转为 camelCase
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
  static async getSLAMonitoring(params?: {
    tenantId?: number;
    startTime?: string;
    endTime?: string;
    slaDefinitionId?: number;
  }): Promise<{
    complianceRate: number;
    violationRate: number;
    totalTickets: number;
    compliantTickets: number;
    violatedTickets: number;
    atRiskTickets: number;
    averageResponseTime: number;
    averageResolutionTime: number;
    responseTimeCompliance: number;
    resolutionTimeCompliance: number;
    alerts: Array<{
      id: string;
      ticketId: number;
      ticketNumber: string;
      ticketTitle: string;
      priority: string;
      alertLevel: string;
      timeRemaining: number;
      slaDefinition: string;
      createdAt: string;
    }>;
     
  }> {
    // 转换为后端期望的 snake_case
    const requestBody: Record<string, string> = {};
    if (params?.startTime) requestBody.start_time = params.startTime;
    if (params?.endTime) requestBody.end_time = params.endTime;
    if (params?.slaDefinitionId) requestBody.sla_definition_id = String(params.slaDefinitionId);

    const response = await httpClient.post<{
      total_tickets: number;
      violated_tickets: number;
      at_risk_tickets?: number;
      compliance_rate?: number;
      average_response_time?: number;
      average_resolution_time?: number;
      response_time_compliance?: number;
      resolution_time_compliance?: number;
      alerts?: Array<{
        id: string;
        ticketId: number;
        ticketNumber: string;
        ticketTitle: string;
        priority: string;
        alertLevel: string;
        timeRemaining: number;
        slaDefinition: string;
        createdAt: string;
      }>;
    }>('/api/v1/sla/monitoring', requestBody);

    // httpClient 已自动将 snake_case 转换为 camelCase
    const violationRate =
      response.totalTickets > 0 ? (response.violatedTickets / response.totalTickets) * 100 : 0;
    const compliantTickets = response.totalTickets - response.violatedTickets;

    // 使用后端返回的风险工单数，如果没有则估算
    const atRiskTickets = response.atRiskTickets ?? Math.floor(response.totalTickets * 0.15);

    return {
      complianceRate:
        response.complianceRate ??
        (response.totalTickets > 0 ? (compliantTickets / response.totalTickets) * 100 : 0),
      violationRate: violationRate,
      totalTickets: response.totalTickets,
      compliantTickets: compliantTickets,
      violatedTickets: response.violatedTickets,
      atRiskTickets: atRiskTickets,
      averageResponseTime: response.averageResponseTime ?? 0,
      averageResolutionTime: response.averageResolutionTime ?? 0,
      responseTimeCompliance: response.responseTimeCompliance ?? 0,
      resolutionTimeCompliance: response.resolutionTimeCompliance ?? 0,
      alerts: response.alerts ?? [],
    };
  }

  // 获取SLA性能指标
  // 由于 /sla/metrics 返回空数据，改用 /sla/monitoring 获取
  static async getSLAMetrics(params?: {
    period?: 'day' | 'week' | 'month' | 'quarter';
    serviceType?: string;
    priority?: string;
    slaDefinitionId?: number;
    metricType?: string;
  }): Promise<{
    responseTimeAvg: number;
    resolutionTimeAvg: number;
    complianceRate: number;
    violationCount: number;
    trendData: Array<{
      date: string;
      complianceRate: number;
      avgResponseTime: number;
      avgResolutionTime: number;
    }>;
  }> {
    // 使用 monitoring 端点获取指标数据
    const monitoring = await httpClient.post<{
      average_response_time?: number;
      averageResolutionTime?: number;
      average_resolution_time?: number;
      compliance_rate?: number;
      complianceRate?: number;
      violated_tickets?: number;
      violatedTickets?: number;
      alerts?: Array<{
        ticket_id?: number;
        ticketId?: number;
        ticket_title?: string;
        ticketTitle?: string;
        priority?: string;
        sla_definition?: string;
        slaDefinition?: string;
        time_remaining?: number;
        timeRemaining?: number;
        alert_level?: 'warning' | 'critical' | 'severe';
        alertLevel?: 'warning' | 'critical' | 'severe';
        created_at?: string;
        createdAt?: string;
      }>;
    }>('/api/v1/sla/monitoring', {});

    // httpClient 已自动将 snake_case 转换为 camelCase
    return {
      responseTimeAvg: monitoring.averageResponseTime || 0,
      resolutionTimeAvg: monitoring.averageResolutionTime || 0,
      complianceRate: monitoring.complianceRate || 0,
      violationCount: monitoring.violatedTickets || 0,
      trendData: [], // 趋势数据需要从专门的 metrics API 获取
    };
  }

  // 获取SLA预警
  // 从 monitoring 端点获取实时预警数据
  static async getSLAAlerts(): Promise<
    Array<{
      ticketId: number;
      ticketTitle: string;
      priority: string;
      slaDefinition: string;
      timeRemaining: number;
      alertLevel: 'warning' | 'critical' | 'severe';
      createdAt: string;
    }>
  > {
    // 从 monitoring 端点获取预警数据
    const monitoring = await httpClient.post<{
      alerts?: Array<{
        ticket_id?: number;
        ticketId?: number;
        ticket_title?: string;
        ticketTitle?: string;
        priority?: string;
        sla_definition?: string;
        slaDefinition?: string;
        time_remaining?: number;
        timeRemaining?: number;
        alert_level?: 'warning' | 'critical' | 'severe';
        alertLevel?: 'warning' | 'critical' | 'severe';
        created_at?: string;
        createdAt?: string;
      }>;
    }>('/api/v1/sla/monitoring', {});

    // 如果有 alerts 则使用，否则返回空数组
    if (monitoring.alerts && Array.isArray(monitoring.alerts)) {
      return monitoring.alerts.map((item) => ({
        ticketId: item.ticketId || item.ticket_id || 0,
        ticketTitle: item.ticketTitle || item.ticket_title || `Ticket #${item.ticketId || item.ticket_id || 0}`,
        priority: item.priority || 'medium',
        slaDefinition: item.slaDefinition || item.sla_definition || 'SLA',
        timeRemaining: item.timeRemaining || item.time_remaining || 0,
        alertLevel: item.alertLevel || item.alert_level || 'warning',
        createdAt: item.createdAt || item.created_at || new Date().toISOString(),
      }));
    }

    return [];
  }

  // 触发SLA监控检查
  static async triggerSLAMonitoring(): Promise<{
    checkedTickets: number;
    violationsFound: number;
    alertsSent: number;
  }> {
    // 修正: 确保路径与后端一致
    return httpClient.post('/api/v1/sla/monitor');
  }

  // 创建SLA预警规则
  static async createAlertRule(data: {
    name: string;
    sla_definition_id: number;
    alert_level: 'warning' | 'critical' | 'severe';
    threshold_percentage: number;
    notification_channels: string[];
    escalation_enabled?: boolean;
    escalation_levels?: Array<{
      level: number;
      threshold: number;
      notify_users: number[];
    }>;
    is_active: boolean;
  }): Promise<unknown> {
    return httpClient.post('/api/v1/sla/alert-rules', data);
  }

  // 更新SLA预警规则
  static async updateAlertRule(
    id: number,
    data: Partial<{
      name: string;
      alert_level: 'warning' | 'critical' | 'severe';
      threshold_percentage: number;
      notification_channels: string[];
      is_active: boolean;
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
    sla_definition_id?: number;
    is_active?: boolean;
    alert_level?: string;
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
    sla_definition_id?: number;
    alert_rule_id?: number;
    ticket_id?: number;
    alert_level?: string;
    start_time?: string;
    end_time?: string;
    page?: number;
    page_size?: number;
  }): Promise<{
    items: unknown[];
    total: number;
    page: number;
    page_size: number;
  }> {
    // 修正: 确保路径与后端一致，去掉 v2
    return httpClient.get('/api/v1/sla/alert-history', params);
  }

  // ==================== 兼容别名（旧代码使用） ====================

  /** @deprecated 使用 getSLADefinitions */
  static getDefinitions(params?: { page?: number; size?: number; is_active?: boolean; name?: string }) {
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
