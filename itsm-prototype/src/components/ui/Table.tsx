'use client';

import React, { useState, useMemo } from 'react';
import { ChevronUp, ChevronDown, Filter } from 'lucide-react';
import { cn } from '@/lib/utils';

/**
 * 表格列定义
 */
export interface TableColumn<T = Record<string, unknown>> {
  /** 列键 */
  key: string;
  /** 列标题 */
  title: string;
  /** 数据索引 */
  dataIndex?: string;
  /** 列宽 */
  width?: number | string;
  /** 最小宽度 */
  minWidth?: number;
  /** 是否可排序 */
  sortable?: boolean;
  /** 是否可筛选 */
  filterable?: boolean;
  /** 筛选选项 */
  filterOptions?: Array<{ label: string; value: unknown }>;
  /** 对齐方式 */
  align?: 'left' | 'center' | 'right';
  /** 是否固定 */
  fixed?: 'left' | 'right';
  /** 自定义渲染 */
  render?: (value: unknown, record: T, index: number) => React.ReactNode;
  /** 排序函数 */
  sorter?: (a: T, b: T) => number;
  /** 筛选函数 */
  onFilter?: (value: unknown, record: T) => boolean;
}

/**
 * 表格属性
 */
export interface TableProps<T = Record<string, unknown>> {
  /** 数据源 */
  dataSource: T[];
  /** 列定义 */
  columns: TableColumn<T>[];
  /** 行键 */
  rowKey?: string | ((record: T) => string);
  /** 是否加载中 */
  loading?: boolean;
  /** 空数据提示 */
  emptyText?: string;
  /** 表格尺寸 */
  size?: 'sm' | 'md' | 'lg';
  /** 是否显示边框 */
  bordered?: boolean;
  /** 是否显示斑马纹 */
  striped?: boolean;
  /** 是否可选择行 */
  rowSelection?: {
    /** 选择类型 */
    type?: 'checkbox' | 'radio';
    /** 选中的行键 */
    selectedRowKeys?: (string | number)[];
    /** 选择变化回调 */
    onChange?: (selectedRowKeys: (string | number)[], selectedRows: T[]) => void;
    /** 获取选择框属性 */
    getCheckboxProps?: (record: T) => { disabled?: boolean };
  };
  /** 分页配置 */
  pagination?: {
    /** 当前页 */
    current?: number;
    /** 每页条数 */
    pageSize?: number;
    /** 总条数 */
    total?: number;
    /** 显示快速跳转 */
    showQuickJumper?: boolean;
    /** 显示每页条数选择器 */
    showSizeChanger?: boolean;
    /** 页码变化回调 */
    onChange?: (page: number, pageSize: number) => void;
  } | false;
  /** 滚动配置 */
  scroll?: {
    x?: number | string;
    y?: number | string;
  };
  /** 表格类名 */
  className?: string;
  /** 行点击事件 */
  onRow?: (record: T, index: number) => {
    onClick?: (event: React.MouseEvent) => void;
    onDoubleClick?: (event: React.MouseEvent) => void;
    onContextMenu?: (event: React.MouseEvent) => void;
  };
}

/**
 * 通用表格组件
 */
