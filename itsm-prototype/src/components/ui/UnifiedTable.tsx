/**
 * 统一表格组件
 * 提供功能完整的表格组件，支持排序、筛选、分页、批量操作等
 */

import React, { useState, useMemo, useCallback } from 'react';
import {
  Table,
  Card,
  Button,
  Space,
  Input,
  Select,
  DatePicker,
  Tag,
  Tooltip,
  Dropdown,
  Menu,
  Checkbox,
  Row,
  Col,
  Typography,
  Spin,
  Empty,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  SearchOutlined,
  FilterOutlined,
  ReloadOutlined,
  ExportOutlined,
  SettingOutlined,
  MoreOutlined,
} from '@ant-design/icons';
import { TableColumn, FilterConfig, SortConfig, ActionButton } from '../../types/common';
import { PaginatedResponse } from '../../types/api';
import { useDebouncedCallback } from '../../lib/component-utils';

const { Search } = Input;
const { Option } = Select;
const { RangePicker } = DatePicker;
const { Text } = Typography;

// 表格属性接口
export interface UnifiedTableProps<T = unknown> {
  // 数据相关
  dataSource: T[];
  loading?: boolean;
  pagination?: {
    current: number;
    pageSize: number;
    total: number;
    showSizeChanger?: boolean;
    showQuickJumper?: boolean;
    showTotal?: (total: number, range: [number, number]) => string;
  };

  // 列配置
  columns: TableColumn<T>[];

  // 筛选配置
  filters?: FilterConfig[];

  // 操作按钮
  actions?: ActionButton[];

  // 批量操作
  batchActions?: ActionButton[];

  // 搜索配置
  search?: {
    placeholder?: string;
    onSearch: (keyword: string) => void;
  };

  // 事件处理
  onPaginationChange?: (page: number, pageSize: number) => void;
  onSortChange?: (sortConfig: SortConfig) => void;
  onFilterChange?: (filters: Record<string, unknown>) => void;
  onRefresh?: () => void;
  onExport?: (format: 'excel' | 'csv' | 'pdf') => void;

  // 样式配置
  title?: string;
  extra?: React.ReactNode;
  className?: string;
  size?: 'small' | 'middle' | 'large';

  // 其他配置
  rowKey?: string | ((record: T) => string);
  rowSelection?: {
    selectedRowKeys: React.Key[];
    onChange: (selectedRowKeys: React.Key[], selectedRows: T[]) => void;
    getCheckboxProps?: (record: T) => { disabled?: boolean };
  };
  scroll?: { x?: number; y?: number };
  expandable?: {
    expandedRowKeys: React.Key[];
    onExpandedRowsChange: (expandedKeys: React.Key[]) => void;
    expandedRowRender?: (record: T) => React.ReactNode;
  };
}

// 筛选状态接口
interface FilterState {
  [key: string]: unknown;
}

