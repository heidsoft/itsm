'use client';

import React, { useEffect, useState } from 'react';
import Link from 'next/link';
import { Card, Space, Button, Table, Tag, App, Empty, Tooltip, Popconfirm, Tabs, message } from 'antd';
import {
  ReloadOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  CheckOutlined,
  CloseOutlined,
  BranchesOutlined,
  UserAddOutlined,
} from '@ant-design/icons';

import { serviceRequestAPI, ServiceRequest } from '@/lib/api/service-request-api';
import { WorkflowApi, type WorkflowTask } from '@/lib/api/workflow-api';
import { useI18n } from '@/lib/i18n';

function statusTag(status: string, t: (key: string) => string) {
  const cfg: Record<string, { color: string; label: string }> = {
    submitted: { color: 'gold', label: t('serviceRequestStatus.submitted') },
    manager_approved: { color: 'blue', label: t('serviceRequestStatus.manager_approved') },
    it_approved: { color: 'geekblue', label: t('serviceRequestStatus.it_approved') },
    security_approved: { color: 'green', label: t('serviceRequestStatus.security_approved') },
    provisioning: { color: 'cyan', label: t('serviceRequestStatus.provisioning') },
    delivered: { color: 'green', label: t('serviceRequestStatus.delivered') },
    failed: { color: 'red', label: t('serviceRequestStatus.failed') },
    rejected: { color: 'red', label: t('serviceRequestStatus.rejected') },
    cancelled: { color: 'default', label: t('serviceRequestStatus.cancelled') },
  };
  const map = cfg[status] || { color: 'default', label: status || '-' };
  return <Tag color={map.color}>{map.label}</Tag>;
}

