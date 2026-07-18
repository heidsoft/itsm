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
  /**
   * 是否显示"仅内部可见"开关。默认 true。
   */
  showInternalToggle?: boolean;
  /**
   * 是否支持 @提及。默认 true。
   */
  showMentions?: boolean;
  /**
   * 当前用户 id，用于判断哪些评论可以编辑/删除
   */
  currentUserId?: number;
  /**
   * 自定义时间格式化，如未提供则使用 toLocaleString
   */
  formatDateTime?: (dateString: string) => string;
}

const defaultFormat = (s: string) => (s ? new Date(s).toLocaleString('zh-CN') : '');

export const CommentPanel: React.FC<CommentPanelProps> = ({
  targetId,
  adapter,
  showInternalToggle = true,
  showMentions = true,
  currentUserId,
  formatDateTime = defaultFormat,
}) => {
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

  const fetchComments = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const { comments } = await adapter.list(targetId);
      setComments(comments || []);
    } catch (e) {
      const msg = e instanceof Error ? e.message : '加载评论失败';
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, [adapter, targetId]);

  useEffect(() => {
    void fetchComments();
  }, [fetchComments]);

  const handleAddComment = async () => {
    if (!newComment.trim()) return;
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
      message.success('评论已发布');
    } catch (e) {
      const msg = e instanceof Error ? e.message : '添加评论失败';
      message.error(msg);
    } finally {
      setSubmitting(false);
    }
  };

  const handleEditComment = async (commentId: number) => {
    if (!editingCommentContent.trim() || !adapter.update) return;
    setSubmitting(true);
    try {
      await adapter.update(targetId, commentId, {
        content: editingCommentContent,
      });
      setEditingCommentId(null);
      setEditingCommentContent('');
      await fetchComments();
    } catch (e) {
      const msg = e instanceof Error ? e.message : '更新评论失败';
      message.error(msg);
    } finally {
      setSubmitting(false);
    }
  };

  const handleDeleteComment = async (commentId: number) => {
    try {
      await adapter.remove(targetId, commentId);
      await fetchComments();
    } catch (e) {
      const msg = e instanceof Error ? e.message : '删除评论失败';
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
      {/* 错误提示 */}
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
              重试
            </Button>
          }
        />
      )}

      {/* 添加评论 */}
      <div className="mb-6">
        <Card title="添加评论" className="shadow-sm">
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
                    仅内部可见
                  </label>
                </div>
              </div>
            )}
            {showMentions && (
              <div>
                <div className="mb-2">
                  <Text type="secondary" className="text-sm">
                    <AtSign className="w-4 h-4 inline mr-1" />
                    @用户（可选）
                  </Text>
                </div>
                <UserSelect
                  value={mentionedUsers}
                  onChange={setMentionedUsers}
                  mode="multiple"
                  placeholder="选择要@的用户"
                  style={{ width: '100%' }}
                />
              </div>
            )}
            <TextArea
              value={newComment}
              onChange={(e) => setNewComment(e.target.value)}
              placeholder="输入您的评论..."
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
                发送评论
              </Button>
            </div>
          </div>
        </Card>
      </div>

      {/* 评论列表 */}
      <div className="space-y-4">
        {comments.map((comment) => (
          <Card key={comment.id} className="shadow-sm">
            {editingCommentId === comment.id ? (
              <div className="space-y-3">
                <TextArea
                  value={editingCommentContent}
                  onChange={(e) => setEditingCommentContent(e.target.value)}
                  rows={3}
                  placeholder="输入编辑后的评论..."
                />
                <div className="flex justify-end space-x-2">
                  <Button onClick={cancelEdit}>取消</Button>
                  <Button
                    type="primary"
                    onClick={() => handleEditComment(comment.id)}
                    disabled={!editingCommentContent.trim()}
                    loading={submitting}
                  >
                    保存
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
                      {comment.user?.name || comment.user?.username || '未知用户'}
                    </Text>
                    {comment.isInternal && <AntTag color="orange">仅内部可见</AntTag>}
                    {comment.mentions && comment.mentions.length > 0 && (
                      <AntTag color="blue" icon={<AtSign className="w-3 h-3" />}>
                        @{comment.mentions.length}人
                      </AntTag>
                    )}
                    <Text type="secondary" className="text-sm">
                      {formatDateTime(comment.createdAt)}
                    </Text>
                    {comment.updatedAt && comment.updatedAt !== comment.createdAt && (
                      <Text type="secondary" className="text-xs">
                        （已编辑）
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
                        编辑
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
                            title: '确认删除',
                            content: '确定要删除这条评论吗？',
                            okText: '删除',
                            okType: 'danger',
                            cancelText: '取消',
                            onOk: () => handleDeleteComment(comment.id),
                          });
                        }}
                      >
                        删除
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
            <Text>暂无评论</Text>
          </div>
        )}
      </div>
    </div>
  );
};

export default CommentPanel;
