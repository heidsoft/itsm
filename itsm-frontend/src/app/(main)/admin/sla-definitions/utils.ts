/**
 * SLA 定义管理页面工具函数
 */

import type { SLADefinition } from './types';
import type { SLADefinition as APISLADefinition } from '@/lib/api/sla-api';

/**
 * 转换API数据格式为页面格式
 * 后端 SLADefinitionResponse 返回 camelCase: responseTime, resolutionTime
 */
export const transformSLA = (item: APISLADefinition): SLADefinition => ({
  id: String(item.id),
  name: item.name,
  description: item.description,
  serviceType: item.serviceType || '',
  priority: (item.priority || 'P3') as SLADefinition['priority'],
  // 后端返回 camelCase 格式: responseTime, resolutionTime
  responseTime: `${item.responseTime || 0}分钟`,
  resolutionTime: `${item.resolutionTime || 0}分钟`,
  // 可用性字段
  availability: `${item.availabilityTarget || item.availability || 99.9}%`,
  businessHours: '7x24',
  escalationRules: [],
  applicableServices: [],
  status: item.isActive ? 'active' : 'inactive',
  createdAt: item.createdAt || '',
  updatedAt: item.updatedAt || '',
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
