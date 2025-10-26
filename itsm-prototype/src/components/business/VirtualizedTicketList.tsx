'use client';

import React, { useMemo, useCallback, useState } from 'react';
import { List } from 'react-window';
import { Checkbox, Button, Space, Tooltip, Dropdown, Badge, Avatar } from 'antd';
import {
  FileText,
  Eye,
  Edit,
  Activity,
  Users,
  TrendingUp,
  AlertTriangle,
  MoreHorizontal,
} from 'lucide-react';
import {
  Ticket,
  TicketStatus,
  TicketPriority,
  TicketType,
} from '../../lib/services/ticket-service';

interface VirtualizedTicketListProps {
  tickets: Ticket[];
  loading: boolean;
  selectedRowKeys: string[];
  onRowSelectionChange: (keys: string[]) => void;
  onEditTicket: (ticket: Ticket) => void;
  onViewActivity: (ticket: Ticket) => void;
  height?: number;
  itemHeight?: number;
}

// 状态标签组件
const StatusTag: React.FC<{ status: TicketStatus }> = React.memo(({ status }) => {
  const statusConfig: Record<string, { color: string; text: string; backgroundColor: string }> = {
    [TicketStatus.OPEN]: { color: '#fa8c16', text: 'Open', backgroundColor: '#fff7e6' },
    [TicketStatus.IN_PROGRESS]: {
      color: '#1890ff',
      text: 'In Progress',
      backgroundColor: '#e6f7ff',
    },
    [TicketStatus.PENDING]: { color: '#faad14', text: 'Pending', backgroundColor: '#fffbe6' },
    [TicketStatus.RESOLVED]: { color: '#52c41a', text: 'Resolved', backgroundColor: '#f6ffed' },
    [TicketStatus.CLOSED]: { color: '#00000073', text: 'Closed', backgroundColor: '#fafafa' },
    [TicketStatus.CANCELLED]: { color: '#00000073', text: 'Cancelled', backgroundColor: '#fafafa' },
  };

  const config = statusConfig[status];

  return (
    <span
      style={{
        padding: '4px 12px',
        borderRadius: 16,
        fontSize: 'small',
        fontWeight: 500,
        color: config.color,
        backgroundColor: config.backgroundColor,
      }}
    >
      {config.text}
    </span>
  );
});

StatusTag.displayName = 'StatusTag';

// 优先级标签组件
const PriorityTag: React.FC<{ priority: TicketPriority }> = React.memo(({ priority }) => {
  const priorityConfig: Record<string, { color: string; text: string; backgroundColor: string }> = {
    [TicketPriority.LOW]: { color: '#52c41a', text: 'Low', backgroundColor: '#f6ffed' },
    [TicketPriority.MEDIUM]: { color: '#1890ff', text: 'Medium', backgroundColor: '#e6f7ff' },
    [TicketPriority.HIGH]: { color: '#fa8c16', text: 'High', backgroundColor: '#fff7e6' },
    [TicketPriority.URGENT]: { color: '#ff4d4f', text: 'Urgent', backgroundColor: '#fff2f0' },
  };

  const config = priorityConfig[priority];

  return (
    <span
      style={{
        padding: '4px 12px',
        borderRadius: 16,
        fontSize: 'small',
        fontWeight: 500,
        color: config.color,
        backgroundColor: config.backgroundColor,
      }}
    >
      {config.text}
    </span>
  );
});

PriorityTag.displayName = 'PriorityTag';

// 类型标签组件
const TypeTag: React.FC<{ type: TicketType }> = React.memo(({ type }) => {
  const typeConfig: Record<string, { color: string; text: string; backgroundColor: string }> = {
    [TicketType.INCIDENT]: { color: '#ff4d4f', text: 'Incident', backgroundColor: '#fff2f0' },
    [TicketType.SERVICE_REQUEST]: {
      color: '#1890ff',
      text: 'Service Request',
      backgroundColor: '#e6f7ff',
    },
    [TicketType.PROBLEM]: { color: '#fa8c16', text: 'Problem', backgroundColor: '#fff7e6' },
    [TicketType.CHANGE]: { color: '#722ed1', text: 'Change', backgroundColor: '#f9f0ff' },
  };

  const config = typeConfig[type];

  return (
    <span
      style={{
        padding: '4px 12px',
        borderRadius: 16,
        fontSize: 'small',
        fontWeight: 500,
        color: config.color,
        backgroundColor: config.backgroundColor,
      }}
    >
      {config.text}
    </span>
  );
});

