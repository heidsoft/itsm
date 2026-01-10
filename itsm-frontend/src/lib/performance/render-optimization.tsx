// @ts-nocheck
/**
 * ITSM前端性能优化 - 渲染性能优化
 *
 * 提供React.memo、useMemo、useCallback等性能优化工具
 */

import React, { memo, useMemo, useCallback, useRef, useEffect, useState } from 'react';
import { Table, Card, Button, Input, Select, Space, Tag, Tooltip } from 'antd';
import { Search, Plus, Edit, Eye, Trash2 } from 'lucide-react';

const { Option } = Select;

// ==================== 性能优化工具 ====================

/**
 * 性能监控Hook
 */
export function usePerformanceMonitor(componentName: string) {
  const renderCountRef = useRef(0);
  const startTimeRef = useRef(Date.now());

  useEffect(() => {
    renderCountRef.current += 1;
    const renderTime = Date.now() - startTimeRef.current;

    if (process.env.NODE_ENV === 'development') {
      console.log(
        `[Performance] ${componentName} rendered ${renderCountRef.current} times in ${renderTime}ms`
      );
    }

    startTimeRef.current = Date.now();
  });

  return {
    renderCount: renderCountRef.current,
    resetRenderCount: () => {
      renderCountRef.current = 0;
    },
  };
}

/**
 * 防抖Hook
 */
export function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState<T>(value);

  useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedValue(value);
    }, delay);

    return () => {
      clearTimeout(handler);
    };
  }, [value, delay]);

  return debouncedValue;
}

/**
 * 节流Hook
 */
export function useThrottle<T>(value: T, limit: number): T {
  const [throttledValue, setThrottledValue] = useState<T>(value);
  const lastRan = useRef(Date.now());

  useEffect(() => {
    const handler = setTimeout(() => {
      if (Date.now() - lastRan.current >= limit) {
        setThrottledValue(value);
        lastRan.current = Date.now();
      }
    }, limit - (Date.now() - lastRan.current));

    return () => {
      clearTimeout(handler);
    };
  }, [value, limit]);

  return throttledValue;
}

// ==================== 优化的搜索输入框组件 ====================

interface SearchInputProps {
  placeholder?: string;
  onSearch: (value: string) => void;
  loading?: boolean;
  debounceMs?: number;
}

export const SearchInput = memo<SearchInputProps>(
  ({ placeholder = '搜索...', onSearch, loading = false, debounceMs = 300 }) => {
    const [value, setValue] = useState('');
    const debouncedValue = useDebounce(value, debounceMs);

    // 使用useCallback优化回调函数
    const handleSearch = useCallback(() => {
      onSearch(debouncedValue);
    }, [debouncedValue, onSearch]);

    const handleChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
      setValue(e.target.value);
    }, []);

    // 监听防抖值变化，自动触发搜索
    useEffect(() => {
      if (debouncedValue !== '') {
        handleSearch();
      }
    }, [debouncedValue, handleSearch]);

    return (
      <Input.Search
        placeholder={placeholder}
        value={value}
        onChange={handleChange}
        onSearch={handleSearch}
        loading={loading}
        enterButton={<Search size={16} />}
        style={{ width: 300 }}
      />
    );
  }
);

SearchInput.displayName = 'SearchInput';

// ==================== 优化的状态标签组件 ====================

interface StatusTagProps {
  status: 'open' | 'in_progress' | 'resolved' | 'closed';
  size?: 'small' | 'default' | 'large';
}

export const StatusTag = memo<StatusTagProps>(({ status, size = 'default' }) => {
  // 使用useMemo缓存状态配置
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
});

StatusTag.displayName = 'StatusTag';

// ==================== 优化的操作按钮组 ====================

interface TicketActionsProps {
  ticketId: number;
  onEdit: (id: number) => void;
  onView: (id: number) => void;
  onDelete: (id: number) => void;
  permissions?: string[];
}

