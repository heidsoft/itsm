'use client';

import {
  FileText,
  Save,
  Download,
  Upload,
  X,
  Send,
  History,
  AlertTriangle,
  Edit,
  User,
  Calendar,
  Clock,
  CheckCircle,
  XCircle,
  Trash2,
  MessageSquare,
  AtSign,
  Eye,
  Image,
  File,
  Bell,
  Star,
  Zap,
} from 'lucide-react';

import React, { useState, useEffect } from 'react';
import {
  Button,
  Card,
  Space,
  Typography,
  App,
  Badge,
  Tag as AntTag,
  Upload as AntUpload,
  Modal,
  Input,
  Select,
  Timeline,
  Avatar,
  List,
  Tabs,
  Tag,
  Progress,
  Image as AntImage,
  Rate,
  Form,
  Divider,
} from 'antd';
import { TicketApi } from '@/lib/api/ticket-api';
import { TicketNotificationApi, TicketNotification } from '@/lib/api/ticket-notification-api';
import { TicketRatingApi, TicketRating } from '@/lib/api/ticket-rating-api';
import { Ticket, Attachment, Comment, WorkflowStep, SLAInfo } from '../lib/api-config';
import { UserSelect } from '@/components/common/UserSelect';
import { useAuthStore } from '@/lib/store/auth-store';
import { SmartAssignmentModal } from './SmartAssignmentModal';
import { TicketAttachmentSection } from './TicketAttachmentSection';
import { TicketNotificationSection } from './TicketNotificationSection';
import { TicketRatingSection } from './TicketRatingSection';

const { Title, Text, Paragraph } = Typography;
const { TextArea } = Input;
const { TabPane } = Tabs;

interface TicketDetailProps {
  ticket: Ticket;
  onApprove?: () => void;
  onReject?: () => void;
  onAssign?: (assignee: string) => void;
  onUpdate?: (updates: unknown) => void;
  canApprove?: boolean;
  canEdit?: boolean;
}

// 获取优先级配置
const getPriorityConfig = (priority: string) => {
  const configs: Record<
    string,
    { color: string; bgColor: string; textColor: string; borderColor: string }
  > = {
    紧急: {
      color: 'red',
      bgColor: 'bg-red-100',
      textColor: 'text-red-800',
      borderColor: 'border-red-300',
    },
    高: {
      color: 'orange',
      bgColor: 'bg-orange-100',
      textColor: 'text-orange-800',
      borderColor: 'border-orange-300',
    },
    中: {
      color: 'yellow',
      bgColor: 'bg-yellow-100',
      textColor: 'text-yellow-800',
      borderColor: 'border-yellow-300',
    },
    低: {
      color: 'blue',
      bgColor: 'bg-blue-100',
      textColor: 'text-blue-800',
      borderColor: 'border-blue-300',
    },
  };
  return configs[priority] || configs['中'];
};

// 获取状态配置
const getStatusConfig = (status: string) => {
  const configs: Record<string, { bgColor: string; textColor: string }> = {
    处理中: { bgColor: 'bg-blue-100', textColor: 'text-blue-800' },
    已分配: { bgColor: 'bg-purple-100', textColor: 'text-purple-800' },
    已解决: { bgColor: 'bg-green-100', textColor: 'text-green-800' },
    已关闭: { bgColor: 'bg-gray-200', textColor: 'text-gray-800' },
    待审批: { bgColor: 'bg-yellow-100', textColor: 'text-yellow-800' },
    已批准: { bgColor: 'bg-green-100', textColor: 'text-green-800' },
  };
  return configs[status] || { bgColor: 'bg-gray-100', textColor: 'text-gray-800' };
};

// 获取类型图标
const getTypeIcon = (type: string) => {
  const icons: Record<string, React.ComponentType<any>> = {
    Incident: AlertTriangle,
    Problem: AlertTriangle,
    Change: Edit,
    'Service Request': FileText,
  };
  return icons[type] || FileText;
};

