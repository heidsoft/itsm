'use client';

/**
 * 审批中心首页
 * 以 BPMN 流程待办为权威审批入口（GET /bpmn/tasks + POST /bpmn/tasks/:id/decisions），
 * 旧的工单/变更/服务请求业务列表保留为只读参考视图。
 */

import React, { useEffect, useState, useCallback } from 'react';
import {
  App,
  Card,
  Tabs,
  Table,
  Tag,
  Button,
  Space,
  Input,
  Modal,
  Row,
  Col,
  Typography,
  Tooltip,
  Badge,
  Skeleton,
  Select,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  CheckCircle,
  Clock,
  RotateCcw,
  FileText,
  Wrench,
  Headphones,
  Check,
  X,
  Hand,
  GitBranch,
  ExternalLink,
  ArrowRightLeft,
  UserPlus,
} from 'lucide-react';
import Link from 'next/link';
import { httpClient } from '@/lib/api/http-client';
import { WorkflowApi, type BpmnMyTask } from '@/lib/api/workflow-api';
import { UserApi, type User } from '@/lib/api/user-api';
import { useAuthStore } from '@/lib/store/auth-store';
import { useI18n } from '@/lib/i18n/useI18n';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';

dayjs.extend(relativeTime);

const { Title, Text } = Typography;
const { TextArea } = Input;

// ==================== BPMN 待办 ====================

// 待处理任务状态（后端 status：created/assigned/started 均视为待办）
const PENDING_TASK_STATUSES = new Set(['created', 'assigned', 'started', 'pending']);

const taskStatusColorMap: Record<string, string> = {
  created: 'gold',
  assigned: 'blue',
  started: 'processing',
  pending: 'gold',
  completed: 'green',
  cancelled: 'default',
};

const taskStatusKeyMap: Record<string, string> = {
  created: 'statusCreated',
  assigned: 'statusAssigned',
  started: 'statusStarted',
  pending: 'statusPending',
  completed: 'statusCompleted',
  cancelled: 'statusCancelled',
};

const taskPurposeKeyMap: Record<string, string> = {
  approval: 'purposeApproval',
  work: 'purposeWork',
  rework: 'purposeRework',
};

// 业务类型 → i18n key + 详情路由
const businessTypeMap: Record<string, { labelKey: string; url: (id: number) => string }> = {
  ticket: { labelKey: 'bizTicket', url: (id) => `/tickets/${id}` },
  change: { labelKey: 'bizChange', url: (id) => `/changes/${id}` },
  incident: { labelKey: 'bizIncident', url: (id) => `/incidents/${id}` },
  problem: { labelKey: 'bizProblem', url: (id) => `/problems/${id}` },
  service_request: { labelKey: 'bizServiceRequest', url: (id) => `/service-requests/${id}` },
};

// 引擎按 taskPurpose 决定任务语义：未声明一律按 work 处理（P0-1 D2），
// 因此只有显式 approval 才使用「批准」文案，其余按「完成」呈现。
function isApprovalTask(task?: BpmnMyTask): boolean {
  return (task?.taskPurpose || '').toLowerCase() === 'approval';
}

// ==================== 旧业务参考视图（只读） ====================

interface LegacyPendingItem {
  id: number | string;
  type: 'ticket' | 'change' | 'service_request';
  title: string;
  status: string;
  priority?: string;
  createdAt: string;
  url: string;
  requester?: string;
}

interface RawListItem {
  id?: number | string;
  title?: string;
  status?: string;
  priority?: string;
  createdAt?: string;
  scheduledStart?: string;
  requesterName?: string;
}

const priorityColorMap: Record<string, string> = {
  critical: 'red',
  high: 'orange',
  medium: 'blue',
  low: 'green',
};

