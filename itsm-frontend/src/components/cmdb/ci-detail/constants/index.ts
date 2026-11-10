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
  critical: 'Critical',
  high: 'High',
  medium: 'Medium',
  low: 'Low',
};

// CI 变更历史的 operation 展示色/标签。
// 键取后端写入值（service/configuration_item_service.go、service/ci_history_service.go）：
// create / update / delete（退役归档）/ revert / lifecycle_update。
// 未知值原样显示、颜色回 default，不做枚举断言以免新值被吞成空白。
export const CI_OPERATION_COLORS: Record<string, string> = {
  create: 'green',
  update: 'blue',
  delete: 'red',
  revert: 'orange',
  lifecycle_update: 'purple',
};

export const CI_OPERATION_LABELS: Record<string, string> = {
  create: 'Create',
  update: 'Update',
  delete: 'Retired',
  revert: 'Version rollback',
  lifecycle_update: 'Lifecycle changes',
};

export const ciOperationColor = (operation: string): string =>
  CI_OPERATION_COLORS[operation] ?? 'default';

export const ciOperationLabel = (operation: string): string =>
  CI_OPERATION_LABELS[operation] ?? operation;
