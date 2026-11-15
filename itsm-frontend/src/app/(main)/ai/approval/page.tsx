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
import { usePermissions } from '@/lib/hooks/use-permissions';
import { PERM } from '@/lib/permissions';
import { formatDateTime } from '@/lib/formatters';

const { Title, Text } = Typography;

const STATE_LABELS: Record<string, string> = {
  pending: '待审批',
  approved: '已通过',
  rejected: '已驳回',
  auto: '自动执行',
  // AI 主体（agent）专属状态：审批人数由权限码策略的 minApprovals 决定，
  // 且后端强制「发起人不可自批」。
  pending_agent: 'AI 待审批',
  pending_agent_two: 'AI 待双人审批',
  awaiting_second: 'AI 待二审',
};

const stateColor = (s: string): string => {
  switch (s) {
    case 'pending':
      return 'gold';
    case 'approved':
      return 'green';
    case 'rejected':
      return 'red';
    case 'pending_agent':
      return 'gold';
    case 'pending_agent_two':
    case 'awaiting_second':
      return 'volcano';
    default:
      return 'default';
  }
};

// AI 发起的双重审批进度：awaiting_second 表示首审已通过、待第二位审批人。
const approvalProgress = (s: string, minApprovals?: number): string | null => {
  if (s === 'pending_agent') return minApprovals && minApprovals > 1 ? '0/2' : '0/1';
  if (s === 'pending_agent_two') return '0/2';
  if (s === 'awaiting_second') return '1/2';
  return null;
};

const RISK_LABELS: Record<string, string> = {
  low: '低风险',
  medium: '中风险',
  high: '高风险',
  critical: '危急',
  unknown: '未知',
};

const riskColor = (r?: string): string => {
  switch (r) {
    case 'low':
      return 'green';
    case 'medium':
      return 'gold';
    case 'high':
      return 'volcano';
    case 'critical':
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
  const { hasPermission } = usePermissions();
  // 通过/驳回按钮与后端审批路由（ai:write）同码；见操作列。
  const canApprove = hasPermission(PERM.ai.write);

  const [items, setItems] = useState<ToolApproval[]>([]);
  // 列表初值必须为 true：mount 必定发起首次拉取，初值 false 会让首帧先渲染
  // 空表格，再被真实数据覆盖。
  const [listLoading, setListLoading] = useState(true);
  const [state, setState] = useState<string>('pending');
  const [rejectId, setRejectId] = useState<number | null>(null);
  const [rejectReason, setRejectReason] = useState('');

  const fetchList = useCallback(async () => {
    setListLoading(true);
    try {
      const res: ToolApprovalListResponse = await aiGetToolApprovals(state);
      setItems(res.items ?? []);
    } catch (e) {
      message.error(`加载审批队列失败：${(e as Error).message}`);
    } finally {
      setListLoading(false);
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
      title: '风险',
      dataIndex: 'riskLevel',
      key: 'riskLevel',
      width: 90,
      render: (v: string, r: ToolApproval) => (
        <TooltipWrapper
          text={r.minApprovals ? `权限码策略要求 ${r.minApprovals} 名互不相同的人类审批人` : undefined}
        >
          <Tag color={riskColor(v)}>{RISK_LABELS[v] ?? v ?? '未知'}</Tag>
        </TooltipWrapper>
      ),
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
      width: 150,
      render: (v: string, r: ToolApproval) => {
        const isAgentFlow = v.startsWith('pending_agent') || v === 'awaiting_second';
        const progress = approvalProgress(v, r.minApprovals);
        return (
          <Space size={4} wrap>
            {isAgentFlow && (
              <TooltipWrapper text="该调用由 AI 发起，需人类审批后才会执行；发起人不可自批">
                <Tag color="purple">AI 发起</Tag>
              </TooltipWrapper>
            )}
            <Tag color={stateColor(v)}>{STATE_LABELS[v] ?? v}</Tag>
            {progress && (
              <Tag color={progress.startsWith('1/') ? 'orange' : 'default'}>审批进度 {progress}</Tag>
            )}
          </Space>
        );
      },
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 170,
      render: (v: string) => formatDateTime(v),
    },
    {
      title: '操作',
      key: 'action',
      width: 150,
      render: (_: unknown, r: ToolApproval) => {
        // 可操作状态：人发起的单人审批 + AI 发起的三种在途状态。
        // （此前只认 pending，AI 发起的调用全部显示「已处理」且无按钮，
        //  人工审批闭环在 UI 上是断的。）
        const actionable =
          r.approvalState === 'pending' ||
          r.approvalState === 'pending_agent' ||
          r.approvalState === 'pending_agent_two' ||
          r.approvalState === 'awaiting_second';
        // 审批动作挂 ai:write（与后端 POST /agent/tools/:id/approve 的
        // RequirePermission 同码）；无权限只隐藏按钮，授权仍以后端为准。
        if (!actionable) return <Text type="secondary">已处理</Text>;
        if (!canApprove) {
          return <Text type="secondary">无审批权限（需 ai:write）</Text>;
        }
        return (
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
        );
      },
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
          loading={listLoading}
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
