import { renderHook, waitFor } from '@testing-library/react';
import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  useCommentsQuery,
  useCommentQuery,
  useCommentStatsQuery,
  useNotificationsQuery,
  useUnreadCountQuery,
  useActivitiesQuery,
  useWatchersQuery,
  useCollaborationStatsQuery,
  useMentionSuggestionsQuery,
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
  COLLABORATION_KEYS,
} from '../useCollaboration';

// Mock antd
jest.mock('antd', () => ({
  message: {
    success: jest.fn(),
    error: jest.fn(),
  },
}));

// Mock CollaborationApi
jest.mock('@/lib/api/collaboration-api', () => ({
  CollaborationApi: {
    getComments: jest.fn(),
    getComment: jest.fn(),
    getCommentStats: jest.fn(),
    getNotifications: jest.fn(),
    getUnreadCount: jest.fn(),
    getActivities: jest.fn(),
    getWatchers: jest.fn(),
    getCollaborationStats: jest.fn(),
    searchMentionSuggestions: jest.fn(),
    createComment: jest.fn(),
    updateComment: jest.fn(),
    deleteComment: jest.fn(),
    likeComment: jest.fn(),
    unlikeComment: jest.fn(),
    markNotificationAsRead: jest.fn(),
    markAllNotificationsAsRead: jest.fn(),
    addWatcher: jest.fn(),
    removeWatcher: jest.fn(),
    watchTicket: jest.fn(),
    unwatchTicket: jest.fn(),
  },
}));

import { CollaborationApi } from '@/lib/api/collaboration-api';
const mockApi = CollaborationApi as jest.Mocked<typeof CollaborationApi>;

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, refetchInterval: false } },
  });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
};

