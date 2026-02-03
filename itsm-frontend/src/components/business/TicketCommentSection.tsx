'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  List,
  Avatar,
  Input,
  Button,
  Space,
  Tag,
  Popconfirm,
  App,
  Typography,
  Divider,
  Switch,
  Tooltip,
} from 'antd';
import {
  Send,
  Edit,
  Trash2,
  User,
  AtSign,
  Lock,
  MessageSquare,
} from 'lucide-react';
import { TicketCommentApi, TicketComment, CreateTicketCommentRequest } from '@/lib/api/ticket-comment-api';
import { useAuthStore } from '@/lib/store/auth-store';
import { useI18n } from '@/lib/i18n';

const { TextArea } = Input;
const { Text, Paragraph } = Typography;

interface TicketCommentSectionProps {
  ticketId: number;
  canComment?: boolean;
  canEditComment?: (comment: TicketComment) => boolean;
  canDeleteComment?: (comment: TicketComment) => boolean;
  onCommentAdded?: (comment: TicketComment) => void;
  onCommentUpdated?: (comment: TicketComment) => void;
  onCommentDeleted?: (commentId: number) => void;
}

export const TicketCommentSection: React.FC<TicketCommentSectionProps> = ({
  ticketId,
  canComment = true,
  canEditComment,
  canDeleteComment,
  onCommentAdded,
  onCommentUpdated,
  onCommentDeleted,
}) => {
  const { message } = App.useApp();
  const { t } = useI18n();
  const { user } = useAuthStore();
  const [comments, setComments] = useState<TicketComment[]>([]);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [newComment, setNewComment] = useState('');
  const [isInternal, setIsInternal] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editingContent, setEditingContent] = useState('');

  // 加载评论列表
  const loadComments = useCallback(async () => {
    setLoading(true);
    try {
      const response = await TicketCommentApi.getComments(ticketId);
      setComments(response.comments || []);
    } catch (error) {
      console.error('Failed to load comments:', error);
      message.error(t('comments.loadFailed') || '加载评论失败');
    } finally {
      setLoading(false);
    }
  }, [ticketId, t]);

  useEffect(() => {
    loadComments();
  }, [loadComments]);

  // 添加评论
  const handleAddComment = async () => {
    if (!newComment.trim()) {
      message.warning(t('comments.contentRequired') || '请输入评论内容');
      return;
    }

    setSubmitting(true);
    try {
      const request: CreateTicketCommentRequest = {
        content: newComment.trim(),
        is_internal: isInternal,
      };

      const comment = await TicketCommentApi.createComment(ticketId, request);
      setComments(prev => [comment, ...prev]);
      setNewComment('');
      setIsInternal(false);
      message.success(t('comments.addSuccess') || '评论添加成功');
      onCommentAdded?.(comment);
    } catch (error) {
      console.error('Failed to add comment:', error);
      message.error(t('comments.addFailed') || '添加评论失败');
    } finally {
      setSubmitting(false);
    }
  };

  // 开始编辑
  const startEdit = (comment: TicketComment) => {
    setEditingId(comment.id);
    setEditingContent(comment.content);
  };

  // 取消编辑
  const cancelEdit = () => {
    setEditingId(null);
    setEditingContent('');
  };

  // 更新评论
  const handleUpdateComment = async (commentId: number) => {
    if (!editingContent.trim()) {
      message.warning(t('comments.contentRequired') || '请输入评论内容');
      return;
    }

    try {
      const updated = await TicketCommentApi.updateComment(ticketId, commentId, {
        content: editingContent.trim(),
      });
      setComments(prev =>
        prev.map(c => (c.id === commentId ? updated : c))
      );
      setEditingId(null);
      setEditingContent('');
      message.success(t('comments.updateSuccess') || '评论更新成功');
      onCommentUpdated?.(updated);
    } catch (error) {
      console.error('Failed to update comment:', error);
      message.error(t('comments.updateFailed') || '更新评论失败');
    }
  };

  // 删除评论
  const handleDeleteComment = async (commentId: number) => {
    try {
      await TicketCommentApi.deleteComment(ticketId, commentId);
      setComments(prev => prev.filter(c => c.id !== commentId));
      message.success(t('comments.deleteSuccess') || '评论删除成功');
      onCommentDeleted?.(commentId);
    } catch (error) {
      console.error('Failed to delete comment:', error);
      message.error(t('comments.deleteFailed') || '删除评论失败');
    }
  };

  // 格式化时间
  const formatTime = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    if (minutes < 1) return '刚刚';
    if (minutes < 60) return `${minutes}分钟前`;
    if (hours < 24) return `${hours}小时前`;
    if (days < 7) return `${days}天前`;
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  // 检查权限
  const canEdit = (comment: TicketComment) => {
    if (canEditComment) return canEditComment(comment);
    return comment.user_id === user?.id;
  };

  const canDelete = (comment: TicketComment) => {
    if (canDeleteComment) return canDeleteComment(comment);
    return comment.user_id === user?.id;
  };

  return (
    <div className='space-y-4'>
      {/* 评论输入框 */}
      {canComment && (
        <Card
          title={
            <Space>
              <MessageSquare className='w-4 h-4' />
              <span>{t('comments.title') || '评论'}</span>
            </Space>
          }
          className='shadow-sm'
        >
          <Space orientation='vertical' style={{ width: '100%' }} size='middle'>
            <TextArea
              value={newComment}
              onChange={e => setNewComment(e.target.value)}
              placeholder={t('comments.placeholder') || '输入评论内容...'}
              rows={4}
              maxLength={5000}
              showCount
            />
            <div className='flex items-center justify-between'>
              <Space>
                <Switch
                  checked={isInternal}
                  onChange={setIsInternal}
                  checkedChildren={<Lock className='w-3 h-3' />}
                  unCheckedChildren={<MessageSquare className='w-3 h-3' />}
                />
                <Text type='secondary' className='text-sm'>
                  {isInternal ? '内部备注' : '公开评论'}
                </Text>
              </Space>
              <Button
                type='primary'
                icon={<Send className='w-4 h-4' />}
                onClick={handleAddComment}
                loading={submitting}
                disabled={!newComment.trim()}
              >
                {t('comments.send') || '发送'}
              </Button>
            </div>
          </Space>
        </Card>
      )}

      {/* 评论列表 */}
      <Card
        title={
          <Space>
            <MessageSquare className='w-4 h-4' />
            <span>
              {t('comments.list') || '评论列表'} ({comments.length})
            </span>
          </Space>
        }
        className='shadow-sm'
        loading={loading}
      >
        {comments.length === 0 ? (
          <div className='text-center py-8 text-gray-400'>
            <MessageSquare className='w-12 h-12 mx-auto mb-2 opacity-50' />
            <Text type='secondary'>{t('comments.empty') || '暂无评论'}</Text>
          </div>
        ) : (
          <List
            dataSource={comments}
            renderItem={comment => (
              <List.Item
                key={comment.id}
                className='!px-0 !py-4 border-b border-gray-100 last:border-b-0'
              >
                <div className='flex items-start space-x-3 w-full'>
                  <Avatar
                    size='default'
                    icon={<User />}
                    src={comment.user?.name?.[0] || comment.user?.username?.[0]}
                  >
                    {comment.user?.name?.[0] || comment.user?.username?.[0] || 'U'}
                  </Avatar>
                  <div className='flex-1 min-w-0'>
                    <div className='flex items-center space-x-2 mb-2'>
                      <Text strong>
                        {comment.user?.name || comment.user?.username || '未知用户'}
                      </Text>
                      {comment.is_internal && (
                        <Tag color='orange' icon={<Lock className='w-3 h-3' />}>
                          {t('comments.internal') || '内部'}
                        </Tag>
                      )}
                      {comment.mentions && comment.mentions.length > 0 && (
                        <Tooltip title={`@${comment.mentions.length}位用户`}>
                          <Tag color='blue' icon={<AtSign className='w-3 h-3' />}>
                            @{comment.mentions.length}
                          </Tag>
                        </Tooltip>
                      )}
                      <Text type='secondary' className='text-xs'>
                        {formatTime(comment.created_at)}
                      </Text>
                      {comment.updated_at !== comment.created_at && (
                        <Text type='secondary' className='text-xs'>
                          （已编辑）
                        </Text>
                      )}
                    </div>
                    {editingId === comment.id ? (
                      <div className='space-y-2'>
                        <TextArea
                          value={editingContent}
                          onChange={e => setEditingContent(e.target.value)}
                          rows={3}
                          maxLength={5000}
                          showCount
                        />
                        <Space>
                          <Button
                            type='primary'
                            size='small'
                            onClick={() => handleUpdateComment(comment.id)}
                          >
                            {t('common.save') || '保存'}
                          </Button>
                          <Button size='small' onClick={cancelEdit}>
                            {t('common.cancel') || '取消'}
                          </Button>
                        </Space>
                      </div>
                    ) : (
                      <>
                        <Paragraph className='mb-2 whitespace-pre-wrap break-words'>
                          {comment.content}
                        </Paragraph>
                        {(canEdit(comment) || canDelete(comment)) && (
                          <Space size='small'>
                            {canEdit(comment) && (
                              <Button
                                type='text'
                                size='small'
                                icon={<Edit className='w-3 h-3' />}
                                onClick={() => startEdit(comment)}
                              >
                                {t('common.edit') || '编辑'}
                              </Button>
                            )}
                            {canDelete(comment) && (
                              <Popconfirm
                                title={t('comments.deleteConfirm') || '确定要删除这条评论吗？'}
                                onConfirm={() => handleDeleteComment(comment.id)}
                                okText={t('common.confirm') || '确定'}
                                cancelText={t('common.cancel') || '取消'}
                              >
                                <Button
                                  type='text'
                                  size='small'
                                  danger
                                  icon={<Trash2 className='w-3 h-3' />}
                                >
                                  {t('common.delete') || '删除'}
                                </Button>
                              </Popconfirm>
                            )}
                          </Space>
                        )}
                      </>
                    )}
                  </div>
                </div>
              </List.Item>
            )}
          />
        )}
      </Card>
    </div>
  );
};

