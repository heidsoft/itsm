/**
 * 变更分类系统类型定义
 * 基于ITIL标准的变更分类和评估
 */

// ==================== 变更类型 ====================

/**
 * 变更类型
 */
export enum ChangeType {
  STANDARD = 'standard',       // 标准变更
  NORMAL = 'normal',           // 普通变更
  EMERGENCY = 'emergency',     // 紧急变更
  MAJOR = 'major',             // 重大变更
}

/**
 * 变更分类
 */
export interface ChangeClassification {
  id: string;
  name: string;
  code: string;
  type: ChangeType;
  description?: string;
  riskLevel: 'low' | 'medium' | 'high' | 'critical';
  approvalRequired: boolean;
  cabRequired: boolean;          // Change Advisory Board
  testingRequired: boolean;
  backoutPlanRequired: boolean;
  businessJustificationRequired: boolean;
  standardDuration?: number;     // 标准工期（小时）
  cooldownPeriod?: number;       // 冷却期（小时）
  isActive: boolean;
  createdAt: Date;
  updatedAt: Date;
}

// ==================== 变更风险评估 ====================

/**
 * 风险因素
 */
export interface RiskFactor {
  category: 'technical' | 'business' | 'compliance' | 'operational';
  factor: string;
  weight: number;                // 权重（0-100）
  value: number;                 // 评分（0-100）
  description: string;
}

/**
 * 风险评估结果
 */
export interface RiskAssessment {
  overallRisk: 'low' | 'medium' | 'high' | 'critical';
  riskScore: number;             // 总风险分数（0-100）
  factors: RiskFactor[];
  recommendations: string[];
  assessedBy?: number;
  assessedByName?: string;
  assessedAt: Date;
}

// ==================== 变更影响分析 ====================

/**
 * 影响范围
 */
export interface ImpactScope {
  services: string[];            // 受影响服务
  cis: string[];                 // 受影响配置项
  users: number;                 // 受影响用户数
  departments: string[];         // 受影响部门
  locations: string[];           // 受影响位置
  estimatedDowntime?: number;    // 预计停机时间（分钟）
}

/**
 * 影响分析
 */
export interface ImpactAnalysis {
  scope: ImpactScope;
  businessImpact: 'none' | 'low' | 'medium' | 'high' | 'critical';
  financialImpact?: number;
  reputationImpact?: 'none' | 'low' | 'medium' | 'high';
  complianceImpact?: boolean;
  analysisNotes?: string;
  analysedBy?: number;
  analysedByName?: string;
  analysedAt: Date;
}

// ==================== 变更分类规则 ====================

/**
 * 分类规则条件
 */
export interface ClassificationRuleCondition {
  field: string;
  operator: 'equals' | 'contains' | 'greater_than' | 'less_than' | 'in' | 'between';
  value: any;
}

/**
 * 分类规则
 */
export interface ClassificationRule {
  id: string;
  name: string;
  priority: number;              // 规则优先级
  conditions: ClassificationRuleCondition[];
  resultClassification: string;  // 分类ID
  enabled: boolean;
  createdAt: Date;
  updatedAt: Date;
}

// ==================== 变更分类建议 ====================

/**
 * 分类建议
 */
export interface ClassificationSuggestion {
  changeId: number;
  currentClassification?: ChangeClassification;
  suggestedClassification: ChangeClassification;
  confidence: number;            // 置信度（0-100）
  reasoning: string;
  basedOn: 'rules' | 'similarity' | 'risk' | 'impact' | 'historical';
  riskAssessment?: RiskAssessment;
  impactAnalysis?: ImpactAnalysis;
}

// ==================== 变更模板 ====================

/**
 * 变更模板
 */
export interface ChangeTemplate {
  id: string;
  name: string;
  classification: ChangeClassification;
  description?: string;
  steps: ChangeTemplateStep[];
  requiredFields: string[];
  estimatedDuration?: number;
  riskMitigation?: string;
  backoutPlan?: string;
  testingProcedure?: string;
  isActive: boolean;
  usageCount: number;
  createdAt: Date;
  updatedAt: Date;
}

/**
 * 变更模板步骤
 */
export interface ChangeTemplateStep {
  id: string;
  order: number;
  title: string;
  description?: string;
  expectedDuration?: number;     // 预计时长（分钟）
  assignedRole?: string;
  verificationRequired: boolean;
}

// ==================== 变更审批矩阵 ====================

/**
 * 审批级别
 */
export interface ApprovalLevel {
  level: number;
  name: string;
  approvers: number[];           // 审批人ID
  requiredApprovals: number;     // 需要的审批数量
  timeoutHours?: number;         // 超时时间
  escalationTo?: number;         // 升级到（审批人ID）
}

/**
 * 审批矩阵
 */
export interface ApprovalMatrix {
  classificationId: string;
  levels: ApprovalLevel[];
  parallelApproval: boolean;     // 是否并行审批
  autoApproveConditions?: ClassificationRuleCondition[];
}

// ==================== 统计和分析 ====================

/**
 * 分类统计
 */
export interface ClassificationStats {
  classification: ChangeClassification;
  count: number;
  successRate: number;           // 成功率（%）
  avgDuration: number;           // 平均时长（小时）
  avgRiskScore: number;
  failureReasons?: Array<{
    reason: string;
    count: number;
  }>;
}

/**
 * 分类分析
 */
export interface ClassificationAnalysis {
  period: {
    start: Date;
    end: Date;
  };
  byType: Record<ChangeType, ClassificationStats[]>;
  trends: {
    date: string;
    byClassification: Record<string, number>;
  }[];
  insights: {
    type: 'warning' | 'info' | 'success';
    message: string;
  }[];
}

// ==================== API 请求/响应 ====================

/**
 * 创建分类请求
 */
export interface CreateClassificationRequest {
  name: string;
  code: string;
  type: ChangeType;
  description?: string;
  riskLevel: 'low' | 'medium' | 'high' | 'critical';
  approvalRequired: boolean;
  cabRequired?: boolean;
  testingRequired?: boolean;
  backoutPlanRequired?: boolean;
  businessJustificationRequired?: boolean;
  standardDuration?: number;
  cooldownPeriod?: number;
}

/**
 * 更新分类请求
 */
export type UpdateClassificationRequest = Partial<CreateClassificationRequest>;

/**
 * 风险评估请求
 */
export interface AssessRiskRequest {
  changeId: number;
  factors: Omit<RiskFactor, 'description'>[];
}

/**
 * 影响分析请求
 */
export interface AnalyzeImpactRequest {
  changeId: number;
  scope: ImpactScope;
  businessImpact: 'none' | 'low' | 'medium' | 'high' | 'critical';
  financialImpact?: number;
  analysisNotes?: string;
}

/**
 * 分类查询
 */
export interface ClassificationQuery {
  type?: ChangeType;
  riskLevel?: string;
  active?: boolean;
  search?: string;
  page?: number;
  pageSize?: number;
}

/**
 * 分类历史
 */
export interface ClassificationHistory {
  id: string;
  changeId: number;
  oldClassification?: ChangeClassification;
  newClassification: ChangeClassification;
  reason: string;
  changedBy: number;
  changedByName: string;
  changedAt: Date;
}

export default ChangeClassification;

