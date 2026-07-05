/**
 * SLA 类型定义
 */

export interface SLADefinition {
  id: number;
  name: string;
  description: string;
  serviceType: string;
  priority: string;
  responseTime: number;
  resolutionTime: number;
  businessHours: Record<string, any>;
  escalationRules: Record<string, any>;
  conditions: Record<string, any>;
  isActive: boolean;
  tenantId: number;
  createdAt: string;
  updatedAt: string;
}

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

export interface SLADefinitionListResponse {
  items: SLADefinition[];
  total: number;
  page: number;
  size: number;
}
