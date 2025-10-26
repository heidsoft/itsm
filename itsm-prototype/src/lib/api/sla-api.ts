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
    return httpClient.get('/api/sla/definitions', params);
  }

  // 创建SLA定义
  static async createSLADefinition(data: CreateSLADefinitionRequest): Promise<SLADefinition> {
    return httpClient.post<SLADefinition>('/api/sla/definitions', data);
  }

  // 获取SLA定义详情
  static async getSLADefinition(id: number): Promise<SLADefinition> {
    return httpClient.get<SLADefinition>(`/api/sla/definitions/${id}`);
  }

  // 更新SLA定义
  static async updateSLADefinition(id: number, data: Partial<CreateSLADefinitionRequest>): Promise<SLADefinition> {
    return httpClient.put<SLADefinition>(`/api/sla/definitions/${id}`, data);
  }

  // 删除SLA定义
  static async deleteSLADefinition(id: number): Promise<void> {
    return httpClient.delete(`/api/sla/definitions/${id}`);
  }

  // 获取SLA违规列表
  static async getSLAViolations(params?: {
    page?: number;
    page_size?: number;
    status?: string;
  }): Promise<{
    items: SLAViolation[];
    total: number;
    page: number;
    page_size: number;
  }> {
    return httpClient.get('/api/sla/violations', params);
  }

  // 更新SLA违规状态
  static async updateSLAViolationStatus(id: number, status: string): Promise<void> {
    return httpClient.put(`/api/sla/violations/${id}/status`, { status });
  }

  // 获取SLA合规报告
  static async getSLAComplianceReport(params: {
    start_date: string;
    end_date: string;
  }): Promise<SLAComplianceReport> {
    return httpClient.get('/api/sla/compliance-report', params);
  }

  // 检查工单SLA违规
  static async checkTicketSLAViolation(ticketId: number): Promise<void> {
    return httpClient.post(`/api/sla/check-violation/${ticketId}`);
  }

  // 获取SLA统计信息
  static async getSLAStats(): Promise<{
    total_definitions: number;
    active_definitions: number;
    total_violations: number;
    open_violations: number;
    overall_compliance_rate: number;
  }> {
    return httpClient.get('/api/sla/stats');
  }

  // 获取SLA性能指标
  static async getSLAMetrics(params?: {
    period?: 'day' | 'week' | 'month' | 'quarter';
    service_type?: string;
    priority?: string;
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
    return httpClient.get('/api/sla/metrics', params);
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
    return httpClient.get('/api/sla/alerts');
  }

  // 触发SLA监控检查
  static async triggerSLAMonitoring(): Promise<{
    checked_tickets: number;
    violations_found: number;
    alerts_sent: number;
  }> {
    return httpClient.post('/api/sla/monitor');
  }
}

export default SLAApi;