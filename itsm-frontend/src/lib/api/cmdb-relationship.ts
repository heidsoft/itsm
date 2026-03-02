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
  source_ci_id: number;
  source_ci_name: string;
  source_ci_type: string;
  target_ci_id: number;
  target_ci_name: string;
  target_ci_type: string;
  relationship_type: CIRelationshipType;
  relationship_type_name: string;
  strength: RelationshipStrength;
  impact_level: ImpactLevel;
  is_active: boolean;
  is_discovered: boolean;
  description: string;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

// 拓扑节点
export interface TopologyNode {
  id: number;
  name: string;
  type: string;
  type_name: string;
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
  relationship_type: string;
  relationship_label: string;
  strength: string;
  impact_level: string;
}

// 拓扑图
export interface TopologyGraph {
  nodes: TopologyNode[];
  edges: TopologyEdge[];
  root_ci_id: number;
  depth: number;
  total_nodes: number;
  total_edges: number;
}

// 影响分析项
export interface ImpactAnalysisItem {
  ci_id: number;
  ci_name: string;
  ci_type: string;
  relationship: string;
  impact_level: ImpactLevel;
  distance: number;
  direction: 'upstream' | 'downstream';
  affected_count: number;
}

// 影响分析响应
export interface ImpactAnalysisResponse {
  target_ci: TopologyNode;
  upstream_impact: ImpactAnalysisItem[];
  downstream_impact: ImpactAnalysisItem[];
  critical_dependencies: ImpactAnalysisItem[];
  affected_tickets: AffectedTicket[];
  affected_incidents: AffectedIncident[];
  risk_level: 'critical' | 'high' | 'medium' | 'low';
  summary: string;
}

// 受影响工单
export interface AffectedTicket {
  id: number;
  title: string;
  status: string;
  priority: string;
  created_at: string;
}

// 受影响事件
export interface AffectedIncident {
  id: number;
  title: string;
  status: string;
  severity: string;
  created_at: string;
}

// 创建关系请求
export interface CreateRelationshipRequest {
  source_ci_id: number;
  target_ci_id: number;
  relationship_type: CIRelationshipType;
  strength?: RelationshipStrength;
  impact_level?: ImpactLevel;
  description?: string;
  metadata?: Record<string, unknown>;
}

// 更新关系请求
export interface UpdateRelationshipRequest {
  relationship_type?: CIRelationshipType;
  strength?: RelationshipStrength;
  impact_level?: ImpactLevel;
  description?: string;
  is_active?: boolean;
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
    return httpClient.get('/api/v1/cmdb/relationships/types');
  },

  // 创建关系
  async createRelationship(data: CreateRelationshipRequest): Promise<CIRelationship> {
    return httpClient.post('/api/v1/cmdb/relationships', data);
  },

  // 更新关系
  async updateRelationship(id: number, data: UpdateRelationshipRequest): Promise<CIRelationship> {
    return httpClient.patch(`/api/v1/cmdb/relationships/${id}`, data);
  },

  // 删除关系
  async deleteRelationship(id: number): Promise<void> {
    return httpClient.delete(`/api/v1/cmdb/relationships/${id}`);
  },

  // 获取CI的所有关系
  async getCIRelationships(
    ciId: number,
    options?: {
      includeOutgoing?: boolean;
      includeIncoming?: boolean;
      relationshipType?: CIRelationshipType;
      activeOnly?: boolean;
    }
  ): Promise<{
    outgoing_relations: CIRelationship[];
    incoming_relations: CIRelationship[];
    total_outgoing: number;
    total_incoming: number;
  }> {
    const params = new URLSearchParams();
    if (options?.includeOutgoing !== false) params.append('include_outgoing', 'true');
    if (options?.includeIncoming !== false) params.append('include_incoming', 'true');
    if (options?.relationshipType) params.append('relationship_type', options.relationshipType);
    if (options?.activeOnly) params.append('active_only', 'true');

    const query = params.toString();
    return httpClient.get(`/api/v1/cmdb/cis/${ciId}/relationships${query ? `?${query}` : ''}`);
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

  // 批量创建关系
  async batchCreateRelationships(relationships: CreateRelationshipRequest[]): Promise<{
    created_count: number;
    failed_count: number;
    errors: string[];
  }> {
    return httpClient.post('/api/v1/cmdb/relationships/batch', { relationships });
  },

  // 获取可用的目标CI列表（用于创建关系时选择）
  async getAvailableCIs(ciId: number, search?: string): Promise<TopologyNode[]> {
    const params = search ? `?search=${encodeURIComponent(search)}` : '';
    return httpClient.get(`/api/v1/cmdb/cis/${ciId}/available-cis${params}`);
  },
};

export default CIRelationshipAPI;
