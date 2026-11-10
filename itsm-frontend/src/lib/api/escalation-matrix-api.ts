/**
 * SLA 升级矩阵 API
 *
 * 契约来源：`service/escalation_matrix.go` 的 `EscalationMatrix` / `EscalationLevel`，
 * 经 `handlers/escalation_matrix` 的 `GET|PUT /api/v1/escalation-matrices` 暴露。
 *
 * - 优先级键与 `Ticket.Priority` 对齐：critical / high / medium / low
 * - `afterMinutes` 是距告警开始的分钟数，不是距上一次升级的间隔
 * - `notifyRoles` 取主角色词表（users.role 精确匹配），`notifyUserIDs` 是具体用户
 * - 未配置自定义矩阵时后端直接返回内置默认矩阵，因此空响应属于异常而非「未配置」
 */

import { httpClient } from '@/lib/api/http-client';

// 单级升级策略
export interface EscalationLevel {
  level: number;
  afterMinutes: number;
  notifyRoles: string[];
  notifyUserIDs: number[];
  description: string;
}

// 优先级 → 升级级别链
export type EscalationMatrix = Record<string, EscalationLevel[]>;

// 与后端 service.DefaultEscalationMatrix 逐字段一致，仅用于加载失败时标注「内置默认」展示。
export const DEFAULT_ESCALATION_MATRIX: EscalationMatrix = {
  critical: [
    { level: 1, afterMinutes: 15, notifyRoles: ['manager'], notifyUserIDs: [], description: 'No response for 15 minutes, escalate to department manager' },
    { level: 2, afterMinutes: 30, notifyRoles: ['it_admin'], notifyUserIDs: [], description: 'No response for 30 minutes. Escalate to IT administrator.' },
    { level: 3, afterMinutes: 60, notifyRoles: ['admin'], notifyUserIDs: [], description: 'No response for 60 minutes. Escalate to tenant administrator.' },
  ],
  high: [
    { level: 1, afterMinutes: 60, notifyRoles: ['manager'], notifyUserIDs: [], description: 'No response for 1 hour. Escalated to department manager.' },
    { level: 2, afterMinutes: 240, notifyRoles: ['it_admin'], notifyUserIDs: [], description: 'No response for 4 hours Escalate to IT administrator' },
  ],
  medium: [
    { level: 1, afterMinutes: 240, notifyRoles: ['manager'], notifyUserIDs: [], description: 'No response for 4 hours. Escalated to department manager.' },
  ],
  low: [
    { level: 1, afterMinutes: 480, notifyRoles: ['manager'], notifyUserIDs: [], description: 'No response for 8 hours. Escalated to department manager.' },
  ],
};

export class EscalationMatrixApi {
  /** 获取当前租户生效的升级矩阵。后端总会返回矩阵（自定义或内置默认），空响应视为不可用并抛出。 */
  static async getMatrix(): Promise<EscalationMatrix> {
    const matrix = await httpClient.get<EscalationMatrix | null>('/api/v1/escalation-matrices');
    if (!matrix || typeof matrix !== 'object' || Object.keys(matrix).length === 0) {
      throw new Error('The upgrade matrix returned is empty and the current effective configuration cannot be confirmed.');
    }
    return matrix;
  }
}

export default EscalationMatrixApi;