/**
 * 工单关联类型定义
 * 支持父子关系、依赖关系、相关关系等多种工单关联类型
 */

import type { Ticket } from './ticket';

// ==================== 关联类型枚举 ====================

/**
 * 工单关联类型
 */
export enum TicketRelationType {
  // 层级关系
  PARENT_CHILD = 'parent_child',         // 父子关系
  
  // 依赖关系
  BLOCKS = 'blocks',                     // 阻塞（A阻塞B，B依赖A）
  BLOCKED_BY = 'blocked_by',             // 被阻塞
  DEPENDS_ON = 'depends_on',             // 依赖于
  
  // 关联关系
  RELATES_TO = 'relates_to',             // 相关
  DUPLICATES = 'duplicates',             // 重复（A重复B）
  DUPLICATED_BY = 'duplicated_by',       // 被重复
  
  // 因果关系
  CAUSES = 'causes',                     // 导致
  CAUSED_BY = 'caused_by',               // 由...导致
  
  // 替代关系
  REPLACES = 'replaces',                 // 替代
  REPLACED_BY = 'replaced_by',           // 被替代
  
  // 分解关系
  SPLITS_FROM = 'splits_from',           // 分离自
  MERGED_INTO = 'merged_into',           // 合并到
}

/**
 * 关联方向
 */
export enum RelationDirection {
  INBOUND = 'inbound',                   // 入向（被其他工单关联）
  OUTBOUND = 'outbound',                 // 出向（关联其他工单）
  BIDIRECTIONAL = 'bidirectional',       // 双向
}

// ==================== 工单关联接口 ====================

/**
 * 工单关联
 */
export interface TicketRelation {
  id: string;
  sourceTicketId: number;                // 源工单ID
  sourceTicketNumber: string;            // 源工单编号
  targetTicketId: number;                // 目标工单ID
  targetTicketNumber: string;            // 目标工单编号
  relationType: TicketRelationType;      // 关联类型
  direction: RelationDirection;          // 关联方向
  description?: string;                  // 关联描述
  createdBy: number;                     // 创建人ID
  createdByName: string;                 // 创建人姓名
  createdAt: Date;                       // 创建时间
  metadata?: Record<string, any>;        // 元数据
}

/**
 * 工单关联详情（包含关联工单的基本信息）
 */
export interface TicketRelationWithDetails extends TicketRelation {
  sourceTicket?: Partial<Ticket>;        // 源工单基本信息
  targetTicket?: Partial<Ticket>;        // 目标工单基本信息
}

// ==================== 父子关系 ====================

/**
 * 父工单信息
 */
export interface ParentTicket {
  id: number;
  ticketNumber: string;
  title: string;
  status: string;
  priority: string;
  assignee?: {
    id: number;
    name: string;
    avatar?: string;
  };
  progress?: number;                     // 进度（基于子工单）
  childrenCount: number;                 // 子工单数量
  completedChildrenCount: number;        // 已完成子工单数量
}

/**
 * 子工单信息
 */
export interface ChildTicket {
  id: number;
  ticketNumber: string;
  title: string;
  status: string;
  priority: string;
  assignee?: {
    id: number;
    name: string;
    avatar?: string;
  };
  order: number;                         // 排序顺序
  isBlocking: boolean;                   // 是否阻塞其他子工单
}

/**
 * 工单层级结构
 */
export interface TicketHierarchy {
  ticket: Ticket;
  parent?: ParentTicket;
  children: ChildTicket[];
  ancestors: ParentTicket[];             // 祖先工单链
  depth: number;                         // 层级深度
  path: string[];                        // 路径（工单编号）
}

// ==================== 依赖关系 ====================

/**
 * 工单依赖
 */
export interface TicketDependency {
  id: string;
  ticketId: number;
  dependsOnTicketId: number;
  dependsOnTicketNumber: string;
  dependsOnTicketTitle: string;
  dependsOnTicketStatus: string;
  dependencyType: 'hard' | 'soft';       // 硬依赖/软依赖
  isBlocking: boolean;                   // 是否当前阻塞
  canStart: boolean;                     // 是否可以开始
  estimatedUnblockDate?: Date;           // 预计解除阻塞日期
  createdAt: Date;
}

/**
 * 依赖图
 */
export interface DependencyGraph {
  nodes: DependencyNode[];
  edges: DependencyEdge[];
  criticalPath: number[];                // 关键路径
  estimatedCompletionDate?: Date;        // 预计完成日期
}

export interface DependencyNode {
  ticketId: number;
  ticketNumber: string;
  title: string;
  status: string;
  priority: string;
  estimatedDuration?: number;            // 预计工期（小时）
  isCritical: boolean;                   // 是否在关键路径上
  level: number;                         // 层级
}

export interface DependencyEdge {
  from: number;
  to: number;
  type: 'blocks' | 'depends_on';
  isBlocking: boolean;
}

// ==================== 关联操作 ====================

/**
 * 创建关联请求
 */
export interface CreateRelationRequest {
  sourceTicketId: number;
  targetTicketId: number;
  relationType: TicketRelationType;
  description?: string;
  metadata?: Record<string, any>;
}

/**
 * 批量创建关联请求
 */
