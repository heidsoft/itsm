/**
 * CI Detail 常量定义
 */

import { CIStatus, CIStatusLabels } from '@/constants/cmdb';

// 状态颜色映射
export const STATUS_COLORS: Record<string, string> = {
  [CIStatus.ACTIVE]: 'green',
  [CIStatus.INACTIVE]: 'default',
  [CIStatus.MAINTENANCE]: 'orange',
  [CIStatus.DECOMMISSIONED]: 'red',
};

// 风险等级颜色
export const RISK_LEVEL_COLORS: Record<string, string> = {
  critical: 'red',
  high: 'orange',
  medium: 'gold',
  low: 'green',
};

// 风险等级标签
export const RISK_LEVEL_LABELS: Record<string, string> = {
  critical: '严重',
  high: '高',
  medium: '中',
  low: '低',
};

// 动作类型颜色
export const ACTION_COLORS: Record<string, string> = {
  create: 'green',
  update: 'blue',
  delete: 'red',
  relationship_added: 'purple',
  relationship_removed: 'volcano',
};
