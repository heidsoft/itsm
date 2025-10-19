/**
 * ITSM前端架构 - 模块化组件示例
 *
 * 工单管理模块组件
 * 展示如何将大型页面组件拆分为可复用的小组件
 */

import React, { useState, useCallback, useMemo } from 'react';
import { Card, Table, Button, Input, Select, Space, Tag, Tooltip, Modal } from 'antd';
import { Search, Plus, Edit, Eye, Trash2, Filter } from 'lucide-react';
import { registerComponent, ComponentType } from '@/lib/architecture/components';
import { useStateManager } from '@/lib/architecture/state';

const { Option } = Select;

// ==================== 原子组件 (Atoms) ====================

/**
 * 搜索输入框组件
 */
interface SearchInputProps {
  placeholder?: string;
  onSearch: (value: string) => void;
  loading?: boolean;
}

export const SearchInput: React.FC<SearchInputProps> = ({
  placeholder = '搜索...',
  onSearch,
  loading = false,
}) => {
  const [value, setValue] = useState('');

  const handleSearch = useCallback(() => {
    onSearch(value);
  }, [value, onSearch]);

  return (
    <Input.Search
      placeholder={placeholder}
      value={value}
      onChange={e => setValue(e.target.value)}
      onSearch={handleSearch}
      loading={loading}
      enterButton={<Search size={16} />}
      style={{ width: 300 }}
    />
  );
};

// 注册搜索输入框组件
registerComponent('SearchInput', {
  name: 'SearchInput',
  type: ComponentType.ATOM,
  module: 'ticket',
  version: '1.0.0',
  description: '可复用的搜索输入框组件',
  props: [
    {
      name: 'placeholder',
      type: 'string',
      required: false,
      defaultValue: '搜索...',
      description: '占位符文本',
      examples: ['搜索工单', '搜索用户'],
    },
    {
      name: 'onSearch',
      type: 'function',
      required: true,
      description: '搜索回调函数',
      examples: ['(value) => console.log(value)'],
    },
    {
      name: 'loading',
      type: 'boolean',
      required: false,
      defaultValue: false,
      description: '加载状态',
      examples: [true, false],
    },
  ],
  examples: [
    {
      title: '基础用法',
      description: '基本的搜索输入框',
      code: '<SearchInput onSearch={(value) => console.log(value)} />',
      props: { onSearch: (value: string) => console.log(value) },
    },
  ],
  dependencies: ['antd', 'lucide-react'],
  tags: ['input', 'search', 'form'],
});

/**
 * 状态标签组件
 */
interface StatusTagProps {
  status: 'open' | 'in_progress' | 'resolved' | 'closed';
  size?: 'small' | 'default' | 'large';
}

export const StatusTag: React.FC<StatusTagProps> = ({ status, size = 'default' }) => {
  const statusConfig = useMemo(() => {
    const configs = {
      open: { color: 'blue', text: '开放' },
      in_progress: { color: 'orange', text: '处理中' },
      resolved: { color: 'green', text: '已解决' },
      closed: { color: 'gray', text: '已关闭' },
    };
    return configs[status];
  }, [status]);

  return (
    <Tag color={statusConfig.color} style={{ fontSize: size === 'small' ? '12px' : '14px' }}>
      {statusConfig.text}
    </Tag>
  );
};

// ==================== 分子组件 (Molecules) ====================

/**
 * 工单操作按钮组
 */
interface TicketActionsProps {
  ticketId: number;
  onEdit: (id: number) => void;
  onView: (id: number) => void;
  onDelete: (id: number) => void;
  permissions?: string[];
}

