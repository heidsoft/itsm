/**
 * 协作 React Query Hooks
 * 提供评论、@提及、通知等协作功能的数据管理
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { message } from 'antd';
import { CollaborationApi } from '@/lib/api/collaboration-api';
import type {
  CommentListQuery,
  CreateCommentRequest,
  UpdateCommentRequest,
  NotificationListQuery,
  ActivityQuery,
} from '@/types/collaboration';

// ==================== Query Keys ====================

export const COLLABORATION_KEYS = {
  all: ['collaboration'] as const,
  comments: () => [...COLLABORATION_KEYS.all, 'comments'] as const,
  commentsList: (query: CommentListQuery) =>
    [...COLLABORATION_KEYS.comments(), 'list', query] as const,
  comment: (commentId: string) =>
    [...COLLABORATION_KEYS.comments(), commentId] as const,
  commentStats: (ticketId: number) =>
    [...COLLABORATION_KEYS.comments(), 'stats', ticketId] as const,
  notifications: () => [...COLLABORATION_KEYS.all, 'notifications'] as const,
  notificationsList: (query?: NotificationListQuery) =>
    [...COLLABORATION_KEYS.notifications(), 'list', query] as const,
  unreadCount: () =>
    [...COLLABORATION_KEYS.notifications(), 'unread-count'] as const,
  activities: (ticketId: number) =>
    [...COLLABORATION_KEYS.all, 'activities', ticketId] as const,
  watchers: (ticketId: number) =>
    [...COLLABORATION_KEYS.all, 'watchers', ticketId] as const,
  collaborationStats: (ticketId: number) =>
    [...COLLABORATION_KEYS.all, 'stats', ticketId] as const,
};

// ==================== Query Hooks ====================

/**
 * 获取评论列表
 */
export function useCommentsQuery(query: CommentListQuery) {
  return useQuery({
    queryKey: COLLABORATION_KEYS.commentsList(query),
    queryFn: () => CollaborationApi.getComments(query),
    enabled: !!query.ticketId,
    staleTime: 30000, // 30秒
  });
}

/**
 * 获取单个评论
 */
export function useCommentQuery(commentId: string, enabled: boolean = true) {
  return useQuery({
    queryKey: COLLABORATION_KEYS.comment(commentId),
    queryFn: () => CollaborationApi.getComment(commentId),
    enabled: enabled && !!commentId,
    staleTime: 60000, // 1分钟
  });
}

/**
 * 获取评论统计
 */
export function useCommentStatsQuery(ticketId: number, enabled: boolean = true) {
  return useQuery({
    queryKey: COLLABORATION_KEYS.commentStats(ticketId),
    queryFn: () => CollaborationApi.getCommentStats(ticketId),
    enabled: enabled && !!ticketId,
    staleTime: 30000,
  });
}

/**
 * 获取通知列表
 */
export function useNotificationsQuery(query?: NotificationListQuery) {
  return useQuery({
    queryKey: COLLABORATION_KEYS.notificationsList(query),
    queryFn: () => CollaborationApi.getNotifications(query || {}),
    staleTime: 10000, // 10秒
    refetchInterval: 30000, // 每30秒自动刷新
  });
}

/**
 * 获取未读通知数量
 */
export function useUnreadCountQuery() {
  return useQuery({
    queryKey: COLLABORATION_KEYS.unreadCount(),
    queryFn: () => CollaborationApi.getUnreadCount(),
    staleTime: 5000, // 5秒
    refetchInterval: 15000, // 每15秒自动刷新
  });
}

/**
 * 获取活动流
 */
export function useActivitiesQuery(query: ActivityQuery, enabled: boolean = true) {
  return useQuery({
    queryKey: COLLABORATION_KEYS.activities(query.ticketId),
    queryFn: () => CollaborationApi.getActivities(query),
    enabled: enabled && !!query.ticketId,
    staleTime: 30000,
  });
}

/**
 * 获取观察者列表
 */
export function useWatchersQuery(ticketId: number, enabled: boolean = true) {
  return useQuery({
    queryKey: COLLABORATION_KEYS.watchers(ticketId),
    queryFn: () => CollaborationApi.getWatchers(ticketId),
    enabled: enabled && !!ticketId,
    staleTime: 60000,
  });
}

/**
 * 获取协作统计
 */
export function useCollaborationStatsQuery(
  ticketId: number,
  enabled: boolean = true
) {
  return useQuery({
    queryKey: COLLABORATION_KEYS.collaborationStats(ticketId),
    queryFn: () => CollaborationApi.getCollaborationStats(ticketId),
    enabled: enabled && !!ticketId,
    staleTime: 60000,
  });
}

/**
 * 搜索@提及建议
 */
export function useMentionSuggestionsQuery(
  params: {
    query: string;
    ticketId?: number;
    limit?: number;
  },
  enabled: boolean = true
) {
  return useQuery({
    queryKey: ['mention-suggestions', params],
    queryFn: () => CollaborationApi.searchMentionSuggestions(params),
    enabled: enabled && params.query.length > 0,
    staleTime: 300000, // 5分钟
  });
}

// ==================== Mutation Hooks ====================

/**
 * 创建评论
 */
export function useCreateCommentMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: CreateCommentRequest) =>
      CollaborationApi.createComment(request),
    onSuccess: (_, variables) => {
      message.success('评论已发布');
      queryClient.invalidateQueries({
        queryKey: COLLABORATION_KEYS.commentsList({
          ticketId: variables.ticketId,
        } as any),
      });
      queryClient.invalidateQueries({
        queryKey: COLLABORATION_KEYS.commentStats(variables.ticketId),
      });
      queryClient.invalidateQueries({
        queryKey: COLLABORATION_KEYS.activities(variables.ticketId),
      });
      queryClient.invalidateQueries({ queryKey: ['tickets'] });
    },
    onError: (error: any) => {
      message.error(`发布评论失败：${error.message || '未知错误'}`);
    },
  });
}