export const TicketActions = memo<TicketActionsProps>(
  ({ ticketId, onEdit, onView, onDelete, permissions = [] }) => {
    // 使用useMemo缓存权限检查结果
    const canEdit = useMemo(() => permissions.includes('ticket:edit'), [permissions]);
    const canDelete = useMemo(() => permissions.includes('ticket:delete'), [permissions]);

    // 使用useCallback优化回调函数
    const handleEdit = useCallback(() => onEdit(ticketId), [ticketId, onEdit]);
    const handleView = useCallback(() => onView(ticketId), [ticketId, onView]);
    const handleDelete = useCallback(() => onDelete(ticketId), [ticketId, onDelete]);

    return (
      <Space size='small'>
        <Tooltip title='查看详情'>
          <Button type='text' icon={<Eye size={16} />} onClick={handleView} />
        </Tooltip>
        {canEdit && (
          <Tooltip title='编辑工单'>
            <Button type='text' icon={<Edit size={16} />} onClick={handleEdit} />
          </Tooltip>
        )}
        {canDelete && (
          <Tooltip title='删除工单'>
            <Button type='text' danger icon={<Trash2 size={16} />} onClick={handleDelete} />
          </Tooltip>
        )}
      </Space>
    );
  }
);

TicketActions.displayName = 'TicketActions';

// ==================== 优化的过滤器组件 ====================

interface TicketFiltersPanelProps {
  onFilterChange: (filters: Record<string, any>) => void;
  loading?: boolean;
}

export const TicketFiltersPanel = memo<TicketFiltersPanelProps>(
  ({ onFilterChange, loading = false }) => {
    const [filters, setFilters] = useState({
      status: undefined,
      priority: undefined,
      assignee: undefined,
    });

    // 使用useCallback优化过滤器更新函数
    const handleFilterChange = useCallback(
      (key: string, value: any) => {
        const newFilters = { ...filters, [key]: value };
        setFilters(newFilters);
        onFilterChange(newFilters);
      },
      [filters, onFilterChange]
    );

    // 使用useMemo缓存选项数据
    const statusOptions = useMemo(
      () => [
        { value: 'open', label: '开放' },
        { value: 'in_progress', label: '处理中' },
        { value: 'resolved', label: '已解决' },
        { value: 'closed', label: '已关闭' },
      ],
      []
    );

    const priorityOptions = useMemo(
      () => [
        { value: 'low', label: '低' },
        { value: 'medium', label: '中' },
        { value: 'high', label: '高' },
        { value: 'urgent', label: '紧急' },
      ],
      []
    );

    const assigneeOptions = useMemo(
      () => [
        { value: 'user1', label: '张三' },
        { value: 'user2', label: '李四' },
        { value: 'user3', label: '王五' },
      ],
      []
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
            options={statusOptions}
          />
          <Select
            placeholder='优先级'
            value={filters.priority}
            onChange={value => handleFilterChange('priority', value)}
            style={{ width: 120 }}
            allowClear
            options={priorityOptions}
          />
          <Select
            placeholder='处理人'
            value={filters.assignee}
            onChange={value => handleFilterChange('assignee', value)}
            style={{ width: 120 }}
            allowClear
            options={assigneeOptions}
          />
        </Space>
      </Card>
    );
  }
);

TicketFiltersPanel.displayName = 'TicketFiltersPanel';

// ==================== 优化的工单列表组件 ====================

interface Ticket {
  id: number;
  title: string;
  status: 'open' | 'in_progress' | 'resolved' | 'closed';
  priority: 'low' | 'medium' | 'high' | 'urgent';
  assignee: string;
  created_at: string;
  updated_at: string;
}

interface TicketTableProps {
  tickets: Ticket[];
  loading?: boolean;
  onEdit: (id: number) => void;
  onView: (id: number) => void;
  onDelete: (id: number) => void;
  permissions?: string[];
}

