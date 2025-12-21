/**
 * SLA管理模块 - 符合ITIL 4.0标准
 * 
 * 核心功能：
 * 1. SLA定义和配置
 * 2. SLA监控和预警
 * 3. SLA升级机制
 * 4. SLA报告和分析
 * 5. 业务时间管理
 * 6. 节假日管理
 */

import { ApiResponse, PaginatedResponse } from '@/types/api';

// SLA状态枚举
export enum SLAStatus {
  ACTIVE = 'active',
  INACTIVE = 'inactive',
  SUSPENDED = 'suspended',
  EXPIRED = 'expired',
}

// SLA类型枚举
export enum SLAType {
  RESPONSE_TIME = 'response_time',      // 响应时间SLA
  RESOLUTION_TIME = 'resolution_time',   // 解决时间SLA
  AVAILABILITY = 'availability',         // 可用性SLA
  PERFORMANCE = 'performance',          // 性能SLA
}

// SLA优先级枚举
export enum SLAPriority {
  CRITICAL = 'critical',    // 关键业务
  HIGH = 'high',           // 高优先级
  MEDIUM = 'medium',       // 中优先级
  LOW = 'low',             // 低优先级
}

// SLA升级级别枚举
export enum EscalationLevel {
  LEVEL_1 = 'level_1',     // 一级升级
  LEVEL_2 = 'level_2',     // 二级升级
  LEVEL_3 = 'level_3',     // 三级升级
  LEVEL_4 = 'level_4',     // 四级升级
  MANAGEMENT = 'management', // 管理层升级
}

// 业务时间配置
export interface BusinessHours {
  id: number;
  name: string;
  description?: string;
  timezone: string;
  workingDays: number[]; // 1-7 (周一到周日)
  workingHours: {
    start: string; // HH:mm 格式
    end: string;   // HH:mm 格式
  };
  isDefault: boolean;
  createdAt: string;
  updatedAt: string;
}

// 节假日配置
export interface Holiday {
  id: number;
  name: string;
  date: string; // YYYY-MM-DD 格式
  type: 'national' | 'company' | 'regional';
  isRecurring: boolean; // 是否每年重复
  description?: string;
  createdAt: string;
  updatedAt: string;
}

