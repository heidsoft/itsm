/**
 * 智能分配机制 - 符合ITIL 4.0标准
 *
 * 核心功能：
 * 1. 基于技能的智能分配
 * 2. 工作负载均衡
 * 3. 地理位置分配
 * 4. 自动分配规则
 * 5. 分配历史分析
 * 6. 分配效果评估
 */

import type { PaginatedResponse } from '@/types/api';
import { httpClient } from '@/lib/api/http-client';

// 分配策略枚举
export enum AssignmentStrategy {
  SKILL_BASED = 'skill_based', // 基于技能
  WORKLOAD_BALANCED = 'workload_balanced', // 工作负载均衡
  GEOGRAPHIC = 'geographic', // 地理位置
  ROUND_ROBIN = 'round_robin', // 轮询分配
  PRIORITY_BASED = 'priority_based', // 基于优先级
  EXPERIENCE_BASED = 'experience_based', // 基于经验
}

// 分配状态枚举
export enum AssignmentStatus {
  PENDING = 'pending', // 待分配
  ASSIGNED = 'assigned', // 已分配
  ACCEPTED = 'accepted', // 已接受
  REJECTED = 'rejected', // 已拒绝
  TRANSFERRED = 'transferred', // 已转移
  ESCALATED = 'escalated', // 已升级
}

// 技能等级枚举
export enum SkillLevel {
  BEGINNER = 'beginner', // 初级
  INTERMEDIATE = 'intermediate', // 中级
  ADVANCED = 'advanced', // 高级
  EXPERT = 'expert', // 专家
}

// 用户技能
export interface UserSkill {
  id: number;
  userId: number;
  skillId: number;
  skillName: string;
  skillCategory: string;
  level: SkillLevel;
  experience: number; // 经验值（0-100）
  certificationDate?: string;
  lastUsed?: string;
  successRate: number; // 成功率（0-1）
  createdAt: string;
  updatedAt: string;
}

// 用户工作负载
export interface UserWorkload {
  userId: number;
  userName: string;
  activeTickets: number;
  pendingTickets: number;
  completedToday: number;
  averageResolutionTime: number; // 平均解决时间（分钟）
  currentUtilization: number; // 当前利用率（0-1）
  maxCapacity: number; // 最大容量
  availableCapacity: number; // 可用容量
  lastUpdated: string;
}

// 分配规则
export interface AssignmentRule {
  id: number;
  name: string;
  description?: string;
  strategy: AssignmentStrategy;
  priority: number; // 规则优先级
  conditions: AssignmentCondition[];
  actions: AssignmentAction[];
  isActive: boolean;
  applicableTo: {
    ticketTypes: string[];
    categories: string[];
    priorities: string[];
    departments: string[];
  };
  createdAt: string;
  updatedAt: string;
}

// 分配条件
export interface AssignmentCondition {
  id: number;
  field: string; // 字段名
  operator: 'equals' | 'not_equals' | 'contains' | 'greater_than' | 'less_than' | 'in' | 'not_in';
  value: unknown;
  logic: 'and' | 'or'; // 逻辑连接符
}

// 分配动作
export interface AssignmentAction {
  id: number;
  type: 'assign' | 'escalate' | 'notify' | 'schedule';
  target: string; // 目标用户/组/角色
  parameters?: Record<string, any>;
  delay?: number; // 延迟时间（分钟）
}

// 分配历史
export interface AssignmentHistory {
  id: number;
  ticketId: number;
  ticketNumber?: string;
  assignedTo: number;
  assignedBy: number;
  assignedAt: string;
  strategy: AssignmentStrategy;
  reason: string;
  status: AssignmentStatus;
  acceptedAt?: string;
  rejectedAt?: string;
  rejectionReason?: string;
  transferredAt?: string;
  transferredTo?: number;
  transferredReason?: string;
  resolutionTime?: number; // 解决时间（分钟）
  satisfaction?: number; // 满意度评分
  createdAt: string;
  updatedAt: string;
}

// 分配建议
export interface AssignmentSuggestion {
  userId: number;
  userName: string;
  score: number; // 匹配分数（0-100）
  reasons: string[]; // 推荐原因
  skills: UserSkill[]; // 匹配的技能
  workload: UserWorkload; // 工作负载信息
  estimatedResolutionTime: number; // 预估解决时间
  successProbability: number; // 成功概率
}

// 分配统计
export interface AssignmentStats {
  totalAssignments: number;
  successfulAssignments: number;
  failedAssignments: number;
  averageResolutionTime: number;
  averageSatisfaction: number;
  strategyPerformance: Array<{
    strategy: AssignmentStrategy;
    count: number;
    successRate: number;
    averageResolutionTime: number;
    averageSatisfaction: number;
  }>;
  userPerformance: Array<{
    userId: number;
    userName: string;
    assignments: number;
    successRate: number;
    averageResolutionTime: number;
    averageSatisfaction: number;
  }>;
}

