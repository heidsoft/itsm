/**
 * ITSM前端性能优化 - 虚拟滚动组件
 *
 * 处理大数据列表的虚拟滚动实现
 */

import React, { useState, useEffect, useRef, useMemo, useCallback } from 'react';
import { List, VariableSizeList } from 'react-window';
import { Table, Card, Button, Input, Space, Tag, Tooltip } from 'antd';
import { Search, Edit, Eye, Trash2 } from 'lucide-react';

// ==================== 虚拟滚动列表组件 ====================

interface VirtualListProps<T> {
  items: T[];
  height: number;
  itemHeight?: number;
  itemRenderer: (props: { index: number; style: React.CSSProperties; item: T }) => React.ReactNode;
  onScroll?: (scrollTop: number) => void;
  overscan?: number;
}

export function VirtualList<T>({
  items,
  height,
  itemHeight = 50,
  itemRenderer,
  onScroll,
  overscan = 5,
}: VirtualListProps<T>) {
  const listRef = useRef<List>(null);

  const handleScroll = useCallback(
    ({ scrollTop }: { scrollTop: number }) => {
      onScroll?.(scrollTop);
    },
    [onScroll]
  );

  return (
    <List
      ref={listRef}
      height={height}
      itemCount={items.length}
      itemSize={itemHeight}
      onScroll={handleScroll}
      overscanCount={overscan}
    >
      {({ index, style }) => itemRenderer({ index, style, item: items[index] })}
    </List>
  );
}

// ==================== 可变高度虚拟滚动列表 ====================

interface VariableVirtualListProps<T> {
  items: T[];
  height: number;
  getItemHeight: (index: number) => number;
  itemRenderer: (props: { index: number; style: React.CSSProperties; item: T }) => React.ReactNode;
  onScroll?: (scrollTop: number) => void;
  overscan?: number;
}

export function VariableVirtualList<T>({
  items,
  height,
  getItemHeight,
  itemRenderer,
  onScroll,
  overscan = 5,
}: VariableVirtualListProps<T>) {
  const listRef = useRef<VariableSizeList>(null);

  const handleScroll = useCallback(
    ({ scrollTop }: { scrollTop: number }) => {
      onScroll?.(scrollTop);
    },
    [onScroll]
  );

  return (
    <VariableSizeList
      ref={listRef}
      height={height}
      itemCount={items.length}
      itemSize={getItemHeight}
      onScroll={handleScroll}
      overscanCount={overscan}
    >
      {({ index, style }) => itemRenderer({ index, style, item: items[index] })}
    </VariableSizeList>
  );
}

// ==================== 虚拟滚动表格组件 ====================

interface Ticket {
  id: number;
  title: string;
  status: 'open' | 'in_progress' | 'resolved' | 'closed';
  priority: 'low' | 'medium' | 'high' | 'urgent';
  assignee: string;
  created_at: string;
  updated_at: string;
}

interface VirtualTableProps {
  tickets: Ticket[];
  loading?: boolean;
  onEdit: (id: number) => void;
  onView: (id: number) => void;
  onDelete: (id: number) => void;
  permissions?: string[];
  height?: number;
}

