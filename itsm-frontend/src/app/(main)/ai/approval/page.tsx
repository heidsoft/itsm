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

const { Title, Text } = Typography;

const STATE_LABELS: Record<string, string> = {
  pending: '待审批',
  approved: '已通过',
  rejected: '已驳回',
  auto: '自动执行',
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
      message.error(`加载审批队列失败：${(e as Error).message}`);
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
      message.success('已通过');
      fetchList();
    } catch (e) {
      message.error(`操作失败：${(e as Error).message}`);
    }
  };

  const handleRejectOk = async () => {
    if (rejectId == null) return;
    try {
      await aiApproveTool(rejectId, { approve: false, reason: rejectReason });
      message.success('已驳回');
      setRejectId(null);
      setRejectReason('');
      fetchList();
    } catch (e) {
      message.error(`操作失败：${(e as Error).message}`);
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
      title: '工具',
      dataIndex: 'toolName',
      key: 'toolName',
      width: 150,
      render: (v: string) => <Tag color="geekblue">{v}</Tag>,
    },
    {
      title: '参数',
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
      title: '权限校验',
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
      title: '状态',
      dataIndex: 'approvalState',
      key: 'approvalState',
      width: 100,
      render: (v: string) => <Tag color={stateColor(v)}>{STATE_LABELS[v] ?? v}</Tag>,
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 170,
      render: (v: string) => new Date(v).toLocaleString('zh-CN', { hour12: false }),
    },
    {
      title: '操作',
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
              通过
            </Button>
            <Button danger size="small" icon={<XCircle size={14} />} onClick={() => setRejectId(r.id)}>
              驳回
            </Button>
          </Space>
        ) : (
          <Text type="secondary">已处理</Text>
        ),
    },
  ];

  return (
    <div className="space-y-6">
      <Space align="center" style={{ justifyContent: 'space-between', width: '100%' }}>
        <Space align="center">
          <ShieldAlert size={22} color="#1677ff" />
          <Title level={4} style={{ margin: 0 }}>
            AI 工具审批队列
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
            <RefreshCw size={16} /> 刷新
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
          pagination={{ pageSize: 20, showTotal: (t) => `共 ${t} 条` }}
          expandable={{
            expandedRowRender: (r) => (
              <pre style={{ margin: 0, whiteSpace: 'pre-wrap', wordBreak: 'break-all', fontSize: 12 }}>
                {prettyArgs(r.arguments)}
              </pre>
            ),
          }}
          locale={{ emptyText: <Empty description="当前筛选条件下没有待处理项" /> }}
        />
      </Card>

      <Modal
        title="驳回工具调用"
        open={rejectId != null}
        onOk={handleRejectOk}
        onCancel={() => {
          setRejectId(null);
          setRejectReason('');
        }}
        okText="确认驳回"
        okButtonProps={{ danger: true }}
      >
        <Input.TextArea
          rows={3}
          placeholder="驳回原因（可选）"
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