// SLA升级规则
export interface EscalationRule {
  id: number;
  slaId: number;
  level: EscalationLevel;
  triggerTime: number; // 触发时间（分钟）
  triggerCondition: 'breach' | 'warning' | 'both';
  actions: EscalationAction[];
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

// SLA升级动作
export interface EscalationAction {
  id: number;
  type: 'notification' | 'assignment' | 'escalation' | 'alert';
  target: string; // 目标用户/组/角色
  message?: string;
  priority?: string;
  autoAssign?: boolean;
}

// SLA定义
export interface SLADefinition {
  id: number;
  name: string;
  description?: string;
  type: SLAType;
  priority: SLAPriority;
  targetTime: number; // 目标时间（分钟）
  warningTime: number; // 预警时间（分钟）
  businessHoursId: number;
  businessHours?: BusinessHours;
  escalationRules: EscalationRule[];
  applicableTo: {
    ticketTypes: string[];
    categories: string[];
    priorities: string[];
    departments: string[];
  };
  status: SLAStatus;
  isDefault: boolean;
  version: number;
  createdBy: number;
  createdAt: string;
  updatedAt: string;
}

// SLA实例（工单关联的SLA）
export interface SLAInstance {
  id: number;
  slaDefinitionId: number;
  slaDefinition?: SLADefinition;
  ticketId: number;
  ticketNumber?: string;
  startTime: string;
  targetTime: string;
  warningTime: string;
  breachTime: string;
  actualResolutionTime?: string;
  status: 'active' | 'warning' | 'breached' | 'resolved' | 'suspended';
  currentLevel: EscalationLevel;
  escalationHistory: EscalationEvent[];
  businessTimeUsed: number; // 已使用业务时间（分钟）
  totalTimeUsed: number;    // 已使用总时间（分钟）
  remainingTime: number;    // 剩余时间（分钟）
  isSuspended: boolean;
  suspensionReason?: string;
  createdAt: string;
  updatedAt: string;
}

// SLA升级事件
export interface EscalationEvent {
  id: number;
  slaInstanceId: number;
  level: EscalationLevel;
  triggeredAt: string;
  triggeredBy: 'system' | 'user';
  triggeredByUserId?: number;
  reason: string;
  actions: EscalationAction[];
  isResolved: boolean;
  resolvedAt?: string;
  resolvedBy?: number;
}

// SLA统计信息
export interface SLAStats {
  totalInstances: number;
  activeInstances: number;
  warningInstances: number;
  breachedInstances: number;
  resolvedInstances: number;
  complianceRate: number; // 合规率
  averageResolutionTime: number; // 平均解决时间
  breachRate: number; // 违约率
  escalationRate: number; // 升级率
}

// SLA报告数据
export interface SLAReport {
  period: {
    start: string;
    end: string;
  };
  slaDefinitionId: number;
  slaDefinition?: SLADefinition;
  stats: SLAStats;
  trends: {
    complianceRate: number[];
    resolutionTime: number[];
    breachCount: number[];
  };
  topBreachReasons: Array<{
    reason: string;
    count: number;
    percentage: number;
  }>;
  departmentPerformance: Array<{
    department: string;
    complianceRate: number;
    averageResolutionTime: number;
    breachCount: number;
  }>;
}

// SLA配置请求
export interface CreateSLARequest {
  name: string;
  description?: string;
  type: SLAType;
  priority: SLAPriority;
  targetTime: number;
  warningTime: number;
  businessHoursId: number;
  escalationRules: Omit<EscalationRule, 'id' | 'slaId' | 'createdAt' | 'updatedAt'>[];
  applicableTo: {
    ticketTypes: string[];
    categories: string[];
    priorities: string[];
    departments: string[];
  };
  isDefault?: boolean;
}

export interface UpdateSLARequest {
  name?: string;
  description?: string;
  type?: SLAType;
  priority?: SLAPriority;
  targetTime?: number;
  warningTime?: number;
  businessHoursId?: number;
  escalationRules?: Omit<EscalationRule, 'id' | 'slaId' | 'createdAt' | 'updatedAt'>[];
  applicableTo?: {
    ticketTypes: string[];
    categories: string[];
    priorities: string[];
    departments: string[];
  };
  status?: SLAStatus;
  isDefault?: boolean;
}

// SLA查询参数
export interface SLAQueryParams {
  page?: number;
  pageSize?: number;
  status?: SLAStatus;
  type?: SLAType;
  priority?: SLAPriority;
  keyword?: string;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
}

// SLA服务类
export class SLAService {
  private readonly baseUrl = '/api/v1/sla';

  // SLA定义管理
  async getSLADefinitions(params?: SLAQueryParams): Promise<PaginatedResponse<SLADefinition>> {
    const response = await fetch(`${this.baseUrl}/definitions?${new URLSearchParams(params as any)}`);
    if (!response.ok) throw new Error('Failed to fetch SLA definitions');
    return response.json();
  }

  async getSLADefinition(id: number): Promise<SLADefinition> {
    const response = await fetch(`${this.baseUrl}/definitions/${id}`);
    if (!response.ok) throw new Error('Failed to fetch SLA definition');
    return response.json();
  }

