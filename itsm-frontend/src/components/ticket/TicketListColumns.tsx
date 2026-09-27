'use client';

import type { ColumnsType } from 'antd/es/table';
import { Button, Space, Tag, Tooltip } from 'antd';
import { CheckCircle, Eye, Pencil } from 'lucide-react';
import dayjs from 'dayjs';
import type { Ticket } from '@/lib/api/types';
import type { TicketStatus, TicketPriority, TicketType } from '@/lib/api/types';
import {
  ITSMMainType,
  ITSMMainTypeConfig,
  TicketStatus as TicketStatusEnum,
  TicketPriorityConfig,
  TicketStatusConfig,
} from '@/constants/taxonomy';

interface StatusConfig {
  readonly color: string;
  readonly text: string;
}

// 词表与配色的唯一来源是 @/constants/taxonomy（状态词表对齐后端 common/constants.go）。
// 下面的表只投影出表格需要的 color/text 两列，不再在组件层维护第二套映射，
// 因此不会出现多出 pending_approval、缺少 assigned/approved 这类漂移。
const projectStatusConfig = (config: Record<string, { color: string; label: string }>) =>
  Object.fromEntries(
    Object.entries(config).map(([key, value]) => [key, { color: value.color, text: value.label }])
  );

export const TICKET_STATUS_CONFIG: Readonly<Record<string, StatusConfig>> = projectStatusConfig(
  TicketStatusConfig
);

export const PRIORITY_CONFIG: Readonly<Record<string, StatusConfig>> = projectStatusConfig(
  TicketPriorityConfig
);

/** 工单 ITIL 类型显示名，仅覆盖后端 type 词表中的 ITIL 主类型。 */
export const TICKET_TYPE_CONFIG: Readonly<Record<string, string>> = {
  [ITSMMainType.INCIDENT]: ITSMMainTypeConfig[ITSMMainType.INCIDENT].label,
  [ITSMMainType.SERVICE_REQUEST]: ITSMMainTypeConfig[ITSMMainType.SERVICE_REQUEST].label,
  [ITSMMainType.PROBLEM]: ITSMMainTypeConfig[ITSMMainType.PROBLEM].label,
  [ITSMMainType.CHANGE]: ITSMMainTypeConfig[ITSMMainType.CHANGE].label,
};

// Terminal statuses hide the "close" action. `cancelled` is also terminal -
// closing a cancelled ticket is a no-op, so don't offer the button.
const TERMINAL_STATUSES: readonly TicketStatusEnum[] = [
  TicketStatusEnum.CLOSED,
  TicketStatusEnum.CANCELLED,
];

export interface TicketListColumnActions {
  readonly onOpen: (ticket: Ticket) => void;
  readonly onEdit: (ticket: Ticket) => void;
  readonly onClose: (ticket: Ticket) => void;
}

/** Server-side sort state, used to keep column indicators in sync with the active query. */
export interface TicketListSortState {
  readonly sortBy?: string;
  readonly sortOrder?: 'asc' | 'desc';
}

/**
 * Builds the Ant Design columns for the tickets table.
 *
 * Receives action handlers as props so the columns module stays pure and is
 * trivially memoizable. The container passes stable `useCallback` handlers.
 */
export function buildTicketListColumns(
  actions: TicketListColumnActions,
  sort?: TicketListSortState
): ColumnsType<Ticket> {
  const { onOpen, onEdit, onClose } = actions;
  // Controlled sortOrder keeps the header indicator in sync with the server
  // query (and lets "clear filters" reset it). null = no active sort.
  const sortOf = (field: string) =>
    sort?.sortBy === field ? (sort.sortOrder === 'asc' ? 'ascend' : 'descend') : null;

  return [
    {
      title: '工单号',
      dataIndex: 'ticketNumber',
      key: 'ticketNumber',
      width: 200,
      fixed: 'left',
      ellipsis: true,
      sorter: true,
      sortOrder: sortOf('ticket_number'),
      render: (ticketNumber: string, record: Ticket) => (
        <Button type='link' size='small' onClick={() => onOpen(record)}>
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
      sorter: true,
      sortOrder: sortOf('status'),
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
      sorter: true,
      sortOrder: sortOf('priority'),
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
      sorter: true,
      sortOrder: sortOf('created_at'),
      render: (createdAt: string) => dayjs(createdAt).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      key: 'updatedAt',
      width: 160,
      sorter: true,
      sortOrder: sortOf('updated_at'),
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
