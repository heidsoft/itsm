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
  sla_def_id: number;
  violation_type: string;
  expected_time: string;
  actual_time: string;
  delay_minutes: number;
  status: string;
  severity: string;
  description: string;
  tenant_id: number;
  created_at: string;
  updated_at: string;
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
    page_size?: number;
  }): Promise<{
    items: SLADefinition[];
    total: number;
    page: number;
    page_size: number;
  }> {
    return httpClient.get('/api/v1/sla/definitions', params);
  }

  // 创建SLA定义
  static async createSLADefinition(data: CreateSLADefinitionRequest): Promise<SLADefinition> {
    return httpClient.post<SLADefinition>('/api/v1/sla/definitions', data);
  }

  // 获取SLA定义详情
  static async getSLADefinition(id: number): Promise<SLADefinition> {
    return httpClient.get<SLADefinition>(`/api/v1/sla/definitions/${id}`);
  }

  // 更新SLA定义
  static async updateSLADefinition(id: number, data: Partial<CreateSLADefinitionRequest>): Promise<SLADefinition> {
    return httpClient.put<SLADefinition>(`/api/v1/sla/definitions/${id}`, data);
  }

  // 删除SLA定义
  static async deleteSLADefinition(id: number): Promise<void> {
    return httpClient.delete(`/api/v1/sla/definitions/${id}`);
  }

  // 获取SLA违规列表
  static async getSLAViolations(params?: {
    page?: number;
    page_size?: number;
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
    return httpClient.get('/api/v1/sla/v2/violations', params);
  }

  // 更新SLA违规状态
  static async updateSLAViolationStatus(id: number, isResolved: boolean, notes?: string): Promise<void> {
    return httpClient.put(`/api/v1/sla/v2/violations/${id}`, { is_resolved: isResolved, notes });
  }

  // 获取SLA合规报告
  static async getSLAComplianceReport(params: {
    start_date: string;
    end_date: string;
  }): Promise<SLAComplianceReport> {
    return httpClient.get('/api/v1/sla/compliance-report', params);
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
  }> {
    const requestBody: any = {};
    if (params?.start_time) requestBody.start_time = params.start_time;
    if (params?.end_time) requestBody.end_time = params.end_time;
    if (params?.sla_definition_id) requestBody.sla_definition_id = params.sla_definition_id;

    const response = await httpClient.post('/api/v1/sla/v2/monitoring', requestBody);

    // 转换后端响应格式到前端格式
    const violationRate = response.total_tickets > 0
      ? (response.violated_tickets / response.total_tickets) * 100
      : 0;
    const compliantTickets = response.total_tickets - response.violated_tickets;

    // 使用后端返回的风险工单数，如果没有则估算
    const atRiskTickets = response.at_risk_tickets ?? Math.floor(response.total_tickets * 0.15);

    return {
      compliance_rate: response.compliance_rate ?? (response.total_tickets > 0 ? (compliantTickets / response.total_tickets) * 100 : 0),
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
    return httpClient.get('/api/v1/sla/v2/metrics', params);
  }

  // 获取SLA预警
  static async getSLAAlerts(): Promise<Array<{
    ticket_id: number;
    ticket_title: string;
    priority: string;
    sla_definition: string;
    time_remaining: number;
    alert_level: 'warning' | 'critical';
    created_at: string;
  }>> {
    return httpClient.get('/api/v1/sla/alerts');
  }

  // 触发SLA监控检查
  static async triggerSLAMonitoring(): Promise<{
    checked_tickets: number;
    violations_found: number;
    alerts_sent: number;
  }> {
    return httpClient.post('/api/v1/sla/monitor');
  }

  // 创建SLA预警规则
  static async createAlertRule(data: {
    name: string;
    sla_definition_id: number;
    alert_level: 'warning' | 'critical';
    threshold_percentage: number;
    notification_channels: string[];
    escalation_enabled?: boolean;
    escalation_levels?: Array<{
      level: number;
      threshold: number;
      notify_users: number[];
    }>;
    is_active: boolean;
  }): Promise<any> {
    return httpClient.post('/api/v1/sla/alert-rules', data);
  }

  // 更新SLA预警规则
  static async updateAlertRule(id: number, data: Partial<{
    name: string;
    alert_level: 'warning' | 'critical';
    threshold_percentage: number;
    notification_channels: string[];
    is_active: boolean;
  }>): Promise<any> {
    return httpClient.put(`/api/v1/sla/v2/alert-rules/${id}`, data);
  }

  // 删除SLA预警规则
  static async deleteAlertRule(id: number): Promise<void> {
    return httpClient.delete(`/api/v1/sla/v2/alert-rules/${id}`);
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
    return httpClient.get(`/api/v1/sla/v2/alert-rules/${id}`);
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
    items: any[];
    total: number;
    page: number;
    page_size: number;
  }> {
    return httpClient.get('/api/v1/sla/v2/alert-history', params);
  }
}

export default SLAApi;