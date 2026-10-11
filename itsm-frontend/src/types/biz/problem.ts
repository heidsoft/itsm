/**
 * 问题管理类型定义
 */

import type { ProblemPriority, ProblemStatus } from '@/constants/problem';
import type { ProblemListParams } from '@/lib/api/problem-api';

/**
 * 问题实体只有一个形状：`@/lib/api/problem-api` 的 `Problem`，它逐字段对齐后端
 * `dto.ProblemResponse`。这里只 re-export（与 `types/user.ts` 同一口径），
 * 不要再维护一份副本，否则调用方只能靠 `as unknown as Problem` 桥接。
 */
export type { Problem } from '@/lib/api/problem-api';

// 创建问题请求
export interface CreateProblemRequest {
  title: string;
  description: string;
  priority: ProblemPriority;
  category: string;
  rootCause?: string;
  impact?: string;
}

// 更新问题请求
export interface UpdateProblemRequest {
  title?: string;
  description?: string;
  status?: ProblemStatus;
  priority?: ProblemPriority;
  category?: string;
  rootCause?: string;
  impact?: string;
  assigneeId?: number;
}

// 列表查询参数同样只有一份：直接复用 ProblemListParams（与后端 dto.ListProblemsRequest
// 逐字段一致），不再维护同构副本。
export type ProblemQuery = ProblemListParams;

// 统计响应同样只有一份：复用 ProblemApi 的 ProblemStatsResponse，不再维护缺
// byStatus/byPriority 的同构副本。
export type { ProblemStatsResponse } from '@/lib/api/problem-api';
