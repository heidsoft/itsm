'use client';

/**
 * 工单详情组件
 * 从 tickets/[ticketId]/page.tsx 抽取，与 IncidentDetail/ProblemDetail/ChangeDetail 域组件模式对齐
 * 包含：基本信息、SLA、审批/拒绝/分配/编辑/抄送/删除操作、详情 Tabs（评论/附件/审批链/历史/关联）
 */

import React, { useState, useEffect, useCallback } from 'react';
import { useParams } from 'next/navigation';
import { TicketApi, type TicketConfigurationItem } from '@/lib/api/ticket-api';
import { TicketApprovalApi } from '@/lib/api/ticket-approval-api';
import type { Ticket } from '@/lib/api/api-config';
import type { User } from '@/lib/api/user-api';
import { useUserListQuery } from '@/lib/hooks/useUserListQuery';
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
  Users,
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
  Tabs,
  Skeleton,
  List,
  Empty,
} from 'antd';
import { useAuthStore } from '@/lib/store/auth-store';
import { useErrorHandler } from '@/lib/hooks/useErrorHandler';
import { formatDateTime } from '@/lib/formatters';
import { SafeTextBlock } from '@/components/common/SafeContent';
import { AISuggestionPanel } from '@/components/business/AISuggestionPanel';
import { WorkflowProgressCard } from '@/components/business/WorkflowProgressCard';
import {
  isValidTransition,
  getAllowedTransitions,
  isFinalStatus,
} from '@/lib/utils/workflow-state-machine';
import {
  CommentPanel,
  AttachmentPanel,
  HistoryTimeline,
  ApprovalWorkflowPanel,
  ticketCommentAdapter,
  ticketAttachmentAdapter,
  fetchAuditLogHistory,
} from '@/components/business/detail-tabs';
import { RelationPanel } from '@/components/ticket-relations/RelationPanel';
import {
  MessageSquare,
  Paperclip,
  History as HistoryIcon,
  GitBranch,
  Link2,
  Info,
} from 'lucide-react';
import { useI18n } from '@/lib/i18n/useI18n';

const { Title, Text } = Typography;
const { TextArea } = Input;

// 状态映射配置接口
interface StatusConfig {
  text: string;
  status: 'success' | 'processing' | 'default' | 'error' | 'warning';
}

// 优先级映射配置接口
interface PriorityConfig {
  label: string;
  color?: string;
}

// 创建状态映射的工厂函数
const createStatusMap = (t: (key: string) => string): Record<string, StatusConfig> => ({
  new: { text: t('ticketDetail.statusNew'), status: 'default' },
  open: { text: t('ticketDetail.statusOpen'), status: 'default' },
  in_progress: { text: t('ticketDetail.statusInProgress'), status: 'processing' },
  assigned: { text: t('ticketDetail.statusAssigned'), status: 'processing' },
  pending: { text: t('ticketDetail.statusPending'), status: 'warning' },
  pending_approval: { text: t('ticketDetail.statusPendingApproval'), status: 'warning' },
  resolved: { text: t('ticketDetail.statusResolved'), status: 'success' },
  closed: { text: t('ticketDetail.statusClosed'), status: 'default' },
  cancelled: { text: t('ticketDetail.statusCancelled'), status: 'error' },
  rejected: { text: t('ticketDetail.statusRejected'), status: 'error' },
  approved: { text: t('ticketDetail.statusApproved'), status: 'success' },
});

// 创建优先级映射的工厂函数
const createPriorityMap = (t: (key: string) => string): Record<string, string> => ({
  critical: t('ticketDetail.priorityCritical'),
  urgent: t('ticketDetail.priorityUrgent'),
  high: t('ticketDetail.priorityHigh'),
  medium: t('ticketDetail.priorityMedium'),
  low: t('ticketDetail.priorityLow'),
});

const ticketPriorities: TicketPriority[] = ['low', 'medium', 'high', 'urgent', 'critical'];
const toTicketPriority = (value: string): TicketPriority =>
  ticketPriorities.includes(value as TicketPriority) ? (value as TicketPriority) : 'medium';

// 把任意 formFields 值渲染为只读文案。避免对 object / array 直接 toString 产生 [object Object]
const renderFormFieldValue = (value: unknown): React.ReactNode => {
  if (value === null || value === undefined || value === '') {
    return <Text type="secondary">-</Text>;
  }
  if (typeof value === 'string') {
    return value;
  }
  if (typeof value === 'number' || typeof value === 'boolean') {
    return String(value);
  }
  if (Array.isArray(value)) {
    return value.map(item => (typeof item === 'object' ? JSON.stringify(item) : String(item))).join(', ');
  }
  if (typeof value === 'object') {
    try {
      return JSON.stringify(value);
    } catch {
      return '-';
    }
  }
  return String(value);
};

