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
