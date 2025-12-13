// @ts-nocheck
'use client';

import React from 'react';
import { Table, TableProps, List, Card, Tag } from 'antd';
import { useResponsive } from '@/hooks/useResponsive';
import { cn } from '@/lib/utils';

export interface ResponsiveTableColumn<T = any> {
  key: string;
  title: string;
  dataIndex?: string;
  render?: (value: any, record: T, index: number) => React.ReactNode;
  /**
   * 移动端是否显示
   * @default true
   */
  showInMobile?: boolean;
  /**
   * 移动端显示时的标签
   */
  mobileLabel?: string;
  /**
   * 移动端渲染函数（优先级高于render）
   */
  mobileRender?: (value: any, record: T, index: number) => React.ReactNode;
}

export interface ResponsiveTableProps<T = any> extends Omit<TableProps<T>, 'columns'> {
  columns: ResponsiveTableColumn<T>[];
  /**
   * 移动端断点
   * @default 'md'
   */
  mobileBreakpoint?: 'sm' | 'md' | 'lg';
  /**
   * 移动端卡片点击回调
   */
  onMobileCardClick?: (record: T) => void;
  /**
   * 移动端卡片额外操作
   */
  mobileCardActions?: (record: T) => React.ReactNode;
  /**
   * 移动端列表模式
   * @default 'card' 卡片模式
   */
  mobileMode?: 'card' | 'list';
}

/**
 * 响应式表格组件
 * 
 * 特性：
 * - 桌面端显示完整表格
 * - 移动端自动切换为卡片/列表模式
 * - 支持自定义移动端渲染
 * - 支持移动端操作
 * 
 * @example
 * ```tsx
 * <ResponsiveTable
 *   columns={[
 *     { key: 'id', title: 'ID', dataIndex: 'id' },
 *     { key: 'title', title: '标题', dataIndex: 'title' },
 *     { key: 'status', title: '状态', dataIndex: 'status' },
 *   ]}
 *   dataSource={data}
 *   onMobileCardClick={(record) => router.push(`/tickets/${record.id}`)}
 * />
 * ```
 */
export const ResponsiveTable = <T extends Record<string, any>>({
  columns,
  dataSource = [],
  mobileBreakpoint = 'md',
  onMobileCardClick,
  mobileCardActions,
  mobileMode = 'card',
  loading,
  pagination,
  rowKey = 'id',
  ...props
}: ResponsiveTableProps<T>) => {
  const { isMobile, isTablet } = useResponsive();

  // 判断是否使用移动端布局
  const useMobileLayout = 
    (mobileBreakpoint === 'sm' && isMobile) ||
    (mobileBreakpoint === 'md' && (isMobile || isTablet)) ||
    (mobileBreakpoint === 'lg' && (isMobile || isTablet));

  // 桌面端表格列配置
  const desktopColumns = columns.map(col => ({
    key: col.key,
    title: col.title,
    dataIndex: col.dataIndex,
    render: col.render,
  }));

  // 如果是桌面端，直接渲染表格
  if (!useMobileLayout) {
    return (
      <Table<T>
        columns={desktopColumns}
        dataSource={dataSource}
        loading={loading}
        pagination={pagination}
        rowKey={rowKey}
        {...props}
      />
    );
  }

  // 移动端卡片模式
  if (mobileMode === 'card') {
    return (
      <List<T>
        dataSource={dataSource}
        loading={loading}
        pagination={pagination}
        rowKey={rowKey}
        renderItem={(record, index) => {
          const key = typeof rowKey === 'function' ? rowKey(record) : record[rowKey];
          
          return (
            <Card
              key={key}
              className={cn(
                'mb-3 shadow-sm transition-all duration-200',
                onMobileCardClick && 'cursor-pointer hover:shadow-md active:scale-98'
              )}
              onClick={() => onMobileCardClick?.(record)}
              bodyStyle={{ padding: 16 }}
            >
              <div className="space-y-3">
                {columns
                  .filter(col => col.showInMobile !== false)
                  .map(col => {
                    const value = col.dataIndex ? record[col.dataIndex] : record;
                    const label = col.mobileLabel || col.title;
                    
                    // 使用移动端专用渲染函数，或降级到普通渲染函数
                    const renderedValue = col.mobileRender
                      ? col.mobileRender(value, record, index)
                      : col.render
                      ? col.render(value, record, index)
                      : value;
                    
                    return (
                      <div key={col.key} className="flex justify-between items-start gap-2">
                        <span className="text-gray-600 text-sm font-medium flex-shrink-0">
                          {label}:
                        </span>
                        <span className="text-gray-900 text-sm text-right flex-1">
                          {renderedValue}
                        </span>
                      </div>
                    );
                  })}
                
                {/* 操作按钮 */}
                {mobileCardActions && (
                  <div className="pt-3 border-t border-gray-100 flex justify-end gap-2">
                    {mobileCardActions(record)}
                  </div>
                )}
              </div>
            </Card>
          );
        }}
      />
    );
  }

  // 移动端列表模式（紧凑）
  return (
    <List<T>
      dataSource={dataSource}
      loading={loading}
      pagination={pagination}
      rowKey={rowKey}
      split
      renderItem={(record, index) => {
        const key = typeof rowKey === 'function' ? rowKey(record) : record[rowKey];
        
        return (
          <List.Item
            key={key}
            className={cn(
              'transition-colors duration-200',
              onMobileCardClick && 'cursor-pointer hover:bg-blue-50 active:bg-blue-100'
            )}
            onClick={() => onMobileCardClick?.(record)}
            actions={mobileCardActions ? [mobileCardActions(record)] : undefined}
          >
            <div className="flex-1 space-y-2">
              {columns
                .filter(col => col.showInMobile !== false)
                .map(col => {
                  const value = col.dataIndex ? record[col.dataIndex] : record;
                  
                  const renderedValue = col.mobileRender
                    ? col.mobileRender(value, record, index)
                    : col.render
                    ? col.render(value, record, index)
                    : value;
                  
                  return (
                    <div key={col.key}>
                      <span className="text-gray-600 text-xs">
                        {col.mobileLabel || col.title}:
                      </span>{' '}
                      <span className="text-gray-900 text-sm font-medium">
                        {renderedValue}
                      </span>
                    </div>
                  );
                })}
            </div>
          </List.Item>
        );
      }}
    />
  );
};

