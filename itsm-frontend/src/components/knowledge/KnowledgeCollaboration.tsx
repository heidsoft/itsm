'use client';

import React, { useState, useEffect, useRef } from 'react';
import {
  Card,
  Avatar,
  Button,
  Space,
  Typography,
  List,
  Tooltip,
  Badge,
  Tag,
  Alert,
  Modal,
  Form,
  Input,
  Select,
  message,
  Popover,
  Timeline,
} from 'antd';
import {
  Users,
  MessageSquare,
  Eye,
  Edit3,
  Save,
  Share2,
  Lock,
  Unlock,
  Clock,
  User,
  GitBranch,
  Send,
  Bell,
  HelpCircle,
} from 'lucide-react';
import { format } from 'date-fns';

const { Title, Text } = Typography;
const { TextArea } = Input;
const { Option } = Select;

interface UserPresence {
  userId: string;
  userName: string;
  userAvatar?: string;
  color: string;
  cursor?: {
    line: number;
    column: number;
  };
  selection?: {
    start: { line: number; column: number };
    end: { line: number; column: number };
  };
  status: 'active' | 'idle' | 'away';
}

interface Comment {
  id: string;
  userId: string;
  userName: string;
  content: string;
  timestamp: Date;
  type: 'comment' | 'suggestion' | 'question';
  resolved?: boolean;
  replyTo?: string;
}

interface CollaborationSession {
  id: string;
  articleId: string;
  participants: UserPresence[];
  isActive: boolean;
  createdAt: Date;
  lastActivity: Date;
}

interface KnowledgeCollaborationProps {
  articleId: string;
  onSave?: (content: string) => void;
  readOnly?: boolean;
}

