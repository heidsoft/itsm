/**
 * 事件优先级矩阵类型定义
 * 基于影响度和紧急度自动计算优先级
 */

import type { TicketPriority } from './ticket';

// ==================== 影响度 ====================

/**
 * 影响度级别
 */
export enum ImpactLevel {
  LOW = 'low',           // 低 - 影响个人或小团队
  MEDIUM = 'medium',     // 中 - 影响部门
  HIGH = 'high',         // 高 - 影响多个部门
  CRITICAL = 'critical', // 严重 - 影响整个组织
}

/**
 * 影响度因素
 */
export interface ImpactFactors {
  affectedUsers: number;              // 受影响用户数
  affectedDepartments: string[];      // 受影响部门
  businessImpact: 'none' | 'low' | 'medium' | 'high' | 'critical'; // 业务影响
  financialImpact?: number;           // 财务影响（金额）
  reputationImpact?: 'none' | 'low' | 'medium' | 'high'; // 声誉影响
  dataLoss?: boolean;                 // 是否涉及数据丢失
  securityImpact?: boolean;           // 是否涉及安全
}

/**
 * 影响度计算结果
 */
export interface ImpactAssessment {
  level: ImpactLevel;
  score: number;                      // 影响度评分（0-100）
  factors: ImpactFactors;
  description: string;
  calculatedAt: Date;
}

// ==================== 紧急度 ====================

/**
 * 紧急度级别
 */
export enum UrgencyLevel {
  LOW = 'low',           // 低 - 可以延后处理
  MEDIUM = 'medium',     // 中 - 需要及时处理
  HIGH = 'high',         // 高 - 需要尽快处理
  CRITICAL = 'critical', // 严重 - 需要立即处理
}

/**
 * 紧急度因素
 */
export interface UrgencyFactors {
  isBlocking: boolean;                // 是否阻塞业务
  deadline?: Date;                    // 截止时间
  timeToDeadline?: number;            // 距离截止时间（小时）
  escalationLevel: number;            // 升级级别（0-5）
  vipUser?: boolean;                  // 是否VIP用户
  peakHours?: boolean;                // 是否高峰时段
  slaBreachRisk?: number;             // SLA违反风险（0-100）
}

/**
 * 紧急度计算结果
 */
export interface UrgencyAssessment {
  level: UrgencyLevel;
  score: number;                      // 紧急度评分（0-100）
  factors: UrgencyFactors;
  description: string;
  calculatedAt: Date;
}

// ==================== 优先级矩阵 ====================

/**
 * 优先级矩阵配置
 */
export interface PriorityMatrixConfig {
  id: string;
  name: string;
  description?: string;
  matrixType: '2x2' | '3x3' | '4x4';  // 矩阵类型
  mappingRules: PriorityMappingRule[];
  isActive: boolean;
  createdAt: Date;
  updatedAt: Date;
}

/**
 * 优先级映射规则
 */
export interface PriorityMappingRule {
  impactLevel: ImpactLevel;
  urgencyLevel: UrgencyLevel;
  resultPriority: TicketPriority;
  slaTarget?: number;                 // SLA目标（小时）
  autoAssign?: boolean;               // 是否自动分配
  notificationRules?: string[];       // 通知规则
}

/**
 * 优先级矩阵单元格
 */
export interface MatrixCell {
  impactLevel: ImpactLevel;
  urgencyLevel: UrgencyLevel;
  priority: TicketPriority;
  color: string;
  label: string;
  count?: number;                     // 当前此优先级的工单数
}

/**
 * 优先级矩阵数据
 */
export interface PriorityMatrixData {
  config: PriorityMatrixConfig;
  cells: MatrixCell[][];
  statistics: {
    totalTickets: number;
    byPriority: Record<TicketPriority, number>;
    byImpact: Record<ImpactLevel, number>;
    byUrgency: Record<UrgencyLevel, number>;
  };
}

// ==================== 优先级计算 ====================

/**
 * 优先级计算请求
 */
export interface PriorityCalculationRequest {
  ticketId?: number;
  impactFactors: ImpactFactors;
  urgencyFactors: UrgencyFactors;
  matrixConfigId?: string;
}

/**
 * 优先级计算结果
 */
export interface PriorityCalculationResult {
  suggestedPriority: TicketPriority;
  impactAssessment: ImpactAssessment;
  urgencyAssessment: UrgencyAssessment;
  confidence: number;                 // 置信度（0-100）
  reasoning: string;
  alternatives?: Array<{
    priority: TicketPriority;
    reason: string;
    confidence: number;
  }>;
}

// ==================== 自动优先级建议 ====================

/**
 * 优先级建议
 */
