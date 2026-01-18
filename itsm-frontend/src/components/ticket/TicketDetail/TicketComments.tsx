'use client';

import React, { useState } from 'react';
import {
  Send,
  Edit,
  Trash2,
  MessageSquare,
  AtSign,
  User,
} from 'lucide-react';
import {
  Card,
  Typography,
  Button,
  Input,
  Avatar,
  Tag as AntTag,
  Modal,
} from 'antd';
import { UserSelect } from '@/components/common/UserSelect';
import { TicketApi } from '@/lib/api/ticket-api';

const { Text, Paragraph } = Typography;
const { TextArea } = Input;

interface Comment {
  id: number;
  ticket_id: number;
  user_id: number;
  content: string;
  is_internal: boolean;
  mentions: number[];
  attachments: number[];
  user?: {
    id: number;
    username: string;
    name: string;
    email: string;
  };
  created_at: string;
  updated_at: string;
}

interface TicketCommentsProps {
  ticketId: number;
  comments: Comment[];
  onRefresh: () => void;
  formatDateTime: (dateString: string) => string;
}

export const TicketComments: React.FC<TicketCommentsProps> = ({
  ticketId,
  comments,
  onRefresh,
  formatDateTime,
}) => {
  const [newComment, setNewComment] = useState('');
  const [isInternal, setIsInternal] = useState(false);
  const [mentionedUsers, setMentionedUsers] = useState<number[]>([]);
  const [editingCommentId, setEditingCommentId] = useState<number | null>(null);
  const [editingCommentContent, setEditingCommentContent] = useState('');

  const handleAddComment = async () => {
    if (!newComment.trim()) return;

    try {
      await TicketApi.addTicketComment(ticketId, {
        content: newComment,
        is_internal: isInternal,
        mentions: mentionedUsers,
      });
      setNewComment('');
      setMentionedUsers([]);
      setIsInternal(false);
      onRefresh();
    } catch (error) {
      console.error('添加评论失败:', error);
    }
  };

  const handleEditComment = async (commentId: number) => {
    if (!editingCommentContent.trim()) return;

    try {
      await TicketApi.updateTicketComment(ticketId, commentId, {
        content: editingCommentContent,
      });
      setEditingCommentId(null);
      setEditingCommentContent('');
      onRefresh();
    } catch (error) {
      console.error('更新评论失败:', error);
    }
  };

  const handleDeleteComment = async (commentId: number) => {
    try {
      await TicketApi.deleteTicketComment(ticketId, commentId);
      onRefresh();
    } catch (error) {
      console.error('删除评论失败:', error);
    }
  };

  const startEditComment = (comment: Comment) => {
    setEditingCommentId(comment.id);
    setEditingCommentContent(comment.content);
  };

  const cancelEdit = () => {
    setEditingCommentId(null);
    setEditingCommentContent('');
  };

  return (
    <div className='p-6'>
      {/* 添加评论 */}
      <div className='mb-6'>
        <Card title='添加评论' className='shadow-sm'>
          <div className='space-y-4'>
            <div className='flex items-center space-x-4'>
              <div className='flex items-center space-x-2'>
                <input
                  type='checkbox'
                  id='internal'
                  checked={isInternal}
                  onChange={(e) => setIsInternal(e.target.checked)}
                />
                <label htmlFor='internal' className='text-sm text-gray-600'>
                  仅内部可见
                </label>
              </div>
            </div>
            <div>
              <div className='mb-2'>
                <Text type='secondary' className='text-sm'>
                  <AtSign className='w-4 h-4 inline mr-1' />
                  @用户（可选）
                </Text>
              </div>
              <UserSelect
                value={mentionedUsers}
                onChange={setMentionedUsers}
                mode='multiple'
                placeholder='选择要@的用户'
                style={{ width: '100%' }}
              />
            </div>
            <TextArea
              value={newComment}
              onChange={(e) => setNewComment(e.target.value)}
              placeholder='输入您的评论...'
              rows={4}
            />
            <div className='flex justify-end'>
              <Button
                type='primary'
                icon={<Send />}
                onClick={handleAddComment}
                disabled={!newComment.trim()}
              >
                发送评论
              </Button>
            </div>
          </div>
        </Card>
      </div>

      {/* 评论列表 */}
      <div className='space-y-4'>
        {comments.map((comment) => (
          <Card key={comment.id} className='shadow-sm'>
            {editingCommentId === comment.id ? (
              <div className='space-y-3'>
                <TextArea
                  value={editingCommentContent}
                  onChange={(e) => setEditingCommentContent(e.target.value)}
                  rows={3}
                />
                <div className='flex justify-end space-x-2'>
                  <Button onClick={cancelEdit}>取消</Button>
                  <Button
                    type='primary'
                    onClick={() => handleEditComment(comment.id)}
                    disabled={!editingCommentContent.trim()}
                  >
                    保存
                  </Button>
                </div>
              </div>
            ) : (
              <div className='flex items-start space-x-3'>
                <Avatar size='small' icon={<User />}>
                  {comment.user?.name?.[0] || comment.user?.username?.[0]}
                </Avatar>
                <div className='flex-1'>
                  <div className='flex items-center space-x-2 mb-2'>
                    <Text strong>
                      {comment.user?.name || comment.user?.username || '未知用户'}
                    </Text>
                    {comment.is_internal && (
                      <AntTag color='orange'>仅内部可见</AntTag>
                    )}
                    {comment.mentions && comment.mentions.length > 0 && (
                      <AntTag color='blue' icon={<AtSign className='w-3 h-3' />}>
                        @{comment.mentions.length}人
                      </AntTag>
                    )}
                    <Text type='secondary' className='text-sm'>
                      {formatDateTime(comment.created_at)}
                    </Text>
                    {comment.updated_at !== comment.created_at && (
                      <Text type='secondary' className='text-xs'>
                        （已编辑）
                      </Text>
                    )}
                  </div>
                  <Paragraph className='mb-2 whitespace-pre-wrap'>
                    {comment.content}
                  </Paragraph>
                  <div className='flex items-center space-x-2'>
                    <Button
                      type='link'
                      size='small'
                      icon={<Edit className='w-3 h-3' />}
                      onClick={() => startEditComment(comment)}
                    >
                      编辑
                    </Button>
                    <Button
                      type='link'
                      size='small'
                      danger
                      icon={<Trash2 className='w-3 h-3' />}
                      onClick={() => {
                        Modal.confirm({
                          title: '确认删除',
                          content: '确定要删除这条评论吗？',
                          onOk: () => handleDeleteComment(comment.id),
                        });
                      }}
                    >
                      删除
                    </Button>
                  </div>
                </div>
              </div>
            )}
          </Card>
        ))}

        {comments.length === 0 && (
          <div className='text-center py-8 text-gray-500'>
            <MessageSquare className='w-16 h-16 mx-auto mb-4 text-gray-300' />
            <Text>暂无评论</Text>
          </div>
        )}
      </div>
    </div>
  );
};
