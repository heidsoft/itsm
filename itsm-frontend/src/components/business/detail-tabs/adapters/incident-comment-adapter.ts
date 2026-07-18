import { IncidentAPI, type IncidentComment } from '@/lib/api/incident-api';
import type { CommentAdapter, CommentItem } from '../types';

/**
 * Incident 评论 adapter
 * 注意：后端 API 是阉割版 —— 只支持 list/add/delete，无 update、无 isInternal/mentions
 * update 字段被有意省略，CommentPanel 会检测到 undefined 并禁用编辑按钮
 */
export const incidentCommentAdapter: CommentAdapter = {
  async list(targetId) {
    const comments = await IncidentAPI.getIncidentComments(Number(targetId));
    const items: CommentItem[] = (comments || []).map((c: IncidentComment) => ({
      id: c.id,
      userId: c.userId,
      user: { id: c.userId, name: c.userName },
      content: c.content,
      createdAt: c.createdAt,
      updatedAt: c.updatedAt,
    }));
    return { comments: items, total: items.length };
  },
  async create(targetId, data) {
    // 后端 addComment 返回的是 Incident 对象，不是评论；再拉一次 list 取新增记录
    await IncidentAPI.addComment(Number(targetId), { content: data.content });
    const comments = await IncidentAPI.getIncidentComments(Number(targetId));
    const latest = comments[comments.length - 1];
    return {
      id: latest?.id ?? 0,
      userId: latest?.userId ?? 0,
      user: { id: latest?.userId ?? 0, name: latest?.userName },
      content: latest?.content ?? data.content,
      createdAt: latest?.createdAt ?? new Date().toISOString(),
    };
  },
  async remove(targetId, commentId) {
    await IncidentAPI.deleteIncidentComment(Number(targetId), commentId);
  },
  // update 有意省略 —— 后端不支持
};
