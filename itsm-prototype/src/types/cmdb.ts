/**
 * CMDB (配置管理数据库) 类型定义
 * 支持配置项、关系图谱、变更追踪
 */

// ==================== 配置项基础类型 ====================

/**
 * CI类型
 */
export enum CIType {
  // 硬件
  SERVER = 'server',
  NETWORK_DEVICE = 'network_device',
  STORAGE = 'storage',
  DESKTOP = 'desktop',
  LAPTOP = 'laptop',
  MOBILE = 'mobile',
  PRINTER = 'printer',
  
  // 软件
  APPLICATION = 'application',
  DATABASE = 'database',
  MIDDLEWARE = 'middleware',
  OPERATING_SYSTEM = 'operating_system',
  LICENSE = 'license',
  
  // 服务
  BUSINESS_SERVICE = 'business_service',
  TECHNICAL_SERVICE = 'technical_service',
  
  // 其他
  LOCATION = 'location',
  DEPARTMENT = 'department',
  PERSON = 'person',
  DOCUMENT = 'document',
}

/**
 * CI状态
 */
export enum CIStatus {
  PLANNING = 'planning',         // 规划中
  ORDERED = 'ordered',           // 已订购
  IN_STOCK = 'in_stock',         // 库存中
  IN_USE = 'in_use',             // 使用中
  IN_MAINTENANCE = 'in_maintenance', // 维护中
  RETIRED = 'retired',           // 已退役
  DISPOSED = 'disposed',         // 已处置
}

/**
 * 配置项 (Configuration Item)
 */
export interface ConfigurationItem {
  id: string;
  ciNumber: string;              // CI编号
  name: string;
  type: CIType;
  status: CIStatus;
  description?: string;
  
  // 基本信息
  manufacturer?: string;         // 制造商
  model?: string;                // 型号
  serialNumber?: string;         // 序列号
  assetTag?: string;             // 资产标签
  
  // 位置信息
  location?: string;
  department?: string;
  owner?: number;                // 所有者ID
  ownerName?: string;
  
  // 技术信息
  ipAddress?: string;
  hostname?: string;
  osType?: string;
  osVersion?: string;
  
  // 财务信息
  purchaseDate?: Date;
  purchaseCost?: number;
  warrantyExpiry?: Date;
  
  // 自定义属性
  attributes: Record<string, any>;
  
  // 关系统计
  relationshipCount?: number;
  dependencyCount?: number;
  
  // 元数据
  createdBy: number;
  createdByName: string;
  updatedBy?: number;
  updatedByName?: string;
  createdAt: Date;
  updatedAt: Date;
  
  // 变更追踪
  lastScanTime?: Date;
  changeCount?: number;
}

// ==================== CI关系类型 ====================

/**
 * 关系类型
 */
export enum RelationType {
  // 物理关系
  CONNECTED_TO = 'connected_to',       // 连接到
  INSTALLED_ON = 'installed_on',       // 安装在
  HOSTED_ON = 'hosted_on',             // 托管在
  RUNS_ON = 'runs_on',                 // 运行在
  CONTAINS = 'contains',               // 包含
  
  // 逻辑关系
  DEPENDS_ON = 'depends_on',           // 依赖于
  PROVIDES_TO = 'provides_to',         // 提供给
  USES = 'uses',                       // 使用
  MANAGES = 'manages',                 // 管理
  
  // 业务关系
  SUPPORTS = 'supports',               // 支持
  OWNED_BY = 'owned_by',               // 归属于
  LOCATED_IN = 'located_in',           // 位于
  MEMBER_OF = 'member_of',             // 成员
  
  // 自定义
  CUSTOM = 'custom',
}

/**
 * CI关系
 */
export interface CIRelationship {
  id: string;
  type: RelationType;
  customType?: string;             // 自定义类型名称
  
  sourceCI: string;                // 源CI ID
  sourceCIName: string;
  sourceCIType: CIType;
  
  targetCI: string;                // 目标CI ID
  targetCIName: string;
  targetCIType: CIType;
  