// 格式化文件大小
const formatFileSize = (bytes: number) => {
  if (bytes === 0) return '0 Bytes';
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};

// 格式化时间
const formatDateTime = (dateString: string) => {
  return new Date(dateString).toLocaleString('zh-CN');
};

export const TicketDetail: React.FC<TicketDetailProps> = ({
  ticket,
  onApprove,
  onReject,
  onAssign,
  onUpdate,
  canApprove = false,
  canEdit = false,
}) => {
  const { message: antMessage } = App.useApp();
  const [isEditing, setIsEditing] = useState(false);
  const [editedTicket, setEditedTicket] = useState<Ticket>(ticket);
  const [assigneeInput, setAssigneeInput] = useState('');
  const [newComment, setNewComment] = useState('');
  const [isInternal, setIsInternal] = useState(false);
  const [mentionedUsers, setMentionedUsers] = useState<number[]>([]);
  const [editingCommentId, setEditingCommentId] = useState<number | null>(null);
  const [editingCommentContent, setEditingCommentContent] = useState('');
  const [showUploadModal, setShowUploadModal] = useState(false);
  const [uploadingFiles, setUploadingFiles] = useState<File[]>([]);
  const [uploadProgress, setUploadProgress] = useState<Record<string, number>>({});
  const [previewAttachment, setPreviewAttachment] = useState<{
    id: number;
    file_name: string;
    file_url: string;
    mime_type: string;
  } | null>(null);
  const [activeTab, setActiveTab] = useState('overview');

  // 状态管理
  const [attachments, setAttachments] = useState<
    Array<{
      id: number;
      ticket_id: number;
      file_name: string;
      file_path: string;
      file_url: string;
      file_size: number;
      file_type: string;
      mime_type: string;
      uploaded_by: number;
      uploader?: {
        id: number;
        username: string;
        name: string;
        email: string;
      };
      created_at: string;
    }>
  >([]);
  const [comments, setComments] = useState<
    Array<{
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
    }>
  >([]);
  const [workflowSteps, setWorkflowSteps] = useState<WorkflowStep[]>([]);
  const [slaInfo, setSlaInfo] = useState<SLAInfo | null>(null);
  const [ticketHistory, setTicketHistory] = useState<unknown[]>([]);
  const [notifications, setNotifications] = useState<TicketNotification[]>([]);
  const [rating, setRating] = useState<TicketRating | null>(null);
  const { user } = useAuthStore();
  const [loading, setLoading] = useState(false);
  const [showSmartAssignModal, setShowSmartAssignModal] = useState(false);

  // 获取工单相关数据
  useEffect(() => {
    fetchTicketData();
  }, [ticket.id]);

  const fetchTicketData = async () => {
    setLoading(true);
    try {
      // 并行获取所有数据
      const [attachmentsRes, commentsRes, workflowRes, slaRes, historyRes, notificationsRes] =
        await Promise.allSettled([
          TicketApi.getTicketAttachments(ticket.id),
          TicketApi.getTicketComments(ticket.id),
          TicketApi.getTicketWorkflow(ticket.id),
          TicketApi.getTicketSLA(ticket.id),
          TicketApi.getTicketHistory(ticket.id),
          TicketNotificationApi.getTicketNotifications(ticket.id),
        ]);

      if (attachmentsRes.status === 'fulfilled') {
        const attachmentsData = attachmentsRes.value as {
          attachments?: unknown[];
          data?: unknown[];
        };
        setAttachments(
          (attachmentsData.attachments || attachmentsData.data || []) as typeof attachments
        );
      }
      if (commentsRes.status === 'fulfilled') {
        const commentsData = commentsRes.value as { comments?: unknown[]; data?: unknown[] };
        setComments((commentsData.comments || commentsData.data || []) as typeof comments);
      }
      if (workflowRes.status === 'fulfilled') {
        setWorkflowSteps((workflowRes.value as WorkflowStep[]) || []);
      }
      if (slaRes.status === 'fulfilled') {
        setSlaInfo((slaRes.value as SLAInfo) || null);
      }
      if (historyRes.status === 'fulfilled') {
        setTicketHistory(historyRes.value || []);
      }
      if (notificationsRes.status === 'fulfilled') {
        const notificationsData = notificationsRes.value as {
          notifications?: TicketNotification[];
          data?: TicketNotification[];
        };
        setNotifications(
          (notificationsData.notifications || notificationsData.data || []) as TicketNotification[]
        );
      }

      // 获取评分
      const ratingRes = await TicketRatingApi.getRating(ticket.id).catch(() => null);
      if (ratingRes) {
        setRating(ratingRes);
      }
    } catch (error) {
      console.error('获取工单数据失败:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleSave = () => {
    onUpdate?.(editedTicket);
    setIsEditing(false);
  };

  const handleCancel = () => {
    setEditedTicket(ticket);
    setIsEditing(false);
  };

  const handleAssign = () => {
    if (assigneeInput.trim()) {
      onAssign?.(assigneeInput.trim());
      setAssigneeInput('');
    }
  };

  // 添加评论
  const handleAddComment = async () => {
    if (!newComment.trim()) return;

    try {
      await TicketApi.addTicketComment(ticket.id, {
        content: newComment,
        is_internal: isInternal,
        mentions: mentionedUsers,
      });

      antMessage.success('评论添加成功');
      setNewComment('');
      setMentionedUsers([]);
      setIsInternal(false);
      fetchTicketData(); // 刷新评论列表
    } catch (error) {
      antMessage.error('添加评论失败');
    }
  };

  // 编辑评论
  const handleEditComment = async (commentId: number) => {
    if (!editingCommentContent.trim()) return;

    try {
      await TicketApi.updateTicketComment(ticket.id, commentId, {
        content: editingCommentContent,
      });

      antMessage.success('评论更新成功');
      setEditingCommentId(null);
      setEditingCommentContent('');
      fetchTicketData(); // 刷新评论列表
    } catch (error) {
      antMessage.error('更新评论失败');
    }
  };

  // 删除评论
  const handleDeleteComment = async (commentId: number) => {
    try {
      await TicketApi.deleteTicketComment(ticket.id, commentId);
      antMessage.success('评论删除成功');
      fetchTicketData(); // 刷新评论列表
    } catch (error) {
      antMessage.error('删除评论失败');
    }
  };

  // 开始编辑评论
  const startEditComment = (comment: (typeof comments)[0]) => {
    setEditingCommentId(comment.id);
    setEditingCommentContent(comment.content);
  };

  // 取消编辑
  const cancelEdit = () => {
    setEditingCommentId(null);
    setEditingCommentContent('');
  };

  // 上传附件
  const handleUploadAttachment = async () => {
    if (uploadingFiles.length === 0) return;

    try {
      // 并行上传所有文件
      const uploadPromises = uploadingFiles.map(file => {
        const fileId = `${file.name}-${file.size}`;
        return TicketApi.uploadTicketAttachment(ticket.id, file, progress => {
          setUploadProgress(prev => ({ ...prev, [fileId]: progress }));
        });
      });

      await Promise.all(uploadPromises);
      antMessage.success(`成功上传 ${uploadingFiles.length} 个附件`);
      setShowUploadModal(false);
      setUploadingFiles([]);
      setUploadProgress({});
      fetchTicketData(); // 刷新附件列表
    } catch (error) {
      antMessage.error('附件上传失败');
      console.error('Upload error:', error);
    }
  };

  // 处理文件选择
  const handleFileSelect = (fileList: File[]) => {
    setUploadingFiles(fileList);
  };

  // 预览附件
  const handlePreviewAttachment = (attachment: (typeof attachments)[0]) => {
    // 检查是否为图片
    const isImage =
      attachment.mime_type?.startsWith('image/') ||
      ['jpg', 'jpeg', 'png', 'gif', 'webp'].some(ext =>
        attachment.file_name.toLowerCase().endsWith(`.${ext}`)
      );

    if (isImage) {
      setPreviewAttachment({
        id: attachment.id,
        file_name: attachment.file_name,
        file_url: TicketApi.getAttachmentPreviewUrl(ticket.id, attachment.id),
        mime_type: attachment.mime_type || attachment.file_type,
      });
    } else {
      // 对于非图片文件，在新窗口打开预览
      window.open(TicketApi.getAttachmentPreviewUrl(ticket.id, attachment.id), '_blank');
    }
  };

  // 下载附件
  const handleDownloadAttachment = (attachment: (typeof attachments)[0]) => {
    const url = TicketApi.getAttachmentDownloadUrl(ticket.id, attachment.id);
    const link = document.createElement('a');
    link.href = url;
    link.download = attachment.file_name;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  // 删除附件
  const handleDeleteAttachment = async (attachmentId: number) => {
    try {
      await TicketApi.deleteTicketAttachment(ticket.id, attachmentId);
      antMessage.success('附件删除成功');
      fetchTicketData(); // 刷新附件列表
    } catch (error) {
      antMessage.error('附件删除失败');
    }
  };

  // 智能分配成功回调
  const handleSmartAssignSuccess = async (userId: number) => {
    try {
      // 调用分配API
      await TicketApi.assignTicket(ticket.id, { assignee_id: userId });
      antMessage.success('工单分配成功');
      // 刷新工单数据
      if (onUpdate) {
        await onUpdate({ assignee_id: userId });
      }
      fetchTicketData();
    } catch (error) {
      console.error('Failed to assign ticket:', error);
      antMessage.error('分配失败');
    }
  };

  const TypeIcon = getTypeIcon(ticket.type || 'Service Request');
  const priorityConfig = getPriorityConfig(ticket.priority || '中');
  const statusConfig = getStatusConfig(ticket.status || '待处理');

  return (
    <div className='max-w-7xl mx-auto bg-gray-50 min-h-screen'>
      {/* 固定顶部状态栏 */}
      <div className='sticky top-0 z-10 bg-white shadow-sm border-b mb-6'>
        <div className='px-6 py-4'>
          <div className='flex items-start justify-between mb-4'>
            <div className='flex items-center space-x-3'>
              <TypeIcon className='w-8 h-8 text-blue-600' />
              <div>
                <Title level={2} className='mb-1'>
                  {ticket.title}
                </Title>
                <Text type='secondary'>
                  {ticket.type || 'Service Request'} #{ticket.ticket_number || ticket.id}
                </Text>
              </div>
            </div>
            <div className='flex items-center space-x-3'>
              {canEdit && (
                <Button
                  icon={isEditing ? <X /> : <Edit />}
                  onClick={() => setIsEditing(!isEditing)}
                >
                  {isEditing ? '取消' : '编辑'}
                </Button>
              )}
              {isEditing && (
                <Button type='primary' icon={<Save />} onClick={handleSave}>
                  保存
                </Button>
              )}
            </div>
          </div>

          {/* 关键信息：状态、优先级、处理人 */}
          <div className='flex items-center flex-wrap gap-4 mb-4'>
            <Badge
              status={
                ticket.status === 'open'
                  ? 'processing'
                  : ticket.status === 'closed'
                  ? 'success'
                  : 'warning'
              }
              text={<Text strong>{ticket.status || '待处理'}</Text>}
            />
            <AntTag color={priorityConfig.color} style={{ fontSize: '14px', padding: '4px 12px' }}>
              优先级: {ticket.priority || '中'}
            </AntTag>
            <div className='flex items-center space-x-2'>
              <User className='w-4 h-4 text-gray-500' />
              <Text>
                <Text type='secondary' className='text-sm'>
                  处理人：
                </Text>
                <Text strong>{ticket.assignee?.name || '未分配'}</Text>
              </Text>
            </div>
            {ticket.category && <AntTag color='blue'>{ticket.category}</AntTag>}
            {ticket.tags &&
              ticket.tags.slice(0, 3).map((tag, index) => (
                <AntTag key={index} color='green'>
                  {tag}
                </AntTag>
              ))}
            {ticket.tags && ticket.tags.length > 3 && <AntTag>+{ticket.tags.length - 3}</AntTag>}
          </div>

          {/* 次要信息：快速查看 */}
          <div className='grid grid-cols-2 md:grid-cols-4 gap-4 pt-3 border-t border-gray-100'>
            <div>
              <Text type='secondary' className='text-xs'>
                报告人
              </Text>
              <div className='flex items-center space-x-1 mt-1'>
                <User className='w-3 h-3 text-gray-400' />
                <Text className='text-sm'>{ticket.requester?.name || '未知'}</Text>
              </div>
            </div>
            <div>
              <Text type='secondary' className='text-xs'>
                创建时间
              </Text>
              <div className='flex items-center space-x-1 mt-1'>
                <Calendar className='w-3 h-3 text-gray-400' />
                <Text className='text-sm'>{formatDateTime(ticket.created_at)}</Text>
              </div>
            </div>
            <div>
              <Text type='secondary' className='text-xs'>
                最后更新
              </Text>
              <div className='flex items-center space-x-1 mt-1'>
                <Clock className='w-3 h-3 text-gray-400' />
                <Text className='text-sm'>{formatDateTime(ticket.updated_at)}</Text>
              </div>
            </div>
            <div>
              <Text type='secondary' className='text-xs'>
                工单类型
              </Text>
              <div className='mt-1'>
                <Text className='text-sm'>{ticket.type || 'Service Request'}</Text>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* 主要内容区域 */}
      <div className='px-6'>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          className='bg-white rounded-lg shadow-sm'
          type='card'
          items={[
            {
              key: 'overview',
              label: '基本信息',
              children: (
                <div className='p-6'>
                  <div className='grid grid-cols-1 lg:grid-cols-3 gap-6'>
                    {/* 左侧：详细信息 */}
                    <div className='lg:col-span-2 space-y-6'>
                      {/* 问题描述 */}
                      <Card title='问题描述' className='shadow-sm'>
                        {isEditing ? (
                          <TextArea
                            value={editedTicket.description}
                            onChange={e =>
                              setEditedTicket({
                                ...editedTicket,
                                description: e.target.value,
                              })
                            }
                            rows={4}
                          />
                        ) : (
                          <Paragraph className='whitespace-pre-wrap'>
                            {ticket.description}
                          </Paragraph>
                        )}
                      </Card>

                      {/* 解决方案 */}
                      {ticket.resolution && (
                        <Card title='解决方案' className='shadow-sm'>
                          {isEditing ? (
                            <TextArea
                              value={editedTicket.resolution}
                              onChange={e =>
                                setEditedTicket({
                                  ...editedTicket,
                                  resolution: e.target.value,
                                })
                              }
                              rows={3}
                            />
                          ) : (
                            <Paragraph className='whitespace-pre-wrap'>
                              {ticket.resolution}
                            </Paragraph>
                          )}
                        </Card>
                      )}

                      {/* 工作流程图 */}
                      {workflowSteps.length > 0 && (
                        <Card title='处理流程' className='shadow-sm'>
                          <Timeline>
                            {workflowSteps.map((step, index) => (
                              <Timeline.Item
                                key={step.id}
                                color={
                                  step.status === 'completed'
                                    ? 'green'
                                    : step.status === 'in_progress'
                                    ? 'blue'
                                    : 'gray'
                                }
                                dot={
                                  step.status === 'completed' ? (
                                    <CheckCircle className='w-4 h-4' />
                                  ) : step.status === 'in_progress' ? (
                                    <Clock className='w-4 h-4' />
                                  ) : undefined
                                }
                              >
                                <div className='flex items-center justify-between'>
                                  <div>
                                    <Text strong>{step.step_name}</Text>
                                    {step.assignee && (
                                      <div className='text-sm text-gray-500'>
                                        负责人: {step.assignee.name}
                                      </div>
                                    )}
                                    {step.comments && (
                                      <div className='text-sm text-gray-600 mt-1'>
                                        {step.comments}
                                      </div>
                                    )}
                                  </div>
                                  <div className='text-right'>
                                    <Badge
                                      status={
                                        step.status === 'completed'
                                          ? 'success'
                                          : step.status === 'in_progress'
                                          ? 'processing'
                                          : 'default'
                                      }
                                      text={
                                        step.status === 'completed'
                                          ? '已完成'
                                          : step.status === 'in_progress'
                                          ? '进行中'
                                          : '待处理'
                                      }
                                    />
                                    {step.completed_at && (
                                      <div className='text-xs text-gray-500 mt-1'>
                                        {formatDateTime(step.completed_at)}
                                      </div>
                                    )}
                                  </div>
                                </div>
                              </Timeline.Item>
                            ))}
                          </Timeline>
                        </Card>
                      )}
                    </div>

                    {/* 右侧：操作和SLA */}
                    <div className='space-y-6'>
                      {/* 审批操作 */}
                      {canApprove && ticket.status === '待审批' && (
                        <Card title='审批操作' className='shadow-sm'>
                          <Space direction='vertical' className='w-full'>
                            <Button
                              type='primary'
                              icon={<CheckCircle />}
                              onClick={onApprove}
                              className='w-full'
                            >
                              批准
                            </Button>
                            <Button danger icon={<XCircle />} onClick={onReject} className='w-full'>
                              拒绝
                            </Button>
                          </Space>
                        </Card>
                      )}

                      {/* 分配操作 */}
                      <Card title='分配工单' className='shadow-sm'>
                        <Space direction='vertical' className='w-full' style={{ width: '100%' }}>
                          <Button
                            type='primary'
                            icon={<Zap />}
                            onClick={() => setShowSmartAssignModal(true)}
                            className='w-full'
                            style={{ marginBottom: 8 }}
                          >
                            智能分配
                          </Button>
                          <Divider style={{ margin: '8px 0' }}>或手动分配</Divider>
                          <Input
                            value={assigneeInput}
                            onChange={e => setAssigneeInput(e.target.value)}
                            placeholder='输入负责人姓名'
                          />
                          <Button
                            onClick={handleAssign}
                            disabled={!assigneeInput.trim()}
                            className='w-full'
                          >
                            手动分配
                          </Button>
                        </Space>
                      </Card>

                      {/* 评分信息 */}
                      <TicketRatingSection
                        ticketId={ticket.id}
                        ticketStatus={ticket.status}
                        requesterId={ticket.requester_id}
                        canRate={canEdit}
                        onRatingSubmitted={newRating => {
                          setRating(newRating);
                          fetchTicketData(); // 刷新数据
                        }}
                      />

                      {/* SLA信息 */}
                      {slaInfo && (
                        <Card title='SLA信息' className='shadow-sm'>
                          <div className='space-y-3'>
                            <div>
                              <Text type='secondary' className='text-sm'>
                                SLA名称
                              </Text>
                              <div className='font-medium'>{slaInfo.sla_name}</div>
                            </div>
                            <div>
                              <Text type='secondary' className='text-sm'>
                                响应时间
                              </Text>
                              <div className='font-medium'>{slaInfo.response_time} 分钟</div>
                            </div>
                            <div>
                              <Text type='secondary' className='text-sm'>
                                解决时间
                              </Text>
                              <div className='font-medium'>{slaInfo.resolution_time} 分钟</div>
                            </div>
                            <div>
                              <Text type='secondary' className='text-sm'>
                                到期时间
                              </Text>
                              <div className='font-medium'>{formatDateTime(slaInfo.due_time)}</div>
                            </div>
                            <div>
                              <Text type='secondary' className='text-sm'>
                                状态
                              </Text>
                              <Badge
                                status={
                                  slaInfo.status === 'active'
                                    ? 'processing'
                                    : slaInfo.status === 'breached'
                                    ? 'error'
                                    : 'success'
                                }
                                text={
                                  slaInfo.status === 'active'
                                    ? '进行中'
                                    : slaInfo.status === 'breached'
                                    ? '已违反'
                                    : '已完成'
                                }
                              />
                            </div>
                          </div>
                        </Card>
                      )}
                    </div>
                  </div>
                </div>
              ),
            },
            {
              key: 'attachments',
              label: '附件',
              children: (
                <div className='p-6'>
                  <TicketAttachmentSection
                    ticketId={ticket.id}
                    canUpload={canEdit}
                    canDelete={attachment => {
                      const { user } = useAuthStore.getState();
                      return canEdit || attachment.uploaded_by === user?.id;
                    }}
                    onAttachmentUploaded={() => {
                      // 刷新附件列表
                      fetchTicketData();
                    }}
                    onAttachmentDeleted={() => {
                      // 刷新附件列表
                      fetchTicketData();
                    }}
                  />
                </div>
              ),
            },
            {
              key: 'comments',
              label: '评论',
              children: (
                <div className='p-6'>
                  <div className='mb-6'>
                    <Card title='添加评论' className='shadow-sm'>
                      <div className='space-y-4'>
                        <div className='flex items-center space-x-4'>
                          <div className='flex items-center space-x-2'>
                            <input
                              type='checkbox'
                              id='internal'
                              checked={isInternal}
                              onChange={e => setIsInternal(e.target.checked)}
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
                          onChange={e => setNewComment(e.target.value)}
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

                  <div className='space-y-4'>
                    {comments.map(comment => (
                      <Card key={comment.id} className='shadow-sm'>
                        {editingCommentId === comment.id ? (
                          <div className='space-y-3'>
                            <TextArea
                              value={editingCommentContent}
                              onChange={e => setEditingCommentContent(e.target.value)}
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
                                {comment.is_internal && <AntTag color='orange'>仅内部可见</AntTag>}
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
              ),
            },
            {
              key: 'notifications',
              label: '通知',
              children: (
                <div className='p-6'>
                  <TicketNotificationSection
                    ticketId={ticket.id}
                    canSend={canEdit}
                    onNotificationSent={() => {
                      // 刷新通知列表
                      fetchTicketData();
                    }}
                  />
                </div>
              ),
            },
            {
              key: 'history',
              label: '历史记录',
              children: (
                <div className='p-6'>
                  <Title level={4}>操作历史</Title>
                  {ticketHistory.length > 0 ? (
                    <Timeline>
                      {ticketHistory.map(record => (
                        <Timeline.Item key={record.id}>
                          <div className='flex items-center justify-between'>
                            <div>
                              <Text strong>{record.user?.name || '系统'}</Text>
                              <div className='text-sm text-gray-600'>
                                修改了 {record.field_name}
                              </div>
                              {record.change_reason && (
                                <div className='text-sm text-gray-500'>
                                  原因: {record.change_reason}
                                </div>
                              )}
                            </div>
                            <div className='text-right'>
                              <div className='text-sm text-gray-500'>
                                {formatDateTime(record.changed_at)}
                              </div>
                              <div className='text-xs text-gray-400'>
                                {record.old_value} → {record.new_value}
                              </div>
                            </div>
                          </div>
                        </Timeline.Item>
                      ))}
                    </Timeline>
                  ) : (
                    <div className='text-center py-8 text-gray-500'>
                      <History className='w-16 h-16 mx-auto mb-4 text-gray-300' />
                      <Text>暂无历史记录</Text>
                    </div>
                  )}
                </div>
              ),
            },
          ]}
        />
      </div>

      {/* 上传附件模态框 */}
      <Modal
        title='上传附件'
        open={showUploadModal}
        onOk={handleUploadAttachment}
        onCancel={() => {
          setShowUploadModal(false);
          setUploadingFiles([]);
          setUploadProgress({});
        }}
        okText='上传'
        cancelText='取消'
        okButtonProps={{ disabled: uploadingFiles.length === 0 }}
        width={600}
      >
        <div className='space-y-4'>
          <AntUpload.Dragger
            multiple
            fileList={uploadingFiles.map((file, index) => ({
              uid: `${index}-${file.name}`,
              name: file.name,
              size: file.size,
              type: file.type,
            }))}
            beforeUpload={file => {
              setUploadingFiles(prev => [...prev, file]);
              return false; // 阻止自动上传
            }}
            onRemove={file => {
              setUploadingFiles(prev => prev.filter((f, i) => `${i}-${f.name}` !== file.uid));
            }}
            accept='*/*'
          >
            <p className='ant-upload-drag-icon'>
              <Upload className='w-10 h-10 text-blue-500 mx-auto' />
            </p>
            <p className='ant-upload-text'>点击或拖拽文件到此区域上传</p>
            <p className='ant-upload-hint'>支持多文件上传，单个文件最大10MB</p>
          </AntUpload.Dragger>

          {uploadingFiles.length > 0 && (
            <div className='space-y-2'>
              <Text strong>待上传文件 ({uploadingFiles.length})</Text>
              {uploadingFiles.map((file, index) => {
                const fileId = `${file.name}-${file.size}`;
                const progress = uploadProgress[fileId] || 0;
                return (
                  <div key={index} className='p-2 bg-gray-50 rounded border'>
                    <div className='flex items-center justify-between mb-1'>
                      <div className='flex items-center space-x-2'>
                        <FileText className='w-4 h-4 text-blue-500' />
                        <span className='text-sm font-medium'>{file.name}</span>
                        <span className='text-xs text-gray-500'>({formatFileSize(file.size)})</span>
                      </div>
                      <Button
                        type='link'
                        size='small'
                        danger
                        icon={<X className='w-3 h-3' />}
                        onClick={() => {
                          setUploadingFiles(prev => prev.filter((f, i) => i !== index));
                          setUploadProgress(prev => {
                            const newProgress = { ...prev };
                            delete newProgress[fileId];
                            return newProgress;
                          });
                        }}
                      >
                        移除
                      </Button>
                    </div>
                    {progress > 0 && progress < 100 && <Progress percent={progress} size='small' />}
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </Modal>

      {/* 附件预览模态框 */}
      <Modal
        title={previewAttachment?.file_name || '附件预览'}
        open={!!previewAttachment}
        onCancel={() => setPreviewAttachment(null)}
        footer={[
          <Button
            key='download'
            icon={<Download />}
            onClick={() => {
              if (previewAttachment) {
                const url = TicketApi.getAttachmentDownloadUrl(ticket.id, previewAttachment.id);
                const link = document.createElement('a');
                link.href = url;
                link.download = previewAttachment.file_name;
                document.body.appendChild(link);
                link.click();
                document.body.removeChild(link);
              }
            }}
          >
            下载
          </Button>,
          <Button key='close' onClick={() => setPreviewAttachment(null)}>
            关闭
          </Button>,
        ]}
        width={800}
      >
        {previewAttachment && (
          <div className='flex justify-center items-center min-h-[400px]'>
            {previewAttachment.mime_type?.startsWith('image/') ? (
              <AntImage
                src={previewAttachment.file_url}
                alt={previewAttachment.file_name}
                style={{ maxWidth: '100%', maxHeight: '600px' }}
              />
            ) : (
              <iframe
                src={previewAttachment.file_url}
                style={{ width: '100%', height: '600px', border: 'none' }}
                title={previewAttachment.file_name}
              />
            )}
          </div>
        )}
      </Modal>

      {/* 智能分配模态框 */}
      <SmartAssignmentModal
        visible={showSmartAssignModal}
        ticketId={ticket.id}
        onCancel={() => setShowSmartAssignModal(false)}
        onSuccess={handleSmartAssignSuccess}
      />
    </div>
  );
};