// 分配请求
export interface AssignmentRequest {
  ticketId: number;
  ticketType: string;
  category: string;
  priority: string;
  description: string;
  requiredSkills?: string[];
  preferredAssignee?: number;
  excludeUsers?: number[];
  maxCandidates?: number;
  strategy?: AssignmentStrategy;
}

// 分配响应
export interface AssignmentResponse {
  success: boolean;
  assignedTo?: number;
  suggestions: AssignmentSuggestion[];
  reason: string;
  estimatedResolutionTime?: number;
}

// 智能分配服务类
// 注意：后端路由为 /api/v1/tickets/assignment-rules
const ASSIGNMENT_API_BASE = '/api/v1/tickets';

export class SmartAssignmentService {
  /**
   * 获取分配建议 - 调用 /api/v1/tickets/assign-recommendations/:ticketId
   * 注意: 此接口用于获取工单的智能分配推荐
   * 规则测试接口 /assignment-rules/test 仅用于带 ruleId 的规则测试
   */
  async getAssignmentSuggestions(request: AssignmentRequest): Promise<AssignmentSuggestion[]> {
    const response = await httpClient.post<{ suggestions: AssignmentSuggestion[] }>(
      `/api/v1/tickets/assign-recommendations/${request.ticketId}`,
      request
    );
    return response?.suggestions || [];
  }

  /**
   * 测试分配规则 - 调用 /api/v1/tickets/assignment-rules/test
   * 此接口仅用于规则测试场景，需要传入 ruleId
   */
  async testAssignmentRule(request: AssignmentRequest & { ruleId: number }): Promise<AssignmentSuggestion[]> {
    const response = await httpClient.post<{ suggestions: AssignmentSuggestion[] }>(
      `/api/v1/tickets/assignment-rules/test`,
      request
    );
    return response?.suggestions || [];
  }

  // 执行智能分配
  async assignTicket(request: AssignmentRequest): Promise<AssignmentResponse> {
    // 后端通过 assignment-rules 提供智能分配能力
    // 这里返回建议而非直接分配，因为分配通过工单接口完成
    const suggestions = await this.getAssignmentSuggestions(request);
    return {
      success: suggestions.length > 0,
      suggestions,
      reason: suggestions.length > 0 ? '智能匹配完成' : '未找到匹配的分配建议',
    };
  }

  // 获取用户技能
  async getUserSkills(userId: number): Promise<UserSkill[]> {
    // 用户技能接口暂无后端实现，返回空数组
    return [];
  }

  // 更新用户技能
  async updateUserSkill(
    userId: number,
    skillId: number,
    data: Partial<UserSkill>
  ): Promise<UserSkill> {
    // 用户技能接口暂无后端实现
    throw new Error('User skills API not implemented');
  }

  // 获取用户工作负载
  async getUserWorkload(userId: number): Promise<UserWorkload> {
    // 工作负载接口暂无后端实现
    throw new Error('User workload API not implemented');
  }

  // 获取所有用户工作负载
  async getAllUserWorkloads(): Promise<UserWorkload[]> {
    // 工作负载接口暂无后端实现
    return [];
  }

  // 分配规则管理 - 使用后端实际路由 /api/v1/tickets/assignment-rules
  async getAssignmentRules(): Promise<AssignmentRule[]> {
    const response = await httpClient.get<AssignmentRule[]>(`${ASSIGNMENT_API_BASE}/assignment-rules`);
    return response || [];
  }

  async createAssignmentRule(
    data: Omit<AssignmentRule, 'id' | 'createdAt' | 'updatedAt'>
  ): Promise<AssignmentRule> {
    return httpClient.post<AssignmentRule>(`${ASSIGNMENT_API_BASE}/assignment-rules`, data);
  }

  async updateAssignmentRule(id: number, data: Partial<AssignmentRule>): Promise<AssignmentRule> {
    return httpClient.put<AssignmentRule>(`${ASSIGNMENT_API_BASE}/assignment-rules/${id}`, data);
  }

  async deleteAssignmentRule(id: number): Promise<void> {
    return httpClient.delete<void>(`${ASSIGNMENT_API_BASE}/assignment-rules/${id}`);
  }

  // 分配历史
  async getAssignmentHistory(params?: {
    page?: number;
    pageSize?: number;
    assignee?: number;
    ticketId?: number;
  }): Promise<PaginatedResponse<AssignmentHistory>> {
    // 分配历史暂无独立接口，通过工单接口获取
    throw new Error('Assignment history API not implemented');
  }

  // 分配统计
  async getAssignmentStats(params?: {
    dateRange?: string;
    assignee?: number;
    team?: string;
  }): Promise<AssignmentStats> {
    // 分配统计暂无独立接口
    throw new Error('Assignment stats API not implemented');
  }

