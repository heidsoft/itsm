'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Table,
  Select,
  Space,
  Tag,
  Typography,
  Button,
  Empty,
  App,
  Input,
  Modal,
} from 'antd';
import { CheckCircle2, XCircle, RefreshCw, ShieldAlert } from 'lucide-react';

import {
  aiGetToolApprovals,
  aiApproveTool,
  type ToolApproval,
  type ToolApprovalListResponse,
} from '@/lib/api/ai-api';
import { useAuthStoreHydration } from '@/lib/store/auth-store';
import { formatDateTime } from '@/lib/formatters';

const { Title, Text } = Typography;

const STATE_LABELS: Record<string, string> = {
  pending: 'Pending Approval',
  approved: 'Approved',
  rejected: 'Rejected',
  auto: 'Automatic execution',
};

const stateColor = (s: string): string => {
  switch (s) {
    case 'pending':
      return 'gold';
    case 'approved':
      return 'green';
    case 'rejected':
      return 'red';
    default:
      return 'default';
  }
};

const permissionColor = (p?: string): string => {
  if (p === 'passed') return 'green';
  if (p === 'denied') return 'red';
  if (p === 'skipped') return 'default';
  return 'default';
};

const prettyArgs = (raw?: string): string => {
  if (!raw) return '-';
  try {
    return JSON.stringify(JSON.parse(raw), null, 2);
  } catch {
    return raw;
  }
};

const AIApprovalQueue: React.FC = () => {
  const { message } = App.useApp();
  useAuthStoreHydration();

  const [items, setItems] = useState<ToolApproval[]>([]);
  const [loading, setLoading] = useState(false);
  const [state, setState] = useState<string>('pending');
  const [rejectId, setRejectId] = useState<number | null>(null);
  const [rejectReason, setRejectReason] = useState('');

  const fetchList = useCallback(async () => {
    setLoading(true);
    try {
      const res: ToolApprovalListResponse = await aiGetToolApprovals(state);
      setItems(res.items ?? []);
    } catch (e) {
      message.error(`Failed to load approval queue:${(e as Error).message}`);
    } finally {
      setLoading(false);
    }
  }, [state, message]);

  useEffect(() => {
    fetchList();
  }, [fetchList]);

  const handleApprove = async (id: number) => {
    try {
      await aiApproveTool(id, { approve: true });
      message.success('Approved');
      fetchList();
    } catch (e) {
      message.error(`Operation failed:${(e as Error).message}`);
    }
  };

  const handleRejectOk = async () => {
    if (rejectId == null) return;
    try {
      await aiApproveTool(rejectId, { approve: false, reason: rejectReason });
      message.success('Rejected');
      setRejectId(null);
      setRejectReason('');
      fetchList();
    } catch (e) {
      message.error(`Operation failed:${(e as Error).message}`);
    }
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 70,
    },
    {
      title: 'Tools',
      dataIndex: 'toolName',
      key: 'toolName',
      width: 150,
      render: (v: string) => <Tag color="geekblue">{v}</Tag>,
    },
    {
      title: 'Parameters',
      dataIndex: 'arguments',
      key: 'arguments',
      ellipsis: true,
      render: (v: string) => (
        <Text type="secondary" style={{ fontSize: 12 }}>
          {prettyArgs(v).slice(0, 120)}
          {prettyArgs(v).length > 120 ? '…' : ''}
        </Text>
      ),
    },
    {
      title: 'Permission verification',
      dataIndex: 'permissionCheck',
      key: 'permissionCheck',
      width: 120,
      render: (v: string, r: ToolApproval) => (
        <TooltipWrapper text={r.permissionReason}>
          <Tag color={permissionColor(v)}>{v || '-'}</Tag>
        </TooltipWrapper>
      ),
    },
    {
      title: 'Status',
      dataIndex: 'approvalState',
      key: 'approvalState',
      width: 100,
      render: (v: string) => <Tag color={stateColor(v)}>{STATE_LABELS[v] ?? v}</Tag>,
    },
    {
      title: 'Created',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 170,
      render: (v: string) => formatDateTime(v),
    },
    {
      title: 'Actions',
      key: 'action',
      width: 150,
      render: (_: unknown, r: ToolApproval) =>
        r.approvalState === 'pending' ? (
          <Space>
            <Button
              type="primary"
              size="small"
              icon={<CheckCircle2 size={14} />}
              onClick={() => handleApprove(r.id)}
            >

              Approve
            </Button>
            <Button danger size="small" icon={<XCircle size={14} />} onClick={() => setRejectId(r.id)}>

              Reject
            </Button>
          </Space>
        ) : (
          <Text type="secondary">Processed</Text>
        ),
    },
  ];

  return (
    <div className="space-y-6">
      <Space align="center" style={{ justifyContent: 'space-between', width: '100%' }}>
        <Space align="center">
          <ShieldAlert size={22} color="#1677ff" />
          <Title level={4} style={{ margin: 0 }}>

            AI tool approval queue
          </Title>
        </Space>
        <Space>
          <Select
            style={{ width: 140 }}
            value={state}
            onChange={(v) => setState(v)}
            options={Object.entries(STATE_LABELS).map(([k, label]) => ({ value: k, label }))}
          />
          <a onClick={fetchList}>
            <RefreshCw size={16} />  Refresh
          </a>
        </Space>
      </Space>

      <Card>
        <Table
          rowKey="id"
          size="middle"
          loading={loading}
          dataSource={items}
          columns={columns}
          pagination={{ pageSize: 20, showTotal: (t) => `Total ${t} items` }}
          expandable={{
            expandedRowRender: (r) => (
              <pre style={{ margin: 0, whiteSpace: 'pre-wrap', wordBreak: 'break-all', fontSize: 12 }}>
                {prettyArgs(r.arguments)}
              </pre>
            ),
          }}
          locale={{ emptyText: <Empty description="There are no pending items under the current filter criteria" /> }}
        />
      </Card>

      <Modal
        title="Reject tool call"
        open={rejectId != null}
        onOk={handleRejectOk}
        onCancel={() => {
          setRejectId(null);
          setRejectReason('');
        }}
        okText="Confirm rejection"
        okButtonProps={{ danger: true }}
      >
        <Input.TextArea
          rows={3}
          placeholder="Reason for rejection (optional)"
          value={rejectReason}
          onChange={(e) => setRejectReason(e.target.value)}
        />
      </Modal>
    </div>
  );
};

// 轻量 Tooltip 包装：避免 antd Tooltip 在 SSR/严格模式下对 children 的告警
const TooltipWrapper: React.FC<{ text?: string; children: React.ReactElement }> = ({ text, children }) => {
  if (!text) return children;
  return (
    <span title={text} style={{ cursor: 'help' }}>
      {children}
    </span>
  );
};

export default AIApprovalQueue;