// 统一表格组件
export function UnifiedTable<T extends Record<string, unknown>>({
  dataSource,
  loading = false,
  pagination,
  columns,
  filters = [],
  actions = [],
  batchActions = [],
  search,
  onPaginationChange,
  onSortChange,
  onFilterChange,
  onRefresh,
  onExport,
  title,
  extra,
  className,
  size = 'middle',
  rowKey = 'id',
  rowSelection,
  scroll,
  expandable,
}: UnifiedTableProps<T>) {
  // 状态管理
  const [filterState, setFilterState] = useState<FilterState>({});
  const [sortConfig, setSortConfig] = useState<SortConfig | null>(null);
  const [searchKeyword, setSearchKeyword] = useState('');

  // 防抖搜索
  const debouncedSearch = useDebouncedCallback(
    ((keyword: string) => {
      search?.onSearch(keyword);
    }) as any,
    300
  );

  // 处理搜索
  const handleSearch = useCallback(
    (value: string) => {
      setSearchKeyword(value);
      debouncedSearch(value);
    },
    [debouncedSearch]
  );

  // 处理筛选变化
  const handleFilterChange = useCallback(
    (key: string, value: unknown) => {
      const newFilterState = { ...filterState, [key]: value };
      setFilterState(newFilterState);
      onFilterChange?.(newFilterState);
    },
    [filterState, onFilterChange]
  );

  // 处理排序变化
  const handleSortChange = useCallback(
    (pagination: unknown, tableFilters: unknown, sorter: unknown) => {
      if (sorter && typeof sorter === 'object' && 'field' in sorter && 'order' in sorter) {
        const newSortConfig: SortConfig = {
          field: sorter.field as string,
          order: sorter.order === 'ascend' ? 'asc' : 'desc',
        };
        setSortConfig(newSortConfig);
        onSortChange?.(newSortConfig);
      }
    },
    [onSortChange]
  );

  // 处理分页变化
  const handlePaginationChange = useCallback(
    (page: number, pageSize: number) => {
      onPaginationChange?.(page, pageSize);
    },
    [onPaginationChange]
  );

  // 处理刷新
  const handleRefresh = useCallback(() => {
    onRefresh?.();
  }, [onRefresh]);

  // 处理导出
  const handleExport = useCallback(
    (format: 'excel' | 'csv' | 'pdf') => {
      onExport?.(format);
    },
    [onExport]
  );

  // 渲染筛选器
  const renderFilters = useMemo(() => {
    if (filters.length === 0) return null;

    return (
      <Row gutter={[16, 16]} className='mb-4'>
        {filters.map(filter => (
          <Col key={filter.key} xs={24} sm={12} md={8} lg={6}>
            <div className='mb-2'>
              <Text strong className='text-sm'>
                {filter.label}
              </Text>
            </div>
            {renderFilterInput(filter)}
          </Col>
        ))}
      </Row>
    );
  }, [filters, filterState]);

  // 渲染筛选输入组件
  const renderFilterInput = (filter: FilterConfig) => {
    const value = filterState[filter.key] ?? filter.defaultValue;

    switch (filter.type) {
      case 'text':
        return (
          <Input
            placeholder={filter.placeholder}
            value={value as string}
            onChange={e => handleFilterChange(filter.key, e.target.value)}
          />
        );

      case 'select':
        return (
          <Select
            placeholder={filter.placeholder}
            value={value}
            onChange={val => handleFilterChange(filter.key, val)}
            style={{ width: '100%' }}
            allowClear
          >
            {filter.options?.map(option => (
              <Option key={option.value} value={option.value} disabled={option.disabled}>
                {option.label}
              </Option>
            ))}
          </Select>
        );

      case 'multiselect':
        return (
          <Select
            mode='multiple'
            placeholder={filter.placeholder}
            value={value as string[]}
            onChange={val => handleFilterChange(filter.key, val)}
            style={{ width: '100%' }}
            allowClear
          >
            {filter.options?.map(option => (
              <Option key={option.value} value={option.value} disabled={option.disabled}>
                {option.label}
              </Option>
            ))}
          </Select>
        );

      case 'date':
        return (
          <DatePicker
            placeholder={filter.placeholder}
            value={value}
            onChange={date => handleFilterChange(filter.key, (date as any)?.format?.('YYYY-MM-DD'))}
            style={{ width: '100%' }}
          />
        );

      case 'daterange':
        return (
          <RangePicker
            placeholder={[filter.placeholder || '开始日期', '结束日期']}
            value={value as any}
            onChange={dates =>
              handleFilterChange(
                filter.key,
                (dates as any)?.map((d: any) => d?.format?.('YYYY-MM-DD'))
              )
            }
            style={{ width: '100%' }}
          />
        );

      case 'number':
        return (
          <Input
            type='number'
            placeholder={filter.placeholder}
            value={value as number}
            onChange={e => handleFilterChange(filter.key, Number(e.target.value))}
          />
        );

      default:
        return null;
    }
  };

  // 渲染操作按钮
  const renderActions = useMemo(() => {
    if (actions.length === 0) return null;

    return (
      <Space>
        {actions.map(action => (
          <Button
            key={action.key}
            type={action.type}
            danger={action.danger}
            disabled={action.disabled}
            icon={action.icon}
            onClick={() => action.onClick({} as T)}
          >
            {action.label}
          </Button>
        ))}
      </Space>
    );
  }, [actions]);

  // 渲染批量操作
  const renderBatchActions = useMemo(() => {
    if (batchActions.length === 0 || !rowSelection?.selectedRowKeys?.length) return null;

    return (
      <Space className='mb-4'>
        <Text strong>已选择 {rowSelection.selectedRowKeys.length} 项</Text>
        {batchActions.map(action => (
          <Button
            key={action.key}
            type={action.type}
            danger={action.danger}
            disabled={action.disabled}
            icon={action.icon}
            onClick={() => action.onClick(rowSelection.selectedRowKeys)}
          >
            {action.label}
          </Button>
        ))}
      </Space>
    );
  }, [batchActions, rowSelection]);

  // 渲染导出菜单
  const renderExportMenu = useMemo(() => {
    if (!onExport) return null;

    const menu = (
      <Menu onClick={({ key }) => handleExport(key as 'excel' | 'csv' | 'pdf')}>
        <Menu.Item key='excel' icon={<ExportOutlined />}>
          导出Excel
        </Menu.Item>
        <Menu.Item key='csv' icon={<ExportOutlined />}>
          导出CSV
        </Menu.Item>
        <Menu.Item key='pdf' icon={<ExportOutlined />}>
          导出PDF
        </Menu.Item>
      </Menu>
    );

    return (
      <Dropdown overlay={menu} trigger={['click']}>
        <Button icon={<ExportOutlined />}>
          导出 <MoreOutlined />
        </Button>
      </Dropdown>
    );
  }, [onExport, handleExport]);

  // 渲染表格头部
  const renderTableHeader = useMemo(() => {
    return (
      <div className='flex justify-between items-center mb-4'>
        <div className='flex items-center gap-4'>
          {title && (
            <Text strong className='text-lg'>
              {title}
            </Text>
          )}
          {search && (
            <Search
              placeholder={search.placeholder || '搜索...'}
              value={searchKeyword}
              onChange={e => handleSearch(e.target.value)}
              onSearch={handleSearch}
              style={{ width: 300 }}
              allowClear
            />
          )}
        </div>

        <Space>
          {renderActions}
          {renderExportMenu}
          <Button icon={<ReloadOutlined />} onClick={handleRefresh} loading={loading}>
            刷新
          </Button>
        </Space>
      </div>
    );
  }, [
    title,
    search,
    searchKeyword,
    handleSearch,
    renderActions,
    renderExportMenu,
    handleRefresh,
    loading,
  ]);

  // 处理表格列配置
  const processedColumns = useMemo(() => {
    return columns.map(col => ({
      ...col,
      title: col.title,
      dataIndex: (col.dataIndex || col.key) as any,
      key: col.key,
      width: col.width,
      align: col.align,
      sorter: col.sortable,
      render: col.render,
    }));
  }, [columns]);

  return (
    <Card className={className}>
      {renderTableHeader}

      {renderBatchActions}

      {renderFilters}

      <Table
        dataSource={dataSource}
        columns={processedColumns as unknown as ColumnsType<T>}
        loading={loading}
        pagination={
          pagination
            ? {
                ...pagination,
                onChange: handlePaginationChange,
                onShowSizeChange: handlePaginationChange,
              }
            : false
        }
        rowKey={rowKey}
        rowSelection={rowSelection}
        scroll={scroll}
        expandable={expandable as any}
        size={size}
        onChange={handleSortChange}
        locale={{
          emptyText: <Empty description='暂无数据' />,
        }}
      />
    </Card>
  );
}

// 导出组件
export default UnifiedTable;