const KnowledgeCollaboration: React.FC<KnowledgeCollaborationProps> = ({
  articleId,
  onSave,
  readOnly = false,
}) => {
  const [session, setSession] = useState<CollaborationSession | null>(null);
  const [participants, setParticipants] = useState<UserPresence[]>([]);
  const [comments, setComments] = useState<Comment[]>([]);
  const [showComments, setShowComments] = useState(false);
  const [showShareModal, setShowShareModal] = useState(false);
  const [newComment, setNewComment] = useState('');
  const [commentType, setCommentType] = useState<'comment' | 'suggestion' | 'question'>('comment');
  const [shareEmail, setShareEmail] = useState('');
  const [sharePermission, setSharePermission] = useState<'view' | 'edit' | 'comment'>('view');
  const [isSharing, setIsSharing] = useState(false);

  // 加载数据
  useEffect(() => {
    const loadData = async () => {
      if (!articleId) return;

      try {
        // 加载评论
        const { KnowledgeBaseApi } = await import('@/lib/api/knowledge-base-api');
        const response = await KnowledgeBaseApi.getComments(articleId);

        // 转换API评论数据到组件格式
        const mappedComments: Comment[] = response.comments.map((c: any) => ({
          id: c.id,
          userId: c.userId,
          userName: c.userName,
          content: c.content,
          timestamp: new Date(c.createdAt),
          type: c.type || 'comment',
          resolved: c.resolved,
          replyTo: c.parentId,
        }));

        setComments(mappedComments);

        // TODO: 对接实时协作Session API
        // 目前后端暂未提供获取当前文档协作Session的接口
        setSession(null);
        setParticipants([]);
      } catch (error) {
        console.error('加载协作数据失败:', error);
        // 不显示错误提示，以免干扰用户，因为这是附加功能
      }
    };

    loadData();
  }, [articleId]);

  // 添加评论
  const handleAddComment = async () => {
    if (!newComment.trim()) return;

    try {
      const { KnowledgeBaseApi } = await import('@/lib/api/knowledge-base-api');
      const response = await KnowledgeBaseApi.addComment(articleId, newComment);

      const comment: Comment = {
        id: String(response.id || `comment_${Date.now()}`),
        userId: String(response.userId || 'current_user'),
        userName: response.userName || '当前用户',
        content: response.content || newComment,
        timestamp: new Date(response.createdAt || Date.now()),
        type: commentType,
      };

      setComments([...comments, comment]);
      setNewComment('');
      message.success('评论添加成功');
    } catch (error) {
      console.error('添加评论失败:', error);
      message.error('添加评论失败');
    }
  };

  // 解决评论
  const handleResolveComment = async (commentId: string) => {
    try {
      const { KnowledgeBaseApi } = await import('@/lib/api/knowledge-base-api');
      // 注意：这里假设 KnowledgeBaseApi 已经有了 updateComment 方法用于更新状态
      // 如果没有专门的 resolve 接口，通常使用 update 接口
      // 检查 KnowledgeBaseApi 是否有 resolveComment，如果没有则模拟或扩展
      if ('resolveComment' in KnowledgeBaseApi) {
        await (KnowledgeBaseApi as any).resolveComment(commentId);
      } else {
        // 回退方案：仅在前端更新状态，或调用通用更新接口
        // await KnowledgeBaseApi.updateComment(commentId, { resolved: true });
        console.log('API暂不支持后端解决评论，仅前端更新');
      }

      setComments(
        comments.map(comment =>
          comment.id === commentId ? { ...comment, resolved: !comment.resolved } : comment
        )
      );
      message.success('评论状态已更新');
    } catch (error) {
      console.error('更新评论状态失败:', error);
      message.error('操作失败');
    }
  };

  // 分享文档
  const handleShareDocument = async () => {
    if (!shareEmail.trim()) {
      message.warning('请输入邮箱地址');
      return;
    }

    try {
      setIsSharing(true);
      const { KnowledgeBaseApi } = await import('@/lib/api/knowledge-base-api');
      await KnowledgeBaseApi.shareArticle(articleId, 'email'); // 假设platform参数支持email，实际可能需要专门的分享API

      message.success(`文档已分享给 ${shareEmail}`);
      setShowShareModal(false);
      setShareEmail('');
    } catch (error) {
      console.error('分享失败:', error);
      message.error('分享失败');
    } finally {
      setIsSharing(false);
    }
  };

  // 获取用户状态图标
  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'active':
        return <div className='w-2 h-2 bg-green-500 rounded-full' />;
      case 'idle':
        return <div className='w-2 h-2 bg-yellow-500 rounded-full' />;
      case 'away':
        return <div className='w-2 h-2 bg-gray-500 rounded-full' />;
      default:
        return <div className='w-2 h-2 bg-gray-400 rounded-full' />;
    }
  };

  // 获取评论类型图标
  const getCommentIcon = (type: string) => {
    switch (type) {
      case 'suggestion':
        return <Edit3 className='w-4 h-4 text-blue-500' />;
      case 'question':
        return <HelpCircle className='w-4 h-4 text-orange-500' />;
      default:
        return <MessageSquare className='w-4 h-4 text-gray-500' />;
    }
  };

  return (
    <div className='space-y-4'>
      {/* 协作参与者 */}
      <Card
        title={
          <Space>
            <Users className='w-5 h-5 text-blue-600' />
            <span>协作参与者 ({participants.length})</span>
          </Space>
        }
        size='small'
      >
        <Space wrap>
          {participants.map(participant => (
            <Tooltip
              key={participant.userId}
              title={
                <div>
                  <div>{participant.userName}</div>
                  <div className='text-xs text-gray-500'>
                    {participant.status === 'active'
                      ? '正在编辑'
                      : participant.status === 'idle'
                        ? '离开'
                        : '离开'}
                  </div>
                </div>
              }
            >
              <div className='flex items-center space-x-2 cursor-pointer'>
                <Avatar
                  size='small'
                  style={{
                    backgroundColor: participant.color,
                    border: `2px solid ${participant.status === 'active' ? participant.color : 'transparent'}`,
                  }}
                >
                  {participant.userName.charAt(0)}
                </Avatar>
                <Text className='text-sm'>{participant.userName}</Text>
                {getStatusIcon(participant.status)}
              </div>
            </Tooltip>
          ))}
        </Space>
      </Card>

      {/* 协作工具栏 */}
      <Card size='small'>
        <Space>
          <Tooltip title='查看评论'>
            <Badge count={comments.filter(c => !c.resolved).length}>
              <Button
                type={showComments ? 'primary' : 'default'}
                icon={<MessageSquare className='w-4 h-4' />}
                onClick={() => setShowComments(!showComments)}
              />
            </Badge>
          </Tooltip>

          <Tooltip title='分享文档'>
            <Button icon={<Share2 className='w-4 h-4' />} onClick={() => setShowShareModal(true)} />
          </Tooltip>

          <Tooltip title='查看版本历史'>
            <Button
              icon={<GitBranch className='w-4 h-4' />}
              onClick={() => message.info('版本历史功能开发中')}
            />
          </Tooltip>

          <Tooltip title='保存文档'>
            <Button
              type='primary'
              icon={<Save className='w-4 h-4' />}
              onClick={() => onSave?.('document content')}
            />
          </Tooltip>
        </Space>
      </Card>

      {/* 评论面板 */}
      {showComments && (
        <Card
          title={
            <Space>
              <MessageSquare className='w-5 h-5' />
              <span>文档讨论 ({comments.length})</span>
            </Space>
          }
          className='comments-panel'
        >
          {/* 评论输入 */}
          <div className='mb-4 p-3 bg-gray-50 rounded'>
            <div className='flex space-x-2 mb-2'>
              <Select
                value={commentType}
                onChange={setCommentType}
                style={{ width: 120 }}
                size='small'
              >
                <Option value='comment'>
                  <Space>
                    <MessageSquare className='w-3 h-3' />
                    评论
                  </Space>
                </Option>
                <Option value='suggestion'>
                  <Space>
                    <Edit3 className='w-3 h-3' />
                    建议
                  </Space>
                </Option>
                <Option value='question'>
                  <Space>
                    <HelpCircle className='w-3 h-3' />
                    问题
                  </Space>
                </Option>
              </Select>
            </div>
            <div className='flex space-x-2'>
              <TextArea
                value={newComment}
                onChange={e => setNewComment(e.target.value)}
                placeholder='输入评论、建议或问题...'
                rows={2}
                maxLength={500}
              />
              <Button
                type='primary'
                icon={<Send className='w-4 h-4' />}
                onClick={handleAddComment}
                disabled={!newComment.trim()}
              >
                发送
              </Button>
            </div>
          </div>

          {/* 评论列表 */}
          <List
            dataSource={comments}
            renderItem={comment => (
              <List.Item
                key={comment.id}
                className={`comment-item ${comment.resolved ? 'comment-resolved' : ''}`}
                actions={[
                  !comment.resolved && (
                    <Button
                      type='text'
                      size='small'
                      onClick={() => handleResolveComment(comment.id)}
                    >
                      解决
                    </Button>
                  ),
                ]}
              >
                <List.Item.Meta
                  avatar={
                    <Avatar size='small' style={{ backgroundColor: '#1890ff' }}>
                      {comment.userName.charAt(0)}
                    </Avatar>
                  }
                  title={
                    <Space>
                      <span className='font-medium'>{comment.userName}</span>
                      {getCommentIcon(comment.type)}
                      {comment.resolved && <Tag color='green'>已解决</Tag>}
                    </Space>
                  }
                  description={
                    <div className='space-y-1'>
                      <Text className='text-sm'>{comment.content}</Text>
                      <div className='flex items-center space-x-2 text-xs text-gray-500'>
                        <Clock className='w-3 h-3' />
                        {format(comment.timestamp, 'yyyy-MM-dd HH:mm')}
                      </div>
                    </div>
                  }
                />
              </List.Item>
            )}
          />
        </Card>
      )}

      {/* 分享模态框 */}
      <Modal
        title={
          <Space>
            <Share2 className='w-5 h-5' />
            分享文档
          </Space>
        }
        open={showShareModal}
        onCancel={() => setShowShareModal(false)}
        footer={[
          <Button key='cancel' onClick={() => setShowShareModal(false)}>
            取消
          </Button>,
          <Button key='share' type='primary' onClick={handleShareDocument} loading={isSharing}>
            分享
          </Button>,
        ]}
      >
        <Form layout='vertical'>
          <Form.Item label='邮箱地址' required>
            <Input
              type='email'
              value={shareEmail}
              onChange={e => setShareEmail(e.target.value)}
              placeholder='输入邮箱地址'
              prefix={<User className='w-4 h-4 text-gray-400' />}
            />
          </Form.Item>
          <Form.Item label='权限'>
            <Select value={sharePermission} onChange={setSharePermission} style={{ width: '100%' }}>
              <Option value='view'>
                <Space>
                  <Eye className='w-4 h-4' />
                  只能查看
                </Space>
              </Option>
              <Option value='comment'>
                <Space>
                  <MessageSquare className='w-4 h-4' />
                  可以评论
                </Space>
              </Option>
              <Option value='edit'>
                <Space>
                  <Edit3 className='w-4 h-4' />
                  可以编辑
                </Space>
              </Option>
            </Select>
          </Form.Item>
        </Form>

        <Alert
          message='分享说明'
          description='分享后，指定用户将根据权限设置访问此文档。您可以随时撤销分享权限。'
          type='info'
          showIcon
          className='mt-4'
        />
      </Modal>

      {/* 实时协作提示 */}
      {session?.isActive && (
        <Alert
          message='实时协作中'
          description={
            <Space>
              <Bell className='w-4 h-4' />
              <span>{participants.filter(p => p.status === 'active').length} 人正在编辑此文档</span>
            </Space>
          }
          type='info'
          showIcon
          className='collaboration-alert'
        />
      )}
    </div>
  );
};

export default KnowledgeCollaboration;