TypeTag.displayName = 'TypeTag';

// 单个工单行组件
const TicketRow: React.FC<{
  ticket: Ticket;
  isSelected: boolean;
  onSelect: (ticketId: string, selected: boolean) => void;
  onEdit: (ticket: Ticket) => void;
  onViewActivity: (ticket: Ticket) => void;
}> = React.memo(({ ticket, isSelected, onSelect, onEdit, onViewActivity }) => {
  const handleSelect = useCallback(() => {
    onSelect(ticket.id, !isSelected);
  }, [ticket.id, isSelected, onSelect]);

  const handleEdit = useCallback(() => {
    onEdit(ticket);
  }, [ticket, onEdit]);

  const handleViewActivity = useCallback(() => {
    onViewActivity(ticket);
  }, [ticket, onViewActivity]);

  return (
    <div
      className={`flex items-center p-4 border-b border-gray-100 hover:bg-blue-50 transition-colors ${
        isSelected ? 'bg-blue-50' : 'bg-white'
      }`}
      style={{ height: '100%' }}
    >
      {/* 选择框 */}
      <div className='w-12 flex justify-center'>
        <Checkbox checked={isSelected} onChange={handleSelect} />
      </div>

      {/* 工单信息 */}
      <div className='flex-1 flex items-center'>
        <div
          style={{
            width: 40,
            height: 40,
            backgroundColor: '#e6f7ff',
            borderRadius: 8,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            marginRight: 12,
          }}
        >
          <FileText size={20} style={{ color: '#1890ff' }} />
        </div>
        <div className='flex-1'>
          <div style={{ fontWeight: 'medium', color: '#000', marginBottom: 4 }}>{ticket.title}</div>
          <div style={{ fontSize: 'small', color: '#666' }}>
            #{ticket.id} • {ticket.category}
          </div>
        </div>
      </div>

      {/* 状态 */}
      <div className='w-32 flex justify-center'>
        <StatusTag status={ticket.status} />
      </div>

      {/* 优先级 */}
      <div className='w-24 flex justify-center'>
        <PriorityTag priority={ticket.priority} />
      </div>

      {/* 类型 */}
      <div className='w-32 flex justify-center'>
        <TypeTag type={ticket.type} />
      </div>

      {/* 分配人 */}
      <div className='w-32 flex justify-center'>
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <Avatar size='small' style={{ backgroundColor: '#1890ff', marginRight: 8 }}>
            {ticket.assignee?.name?.[0] || 'U'}
          </Avatar>
          <span style={{ fontSize: 'small' }}>{ticket.assignee?.name || 'Unassigned'}</span>
        </div>
      </div>

      {/* 创建时间 */}
      <div className='w-32 flex justify-center'>
        <div style={{ fontSize: 'small', color: '#666' }}>
          {new Date(ticket.created_at).toLocaleDateString('zh-CN')}
        </div>
      </div>

      {/* 操作按钮 */}
      <div className='w-48 flex justify-center'>
        <Space size='small'>
          <Tooltip title='View Details'>
            <Button
              type='text'
              size='small'
              icon={<Eye size={16} />}
              onClick={() => window.open(`/tickets/${ticket.id}`)}
            />
          </Tooltip>
          <Tooltip title='Edit'>
            <Button type='text' size='small' icon={<Edit size={16} />} onClick={handleEdit} />
          </Tooltip>
          <Tooltip title='View Activity Log'>
            <Button
              type='text'
              size='small'
              icon={<Activity size={16} />}
              onClick={handleViewActivity}
            />
          </Tooltip>
          <Dropdown
            menu={{
              items: [
                { key: 'assign', label: 'Assign Handler', icon: <Users size={16} /> },
                { key: 'escalate', label: 'Escalate Ticket', icon: <TrendingUp size={16} /> },
                { type: 'divider' },
                {
                  key: 'delete',
                  label: 'Delete Ticket',
                  icon: <AlertTriangle size={16} />,
                  danger: true,
                },
              ],
            }}
            trigger={['click']}
          >
            <Button type='text' size='small' icon={<MoreHorizontal size={16} />} />
          </Dropdown>
        </Space>
      </div>
    </div>
  );
});