export const TicketTable = memo<TicketTableProps>(
  ({ tickets, loading = false, onEdit, onView, onDelete, permissions = [] }) => {
    // 使用useMemo缓存列配置
    const columns = useMemo(
      () => [
        {
          title: 'ID',
          dataIndex: 'id',
          key: 'id',
          width: 80,
          sorter: (a: Ticket, b: Ticket) => a.id - b.id,
        },
        {
          title: '标题',
          dataIndex: 'title',
          key: 'title',
          ellipsis: true,
          sorter: (a: Ticket, b: Ticket) => a.title.localeCompare(b.title),
        },
        {
          title: '状态',
          dataIndex: 'status',
          key: 'status',
          width: 100,
          render: (status: Ticket['status']) => <StatusTag status={status} />,
          filters: [
            { text: '开放', value: 'open' },
            { text: '处理中', value: 'in_progress' },
            { text: '已解决', value: 'resolved' },
            { text: '已关闭', value: 'closed' },
          ],
          onFilter: (value: string, record: Ticket) => record.status === value,
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
          filters: [
            { text: '低', value: 'low' },
            { text: '中', value: 'medium' },
            { text: '高', value: 'high' },
            { text: '紧急', value: 'urgent' },
          ],
          onFilter: (value: string, record: Ticket) => record.priority === value,
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
          sorter: (a: Ticket, b: Ticket) =>
            new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
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

    // 使用useMemo缓存分页配置
    const paginationConfig = useMemo(
      () => ({
        pageSize: 20,
        showSizeChanger: true,
        showQuickJumper: true,
        showTotal: (total: number) => `共 ${total} 条记录`,
        pageSizeOptions: ['10', '20', '50', '100'],
      }),
      []
    );

    return (
      <Table
        columns={columns}
        dataSource={tickets}
        loading={loading}
        rowKey='id'
        pagination={paginationConfig}
        scroll={{ x: 800 }}
        size='small'
      />
    );
  }
);

TicketTable.displayName = 'TicketTable';

// ==================== 优化的工单管理页面 ====================

interface TicketManagementViewProps {
  initialTickets?: Ticket[];
}

export const TicketManagementView = memo<TicketManagementViewProps>(
  ({ initialTickets = [] }) => {
    const [tickets, setTickets] = useState<Ticket[]>(initialTickets);
    const [loading, setLoading] = useState(false);
    const [searchValue, setSearchValue] = useState('');
    const [filters, setFilters] = useState<Record<string, any>>({});

    // 性能监控
    const { renderCount } = usePerformanceMonitor('TicketManagementPage');

    // 使用useCallback优化回调函数
    const handleSearch = useCallback((value: string) => {
      setSearchValue(value);
      // 实现搜索逻辑
    }, []);

    const handleFilterChange = useCallback((newFilters: Record<string, any>) => {
      setFilters(newFilters);
      // 实现过滤逻辑
    }, []);

    const handleEdit = useCallback((id: number) => {
      console.log('Edit ticket:', id);
    }, []);

    const handleView = useCallback((id: number) => {
      console.log('View ticket:', id);
    }, []);

    const handleDelete = useCallback((id: number) => {
      setTickets(prev => prev.filter(ticket => ticket.id !== id));
    }, []);

    const handleCreate = useCallback(() => {
      console.log('Create new ticket');
    }, []);

    // 使用useMemo缓存过滤后的数据
    const filteredTickets = useMemo(() => {
      let result = tickets;

      // 应用搜索过滤
      if (searchValue) {
        result = result.filter(
          ticket =>
            ticket.title.toLowerCase().includes(searchValue.toLowerCase()) ||
            ticket.assignee.toLowerCase().includes(searchValue.toLowerCase())
        );
      }

      // 应用其他过滤器
      if (filters.status) {
        result = result.filter(ticket => ticket.status === filters.status);
      }

      if (filters.priority) {
        result = result.filter(ticket => ticket.priority === filters.priority);
      }

      return result;
    }, [tickets, searchValue, filters]);

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
          <h1 style={{ margin: 0 }}>工单管理</h1>
          <Button type='primary' icon={<Plus size={16} />} onClick={handleCreate}>
            新建工单
          </Button>
        </div>

        <Card>
          <div style={{ marginBottom: 16 }}>
            <SearchInput
              placeholder='搜索工单...'
              onSearch={handleSearch}
              loading={loading}
              debounceMs={300}
            />
          </div>

          <TicketFiltersPanel onFilterChange={handleFilterChange} loading={loading} />

          <TicketTable
            tickets={filteredTickets}
            loading={loading}
            onEdit={handleEdit}
            onView={handleView}
            onDelete={handleDelete}
            permissions={['ticket:edit', 'ticket:delete']}
          />
        </Card>

        {process.env.NODE_ENV === 'development' && (
          <div style={{ marginTop: 16, fontSize: '12px', color: '#666' }}>
            渲染次数: {renderCount}
          </div>
        )}
      </div>
    );
  }
);

TicketManagementView.displayName = 'TicketManagementView';

export default {
  usePerformanceMonitor,
  useDebounce,
  useThrottle,
  SearchInput,
  StatusTag,
  TicketActions,
  TicketFiltersPanel,
  TicketTable,
  TicketManagementView,
};