  // 关系属性
  attributes?: Record<string, any>;
  
  // 影响信息
  impactLevel?: 'low' | 'medium' | 'high' | 'critical';
  isActive: boolean;
  
  // 元数据
  createdBy: number;
  createdByName: string;
  createdAt: Date;
  updatedAt: Date;
}

// ==================== CI类型定义 ====================

/**
 * CI类型定义
 */
export interface CITypeDefinition {
  id: string;
  type: CIType;
  name: string;
  icon?: string;
  description?: string;
  
  // 属性定义
  attributes: CIAttributeDefinition[];
  
  // 允许的关系类型
  allowedRelationships: {
    type: RelationType;
    targetTypes: CIType[];
  }[];
  
  // 配置
  isActive: boolean;
  requiresApproval: boolean;
  
  createdAt: Date;
  updatedAt: Date;
}

/**
 * CI属性定义
 */
export interface CIAttributeDefinition {
  name: string;
  label: string;
  type: 'string' | 'number' | 'boolean' | 'date' | 'select' | 'multiselect';
  required: boolean;
  defaultValue?: any;
  options?: string[];              // 用于select类型
  validation?: {
    min?: number;
    max?: number;
    pattern?: string;
  };
}

// ==================== 关系图谱 ====================

/**
 * 图谱节点
 */
export interface GraphNode {
  id: string;
  ciId: string;
  name: string;
  type: CIType;
  status: CIStatus;
  
  // 视觉属性
  x?: number;
  y?: number;
  size?: number;
  color?: string;
  icon?: string;
  
  // 数据
  data: ConfigurationItem;
  
  // 统计
  inDegree?: number;              // 入度
  outDegree?: number;             // 出度
}

/**
 * 图谱边
 */
export interface GraphEdge {
  id: string;
  relationshipId: string;
  source: string;                 // 源节点ID
  target: string;                 // 目标节点ID
  type: RelationType;
  label?: string;
  
  // 视觉属性
  color?: string;
  width?: number;
  style?: 'solid' | 'dashed' | 'dotted';
  
  // 数据
  data: CIRelationship;
}

/**
 * 关系图谱
 */
export interface RelationshipGraph {
  nodes: GraphNode[];
  edges: GraphEdge[];
  
  // 布局信息
  layout?: 'force' | 'hierarchical' | 'circular' | 'grid';
  
  // 统计信息
  stats: {
    totalNodes: number;
    totalEdges: number;
    nodesByType: Record<CIType, number>;
    edgesByType: Record<RelationType, number>;
  };
}

// ==================== 影响分析 ====================

/**
 * 影响分析请求
 */
export interface ImpactAnalysisRequest {
  ciId: string;
  analysisType: 'upstream' | 'downstream' | 'both';
  maxDepth?: number;               // 最大深度
  includeTypes?: CIType[];         // 包含的CI类型
  excludeTypes?: CIType[];         // 排除的CI类型
}

/**
 * 影响分析结果
 */
export interface ImpactAnalysisResult {
  rootCI: ConfigurationItem;
  
  // 上游依赖（影响这个CI的）
  upstreamCIs: {
    ci: ConfigurationItem;
    path: string[];               // 依赖路径
    distance: number;             // 距离（层级）
    impactLevel: 'low' | 'medium' | 'high' | 'critical';
  }[];
  
  // 下游影响（这个CI影响的）
  downstreamCIs: {
    ci: ConfigurationItem;
    path: string[];
    distance: number;
    impactLevel: 'low' | 'medium' | 'high' | 'critical';
  }[];
  
  // 汇总
  summary: {
    totalUpstream: number;
    totalDownstream: number;
    criticalCount: number;
    highImpactCount: number;
    affectedServices: string[];
  };
  
  // 建议
  recommendations: string[];
}

// ==================== CI变更追踪 ====================

/**
 * CI变更记录
 */
export interface CIChangeRecord {
  id: string;
  ciId: string;
  ciName: string;
  changeType: 'created' | 'updated' | 'deleted' | 'relationship_added' | 'relationship_removed';
  
