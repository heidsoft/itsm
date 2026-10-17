/**
 * SLA 定义管理页面类型定义
 */

import type { SLAEscalationLevel } from '@/lib/api/sla-api';

// SLA定义数据类型
export interface SLADefinition {
  id: string;
  name: string;
  description: string;
  serviceType: string;
  /** 后端原始优先级值，展示时经 priorityLabel 归一，未识别值原样显示 */
  priority: string;
  responseTimeMinutes: number;
  resolutionTimeMinutes: number;
  /** 由 businessHours 配置投影出的展示文案，未配置时为「未配置」 */
  businessHours: string;
  escalationLevels: SLAEscalationLevel[];
  status: 'active' | 'inactive';
  createdAt: string;
  updatedAt: string;
}

// 表单值类型：时限字段与后端请求体同单位（分钟），页面不做换算
export interface SLAFormValues {
  name: string;
  description: string;
  serviceType: string;
  priority: string;
  responseTimeMinutes: number;
  resolutionTimeMinutes: number;
  isActive: boolean;
}

// 列表统计，全部来自后端返回的真实字段
export interface SLAStats {
  total: number;
  active: number;
  inactive: number;
  serviceTypeCount: number;
}
