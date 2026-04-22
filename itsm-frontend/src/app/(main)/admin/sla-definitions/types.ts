/**
 * SLA 定义管理页面类型定义
 */

// SLA定义数据类型
export interface SLADefinition {
  id: string;
  name: string;
  description: string;
  serviceType: string;
  priority: 'P1' | 'P2' | 'P3' | 'P4';
  responseTime: string;
  resolutionTime: string;
  availability: string;
  businessHours: string;
  escalationRules: string[];
  applicableServices: string[];
  status: 'active' | 'inactive' | 'draft';
  createdAt: string;
  updatedAt: string;
  createdBy: string;
}

// 表单值类型
export interface SLAFormValues {
  name: string;
  description: string;
  serviceType: string;
  priority: 'P1' | 'P2' | 'P3' | 'P4';
  responseTime: number;
  resolutionTime: number;
  availability: number;
  businessHours: string;
  status: 'active' | 'inactive' | 'draft';
}

// 筛选条件类型
export interface SLAFilters {
  search: string;
  priority: string;
  status: string;
  serviceType: string;
}
