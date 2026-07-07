'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useParams } from 'next/navigation';
import { TicketApi } from '@/lib/api/ticket-api';
import { UserApi } from '@/lib/api/user-api';
import type { Ticket } from '@/lib/api/api-config';
import type { User } from '@/lib/api/user-api';
import type { TicketPriority } from '@/types/ticket';
import {
  ArrowLeft,
  AlertCircle,
  XCircle,
  UserCheck,
  Edit,
  Save,
  X,
  Trash2,
  Check,
  XIcon,
} from 'lucide-react';
import Link from 'next/link';
import {
  Button,
  Card,
  Typography,
  App,
  Badge,
  Tag,
  Descriptions,
  Space,
  Modal,
  Form,
  Select,
  Input,
} from 'antd';
import { useAuthStore } from '@/lib/store/auth-store';
import { useErrorHandler } from '@/lib/hooks/useErrorHandler';
import { formatDateTime } from '@/lib/formatters';
import { SafeTextBlock } from '@/components/common/SafeContent';
import { AISuggestionPanel } from '@/components/business/AISuggestionPanel';
import {
  isValidTransition,
  getAllowedTransitions,
  isFinalStatus,
} from '@/lib/utils/workflow-state-machine';

const { Title, Text } = Typography;
const { TextArea } = Input;

const ticketPriorities: TicketPriority[] = ['low', 'medium', 'high', 'urgent', 'critical'];
const toTicketPriority = (value: string): TicketPriority =>
  ticketPriorities.includes(value as TicketPriority) ? (value as TicketPriority) : 'medium';

const statusMap: Record<
  string,
  { text: string; status: 'success' | 'processing' | 'default' | 'error' | 'warning' }
> = {
  new: { text: '新建', status: 'default' },
  open: { text: '处理中', status: 'processing' },
  inProgress: { text: '进行中', status: 'processing' },
  pending: { text: '挂起', status: 'warning' },
  resolved: { text: '已解决', status: 'success' },
  closed: { text: '已关闭', status: 'default' },
  cancelled: { text: '已取消', status: 'error' },
  rejected: { text: '已拒绝', status: 'error' },
  approved: { text: '已批准', status: 'success' },
};

// Bug #9 修复：添加优先级本地化映射
const priorityMap: Record<string, string> = {
  critical: '紧急',
  urgent: '紧急',
  high: '高优先级',
  medium: '中优先级',
  low: '低优先级',
};

const getPriorityText = (priority: string): string => {
  return priorityMap[priority] || priority;
};