export const VirtualTable = memo<VirtualTableProps>(
  ({ tickets, loading = false, onEdit, onView, onDelete, permissions = [], height = 400 }) => {
    const [scrollTop, setScrollTop] = useState(0);
    const [visibleRange, setVisibleRange] = useState({ start: 0, end: 20 });

    // 计算可见范围
    const updateVisibleRange = useCallback(
      (scrollTop: number) => {
        const itemHeight = 50;
        const containerHeight = height;
        const start = Math.floor(scrollTop / itemHeight);
        const end = Math.min(start + Math.ceil(containerHeight / itemHeight) + 5, tickets.length);

        setVisibleRange({ start, end });
      },
      [height, tickets.length]
    );

    // 处理滚动事件
    const handleScroll = useCallback(
      (scrollTop: number) => {
        setScrollTop(scrollTop);
        updateVisibleRange(scrollTop);
      },
      [updateVisibleRange]
    );

    // 渲染表格行
    const renderRow = useCallback(
      ({ index, style, item }: { index: number; style: React.CSSProperties; item: Ticket }) => {
        const statusColors = {
          open: 'blue',
          in_progress: 'orange',
          resolved: 'green',
          closed: 'gray',
        };

        const priorityColors = {
          low: 'green',
          medium: 'blue',
          high: 'orange',
          urgent: 'red',
        };

        return (
          <div style={style} className='virtual-table-row'>
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                padding: '8px 16px',
                borderBottom: '1px solid #f0f0f0',
              }}
            >
              <div style={{ width: 80, flexShrink: 0 }}>{item.id}</div>
              <div style={{ flex: 1, minWidth: 0, paddingRight: 16 }}>
                <div
                  style={{
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {item.title}
                </div>
              </div>
              <div style={{ width: 100, flexShrink: 0 }}>
                <Tag color={statusColors[item.status]}>{item.status}</Tag>
              </div>
              <div style={{ width: 100, flexShrink: 0 }}>
                <Tag color={priorityColors[item.priority]}>{item.priority}</Tag>
              </div>
              <div style={{ width: 120, flexShrink: 0 }}>{item.assignee}</div>
              <div style={{ width: 150, flexShrink: 0 }}>
                {new Date(item.created_at).toLocaleDateString()}
              </div>
              <div style={{ width: 120, flexShrink: 0 }}>
                <Space size='small'>
                  <Tooltip title='查看详情'>
                    <Button
                      type='text'
                      size='small'
                      icon={<Eye size={14} />}
                      onClick={() => onView(item.id)}
                    />
                  </Tooltip>
                  {permissions.includes('ticket:edit') && (
                    <Tooltip title='编辑工单'>
                      <Button
                        type='text'
                        size='small'
                        icon={<Edit size={14} />}
                        onClick={() => onEdit(item.id)}
                      />
                    </Tooltip>
                  )}
                  {permissions.includes('ticket:delete') && (
                    <Tooltip title='删除工单'>
                      <Button
                        type='text'
                        size='small'
                        danger
                        icon={<Trash2 size={14} />}
                        onClick={() => onDelete(item.id)}
                      />
                    </Tooltip>
                  )}
                </Space>
              </div>
            </div>
          </div>
        );
      },
      [onEdit, onView, onDelete, permissions]
    );

    // 表格头部
    const tableHeader = useMemo(
      () => (
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            padding: '12px 16px',
            backgroundColor: '#fafafa',
            borderBottom: '1px solid #d9d9d9',
            fontWeight: 600,
            fontSize: '14px',
          }}
        >
          <div style={{ width: 80, flexShrink: 0 }}>ID</div>
          <div style={{ flex: 1, minWidth: 0, paddingRight: 16 }}>标题</div>
          <div style={{ width: 100, flexShrink: 0 }}>状态</div>
          <div style={{ width: 100, flexShrink: 0 }}>优先级</div>
          <div style={{ width: 120, flexShrink: 0 }}>处理人</div>
          <div style={{ width: 150, flexShrink: 0 }}>创建时间</div>
          <div style={{ width: 120, flexShrink: 0 }}>操作</div>
        </div>
      ),
      []
    );

    if (loading) {
      return (
        <Card>
          {tableHeader}
          <div style={{ height, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            加载中...
          </div>
        </Card>
      );
    }

    return (
      <Card>
        {tableHeader}
        <div style={{ height }}>
          <VirtualList
            items={tickets}
            height={height}
            itemHeight={50}
            itemRenderer={renderRow}
            onScroll={handleScroll}
            overscan={10}
          />
        </div>
        <div
          style={{
            padding: '8px 16px',
            backgroundColor: '#fafafa',
            borderTop: '1px solid #d9d9d9',
            fontSize: '12px',
            color: '#666',
          }}
        >
          显示 {visibleRange.start + 1}-{visibleRange.end} 条，共 {tickets.length} 条记录
        </div>
      </Card>
    );
  }
);

VirtualTable.displayName = 'VirtualTable';

// ==================== 虚拟滚动网格组件 ====================

interface VirtualGridProps<T> {
  items: T[];
  height: number;
  itemHeight: number;
  itemWidth: number;
  columns: number;
  itemRenderer: (props: {
    index: number;
    style: React.CSSProperties;
    item: T;
    row: number;
    column: number;
  }) => React.ReactNode;
  onScroll?: (scrollTop: number) => void;
}

export function VirtualGrid<T>({
  items,
  height,
  itemHeight,
  itemWidth,
  columns,
  itemRenderer,
  onScroll,
}: VirtualGridProps<T>) {
  const rows = Math.ceil(items.length / columns);

  const renderRow = useCallback(
    ({ index, style }: { index: number; style: React.CSSProperties }) => {
      const rowItems = [];
      const startIndex = index * columns;

      for (let i = 0; i < columns; i++) {
        const itemIndex = startIndex + i;
        if (itemIndex < items.length) {
          rowItems.push(
            itemRenderer({
              index: itemIndex,
              style: { width: itemWidth, height: itemHeight },
              item: items[itemIndex],
              row: index,
              column: i,
            })
          );
        }
      }

      return (
        <div style={style}>
          <div style={{ display: 'flex' }}>{rowItems}</div>
        </div>
      );
    },
    [items, columns, itemWidth, itemHeight, itemRenderer]
  );

  return (
    <List
      height={height}
      itemCount={rows}
      itemSize={itemHeight}
      onScroll={({ scrollTop }) => onScroll?.(scrollTop)}
    >
      {renderRow}
    </List>
  );
}