  async createSLADefinition(data: CreateSLARequest): Promise<SLADefinition> {
    const response = await fetch(`${this.baseUrl}/definitions`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (!response.ok) throw new Error('Failed to create SLA definition');
    return response.json();
  }

  async updateSLADefinition(id: number, data: UpdateSLARequest): Promise<SLADefinition> {
    const response = await fetch(`${this.baseUrl}/definitions/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (!response.ok) throw new Error('Failed to update SLA definition');
    return response.json();
  }

  async deleteSLADefinition(id: number): Promise<void> {
    const response = await fetch(`${this.baseUrl}/definitions/${id}`, {
      method: 'DELETE',
    });
    if (!response.ok) throw new Error('Failed to delete SLA definition');
  }

  // SLA实例管理
  async getSLAInstances(params?: any): Promise<PaginatedResponse<SLAInstance>> {
    const response = await fetch(`${this.baseUrl}/instances?${new URLSearchParams(params)}`);
    if (!response.ok) throw new Error('Failed to fetch SLA instances');
    return response.json();
  }

  async getSLAInstance(id: number): Promise<SLAInstance> {
    const response = await fetch(`${this.baseUrl}/instances/${id}`);
    if (!response.ok) throw new Error('Failed to fetch SLA instance');
    return response.json();
  }

  async suspendSLAInstance(id: number, reason: string): Promise<void> {
    const response = await fetch(`${this.baseUrl}/instances/${id}/suspend`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ reason }),
    });
    if (!response.ok) throw new Error('Failed to suspend SLA instance');
  }

  async resumeSLAInstance(id: number): Promise<void> {
    const response = await fetch(`${this.baseUrl}/instances/${id}/resume`, {
      method: 'POST',
    });
    if (!response.ok) throw new Error('Failed to resume SLA instance');
  }

  // SLA统计和报告
  async getSLAStats(params?: any): Promise<SLAStats> {
    const response = await fetch(`${this.baseUrl}/stats?${new URLSearchParams(params)}`);
    if (!response.ok) throw new Error('Failed to fetch SLA stats');
    return response.json();
  }

  async getSLAReport(params: any): Promise<SLAReport> {
    const response = await fetch(`${this.baseUrl}/reports?${new URLSearchParams(params)}`);
    if (!response.ok) throw new Error('Failed to fetch SLA report');
    return response.json();
  }

  // 业务时间管理
  async getBusinessHours(): Promise<BusinessHours[]> {
    const response = await fetch(`${this.baseUrl}/business-hours`);
    if (!response.ok) throw new Error('Failed to fetch business hours');
    return response.json();
  }

  async createBusinessHours(data: Omit<BusinessHours, 'id' | 'createdAt' | 'updatedAt'>): Promise<BusinessHours> {
    const response = await fetch(`${this.baseUrl}/business-hours`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (!response.ok) throw new Error('Failed to create business hours');
    return response.json();
  }

  // 节假日管理
  async getHolidays(): Promise<Holiday[]> {
    const response = await fetch(`${this.baseUrl}/holidays`);
    if (!response.ok) throw new Error('Failed to fetch holidays');
    return response.json();
  }

  async createHoliday(data: Omit<Holiday, 'id' | 'createdAt' | 'updatedAt'>): Promise<Holiday> {
    const response = await fetch(`${this.baseUrl}/holidays`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (!response.ok) throw new Error('Failed to create holiday');
    return response.json();
  }

  // SLA计算工具
  calculateBusinessTime(startTime: string, endTime: string, businessHours: BusinessHours, holidays: Holiday[]): number {
    // 计算两个时间点之间的业务时间（分钟）
    // 这里需要实现复杂的业务时间计算逻辑
    // 考虑工作日、工作时间、节假日等因素
    return 0; // 占位符实现
  }

  calculateSLAStatus(instance: SLAInstance): 'active' | 'warning' | 'breached' | 'resolved' {
    const now = new Date();
    const warningTime = new Date(instance.warningTime);
    const breachTime = new Date(instance.breachTime);

    if (instance.status === 'resolved') return 'resolved';
    if (now >= breachTime) return 'breached';
    if (now >= warningTime) return 'warning';
    return 'active';
  }

  // SLA预警检查
  async checkSLAWarnings(): Promise<SLAInstance[]> {
    const response = await fetch(`${this.baseUrl}/warnings`);
    if (!response.ok) throw new Error('Failed to check SLA warnings');
    return response.json();
  }

  // SLA升级处理
  async processEscalation(slaInstanceId: number, level: EscalationLevel): Promise<void> {
    const response = await fetch(`${this.baseUrl}/instances/${slaInstanceId}/escalate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ level }),
    });
    if (!response.ok) throw new Error('Failed to process escalation');
  }
}

// 导出单例实例
export const slaService = new SLAService();
export default SLAService;
