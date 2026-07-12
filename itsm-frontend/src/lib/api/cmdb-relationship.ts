/**
 * CMDB CI关系 API 客户端
 */

import { httpClient } from '@/lib/api/http-client';

// 关系类型
export type CIRelationshipType =
  | 'depends_on'
  | 'hosts'
  | 'hosted_on'
  | 'connects_to'
  | 'runs_on'
  | 'contains'
  | 'part_of'
  | 'impacts'
  | 'impacted_by'
  | 'owns'
  | 'owned_by'
  | 'uses'
  | 'used_by';

// 关系强度
export type RelationshipStrength = 'critical' | 'high' | 'medium' | 'low';

// 影响程度
export type ImpactLevel = 'critical' | 'high' | 'medium' | 'low';

// CI关系
export interface CIRelationship {
  id: number;
  sourceCiId: number;
  sourceCiName: string;
  sourceCiType: string;
  targetCiId: number;
  targetCiName: string;
  targetCiType: string;
  relationshipType: CIRelationshipType;
  relationshipTypeName: string;
  strength: RelationshipStrength;
  impactLevel: ImpactLevel;
  isActive: boolean;
  isDiscovered: boolean;
  description: string;
  metadata: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
}

// 拓扑节点
export interface TopologyNode {
  id: number;
  name: string;
  type: string;
  typeName: string;
  status: string;
  criticality: string;
  icon?: string;
  attributes: Record<string, unknown>;
}

// 拓扑边
export interface TopologyEdge {
  id: number;
  source: number;
  target: number;
  relationshipType: string;
  relationshipLabel: string;
  strength: string;
  impactLevel: string;
}

// 拓扑图
export interface TopologyGraph {
  nodes: TopologyNode[];
  edges: TopologyEdge[];
  rootCiId: number;
  depth: number;
  totalNodes: number;
  totalEdges: number;
}

// 影响分析项
export interface ImpactAnalysisItem {
  ciId: number;
  ciName: string;
  ciType: string;
  relationship: string;
  impactLevel: ImpactLevel;
  distance: number;
  direction: 'upstream' | 'downstream';
  affectedCount: number;
}

// 影响分析响应
export interface ImpactAnalysisResponse {
  sourceCiId: number;
  targetCi: TopologyNode;
  graph: TopologyGraph;
  upstreamImpact: ImpactAnalysisItem[];
  downstreamImpact: ImpactAnalysisItem[];
  criticalDependencies: ImpactAnalysisItem[];
  affectedTickets: AffectedTicket[];
  affectedIncidents: AffectedIncident[];
  riskLevel: 'critical' | 'high' | 'medium' | 'low';
  summary: string;
  totalImpacted: number;
}

// 受影响工单
export interface AffectedTicket {
  id: number;
  title: string;
  status: string;
  priority: string;
  createdAt: string;
}

// 受影响事件
export interface AffectedIncident {
  id: number;
  title: string;
  status: string;
  severity: string;
  createdAt: string;
}

// 创建关系请求
export interface CreateRelationshipRequest {
  sourceCiId: number;
  targetCiId: number;
  relationshipType: CIRelationshipType;
  strength?: RelationshipStrength;
  impactLevel?: ImpactLevel;
  description?: string;
  metadata?: Record<string, unknown>;
}

// 更新关系请求
export interface UpdateRelationshipRequest {
  relationshipType?: CIRelationshipType;
  strength?: RelationshipStrength;
  impactLevel?: ImpactLevel;
  description?: string;
  isActive?: boolean;
}

// 关系类型信息
export interface RelationshipTypeInfo {
  type: CIRelationshipType;
  name: string;
  description: string;
  direction: 'uni-directional' | 'bi-directional';
  icon: string;
}

// CI关系API
export const CIRelationshipAPI = {
  // 获取关系类型列表
  async getRelationshipTypes(): Promise<RelationshipTypeInfo[]> {
	const response = await httpClient.get<RelationshipTypeInfo[] | { types: RelationshipTypeInfo[] }>(
	  '/api/v1/configuration-items/relationship-types'
	);
	return Array.isArray(response) ? response : response.types ?? [];
  },

  // 创建关系
  async createRelationship(data: CreateRelationshipRequest): Promise<CIRelationship> {
    return httpClient.post('/api/v1/configuration-items/relationships', data);
  },

  async updateRelationship(id: number, data: UpdateRelationshipRequest): Promise<CIRelationship> {
	return httpClient.put(`/api/v1/configuration-items/relationships/${id}`, data);
  },

  // 删除关系
  async deleteRelationship(id: number): Promise<void> {
    return httpClient.delete(`/api/v1/configuration-items/relationships/${id}`);
  },

  // 获取CI的所有关系（返回平铺列表，前端自行按 source_ci_id/target_ci_id 分组）
  async getCIRelationships(
    ciId: number,
    options?: {
      includeOutgoing?: boolean;
      includeIncoming?: boolean;
      relationshipType?: CIRelationshipType;
      activeOnly?: boolean;
    }
  ): Promise<{
    outgoingRelations: CIRelationship[];
    incomingRelations: CIRelationship[];
    totalOutgoing: number;
    totalIncoming: number;
  }> {
	const list = await httpClient.get<CIRelationship[]>(
	  `/api/v1/configuration-items/${ciId}/relationships`,
	  options?.relationshipType ? { relationshipType: options.relationshipType } : undefined
	);
	const arr = Array.isArray(list) ? list : [];
    const outgoing = arr.filter(r => r.sourceCiId === ciId);
    const incoming = arr.filter(r => r.targetCiId === ciId);
    return {
      outgoingRelations: outgoing,
      incomingRelations: incoming,
      totalOutgoing: outgoing.length,
      totalIncoming: incoming.length,
    };
  },

  // 获取拓扑图
  async getTopologyGraph(ciId: number, depth?: number): Promise<TopologyGraph> {
    const params = depth ? `?depth=${depth}` : '';
    return httpClient.get(`/api/v1/configuration-items/${ciId}/topology${params}`);
  },

  // 影响分析
  async analyzeImpact(ciId: number): Promise<ImpactAnalysisResponse> {
    return httpClient.get(`/api/v1/configuration-items/${ciId}/impact-analysis`);
  },

  // 批量创建关系（逐个调用，后端无 batch 路由）
  async batchCreateRelationships(relationships: CreateRelationshipRequest[]): Promise<{
    createdCount: number;
    failedCount: number;
    errors: string[];
  }> {
    let created = 0;
    let failed = 0;
    const errors: string[] = [];
    for (const rel of relationships) {
      try {
        await CIRelationshipAPI.createRelationship(rel);
        created++;
      } catch (e: any) {
        failed++;
        errors.push(e?.message ?? String(e));
      }
    }
    return { createdCount: created, failedCount: failed, errors };
  },

  // 获取可用的目标CI列表（通过 getCIs 搜索）
  async getAvailableCIs(ciId: number, search?: string): Promise<TopologyNode[]> {
    const { CMDBApi } = await import('./cmdb-api');
	const result = await CMDBApi.getCIs({ search, size: 200 });
	const items = result.items ?? [];
    const filtered = items.filter((ci: any) => ci.id !== ciId);
    return filtered.map((ci: any) => ({
      id: ci.id,
      name: ci.name,
      type: ci.ciType ?? ci.type ?? '',
      typeName: ci.ciType ?? ci.type ?? '',
      status: ci.status ?? '',
      criticality: ci.criticality ?? '',
      attributes: ci.attributes ?? {},
    }));
  },
};

export default CIRelationshipAPI;
