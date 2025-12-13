// @ts-nocheck
'use client';

import React from 'react';
import { Table, TableProps, Empty } from 'antd';
import { cn } from '@/lib/utils';
import { EnterpriseCard } from './EnterpriseCard';

export interface EnterpriseTableProps<T = any> extends TableProps<T> {
  /**
   * 是否包裹在卡片中
   * @default true
   */
  card?: boolean;
  
  /**
   * 是否启用悬浮行高亮
   * @default true
   */
  hoverRow?: boolean;
  
  /**
   * 是否显示斑马纹
   * @default false
   */
  striped?: boolean;
  
  /**
   * 卡片标题
   */
  cardTitle?: string;
  
  /**
   * 卡片右侧额外内容
   */
  cardExtra?: React.ReactNode;
  
  /**
   * 是否紧凑模式
   * @default false
   */
  compact?: boolean;
}

/**
 * 企业级表格组件
 * 
 * 特性：
 * - 统一的样式和间距
 * - 悬浮行高亮效果
 * - 可选斑马纹
 * - 卡片包裹
 * - 响应式分页
 * - 空状态优化
 * 
 * @example
 * ```tsx
 * <EnterpriseTable
 *   cardTitle="工单列表"
 *   columns={columns}
 *   dataSource={data}
 *   hoverRow
 * />
 * ```
 */
export const EnterpriseTable = <T extends Record<string, any>>({
  card = true,
  hoverRow = true,
  striped = false,
  cardTitle,
  cardExtra,
  compact = false,
  className,
  rowClassName,
  pagination,
  locale,
  ...props
}: EnterpriseTableProps<T>) => {
  // 默认分页配置
  const defaultPagination = {
    showSizeChanger: true,
    showQuickJumper: true,
    showTotal: (total: number) => `共 ${total} 条记录`,
    pageSizeOptions: ['10', '20', '50', '100'],
    defaultPageSize: 20,
    style: { marginTop: compact ? 16 : 24 },
    ...pagination,
  };

  // 行类名
  const getRowClassName = (record: T, index: number) => {
    const customClass = typeof rowClassName === 'function' 
      ? rowClassName(record, index) 
      : rowClassName || '';
    
    return cn(
      customClass,
      hoverRow && 'hover:bg-blue-50 transition-colors duration-200',
      striped && index % 2 === 1 && 'bg-gray-50'
    );
  };

  // 自定义空状态
  const defaultLocale = {
    emptyText: (
      <Empty
        image={Empty.PRESENTED_IMAGE_SIMPLE}
        description="暂无数据"
        style={{ padding: '48px 0' }}
      />
    ),
    ...locale,
  };

  const tableNode = (
    <Table<T>
      {...props}
      className={cn(
        'enterprise-table',
        compact && 'enterprise-table-compact',
        className
      )}
      rowClassName={getRowClassName}
      pagination={pagination !== false ? defaultPagination : false}
      locale={defaultLocale}
      bordered={false}
      size={compact ? 'small' : 'middle'}
      style={{
        ...props.style,
      }}
    />
  );

  // 如果需要卡片包裹
  if (card) {
    return (
      <EnterpriseCard 
        hover={false}
        style={{ padding: compact ? 16 : 24 }}
      >
        {cardTitle && (
          <div className="flex items-center justify-between mb-4 pb-3 border-b border-gray-100">
            <h3 className="text-lg font-semibold text-gray-900">
              {cardTitle}
            </h3>
            {cardExtra && <div>{cardExtra}</div>}
          </div>
        )}
        {tableNode}
      </EnterpriseCard>
    );
  }

  return tableNode;
};

/**
 * 企业级表格操作列
 * 
 * 用于生成统一样式的操作列
 */
export interface TableAction {
  label: string;
  onClick: () => void;
  icon?: React.ReactNode;
  type?: 'primary' | 'danger' | 'default';
  disabled?: boolean;
  tooltip?: string;
}

export const renderTableActions = (actions: TableAction[]) => {
  return (
    <div className="flex items-center gap-2">
      {actions.map((action, index) => (
        <button
          key={index}
          onClick={action.onClick}
          disabled={action.disabled}
          title={action.tooltip}
          className={cn(
            'px-3 py-1 rounded text-sm font-medium transition-colors duration-200',
            'inline-flex items-center gap-1',
            action.disabled && 'opacity-50 cursor-not-allowed',
            !action.disabled && {
              'text-blue-600 hover:bg-blue-50': action.type === 'primary' || !action.type,
              'text-red-600 hover:bg-red-50': action.type === 'danger',
              'text-gray-600 hover:bg-gray-50': action.type === 'default',
            }
          )}
        >
          {action.icon}
          {action.label}
        </button>
      ))}
    </div>
  );
};

/**
 * 企业级表格工具栏
 * 
 * 表格上方的工具栏，包含搜索、筛选、操作等
 */
export interface EnterpriseTableToolbarProps {
  title?: string;
  search?: React.ReactNode;
  filters?: React.ReactNode;
  actions?: React.ReactNode;
  extra?: React.ReactNode;
}

export const EnterpriseTableToolbar: React.FC<EnterpriseTableToolbarProps> = ({
  title,
  search,
  filters,
  actions,
  extra,
}) => {
  return (
    <div className="mb-4 space-y-4">
      {/* 标题和主要操作 */}
      {(title || actions) && (
        <div className="flex items-center justify-between">
          {title && (
            <h2 className="text-xl font-semibold text-gray-900">
              {title}
            </h2>
          )}
          {actions && <div className="flex items-center gap-2">{actions}</div>}
        </div>
      )}
      
      {/* 搜索和筛选 */}
      {(search || filters || extra) && (
        <div className="flex items-center gap-4 flex-wrap">
          {search && <div className="flex-1 min-w-[200px]">{search}</div>}
          {filters && <div className="flex items-center gap-2">{filters}</div>}
          {extra && <div>{extra}</div>}
        </div>
      )}
    </div>
  );
};

export default EnterpriseTable;

