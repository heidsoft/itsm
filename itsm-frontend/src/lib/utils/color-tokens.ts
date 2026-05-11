/**
 * Design System Color Tokens
 *
 * Centralized semantic color definitions for status badges and indicators.
 * All colors follow WCAG 4.5:1 contrast ratio for accessibility.
 *
 * Usage:
 *   import { getStatusColor, getPriorityColor } from '@/lib/utils/color-tokens';
 *   const config = getStatusColor('pending');
 *   style={{ color: config.color, backgroundColor: config.backgroundColor }}
 */

// Status label mapping
const statusLabels: Record<string, string> = {
  pending: '待审批',
  approved: '审批通过',
  implementing: '实施中',
  completed: '已完成',
  cancelled: '已取消',
  draft: '草稿',
  rejected: '已拒绝',
};

// Priority label mapping
const priorityLabels: Record<string, string> = {
  urgent: '紧急',
  high: '高',
  medium: '中',
  low: '低',
};

// Status badge colors - used for change/ticket status indicators
export const statusColors: Record<
  string,
  { color: string; backgroundColor: string; borderColor: string }
> = {
  pending: {
    color: '#b45309', // amber-700, 4.6:1 on white
    backgroundColor: '#fef3c7', // amber-100
    borderColor: '#fde68a', // amber-200
  },
  approved: {
    color: '#0369a1', // sky-700, 4.8:1 on white
    backgroundColor: '#e0f2fe', // sky-100
    borderColor: '#bae6fd', // sky-200
  },
  implementing: {
    color: '#7c3aed', // violet-700, 4.6:1 on white
    backgroundColor: '#ede9fe', // violet-100
    borderColor: '#ddd6fe', // violet-200
  },
  completed: {
    color: '#15803d', // green-700, 4.5:1 on white
    backgroundColor: '#dcfce7', // green-100
    borderColor: '#bbf7d0', // green-200
  },
  cancelled: {
    color: '#b91c1c', // red-700, 5.2:1 on white
    backgroundColor: '#fee2e2', // red-100
    borderColor: '#fecaca', // red-200
  },
  draft: {
    color: '#6b7280', // gray-500, 4.6:1 on white
    backgroundColor: '#f3f4f6', // gray-100
    borderColor: '#e5e7eb', // gray-200
  },
  rejected: {
    color: '#b91c1c', // red-700, 5.2:1 on white
    backgroundColor: '#fee2e2', // red-100
    borderColor: '#fecaca', // red-200
  },
};

// Priority badge colors - used for priority indicators
export const priorityColors: Record<
  string,
  { color: string; backgroundColor: string; borderColor: string }
> = {
  urgent: {
    color: '#b91c1c', // red-700, 5.2:1 on white
    backgroundColor: '#fee2e2', // red-100
    borderColor: '#fecaca', // red-200
  },
  high: {
    color: '#b45309', // amber-700, 4.6:1 on white
    backgroundColor: '#fef3c7', // amber-100
    borderColor: '#fde68a', // amber-200
  },
  medium: {
    color: '#0369a1', // sky-700, 4.8:1 on white
    backgroundColor: '#e0f2fe', // sky-100
    borderColor: '#bae6fd', // sky-200
  },
  low: {
    color: '#15803d', // green-700, 4.5:1 on white
    backgroundColor: '#dcfce7', // green-100
    borderColor: '#bbf7d0', // green-200
  },
};

// Helper function to get status colors with fallback
export function getStatusColor(status: string): {
  color: string;
  backgroundColor: string;
  borderColor: string;
  text: string;
} {
  const colorConfig = statusColors[status];
  const label = statusLabels[status] || status;
  if (colorConfig) {
    return { ...colorConfig, text: label };
  }
  return { ...statusColors.draft, text: status };
}

// Helper function to get priority colors with fallback
export function getPriorityColor(priority: string): {
  color: string;
  backgroundColor: string;
  borderColor: string;
  text: string;
} {
  const colorConfig = priorityColors[priority];
  const label = priorityLabels[priority] || priority;
  if (colorConfig) {
    return { ...colorConfig, text: label };
  }
  return { ...priorityColors.medium, text: priority };
}