export interface PrioritySuggestion {
  ticketId: number;
  currentPriority?: TicketPriority;
  suggestedPriority: TicketPriority;
  reason: string;
  confidence: number;
  basedOn: 'historical' | 'keywords' | 'similarity' | 'rules' | 'ml';
  similarTickets?: Array<{
    ticketId: number;
    ticketNumber: string;
    priority: TicketPriority;
    similarity: number;
  }>;
}

/**
 * 批量优先级建议
 */
export interface BatchPrioritySuggestions {
  suggestions: PrioritySuggestion[];
  total: number;
  highConfidence: number;             // 高置信度建议数量
  needsReview: number;                // 需要审查的数量
}

// ==================== 优先级规则 ====================

/**
 * 优先级规则类型
 */
export enum PriorityRuleType {
  KEYWORD_MATCH = 'keyword_match',      // 关键词匹配
  USER_BASED = 'user_based',            // 基于用户
  TIME_BASED = 'time_based',            // 基于时间
  CATEGORY_BASED = 'category_based',    // 基于分类
  SLA_BASED = 'sla_based',              // 基于SLA
  CUSTOM = 'custom',                    // 自定义
}

/**
 * 优先级规则
 */
export interface PriorityRule {
  id: string;
  name: string;
  type: PriorityRuleType;
  enabled: boolean;
  priority: number;                   // 规则优先级（数字越小越优先）
  conditions: PriorityRuleCondition[];
  actions: PriorityRuleAction[];
  createdAt: Date;
  updatedAt: Date;
}

/**
 * 规则条件
 */
export interface PriorityRuleCondition {
  field: string;
  operator: 'equals' | 'contains' | 'greater_than' | 'less_than' | 'in' | 'regex';
  value: any;
  caseSensitive?: boolean;
}

/**
 * 规则动作
 */
export interface PriorityRuleAction {
  type: 'set_priority' | 'set_impact' | 'set_urgency' | 'notify' | 'assign';
  params: Record<string, any>;
}

/**
 * 规则执行结果
 */
export interface RuleExecutionResult {
  ruleId: string;
  ruleName: string;
  matched: boolean;
  appliedActions: string[];
  executedAt: Date;
}

// ==================== 优先级历史 ====================

/**
 * 优先级变更历史
 */
export interface PriorityChangeHistory {
  id: string;
  ticketId: number;
  oldPriority?: TicketPriority;
  newPriority: TicketPriority;
  changeReason: 'manual' | 'automatic' | 'rule_based' | 'suggestion_accepted';
  changedBy: number;
  changedByName: string;
  comment?: string;
  impactAssessment?: ImpactAssessment;
  urgencyAssessment?: UrgencyAssessment;
  appliedRules?: RuleExecutionResult[];
  changedAt: Date;
}

// ==================== 优先级分析 ====================

/**
 * 优先级分布分析
 */
export interface PriorityDistributionAnalysis {
  period: {
    start: Date;
    end: Date;
  };
  distribution: {
    priority: TicketPriority;
    count: number;
    percentage: number;
    avgResolutionTime: number;
    slaCompliance: number;
  }[];
  trends: {
    date: string;
    byPriority: Record<TicketPriority, number>;
  }[];
  insights: {
    type: 'warning' | 'info' | 'success';
    message: string;
    recommendation?: string;
  }[];
}

/**
 * 优先级准确性分析
 */
export interface PriorityAccuracyAnalysis {
  totalTickets: number;
  accurateAssignments: number;
  accuracyRate: number;               // 准确率（%）
  commonMisclassifications: Array<{
    assignedPriority: TicketPriority;
    shouldBePriority: TicketPriority;
    count: number;
    examples: number[];               // 工单ID示例
  }>;
  improvementSuggestions: string[];
}

// ==================== API请求/响应 ====================

/**
 * 创建优先级矩阵配置请求
 */
export interface CreateMatrixConfigRequest {
  name: string;
  description?: string;
  matrixType: '2x2' | '3x3' | '4x4';
  mappingRules: Omit<PriorityMappingRule, 'id'>[];
}

/**
 * 更新优先级矩阵配置请求
 */
export interface UpdateMatrixConfigRequest {
  name?: string;
  description?: string;
  mappingRules?: PriorityMappingRule[];
  isActive?: boolean;
}

/**
 * 优先级规则查询
 */
export interface PriorityRuleQuery {
  type?: PriorityRuleType;
  enabled?: boolean;
  search?: string;
  page?: number;
  pageSize?: number;
}

/**
 * 优先级分析查询
 */
export interface PriorityAnalysisQuery {
  startDate: string;
  endDate: string;
  groupBy?: 'day' | 'week' | 'month';
  priorities?: TicketPriority[];
}

// ==================== 导出所有类型 ====================

export default PriorityMatrixConfig;