const TicketDetailPage: React.FC = () => {
  const params = useParams();
  const { message: antMessage } = App.useApp();
  const { user: currentUser } = useAuthStore();
  const { handleError } = useErrorHandler();
  const [ticket, setTicket] = useState<Ticket | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [assignModalVisible, setAssignModalVisible] = useState(false);
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [deleteModalVisible, setDeleteModalVisible] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [assigning, setAssigning] = useState(false);
  const [approving, setApproving] = useState(false);
  const [rejecting, setRejecting] = useState(false);
  const [users, setUsers] = useState<User[]>([]);
  const [loadingUsers, setLoadingUsers] = useState(false);
  const [slaInfo, setSlaInfo] = useState<{
    slaName: string;
    responseDeadline: string | null;
    resolutionDeadline: string | null;
    isBreached: boolean;
    responseTimeRemaining: number | null;
    resolutionTimeRemaining: number | null;
  } | null>(null);
  const [assignForm] = Form.useForm();
  const [editForm] = Form.useForm();

  const ticketId = parseInt(params.ticketId as string);

  // 判断当前用户是否是工单申请人
  const isRequester = ticket?.requesterId === currentUser?.id;

  // 判断工单是否处于终态（不可再操作）
  const isTicketFinal = ticket ? isFinalStatus(ticket.status as any) : false;

  // Get ticket details
  const fetchTicket = useCallback(async () => {
    // Skip if ticketId is not a valid number
    if (!ticketId || isNaN(ticketId) || ticketId <= 0) {
      setError('无效的工单ID');
      setLoading(false);
      return;
    }

    try {
      setLoading(true);
      setError(null);
      console.log('[TicketDetail] Fetching ticket:', ticketId);
      const data = await TicketApi.getTicket(ticketId);
      console.log('[TicketDetail] Received ticket data:', data);
      setTicket(data as Ticket);
    } catch (error) {
      console.error('[TicketDetail] Fetch error:', error);
      setError(error instanceof Error ? error.message : 'Network error');
    } finally {
      setLoading(false);
    }
  }, [ticketId]);

  // Get users for assignment
  const fetchUsers = useCallback(async () => {
    try {
      setLoadingUsers(true);
      const data = await UserApi.getUsers({ pageSize: 100 });
      setUsers(data.users || []);
    } catch (error) {
      // 用户获取失败时使用空数组，不显示错误提示
      setUsers([]);
    } finally {
      setLoadingUsers(false);
    }
  }, [antMessage]);

  // Get ticket SLA info
  const fetchSLAInfo = useCallback(async () => {
    try {
      const data = await TicketApi.getTicketSLA(ticketId);
      setSlaInfo(data);
    } catch (error) {
      // SLA获取失败时不显示SLA信息
      setSlaInfo(null);
    }
  }, [ticketId]);

  useEffect(() => {
    if (ticketId) {
      fetchTicket();
    }
  }, [ticketId, fetchTicket]);

  useEffect(() => {
    if (ticketId) {
      fetchSLAInfo();
    }
  }, [ticketId, fetchSLAInfo]);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  // Handle approval (轻量版：仅改状态)
  const handleApprove = async () => {
    try {
      setApproving(true);
      await TicketApi.updateTicketStatus(ticketId, 'approved');
      antMessage.success('批准成功');
      fetchTicket();
    } catch (error) {
      handleError(error, 'approveTicket', '批准失败');
    } finally {
      setApproving(false);
    }
  };

  // Handle rejection (轻量版：仅改状态)
  const handleReject = async () => {
    try {
      setRejecting(true);
      await TicketApi.updateTicketStatus(ticketId, 'rejected');
      antMessage.success('已拒绝');
      fetchTicket();
    } catch (error) {
      handleError(error, 'rejectTicket', '拒绝失败');
    } finally {
      setRejecting(false);
    }
  };

  // Handle assignment
  const handleAssign = () => {
    setAssignModalVisible(true);
  };

  // Handle assignment submit
  const handleAssignSubmit = async (values: { assigneeId: number; comment?: string }) => {
    try {
      setAssigning(true);
      await TicketApi.assignTicket(ticketId, values);
      antMessage.success('工单分配成功');
      setAssignModalVisible(false);
      assignForm.resetFields();
      fetchTicket();
    } catch (error) {
      handleError(error, 'assignTicket', '分配失败');
    } finally {
      setAssigning(false);
    }
  };

  // Handle edit
  const handleUpdate = () => {
    if (ticket) {
      editForm.setFieldsValue({
        title: ticket.title,
        description: ticket.description,
        priority: ticket.priority,
        status: ticket.status,
      });
      setEditModalVisible(true);
    }
  };

  // Handle edit submit
  const handleEditSubmit = async (values: Partial<Ticket>) => {
    try {
      // 状态转换验证
      if (values.status && ticket?.status && values.status !== ticket.status) {
        if (!isValidTransition(ticket.status as any, values.status as any)) {
          antMessage.error(
            `不允许从 "${statusMap[ticket.status]?.text || ticket.status}" 转换到 "${statusMap[values.status]?.text || values.status}"`
          );
          return;
        }
      }

      // 添加版本号用于乐观锁
      const updatePayload = {
        ...values,
        version: ticket?.version,
      };

      await TicketApi.updateTicket(ticketId, updatePayload);
      antMessage.success('工单更新成功');
      setEditModalVisible(false);
      fetchTicket();
    } catch (error) {
      handleError(error, 'updateTicket', '更新失败');
    }
  };

  // Handle delete click
  const handleDeleteClick = () => {
    setDeleteModalVisible(true);
  };

  // Handle delete confirm
  const handleDeleteConfirm = async () => {
    try {
      setDeleting(true);
      await TicketApi.deleteTicket(ticketId);
      antMessage.success('工单删除成功');
      setDeleteModalVisible(false);
      // Navigate back to ticket list
      window.location.href = '/tickets';
    } catch (error) {
      handleError(error, 'deleteTicket', '删除失败');
    } finally {
      setDeleting(false);
    }
  };

  if (loading) {
    return (
      <div className="p-6">
        <Card>
          <div className="text-center py-8">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
            <Text className="mt-4 block">加载中...</Text>
          </div>
        </Card>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <Card>
          <div className="text-center py-8">
            <AlertCircle className="w-12 h-12 text-red-500 mx-auto mb-4" />
            <Title level={4} className="text-red-600 mb-2">
              加载失败
            </Title>
            <Text type="secondary">{error}</Text>
            <div className="mt-4">
              <Button type="primary" onClick={fetchTicket}>
                重试
              </Button>
            </div>
          </div>
        </Card>
      </div>
    );
  }

  if (!ticket) {
    return (
      <div className="p-6">
        <Card>
          <div className="text-center py-8">
            <XCircle className="w-12 h-12 text-gray-400 mx-auto mb-4" />
            <Title level={4} className="text-gray-600 mb-2">
              未找到工单
            </Title>
            <Text type="secondary">未找到指定的工单</Text>
            <div className="mt-4">
              <Link href="/tickets">
                <Button type="primary">返回工单列表</Button>
              </Link>
            </div>
          </div>
        </Card>
      </div>
    );
  }

  return (
    <div className="p-6">
      {/* Page header */}
      <div className="mb-6">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center space-x-4">
            <Link href="/tickets">
              <Button icon={<ArrowLeft />} type="text">
                返回列表
              </Button>
            </Link>
            <div>
              <Title level={2} className="!mb-1 !text-gray-900">
                工单详情 #{ticket.id}
              </Title>
              <Text type="secondary">{ticket.title}</Text>
            </div>
          </div>
          <div className="flex items-center space-x-2">
            <Badge
              status={statusMap[ticket.status]?.status || 'default'}
              text={statusMap[ticket.status]?.text || ticket.status}
            />
            <Tag
              color={
                ticket.priority === 'high'
                  ? 'red'
                  : ticket.priority === 'medium'
                    ? 'orange'
                    : 'green'
              }
            >
              {ticket.priority === 'high'
                ? '高优先级'
                : ticket.priority === 'medium'
                  ? '中优先级'
                  : '低优先级'}
            </Tag>
          </div>
        </div>
      </div>

      {/* AI Suggestion Panel */}
      {ticket && (
        <AISuggestionPanel
          title={ticket.title}
          description={ticket.description}
          onAccept={async suggestion => {
            // Bug 11 修复：onAccept 之前只打开编辑弹窗没有真正落库
            // 现在直接调 updateTicket 写入 AI 建议的 category + priority
            if (
              suggestion.priority === ticket.priority &&
              suggestion.category === ticket.category
            ) {
              antMessage.info('AI建议与当前分类/优先级一致，无需更新');
              return;
            }
            try {
              await TicketApi.updateTicket(ticketId, {
                category: suggestion.category,
                priority: toTicketPriority(suggestion.priority),
                version: ticket.version,
              });
              antMessage.success(
                `已采纳AI建议：分类 ${suggestion.category}，优先级 ${suggestion.priority}`,
              );
              fetchTicket();
            } catch (err) {
              handleError(err, 'applyAISuggestion', '采纳建议失败');
            }
          }}
        />
      )}

      <Card className="rounded-lg shadow-sm border border-gray-200">
        <Space orientation="vertical" size={16} style={{ width: '100%' }}>
          <Descriptions column={2} bordered size="middle">
            <Descriptions.Item label="标题">{ticket.title}</Descriptions.Item>
            <Descriptions.Item label="编号">{ticket.ticketNumber || '-'}</Descriptions.Item>
            <Descriptions.Item label="状态">{statusMap[ticket.status]?.text || ticket.status}</Descriptions.Item>
            <Descriptions.Item label="优先级">{getPriorityText(ticket.priority)}</Descriptions.Item>
            <Descriptions.Item label="创建时间">{formatDateTime(ticket.createdAt)}</Descriptions.Item>
            <Descriptions.Item label="更新时间">{formatDateTime(ticket.updatedAt)}</Descriptions.Item>
            <Descriptions.Item label="描述" span={2}>
              <SafeTextBlock content={ticket.description} fallback="暂无描述" />
            </Descriptions.Item>
          </Descriptions>

          {/* SLA Information */}
          {slaInfo && (
            <Card size="small" title="SLA信息" className="mt-4">
              <Space orientation="vertical" style={{ width: '100%' }}>
                <div className="flex justify-between">
                  <Text type="secondary">SLA定义:</Text>
                  <Tag color={slaInfo.isBreached ? 'red' : 'blue'}>{slaInfo.slaName}</Tag>
                </div>
                {slaInfo.responseDeadline && (
                  <div className="flex justify-between">
                    <Text type="secondary">响应截止:</Text>
                    <Text
                      type={
                        slaInfo.responseTimeRemaining !== null &&
                        slaInfo.responseTimeRemaining < 0
                          ? 'danger'
                          : undefined
                      }
                    >
                      {new Date(slaInfo.responseDeadline).toLocaleString()}
                      {slaInfo.responseTimeRemaining !== null &&
                        slaInfo.responseTimeRemaining < 0 &&
                        ' (已超时)'}
                    </Text>
                  </div>
                )}
                {slaInfo.resolutionDeadline && (
                  <div className="flex justify-between">
                    <Text type="secondary">解决截止:</Text>
                    <Text
                      type={
                        slaInfo.resolutionTimeRemaining !== null &&
                        slaInfo.resolutionTimeRemaining < 0
                          ? 'danger'
                          : undefined
                      }
                    >
                      {new Date(slaInfo.resolutionDeadline).toLocaleString()}
                      {slaInfo.resolutionTimeRemaining !== null &&
                        slaInfo.resolutionTimeRemaining < 0 &&
                        ' (已超时)'}
                    </Text>
                  </div>
                )}
                {slaInfo.isBreached && <Tag color="red">SLA已违规</Tag>}
              </Space>
            </Card>
          )}

          <Space>
            <Button
              type="primary"
              icon={<Check size={16} />}
              onClick={handleApprove}
              loading={approving}
              disabled={isRequester || isTicketFinal}
              title={
                isRequester ? '不能审批自己提交的工单' : isTicketFinal ? '工单已结束，无法操作' : ''
              }
            >
              批准
            </Button>
            <Button
              danger
              icon={<XIcon size={16} />}
              onClick={handleReject}
              loading={rejecting}
              disabled={isRequester || isTicketFinal}
              title={
                isRequester ? '不能拒绝自己提交的工单' : isTicketFinal ? '工单已结束，无法操作' : ''
              }
            >
              拒绝
            </Button>
            <Button
              icon={<UserCheck size={16} />}
              onClick={handleAssign}
              loading={loadingUsers}
              disabled={isTicketFinal}
              title={isTicketFinal ? '工单已结束，无法分配' : ''}
            >
              分配
            </Button>
            <Button
              icon={<Edit size={16} />}
              onClick={handleUpdate}
              disabled={isTicketFinal}
              title={isTicketFinal ? '工单已结束，无法编辑' : ''}
            >
              编辑
            </Button>
            <Button danger icon={<Trash2 size={16} />} onClick={handleDeleteClick}>
              删除
            </Button>
          </Space>

          {isTicketFinal && (
            <Text type="secondary" className="block mt-2">
              工单已结束，无法进行操作
            </Text>
          )}

          {isRequester && !isTicketFinal && (
            <Text type="secondary" className="block mt-2">
              您是此工单的申请人，无法进行审批操作
            </Text>
          )}

          {!isRequester && !isTicketFinal && (
            <Text type="secondary">支持状态变更、审批流程、分配处理人等完整工单操作</Text>
          )}
        </Space>

        {/* Assignment Modal */}
        <Modal
          title={
            <Space>
              <UserCheck className="w-5 h-5 text-blue-600" />
              分配工单
            </Space>
          }
          open={assignModalVisible}
          onCancel={() => {
            setAssignModalVisible(false);
            assignForm.resetFields();
          }}
          footer={null}
          width={500}
        >
          <Form form={assignForm} layout="vertical" onFinish={handleAssignSubmit}>
            <Form.Item
              label="分配给"
              name="assignee_id"
              rules={[{ required: true, message: '请选择处理人' }]}
            >
              <Select
                placeholder="请选择处理人"
                loading={loadingUsers}
                showSearch
                filterOption={(input, option) =>
                  (option?.label as unknown as string)?.toLowerCase().includes(input.toLowerCase())
                }
                options={users.map(user => ({
                  value: user.id,
                  label: (
                    <Space>
                      <span>{user.name}</span>
                      <Text type="secondary" className="text-xs">
                        ({user.username})
                      </Text>
                      {user.department && <Tag color="blue">{user.department}</Tag>}
                    </Space>
                  ),
                }))}
              />
            </Form.Item>
            <Form.Item label="备注" name="comment">
              <TextArea rows={3} placeholder="请输入分配备注（可选）" maxLength={500} showCount />
            </Form.Item>
            <Form.Item className="mb-0">
              <Space className="w-full justify-end">
                <Button
                  icon={<X />}
                  onClick={() => {
                    setAssignModalVisible(false);
                    assignForm.resetFields();
                  }}
                >
                  取消
                </Button>
                <Button type="primary" htmlType="submit" icon={<Save />} loading={assigning}>
                  确认分配
                </Button>
              </Space>
            </Form.Item>
          </Form>
        </Modal>

        {/* Edit Modal */}
        <Modal
          title={
            <Space>
              <Edit className="w-5 h-5 text-green-600" />
              编辑工单
            </Space>
          }
          open={editModalVisible}
          onCancel={() => {
            setEditModalVisible(false);
            editForm.resetFields();
          }}
          footer={null}
          width={600}
        >
          <Form form={editForm} layout="vertical" onFinish={handleEditSubmit}>
            <Form.Item
              label="工单标题"
              name="title"
              rules={[
                { required: true, message: '请输入工单标题' },
                { max: 100, message: '标题不能超过100个字符' },
              ]}
            >
              <Input placeholder="请输入工单标题" />
            </Form.Item>
            <Form.Item
              label="工单描述"
              name="description"
              rules={[
                { required: true, message: '请输入工单描述' },
                { max: 2000, message: '描述不能超过2000个字符' },
              ]}
            >
              <TextArea rows={6} placeholder="请输入工单描述" showCount maxLength={2000} />
            </Form.Item>
            <div className="grid grid-cols-2 gap-4">
              <Form.Item
                label="优先级"
                name="priority"
                rules={[{ required: true, message: '请选择优先级' }]}
              >
                <Select
                  placeholder="请选择优先级"
                  options={[
                    {
                      value: 'low',
                      label: (
                        <>
                          <Tag color="green">低优先级</Tag>
                        </>
                      ),
                    },
                    {
                      value: 'medium',
                      label: (
                        <>
                          <Tag color="orange">中优先级</Tag>
                        </>
                      ),
                    },
                    {
                      value: 'high',
                      label: (
                        <>
                          <Tag color="red">高优先级</Tag>
                        </>
                      ),
                    },
                  ]}
                />
              </Form.Item>
              <Form.Item
                label="状态"
                name="status"
                rules={[{ required: true, message: '请选择状态' }]}
                extra={ticket ? `当前状态: ${statusMap[ticket.status]?.text || ticket.status}` : ''}
              >
                <Select
                  placeholder="请选择状态"
                  options={[
                    { value: 'new', label: '待处理' },
                    { value: 'open', label: '处理中' },
                    { value: 'pending', label: '暂停' },
                    { value: 'resolved', label: '已解决' },
                    { value: 'closed', label: '已关闭' },
                  ]}
                />
              </Form.Item>
            </div>
            <Form.Item className="mb-0">
              <Space className="w-full justify-end">
                <Button
                  icon={<X />}
                  onClick={() => {
                    setEditModalVisible(false);
                    editForm.resetFields();
                  }}
                >
                  取消
                </Button>
                <Button type="primary" htmlType="submit" icon={<Save />}>
                  保存修改
                </Button>
              </Space>
            </Form.Item>
          </Form>
        </Modal>

        {/* Delete Confirmation Modal */}
        <Modal
          title={
            <Space>
              <Trash2 className="w-5 h-5 text-red-600" />
              删除工单
            </Space>
          }
          open={deleteModalVisible}
          onCancel={() => setDeleteModalVisible(false)}
          footer={null}
          width={400}
        >
          <div className="py-4">
            <div className="flex items-start gap-3 mb-4">
              <AlertCircle className="w-6 h-6 text-red-500 flex-shrink-0 mt-0.5" />
              <div>
                <Typography.Text strong className="text-lg">
                  确定要删除此工单吗？
                </Typography.Text>
                <Typography.Paragraph type="secondary" className="mb-0 mt-1">
                  此操作不可恢复，工单编号 #{ticket.id} 将被永久删除。
                </Typography.Paragraph>
              </div>
            </div>
            <div className="bg-gray-50 rounded p-3 mb-4">
              <Typography.Text type="secondary" className="text-sm">
                工单信息：
              </Typography.Text>
              <div className="mt-1">
                <Text strong>{ticket.title}</Text>
              </div>
            </div>
          </div>
          <Space className="w-full justify-end">
            <Button onClick={() => setDeleteModalVisible(false)} disabled={deleting}>
              取消
            </Button>
            <Button
              danger
              type="primary"
              onClick={handleDeleteConfirm}
              loading={deleting}
              icon={<Trash2 size={14} />}
            >
              确认删除
            </Button>
          </Space>
        </Modal>
      </Card>
    </div>
  );
};

export default TicketDetailPage;