  // 技能匹配算法
  calculateSkillMatch(requiredSkills: string[], userSkills: UserSkill[]): number {
    if (!requiredSkills || requiredSkills.length === 0) return 100;

    let totalScore = 0;
    let matchedSkills = 0;

    for (const requiredSkill of requiredSkills) {
      const userSkill = userSkills.find(
        skill =>
          skill.skillName.toLowerCase().includes(requiredSkill.toLowerCase()) ||
          skill.skillCategory.toLowerCase().includes(requiredSkill.toLowerCase())
      );

      if (userSkill) {
        matchedSkills++;
        // 根据技能等级和经验计算分数
        const levelScore = this.getSkillLevelScore(userSkill.level);
        const experienceScore = userSkill.experience / 100;
        const successRateScore = userSkill.successRate;

        totalScore += levelScore * 0.4 + experienceScore * 0.3 + successRateScore * 0.3;
      }
    }

    return matchedSkills > 0
      ? (totalScore / matchedSkills) * (matchedSkills / requiredSkills.length) * 100
      : 0;
  }

  // 获取技能等级分数
  private getSkillLevelScore(level: SkillLevel): number {
    switch (level) {
      case SkillLevel.BEGINNER:
        return 0.3;
      case SkillLevel.INTERMEDIATE:
        return 0.6;
      case SkillLevel.ADVANCED:
        return 0.8;
      case SkillLevel.EXPERT:
        return 1.0;
      default:
        return 0;
    }
  }

  // 工作负载均衡算法
  calculateWorkloadScore(userWorkload: UserWorkload): number {
    const utilizationScore = 1 - userWorkload.currentUtilization; // 利用率越低分数越高
    const capacityScore = userWorkload.availableCapacity / userWorkload.maxCapacity;
    const performanceScore = 1 - userWorkload.averageResolutionTime / 1440; // 假设最大解决时间为24小时

    return (utilizationScore * 0.4 + capacityScore * 0.3 + performanceScore * 0.3) * 100;
  }

  // 地理位置匹配算法
  calculateGeographicScore(userLocation: string, ticketLocation: string): number {
    if (!ticketLocation) return 100; // 如果没有位置要求，返回满分

    // 简单的位置匹配逻辑，实际应用中可以使用更复杂的地理位置API
    if (userLocation === ticketLocation) return 100;
    if (userLocation.includes(ticketLocation) || ticketLocation.includes(userLocation)) return 80;

    return 50; // 默认分数
  }

  // 综合评分算法
  calculateOverallScore(
    skillScore: number,
    workloadScore: number,
    geographicScore: number,
    strategy: AssignmentStrategy
  ): number {
    let weights = { skill: 0.4, workload: 0.3, geographic: 0.3 };

    // 根据策略调整权重
    switch (strategy) {
      case AssignmentStrategy.SKILL_BASED:
        weights = { skill: 0.6, workload: 0.2, geographic: 0.2 };
        break;
      case AssignmentStrategy.WORKLOAD_BALANCED:
        weights = { skill: 0.2, workload: 0.6, geographic: 0.2 };
        break;
      case AssignmentStrategy.GEOGRAPHIC:
        weights = { skill: 0.2, workload: 0.2, geographic: 0.6 };
        break;
    }

    return (
      skillScore * weights.skill +
      workloadScore * weights.workload +
      geographicScore * weights.geographic
    );
  }

  // 预估解决时间
  estimateResolutionTime(
    userSkills: UserSkill[],
    userWorkload: UserWorkload,
    ticketComplexity: 'low' | 'medium' | 'high'
  ): number {
    const baseTime = userWorkload.averageResolutionTime;
    const skillFactor = this.calculateSkillMatch([], userSkills) / 100;
    const workloadFactor = userWorkload.currentUtilization;

    let complexityFactor = 1;
    switch (ticketComplexity) {
      case 'low':
        complexityFactor = 0.7;
        break;
      case 'medium':
        complexityFactor = 1.0;
        break;
      case 'high':
        complexityFactor = 1.5;
        break;
    }

    return Math.round(baseTime * complexityFactor * (1 + workloadFactor) * (1 - skillFactor * 0.3));
  }

  // 成功概率计算
  calculateSuccessProbability(
    userSkills: UserSkill[],
    userWorkload: UserWorkload,
    ticketComplexity: 'low' | 'medium' | 'high'
  ): number {
    const skillMatch = this.calculateSkillMatch([], userSkills) / 100;
    const workloadFactor = 1 - userWorkload.currentUtilization;
    const performanceFactor = userWorkload.completedToday > 0 ? 0.8 : 0.6;

    let complexityFactor = 1;
    switch (ticketComplexity) {
      case 'low':
        complexityFactor = 1.2;
        break;
      case 'medium':
        complexityFactor = 1.0;
        break;
      case 'high':
        complexityFactor = 0.8;
        break;
    }

    return Math.min(1, skillMatch * workloadFactor * performanceFactor * complexityFactor);
  }
}

// 导出单例实例
export const smartAssignmentService = new SmartAssignmentService();
export default SmartAssignmentService;