const TicketDetail: React.FC<{ id?: string }> = ({ id: propId }) => {
  const params = useParams();
  const { message: antMessage } = App.useApp();
  const { t } = useI18n();
  const { user: currentUser } = useAuthStore();
  const { handleError } = useErrorHandler();

  // 创建基于翻译的状态和优先级映射
  const statusMap = createStatusMap(t);
  const priorityMap = createPriorityMap(t);
  const getPriorityText = (priority: string): string => {
    return priorityMap[priority] || priority;
  };

  const [ticket, setTicket] = useState<Ticket | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [assignModalVisible, setAssignModalVisible] = useState(false);
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [ccModalVisible, setCCModalVisible] = useState(false);
  const [deleteModalVisible, setDeleteModalVisible] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [assigning, setAssigning] = useState(false);
  const [ccing, setCCing] = useState(false);
  const [approving, setApproving] = useState(false);
  const [rejecting, setRejecting] = useState(false);
  // 审批/驳回走 BPMN bridge：保留原 handleApprove/handleReject 触点，但改为打开评论 modal
  // 并改调 TicketApprovalApi.submitApproval（其底层即 /api/v1/tickets/workflow/approve），
  // 避免旧 updateTicketStatus(ticketId,'approved') 直接改 ticket.status 造成的双轨分叉。
  const [approvalModalVisible, setApprovalModalVisible] = useState(false);
  const [approvalAction, setApprovalAction] = useState<'approve' | 'reject'>('approve');

  // AI-Native：受影响配置项（工单→CI 反向查询）
  const [cis, setCis] = useState<TicketConfigurationItem[]>([]);
  const [cisLoading, setCisLoading] = useState(false);
  // users / loadingUsers 由 useUserListQuery 提供（带缓存）
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
  const [ccForm] = Form.useForm();
  const [approvalForm] = Form.useForm();

  // 支持通过 props 传入 id，或通过 useParams 获取
  const ticketId = parseInt((propId ?? (params?.ticketId as string)) || '');

  // 判断当前用户是否是工单申请人
  const isRequester = ticket?.requesterId === currentUser?.id;

  // 判断工单是否处于终态（不可再操作）
  const isTicketFinal = ticket ? isFinalStatus(ticket.status as any) : false;

  // Get ticket details
  const fetchTicket = useCallback(async () => {
    // Skip if ticketId is not a valid number
    if (!ticketId || isNaN(ticketId) || ticketId <= 0) {
      setError(t('ticketDetail.invalidTicketId'));
      setLoading(false);
      return;
    }

    try {
      setLoading(true);
      setError(null);
      const data = await TicketApi.getTicket(ticketId);
      setTicket(data as Ticket);
    } catch (error) {
      setError(error instanceof Error ? error.message : 'Network error');
    } finally {
      setLoading(false);
    }
  }, [ticketId]);

  // Get users for assignment（通过 React Query 缓存，避免每次详情页挂载都重新拉取）
  const usersQuery = useUserListQuery({ pageSize: 100 });
  const users = (usersQuery.data?.users ?? []) as User[];
  const loadingUsers = usersQuery.isLoading;

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

  // AI-Native：工单→配置项反向查询（受影响配置项）
  const fetchCIs = useCallback(async () => {
    if (!ticketId || isNaN(ticketId) || ticketId <= 0) return;
    setCisLoading(true);
    try {
      const data = await TicketApi.getTicketConfigurationItems(ticketId);
      setCis(Array.isArray(data) ? data : []);
    } catch {
      // CI 查询失败不阻塞工单详情展示
      setCis([]);
    } finally {
      setCisLoading(false);
    }
  }, [ticketId]);

  useEffect(() => {
    if (ticketId) {
      fetchTicket();
    }
  }, [ticketId]);

  useEffect(() => {
    if (ticketId) {
      fetchSLAInfo();
      fetchCIs();
    }
  }, [ticketId]);

  // 用户列表由 useUserListQuery 内部处理挂载与缓存。

  // 真正提交审批/驳回：通过 TicketApprovalApi 找到当前用户的待审批记录，
  // 调 /api/v1/tickets/workflow/approve（其内部触发 BPMN bridge）。
  // 找不到 pending approval 时回退到旧 updateTicketStatus（简化模式兜底）。
  const handleApprovalSubmit = async (values: { comment?: string }) => {
    const isApprove = approvalAction === 'approve';
    const setBusy = isApprove ? setApproving : setRejecting;
    setBusy(true);
    try {
      let submitted = false;
      try {
        const recRes = await TicketApprovalApi.getApprovalRecords({
          ticketId,
          page: 1,
          pageSize: 100,
        });
        const myPending = (recRes.items || []).find(
          (r) =>
            r.status === 'pending' &&
            currentUser?.id != null &&
            r.approverId === currentUser.id,
        );
        if (myPending) {
          await TicketApprovalApi.submitApproval({
            ticketId,
            approvalId: myPending.id,
            action: approvalAction,
            comment: values.comment || '',
          });
          submitted = true;
        }
      } catch (e) {
        console.warn('approval submit via bridge failed, fallback to status update', e);
      }
      if (!submitted) {
        // 兜底：未发现审批链（例如简单审批工单），保留旧行为以不阻塞用户。
        await TicketApi.updateTicketStatus(ticketId, isApprove ? 'approved' : 'rejected');
      }
      antMessage.success(
        isApprove ? t('ticketDetail.approveSuccess') : t('ticketDetail.rejectSuccess'),
      );
      setApprovalModalVisible(false);
      approvalForm.resetFields();
      fetchTicket();
    } catch (error) {
      handleError(
        error,
        isApprove ? 'approveTicket' : 'rejectTicket',
        isApprove ? t('ticketDetail.approveFailed') : t('ticketDetail.rejectFailed'),
      );
    } finally {
      setBusy(false);
    }
  };

  // 打开审批 modal：实际提交走 handleApprovalSubmit -> TicketApprovalApi.submitApproval
  // 走 BPMN bridge 后由后端更新 ticket.status，避免与流程分叉。
  const handleApprove = () => {
    setApprovalAction('approve');
    setApprovalModalVisible(true);
  };

  const handleCCSubmit = async (values: {
    ccUsers: number[];
    comment?: string;
    notifyChannels?: string[];
  }) => {
    try {
      setCCing(true);
      await TicketApi.ccTicket(
        ticketId,
        values.ccUsers,
        values.comment,
        values.notifyChannels || ['in_app']
      );
      antMessage.success(t('ticketDetail.ccSuccess'));
      setCCModalVisible(false);
      ccForm.resetFields();
      fetchTicket();
    } catch (error) {
      handleError(error, 'ccTicket', t('ticketDetail.ccFailed'));
    } finally {
      setCCing(false);
    }
  };

  // 打开驳回 modal：实际提交走 handleApprovalSubmit -> TicketApprovalApi.submitApproval
  const handleReject = () => {
    setApprovalAction('reject');
    setApprovalModalVisible(true);
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
      antMessage.success(t('ticketDetail.assignSuccess'));
      setAssignModalVisible(false);
      assignForm.resetFields();
      fetchTicket();
    } catch (error) {
      handleError(error, 'assignTicket', t('ticketDetail.assignFailed'));
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
      antMessage.success(t('ticketDetail.editSuccess'));
      setEditModalVisible(false);
      fetchTicket();
    } catch (error) {
      handleError(error, 'updateTicket', t('ticketDetail.editFailed'));
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
      antMessage.success(t('ticketDetail.deleteSuccess'));
      setDeleteModalVisible(false);
      // Navigate back to ticket list
      window.location.href = '/tickets';
    } catch (error) {
      handleError(error, 'deleteTicket', t('ticketDetail.deleteFailed'));
    } finally {
      setDeleting(false);
    }
  };

  // 工单操作快捷键：Alt+R 刷新，Alt+E 编辑，Esc 关闭当前弹窗。
  useEffect(() => {
    const handleShortcut = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      if (target?.closest('input, textarea, select, [contenteditable="true"]')) return;

      if (event.key === 'Escape') {
        setAssignModalVisible(false);
        setEditModalVisible(false);
        setCCModalVisible(false);
        setDeleteModalVisible(false);
        return;
      }
      if (!event.altKey) return;
      if (event.key.toLowerCase() === 'r') {
        event.preventDefault();
        fetchTicket();
      } else if (event.key.toLowerCase() === 'e' && ticket && !isTicketFinal) {
        event.preventDefault();
        editForm.setFieldsValue({
          title: ticket.title,
          description: ticket.description,
          priority: ticket.priority,
          status: ticket.status,
        });
        setEditModalVisible(true);
      }
    };

    window.addEventListener('keydown', handleShortcut);
    return () => window.removeEventListener('keydown', handleShortcut);
  }, [editForm, fetchTicket, isTicketFinal, ticket]);

  if (loading) {
    return (
      <div className="p-6">
        <Card>
          <Skeleton active title={{ width: '45%' }} paragraph={{ rows: 10 }} />
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
              {t('ticketDetail.loadFailed')}
            </Title>
            <Text type="secondary">{error}</Text>
            <div className="mt-4">
              <Button type="primary" onClick={fetchTicket}>
                {t('ticketDetail.retry')}
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
              {t('ticketDetail.notFound')}
            </Title>
            <Text type="secondary">{t('ticketDetail.notFoundDesc')}</Text>
            <div className="mt-4">
              <Link href="/tickets">
                <Button type="primary">{t('ticketDetail.backToList')}</Button>
              </Link>
            </div>
          </div>
        </Card>
      </div>
    );
  }

  return (
    <div className="p-4 sm:p-6">
      {/* Page header */}
      <div className="mb-6">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between mb-4">
          <div className="flex min-w-0 items-start sm:items-center space-x-2 sm:space-x-4">
            <Link href="/tickets">
              <Button icon={<ArrowLeft />} type="text">
                {t('common.back')}
              </Button>
            </Link>
            <div className="min-w-0">
              <Title level={2} className="!mb-1 !text-gray-900">
                {t('ticketDetail.title')} #{ticket.id}
              </Title>
              <Text type="secondary" className="block truncate">{ticket.title}</Text>
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-2">
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
              {getPriorityText(ticket.priority)}
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
              const updated = await TicketApi.updateTicket(ticketId, {
                category: suggestion.category,
                priority: toTicketPriority(suggestion.priority),
                version: ticket.version,
              } as any);
              antMessage.success(
                `已采纳AI建议：分类 ${suggestion.category}，优先级 ${suggestion.priority}`,
              );
              // Update local state immediately with the server response, then refetch in background
              if (updated && (updated as any).id) {
                setTicket(prev => (prev ? { ...prev, ...(updated as Partial<Ticket>) } : prev));
              }
              await fetchTicket();
            } catch (err) {
              handleError(err, 'applyAISuggestion', '采纳建议失败');
            }
          }}
        />
      )}

      {/* AI-Native：受影响配置项（工单→CI 反向查询，复用本体链路外键） */}
      <Card
        className="rounded-lg shadow-sm border border-gray-200"
        title={
          <Space>
            <Tag color="purple">AI-Native</Tag>
            <span>受影响配置项</span>
            {cis.length > 0 && <Tag>{cis.length}</Tag>}
          </Space>
        }
      >
        {cisLoading ? (
          <Skeleton active paragraph={{ rows: 2 }} />
        ) : cis.length === 0 ? (
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description="该工单尚未关联配置项"
          />
        ) : (
          <List
            size="small"
            dataSource={cis}
            renderItem={(ci) => (
              <List.Item
                actions={[
                  <Link key="open" href={`/cmdb/cis/${ci.id}`}>
                    查看
                  </Link>,
                ]}
              >
                <List.Item.Meta
                  title={
                    <Space>
                      <Link href={`/cmdb/cis/${ci.id}`}>{ci.name}</Link>
                      <Tag color="blue">{ci.ciType}</Tag>
                    </Space>
                  }
                  description={
                    <Space size={4} wrap>
                      <Text type="secondary">#{ci.id}</Text>
                      {ci.serialNumber ? <Text type="secondary">SN: {ci.serialNumber}</Text> : null}
                      <Tag color={ci.status === 'active' ? 'green' : 'default'}>{ci.status}</Tag>
                    </Space>
                  }
                />
              </List.Item>
            )}
          />
        )}
      </Card>

      <Card className="rounded-lg shadow-sm border border-gray-200">
        <Space orientation="vertical" size={16} style={{ width: '100%' }}>
          <Descriptions column={2} bordered size="middle">
            <Descriptions.Item label={t('ticketDetail.labelTitle')}>{ticket.title}</Descriptions.Item>
            <Descriptions.Item label={t('ticketDetail.labelNumber')}>{ticket.ticketNumber || '-'}</Descriptions.Item>
            <Descriptions.Item label={t('ticketDetail.labelStatus')}>{statusMap[ticket.status]?.text || ticket.status}</Descriptions.Item>
            <Descriptions.Item label={t('ticketDetail.labelPriority')}>{getPriorityText(ticket.priority)}</Descriptions.Item>
            <Descriptions.Item label={t('ticketDetail.labelCreatedAt')}>{formatDateTime(ticket.createdAt)}</Descriptions.Item>
            <Descriptions.Item label={t('ticketDetail.labelUpdatedAt')}>{formatDateTime(ticket.updatedAt)}</Descriptions.Item>
            <Descriptions.Item label={t('ticketDetail.labelDescription')} span={2}>
              <SafeTextBlock content={ticket.description} fallback={t('ticketDetail.labelNoDescription')} />
            </Descriptions.Item>
          </Descriptions>

          {/* 动态表单字段（来自 tenant 工单类型配置） */}
          {ticket.formFields && Object.keys(ticket.formFields).length > 0 && (
            <Card size="small" title={t('ticketDetail.formFields') || '动态表单字段'} className="mt-2">
              <Descriptions column={2} bordered size="small">
                {Object.entries(ticket.formFields).map(([key, value]) => (
                  <Descriptions.Item key={key} label={key}>
                    {renderFormFieldValue(value)}
                  </Descriptions.Item>
                ))}
              </Descriptions>
            </Card>
          )}

          {/* SLA Information */}
          {slaInfo && (
            <Card size="small" title={t('ticketDetail.slaInfo')} className="mt-4">
              <Space orientation="vertical" style={{ width: '100%' }}>
                <div className="flex justify-between">
                  <Text type="secondary">{t('ticketDetail.slaDefinition')}:</Text>
                  <Tag color={slaInfo.isBreached ? 'red' : 'blue'}>{slaInfo.slaName}</Tag>
                </div>
                {slaInfo.responseDeadline && (
                  <div className="flex justify-between">
                    <Text type="secondary">{t('ticketDetail.responseDeadline')}:</Text>
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
                        ` (${t('ticketDetail.responseTimeout')})`}
                    </Text>
                  </div>
                )}
                {slaInfo.resolutionDeadline && (
                  <div className="flex justify-between">
                    <Text type="secondary">{t('ticketDetail.resolutionDeadline')}:</Text>
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
                        ` (${t('ticketDetail.resolutionTimeout')})`}
                    </Text>
                  </div>
                )}
                {slaInfo.isBreached && <Tag color="red">{t('ticketDetail.slaBreached')}</Tag>}
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
                isRequester ? t('ticketDetail.cannotApproveOwnTicket') : isTicketFinal ? t('ticketDetail.ticketFinalNoAction') : ''
              }
            >
              {t('ticketDetail.approve')}
            </Button>
            <Button
              danger
              icon={<XIcon size={16} />}
              onClick={handleReject}
              loading={rejecting}
              disabled={isRequester || isTicketFinal}
              title={
                isRequester ? t('ticketDetail.cannotApproveOwnTicket') : isTicketFinal ? t('ticketDetail.ticketFinalNoAction') : ''
              }
            >
              {t('ticketDetail.reject')}
            </Button>
            <Button
              icon={<UserCheck size={16} />}
              onClick={handleAssign}
              loading={loadingUsers}
              disabled={isTicketFinal}
              title={isTicketFinal ? t('ticketDetail.ticketFinalNoAssign') : ''}
            >
              {t('ticketDetail.assign')}
            </Button>
            <Button
              icon={<Edit size={16} />}
              onClick={handleUpdate}
              disabled={isTicketFinal}
              title={isTicketFinal ? t('ticketDetail.ticketFinalNoEdit') : ''}
            >
              {t('common.edit')}
            </Button>
            <Button
              icon={<Users size={16} />}
              onClick={() => setCCModalVisible(true)}
              disabled={isTicketFinal}
              title={isTicketFinal ? t('ticketDetail.ticketFinalNoCc') : ''}
            >
              {t('ticketDetail.cc')}
            </Button>
            <Button danger icon={<Trash2 size={16} />} onClick={handleDeleteClick}>
              {t('common.delete')}
            </Button>
          </Space>

          {isTicketFinal && (
            <Text type="secondary" className="block mt-2">
              {t('ticketDetail.ticketFinalHint')}
            </Text>
          )}

          {isRequester && !isTicketFinal && (
            <Text type="secondary" className="block mt-2">
              {t('ticketDetail.requesterCannotApprove')}
            </Text>
          )}

          {!isRequester && !isTicketFinal && (
            <Text type="secondary">{t('ticketDetail.fullOperationSupport')}</Text>
          )}
        </Space>

        {/* Assignment Modal */}
        <Modal
          title={
            <Space>
              <UserCheck className="w-5 h-5 text-blue-600" />
              {t('ticketDetail.assignTitle')}
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
              label={t('ticketDetail.assignTo')}
              name="assigneeId"
              rules={[{ required: true, message: t('ticketDetail.assigneeRequired') }]}
            >
              <Select
                placeholder={t('ticketDetail.selectAssignee')}
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
            <Form.Item label={t('ticketDetail.remark')} name="comment">
              <TextArea rows={3} placeholder={t('ticketDetail.assignRemarkPlaceholder')} maxLength={500} showCount />
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
                  {t('common.cancel')}
                </Button>
                <Button type="primary" htmlType="submit" icon={<Save />} loading={assigning}>
                  {t('ticketDetail.confirmAssign')}
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
              {t('ticketDetail.editTitle')}
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
              label={t('ticketDetail.ticketTitle')}
              name="title"
              rules={[
                { required: true, message: t('ticketDetail.ticketTitleRequired') },
                { max: 100, message: t('ticketDetail.ticketTitleMaxLength') },
              ]}
            >
              <Input placeholder={t('ticketDetail.ticketTitlePlaceholder')} />
            </Form.Item>
            <Form.Item
              label={t('ticketDetail.ticketDescription')}
              name="description"
              rules={[
                { required: true, message: t('ticketDetail.ticketDescriptionRequired') },
                { max: 2000, message: t('ticketDetail.ticketDescriptionMaxLength') },
              ]}
            >
              <TextArea rows={6} placeholder={t('ticketDetail.ticketDescriptionPlaceholder')} showCount maxLength={2000} />
            </Form.Item>
            <div className="grid grid-cols-2 gap-4">
              <Form.Item
                label={t('ticketDetail.priority')}
                name="priority"
                rules={[{ required: true, message: t('ticketDetail.priorityRequired') }]}
              >
                <Select
                  placeholder={t('ticketDetail.selectPriority')}
                  options={[
                    {
                      value: 'low',
                      label: (
                        <>
                          <Tag color="green">{t('ticketDetail.priorityLow')}</Tag>
                        </>
                      ),
                    },
                    {
                      value: 'medium',
                      label: (
                        <>
                          <Tag color="orange">{t('ticketDetail.priorityMedium')}</Tag>
                        </>
                      ),
                    },
                    {
                      value: 'high',
                      label: (
                        <>
                          <Tag color="red">{t('ticketDetail.priorityHigh')}</Tag>
                        </>
                      ),
                    },
                  ]}
                />
              </Form.Item>
              <Form.Item
                label={t('ticketDetail.status')}
                name="status"
                rules={[{ required: true, message: t('ticketDetail.statusRequired') }]}
                extra={ticket ? `${t('ticketDetail.currentStatus')}: ${statusMap[ticket.status]?.text || ticket.status}` : ''}
              >
                <Select
                  placeholder={t('ticketDetail.selectStatus')}
                  options={Object.entries(statusMap).map(([value, cfg]) => ({
                    value,
                    label: cfg.text,
                  }))}
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
                  {t('common.cancel')}
                </Button>
                <Button type="primary" htmlType="submit" icon={<Save />}>
                  {t('ticketDetail.saveChanges')}
                </Button>
              </Space>
            </Form.Item>
          </Form>
        </Modal>

        {/* Approval / Reject Modal — 走 BPMN bridge */}
        <Modal
          title={
            <Space>
              {approvalAction === 'approve' ? (
                <UserCheck className="w-5 h-5 text-green-600" />
              ) : (
                <XCircle className="w-5 h-5 text-red-600" />
              )}
              {approvalAction === 'approve'
                ? t('ticketDetail.approve') || '审批通过'
                : t('ticketDetail.reject') || '审批驳回'}
            </Space>
          }
          open={approvalModalVisible}
          onCancel={() => {
            setApprovalModalVisible(false);
            approvalForm.resetFields();
          }}
          footer={null}
          width={480}
        >
          <Form form={approvalForm} layout="vertical" onFinish={handleApprovalSubmit}>
            <Form.Item
              label={t('ticketDetail.remark') || '备注'}
              name="comment"
              rules={[
                {
                  required: approvalAction === 'reject',
                  message:
                    approvalAction === 'reject'
                      ? t('ticketDetail.assigneeRequired') || '请填写驳回原因'
                      : undefined,
                },
              ]}
            >
              <TextArea
                rows={3}
                placeholder={
                  approvalAction === 'approve'
                    ? t('ticketDetail.assignRemarkPlaceholder') || '请输入审批意见（可选）'
                    : t('ticketDetail.ccRemarkPlaceholder') || '请输入驳回原因'
                }
                maxLength={500}
                showCount
              />
            </Form.Item>
            <Form.Item className="mb-0">
              <Space className="w-full justify-end">
                <Button
                  icon={<X />}
                  onClick={() => {
                    setApprovalModalVisible(false);
                    approvalForm.resetFields();
                  }}
                >
                  {t('common.cancel')}
                </Button>
                <Button
                  type="primary"
                  danger={approvalAction === 'reject'}
                  htmlType="submit"
                  icon={approvalAction === 'approve' ? <Check /> : <XIcon />}
                  loading={approvalAction === 'approve' ? approving : rejecting}
                >
                  {approvalAction === 'approve'
                    ? t('ticketDetail.confirmAssign') || '确认通过'
                    : t('ticketDetail.confirmDelete') || '确认驳回'}
                </Button>
              </Space>
            </Form.Item>
          </Form>
        </Modal>

        {/* CC Modal */}
        <Modal
          title={
            <Space>
              <Users className="w-5 h-5 text-blue-600" />
              {t('ticketDetail.ccTitle')}
            </Space>
          }
          open={ccModalVisible}
          onCancel={() => {
            setCCModalVisible(false);
            ccForm.resetFields();
          }}
          footer={null}
          width={520}
        >
          <Form
            form={ccForm}
            layout="vertical"
            initialValues={{ notifyChannels: ['in_app'] }}
            onFinish={handleCCSubmit}
          >
            <Form.Item
              label={t('ticketDetail.ccTo')}
              name="ccUsers"
              rules={[{ required: true, message: t('ticketDetail.ccUsersRequired') }]}
            >
              <Select
                mode="multiple"
                placeholder={t('ticketDetail.selectCcUsers')}
                loading={loadingUsers}
                showSearch
                optionFilterProp="label"
                options={users.map(user => ({
                  value: user.id,
                  label: `${user.name || user.username}${user.department ? ` (${user.department})` : ''}`,
                }))}
              />
            </Form.Item>
            <Form.Item label={t('ticketDetail.notifyChannels')} name="notifyChannels">
              <Select
                mode="multiple"
                placeholder={t('ticketDetail.selectNotifyChannels')}
                options={[
                  { value: 'in_app', label: t('ticketDetail.channelInApp') },
                  { value: 'email', label: t('ticketDetail.channelEmail') },
                  { value: 'sms', label: t('ticketDetail.channelSms') },
                  { value: 'feishu', label: t('ticketDetail.channelFeishu') },
                  { value: 'dingtalk', label: t('ticketDetail.channelDingtalk') },
                  { value: 'wecom', label: t('ticketDetail.channelWecom') },
                  { value: 'webhook', label: t('ticketDetail.channelWebhook') },
                ]}
              />
            </Form.Item>
            <Form.Item label={t('ticketDetail.remark')} name="comment">
              <TextArea rows={3} placeholder={t('ticketDetail.ccRemarkPlaceholder')} maxLength={500} showCount />
            </Form.Item>
            <Form.Item className="mb-0">
              <Space className="w-full justify-end">
                <Button
                  icon={<X />}
                  onClick={() => {
                    setCCModalVisible(false);
                    ccForm.resetFields();
                  }}
                >
                  {t('common.cancel')}
                </Button>
                <Button type="primary" htmlType="submit" icon={<Users />} loading={ccing}>
                  {t('ticketDetail.confirmCc')}
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
              {t('ticketDetail.deleteTitle')}
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
                  {t('ticketDetail.deleteConfirm')}
                </Typography.Text>
                <Typography.Paragraph type="secondary" className="mb-0 mt-1">
                  {t('ticketDetail.deleteWarning', { id: ticket.id })}
                </Typography.Paragraph>
              </div>
            </div>
            <div className="bg-gray-50 rounded p-3 mb-4">
              <Typography.Text type="secondary" className="text-sm">
                {t('ticketDetail.ticketInfo')}：
              </Typography.Text>
              <div className="mt-1">
                <Text strong>{ticket.title}</Text>
              </div>
            </div>
          </div>
          <Space className="w-full justify-end">
            <Button onClick={() => setDeleteModalVisible(false)} disabled={deleting}>
              {t('common.cancel')}
            </Button>
            <Button
              danger
              type="primary"
              onClick={handleDeleteConfirm}
              loading={deleting}
              icon={<Trash2 size={14} />}
            >
              {t('ticketDetail.confirmDelete')}
            </Button>
          </Space>
        </Modal>
      </Card>

      {/* 流转进度卡片（工单三件套改造）：详情页头部下方、详情 Tabs 上方。
          hidden 在 isTicketFinal 时折叠，避免无效请求。 */}
      <WorkflowProgressCard
        ticketId={ticketId}
        hidden={isTicketFinal}
        onRefresh={fetchTicket}
      />

      {/* 详情 Tabs（评论/附件/审批/历史/关联） */}
      <TicketDetailTabs
        ticketId={ticketId}
        ticketNumber={ticket.ticketNumber}
        ticketType={ticket.type as string | undefined}
        ticketPriority={ticket.priority as string | undefined}
        currentUserId={currentUser?.id}
        isTicketFinal={isTicketFinal}
        onRefresh={fetchTicket}
        t={t}
      />
    </div>
  );
};

