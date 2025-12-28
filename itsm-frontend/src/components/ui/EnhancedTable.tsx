'use client';

import React, { useState, useCallback, useMemo } from 'react';
import {
  Table,
  Card,
  Button,
  Input,
  Space,
  Tooltip,
  Dropdown,
  Empty,
  Spin,
  Typography,
  Badge,
  Avatar,
  Pagination,
} from 'antd';
import {
  FilterOutlined,
  ReloadOutlined,
  ExportOutlined,
  SettingOutlined,
  MoreOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import type { ColumnsType, TableProps } from 'antd/es/table';
import type { MenuProps } from 'antd';
import { motion, AnimatePresence } from 'framer-motion';
import clsx from 'clsx';

const { Search } = Input;
const { Text } = Typography;

export interface EnhancedTableColumn<T> extends Omit<ColumnsType<T>[0], 'render'> {
  render?: (value: any, record: T, index: number) => React.ReactNode;
  sortable?: boolean;
  filterable?: boolean;
  searchable?: boolean;
  width?: number | string;
  minWidth?: number;
  resizable?: boolean;
}

export interface EnhancedTableAction<T> {
  key: string;
  label: string;
  icon?: React.ReactNode;
  onClick: (record: T) => void;
  disabled?: (record: T) => boolean;
  hidden?: (record: T) => boolean;
  danger?: boolean;
  loading?: (record: T) => boolean;
}

export interface EnhancedTableProps<T> {
  // 数据
  data: T[];
  loading?: boolean;
  total?: number;
  
  // 列配置
  columns: EnhancedTableColumn<T>[];
  rowKey: string | ((record: T) => string);
  
  // 分页
  pagination?: {
    current: number;
    pageSize: number;
    total: number;
    showSizeChanger?: boolean;
    showQuickJumper?: boolean;
    showTotal?: boolean;
    onChange?: (page: number, pageSize: number) => void;
  };
  
  // 选择
  selection?: {
    selectedRowKeys: React.Key[];
    onChange: (selectedRowKeys: React.Key[], selectedRows: T[]) => void;
    getCheckboxProps?: (record: T) => any;
  };
  
  // 搜索
  searchable?: {
    placeholder?: string;
    onSearch: (value: string) => void;
    loading?: boolean;
  };
  
  // 过滤
  filters?: Array<{
    key: string;
    label: string;
    options: Array<{ label: string; value: string }>;
    onChange: (values: string[]) => void;
    value?: string[];
  }>;
  
  // 操作
  actions?: EnhancedTableAction<T>[];
  batchActions?: Array<{
    key: string;
    label: string;
    icon?: React.ReactNode;
    onClick: (selectedRows: T[]) => void;
    disabled?: (selectedRows: T[]) => boolean;
    danger?: boolean;
  }>;
  
  // 工具栏
  toolbar?: {
    title?: string;
    extra?: React.ReactNode;
    showRefresh?: boolean;
    onRefresh?: () => void;
    showExport?: boolean;
    onExport?: () => void;
    showSettings?: boolean;
    onSettingsChange?: (settings: any) => void;
  };
  
  // 样式
  size?: 'small' | 'middle' | 'large';
  bordered?: boolean;
  showHeader?: boolean;
  sticky?: boolean;
  
  // 事件
  onRowClick?: (record: T, index: number) => void;
  onRowDoubleClick?: (record: T, index: number) => void;
  
  // 自定义渲染
  emptyText?: React.ReactNode;
  expandable?: TableProps<T>['expandable'];
  
  // 性能优化
  virtual?: boolean;
  scroll?: { x?: number; y?: number };
}

// 动画配置
const tableVariants = {
  hidden: { opacity: 0, y: 20 },
  visible: { 
    opacity: 1, 
    y: 0,
    transition: { duration: 0.3, ease: 'easeOut' }
  },
  exit: { 
    opacity: 0, 
    y: -20,
    transition: { duration: 0.2, ease: 'easeIn' }
  }
};

const rowVariants = {
  hidden: { opacity: 0, x: -10 },
  visible: (i: number) => ({
    opacity: 1,
    x: 0,
    transition: {
      delay: i * 0.05,
      duration: 0.3,
      ease: 'easeOut'
    }
  })
};

export function EnhancedTable<T extends Record<string, any>>({
  data,
  loading = false,
  total,
  columns,
  rowKey,
  pagination,
  selection,
  searchable,
  filters,
  actions,
  batchActions,
  toolbar,
  size = 'middle',
  bordered = false,
  showHeader = true,
  sticky = false,
  onRowClick,
  onRowDoubleClick,
  emptyText,
  expandable,
  virtual = false,
  scroll,
}: EnhancedTableProps<T>) {
  const [searchValue, setSearchValue] = useState('');
  const [selectedFilters, setSelectedFilters] = useState<Record<string, string[]>>({});

  // 处理列配置
  const enhancedColumns = useMemo(() => {
    const processedColumns: ColumnsType<T> = columns.map(col => ({
      ...col,
      sorter: col.sortable ? true : col.sorter,
      filterDropdown: col.filterable ? () => null : col.filterDropdown,
    }));

    // 添加操作列
    if (actions && actions.length > 0) {
      processedColumns.push({
        title: '操作',
        key: 'actions',
        width: Math.min(actions.length * 40 + 20, 160),
        fixed: 'right',
        render: (_, record, index) => (
          <Space size="small">
            {actions.slice(0, 2).map(action => {
              if (action.hidden?.(record)) return null;
              
              return (
                <Tooltip key={action.key} title={action.label}>
                  <Button
                    type="text"
                    size="small"
                    icon={action.icon}
                    disabled={action.disabled?.(record)}
                    loading={action.loading?.(record)}
                    danger={action.danger}
                    onClick={(e) => {
                      e.stopPropagation();
                      action.onClick(record);
                    }}
                    className="hover:scale-105 transition-transform duration-200"
                  />
                </Tooltip>
              );
            })}
            {actions.length > 2 && (
              <Dropdown
                menu={{
                  items: actions.slice(2).map(action => ({
                    key: action.key,
                    label: action.label,
                    icon: action.icon,
                    disabled: action.disabled?.(record),
                    danger: action.danger,
                    onClick: () => action.onClick(record),
                  })),
                }}
                trigger={['click']}
              >
                <Button
                  type="text"
                  size="small"
                  icon={<MoreOutlined />}
                  onClick={(e) => e.stopPropagation()}
                  className="hover:scale-105 transition-transform duration-200"
                />
              </Dropdown>
            )}
          </Space>
        ),
      });
    }

    return processedColumns;
  }, [columns, actions]);

  // 工具栏渲染
  const renderToolbar = () => {
    if (!toolbar && !searchable && !filters) return null;

    return (
      <div className="mb-4 space-y-4">
        {/* 标题和额外操作 */}
        {(toolbar?.title || toolbar?.extra) && (
          <div className="flex justify-between items-center">
            {toolbar?.title && (
              <div className="flex items-center space-x-2">
                <Typography.Title level={4} className="!mb-0">
                  {toolbar.title}
                </Typography.Title>
                {total !== undefined && (
                  <Badge 
                    count={total} 
                    style={{ backgroundColor: '#f0f0f0', color: '#666' }}
                    overflowCount={99999}
                  />
                )}
              </div>
            )}
            <Space size="small">
              {toolbar?.extra}
              {toolbar?.showRefresh && (
                <Button
                  icon={<ReloadOutlined />}
                  onClick={toolbar.onRefresh}
                  loading={loading}
                />
              )}
              {toolbar?.showExport && (
                <Button
                  icon={<ExportOutlined />}
                  onClick={toolbar.onExport}
                />
              )}
              {toolbar?.showSettings && (
                <Button
                  icon={<SettingOutlined />}
                  onClick={() => toolbar.onSettingsChange?.({})}
                />
              )}
            </Space>
          </div>
        )}

        {/* 搜索和过滤 */}
        {(searchable || filters) && (
          <div className="flex flex-col sm:flex-row gap-4">
            {searchable && (
              <div className="flex-1 max-w-sm">
                <Search
                  placeholder={searchable.placeholder || '搜索...'}
                  value={searchValue}
                  onChange={(e) => setSearchValue(e.target.value)}
                  onSearch={searchable.onSearch}
                  loading={searchable.loading}
                  allowClear
                  className="w-full"
                />
              </div>
            )}
            {filters && (
              <Space wrap>
                {filters.map(filter => (
                  <Select
                    key={filter.key}
                    mode="multiple"
                    placeholder={filter.label}
                    value={selectedFilters[filter.key] || filter.value}
                    onChange={(values) => {
                      setSelectedFilters(prev => ({ ...prev, [filter.key]: values }));
                      filter.onChange(values);
                    }}
                    allowClear
                    maxTagCount={2}
                    style={{ minWidth: 120 }}
                  >
                    {filter.options.map(option => (
                      <Select.Option key={option.value} value={option.value}>
                        {option.label}
                      </Select.Option>
                    ))}
                  </Select>
                ))}
              </Space>
            )}
          </div>
        )}

        {/* 批量操作 */}
        {selection && batchActions && selection.selectedRowKeys.length > 0 && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: 'auto' }}
            exit={{ opacity: 0, height: 0 }}
            className="p-3 bg-blue-50 rounded-lg border border-blue-200"
          >
            <div className="flex items-center justify-between">
              <Text type="secondary">
                已选择 {selection.selectedRowKeys.length} 项
              </Text>
              <Space>
                {batchActions.map(action => (
                  <Button
                    key={action.key}
                    size="small"
                    icon={action.icon}
                    danger={action.danger}
                    disabled={action.disabled?.(data.filter(item => 
                      selection.selectedRowKeys.includes(
                        typeof rowKey === 'string' ? item[rowKey] : rowKey(item)
                      )
                    ))}
                    onClick={() => action.onClick(data.filter(item => 
                      selection.selectedRowKeys.includes(
                        typeof rowKey === 'string' ? item[rowKey] : rowKey(item)
                      )
                    ))}
                  >
                    {action.label}
                  </Button>
                ))}
              </Space>
            </div>
          </motion.div>
        )}
      </div>
    );
  };

  // 空状态渲染
  const renderEmpty = () => {
    if (emptyText) return emptyText;
    
    return (
      <Empty
        image={Empty.PRESENTED_IMAGE_SIMPLE}
        description="暂无数据"
        className="py-8"
      />
    );
  };

  return (
    <motion.div
      variants={tableVariants}
      initial="hidden"
      animate="visible"
      exit="exit"
      className="enhanced-table-container"
    >
      <Card 
        bordered={bordered} 
        className={clsx(
          'rounded-lg shadow-sm',
          'hover:shadow-md transition-shadow duration-200'
        )}
        bodyStyle={{ padding: 0 }}
      >
        {/* 工具栏 */}
        {renderToolbar() && (
          <div className="p-6 border-b border-gray-100">
            {renderToolbar()}
          </div>
        )}

        {/* 表格主体 */}
        <div className="overflow-hidden">
          <Spin spinning={loading} size="large">
            <Table
              columns={enhancedColumns}
              dataSource={data}
              rowKey={rowKey}
              pagination={false} // 使用自定义分页
              size={size}
              showHeader={showHeader}
              sticky={sticky}
              scroll={scroll}
              locale={{ emptyText: renderEmpty() }}
              rowSelection={selection ? {
                selectedRowKeys: selection.selectedRowKeys,
                onChange: selection.onChange,
                getCheckboxProps: selection.getCheckboxProps,
                preserveSelectedRowKeys: true,
              } : undefined}
              onRow={(record, index) => ({
                onClick: () => onRowClick?.(record, index || 0),
                onDoubleClick: () => onRowDoubleClick?.(record, index || 0),
                className: clsx(
                  'cursor-pointer transition-colors duration-200',
                  'hover:bg-blue-50 hover:shadow-sm'
                ),
              })}
              expandable={expandable}
              components={virtual ? {
                body: {
                  wrapper: ({ children }) => (
                    <div style={{ height: scroll?.y || 400, overflow: 'auto' }}>
                      {children}
                    </div>
                  ),
                },
              } : undefined}
              className="enhanced-table"
            />
          </Spin>
        </div>

        {/* 自定义分页 */}
        {pagination && (
          <div className="p-4 border-t border-gray-100 flex justify-between items-center">
            <div className="text-sm text-gray-500">
              {pagination.showTotal && total !== undefined && (
                <span>共 {total} 条记录</span>
              )}
            </div>
            <Pagination
              current={pagination.current}
              pageSize={pagination.pageSize}
              total={pagination.total}
              showSizeChanger={pagination.showSizeChanger}
              showQuickJumper={pagination.showQuickJumper}
              onChange={pagination.onChange}
              showTotal={pagination.showTotal ? (total, range) => 
                `第 ${range[0]}-${range[1]} 条，共 ${total} 条`
              : undefined}
              size="small"
            />
          </div>
        )}
      </Card>

      <style jsx global>{`
        .enhanced-table .ant-table-thead > tr > th {
          background: #fafafa;
          font-weight: 600;
          color: #262626;
          border-bottom: 2px solid #f0f0f0;
        }
        
        .enhanced-table .ant-table-tbody > tr:hover > td {
          background: #f8faff !important;
        }
        
        .enhanced-table .ant-table-tbody > tr.ant-table-row-selected > td {
          background: #e6f7ff !important;
        }
        
        .enhanced-table .ant-pagination-item-active {
          border-color: #1890ff;
          background: #1890ff;
        }
        
        .enhanced-table .ant-pagination-item-active a {
          color: #fff;
        }
        
        @media (max-width: 768px) {
          .enhanced-table .ant-table-content {
            overflow-x: auto;
          }
          
          .enhanced-table-container .ant-space-horizontal {
            flex-wrap: wrap;
          }
        }
      `}</style>
    </motion.div>
  );
}

export default EnhancedTable;