export const Table = <T extends Record<string, unknown>>({
  dataSource,
  columns,
  rowKey = 'id',
  loading = false,
  emptyText = '暂无数据',
  size = 'md',
  bordered = false,
  striped = false,
  rowSelection,
  pagination,
  scroll,
  className,
  onRow,
}: TableProps<T>) => {
  const [sortConfig, setSortConfig] = useState<{
    key: string;
    direction: 'asc' | 'desc';
  } | null>(null);
  
  const [selectedRowKeys, setSelectedRowKeys] = useState<(string | number)[]>(
    rowSelection?.selectedRowKeys || []
  );

  // 获取行键
  const getRowKey = (record: T, index: number): string => {
    if (typeof rowKey === 'function') {
      return rowKey(record);
    }
    return String(record[rowKey as keyof T] || index);
  };

  // 处理排序
  const handleSort = (column: TableColumn<T>) => {
    if (!column.sortable) return;

    const newDirection = 
      sortConfig?.key === column.key && sortConfig.direction === 'asc' 
        ? 'desc' 
        : 'asc';

    setSortConfig({
      key: column.key,
      direction: newDirection,
    });
  };



  // 处理行选择
  const handleRowSelect = (recordKey: string | number, selected: boolean) => {
    const newSelectedKeys = selected
      ? [...selectedRowKeys, recordKey]
      : selectedRowKeys.filter(key => key !== recordKey);

    setSelectedRowKeys(newSelectedKeys);
    
    if (rowSelection?.onChange) {
      const selectedRecords = dataSource.filter(record => 
        newSelectedKeys.includes(getRowKey(record, 0))
      );
      rowSelection.onChange(newSelectedKeys, selectedRecords);
    }
  };

  // 全选处理
  const handleSelectAll = (selected: boolean) => {
    const newSelectedKeys = selected
      ? dataSource.map((record, index) => getRowKey(record, index))
      : [];

    setSelectedRowKeys(newSelectedKeys);
    
    if (rowSelection?.onChange) {
      const selectedRecords = selected ? dataSource : [];
      rowSelection.onChange(newSelectedKeys, selectedRecords);
    }
  };

  // 处理数据排序和筛选
  const processedData = useMemo(() => {
    const result = [...dataSource];

    // 应用排序
    if (sortConfig) {
      const column = columns.find(col => col.key === sortConfig.key);
      if (column?.sorter) {
        result.sort((a, b) => {
          const sortResult = column.sorter!(a, b);
          return sortConfig.direction === 'desc' ? -sortResult : sortResult;
        });
      } else if (column?.dataIndex) {
        result.sort((a, b) => {
          const aValue = a[column.dataIndex as keyof T];
          const bValue = b[column.dataIndex as keyof T];
          
          if (aValue < bValue) return sortConfig.direction === 'asc' ? -1 : 1;
          if (aValue > bValue) return sortConfig.direction === 'asc' ? 1 : -1;
          return 0;
        });
      }
    }

    return result;
  }, [dataSource, columns, sortConfig]);

  // 渲染单元格内容
  const renderCellContent = (column: TableColumn<T>, record: T, index: number) => {
    const value = column.dataIndex ? record[column.dataIndex as keyof T] : undefined;
    
    if (column.render) {
      return column.render(value, record, index);
    }
    
    return String(value || '');
  };

  // 表格尺寸样式
  const sizeClasses = {
    sm: 'text-sm',
    md: 'text-base',
    lg: 'text-lg',
  };

  // 单元格内边距
  const cellPadding = {
    sm: 'px-2 py-1',
    md: 'px-3 py-2',
    lg: 'px-4 py-3',
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  return (
    <div className={cn('overflow-hidden', className)}>
      <div className={cn(
        'overflow-auto',
        scroll?.x && 'overflow-x-auto',
        scroll?.y && 'overflow-y-auto'
      )} style={{ maxHeight: scroll?.y }}>
        <table className={cn(
          'w-full table-auto',
          sizeClasses[size],
          bordered && 'border border-gray-200',
          'bg-white'
        )}>
          <thead className="bg-gray-50">
            <tr>
              {rowSelection && (
                <th className={cn(
                  'text-left font-medium text-gray-900',
                  cellPadding[size],
                  bordered && 'border-b border-gray-200'
                )}>
                  {rowSelection.type !== 'radio' && (
                    <input
                      type="checkbox"
                      checked={selectedRowKeys.length === dataSource.length && dataSource.length > 0}
                      onChange={(e) => handleSelectAll(e.target.checked)}
                      className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                    />
                  )}
                </th>
              )}
              {columns.map((column) => (
                <th
                  key={column.key}
                  className={cn(
                    'text-left font-medium text-gray-900',
                    cellPadding[size],
                    bordered && 'border-b border-gray-200',
                    column.align === 'center' && 'text-center',
                    column.align === 'right' && 'text-right',
                    column.sortable && 'cursor-pointer hover:bg-gray-100'
                  )}
                  style={{ 
                    width: column.width,
                    minWidth: column.minWidth 
                  }}
                  onClick={() => handleSort(column)}
                >
                  <div className="flex items-center gap-1">
                    <span>{column.title}</span>
                    {column.sortable && (
                      <div className="flex flex-col">
                        <ChevronUp 
                          className={cn(
                            'h-3 w-3',
                            sortConfig?.key === column.key && sortConfig.direction === 'asc'
                              ? 'text-blue-600'
                              : 'text-gray-400'
                          )}
                        />
                        <ChevronDown 
                          className={cn(
                            'h-3 w-3 -mt-1',
                            sortConfig?.key === column.key && sortConfig.direction === 'desc'
                              ? 'text-blue-600'
                              : 'text-gray-400'
                          )}
                        />
                      </div>
                    )}
                    {column.filterable && (
                      <Filter className="h-3 w-3 text-gray-400" />
                    )}
                  </div>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {processedData.length === 0 ? (
              <tr>
                <td 
                  colSpan={columns.length + (rowSelection ? 1 : 0)}
                  className={cn(
                    'text-center text-gray-500 py-8',
                    cellPadding[size]
                  )}
                >
                  {emptyText}
                </td>
              </tr>
            ) : (
              processedData.map((record, index) => {
                const recordKey = getRowKey(record, index);
                const isSelected = selectedRowKeys.includes(recordKey);
                const rowProps = onRow?.(record, index) || {};

                return (
                  <tr
                    key={recordKey}
                    className={cn(
                      'hover:bg-gray-50',
                      striped && index % 2 === 1 && 'bg-gray-25',
                      isSelected && 'bg-blue-50',
                      bordered && 'border-b border-gray-200'
                    )}
                    {...rowProps}
                  >
                    {rowSelection && (
                      <td className={cn(cellPadding[size])}>
                        <input
                          type={rowSelection.type || 'checkbox'}
                          checked={isSelected}
                          onChange={(e) => handleRowSelect(recordKey, e.target.checked)}
                          disabled={rowSelection.getCheckboxProps?.(record)?.disabled}
                          className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                        />
                      </td>
                    )}
                    {columns.map((column) => (
                      <td
                        key={column.key}
                        className={cn(
                          cellPadding[size],
                          column.align === 'center' && 'text-center',
                          column.align === 'right' && 'text-right'
                        )}
                        style={{ 
                          width: column.width,
                          minWidth: column.minWidth 
                        }}
                      >
                        {renderCellContent(column, record, index)}
                      </td>
                    ))}
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>
      
      {pagination && pagination !== false && (
        <div className="flex items-center justify-between px-4 py-3 bg-white border-t border-gray-200">
          <div className="flex items-center text-sm text-gray-700">
            共 {pagination.total || 0} 条记录
          </div>
          <div className="flex items-center gap-2">
            <button
              disabled={pagination.current === 1}
              onClick={() => pagination.onChange?.(pagination.current! - 1, pagination.pageSize!)}
              className="px-3 py-1 text-sm border border-gray-300 rounded hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              上一页
            </button>
            <span className="px-3 py-1 text-sm">
              第 {pagination.current} 页
            </span>
            <button
              disabled={pagination.current === Math.ceil((pagination.total || 0) / (pagination.pageSize || 10))}
              onClick={() => pagination.onChange?.(pagination.current! + 1, pagination.pageSize!)}
              className="px-3 py-1 text-sm border border-gray-300 rounded hover:bg-gray-50 disabled:cursor-not-allowed"
            >
              下一页
            </button>
          </div>
        </div>
      )}
    </div>
  );
};

/**
 * 简单表格组件（无分页和行选择）
 */
export const SimpleTable = <T extends Record<string, unknown>>(
  props: Omit<TableProps<T>, 'pagination' | 'rowSelection'>
) => {
  return <Table {...props} pagination={false} />;
};