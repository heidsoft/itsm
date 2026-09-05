/**
 * 问题管理相关常量定义
 */

// 问题状态
export enum ProblemStatus {
  OPEN = 'open',
  INVESTIGATING = 'investigating',
  /** @deprecated 仅用于兼容历史数据，新请求使用 INVESTIGATING。 */
  IN_PROGRESS = 'in_progress',
  RESOLVED = 'resolved',
  CLOSED = 'closed',
  IDENTIFIED = 'identified',
}

// 状态描述映射
export const ProblemStatusLabels: Record<ProblemStatus, string> = {
  [ProblemStatus.OPEN]: '待处理',
  [ProblemStatus.INVESTIGATING]: '调查中',
  [ProblemStatus.IN_PROGRESS]: '处理中',
  [ProblemStatus.RESOLVED]: '已解决',
  [ProblemStatus.CLOSED]: '已关闭',
  [ProblemStatus.IDENTIFIED]: '已识别',
};

// 优先级
export enum ProblemPriority {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  CRITICAL = 'critical',
}

export const ProblemPriorityLabels: Record<ProblemPriority, string> = {
  [ProblemPriority.LOW]: '低',
  [ProblemPriority.MEDIUM]: '中',
  [ProblemPriority.HIGH]: '高',
  [ProblemPriority.CRITICAL]: '极高',
};

// 问题分类——中文字面量为存储值，不走 i18n，避免后端/前端枚举不一致。
// 枚举同时被 new/edit/list 过滤器以及后端 Problem.Category 字段共享。
// 该枚举为业务可观测字段：其它模块（事件/请求/知识库）如有分类需各自定义。
export const ProblemCategoryValues = [
  '系统问题',
  '网络问题',
  '数据库问题',
  '应用问题',
  '安全问题',
  '硬件问题',
  '其他',
] as const;

export type ProblemCategory = (typeof ProblemCategoryValues)[number];

export const ProblemCategoryLabels: Record<ProblemCategory, string> = {
  系统问题: '系统问题',
  网络问题: '网络问题',
  数据库问题: '数据库问题',
  应用问题: '应用问题',
  安全问题: '安全问题',
  硬件问题: '硬件问题',
  其他: '其他',
};

// 给 Select options 用的全量列表，便于 new/edit/filter 共享。
export const ProblemCategoryOptions: Array<{ value: ProblemCategory; label: string }> =
  ProblemCategoryValues.map(v => ({ value: v, label: ProblemCategoryLabels[v] }));

// 是已知分类（防止后端脏数据带未知值导致 antd v6 Select 看起来“空白”）。
export function isKnownProblemCategory(v: unknown): v is ProblemCategory {
  return typeof v === 'string' && (ProblemCategoryValues as readonly string[]).includes(v);
}
