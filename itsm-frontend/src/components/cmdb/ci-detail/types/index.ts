/**
 * CI Detail 相关类型定义
 */

import type { ConfigurationItem, CIType } from '@/types/biz/cmdb';
import type { FC, ReactNode } from 'react';

// ============ 影响分析类型 ============

export interface ImpactAnalysisData {
  target_ci: unknown;
  upstream_impact: ImpactAnalysisItem[];
  downstream_impact: ImpactAnalysisItem[];
  critical_dependencies: ImpactAnalysisItem[];
  affected_tickets: AffectedTicket[];
  affected_incidents: AffectedIncident[];
  risk_level: 'critical' | 'high' | 'medium' | 'low';
  summary: string;
}

export interface ImpactAnalysisItem {
  ci_id: number;
  ci_name: string;
  ci_type: string;
  relationship: string;
  impact_level: 'critical' | 'high' | 'medium' | 'low';
  distance: number;
  direction: string; // 'upstream' or 'downstream'
}

export interface AffectedTicket {
  id: number;
  ticketNumber: string;
  title: string;
  status: string;
  priority: string;
  [key: string]: unknown;
}

export interface AffectedIncident {
  id: number;
  title: string;
  status: string;
  severity: string;
  [key: string]: unknown;
}

// ============ 变更历史类型 ============

export interface ChangeHistoryData {
  logs: ChangeLog[];
  total: number;
  page: number;
  page_size: number;
}

export interface ChangeLog {
  id: number;
  action: 'create' | 'update' | 'delete' | 'relationship_added' | 'relationship_removed';
  resource?: string;
  path?: string;
  Method?: string;
  StatusCode?: number | string;
  created_at?: string;
  updated_by?: string;
  updated_at?: string;
  description?: string;
  [key: string]: unknown;
}

// ============ Hook 返回类型 ============

export interface UseCIDetailReturn {
  ci: ConfigurationItem | null;
  types: CIType[];
  loading: boolean;
  impactAnalysis: ImpactAnalysisData | null;
  impactLoading: boolean;
  changeHistory: ChangeHistoryData | null;
  historyLoading: boolean;
  loadDetail: () => Promise<void>;
  loadImpactAnalysis: () => Promise<void>;
  loadChangeHistory: () => Promise<void>;
  typeInfo: CIType | undefined;
}

// ============ Props 类型 ============

export interface CIDetailProps {
  // 可以添加 props，目前为空
}

export interface CIProps {
  ci: ConfigurationItem;
  typeInfo?: CIType;
  onRefresh?: () => void;
}
