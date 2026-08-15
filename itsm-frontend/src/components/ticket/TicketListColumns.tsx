'use client';

import type { ColumnsType } from 'antd/es/table';
import { Button, Space, Tag, Tooltip } from 'antd';
import { CheckCircle, Eye, Pencil } from 'lucide-react';
import dayjs from 'dayjs';
import type { Ticket } from '@/lib/api/types';
import type { TicketStatus, TicketPriority, TicketType } from '@/lib/api/types';

interface StatusConfig {
  readonly color: string;
  readonly text: string;
}

// Keyed by `string` (not the api/types TicketStatus union) so runtime statuses
// the backend emits but the shared union omits - cancelled, pending_approval,
// rejected - still resolve to a label instead of rendering the raw enum value.
// taxonomy.ts is the canonical source; this is a table-local colour mapping.
export const TICKET_STATUS_CONFIG: Readonly<Record<string, StatusConfig>> = {
  new: { color: 'blue', text: '新建' },
  open: { color: 'blue', text: '待处理' },
  in_progress: { color: 'orange', text: '处理中' },
  pending: { color: 'yellow', text: '等待中' },
  pending_approval: { color: 'gold', text: '待审批' },
  resolved: { color: 'green', text: '已解决' },
  closed: { color: 'default', text: '已关闭' },
  cancelled: { color: 'red', text: '已取消' },
  rejected: { color: 'red', text: '已拒绝' },
} as const;

export const PRIORITY_CONFIG: Readonly<Record<string, StatusConfig>> = {
  low: { color: 'green', text: '低' },
  medium: { color: 'orange', text: '中' },
  high: { color: 'red', text: '高' },
  urgent: { color: 'purple', text: '紧急' },
  critical: { color: 'purple', text: '紧急' },
} as const;

export const TICKET_TYPE_CONFIG: Readonly<Record<string, string>> = {
  incident: '事件',
  problem: '问题',
  change: '变更',
  service_request: '服务请求',
  request: '请求',
  task: '任务',
} as const;

// Terminal statuses hide the "close" action. `cancelled` is also terminal -
// closing a cancelled ticket is a no-op, so don't offer the button.
const TERMINAL_STATUSES: readonly string[] = ['closed', 'cancelled'];

export interface TicketListColumnActions {
  readonly onOpen: (ticket: Ticket) => void;
  readonly onEdit: (ticket: Ticket) => void;
  readonly onClose: (ticket: Ticket) => void;
}

/**
 * Builds the Ant Design columns for the tickets table.
 *
 * Receives action handlers as props so the columns module stays pure and is
 * trivially memoizable. The container passes stable `useCallback` handlers.
 */
export function buildTicketListColumns(actions: TicketListColumnActions): ColumnsType<Ticket> {
  const { onOpen, onEdit, onClose } = actions;

  return [
    {
      title: '工单号',
      dataIndex: 'ticketNumber',
      key: 'ticketNumber',
      width: 150,
      fixed: 'left',
      ellipsis: true,
      render: (ticketNumber: string, record: Ticket) => (
        <Button type='link' onClick={() => onOpen(record)}>
          {ticketNumber || '-'}
        </Button>
      ),
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      width: 280,
      ellipsis: { showTitle: false },
      render: (title: string) => (
        <Tooltip placement='topLeft' title={title}>
          {title}
        </Tooltip>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: TicketStatus) => {
        const config = TICKET_STATUS_CONFIG[status] ?? { color: 'default', text: status };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      render: (priority: TicketPriority) => {
        const config = PRIORITY_CONFIG[priority] ?? { color: 'default', text: priority };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type: TicketType) => <Tag>{TICKET_TYPE_CONFIG[type] ?? type}</Tag>,
    },
    {
      title: '来源',
      dataIndex: 'source',
      key: 'source',
      width: 100,
      render: (source: string) => <Tag color='blue'>{source}</Tag>,
    },
    {
      title: '处理人',
      key: 'assignee',
      width: 120,
      render: (_, record: Ticket) =>
        record.assignee?.name ?? (record.assigneeId ? `用户 #${record.assigneeId}` : '未分配'),
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 160,
      render: (createdAt: string) => dayjs(createdAt).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      key: 'updatedAt',
      width: 160,
      render: (updatedAt: string) => dayjs(updatedAt).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作',
      key: 'actions',
      width: 150,
      fixed: 'right',
      render: (_, record: Ticket) => {
        const isTerminal = TERMINAL_STATUSES.includes(record.status);
        return (
          <Space size={0} className='opacity-70 transition-opacity hover:opacity-100'>
            <Tooltip title='查看 (o)'>
              <Button
                type='text'
                aria-label='查看工单'
                icon={<Eye size={16} />}
                onClick={() => onOpen(record)}
              />
            </Tooltip>
            <Tooltip title='编辑'>
              <Button
                type='text'
                aria-label='编辑工单'
                icon={<Pencil size={16} />}
                onClick={() => onEdit(record)}
              />
            </Tooltip>
            {!isTerminal && (record.status === 'resolved' || record.status === 'approved') && (
              <Tooltip title='关闭'>
                <Button
                  type='text'
                  aria-label='关闭工单'
                  icon={<CheckCircle size={16} />}
                  onClick={() => onClose(record)}
                />
              </Tooltip>
            )}
          </Space>
        );
      },
    },
  ];
}