/**
 * 响应式表格工具函数 - 状态渲染
 * 
 * 生成适合移动端和桌面端的状态标签
 */
export const renderResponsiveStatus = (
  status: string,
  statusMap: Record<string, { label: string; color: string }>,
  isMobile = false
) => {
  const config = statusMap[status] || { label: status, color: 'default' };
  
  return (
    <Tag
      color={config.color}
      className={cn(
        'rounded-full',
        isMobile && 'text-xs px-2'
      )}
    >
      {config.label}
    </Tag>
  );
};

/**
 * 响应式表格工具函数 - 操作按钮渲染
 * 
 * 生成适合移动端和桌面端的操作按钮
 */
export const renderResponsiveActions = (
  actions: Array<{
    label: string;
    icon?: React.ReactNode;
    onClick: () => void;
    type?: 'primary' | 'danger' | 'default';
  }>,
  isMobile = false
) => {
  if (isMobile) {
    return (
      <div className="flex gap-2">
        {actions.map((action, index) => (
          <button
            key={index}
            onClick={(e) => {
              e.stopPropagation();
              action.onClick();
            }}
            className={cn(
              'px-3 py-1.5 rounded-lg text-xs font-medium transition-colors',
              'inline-flex items-center gap-1 min-h-[44px]',
              {
                'bg-blue-500 text-white active:bg-blue-600': action.type === 'primary',
                'bg-red-500 text-white active:bg-red-600': action.type === 'danger',
                'bg-gray-100 text-gray-700 active:bg-gray-200': !action.type || action.type === 'default',
              }
            )}
          >
            {action.icon}
            <span>{action.label}</span>
          </button>
        ))}
      </div>
    );
  }

  return (
    <div className="flex items-center gap-2">
      {actions.map((action, index) => (
        <button
          key={index}
          onClick={(e) => {
            e.stopPropagation();
            action.onClick();
          }}
          className={cn(
            'px-3 py-1 rounded text-sm font-medium transition-colors',
            'inline-flex items-center gap-1',
            {
              'text-blue-600 hover:bg-blue-50': action.type === 'primary' || !action.type,
              'text-red-600 hover:bg-red-50': action.type === 'danger',
              'text-gray-600 hover:bg-gray-50': action.type === 'default',
            }
          )}
        >
          {action.icon}
          <span>{action.label}</span>
        </button>
      ))}
    </div>
  );
};

export default ResponsiveTable;