// ==================== 详情 Tabs 子组件 ====================

interface TicketDetailTabsProps {
  ticketId: number;
  ticketNumber?: string;
  ticketType?: string;
  ticketPriority?: string;
  currentUserId?: number;
  isTicketFinal: boolean;
  onRefresh: () => void;
  t: (key: string) => string;
}

const TicketDetailTabs: React.FC<TicketDetailTabsProps> = ({
  ticketId,
  ticketNumber,
  ticketType,
  ticketPriority,
  currentUserId,
  isTicketFinal,
  onRefresh,
  t,
}) => {
  const items = [
    {
      key: 'comments',
      label: (
        <span>
          <MessageSquare size={14} className="inline mr-1" />
          {t('ticketDetail.tabComments')}
        </span>
      ),
      children: (
        <CommentPanel
          targetType="ticket"
          targetId={ticketId}
          adapter={ticketCommentAdapter}
          currentUserId={currentUserId}
          formatDateTime={formatDateTime}
        />
      ),
    },
    {
      key: 'attachments',
      label: (
        <span>
          <Paperclip size={14} className="inline mr-1" />
          {t('ticketDetail.tabAttachments')}
        </span>
      ),
      children: (
        <AttachmentPanel
          targetType="ticket"
          targetId={ticketId}
          adapter={ticketAttachmentAdapter}
          currentUserId={currentUserId}
          formatDateTime={formatDateTime}
        />
      ),
    },
    {
      key: 'approvals',
      label: (
        <span>
          <GitBranch size={14} className="inline mr-1" />
          {t('ticketDetail.tabApprovals')}
        </span>
      ),
      children: (
        <ApprovalWorkflowPanel
          ticketId={ticketId}
          ticketType={ticketType}
          priority={ticketPriority}
          currentUserId={currentUserId}
          isTicketFinal={isTicketFinal}
          onRefresh={onRefresh}
          formatDateTime={formatDateTime}
        />
      ),
    },
    {
      key: 'history',
      label: (
        <span>
          <HistoryIcon size={14} className="inline mr-1" />
          {t('ticketDetail.tabHistory')}
        </span>
      ),
      children: (
        <HistoryTimeline
          targetType="ticket"
          targetId={ticketId}
          fetchHistory={async (id) => {
            const list = await TicketApi.getTicketHistory(Number(id));
            const arr = Array.isArray(list) ? list : [];
            // 后端返回活动流水（camelCase：id/action/details/createdAt/userName/oldValue/newValue）
            return arr.map((raw, idx) => {
              const r = raw as unknown as Record<string, unknown>;
              return {
                id: (r.id as number | undefined) ?? idx,
                createdAt: String(r.createdAt ?? r.timestamp ?? ''),
                user: { name: String(r.userName ?? r.user_name ?? '') },
                action: r.action as string | undefined,
                details: r.details as string | undefined,
                fieldName: r.fieldName as string | undefined,
                oldValue:
                  r.oldValue !== undefined && r.oldValue !== null
                    ? String(r.oldValue)
                    : undefined,
                newValue:
                  r.newValue !== undefined && r.newValue !== null
                    ? String(r.newValue)
                    : undefined,
                changeReason: r.changeReason as string | undefined,
              };
            });
          }}
          fetchAuditLog={fetchAuditLogHistory}
          formatDateTime={formatDateTime}
        />
      ),
    },
    {
      key: 'relations',
      label: (
        <span>
          <Link2 size={14} className="inline mr-1" />
          {t('ticketDetail.tabRelations')}
        </span>
      ),
      children: (
        <div className="p-6">
          <RelationPanel ticketId={ticketId} ticketNumber={ticketNumber || String(ticketId)} />
        </div>
      ),
    },
  ];

  return (
    <Card className="mt-4 rounded-lg shadow-sm border border-gray-200">
      <div className="flex items-center gap-2 mb-2 px-2 pt-2 text-gray-500 text-sm">
        <Info size={14} />
        {t('ticketDetail.detailTabsTitle')}
      </div>
      <Tabs items={items} defaultActiveKey="comments" />
    </Card>
  );
};

export default TicketDetail;