export default function PendingApprovalsPage() {
  const { message } = App.useApp();
  const { t: i18nT } = useI18n();
  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState<ServiceRequest[]>([]);
  const [page, setPage] = useState(1);
  const [size, setSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [processingIds, setProcessingIds] = useState<Set<number>>(new Set());

  const load = async (p = page, s = size) => {
    try {
      setLoading(true);
      const data = await serviceRequestAPI.getPendingApprovals({ page: p, size: s });
      setRows(data.requests || []);
      setTotal(data.total || 0);
      setPage(data.page || p);
      setSize(data.size || s);
    } catch (e) {
      console.error(e);
      message.error(e instanceof Error ? e.message : '加载待办失败');
      setRows([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  };

  const handleApprove = async (id: number) => {
    try {
      setProcessingIds(prev => new Set(prev).add(id));
      await serviceRequestAPI.applyApprovalAction(id, { action: 'approve' });
      message.success('审批通过');
      load();
    } catch (e) {
      console.error(e);
      message.error('审批失败');
    } finally {
      setProcessingIds(prev => {
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
    }
  };

  const handleReject = async (id: number) => {
    try {
      setProcessingIds(prev => new Set(prev).add(id));
      await serviceRequestAPI.applyApprovalAction(id, { action: 'reject' });
      message.success('已拒绝');
      load();
    } catch (e) {
      console.error(e);
      message.error('操作失败');
    } finally {
      setProcessingIds(prev => {
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
    }
  };

  useEffect(() => {
    load(1, size);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="max-w-6xl mx-auto p-6">
      <Space orientation="vertical" size={16} className="w-full">
        <Card className="rounded-lg shadow-sm border border-gray-200">
          <div className="flex items-center justify-between w-full">
            <div>
              <h2 className="text-2xl font-bold text-gray-900 mb-0">我的待办审批</h2>
              <p className="text-gray-500 mt-1 mb-0">V0：按角色返回当前需要我处理的服务请求 + BPMN 任务</p>
            </div>
            <Button icon={<ReloadOutlined />} onClick={() => load(page, size)} loading={loading}>
              刷新
            </Button>
          </div>
        </Card>

        <Card className="rounded-lg shadow-sm border border-gray-200">
          <Tabs
            defaultActiveKey="service-request"
            items={[
              {
                key: 'service-request',
                label: (
                  <span>
                    <CheckOutlined /> 服务请求审批
                  </span>
                ),
                children: (
                  <ServiceRequestTable
                    rows={rows}
                    loading={loading}
                    page={page}
                    size={size}
                    total={total}
                    processingIds={processingIds}
                    onLoad={load}
                    onApprove={handleApprove}
                    onReject={handleReject}
                    t={i18nT}
                  />
                ),
              },
              {
                key: 'bpmn-task',
                label: (
                  <span>
                    <BranchesOutlined /> 我作为候选组员（BPMN）
                  </span>
                ),
                children: <MyBpmnTaskTab />,
              },
            ]}
          />
        </Card>
      </Space>
    </div>
  );
}

// === 服务请求审批 Tab（保持原逻辑） ===
function ServiceRequestTable(props: {
  rows: ServiceRequest[];
  loading: boolean;
  page: number;
  size: number;
  total: number;
  processingIds: Set<number>;
  onLoad: (p: number, s: number) => void;
  onApprove: (id: number) => void;
  onReject: (id: number) => void;
  t: (key: string) => string;
}) {
  const { rows, loading, page, size, total, processingIds, onLoad, onApprove, onReject, t } = props;
  return (
    <>
      {rows.length === 0 && !loading ? (
        <Empty description="暂无待审批事项" image={Empty.PRESENTED_IMAGE_SIMPLE}>
          <div className="text-center">
            <CheckCircleOutlined className="text-4xl text-green-500 mb-4" />
            <p className="text-gray-500">所有服务请求审批已处理完成</p>
          </div>
        </Empty>
      ) : (
        <Table<ServiceRequest>
          rowKey="id"
          loading={loading}
          dataSource={rows}
          pagination={{
            current: page,
            pageSize: size,
            total,
            showSizeChanger: true,
            onChange: (p, s) => onLoad(p, s),
          }}
          columns={[
            {
              title: 'ID',
              dataIndex: 'id',
              width: 90,
            },
            {
              title: '标题',
              key: 'title',
              render: (_, r) => r.title || r.catalog?.name || '-',
            },
            {
              title: '状态',
              dataIndex: 'status',
              width: 180,
              render: (v: string) => statusTag(String(v), t),
            },
            {
              title: '申请人',
              key: 'requester',
              width: 160,
              render: (_, r) =>
                r.requester?.name || r.requester?.username || String(r.requesterId || '-'),
            },
            {
              title: '创建时间',
              dataIndex: 'createdAt',
              width: 220,
              render: (v: string) => (v ? new Date(v).toLocaleString() : '-'),
            },
            {
              title: '操作',
              key: 'actions',
              width: 240,
              render: (_, r) => (
                <Space size="small">
                  <Tooltip title="批准">
                    <Button
                      type="primary"
                      size="small"
                      icon={<CheckOutlined />}
                      loading={processingIds.has(r.id)}
                      onClick={() => onApprove(r.id)}
                    >
                      批准
                    </Button>
                  </Tooltip>
                  <Popconfirm
                    title="确认拒绝此申请？"
                    description="拒绝后将通知申请人"
                    onConfirm={() => onReject(r.id)}
                    okText="确认拒绝"
                    cancelText="取消"
                    okButtonProps={{ danger: true }}
                  >
                    <Tooltip title="拒绝">
                      <Button
                        danger
                        size="small"
                        icon={<CloseOutlined />}
                        loading={processingIds.has(r.id)}
                      >
                        拒绝
                      </Button>
                    </Tooltip>
                  </Popconfirm>
                  <Link href={`/my-requests/${r.id}`}>
                    <Button size="small">详情</Button>
                  </Link>
                </Space>
              ),
            },
          ]}
        />
      )}
    </>
  );
}

// === BPMN 任务 Tab（作为候选组员/被指派人） ===
function MyBpmnTaskTab() {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [tasks, setTasks] = useState<WorkflowTask[]>([]);
  const [page, setPage] = useState(1);
  const [size, setSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [claimingIds, setClaimingIds] = useState<Set<string | number>>(new Set());

  const load = async (p = page, s = size) => {
    try {
      setLoading(true);
      const data = await WorkflowApi.listMyTasks({ page: p, pageSize: s });
      setTasks(data.items || []);
      setTotal(data.total);
      setPage(data.page);
      setSize(data.size);
    } catch (e) {
      console.error(e);
      message.error('加载 BPMN 待办任务失败');
      setTasks([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  };

  const handleClaim = async (taskId: string | number) => {
    try {
      setClaimingIds(prev => new Set(prev).add(taskId));
      await WorkflowApi.claimMyTask(taskId);
      message.success('已领取');
      load();
    } catch (e) {
      console.error(e);
      message.error('领取失败：' + (e instanceof Error ? e.message : String(e)));
    } finally {
      setClaimingIds(prev => {
        const next = new Set(prev);
        next.delete(taskId);
        return next;
      });
    }
  };

  useEffect(() => {
    load(1, size);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <div className="text-sm text-gray-500">
          包含我作为 <b>被指派人</b>、<b>候选人</b> 或 <b>候选组员</b> 的待办 BPMN UserTask。
        </div>
        <Button size="small" icon={<ReloadOutlined />} onClick={() => load(page, size)} loading={loading}>
          刷新
        </Button>
      </div>
      {tasks.length === 0 && !loading ? (
        <Empty description="暂无 BPMN 待办任务" image={Empty.PRESENTED_IMAGE_SIMPLE}>
          <div className="text-center">
            <CheckCircleOutlined className="text-4xl text-green-500 mb-4" />
            <p className="text-gray-500">没有需要我处理的 BPMN 审批节点</p>
          </div>
        </Empty>
      ) : (
        <Table<WorkflowTask>
          rowKey={(t) => String(t.id)}
          loading={loading}
          dataSource={tasks}
          pagination={{
            current: page,
            pageSize: size,
            total,
            showSizeChanger: true,
            onChange: (p, s) => load(p, s),
          }}
          columns={[
            { title: 'ID', dataIndex: 'id', width: 90 },
            { title: '节点', dataIndex: 'nodeName', width: 180 },
            {
              title: '实例',
              dataIndex: 'instanceId',
              width: 140,
              render: (v: string) => (v ? String(v) : '-'),
            },
            {
              title: '状态',
              dataIndex: 'status',
              width: 120,
              render: (v: string) => <Tag>{String(v)}</Tag>,
            },
            {
              title: '当前指派',
              dataIndex: 'assignee',
              width: 140,
              render: (v: string) => (v ? String(v) : <Tag color="default">未领取</Tag>),
            },
            {
              title: '创建时间',
              dataIndex: 'createdAt',
              width: 200,
              render: (v: string) => (v ? new Date(v).toLocaleString() : '-'),
            },
            {
              title: '操作',
              key: 'actions',
              width: 220,
              render: (_, r) => (
                <Space size="small">
                  {!r.assignee && (
                    <Tooltip title="领取任务并成为指派人">
                      <Button
                        type="primary"
                        size="small"
                        icon={<UserAddOutlined />}
                        loading={claimingIds.has(r.id)}
                        onClick={() => handleClaim(r.id)}
                      >
                        领取
                      </Button>
                    </Tooltip>
                  )}
                  <Link href={`/workflow/instances/${r.instanceId}?task=${r.id}`}>
                    <Button size="small">详情</Button>
                  </Link>
                </Space>
              ),
            },
          ]}
        />
      )}
    </div>
  );
}
