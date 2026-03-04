/**
 * SLA Violation Monitor 相关类型定义
 */

import type { SLAViolation } from '@/lib/api/sla-api';
export type { SLAViolation };
import type { FC, ReactNode } from 'react';
import type dayjs from 'dayjs';

// ============ Props 类型 ============

export interface SLAViolationMonitorProps {
  autoRefresh?: boolean;
  refreshInterval?: number;
  onViolationUpdate?: (violation: SLAViolation) => void;
}

export interface SLAFilterPanelProps {
  filters: SLAFilters;
  onFiltersChange: (filters: SLAFilters) => void;
  onRefresh: () => void;
  loading?: boolean;
}

export interface SLAStatisticsCardsProps {
  stats: SLAStats;
  loading?: boolean;
}

export interface SLATableProps {
  violations: SLAViolation[];
  loading: boolean;
  selectedRowKeys: React.Key[];
  onRowSelect: (keys: React.Key[]) => void;
  onView: (violation: SLAViolation) => void;
  onResolve: (violation: SLAViolation) => void;
  onAcknowledge: (violation: SLAViolation) => void;
}

export interface SLAChartPanelProps {
  violations: SLAViolation[];
}

export interface SLAViolationDetailModalProps {
  violation: SLAViolation | null;
  visible: boolean;
  onClose: () => void;
  onResolve: () => void;
  onAcknowledge: () => void;
}

export interface SLAAlertRulesPanelProps {
  rules: SLAAlertRule[];
  onAddRule: () => void;
  onEditRule: (rule: SLAAlertRule) => void;
  onDeleteRule: (ruleId: number) => void;
  onToggleRule: (rule: SLAAlertRule) => void;
}

export interface SLASettingsDrawerProps {
  visible: boolean;
  onClose: () => void;
}

// ============ 过滤器类型 ============

export interface SLAFilters {
  status: string;
  severity: string;
  type: string;
  dateRange: [any, any] | null; // 简化: dayjs range
  search: string;
}

// ============ 统计数据 ============

export interface SLAStats {
  total: number;
  open: number;
  resolved: number;
  critical: number;
}

// ============ 告警规则类型 ============

export interface SLAAlertRule {
  id: number;
  name: string;
  sla_definition_id: number;
  alert_level: 'warning' | 'critical';
  trigger_conditions: {
    time_threshold_percent: number;
    violation_types: string[];
  };
  notification_channels: string[];
  is_active: boolean;
  created_at: string;
}

// ============ Hook 返回类型 ============

export interface UseSLAViolationsReturn {
  violations: SLAViolation[];
  filteredViolations: SLAViolation[];
  filters: SLAFilters;
  loading: boolean;
  selectedRowKeys: React.Key[];
  setFilters: (filters: SLAFilters) => void;
  refresh: () => Promise<void>;
  selectRow: (keys: React.Key[]) => void;
  clearSelection: () => void;
}

export interface UseSLAStatisticsReturn {
  stats: SLAStats;
  calculateStats: (violations: SLAViolation[]) => SLAStats;
}

export interface UseSLARefreshReturn {
  isRefreshing: boolean;
  lastRefresh: Date | null;
  startAutoRefresh: (interval: number, callback: () => Promise<void>) => void;
  stopAutoRefresh: () => void;
  refreshNow: (callback: () => Promise<void>) => Promise<void>;
}
