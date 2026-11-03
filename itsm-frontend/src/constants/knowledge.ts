/**
 * 知识库相关常量
 */

export const KnowledgeStatus = {
  PUBLISHED: 'published',
  DRAFT: 'draft',
};

export const KnowledgeStatusLabels = {
  [KnowledgeStatus.PUBLISHED]: '已发布',
  [KnowledgeStatus.DRAFT]: '草稿',
};

export const KnowledgeStatusColors = {
  [KnowledgeStatus.PUBLISHED]: 'green',
  [KnowledgeStatus.DRAFT]: 'orange',
};

/**
 * 评审阶段（后端 review_status，权威字段）。
 * 与上面 status（可见性投影 published/draft）是两条独立的轴：
 * 徽标必须展示评审阶段，否则会出现「徽标=草稿、按钮=评审中」的矛盾。
 */
export const ReviewStatus = {
  DRAFT: 'draft',
  UNDER_REVIEW: 'under_review',
  APPROVED: 'approved',
  REJECTED: 'rejected',
} as const;

export type ReviewStatus = (typeof ReviewStatus)[keyof typeof ReviewStatus];

export const ReviewStatusLabels: Record<ReviewStatus, string> = {
  [ReviewStatus.DRAFT]: '草稿',
  [ReviewStatus.UNDER_REVIEW]: '评审中',
  [ReviewStatus.APPROVED]: '已通过',
  [ReviewStatus.REJECTED]: '已驳回',
};

export const ReviewStatusColors: Record<ReviewStatus, string> = {
  [ReviewStatus.DRAFT]: 'orange',
  [ReviewStatus.UNDER_REVIEW]: 'blue',
  [ReviewStatus.APPROVED]: 'green',
  [ReviewStatus.REJECTED]: 'red',
};
