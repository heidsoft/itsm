/**
 * CI Detail 相关类型定义
 */

import type { ConfigurationItem, CIType } from '@/types/biz/cmdb';
import type { FC, ReactNode } from 'react';

// ============ 影响分析类型 ============

export interface ImpactAnalysisData {
  targetCi: unknown;
  upstreamImpact: ImpactAnalysisItem[];
  downstreamImpact: ImpactAnalysisItem[];
  criticalDependencies: ImpactAnalysisItem[];
  affectedTickets: AffectedTicket[];
  affectedIncidents: AffectedIncident[];
  riskLevel: 'critical' | 'high' | 'medium' | 'low';
  summary: string;
}

export interface ImpactAnalysisItem {
  ciId: number;
  ciName: string;
  ciType: string;
  relationship: string;
  impactLevel: 'critical' | 'high' | 'medium' | 'low';
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
  pageSize: number;
}

export interface ChangeLog {
  id: number;
  action: 'create' | 'update' | 'delete' | 'relationship_added' | 'relationship_removed';
  resource?: string;
  path?: string;
  Method?: string;
  StatusCode?: number | string;
  createdAt?: string;
  updatedBy?: string;
  updatedAt?: string;
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