// ==================== 虚拟滚动选择器组件 ====================

interface VirtualSelectProps {
  options: Array<{ value: string; label: string }>;
  value?: string;
  onChange: (value: string) => void;
  placeholder?: string;
  height?: number;
  itemHeight?: number;
}

export const VirtualSelect = memo<VirtualSelectProps>(
  ({ options, value, onChange, placeholder = '请选择', height = 200, itemHeight = 32 }) => {
    const [isOpen, setIsOpen] = useState(false);
    const [searchValue, setSearchValue] = useState('');

    // 过滤选项
    const filteredOptions = useMemo(() => {
      if (!searchValue) return options;
      return options.filter(option =>
        option.label.toLowerCase().includes(searchValue.toLowerCase())
      );
    }, [options, searchValue]);

    // 渲染选项
    const renderOption = useCallback(
      ({
        index,
        style,
        item,
      }: {
        index: number;
        style: React.CSSProperties;
        item: { value: string; label: string };
      }) => (
        <div
          style={{
            ...style,
            padding: '8px 12px',
            cursor: 'pointer',
            backgroundColor: item.value === value ? '#e6f7ff' : 'transparent',
            borderBottom: '1px solid #f0f0f0',
          }}
          onClick={() => {
            onChange(item.value);
            setIsOpen(false);
            setSearchValue('');
          }}
          onMouseEnter={e => {
            if (item.value !== value) {
              e.currentTarget.style.backgroundColor = '#f5f5f5';
            }
          }}
          onMouseLeave={e => {
            if (item.value !== value) {
              e.currentTarget.style.backgroundColor = 'transparent';
            }
          }}
        >
          {item.label}
        </div>
      ),
      [value, onChange]
    );

    const selectedOption = options.find(option => option.value === value);

    return (
      <div style={{ position: 'relative', width: '100%' }}>
        <div
          style={{
            padding: '8px 12px',
            border: '1px solid #d9d9d9',
            borderRadius: '6px',
            cursor: 'pointer',
            backgroundColor: '#fff',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
          }}
          onClick={() => setIsOpen(!isOpen)}
        >
          <span style={{ color: selectedOption ? '#000' : '#999' }}>
            {selectedOption ? selectedOption.label : placeholder}
          </span>
          <span style={{ color: '#999' }}>{isOpen ? '▲' : '▼'}</span>
        </div>

        {isOpen && (
          <div
            style={{
              position: 'absolute',
              top: '100%',
              left: 0,
              right: 0,
              backgroundColor: '#fff',
              border: '1px solid #d9d9d9',
              borderRadius: '6px',
              boxShadow: '0 2px 8px rgba(0,0,0,0.15)',
              zIndex: 1000,
              marginTop: '4px',
            }}
          >
            <div style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0' }}>
              <Input
                placeholder='搜索...'
                value={searchValue}
                onChange={e => setSearchValue(e.target.value)}
                size='small'
                style={{ border: 'none', boxShadow: 'none' }}
              />
            </div>
            <div style={{ height }}>
              <VirtualList
                items={filteredOptions}
                height={height}
                itemHeight={itemHeight}
                itemRenderer={renderOption}
              />
            </div>
          </div>
        )}
      </div>
    );
  }
);

VirtualSelect.displayName = 'VirtualSelect';

// ==================== 虚拟滚动性能监控 ====================

export function useVirtualScrollPerformance() {
  const [metrics, setMetrics] = useState({
    renderTime: 0,
    scrollFPS: 0,
    memoryUsage: 0,
  });

  const frameCountRef = useRef(0);
  const lastTimeRef = useRef(Date.now());

  const updateMetrics = useCallback(() => {
    const now = Date.now();
    const deltaTime = now - lastTimeRef.current;

    if (deltaTime >= 1000) {
      const fps = (frameCountRef.current * 1000) / deltaTime;

      setMetrics(prev => ({
        ...prev,
        scrollFPS: Math.round(fps),
        memoryUsage: (performance as any).memory?.usedJSHeapSize || 0,
      }));

      frameCountRef.current = 0;
      lastTimeRef.current = now;
    }

    frameCountRef.current++;
    requestAnimationFrame(updateMetrics);
  }, []);

  useEffect(() => {
    const animationId = requestAnimationFrame(updateMetrics);
    return () => cancelAnimationFrame(animationId);
  }, [updateMetrics]);

  return metrics;
}

export default {
  VirtualList,
  VariableVirtualList,
  VirtualTable,
  VirtualGrid,
  VirtualSelect,
  useVirtualScrollPerformance,
};
