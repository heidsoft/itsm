import { ProblemApi, type ProblemComment } from '@/lib/api/problem-api';
import type { CommentAdapter, CommentItem } from '../types';

/**
 * Problem 评论 adapter
 *
 * 后端契约（2026-10-21 GA 落地）：只支持 list/add，无 update/delete ——
 * 与 incident adapter 同样的取舍：update 字段有意省略，
 * CommentPanel 检测到 undefined 会禁用编辑按钮；remove 抛「不支持」。
 * isInternal/mentions 后端真实落库，展示与新建都走完整契约。
 */
export const problemCommentAdapter: CommentAdapter = {
  async list(targetId) {
    const { items, total } = await ProblemApi.getProblemComments(Number(targetId));
    const comments: CommentItem[] = (items || []).map((c: ProblemComment) => ({
      id: c.id,
      userId: c.userId,
      user: { id: c.userId, name: c.userName },
      content: c.content,
      isInternal: c.isInternal,
      mentions: c.mentions,
      attachments: c.attachments,
      createdAt: c.createdAt,
      updatedAt: c.updatedAt,
    }));
    return { comments, total };
  },
  async create(targetId, data) {
    const created = await ProblemApi.addProblemComment(Number(targetId), {
      content: data.content,
      isInternal: data.isInternal,
      mentions: data.mentions,
      attachments: data.attachments,
    });
    return {
      id: created.id,
      userId: created.userId,
      user: { id: created.userId, name: created.userName },
      content: created.content,
      isInternal: created.isInternal,
      mentions: created.mentions,
      attachments: created.attachments,
      createdAt: created.createdAt,
      updatedAt: created.updatedAt,
    };
  },
  async remove() {
    throw new Error('问题评论暂不支持删除');
  },
  // update 有意省略 —— 后端不支持
};
