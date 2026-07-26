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
} from 'antd';
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
} from 'lucide-react';
import Link from 'next/link';
import { httpClient } from '@/lib/api/http-client';
import { WorkflowApi, type BpmnMyTask } from '@/lib/api/workflow-api';
import { useAuthStore } from '@/lib/store/auth-store';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';

dayjs.extend(relativeTime);

const { Title, Text } = Typography;
const { TextArea } = Input;

// ==================== BPMN 待办 ====================

// 待处理任务状态（后端 status：created/assigned/started 均视为待办）
const PENDING_TASK_STATUSES = new Set(['created', 'assigned', 'started', 'pending']);

const taskStatusMap: Record<string, { text: string; color: string }> = {
  created: { text: '待领取', color: 'gold' },
  assigned: { text: '已分配', color: 'blue' },
  started: { text: '处理中', color: 'processing' },
  pending: { text: '待处理', color: 'gold' },
  completed: { text: '已完成', color: 'green' },
  cancelled: { text: '已取消', color: 'default' },
};

// 业务类型 → 展示名 + 详情路由
const businessTypeMap: Record<string, { label: string; url: (id: number) => string }> = {
  ticket: { label: '工单', url: (id) => `/tickets/${id}` },
  change: { label: '变更', url: (id) => `/changes/${id}` },
  incident: { label: '事件', url: (id) => `/incidents/${id}` },
  problem: { label: '问题', url: (id) => `/problems/${id}` },
  service_request: { label: '服务请求', url: (id) => `/service-requests/${id}` },
};

