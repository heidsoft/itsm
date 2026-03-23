import { httpClient } from './http-client';

// SLA定义接口
export interface SLADefinition {
  id: number;
  name: string;
  description: string;
  service_type: string;
  priority: string;
  response_time_minutes: number;
  resolution_time_minutes: number;
  availability_target: number;
  is_active: boolean;
  tenant_id: number;
  created_at: string;
  updated_at: string;
}

// SLA违规接口
export interface SLAViolation {
  id: number;
  ticket_id: number;
  ticketId?: number;
  sla_def_id: number;
  slaDefId?: number;
  violation_type: string;
  violationType?: string;
  expected_time: string;
  expectedTime?: string;
  actual_time: string;
  actualTime?: string;
  delay_minutes: number;
  delayMinutes?: number;
  status: string;
  severity: string;
  description: string;
  tenant_id: number;
  tenantId?: number;
  created_at: string;
  createdAt?: string;
  updated_at: string;
  updatedAt?: string;
}

// SLA合规报告接口
export interface SLAComplianceReport {
  total_tickets: number;
  met_sla: number;
  violated_sla: number;
  compliance_rate: number;
  avg_response_time: number;
  avg_resolution_time: number;
  report_period: {
    start_date: string;
    end_date: string;
  };
}

// 创建SLA定义请求
export interface CreateSLADefinitionRequest {
  name: string;
  description: string;
  service_type: string;
  priority: string;
  response_time_minutes: number;
  resolution_time_minutes: number;
  availability_target: number;
}

