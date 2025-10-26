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
} from 'antd';
import { TicketApi } from '@/lib/api/ticket-api';
import { Ticket, Attachment, Comment, WorkflowStep, SLAInfo } from '../lib/api-config';

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
  const [commentType, setCommentType] = useState<'comment' | 'work_note'>('comment');
  const [isInternal, setIsInternal] = useState(false);
  const [showUploadModal, setShowUploadModal] = useState(false);
  const [uploadingFile, setUploadingFile] = useState<File | null>(null);
  const [activeTab, setActiveTab] = useState('overview');

  // 状态管理
  const [attachments, setAttachments] = useState<Attachment[]>([]);
  const [comments, setComments] = useState<Comment[]>([]);
  const [workflowSteps, setWorkflowSteps] = useState<WorkflowStep[]>([]);
  const [slaInfo, setSlaInfo] = useState<SLAInfo | null>(null);
  const [ticketHistory, setTicketHistory] = useState<unknown[]>([]);
  const [loading, setLoading] = useState(false);

  // 获取工单相关数据
  useEffect(() => {
    fetchTicketData();
  }, [ticket.id]);

  const fetchTicketData = async () => {
    setLoading(true);
    try {
      // 并行获取所有数据
      const [attachmentsRes, commentsRes, workflowRes, slaRes, historyRes] =
        await Promise.allSettled([
          TicketApi.getTicketAttachments(ticket.id),
          TicketApi.getTicketComments(ticket.id),
          TicketApi.getTicketWorkflow(ticket.id),
          TicketApi.getTicketSLA(ticket.id),
          TicketApi.getTicketHistory(ticket.id),
        ]);

      if (attachmentsRes.status === 'fulfilled') {
        setAttachments(attachmentsRes.value || []);
      }
      if (commentsRes.status === 'fulfilled') {
        setComments((commentsRes.value as Comment[]) || []);
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
        type: commentType,
        is_internal: isInternal,
      });

      antMessage.success('评论添加成功');
      setNewComment('');
      fetchTicketData(); // 刷新评论列表
    } catch (error) {
      antMessage.error('添加评论失败');
    }
  };

  // 上传附件
  const handleUploadAttachment = async () => {
    if (!uploadingFile) return;

    try {
      await TicketApi.uploadTicketAttachment(ticket.id, uploadingFile);
      antMessage.success('附件上传成功');
      setShowUploadModal(false);
      setUploadingFile(null);
      fetchTicketData(); // 刷新附件列表
    } catch (error) {
      antMessage.error('附件上传失败');
    }
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

  const TypeIcon = getTypeIcon(ticket.type || 'Service Request');
  const priorityConfig = getPriorityConfig(ticket.priority || '中');
  const statusConfig = getStatusConfig(ticket.status || '待处理');

  return (
    <div className='max-w-7xl mx-auto bg-gray-50 min-h-screen'>
      {/* 页面头部 */}
      <div className='bg-white shadow-sm border-b mb-6'>
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

          {/* 状态和优先级标签 */}
          <div className='flex items-center space-x-4 mb-4'>
            <Badge
              status={
                ticket.status === 'open'
                  ? 'processing'
                  : ticket.status === 'closed'
                  ? 'success'
                  : 'warning'
              }
              text={ticket.status || '待处理'}
            />
            <AntTag color={priorityConfig.color}>优先级: {ticket.priority || '中'}</AntTag>
            {ticket.category && <AntTag color='blue'>{ticket.category}</AntTag>}
            {ticket.tags &&
              ticket.tags.map((tag, index) => (
                <AntTag key={index} color='green'>
                  {tag}
                </AntTag>
              ))}
          </div>

          {/* 基本信息网格 */}
          <div className='grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4'>
            <div>
              <Text type='secondary' className='text-sm'>
                负责人
              </Text>
              <div className='flex items-center space-x-2'>
                <User className='w-4 h-4 text-gray-500' />
                <Text>{ticket.assignee?.name || '未分配'}</Text>
              </div>
            </div>
            <div>
              <Text type='secondary' className='text-sm'>
                报告人
              </Text>
              <div className='flex items-center space-x-2'>
                <User className='w-4 h-4 text-gray-500' />
                <Text>{ticket.requester?.name || '未知'}</Text>
              </div>
            </div>
            <div>
              <Text type='secondary' className='text-sm'>
                创建时间
              </Text>
              <div className='flex items-center space-x-2'>
                <Calendar className='w-4 h-4 text-gray-500' />
                <Text>{formatDateTime(ticket.created_at)}</Text>
              </div>
            </div>
            <div>
              <Text type='secondary' className='text-sm'>
                最后更新
              </Text>
              <div className='flex items-center space-x-2'>
                <Clock className='w-4 h-4 text-gray-500' />
                <Text>{formatDateTime(ticket.updated_at)}</Text>
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
        >
          {/* 概览标签页 */}
          <TabPane tab='概览' key='overview'>
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
                      <Paragraph className='whitespace-pre-wrap'>{ticket.description}</Paragraph>
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
                        <Paragraph className='whitespace-pre-wrap'>{ticket.resolution}</Paragraph>
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
                                  <div className='text-sm text-gray-600 mt-1'>{step.comments}</div>
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
                    <Space direction='vertical' className='w-full'>
                      <Input
                        value={assigneeInput}
                        onChange={e => setAssigneeInput(e.target.value)}
                        placeholder='输入负责人姓名'
                      />
                      <Button
                        type='primary'
                        onClick={handleAssign}
                        disabled={!assigneeInput.trim()}
                        className='w-full'
                      >
                        分配
                      </Button>
                    </Space>
                  </Card>

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
          </TabPane>

          {/* 附件标签页 */}
          <TabPane tab='附件' key='attachments'>
            <div className='p-6'>
              <div className='flex justify-between items-center mb-4'>
                <Title level={4}>附件管理</Title>
                <Button type='primary' icon={<Upload />} onClick={() => setShowUploadModal(true)}>
                  上传附件
                </Button>
              </div>

              {attachments.length > 0 ? (
                <List
                  dataSource={attachments}
                  renderItem={attachment => (
                    <List.Item
                      actions={[
                        <Button
                          key='download'
                          type='link'
                          icon={<Download />}
                          onClick={() => window.open(attachment.url)}
                        >
                          下载
                        </Button>,
                        <Button
                          key='delete'
                          type='link'
                          danger
                          icon={<Trash2 />}
                          onClick={() => handleDeleteAttachment(attachment.id)}
                        >
                          删除
                        </Button>,
                      ]}
                    >
                      <List.Item.Meta
                        avatar={<FileText className='w-8 h-8 text-blue-500' />}
                        title={attachment.original_name}
                        description={
                          <div className='text-sm text-gray-500'>
                            <div>大小: {formatFileSize(attachment.file_size)}</div>
                            <div>类型: {attachment.mime_type}</div>
                            <div>上传时间: {formatDateTime(attachment.uploaded_at)}</div>
                          </div>
                        }
                      />
                    </List.Item>
                  )}
                />
              ) : (
                <div className='text-center py-8 text-gray-500'>
                  <FileText className='w-16 h-16 mx-auto mb-4 text-gray-300' />
                  <Text>暂无附件</Text>
                </div>
              )}
            </div>
          </TabPane>

          {/* 评论标签页 */}
          <TabPane tab='评论' key='comments'>
            <div className='p-6'>
              <div className='mb-6'>
                <Card title='添加评论' className='shadow-sm'>
                  <div className='space-y-4'>
                    <div className='flex space-x-4'>
                      <Select value={commentType} onChange={setCommentType} style={{ width: 120 }}>
                        <Select.Option value='comment'>公开评论</Select.Option>
                        <Select.Option value='work_note'>工作备注</Select.Option>
                      </Select>
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
                    <TextArea
                      value={newComment}
                      onChange={e => setNewComment(e.target.value)}
                      placeholder='输入您的评论...'
                      rows={3}
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
                    <div className='flex items-start space-x-3'>
                      <Avatar size='small' icon={<User />} />
                      <div className='flex-1'>
                        <div className='flex items-center space-x-2 mb-2'>
                          <Text strong>{comment.author?.name || '未知用户'}</Text>
                          <AntTag color={comment.type === 'work_note' ? 'blue' : 'green'}>
                            {comment.type === 'work_note' ? '工作备注' : '公开评论'}
                          </AntTag>
                          {comment.is_internal && <AntTag color='orange'>仅内部可见</AntTag>}
                          <Text type='secondary' className='text-sm'>
                            {formatDateTime(comment.created_at)}
                          </Text>
                        </div>
                        <Paragraph className='mb-0'>{comment.content}</Paragraph>
                      </div>
                    </div>
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
          </TabPane>

          {/* 历史记录标签页 */}
          <TabPane tab='历史记录' key='history'>
            <div className='p-6'>
              <Title level={4}>操作历史</Title>
              {ticketHistory.length > 0 ? (
                <Timeline>
                  {ticketHistory.map(record => (
                    <Timeline.Item key={record.id}>
                      <div className='flex items-center justify-between'>
                        <div>
                          <Text strong>{record.user?.name || '系统'}</Text>
                          <div className='text-sm text-gray-600'>修改了 {record.field_name}</div>
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
          </TabPane>
        </Tabs>
      </div>

      {/* 上传附件模态框 */}
      <Modal
        title='上传附件'
        open={showUploadModal}
        onOk={handleUploadAttachment}
        onCancel={() => {
          setShowUploadModal(false);
          setUploadingFile(null);
        }}
        okText='上传'
        cancelText='取消'
        okButtonProps={{ disabled: !uploadingFile }}
      >
        <div className='space-y-4'>
          <AntUpload
            beforeUpload={file => {
              setUploadingFile(file);
              return false; // 阻止自动上传
            }}
            maxCount={1}
            accept='*/*'
          >
            <Button icon={<Upload />}>选择文件</Button>
          </AntUpload>
          {uploadingFile && (
            <div className='p-3 bg-gray-50 rounded border'>
              <div className='flex items-center space-x-2'>
                <FileText className='w-4 h-4 text-blue-500' />
                <span className='text-sm font-medium'>{uploadingFile.name}</span>
                <span className='text-xs text-gray-500'>
                  ({formatFileSize(uploadingFile.size)})
                </span>
              </div>
            </div>
          )}
        </div>
      </Modal>
    </div>
  );
};
