'use client';
// @ts-nocheck

import React, { useMemo, useCallback } from 'react';
import { Card, Select, Input, DatePicker, Button, Space, Row, Col, Skeleton, Tag, Badge } from 'antd';
import { SearchOutlined, ReloadOutlined, FilterOutlined, ClearOutlined, CloseOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { TicketViewSelector } from './TicketViewSelector';
import { TicketView } from '@/lib/api/ticket-view-api';
import { FilterPresetSelector } from './FilterPresetSelector';
import { debounce } from 'lodash-es';

const { RangePicker } = DatePicker;

export type TicketFilterState = {
  status: 'all' | 'open' | 'in_progress' | 'resolved' | 'closed';
  priority: 'all' | 'p1' | 'p2' | 'p3' | 'p4';
  type?: 'all' | string;
  keyword: string;
  dateStart: string; // YYYY-MM-DD
  dateEnd: string; // YYYY-MM-DD
  sortBy: 'createdAt_desc' | 'createdAt_asc' | 'priority_desc' | 'priority_asc';
};

interface Props {
  filters: TicketFilterState;
  onFilterChange: (filters: Partial<TicketFilterState>) => void;
  onSearch?: (keyword: string) => void;
  onRefresh?: () => void;
  loading?: boolean;
  currentViewId?: number;
  onViewChange?: (view: TicketView | null) => void;
}

const DEFAULT_VALUE: TicketFilterState = {
  status: 'all',
  priority: 'all',
  type: 'all',
  keyword: '',
  dateStart: '',
  dateEnd: '',
  sortBy: 'createdAt_desc',
};

const TicketFiltersSkeleton: React.FC = () => (
  <Card className='mb-4' styles={{ body: { padding: '16px' } }}>
    <Space direction='vertical' style={{ width: '100%' }} size='middle'>
      <Row gutter={[12, 12]} align='middle'>
                <Col xs={24} sm={24} md={8} lg={6}>
                    <Skeleton.Input active style={{ width: '100%' }} />
                </Col>
                <Col xs={12} sm={8} md={4} lg={3}>
                    <Skeleton.Input active style={{ width: '100%' }} />
                </Col>
                <Col xs={12} sm={8} md={4} lg={3}>
                    <Skeleton.Input active style={{ width: '100%' }} />
                </Col>
                <Col xs={24} sm={12} md={6} lg={5}>
                    <Skeleton.Input active style={{ width: '100%' }} />
                </Col>
                <Col xs={12} sm={8} md={4} lg={3}>
                    <Skeleton.Input active style={{ width: '100%' }} />
                </Col>
                <Col xs={12} sm={8} md={6} lg={4}>
                    <Skeleton.Button active />
                </Col>
            </Row>
        </Space>
    </Card>
);

// 组件策略：使用Ant Design组件提升UI质量
function TicketFilters({
  filters = DEFAULT_VALUE,
  onFilterChange,
  onSearch,
  onRefresh,
  loading = false,
  currentViewId,
  onViewChange,
}: Props) {
  const [localKeyword, setLocalKeyword] = React.useState(filters?.keyword || '');

  // 同步外部filters到本地keyword
  React.useEffect(() => {
    if (filters?.keyword !== undefined) {
      setLocalKeyword(filters.keyword);
    }
  }, [filters?.keyword]);

  // 实时筛选：状态、优先级、排序立即生效
  const handleStatusChange = (value: string) => {
    onFilterChange({ status: value as TicketFilterState['status'] });
  };

  const handlePriorityChange = (value: string) => {
    onFilterChange({ priority: value as TicketFilterState['priority'] });
  };

  const handleSortChange = (value: string) => {
    onFilterChange({ sortBy: value as TicketFilterState['sortBy'] });
  };

  // 搜索框防抖处理
  const debouncedSearch = useMemo(
    () =>
      debounce((keyword: string) => {
        if (onSearch) {
          onSearch(keyword);
        } else {
          onFilterChange({ keyword });
        }
      }, 300),
    [onSearch, onFilterChange]
  );

  const handleSearchInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    setLocalKeyword(value);
    debouncedSearch(value);
  };

  const handleSearch = () => {
    if (onSearch) {
      onSearch(localKeyword);
    } else {
      onFilterChange({ keyword: localKeyword });
    }
  };

  const handleDateRangeChange = (dates: [dayjs.Dayjs, dayjs.Dayjs] | null) => {
    if (dates && dates[0] && dates[1]) {
      onFilterChange({
        dateStart: dates[0].format('YYYY-MM-DD'),
        dateEnd: dates[1].format('YYYY-MM-DD'),
      });
    } else {
      onFilterChange({
        dateStart: '',
        dateEnd: '',
      });
    }
  };

  const handleReset = () => {
    setLocalKeyword('');
    debouncedSearch.cancel(); // 取消待执行的防抖搜索
    onFilterChange(DEFAULT_VALUE);
  };

  const dateRange =
    filters?.dateStart && filters?.dateEnd
      ? [dayjs(filters.dateStart), dayjs(filters.dateEnd)]
      : null;

  // 计算已应用的筛选条件
  const activeFilters = useMemo(() => {
    const active: Array<{ key: string; label: string; value: string }> = [];
    
    if (filters?.status && filters.status !== 'all') {
      const statusLabels: Record<string, string> = {
        open: '待处理',
        in_progress: '处理中',
        resolved: '已解决',
        closed: '已关闭',
      };
      active.push({ key: 'status', label: '状态', value: statusLabels[filters.status] || filters.status });
    }
    
    if (filters?.priority && filters.priority !== 'all') {
      const priorityLabels: Record<string, string> = {
        p1: 'P1-紧急',
        p2: 'P2-高',
        p3: 'P3-中',
        p4: 'P4-低',
      };
      active.push({ key: 'priority', label: '优先级', value: priorityLabels[filters.priority] || filters.priority });
    }
    
    if (filters?.keyword && filters.keyword.trim()) {
      active.push({ key: 'keyword', label: '关键词', value: filters.keyword });
    }
    
    if (filters?.dateStart && filters?.dateEnd) {
      active.push({ 
        key: 'dateRange', 
        label: '日期范围', 
        value: `${filters.dateStart} 至 ${filters.dateEnd}` 
      });
    }
    
    return active;
  }, [filters]);

  // 移除单个筛选条件
  const handleRemoveFilter = useCallback((key: string) => {
    if (key === 'status') {
      onFilterChange({ status: 'all' });
    } else if (key === 'priority') {
      onFilterChange({ priority: 'all' });
    } else if (key === 'keyword') {
      setLocalKeyword('');
      onFilterChange({ keyword: '' });
    } else if (key === 'dateRange') {
      onFilterChange({ dateStart: '', dateEnd: '' });
    }
  }, [onFilterChange]);

    if (loading) {
        return <TicketFiltersSkeleton />;
    }

  return (
    <Card className='mb-4' styles={{ body: { padding: '16px' } }} data-testid='ticket-filters'>
      <Space direction='vertical' style={{ width: '100%' }} size='middle'>
        {/* 视图选择器和筛选预设 */}
        <Row gutter={[12, 12]}>
          {onViewChange && (
            <Col xs={24} sm={12} md={8} lg={6}>
              <TicketViewSelector
                currentViewId={currentViewId}
                filters={filters}
                onViewChange={onViewChange}
                onFiltersChange={onFilterChange}
              />
            </Col>
          )}
          <Col xs={24} sm={12} md={8} lg={6}>
            <FilterPresetSelector filters={filters} onFiltersChange={onFilterChange} />
          </Col>
        </Row>

        <Row gutter={[12, 12]} align='middle'>
          {/* 搜索框 */}
          <Col xs={24} sm={24} md={8} lg={6}>
            <Input
              placeholder='搜索工单标题或描述...'
              prefix={<SearchOutlined />}
              value={localKeyword}
              onChange={handleSearchInputChange}
              onPressEnter={handleSearch}
              allowClear
              data-testid='filter-keyword-input'
              disabled={loading}
            />
          </Col>

          {/* 状态筛选 */}
          <Col xs={12} sm={8} md={4} lg={3}>
            <Select
              style={{ width: '100%' }}
              placeholder='状态'
              value={filters?.status || 'all'}
              onChange={handleStatusChange}
              data-testid='filter-status-select'
              options={[
                { label: '全部状态', value: 'all' },
                { label: '待处理', value: 'open' },
                { label: '处理中', value: 'in_progress' },
                { label: '已解决', value: 'resolved' },
                { label: '已关闭', value: 'closed' },
              ]}
              disabled={loading}
            />
          </Col>

          {/* 优先级筛选 */}
          <Col xs={12} sm={8} md={4} lg={3}>
            <Select
              style={{ width: '100%' }}
              placeholder='优先级'
              value={filters?.priority || 'all'}
              onChange={handlePriorityChange}
              data-testid='filter-priority-select'
              options={[
                { label: '全部优先级', value: 'all' },
                { label: 'P1-紧急', value: 'p1' },
                { label: 'P2-高', value: 'p2' },
                { label: 'P3-中', value: 'p3' },
                { label: 'P4-低', value: 'p4' },
              ]}
              disabled={loading}
            />
          </Col>

          {/* 日期范围 */}
          <Col xs={24} sm={12} md={6} lg={5}>
            <RangePicker
              style={{ width: '100%' }}
              value={dateRange as any}
              onChange={handleDateRangeChange}
              format='YYYY-MM-DD'
              placeholder={['开始日期', '结束日期']}
              data-testid='filter-date-range'
              disabled={loading}
            />
          </Col>

          {/* 排序 */}
          <Col xs={12} sm={8} md={4} lg={3}>
            <Select
              style={{ width: '100%' }}
              placeholder='排序'
              value={filters?.sortBy || 'createdAt_desc'}
              onChange={handleSortChange}
              data-testid='filter-sort-select'
              options={[
                { label: '最新创建', value: 'createdAt_desc' },
                { label: '最早创建', value: 'createdAt_asc' },
                { label: '优先级↓', value: 'priority_desc' },
                { label: '优先级↑', value: 'priority_asc' },
              ]}
              disabled={loading}
            />
          </Col>

          {/* 操作按钮 */}
          <Col xs={12} sm={8} md={6} lg={4}>
            <Space.Compact style={{ width: '100%' }}>
              <Button
                icon={<SearchOutlined />}
                type={activeFilters.length > 0 ? 'primary' : 'default'}
                onClick={handleSearch}
                loading={loading}
                data-testid='filter-apply-btn'
                disabled={loading}
              >
                搜索
                {activeFilters.length > 0 && (
                  <Badge count={activeFilters.length} offset={[8, -2]} />
                )}
              </Button>
              <Button
                icon={<ClearOutlined />}
                onClick={handleReset}
                data-testid='filter-reset-btn'
                disabled={loading || activeFilters.length === 0}
              >
                重置
              </Button>
              {onRefresh && (
                <Button
                  icon={<ReloadOutlined />}
                  onClick={onRefresh}
                  loading={loading}
                  data-testid='filter-refresh-btn'
                  disabled={loading}
                >
                  刷新
                </Button>
              )}
            </Space.Compact>
          </Col>
        </Row>

        {/* 已应用的筛选标签 */}
        {activeFilters.length > 0 && (
          <Row>
            <Col span={24}>
              <div className='flex items-center gap-2 flex-wrap pt-2 border-t border-gray-100'>
                <span className='text-sm text-gray-500 flex items-center gap-1'>
                  <FilterOutlined />
                  已应用筛选：
                </span>
                {activeFilters.map(filter => (
                  <Tag
                    key={filter.key}
                    closable
                    onClose={() => handleRemoveFilter(filter.key)}
                    color='blue'
                    className='mb-0'
                  >
                    {filter.label}: {filter.value}
                  </Tag>
                ))}
                <Button
                  type='link'
                  size='small'
                  icon={<CloseOutlined />}
                  onClick={handleReset}
                  className='p-0 h-auto'
                >
                  清除全部
                </Button>
              </div>
            </Col>
          </Row>
        )}
      </Space>
    </Card>
  );
}

export default TicketFilters;
export { TicketFilters };
