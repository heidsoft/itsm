/**
 * 工单表格视图组件
 * 显示工单列表表格
 */

import React from 'react';
import { Table, Tag, Space, Button, Tooltip, Popconfirm } from 'antd';
import { Pencil, Trash2, User, Eye } from 'lucide-react';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import type { Ticket, TicketPriority, TicketStatus } from '@/types/ticket';
import { TicketPriorityConfig, TicketStatusConfig } from '@/constants/taxonomy';
import { AvatarImage } from '@/components/ui/OptimizedImage';

export interface TicketsTableViewProps {
  tickets: Ticket[];
  loading: boolean;
  pagination: TablePaginationConfig;
  selectedRowKeys: React.Key[];
  onRowSelectionChange: (selectedRowKeys: React.Key[]) => void;
  onPaginationChange: (pagination: TablePaginationConfig) => void;
  onView: (ticket: Ticket) => void;
  onEdit: (ticket: Ticket) => void;
  onDelete: (ticket: Ticket) => void;
  canEdit?: boolean;
  canDelete?: boolean;
}

/**
 * 工单表格视图
 */
export const TicketsTableView: React.FC<TicketsTableViewProps> = ({
  tickets,
  loading,
  pagination,
  selectedRowKeys,
  onRowSelectionChange,
  onPaginationChange,
  onView,
  onEdit,
  onDelete,
  canEdit = true,
  canDelete = true,
}) => {
  // 表格列定义
  // 状态/优先级文案与配色一律取自 @/constants/taxonomy。
  // 旧的本地 map 用 inProgress/pendingApproval 等 camelCase 键索引，
  // 而后端 ticket.status 永远是 snake_case（in_progress），因此永远命中不到，
  // 处理中的工单只能显示原始枚举值；pending_approval 更不是工单状态词表里的值。
  const columns: ColumnsType<Ticket> = [
    {
      title: '工单编号',
      dataIndex: 'ticketNumber',
      key: 'ticketNumber',
      width: 120,
      fixed: 'left',
      render: (text: string) => (
        <span className="font-mono text-blue-600 font-semibold">{text}</span>
      ),
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      ellipsis: true,
      render: (text: string, record: Ticket) => (
        <Tooltip title={record.description}>
          <span className="cursor-pointer hover:text-blue-600">{text}</span>
        </Tooltip>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => {
        const config = TicketStatusConfig[status as TicketStatus];
        return (
          <Tag color={config?.color ?? 'default'}>{config?.label ?? status}</Tag>
        );
      },
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 90,
      render: (priority: string) => {
        const config = TicketPriorityConfig[priority as TicketPriority];
        return (
          <Tag color={config?.color ?? 'default'}>{config?.label ?? priority}</Tag>
        );
      },
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type: string) => <Tag>{type || '未分类'}</Tag>,
    },
    {
      title: '创建人',
      dataIndex: 'requester',
      key: 'requester',
      width: 120,
      render: (requester: { id: number; name: string; avatar?: string }) => (
        <Space>
          <AvatarImage src={requester?.avatar} name={requester?.name || '未知'} size={24} />
          <span>{requester?.name || '未知'}</span>
        </Space>
      ),
    },
    {
      title: '指派人',
      dataIndex: 'assignee',
      key: 'assignee',
      width: 120,
      render: (assignee: { id: number; name: string; avatar?: string }) =>
        assignee ? (
          <Space>
            <AvatarImage src={assignee.avatar} name={assignee.name} size={24} />
            <span>{assignee.name}</span>
          </Space>
        ) : (
          <span className="text-gray-400">未分配</span>
        ),
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 160,
      sorter: true,
      render: (date: string) => (
        <span className="text-gray-600">{new Date(date).toLocaleString('zh-CN')}</span>
      ),
    },
    {
      title: '更新时间',
      dataIndex: 'updatedAt',
      key: 'updatedAt',
      width: 160,
      sorter: true,
      render: (date: string) => (
        <span className="text-gray-600">{new Date(date).toLocaleString('zh-CN')}</span>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      fixed: 'right',
      width: 150,
      render: (_, record: Ticket) => (
        <Space size="small">
          <Tooltip title="查看详情">
            <Button
              type="link"
              size="small"
              icon={<Eye />}
              onClick={() => onView(record)}
            />
          </Tooltip>

          {canEdit && (
            <Tooltip title="编辑">
              <Button
                type="link"
                size="small"
                icon={<Pencil />}
                onClick={() => onEdit(record)}
              />
            </Tooltip>
          )}

          {canDelete && (
            <Popconfirm
              title="确定要删除这个工单吗？"
              onConfirm={() => onDelete(record)}
              okText="确定"
              cancelText="取消"
            >
              <Tooltip title="删除">
                <Button type="link" size="small" danger icon={<Trash2 />} />
              </Tooltip>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  // 行选择配置
  const rowSelection = {
    selectedRowKeys,
    onChange: onRowSelectionChange,
    selections: [Table.SELECTION_ALL, Table.SELECTION_INVERT, Table.SELECTION_NONE],
  };

  return (
    <Table<Ticket>
      columns={columns}
      dataSource={tickets}
      rowKey="id"
      loading={loading}
      pagination={pagination}
      rowSelection={rowSelection}
      onChange={pagination => onPaginationChange(pagination)}
      scroll={{ x: 1500 }}
      className="bg-white rounded-lg shadow-sm"
      size="middle"
    />
  );
};

export default TicketsTableView;