const legacyTypeMeta: Record<LegacyPendingItem['type'], { labelKey: string; icon: React.ReactNode }> = {
  ticket: { labelKey: 'bizTicket', icon: <FileText className="w-3 h-3 inline mr-1" /> },
  change: { labelKey: 'bizChange', icon: <Wrench className="w-3 h-3 inline mr-1" /> },
  service_request: { labelKey: 'bizServiceRequest', icon: <Headphones className="w-3 h-3 inline mr-1" /> },
};

export default function ApprovalsCenterPage() {
  const { message } = App.useApp();
  const { t } = useI18n();
  const { user } = useAuthStore();

  // BPMN 待办
  const [taskLoading, setTaskLoading] = useState(false);
  const [tasks, setTasks] = useState<BpmnMyTask[]>([]);
  const [claiming, setClaiming] = useState<number | null>(null);
  const [decision, setDecision] = useState<{
    task: BpmnMyTask;
    action: 'approve' | 'reject';
  } | null>(null);
  const [decisionComment, setDecisionComment] = useState('');
  const [submitting, setSubmitting] = useState(false);

  // 委托
  const [delegation, setDelegation] = useState<{ task: BpmnMyTask } | null>(null);
  const [delegateToUserId, setDelegateToUserId] = useState<number | undefined>(undefined);
  const [userOptions, setUserOptions] = useState<User[]>([]);
  const [userLoading, setUserLoading] = useState(false);

  // 加签
  const [addApprover, setAddApprover] = useState<{ task: BpmnMyTask } | null>(null);
  const [addApproverUserId, setAddApproverUserId] = useState<number | undefined>(undefined);
  const [addApproverUserOptions, setAddApproverUserOptions] = useState<User[]>([]);
  const [addApproverUserLoading, setAddApproverUserLoading] = useState(false);

  // 旧业务参考视图
  const [legacyLoading, setLegacyLoading] = useState(false);
  const [legacyItems, setLegacyItems] = useState<LegacyPendingItem[]>([]);
  const [activeTab, setActiveTab] = useState('bpmn-tasks');

  const getBusinessLink = useCallback(
    (task: BpmnMyTask): { label: string; url: string } | null => {
      if (task.businessType && task.businessId) {
        const meta = businessTypeMap[task.businessType];
        if (meta) {
          return { label: `${t(`approvalCenter.${meta.labelKey}`)} #${task.businessId}`, url: meta.url(task.businessId) };
        }
      }
      // 流程实例统一以 PI-* 业务键寻址（processInstanceKey）；数字 Ent ID 不再是任何端点的寻址键。
      if (task.processInstanceKey) {
        return {
          label: t('approvalCenter.processInstanceRef', { key: task.processInstanceKey }),
          url: `/workflow/instances/${task.processInstanceKey}`,
        };
      }
      return null;
    },
    [t]
  );

  const statusTag = useCallback(
    (status: string) => {
      const key = (status || '').toLowerCase();
      const labelKey = taskStatusKeyMap[key];
      return (
        <Tag color={taskStatusColorMap[key] || 'default'}>
          {labelKey ? t(`approvalCenter.${labelKey}`) : status}
        </Tag>
      );
    },
    [t]
  );

  const loadTasks = useCallback(async () => {
    setTaskLoading(true);
    try {
      const res = await WorkflowApi.listMyApprovalTasks({ page: 1, pageSize: 100 });
      // 后端仅支持单状态过滤，这里客户端过滤出待处理任务
      setTasks(res.items.filter((t) => PENDING_TASK_STATUSES.has((t.status || '').toLowerCase())));
    } catch {
      message.error(t('approvalCenter.loadTasksFailed'));
    } finally {
      setTaskLoading(false);
    }
  }, [message, t]);

  const loadLegacy = useCallback(async () => {
    setLegacyLoading(true);
    try {
      // 查询参数统一 camelCase，与后端 DTO form tag 契约一致。
      // 三个列表接口都返回标准信封，集合只在 data.items 下。
      const [ticketsResp, changesResp, srResp] = await Promise.all([
        httpClient.get<{ items?: RawListItem[] }>('/api/v1/tickets', { status: 'pending', page: 1, pageSize: 20 }).catch(() => ({ items: [] as RawListItem[] })),
        httpClient.get<{ items?: RawListItem[] }>('/api/v1/changes', { status: 'pending', page: 1, pageSize: 20 }).catch(() => ({ items: [] as RawListItem[] })),
        httpClient.get<{ items?: RawListItem[] }>('/api/v1/service-requests', { status: 'pending', page: 1, pageSize: 20 }).catch(() => ({ items: [] as RawListItem[] })),
      ]);
      const items: LegacyPendingItem[] = [
        ...(ticketsResp.items ?? []).filter((it) => it.id != null).map((it) => ({
          id: it.id!,
          type: 'ticket' as const,
          title: it.title || `${t('approvalCenter.bizTicket')} #${it.id}`,
          status: it.status || 'unknown',
          priority: it.priority,
          createdAt: it.createdAt || '',
          url: `/tickets/${it.id}`,
          requester: it.requesterName,
        })),
        ...(changesResp.items ?? []).filter((c) => c.id != null).map((c) => ({
          id: c.id!,
          type: 'change' as const,
          title: c.title || `${t('approvalCenter.bizChange')} #${c.id}`,
          status: c.status || 'unknown',
          priority: c.priority,
          createdAt: c.scheduledStart || c.createdAt || '',
          url: `/changes/${c.id}`,
          requester: c.requesterName,
        })),
        ...(srResp.items ?? []).filter((s) => s.id != null).map((s) => ({
          id: s.id!,
          type: 'service_request' as const,
          title: s.title || `${t('approvalCenter.bizServiceRequest')} #${s.id}`,
          status: s.status || 'unknown',
          priority: s.priority,
          createdAt: s.createdAt || '',
          url: `/service-requests/${s.id}`,
          requester: s.requesterName,
        })),
      ];
      setLegacyItems(items);
    } catch {
      message.error(t('approvalCenter.loadLegacyFailed'));
    } finally {
      setLegacyLoading(false);
    }
  }, [message, t]);

  useEffect(() => {
    loadTasks();
    loadLegacy();
  }, [loadTasks, loadLegacy]);

  const handleRefresh = () => {
    loadTasks();
    loadLegacy();
  };

  // 领取任务（无负责人时）
  const handleClaim = async (task: BpmnMyTask) => {
    setClaiming(task.id);
    try {
      await WorkflowApi.claimMyTask(task.id);
      message.success(t('approvalCenter.claimed'));
      loadTasks();
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('approvalCenter.claimFailed'));
    } finally {
      setClaiming(null);
    }
  };

  // 打开审批决策弹窗
  const openDecision = (task: BpmnMyTask, action: 'approve' | 'reject') => {
    setDecisionComment('');
    setDecision({ task, action });
  };

  // 提交审批决策：批准可不填意见，拒绝必须填写（与后端校验一致）
  const submitDecision = async () => {
    if (!decision) return;
    const comment = decisionComment.trim();
    if (decision.action === 'reject' && !comment) {
      message.warning(t('approvalCenter.rejectCommentRequired'));
      return;
    }
    setSubmitting(true);
    try {
      await WorkflowApi.submitTaskDecision(decision.task.id, {
        action: decision.action,
        comment,
      });
      if (decision.action === 'reject') {
        message.success(t('approvalCenter.rejectedDone'));
      } else {
        message.success(
          isApprovalTask(decision.task) ? t('approvalCenter.approvedDone') : t('approvalCenter.completedDone')
        );
      }
      setDecision(null);
      loadTasks();
      loadLegacy();
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('approvalCenter.decisionFailed'));
    } finally {
      setSubmitting(false);
    }
  };

  const openDelegation = async (task: BpmnMyTask) => {
    setDelegation({ task });
    setDelegateToUserId(undefined);
    setUserLoading(true);
    try {
      const res = await UserApi.getUsers({ page: 1, pageSize: 200 });
      setUserOptions(res.items.filter((u) => u.id !== user?.id && u.active));
    } catch {
      message.error(t('approvalCenter.usersLoadFailed'));
    } finally {
      setUserLoading(false);
    }
  };

  const submitDelegation = async () => {
    if (!delegation || !delegateToUserId) return;
    setSubmitting(true);
    try {
      await WorkflowApi.submitTaskDecision(delegation.task.id, {
        action: 'delegate',
        delegateToUserId,
      });
      message.success(t('approvalCenter.delegated'));
      setDelegation(null);
      loadTasks();
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('approvalCenter.delegateFailed'));
    } finally {
      setSubmitting(false);
    }
  };

  const openAddApprover = async (task: BpmnMyTask) => {
    setAddApprover({ task });
    setAddApproverUserId(undefined);
    setAddApproverUserLoading(true);
    try {
      const res = await UserApi.getUsers({ page: 1, pageSize: 200 });
      setAddApproverUserOptions(res.items.filter((u) => u.id !== user?.id && u.active));
    } catch {
      message.error(t('approvalCenter.usersLoadFailed'));
    } finally {
      setAddApproverUserLoading(false);
    }
  };

  const submitAddApprover = async () => {
    if (!addApprover || !addApproverUserId) return;
    setSubmitting(true);
    try {
      await WorkflowApi.submitTaskDecision(addApprover.task.id, {
        action: 'add_approver',
        addApproverUserId,
      });
      message.success(t('approvalCenter.addApproverDone'));
      setAddApprover(null);
      loadTasks();
    } catch (e) {
      message.error(e instanceof Error ? e.message : t('approvalCenter.addApproverFailed'));
    } finally {
      setSubmitting(false);
    }
  };

  // BPMN 待办表格列
  const taskColumns: ColumnsType<BpmnMyTask> = [
    {
      title: t('approvalCenter.colTask'),
      dataIndex: 'taskName',
      key: 'taskName',
      // 必须给显式宽度：其余列都是固定宽，scroll.x 又小于这些宽度之和，
      // 唯一自适应的一列会被压到接近 0px，中文按单字换行（2026-10-25 实测
      // 审批中心出现「任/务」「请/求/受/理」竖排）。
      width: 160,
      render: (text: string, record: BpmnMyTask) => {
        const purposeKey = taskPurposeKeyMap[(record.taskPurpose || '').toLowerCase()];
        return (
          <div>
            <div className="font-medium text-gray-900">{text || record.taskDefinitionKey}</div>
            {purposeKey && (
              <Text type="secondary" className="text-xs">{t(`approvalCenter.${purposeKey}`)}</Text>
            )}
          </div>
        );
      },
    },
    {
      title: t('approvalCenter.colBusiness'),
      key: 'business',
      width: 180,
      render: (_: unknown, record: BpmnMyTask) => {
        const link = getBusinessLink(record);
        if (!link) return <Text type="secondary">-</Text>;
        return (
          <Link href={link.url} className="text-blue-600 hover:text-blue-700 inline-flex items-center gap-1">
            {link.label}
            <ExternalLink className="w-3 h-3" />
          </Link>
        );
      },
    },
    {
      title: t('approvalCenter.colProcess'),
      dataIndex: 'processDefinitionKey',
      key: 'processDefinitionKey',
      width: 180,
      responsive: ['lg'],
      render: (key: string) => key ? <Tag icon={<GitBranch className="w-3 h-3 inline mr-1" />}>{key}</Tag> : '-',
    },
    {
      title: t('approvalCenter.colStatus'),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => statusTag(status),
    },
    {
      title: t('approvalCenter.colOwner'),
      dataIndex: 'assignee',
      key: 'assignee',
      // 「系统管理员」这类 5 字姓名在 110px 里会折成三行，放宽到 130 并允许省略
      width: 130,
      ellipsis: true,
      responsive: ['md'],
      render: (assignee: string, record: BpmnMyTask) =>
        record.assigneeName ||
        (assignee ? t('approvalCenter.ownerUnknown') : <Text type="secondary">{t('approvalCenter.ownerUnclaimed')}</Text>),
    },
    {
      title: t('approvalCenter.colCreated'),
      dataIndex: 'createdTime',
      key: 'createdTime',
      width: 130,
      responsive: ['xl'],
      render: (value: string) => value ? (
        <Tooltip title={dayjs(value).format('YYYY-MM-DD HH:mm:ss')}>
          <span className="text-gray-500">{dayjs(value).fromNow()}</span>
        </Tooltip>
      ) : '-',
    },
    {
      title: t('approvalCenter.colActions'),
      key: 'action',
      width: 280,
      // 审批页的主操作（完成/驳回/领取）必须常驻可见：列宽改到 1160 之后
      // 表格会横向滚动，不固定这一列就要先滚到最右才能处理待办。
      fixed: 'right',
      render: (_: unknown, record: BpmnMyTask) => {
        const approval = isApprovalTask(record);
        return (
          <Space size="small">
            {!record.assignee && (
              <Button
                size="small"
                icon={<Hand className="w-3 h-3" />}
                loading={claiming === record.id}
                onClick={() => handleClaim(record)}
              >
                {t('approvalCenter.claim')}
              </Button>
            )}
            <Button
              type="primary"
              size="small"
              icon={<Check className="w-3 h-3" />}
              onClick={() => openDecision(record, 'approve')}
              className="!bg-green-500 !border-green-500 hover:!bg-green-600 hover:!border-green-600"
            >
              {approval ? t('approvalCenter.approve') : t('approvalCenter.complete')}
            </Button>
            <Button
              danger
              size="small"
              icon={<X className="w-3 h-3" />}
              onClick={() => openDecision(record, 'reject')}
            >
              {t('approvalCenter.reject')}
            </Button>
            {!!((record.taskVariables as Record<string, unknown>)?.allowDelegate) && (
              <Button
                size="small"
                icon={<ArrowRightLeft className="w-3 h-3" />}
                onClick={() => openDelegation(record)}
              >
                {t('approvalCenter.delegate')}
              </Button>
            )}
            {!!((record.taskVariables as Record<string, unknown>)?.allowAddApprover) && (
              <Button
                size="small"
                icon={<UserPlus className="w-3 h-3" />}
                onClick={() => openAddApprover(record)}
              >
                {t('approvalCenter.addApprover')}
              </Button>
            )}
          </Space>
        );
      },
    },
  ];

  // 旧业务参考视图表格列（只读，仅提供跳转）
  const legacyColumns: ColumnsType<LegacyPendingItem> = [
    {
      title: t('approvalCenter.colType'),
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type: LegacyPendingItem['type']) => {
        const meta = legacyTypeMeta[type];
        return <Tag>{meta.icon}{t(`approvalCenter.${meta.labelKey}`)}</Tag>;
      },
    },
    {
      title: t('approvalCenter.colTitle'),
      dataIndex: 'title',
      key: 'title',
      render: (text: string, record: LegacyPendingItem) => (
        <Link href={record.url} className="font-medium text-gray-900 hover:text-blue-600 transition-colors">
          {text}
        </Link>
      ),
    },
    {
      title: t('approvalCenter.colPriority'),
      dataIndex: 'priority',
      key: 'priority',
      width: 90,
      render: (p: string) => p ? <Tag color={priorityColorMap[p] || 'default'}>{p.toUpperCase()}</Tag> : '-',
    },
    {
      title: t('approvalCenter.colStatus'),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (s: string) => <Tag color="gold">{s}</Tag>,
    },
    {
      title: t('approvalCenter.colRequester'),
      dataIndex: 'requester',
      key: 'requester',
      width: 110,
      responsive: ['md'],
      render: (name: string) => name || '-',
    },
    {
      title: t('approvalCenter.colCreated'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 130,
      responsive: ['lg'],
      render: (value: string) => value ? (
        <Tooltip title={dayjs(value).format('YYYY-MM-DD HH:mm:ss')}>
          <span className="text-gray-500">{dayjs(value).fromNow()}</span>
        </Tooltip>
      ) : '-',
    },
    {
      title: t('approvalCenter.colActions'),
      key: 'action',
      width: 90,
      render: (_: unknown, record: LegacyPendingItem) => (
        <Link href={record.url}>
          <Button size="small" icon={<ExternalLink className="w-3 h-3" />}>{t('approvalCenter.view')}</Button>
        </Link>
      ),
    },
  ];

  const LoadingSkeleton = () => (
    <div className="space-y-4">
      {[1, 2, 3].map((i) => (
        <Skeleton key={i} active paragraph={{ rows: 2 }} />
      ))}
    </div>
  );

  const pagination = {
    pageSize: 10,
    showSizeChanger: true,
    showTotal: (total: number) => t('approvalCenter.totalCount', { total }),
  };

  const decisionIsApproval = isApprovalTask(decision?.task);
  const decisionTitle = !decision
    ? ''
    : decision.action === 'reject'
      ? t('approvalCenter.rejectTitle')
      : decisionIsApproval
        ? t('approvalCenter.approveTitle')
        : t('approvalCenter.completeTitle');
  const decisionOkText =
    decision?.action === 'reject'
      ? t('approvalCenter.confirmReject')
      : decisionIsApproval
        ? t('approvalCenter.confirmApprove')
        : t('approvalCenter.confirmComplete');

  return (
    <div className="p-4 md:p-6">
      {/* 头部区域 */}
      <div className="mb-4 md:mb-6 flex flex-col md:flex-row md:justify-between md:items-center gap-4">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 md:w-12 md:h-12 rounded-xl bg-blue-50 flex items-center justify-center">
            <CheckCircle className="text-xl md:text-2xl text-blue-500" />
          </div>
          <div>
            <Title level={3} className="!mb-0 !text-xl md:!text-2xl">{t('approvalCenter.title')}</Title>
            <Text type="secondary" className="text-sm">
              {user?.username
                ? t('approvalCenter.greetingWithName', { username: user.username, count: tasks.length })
                : t('approvalCenter.greeting', { count: tasks.length })}
            </Text>
          </div>
        </div>
        <Button
          icon={<RotateCcw className={taskLoading || legacyLoading ? 'animate-spin' : ''} />}
          onClick={handleRefresh}
          loading={taskLoading || legacyLoading}
        >
          {t('approvalCenter.refresh')}
        </Button>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[12, 12]} className="mb-4 md:mb-6">
        <Col xs={12} sm={8}>
          <Card className="border-l-4 border-l-blue-500">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-xs md:text-sm text-gray-500">{t('approvalCenter.statProcessTasks')}</div>
                <div className="text-2xl md:text-3xl font-bold text-blue-600">{tasks.length}</div>
              </div>
              <div className="w-10 h-10 md:w-12 md:h-12 rounded-lg bg-blue-50 flex items-center justify-center">
                <GitBranch className="text-lg md:text-xl text-blue-500" />
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={12} sm={8}>
          <Card className="border-l-4 border-l-gold-500">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-xs md:text-sm text-gray-500">{t('approvalCenter.statClaimable')}</div>
                <div className="text-2xl md:text-3xl font-bold text-amber-500">
                  {tasks.filter((t) => !t.assignee).length}
                </div>
              </div>
              <div className="w-10 h-10 md:w-12 md:h-12 rounded-lg bg-amber-50 flex items-center justify-center">
                <Hand className="text-lg md:text-xl text-amber-500" />
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={12} sm={8}>
          <Card className="border-l-4 border-l-purple-500">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-xs md:text-sm text-gray-500">{t('approvalCenter.statLegacy')}</div>
                <div className="text-2xl md:text-3xl font-bold text-purple-600">{legacyItems.length}</div>
              </div>
              <div className="w-10 h-10 md:w-12 md:h-12 rounded-lg bg-purple-50 flex items-center justify-center">
                <Clock className="text-lg md:text-xl text-purple-500" />
              </div>
            </div>
          </Card>
        </Col>
      </Row>

      {/* Tab 内容 */}
      <Card>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            {
              key: 'bpmn-tasks',
              label: (
                <span className="flex items-center gap-2">
                  <GitBranch /> {t('approvalCenter.tabProcessTasks')}
                  <Badge count={tasks.length} showZero={false} />
                </span>
              ),
              children: taskLoading ? (
                <LoadingSkeleton />
              ) : (
                <Table
                  rowKey="id"
                  dataSource={tasks}
                  columns={taskColumns}
                  pagination={pagination}
                  // 必须 >= 各列固定宽度之和（160+180+180+100+130+130+280=1160），
                  // 否则自适应列被压扁，中文逐字换行。
                  scroll={{ x: 1160 }}
                  locale={{
                    emptyText: (
                      <div className="py-12 text-center">
                        <Clock className="mx-auto mb-3 text-gray-300 w-10 h-10" />
                        <div className="text-gray-500 mb-1">{t('approvalCenter.emptyTasksTitle')}</div>
                        <Text type="secondary" className="text-sm">{t('approvalCenter.emptyTasksDesc')}</Text>
                      </div>
                    ),
                  }}
                />
              ),
            },
            {
              key: 'legacy',
              label: (
                <span className="flex items-center gap-2">
                  <FileText /> {t('approvalCenter.tabLegacy')}
                  <Badge count={legacyItems.length} showZero={false} color="#999" />
                </span>
              ),
              children: (
                <div>
                  <div className="mb-3 px-3 py-2 bg-amber-50 border border-amber-200 rounded text-sm text-amber-700">
                    {t('approvalCenter.legacyNotice')}
                  </div>
                  {legacyLoading ? (
                    <LoadingSkeleton />
                  ) : (
                    <Table
                      rowKey={(r) => `${r.type}-${r.id}`}
                      dataSource={legacyItems}
                      columns={legacyColumns}
                      pagination={pagination}
                      scroll={{ x: 800 }}
                    />
                  )}
                </div>
              ),
            },
          ]}
        />
      </Card>

      {/* 审批决策弹窗 */}
      <Modal
        title={decisionTitle}
        open={!!decision}
        onOk={submitDecision}
        onCancel={() => setDecision(null)}
        okText={decisionOkText}
        okButtonProps={{
          danger: decision?.action === 'reject',
          loading: submitting,
        }}
        cancelText={t('approvalCenter.cancel')}
        destroyOnHidden
      >
        {decision && (
          <div className="space-y-3">
            <div>
              <Text type="secondary">{t('approvalCenter.lblTask')}</Text>
              <Text strong>{decision.task.taskName || decision.task.taskDefinitionKey}</Text>
            </div>
            {(() => {
              const link = getBusinessLink(decision.task);
              return link ? (
                <div>
                  <Text type="secondary">{t('approvalCenter.lblBusiness')}</Text>
                  <Link href={link.url} className="text-blue-600">{link.label}</Link>
                </div>
              ) : null;
            })()}
            <div>
              <Text type="secondary">
                {decision.action === 'reject' || decisionIsApproval
                  ? t('approvalCenter.lblApprovalComment')
                  : t('approvalCenter.lblWorkComment')}
                {decision.action === 'reject'
                  ? t('approvalCenter.commentRequiredSuffix')
                  : t('approvalCenter.commentOptionalSuffix')}
              </Text>
              <TextArea
                rows={3}
                className="mt-1"
                value={decisionComment}
                onChange={(e) => setDecisionComment(e.target.value)}
                placeholder={
                  decision.action === 'reject'
                    ? t('approvalCenter.placeholderReject')
                    : decisionIsApproval
                      ? t('approvalCenter.placeholderApprove')
                      : t('approvalCenter.placeholderWork')
                }
                maxLength={500}
                showCount
              />
            </div>
          </div>
        )}
      </Modal>

      {/* 委托弹窗 */}
      <Modal
        title={t('approvalCenter.delegateTitle')}
        open={!!delegation}
        onOk={submitDelegation}
        onCancel={() => setDelegation(null)}
        okText={t('approvalCenter.confirmDelegate')}
        okButtonProps={{ disabled: !delegateToUserId, loading: submitting }}
        cancelText={t('approvalCenter.cancel')}
        destroyOnHidden
      >
        {delegation && (
          <div className="space-y-3">
            <div>
              <Text type="secondary">{t('approvalCenter.lblTask')}</Text>
              <Text strong>{delegation.task.taskName || delegation.task.taskDefinitionKey}</Text>
            </div>
            {(() => {
              const link = getBusinessLink(delegation.task);
              return link ? (
                <div>
                  <Text type="secondary">{t('approvalCenter.lblBusiness')}</Text>
                  <Link href={link.url} className="text-blue-600">{link.label}</Link>
                </div>
              ) : null;
            })()}
            <div>
              <Text type="secondary">{t('approvalCenter.lblDelegateTo')}</Text>
              <Select
                className="mt-1 w-full"
                placeholder={t('approvalCenter.placeholderSelectDelegate')}
                loading={userLoading}
                value={delegateToUserId}
                onChange={setDelegateToUserId}
                showSearch
                optionFilterProp="label"
                options={userOptions.map((u) => ({
                  value: u.id,
                  label: `${u.name}（${u.username}）`,
                }))}
                notFoundContent={userLoading ? t('approvalCenter.loading') : t('approvalCenter.noUsers')}
              />
            </div>
            <div className="text-xs text-gray-400">
              {t('approvalCenter.delegateHint')}
            </div>
          </div>
        )}
      </Modal>

      {/* 加签弹窗 */}
      <Modal
        title={t('approvalCenter.addApproverTitle')}
        open={!!addApprover}
        onOk={submitAddApprover}
        onCancel={() => setAddApprover(null)}
        okText={t('approvalCenter.confirmAddApprover')}
        okButtonProps={{ disabled: !addApproverUserId, loading: submitting }}
        cancelText={t('approvalCenter.cancel')}
        destroyOnHidden
      >
        {addApprover && (
          <div className="space-y-3">
            <div>
              <Text type="secondary">{t('approvalCenter.lblTask')}</Text>
              <Text strong>{addApprover.task.taskName || addApprover.task.taskDefinitionKey}</Text>
            </div>
            {(() => {
              const link = getBusinessLink(addApprover.task);
              return link ? (
                <div>
                  <Text type="secondary">{t('approvalCenter.lblBusiness')}</Text>
                  <Link href={link.url} className="text-blue-600">{link.label}</Link>
                </div>
              ) : null;
            })()}
            <div>
              <Text type="secondary">{t('approvalCenter.lblAddApproverTo')}</Text>
              <Select
                className="mt-1 w-full"
                placeholder={t('approvalCenter.placeholderSelectAddApprover')}
                loading={addApproverUserLoading}
                value={addApproverUserId}
                onChange={setAddApproverUserId}
                showSearch
                optionFilterProp="label"
                options={addApproverUserOptions.map((u) => ({
                  value: u.id,
                  label: `${u.name}（${u.username}）`,
                }))}
                notFoundContent={addApproverUserLoading ? t('approvalCenter.loading') : t('approvalCenter.noUsers')}
              />
            </div>
            <div className="text-xs text-gray-400">
              {t('approvalCenter.addApproverHint')}
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}