export const TicketActions: React.FC<TicketActionsProps> = ({
  ticketId,
  onEdit,
  onView,
  onDelete,
  permissions = [],
}) => {
  const canEdit = permissions.includes('ticket:edit');
  const canDelete = permissions.includes('ticket:delete');

  return (
    <Space size='small'>
      <Tooltip title='查看详情'>
        <Button type='text' icon={<Eye size={16} />} onClick={() => onView(ticketId)} />
      </Tooltip>
      {canEdit && (
        <Tooltip title='编辑工单'>
          <Button type='text' icon={<Edit size={16} />} onClick={() => onEdit(ticketId)} />
        </Tooltip>
      )}
      {canDelete && (
        <Tooltip title='删除工单'>
          <Button
            type='text'
            danger
            icon={<Trash2 size={16} />}
            onClick={() => onDelete(ticketId)}
          />
        </Tooltip>
      )}
    </Space>
  );
};

/**
 * 工单过滤器组件
 */
interface TicketFiltersProps {
  onFilterChange: (filters: Record<string, any>) => void;
  loading?: boolean;
}

export const TicketFilters: React.FC<TicketFiltersProps> = ({
  onFilterChange,
  loading = false,
}) => {
  const [filters, setFilters] = useState({
    status: undefined,
    priority: undefined,
    assignee: undefined,
  });

  const handleFilterChange = useCallback(
    (key: string, value: any) => {
      const newFilters = { ...filters, [key]: value };
      setFilters(newFilters);
      onFilterChange(newFilters);
    },
    [filters, onFilterChange]
  );

  return (
    <Card size='small' style={{ marginBottom: 16 }}>
      <Space wrap>
        <span>筛选条件：</span>
        <Select
          placeholder='状态'
          value={filters.status}
          onChange={value => handleFilterChange('status', value)}
          style={{ width: 120 }}
          allowClear
        >
          <Option value='open'>开放</Option>
          <Option value='in_progress'>处理中</Option>
          <Option value='resolved'>已解决</Option>
          <Option value='closed'>已关闭</Option>
        </Select>
        <Select
          placeholder='优先级'
          value={filters.priority}
          onChange={value => handleFilterChange('priority', value)}
          style={{ width: 120 }}
          allowClear
        >
          <Option value='low'>低</Option>
          <Option value='medium'>中</Option>
          <Option value='high'>高</Option>
          <Option value='urgent'>紧急</Option>
        </Select>
        <Select
          placeholder='处理人'
          value={filters.assignee}
          onChange={value => handleFilterChange('assignee', value)}
          style={{ width: 120 }}
          allowClear
        >
          <Option value='user1'>张三</Option>
          <Option value='user2'>李四</Option>
          <Option value='user3'>王五</Option>
        </Select>
      </Space>
    </Card>
  );
};

// ==================== 有机体组件 (Organisms) ====================

/**
 * 工单列表组件
 */
interface Ticket {
  id: number;
  title: string;
  status: 'open' | 'in_progress' | 'resolved' | 'closed';
  priority: 'low' | 'medium' | 'high' | 'urgent';
  assignee: string;
  created_at: string;
  updated_at: string;
}

interface TicketListProps {
  tickets: Ticket[];
  loading?: boolean;
  onEdit: (id: number) => void;
  onView: (id: number) => void;
  onDelete: (id: number) => void;
  permissions?: string[];
}

