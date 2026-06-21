/**
 * SLA 升级矩阵 API
 *
 * 对应后端 service/escalation_matrix.go（service 层）
 *
 * 状态说明：
 * - 当前后端**没有** HTTP controller/router，矩阵是进程内缓存
 * - 矩阵被 EscalationMatrixService.GetMatrix() 在以下场景隐式使用：
 *   - SLA 告警升级
 *   - 工单超时升级
 *   - BPMN 任务超时
 * - 这里仅提供前端类型定义和可视化辅助函数，UI 可作为"只读预览"
 *
 * 字段与 service/escalation_matrix.go 保持一致。
 */

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
   * 当前后端没有 HTTP 端点暴露矩阵，因此前端展示的是默认矩阵。
   * 后续若后端添加 controller，可替换实现为 httpClient.get(...)。
   */
  static async getMatrix(): Promise<EscalationMatrix> {
    // 后端当前为进程内缓存，无 HTTP API。
    // 兜底返回默认矩阵，避免页面空白。
    return Promise.resolve(DEFAULT_ESCALATION_MATRIX);
  }
}

export default EscalationMatrixApi;