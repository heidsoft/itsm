/**
 * SLA 升级矩阵 API
 *
 * 对应后端 service/escalation_matrix.go（service 层）和
 * controller/escalation_matrix_controller.go（HTTP 层）。
 *
 * - 后端已注册路由 /api/v1/escalation-matrices（EscalationMatrixController）
 * - 矩阵被 EscalationMatrixService.GetMatrix() 在以下场景隐式使用：
 *   - SLA 告警升级
 *   - 工单超时升级
 *   - BPMN 任务超时
 * - 这里提供前端 HTTP 客户端 + 默认值兜底，便于 UI 展示当前生效矩阵。
 */

import { httpClient } from '@/lib/api/http-client';

// 单级升级策略
export interface EscalationLevel {
  level: number; // 1, 2, 3
  thresholdMinutes: number; // 触发阈值（分钟）
  targetType: string; // user | role | group
  targetIds: number[];
  notifyChannels: string[]; // email | sms | in_app | webhook
}

// 优先级 → 升级级别链
export type EscalationMatrix = Record<string, EscalationLevel[]>;

// 默认矩阵（与后端 service/escalation_matrix.go DefaultEscalationMatrix 一致）
export const DEFAULT_ESCALATION_MATRIX: EscalationMatrix = {
  P1: [
    {
      level: 1,
      thresholdMinutes: 5,
      targetType: 'role',
      targetIds: [1],
      notifyChannels: ['email', 'sms', 'in_app'],
    },
    {
      level: 2,
      thresholdMinutes: 15,
      targetType: 'role',
      targetIds: [2],
      notifyChannels: ['email', 'sms', 'in_app', 'webhook'],
    },
    {
      level: 3,
      thresholdMinutes: 30,
      targetType: 'user',
      targetIds: [],
      notifyChannels: ['email', 'sms', 'in_app', 'webhook'],
    },
  ],
  P2: [
    {
      level: 1,
      thresholdMinutes: 30,
      targetType: 'role',
      targetIds: [1],
      notifyChannels: ['email', 'in_app'],
    },
    {
      level: 2,
      thresholdMinutes: 120,
      targetType: 'role',
      targetIds: [2],
      notifyChannels: ['email', 'in_app', 'webhook'],
    },
  ],
  P3: [
    {
      level: 1,
      thresholdMinutes: 240,
      targetType: 'role',
      targetIds: [1],
      notifyChannels: ['email', 'in_app'],
    },
  ],
};

// 升级历史条目（前端展示用）
export interface EscalationHistoryEntry {
  ticketId: number;
  ticketNumber: string;
  priority: string;
  level: number;
  triggeredAt: string;
  resolvedAt?: string;
  targetType: string;
  targetName: string;
}

export class EscalationMatrixApi {
  /**
   * 获取当前生效的升级矩阵。
    * - 优先调用 /api/v1/escalation-matrices（EscalationMatrixController）
    * - 后端返回空 payload 时回落到默认矩阵（正常降级）；请求失败则抛出，由页面显式报错
    */
  static async getMatrix(): Promise<EscalationMatrix> {
    try {
      const resp = await httpClient.get<{
        matrix?: EscalationMatrix;
        data?: EscalationMatrix;
        priorities?: Record<string, EscalationLevel[]>;
      }>('/api/v1/escalation-matrices');
      const payload = (resp as { matrix?: EscalationMatrix; data?: EscalationMatrix }).matrix
        || (resp as { data?: EscalationMatrix }).data
        || (resp as { priorities?: Record<string, EscalationLevel[]> }).priorities;
      if (payload && typeof payload === 'object' && Object.keys(payload).length > 0) {
        return payload as EscalationMatrix;
      }
    } catch (err) {
      // 2026-10-14 修复：失败时不再静默回落到硬编码默认矩阵——
      // 用户无法分辨看到的是真实配置还是内置占位，会把演示数据当生产配置。
      // 后端返回空 payload 时仍用默认矩阵兜底（属正常降级），请求失败则抛出。
      console.error('EscalationMatrixApi.getMatrix failed:', err);
      throw new Error('升级矩阵加载失败，请检查网络或联系管理员');
    }
    // 后端 200 但 payload 为空：属正常降级，用内置默认矩阵兜底
    return DEFAULT_ESCALATION_MATRIX;
  }
}

export default EscalationMatrixApi;