describe('useCollaboration hooks', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('COLLABORATION_KEYS', () => {
    it('should generate correct query keys', () => {
      expect(COLLABORATION_KEYS.all).toEqual(['collaboration']);
      expect(COLLABORATION_KEYS.comments()).toEqual(['collaboration', 'comments']);
      expect(COLLABORATION_KEYS.comment('c1')).toEqual(['collaboration', 'comments', 'c1']);
      expect(COLLABORATION_KEYS.commentStats(1)).toEqual(['collaboration', 'comments', 'stats', 1]);
      expect(COLLABORATION_KEYS.notifications()).toEqual(['collaboration', 'notifications']);
      expect(COLLABORATION_KEYS.unreadCount()).toEqual(['collaboration', 'notifications', 'unread-count']);
      expect(COLLABORATION_KEYS.activities(1)).toEqual(['collaboration', 'activities', 1]);
      expect(COLLABORATION_KEYS.watchers(1)).toEqual(['collaboration', 'watchers', 1]);
      expect(COLLABORATION_KEYS.collaborationStats(1)).toEqual(['collaboration', 'stats', 1]);
    });
  });

  describe('useCommentsQuery', () => {
    it('should fetch comments when ticketId is valid', async () => {
      mockApi.getComments.mockResolvedValue({ items: [], total: 0 });

      const { result } = renderHook(
        () => useCommentsQuery({ ticketId: 1 } as any),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.getComments).toHaveBeenCalledWith({ ticketId: 1 });
    });

    it('should not fetch when ticketId is 0', () => {
      renderHook(
        () => useCommentsQuery({ ticketId: 0 } as any),
        { wrapper: createWrapper() }
      );

      expect(mockApi.getComments).not.toHaveBeenCalled();
    });
  });

  describe('useCommentQuery', () => {
    it('should fetch a single comment', async () => {
      mockApi.getComment.mockResolvedValue({ id: 'c1', content: 'Hello' });

      const { result } = renderHook(() => useCommentQuery('c1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.getComment).toHaveBeenCalledWith('c1');
    });

    it('should not fetch when commentId is empty', () => {
      renderHook(() => useCommentQuery(''), { wrapper: createWrapper() });
      expect(mockApi.getComment).not.toHaveBeenCalled();
    });

    it('should not fetch when disabled', () => {
      renderHook(() => useCommentQuery('c1', false), { wrapper: createWrapper() });
      expect(mockApi.getComment).not.toHaveBeenCalled();
    });
  });

  describe('useCommentStatsQuery', () => {
    it('should fetch comment stats', async () => {
      mockApi.getCommentStats.mockResolvedValue({ total: 5, replied: 3 });

      const { result } = renderHook(() => useCommentStatsQuery(1), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.getCommentStats).toHaveBeenCalledWith(1);
    });

    it('should not fetch when ticketId is 0', () => {
      renderHook(() => useCommentStatsQuery(0), { wrapper: createWrapper() });
      expect(mockApi.getCommentStats).not.toHaveBeenCalled();
    });
  });

  describe('useNotificationsQuery', () => {
    it('should fetch notifications', async () => {
      mockApi.getNotifications.mockResolvedValue({ items: [], total: 0 });

      const { result } = renderHook(() => useNotificationsQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.getNotifications).toHaveBeenCalledWith({});
    });

    it('should pass query params', async () => {
      mockApi.getNotifications.mockResolvedValue({ items: [], total: 0 });
      const query = { unread: true } as any;

      const { result } = renderHook(() => useNotificationsQuery(query), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.getNotifications).toHaveBeenCalledWith(query);
    });
  });

  describe('useUnreadCountQuery', () => {
    it('should fetch unread count', async () => {
      mockApi.getUnreadCount.mockResolvedValue({ count: 7 });

      const { result } = renderHook(() => useUnreadCountQuery(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.getUnreadCount).toHaveBeenCalled();
    });
  });

  describe('useActivitiesQuery', () => {
    it('should fetch activities', async () => {
      mockApi.getActivities.mockResolvedValue([]);
      const query = { ticketId: 1 } as any;

      const { result } = renderHook(() => useActivitiesQuery(query), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.getActivities).toHaveBeenCalledWith(query);
    });

    it('should not fetch when ticketId is 0', () => {
      renderHook(() => useActivitiesQuery({ ticketId: 0 } as any), {
        wrapper: createWrapper(),
      });
      expect(mockApi.getActivities).not.toHaveBeenCalled();
    });
  });

  describe('useWatchersQuery', () => {
    it('should fetch watchers', async () => {
      mockApi.getWatchers.mockResolvedValue([{ id: 'w1' }]);

      const { result } = renderHook(() => useWatchersQuery(1), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.getWatchers).toHaveBeenCalledWith(1);
    });

    it('should not fetch when ticketId is 0', () => {
      renderHook(() => useWatchersQuery(0), { wrapper: createWrapper() });
      expect(mockApi.getWatchers).not.toHaveBeenCalled();
    });
  });

  describe('useCollaborationStatsQuery', () => {
    it('should fetch collaboration stats', async () => {
      mockApi.getCollaborationStats.mockResolvedValue({ comments: 5 });

      const { result } = renderHook(() => useCollaborationStatsQuery(1), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.getCollaborationStats).toHaveBeenCalledWith(1);
    });
  });

  describe('useMentionSuggestionsQuery', () => {
    it('should fetch suggestions when query has content', async () => {
      mockApi.searchMentionSuggestions.mockResolvedValue([{ id: 'u1', name: 'User' }]);

      const { result } = renderHook(
        () => useMentionSuggestionsQuery({ query: 'user', ticketId: 1 }),
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.searchMentionSuggestions).toHaveBeenCalledWith({ query: 'user', ticketId: 1 });
    });

    it('should not fetch when query is empty', () => {
      renderHook(
        () => useMentionSuggestionsQuery({ query: '' }),
        { wrapper: createWrapper() }
      );
      expect(mockApi.searchMentionSuggestions).not.toHaveBeenCalled();
    });
  });

  describe('useCreateCommentMutation', () => {
    it('should create a comment', async () => {
      mockApi.createComment.mockResolvedValue({ id: 'new-c' });

      const { result } = renderHook(() => useCreateCommentMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ ticketId: 1, content: 'Hello' } as any);

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.createComment).toHaveBeenCalled();
    });

    it('should handle create error', async () => {
      mockApi.createComment.mockRejectedValue(new Error('Create failed'));

      const { result } = renderHook(() => useCreateCommentMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ ticketId: 1, content: 'Hello' } as any);

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });
    });
  });

  describe('useUpdateCommentMutation', () => {
    it('should update a comment', async () => {
      mockApi.updateComment.mockResolvedValue({ id: 'c1', content: 'Updated' });

      const { result } = renderHook(() => useUpdateCommentMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ commentId: 'c1', request: { content: 'Updated' } as any });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.updateComment).toHaveBeenCalledWith('c1', { content: 'Updated' });
    });
  });

  describe('useDeleteCommentMutation', () => {
    it('should delete a comment', async () => {
      mockApi.deleteComment.mockResolvedValue(undefined);

      const { result } = renderHook(() => useDeleteCommentMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate('c1');

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.deleteComment).toHaveBeenCalledWith('c1');
    });
  });

  describe('useLikeCommentMutation', () => {
    it('should like a comment', async () => {
      mockApi.likeComment.mockResolvedValue(undefined);

      const { result } = renderHook(() => useLikeCommentMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate('c1');

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.likeComment).toHaveBeenCalledWith('c1');
    });
  });

  describe('useUnlikeCommentMutation', () => {
    it('should unlike a comment', async () => {
      mockApi.unlikeComment.mockResolvedValue(undefined);

      const { result } = renderHook(() => useUnlikeCommentMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate('c1');

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.unlikeComment).toHaveBeenCalledWith('c1');
    });
  });

  describe('useMarkNotificationAsReadMutation', () => {
    it('should mark notification as read', async () => {
      mockApi.markNotificationAsRead.mockResolvedValue(undefined);

      const { result } = renderHook(() => useMarkNotificationAsReadMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate('n1');

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.markNotificationAsRead).toHaveBeenCalledWith('n1');
    });
  });

  describe('useMarkAllNotificationsAsReadMutation', () => {
    it('should mark all notifications as read', async () => {
      mockApi.markAllNotificationsAsRead.mockResolvedValue(undefined);

      const { result } = renderHook(() => useMarkAllNotificationsAsReadMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate();

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.markAllNotificationsAsRead).toHaveBeenCalled();
    });
  });

  describe('useAddWatcherMutation', () => {
    it('should add a watcher', async () => {
      mockApi.addWatcher.mockResolvedValue(undefined);

      const { result } = renderHook(() => useAddWatcherMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ ticketId: 1, userId: 2 });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.addWatcher).toHaveBeenCalledWith({ ticketId: 1, userId: 2 });
    });

    it('should handle add watcher error', async () => {
      mockApi.addWatcher.mockRejectedValue(new Error('Failed'));

      const { result } = renderHook(() => useAddWatcherMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ ticketId: 1, userId: 2 });

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });
    });
  });

  describe('useRemoveWatcherMutation', () => {
    it('should remove a watcher', async () => {
      mockApi.removeWatcher.mockResolvedValue(undefined);

      const { result } = renderHook(() => useRemoveWatcherMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ ticketId: 1, watcherId: 'w1' });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.removeWatcher).toHaveBeenCalledWith(1, 'w1');
    });
  });

  describe('useWatchTicketMutation', () => {
    it('should watch a ticket', async () => {
      mockApi.watchTicket.mockResolvedValue(undefined);

      const { result } = renderHook(() => useWatchTicketMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate(1);

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.watchTicket).toHaveBeenCalledWith(1);
    });
  });

  describe('useUnwatchTicketMutation', () => {
    it('should unwatch a ticket', async () => {
      mockApi.unwatchTicket.mockResolvedValue(undefined);

      const { result } = renderHook(() => useUnwatchTicketMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate(1);

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(mockApi.unwatchTicket).toHaveBeenCalledWith(1);
    });

    it('should handle unwatch error', async () => {
      mockApi.unwatchTicket.mockRejectedValue(new Error('Unwatch failed'));

      const { result } = renderHook(() => useUnwatchTicketMutation(), {
        wrapper: createWrapper(),
      });

      result.current.mutate(1);

      await waitFor(() => {
        expect(result.current.isError).toBe(true);
      });
    });
  });
});