/**
 * 更新评论
 */
export function useUpdateCommentMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      commentId,
      request,
    }: {
      commentId: string;
      request: UpdateCommentRequest;
    }) => CollaborationApi.updateComment(commentId, request),
    onSuccess: (_, variables) => {
      message.success('评论已更新');
      queryClient.invalidateQueries({
        queryKey: COLLABORATION_KEYS.comment(variables.commentId),
      });
      queryClient.invalidateQueries({
        queryKey: COLLABORATION_KEYS.comments(),
      });
    },
    onError: (error: any) => {
      message.error(`更新评论失败：${error.message || '未知错误'}`);
    },
  });
}

/**
 * 删除评论
 */
export function useDeleteCommentMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (commentId: string) => CollaborationApi.deleteComment(commentId),
    onSuccess: () => {
      message.success('评论已删除');
      queryClient.invalidateQueries({
        queryKey: COLLABORATION_KEYS.comments(),
      });
    },
    onError: (error: any) => {
      message.error(`删除评论失败：${error.message || '未知错误'}`);
    },
  });
}

/**
 * 点赞评论
 */
export function useLikeCommentMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (commentId: string) => CollaborationApi.likeComment(commentId),
    onSuccess: (_, commentId) => {
      queryClient.invalidateQueries({
        queryKey: COLLABORATION_KEYS.comment(commentId),
      });
    },
  });
}

/**
 * 取消点赞
 */
export function useUnlikeCommentMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (commentId: string) => CollaborationApi.unlikeComment(commentId),
    onSuccess: (_, commentId) => {
      queryClient.invalidateQueries({
        queryKey: COLLABORATION_KEYS.comment(commentId),
      });
    },
  });
}

/**
 * 标记通知为已读
 */
export function useMarkNotificationAsReadMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (notificationId: string) =>
      CollaborationApi.markNotificationAsRead(notificationId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: COLLABORATION_KEYS.notifications(),
      });
      queryClient.invalidateQueries({
        queryKey: COLLABORATION_KEYS.unreadCount(),
      });
    },
  });
}

/**
 * 标记所有通知为已读
 */
export function useMarkAllNotificationsAsReadMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => CollaborationApi.markAllNotificationsAsRead(),
    onSuccess: () => {
      message.success('所有通知已标记为已读');
      queryClient.invalidateQueries({
        queryKey: COLLABORATION_KEYS.notifications(),
      });
      queryClient.invalidateQueries({
        queryKey: COLLABORATION_KEYS.unreadCount(),
      });
    },
  });
}

/**
 * 添加观察者
 */
export function useAddWatcherMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: { ticketId: number; userId: number }) =>
      CollaborationApi.addWatcher(data),
    onSuccess: (_, variables) => {
      message.success('观察者已添加');
      queryClient.invalidateQueries({
        queryKey: COLLABORATION_KEYS.watchers(variables.ticketId),
      });
    },
    onError: (error: any) => {
      message.error(`添加观察者失败：${error.message || '未知错误'}`);
    },
  });
}

/**
 * 移除观察者
 */
export function useRemoveWatcherMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ ticketId, watcherId }: { ticketId: number; watcherId: string }) =>
      CollaborationApi.removeWatcher(ticketId, watcherId),
    onSuccess: (_, variables) => {
      message.success('观察者已移除');
      queryClient.invalidateQueries({
        queryKey: COLLABORATION_KEYS.watchers(variables.ticketId),
      });
    },
    onError: (error: any) => {
      message.error(`移除观察者失败：${error.message || '未知错误'}`);
    },
  });
}

/**
 * 观察工单
 */
export function useWatchTicketMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (ticketId: number) => CollaborationApi.watchTicket(ticketId),
    onSuccess: (_, ticketId) => {
      message.success('已开始观察此工单');
      queryClient.invalidateQueries({
        queryKey: COLLABORATION_KEYS.watchers(ticketId),
      });
    },
    onError: (error: any) => {
      message.error(`观察失败：${error.message || '未知错误'}`);
    },
  });
}

/**
 * 取消观察工单
 */
export function useUnwatchTicketMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (ticketId: number) => CollaborationApi.unwatchTicket(ticketId),
    onSuccess: (_, ticketId) => {
      message.success('已取消观察');
      queryClient.invalidateQueries({
        queryKey: COLLABORATION_KEYS.watchers(ticketId),
      });
    },
    onError: (error: any) => {
      message.error(`取消观察失败：${error.message || '未知错误'}`);
    },
  });
}

// ==================== 导出所有 Hooks ====================

const CollaborationHooks = {
  // Query Hooks
  useCommentsQuery,
  useCommentQuery,
  useCommentStatsQuery,
  useNotificationsQuery,
  useUnreadCountQuery,
  useActivitiesQuery,
  useWatchersQuery,
  useCollaborationStatsQuery,
  useMentionSuggestionsQuery,
  // Mutation Hooks
  useCreateCommentMutation,
  useUpdateCommentMutation,
  useDeleteCommentMutation,
  useLikeCommentMutation,
  useUnlikeCommentMutation,
  useMarkNotificationAsReadMutation,
  useMarkAllNotificationsAsReadMutation,
  useAddWatcherMutation,
  useRemoveWatcherMutation,
  useWatchTicketMutation,
  useUnwatchTicketMutation,
};

export default CollaborationHooks;