export interface BatchCreateRelationsRequest {
  sourceTicketId: number;
  relations: Array<{
    targetTicketId: number;
    relationType: TicketRelationType;
    description?: string;
  }>;
}

/**
 * 更新关联请求
 */
export interface UpdateRelationRequest {
  description?: string;
  metadata?: Record<string, any>;
}

/**
 * 删除关联请求
 */
export interface DeleteRelationRequest {
  relationId: string;
  reason?: string;
}

// ==================== 关联验证 ====================

/**
 * 关联验证结果
 */
export interface RelationValidation {
  isValid: boolean;
  canCreate: boolean;
  errors: ValidationError[];
  warnings: ValidationWarning[];
  suggestions: string[];
}

export interface ValidationError {
  code: string;
  message: string;
  field?: string;
}

export interface ValidationWarning {
  code: string;
  message: string;
  severity: 'low' | 'medium' | 'high';
}

/**
 * 关联冲突
 */
export interface RelationConflict {
  type: 'circular' | 'duplicate' | 'invalid_state' | 'permission';
  message: string;
  conflictingRelations: TicketRelation[];
  resolution?: string;
}

// ==================== 关联搜索和过滤 ====================

/**
 * 关联查询参数
 */
export interface RelationQuery {
  ticketId: number;
  relationType?: TicketRelationType;
  direction?: RelationDirection;
  includeDetails?: boolean;
  recursive?: boolean;                   // 是否递归查询
  maxDepth?: number;                     // 最大递归深度
}

/**
 * 关联搜索结果
 */
export interface RelationSearchResult {
  relations: TicketRelationWithDetails[];
  total: number;
  hasMore: boolean;
}

// ==================== 关联统计 ====================

/**
 * 工单关联统计
 */
export interface TicketRelationStats {
  totalRelations: number;
  relationsByType: Record<TicketRelationType, number>;
  inboundCount: number;
  outboundCount: number;
  parentCount: number;
  childrenCount: number;
  blockedByCount: number;
  blockingCount: number;
  relatedCount: number;
  duplicateCount: number;
}

/**
 * 关联影响分析
 */
export interface RelationImpactAnalysis {
  ticketId: number;
  affectedTickets: Array<{
    ticketId: number;
    ticketNumber: string;
    title: string;
    impactType: 'direct' | 'indirect';
    impactLevel: 'low' | 'medium' | 'high';
    reason: string;
  }>;
  totalAffected: number;
  criticalImpacts: number;
  recommendations: string[];
}

// ==================== 关联图谱 ====================

/**
 * 关系图谱节点
 */
export interface RelationGraphNode {
  id: number;
  ticketNumber: string;
  title: string;
  status: string;
  priority: string;
  type: string;
  assignee?: {
    id: number;
    name: string;
    avatar?: string;
  };
  depth: number;
  children?: RelationGraphNode[];
}

/**
 * 关系图谱配置
 */
export interface RelationGraphConfig {
  maxDepth: number;
  relationTypes: TicketRelationType[];
  direction: RelationDirection;
  showLabels: boolean;
  showStatus: boolean;
  layout: 'tree' | 'force' | 'hierarchical';
  highlightCriticalPath: boolean;
}

/**
 * 关系图谱数据
 */
export interface RelationGraphData {
  nodes: RelationGraphNode[];
  edges: Array<{
    source: number;
    target: number;
    type: TicketRelationType;
    label?: string;
  }>;
  config: RelationGraphConfig;
}

// ==================== 关联建议 ====================

/**
 * 关联建议
 */
export interface RelationSuggestion {
  targetTicketId: number;
  targetTicketNumber: string;
  targetTicketTitle: string;
  suggestedRelationType: TicketRelationType;
  confidence: number;                    // 置信度（0-1）
  reason: string;
  similarity?: number;                   // 相似度
}

/**
 * 智能关联推荐
 */
export interface SmartRelationRecommendation {
  ticketId: number;
  suggestions: RelationSuggestion[];
  duplicateCandidates: Array<{
    ticketId: number;
    ticketNumber: string;
    title: string;
    similarity: number;
    reason: string;
  }>;
  relatedTickets: Array<{
    ticketId: number;
    ticketNumber: string;
    title: string;
    relevanceScore: number;
    commonTags: string[];
    commonUsers: string[];
  }>;
}

// ==================== 关联历史 ====================

/**
 * 关联操作历史
 */
export interface RelationHistory {
  id: string;
  relationId: string;
  action: 'created' | 'updated' | 'deleted';
  relationType: TicketRelationType;
  sourceTicketId: number;
  targetTicketId: number;
  performedBy: number;
  performedByName: string;
  performedAt: Date;
  reason?: string;
  metadata?: Record<string, any>;
}

// ==================== 批量关联操作 ====================

/**
 * 批量关联操作结果
 */
export interface BatchRelationResult {
  success: boolean;
  created: number;
  failed: number;
  errors: Array<{
    targetTicketId: number;
    error: string;
  }>;
}

// ==================== 关联权限 ====================

/**
 * 关联权限
 */
export interface RelationPermissions {
  canCreate: boolean;
  canUpdate: boolean;
  canDelete: boolean;
  canViewAll: boolean;
  allowedRelationTypes: TicketRelationType[];
  restrictions: string[];
}

// ==================== 导出所有类型 ====================

export default TicketRelation;