  // 变更内容
  changes: {
    field: string;
    oldValue?: any;
    newValue?: any;
  }[];
  
  // 关联信息
  relatedTicketId?: number;
  relatedChangeId?: number;
  
  // 元数据
  changedBy: number;
  changedByName: string;
  changedAt: Date;
  reason?: string;
}

/**
 * CI审计日志
 */
export interface CIAuditLog {
  id: string;
  ciId: string;
  action: string;
  details: Record<string, any>;
  userId: number;
  userName: string;
  timestamp: Date;
  ipAddress?: string;
}

// ==================== CI发现和扫描 ====================

/**
 * 发现规则
 */
export interface DiscoveryRule {
  id: string;
  name: string;
  type: 'network_scan' | 'agent' | 'api' | 'manual';
  enabled: boolean;
  
  // 扫描配置
  config: {
    schedule?: string;            // Cron表达式
    targets?: string[];           // 目标范围
    protocols?: string[];         // 协议
    credentials?: string[];       // 凭证
  };
  
  // 映射规则
  mapping: {
    ciType: CIType;
    attributeMapping: Record<string, string>;
  };
  
  lastRunTime?: Date;
  nextRunTime?: Date;
  createdAt: Date;
  updatedAt: Date;
}

/**
 * 扫描结果
 */
export interface DiscoveryResult {
  id: string;
  ruleId: string;
  ruleName: string;
  
  status: 'running' | 'completed' | 'failed';
  startTime: Date;
  endTime?: Date;
  
  // 结果统计
  discovered: number;             // 发现数量
  created: number;                // 新建数量
  updated: number;                // 更新数量
  unchanged: number;              // 未变化数量
  errors: number;                 // 错误数量
  
  // 详细结果
  items: {
    ciId?: string;
    action: 'create' | 'update' | 'skip' | 'error';
    data: Record<string, any>;
    error?: string;
  }[];
}

// ==================== CMDB统计 ====================

/**
 * CMDB统计
 */
export interface CMDBStats {
  totalCIs: number;
  totalRelationships: number;
  
  cisByType: Record<CIType, number>;
  cisByStatus: Record<CIStatus, number>;
  relationshipsByType: Record<RelationType, number>;
  
  trends: {
    date: string;
    created: number;
    updated: number;
    deleted: number;
  }[];
  
  topCIsByConnections: {
    ci: ConfigurationItem;
    connectionCount: number;
  }[];
  
  orphanedCIs: number;            // 孤立CI数量
  criticalCIs: number;            // 关键CI数量
}

// ==================== API请求/响应 ====================

/**
 * 创建CI请求
 */
export interface CreateCIRequest {
  name: string;
  type: CIType;
  status: CIStatus;
  description?: string;
  manufacturer?: string;
  model?: string;
  serialNumber?: string;
  assetTag?: string;
  location?: string;
  department?: string;
  owner?: number;
  attributes?: Record<string, any>;
}

/**
 * 更新CI请求
 */
export type UpdateCIRequest = Partial<CreateCIRequest>;

/**
 * 创建关系请求
 */
export interface CreateRelationshipRequest {
  type: RelationType;
  customType?: string;
  sourceCI: string;
  targetCI: string;
  attributes?: Record<string, any>;
  impactLevel?: 'low' | 'medium' | 'high' | 'critical';
}

/**
 * CI查询参数
 */
export interface CIQuery {
  type?: CIType;
  status?: CIStatus;
  owner?: number;
  location?: string;
  department?: string;
  search?: string;
  page?: number;
  pageSize?: number;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
}

/**
 * 图谱查询参数
 */
export interface GraphQuery {
  rootCI: string;
  depth?: number;                 // 展开深度
  types?: CIType[];               // 过滤CI类型
  relationshipTypes?: RelationType[]; // 过滤关系类型
  layout?: 'force' | 'hierarchical' | 'circular' | 'grid';
}

export default ConfigurationItem;

