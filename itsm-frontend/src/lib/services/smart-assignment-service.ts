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

import { ApiResponse, PaginatedResponse } from '@/types/api';

// 分配策略枚举
export enum AssignmentStrategy {
  SKILL_BASED = 'skill_based',           // 基于技能
  WORKLOAD_BALANCED = 'workload_balanced', // 工作负载均衡
  GEOGRAPHIC = 'geographic',             // 地理位置
  ROUND_ROBIN = 'round_robin',           // 轮询分配
  PRIORITY_BASED = 'priority_based',     // 基于优先级
  EXPERIENCE_BASED = 'experience_based', // 基于经验
}

// 分配状态枚举
export enum AssignmentStatus {
  PENDING = 'pending',       // 待分配
  ASSIGNED = 'assigned',     // 已分配
  ACCEPTED = 'accepted',     // 已接受
  REJECTED = 'rejected',     // 已拒绝
  TRANSFERRED = 'transferred', // 已转移
  ESCALATED = 'escalated',   // 已升级
}

// 技能等级枚举
export enum SkillLevel {
  BEGINNER = 'beginner',     // 初级
  INTERMEDIATE = 'intermediate', // 中级
  ADVANCED = 'advanced',     // 高级
  EXPERT = 'expert',         // 专家
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
  value: any;
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
export class SmartAssignmentService {
  private readonly baseUrl = '/api/v1/assignment';

  // 获取分配建议
  async getAssignmentSuggestions(request: AssignmentRequest): Promise<AssignmentSuggestion[]> {
    const response = await fetch(`${this.baseUrl}/suggestions`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request),
    });
    if (!response.ok) throw new Error('Failed to get assignment suggestions');
    const result = await response.json();
    return result.suggestions;
  }

  // 执行智能分配
  async assignTicket(request: AssignmentRequest): Promise<AssignmentResponse> {
    const response = await fetch(`${this.baseUrl}/assign`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request),
    });
    if (!response.ok) throw new Error('Failed to assign ticket');
    return response.json();
  }

  // 获取用户技能
  async getUserSkills(userId: number): Promise<UserSkill[]> {
    const response = await fetch(`${this.baseUrl}/users/${userId}/skills`);
    if (!response.ok) throw new Error('Failed to get user skills');
    return response.json();
  }

  // 更新用户技能
  async updateUserSkill(userId: number, skillId: number, data: Partial<UserSkill>): Promise<UserSkill> {
    const response = await fetch(`${this.baseUrl}/users/${userId}/skills/${skillId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (!response.ok) throw new Error('Failed to update user skill');
    return response.json();
  }

  // 获取用户工作负载
  async getUserWorkload(userId: number): Promise<UserWorkload> {
    const response = await fetch(`${this.baseUrl}/users/${userId}/workload`);
    if (!response.ok) throw new Error('Failed to get user workload');
    return response.json();
  }

  // 获取所有用户工作负载
  async getAllUserWorkloads(): Promise<UserWorkload[]> {
    const response = await fetch(`${this.baseUrl}/workloads`);
    if (!response.ok) throw new Error('Failed to get user workloads');
    return response.json();
  }

  // 分配规则管理
  async getAssignmentRules(): Promise<AssignmentRule[]> {
    const response = await fetch(`${this.baseUrl}/rules`);
    if (!response.ok) throw new Error('Failed to get assignment rules');
    return response.json();
  }

  async createAssignmentRule(data: Omit<AssignmentRule, 'id' | 'createdAt' | 'updatedAt'>): Promise<AssignmentRule> {
    const response = await fetch(`${this.baseUrl}/rules`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (!response.ok) throw new Error('Failed to create assignment rule');
    return response.json();
  }

  async updateAssignmentRule(id: number, data: Partial<AssignmentRule>): Promise<AssignmentRule> {
    const response = await fetch(`${this.baseUrl}/rules/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (!response.ok) throw new Error('Failed to update assignment rule');
    return response.json();
  }

  async deleteAssignmentRule(id: number): Promise<void> {
    const response = await fetch(`${this.baseUrl}/rules/${id}`, {
      method: 'DELETE',
    });
    if (!response.ok) throw new Error('Failed to delete assignment rule');
  }

  // 分配历史
  async getAssignmentHistory(params?: any): Promise<PaginatedResponse<AssignmentHistory>> {
    const response = await fetch(`${this.baseUrl}/history?${new URLSearchParams(params)}`);
    if (!response.ok) throw new Error('Failed to get assignment history');
    return response.json();
  }

  // 分配统计
  async getAssignmentStats(params?: any): Promise<AssignmentStats> {
    const response = await fetch(`${this.baseUrl}/stats?${new URLSearchParams(params)}`);
    if (!response.ok) throw new Error('Failed to get assignment stats');
    return response.json();
  }

  // 技能匹配算法
  calculateSkillMatch(requiredSkills: string[], userSkills: UserSkill[]): number {
    if (!requiredSkills || requiredSkills.length === 0) return 100;
    
    let totalScore = 0;
    let matchedSkills = 0;

    for (const requiredSkill of requiredSkills) {
      const userSkill = userSkills.find(skill => 
        skill.skillName.toLowerCase().includes(requiredSkill.toLowerCase()) ||
        skill.skillCategory.toLowerCase().includes(requiredSkill.toLowerCase())
      );

      if (userSkill) {
        matchedSkills++;
        // 根据技能等级和经验计算分数
        const levelScore = this.getSkillLevelScore(userSkill.level);
        const experienceScore = userSkill.experience / 100;
        const successRateScore = userSkill.successRate;
        
        totalScore += (levelScore * 0.4 + experienceScore * 0.3 + successRateScore * 0.3);
      }
    }

    return matchedSkills > 0 ? (totalScore / matchedSkills) * (matchedSkills / requiredSkills.length) * 100 : 0;
  }

  // 获取技能等级分数
  private getSkillLevelScore(level: SkillLevel): number {
    switch (level) {
      case SkillLevel.BEGINNER: return 0.3;
      case SkillLevel.INTERMEDIATE: return 0.6;
      case SkillLevel.ADVANCED: return 0.8;
      case SkillLevel.EXPERT: return 1.0;
      default: return 0;
    }
  }

  // 工作负载均衡算法
  calculateWorkloadScore(userWorkload: UserWorkload): number {
    const utilizationScore = 1 - userWorkload.currentUtilization; // 利用率越低分数越高
    const capacityScore = userWorkload.availableCapacity / userWorkload.maxCapacity;
    const performanceScore = 1 - (userWorkload.averageResolutionTime / 1440); // 假设最大解决时间为24小时

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
      case 'low': complexityFactor = 0.7; break;
      case 'medium': complexityFactor = 1.0; break;
      case 'high': complexityFactor = 1.5; break;
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
      case 'low': complexityFactor = 1.2; break;
      case 'medium': complexityFactor = 1.0; break;
      case 'high': complexityFactor = 0.8; break;
    }

    return Math.min(1, skillMatch * workloadFactor * performanceFactor * complexityFactor);
  }
}

// 导出单例实例
export const smartAssignmentService = new SmartAssignmentService();
export default SmartAssignmentService;
