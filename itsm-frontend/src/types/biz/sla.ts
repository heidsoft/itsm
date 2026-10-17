/**
 * SLA 类型定义
 *
 * SLA 定义的响应结构以 dto.SLADefinitionResponse 为准，前端类型统一用
 * `@/lib/api/sla-api` 的 SLADefinition，此处不再重复声明第二套契约。
 */

export interface SLAViolation {
  id: number;
  ticketId: number;
  slaDefinitionId: number;
  violationType: string;
  violationTime: string;
  description: string;
  severity: string;
  isResolved: boolean;
  resolvedAt?: string;
  resolutionNotes?: string;
  createdAt: string;
  updatedAt: string;
  // 时间相关
  timeRemaining?: number;
  alertLevel?: 'warning' | 'critical' | 'severe';
  slaDefinition?: string;
  ticketTitle?: string;
  priority?: string;
}

export interface SLAAlertRule {
  id: number;
  slaDefinitionId: number;
  name: string;
  thresholdPercentage: number;
  alertLevel: string;
  notificationChannels: string[];
  isActive: boolean;
}
