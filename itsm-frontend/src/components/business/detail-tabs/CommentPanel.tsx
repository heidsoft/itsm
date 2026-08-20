'use client';

import React, { useEffect, useState, useCallback } from 'react';
import { Send, Edit, Trash2, MessageSquare, AtSign, User } from 'lucide-react';
import {
  Card,
  Typography,
  Button,
  Input,
  Avatar,
  Tag as AntTag,
  Modal,
  App,
  Spin,
  Alert,
} from 'antd';
import { UserSelect } from '@/components/common/UserSelect';
import { useI18n } from '@/lib/i18n/useI18n';
import type {
  CommentAdapter,
  CommentItem,
  TargetType,
} from './types';

const { Text, Paragraph } = Typography;
const { TextArea } = Input;

export interface CommentPanelProps {
  targetType: TargetType;
  targetId: number | string;
  adapter: CommentAdapter;
  showInternalToggle?: boolean;
  showMentions?: boolean;
  currentUserId?: number;
  formatDateTime?: (dateString: string) => string;
}

export const CommentPanel: React.FC<CommentPanelProps> = ({
  targetId,
  adapter,
  showInternalToggle = true,
  showMentions = true,
  currentUserId,
  formatDateTime,
}) => {
  const { t, language } = useI18n();
  const { message } = App.useApp();
  const [comments, setComments] = useState<CommentItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [newComment, setNewComment] = useState('');
  const [isInternal, setIsInternal] = useState(false);
  const [mentionedUsers, setMentionedUsers] = useState<number[]>([]);
  const [editingCommentId, setEditingCommentId] = useState<number | null>(null);
  const [editingCommentContent, setEditingCommentContent] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const canEditByAdapter = typeof adapter.update === 'function';
  const canEditByUser = (c: CommentItem) => (currentUserId ? c.userId === currentUserId : true);

  const defaultFormat = useCallback(
    (s: string) => (s ? new Date(s).toLocaleString(language === 'en-US' ? 'en-US' : 'zh-CN') : ''),
    [language]
  );
  const fmt = formatDateTime ?? defaultFormat;

  const fetchComments = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const { comments } = await adapter.list(targetId);
      setComments(comments || []);
    } catch (e) {
      const msg = e instanceof Error ? e.message : t('detailTabs.commentFailed');
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, [adapter, targetId, t]);

  useEffect(() => {
    void fetchComments();
  }, [fetchComments]);

  const handleAddComment = async () => {
    if (!newComment.trim()) {
      message.warning(t('detailTabs.commentRequired'));
      return;
    }
    setSubmitting(true);
    try {
      await adapter.create(targetId, {
        content: newComment,
        isInternal: showInternalToggle ? isInternal : undefined,
        mentions: showMentions ? mentionedUsers : undefined,
      });
      setNewComment('');
      setMentionedUsers([]);
      setIsInternal(false);
      await fetchComments();
      message.success(t('detailTabs.commentSuccess'));
    } catch (e) {
      const msg = e instanceof Error ? e.message : t('detailTabs.commentFailed');
      message.error(msg);
    } finally {
      setSubmitting(false);
    }
  };

  const handleEditComment = async (commentId: number) => {
    if (!editingCommentContent.trim() || !adapter.update) {
      message.warning(t('detailTabs.commentRequired'));
      return;
    }
    setSubmitting(true);
    try {
      await adapter.update(targetId, commentId, {
        content: editingCommentContent,
      });
      setEditingCommentId(null);
      setEditingCommentContent('');
      await fetchComments();
      message.success(t('detailTabs.commentSuccess'));
    } catch (e) {
      const msg = e instanceof Error ? e.message : t('detailTabs.commentFailed');
      message.error(msg);
    } finally {
      setSubmitting(false);
    }
  };

  const handleDeleteComment = async (commentId: number) => {
    try {
      await adapter.remove(targetId, commentId);
      await fetchComments();
      message.success(t('detailTabs.commentSuccess'));
    } catch (e) {
      const msg = e instanceof Error ? e.message : t('detailTabs.commentFailed');
      message.error(msg);
    }
  };

  const startEditComment = (comment: CommentItem) => {
    setEditingCommentId(comment.id);
    setEditingCommentContent(comment.content);
  };

  const cancelEdit = () => {
    setEditingCommentId(null);
    setEditingCommentContent('');
  };

  if (loading && comments.length === 0) {
    return (
      <div className="p-6 text-center">
        <Spin />
      </div>
    );
  }

  return (
    <div className="p-6">
      {error && (
        <Alert
          message={error}
          type="error"
          showIcon
          closable
          className="mb-4"
          onClose={() => setError(null)}
          action={
            <Button size="small" type="link" onClick={() => void fetchComments()}>
              {t('common.retry')}
            </Button>
          }
        />
      )}

      <div className="mb-6">
        <Card title={t('detailTabs.addComment')} className="shadow-sm">
          <div className="space-y-4">
            {showInternalToggle && (
              <div className="flex items-center space-x-4">
                <div className="flex items-center space-x-2">
                  <input
                    type="checkbox"
                    id={`internal-${targetId}`}
                    checked={isInternal}
                    onChange={(e) => setIsInternal(e.target.checked)}
                  />
                  <label
                    htmlFor={`internal-${targetId}`}
                    className="text-sm text-gray-600"
                  >
                    {t('detailTabs.internalOnly')}
                  </label>
                </div>
              </div>
            )}
            {showMentions && (
              <div>
                <div className="mb-2">
                  <Text type="secondary" className="text-sm">
                    <AtSign className="w-4 h-4 inline mr-1" />
                    @ {t('detailTabs.mentions')}
                  </Text>
                </div>
                <UserSelect
                  value={mentionedUsers}
                  onChange={setMentionedUsers}
                  mode="multiple"
                  placeholder={t('detailTabs.mentionsPlaceholder')}
                  style={{ width: '100%' }}
                />
              </div>
            )}
            <label htmlFor={`comment-content-${targetId}`} className="sr-only">
              {t('detailTabs.commentPlaceholder')}
            </label>
            <TextArea
              id={`comment-content-${targetId}`}
              name="newComment"
              aria-label={t('detailTabs.commentPlaceholder')}
              value={newComment}
              onChange={(e) => setNewComment(e.target.value)}
              placeholder={t('detailTabs.commentPlaceholder')}
              rows={4}
            />
            <div className="flex justify-end">
              <Button
                type="primary"
                icon={<Send size={14} />}
                onClick={handleAddComment}
                disabled={!newComment.trim()}
                loading={submitting}
              >
                {t('detailTabs.postComment')}
              </Button>
            </div>
          </div>
        </Card>
      </div>

      <div className="space-y-4">
        {comments.map((comment) => (
          <Card key={comment.id} className="shadow-sm">
            {editingCommentId === comment.id ? (
              <div className="space-y-3">
                <label
                  htmlFor={`edit-comment-${comment.id}`}
                  className="sr-only"
                >
                  {t('detailTabs.commentPlaceholder')}
                </label>
                <TextArea
                  id={`edit-comment-${comment.id}`}
                  name={`editComment-${comment.id}`}
                  aria-label={t('detailTabs.commentPlaceholder')}
                  value={editingCommentContent}
                  onChange={(e) => setEditingCommentContent(e.target.value)}
                  rows={3}
                  placeholder={t('detailTabs.commentPlaceholder')}
                />
                <div className="flex justify-end space-x-2">
                  <Button onClick={cancelEdit}>{t('common.cancel')}</Button>
                  <Button
                    type="primary"
                    onClick={() => handleEditComment(comment.id)}
                    disabled={!editingCommentContent.trim()}
                    loading={submitting}
                  >
                    {t('common.save')}
                  </Button>
                </div>
              </div>
            ) : (
              <div className="flex items-start space-x-3">
                <Avatar size="small" icon={<User size={14} />}>
                  {comment.user?.name?.[0] || comment.user?.username?.[0]}
                </Avatar>
                <div className="flex-1">
                  <div className="flex items-center space-x-2 mb-2 flex-wrap">
                    <Text strong>
                      {comment.user?.name || comment.user?.username || t('detailTabs.unknownUser')}
                    </Text>
                    {comment.isInternal && <AntTag color="orange">{t('detailTabs.internalOnly')}</AntTag>}
                    {comment.mentions && comment.mentions.length > 0 && (
                      <AntTag color="blue" icon={<AtSign className="w-3 h-3" />}>
                        {t('detailTabs.mentionCount', { count: comment.mentions.length })}
                      </AntTag>
                    )}
                    <Text type="secondary" className="text-sm">
                      {fmt(comment.createdAt)}
                    </Text>
                    {comment.updatedAt && comment.updatedAt !== comment.createdAt && (
                      <Text type="secondary" className="text-xs">
                        （{t('detailTabs.edited')}）
                      </Text>
                    )}
                  </div>
                  <Paragraph className="mb-2 whitespace-pre-wrap">
                    {comment.content}
                  </Paragraph>
                  <div className="flex items-center space-x-2">
                    {canEditByAdapter && canEditByUser(comment) && (
                      <Button
                        type="link"
                        size="small"
                        icon={<Edit className="w-3 h-3" />}
                        onClick={() => startEditComment(comment)}
                      >
                        {t('common.edit')}
                      </Button>
                    )}
                    {canEditByUser(comment) && (
                      <Button
                        type="link"
                        size="small"
                        danger
                        icon={<Trash2 className="w-3 h-3" />}
                        onClick={() => {
                          Modal.confirm({
                            title: t('detailTabs.deleteConfirmTitle'),
                            content: t('detailTabs.deleteConfirmContent'),
                            okText: t('common.delete'),
                            okType: 'danger',
                            cancelText: t('common.cancel'),
                            onOk: () => handleDeleteComment(comment.id),
                          });
                        }}
                      >
                        {t('common.delete')}
                      </Button>
                    )}
                  </div>
                </div>
              </div>
            )}
          </Card>
        ))}

        {comments.length === 0 && !loading && (
          <div className="text-center py-8 text-gray-500">
            <MessageSquare className="w-16 h-16 mx-auto mb-4 text-gray-300" />
            <Text>{t('detailTabs.noComments')}</Text>
          </div>
        )}
      </div>
    </div>
  );
};

export default CommentPanel;