function getBusinessLink(task: BpmnMyTask): { label: string; url: string } | null {
  if (task.businessType && task.businessId) {
    const meta = businessTypeMap[task.businessType];
    if (meta) {
      return { label: `${meta.label} #${task.businessId}`, url: meta.url(task.businessId) };
    }
  }
  if (task.processInstanceId) {
    return { label: `流程实例 #${task.processInstanceId}`, url: `/workflow/instances/${task.processInstanceId}` };
  }
  return null;
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

const priorityColorMap: Record<string, string> = {
  critical: 'red',
  high: 'orange',
  medium: 'blue',
  low: 'green',
};

export default function ApprovalsCenterPage() {
  const { message, modal } = App.useApp();
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

  // 旧业务参考视图
  const [legacyLoading, setLegacyLoading] = useState(false);
  const [legacyItems, setLegacyItems] = useState<LegacyPendingItem[]>([]);
  const [activeTab, setActiveTab] = useState('bpmn-tasks');

  const loadTasks = useCallback(async () => {
    setTaskLoading(true);
    try {
      const res = await WorkflowApi.listMyApprovalTasks({ page: 1, pageSize: 100 });
      // 后端仅支持单状态过滤，这里客户端过滤出待处理任务
      setTasks(res.items.filter((t) => PENDING_TASK_STATUSES.has((t.status || '').toLowerCase())));
    } catch {
      message.error('加载流程待办失败');
    } finally {
      setTaskLoading(false);
    }
  }, [message]);

  const loadLegacy = useCallback(async () => {
    setLegacyLoading(true);
    try {
      const [ticketsResp, changesResp, srResp] = await Promise.all([
        httpClient.get<{ tickets?: any[] }>('/api/v1/tickets?status=pending&page=1&page_size=20').catch(() => ({ tickets: [] })),
        httpClient.get<{ changes?: any[] }>('/api/v1/changes?status=pending&page=1&page_size=20').catch(() => ({ changes: [] })),
        httpClient.get<{ items?: any[] }>('/api/v1/service-requests?status=pending&page=1&page_size=20').catch(() => ({ items: [] })),
      ]);
      const items: LegacyPendingItem[] = [
        ...(ticketsResp.tickets || []).map((t: any) => ({
          id: t.id,
          type: 'ticket' as const,
          title: t.title || `工单 #${t.id}`,
          status: t.status,
          priority: t.priority,
          createdAt: t.createdAt,
          url: `/tickets/${t.id}`,
          requester: t.requesterName,
        })),
        ...(changesResp.changes || []).map((c: any) => ({
          id: c.id,
          type: 'change' as const,
          title: c.title || `变更 #${c.id}`,
          status: c.status,
          priority: c.priority,
          createdAt: c.scheduledStart || c.createdAt,
          url: `/changes/${c.id}`,
          requester: c.requesterName,
        })),
        ...(srResp.items || []).map((s: any) => ({
          id: s.id,
          type: 'service_request' as const,
          title: s.title || `服务请求 #${s.id}`,
          status: s.status,
          priority: s.priority,
          createdAt: s.createdAt,
          url: `/service-requests/${s.id}`,
          requester: s.requesterName,
        })),
      ];
      setLegacyItems(items);
    } catch {
      message.error('加载业务待审列表失败');
    } finally {
      setLegacyLoading(false);
    }
  }, [message]);

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
      message.success('任务已领取');
      loadTasks();
    } catch (e) {
      message.error(e instanceof Error ? e.message : '领取任务失败');
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
      message.warning('拒绝时必须填写审批意见');
      return;
    }
    setSubmitting(true);
    try {
      await WorkflowApi.submitTaskDecision(decision.task.id, {
        action: decision.action,
        comment,
      });
      message.success(decision.action === 'approve' ? '已批准' : '已拒绝');
      setDecision(null);
      loadTasks();
      loadLegacy();
    } catch (e) {
      message.error(e instanceof Error ? e.message : '提交审批决策失败');
    } finally {
      setSubmitting(false);
    }
  };

  // BPMN 待办表格列
  const taskColumns = [
    {
      title: '任务',
      dataIndex: 'taskName',
      key: 'taskName',
      render: (text: string, record: BpmnMyTask) => (
        <div>
          <div className="font-medium text-gray-900">{text || record.taskDefinitionKey}</div>
          {record.taskPurpose && (
            <Text type="secondary" className="text-xs">{record.taskPurpose}</Text>
          )}
        </div>
      ),
    },
    {
      title: '业务单据',
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
      title: '流程',
      dataIndex: 'processDefinitionKey',
      key: 'processDefinitionKey',
      width: 180,
      responsive: ['lg'] as any,
      render: (key: string) => key ? <Tag icon={<GitBranch className="w-3 h-3 inline mr-1" />}>{key}</Tag> : '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => {
        const meta = taskStatusMap[(status || '').toLowerCase()];
        return <Tag color={meta?.color || 'default'}>{meta?.text || status}</Tag>;
      },
    },
    {
      title: '负责人',
      dataIndex: 'assignee',
      key: 'assignee',
      width: 110,
      responsive: ['md'] as any,
      render: (assignee: string) => assignee || <Text type="secondary">未领取</Text>,
    },
    {
      title: '创建时间',
      dataIndex: 'createdTime',
      key: 'createdTime',
      width: 130,
      responsive: ['xl'] as any,
      render: (t: string) => t ? (
        <Tooltip title={dayjs(t).format('YYYY-MM-DD HH:mm:ss')}>
          <span className="text-gray-500">{dayjs(t).fromNow()}</span>
        </Tooltip>
      ) : '-',
    },
    {
      title: '操作',
      key: 'action',
      width: 220,
      render: (_: unknown, record: BpmnMyTask) => (
        <Space size="small">
          {!record.assignee && (
            <Button
              size="small"
              icon={<Hand className="w-3 h-3" />}
              loading={claiming === record.id}
              onClick={() => handleClaim(record)}
            >
              领取
            </Button>
          )}
          <Button
            type="primary"
            size="small"
            icon={<Check className="w-3 h-3" />}
            onClick={() => openDecision(record, 'approve')}
            className="!bg-green-500 !border-green-500 hover:!bg-green-600 hover:!border-green-600"
          >
            批准
          </Button>
          <Button
            danger
            size="small"
            icon={<X className="w-3 h-3" />}
            onClick={() => openDecision(record, 'reject')}
          >
            拒绝
          </Button>
        </Space>
      ),
    },
  ];

  // 旧业务参考视图表格列（只读，仅提供跳转）
  const legacyColumns = [
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type: LegacyPendingItem['type']) => {
        const map: Record<LegacyPendingItem['type'], { label: string; icon: React.ReactNode }> = {
          ticket: { label: '工单', icon: <FileText className="w-3 h-3 inline mr-1" /> },
          change: { label: '变更', icon: <Wrench className="w-3 h-3 inline mr-1" /> },
          service_request: { label: '服务请求', icon: <Headphones className="w-3 h-3 inline mr-1" /> },
        };
        return <Tag>{map[type].icon}{map[type].label}</Tag>;
      },
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      render: (text: string, record: LegacyPendingItem) => (
        <Link href={record.url} className="font-medium text-gray-900 hover:text-blue-600 transition-colors">
          {text}
        </Link>
      ),
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 90,
      render: (p: string) => p ? <Tag color={priorityColorMap[p] || 'default'}>{p.toUpperCase()}</Tag> : '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (s: string) => <Tag color="gold">{s}</Tag>,
    },
    {
      title: '申请人',
      dataIndex: 'requester',
      key: 'requester',
      width: 110,
      responsive: ['md'] as any,
      render: (name: string) => name || '-',
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 130,
      responsive: ['lg'] as any,
      render: (t: string) => t ? (
        <Tooltip title={dayjs(t).format('YYYY-MM-DD HH:mm:ss')}>
          <span className="text-gray-500">{dayjs(t).fromNow()}</span>
        </Tooltip>
      ) : '-',
    },
    {
      title: '操作',
      key: 'action',
      width: 90,
      render: (_: unknown, record: LegacyPendingItem) => (
        <Link href={record.url}>
          <Button size="small" icon={<ExternalLink className="w-3 h-3" />}>查看</Button>
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

  return (
    <div className="p-4 md:p-6">
      {/* 头部区域 */}
      <div className="mb-4 md:mb-6 flex flex-col md:flex-row md:justify-between md:items-center gap-4">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 md:w-12 md:h-12 rounded-xl bg-blue-50 flex items-center justify-center">
            <CheckCircle className="text-xl md:text-2xl text-blue-500" />
          </div>
          <div>
            <Title level={3} className="!mb-0 !text-xl md:!text-2xl">审批中心</Title>
            <Text type="secondary" className="text-sm">
              {user?.username ? `${user.username}，` : ''}您有 {tasks.length} 项流程待办
            </Text>
          </div>
        </div>
        <Button
          icon={<RotateCcw className={taskLoading || legacyLoading ? 'animate-spin' : ''} />}
          onClick={handleRefresh}
          loading={taskLoading || legacyLoading}
        >
          刷新
        </Button>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[12, 12]} className="mb-4 md:mb-6">
        <Col xs={12} sm={8}>
          <Card className="border-l-4 border-l-blue-500">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-xs md:text-sm text-gray-500">流程待办</div>
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
                <div className="text-xs md:text-sm text-gray-500">待领取</div>
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
                <div className="text-xs md:text-sm text-gray-500">业务待审（参考）</div>
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
                  <GitBranch /> 流程待办
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
                  pagination={{ pageSize: 10, showSizeChanger: true, showTotal: (total) => `共 ${total} 项` }}
                  scroll={{ x: 900 }}
                  locale={{
                    emptyText: (
                      <div className="py-12 text-center">
                        <Clock className="mx-auto mb-3 text-gray-300 w-10 h-10" />
                        <div className="text-gray-500 mb-1">暂无流程待办</div>
                        <Text type="secondary" className="text-sm">当前没有分配给您或待您领取的审批任务</Text>
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
                  <FileText /> 业务待审（参考）
                  <Badge count={legacyItems.length} showZero={false} color="#999" />
                </span>
              ),
              children: (
                <div>
                  <div className="mb-3 px-3 py-2 bg-amber-50 border border-amber-200 rounded text-sm text-amber-700">
                    此视图为业务单据状态参考，审批操作请在「流程待办」中完成，以保证流程实例正常推进。
                  </div>
                  {legacyLoading ? (
                    <LoadingSkeleton />
                  ) : (
                    <Table
                      rowKey={(r) => `${r.type}-${r.id}`}
                      dataSource={legacyItems}
                      columns={legacyColumns}
                      pagination={{ pageSize: 10, showSizeChanger: true, showTotal: (total) => `共 ${total} 项` }}
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
        title={decision?.action === 'approve' ? '批准任务' : '拒绝任务'}
        open={!!decision}
        onOk={submitDecision}
        onCancel={() => setDecision(null)}
        okText={decision?.action === 'approve' ? '确认批准' : '确认拒绝'}
        okButtonProps={{
          danger: decision?.action === 'reject',
          loading: submitting,
        }}
        cancelText="取消"
        destroyOnHidden
      >
        {decision && (
          <div className="space-y-3">
            <div>
              <Text type="secondary">任务：</Text>
              <Text strong>{decision.task.taskName || decision.task.taskDefinitionKey}</Text>
            </div>
            {(() => {
              const link = getBusinessLink(decision.task);
              return link ? (
                <div>
                  <Text type="secondary">业务单据：</Text>
                  <Link href={link.url} className="text-blue-600">{link.label}</Link>
                </div>
              ) : null;
            })()}
            <div>
              <Text type="secondary">
                审批意见{decision.action === 'reject' ? '（必填）' : '（选填）'}：
              </Text>
              <TextArea
                rows={3}
                className="mt-1"
                value={decisionComment}
                onChange={(e) => setDecisionComment(e.target.value)}
                placeholder={decision.action === 'reject' ? '请填写拒绝原因' : '可填写审批意见'}
                maxLength={500}
                showCount
              />
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
}
