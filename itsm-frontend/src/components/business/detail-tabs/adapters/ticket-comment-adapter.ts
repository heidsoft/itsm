import { TicketCommentApi } from '@/lib/api/ticket-comment-api';
import type { CommentAdapter, CommentItem } from '../types';

export const ticketCommentAdapter: CommentAdapter = {
  async list(targetId) {
    const { comments, total } = await TicketCommentApi.getComments(Number(targetId));
    return { comments: (comments || []) as unknown as CommentItem[], total };
  },
  async create(targetId, data) {
    const res = await TicketCommentApi.createComment(Number(targetId), data);
    return res as unknown as CommentItem;
  },
  async update(targetId, commentId, data) {
    const res = await TicketCommentApi.updateComment(Number(targetId), commentId, data);
    return res as unknown as CommentItem;
  },
  async remove(targetId, commentId) {
    await TicketCommentApi.deleteComment(Number(targetId), commentId);
  },
};
