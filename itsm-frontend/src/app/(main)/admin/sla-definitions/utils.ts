/**
 * SLA 定义管理页面工具函数
 */

import type { SLADefinition } from './types';
import type { SLADefinition as APISLADefinition } from '@/lib/api/sla-api';

/**
 * 转换API数据格式为页面格式
 */
export const transformSLA = (item: APISLADefinition): SLADefinition => ({
  id: String(item.id),
  name: item.name,
  description: item.description,
  serviceType: item.service_type,
  priority: item.priority as SLADefinition['priority'],
  responseTime: `${item.response_time_minutes}分钟`,
  resolutionTime: `${item.resolution_time_minutes}分钟`,
  availability: `${item.availability_target}%`,
  businessHours: '7x24',
  escalationRules: [],
  applicableServices: [],
  status: item.is_active ? 'active' : 'inactive',
  createdAt: item.created_at,
  updatedAt: item.updated_at,
  createdBy: '系统',
});

/**
 * 获取优先级颜色
 */
export const getPriorityColor = (priority: string): string => {
  const colors: Record<string, string> = {
    P1: 'red',
    P2: 'orange',
    P3: 'blue',
    P4: 'green',
  };
  return colors[priority] || 'default';
};

/**
 * 获取状态颜色
 */
export const getStatusColor = (status: string): string => {
  const colors: Record<string, string> = {
    active: 'green',
    inactive: 'default',
    draft: 'orange',
  };
  return colors[status] || 'default';
};

/**
 * 获取状态文本
 */
export const getStatusText = (status: string): string => {
  const texts: Record<string, string> = {
    active: '已激活',
    inactive: '已停用',
    draft: '草稿',
  };
  return texts[status] || status;
};
