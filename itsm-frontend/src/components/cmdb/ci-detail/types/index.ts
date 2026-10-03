/**
 * CI Detail 相关类型定义
 */

import type { CIHistoryItem, ConfigurationItem, CIType } from '@/types/biz/cmdb';
import type { PaginationResponse } from '@/lib/api/types';

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

// GET /api/v1/cmdb/cis/:id/history 就是平台五键信封（items/total/page/pageSize/totalPages），
// 后端从未返回过 logs 键；条目字段以 dto.CIHistoryResponse 为唯一真相。
export type ChangeHistoryData = PaginationResponse<CIHistoryItem>;

// ============ Hook 返回类型 ============

export interface UseCIDetailReturn {
  ci: ConfigurationItem | null;
  types: CIType[];
  loading: boolean;
  impactAnalysis: ImpactAnalysisData | null;
  impactLoading: boolean;
  changeHistory: ChangeHistoryData | null;
  historyLoading: boolean;
  historyError: boolean;
  loadDetail: () => Promise<void>;
  loadImpactAnalysis: () => Promise<void>;
  loadChangeHistory: () => Promise<void>;
  loadHistoryPage: (page: number) => Promise<void>;
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