export class SLAApi {
  // 获取SLA定义列表
  static async getSLADefinitions(params?: {
    page?: number;
    size?: number;
    is_active?: boolean;
    name?: string;
  }): Promise<{
    items: SLADefinition[];
    total: number;
    page: number;
    page_size: number;
  }> {
    // 后端期望参数是 page 和 size
    const queryParams: Record<string, string> = {};
    if (params) {
      if (params.page) queryParams.page = String(params.page);
      if (params.size) queryParams.size = String(params.size);
      if (params.is_active !== undefined) queryParams.is_active = String(params.is_active);
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
    return httpClient.post(`/api/v1/sla/compliance/${ticketId}`);
  }

  // 获取SLA违规列表
  static async getSLAViolations(params?: {
    page?: number;
    size?: number;
    is_resolved?: boolean;
    severity?: string;
    violation_type?: string;
    sla_definition_id?: number;
  }): Promise<{
    items: SLAViolation[];
    total: number;
    page: number;
    page_size: number;
  }> {
    // 后端期望参数是 page 和 size
    const queryParams: Record<string, string> = {};
    if (params) {
      if (params.page) queryParams.page = String(params.page);
      if (params.size) queryParams.size = String(params.size);
      if (params.is_resolved !== undefined) queryParams.is_resolved = String(params.is_resolved);
      if (params.severity) queryParams.severity = params.severity;
      if (params.violation_type) queryParams.violation_type = params.violation_type;
      if (params.sla_definition_id)
        queryParams.sla_definition_id = String(params.sla_definition_id);
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
  // 注意：后端暂无 /api/v1/sla/compliance-report 端点，使用 monitoring 数据转换
  static async getSLAComplianceReport(params: {
    start_date: string;
    end_date: string;
  }): Promise<SLAComplianceReport> {
    // 使用监控端点获取汇总数据
    const monitoring = await httpClient.post('/api/v1/sla/monitoring', {
      start_time: params.start_date,
      end_time: params.end_date,
    });

    // 转换为合规报告格式
    const total_tickets = monitoring.total_tickets || 0;
    const violated_tickets = monitoring.violated_tickets || 0;
    const compliant_tickets = total_tickets - violated_tickets;
    const compliance_rate = total_tickets > 0 ? (compliant_tickets / total_tickets) * 100 : 0;

    return {
      total_tickets,
      met_sla: compliant_tickets,
      violated_sla: violated_tickets,
      compliance_rate,
      avg_response_time: monitoring.average_response_time || 0,
      avg_resolution_time: monitoring.average_resolution_time || 0,
      report_period: {
        start_date: params.start_date,
        end_date: params.end_date,
      },
    };
  }

  // 检查工单SLA违规
  static async checkTicketSLAViolation(ticketId: number): Promise<void> {
    return httpClient.post(`/api/v1/sla/v2/check-compliance/${ticketId}`);
  }

  // 获取SLA统计信息
  static async getSLAStats(): Promise<{
    total_definitions: number;
    active_definitions: number;
    total_violations: number;
    open_violations: number;
    overall_compliance_rate: number;
  }> {
    return httpClient.get('/api/v1/sla/stats');
  }

  // 获取SLA实时监控数据
  static async getSLAMonitoring(params?: {
    tenant_id?: number;
    start_time?: string;
    end_time?: string;
    sla_definition_id?: number;
  }): Promise<{
    compliance_rate: number;
    violation_rate: number;
    total_tickets: number;
    compliant_tickets: number;
    violated_tickets: number;
    at_risk_tickets: number;
    average_response_time: number;
    average_resolution_time: number;
    response_time_compliance: number;
    resolution_time_compliance: number;
    alerts: Array<{
      id: string;
      ticket_id: number;
      ticket_number: string;
      ticket_title: string;
      priority: string;
      alert_level: string;
      time_remaining: number;
      sla_definition: string;
      created_at: string;
    }>;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
  }> {
    const requestBody: any = {};
    if (params?.start_time) requestBody.start_time = params.start_time;
    if (params?.end_time) requestBody.end_time = params.end_time;
    if (params?.sla_definition_id) requestBody.sla_definition_id = params.sla_definition_id;

    const response = await httpClient.post('/api/v1/sla/monitoring', requestBody);

    // 转换后端响应格式到前端格式
    const violationRate =
      response.total_tickets > 0 ? (response.violated_tickets / response.total_tickets) * 100 : 0;
    const compliantTickets = response.total_tickets - response.violated_tickets;

    // 使用后端返回的风险工单数，如果没有则估算
    const atRiskTickets = response.at_risk_tickets ?? Math.floor(response.total_tickets * 0.15);

    return {
      compliance_rate:
        response.compliance_rate ??
        (response.total_tickets > 0 ? (compliantTickets / response.total_tickets) * 100 : 0),
      violation_rate: violationRate,
      total_tickets: response.total_tickets,
      compliant_tickets: compliantTickets,
      violated_tickets: response.violated_tickets,
      at_risk_tickets: atRiskTickets,
      average_response_time: response.average_response_time ?? 0,
      average_resolution_time: response.average_resolution_time ?? 0,
      response_time_compliance: response.response_time_compliance ?? 0,
      resolution_time_compliance: response.resolution_time_compliance ?? 0,
      alerts: response.alerts ?? [],
    };
  }

  // 获取SLA性能指标
  static async getSLAMetrics(params?: {
    period?: 'day' | 'week' | 'month' | 'quarter';
    service_type?: string;
    priority?: string;
    sla_definition_id?: number;
    metric_type?: string;
  }): Promise<{
    response_time_avg: number;
    resolution_time_avg: number;
    compliance_rate: number;
    violation_count: number;
    trend_data: Array<{
      date: string;
      compliance_rate: number;
      avg_response_time: number;
      avg_resolution_time: number;
    }>;
  }> {
    // 修正: 确保路径与后端一致，后端可能是 /api/v1/sla/metrics
    return httpClient.get('/api/v1/sla/metrics', params);
  }

  // 获取SLA预警
  static async getSLAAlerts(): Promise<
    Array<{
      ticket_id: number;
      ticketId?: number;
      ticket_title: string;
      ticketTitle?: string;
      priority: string;
      sla_definition: string;
      slaDefinition?: string;
      time_remaining: number;
      timeRemaining?: number;
      alert_level: 'warning' | 'critical' | 'severe';
      alertLevel?: 'warning' | 'critical' | 'severe';
      created_at: string;
      createdAt?: string;
    }>
  > {
    // 修正: 确保路径与后端一致
    // 注意：后端目前似乎没有直接的 /api/v1/sla/alerts 端点，可能需要从 monitoring 或 alert-history 获取
    // 这里暂时假设后端会增加此端点，或者我们使用 alert-history 替代
    // 根据 PRD，告警历史是 /api/v1/sla/alert-history
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const history = await httpClient.get<{ items: unknown[] }>('/api/v1/sla/alert-history', {
      page: 1,
      size: 10,
    });
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    return history.items.map((item: any) => ({
      ticket_id: item.ticket_id || item.ticketId,
      ticketId: item.ticketId || item.ticket_id,
      ticket_title: item.ticket_title || item.ticketTitle || `Ticket #${item.ticket_id || item.ticketId}`,
      ticketTitle: item.ticketTitle || item.ticket_title || `Ticket #${item.ticket_id || item.ticketId}`,
      priority: item.priority || 'medium',
      sla_definition: item.sla_definition || item.slaDefinition || 'SLA',
      slaDefinition: item.slaDefinition || item.sla_definition || 'SLA',
      time_remaining: item.time_remaining || item.timeRemaining || 0,
      timeRemaining: item.timeRemaining || item.time_remaining || 0,
      alert_level: item.alert_level || item.alertLevel || 'warning',
      alertLevel: item.alertLevel || item.alert_level || 'warning',
      created_at: item.created_at || item.createdAt,
      createdAt: item.createdAt || item.created_at,
    }));
  }

  // 触发SLA监控检查
  static async triggerSLAMonitoring(): Promise<{
    checked_tickets: number;
    violations_found: number;
    alerts_sent: number;
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