TicketRow.displayName = 'TicketRow';

// 表头组件
const TableHeader: React.FC<{
  onSelectAll: (selected: boolean) => void;
  allSelected: boolean;
  indeterminate: boolean;
}> = React.memo(({ onSelectAll, allSelected, indeterminate }) => {
  return (
    <div className='flex items-center p-4 bg-gray-50 border-b-2 border-gray-200 font-semibold text-gray-700'>
      <div className='w-12 flex justify-center'>
        <Checkbox
          checked={allSelected}
          indeterminate={indeterminate}
          onChange={e => onSelectAll(e.target.checked)}
        />
      </div>
      <div className='flex-1'>Ticket Information</div>
      <div className='w-32 text-center'>Status</div>
      <div className='w-24 text-center'>Priority</div>
      <div className='w-32 text-center'>Type</div>
      <div className='w-32 text-center'>Assignee</div>
      <div className='w-32 text-center'>Created Time</div>
      <div className='w-48 text-center'>Actions</div>
    </div>
  );
});

TableHeader.displayName = 'TableHeader';

export const VirtualizedTicketList: React.FC<VirtualizedTicketListProps> = React.memo(
  ({
    tickets,
    loading,
    selectedRowKeys,
    onRowSelectionChange,
    onEditTicket,
    onViewActivity,
    height = 600,
    itemHeight = 80,
  }) => {
    const [allSelected, setAllSelected] = useState(false);
    const [indeterminate, setIndeterminate] = useState(false);

    // 计算选择状态
    useMemo(() => {
      const selectedCount = selectedRowKeys.length;
      const totalCount = tickets.length;

      setAllSelected(selectedCount === totalCount && totalCount > 0);
      setIndeterminate(selectedCount > 0 && selectedCount < totalCount);
    }, [selectedRowKeys.length, tickets.length]);

    // 处理全选
    const handleSelectAll = useCallback(
      (selected: boolean) => {
        if (selected) {
          onRowSelectionChange(tickets.map(ticket => ticket.id));
        } else {
          onRowSelectionChange([]);
        }
      },
      [tickets, onRowSelectionChange]
    );

    // 处理单个选择
    const handleSelect = useCallback(
      (ticketId: string, selected: boolean) => {
        if (selected) {
          onRowSelectionChange([...selectedRowKeys, ticketId]);
        } else {
          onRowSelectionChange(selectedRowKeys.filter(id => id !== ticketId));
        }
      },
      [selectedRowKeys, onRowSelectionChange]
    );

    // 渲染行组件
    const Row = useCallback(
      ({ index, style }: { index: number; style: React.CSSProperties }) => {
        const ticket = tickets[index];
        const isSelected = selectedRowKeys.includes(ticket.id);

        return (
          <div style={style}>
            <TicketRow
              ticket={ticket}
              isSelected={isSelected}
              onSelect={handleSelect}
              onEdit={onEditTicket}
              onViewActivity={onViewActivity}
            />
          </div>
        );
      },
      [tickets, selectedRowKeys, handleSelect, onEditTicket, onViewActivity]
    );

    if (loading) {
      return (
        <div className='flex items-center justify-center h-64'>
          <div className='text-center'>
            <div className='animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto mb-4'></div>
            <div className='text-gray-600'>Loading tickets...</div>
          </div>
        </div>
      );
    }

    if (tickets.length === 0) {
      return (
        <div className='flex items-center justify-center h-64'>
          <div className='text-center'>
            <FileText size={48} className='text-gray-400 mx-auto mb-4' />
            <div className='text-gray-600'>No tickets found</div>
          </div>
        </div>
      );
    }

    return (
      <div className='bg-white rounded-lg border border-gray-100 shadow-sm overflow-hidden'>
        <TableHeader
          onSelectAll={handleSelectAll}
          allSelected={allSelected}
          indeterminate={indeterminate}
        />
        <List
          height={height}
          itemCount={tickets.length}
          itemSize={itemHeight}
          width='100%'
          className='virtual-list'
        >
          {Row}
        </List>
      </div>
    );
  }
);

VirtualizedTicketList.displayName = 'VirtualizedTicketList';