export const TicketList: React.FC<TicketListProps> = ({
  tickets,
  loading = false,
  onEdit,
  onView,
  onDelete,
  permissions = [],
}) => {
  const columns = useMemo(
    () => [
      {
        title: 'ID',
        dataIndex: 'id',
        key: 'id',
        width: 80,
      },
      {
        title: '标题',
        dataIndex: 'title',
        key: 'title',
        ellipsis: true,
      },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        width: 100,
        render: (status: Ticket['status']) => <StatusTag status={status} />,
      },
      {
        title: '优先级',
        dataIndex: 'priority',
        key: 'priority',
        width: 100,
        render: (priority: Ticket['priority']) => {
          const colors = {
            low: 'green',
            medium: 'blue',
            high: 'orange',
            urgent: 'red',
          };
          return <Tag color={colors[priority]}>{priority}</Tag>;
        },
      },
      {
        title: '处理人',
        dataIndex: 'assignee',
        key: 'assignee',
        width: 120,
      },
      {
        title: '创建时间',
        dataIndex: 'created_at',
        key: 'created_at',
        width: 150,
      },
      {
        title: '操作',
        key: 'actions',
        width: 120,
        render: (_: any, record: Ticket) => (
          <TicketActions
            ticketId={record.id}
            onEdit={onEdit}
            onView={onView}
            onDelete={onDelete}
            permissions={permissions}
          />
        ),
      },
    ],
    [onEdit, onView, onDelete, permissions]
  );

  return (
    <Table
      columns={columns}
      dataSource={tickets}
      loading={loading}
      rowKey='id'
      pagination={{
        pageSize: 20,
        showSizeChanger: true,
        showQuickJumper: true,
        showTotal: total => `共 ${total} 条记录`,
      }}
    />
  );
};

// ==================== 模板组件 (Templates) ====================

/**
 * 工单管理页面模板
 */
interface TicketManagementTemplateProps {
  children: React.ReactNode;
  title?: string;
  extra?: React.ReactNode;
}

export const TicketManagementTemplate: React.FC<TicketManagementTemplateProps> = ({
  children,
  title = '工单管理',
  extra,
}) => {
  return (
    <div style={{ padding: 24 }}>
      <div
        style={{
          marginBottom: 24,
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}
      >
        <h1 style={{ margin: 0 }}>{title}</h1>
        {extra}
      </div>
      {children}
    </div>
  );
};

// ==================== 页面组件 (Pages) ====================

/**
 * 工单管理页面
 */
interface TicketManagementPageProps {
  initialTickets?: Ticket[];
}

export const TicketManagementPage: React.FC<TicketManagementPageProps> = ({
  initialTickets = [],
}) => {
  const [tickets, setTickets] = useState<Ticket[]>(initialTickets);
  const [loading, setLoading] = useState(false);
  const [searchValue, setSearchValue] = useState('');
  const [filters, setFilters] = useState<Record<string, any>>({});

  // 使用状态管理器
  const ticketStore = useStateManager('ticket-store');

  const handleSearch = useCallback((value: string) => {
    setSearchValue(value);
    // 实现搜索逻辑
  }, []);

  const handleFilterChange = useCallback((newFilters: Record<string, any>) => {
    setFilters(newFilters);
    // 实现过滤逻辑
  }, []);

  const handleEdit = useCallback((id: number) => {
    // 实现编辑逻辑
    console.log('Edit ticket:', id);
  }, []);

  const handleView = useCallback((id: number) => {
    // 实现查看逻辑
    console.log('View ticket:', id);
  }, []);

  const handleDelete = useCallback((id: number) => {
    Modal.confirm({
      title: '确认删除',
      content: '确定要删除这个工单吗？',
      onOk: () => {
        setTickets(prev => prev.filter(ticket => ticket.id !== id));
      },
    });
  }, []);

  const handleCreate = useCallback(() => {
    // 实现创建逻辑
    console.log('Create new ticket');
  }, []);

  return (
    <TicketManagementTemplate
      title='工单管理'
      extra={
        <Button type='primary' icon={<Plus size={16} />} onClick={handleCreate}>
          新建工单
        </Button>
      }
    >
      <Card>
        <div style={{ marginBottom: 16 }}>
          <SearchInput placeholder='搜索工单...' onSearch={handleSearch} loading={loading} />
        </div>

        <TicketFilters onFilterChange={handleFilterChange} loading={loading} />

        <TicketList
          tickets={tickets}
          loading={loading}
          onEdit={handleEdit}
          onView={handleView}
          onDelete={handleDelete}
          permissions={['ticket:edit', 'ticket:delete']}
        />
      </Card>
    </TicketManagementTemplate>
  );
};

export default TicketManagementPage;
