'use client';

import React, { useMemo, useCallback } from 'react';
import { Card, Select, Input, DatePicker, Button, Space, Row, Col } from 'antd';
import { SearchOutlined, ReloadOutlined, ClearOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';

const { RangePicker } = DatePicker;

export interface FilterOption {
  key: string;
  label: string;
  options: Array<{ value: string; label: string }>;
}

export interface UnifiedFilterState {
  [key: string]: string | number | undefined;
  keyword?: string;
  dateStart?: string;
  dateEnd?: string;
  sortBy?: string;
}

interface UnifiedFiltersProps {
  filters: UnifiedFilterState;
  onFilterChange: (filters: Partial<UnifiedFilterState>) => void;
  onSearch?: (keyword: string) => void;
  onRefresh?: () => void;
  onReset?: () => void;
  loading?: boolean;
  filterOptions?: FilterOption[];
  showKeyword?: boolean;
  showDateRange?: boolean;
  showSort?: boolean;
  sortOptions?: Array<{ value: string; label: string }>;
  className?: string;
}

const DEFAULT_SORT_OPTIONS = [
  { value: 'createdAt_desc', label: '创建时间（新到旧）' },
  { value: 'createdAt_asc', label: '创建时间（旧到新）' },
  { value: 'priority_desc', label: '优先级（高到低）' },
  { value: 'priority_asc', label: '优先级（低到高）' },
];

export function UnifiedFilters({
  filters = {},
  onFilterChange,
  onSearch,
  onRefresh,
  onReset,
  loading = false,
  filterOptions = [],
  showKeyword = true,
  showDateRange = true,
  showSort = true,
  sortOptions = DEFAULT_SORT_OPTIONS,
  className,
}: UnifiedFiltersProps) {
  const [localKeyword, setLocalKeyword] = React.useState(filters?.keyword || '');

  // 同步外部filters到本地keyword
  React.useEffect(() => {
    if (filters?.keyword !== undefined) {
      setLocalKeyword(filters.keyword);
    }
  }, [filters?.keyword]);

  // 处理筛选变化
  const handleFilterChange = useCallback(
    (key: string, value: string | undefined) => {
      onFilterChange({ [key]: value });
    },
    [onFilterChange]
  );

  // 处理关键词搜索
  const handleKeywordSearch = useCallback(() => {
    onFilterChange({ keyword: localKeyword });
    onSearch?.(localKeyword);
  }, [localKeyword, onFilterChange, onSearch]);

  // 处理日期范围变化
  const handleDateRangeChange = useCallback(
    (dates: [dayjs.Dayjs | null, dayjs.Dayjs | null] | null) => {
      if (dates && dates[0] && dates[1]) {
        onFilterChange({
          dateStart: dates[0].format('YYYY-MM-DD'),
          dateEnd: dates[1].format('YYYY-MM-DD'),
        });
      } else {
        onFilterChange({
          dateStart: undefined,
          dateEnd: undefined,
        });
      }
    },
    [onFilterChange]
  );

  // 处理排序变化
  const handleSortChange = useCallback(
    (value: string) => {
      onFilterChange({ sortBy: value });
    },
    [onFilterChange]
  );

  // 处理重置
  const handleReset = useCallback(() => {
    setLocalKeyword('');
    onReset?.();
    onFilterChange({
      keyword: undefined,
      dateStart: undefined,
      dateEnd: undefined,
      sortBy: undefined,
    });
    filterOptions.forEach(option => {
      onFilterChange({ [option.key]: undefined });
    });
  }, [onReset, onFilterChange, filterOptions]);

  // 计算日期范围值
  const dateRangeValue = useMemo(() => {
    if (filters.dateStart && filters.dateEnd) {
      return [dayjs(filters.dateStart), dayjs(filters.dateEnd)] as [dayjs.Dayjs, dayjs.Dayjs];
    }
    return null;
  }, [filters.dateStart, filters.dateEnd]);

  return (
    <Card className={className} styles={{ body: { padding: '16px' } }}>
      <Space orientation='vertical' style={{ width: '100%' }} size='middle'>
        <Row gutter={[12, 12]} align='middle'>
          {/* 关键词搜索 */}
          {showKeyword && (
            <Col xs={24} sm={24} md={8} lg={6}>
              <Input
                placeholder='搜索关键词...'
                prefix={<SearchOutlined />}
                value={localKeyword}
                onChange={e => setLocalKeyword(e.target.value)}
                onPressEnter={handleKeywordSearch}
                allowClear
              />
            </Col>
          )}

          {/* 动态筛选选项 */}
          {filterOptions.map(option => (
            <Col xs={12} sm={8} md={4} lg={3} key={option.key}>
              <Select
                placeholder={option.label}
                value={filters[option.key] as string}
                onChange={value => handleFilterChange(option.key, value)}
                allowClear
                style={{ width: '100%' }}
                options={option.options}
              />
            </Col>
          ))}

          {/* 日期范围 */}
          {showDateRange && (
            <Col xs={24} sm={12} md={6} lg={5}>
              <RangePicker
                value={dateRangeValue}
                onChange={handleDateRangeChange}
                style={{ width: '100%' }}
                format='YYYY-MM-DD'
              />
            </Col>
          )}

          {/* 排序 */}
          {showSort && (
            <Col xs={12} sm={8} md={4} lg={3}>
              <Select
                placeholder='排序'
                value={filters.sortBy as string}
                onChange={handleSortChange}
                style={{ width: '100%' }}
                options={sortOptions}
              />
            </Col>
          )}

          {/* 操作按钮 */}
          <Col xs={12} sm={8} md={6} lg={4}>
            <Space>
              {showKeyword && (
                <Button
                  type='primary'
                  icon={<SearchOutlined />}
                  onClick={handleKeywordSearch}
                  loading={loading}
                >
                  搜索
                </Button>
              )}
              {onRefresh && (
                <Button icon={<ReloadOutlined />} onClick={onRefresh} loading={loading}>
                  刷新
                </Button>
              )}
              {onReset && (
                <Button icon={<ClearOutlined />} onClick={handleReset} disabled={loading}>
                  重置
                </Button>
              )}
            </Space>
          </Col>
        </Row>
      </Space>
    </Card>
  );
}
